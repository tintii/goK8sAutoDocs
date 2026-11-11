package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func rancherFixture(t testing.TB, handler http.HandlerFunc) (rancherConfig, *httptest.Server) {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	ca := base64.StdEncoding.EncodeToString(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}))
	path := filepath.Join(t.TempDir(), "private-kubeconfig")
	body := strings.Replace(fixtureKubeconfig(), "https://api.example.invalid", server.URL, 1)
	body = strings.Replace(body, "server: "+server.URL, "server: "+server.URL+"\n    certificate-authority-data: "+ca, 1)
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return rancherConfig{ID: "r", Kubeconfig: kubeconfigSource{Path: path, Context: "same"}}, server
}
func listJSON(w http.ResponseWriter, ids []string, continuation string) {
	items := []any{}
	for _, id := range ids {
		items = append(items, map[string]any{"metadata": map[string]any{"name": id}, "spec": map[string]any{"displayName": id}})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"apiVersion": "management.cattle.io/v3", "kind": "ClusterList", "metadata": map[string]any{"continue": continuation}, "items": items})
}
func TestRancherPaginationAndCredentialRotation(t *testing.T) {
	var token atomic.Value
	token.Store("DO-NOT-PERSIST-THIS-TOKEN")
	var calls atomic.Int32
	src, _ := rancherFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer "+token.Load().(string) {
			w.WriteHeader(401)
			return
		}
		if r.URL.Path != "/apis/management.cattle.io/v3/clusters" {
			t.Error("unexpected route")
			w.WriteHeader(404)
			return
		}
		if r.URL.Query().Get("continue") == "" {
			listJSON(w, []string{"local", "c-imported"}, "opaque+/=")
		} else if r.URL.Query().Get("continue") == "opaque+/=" {
			listJSON(w, []string{"c-m-provisioned"}, "")
		} else {
			t.Error("continuation changed")
			w.WriteHeader(400)
		}
	})
	src.ExcludeClusterIDs = []string{"local"}
	members, cfg, err := enumerateRancher(context.Background(), src)
	if err != nil || len(members) != 2 || members[1].ID != "c-m-provisioned" {
		t.Fatalf("enumerate: %v %v", members, err)
	}
	endpoint := sourceEndpoint(cfg)
	b, _ := os.ReadFile(src.Kubeconfig.Path)
	b = []byte(strings.Replace(string(b), "DO-NOT-PERSIST-THIS-TOKEN", "replacement-token", 1))
	_ = os.WriteFile(src.Kubeconfig.Path, b, 0600)
	token.Store("replacement-token")
	_, rotated, err := enumerateRancher(context.Background(), src)
	if err != nil || sourceEndpoint(rotated) != endpoint {
		t.Fatal("rotation changed identity", err)
	}
	for _, bad := range []string{"invalid", strings.Replace(string(b), "token: replacement-token", "exec:\n      command: never-execute", 1), strings.Replace(string(b), "token: replacement-token", "client-certificate-data: Y2VydA==\n    client-key-data: a2V5", 1), strings.Replace(string(b), "token: replacement-token", "tokenFile: /private/token", 1)} {
		before := calls.Load()
		_ = os.WriteFile(src.Kubeconfig.Path, []byte(bad), 0600)
		if _, _, err := enumerateRancher(context.Background(), src); err == nil || calls.Load() != before {
			t.Fatal("invalid replacement used old credential")
		}
	}
	_ = os.Remove(src.Kubeconfig.Path)
	if _, _, err := enumerateRancher(context.Background(), src); err == nil {
		t.Fatal("missing file accepted")
	}
}
func TestRancherInvalidListingsAndUntrustedNavigation(t *testing.T) {
	var leaked atomic.Int32
	evil := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked.Add(1) }))
	defer evil.Close()
	for _, scenario := range []string{"later-page", "repeat-token", "malformed", "wrong-kind", "invalid-id", "duplicate", "overflow", "oversize", "redirect", "next-link", "forbidden", "unsupported", "empty"} {
		t.Run(scenario, func(t *testing.T) {
			src, _ := rancherFixture(t, func(w http.ResponseWriter, r *http.Request) {
				switch scenario {
				case "later-page":
					if r.URL.Query().Get("continue") != "" {
						w.WriteHeader(503)
					} else {
						listJSON(w, []string{"local"}, "next")
					}
				case "repeat-token":
					listJSON(w, []string{}, "next")
				case "malformed":
					fmt.Fprint(w, `{"items":`)
				case "wrong-kind":
					fmt.Fprint(w, `{"apiVersion":"v1","kind":"List","items":[]}`)
				case "invalid-id":
					listJSON(w, []string{"../other"}, "")
				case "duplicate":
					listJSON(w, []string{"local", "local"}, "")
				case "overflow":
					ids := []string{}
					for i := 0; i < 101; i++ {
						ids = append(ids, fmt.Sprintf("c-%d", i))
					}
					listJSON(w, ids, "")
				case "oversize":
					fmt.Fprint(w, strings.Repeat(" ", maxPageBytes+1))
				case "redirect":
					http.Redirect(w, r, evil.URL, 302)
				case "next-link":
					fmt.Fprintf(w, `{"apiVersion":"management.cattle.io/v3","kind":"ClusterList","metadata":{"continue":"%s"},"items":[]}`, evil.URL)
				case "forbidden":
					w.WriteHeader(403)
				case "unsupported":
					w.WriteHeader(404)
				case "empty":
					listJSON(w, []string{}, "")
				}
			})
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			members, _, err := enumerateRancher(ctx, src)
			if scenario == "empty" {
				if err != nil || len(members) != 0 {
					t.Fatal(err)
				}
			} else if err == nil || members != nil {
				t.Fatal("partial or invalid listing accepted")
			}
		})
	}
	if leaked.Load() != 0 {
		t.Fatal("untrusted destination received a request")
	}
}
func TestRancherProxyReusesCollectors(t *testing.T) {
	var calls atomic.Int32
	src, _ := rancherFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer DO-NOT-PERSIST-THIS-TOKEN" {
			w.WriteHeader(401)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/k8s/clusters/c-imported/") {
			t.Error("collector bypassed proxy")
			w.WriteHeader(404)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(r.URL.Path, "/k8s/clusters/c-imported")
		if path == "/api/v1/namespaces/kube-system" {
			fmt.Fprint(w, `{"apiVersion":"v1","kind":"Namespace","metadata":{"name":"kube-system","uid":"marker"}}`)
			return
		}
		if strings.Contains(path, "gateway.networking.k8s.io") {
			w.WriteHeader(404)
			return
		}
		resource := path[strings.LastIndex(path, "/")+1:]
		kind := map[string]string{"services": "ServiceList", "pods": "PodList", "configmaps": "ConfigMapList", "ingresses": "IngressList", "deployments": "DeploymentList", "statefulsets": "StatefulSetList", "daemonsets": "DaemonSetList", "replicasets": "ReplicaSetList"}[resource]
		apiVersion := "v1"
		if strings.Contains(path, "/apps/") {
			apiVersion = "apps/v1"
		}
		if strings.Contains(path, "/networking.k8s.io/") {
			apiVersion = "networking.k8s.io/v1"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"apiVersion": apiVersion, "kind": kind, "items": []any{}})
	})
	cfg, err := loadRancher(src)
	if err != nil {
		t.Fatal(err)
	}
	clients, err := proxyClients(cfg, "c-imported")
	if err != nil {
		t.Fatal(err)
	}
	s := newInventoryService(fixtureConfig(t))
	results := s.collectTarget(context.Background(), targetConfig{ID: "canonical", Name: "Canonical"}, clients)
	for key, result := range results {
		if result.Err != nil {
			t.Errorf("%s: %v", key, result.Err)
		}
	}
	if marker, err := probeIdentity(context.Background(), clients); err != nil || marker == "" {
		t.Fatal("proxy probe failed", err)
	}
	if calls.Load() < 8 {
		t.Fatal("collectors not exercised")
	}
	if _, err := proxyClients(cfg, "../evil"); err == nil {
		t.Fatal("invalid route accepted")
	}
}
func TestRancherConfigBindings(t *testing.T) {
	base := func() inventoryConfig {
		c := defaultInventoryConfig()
		c.Clusters = []targetConfig{{ID: "home", InCluster: true}}
		c.Ranchers = []rancherConfig{{ID: "r", Kubeconfig: kubeconfigSource{Path: "/private/source", Context: "r"}}}
		c.ClusterBindings = []clusterBinding{{ID: "home", Connections: []connectionRef{{Direct: "home"}, {Rancher: "r", ClusterID: "local"}}, Primary: connectionRef{Direct: "home"}}}
		return c
	}
	for name, mutate := range map[string]func(*inventoryConfig){"duplicate-source": func(c *inventoryConfig) { c.Ranchers = append(c.Ranchers, c.Ranchers[0]) }, "duplicate-canonical": func(c *inventoryConfig) { c.ClusterBindings = append(c.ClusterBindings, c.ClusterBindings[0]) }, "malformed": func(c *inventoryConfig) { c.ClusterBindings[0].Connections[0].Rancher = "r" }, "unknown-source": func(c *inventoryConfig) { c.ClusterBindings[0].Connections[1].Rancher = "missing" }, "excluded": func(c *inventoryConfig) { c.Ranchers[0].ExcludeClusterIDs = []string{"local"} }, "outside-primary": func(c *inventoryConfig) {
		c.ClusterBindings[0].Primary = connectionRef{Rancher: "r", ClusterID: "other"}
	}, "migration": func(c *inventoryConfig) { c.ClusterBindings[0].ID = "new-id" }, "assigned-twice": func(c *inventoryConfig) {
		c.ClusterBindings[0].Connections = append(c.ClusterBindings[0].Connections, connectionRef{Direct: "home"})
	}} {
		t.Run(name, func(t *testing.T) {
			c := base()
			mutate(&c)
			if c.validate() == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
	c := base()
	if err := c.validate(); err != nil {
		t.Fatal(err)
	}
	registry, err := c.resolveRegistry(nil)
	if err != nil || len(registry) != 1 || registry[0].Connections[1].Available {
		t.Fatal("undiscovered binding rejected")
	}
	c.Clusters = nil
	c.ClusterBindings = nil
	if err := c.validate(); err != nil {
		t.Fatal("Rancher-only rejected")
	}
	registry, err = c.resolveRegistry(nil)
	if err != nil || len(registry) != 0 {
		t.Fatal("implicit fallback")
	}
	a := connectionRef{Rancher: "a:b", ClusterID: "c"}
	b := connectionRef{Rancher: "a", ClusterID: "b-c"}
	if a.canonicalID() == b.canonicalID() {
		t.Fatal("ambiguous identity")
	}
}
func TestIdentityProbeDeniedAndCloneEvidence(t *testing.T) {
	k := fake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system", UID: types.UID("cloneable-marker")}})
	clients := targetClients{Kube: k}
	a, err := probeIdentity(context.Background(), clients)
	if err != nil || a == "" {
		t.Fatal(err)
	}
	b, _ := probeIdentity(context.Background(), clients)
	if a != b {
		t.Fatal("same marker not stable")
	}
	_ = k.CoreV1().Namespaces().Delete(context.Background(), "kube-system", metav1.DeleteOptions{})
	if _, err := probeIdentity(context.Background(), clients); err == nil {
		t.Fatal("missing marker accepted")
	}
	if safeCollectionIssue(apierrors.NewForbidden(schema.GroupResource{Resource: "namespaces"}, "kube-system", errors.New("private"))) != "access forbidden" {
		t.Fatal("denial not sanitized")
	}
}

// Keep source fixtures injectable without persisting or manufacturing credentials.
func sourceConfig(t *testing.T) inventoryConfig {
	c := fixtureConfig(t)
	c.Clusters = c.Clusters[:1]
	c.Ranchers = []rancherConfig{{ID: "r", Kubeconfig: kubeconfigSource{Path: filepath.Join(t.TempDir(), "absent"), Context: "r"}}}
	if err := c.validate(); err != nil {
		t.Fatal(err)
	}
	return c
}
func setupRegistryService(t *testing.T, c inventoryConfig) *inventoryService {
	s := newInventoryService(c)
	s.connect = func(targetConfig) (targetClients, error) { return targetClients{EndpointID: "direct"}, nil }
	s.probe = func(context.Context, targetClients) (string, error) { return "same-marker", nil }
	s.collect = func(_ context.Context, target targetConfig, _ targetClients) map[string]collectionResult {
		out := map[string]collectionResult{}
		for _, key := range []string{"ingresses", "httproutes", "services", "workloads", "filtered"} {
			out[key] = collectionResult{Data: []map[string]any{{"clusterId": target.ID, "name": "same", "kind": "Service", "labels": map[string]string{publishLabel: "true"}}}}
		}
		return out
	}
	return s
}
func sourceResult(ids ...string) func(context.Context, rancherConfig) ([]rancherMember, *rest.Config, error) {
	return func(context.Context, rancherConfig) ([]rancherMember, *rest.Config, error) {
		members := []rancherMember{}
		for _, id := range ids {
			members = append(members, rancherMember{ID: id, Name: id})
		}
		return members, &rest.Config{Host: "https://fixture.invalid", BearerToken: "PRIVATE-TOKEN"}, nil
	}
}
func TestDynamicMembershipFailureEmptyAndRecovery(t *testing.T) {
	c := sourceConfig(t)
	s := setupRegistryService(t, c)
	s.enumerate = sourceResult("local", "c-one")
	s.refresh("test")
	if len(s.snapshot.Targets) != 3 || len(s.snapshot.Sources["r"].Members) != 2 {
		t.Fatal("members not added")
	}
	s.enumerate = func(context.Context, rancherConfig) ([]rancherMember, *rest.Config, error) {
		return nil, nil, errors.New("failure PRIVATE-TOKEN /private/source")
	}
	s.refresh("test")
	id := (connectionRef{Rancher: "r", ClusterID: "c-one"}).canonicalID()
	part := s.snapshot.Targets[id].Parts["services"]
	if len(part.Data) != 1 || part.Error == "" || s.snapshot.Targets["a"].Parts["services"].Error != "" {
		t.Fatal("outage did not isolate source")
	}
	restored := setupRegistryService(t, c)
	if len(restored.snapshot.Targets) != 3 || restored.snapshot.Sources["r"].Error == "" {
		t.Fatal("outage restart lost membership")
	}
	s.enumerate = sourceResult()
	s.refresh("test")
	if len(s.snapshot.Targets) != 1 || len(s.snapshot.Sources["r"].Members) != 0 {
		t.Fatal("empty listing retained members")
	}
	s.enumerate = sourceResult("c-new")
	s.refresh("test")
	if len(s.snapshot.Targets) != 2 {
		t.Fatal("new member not added")
	}
	c.Ranchers = nil
	removed := newInventoryService(c)
	if len(removed.snapshot.Targets) != 1 {
		t.Fatal("removed source resurrected")
	}
	b, _ := os.ReadFile(s.config.Cache.Path)
	status, _ := json.Marshal(s.status())
	for _, secret := range []string{"PRIVATE-TOKEN", "/private/source", c.Cache.Path, s.config.Ranchers[0].Kubeconfig.Path, "fixture.invalid"} {
		if strings.Contains(string(b), secret) || strings.Contains(string(status), secret) {
			t.Fatal("private metadata leaked")
		}
	}
}
func TestBindingPrimaryOnlyConflictAndPrimarySwitch(t *testing.T) {
	c := sourceConfig(t)
	c.ClusterBindings = []clusterBinding{{ID: "a", Connections: []connectionRef{{Direct: "a"}, {Rancher: "r", ClusterID: "local"}}, Primary: connectionRef{Direct: "a"}}}
	if err := c.validate(); err != nil {
		t.Fatal(err)
	}
	s := setupRegistryService(t, c)
	s.enumerate = sourceResult("local")
	var collections int
	original := s.collect
	s.collect = func(ctx context.Context, t targetConfig, clients targetClients) map[string]collectionResult {
		collections++
		return original(ctx, t, clients)
	}
	s.probe = func(_ context.Context, clients targetClients) (string, error) {
		if clients.EndpointID == "direct" {
			return "direct-marker", nil
		}
		return "proxy-marker", nil
	}
	s.refresh("test")
	if collections != 1 || len(s.snapshot.Targets) != 1 || !strings.Contains(strings.Join(s.snapshot.Registry[0].Warnings, " "), "conflict") {
		t.Fatal("binding merged or double collected")
	}
	s.connect = func(targetConfig) (targetClients, error) {
		return targetClients{}, errors.New("credentials unavailable")
	}
	s.refresh("test")
	if collections != 1 || s.snapshot.Targets["a"].Parts["services"].Error == "" {
		t.Fatal("secondary failed over")
	}
	c.ClusterBindings[0].Primary = connectionRef{Rancher: "r", ClusterID: "local"}
	switched := setupRegistryService(t, c)
	if len(switched.snapshot.Targets) != 0 {
		t.Fatal("previous-primary inventory restored")
	}
	switched.enumerate = sourceResult("local")
	switched.refresh("test")
	if len(switched.snapshot.Targets) != 1 {
		t.Fatal("canonical ID changed")
	}
	switched.enumerate = sourceResult()
	switched.refresh("test")
	if switched.snapshot.Targets["a"].Parts["services"].Error == "" {
		t.Fatal("missing primary not reported")
	}
}
func TestOverlapCurrentEvidenceAndIdentityReplacement(t *testing.T) {
	c := sourceConfig(t)
	s := setupRegistryService(t, c)
	s.enumerate = sourceResult("local")
	s.refresh("test")
	if len(s.snapshot.Registry) != 2 || !strings.Contains(strings.Join(s.snapshot.Registry[0].Warnings, " "), "Possible overlap") {
		t.Fatal("no duplicate warning")
	}
	s.probe = func(context.Context, targetClients) (string, error) { return "", errors.New("denied") }
	s.refresh("test")
	if strings.Contains(strings.Join(s.snapshot.Registry[0].Warnings, " "), "Possible overlap") {
		t.Fatal("stale observations compared")
	}
	s.probe = func(context.Context, targetClients) (string, error) { return "replacement", nil }
	s.collect = func(context.Context, targetConfig, targetClients) map[string]collectionResult {
		return map[string]collectionResult{"services": {Err: errors.New("denied")}}
	}
	s.refresh("test")
	if len(s.snapshot.Targets["a"].Parts["services"].Data) != 0 {
		t.Fatal("replacement retained previous resources")
	}
}
func TestRegistryReadersSeeOneGeneration(t *testing.T) {
	c := sourceConfig(t)
	s := setupRegistryService(t, c)
	s.enumerate = sourceResult("local")
	s.refresh("test")
	entered, release := make(chan struct{}), make(chan struct{})
	s.enumerate = func(context.Context, rancherConfig) ([]rancherMember, *rest.Config, error) {
		close(entered)
		<-release
		return sourceResult("c-new")(context.Background(), c.Ranchers[0])
	}
	s.request(nil, true)
	<-entered
	for i := 0; i < 20; i++ {
		status := s.status()
		if status["generation"].(uint64) != 1 || len(status["clusters"].([]map[string]any)) != 2 {
			t.Fatal("partial registry published")
		}
		w := httptest.NewRecorder()
		s.dataHandler("services")(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
	}
	close(release)
	waitInventory(t, s)
	if s.snapshot.Generation != 2 {
		t.Fatal("generation not committed")
	}
}

func TestRancherImportedAndProvisionedFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/rancher/clusters.json")
	if err != nil {
		t.Fatal(err)
	}
	src, _ := rancherFixture(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(body) })
	members, _, err := enumerateRancher(context.Background(), src)
	if err != nil || len(members) != 3 {
		t.Fatal("coverage fixture rejected", err)
	}
	for _, m := range members {
		if m.ID == "different-provisioning-name" {
			t.Fatal("provisioning name used as management ID")
		}
	}
}
func TestIdentityDirectAndProxyWithDeniedProbe(t *testing.T) {
	var deny atomic.Bool
	var redirect atomic.Bool
	src, _ := rancherFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if redirect.Load() {
			http.Redirect(w, r, "/other", 302)
			return
		}
		if deny.Load() {
			w.WriteHeader(403)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v1/namespaces/kube-system" && r.URL.Path != "/k8s/clusters/local/api/v1/namespaces/kube-system" {
			w.WriteHeader(404)
			return
		}
		fmt.Fprint(w, `{"apiVersion":"v1","kind":"Namespace","metadata":{"name":"kube-system","uid":"shared-direct-proxy-marker"}}`)
	})
	direct, err := connectTarget(targetConfig{ID: "direct", Kubeconfig: &src.Kubeconfig})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := loadRancher(src)
	if err != nil {
		t.Fatal(err)
	}
	proxy, err := proxyClients(cfg, "local")
	if err != nil {
		t.Fatal(err)
	}
	a, ea := probeIdentity(context.Background(), direct)
	b, eb := probeIdentity(context.Background(), proxy)
	if ea != nil || eb != nil || a == "" || a != b {
		t.Fatal("direct/proxy signals differ", ea, eb)
	}
	deny.Store(true)
	if _, err := probeIdentity(context.Background(), proxy); err == nil {
		t.Fatal("denied probe accepted")
	}
	// Authenticated proxy redirects must never be followed, even to another path
	// on the same host (which can otherwise bypass the intended route).
	deny.Store(false)
	redirect.Store(true)
	if _, err := probeIdentity(context.Background(), proxy); err == nil {
		t.Fatal("proxy redirect accepted")
	}
}
func TestSourceOverflowRetainsWholeMembership(t *testing.T) {
	c := sourceConfig(t)
	c.Ranchers = append(c.Ranchers, rancherConfig{ID: "s", Kubeconfig: kubeconfigSource{Path: "/absent", Context: "s"}})
	s := setupRegistryService(t, c)
	s.enumerate = func(_ context.Context, r rancherConfig) ([]rancherMember, *rest.Config, error) {
		ids := []string{}
		for i := 0; i < 100; i++ {
			ids = append(ids, fmt.Sprintf("c-%d", i))
		}
		return sourceResult(ids...)(context.Background(), r)
	}
	s.refresh("test")
	if len(s.snapshot.Sources["r"].Members) != 100 || len(s.snapshot.Sources["s"].Members) != 0 || s.snapshot.Sources["s"].Error == "" || len(s.snapshot.Targets) != 101 {
		t.Fatal("aggregate overflow silently truncated or dropped direct target")
	}
}
func TestCacheExclusionsTrustAndBindings(t *testing.T) {
	c := sourceConfig(t)
	s := setupRegistryService(t, c)
	s.enumerate = sourceResult("local")
	s.refresh("test")
	for name, mutate := range map[string]func(*inventoryConfig){"excluded": func(c *inventoryConfig) { c.Ranchers[0].ExcludeClusterIDs = []string{"local"} }, "context": func(c *inventoryConfig) { c.Ranchers[0].Kubeconfig.Context = "changed" }, "binding": func(c *inventoryConfig) {
		c.ClusterBindings = []clusterBinding{{ID: "a", Connections: []connectionRef{{Direct: "a"}, {Rancher: "r", ClusterID: "local"}}, Primary: connectionRef{Direct: "a"}}}
	}} {
		t.Run(name, func(t *testing.T) {
			copy := c
			copy.Ranchers = append([]rancherConfig{}, c.Ranchers...)
			mutate(&copy)
			if err := copy.validate(); err != nil {
				t.Fatal(err)
			}
			restored := newInventoryService(copy)
			if len(restored.snapshot.Targets) > 1 {
				t.Fatal("incompatible dynamic target restored")
			}
		})
	}
	state := s.snapshot.Sources["r"]
	state.Members = append(state.Members, state.Members[0])
	if restoredSourceValid(c.Ranchers[0], state) {
		t.Fatal("duplicate saved member accepted")
	}
}
func TestRancherOnlyEmptyHTTP(t *testing.T) {
	c := sourceConfig(t)
	c.Clusters = nil
	s := setupRegistryService(t, c)
	s.enumerate = sourceResult()
	s.refresh("test")
	for _, kind := range []string{"directory", "workloads", "filtered"} {
		w := httptest.NewRecorder()
		s.dataHandler(kind)(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
			t.Fatal("empty inventory not successful", w.Code, w.Body.String())
		}
	}
}

