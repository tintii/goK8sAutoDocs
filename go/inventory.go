package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type inventoryPart struct {
	Data          []map[string]any `json:"data"`
	LastSuccessAt *time.Time       `json:"lastSuccessAt"`
	LastAttemptAt time.Time        `json:"lastAttemptAt"`
	Error         string           `json:"error,omitempty"`
	Warnings      []string         `json:"warnings"`
}
type targetSnapshot struct {
	ConfigID   string                   `json:"configId"`
	EndpointID string                   `json:"endpointId,omitempty"`
	Parts      map[string]inventoryPart `json:"parts"`
}
type inventorySnapshot struct {
	Version    int                       `json:"version"`
	Generation uint64                    `json:"generation"`
	Targets    map[string]targetSnapshot `json:"targets"`
}
type collectionResult struct {
	Duration time.Duration
	Data     []map[string]any
	Warnings []string
	Err      error
}
type inventoryService struct {
	logger       *slog.Logger
	mu           sync.RWMutex
	config       inventoryConfig
	snapshot     inventorySnapshot
	running      bool
	completed    int
	current      string
	lastFinished time.Time
	nextAuto     time.Time
	cacheError   string
	ctx          context.Context
	now          func() time.Time
	connect      func(targetConfig) (targetClients, error)
	collect      func(context.Context, targetConfig, targetClients) map[string]collectionResult
}

func newInventoryService(c inventoryConfig) *inventoryService {
	return newInventoryServiceWithLogger(c, appLogger)
}
func newInventoryServiceWithLogger(c inventoryConfig, logger *slog.Logger) *inventoryService {
	s := &inventoryService{logger: logger, config: c, ctx: context.Background(), now: func() time.Time { return time.Now().UTC() }, connect: connectTarget}
	s.snapshot = inventorySnapshot{Version: 1, Targets: map[string]targetSnapshot{}}
	s.collect = s.collectTarget
	s.restore()
	return s
}
func (s *inventoryService) start(ctx context.Context) {
	s.ctx = ctx
	s.requestWithTrigger(nil, "startup")
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.mu.RLock()
				due := !s.running && !s.nextAuto.IsZero() && !s.now().Before(s.nextAuto)
				s.mu.RUnlock()
				if due {
					s.request(nil, true)
				}
			}
		}
	}()
}

