package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const manualSourceLabel = "k8sautodocs.io/link-source"
const manualComponent = "manual-links"

type manualDocument struct {
	Data          []map[string]any `json:"data"`
	LastSuccessAt *time.Time       `json:"lastSuccessAt"`
	LastAttemptAt time.Time        `json:"lastAttemptAt"`
	Error         string           `json:"error,omitempty"`
}

// Validate the YAML tree before decoding: YAML's scalar coercions, aliases and
// explicit nulls are not part of the declaration language.
func manualMap(n *yaml.Node, allowed ...string) (map[string]*yaml.Node, error) {
	if n.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected mapping")
	}
	out := map[string]*yaml.Node{}
	for i := 0; i < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		valid := false
		for _, name := range allowed {
			valid = valid || k.Value == name
		}
		if k.Tag != "!!str" || !valid {
			return nil, fmt.Errorf("unknown field")
		}
		if _, exists := out[k.Value]; exists {
			return nil, fmt.Errorf("duplicate field")
		}
		out[k.Value] = v
	}
	return out, nil
}
func manualString(n *yaml.Node) (string, error) {
	if n == nil || n.Kind != yaml.ScalarNode || n.Tag != "!!str" || strings.TrimSpace(n.Value) == "" {
		return "", fmt.Errorf("expected nonempty string")
	}
	return n.Value, nil
}
func validManualURL(raw string) bool {
	if strings.ContainsAny(raw, "\\") || strings.IndexFunc(raw, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 || strings.ContainsAny(raw, " \u007f") {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Opaque != "" {
		return false
	}
	host := u.Hostname()
	if strings.ContainsAny(host, "[]") {
		return false
	}
	if strings.Contains(host, ":") && net.ParseIP(strings.Split(host, "%")[0]) == nil {
		return false
	}
	if strings.HasSuffix(u.Host, ":") {
		return false
	}
	if port := u.Port(); port != "" {
		n, e := strconv.Atoi(port)
		if e != nil || n < 1 || n > 65535 {
			return false
		}
	}
	return true
}
func parseManualLinks(raw string, target targetConfig, namespace, name string, rules map[string]classificationRule) ([]map[string]any, error) {
	if len(raw) > 256<<10 {
		return nil, fmt.Errorf("document exceeds 256 KiB")
	}
	decoder := yaml.NewDecoder(strings.NewReader(raw))
	var doc yaml.Node
	if decoder.Decode(&doc) != nil || len(doc.Content) != 1 {
		return nil, fmt.Errorf("invalid YAML document")
	}
	var extra yaml.Node
	if decoder.Decode(&extra) != io.EOF {
		return nil, fmt.Errorf("expected one YAML document")
	}
	root, err := manualMap(doc.Content[0], "version", "links")
	if err != nil {
		return nil, fmt.Errorf("document: %s", err)
	}
	version := root["version"]
	if version == nil || version.Kind != yaml.ScalarNode || version.Tag != "!!int" || version.Value != "1" {
		return nil, fmt.Errorf("version: expected 1")
	}
	links := root["links"]
	if links == nil || links.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("links: expected list")
	}
	if len(links.Content) > 500 {
		return nil, fmt.Errorf("links: exceeds 500 entries")
	}
	rows := []map[string]any{}
	ids := map[string]bool{}
	for index, node := range links.Content {
		fields, e := manualMap(node, "id", "title", "url", "group", "scope")
		if e != nil {
			return nil, fmt.Errorf("links[%d]: %s", index, e)
		}
		texts := map[string]string{}
		for _, key := range []string{"id", "title", "url", "group"} {
			if key == "group" && fields[key] == nil {
				continue
			}
			texts[key], e = manualString(fields[key])
			if e != nil {
				return nil, fmt.Errorf("links[%d].%s: %s", index, key, e)
			}
		}
		if ids[texts["id"]] {
			return nil, fmt.Errorf("links[%d].id: duplicate ID", index)
		}
		ids[texts["id"]] = true
		if !validManualURL(texts["url"]) {
			return nil, fmt.Errorf("links[%d].url: invalid HTTP(S) destination", index)
		}
		provenance := map[string]string{"clusterId": target.ID, "namespace": namespace, "configMapName": name, "entryId": texts["id"]}
		identity, _ := json.Marshal([]string{"manual-link", target.ID, namespace, name, texts["id"]})
		row := map[string]any{"sourceType": "manual-link", "sourceIdentity": string(identity), "provenance": provenance, "kind": "manual-link", "name": texts["title"], "title": texts["title"], "endpoints": []map[string]any{{"url": texts["url"]}}, "labels": map[string]string{publishLabel: "true"}, "classification": map[string]classifiedValue{}}
		if texts["group"] != "" {
			row["labels"].(map[string]string)["k8sautodocs.io/group"] = texts["group"]
		}
		if fields["scope"] != nil {
			scope, e := manualMap(fields["scope"], "clusterId", "namespace", "environment", "datacentre")
			if e != nil {
				return nil, fmt.Errorf("links[%d].scope: %s", index, e)
			}
			for _, key := range []string{"clusterId", "namespace", "environment", "datacentre"} {
				if scope[key] == nil {
					continue
				}
				value, e := manualString(scope[key])
				if e != nil {
					return nil, fmt.Errorf("links[%d].scope.%s: %s", index, key, e)
				}
				if key == "clusterId" || key == "namespace" {
					row[key] = value
					continue
				}
				rule, ok := rules[key]
				if !ok {
					rule = classificationRule{CaseSensitive: true}
				}
				row["classification"].(map[string]classifiedValue)[key] = normalizeClassification(rule, value)
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}
func (s *inventoryService) collectManual(ctx context.Context, t targetConfig, c targetClients) collectionResult {
	started := time.Now()
	list, err := c.Kube.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{LabelSelector: manualSourceLabel + "=true"})
	result := collectionResult{Err: err, Documents: map[string]manualDocument{}}
	if err == nil {
		for _, cm := range list.Items {
			if cm.Labels[manualSourceLabel] != "true" {
				continue
			}
			rows, err := parseManualLinks(cm.Data["links.yaml"], t, cm.Namespace, cm.Name, s.config.Classification)
			document := manualDocument{Data: rows}
			if err != nil {
				document.Error = err.Error()
			}
			result.Documents[cm.Namespace+"/"+cm.Name] = document
		}
	}
	result.Duration = time.Since(started)
	return result
}
func mergeManualDocuments(part inventoryPart, result collectionResult, at time.Time) inventoryPart {
	previous := part.Documents
	part.Documents = map[string]manualDocument{}
	part.Data = []map[string]any{}
	part.Warnings = nil
	part.Error = ""
	part.LastAttemptAt = at
	partial := false
	for key, doc := range result.Documents {
		if doc.Error != "" {
			old := previous[key]
			doc.Data = old.Data
			doc.LastSuccessAt = old.LastSuccessAt
			partial = true
			part.Warnings = append(part.Warnings, key+": "+doc.Error)
		} else {
			success := at
			doc.LastSuccessAt = &success
		}
		doc.LastAttemptAt = at
		part.Documents[key] = doc
		part.Data = append(part.Data, doc.Data...)
	}
	if partial {
		// The component timestamp is the last fully valid collection; document
		// timestamps describe mixed results without resetting that history.
		part.Error = "manual declaration validation incomplete"
	} else {
		part.LastSuccessAt = &at
	}
	return part
}

// Re-resolve display names on each generation, without using storage as scope.
func resolveManualClusters(targets map[string]targetSnapshot, registry []resolvedTarget) {
	names := map[string]string{}
	for _, t := range registry {
		names[t.ID] = t.Name
	}
	for _, target := range targets {
		part, exists := target.Parts[manualComponent]
		if !exists {
			continue
		}
		part.Data = []map[string]any{}
		for _, doc := range part.Documents {
			for _, row := range doc.Data {
				delete(row, "issues")
				if id, ok := row["clusterId"].(string); ok {
					row["clusterName"] = names[id]
					if names[id] == "" {
						row["clusterName"] = id
						row["issues"] = []string{"Declared cluster ID is not in the current registry"}
					}
				}
				issue := doc.Error
				if issue == "" && part.Error != "manual declaration validation incomplete" {
					issue = part.Error
				}
				row["collection"] = map[string]any{"lastSuccessAt": doc.LastSuccessAt, "lastAttemptAt": doc.LastAttemptAt, "error": issue}
				part.Data = append(part.Data, row)
			}
		}
		target.Parts[manualComponent] = part
	}
}
