package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	ktesting "k8s.io/client-go/testing"
)

func object(t *testing.T, data string) *unstructured.Unstructured {
	t.Helper()
	var result unstructured.Unstructured
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		t.Fatal(err)
	}
	return &result
}

func routeFixture(t *testing.T) *unstructured.Unstructured {
	return object(t, `{"apiVersion":"gateway.networking.k8s.io/v1","kind":"HTTPRoute","metadata":{"name":"app","namespace":"media","generation":2,"labels":{"k8sautodocs.io/publish":"true"}},"spec":{"hostnames":["app.example.com"],"parentRefs":[{"name":"shared","namespace":"edge","sectionName":"https"}]}}`)
}

func gatewayFixture(t *testing.T) *unstructured.Unstructured {
	return object(t, `{"apiVersion":"gateway.networking.k8s.io/v1","kind":"Gateway","metadata":{"name":"shared","namespace":"edge","generation":1},"spec":{"listeners":[{"name":"https","hostname":"*.example.com","protocol":"HTTPS","port":443,"allowedRoutes":{"namespaces":{"from":"All"}}}]}}`)
}

func dynamicFixture(t *testing.T, objects ...*unstructured.Unstructured) *dynamicfake.FakeDynamicClient {
	t.Helper()
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme.Scheme,
		map[schema.GroupVersionResource]string{httpRouteGVR: "HTTPRouteList", gatewayGVR: "GatewayList"})
	for _, object := range objects {
		gvr := httpRouteGVR
		if object.GetKind() == "Gateway" {
			gvr = gatewayGVR
		}
		// Register the actual plural explicitly: the fake's heuristic guesses "gatewaies".
		if err := client.Tracker().Create(gvr, object, object.GetNamespace()); err != nil {
			t.Fatal(err)
		}
	}
	return client
}

func setField(t *testing.T, item *unstructured.Unstructured, value any, fields ...string) {
	t.Helper()
	if err := unstructured.SetNestedField(item.Object, value, fields...); err != nil {
		t.Fatal(err)
	}
}

func collectOne(t *testing.T, route, gateway *unstructured.Unstructured) directoryEntry {
	t.Helper()
	entries, warnings, err := collectAutoDocs(context.Background(), fake.NewSimpleClientset(), dynamicFixture(t, route, gateway), "")
	if err != nil || len(warnings) != 0 || len(entries) != 1 {
		t.Fatalf("entries=%v warnings=%v err=%v", entries, warnings, err)
	}
	return entries[0]
}

func TestPublicationLabels(t *testing.T) {
	for _, value := range []string{"true", "false", "TRUE", "", "missing"} {
		t.Run(value, func(t *testing.T) {
			metadata := metav1.ObjectMeta{Name: "ingress", Namespace: "media", Annotations: map[string]string{
				publishLabel: "true", "k8sautodocs.io/fqdn-override": "ignored.example.net",
			}}
			metadata.Labels = map[string]string{publishLabel: value, "k8sautodocs.io/fqdn-override": "ignored.example.net"}
			if value == "missing" {
				delete(metadata.Labels, publishLabel)
			}
			ingress := &networkingv1.Ingress{ObjectMeta: metadata, Spec: networkingv1.IngressSpec{Rules: []networkingv1.IngressRule{{Host: "original.example.com"}}}}
			route, gateway := routeFixture(t), gatewayFixture(t)
			route.SetLabels(metadata.Labels)
			route.SetAnnotations(metadata.Annotations)
			client := dynamicFixture(t, route, gateway)
			entries, warnings, err := collectAutoDocs(context.Background(), fake.NewSimpleClientset(ingress), client, "")
			want := 0
			if value == "true" {
				want = 2
			}
			if err != nil || len(warnings) > 0 || len(entries) != want {
				t.Fatalf("entries=%v warnings=%v err=%v, want %d", entries, warnings, err, want)
			}
			for _, entry := range entries {
				for _, endpoint := range entry.Endpoints {
					if strings.Contains(endpoint.URL, "ignored") {
						t.Fatalf("legacy override affected URL: %s", endpoint.URL)
					}
				}
			}
			filtered, _, err := collectFilteredResources(context.Background(), client, "", []GVRResource{{Group: httpRouteGVR.Group, Version: "v1", Resource: "httproutes"}})
			if err != nil || len(filtered) != want/2 {
				t.Fatalf("filtered=%v err=%v", filtered, err)
			}
			for _, action := range client.Actions() {
				if list, ok := action.(ktesting.ListAction); ok && list.GetListRestrictions().Labels.String() != publishSelector {
					t.Fatalf("missing server-side selector: %v", action)
				}
			}
		})
	}
	t.Run("Gateway label does not cascade", func(t *testing.T) {
		route, gateway := routeFixture(t), gatewayFixture(t)
		route.SetLabels(nil)
		gateway.SetLabels(map[string]string{publishLabel: "true"})
		entries, _, _ := collectAutoDocs(context.Background(), fake.NewSimpleClientset(), dynamicFixture(t, route, gateway), "")
		if len(entries) != 0 {
			t.Fatalf("unlabelled route included: %v", entries)
		}
	})
}

