# dynamic-inventory-membership Specification

## Purpose
Define authoritative dynamic membership reconciliation, coherent inventory generations, compatible snapshot recovery and source/connection status.

## Requirements
### Requirement: Authoritative complete membership reconciliation
The shared automatic/manual refresh SHALL enumerate configured Rancher sources and build a resolved registry before collection. Only a complete, validated listing across all pages SHALL replace a source's membership. Newly visible members SHALL participate in that run. Members absent from a successful complete listing, or explicitly excluded/removed from configuration, SHALL leave active source membership and SHALL NOT be restored from cache. A canonical cluster with a remaining direct/configured member SHALL remain independent of source removal.

#### Scenario: New cluster appears
- **WHEN** a complete source listing includes a previously unseen eligible cluster
- **THEN** that refresh resolves its identity and attempts collection through its selected connection

#### Scenario: Successful empty listing
- **WHEN** a source returns a complete successful list containing no eligible clusters
- **THEN** its previously discovered members leave active inventory without being described as a discovery failure

#### Scenario: Shared cluster loses Rancher membership
- **WHEN** a complete listing no longer contains a Rancher secondary bound to a configured direct target
- **THEN** the direct canonical cluster remains active and its direct primary continues collection

#### Scenario: Missing primary with an available secondary
- **WHEN** a binding's Rancher primary disappears from the successful listing but its direct secondary remains configured
- **THEN** the canonical cluster remains visible with unavailable-primary status and compatible stale data, without automatic failover

### Requirement: Source failure retention and isolation
Timeouts, authentication/API errors, incomplete pagination, malformed results and exceeded response/count budgets SHALL fail source enumeration without treating it as an empty authoritative list. AutoDocs SHALL preserve compatible previous membership/inventory as stale, skip that source's proxy collection for the failed run, and continue independent sources/direct primaries. The implementation SHALL bound discovery work and report overflow without silent truncation.

#### Scenario: A later page fails
- **WHEN** initial pages are returned but a later page fails
- **THEN** previously known membership is retained as stale, no partial-list deletions are committed and the source shows discovery failure

#### Scenario: Rancher outage with direct primary
- **WHEN** Rancher discovery fails and a bound hosting cluster has a healthy direct primary
- **THEN** the hosting cluster refreshes through that direct primary while Rancher membership remains stale

#### Scenario: Discovery budget exceeded
- **WHEN** an enumeration exceeds its configured or documented source/aggregate budget
- **THEN** status reports a bounded discovery failure and no truncated result is treated as authoritative membership

### Requirement: Coherent registry and inventory publication
Resolved membership, selected connections and inventory SHALL publish as one coherent generation through the existing shared refresh coordinator. Data/status projections SHALL use resolved canonical targets rather than only static configuration. Concurrent refresh requests SHALL retain existing coalescing and cooldown semantics, and source work/collection progress SHALL be distinguishable without exposing a partially reconciled committed inventory.

#### Scenario: Readers during reconciliation
- **WHEN** a browser reads data while source membership is being refreshed
- **THEN** it sees the previous coherent committed inventory and separately reported running progress until the new generation publishes

### Requirement: Compatible persisted discovery recovery
Snapshots SHALL store sanitized dynamic membership, canonical identities, primary provenance, compatible inventory and source/collection freshness using a versioned format. AutoDocs SHALL restore existing version-1 direct snapshots where configuration is compatible. New-format membership SHALL restore only for matching source/binding configuration and SHALL be stale until refreshed. Removed sources, excluded members, incompatible primary selections and malformed caches SHALL NOT resurrect unrelated inventory. Persistence failures SHALL remain isolated from serving the valid in-memory snapshot.

#### Scenario: Restart while Rancher is unavailable
- **WHEN** compatible saved membership exists and source discovery fails after restart
- **THEN** AutoDocs serves the restored inventory as stale and reports source discovery failure

#### Scenario: Upgrade with direct-only cache
- **WHEN** a version-1 snapshot exists for unchanged direct targets
- **THEN** those compatible targets restore with their original canonical IDs without requiring Rancher availability

#### Scenario: Source removed from configuration
- **WHEN** a saved snapshot contains members belonging only to a source that is no longer configured
- **THEN** those members are not restored into active inventory

### Requirement: Visible source and connection diagnostics
Discovery status and shared UI SHALL distinguish source-enumeration health, last successful enumeration, per-cluster collection freshness, primary/secondary selection, possible overlap, identity-check unavailability and binding conflicts. Diagnostics SHALL use sanitized identifiers and names without exposing credentials or credential paths. Directory, overview and versions SHALL use the same canonical identities and retain existing publication/classification rules. Browser filters SHALL NOT be presented as authorization boundaries.

#### Scenario: Enumeration succeeds but one cluster fails
- **WHEN** a source lists its clusters successfully but one primary's resource collection fails
- **THEN** source status reports enumeration success and the affected cluster separately reports failed/stale collection

#### Scenario: Shared overlap warning
- **WHEN** a possible overlap exists
- **THEN** users can inspect the same connection warning and resolution guidance from shared discovery controls without receiving Rancher credentials or a credential-editing form
