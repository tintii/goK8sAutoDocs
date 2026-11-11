## ADDED Requirements

### Requirement: Cluster-aware source and personal preference identities
Source identity SHALL include stable cluster ID, kind, namespace and name; link identity SHALL also include its current exact resolved URL. Environment/datacentre labels and display names SHALL NOT change source identity. Namespace-derived group identity SHALL include cluster ID. Canonical environment/datacentre groups SHALL intentionally span clusters; nested group identity SHALL include its typed path. A versioned preference migration SHALL preserve theme, custom group definitions and versions layout, and SHALL migrate legacy source/link/namespace records only to an explicitly designated original cluster ID. Ambiguous legacy records SHALL remain dormant with a notice rather than attaching to an arbitrary cluster. Current response data SHALL remain the only source of clickable destinations.

#### Scenario: Identical names in different clusters
- **WHEN** two targets contain the same kind, namespace and source name and one source is starred
- **THEN** only the selected target/source receives that independent favourite or assignment

#### Scenario: Environment label changes
- **WHEN** a source's environment classification changes without changing its cluster/kind/namespace/name
- **THEN** it moves classification groups while retaining its independent link favourites and custom assignment

#### Scenario: Unambiguous legacy migration
- **WHEN** version-1 preferences exist and `legacyClusterID` identifies the original configured target
- **THEN** legacy source/link/namespace records migrate only to that target and theme/custom group definitions/versions layout survive

#### Scenario: No safe legacy target
- **WHEN** legacy source preferences exist without an unambiguous original target mapping
- **THEN** AutoDocs preserves them dormant, displays a migration notice and does not transfer them to matching names in other targets

### Requirement: Independent classification scope across inventory pages
The directory, versions inventory and overview favourites SHALL support independent cluster, environment and datacentre scope. Scope predicates SHALL combine with AND and with each page's existing applicable search, view, namespace, kind and favourites predicates. Scope SHALL be represented in URLs and preserved across page navigation, reload and browser history. Unavailable selected values SHALL remain visible and clearable without silently broadening scope. Search SHALL include cluster display/identity and resolved classification values. Counters SHALL describe the selected loaded scope and distinguish namespaces using cluster plus namespace identity. Scope changes SHALL NOT initiate Kubernetes discovery.

#### Scenario: Environment spans clusters
- **WHEN** the user selects an environment present on multiple targets
- **THEN** eligible resources from every contributing target appear, and adding a cluster filter restricts them to that target

#### Scenario: Preserve scope between pages
- **WHEN** the user opens Versions or Overview from a scoped Directory page and later navigates back
- **THEN** the selected scope remains represented and restored without altering personal favourites

#### Scenario: Unavailable filter value
- **WHEN** a selected environment is absent from the latest snapshot
- **THEN** the selection remains clearable and the page explains no matches rather than switching to all environments

### Requirement: Best-effort grouping and visible provenance
Both directory views SHALL add Cluster, Environment, Datacentre, Environment then Cluster and Cluster then Environment grouping modes while preserving existing label/namespace/kind/custom/none semantics. Missing metadata SHALL use a distinct fallback at the relevant level and SHALL NOT suppress the resource or other grouping dimensions. Cards and inventory rows SHALL expose sufficient cluster/environment/datacentre context to distinguish otherwise identical names. Grouping SHALL operate immediately on loaded data and SHALL preserve existing favourites, safe endpoint links, diagnostics, responsive layout and keyboard access.

#### Scenario: Nested grouping with an unclassified source
- **WHEN** the user groups by Environment then Cluster and a source has no environment
- **THEN** it appears beneath a distinct unclassified environment section and its actual cluster, with its links and diagnostics preserved

#### Scenario: Change grouping without collection
- **WHEN** the user switches from a label group to a datacentre or cluster group
- **THEN** loaded sources regroup immediately without a Kubernetes request

### Requirement: Shared update controls and accurate freshness UI
All inventory refresh controls SHALL request the same server-owned update process and display active progress/cooldown consistently across viewers. Visibility-aware browser polling SHALL query only snapshot/status APIs and SHALL reload relevant data when generation changes. Freshness UI SHALL show actual successful collection age, next automatic attempt and mixed stale/failed states, using server timing metadata. Application availability SHALL remain a separate indicator. Updates SHALL preserve active scope, personal settings and the existing late-response view guard; requests SHALL NOT replace another active view with stale responses.

#### Scenario: Another viewer starts discovery
- **WHEN** one viewer starts a shared refresh while another is viewing an inventory page
- **THEN** both can see the same run state and receive the completed generation without the second viewer starting another scan

#### Scenario: Service runs with stale inventory
- **WHEN** AutoDocs responds to its liveness check but a target refresh fails
- **THEN** the UI can show Running alongside that target's stale collection age and next attempt

#### Scenario: Directory view changes during refresh
- **WHEN** a refresh response for a prior view arrives after the user has switched tabs
- **THEN** it does not replace the active tab's data, warnings or scope
