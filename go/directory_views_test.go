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

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestDirectoryViewClassification(t *testing.T) {
	for _, test := range []struct {
		name                  string
		labels                map[string]string
		published, discovered bool
	}{
		{"absent", nil, false, true},
		{"group only", map[string]string{groupLabel: "media"}, false, true},
		{"published", map[string]string{publishLabel: "true", groupLabel: "media"}, true, false},
		{"false", map[string]string{publishLabel: "false"}, false, false},
		{"empty", map[string]string{publishLabel: ""}, false, false},
		{"mixed case", map[string]string{publishLabel: "True"}, false, false},
		{"other", map[string]string{publishLabel: "yes"}, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, view := range []directoryView{publishedView, discoveredView} {
				ingress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "ingress", Namespace: "apps", Labels: test.labels,
					Annotations: map[string]string{publishLabel: "true"}}, Spec: networkingv1.IngressSpec{Rules: []networkingv1.IngressRule{{Host: "ingress.example.com"}}}}
				route, gateway := routeFixture(t), gatewayFixture(t)
				route.SetLabels(test.labels)
				gateway.SetLabels(map[string]string{publishLabel: "true", groupLabel: "inherited-group"})
				client, kube := dynamicFixture(t, route, gateway), fake.NewSimpleClientset(ingress)
				entries, warnings, err := collectAutoDocsView(context.Background(), kube, client, "", view)
				want := test.published
				if view == discoveredView {
					want = test.discovered
				}
				count := 0
				if want {
					count = 2
				}
				if err != nil || len(warnings) != 0 || len(entries) != count {
					t.Fatalf("view=%s entries=%v warnings=%v err=%v", view, entries, warnings, err)
				}
				for _, entry := range entries {
					if !reflect.DeepEqual(entry.Labels, test.labels) || len(entry.Endpoints) != 1 {
						t.Fatalf("source metadata or URL resolution changed: %+v", entry)
					}
				}
				for _, actions := range [][]ktesting.Action{kube.Actions(), client.Actions()} {
					for _, action := range actions {
						if list, ok := action.(ktesting.ListAction); ok && (action.GetResource().Resource == "ingresses" || action.GetResource().Resource == "httproutes") {
							if list.GetListRestrictions().Labels.String() != view.selector() {
								t.Fatalf("wrong selector: %v", action)
							}
						}
						if !want && action.GetVerb() == "get" {
							t.Fatalf("excluded source resolved a dependency: %v", action)
						}
					}
				}
				filtered, _, err := collectFilteredResources(context.Background(), client, "", []GVRResource{{Group: httpRouteGVR.Group, Version: "v1", Resource: "httproutes"}})
				filteredCount := 0
				if test.published {
					filteredCount = 1
				}
				if err != nil || len(filtered) != filteredCount {
					t.Fatalf("filtered=%v err=%v", filtered, err)
				}
			}
		})
	}
}

func TestDiscoveredPartialCollectionAndUnresolvedEntries(t *testing.T) {
	for _, reason := range []string{"absent API", "forbidden", "missing parent"} {
		t.Run(reason, func(t *testing.T) {
			route := routeFixture(t)
			route.SetLabels(nil)
			client := dynamicFixture(t, route)
			if reason != "missing parent" {
				client.PrependReactor("list", "httproutes", func(ktesting.Action) (bool, runtime.Object, error) {
					if reason == "forbidden" {
						return true, nil, apierrors.NewForbidden(httpRouteGVR.GroupResource(), "", errors.New("denied"))
					}
					return true, nil, apierrors.NewNotFound(httpRouteGVR.GroupResource(), "")
				})
			}
			ingress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "existing", Namespace: "apps"}}
			entries, warnings, err := collectAutoDocsView(context.Background(), fake.NewSimpleClientset(ingress), client, "", discoveredView)
			want := 1
			if reason == "missing parent" {
				want = 2
			}
			if err != nil || len(entries) != want || (len(warnings) > 0) != (reason == "forbidden") {
				t.Fatalf("entries=%v warnings=%v err=%v", entries, warnings, err)
			}
			if reason == "missing parent" && (len(entries[1].Endpoints) != 0 || len(entries[1].Issues) == 0) {
				t.Fatalf("unresolved source lost its issues: %+v", entries[1])
			}
		})
	}
}

