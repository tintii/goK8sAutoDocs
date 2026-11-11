package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// These are hard failure budgets, not truncation thresholds. See verification.md.
const maxRanchers = 10
const maxSourceClusters = 100
const maxInventoryConnections = 200
const maxSourceBytes = 16 << 20
const maxPageBytes = 4 << 20
const sourceTimeout = 15 * time.Second
const identityTimeout = 2 * time.Second
const identityGenerationTimeout = 20 * time.Second

var routeID = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,251}[a-z0-9])?$`)

type rancherConfig struct {
	ID                string           `yaml:"id" json:"id"`
	Name              string           `yaml:"name" json:"name"`
	Kubeconfig        kubeconfigSource `yaml:"kubeconfig" json:"kubeconfig"`
	ExcludeClusterIDs []string         `yaml:"excludeClusterIDs" json:"excludeClusterIDs,omitempty"`
}
type connectionRef struct {
	Direct    string `yaml:"direct,omitempty" json:"direct,omitempty"`
	Rancher   string `yaml:"rancher,omitempty" json:"rancher,omitempty"`
	ClusterID string `yaml:"clusterID,omitempty" json:"clusterID,omitempty"`
}

func (r connectionRef) key() string { b, _ := json.Marshal(r); return string(b) }
func (r connectionRef) canonicalID() string {
	if r.Direct != "" {
		return r.Direct
	}
	return "rancher:" + base64.RawURLEncoding.EncodeToString([]byte(r.Rancher)) + ":" + base64.RawURLEncoding.EncodeToString([]byte(r.ClusterID))
}

type clusterBinding struct {
	ID          string            `yaml:"id" json:"id"`
	Name        string            `yaml:"name" json:"name"`
	Labels      map[string]string `yaml:"labels" json:"labels,omitempty"`
	Connections []connectionRef   `yaml:"connections" json:"connections"`
	Primary     connectionRef     `yaml:"primary" json:"primary"`
}
type rancherMember struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type rancherSnapshot struct {
	ConfigID      string          `json:"configId"`
	EndpointID    string          `json:"endpointId,omitempty"`
	Members       []rancherMember `json:"members"`
	LastAttemptAt time.Time       `json:"lastAttemptAt"`
	LastSuccessAt *time.Time      `json:"lastSuccessAt"`
	Error         string          `json:"error,omitempty"`
	Restored      bool            `json:"restored,omitempty"`
}
type connectionStatus struct {
	Ref       connectionRef `json:"ref"`
	Role      string        `json:"role"`
	Available bool          `json:"available"`
	Identity  string        `json:"identity"` // Only a category; the marker is private compatibility metadata.
}
type resolvedTarget struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Labels      map[string]string  `json:"labels,omitempty"`
	Primary     connectionRef      `json:"primary"`
	Connections []connectionStatus `json:"connections"`
	Warnings    []string           `json:"warnings"`
	ConfigID    string             `json:"configId"`
}

func validStableID(id string) bool {
	return id != "" && id == strings.TrimSpace(id) && len(id) <= 1024 && !strings.ContainsAny(id, "\x00\r\n")
}
func excluded(r rancherConfig, id string) bool {
	for _, e := range r.ExcludeClusterIDs {
		if e == id {
			return true
		}
	}
	return false
}
func (c inventoryConfig) validateRanchers() error {
	sources := map[string]rancherConfig{}
	direct := map[string]targetConfig{}
	if len(c.Ranchers) > maxRanchers {
		return errors.New("too many Rancher sources")
	}
	for _, t := range c.Clusters {
		if !validStableID(t.ID) {
			return errors.New("invalid direct cluster ID")
		}
		direct[t.ID] = t
	}
	for _, s := range c.Ranchers {
		if !validStableID(s.ID) || len(s.ID) > 256 || sources[s.ID].ID != "" || s.Kubeconfig.Path == "" || s.Kubeconfig.Context == "" {
			return errors.New("Rancher sources require unique IDs and explicit kubeconfig path/context")
		}
		seen := map[string]bool{}
		for _, id := range s.ExcludeClusterIDs {
			if !routeID.MatchString(id) || seen[id] {
				return errors.New("invalid or duplicate Rancher exclusion")
			}
			seen[id] = true
		}
		sources[s.ID] = s
	}
	validateRef := func(r connectionRef) bool {
		if r.Direct != "" {
			_, ok := direct[r.Direct]
			return ok && r.Rancher == "" && r.ClusterID == ""
		}
		s, ok := sources[r.Rancher]
		return ok && routeID.MatchString(r.ClusterID) && !excluded(s, r.ClusterID)
	}
	assigned := map[string]bool{}
	ids := map[string]bool{}
	for _, b := range c.ClusterBindings {
		if len(c.ClusterBindings) > maxInventoryConnections || !validStableID(b.ID) || ids[b.ID] || len(b.Connections) == 0 || len(b.Connections) > maxInventoryConnections {
			return errors.New("invalid or duplicate canonical binding")
		}
		ids[b.ID] = true
		hasPrimary := false
		ownsDirect := false
		hasExistingDirect := false
		for _, r := range b.Connections {
			if !validateRef(r) || assigned[r.key()] {
				return errors.New("invalid, excluded or multiply assigned connection")
			}
			assigned[r.key()] = true
			if len(assigned) > maxInventoryConnections {
				return errors.New("too many bound connections")
			}
			hasPrimary = hasPrimary || r == b.Primary
			if r.Direct != "" {
				hasExistingDirect = true
				ownsDirect = ownsDirect || r.Direct == b.ID
			}
		}
		if !hasPrimary || !validateRef(b.Primary) {
			return errors.New("primary must be a member of its binding")
		}
		// Preserve an existing identity rather than silently moving browser preferences.
		if hasExistingDirect && !ownsDirect {
			return errors.New("binding containing direct connections must retain a member's direct ID")
		}
		if strings.HasPrefix(b.ID, "rancher:") {
			found := false
			for _, r := range b.Connections {
				found = found || r.canonicalID() == b.ID
			}
			if !found {
				return errors.New("reserved canonical ID must belong to a binding member")
			}
		}
	}
	for id := range direct {
		if ids[id] && !assigned[(connectionRef{Direct: id}).key()] {
			return errors.New("canonical binding collides with unmapped direct target")
		}
	}
	for id := range direct {
		assigned[(connectionRef{Direct: id}).key()] = true
	}
	if len(assigned) > maxInventoryConnections {
		return errors.New("too many configured connections")
	}
	return nil
}
func sourceFingerprint(r rancherConfig) string { return fingerprint(r) }
func (c inventoryConfig) rancherTargetFingerprint(source rancherConfig, endpoint, cluster string) string {
	return fingerprint(struct {
		Source, Endpoint, Cluster string
		Rules                     map[string]classificationRule
	}{sourceFingerprint(source), endpoint, cluster, c.Classification})
}
func sourceEndpoint(cfg *rest.Config) string {
	return fingerprint(struct {
		Host, ServerName string
		CA               []byte
	}{cfg.Host, cfg.ServerName, cfg.CAData})
}
func loadRancher(r rancherConfig) (*rest.Config, error) {
	b, err := boundedFile(r.Kubeconfig.Path, 4<<20)
	if err != nil {
		return nil, errors.New("credentials unavailable")
	}
	cfg, err := configFromKubeconfig(b, r.Kubeconfig.Context)
	if err != nil || cfg.BearerToken == "" || len(cfg.CertData) != 0 || len(cfg.KeyData) != 0 {
		return nil, errors.New("invalid or unsupported credentials")
	}
	u, err := url.Parse(cfg.Host)
	if err != nil || u.Path != "" && u.Path != "/" || u.RawPath != "" || cfg.Proxy != nil {
		return nil, errors.New("invalid or unsupported credentials")
	}
	cfg.Host = strings.TrimRight(cfg.Host, "/")
	cfg.Timeout = 10 * time.Second
	cfg.QPS = 10
	cfg.Burst = 15
	return cfg, nil
}
func rancherHTTP(cfg *rest.Config) (*http.Client, error) {
	transport, err := rest.TransportFor(cfg)
	if err != nil {
		return nil, errors.New("invalid or unsupported credentials")
	}
	return &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("Rancher redirect rejected") }}, nil
}
func enumerateRancher(ctx context.Context, r rancherConfig) ([]rancherMember, *rest.Config, error) {
	cfg, err := loadRancher(r)
	if err != nil {
		return nil, nil, err
	}
	client, err := rancherHTTP(cfg)
	if err != nil {
		return nil, nil, err
	}
	members := []rancherMember{}
	seen := map[string]bool{}
	tokens := map[string]bool{}
	continuation := ""
	totalBytes := 0
	for page := 0; page < 100; page++ {
		q := url.Values{"limit": {"50"}}
		if continuation != "" {
			q.Set("continue", continuation)
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, cfg.Host+"/apis/management.cattle.io/v3/clusters?"+q.Encode(), nil)
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return nil, nil, errors.New("Rancher enumeration unavailable")
		}
		b, readErr := io.ReadAll(io.LimitReader(resp.Body, maxPageBytes+1))
		resp.Body.Close()
		switch resp.StatusCode {
		case 401:
			return nil, nil, errors.New("Rancher authentication rejected")
		case 403:
			return nil, nil, errors.New("Rancher enumeration forbidden; check token scope and management read permissions")
		case 404, 405:
			return nil, nil, errors.New("Rancher RK-API incompatible or unavailable")
		}
		totalBytes += len(b)
		if readErr != nil || resp.StatusCode != 200 {
			return nil, nil, errors.New("Rancher enumeration unavailable")
		}
		if len(b) > maxPageBytes || totalBytes > maxSourceBytes {
			return nil, nil, errors.New("Rancher discovery budget exceeded")
		}
		var list struct {
			APIVersion string `json:"apiVersion"`
			Kind       string `json:"kind"`
			Metadata   struct {
				Continue string `json:"continue"`
			} `json:"metadata"`
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
				Spec struct {
					DisplayName string `json:"displayName"`
				} `json:"spec"`
			} `json:"items"`
		}
		if json.Unmarshal(b, &list) != nil || list.APIVersion != "management.cattle.io/v3" || list.Kind != "ClusterList" || list.Items == nil {
			return nil, nil, errors.New("Rancher RK-API incompatible response")
		}
		for _, item := range list.Items {
			id := item.Metadata.Name
			if !routeID.MatchString(id) || seen[id] || len(item.Spec.DisplayName) > 512 || strings.ContainsAny(item.Spec.DisplayName, "\x00\r\n") {
				return nil, nil, errors.New("Rancher invalid cluster response")
			}
			seen[id] = true
			if len(seen) > maxSourceClusters {
				return nil, nil, errors.New("Rancher discovery budget exceeded")
			}
			if excluded(r, id) {
				continue
			}
			name := item.Spec.DisplayName
			if name == "" {
				name = id
			}
			members = append(members, rancherMember{ID: id, Name: name})
		}
		continuation = list.Metadata.Continue
		if continuation == "" {
			sort.Slice(members, func(i, j int) bool { return members[i].ID < members[j].ID })
			return members, cfg, nil
		}
		if len(continuation) > 8192 || tokens[continuation] {
			return nil, nil, errors.New("Rancher invalid pagination")
		}
		tokens[continuation] = true
	}
	return nil, nil, errors.New("Rancher discovery budget exceeded")
}
func proxyClients(cfg *rest.Config, id string) (targetClients, error) {
	if !routeID.MatchString(id) {
		return targetClients{}, errors.New("invalid or unsupported credentials")
	}
	copy := rest.CopyConfig(cfg)
	copy.Host = strings.TrimRight(cfg.Host, "/") + "/k8s/clusters/" + id
	h, err := rancherHTTP(copy)
	if err != nil {
		return targetClients{}, err
	}
	k, err := kubernetes.NewForConfigAndClient(copy, h)
	if err != nil {
		return targetClients{}, errors.New("cannot initialize Kubernetes client")
	}
	d, err := dynamic.NewForConfigAndClient(copy, h)
	if err != nil {
		return targetClients{}, errors.New("cannot initialize resource client")
	}
	return targetClients{Kube: k, Dynamic: d, EndpointID: sourceEndpoint(copy)}, nil
}
func probeIdentity(ctx context.Context, c targetClients) (string, error) {
	if c.Kube == nil {
		return "", errors.New("identity check unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, identityTimeout)
	defer cancel()
	ns, err := c.Kube.CoreV1().Namespaces().Get(ctx, "kube-system", metav1.GetOptions{})
	if err != nil || ns.UID == "" {
		return "", errors.New("identity check unavailable")
	}
	return fingerprint(string(ns.UID)), nil
}

// Registry construction uses sanitized metadata only, never credential-bearing config.
func (c inventoryConfig) resolveRegistry(sources map[string]rancherSnapshot) ([]resolvedTarget, error) {
	type candidate struct {
		ref       connectionRef
		name      string
		labels    map[string]string
		available bool
		configID  string
	}
	candidates := map[string]candidate{}
	for _, t := range c.Clusters {
		r := connectionRef{Direct: t.ID}
		candidates[r.key()] = candidate{r, t.Name, t.Labels, true, c.targetFingerprint(t)}
	}
	for _, source := range c.Ranchers {
		state := sources[source.ID]
		for _, m := range state.Members {
			r := connectionRef{Rancher: source.ID, ClusterID: m.ID}
			candidates[r.key()] = candidate{r, m.Name, nil, state.Error == "" && !state.Restored, c.rancherTargetFingerprint(source, state.EndpointID, m.ID)}
		}
	}
	allRefs := map[string]bool{}
	for key := range candidates {
		allRefs[key] = true
	}
	for _, binding := range c.ClusterBindings {
		for _, ref := range binding.Connections {
			allRefs[ref.key()] = true
		}
	}
	if len(allRefs) > maxInventoryConnections {
		return nil, errors.New("Rancher aggregate discovery budget exceeded")
	}
	assigned := map[string]bool{}
	out := []resolvedTarget{}
	ids := map[string]bool{}
	add := func(t resolvedTarget) error {
		if ids[t.ID] {
			return errors.New("Rancher canonical ID collision")
		}
		ids[t.ID] = true
		out = append(out, t)
		return nil
	}
	for _, b := range c.ClusterBindings {
		name := b.Name
		labels := b.Labels
		for _, r := range b.Connections {
			assigned[r.key()] = true
			if v, ok := candidates[r.key()]; ok {
				if r.Direct == b.ID {
					if name == "" {
						name = v.name
					}
					if labels == nil {
						labels = v.labels
					}
				}
			}
		}
		if name == "" {
			name = b.ID
		}
		t := resolvedTarget{ID: b.ID, Name: name, Labels: labels, Primary: b.Primary, Warnings: []string{}}
		for _, r := range b.Connections {
			v, ok := candidates[r.key()]
			role := "secondary"
			if r == b.Primary {
				role = "primary"
			}
			t.Connections = append(t.Connections, connectionStatus{Ref: r, Role: role, Available: ok && v.available, Identity: "unavailable"})
		}
		primary := candidates[b.Primary.key()]
		// Missing dynamic primaries retain compatibility using source identity and endpoint.
		primaryID := primary.configID
		if primaryID == "" && b.Primary.Rancher != "" {
			for _, src := range c.Ranchers {
				if src.ID == b.Primary.Rancher {
					primaryID = c.rancherTargetFingerprint(src, sources[src.ID].EndpointID, b.Primary.ClusterID)
				}
			}
		}
		t.ConfigID = fingerprint(struct {
			ID, Name, Primary, Config string
			Labels                    map[string]string
			Rules                     map[string]classificationRule
		}{t.ID, t.Name, t.Primary.key(), primaryID, t.Labels, c.Classification})
		if err := add(t); err != nil {
			return nil, err
		}
	}
	for key, v := range candidates {
		if assigned[key] {
			continue
		}
		t := resolvedTarget{ID: v.ref.canonicalID(), Name: v.name, Labels: v.labels, Primary: v.ref, ConfigID: v.configID, Connections: []connectionStatus{{Ref: v.ref, Role: "primary", Available: v.available, Identity: "unavailable"}}, Warnings: []string{}}
		if err := add(t); err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
