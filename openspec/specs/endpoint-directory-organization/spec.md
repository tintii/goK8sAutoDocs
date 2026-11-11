# endpoint-directory-organization Specification

## Purpose

Define application grouping and separate published/discovered directory views, including exact publication eligibility, shared inventory snapshots, search, navigation and browser presentation.

## Requirements

### Requirement: Separate Applications and Discovered views

The dashboard SHALL provide Applications and Discovered tabs, defaulting to Applications. Applications SHALL show exact-true published Ingresses, HTTPRoutes and Services. Discovered SHALL show supported Ingresses and HTTPRoutes with an absent publication-label key, regardless of other labels or legacy annotations; Services SHALL additionally require a supported ExternalDNS hostname annotation key to appear in Discovered. A group label SHALL NOT publish a resource. Exact `"false"` and all other present non-true publication values SHALL exclude resources from both views.

#### Scenario: Curated and discovered resources remain separate
- **WHEN** the cluster contains exact-true, exact-false, absent, empty, and `"True"` publication-label cases
- **THEN** Applications shows only the exact-true sources
- **AND** Discovered shows only sources with an absent publication key, with Services additionally requiring a supported hostname annotation key

#### Scenario: Group without publication intent
- **WHEN** an HTTPRoute has `k8sautodocs.io/group: media` and no publication label
- **THEN** it appears in Discovered and not Applications

#### Scenario: Group-only Service
- **WHEN** a Service has a group label but neither a publication label nor an ExternalDNS hostname annotation
- **THEN** it appears in neither view

### Requirement: Label-based application groups and Ungrouped

When Label grouping is selected, Applications SHALL group resources by their own non-empty `k8sautodocs.io/group` label, with one group per source resource. Missing or empty group values SHALL place published entries in a distinct Ungrouped fallback section. Group identity SHALL preserve exact spelling and case, SHALL allow membership across clusters, namespaces and source kinds, and SHALL NOT inherit this group label from related resources or normalize it using environment/datacentre matching rules. Group headings SHALL be ordered by group value, followed by the fallback section. Cards SHALL be ordered deterministically by cluster ID, namespace, kind, and name and show cluster, namespace and kind. Each source SHALL appear once with all its resolved links and available DNS metadata, except that link-only Favourites matches SHALL show only individually starred URLs. DNS-only Service entries SHALL remain eligible for group favourites. Applications SHALL default to label grouping; personal groups SHALL retain saved ordering and the unassigned fallback. Additional personal and classification grouping modes SHALL NOT alter publication eligibility or source identity.

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
- **THEN** it appears once according to the selected presentation mode
- **AND** its original media label remains unchanged and searchable

### Requirement: Discovered resource presentation

Discovered SHALL support the same personal and classification grouping controls as Applications, including a flat None mode. Within applicable groups, source cards SHALL be ordered by cluster ID, namespace, kind, and name, with cluster, namespace and kind visible. Both views SHALL retain resolved links, DNS-only Service entries, unresolved entries, issues, and related-resource context. Regrouping and personal assignments SHALL operate on the current loaded view without changing publication or fetching another view. Links SHALL use the exact server-resolved URL with `target="_blank"` and `rel="noopener noreferrer"`. Discovered SHALL remain available on Ingress-only clusters and SHALL expose partial/stale collection warnings when a supported source API cannot be read. Classification labels or personal grouping SHALL NOT publish a resource.

#### Scenario: Discovered endpoint has no resolved URL
- **WHEN** a discovered HTTPRoute has an unavailable Gateway parent
- **THEN** its card remains visible with its identity and resolution issue and no invented link

#### Scenario: Optional or inaccessible Gateway API
- **WHEN** the Gateway API is absent or HTTPRoute listing is forbidden
- **THEN** available discovered Ingresses remain visible
- **AND** a forbidden list request is reported as incomplete discovery while an absent optional API is tolerated

#### Scenario: Change grouping in Discovered
- **WHEN** a user selects namespace, kind, source label, personal groups or no grouping in Discovered
- **THEN** the current discovered sources are regrouped immediately without a new directory request
- **AND** publication labels and the Applications resource set remain unchanged

#### Scenario: DNS-only Service in Discovered
- **WHEN** a Service has a hostname annotation, no publication label and no web annotations
- **THEN** Discovered displays its copyable DNS information and source context without inventing a web link

