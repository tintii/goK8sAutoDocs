## Context

AutoDocs uses one global typed/dynamic Kubernetes client pair. Resource handlers own independent 60-second caches protected by mutexes, and refresh inside requests. The directory has strict published/discovered selection and diagnostic behaviour; versions list Deployments, StatefulSets and DaemonSets. The current UI has a shared application-availability indicator and overview favourites; earlier workload-health layouts must not be restored accidentally.

The user requires independent cluster/environment/datacentre dimensions, arbitrary label-derived values, optional case-insensitive matching, best-effort classification, one Secret containing a regular kubeconfig per remote cluster, persistent startup inventory, and a globally coordinated refresh with freshness/countdown UI. A single active process is sufficient. Repository deployment conventions require a published image and chart, GitOps values, no committed credentials, and accurate documentation after implementation and rollout.

## Goals / Non-Goals

**Goals:** Discover multiple independently authenticated clusters, retain source provenance, preserve publication and link-resolution contracts, classify available metadata without excluding unclassified sources, limit Kubernetes traffic independently of browser count, recover prior inventory on restart, and communicate partial/stale results honestly.

**Non-Goals:** Multiple active application replicas or distributed locking, a database, user accounts/permission-filtered inventories, Kubernetes metadata editing, remote credential issuance/renewal, executable cloud-login plugins, automatic cluster discovery from arbitrary workstation contexts, parsing environment/datacentre semantics from cluster names, or replacing the UI framework.

## Decisions

### 1. Explicit file-based cluster registry

Add an `AUTODOCS_CONFIG` mounted YAML configuration. Its cluster list contains unique, non-empty, stable `id` values, optional display `name` and cluster `labels`, and exactly one connection method: `inCluster: true` or `kubeconfig: {path, context}`. `context` is explicit even for a single-context file. Paths are resolved as configured in the container, never by merging files through `KUBECONFIG`. IDs are opaque and are not case-folded; they must not change during ordinary credential rotation or display renaming.

Illustrative configuration contract:

```yaml
clusters:
  - id: homelab
    name: Homelab
    inCluster: true
  - id: k00.npe90.pgt
    kubeconfig:
      path: /etc/autodocs/clusters/k00.npe90.pgt/kubeconfig
      context: autodocs
    labels:
      location: pgt
legacyClusterID: homelab
classification:
  environment:
    labelKeys: [example.invalid/environment]
    sources: [resource, namespace]
    caseSensitive: false
    displayCase: upper
  datacentre:
    labelKeys: [example.invalid/datacentre, location]
    sources: [resource, namespace, cluster]
    caseSensitive: false
    displayCase: lower
refresh:
  interval: 5m
  cooldown: 30s
  clusterTimeout: 30s
cache:
  path: /var/cache/autodocs/snapshot.json
```

The `example.invalid` keys and remote target illustrate the schema; they are not real deployment settings. When no registry configuration is provided, retain single-target startup compatibility using an explicit internal `local` ID and existing in-cluster/local-file credential selection. That compatibility mode never enumerates additional contexts. An explicit registry is authoritative: invalid target credentials must not trigger fallback to another cluster. Validate global syntax, duplicate IDs, methods, durations and classification options before startup; isolate individual credential/connection failures so other configured targets remain usable. Configuration changes take effect through a controlled restart for this version; mounted credential changes reload during refresh.

Alternative: let the application fetch Secret references through Kubernetes. File references also work locally, avoid Secret-read RBAC, and reuse the existing chart's native volume support.

### 2. A precise kubeconfig Secret and reload contract

Each remote target uses one `Opaque` Secret in the hosting AutoDocs namespace, containing key `kubeconfig` with a standard `apiVersion: v1`, `kind: Config` document. The deployment maps that key to a read-only directory file. Use embedded bearer-token authentication or embedded client-certificate/key authentication; embed private CA data when system trust is insufficient. Require an HTTPS endpoint and TLS verification. Reject external credential-file references, `exec`, legacy auth-provider plugins, impersonation and interactive dependencies in the initial contract. Parse/validate before constructing clients so unsupported configurations are rejected without executing helpers.

