package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func manualYAML(url string) string {
	return "version: 1\nlinks:\n  - id: wiki\n    title: Wiki\n    url: " + url + "\n"
}
func TestManualSchema(t *testing.T) {
	good := manualYAML("https://wiki.example:8006/a/b?token=PRIVATE#recovery")
	cases := map[string]string{
		"empty":           "",
		"null root":       "null",
		"missing links":   "version: 1",
		"missing version": "links: []",
		"version":         "version: 2\nlinks: []",
		"quoted version":  "version: '1'\nlinks: []",
		"unknown":         "version: 1\nextra: x\nlinks: []",
		"duplicate key":   "version: 1\nversion: 1\nlinks: []",
		"multiple":        good + "---\nversion: 1\nlinks: []",
		"null links":      "version: 1\nlinks: null",
		"links map":       "version: 1\nlinks: {}",
		"null title": strings.Replace(good,
			"title: Wiki",
			"title: null", 1),
		"numeric id": strings.Replace(good,
			"id: wiki",
			"id: 1", 1),
		"missing id":            "version: 1\nlinks: [{title: Wiki, url: 'https://wiki.test'}]",
		"null group":            good + "    group: null\n",
		"empty group":           good + "    group: ''\n",
		"null scope":            good + "    scope: null\n",
		"array scope":           good + "    scope: []\n",
		"unknown scope":         good + "    scope: {zone: home}\n",
		"null scope value":      good + "    scope: {clusterId: null}\n",
		"empty scope value":     good + "    scope: {namespace: ''}\n",
		"duplicate id":          good + "  - id: wiki\n    title: Other\n    url: https://other.test\n",
		"duplicate entry field": good + "    title: Again\n",
		"oversized":             strings.Repeat("x", (256<<10)+1),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseManualLinks(raw, targetConfig{ID: "a"}, "storage", "links", nil); err == nil {
				t.Fatal("accepted invalid document")
			} else if strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("diagnostic disclosed URL")
			}
		})
	}
	for _, raw := range []string{"version: 1\nlinks: []", good, good + "    scope: {}\n"} {
		if _, err := parseManualLinks(raw, targetConfig{ID: "a"}, "storage", "links", nil); err != nil {
			t.Fatal(err)
		}
	}
	var many strings.Builder
	many.WriteString("version: 1\nlinks:\n")
	for i := 0; i < 501; i++ {
		many.WriteString("- {id: x, title: x, url: 'https://wiki.test'}\n")
	}
	if _, err := parseManualLinks(many.String(), targetConfig{}, "", "", nil); err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatal(err)
	}
	for _, raw := range []string{"javascript:alert(1)", "/relative", "https:///path", "https://user:secret@wiki.test/?token=PRIVATE", "https://wiki.test:0/", "https://wiki.test:65536/", "https://wiki.test:/", "https://wiki.test:bad/", "https://wiki.test\\evil/path", "https://wiki.test/\nprivate", "https://wiki.test/ space"} {
		if validManualURL(raw) {
			t.Errorf("accepted unsafe destination %q", raw)
		}
	}
	for _, raw := range []string{"http://localhost/", "https://[::1]:8006/a?x=y#z", "https://wiki.test:65535/a%20b?q=a+b#z"} {
		if !validManualURL(raw) {
			t.Errorf("rejected %q", raw)
		}
	}
}
func TestManualRelevanceAndIdentity(t *testing.T) {
	rules := map[string]classificationRule{"environment": {DisplayCase: "upper"}}
	for mask := 0; mask < 16; mask++ {
		raw := manualYAML("https://wiki.test/a?q=1#z")
		fields := []string{"clusterId: missing", "namespace: media", "environment: Prod", "datacentre: Home"}
		raw += "    scope:\n"
		if mask == 0 {
			raw = manualYAML("https://wiki.test/a?q=1#z")
		}
		for i, field := range fields {
			if mask&(1<<i) != 0 {
				raw += "      " + field + "\n"
			}
		}
		rows, err := parseManualLinks(raw, targetConfig{ID: "storage", Name: "Storage", Labels: map[string]string{"env": "wrong"}}, "storage-ns", "links", rules)
		if err != nil {
			t.Fatal(err)
		}
		row := rows[0]
		if row["sourceIdentity"] != `["manual-link","storage","storage-ns","links","wiki"]` {
			t.Fatal(row)
		}
		if mask&1 == 0 && row["clusterId"] != nil || mask&2 == 0 && row["namespace"] != nil {
			t.Fatal("inherited relevance", row)
		}
		if mask&4 != 0 && classificationOf(row)["environment"] != (classifiedValue{ID: "prod", Display: "PROD"}) {
			t.Fatal(row)
		}
		if mask&8 != 0 && classificationOf(row)["datacentre"] != (classifiedValue{ID: "Home", Display: "Home"}) {
			t.Fatal(row)
		}
		parts := map[string]targetSnapshot{"storage": {Parts: map[string]inventoryPart{manualComponent: {Data: rows, Documents: map[string]manualDocument{"storage-ns/links": {Data: rows}}}}}}
		resolveManualClusters(parts, nil)
		if mask&1 != 0 && (row["clusterName"] != "missing" || row["issues"] == nil) {
			t.Fatal("unresolved scope lost", row)
		}
	}
}
func TestManualCollectionRetentionAndRestore(t *testing.T) {
	c := fixtureConfig(t)
	c.Cache.Path = filepath.Join(t.TempDir(), "snapshot.json")
	kube := fake.NewSimpleClientset()
	s := newInventoryService(c)
	clock := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return clock }
	s.connect = func(targetConfig) (targetClients, error) {
		return targetClients{Kube: kube, Dynamic: dynamicFixture(t), EndpointID: "fixture"}, nil
	}
	s.collect = func(ctx context.Context, target targetConfig, clients targetClients) map[string]collectionResult {
		return map[string]collectionResult{manualComponent: s.collectManual(ctx, target, clients), "ingresses": {}, "httproutes": {}, "services": {}, "workloads": {}, "filtered": {}}
	}
	cm := func(name, raw string) *corev1.ConfigMap {
		return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: "central", Name: name, Labels: map[string]string{manualSourceLabel: "true"}}, Data: map[string]string{"links.yaml": raw, "ignored": "RAW-PRIVATE-BODY"}}
	}
	for _, obj := range []*corev1.ConfigMap{cm("one", manualYAML("https://wiki.test/?private=keep")), cm("two", manualYAML("https://two.test/")), cm("unmarked", "bad")} {
		if obj.Name == "unmarked" {
			obj.Labels[manualSourceLabel] = "True"
		}
		if _, err := kube.CoreV1().ConfigMaps("central").Create(context.Background(), obj, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	s.refresh("test")
	rows, _ := inventoryRows(t, s, "directory", "published")
	if len(rows) != 4 {
		t.Fatal(rows)
	}
	first := s.snapshot.Targets["a"].Parts[manualComponent].Documents["central/one"].LastSuccessAt
	clock = clock.Add(time.Minute)
	one, _ := kube.CoreV1().ConfigMaps("central").Get(context.Background(), "one", metav1.GetOptions{})
	one.Data["links.yaml"] = "version: null\nRAW-PRIVATE-BODY: bad"
	kube.CoreV1().ConfigMaps("central").Update(context.Background(), one, metav1.UpdateOptions{})
	two, _ := kube.CoreV1().ConfigMaps("central").Get(context.Background(), "two", metav1.GetOptions{})
	two.Data["links.yaml"] = manualYAML("https://two.test/new")
	kube.CoreV1().ConfigMaps("central").Update(context.Background(), two, metav1.UpdateOptions{})
	kube.CoreV1().ConfigMaps("central").Create(context.Background(), cm("new-invalid", "bad"), metav1.CreateOptions{})
	s.refresh("test")
	part := s.snapshot.Targets["a"].Parts[manualComponent]
	if part.Error == "" || !part.LastSuccessAt.Equal(*first) || !part.Documents["central/one"].LastSuccessAt.Equal(*first) || part.Documents["central/new-invalid"].LastSuccessAt != nil || !part.Documents["central/two"].LastSuccessAt.Equal(clock) {
		t.Fatal(part)
	}
	rows, warnings := inventoryRows(t, s, "directory", "published")
	if len(rows) != 4 || warnings == "" {
		t.Fatal(rows, warnings)
	}
	before := len(kube.Actions())
	for _, kind := range []string{"directory", "workloads", "filtered"} {
		if kind == "directory" {
			rows, _ := inventoryRows(t, s, kind, "discovered")
			if len(rows) != 0 {
				t.Fatal("manual entry entered Discovered")
			}
		} else {
			s.snapshot.Targets["a"].Parts[kind] = inventoryPart{LastSuccessAt: &clock}
			rows, _ := inventoryRows(t, s, kind, "")
			if len(rows) != 0 {
				t.Fatal("manual entry entered Kubernetes inventory")
			}
		}
	}
	s.status()
	if before != len(kube.Actions()) {
		t.Fatal("reader collected")
	}
	// Derived cache has exact destination URLs, never raw documents or other keys.
	bytes, err := os.ReadFile(c.Cache.Path)
	if err != nil || strings.Contains(string(bytes), "RAW-PRIVATE-BODY") {
		t.Fatal("raw document persisted", err)
	}
	restored := newInventoryService(c)
	if len(restored.snapshot.Targets["a"].Parts[manualComponent].Data) != 2 || !restored.snapshot.Targets["a"].Parts[manualComponent].Documents["central/one"].LastSuccessAt.Equal(*first) {
		t.Fatal("restore lost manual data")
	}
	statusJSON, _ := json.Marshal(restored.status())
	if strings.Contains(string(statusJSON), "private=keep") {
		t.Fatal("status leaked URL")
	}
	kube.PrependReactor("list", "configmaps", func(ktesting.Action) (bool, runtime.Object, error) { return true, nil, errors.New("PRIVATE") })
	clock = clock.Add(time.Minute)
	s.refresh("test")
	part = s.snapshot.Targets["a"].Parts[manualComponent]
	if len(part.Data) != 2 || !part.Documents["central/one"].LastSuccessAt.Equal(*first) {
		t.Fatal("failed list lost data")
	}
	kube.ReactionChain = kube.ReactionChain[1:]
	// Successful marker removal, deletion and valid [] all withdraw old entries.
	delete(one.Labels, manualSourceLabel)
	kube.CoreV1().ConfigMaps("central").Update(context.Background(), one, metav1.UpdateOptions{})
	two.Data["links.yaml"] = "version: 1\nlinks: []"
	kube.CoreV1().ConfigMaps("central").Update(context.Background(), two, metav1.UpdateOptions{})
	kube.CoreV1().ConfigMaps("central").Delete(context.Background(), "new-invalid", metav1.DeleteOptions{})
	s.refresh("test")
	if p := s.snapshot.Targets["a"].Parts[manualComponent]; len(p.Data) != 0 || p.Error != "" || len(p.Documents) != 1 {
		t.Fatal(p)
	}
	// Cache identity/removal protection follows storage target, not relevance.
	restoredConfig := c
	restoredConfig.Clusters = c.Clusters[1:]
	reduced := newInventoryService(restoredConfig)
	if _, ok := reduced.snapshot.Targets["a"]; ok {
		t.Fatal("removed storage target restored")
	}
	// Exact selector is applied even with test clients; no destination gets fetched.
	for _, action := range kube.Actions() {
		if action.GetVerb() == "list" {
			list := action.(ktesting.ListAction)
			if action.GetResource() == (schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}) && list.GetListRestrictions().Labels.String() != manualSourceLabel+"=true" {
				t.Fatal(action)
			}
		}
	}
}

