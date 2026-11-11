## MODIFIED Requirements

### Requirement: Label-based application groups and Ungrouped

When Label grouping is selected, Applications SHALL group resources by their own non-empty `k8sautodocs.io/group` label, with one group per source resource. Missing or empty group values SHALL place published entries in a distinct Ungrouped fallback section. Group identity SHALL preserve exact spelling and case, SHALL allow membership across clusters, namespaces and source kinds, and SHALL NOT inherit this group label from related resources or normalize it using environment/datacentre matching rules. Group headings SHALL be ordered by group value, followed by the fallback section. Cards SHALL be ordered deterministically by cluster ID, namespace, kind, and name and show cluster, namespace and kind. Each source SHALL appear once with all its resolved links. Additional personal and classification grouping modes SHALL NOT alter publication eligibility or source identity.

#### Scenario: Group spans namespaces
- **WHEN** a published Ingress and published HTTPRoute in different namespaces share group `media`
- **THEN** Applications renders both under the same media section with their own cluster, namespace and kind visible

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

### Requirement: Discovered resource presentation

Discovered SHALL support the same personal and classification grouping controls as Applications, including a flat None mode. Within applicable groups, source cards SHALL be ordered by cluster ID, namespace, kind, and name, with cluster, namespace and kind visible. Both views SHALL retain resolved links, unresolved entries, issues, and related-resource context. Links SHALL use the exact server-resolved URL with `target="_blank"` and `rel="noopener noreferrer"`. Discovered SHALL remain available on Ingress-only clusters and SHALL expose partial/stale collection warnings when a supported source API cannot be read. Classification labels or personal grouping SHALL NOT publish a resource.

#### Scenario: Discovered endpoint has no resolved URL
- **WHEN** a discovered HTTPRoute has an unavailable Gateway parent
- **THEN** its card remains visible with its identity and resolution issue and no invented link

#### Scenario: Optional or inaccessible Gateway API
- **WHEN** the Gateway API is absent or HTTPRoute listing is forbidden
- **THEN** available discovered Ingresses remain visible
- **AND** a forbidden list request is reported as incomplete discovery while an absent optional API is tolerated

### Requirement: View-aware API and bounded caches

The Auto-Docs API SHALL accept an omitted or empty `view`, or `view=published`, for published results and `view=discovered` for discovered results. Both views SHALL retain the existing array schema with additive cluster/classification metadata and the warnings mechanism. Unknown or repeated view parameters SHALL return HTTP 400 before snapshot access. Payloads and warnings SHALL be projected by normalized view and remain distinct from filtered-resources. Concurrent GETs SHALL be safe and SHALL NOT initiate Kubernetes collection. Published and discovered SHALL use a shared published application generation and source partitions rather than independently expiring request caches. Filtered-resources SHALL remain published-only.

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

### Requirement: Active-view search and navigation

The selected tab SHALL be represented in the page URL and restored through direct links, reloads, and browser back/forward navigation. An absent or unknown page view SHALL select Applications. Tabs SHALL support keyboard navigation and expose their selected state. Search SHALL operate within the active view, preserve the search text and cluster/environment/datacentre scope across switches, retain existing searchable fields, and include source group values, cluster context and resolved classifications. Counters SHALL describe the active loaded scope and distinguish namespaces by cluster ID plus namespace; the search result message SHALL describe its filtered subset. The dashboard SHALL distinguish an empty published view, an empty discovered view, no initial usable data, partial/stale inventory and no search matches, and SHALL omit empty group sections.

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

### Requirement: Metadata refresh remains read-only

Group and publication changes SHALL be read from Kubernetes and reflected after the source partition successfully refreshes through the shared coordinator, without writing metadata or maintaining a separate publication database. Published and discovered SHALL classify the same retained source partition within a published generation. A generation SHALL be an application publication boundary and SHALL NOT claim an atomic read of Kubernetes. Failed collection SHALL retain prior successful data and timestamps with visible diagnostics rather than inventing deletion or fresh metadata.

#### Scenario: Move from discovery to a named group
- **WHEN** a discovered source receives exact-true publication and group `media`, and its source partition refreshes successfully
- **THEN** refreshed Applications places it in media and refreshed Discovered excludes it

#### Scenario: Remove group, remove publication, then hide
- **WHEN** a published source's group is removed and its source partition refreshes successfully
- **THEN** it moves to Ungrouped
- **WHEN** its publication label is subsequently removed and its source partition refreshes successfully
- **THEN** it moves to Discovered
- **WHEN** its publication label is then set to `"false"` and its source partition refreshes successfully
- **THEN** it appears in neither view
