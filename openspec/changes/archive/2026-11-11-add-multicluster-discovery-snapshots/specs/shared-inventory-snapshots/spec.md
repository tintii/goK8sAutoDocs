## ADDED Requirements

### Requirement: Data reads use a shared published snapshot
All inventory GET endpoints, including published/discovered directory, versions, filtered resources and retained workload-status APIs, SHALL read a shared published snapshot without initiating Kubernetes discovery. Background collection SHALL leave the prior snapshot available to readers and atomically publish a new application generation after a run. Inventory responses SHALL expose their generation and use no-store or generation-validated revalidation so a detected generation change cannot silently return an old browser-cached payload. Publication SHALL NOT claim a transactional Kubernetes snapshot. Successful empty collections SHALL replace previous results; no usable initial data SHALL produce an explicit loading/unavailable state instead of a successful empty inventory.

#### Scenario: Many viewers read during refresh
- **WHEN** multiple users request different inventory pages while a refresh is running
- **THEN** they receive the previously published inventory without waiting for Kubernetes reads or starting additional scans

#### Scenario: Empty successful inventory
- **WHEN** a component is successfully collected and contains no eligible resources
- **THEN** its published result becomes empty rather than retaining removed objects

### Requirement: One server-owned automatic and manual refresh
AutoDocs SHALL have at most one active refresh job across startup, automatic scheduling and all manual requests. The job SHALL have a service-owned context independent of any requesting browser, bounded per-target work and advertised progress. Automatic ticks and concurrent requests SHALL coalesce without building a refresh queue. A configurable interval and manual cooldown SHALL bound scans, with initial defaults of five minutes and thirty seconds respectively. The next automatic attempt SHALL be scheduled from run completion; cooldown requests SHALL NOT postpone it. A same-origin manual POST API SHALL return shared run/generation state promptly and SHALL reject unsupported methods and disallowed browser origins.

#### Scenario: Two users request an update
- **WHEN** two users submit Update now while no run is active
- **THEN** one refresh starts and both users receive the same shared run state

#### Scenario: Requesting browser disconnects
- **WHEN** the initiating browser closes before collection finishes
- **THEN** the shared job continues and its result becomes available to other users

#### Scenario: Late duplicate click
- **WHEN** another request arrives after a run finishes but within cooldown or with an obsolete observed generation
- **THEN** it receives current status without starting or queuing another immediate scan

#### Scenario: Timer fires during a manual run
- **WHEN** the automatic schedule becomes due during an active manual refresh
- **THEN** no overlapping or queued duplicate run starts and the next due time reflects completion of the active job

### Requirement: Preserve successful partitions on collection failure
The coordinator SHALL track collection outcomes per target and component, including separate directory source-kind partitions. Failed required lists SHALL retain the prior successful partition, original success time and prior diagnostics while adding current failure diagnostics. Healthy partitions and other targets SHALL update independently. Related-object issues SHALL retain their source. Unsupported optional Gateway API absence SHALL be distinguished from forbidden/unreachable collection. Published and discovered projections SHALL use the same source partition so retention cannot introduce contradictory view membership. A failed attempt SHALL never reset last successful collection time.

#### Scenario: One cluster is unreachable
- **WHEN** one target times out while another refreshes successfully
- **THEN** the healthy target updates and the unreachable target retains its prior inventory visibly marked stale with its original successful timestamp

#### Scenario: HTTPRoute list fails
- **WHEN** Ingress collection succeeds and HTTPRoute listing is forbidden
- **THEN** fresh Ingresses and any retained HTTPRoutes remain accessible with component-specific incomplete/stale diagnostics

#### Scenario: Resource changes publication classification
- **WHEN** a successfully collected source changes from absent publication label to exact-true publication
- **THEN** the new generation moves it between the discovered and published projections without presenting contradictory source copies

### Requirement: Restore and atomically persist compatible inventory
When a cache path is configured, AutoDocs SHALL restore compatible derived inventory before its initial background refresh and SHALL persist published snapshots using a temporary file and atomic rename. Files SHALL include schema/configuration compatibility information, source partitions, collection times and diagnostics, and SHALL exclude credentials, raw kubeconfigs, ConfigMap bodies and unneeded raw resource lists. Reads SHALL be bounded and writes SHALL use restrictive permissions. Removed/disabled targets and known target-identity mismatches SHALL NOT appear in restored data. Corrupt, oversized or incompatible files SHALL yield a cache diagnostic and fresh collection without preventing application startup. Persistence failure SHALL keep successful memory data available and expose a diagnostic.

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

### Requirement: Authoritative shared freshness metadata
A no-store discovery-status API SHALL expose a generation, server time, current run state/progress, next automatic attempt, next eligible manual request and per-target/component last attempt/last success/outcome. A never-successful component SHALL have no successful timestamp. Reading an API, restoring disk data, completing a partial/failed run or checking application liveness SHALL NOT mark retained data freshly collected. Metadata SHALL remain sanitized and sufficient to display mixed freshness for any selected scope.

#### Scenario: Scope spans fresh and stale clusters
- **WHEN** the displayed environment contains data from a successful target and a failed target
- **THEN** metadata allows the UI to show both states and each target's actual data age instead of declaring the whole environment just updated

#### Scenario: Browser clock differs
- **WHEN** a browser clock is offset from server time
- **THEN** server time and due timestamps allow a correct relative update/countdown display without additional discovery