func TestIngressDestinations(t *testing.T) {
	ingress := networkingv1.Ingress{Spec: networkingv1.IngressSpec{
		Rules: []networkingv1.IngressRule{{Host: "plain.example.net"}, {Host: "secure.example.com"}, {Host: "secure.example.com"}, {Host: "deep.sub.example.com"}, {Host: "*.example.com"}, {}},
		TLS:   []networkingv1.IngressTLS{{Hosts: []string{"*.example.com"}}},
	}}
	entry := ingressEntry(ingress)
	want := []string{"http://plain.example.net", "https://secure.example.com", "http://deep.sub.example.com"}
	var got []string
	for _, endpoint := range entry.Endpoints {
		got = append(got, endpoint.URL)
	}
	if !reflect.DeepEqual(got, want) || len(entry.Issues) != 2 {
		t.Fatalf("urls=%v issues=%v", got, entry.Issues)
	}
}

func TestHostnameIntersection(t *testing.T) {
	for _, test := range []struct {
		a, b, host string
		ok         bool
	}{
		{"app.example.com", "*.example.com", "app.example.com", true},
		{"deep.app.example.com", "*.example.com", "deep.app.example.com", true},
		{"*.example.com", "app.example.com", "app.example.com", true},
		{"*.com", "*.example.com", "*.example.com", true},
		{"*.example.com", "*.com", "*.example.com", true},
		{"", "app.example.com", "app.example.com", true},
		{"app.example.com", "", "app.example.com", true},
		{"example.com", "*.example.com", "", false},
		{"app.example.net", "*.example.com", "", false},
	} {
		host, ok := intersectHostname(test.a, test.b)
		if host != test.host || ok != test.ok {
			t.Errorf("intersection(%q,%q)=(%q,%v), want (%q,%v)", test.a, test.b, host, ok, test.host, test.ok)
		}
	}
}

