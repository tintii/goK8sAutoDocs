## 1. Registry and credential loading

- [x] 1.1 Add and validate the `AUTODOCS_CONFIG` schema for stable cluster targets, explicit connection methods/contexts, classification settings, refresh timing, cache path and legacy preference target; preserve explicit single-target compatibility when configuration is absent.
- [x] 1.2 Replace global client ownership with per-target typed/dynamic clients and add self-contained token/certificate kubeconfig validation before client creation; sanitize credential-related errors.
- [x] 1.3 Reload mounted kubeconfigs coherently at the start of each refresh and isolate missing/invalid/rotated credentials without falling back to another cluster or retaining superseded authentication.
- [x] 1.4 Verify duplicate IDs, repeated internal context names across files, both supported authentication forms, unsupported exec/file/plugin/impersonation settings, missing files and rotation with fixture-only tests.

## 2. Classification and cluster-aware collection

- [x] 2.1 Implement independent environment/datacentre label resolution with ordered keys/sources, exact label-key lookup, configurable case equality, independent deterministic display casing and typed missing values.
- [x] 2.2 Add per-run namespace metadata lookup reuse and provenance/diagnostics; attach stable cluster context and resolved classifications to directory/workload/filtered inventory data.
- [x] 2.3 Refactor collection to return explicit per-target/component and directory source-kind outcomes, share collected source data between published/discovered, reuse top-level lists where practical and retain all existing endpoint/publication rules.
- [x] 2.4 Verify multi-cluster identical source names, cross-cluster environment membership, arbitrary environment names, case collisions, absent/forbidden labels, deterministic precedence and own-target dependency resolution.

## 3. Shared refresh and snapshot APIs

- [x] 3.1 Implement a service-owned coordinator with one active run, bounded target timeouts, startup/periodic scheduling, observed-generation coalescing, manual cooldown and progress without per-view or per-user jobs.
- [x] 3.2 Publish immutable snapshot generations with per-component successful timestamps and retained prior partitions on failed lists; distinguish successful empty results, optional API absence and no usable initial snapshot.
- [x] 3.3 Make all inventory GET endpoints serve snapshot projections only, retain array/warnings/view-validation compatibility and derive cluster-info from registry state.
- [x] 3.4 Add no-store discovery-status and same-origin POST discovery-refresh APIs with shared run/cooldown/generation state; ensure a browser disconnect cannot cancel the shared job.
- [x] 3.5 Verify concurrent viewers/manual/timer overlap, late duplicate clicks, failed-target isolation, failed source-kind retention, empty successful replacement, browser cancellation and zero Kubernetes collection from GET handlers using deterministic clocks/counters.

## 4. Durable local snapshots

- [x] 4.1 Add a bounded versioned snapshot format containing derived inventory, compatibility metadata, original success timestamps and diagnostics while excluding raw lists, ConfigMap bodies and credentials.
- [x] 4.2 Restore compatible configured target partitions before startup refresh, exclude removed/changed targets, and handle corrupt/incompatible/oversized files without preventing startup.
- [x] 4.3 Persist new snapshots using restrictive temporary files and atomic replacement; expose persistence errors while keeping the memory snapshot available.
- [x] 4.4 Verify restart hydration, stale original timestamps, removed/changed targets, classification compatibility, interrupted/failed writes and absence of known credential/raw-data fixture markers on disk.

## 5. Shared UI scope, grouping and preferences

- [x] 5.1 Extend the shared directory model and validated version-2 preferences for cluster-aware sources/links/namespaces, canonical dimension groups and typed nested paths; migrate legacy records only through the explicit original cluster mapping.
- [x] 5.2 Add URL-restorable cluster/environment/datacentre scope and classification search to Directory, Versions and Overview favourites; preserve unavailable selections, existing search/favourites and accurate scoped counters.
- [x] 5.3 Add Cluster, Environment, Datacentre, Environment then Cluster and Cluster then Environment grouping with best-effort fallback sections and visible provenance; preserve label/custom semantics and versions sorting/resizing.
- [x] 5.4 Integrate shared freshness/countdown/progress/cooldown controls with visibility-aware status polling and generation-triggered data reload, keeping the current application-availability indicator and overview favourites intact.
- [x] 5.5 Verify identity isolation, legacy migration/dormant records, case-normalized groups, missing dimensions, cross-page scope/history, shared update visibility, clock skew, partial/stale states and late view responses with meaningful model/browser fixtures.

## 6. Integrated validation and packaging

- [x] 6.1 Run Go tests and race tests plus the complete applicable JavaScript/browser checks; resolve failures caused by the implementation without changing unrelated user work.
- [x] 6.2 Extend and run container smoke checks for explicit fixture registry, multiple fixture targets, application health independent of cluster reachability, credential mounts, writable cache with read-only root/non-root runtime and required local assets.
- [x] 6.3 Review representative desktop/mobile UI states for scope, missing metadata, stale/mixed results, refresh progress and overview favourites; verify no regression to the superseded homepage workload-health layout.

## 7. Documentation after apply

- [x] 7.1 AFTER implementation tasks 1–6 are complete and checks pass, update README and its changelog with the actual registry/classification/Secret schemas, supported authentication and required RBAC, rotation/renewal ownership, global refresh/cooldown/timer semantics, cache/PVC behaviour, scope/grouping, preference migration and single-active-instance limit.
- [x] 7.2 AFTER implementation, reconcile the proposal/design/specs with the implemented contract, record validation evidence and link the companion GitOps rollout without claiming an unpublished image or unconfigured remote target is live.

## 8. Publication and deployment handoff

- [ ] 8.1 Publish the reviewed tested application through its existing image workflow as authorized for deployment, preserving unrelated working-tree changes; verify the exact source revision, required native architecture, image digest and pull access.
- [ ] 8.2 Execute the linked `../homelab-gitops/openspec/changes/deploy-autodocs-multicluster-snapshots` change using that verified digest; do not configure illustrative remote clusters without real credential references.
- [ ] 8.3 After live verification, record actual release/deployment results and finish any post-rollout documentation corrections; leave unperformed checks or unavailable real-target inputs explicitly outstanding.