func TestManualTargetIdentityAndLegacyCache(t *testing.T) {
	c := fixtureConfig(t)
	c.Cache.Path = filepath.Join(t.TempDir(), "snapshot.json")
	s := newInventoryService(c)
	kube := fake.NewSimpleClientset(&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "links", Namespace: "central", Labels: map[string]string{manualSourceLabel: "true"}}, Data: map[string]string{"links.yaml": manualYAML("https://wiki.test/path?q=1#z") + "    scope: {clusterId: b, environment: Unique}\n"}})
	endpoint := "original"
	s.connect = func(targetConfig) (targetClients, error) {
		return targetClients{Kube: kube, Dynamic: dynamicFixture(t), EndpointID: endpoint}, nil
	}
	// Use the real collector to ensure marker discovery is wired into normal runs.
	s.refresh("test")
	if len(s.snapshot.Targets["a"].Parts[manualComponent].Data) != 1 {
		t.Fatal("normal collector omitted manual links")
	}
	status := s.status()
	if status["dimensions"].(map[string]map[string]string)["environment"]["Unique"] != "Unique" {
		t.Fatal("manual-only dimension unavailable")
	}
	restored := newInventoryService(c)
	if row := restored.snapshot.Targets["a"].Parts[manualComponent].Data[0]; row["clusterId"] != "b" || row["collection"].(map[string]any)["error"] == "" {
		t.Fatal("restored provenance or stale status lost", row)
	}
	// Configuration identity changes and removal are governed by source target A.
	changed := c
	changed.Clusters = append([]targetConfig(nil), c.Clusters...)
	changed.Clusters[0].Kubeconfig = &kubeconfigSource{Path: "/different", Context: "different"}
	if _, ok := newInventoryService(changed).snapshot.Targets["a"]; ok {
		t.Fatal("changed source identity restored")
	}
	cm, _ := kube.CoreV1().ConfigMaps("central").Get(context.Background(), "links", metav1.GetOptions{})
	cm.Data["links.yaml"] = "invalid"
	kube.CoreV1().ConfigMaps("central").Update(context.Background(), cm, metav1.UpdateOptions{})
	endpoint = "replacement"
	s.refresh("test")
	if len(s.snapshot.Targets["a"].Parts[manualComponent].Data) != 0 {
		t.Fatal("old destination survived target identity change")
	}
	// Earlier compatible cache files do not claim a successful empty manual list.
	saved := cloneSnapshot(s.snapshot)
	for id, target := range saved.Targets {
		delete(target.Parts, manualComponent)
		saved.Targets[id] = target
	}
	if s.persist(saved) != "" {
		t.Fatal("cannot save legacy fixture")
	}
	old := newInventoryService(c)
	part := old.status()["clusters"].([]map[string]any)[0]["parts"].(map[string]any)[manualComponent].(map[string]any)
	if part["lastSuccessAt"].(*time.Time) != nil {
		t.Fatal("legacy cache fabricated manual success")
	}
}