func TestGatewayDestinations(t *testing.T) {
	for _, test := range []struct {
		name   string
		modify func(*unstructured.Unstructured, *unstructured.Unstructured)
		urls   []string
		issue  string
	}{
		{name: "HTTPS cross-namespace", urls: []string{"https://app.example.com"}},
		{name: "non-default port", modify: func(r, g *unstructured.Unstructured) {
			setField(t, g, []any{map[string]any{"name": "https", "protocol": "HTTP", "port": int64(8080), "allowedRoutes": map[string]any{"namespaces": map[string]any{"from": "All"}}}}, "spec", "listeners")
		}, urls: []string{"http://app.example.com:8080"}},
		{name: "default parent fields", modify: func(r, g *unstructured.Unstructured) {
			g.SetNamespace("media")
			setField(t, r, []any{map[string]any{"name": "shared"}}, "spec", "parentRefs")
		}, urls: []string{"https://app.example.com"}},
		{name: "both listener restrictions match", modify: func(r, g *unstructured.Unstructured) {
			setField(t, r, []any{map[string]any{"name": "shared", "namespace": "edge", "sectionName": "https", "port": int64(443)}}, "spec", "parentRefs")
		}, urls: []string{"https://app.example.com"}},
		{name: "port restriction mismatch", modify: func(r, g *unstructured.Unstructured) {
			setField(t, r, []any{map[string]any{"name": "shared", "namespace": "edge", "sectionName": "https", "port": int64(80)}}, "spec", "parentRefs")
		}, issue: "no compatible listener"},
		{name: "section mismatch", modify: func(r, g *unstructured.Unstructured) {
			setField(t, r, []any{map[string]any{"name": "shared", "namespace": "edge", "sectionName": "missing"}}, "spec", "parentRefs")
		}, issue: "no compatible listener"},
		{name: "hostname mismatch", modify: func(r, g *unstructured.Unstructured) {
			setField(t, r, []any{"other.example.net"}, "spec", "hostnames")
		}, issue: "no compatible listener"},
		{name: "wildcard only", modify: func(r, g *unstructured.Unstructured) {
			setField(t, r, []any{"*.example.com"}, "spec", "hostnames")
		}, issue: "no concrete hostname"},
		{name: "concrete listener and unspecified route host", modify: func(r, g *unstructured.Unstructured) {
			unstructured.RemoveNestedField(r.Object, "spec", "hostnames")
			setField(t, g, []any{map[string]any{"name": "https", "hostname": "listener.example.com", "protocol": "HTTPS", "port": int64(443), "allowedRoutes": map[string]any{"namespaces": map[string]any{"from": "All"}}}}, "spec", "listeners")
		}, urls: []string{"https://listener.example.com"}},
		{name: "namespace denied by default", modify: func(r, g *unstructured.Unstructured) {
			setField(t, g, []any{map[string]any{"name": "https", "protocol": "HTTPS", "port": int64(443)}}, "spec", "listeners")
		}, issue: "namespace media is not allowed"},
		{name: "missing Gateway", modify: func(r, g *unstructured.Unstructured) {
			g.SetName("different")
		}, issue: "Cannot read Gateway"},
		{name: "no parents", modify: func(r, g *unstructured.Unstructured) {
			unstructured.RemoveNestedField(r.Object, "spec", "parentRefs")
		}, issue: "no Gateway parent"},
	} {
		t.Run(test.name, func(t *testing.T) {
			route, gateway := routeFixture(t), gatewayFixture(t)
			if test.modify != nil {
				test.modify(route, gateway)
			}
			entry := collectOne(t, route, gateway)
			var urls []string
			for _, endpoint := range entry.Endpoints {
				urls = append(urls, endpoint.URL)
			}
			if !reflect.DeepEqual(urls, test.urls) {
				t.Fatalf("urls=%v want=%v issues=%v", urls, test.urls, entry.Issues)
			}
			if test.issue != "" && !strings.Contains(strings.Join(entry.Issues, " "), test.issue) {
				t.Fatalf("missing issue %q: %v", test.issue, entry.Issues)
			}
		})
	}
}

func TestMultipleListenersAndGatewayLookupReuse(t *testing.T) {
	route, gateway := routeFixture(t), gatewayFixture(t)
	setField(t, route, []any{map[string]any{"name": "shared", "namespace": "edge"}}, "spec", "parentRefs")
	listener := func(name, protocol string, port int64) any {
		return map[string]any{"name": name, "protocol": protocol, "port": port, "allowedRoutes": map[string]any{"namespaces": map[string]any{"from": "All"}}}
	}
	setField(t, gateway, []any{listener("http", "HTTP", 80), listener("https", "HTTPS", 443), listener("duplicate", "HTTPS", 443)}, "spec", "listeners")
	other := route.DeepCopy()
	other.SetName("other")
	client := dynamicFixture(t, route, other, gateway)
	entries, _, err := collectAutoDocs(context.Background(), fake.NewSimpleClientset(), client, "")
	if err != nil || len(entries) != 2 || len(entries[0].Endpoints) != 2 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	gets := 0
	for _, action := range client.Actions() {
		if action.GetVerb() == "get" && action.GetResource() == gatewayGVR {
			gets++
		}
		if action.GetVerb() != "get" && action.GetVerb() != "list" {
			t.Fatalf("unexpected write: %v", action)
		}
	}
	if gets != 1 {
		t.Fatalf("Gateway fetched %d times", gets)
	}
}

