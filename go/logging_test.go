package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// A shared capture supports concurrent handler/refresh tests without a global logger.
type logCapture struct {
	mu sync.Mutex
	bytes.Buffer
}

func (b *logCapture) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.Write(p)
}
func (b *logCapture) output() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.String()
}
func logRecords(t *testing.T, output string) []map[string]any {
	t.Helper()
	var records []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("invalid JSON log: %q: %v", line, err)
		}
		for _, key := range []string{"time", "level", "msg", "event"} {
			if record[key] == nil {
				t.Fatalf("missing %s in log: %v", key, record)
			}
		}
		records = append(records, record)
	}
	return records
}
func events(t *testing.T, output, name string) []map[string]any {
	t.Helper()
	var found []map[string]any
	for _, r := range logRecords(t, output) {
		if r["event"] == name {
			found = append(found, r)
		}
	}
	return found
}

func TestLoggingLevelConfiguration(t *testing.T) {
	for _, tc := range []struct {
		level, debug string
		want         slog.Level
	}{
		{"", "", slog.LevelInfo}, {"  ", "true", slog.LevelDebug},
		{"", "TRUE", slog.LevelInfo}, {"debug", "", slog.LevelDebug},
		{" INFO ", "true", slog.LevelInfo}, {" Warn ", "true", slog.LevelWarn},
		{"error", "true", slog.LevelError},
	} {
		t.Run(tc.level+"/"+tc.debug, func(t *testing.T) {
			var capture logCapture
			logger, err := configuredLogger(&capture, tc.level, tc.debug)
			if err != nil {
				t.Fatal(err)
			}
			for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
				logger.Log(context.Background(), level, "fixture", "event", "fixture.level")
			}
			records := logRecords(t, capture.output())
			want := int((slog.LevelError-tc.want)/4) + 1
			if len(records) != want || records[0]["level"] != tc.want.String() {
				t.Fatalf("threshold %s: %v", tc.want, records)
			}
		})
	}
}

func TestLoggingStartupProcess(t *testing.T) {
	if os.Getenv("AUTODOCS_LOGGING_SUBPROCESS") == "1" {
		main()
		return
	}
	for _, tc := range []struct{ level, event string }{
		{"SECRET-INVALID-LEVEL", "startup.logging_failed"},
		{"info", "startup.config_failed"},
	} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestLoggingStartupProcess$")
		cmd.Env = append(os.Environ(), "AUTODOCS_LOGGING_SUBPROCESS=1", "LOG_LEVEL="+tc.level, "DEBUG=true", "AUTODOCS_CONFIG="+filepath.Join(t.TempDir(), "SECRET-CONFIG-PATH"))
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 || stdout.Len() != 0 {
			t.Fatalf("startup exit/output: %v stdout=%s stderr=%s", err, &stdout, &stderr)
		}
		records := logRecords(t, stderr.String())
		if len(records) != 1 || records[0]["event"] != tc.event || records[0]["level"] != "ERROR" || strings.Contains(stderr.String(), "SECRET-") {
			t.Fatalf("unsafe or late startup failure: %s", &stderr)
		}
	}
}