The selected context must resolve to valid server and user entries; unused contexts are never scanned. Namespace in a kubeconfig is a default, not an inventory scope: collectors keep their explicit namespace behaviour. RBAC on each remote target grants the actual collector reads and namespace metadata access, without Secret reads or writes. Credentials and raw kubeconfigs must never enter snapshots, HTTP responses or logs. Error reporting identifies the target and failure category without serializing credential content.

Mount directories without `subPath`, ensure the non-root application can read them, and allow optional remote Secret mounts so missing credentials do not block the hosting pod. At the start of every refresh, read each target's kubeconfig once, validate that byte snapshot, and replace clients when credentials change; never mix old and new file reads within a target run. Updated malformed/expired credentials mark the target stale; do not silently continue authenticating with a superseded kubeconfig. The supplier/ESO rotates expiring credentials; AutoDocs consumes them and does not renew them. In-cluster clients retain client-go's projected token rotation.

Alternative: one merged kubeconfig for all clusters. Per-target files isolate ownership and rotation and avoid collisions between repeated internal context/user/cluster names.

### 3. Configurable classification with deterministic precedence

Resolve environment and datacentre independently using ordered `sources` and `labelKeys`. For each source in order, choose the first configured key with a non-empty value. `resource` means the documented source itself (Ingress/HTTPRoute for the directory, workload for versions/status); `namespace` means that resource's namespace; `cluster` means configured target labels, not labels from an assumed Kubernetes Cluster object. Cache namespace lookups within a target run. Do not inherit from arbitrary related objects, infer from names, enumerate valid environment values, or truncate NPE/PRD/CORE suffixes.

Label keys use exact lookup. Values use equality controlled by `caseSensitive`; case-insensitive mode derives one deterministic canonical identity for values such as `npe00`, `NPE00` and `Npe00`. `displayCase` is `preserve`, `upper` or `lower` and does not change identity. In preserve mode choose a deterministic observed spelling across the loaded inventory, not collection-arrival order. Canonicalization is shared by grouping, filtering, search indexes, counters and preference IDs, and is performed by the backend. Preserve original source labels for inspection. No fuzzy, substring or regex matching is implied by case-insensitivity.

Missing/empty labels produce no classification value. Unavailable namespace metadata yields diagnostics and permits any usable lower-priority configured fallback, without dropping the resource. Absence of one dimension never invalidates another. Unknown classifications use distinct typed fallback group IDs and are selectable/clearable in filters; a real value named `Unclassified` never collides with the synthetic fallback.

Alternative: assign environments to clusters. A cluster can host multiple environments and an environment can span clusters, so classification belongs to each discovered resource.

### 4. Cluster-aware data, views and preference migration

Add `clusterId`, cluster display context and resolved classification metadata to directory entries, workload inventories and related context. Keep array responses for existing data endpoints. Add `/api/v1/discovery-status` for sanitized registry metadata, per-target/component outcomes, snapshot generation and authoritative timing; add `POST /api/v1/discovery-refresh` for shared requests. Update `/api/v1/cluster-info` to describe registry-backed configured targets rather than reading an unrelated workstation kubeconfig; preserve useful single-target display fields for compatibility.

Source IDs become serialized `[clusterId, kind, namespace, name]` tuples; link IDs retain the exact currently resolved URL in addition to that tuple. Environment/datacentre never enter source identity. Namespace group identities include cluster ID to avoid joining unrelated same-name namespaces; environment and datacentre groups intentionally span clusters using canonical dimension values. Label/custom group cross-cluster semantics remain explicit and retain existing personal behaviour. New grouping modes include cluster, environment, datacentre, environment then cluster, and cluster then environment. Nested group identities include the typed path; equal display names do not transfer stars or collapsed state. The UI presents composite groups as ordered path headings (for example Environment / Cluster), retaining one card per source rather than duplicating it at each level.

Introduce a validated version-2 browser preference envelope. Preserve theme, personal group definitions and versions layout. Migrate old source/link assignments only to the explicitly designated original `legacyClusterID`; never guess by the first returned target, namespace/name, current display text or kubeconfig context. Preserve legacy entries dormant with a clear notice when no unambiguous migration target is configured. Namespace stars/collapse records require the same target scoping; derived label/custom groups retain their documented cross-source semantics. Old clients already treat newer envelopes as session-only, making rollback non-destructive.

