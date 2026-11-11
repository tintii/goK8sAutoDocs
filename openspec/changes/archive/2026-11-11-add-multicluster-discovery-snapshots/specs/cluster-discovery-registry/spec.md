## ADDED Requirements

### Requirement: Explicit independent cluster targets
AutoDocs SHALL load a registry of uniquely identified targets, each with a stable non-empty cluster ID, optional display name and configured metadata labels, and exactly one credential method: in-cluster credentials or a mounted kubeconfig path with an explicit context. Each target SHALL have its own typed/dynamic client pair. Target IDs SHALL remain independent of environment, datacentre, display names and internal kubeconfig names. Explicit registry configuration SHALL be authoritative; authentication failures SHALL NOT cause fallback to another target or context. Invalid global structure and duplicate IDs SHALL be rejected before discovery. Without explicit configuration, the existing single-target connection selection SHALL remain available with a stable internal `local` ID and SHALL NOT enumerate other contexts.

#### Scenario: Two files reuse internal names
- **WHEN** two configured cluster IDs use separate files whose context, user and cluster entries have identical names
- **THEN** AutoDocs loads each file independently and discovers the explicitly selected target with its own provenance

#### Scenario: One target has invalid credentials
- **WHEN** one configured target cannot load valid credentials and another target is usable
- **THEN** discovery continues for the usable target and reports the failing target independently without connecting elsewhere on its behalf

#### Scenario: Duplicate identity
- **WHEN** registry configuration contains the same cluster ID twice
- **THEN** AutoDocs reports a configuration error before collecting any target

### Requirement: Self-contained kubeconfig Secret contract
The documented remote credential contract SHALL be one `Opaque` Secret per target in the hosting namespace, with required key `kubeconfig` containing a standard version-1 Config document. The deployment SHALL mount that key as a read-only file accessible to the non-root application. The configured context SHALL resolve to a valid cluster server and authentication entry. The initial supported format SHALL use HTTPS with certificate verification and either an embedded bearer token or an embedded client certificate/private key, with embedded CA trust where system trust is insufficient. AutoDocs SHALL reject executable/interactive authentication, legacy auth-provider plugins, external credential-file references and impersonation before constructing clients. A kubeconfig namespace SHALL NOT implicitly restrict explicit all-namespace collection. Extra unselected contexts SHALL NOT trigger discovery.

#### Scenario: Embedded token credentials
- **WHEN** the mounted kubeconfig contains the selected context, verified HTTPS endpoint and embedded token
- **THEN** AutoDocs uses that context for the configured cluster ID without reading other credential files

#### Scenario: Embedded client certificate credentials
- **WHEN** the selected user contains valid embedded client certificate/key data instead of a token
- **THEN** AutoDocs can authenticate using that pair without a runtime login helper

#### Scenario: Unsupported authentication dependency
- **WHEN** a selected kubeconfig uses `exec`, a workstation file path, impersonation or an unsupported auth-provider plugin
- **THEN** AutoDocs reports an unsupported credential configuration without executing commands or reading those external credential paths

### Requirement: Read-only provenance and credential isolation
Collectors SHALL request only their required discovery reads on each target and SHALL NOT require Secret API reads or Kubernetes writes. Namespace label resolution SHALL use explicit namespace reads. Registry status, resource responses, logs and persisted snapshots SHALL NOT contain credential bytes, raw kubeconfigs, private keys or bearer tokens. Sanitized diagnostics SHALL identify the failing target and failure category. The host's projected ServiceAccount SHALL remain usable independently of remote target Secrets.

#### Scenario: Secret data reaches the pod through a mount
- **WHEN** the deployment mounts the remote credential Secret
- **THEN** AutoDocs reads the local file without invoking a Secret API request

#### Scenario: Credential parsing fails
- **WHEN** malformed credential content causes a target load error
- **THEN** public diagnostics identify the target/configuration failure without reproducing supplied credential values

### Requirement: Reload mounted credentials per refresh
Each target refresh SHALL read and validate one coherent kubeconfig file version and SHALL rebuild clients when its contents change. Secret directory mounts SHALL avoid `subPath` so mounted updates can propagate. Missing or invalid credentials SHALL fail that target's refresh without crashing the service, renewing credentials, or silently reusing superseded kubeconfig authentication. Credential issuance and renewal SHALL remain the supplier's responsibility. Remote Secret mounts SHALL permit the process to start while a target Secret is absent, and credential availability SHALL be represented explicitly.

#### Scenario: Secret rotates
- **WHEN** a replacement kubeconfig reaches the mounted directory before the next run
- **THEN** that run uses the replacement credentials without restarting AutoDocs or changing the stable cluster ID

#### Scenario: Missing target Secret
- **WHEN** one optional remote credential Secret is missing
- **THEN** AutoDocs starts, marks that target credentials unavailable, serves any compatible prior snapshot as stale and refreshes other targets

#### Scenario: Rotation supplies malformed credentials
- **WHEN** a previously usable mounted kubeconfig is replaced with invalid contents
- **THEN** the next run reports credential failure and does not silently keep authenticating with the superseded contents