func TestLoggingRefreshOutcomesAndSafeContext(t *testing.T) {
	for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		t.Run(level.String(), func(t *testing.T) {
			var capture logCapture
			c := fixtureConfig(t)
			c.Cache.Path = ""
			s := newInventoryServiceWithLogger(c, newLogger(&capture, level))
			phase := 0
			s.connect = func(target targetConfig) (targetClients, error) {
				if phase == 2 && target.ID == "a" {
					return targetClients{}, errors.New("https://token:SECRET-TOKEN@api.invalid?secret=SECRET-QUERY")
				}
				return targetClients{}, nil
			}
			s.collect = func(_ context.Context, target targetConfig, _ targetClients) map[string]collectionResult {
				out := map[string]collectionResult{}
				for _, key := range []string{"ingresses", "httproutes", "workloads", "filtered"} {
					out[key] = collectionResult{Data: []map[string]any{{"name": "SECRET-OBJECT", "namespace": "apps", "kind": key, "clusterId": target.ID, "labels": map[string]string{publishLabel: "true", "fixture": "SECRET-CONFIGMAP"}}}}
				}
				if phase == 1 && target.ID == "a" {
					out["workloads"] = collectionResult{Err: apierrors.NewForbidden(schema.GroupResource{Resource: "deployments"}, "SECRET-RESOURCE", errors.New("SECRET-TOKEN"))}
					out["ingresses"] = collectionResult{Data: out["ingresses"].Data, Warnings: []string{"SECRET-WARNING"}}
					out["httproutes"] = collectionResult{Data: []map[string]any{{"issues": []any{"SECRET-ENDPOINT"}}}}
				}
				return out
			}
			for _, trigger := range []string{"startup", "automatic", "manual", "manual"} {
				s.refresh(trigger)
				phase++
			}
			output := capture.output()
			if strings.Contains(output, "SECRET-") || strings.Contains(output, "api.invalid") {
				t.Fatalf("private fixture data reached logs: %s", output)
			}
			if level <= slog.LevelInfo {
				completed := events(t, output, "refresh.completed")
				if len(completed) != 4 || completed[1]["failed_components"] != float64(1) || completed[2]["failed_components"] != float64(4) || completed[0]["trigger"] != "startup" || completed[1]["trigger"] != "automatic" {
					t.Fatalf("incorrect run summaries: %v", completed)
				}
				if len(events(t, output, "collection.recovered")) != 4 {
					t.Fatal("missing collection recovery transitions")
				}
			}
			if level <= slog.LevelWarn {
				connection := events(t, output, "target.connection_failed")
				failed := events(t, output, "collection.failed")
				if len(connection) != 1 || connection[0]["cluster_id"] != "a" || connection[0]["retained_previous"] != true || len(failed) != 1 || failed[0]["reason"] != "access forbidden" || len(events(t, output, "collection.partial")) != 2 {
					t.Fatalf("failure boundaries: connection=%v failed=%v", connection, failed)
				}
			}
			if level == slog.LevelError && output != "" {
				t.Fatal("recoverable collection failure emitted an error")
			}
		})
	}
}

func TestLoggingCoalescingAndQuietReaders(t *testing.T) {
	var capture logCapture
	c := fixtureConfig(t)
	c.Cache.Path = ""
	s := newInventoryServiceWithLogger(c, newLogger(&capture, slog.LevelDebug))
	now := time.Now().UTC()
	s.now = func() time.Time { return now }
	var calls atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	s.connect = func(targetConfig) (targetClients, error) { return targetClients{}, nil }
	s.collect = func(_ context.Context, target targetConfig, _ targetClients) map[string]collectionResult {
		if calls.Add(1) == 1 {
			close(entered)
		}
		<-release
		return map[string]collectionResult{"ingresses": {Data: []map[string]any{{"clusterId": target.ID, "labels": map[string]string{publishLabel: "true"}}}}}
	}
	s.request(nil, true)
	<-entered
	var group sync.WaitGroup
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if s.request(nil, false) {
				t.Error("overlapping refresh accepted")
			}
			w := httptest.NewRecorder()
			s.dataHandler("directory")(w, httptest.NewRequest("GET", "/api/v1/auto-docs-dashboard", nil))
			if w.Code != 503 {
				t.Error("initial snapshot status changed")
			}
		}()
	}
	group.Wait()
	close(release)
	waitInventory(t, s)
	s.request(nil, false)
	now = now.Add(time.Minute)
	old := uint64(0)
	s.request(&old, false)
	decisions := events(t, capture.output(), "refresh.requested")
	if len(decisions) != 23 || decisions[21]["reason"] != "cooldown" || decisions[22]["reason"] != "outdated generation" || len(events(t, capture.output(), "refresh.started")) != 1 || len(events(t, capture.output(), "refresh.completed")) != 1 {
		t.Fatalf("incorrect scheduling records: %v", decisions)
	}
	// A stale partition is served silently; warnings belong to its refresh.
	s.snapshot.Targets["a"].Parts["ingresses"] = inventoryPart{LastSuccessAt: &now, Error: "collection timed out", Data: []map[string]any{}}
	before := capture.output()
	for i := 0; i < 10; i++ {
		for _, handler := range []func(*httptest.ResponseRecorder){
			func(w *httptest.ResponseRecorder) {
				s.statusHandler(w, httptest.NewRequest("GET", "/api/v1/discovery-status", nil))
			},
			func(w *httptest.ResponseRecorder) {
				s.dataHandler("directory")(w, httptest.NewRequest("GET", "/api/v1/auto-docs-dashboard", nil))
			},
		} {
			w := httptest.NewRecorder()
			handler(w)
			if w.Code != 200 {
				t.Fatalf("snapshot read failed: %d", w.Code)
			}
		}
		writeJSON(httptest.NewRecorder(), 200, map[string]string{"status": "running"}, s.logger, "health")
	}
	if capture.output() != before || calls.Load() != 2 {
		t.Fatal("polling amplified logs or Kubernetes collection")
	}
}