func TestAttachmentStatus(t *testing.T) {
	for _, test := range []struct {
		name, status string
		generation   int64
		parent       string
		wantURLs     int
		issue        string
	}{
		{"accepted", "True", 2, "shared", 1, ""},
		{"rejected", "False", 2, "shared", 0, "rejected"},
		{"stale rejection", "False", 1, "shared", 1, "unknown or stale"},
		{"unrelated rejection", "False", 2, "other", 1, "unknown or stale"},
	} {
		t.Run(test.name, func(t *testing.T) {
			route, gateway := routeFixture(t), gatewayFixture(t)
			setField(t, route, map[string]any{"parents": []any{map[string]any{"parentRef": map[string]any{"name": test.parent, "namespace": "edge", "sectionName": "https"}, "conditions": []any{map[string]any{"type": "Accepted", "status": test.status, "observedGeneration": test.generation, "reason": "Test"}}}}}, "status")
			entry := collectOne(t, route, gateway)
			if len(entry.Endpoints) != test.wantURLs || (test.issue != "" && !strings.Contains(strings.Join(entry.Issues, " "), test.issue)) {
				t.Fatalf("entry=%+v", entry)
			}
			if test.name == "accepted" && len(entry.Issues) != 0 {
				t.Fatalf("accepted route issues: %v", entry.Issues)
			}
		})
	}
}

func TestNamespaceSelectorAndBackends(t *testing.T) {
	route, gateway := routeFixture(t), gatewayFixture(t)
	setField(t, gateway, []any{map[string]any{"name": "https", "protocol": "HTTPS", "port": int64(443), "allowedRoutes": map[string]any{"namespaces": map[string]any{"from": "Selector", "selector": map[string]any{"matchLabels": map[string]any{"kubernetes.io/metadata.name": "media"}}}}}}, "spec", "listeners")
	setField(t, route, []any{map[string]any{"backendRefs": []any{map[string]any{"name": "backend", "namespace": "backend-ns", "port": int64(8989)}, map[string]any{"name": "missing"}}}}, "spec", "rules")
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "media", Labels: map[string]string{"kubernetes.io/metadata.name": "media"}}}
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend", Namespace: "backend-ns"}, Spec: corev1.ServiceSpec{Selector: map[string]string{"app": "backend"}}}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pending", Namespace: "backend-ns", Labels: map[string]string{"app": "backend"}}}
	kube := fake.NewSimpleClientset(namespace, service, pod)
	entries, warnings, err := collectAutoDocs(context.Background(), kube, dynamicFixture(t, route, gateway), "")
	if err != nil || len(warnings) > 0 || len(entries) != 1 || len(entries[0].Endpoints) != 1 || entries[0].Endpoints[0].URL != "https://app.example.com" {
		t.Fatalf("entries=%v warnings=%v err=%v", entries, warnings, err)
	}
	found := false
	for _, related := range entries[0].RelatedResources {
		if related.Kind == "Pod" && related.Name == "pending" && related.Namespace == "backend-ns" {
			found = true
		}
	}
	if !found || !strings.Contains(strings.Join(entries[0].Issues, " "), "Cannot read backend Service media/missing") {
		t.Fatalf("entry=%+v", entries[0])
	}
	for _, action := range kube.Actions() {
		if action.GetVerb() != "get" && action.GetVerb() != "list" {
			t.Fatalf("unexpected write: %v", action)
		}
	}
}