func TestDirectoryViewAPICachesAndValidation(t *testing.T) {
	for _, first := range []directoryView{publishedView, discoveredView} {
		t.Run(string(first), func(t *testing.T) {
			var publishedCalls, discoveredCalls atomic.Int32
			handler := directoryViewsHandler(time.Minute, func(ctx context.Context, view directoryView) (any, []string, error) {
				warnings := []string{}
				if view == discoveredView {
					discoveredCalls.Add(1)
					warnings = append(warnings, "discovery warning")
				} else {
					publishedCalls.Add(1)
				}
				return []map[string]string{{"view": string(view)}}, warnings, nil
			})
			for _, query := range []string{"?view=unknown", "?view=discovered&view=published", "?view=&view="} {
				response := httptest.NewRecorder()
				handler(response, httptest.NewRequest("GET", "/auto"+query, nil))
				if response.Code != 400 {
					t.Fatalf("query=%s status=%d", query, response.Code)
				}
			}
			if publishedCalls.Load()+discoveredCalls.Load() != 0 {
				t.Fatal("invalid requests collected resources")
			}
			request := func(query string, view directoryView) {
				response := httptest.NewRecorder()
				handler(response, httptest.NewRequest("GET", "/auto"+query, nil))
				if response.Code != 200 || !strings.Contains(response.Body.String(), `"view": "`+string(view)+`"`) {
					t.Errorf("query=%s response=%v", query, response)
				}
				if (response.Header().Get(warningsHeader) != "") != (view == discoveredView) {
					t.Errorf("warnings crossed views: %v", response.Header())
				}
			}
			request("?view="+string(first), first)
			var wg sync.WaitGroup
			for i := 0; i < 20; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					request("?view=published", publishedView)
					request("?view=discovered", discoveredView)
				}()
			}
			wg.Wait()
			request("", publishedView)
			request("?view=", publishedView)
			if publishedCalls.Load() != 1 || discoveredCalls.Load() != 1 {
				t.Fatalf("cache loads: published=%d discovered=%d", publishedCalls.Load(), discoveredCalls.Load())
			}
		})
	}
}

func TestDirectoryMetadataTransitionsAfterExpiry(t *testing.T) {
	ctx := context.Background()
	route := routeFixture(t)
	route.SetLabels(nil)
	client := dynamicFixture(t, route, gatewayFixture(t))
	kube := fake.NewSimpleClientset()
	// Zero TTL makes every request an expired-cache refresh without timing-dependent sleeps.
	handler := directoryViewsHandler(0, func(ctx context.Context, view directoryView) (any, []string, error) {
		return collectAutoDocsView(ctx, kube, client, "", view)
	})
	for _, labels := range []map[string]string{nil, {publishLabel: "true", groupLabel: "media"}, {publishLabel: "true"}, nil, {publishLabel: "false"}} {
		route.SetLabels(labels)
		if _, err := client.Resource(httpRouteGVR).Namespace(route.GetNamespace()).Update(ctx, route, metav1.UpdateOptions{}); err != nil {
			t.Fatal(err)
		}
		for _, view := range []directoryView{publishedView, discoveredView} {
			response := httptest.NewRecorder()
			handler(response, httptest.NewRequest("GET", "/auto?view="+string(view), nil))
			var entries []directoryEntry
			if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &entries) != nil {
				t.Fatalf("response=%v", response)
			}
			count := 0
			if view.includes(labels) {
				count = 1
			}
			if len(entries) != count {
				t.Fatalf("view=%s labels=%v entries=%v", view, labels, entries)
			}
			if count > 0 && !reflect.DeepEqual(entries[0].Labels, labels) {
				t.Fatalf("stale labels: %+v", entries[0])
			}
		}
	}
}
