## ADDED Requirements

### Requirement: Optional Rancher sources alongside direct targets
AutoDocs SHALL accept Rancher sources with unique stable IDs, mounted kubeconfig paths and explicit contexts alongside existing direct targets. Each source SHALL discover all credential-visible managed clusters, including the management cluster when visible, except explicitly excluded cluster IDs. Rancher-only configuration and a successful empty discovery SHALL be valid. An explicit configuration SHALL NOT trigger implicit local-cluster fallback.

#### Scenario: Mixed connections
- **WHEN** configuration contains a direct target and a Rancher source
- **THEN** both participate in discovery through the shared coordinator and retain independent connection identities

#### Scenario: Rancher-only empty inventory
- **WHEN** a valid Rancher-only configuration successfully discovers no accessible clusters
- **THEN** AutoDocs serves an empty inventory with successful source status and does not connect to an implicit local target

#### Scenario: Explicit exclusion
- **WHEN** Rancher lists a cluster whose management ID is excluded for that source
- **THEN** AutoDocs does not collect it or restore its inventory as an active source member

### Requirement: Verified management API compatibility
The Rancher adapter SHALL use a documented and verified API contract for supported Rancher releases, covering imported and provisioned clusters, management IDs, visibility, pagination and downstream proxy access. The initial compatibility investigation SHALL prefer supported RK-API. An unsupported API/resource SHALL produce a sanitized source failure, with no silent fallback to legacy v3 or a different cluster.

#### Scenario: Imported and provisioned clusters
- **WHEN** the supported management API lists one accessible imported cluster and one accessible provisioned cluster
- **THEN** both become candidates using their verified downstream management IDs rather than assumed provisioning names

#### Scenario: Required API unavailable
- **WHEN** the configured Rancher version does not expose the required supported cluster resource
- **THEN** source discovery reports API incompatibility and does not silently invoke legacy enumeration

### Requirement: Source credential isolation and refresh
Rancher sources SHALL use a self-contained, verified HTTPS kubeconfig with embedded bearer authentication and an explicit context. Unsupported executable, interactive, external-file or impersonation credentials SHALL be rejected before client construction. Each source refresh SHALL consume one coherent mounted credential version, and SHALL NOT authenticate with superseded credentials after replacement validation fails. Credentials, raw kubeconfigs and credential paths SHALL NOT appear in public diagnostics, inventory snapshots or logs. Issuance/renewal SHALL remain external to AutoDocs.

#### Scenario: Mounted credential rotation
- **WHEN** a valid replacement source kubeconfig is present at the next refresh
- **THEN** enumeration and that run's proxy clients use the replacement credential without changing source or cluster identities

#### Scenario: Invalid replacement
- **WHEN** a previously usable source kubeconfig is replaced by invalid content
- **THEN** the source reports credential failure, retains compatible old inventory as stale and does not reuse its superseded credential

#### Scenario: Secret absent at startup
- **WHEN** a source's optional credential mount is absent
- **THEN** AutoDocs starts, exposes sanitized source unavailability, and continues independent direct/source work

### Requirement: Read-only collection through the Rancher proxy
AutoDocs SHALL construct downstream clients using the verified Rancher proxy route and the source's trust/authentication, then run existing read-only collectors. Discovery SHALL NOT require direct downstream API connectivity, per-cluster kubeconfig downloads, Kubernetes Secret reads, Rancher writes or additional login helpers. Authenticated pagination/redirects SHALL remain on the configured trusted origin, and identifiers SHALL be validated before route construction.

#### Scenario: Downstream API reachable only through Rancher
- **WHEN** Rancher can proxy a permitted downstream cluster and its direct API endpoint is unreachable from AutoDocs
- **THEN** AutoDocs can collect that cluster through Rancher using the ordinary collectors

#### Scenario: Credential visibility is restricted
- **WHEN** the Rancher identity can discover a cluster but lacks a collector's resource permission
- **THEN** AutoDocs reports that collection failure independently without claiming full access or requesting broader credentials automatically

#### Scenario: Untrusted navigation
- **WHEN** a response redirects or paginates to a different origin or supplies an invalid downstream route identifier
- **THEN** AutoDocs rejects the operation without forwarding the Rancher credential to that destination