Use the shared classification/scope model for directory, versions and overview favourites. Environment/cluster/datacentre filters combine with AND and existing active view/query/namespace/kind/favourites rules. Persist scope in URL parameters across page navigation, reload and back/forward; selected unavailable values remain clearable rather than silently broadened. Scope changes and grouping operate on already-loaded snapshots, never trigger Kubernetes collection. Count namespaces using cluster plus namespace identity. Preserve current overview favourites, safe links, versions sorting/resizing and accessibility.

Inventory GET responses use no-store or explicit generation-validated revalidation, and expose their published generation. A status poll announcing a new generation must not be followed by a browser-cached response from the old generation. This replaces the old unconditional `max-age=30` inventory header.

Alternative: migrate preferences by matching names across all loaded clusters. That would silently attach personal settings to unrelated sources.

### 5. One coordinator with immutable published snapshots

Replace per-handler discovery with a service-lifetime coordinator owning clients, scheduling, immutable published snapshot state and refresh state. HTTP GET handlers only project the current snapshot; they never list Kubernetes resources or start a refresh. This includes filtered-resources and service-status if retained. Published/discovered remain logically distinct projections and warnings, but share a collection generation and classify source publication from one collection pass.

Use a configurable 5-minute automatic interval and 30-second manual cooldown initially, with strictly validated positive durations and bounded target timeouts. There is at most one refresh job across automatic and manual triggers. Process targets sequentially initially; this is a global run, not one run per viewer, page, scope or view. Each run has a server-owned context independent of the requesting browser and performs bounded per-target work. Reuse resource lists/namespace lookups within a run where possible instead of discovering each dashboard independently.

POST returns `202` plus the active run/generation when work starts or is already running. A cooldown response returns the shared current status and next eligible manual time without queuing another job. Accept an observed generation from the client so near-simultaneous requests for an already-replaced snapshot do not start another scan. Automatic ticks during a run coalesce rather than enqueueing work. Set the next automatic due time from completion; a cooldown manual request does not postpone the schedule. Failed runs remain rate-bounded and retry at the advertised next scheduled time. Restrict refresh to same-origin browser requests and do not retain wildcard cross-origin permission on the mutating endpoint.

Perform collection away from the snapshot read lock, build a candidate snapshot, then atomically publish a new generation when the run completes. HTTP readers continue using the previous immutable snapshot throughout. This is an application publication boundary, not a transactional read of Kubernetes. With no usable snapshot, report explicit loading/unavailable state rather than a successful empty inventory. A complete successful empty collection legitimately removes prior objects.

Alternative: keep handler mutexes and extend TTL. That still blocks readers on discovery, duplicates work across views/pages, ties updates to visitors, and cannot truthfully supply the next automatic refresh time.

### 6. Failure isolation and durable startup data

Track success at least per target and collection component; source-kind partitions distinguish Ingress and HTTPRoute collection. A failed list retains the prior successful partition and original `lastSuccessAt`, plus the new attempt/error outcome. Successfully collected partitions replace prior contents, including intentional empty results. Related-resource/namespace lookup issues remain attached diagnostics rather than hiding their source. Missing optional Gateway API is a supported successful absence, unlike forbidden access or an unreachable API. Classify published/discovered from the same retained source partition so a failed refresh cannot duplicate/move an entry inconsistently across views. Preserve previous warnings with retained data and add current failure diagnostics.

Persist only derived endpoint/workload/filtered metadata, classifications, successful timestamps and diagnostics. Do not persist raw Kubernetes lists, ConfigMap bodies or credentials. The file contains schema version, non-secret registry/classification compatibility information and source partitions. Restore compatible configured target partitions at startup and serve them immediately as cached data while starting one background refresh. Remove disabled/removed targets immediately; reject known target-identity/configuration mismatches so a reused ID cannot expose a previous target's inventory. Credential rotation alone does not invalidate otherwise compatible inventory. Incompatible/corrupt/oversized files are ignored with a visible cache diagnostic, followed by fresh collection.

