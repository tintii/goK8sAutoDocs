package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestHomeConfigDefaultsAndMountedUpdates(t *testing.T) {
	t.Setenv("HOMEPAGE_CONFIG", "")
	w := httptest.NewRecorder()
	homeConfigHandler(w, httptest.NewRequest("GET", "/api/v1/home", nil))
	var config homeConfig
	if err := json.Unmarshal(w.Body.Bytes(), &config); err != nil || w.Code != 200 || config.Title != "Cluster overview" || config.Links == nil {
		t.Fatalf("defaults: %d %s %v", w.Code, w.Body, err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "home.yaml")
	t.Setenv("HOMEPAGE_CONFIG", path)
	for _, message := range []string{"Maintenance tonight.", "Maintenance complete. <script>alert(1)</script>"} {
		// Replace the file, like Kubernetes replacing a projected ConfigMap.
		tmp := filepath.Join(dir, "next.yaml")
		if err := os.WriteFile(tmp, []byte("motd: "+message+"\nlinks:\n  - label: Docs\n    url: /docs\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(tmp, path); err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		homeConfigHandler(w, httptest.NewRequest("GET", "/api/v1/home", nil))
		if err := json.Unmarshal(w.Body.Bytes(), &config); err != nil || config.MOTD != message || config.Title != "Cluster overview" || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("updated config: %s %v", w.Body, err)
		}
	}
}

func TestHomeConfigRejectsInvalidInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "home.yaml")
	t.Setenv("HOMEPAGE_CONFIG", path)
	for name, input := range map[string]string{
		"unknown field":      "titlle: Typo",
		"malformed":          "motd: [broken",
		"blank title":        "title: ' '",
		"multiple documents": "motd: first\n---\nmotd: second",
		"large":              "motd: " + strings.Repeat("x", 65536),
		"script URL":         "links: [{label: X, url: 'javascript:alert(1)'}]",
		"protocol relative":  "links: [{label: X, url: '//example.com'}]",
		"credentials":        "links: [{label: X, url: 'https://user:pass@example.com'}]",
		"blank label":        "links: [{label: ' ', url: '/docs'}]",
		"empty file":         "",
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(input), 0600); err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			homeConfigHandler(w, httptest.NewRequest("GET", "/api/v1/home", nil))
			if w.Code != 500 || strings.Contains(w.Body.String(), path) {
				t.Fatalf("expected safe configuration error, got %d %s", w.Code, w.Body)
			}
		})
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := loadHomeConfig(path); err == nil {
		t.Fatal("missing configured file must report an error")
	}
	for _, raw := range []string{"https://example.com/path?q=hello#section", "http://example.com:8080", "/docs"} {
		if !safeHomeURL(raw) {
			t.Errorf("valid link rejected: %s", raw)
		}
	}
	for _, raw := range []string{"/\\example.com", "https://example.com\n", "data:text/html,hello", "docs", "https:///missing-host"} {
		if safeHomeURL(raw) {
			t.Errorf("unsafe link accepted: %q", raw)
		}
	}
}

func TestServiceStatusAvailabilityAndInactiveWorkloads(t *testing.T) {
	two, zero := int32(2), int32(0)
	client := fake.NewSimpleClientset(
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "ready", Namespace: "apps", Generation: 2}, Spec: appsv1.DeploymentSpec{Replicas: &two}, Status: appsv1.DeploymentStatus{AvailableReplicas: 2, ObservedGeneration: 2}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "pending-generation", Namespace: "apps", Generation: 3}, Spec: appsv1.DeploymentSpec{Replicas: &two}, Status: appsv1.DeploymentStatus{AvailableReplicas: 2, ObservedGeneration: 2}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "scaled-down", Namespace: "media"}, Spec: appsv1.DeploymentSpec{Replicas: &zero}},
		&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "database", Namespace: "data"}, Spec: appsv1.StatefulSetSpec{Replicas: &two}, Status: appsv1.StatefulSetStatus{AvailableReplicas: 1}},
		&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "system"}, Status: appsv1.DaemonSetStatus{DesiredNumberScheduled: 2, NumberAvailable: 2}},
		&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "no-nodes", Namespace: "system"}},
	)
	result, err := collectServiceStatus(context.Background(), client)
	if err != nil || result.Ready != 2 || result.Attention != 2 || result.Inactive != 2 || len(result.Workloads) != 6 || result.CheckedAt.IsZero() {
		t.Fatalf("incorrect availability: %+v %v", result, err)
	}
	if result.Workloads[0].Name != "pending-generation" || result.Workloads[1].Name != "database" || result.Workloads[5].State != "inactive" {
		t.Fatalf("attention must be first, with namespace ordering: %+v", result.Workloads)
	}
	for _, action := range client.Actions() {
		if action.GetNamespace() != "" || action.GetVerb() != "list" {
			t.Fatalf("status must only list cluster-wide: %+v", action)
		}
	}
}

func TestServiceStatusEmptyAndCollectionFailures(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := serviceStatusHandler(client)
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest("GET", "/api/v1/service-status", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"workloads":[]`) {
		t.Fatalf("empty result: %d %s", w.Code, w.Body)
	}
	for _, resource := range []string{"deployments", "statefulsets", "daemonsets"} {
		t.Run(resource, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			client.PrependReactor("list", resource, func(ktesting.Action) (bool, runtime.Object, error) {
				return true, nil, fmt.Errorf("forbidden")
			})
			handler := serviceStatusHandler(client)
			w := httptest.NewRecorder()
			handler(w, httptest.NewRequest("GET", "/api/v1/service-status", nil))
			if w.Code != 503 || strings.Contains(w.Body.String(), `"ready"`) || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("partial lists must not report healthy: %d %s", w.Code, w.Body)
			}
			client.ReactionChain = nil
			client.AddReactor("*", "*", ktesting.ObjectReaction(client.Tracker()))
			w = httptest.NewRecorder()
			handler(w, httptest.NewRequest("GET", "/api/v1/service-status", nil))
			if w.Code != 200 {
				t.Fatalf("failed status must not be cached: %d %s", w.Code, w.Body)
			}
		})
	}
}
