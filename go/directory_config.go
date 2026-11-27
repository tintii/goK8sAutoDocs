package main

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"os"

	"gopkg.in/yaml.v3"
)

type directorySource struct {
	manual                   bool
	cluster, kind, namespace string
	name, entry              string
}

type directoryFeature struct {
	source directorySource
	url    string
}

type directoryConfig struct {
	features map[directoryFeature]int
	groups   map[string]int
}

func loadDirectoryConfig(path string) (directoryConfig, error) {
	config := directoryConfig{}
	if path == "" {
		return config, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return config, err
	}
	defer file.Close()
	const limit = 64 << 10
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || len(data) > limit {
		return config, fmt.Errorf("directory settings must be at most 64 KiB")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var doc, extra yaml.Node
	if decoder.Decode(&doc) != nil || len(doc.Content) != 1 || decoder.Decode(&extra) != io.EOF {
		return config, fmt.Errorf("expected one YAML document")
	}
	fields, err := manualMap(doc.Content[0], "featured", "groupOrder")
	if err != nil {
		return config, err
	}
	for _, node := range fields {
		if node.Kind != yaml.SequenceNode {
			return config, fmt.Errorf("expected a list")
		}
	}

	features := map[directoryFeature]int{}
	if list := fields["featured"]; list != nil {
		for i, node := range list.Content {
			feature, err := parseDirectoryFeature(node)
			if err != nil {
				return config, fmt.Errorf("featured entry %d: %w", i+1, err)
			}
			if features[feature] != 0 {
				return config, fmt.Errorf("duplicate featured link")
			}
			features[feature] = i + 1
		}
	}
	groups := map[string]int{}
	if list := fields["groupOrder"]; list != nil {
		for i, node := range list.Content {
			value, err := manualString(node)
			if err != nil || groups[value] != 0 {
				return config, fmt.Errorf("expected a unique, nonempty group value")
			}
			groups[value] = i + 1
		}
	}
	return directoryConfig{features: features, groups: groups}, nil
}

func parseDirectoryFeature(node *yaml.Node) (directoryFeature, error) {
	feature := directoryFeature{}
	fields, err := manualMap(node, "source", "manualLink", "url")
	if err != nil {
		return feature, err
	}
	feature.url, err = manualString(fields["url"])
	if err != nil || !validManualURL(feature.url) {
		return feature, fmt.Errorf("expected an absolute HTTP(S) URL")
	}
	source, manual := fields["source"], fields["manualLink"]
	if (source == nil) == (manual == nil) {
		return feature, fmt.Errorf("expected exactly one source or manualLink selector")
	}
	keys := []string{"clusterId", "namespace", "name", "kind"}
	if manual != nil {
		source = manual
		keys = []string{"clusterId", "namespace", "configMapName", "entryId"}
	}
	identity, err := manualMap(source, keys...)
	if err != nil {
		return feature, err
	}
	values := map[string]string{}
	for _, key := range keys {
		values[key], err = manualString(identity[key])
		if err != nil {
			return feature, fmt.Errorf("%s: %w", key, err)
		}
	}
	feature.source = directorySource{cluster: values["clusterId"], namespace: values["namespace"], name: values["name"], kind: values["kind"]}
	if manual != nil {
		feature.source.manual = true
		feature.source.name = values["configMapName"]
		feature.source.entry = values["entryId"]
	} else if feature.source.kind == "Ingress" {
		// The directory API uses the historical lowercase Ingress kind.
		feature.source.kind = "ingress"
	}
	return feature, nil
}

func rowString(row map[string]any, key string) string {
	value, _ := row[key].(string)
	return value
}

func directorySourceOf(row map[string]any) directorySource {
	if row["sourceType"] == "manual-link" {
		switch p := row["provenance"].(type) {
		case map[string]string:
			return directorySource{manual: true, cluster: p["clusterId"], namespace: p["namespace"], name: p["configMapName"], entry: p["entryId"]}
		case map[string]any:
			return directorySource{manual: true, cluster: rowString(p, "clusterId"), namespace: rowString(p, "namespace"), name: rowString(p, "configMapName"), entry: rowString(p, "entryId")}
		}
		return directorySource{manual: true}
	}
	return directorySource{cluster: rowString(row, "clusterId"), kind: rowString(row, "kind"), namespace: rowString(row, "namespace"), name: rowString(row, "name")}
}

func (config directoryConfig) curate(rows []map[string]any) {
	if len(config.groups) == 0 && len(config.features) == 0 {
		return
	}
	for i, row := range rows {
		group := config.groups[labelsOf(row)[groupLabel]]
		var links map[string]int
		if len(config.features) > 0 {
			source := directorySourceOf(row)
			add := func(endpoint map[string]any) {
				url := rowString(endpoint, "url")
				if rank := config.features[directoryFeature{source: source, url: url}]; rank != 0 {
					if links == nil {
						links = map[string]int{}
					}
					links[url] = rank
				}
			}
			switch endpoints := row["endpoints"].(type) {
			case []map[string]any: // Live manual links.
				for _, endpoint := range endpoints {
					add(endpoint)
				}
			case []any: // JSON-backed snapshot rows.
				for _, endpoint := range endpoints {
					if fields, ok := endpoint.(map[string]any); ok {
						add(fields)
					}
				}
			}
		}
		if group == 0 && len(links) == 0 {
			continue
		}
		row = maps.Clone(row)
		if group != 0 {
			row["directoryGroupRank"] = group
		}
		if len(links) > 0 {
			row["featuredLinkRanks"] = links
		}
		rows[i] = row
	}
}
