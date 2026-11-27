## MODIFIED Requirements

### Requirement: Label-based application groups and Ungrouped

When Label grouping is selected, Applications SHALL group resources by their own non-empty `k8sautodocs.io/group` label, with one group per source resource. Missing or empty group values SHALL place published entries in a distinct Ungrouped fallback section. Group identity SHALL preserve exact spelling and case, SHALL allow membership across clusters, namespaces and source kinds, and SHALL NOT inherit this group label from related resources or normalize it using environment/datacentre matching rules. Group headings SHALL follow the configured shared order for exact group values, then unlisted values in the existing exact-value order, followed by the fallback section. Kubernetes source cards SHALL be ordered deterministically by cluster ID, namespace, kind, and name and show cluster, namespace and kind. Manual cards SHALL show supplied relevance fields and Manual link kind, use declared group projected into the group-label metadata, and have deterministic ordering with stable source identity as a tie-breaker. Omitted relevance fields SHALL NOT display storage location as a substitute. Each source SHALL appear once in the ordinary groups with all its resolved links and available DNS metadata, except that link-only Favourites matches SHALL show only individually starred URLs. DNS-only Service entries SHALL remain eligible for group favourites. Applications SHALL default to label grouping; personal groups SHALL retain saved ordering and the unassigned fallback. Additional personal and classification grouping modes SHALL NOT alter publication eligibility or source identity.

#### Scenario: Group spans namespaces
- **WHEN** a published Ingress, published HTTPRoute and published Service in different namespaces share group `media`
- **THEN** Applications renders all three under the same media section with their own cluster, namespace and kind visible

#### Scenario: Published without a usable group
- **WHEN** published sources have missing or empty group labels
- **THEN** they appear under Ungrouped and remain absent from Discovered

#### Scenario: Group comes from the source resource
- **WHEN** a published source has no group label but its namespace, Gateway, Service, Deployment, or Argo Application has one
- **THEN** the source remains in Ungrouped

#### Scenario: Exact group identity and fallback collision
- **WHEN** published sources have group values `media`, `Media`, and `Ungrouped`, and another source has no group
- **THEN** each exact group value has its own section
- **AND** the unlabelled source belongs to a separate fallback section without merging into the literal `Ungrouped` group

#### Scenario: Personal layout preserves original metadata
- **WHEN** a published source labelled `media` is assigned to a custom group and the user selects Custom or Namespace
- **THEN** its ordinary card appears once according to the selected presentation mode
- **AND** its original media label remains unchanged and searchable

### Requirement: View-aware API and bounded caches

The Auto-Docs API SHALL accept an omitted or empty `view`, or `view=published`, for published results and `view=discovered` for discovered results. Both views SHALL retain the existing array schema with additive cluster/classification and Service DNS metadata and the warnings mechanism. Unknown or repeated view parameters SHALL return HTTP 400 before snapshot access. Payloads and warnings SHALL be projected by normalized view and remain distinct from filtered-resources. Concurrent GETs SHALL be safe and SHALL NOT initiate Kubernetes collection. Published and discovered SHALL use a shared published application generation and source partitions rather than independently expiring request caches. Filtered-resources SHALL remain published-only.

Published results SHALL additionally include normalized manual-link entries with explicit source type, stable identity, title, optional relevance metadata and separate declaration provenance. Discovered and the Kubernetes filtered-resources projection SHALL NOT include synthetic manual-link rows. Existing generic ConfigMap resource eligibility SHALL remain unchanged. Published directory rows SHALL support optional featured-link ranks for matching URLs and a shared group rank for ranked source labels. Curation SHALL preserve the response array shape and SHALL be limited to the Applications view.

#### Scenario: Default API compatibility
- **WHEN** a client calls the Auto-Docs API without a view, with an empty view, or with `view=published`
- **THEN** it receives the published-only array schema with additive cluster/classification context

#### Scenario: Request order does not mix views
- **WHEN** published and discovered requests arrive in either order or concurrently within a snapshot generation
- **THEN** each response contains only its own view's entries and warnings
- **AND** a filtered-resources response never supplies either Auto-Docs view's payload

#### Scenario: Invalid view request
- **WHEN** a request supplies an unknown view or more than one view parameter
- **THEN** the API responds with HTTP 400 without collecting resources or accessing the snapshot

#### Scenario: Services share one directory generation
- **WHEN** Applications and Discovered are requested after a Service partition refresh
- **THEN** both project that same partition according to publication metadata without new Kubernetes reads

#### Scenario: Manual declarations do not synthesize workloads or filtered resources
- **WHEN** a snapshot includes manual declarations and ordinary endpoint/workload resources
- **THEN** manual rows occur only in published directory data
- **AND** no manual-link rows are inserted into Discovered, Versions, workload status or Kubernetes filtered-resources data

### Requirement: Active-view search and navigation