// A request only schedules server-owned work. Readers never hold this lock for collection.
func (s *inventoryService) request(observed *uint64, automatic bool) bool {
	trigger := "manual"
	if automatic {
		trigger = "automatic"
	}
	return s.requestWithTrigger(observed, trigger)
}
func (s *inventoryService) requestWithTrigger(observed *uint64, trigger string) bool {
	s.mu.Lock()
	reason := "accepted"
	switch {
	case s.running:
		reason = "already running"
	case trigger == "manual" && observed != nil && *observed < s.snapshot.Generation:
		reason = "outdated generation"
	case trigger == "manual" && s.now().Before(s.lastFinished.Add(duration(s.config.Refresh.Cooldown))):
		reason = "cooldown"
	}
	generation := s.snapshot.Generation
	if reason == "accepted" {
		s.running = true
		s.completed = 0
		s.current = ""
	}
	s.mu.Unlock()
	s.logger.Debug("Refresh scheduling decision", "event", "refresh.requested", "trigger", trigger, "generation", generation, "reason", reason)
	if reason != "accepted" {
		return false
	}
	go s.refresh(trigger)
	return true
}
func cloneSnapshot(v inventorySnapshot) inventorySnapshot {
	b, _ := json.Marshal(v)
	var out inventorySnapshot
	_ = json.Unmarshal(b, &out)
	return out
}
func (s *inventoryService) refresh(trigger string) {
	started := time.Now()
	s.mu.RLock()
	next := cloneSnapshot(s.snapshot)
	previousCacheError := s.cacheError
	s.mu.RUnlock()
	logger := s.logger.With("generation", next.Generation+1, "trigger", trigger)
	logger.Info("Inventory refresh started", "event", "refresh.started", "target_count", len(s.config.Clusters))
	succeeded, failed, warningCount := 0, 0, 0
	for _, t := range s.config.Clusters {
		targetStarted := time.Now()
		targetLogger := logger.With("cluster_id", logIdentity(t.ID))
		s.mu.Lock()
		s.current = t.ID
		s.mu.Unlock()
		old := next.Targets[t.ID]
		if old.ConfigID != s.config.targetFingerprint(t) {
			old = targetSnapshot{Parts: map[string]inventoryPart{}}
		}
		old.ConfigID = s.config.targetFingerprint(t)
		clients, err := s.connect(t)
		results := map[string]collectionResult{}
		if err != nil {
			retained := false
			for _, part := range old.Parts {
				retained = retained || part.LastSuccessAt != nil
			}
			targetLogger.Warn("Cannot connect to cluster", "event", "target.connection_failed", "reason", safeCollectionIssue(err), "retained_previous", retained)
			for _, key := range []string{"ingresses", "httproutes", "workloads", "filtered"} {
				results[key] = collectionResult{Err: err}
			}
		} else {
			if old.EndpointID != "" && old.EndpointID != clients.EndpointID {
				old.Parts = map[string]inventoryPart{}
			}
			old.EndpointID = clients.EndpointID
			ctx, cancel := context.WithTimeout(s.ctx, duration(s.config.Refresh.ClusterTimeout))
			results = s.collect(ctx, t, clients)
			cancel()
		}
		for key, result := range results {
			part := old.Parts[key]
			part.LastAttemptAt = s.now()
			componentLogger := targetLogger.With("component", key)
			if result.Err != nil {
				failed++
				part.Error = safeCollectionIssue(result.Err)
				if err == nil { // The connection boundary already reports target-wide failures.
					componentLogger.Warn("Inventory collection failed", "event", "collection.failed", "reason", part.Error, "retained_previous", part.LastSuccessAt != nil)
				}
			} else {
				succeeded++
				issues := endpointIssueCount(result.Data)
				warningCount += len(result.Warnings) + issues
				if len(result.Warnings)+issues > 0 {
					componentLogger.Warn("Inventory collection incomplete", "event", "collection.partial", "reason", "some metadata or endpoints unavailable", "warning_count", len(result.Warnings), "issue_count", issues)
				}
				if (part.Error != "" || len(part.Warnings) > 0 || endpointIssueCount(part.Data) > 0) && len(result.Warnings)+issues == 0 {
					componentLogger.Info("Inventory collection recovered", "event", "collection.recovered")
				}
				componentLogger.Debug("Inventory component collected", "event", "collection.completed", "count", len(result.Data), "duration_ms", result.Duration.Milliseconds(), "warning_count", len(result.Warnings), "issue_count", issues)
				part.Data = result.Data
				if part.Data == nil {
					part.Data = []map[string]any{}
				}
				at := s.now()
				part.LastSuccessAt = &at
				part.Error = ""
				part.Warnings = result.Warnings
			}
			old.Parts[key] = part
		}
		next.Targets[t.ID] = old
		targetLogger.Debug("Cluster collection completed", "event", "target.completed", "duration_ms", time.Since(targetStarted).Milliseconds())
		s.mu.Lock()
		s.completed++
		s.mu.Unlock()
	}
	// Preserve-mode spelling is chosen from a sorted set, never arrival order.
	spellings := map[string]map[string]string{}
	for _, target := range next.Targets {
		for _, part := range target.Parts {
			for _, row := range part.Data {
				for key, v := range classificationOf(row) {
					if spellings[key] == nil {
						spellings[key] = map[string]string{}
					}
					if previous := spellings[key][v.ID]; previous == "" || v.Display < previous {
						spellings[key][v.ID] = v.Display
					}
				}
			}
		}
	}
	for _, target := range next.Targets {
		for _, part := range target.Parts {
			for _, row := range part.Data {
				values := classificationOf(row)
				for key, v := range values {
					v.Display = spellings[key][v.ID]
					values[key] = v
				}
				row["classification"] = values
			}
		}
	}
	next.Generation++
	s.mu.Lock()
	s.snapshot = next
	s.mu.Unlock()
	cacheErr := s.persist(next)
	if s.config.Cache.Path != "" && previousCacheError != "" && cacheErr == "" {
		logger.Info("Snapshot storage recovered", "event", "snapshot.recovered")
	}
	logger.Info("Inventory refresh completed", "event", "refresh.completed", "duration_ms", time.Since(started).Milliseconds(), "successful_components", succeeded, "failed_components", failed, "warning_count", warningCount, "cache_available", cacheErr == "")
	s.mu.Lock()
	s.cacheError = cacheErr
	s.lastFinished = s.now()
	s.nextAuto = s.lastFinished.Add(duration(s.config.Refresh.Interval))
	s.running = false
	s.current = ""
	s.mu.Unlock()
}
func safeCollectionIssue(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "collection timed out"
	case apierrors.IsForbidden(err):
		return "access forbidden"
	case apierrors.IsUnauthorized(err):
		return "authentication rejected"
	case os.IsNotExist(err):
		return "credentials unavailable"
	}
	// Error values can include request URLs or credential parser contents. Only return owned categories.
	for _, message := range []string{"credentials unavailable", "invalid or unsupported credentials", "cannot initialize Kubernetes client", "cannot initialize resource client"} {
		if err.Error() == message {
			return message
		}
	}
	return "collection unavailable; check connectivity, credentials and read permissions"
}
func (s *inventoryService) restore() {
	if s.config.Cache.Path == "" {
		return
	}
	b, err := boundedFile(s.config.Cache.Path, 32<<20)
	if os.IsNotExist(err) {
		s.logger.Debug("No saved inventory", "event", "snapshot.absent")
		return
	}
	if err != nil {
		s.cacheError = "saved inventory unavailable"
		s.logger.Warn("Cannot restore inventory", "event", "snapshot.restore_failed", "reason", s.cacheError)
		return
	}
	var saved inventorySnapshot
	if json.Unmarshal(b, &saved) != nil || saved.Version != 1 || saved.Targets == nil {
		s.cacheError = "saved inventory invalid or incompatible"
		s.logger.Warn("Cannot restore inventory", "event", "snapshot.restore_failed", "reason", s.cacheError)
		return
	}
	s.snapshot.Generation = saved.Generation
	for _, t := range s.config.Clusters {
		if v, ok := saved.Targets[t.ID]; ok && v.ConfigID == s.config.targetFingerprint(t) && v.Parts != nil {
			s.snapshot.Targets[t.ID] = v
		}
	}
	if len(s.snapshot.Targets) > 0 {
		s.logger.Info("Saved inventory restored", "event", "snapshot.restored", "generation", saved.Generation, "target_count", len(s.snapshot.Targets))
	} else {
		s.logger.Warn("No compatible saved targets", "event", "snapshot.restore_failed", "reason", "saved targets do not match configuration")
	}
}
func (s *inventoryService) persist(v inventorySnapshot) (issue string) {
	path := s.config.Cache.Path
	if path == "" {
		return ""
	}
	defer func() {
		if issue != "" {
			s.logger.Warn("Cannot persist inventory", "event", "snapshot.persist_failed", "generation", v.Generation, "reason", issue)
		} else {
			s.logger.Debug("Inventory snapshot saved", "event", "snapshot.persisted", "generation", v.Generation)
		}
	}()
	data, err := json.Marshal(v)
	if err != nil || len(data) > 32<<20 {
		return "inventory snapshot exceeds persistence limit"
	}
	dir := filepath.Dir(path)
	if os.MkdirAll(dir, 0700) != nil {
		return "inventory cache directory is not writable"
	}
	f, err := os.CreateTemp(dir, ".snapshot-*")
	if err != nil {
		return "inventory snapshot could not be written"
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(name, path)
	}
	if err != nil {
		return "inventory snapshot could not be saved"
	}
	return ""
}
func labelsOf(row map[string]any) map[string]string {
	if labels, ok := row["labels"].(map[string]string); ok {
		return labels
	}
	labels := map[string]string{}
	if raw, ok := row["labels"].(map[string]any); ok {
		for k, v := range raw {
			if text, ok := v.(string); ok {
				labels[k] = text
			}
		}
	}
	return labels
}
func classificationOf(row map[string]any) map[string]classifiedValue {
	if v, ok := row["classification"].(map[string]classifiedValue); ok {
		return v
	}
	result := map[string]classifiedValue{}
	b, _ := json.Marshal(row["classification"])
	_ = json.Unmarshal(b, &result)
	return result
}
func asRow(v any) map[string]any {
	b, _ := json.Marshal(v)
	out := map[string]any{}
	_ = json.Unmarshal(b, &out)
	return out
}
func (s *inventoryService) collectTarget(ctx context.Context, t targetConfig, c targetClients) map[string]collectionResult {
	results := map[string]collectionResult{}
	namespaces := map[string]map[string]string{}
	namespaceErrors := map[string]bool{}
	decorate := func(rows []map[string]any) []string {
		warnings := []string{}
		for _, row := range rows {
			row["clusterId"], row["clusterName"] = t.ID, t.Name
			ns, _ := row["namespace"].(string)
			needNamespace := false
			for _, rule := range s.config.Classification {
				for _, source := range rule.Sources {
					if source == "namespace" && len(rule.LabelKeys) > 0 {
						needNamespace = true
					}
				}
			}
			if _, ok := namespaces[ns]; !ok && needNamespace {
				object, err := c.Kube.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
				if err != nil {
					namespaces[ns] = nil
					namespaceErrors[ns] = true
				} else {
					namespaces[ns] = object.Labels
				}
			}
			if namespaceErrors[ns] {
				warnings = append(warnings, "Namespace classification unavailable: "+ns)
			}
			values := map[string]classifiedValue{}
			for key, rule := range s.config.Classification {
				if value := classify(rule, map[string]map[string]string{"resource": labelsOf(row), "namespace": namespaces[ns], "cluster": t.Labels}); value != nil {
					values[key] = *value
				}
			}
			row["classification"] = values
		}
		return warnings
	}
	partStarted := time.Now()
	ingresses, err := c.Kube.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	part := collectionResult{Data: []map[string]any{}, Err: err}
	if err == nil {
		for _, obj := range ingresses.Items {
			if value, present := obj.Labels[publishLabel]; present && value != "true" {
				continue
			}
			entry := ingressEntry(obj)
			related, e := findRelatedIngressResources(ctx, c.Kube, obj)
			if e != nil {
				entry.Issues = append(entry.Issues, "Related resources unavailable")
			}
			for _, r := range related {
				entry.addRelated(r)
			}
			part.Data = append(part.Data, asRow(entry))
		}
		part.Warnings = decorate(part.Data)
	}
	part.Duration = time.Since(partStarted)
	results["ingresses"] = part
	partStarted = time.Now()
	routes, err := c.Dynamic.Resource(httpRouteGVR).Namespace("").List(ctx, metav1.ListOptions{})
	part = collectionResult{Data: []map[string]any{}, Err: err}
	if apierrors.IsNotFound(err) {
		part.Err = nil
		s.logger.Debug("Optional source API absent", "event", "collection.optional_absent", "cluster_id", logIdentity(t.ID), "component", "httproutes")
	}
	if err == nil {
		collector := endpointCollector{kube: c.Kube, dynamic: c.Dynamic, gateways: map[resourceReference]gatewayLookup{}, namespaces: namespaces}
		for _, obj := range routes.Items {
			if value, present := obj.GetLabels()[publishLabel]; present && value != "true" {
				continue
			}
			part.Data = append(part.Data, asRow(collector.routeEntry(ctx, obj)))
		}
		part.Warnings = decorate(part.Data)
	}
	part.Duration = time.Since(partStarted)
	results["httproutes"] = part
	partStarted = time.Now()
	workloads, warnings, err := collectSnapshotWorkloads(ctx, c.Kube)
	warnings = append(warnings, decorate(workloads)...)
	results["workloads"] = collectionResult{Data: workloads, Warnings: warnings, Err: err, Duration: time.Since(partStarted)}
	partStarted = time.Now()
	// Endpoint metadata is derived from the exact same source partition as the directory.
	filtered := []map[string]any{}
	fw := []string{}
	for _, r := range []string{"ingresses", "httproutes"} {
		for _, row := range results[r].Data {
			if labelsOf(row)[publishLabel] == "true" {
				filtered = append(filtered, map[string]any{"name": row["name"], "namespace": row["namespace"], "kind": row["kind"], "labels": labelsOf(row), "clusterId": t.ID, "clusterName": t.Name, "classification": row["classification"]})
			}
		}
	}
	for _, resource := range filteredResourcesGVR {
		if resource.Resource == "ingresses" || resource.Resource == "httproutes" {
			continue
		}
		rows, issues, e := collectFilteredResources(ctx, c.Dynamic, "", []GVRResource{resource})
		if e != nil || len(issues) > 0 {
			fw = append(fw, "Filtered metadata collection incomplete: "+resource.Resource)
			continue
		}
		decorate(rows)
		filtered = append(filtered, rows...)
	}
	// Retain the previous whole filtered partition if any required source list failed.
	var filteredErr error
	if len(fw) > 0 || results["ingresses"].Err != nil || results["httproutes"].Err != nil {
		filteredErr = errors.New("filtered collection incomplete")
	}
	results["filtered"] = collectionResult{Data: filtered, Warnings: fw, Err: filteredErr, Duration: time.Since(partStarted)}
	return results
}
func collectSnapshotWorkloads(ctx context.Context, k kubernetes.Interface) ([]map[string]any, []string, error) {
	rows := []map[string]any{}
	warnings := []string{}
	add := func(meta metav1.Object, kind string, spec corev1.PodSpec, ready, desired int32, observed int64) {
		state := "attention"
		if desired == 0 {
			state = "inactive"
		} else if ready >= desired && observed >= meta.GetGeneration() {
			state = "ready"
		}
		rows = append(rows, map[string]any{"name": meta.GetName(), "namespace": meta.GetNamespace(), "kind": kind, "labels": meta.GetLabels(), "containers": extractContainersFromPodSpec(&spec), "ready": ready, "desired": desired, "state": state})
	}
	deps, err := k.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	for _, d := range deps.Items {
		desired := int32(1)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		add(&d, "Deployment", d.Spec.Template.Spec, d.Status.AvailableReplicas, desired, d.Status.ObservedGeneration)
		related, e := findRelatedDeploymentResources(ctx, k, appsv1.Deployment(d))
		if e != nil {
			warnings = append(warnings, "Pod image digests unavailable: "+d.Namespace+"/"+d.Name)
		} else {
			for _, container := range rows[len(rows)-1]["containers"].([]map[string]any) {
				for _, r := range related {
					if container["name"] == r.Name {
						container["sha"] = r.ImageSha
						break
					}
				}
			}
		}
	}
	sets, err := k.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	for _, d := range sets.Items {
		desired := int32(1)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		add(&d, "StatefulSet", d.Spec.Template.Spec, d.Status.AvailableReplicas, desired, d.Status.ObservedGeneration)
	}
	daemons, err := k.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	for _, d := range daemons.Items {
		add(&d, "DaemonSet", d.Spec.Template.Spec, d.Status.NumberAvailable, d.Status.DesiredNumberScheduled, d.Status.ObservedGeneration)
	}
	sort.Slice(rows, func(i, j int) bool {
		return fmt.Sprint(rows[i]["namespace"], "/", rows[i]["kind"], "/", rows[i]["name"]) < fmt.Sprint(rows[j]["namespace"], "/", rows[j]["kind"], "/", rows[j]["name"])
	})
	return rows, warnings, nil
}