func TestLoggingSnapshotsAndOptionalAPI(t *testing.T) {
	var capture logCapture
	logger := newLogger(&capture, slog.LevelDebug)
	c := fixtureConfig(t)
	s := newInventoryServiceWithLogger(c, logger)
	if len(events(t, capture.output(), "snapshot.absent")) != 1 {
		t.Fatal("missing first-start diagnostic")
	}
	s.connect = func(targetConfig) (targetClients, error) { return targetClients{}, nil }
	s.collect = func(context.Context, targetConfig, targetClients) map[string]collectionResult {
		return map[string]collectionResult{"ingresses": {}}
	}
	s.refresh("startup")
	newInventoryServiceWithLogger(c, logger)
	if len(events(t, capture.output(), "snapshot.restored")) != 1 {
		t.Fatal("missing compatible restore event")
	}
	if err := os.WriteFile(c.Cache.Path, []byte("SECRET-CORRUPT-SNAPSHOT"), 0600); err != nil {
		t.Fatal(err)
	}
	newInventoryServiceWithLogger(c, logger)
	s.config.Cache.Path = filepath.Join(c.Cache.Path, "snapshot")
	s.refresh("automatic")
	s.config.Cache.Path = c.Cache.Path
	s.refresh("automatic")
	if len(events(t, capture.output(), "snapshot.restore_failed")) != 1 || len(events(t, capture.output(), "snapshot.persist_failed")) != 1 || len(events(t, capture.output(), "snapshot.recovered")) != 1 {
		t.Fatal("missing snapshot failure/recovery events")
	}
	k := fake.NewSimpleClientset()
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	d := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{httpRouteGVR: "HTTPRouteList"})
	d.PrependReactor("list", "httproutes", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewNotFound(httpRouteGVR.GroupResource(), "")
	})
	before := len(logRecords(t, capture.output()))
	s.collectTarget(context.Background(), c.Clusters[0], targetClients{Kube: k, Dynamic: d})
	records := logRecords(t, capture.output())[before:]
	if len(records) != 1 || records[0]["event"] != "collection.optional_absent" || records[0]["level"] != "DEBUG" || strings.Contains(capture.output(), "SECRET-") {
		t.Fatalf("optional API/no credential disclosure: %v", records)
	}
}

func TestLoggingHandlerFailures(t *testing.T) {
	var capture logCapture
	logger := newLogger(&capture, slog.LevelDebug)
	path := filepath.Join(t.TempDir(), "home.yaml")
	if err := os.WriteFile(path, []byte("SECRET-YAML-FIELD: [invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOMEPAGE_CONFIG", path)
	w := httptest.NewRecorder()
	serveHomeConfig(w, httptest.NewRequest("GET", "/api/v1/home?token=SECRET-QUERY", nil), logger)
	if w.Code != 500 || len(events(t, capture.output(), "home.config_failed")) != 1 {
		t.Fatal("missing homepage configuration diagnostic")
	}
	w = httptest.NewRecorder()
	writeJSON(w, 200, map[string]any{"SECRET-DATA": make(chan int)}, logger, "fixture")
	if w.Code != 500 || len(events(t, capture.output(), "http.encoding_failed")) != 1 {
		t.Fatal("encoding failure committed a successful status or omitted diagnostic")
	}
	handler := cachedResourceHandlerWithLogger(time.Minute, func(context.Context) (any, []string, error) {
		return nil, nil, errors.New("https://SECRET-TOKEN@api.invalid?token=SECRET-QUERY")
	}, logger)
	w = httptest.NewRecorder()
	handler(w, httptest.NewRequest("GET", "/legacy", nil))
	if w.Code != 500 || len(events(t, capture.output(), "collection.legacy_failed")) != 1 || strings.Contains(capture.output(), "SECRET-") {
		t.Fatalf("unsafe handler logs: %s", capture.output())
	}
	if len(logIdentity(strings.Repeat("x", 1000))) > 132 {
		t.Fatal("unbounded identity attribute")
	}
}
