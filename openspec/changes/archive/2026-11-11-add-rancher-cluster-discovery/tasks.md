## 1. Verify Rancher and identity contracts

Implementation resumed after the supplied development credential verified Rancher
2.15.2 enumeration and proxy reads. See [verification evidence](verification.md)
for source-derived permissions, fixtures and limits of live qualification.

- [x] 1.1 Record supported Rancher release/API evidence: RK-API resources covering imported and provisioned clusters, management IDs, display names, pagination and credential visibility. Document incompatibility rather than silently adding legacy-v3 fallback.
- [x] 1.2 Verify downstream proxy paths, supported bearer-token behavior, verified TLS and minimal read permissions for enumeration and existing collectors; create sanitized response fixtures without retaining credentials.
- [x] 1.3 Validate the proposed `kube-system` UID probe through direct and Rancher connections, including denied access and clone/restore caveats; record the selected signal and limits in the design.
- [x] 1.4 Resolve source/aggregate count, response-size and timeout budgets using representative fixture measurements; specify overflow as discovery failure rather than truncation.

## 2. Model sources, connections and canonical clusters

- [x] 2.1 Extend configuration with Rancher sources, exclusions and explicit canonical bindings/primary references; preserve existing direct-only and no-config behavior, and permit Rancher-only startup with zero discovered clusters.
- [x] 2.2 Add validation fixtures for duplicate source/canonical IDs, malformed references, conflicting assignments, excluded bound connections, invalid primaries and temporarily undiscovered Rancher members.
- [x] 2.3 Introduce deterministic connection/canonical identities and an immutable resolved-registry model; preserve existing direct IDs and canonical labels/classification when connections change.
- [x] 2.4 Separate canonical identity from cache compatibility and primary provenance; specify direct version-1 snapshot compatibility and stable-ID migration for existing overlaps.

## 3. Discover and connect through Rancher

- [x] 3.1 Implement source kubeconfig loading with coherent per-refresh reads, bearer-only Rancher authentication, verified TLS, credential rotation and sanitized failures.
- [x] 3.2 Implement bounded complete enumeration using the verified supported API contract, with exclusions, pagination, imported/provisioned coverage and explicit API incompatibility reporting.
- [x] 3.3 Construct downstream proxy clients for existing collectors; enforce trusted-origin redirects/pagination and validated route identifiers without direct-cluster fallback or kubeconfig generation.
- [x] 3.4 Add fake-server fixtures for credential replacement, missing files, unsupported auth, pagination failures, source limits, untrusted destinations and proxy collection without direct connectivity.

## 4. Detect and resolve overlap

- [x] 4.1 Implement bounded per-generation identity observations and possible-overlap, unavailable-check and binding-conflict outcomes; do not infer equivalence from names or endpoint strings.
- [x] 4.2 Resolve explicit bindings to one primary and restrict secondaries to identity probes; preserve independent collection for unresolved connections and prohibit automatic failover/union.
- [x] 4.3 Invalidate inventory on primary or observed underlying identity changes while preserving canonical source/link IDs; keep ordinary credential rotation compatible with existing semantics.
- [x] 4.4 Add fixtures for matching markers, missing/forbidden/stale observations, cloned-marker ambiguity, conflicts, primary outages, primary switches with reduced permissions and duplicate-ID preference boundaries.

## 5. Reconcile membership and persist snapshots

- [x] 5.1 Integrate source discovery and resolved-registry collection into the shared coordinator, retaining coalescing/cooldown and independent direct/source progress.
- [x] 5.2 Reconcile additions/removals only from complete listings; retain stale membership on failures, skip failed-source proxy work, preserve direct primaries and report absent primaries without failover.
- [x] 5.3 Publish membership and data atomically as one generation; update data/status projections to use canonical resolved targets instead of static `config.Clusters` iteration.
- [x] 5.4 Introduce versioned sanitized membership/provenance persistence and compatible version-1 direct recovery; reject removed, excluded, reassigned or incompatible saved targets.
- [x] 5.5 Add recovery/race fixtures for empty and partial listings, source deletion, cache failure, restart during outage, primary disappearance, configuration changes and readers during reconciliation.

## 6. Expose discovery and connection status

- [x] 6.1 Add sanitized source enumeration timestamps/errors, connection roles, primary availability and overlap/identity diagnostics to discovery status while preserving existing inventory API shapes.
- [x] 6.2 Extend shared UI controls with accessible source/connection status and actionable possible-duplicate/conflict warnings; keep unresolved inventories separately attributable and provide configuration guidance without a credential form.
- [x] 6.3 Verify canonical identities across directory, overview, versions, grouping/filtering and favourites; add browser fixtures for unresolved overlaps, resolution, primary changes, unavailable probes and source-versus-collection failures.
- [x] 6.4 Add disclosure checks ensuring credentials, raw kubeconfigs and credential paths never appear in persisted snapshots, logs or HTTP responses.

## 7. Validate and document implemented behavior

- [x] 7.1 Run backend tests and race checks, static browser tests and the applicable container smoke checks; record meaningful results and any unavailable external verification.
- [x] 7.2 Update README and CHANGELOG for implemented source configuration, verified API/version/RBAC support, Secret directory mounts, exclusions, overlap binding, canonical-ID migration and rollback/cache compatibility.
- [x] 7.3 Document the homelab example with its existing direct hosting-cluster ID as primary and Rancher's `local` as secondary; leave actual credential creation, image publication and GitOps rollout to a separately authorized deployment task.
- [x] 7.4 Reconcile against the multi-cluster baseline if its currently unarchived specs have moved, run strict OpenSpec validation and review all requirement scenarios against the completed implementation.

Validation results and external qualification limits are recorded in [validation.md](validation.md). Container smoke execution was unavailable because the Docker daemon is absent; no runtime success is claimed.
