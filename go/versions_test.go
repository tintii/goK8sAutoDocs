package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func versionsFixture() *fake.Clientset {
	template := corev1.PodTemplateSpec{Spec: corev1.PodSpec{
		Containers:     []corev1.Container{{Name: "app", Image: "example/app:1.2"}},
		InitContainers: []corev1.Container{{Name: "setup", Image: "example/setup:2.0"}},
	}}
	return fake.NewSimpleClientset(
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "apps"}, Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}}, Template: template,
		}},
		&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "data"}, Spec: appsv1.StatefulSetSpec{Template: template}},
		&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "system"}, Spec: appsv1.DaemonSetSpec{Template: template}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "web-pod", Namespace: "apps", Labels: map[string]string{"app": "web"}}, Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{Name: "app", ImageID: "example/app@sha256:abc123"}},
		}},
	)
}

func TestVersionsDashboardAcrossNamespaces(t *testing.T) {
	client := versionsFixture()
	response := httptest.NewRecorder()
	versionsDashboardHandler(client)(response, httptest.NewRequest(http.MethodGet, "/api/v1/versions-dashboard", nil))
	var resources []struct {
		Name, Namespace, Kind string
		Containers            []map[string]string
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &resources) != nil {
		t.Fatalf("response: %d %s", response.Code, response.Body.String())
	}
	if len(resources) != 3 {
		t.Fatalf("expected workloads from three namespaces, got %s", response.Body.String())
	}
	for i, want := range []struct{ name, namespace, kind string }{{"web", "apps", "Deployment"}, {"db", "data", "StatefulSet"}, {"agent", "system", "DaemonSet"}} {
		got := resources[i]
		if got.Name != want.name || got.Namespace != want.namespace || got.Kind != want.kind {
			t.Errorf("resource %d: %+v", i, got)
		}
		if len(got.Containers) != 2 || got.Containers[0]["tag"] != "1.2" || got.Containers[1]["type"] != "init" {
			t.Errorf("container inventory: %+v", got.Containers)
		}
	}
	if resources[0].Containers[0]["sha"] != "abc123" {
		t.Errorf("running deployment digest missing: %+v", resources[0].Containers)
	}
	for _, action := range client.Actions() {
		if action.GetResource().Group == "apps" && action.GetNamespace() != metav1.NamespaceAll {
			t.Errorf("workload discovery restricted to namespace %q", action.GetNamespace())
		}
	}
}

func TestVersionsDashboardEmptyArray(t *testing.T) {
	response := httptest.NewRecorder()
	versionsDashboardHandler(fake.NewSimpleClientset())(response, httptest.NewRequest(http.MethodGet, "/versions", nil))
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != "[]" {
		t.Fatalf("expected empty array, got %d %s", response.Code, response.Body.String())
	}
}

func TestVersionsDashboardCollectionFailuresAreNotCached(t *testing.T) {
	for _, resource := range []string{"deployments", "statefulsets", "daemonsets", "pods"} {
		t.Run(resource, func(t *testing.T) {
			client := versionsFixture()
			fail := true
			client.PrependReactor("list", resource, func(clienttesting.Action) (bool, runtime.Object, error) {
				if fail {
					return true, nil, fmt.Errorf("forbidden")
				}
				return false, nil, nil
			})
			handler := versionsDashboardHandler(client)
			response := httptest.NewRecorder()
			handler(response, httptest.NewRequest(http.MethodGet, "/versions", nil))
			if response.Code != http.StatusInternalServerError {
				t.Fatalf("collection failure hidden: %d %s", response.Code, response.Body.String())
			}
			fail = false
			response = httptest.NewRecorder()
			handler(response, httptest.NewRequest(http.MethodGet, "/versions", nil))
			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"web"`) {
				t.Fatalf("recovery failed: %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestVersionsDashboardConcurrentCache(t *testing.T) {
	client := versionsFixture()
	handler := versionsDashboardHandler(client)
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response := httptest.NewRecorder()
			handler(response, httptest.NewRequest(http.MethodGet, "/versions", nil))
			if response.Code != http.StatusOK {
				t.Errorf("response: %d", response.Code)
			}
		}()
	}
	wg.Wait()
	if len(client.Actions()) != 4 { // Three workload lists and one deployment pod list.
		t.Fatalf("expected one cache load, got %d API calls", len(client.Actions()))
	}
}
