package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const directoryTestFeature = `  - source: {clusterId: a, kind: Ingress, namespace: apps, name: dashboard}
    url: https://example.test/second
`

func TestDirectoryConfigValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "directory.yaml")
	for _, test := range []struct{ name, yaml string }{
		{"unknown field", "groups: []"},
		{"duplicate field", "featured: []\nfeatured: []"},
		{"duplicate feature", "featured:\n" + directoryTestFeature + directoryTestFeature},
		{"duplicate group", "groupOrder: [media, media]"},
		{"non-string group", "groupOrder: [true]"},
		{"null list", "featured: null"},
		{"missing identity", "featured:\n" + strings.Replace(directoryTestFeature, ", namespace: apps", "", 1)},
		{"relative URL", "featured:\n" + strings.Replace(directoryTestFeature, "https://example.test/second", "/second", 1)},
		{"both selectors", "featured:\n" + directoryTestFeature + "    manualLink: {}"},
		{"no selector", "featured: [{url: 'https://example.test/'}]"},
		{"multiple documents", "{}\n---\n{}"},
		{"oversize", "{}" + strings.Repeat(" ", 64<<10)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(test.yaml), 0600); err != nil {
				t.Fatal(err)
			}
			if config, err := loadDirectoryConfig(path); err == nil || len(config.groups) != 0 || len(config.features) != 0 {
				t.Fatalf("expected rejection and default config, got %+v, %v", config, err)
			}
		})
	}
}

func TestPublishedDirectoryCurationReloadAndRestore(t *testing.T) {
	c := fixtureConfig(t)
	c.Clusters = c.Clusters[:1]
	s := newInventoryService(c)
	at := time.Now().UTC()
	labels := map[string]string{publishLabel: "true", groupLabel: "media"}
	ingress := func(name string) map[string]any {
		return asRow(map[string]any{"name": name, "clusterId": "a", "namespace": "apps", "kind": "ingress", "labels": labels,
			"endpoints": []map[string]string{{"url": "https://example.test/first"}, {"url": "https://example.test/second"}}})
	}
	manual, err := parseManualLinks("version: 1\nlinks: [{id: guide, title: Guide, url: 'https://example.test/first', group: Media, scope: {clusterId: b, namespace: other}}]", c.Clusters[0], "central", "links", c.Classification)
	if err != nil {
		t.Fatal(err)
	}
	discovered := ingress("draft")
	discovered["labels"] = map[string]string{groupLabel: "media"}
	s.snapshot.Generation = 1
	s.snapshot.Targets["a"] = targetSnapshot{ConfigID: c.targetFingerprint(c.Clusters[0]), Parts: map[string]inventoryPart{
		"ingresses":  {Data: []map[string]any{ingress("dashboard"), ingress("same-url"), discovered}, LastSuccessAt: &at},
		"httproutes": {LastSuccessAt: &at}, "services": {LastSuccessAt: &at},
		manualComponent: {Data: manual, LastSuccessAt: &at, Documents: map[string]manualDocument{"central/links": {Data: manual, LastSuccessAt: &at}}},
	}}
	path := filepath.Join(t.TempDir(), "directory.yaml")
	t.Setenv("AUTODOCS_DIRECTORY_CONFIG", path)
	write := func(value string) {
		t.Helper()
		tmp := path + ".new"
		if err := os.WriteFile(tmp, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(tmp, path); err != nil {
			t.Fatal(err)
		}
	}
	read := func(service *inventoryService, view string) (map[string]map[string]any, string) {
		t.Helper()
		w := httptest.NewRecorder()
		service.dataHandler("directory")(w, httptest.NewRequest("GET", "/api/v1/auto-docs-dashboard"+view, nil))
		var rows []map[string]any
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &rows) != nil {
			t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
		}
		result := map[string]map[string]any{}
		for _, row := range rows {
			result[rowString(row, "name")] = row
		}
		return result, w.Header().Get(warningsHeader)
	}
	assertCurated := func(rows map[string]map[string]any, ranks map[string]float64, features map[string]map[string]any) {
		t.Helper()
		if len(rows) != len(ranks) {
			t.Fatalf("unexpected rows: %+v", rows)
		}
		for name, rank := range ranks {
			row := rows[name]
			actual, _ := row["directoryGroupRank"].(float64)
			links, _ := row["featuredLinkRanks"].(map[string]any)
			if row == nil || actual != rank || !reflect.DeepEqual(links, features[name]) {
				t.Fatalf("%s: group=%v, links=%v", name, actual, links)
			}
		}
	}
	defaultRanks := map[string]float64{"dashboard": 0, "same-url": 0, "Guide": 0}
	rows, warning := read(s, "")
	assertCurated(rows, defaultRanks, nil)
	if !strings.Contains(warning, "Directory settings") || strings.Contains(warning, path) {
		t.Fatalf("missing or unsanitized warning: %s", warning)
	}
	settings := "groupOrder: [Absent, Media, media]\nfeatured:\n" + directoryTestFeature + `  - manualLink: {clusterId: a, namespace: central, configMapName: links, entryId: guide}
    url: https://example.test/first
`
	write(settings)
	ranks := map[string]float64{"dashboard": 3, "same-url": 3, "Guide": 2}
	features := map[string]map[string]any{"dashboard": {"https://example.test/second": float64(1)}, "Guide": {"https://example.test/first": float64(2)}}
	rows, warning = read(s, "?view=published")
	assertCurated(rows, ranks, features)
	if warning != "" {
		t.Fatalf("unexpected warning: %s", warning)
	}
	rows, warning = read(s, "?view=discovered")
	assertCurated(rows, map[string]float64{"draft": 0}, nil)
	if warning != "" {
		t.Fatalf("unexpected Discovered warning: %s", warning)
	}
	if err := s.persist(s.snapshot); err != "" {
		t.Fatal(err)
	}
	rows, _ = read(newInventoryService(c), "")
	assertCurated(rows, ranks, features)

	write("groupOrder: [media]\nfeatured:\n" + strings.Replace(settings[strings.Index(settings, "  - manualLink:"):], "/first", "/changed", 1))
	rows, _ = read(s, "")
	assertCurated(rows, map[string]float64{"dashboard": 1, "same-url": 1, "Guide": 0}, nil)
	write("groupOrder: [media, media]\nunknown: sensitive-value")
	rows, warning = read(s, "")
	assertCurated(rows, defaultRanks, nil)
	if !strings.Contains(warning, "Directory settings") || strings.Contains(warning, "sensitive-value") {
		t.Fatalf("unexpected warning: %s", warning)
	}
	_, warning = read(s, "?view=discovered")
	if warning != "" {
		t.Fatalf("settings warning leaked to Discovered: %s", warning)
	}
	t.Setenv("AUTODOCS_DIRECTORY_CONFIG", "")
	rows, warning = read(s, "")
	assertCurated(rows, defaultRanks, nil)
	if warning != "" {
		t.Fatalf("unexpected warning with unset configuration: %s", warning)
	}
}
