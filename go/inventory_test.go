package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func fixtureConfig(t *testing.T) inventoryConfig {
	t.Helper()
	c := defaultInventoryConfig()
	c.Clusters = []targetConfig{{ID: "a", Name: "A", InCluster: true}, {ID: "b", Name: "B", InCluster: true}}
	c.Cache.Path = filepath.Join(t.TempDir(), "snapshot.json")
	if err := c.validate(); err != nil {
		t.Fatal(err)
	}
	return c
}
func waitInventory(t *testing.T, s *inventoryService) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.RLock()
		running := s.running
		s.mu.RUnlock()
		if !running {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("inventory did not finish")
}
func fixtureKubeconfig() string {
	return `apiVersion: v1
kind: Config
clusters:
- name: same
  cluster:
    server: https://api.example.invalid
users:
- name: same
  user:
    token: DO-NOT-PERSIST-THIS-TOKEN
contexts:
- name: same
  context:
    cluster: same
    user: same
current-context: same
`
}
func TestInventoryConfigurationAndCredentials(t *testing.T) {
	b := fixtureKubeconfig()
	cfg, err := configFromKubeconfig([]byte(b), "same")
	if err != nil || cfg.BearerToken != "DO-NOT-PERSIST-THIS-TOKEN" {
		t.Fatalf("token credentials: %v", err)
	}
	for name, body := range map[string]string{"exec": strings.Replace(b, "token: DO-NOT-PERSIST-THIS-TOKEN", "exec:\n      command: never-execute", 1), "file": strings.Replace(b, "token: DO-NOT-PERSIST-THIS-TOKEN", "tokenFile: /private/file", 1), "http": strings.Replace(b, "https://", "http://", 1), "impersonate": strings.Replace(b, "token: DO-NOT-PERSIST-THIS-TOKEN", "token: x\n    as: admin", 1)} {
		t.Run(name, func(t *testing.T) {
			if _, err := configFromKubeconfig([]byte(body), "same"); err == nil {
				t.Fatal("unsupported credentials accepted")
			}
		})
	}
	if _, err := configFromKubeconfig([]byte(b), "missing"); err == nil {
		t.Fatal("missing context accepted")
	}
	cert := strings.Replace(b, "token: DO-NOT-PERSIST-THIS-TOKEN", "client-certificate-data: Y2VydA==\n    client-key-data: a2V5", 1)
	if cfg, err := configFromKubeconfig([]byte(cert), "same"); err != nil || string(cfg.CertData) != "cert" || string(cfg.KeyData) != "key" {
		t.Fatalf("embedded cert/key: %v", err)
	}
	c := fixtureConfig(t)
	c.Clusters[1].ID = "a"
	if c.validate() == nil {
		t.Fatal("duplicate IDs accepted")
	}
	path := filepath.Join(t.TempDir(), "kubeconfig")
	if err := os.WriteFile(path, []byte(b), 0600); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b"} {
		target := targetConfig{ID: id, Kubeconfig: &kubeconfigSource{Path: path, Context: "same"}}
		if _, err := connectTarget(target); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(path, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := connectTarget(targetConfig{ID: "a", Kubeconfig: &kubeconfigSource{Path: path, Context: "same"}}); err == nil {
		t.Fatal("superseded credentials reused")
	}
}
func TestClassificationPrecedenceAndEquality(t *testing.T) {
	rule := classificationRule{LabelKeys: []string{"env", "fallback"}, Sources: []string{"resource", "namespace"}, DisplayCase: "upper"}
	sources := map[string]map[string]string{"resource": {"env": "npe00"}, "namespace": {"env": "PRD00"}}
	a := classify(rule, sources)
	if a.ID != "npe00" || a.Display != "NPE00" {
		t.Fatalf("classification: %+v", a)
	}
	sources["resource"]["env"] = "NPE00"
	if classify(rule, sources).ID != a.ID {
		t.Fatal("case-fold mismatch")
	}
	rule.CaseSensitive = true
	if classify(rule, sources).ID == a.ID {
		t.Fatal("case sensitive mode collapsed values")
	}
	sources["resource"] = map[string]string{}
	if classify(rule, sources).Display != "PRD00" {
		t.Fatal("namespace fallback failed")
	}
	delete(sources, "namespace")
	if classify(rule, sources) != nil {
		t.Fatal("missing metadata invented")
	}
}
func TestInventoryCoalescesAndRetainsFailures(t *testing.T) {
	c := fixtureConfig(t)
	s := newInventoryService(c)
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	var calls atomic.Int32
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	s.connect = func(targetConfig) (targetClients, error) { return targetClients{EndpointID: "same"}, nil }
	s.collect = func(ctx context.Context, target targetConfig, _ targetClients) map[string]collectionResult {
		calls.Add(1)
		started <- struct{}{}
		<-release
		return map[string]collectionResult{"ingresses": {Data: []map[string]any{{"name": "same", "namespace": "default", "kind": "ingress", "clusterId": target.ID, "labels": map[string]string{publishLabel: "true"}}}}}
	}
	if !s.request(nil, false) {
		t.Fatal("first request rejected")
	}
	<-started
	for i := 0; i < 20; i++ {
		if s.request(nil, false) {
			t.Fatal("overlapping request started")
		}
		w := httptest.NewRecorder()
		s.dataHandler("directory")(w, httptest.NewRequest("GET", "/api/v1/auto-docs-dashboard", nil))
		if w.Code != 503 {
			t.Fatal("invented initial inventory")
		}
	}
	close(release)
	waitInventory(t, s)
	if calls.Load() != 2 {
		t.Fatalf("expected one collection per target, got %d", calls.Load())
	}
	first := *s.snapshot.Targets["a"].Parts["ingresses"].LastSuccessAt
	if s.request(nil, false) {
		t.Fatal("cooldown did not suppress late click")
	}
	now = now.Add(time.Minute)
	old := uint64(0)
	if s.request(&old, false) {
		t.Fatal("obsolete generation triggered scan")
	}
	s.collect = func(_ context.Context, target targetConfig, _ targetClients) map[string]collectionResult {
		if target.ID == "a" {
			return map[string]collectionResult{"ingresses": {Err: context.DeadlineExceeded}}
		}
		return map[string]collectionResult{"ingresses": {Data: []map[string]any{}}}
	}
	s.request(nil, false)
	waitInventory(t, s)
	if p := s.snapshot.Targets["a"].Parts["ingresses"]; len(p.Data) != 1 || !p.LastSuccessAt.Equal(first) || p.Error == "" {
		t.Fatalf("failed partition lost: %+v", p)
	}
	if len(s.snapshot.Targets["b"].Parts["ingresses"].Data) != 0 {
		t.Fatal("successful empty result retained deleted objects")
	}
	w := httptest.NewRecorder()
	s.dataHandler("directory")(w, httptest.NewRequest("GET", "/api/v1/auto-docs-dashboard", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"clusterId":"a"`) || w.Header().Get(warningsHeader) == "" {
		t.Fatalf("retained response: %s", w.Body.String())
	}
	if s.nextAuto.Sub(now) != 5*time.Minute {
		t.Fatal("schedule does not follow completion")
	}
}
func TestInventoryPersistenceAndCompatibility(t *testing.T) {
	c := fixtureConfig(t)
	s := newInventoryService(c)
	at := time.Now().UTC()
	s.snapshot.Targets["a"] = targetSnapshot{ConfigID: c.targetFingerprint(c.Clusters[0]), Parts: map[string]inventoryPart{"ingresses": {Data: []map[string]any{{"name": "old"}}, LastSuccessAt: &at}}}
	s.snapshot.Generation = 7
	if err := s.persist(s.snapshot); err != "" {
		t.Fatal(err)
	}
	restored := newInventoryService(c)
	if restored.snapshot.Generation != 7 || !restored.snapshot.Targets["a"].Parts["ingresses"].LastSuccessAt.Equal(at) {
		t.Fatal("hydration lost collection time")
	}
	info, _ := os.Stat(c.Cache.Path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("snapshot permissions")
	}
	c.Clusters = c.Clusters[1:]
	if len(newInventoryService(c).snapshot.Targets) != 0 {
		t.Fatal("removed target restored")
	}
	if err := os.WriteFile(c.Cache.Path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if newInventoryService(c).cacheError == "" {
		t.Fatal("corrupt file ignored silently")
	}
	s.config.Cache.Path = filepath.Join(c.Cache.Path, "snapshot")
	if s.persist(s.snapshot) == "" {
		t.Fatal("write failure ignored")
	}
}
func TestInventoryRefreshHTTPAndViewValidation(t *testing.T) {
	s := newInventoryService(fixtureConfig(t))
	s.connect = func(targetConfig) (targetClients, error) {
		return targetClients{}, errors.New("credentials unavailable")
	}
	for _, tc := range []struct {
		method, path, origin string
		want                 int
	}{{"GET", "/api/v1/discovery-refresh", "", 405}, {"POST", "/api/v1/discovery-refresh", "https://evil.invalid", 403}, {"POST", "/api/v1/discovery-refresh", "", 202}} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
		r.Header.Set("Origin", tc.origin)
		s.refreshHandler(w, r)
		if w.Code != tc.want {
			t.Fatalf("%s got %d", tc.method, w.Code)
		}
	}
	waitInventory(t, s)
	w := httptest.NewRecorder()
	s.dataHandler("directory")(w, httptest.NewRequest("GET", "/?view=published&view=discovered", nil))
	if w.Code != 400 {
		t.Fatal("repeated view accepted")
	}
	status, _ := json.Marshal(s.status())
	if strings.Contains(string(status), "kubeconfig") || strings.Contains(string(status), "DO-NOT-PERSIST") {
		t.Fatal("credentials exposed")
	}
}
func TestInventoryActualCollectionIsClusterScoped(t *testing.T) {
	c := fixtureConfig(t)
	c.Classification = map[string]classificationRule{"environment": {LabelKeys: []string{"env"}, Sources: []string{"resource", "namespace"}, DisplayCase: "upper"}}
	s := newInventoryService(c)
	for _, target := range c.Clusters {
		k := fake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "apps", Labels: map[string]string{"env": "npe00"}}}, &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "same", Namespace: "apps", Labels: map[string]string{publishLabel: "true"}}}, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "apps"}})
		scheme := runtime.NewScheme()
		_ = appsv1.AddToScheme(scheme)
		_ = corev1.AddToScheme(scheme)
		d := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{httpRouteGVR: "HTTPRouteList"})
		results := s.collectTarget(context.Background(), target, targetClients{Kube: k, Dynamic: d})
		row := results["ingresses"].Data[0]
		if row["clusterId"] != target.ID || classificationOf(row)["environment"].Display != "NPE00" {
			t.Fatalf("wrong provenance: %+v", row)
		}
		for _, action := range k.Actions() {
			if action.GetResource().Resource == "secrets" {
				t.Fatal("Secret API read")
			}
		}
		k.PrependReactor("list", "ingresses", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "ingresses"}, "", errors.New("no"))
		})
		if s.collectTarget(context.Background(), target, targetClients{Kube: k, Dynamic: d})["ingresses"].Err == nil {
			t.Fatal("failed list presented as success")
		}
	}
}