func TestRancherClassificationInvalidatesDynamicCache(t *testing.T) {
	c := sourceConfig(t)
	s := setupRegistryService(t, c)
	s.enumerate = sourceResult("local")
	s.refresh("test")
	c.Classification = map[string]classificationRule{"environment": {LabelKeys: []string{"changed"}, Sources: []string{"resource"}, DisplayCase: "lower"}}
	if err := c.validate(); err != nil {
		t.Fatal(err)
	}
	restored := newInventoryService(c)
	if len(restored.snapshot.Targets) != 0 {
		t.Fatal("classification change restored incompatible rows")
	}
}

// Optional read-only release qualification. Only an explicitly supplied path is
// read; normal tests never inspect workstation credentials or contact Rancher.
func TestRancherLiveContract(t *testing.T) {
	path := os.Getenv("AUTODOCS_TEST_RANCHER_KUBECONFIG")
	if path == "" {
		t.Skip("explicit live Rancher kubeconfig not supplied")
	}
	src := rancherConfig{ID: "verification", Kubeconfig: kubeconfigSource{Path: path, Context: "rancher"}}
	ctx, cancel := context.WithTimeout(context.Background(), sourceTimeout)
	defer cancel()
	members, cfg, err := enumerateRancher(ctx, src)
	if err != nil {
		t.Fatal(safeSourceIssue(err))
	}
	t.Logf("complete RK-API listing: %d clusters", len(members))
	if len(members) == 0 {
		return
	}
	probeCtx, probeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer probeCancel()
	clients, err := proxyClients(cfg, members[0].ID)
	if err != nil {
		t.Fatal("cannot construct proxy client")
	}
	if _, err := probeIdentity(probeCtx, clients); err != nil {
		t.Fatal("proxy namespace identity probe unavailable")
	}
	t.Log("verified TLS, bearer authentication and proxy identity probe succeeded")
}