### Requirement: View-aware API and bounded caches

The Auto-Docs API SHALL accept an omitted or empty `view`, or `view=published`, for published results and `view=discovered` for discovered results. Both views SHALL retain the existing array schema with additive cluster/classification and Service DNS metadata and the warnings mechanism. Unknown or repeated view parameters SHALL return HTTP 400 before snapshot access. Payloads and warnings SHALL be projected by normalized view and remain distinct from filtered-resources. Concurrent GETs SHALL be safe and SHALL NOT initiate Kubernetes collection. Published and discovered SHALL use a shared published application generation and source partitions rather than independently expiring request caches. Filtered-resources SHALL remain published-only.

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

### Requirement: Active-view search and navigation

The selected tab SHALL be represented in the page URL and restored through direct links, reloads, and browser back/forward navigation. An absent or unknown page view SHALL select Applications. Tabs SHALL support keyboard navigation and expose their selected state. Search SHALL operate within the active view, preserve the search text and cluster/environment/datacentre scope across switches, retain existing searchable fields, and include source group values, cluster context and resolved classifications. Counters SHALL describe the active loaded scope and distinguish namespaces by cluster ID plus namespace; the search result message SHALL describe its filtered subset. The dashboard SHALL distinguish an empty published view, an empty discovered view, no initial usable data, partial/stale inventory and no search matches, and SHALL omit empty group sections. Search SHALL trim input, match case-insensitively and require every whitespace-separated literal term to match a searchable field. Existing searchable fields SHALL include ingress class, personal group names, endpoint URLs, Gateway/listener metadata and related-resource names, namespaces, kinds and label values. Service search SHALL additionally include Service type, advertised DNS names, selected targets and declared externalName. Exact namespace/kind selectors and Favourites SHALL intersect with search and cluster/environment/datacentre scope, retaining unavailable selections as clearable values. Counters SHALL distinguish matching resources from displayed web URLs so DNS-only entries count as resources but not links. The interface SHALL distinguish no favourites from no filter matches. Ctrl/Cmd+K outside editable controls SHALL focus search; Escape in search and a clear button SHALL clear only the query. User strings SHALL be treated as text rather than markup or executable patterns.

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

### Requirement: Safe transitions between views

Switching tabs SHALL clear prior cards and warnings and show the requested view's loading state. A late response for a previous selection SHALL NOT replace the active view. A failed request SHALL show the active view's error without presenting another view's stale results.

#### Scenario: Responses arrive out of order
- **WHEN** a published request is pending, the user selects Discovered, and the published response arrives last
- **THEN** the page remains in Discovered and displays only the discovered request's state

#### Scenario: Requested view fails
- **WHEN** Discovered fails to load after Applications was visible
- **THEN** the dashboard shows a Discovered loading error without the old Applications cards or warnings

### Requirement: Metadata refresh remains read-only

Group and publication changes SHALL be read from Kubernetes and reflected after the source partition successfully refreshes through the shared coordinator, without writing metadata or maintaining a separate publication database. Published and discovered SHALL classify the same retained source partition within a published generation. A generation SHALL be an application publication boundary and SHALL NOT claim an atomic read of Kubernetes. Failed collection SHALL retain prior successful data and timestamps with visible diagnostics rather than inventing deletion or fresh metadata.

#### Scenario: Move from discovery to a named group
- **WHEN** a discovered source receives exact-true publication and group `media`, and its source partition refreshes successfully
- **THEN** refreshed Applications places it in media and refreshed Discovered excludes it

#### Scenario: Remove group, remove publication, then hide
- **WHEN** a published source's group is removed and its source partition refreshes successfully
- **THEN** it moves to Ungrouped
- **WHEN** its publication label is subsequently removed and its source partition refreshes successfully
- **THEN** it moves to Discovered if it is an Ingress, HTTPRoute or a Service with a supported hostname annotation; otherwise the Service disappears from both views
- **WHEN** its publication label is then set to `"false"` and its source partition refreshes successfully
- **THEN** it appears in neither view

#### Scenario: Hostname annotation removed from a discovered Service
- **WHEN** an unlabelled Service loses its last supported hostname annotation and the Services partition refreshes successfully
- **THEN** it disappears from Discovered
- **AND** if the refresh instead fails, its previous entry remains with stale diagnostics until a successful collection