func TestOptionalGatewayAPIAndCollectionErrors(t *testing.T) {
	for _, forbidden := range []bool{false, true} {
		client := dynamicFixture(t)
		client.PrependReactor("list", "httproutes", func(action ktesting.Action) (bool, runtime.Object, error) {
			if forbidden {
				return true, nil, apierrors.NewForbidden(httpRouteGVR.GroupResource(), "", errors.New("denied"))
			}
			return true, nil, apierrors.NewNotFound(httpRouteGVR.GroupResource(), "")
		})
		ingress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "existing", Labels: map[string]string{publishLabel: "true"}}}
		entries, warnings, err := collectAutoDocs(context.Background(), fake.NewSimpleClientset(ingress), client, "")
		if err != nil || len(entries) != 1 || (len(warnings) > 0) != forbidden {
			t.Fatalf("entries=%v warnings=%v err=%v", entries, warnings, err)
		}
		filtered, warnings, err := collectFilteredResources(context.Background(), client, "", []GVRResource{{Group: httpRouteGVR.Group, Version: "v1", Resource: "httproutes"}})
		if err != nil || filtered == nil || (len(warnings) > 0) != forbidden {
			t.Fatalf("filtered=%v warnings=%v err=%v", filtered, warnings, err)
		}
	}
}

func TestResourceCacheIsolationConcurrencyAndExpiry(t *testing.T) {
	client := dynamicFixture(t, routeFixture(t), gatewayFixture(t))
	kube := fake.NewSimpleClientset()
	var calls atomic.Int32
	auto := cachedResourceHandler(time.Minute, func(ctx context.Context) (any, []string, error) {
		calls.Add(1)
		return collectAutoDocs(ctx, kube, client, "")
	})
	filtered := cachedResourceHandler(time.Minute, func(ctx context.Context) (any, []string, error) {
		return collectFilteredResources(ctx, client, "", []GVRResource{{Group: httpRouteGVR.Group, Version: "v1", Resource: "httproutes"}})
	})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, handler := range []func(*httptest.ResponseRecorder){
				func(w *httptest.ResponseRecorder) {
					auto(w, httptest.NewRequest("GET", "/auto", nil))
					if !strings.Contains(w.Body.String(), `"endpoints"`) {
						t.Errorf("Auto-Docs payload: %s", w.Body.String())
					}
				},
				func(w *httptest.ResponseRecorder) {
					filtered(w, httptest.NewRequest("GET", "/filtered", nil))
					if strings.Contains(w.Body.String(), `"endpoints"`) {
						t.Errorf("Filtered payload: %s", w.Body.String())
					}
				},
			} {
				handler(httptest.NewRecorder())
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("refreshes=%d", calls.Load())
	}
	refresh := cachedResourceHandler(0, func(ctx context.Context) (any, []string, error) { return collectAutoDocs(ctx, kube, client, "") })
	before := httptest.NewRecorder()
	refresh(before, httptest.NewRequest("GET", "/", nil))
	route, err := client.Resource(httpRouteGVR).Namespace("media").Get(context.Background(), "app", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	route.SetLabels(nil)
	if _, err = client.Resource(httpRouteGVR).Namespace("media").Update(context.Background(), route, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	after := httptest.NewRecorder()
	refresh(after, httptest.NewRequest("GET", "/", nil))
	if before.Body.String() == "[]" || after.Body.String() != "[]" {
		t.Fatalf("before=%s after=%s", before.Body.String(), after.Body.String())
	}
}

func TestWarningsAndFailedLoads(t *testing.T) {
	calls := 0
	handler := cachedResourceHandler(time.Minute, func(context.Context) (any, []string, error) {
		calls++
		return []directoryEntry{}, []string{"HTTPRoutes forbidden\ntry permissions"}, nil
	})
	for i := 0; i < 2; i++ {
		response := httptest.NewRecorder()
		handler(response, httptest.NewRequest("GET", "/", nil))
		var warnings []string
		if response.Code != 200 || response.Body.String() != "[]" || json.Unmarshal([]byte(response.Header().Get(warningsHeader)), &warnings) != nil || len(warnings) != 1 {
			t.Fatalf("response=%v", response)
		}
	}
	if calls != 1 {
		t.Fatalf("refreshes=%d", calls)
	}
	failing := cachedResourceHandler(time.Minute, func(context.Context) (any, []string, error) { return nil, nil, errors.New("failure") })
	response := httptest.NewRecorder()
	failing(response, httptest.NewRequest("GET", "/", nil))
	if response.Code != 500 || strings.Contains(response.Body.String(), "null") {
		t.Fatalf("failed response=%v", response)
	}
}