The selected tab SHALL be represented in the page URL and restored through direct links, reloads, and browser back/forward navigation. An absent or unknown page view SHALL select Applications. Tabs SHALL support keyboard navigation and expose their selected state. Search SHALL operate within the active view, preserve the search text and cluster/environment/datacentre scope across switches, retain existing searchable fields, and include source group values, cluster context and resolved classifications. Counters SHALL describe the active loaded scope and distinguish namespaces by cluster ID plus namespace; the search result message SHALL describe its filtered subset. The dashboard SHALL distinguish an empty published view, an empty discovered view, no initial usable data, partial/stale inventory and no search matches, and SHALL omit empty group sections. Search SHALL trim input, match case-insensitively and require every whitespace-separated literal term to match a searchable field. Existing searchable fields SHALL include ingress class, personal group names, endpoint URLs, Gateway/listener metadata and related-resource names, namespaces, kinds and label values. Service search SHALL additionally include Service type, advertised DNS names, selected targets and declared externalName. Namespace/kind selectors and Favourites SHALL intersect with search and cluster/environment/datacentre scope, retaining unavailable selections as clearable values. Namespace matching SHALL remain exact for discovered resources; manual links SHALL match every selected namespace when namespace relevance is omitted, and otherwise require exact namespace equality. Kind matching SHALL remain exact for all sources. Manual-link title, URL, group and declared normalized relevance SHALL be searchable; storage provenance SHALL NOT populate relevance filters or namespace counts. Counters SHALL distinguish matching resources from displayed web URLs so DNS-only entries count as resources but not links. The interface SHALL distinguish no favourites from no filter matches. Ctrl/Cmd+K outside editable controls SHALL focus search; Escape in search and a clear button SHALL clear only the query. User strings SHALL be treated as text rather than markup or executable patterns.

Both directory routes SHALL focus search without scrolling on initial desktop entry, including direct links to Discovered. Touch-first entry SHALL leave opening the software keyboard to a deliberate tap. Later renders, view transitions, and browser history restoration SHALL respect the user’s current focus. While search is active and the trimmed query is non-empty, the directory SHALL offer quick results from the same final filtered rows as the main directory. Each source/URL pair SHALL appear at most once in quick results. Matching sources without web URLs SHALL offer a way to focus their ordinary source card. Users SHALL be able to navigate quick results with Up and Down and use Enter to open the active link or focus the active source. During loading or error, quick search SHALL reflect the active view’s status, and it SHALL identify an empty match set without offering a stale result. Clearing the query SHALL dismiss quick results without clearing other filters.

#### Scenario: Search by group across namespaces
- **WHEN** a user searches for `media` in Applications
- **THEN** matching group-labelled sources are found even when their names and namespaces do not contain `media`
- **AND** empty sections are omitted

#### Scenario: Switch tabs with a search
- **WHEN** a user enters a query and selects Discovered
- **THEN** the query and scope are retained and applied only to discovered entries
- **AND** counters and empty-state messages refer to Discovered

#### Scenario: Restore a discovered link
- **WHEN** a user opens or reloads a page URL selecting Discovered, or navigates back to it
- **THEN** Discovered is selected and loads its own results with the encoded scope

#### Scenario: Terms span metadata fields
- **WHEN** a user searches `media httproute` and selects namespace `apps`
- **THEN** only active-view sources in apps with both terms matched across their indexed fields are displayed
- **AND** a matching personal group name can satisfy a term without changing source labels

#### Scenario: Retain a filter unavailable in the next view
- **WHEN** a user switches tabs with a namespace filter that has no sources in the requested view
- **THEN** the selection remains visible and clearable, and the result explains that no sources match
- **AND** no resources from the previous tab remain visible

#### Scenario: Search shortcuts preserve editing
- **WHEN** a user invokes Ctrl/Cmd+K outside an editable control
- **THEN** search receives focus
- **WHEN** a user presses Escape in search
- **THEN** the query clears while namespace/kind and Favourites selections remain
- **AND** typing shortcuts within the custom-group editor is not intercepted

#### Scenario: Search external destinations
- **WHEN** a user searches for a Service DNS target and selects kind Service
- **THEN** only matching Service entries in the active view and scope are displayed
- **AND** a matching DNS-only entry contributes to the resource count but not the web URL count

#### Scenario: Manual link obeys non-scope restrictions
- **WHEN** an unscoped manual link is present and the user selects kind Ingress or a nonmatching query, group or favourites restriction
- **THEN** the shared link is excluded by that restriction

#### Scenario: Open the directory with an empty query
- **WHEN** a desktop user opens either directory route on Applications or Discovered
- **THEN** the search field in the header is focused with the cursor ready for typing
- **AND** the empty field displays the muted placeholder “Just type. Look, no clicks.” while retaining its accessible search label
- **AND** the directory stays undimmed and available for browsing while the query is empty or contains only whitespace

#### Scenario: Type immediately after opening the directory
- **WHEN** a desktop user opens either directory route on Applications or Discovered and begins typing
- **THEN** the first keystroke goes into search
- **AND** the search field and quick results appear together in a centered palette above the dimmed directory
- **AND** the query, caret and keyboard focus are retained

#### Scenario: Activate a result from the keyboard
- **WHEN** a user enters a query and uses the arrow keys followed by Enter
- **THEN** the active match opens its exact current URL in a new tab or focuses the matching source card when there is no URL
- **AND** a match excluded by the current scope, group or Favourites settings is absent

#### Scenario: Switch views while a query is present
- **WHEN** a user switches views while search has a non-empty query
- **THEN** old quick results stop being available immediately
- **AND** returning to search shows the new view’s matching results or its loading, unavailable or no-match state

#### Scenario: Open on a touch-first device
- **WHEN** a user opens the directory on a touch-first device
- **THEN** the software keyboard stays closed until they activate search
