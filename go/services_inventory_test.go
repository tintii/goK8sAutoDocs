package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func inventoryRows(t *testing.T, s *inventoryService, kind, view string) ([]map[string]any, string) {
	t.Helper()
	w := httptest.NewRecorder()
	s.dataHandler(kind)(w, httptest.NewRequest("GET", "/?view="+view, nil))
	var rows []map[string]any
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &rows) != nil {
		t.Fatalf("%s/%s: %d %s", kind, view, w.Code, w.Body.String())
	}
	return rows, w.Header().Get(warningsHeader)
}

func TestServicesSharedInventory(t *testing.T) {
	c := fixtureConfig(t)
	c.Classification = map[string]classificationRule{"environment": {LabelKeys: []string{"env"}, Sources: []string{"resource", "namespace"}, DisplayCase: "upper"}}
	s := newInventoryService(c)
	service := serviceFixture()
	service.Name = "same"
	ingress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "same", Namespace: "apps", Labels: map[string]string{publishLabel: "true"}}, Spec: networkingv1.IngressSpec{Rules: []networkingv1.IngressRule{{Host: "app.example.com", IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "same"}}}}}}}}}}
	route := routeFixture(t)
	route.SetNamespace("apps")
	setField(t, route, []any{map[string]any{"backendRefs": []any{map[string]any{"name": "same"}}}}, "spec", "rules")
	dynamic := dynamicFixture(t, route, gatewayFixture(t))
	kube := fake.NewSimpleClientset(&service, ingress, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "apps", Labels: map[string]string{"env": "npe00"}}})
	s.connect = func(targetConfig) (targetClients, error) {
		return targetClients{Kube: kube, Dynamic: dynamic, EndpointID: "fixture"}, nil
	}
	s.refresh("startup")
	rows, warnings := inventoryRows(t, s, "directory", "published")
	if len(rows) != 6 || warnings != "" {
		t.Fatalf("mixed sources: rows=%v warnings=%s", rows, warnings)
	}
	services := 0
	for _, row := range rows {
		if row["kind"] == "Service" {
			services++
			if row["name"] != "same" || row["clusterId"] == "" || classificationOf(row)["environment"].Display != "NPE00" {
				t.Fatalf("identity/classification: %v", row)
			}
		}
	}
	if services != 2 {
		t.Fatalf("Service source duplicated or merged: %v", rows)
	}
	filtered, _ := inventoryRows(t, s, "filtered", "")
	if len(filtered) != 6 {
		t.Fatalf("filtered rows duplicated: %v", filtered)
	}
	standaloneLists := 0
	for _, action := range kube.Actions() {
		if action.GetVerb() == "list" && action.GetResource().Resource == "services" {
			standaloneLists++
		}
	}
	if standaloneLists != 2 {
		t.Fatalf("expected one standalone Service list per cluster, got %d", standaloneLists)
	}
	for _, action := range dynamic.Actions() {
		if action.GetResource().Resource == "services" {
			t.Fatal("generic Service metadata query duplicated standalone list")
		}
	}
	before := len(kube.Actions()) + len(dynamic.Actions())
	inventoryRows(t, s, "directory", "discovered")
	inventoryRows(t, s, "filtered", "")
	if before != len(kube.Actions())+len(dynamic.Actions()) {
		t.Fatal("inventory GET queried Kubernetes")
	}

	// Both views classify the same refreshed partition; hostname-key presence matters.
	service.Labels = nil
	if _, err := kube.CoreV1().Services("apps").Update(context.Background(), &service, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	s.refresh("manual")
	published, _ := inventoryRows(t, s, "directory", "published")
	discovered, _ := inventoryRows(t, s, "directory", "discovered")
	filtered, _ = inventoryRows(t, s, "filtered", "")
	if len(published) != 4 || len(discovered) != 2 || len(filtered) != 4 {
		t.Fatalf("publication removal: published=%d discovered=%d filtered=%d", len(published), len(discovered), len(filtered))
	}
	service.Annotations = nil
	if _, err := kube.CoreV1().Services("apps").Update(context.Background(), &service, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	s.refresh("manual")
	discovered, _ = inventoryRows(t, s, "directory", "discovered")
	if len(discovered) != 0 {
		t.Fatal("removed hostname remained discovered")
	}
	service.Labels = map[string]string{publishLabel: "true"}
	if _, err := kube.CoreV1().Services("apps").Update(context.Background(), &service, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	s.refresh("manual")
	if len(s.snapshot.Targets["a"].Parts["services"].Data) != 1 {
		t.Fatal("published Service without hostname lost")
	}
	service.Labels = nil
	if _, err := kube.CoreV1().Services("apps").Update(context.Background(), &service, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	s.refresh("manual")
	if len(s.snapshot.Targets["a"].Parts["services"].Data) != 0 {
		t.Fatal("label-only Service remained after publication removal")
	}
}

func TestServicesFailureRestoreAndDeletion(t *testing.T) {
	c := fixtureConfig(t)
	s := newInventoryService(c)
	service := serviceFixture()
	kube := fake.NewSimpleClientset(&service)
	dynamic := dynamicFixture(t)
	s.connect = func(targetConfig) (targetClients, error) {
		return targetClients{Kube: kube, Dynamic: dynamic, EndpointID: "fixture"}, nil
	}
	at := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return at }
	s.refresh("startup")
	first := *s.snapshot.Targets["a"].Parts["services"].LastSuccessAt
	serviceFail, ingressFail := true, false
	kube.PrependReactor("list", "services", func(ktesting.Action) (bool, runtime.Object, error) {
		if serviceFail {
			return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "services"}, "", errors.New("no"))
		}
		return false, nil, nil
	})
	kube.PrependReactor("list", "ingresses", func(ktesting.Action) (bool, runtime.Object, error) {
		if ingressFail {
			return true, nil, errors.New("unavailable")
		}
		return false, nil, nil
	})
	at = at.Add(time.Minute)
	s.refresh("manual")
	part := s.snapshot.Targets["a"].Parts["services"]
	if len(part.Data) != 1 || !part.LastSuccessAt.Equal(first) || part.Error != "access forbidden" {
		t.Fatalf("Service retention: %+v", part)
	}
	if !s.snapshot.Targets["a"].Parts["ingresses"].LastSuccessAt.Equal(at) || !s.snapshot.Targets["a"].Parts["filtered"].LastSuccessAt.Equal(first) {
		t.Fatal("partition isolation or whole-filtered retention failed")
	}
	_, warnings := inventoryRows(t, s, "directory", "published")
	if !strings.Contains(warnings, "services: access forbidden") {
		t.Fatalf("missing Service warning: %s", warnings)
	}

	// A new reader restores the additive v1 partition with original timestamps.
	restored := newInventoryService(c)
	if len(restored.snapshot.Targets["a"].Parts["services"].Data) != 1 || !restored.snapshot.Targets["a"].Parts["services"].LastSuccessAt.Equal(first) {
		t.Fatal("restored Service data/time lost")
	}
	restored.now = s.now
	restored.connect = func(targetConfig) (targetClients, error) {
		return targetClients{}, errors.New("credentials unavailable")
	}
	restored.refresh("startup")
	part = restored.snapshot.Targets["a"].Parts["services"]
	if len(part.Data) != 1 || !part.LastSuccessAt.Equal(first) || part.Error == "" {
		t.Fatalf("connection failure discarded Service: %+v", part)
	}
	_, warnings = inventoryRows(t, restored, "directory", "published")
	if !strings.Contains(warnings, "services:") {
		t.Fatal("connection failure not projected")
	}

	// Other lists may fail while a Service deletion still converges.
	serviceFail, ingressFail = false, true
	if err := kube.CoreV1().Services("apps").Delete(context.Background(), service.Name, metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	at = at.Add(time.Minute)
	s.refresh("manual")
	part = s.snapshot.Targets["a"].Parts["services"]
	if len(part.Data) != 0 || part.Error != "" || !part.LastSuccessAt.Equal(at) {
		t.Fatalf("successful deletion: %+v", part)
	}
	if s.snapshot.Targets["a"].Parts["ingresses"].Error == "" {
		t.Fatal("independent ingress error lost")
	}
}

func TestServicesSnapshotV1Compatibility(t *testing.T) {
	c := fixtureConfig(t)
	s := newInventoryService(c)
	at := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	legacy := inventoryPart{LastSuccessAt: &at, Data: []map[string]any{{"name": "old", "kind": "ingress", "namespace": "apps", "clusterId": "a", "labels": map[string]string{publishLabel: "true"}}}}
	s.snapshot.Generation = 7
	s.snapshot.Targets["a"] = targetSnapshot{ConfigID: c.targetFingerprint(c.Clusters[0]), Parts: map[string]inventoryPart{"ingresses": legacy, "httproutes": {LastSuccessAt: &at}}}
	if issue := s.persist(s.snapshot); issue != "" {
		t.Fatal(issue)
	}
	restored := newInventoryService(c)
	rows, warnings := inventoryRows(t, restored, "directory", "published")
	if len(rows) != 1 || !strings.Contains(warnings, "services: never collected") || restored.snapshot.Targets["a"].Parts["services"].LastSuccessAt != nil {
		t.Fatal("old snapshot invalidated or Services marked fresh")
	}
	status := restored.status()["clusters"].([]map[string]any)[0]["parts"].(map[string]any)["services"].(map[string]any)
	if status["lastSuccessAt"] != (*time.Time)(nil) {
		t.Fatalf("Services should be pending: %v", status)
	}

	// The prior v1 reader decoded a map of parts and projected only named keys.
	// Keep that exact envelope/read pattern here to verify additive partitions
	// and new entry fields cannot invalidate existing Ingress/HTTPRoute data.
	v := restored.snapshot.Targets["a"]
	v.Parts["services"] = inventoryPart{LastSuccessAt: &at, Data: []map[string]any{asRow(serviceEntry(serviceFixture()))}}
	restored.snapshot.Targets["a"] = v
	encoded, err := json.Marshal(restored.snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var oldReader struct {
		Version    int    `json:"version"`
		Generation uint64 `json:"generation"`
		Targets    map[string]struct {
			ConfigID string `json:"configId"`
			Parts    map[string]struct {
				Data          []map[string]any `json:"data"`
				LastSuccessAt *time.Time       `json:"lastSuccessAt"`
			} `json:"parts"`
		} `json:"targets"`
	}
	if json.Unmarshal(encoded, &oldReader) != nil || oldReader.Version != 1 || oldReader.Targets == nil {
		t.Fatal("old v1 reader rejected additive snapshot")
	}
	oldRows := []map[string]any{}
	for _, key := range []string{"ingresses", "httproutes"} {
		oldRows = append(oldRows, oldReader.Targets["a"].Parts[key].Data...)
	}
	if len(oldRows) != 1 || oldRows[0]["name"] != "old" || !oldReader.Targets["a"].Parts["ingresses"].LastSuccessAt.Equal(at) {
		t.Fatal("older projection changed existing data")
	}
}
