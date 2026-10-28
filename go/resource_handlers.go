package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const publishLabel = "k8sautodocs.io/publish"
const publishSelector = publishLabel + "=true"
const groupLabel = "k8sautodocs.io/group"
const warningsHeader = "X-AutoDocs-Warnings"

type directoryView string

const (
	publishedView  directoryView = "published"
	discoveredView directoryView = "discovered"
)

func (view directoryView) selector() string {
	if view == discoveredView {
		return "!" + publishLabel
	}
	return publishSelector
}

func (view directoryView) includes(resourceLabels map[string]string) bool {
	value, present := resourceLabels[publishLabel]
	if view == discoveredView {
		return !present
	}
	return value == "true"
}

var httpRouteGVR = schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"}
var gatewayGVR = schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways"}

type resourceLoader func(context.Context) (any, []string, error)

// Every handler owns its cache, including warnings. The lock also coalesces refreshes.
func cachedResourceHandler(ttl time.Duration, load resourceLoader) http.HandlerFunc {
	var mu sync.Mutex
	var payload []byte
	var warnings string
	var refreshed time.Time
	return func(w http.ResponseWriter, r *http.Request) {
		for key, value := range headerMap {
			w.Header().Set(key, value)
		}
		w.Header().Set("Access-Control-Expose-Headers", warningsHeader)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if payload == nil || time.Since(refreshed) >= ttl {
			data, issues, err := load(r.Context())
			if err != nil {
				log.Printf("Resource collection failed: %v", err)
				http.Error(w, "Error getting resources", http.StatusInternalServerError)
				return
			}
			encoded, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				http.Error(w, "Error marshaling JSON", http.StatusInternalServerError)
				return
			}
			warnings = ""
			if len(issues) > 0 {
				// JSON encoding prevents raw line breaks in the warning header.
				encodedIssues, _ := json.Marshal(issues)
				warnings = string(encodedIssues)
				log.Printf("Incomplete resource discovery: %s", warnings)
			}
			payload, refreshed = encoded, time.Now()
		}
		if warnings != "" {
			w.Header().Set(warningsHeader, warnings)
		}
		w.Write(payload)
	}
}

// Two fixed handlers keep cache state bounded and isolate payloads and warnings.
func directoryViewsHandler(ttl time.Duration, load func(context.Context, directoryView) (any, []string, error)) http.HandlerFunc {
	handlers := map[directoryView]http.HandlerFunc{}
	for _, view := range []directoryView{publishedView, discoveredView} {
		handlers[view] = cachedResourceHandler(ttl, func(ctx context.Context) (any, []string, error) {
			return load(ctx, view)
		})
	}
	return func(w http.ResponseWriter, r *http.Request) {
		for key, value := range headerMap {
			w.Header().Set(key, value)
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		query := r.URL.Query()["view"]
		view := publishedView
		if len(query) == 1 && query[0] != "" {
			view = directoryView(query[0])
		}
		handler, valid := handlers[view]
		if len(query) > 1 || !valid {
			http.Error(w, "view must be published or discovered and specified at most once", http.StatusBadRequest)
			return
		}
		handler(w, r)
	}
}

func collectFilteredResources(ctx context.Context, client dynamic.Interface, namespace string, resources []GVRResource) ([]map[string]any, []string, error) {
	result := make([]map[string]any, 0)
	warnings := make([]string, 0)
	for _, resource := range resources {
		gvr := schema.GroupVersionResource{Group: resource.Group, Version: resource.Version, Resource: resource.Resource}
		list, err := client.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{LabelSelector: publishSelector})
		if err != nil {
			if gvr == httpRouteGVR && apierrors.IsNotFound(err) {
				continue // Gateway API is optional.
			}
			warnings = append(warnings, fmt.Sprintf("Cannot list %s: %v", gvr.Resource, err))
			continue
		}
		for _, item := range list.Items {
			if item.GetLabels()[publishLabel] != "true" {
				continue
			}
			result = append(result, map[string]any{
				"name": item.GetName(), "namespace": item.GetNamespace(), "kind": item.GetKind(),
				"labels": item.GetLabels(), "annotations": map[string]string{},
			})
		}
	}
	return result, warnings, ctx.Err()
}
