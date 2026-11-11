## Why

AutoDocs currently requires an explicit connection for every cluster. A Rancher connection should discover accessible managed clusters and collect their inventories through Rancher, while making overlap with existing direct connections visible and configurable.

## What Changes

- Add optional file-configured Rancher sources with stable source IDs and mounted kubeconfig credentials. Discover all clusters visible to the credential by default, with explicit exclusions, and access their Kubernetes APIs through Rancher's proxy.
- Keep in-cluster and mounted-kubeconfig direct targets. Reuse existing collectors, publication rules, classification, directory and versions views.
- Refresh discovered membership through the existing shared coordinator. Add newly visible clusters, remove clusters absent from a complete successful listing, and retain stale membership/inventory when discovery fails.
- Separate connection identity from canonical AutoDocs cluster identity. Preserve existing direct IDs and derive stable identities for unmapped Rancher clusters from the source ID and Rancher cluster ID.
- Detect possible overlapping connections using a validated cluster identity signal; expose possible overlap, unavailable identity checks and conflicting evidence in discovery status. Never infer equivalence from display names or API addresses alone.
- Resolve overlap through explicit configuration mapping connections to one canonical cluster with one primary connection. Collect resource inventory only through that primary; do not merge permission-dependent results or automatically fail over.
- Keep unresolved connections separately identified, warn that inventory may be duplicated, and preserve canonical cluster identities and browser preferences when changing the primary.
- Extend sanitized status and persisted snapshots for Rancher source discovery, dynamic membership, connection selection and overlap diagnostics. Preserve independent direct-cluster operation when Rancher is unavailable.

## Capabilities

### New Capabilities

- `rancher-cluster-discovery`: Rancher source configuration, credential handling, supported API verification, managed-cluster enumeration and proxy connections.
- `cluster-connection-overlap`: Canonical cluster/connection identities, evidence-based overlap warnings, explicit mappings and primary-only collection.
- `dynamic-inventory-membership`: Refresh reconciliation, safe removals, source-level failure retention, restart recovery and discovery-status presentation for dynamic targets.

### Modified Capabilities

None of the currently archived base specifications change their publication or organization rules. These additive capabilities extend the implemented `add-multicluster-discovery-snapshots` contract, archived on 2026-11-11; its current registry and snapshot specifications remain the baseline for direct connections. The archived ExternalDNS Service discovery contract must also be preserved.

## Impact

- Go registry validation/client construction, refresh coordination, snapshot persistence, status/data projections, shared discovery UI and backend/browser/container fixtures.
- Existing array-based inventory APIs and resource identities remain compatible for direct targets. Status gains source/connection diagnostics; the persisted format needs an explicit migration path for existing snapshots.
- No new database, frontend framework, credential-management UI or Rancher resource writes. Credentials remain mounted files and never enter Git, snapshots, logs or HTTP responses.
- Application implementation and documentation belong here. A later deployment change can mount a Rancher credential Secret through the existing workload chart and pin a published image; this proposal does not change deployments, create credentials, commit or push.
- Validate supported Rancher API resources, token behavior, proxy paths and minimum read permissions before committing to an endpoint/version contract. Prefer supported RK-API; do not silently rely on the unsupported legacy v3 API.

## Non-goals

Rancher login/account administration, token issuance or renewal, automatic connection failover, automatic inventory merging, browser-user-specific Rancher permissions, project-level inventory scoping, or management of downstream clusters.