func BenchmarkRancherBudget(b *testing.B) {
	src, _ := rancherFixture(b, func(w http.ResponseWriter, r *http.Request) {
		start := 0
		if r.URL.Query().Get("continue") == "page2" {
			start = 50
		}
		items := []any{}
		for i := start; i < start+50; i++ {
			items = append(items, map[string]any{"metadata": map[string]any{"name": fmt.Sprintf("c-%d", i)}, "spec": map[string]any{"displayName": "Representative cluster", "unused": strings.Repeat("x", 20000)}})
		}
		next := "page2"
		if start == 50 {
			next = ""
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"apiVersion": "management.cattle.io/v3", "kind": "ClusterList", "metadata": map[string]any{"continue": next}, "items": items})
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		members, _, err := enumerateRancher(context.Background(), src)
		if err != nil || len(members) != 100 {
			b.Fatal("representative enumeration failed")
		}
	}
}

func TestFailedEmptyDiscoveryIsNotSuccessfulEmptyInventory(t *testing.T) {
	c := sourceConfig(t)
	c.Clusters = nil
	s := setupRegistryService(t, c)
	s.enumerate = func(context.Context, rancherConfig) ([]rancherMember, *rest.Config, error) {
		return nil, nil, errors.New("credentials unavailable")
	}
	s.refresh("test")
	w := httptest.NewRecorder()
	s.dataHandler("directory")(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 503 {
		t.Fatal("failed initial discovery advertised a successful empty inventory")
	}
}
func TestRancherFailureDisclosure(t *testing.T) {
	c := sourceConfig(t)
	capture := &logCapture{}
	s := setupRegistryService(t, c)
	s.logger = slog.New(slog.NewJSONHandler(capture, &slog.HandlerOptions{Level: slog.LevelDebug}))
	s.enumerate = sourceResult("local")
	s.refresh("test")
	s.enumerate = func(context.Context, rancherConfig) ([]rancherMember, *rest.Config, error) {
		return nil, nil, errors.New("DO-NOT-PERSIST-THIS-TOKEN " + c.Ranchers[0].Kubeconfig.Path + " raw kubeconfig private data")
	}
	s.refresh("test")
	saved, _ := os.ReadFile(c.Cache.Path)
	w := httptest.NewRecorder()
	s.statusHandler(w, httptest.NewRequest("GET", "/", nil))
	for _, output := range []string{capture.output(), string(saved), w.Body.String()} {
		for _, secret := range []string{"DO-NOT-PERSIST-THIS-TOKEN", c.Ranchers[0].Kubeconfig.Path, "raw kubeconfig private data", "fixture.invalid"} {
			if strings.Contains(output, secret) {
				t.Fatal("credential metadata leaked")
			}
		}
	}
}

func TestRefreshCredentialVersionIsCoherent(t *testing.T) {
	var expected atomic.Value
	expected.Store("DO-NOT-PERSIST-THIS-TOKEN")
	src, _ := rancherFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+expected.Load().(string) {
			w.WriteHeader(401)
			return
		}
		if r.URL.Path == "/apis/management.cattle.io/v3/clusters" {
			listJSON(w, []string{"local"}, "")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"apiVersion":"v1","kind":"Namespace","metadata":{"name":"kube-system","uid":"marker"}}`)
	})
	_, cfg, err := enumerateRancher(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(src.Kubeconfig.Path)
	_ = os.WriteFile(src.Kubeconfig.Path, []byte(strings.Replace(string(b), "DO-NOT-PERSIST-THIS-TOKEN", "rotated", 1)), 0600)
	clients, err := proxyClients(cfg, "local")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := probeIdentity(context.Background(), clients); err != nil {
		t.Fatal("same refresh did not retain coherent credential version")
	}
	expected.Store("rotated")
	_, cfg, err = enumerateRancher(context.Background(), src)
	if err != nil {
		t.Fatal("next refresh did not reload credential")
	}
	clients, err = proxyClients(cfg, "local")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := probeIdentity(context.Background(), clients); err != nil {
		t.Fatal("next proxy did not use rotated credential")
	}
}

func TestUndiscoveredRancherOnlyBindingRemainsVisible(t *testing.T) {
	c := sourceConfig(t)
	c.Clusters = nil
	c.ClusterBindings = []clusterBinding{{ID: "canonical", Connections: []connectionRef{{Rancher: "r", ClusterID: "not-yet-visible"}}, Primary: connectionRef{Rancher: "r", ClusterID: "not-yet-visible"}}}
	if err := c.validate(); err != nil {
		t.Fatal(err)
	}
	s := setupRegistryService(t, c)
	s.enumerate = sourceResult()
	s.refresh("test")
	targets := s.status()["clusters"].([]map[string]any)
	if len(targets) != 1 || targets[0]["id"] != "canonical" || s.snapshot.Registry[0].Connections[0].Available || s.snapshot.Targets["canonical"].Parts["services"].Error == "" {
		t.Fatal("configured absent primary hidden or invented")
	}
}

func TestSourceCacheTrustCompatibility(t *testing.T) {
	src, _ := rancherFixture(t, func(w http.ResponseWriter, r *http.Request) { listJSON(w, []string{"local"}, "") })
	cfg, err := loadRancher(src)
	if err != nil {
		t.Fatal(err)
	}
	state := rancherSnapshot{ConfigID: sourceFingerprint(src), EndpointID: sourceEndpoint(cfg), Members: []rancherMember{{ID: "local", Name: "Local"}}}
	if !restoredSourceValid(src, state) {
		t.Fatal("compatible source rejected")
	}
	original, _ := os.ReadFile(src.Kubeconfig.Path)
	rotated := strings.Replace(string(original), "DO-NOT-PERSIST-THIS-TOKEN", "ordinary-rotation", 1)
	_ = os.WriteFile(src.Kubeconfig.Path, []byte(rotated), 0600)
	if !restoredSourceValid(src, state) {
		t.Fatal("token rotation changed cache compatibility")
	}
	changed := strings.Replace(rotated, cfg.Host, "https://changed.example.invalid", 1)
	_ = os.WriteFile(src.Kubeconfig.Path, []byte(changed), 0600)
	if restoredSourceValid(src, state) {
		t.Fatal("changed origin resurrected incompatible membership")
	}
}
