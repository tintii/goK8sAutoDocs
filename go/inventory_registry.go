package main

import (
	"context"
	"errors"
	"strings"

	"k8s.io/client-go/rest"
)

type refreshConnection struct {
	clients targetClients
	err     error
	marker  string
}

func safeSourceIssue(err error) string {
	if err == nil {
		return ""
	}
	for _, s := range []string{"credentials unavailable", "invalid or unsupported credentials", "Rancher enumeration unavailable", "Rancher authentication rejected", "Rancher enumeration forbidden; check token scope and management read permissions", "Rancher RK-API incompatible or unavailable", "Rancher RK-API incompatible response", "Rancher discovery budget exceeded", "Rancher aggregate discovery budget exceeded", "Rancher invalid cluster response", "Rancher invalid pagination", "Rancher canonical ID collision"} {
		if err.Error() == s {
			return s
		}
	}
	return "Rancher enumeration unavailable"
}
func (s *inventoryService) prepareRegistry(next *inventorySnapshot) ([]resolvedTarget, map[string]refreshConnection) {
	sources := map[string]rancherSnapshot{}
	configs := map[string]*rest.Config{}
	for _, src := range s.config.Ranchers {
		old := next.Sources[src.ID]
		if old.ConfigID == sourceFingerprint(src) {
			sources[src.ID] = old
		}
	}
	for _, src := range s.config.Ranchers {
		s.mu.Lock()
		s.currentSource = src.ID
		s.mu.Unlock()
		state := sources[src.ID]
		state.ConfigID = sourceFingerprint(src)
		state.LastAttemptAt = s.now()
		state.Restored = false
		ctx, cancel := context.WithTimeout(s.ctx, sourceTimeout)
		members, cfg, err := s.enumerate(ctx, src)
		cancel()
		if err == nil {
			trial := state
			trial.Members = members
			trial.EndpointID = sourceEndpoint(cfg)
			trial.Error = ""
			sources[src.ID] = trial
			if _, e := s.config.resolveRegistry(sources); e != nil {
				err = e
			}
		}
		if err != nil {
			state.Error = safeSourceIssue(err)
			sources[src.ID] = state
			s.logger.Warn("Rancher discovery failed", "event", "source.discovery_failed", "source_id", logIdentity(src.ID), "reason", state.Error)
		} else {
			state = sources[src.ID]
			at := s.now()
			state.LastSuccessAt = &at
			configs[src.ID] = cfg
			sources[src.ID] = state
		}
		s.mu.Lock()
		s.completedSources++
		s.mu.Unlock()
	}
	s.mu.Lock()
	s.currentSource = ""
	s.mu.Unlock()
	next.Sources = sources
	registry, err := s.config.resolveRegistry(sources)
	if err != nil { // Validated config plus bounded authoritative membership must resolve.
		s.logger.Warn("Registry resolution failed", "event", "source.discovery_failed", "reason", safeSourceIssue(err))
		registry, _ = s.config.resolveRegistry(nil)
	}
	runtime := map[string]refreshConnection{}
	for _, t := range s.config.Clusters {
		r := connectionRef{Direct: t.ID}
		clients, e := s.connect(t)
		runtime[r.key()] = refreshConnection{clients: clients, err: e}
	}
	for _, t := range registry {
		for _, c := range t.Connections {
			r := c.Ref
			if r.Direct != "" {
				continue
			}
			if _, ok := runtime[r.key()]; ok {
				continue
			}
			cfg := configs[r.Rancher]
			if !c.Available || cfg == nil {
				runtime[r.key()] = refreshConnection{err: errors.New("primary connection unavailable")}
				continue
			}
			clients, e := proxyClients(cfg, r.ClusterID)
			runtime[r.key()] = refreshConnection{clients: clients, err: e}
		}
	}
	// Only observations from this refresh participate in diagnostics.
	ctx, cancel := context.WithTimeout(s.ctx, identityGenerationTimeout)
	defer cancel()
	for ti := range registry {
		for ci := range registry[ti].Connections {
			c := &registry[ti].Connections[ci]
			v, ok := runtime[c.Ref.key()]
			c.Available = c.Available && ok && v.err == nil
			if !c.Available {
				continue
			}
			if ctx.Err() == nil {
				v.marker, _ = s.probe(ctx, v.clients)
				runtime[c.Ref.key()] = v
			}
			if v.marker != "" {
				c.Identity = "observed"
			}
		}
	}
	for i := range registry {
		markers := map[string]bool{}
		for _, c := range registry[i].Connections {
			if m := runtime[c.Ref.key()].marker; m != "" {
				markers[m] = true
			}
		}
		if len(markers) > 1 {
			registry[i].Warnings = append(registry[i].Warnings, "Binding identity conflict: member observations differ. Only the configured primary is collected; review clusterBindings.")
		}
		for _, c := range registry[i].Connections {
			if c.Identity == "unavailable" {
				registry[i].Warnings = append(registry[i].Warnings, "Identity check unavailable for "+c.Ref.key()+"; overlap cannot be checked for this connection.")
			}
		}
	}
	for i := range registry {
		for j := i + 1; j < len(registry); j++ {
			match := false
			for _, a := range registry[i].Connections {
				for _, b := range registry[j].Connections {
					m := runtime[a.Ref.key()].marker
					if m != "" && m == runtime[b.Ref.key()].marker {
						match = true
					}
				}
			}
			if match {
				registry[i].Warnings = append(registry[i].Warnings, "Possible overlap with "+registry[j].ID+": resources may appear twice. Configure clusterBindings with one canonical ID and primary; clone/restore can also duplicate identity markers.")
				registry[j].Warnings = append(registry[j].Warnings, "Possible overlap with "+registry[i].ID+": resources may appear twice. Configure clusterBindings with one canonical ID and primary; clone/restore can also duplicate identity markers.")
			}
		}
	}
	active := map[string]targetSnapshot{}
	for _, t := range registry {
		if old, ok := next.Targets[t.ID]; ok {
			active[t.ID] = old
		}
	}
	next.Targets = active
	next.Registry = registry
	if len(s.config.Ranchers) > 0 || len(s.config.ClusterBindings) > 0 {
		next.Version = 2
	}
	s.mu.Lock()
	s.total = len(registry)
	s.mu.Unlock()
	return registry, runtime
}

// Call with the service read lock held. Old version-1 fixtures and empty startup
// snapshots have no registry; configured direct targets still have status then.
func (s *inventoryService) registryLocked() []resolvedTarget {
	if s.snapshot.Registry != nil {
		return s.snapshot.Registry
	}
	targets, _ := s.config.resolveRegistry(s.snapshot.Sources)
	return targets
}
func restoredSourceValid(src rancherConfig, state rancherSnapshot) bool {
	if state.ConfigID != sourceFingerprint(src) || len(state.Members) > maxSourceClusters {
		return false
	}
	seen := map[string]bool{}
	for _, m := range state.Members {
		if !routeID.MatchString(m.ID) || seen[m.ID] || excluded(src, m.ID) || len(m.Name) > 512 || strings.ContainsAny(m.Name, "\x00\r\n") {
			return false
		}
		seen[m.ID] = true
	}
	// Validate current trust when credentials are available; absent/broken mounts
	// may still restore compatible stale data, but never clients or authentication.
	if cfg, err := loadRancher(src); err == nil && state.EndpointID != sourceEndpoint(cfg) {
		return false
	}
	return true
}
