## MODIFIED Requirements

### Requirement: Preserve successful partitions on collection failure
The coordinator SHALL track collection outcomes per target and component, including separate directory source-kind partitions. Failed required lists SHALL retain the prior successful partition, original success time and prior diagnostics while adding current failure diagnostics. Healthy partitions and other targets SHALL update independently. Related-object issues SHALL retain their source. Unsupported optional Gateway API absence SHALL be distinguished from forbidden/unreachable collection. Published and discovered projections SHALL use the same source partition so retention cannot introduce contradictory view membership. A failed attempt SHALL never reset last successful collection time.

Manual declarations SHALL have a separate component per collection target with last-good normalized results and success times per ConfigMap document. List failure SHALL retain the prior component. After a successful list, each valid document SHALL replace its result, each invalid document SHALL retain only its own previous result and original success time with diagnostics, and a newly invalid document SHALL contribute no entries. Other valid documents SHALL still update. Deleted or unmarked documents and valid empty links lists SHALL remove their previous entries. A successful list with invalid documents SHALL report a partial outcome rather than fresh success for retained data.

#### Scenario: One cluster is unreachable
- **WHEN** one target times out while another refreshes successfully
- **THEN** the healthy target updates and the unreachable target retains its prior inventory visibly marked stale with its original successful timestamp

#### Scenario: HTTPRoute list fails
- **WHEN** Ingress collection succeeds and HTTPRoute listing is forbidden
- **THEN** fresh Ingresses and any retained HTTPRoutes remain accessible with component-specific incomplete/stale diagnostics

#### Scenario: Resource changes publication classification
- **WHEN** a successfully collected source changes from absent publication label to exact-true publication
- **THEN** the new generation moves it between the discovered and published projections without presenting contradictory source copies

#### Scenario: One malformed manual document
- **WHEN** a successful ConfigMap list contains a valid updated document and a malformed previously valid document
- **THEN** the valid document updates and only the malformed document retains its previous entries and success time
- **AND** the component exposes mixed freshness with a sanitized document error

#### Scenario: Intentional removal of manual entries
- **WHEN** a successful list observes document deletion, marker removal or a valid empty links list
- **THEN** the corresponding previous entries are removed from the new generation rather than retained as stale

#### Scenario: ConfigMap list forbidden
- **WHEN** manual ConfigMap listing fails after a successful collection
- **THEN** the previous manual component remains available with original successful timestamps and an incomplete collection diagnostic

### Requirement: Restore and atomically persist compatible inventory
When a cache path is configured, AutoDocs SHALL restore compatible derived inventory before its initial background refresh and SHALL persist published snapshots using a temporary file and atomic rename. Files SHALL include schema/configuration compatibility information, source partitions, collection times and diagnostics, and SHALL exclude credentials, raw kubeconfigs, ConfigMap bodies and unneeded raw resource lists. Reads SHALL be bounded and writes SHALL use restrictive permissions. Removed/disabled targets and known target-identity mismatches SHALL NOT appear in restored data. Corrupt, oversized or incompatible files SHALL yield a cache diagnostic and fresh collection without preventing application startup. Persistence failure SHALL keep successful memory data available and expose a diagnostic.

Normalized manual entries, stable identities, declaration provenance and per-document collection times/diagnostics SHALL be permitted derived cache data; raw declaration documents SHALL remain excluded. Restore/removal and target-identity protections SHALL use the collection target from provenance rather than relevance cluster ID. A cache without a manual component SHALL treat that component as never collected, not successful empty data.

#### Scenario: Restart with a valid cache
- **WHEN** AutoDocs starts with a compatible saved snapshot
- **THEN** it serves that inventory immediately using its original successful timestamps while one initial refresh runs

#### Scenario: Corrupt or incompatible cache
- **WHEN** the snapshot cannot be safely restored
- **THEN** AutoDocs starts, reports the cache issue and collects configured targets without claiming an empty successful inventory

#### Scenario: Removed cluster remains in an old file
- **WHEN** a saved partition belongs to a cluster ID no longer enabled in configuration
- **THEN** it is excluded from all restored APIs and UI projections

#### Scenario: Snapshot write fails
- **WHEN** Kubernetes collection succeeds but the cache volume cannot be written
- **THEN** the new memory inventory remains usable, successful collection times remain accurate and persistence failure is visible

#### Scenario: Restore shared declaration from cache
- **WHEN** a compatible cache contains a manual link stored in target A with relevance target B
- **THEN** it restores with its original document timestamp and provenance while A remains an eligible source
- **AND** removing source target A excludes its cached declaration even if target B remains enabled

### Requirement: Authoritative shared freshness metadata
A no-store discovery-status API SHALL expose a generation, server time, current run state/progress, next automatic attempt, next eligible manual request and per-target/component last attempt/last success/outcome. A never-successful component SHALL have no successful timestamp. Reading an API, restoring disk data, completing a partial/failed run or checking application liveness SHALL NOT mark retained data freshly collected. Metadata SHALL remain sanitized and sufficient to display mixed freshness for any selected scope.

Manual-link status SHALL additionally expose sanitized per-document success/outcome metadata sufficient to distinguish retained documents from fresh ones and associate entries with their declaration source. A successful ConfigMap list SHALL NOT refresh the successful timestamp of invalid retained documents. Status SHALL support relevance-scope freshness across declaration clusters and advertise explicit normalized manual classification values without inventing registry targets.

#### Scenario: Scope spans fresh and stale clusters
- **WHEN** the displayed environment contains data from a successful target and a failed target
- **THEN** metadata allows the UI to show both states and each target's actual data age instead of declaring the whole environment just updated

#### Scenario: Browser clock differs
- **WHEN** a browser clock is offset from server time
- **THEN** server time and due timestamps allow a correct relative update/countdown display without additional discovery

#### Scenario: List succeeds but document validation fails
- **WHEN** a marked ConfigMap is readable but its document is invalid
- **THEN** status records the attempt and validation issue while preserving the prior document success time or no success time on first failure
- **AND** the directory can report incomplete manual-link discovery in scopes that include retained or potentially shared entries
