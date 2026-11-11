# cluster-connection-overlap Specification

## Purpose
Define stable canonical cluster identities, evidence-based overlap diagnostics and explicit primary-only connection bindings.

## Requirements
### Requirement: Separate canonical cluster and connection identities
AutoDocs SHALL distinguish direct/Rancher connection IDs from canonical cluster IDs. Existing unmapped direct targets SHALL retain their current cluster IDs. Unmapped Rancher identities SHALL derive deterministically from the source ID and management cluster ID with unambiguous encoding. Display names and credentials SHALL NOT determine identity. Canonical IDs SHALL remain stable when a primary connection changes within an explicit binding.

#### Scenario: Same names on different sources
- **WHEN** two Rancher sources each expose a cluster with the same name and management ID
- **THEN** their unmapped connections and canonical IDs remain distinct

#### Scenario: Rename or token rotation
- **WHEN** a Rancher cluster is renamed or its source credential rotates
- **THEN** its source/link identities and eligible browser preferences remain unchanged

#### Scenario: Existing hosting cluster gains a Rancher connection
- **WHEN** an explicit binding associates Rancher's management-cluster connection with an existing direct target
- **THEN** the canonical cluster retains the direct target's existing ID and preferences

### Requirement: Evidence-based overlap diagnostics
AutoDocs SHALL compare bounded, successful identity observations from the same refresh generation to flag possible overlaps. The implementation SHALL validate and document the chosen signal, initially investigating the `kube-system` Namespace UID and its clone/restore limitations. Matching names or endpoints alone SHALL NOT establish equivalence. Missing, denied, failed or obsolete observations SHALL be reported as identity-check unavailable, not as proof that overlap is absent. A failed identity probe SHALL NOT block otherwise authorized collection.

#### Scenario: Matching observed marker
- **WHEN** two unbound connections return the same validated identity marker in one refresh
- **THEN** discovery status flags possible overlap and names both connections without automatically joining them

#### Scenario: Insufficient identity permission
- **WHEN** a connection can collect some resources but cannot read the identity marker
- **THEN** collection continues and status reports identity-check unavailable

#### Scenario: Stale evidence
- **WHEN** only one connection has a current observation and another has a previous generation's observation
- **THEN** AutoDocs does not use their comparison to assert a current match or conflict

### Requirement: Explicit bindings with one primary
Configuration SHALL support binding multiple connections to one canonical cluster with exactly one explicit primary. Each connection SHALL belong to at most one binding. Configuration SHALL reject ambiguous IDs, unknown source/direct references, contradictory exclusions, malformed connection references and a primary outside its binding. A known source's temporarily undiscovered cluster SHALL remain a valid reference whose availability is reported at runtime.

#### Scenario: Invalid primary
- **WHEN** a binding's primary is absent from its configured member list
- **THEN** configuration validation fails before collection

#### Scenario: Cluster not yet discovered
- **WHEN** a valid binding references a cluster on a configured Rancher source that has not successfully enumerated it
- **THEN** AutoDocs reports that connection unavailable without rejecting the source configuration or inventing a replacement

### Requirement: Primary-only resource collection
For an explicit binding, AutoDocs SHALL run resource collectors only through the selected primary and SHALL NOT union results or automatically switch connections. Secondary connections SHALL perform at most the required identity probes. If available marker observations contradict a binding, status SHALL flag a binding identity conflict while retaining explicit primary selection. If a primary fails, compatible inventory from that primary SHALL be stale; a healthy secondary SHALL NOT make the cluster appear refreshed.

#### Scenario: Direct primary and Rancher secondary
- **WHEN** both connections in the binding are available
- **THEN** resource inventory is collected once through the direct primary and Rancher is shown as a secondary connection

#### Scenario: Primary outage
- **WHEN** the primary fails while a secondary is reachable
- **THEN** AutoDocs reports primary failure and retains only compatible stale primary inventory without collecting through the secondary

#### Scenario: Conflicting binding evidence
- **WHEN** two currently observed member identity markers differ
- **THEN** AutoDocs displays a binding identity conflict and continues only the explicitly selected primary's resource collection

### Requirement: Unresolved overlaps remain separate and actionable
Unbound connections with possible overlap SHALL keep separate canonical identities and independent inventories. The shared discovery UI SHALL identify the connections, explain that resources can appear twice, and direct the operator to configure a canonical binding and primary. It SHALL NOT silently suppress either inventory or automatically merge browser preferences.

#### Scenario: Duplicate inventory before resolution
- **WHEN** matching markers are found but no explicit binding exists
- **THEN** both inventories remain separately attributable and the UI displays a possible-duplicate warning with configuration guidance

### Requirement: Primary changes preserve identity without reusing incompatible inventory
Changing a binding's primary SHALL preserve its canonical ID and source/link preferences for unchanged resources/URLs. Previous-primary cached inventory SHALL be invalidated before serving the new configuration. An observed underlying cluster identity change SHALL also invalidate prior inventory. Preferences attached to a discarded duplicate canonical ID SHALL NOT be automatically merged into the retained ID.

#### Scenario: Primary changes to a narrower credential
- **WHEN** configuration switches a canonical cluster to a different primary with fewer resource permissions
- **THEN** the canonical ID remains unchanged but previous-primary rows are not served as fallback while the new primary refreshes or fails

#### Scenario: Replacement cluster at the same connection
- **WHEN** a successful identity probe reveals a different underlying identity for an existing connection
- **THEN** its prior inventory is discarded and status exposes the identity change rather than presenting old resources as current
