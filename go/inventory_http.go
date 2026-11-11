package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (s *inventoryService) json(w http.ResponseWriter, status int, value any) {
	writeJSON(w, status, value, s.logger, "inventory")
}
func (s *inventoryService) status() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	targets := []map[string]any{}
	dimensions := map[string]map[string]string{"environment": {}, "datacentre": {}}
	for _, t := range s.registryLocked() {
		parts := map[string]any{}
		for _, key := range []string{"ingresses", "httproutes", "services", "workloads", "filtered"} {
			p := s.snapshot.Targets[t.ID].Parts[key]
			values := map[string]map[string]bool{"environment": {}, "datacentre": {}}
			for _, row := range p.Data {
				classifications := classificationOf(row)
				for name := range values {
					if v, ok := classifications[name]; ok {
						values[name][v.ID] = true
						dimensions[name][v.ID] = v.Display
					} else {
						values[name]["\x00"] = true
					}
				}
			}
			parts[key] = map[string]any{"lastSuccessAt": p.LastSuccessAt, "lastAttemptAt": p.LastAttemptAt, "error": p.Error, "warnings": p.Warnings, "count": len(p.Data), "values": values}
		}
		targets = append(targets, map[string]any{"id": t.ID, "name": t.Name, "parts": parts, "primary": t.Primary, "connections": t.Connections, "warnings": t.Warnings})
	}
	sources := []map[string]any{}
	for _, src := range s.config.Ranchers {
		v := s.snapshot.Sources[src.ID]
		name := src.Name
		if name == "" {
			name = src.ID
		}
		sources = append(sources, map[string]any{"id": src.ID, "name": name, "lastSuccessAt": v.LastSuccessAt, "lastAttemptAt": v.LastAttemptAt, "error": v.Error, "restored": v.Restored, "memberCount": len(v.Members)})
	}
	return map[string]any{"sources": sources, "completedSources": s.completedSources, "totalSources": len(s.config.Ranchers), "currentSource": s.currentSource, "generation": s.snapshot.Generation, "serverTime": s.now(), "running": s.running, "completed": s.completed, "total": s.total, "currentCluster": s.current, "lastFinishedAt": s.lastFinished, "nextAutomaticAt": s.nextAuto, "nextManualAt": s.lastFinished.Add(duration(s.config.Refresh.Cooldown)), "cacheError": s.cacheError, "clusters": targets, "dimensions": dimensions, "legacyClusterID": s.config.LegacyClusterID}
}
func (s *inventoryService) statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "GET required", 405)
		return
	}
	s.json(w, 200, s.status())
}
func (s *inventoryService) refreshHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "POST required", 405)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host != r.Host || (u.Scheme != "http" && u.Scheme != "https") {
			http.Error(w, "same-origin request required", 403)
			return
		}
	}
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		http.Error(w, "same-origin request required", 403)
		return
	}
	var body struct {
		Generation *uint64 `json:"generation"`
	}
	d := json.NewDecoder(io.LimitReader(r.Body, 1025))
	d.DisallowUnknownFields()
	if err := d.Decode(&body); err != nil && err != io.EOF {
		http.Error(w, "invalid refresh request", 400)
		return
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		http.Error(w, "invalid refresh request", 400)
		return
	}
	s.request(body.Generation, false)
	s.json(w, http.StatusAccepted, s.status())
}
func (s *inventoryService) dataHandler(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "GET required", 405)
			return
		}
		view := publishedView
		if kind == "directory" {
			values := r.URL.Query()["view"]
			if len(values) > 1 {
				http.Error(w, "view must be specified at most once", 400)
				return
			}
			if len(values) == 1 && values[0] != "" {
				view = directoryView(values[0])
			}
			if view != publishedView && view != discoveredView {
				http.Error(w, "view must be published or discovered", 400)
				return
			}
		}
		s.mu.RLock()
		defer s.mu.RUnlock()
		keys := []string{kind}
		if kind == "directory" {
			keys = []string{"ingresses", "httproutes", "services"}
		}
		rows := []map[string]any{}
		warnings := []string{}
		usable := false
		if s.snapshot.Generation > 0 && len(s.registryLocked()) == 0 {
			for _, source := range s.snapshot.Sources {
				usable = usable || source.LastSuccessAt != nil
			}
		}
		for _, source := range s.config.Ranchers {
			state := s.snapshot.Sources[source.ID]
			if state.Error != "" {
				warnings = append(warnings, "Rancher "+source.ID+": "+state.Error)
			} else if state.LastSuccessAt == nil {
				warnings = append(warnings, "Rancher "+source.ID+": discovery pending")
			}
		}
		for _, t := range s.registryLocked() {
			for _, key := range keys {
				part := s.snapshot.Targets[t.ID].Parts[key]
				if part.LastSuccessAt != nil {
					usable = true
				}
				if part.Error != "" {
					warnings = append(warnings, t.Name+" / "+key+": "+part.Error+"; showing last successful inventory if available")
				} else if part.LastSuccessAt == nil {
					warnings = append(warnings, t.Name+" / "+key+": never collected; discovery pending")
				}
				for _, warning := range part.Warnings {
					warnings = append(warnings, t.Name+" / "+key+": "+warning)
				}
				for _, row := range part.Data {
					if kind == "directory" && !view.includes(labelsOf(row)) {
						continue
					}
					rows = append(rows, row)
				}
			}
		}
		if !usable {
			s.json(w, 503, map[string]string{"error": "Inventory is not available yet; discovery runs in the background."})
			return
		}
		sort.SliceStable(rows, func(i, j int) bool {
			for _, key := range []string{"clusterId", "namespace", "kind", "name"} {
				a, _ := rows[i][key].(string)
				b, _ := rows[j][key].(string)
				if a != b {
					return a < b
				}
			}
			return false
		})
		w.Header().Set("X-AutoDocs-Generation", strconv.FormatUint(s.snapshot.Generation, 10))
		w.Header().Set("Access-Control-Expose-Headers", warningsHeader+", X-AutoDocs-Generation")
		if len(warnings) > 0 {
			encoded, _ := json.Marshal(warnings)
			if len(encoded) > 12000 {
				encoded, _ = json.Marshal([]string{"Inventory incomplete; see discovery status for per-cluster details."})
			}
			w.Header().Set(warningsHeader, string(encoded))
		}
		if kind == "workloads" && strings.HasSuffix(r.URL.Path, "service-status") {
			result := map[string]any{"workloads": rows, "ready": 0, "attention": 0, "inactive": 0}
			var oldest *time.Time
			for _, target := range s.snapshot.Targets {
				p := target.Parts["workloads"]
				if p.LastSuccessAt != nil && (oldest == nil || p.LastSuccessAt.Before(*oldest)) {
					oldest = p.LastSuccessAt
				}
			}
			result["checkedAt"] = oldest
			for _, row := range rows {
				state, _ := row["state"].(string)
				if n, ok := result[state].(int); ok {
					result[state] = n + 1
				}
			}
			s.json(w, 200, result)
			return
		}
		s.json(w, 200, rows)
	}
}
func (s *inventoryService) clusterHandler(w http.ResponseWriter, r *http.Request) {
	status := s.status()
	name := "Multiple clusters"
	if clusters, ok := status["clusters"].([]map[string]any); ok && len(clusters) == 1 {
		name, _ = clusters[0]["name"].(string)
	}
	s.json(w, 200, map[string]any{"clusterName": name, "clusters": status["clusters"]})
}