Write to a restrictive-permission temporary file in the cache directory, close/sync it and atomically rename over the prior snapshot. Bound snapshot reads. Persistence failures keep the memory snapshot usable and expose a warning without changing successful Kubernetes timestamps. A dedicated persistent cache volume is required to survive pod replacement; the root filesystem stays read-only. A loaded file never gets a new collection timestamp merely because it was read or rewritten. If classification rules change incompatibly, discard or explicitly reclassify only data for which original required metadata was retained; never serve old canonical groups as if they match new rules.

Alternative: SQLite or a shared cache service. A replaceable local snapshot fits the accepted single-process model and avoids database administration.

### 7. Shared freshness UI without conflating service availability

Keep the current application health indicator independent. Add inventory freshness/progress near it and make all existing refresh controls use the coordinator. Show last successful collection age, next automatic attempt/countdown, active run progress, and manual cooldown. Fetch status with `Cache-Control: no-store`; derive countdowns from server timestamps/server time to tolerate browser clock skew. Lightweight visibility-aware polling reads AutoDocs only and never Kubernetes; poll more frequently during a run and reload active data when generation changes. Preserve filters, scroll/focus and personal layout where practical during same-view data replacement. Retain the existing guard against late responses replacing another active view.

When a scope spans current and stale targets, present that mixed state and expose per-target/component ages. A completed run, API fetch, disk restore or service health check is not a successful inventory update. A never-successful component has no successful timestamp. Show explicit next-attempt language after errors, rather than claiming the next attempt will succeed. Refresh starts/finishes are shared among viewers, but browser favourites remain personal.

Alternative: a frontend timer based on fetch completion. That would label cached or failed data as fresh and create inconsistent user-visible schedules.

## Risks / Trade-offs

- [A one-replica Deployment can overlap during rolling updates] → The linked GitOps change uses a non-overlapping strategy; multiple active replicas remain unsupported.
- [Sequential scans grow with cluster count] → Bound each target and total work, expose progress, reuse lists, and measure before adding bounded internal concurrency.
- [Credentials expire or rotate while running] → Reopen mounted kubeconfigs per run, replace clients from one validated file read and report failure without credential leakage; renewal remains external.
- [Missing labels or namespace permissions obscure classification] → Keep sources visible, isolate dimensions, report lookup issues and provide distinct fallback groups.
- [Persisted status can be mistaken for live health] → Preserve actual collection ages and keep AutoDocs availability separate from inventory freshness.
- [Canonicalization or IDs can invalidate browser state] → Stable opaque cluster IDs, explicit legacy migration, versioned preferences and deterministic canonical dimension IDs.
- [Restoring unrelated inventory leaks misleading data] → Validate snapshot compatibility, remove unconfigured targets and reject known target changes.
- [A manual refresh endpoint increases API load] → One global run, observed-generation coalescing, cooldown, same-origin request handling and scheduler-owned updates.

## Migration Plan

1. Implement registry/classification/collector changes behind compatible API arrays; preserve the latest UI and package fixtures. Add meaningful tests for true multi-target and failure/concurrency behaviour.
2. Implement coordinator, persistence, metadata APIs and versioned preference migration; verify with deterministic clocks and temporary files, then integrate shared scope/grouping/freshness UI.
3. After implementation passes checks, complete the explicit documentation task: update README and embedded changelog with actual configuration/Secret schemas, required RBAC, classification semantics, intervals, migration, cache/rotation/failure behaviour, single-instance constraint and limitations. Do not document planned behaviour as shipped.
4. Publish only the tested source/image through the existing workflow, verify image platform/digest/access, and execute the linked `../homelab-gitops/openspec/changes/deploy-autodocs-multicluster-snapshots` rollout. Preserve unrelated user changes and do not invent real credentials, label keys or Bitwarden UUIDs.
5. Verify live APIs/assets, freshness, shared manual refresh, persistence and Argo state. Record actual rollout observations and documentation status. Roll back by restoring prior verified image/config values; retain the cache PVC and versioned preferences so incompatible old code can ignore newer state safely.

## Open Questions

The accepted behaviour is settled. Deployment inputs still require inspection: exact real environment/datacentre label keys, additional target Secret names/contexts and credential ownership, and the cache storage class/claim policy. These do not block implementation against fixtures or an initial hosting-cluster rollout. Do not connect illustrative remote targets until real configured credentials exist.
