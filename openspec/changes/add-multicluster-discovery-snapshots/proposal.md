## Why

AutoDocs currently discovers one cluster through global clients and request-triggered, short-lived memory caches. It needs to aggregate independently configured clusters, classify their applications using existing labels, and serve a shared durable inventory without multiplying Kubernetes requests as more people use the UI.

## What Changes

- Add an explicit cluster registry with stable IDs, optional in-cluster credentials, and one mounted, self-contained kubeconfig per remote cluster. Document the `Opaque` Secret / `kubeconfig` key contract and require an explicitly selected context.
- Resolve environment and datacentre from configurable resource/namespace/cluster-metadata label mappings. Discover arbitrary values, support case-sensitive or case-insensitive equality independently of display casing, and retain resources with missing labels.
- Add cluster/environment/datacentre context, grouping and composable filtering to the directory, overview and versions inventory. Environment and cluster have a many-to-many relationship; neither names nor prefixes encode classification rules.
- Extend source identities with stable cluster IDs and migrate eligible legacy browser preferences without transferring favourites between clusters.
- Replace request-owned discovery with one process-owned refresh coordinator. Serve snapshots immediately, refresh periodically or through a global Update now request, coalesce concurrent requests, and enforce a shared cooldown.
- Restore a versioned local snapshot on startup; atomically persist derived inventory, original collection timestamps and diagnostics. Preserve successful prior data when collection fails and update healthy clusters independently.
- Present last successful update, next scheduled attempt, shared progress and per-cluster/component stale status beside the existing application-availability indicator.
- Keep existing array-based data APIs and publication/endpoint-resolution rules; add metadata fields and dedicated refresh/status APIs. Explicitly supersede the old independent request-expiry cache requirements.
- Include meaningful backend/race/browser/container validation, followed by documentation updates after implementation, then publication and the linked GitOps rollout.

## Capabilities

### New Capabilities
- `cluster-discovery-registry`: Explicit cluster identity, mounted kubeconfig contract, connection validation, read-only access and credential reload.
- `inventory-classification`: Configurable environment/datacentre label resolution, canonical matching, independent display casing and missing-value handling.
- `shared-inventory-snapshots`: Coordinated automatic/manual refresh, persisted snapshots, failure isolation and authoritative freshness metadata.
- `multicluster-inventory-experience`: Cluster-aware identities and browser preference migration, cross-page scope/filtering/grouping, and shared refresh UI.

### Modified Capabilities
- `published-endpoint-directory`: Add cluster context and serve derived endpoint data from shared snapshots while preserving publication and resolution guarantees.
- `endpoint-directory-organization`: Replace independent on-demand view caches with snapshot projections and extend grouping/search/filtering to classification dimensions.

## Impact

Application changes belong in this repository: Go client initialization/collectors/HTTP handlers, static shared shell/directory model/overview/versions UI, browser preference schema, fixtures/tests, runtime smoke checks and README documentation/changelog. Kubernetes credentials must never enter Git, API responses, logs or the snapshot file. No external database or frontend framework is needed.

Deployment belongs to the companion `deploy-autodocs-multicluster-snapshots` change in `../homelab-gitops`: keep using a verified published `helm-k8sworkload` chart, provide registry configuration, optional read-only Secret directory mounts and a writable persistent cache volume, enforce a single active instance, and pin the tested published image digest. The existing hosting-cluster ServiceAccount can remain the first target. Additional real clusters require actual credential Secret references, target label keys and reachable API endpoints; illustrative cluster names are not deployable credentials.
