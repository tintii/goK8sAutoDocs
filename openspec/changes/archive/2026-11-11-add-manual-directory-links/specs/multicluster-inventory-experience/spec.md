## MODIFIED Requirements

### Requirement: Cluster-aware source and personal preference identities
Discovered source identity SHALL include stable cluster ID, kind, namespace and name; link identity SHALL also include its current exact resolved URL. Environment/datacentre labels and display names SHALL NOT change source identity. Namespace-derived group identity SHALL include cluster ID. Canonical environment/datacentre groups SHALL intentionally span clusters; nested group identity SHALL include its typed path. A versioned preference migration SHALL preserve theme, custom group definitions and versions layout, and SHALL migrate legacy source/link/namespace records only to an explicitly designated original cluster ID. Ambiguous legacy records SHALL remain dormant with a notice rather than attaching to an arbitrary cluster. Current response data SHALL remain the only source of clickable destinations.

Manual source identity SHALL instead use a typed tuple of declaration cluster ID, ConfigMap namespace/name and entry ID, independently of title, group and all declared relevance dimensions. Manual link identity SHALL append the exact current URL. Namespace grouping for manual entries SHALL use typed declared/shared cluster and namespace components rather than declaration provenance. Existing discovered identities and migration rules SHALL remain unchanged.

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

#### Scenario: Edit manual relevance without losing personal identity
- **WHEN** a manual declaration changes title, group or scope but retains declaration location, entry ID and exact URL
- **THEN** its source identity, custom assignment and independent link star remain unchanged
- **AND** current group membership and filters reflect its new relevance

### Requirement: Independent classification scope across inventory pages
The directory, versions inventory and overview favourites SHALL support independent cluster, environment and datacentre scope. Scope predicates SHALL combine with AND and with each page's existing applicable search, view, namespace, kind and favourites predicates. Scope SHALL be represented in URLs and preserved across page navigation, reload and browser history. Unavailable selected values SHALL remain visible and clearable without silently broadening scope. Search SHALL include cluster display/identity and resolved classification values. Counters SHALL describe the selected loaded scope and distinguish namespaces using cluster plus namespace identity. Scope changes SHALL NOT initiate Kubernetes discovery.

Manual links SHALL use optional relevance predicates: an omitted cluster/environment/datacentre/namespace dimension is unrestricted, while a declared value must match its selection. Shared links SHALL still satisfy ordinary view, query, kind, selected-group and favourites restrictions. This exception SHALL NOT change discovered-resource unknown matching or Versions membership. Explicit manual environment/datacentre values SHALL contribute to available scope choices using shared normalization.

#### Scenario: Environment spans clusters
- **WHEN** the user selects an environment present on multiple targets
- **THEN** eligible resources from every contributing target appear, and adding a cluster filter restricts them to that target

#### Scenario: Preserve scope between pages
- **WHEN** the user opens Versions or Overview from a scoped Directory page and later navigates back
- **THEN** the selected scope remains represented and restored without altering personal favourites

#### Scenario: Unavailable filter value
- **WHEN** a selected environment is absent from the latest snapshot
- **THEN** the selection remains clearable without switching to all environments; scoped resources with other values remain excluded, while intentionally shared manual links remain eligible and no-matches appears only if no entries satisfy all predicates

#### Scenario: Shared links remain under a concrete scope
- **WHEN** the directory or overview is filtered to a cluster and environment and a manual link omits both dimensions
- **THEN** those scope filters permit the link while other active restrictions still apply
- **AND** a discovered resource lacking that environment remains excluded by the concrete environment filter

### Requirement: Best-effort grouping and visible provenance
Both directory views SHALL add Cluster, Environment, Datacentre, Environment then Cluster and Cluster then Environment grouping modes while preserving existing label/namespace/kind/custom/none semantics. Missing metadata SHALL use a distinct fallback at the relevant level and SHALL NOT suppress the resource or other grouping dimensions. Cards and inventory rows SHALL expose sufficient cluster/environment/datacentre context to distinguish otherwise identical names. Grouping SHALL operate immediately on loaded data and SHALL preserve existing favourites, safe endpoint links, diagnostics, responsive layout and keyboard access.

Manual omitted dimensions SHALL use typed shared buckets rather than unknown fallbacks. Composite grouping SHALL preserve each shared component, and namespace grouping SHALL distinguish cluster-qualified and shared-cluster namespace identities. Shared cards SHALL appear once, with supplied relevance and separate declaration provenance; storage location SHALL NOT be shown as relevance. Concrete groups SHALL sort before shared groups and shared before unknown groups without changing existing concrete/unknown order. Cards without a supplied dimension SHALL omit its relevance row.

#### Scenario: Nested grouping with an unclassified source
- **WHEN** the user groups by Environment then Cluster and a source has no environment
- **THEN** it appears beneath a distinct unclassified environment section and its actual cluster, with its links and diagnostics preserved

#### Scenario: Change grouping without collection
- **WHEN** the user switches from a label group to a datacentre or cluster group
- **THEN** loaded sources regroup immediately without a Kubernetes request

#### Scenario: Shared versus unclassified grouping
- **WHEN** a global manual link and a discovered resource with missing environment are grouped by environment
- **THEN** the manual link appears under Shared across environments and the discovered source under the existing unclassified fallback
- **AND** neither bucket collides with an actual value having the same displayed text

### Requirement: Shared update controls and accurate freshness UI
All inventory refresh controls SHALL request the same server-owned update process and display active progress/cooldown consistently across viewers. Visibility-aware browser polling SHALL query only snapshot/status APIs and SHALL reload relevant data when generation changes. Freshness UI SHALL show actual successful collection age, next automatic attempt and mixed stale/failed states, using server timing metadata. Application availability SHALL remain a separate indicator. Updates SHALL preserve active scope, personal settings and the existing late-response view guard; requests SHALL NOT replace another active view with stale responses.

Manual-link freshness SHALL follow declaration provenance and per-document outcomes, independently of declared relevance. Scope-specific status SHALL include any manual document contributing entries to that scope even when stored in a different cluster. Never-successful manual collectors SHALL remain visible as incomplete discovery because their unseen entries could be shared. Mixed document outcomes SHALL NOT present retained entries as freshly collected.

#### Scenario: Another viewer starts discovery
- **WHEN** one viewer starts a shared refresh while another is viewing an inventory page
- **THEN** both can see the same run state and receive the completed generation without the second viewer starting another scan

#### Scenario: Service runs with stale inventory
- **WHEN** AutoDocs responds to its liveness check but a target refresh fails
- **THEN** the UI can show Running alongside that target's stale collection age and next attempt

#### Scenario: Directory view changes during refresh
- **WHEN** a refresh response for a prior view arrives after the user has switched tabs
- **THEN** it does not replace the active tab's data, warnings or scope

#### Scenario: Shared link stored outside selected cluster
- **WHEN** a shared link stored in cluster A is visible while cluster B is selected and A cannot refresh
- **THEN** the retained link and its original success time remain available with A's collection failure visible
- **AND** successful refresh of B does not mark that shared link fresh