func TestManualExactBoundariesAndDisplayPool(t *testing.T) {
	raw := manualYAML("https://wiki.test:8006/a%20b?next=%2F&x=1#Recovery")
	padded := raw + "#" + strings.Repeat("x", (256<<10)-len(raw)-2) + "\n"
	rows, err := parseManualLinks(padded, targetConfig{ID: "a"}, "ns", "links", nil)
	if err != nil || rows[0]["endpoints"].([]map[string]any)[0]["url"] != "https://wiki.test:8006/a%20b?next=%2F&x=1#Recovery" {
		t.Fatal("boundary or URL changed", err)
	}
	var many strings.Builder
	many.WriteString("version: 1\nlinks:\n")
	for i := 0; i < 500; i++ {
		many.WriteString(fmt.Sprintf("- {id: 'id%d', title: x, url: 'https://wiki.test'}\n", i))
	}
	if rows, err := parseManualLinks(many.String(), targetConfig{}, "", "", nil); err != nil || len(rows) != 500 {
		t.Fatal("500 entries rejected", err)
	}
	// A single spelling pool includes manual and discovered rows even after cache cloning.
	c := fixtureConfig(t)
	c.Classification = map[string]classificationRule{"environment": {}}
	s := newInventoryService(c)
	s.connect = func(targetConfig) (targetClients, error) { return targetClients{}, nil }
	s.collect = func(_ context.Context, target targetConfig, _ targetClients) map[string]collectionResult {
		rows, _ := parseManualLinks(manualYAML("https://wiki.test/")+"    scope: {environment: prod}\n", target, "ns", "links", c.Classification)
		return map[string]collectionResult{manualComponent: {Documents: map[string]manualDocument{"ns/links": {Data: rows}}}, "ingresses": {Data: []map[string]any{{"kind": "ingress", "labels": map[string]string{publishLabel: "true"}, "classification": map[string]classifiedValue{"environment": {ID: "prod", Display: "Prod"}}}}}}
	}
	s.refresh("test")
	for _, target := range s.snapshot.Targets {
		if classificationOf(target.Parts[manualComponent].Data[0])["environment"].Display != "Prod" {
			t.Fatal("manual spelling outside deterministic pool")
		}
	}
}
