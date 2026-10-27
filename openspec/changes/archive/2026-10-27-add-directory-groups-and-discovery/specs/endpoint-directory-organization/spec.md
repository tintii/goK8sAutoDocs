## ADDED Requirements

### Requirement: Separate Applications and Discovered views

The dashboard SHALL provide Applications and Discovered tabs, defaulting to Applications. Applications SHALL show exact-true published Ingresses and HTTPRoutes. Discovered SHALL show supported Ingresses and HTTPRoutes with an absent publication-label key, regardless of other labels or legacy annotations. A group label SHALL NOT publish a resource. Exact `"false"` and all other present non-true publication values SHALL exclude resources from both views.

#### Scenario: Curated and discovered resources remain separate
- **WHEN** the cluster contains exact-true, exact-false, absent, empty, and `"True"` publication-label cases
- **THEN** Applications shows only the exact-true sources
- **AND** Discovered shows only the sources with an absent key

#### Scenario: Group without publication intent
- **WHEN** an HTTPRoute has `k8sautodocs.io/group: media` and no publication label
- **THEN** it appears in Discovered and not Applications

### Requirement: Label-based application groups and Ungrouped

Applications SHALL group resources by their own non-empty `k8sautodocs.io/group` label, with one group per source resource. Missing or empty group values SHALL place published entries in a distinct Ungrouped fallback section. Group identity SHALL preserve exact spelling and case, SHALL allow membership across namespaces and source kinds, and SHALL NOT inherit labels from related resources. Group headings SHALL be ordered by group value, followed by the fallback section. Cards SHALL be ordered by namespace, kind, and name and show namespace and kind. Each source SHALL appear once with all its resolved links.

#### Scenario: Group spans namespaces
- **WHEN** a published Ingress and published HTTPRoute in different namespaces share group `media`
- **THEN** Applications renders both under the same media section with their own namespace and kind visible

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

Discovered SHALL use a flat list of source cards ordered by namespace, kind, and name, with namespace and kind visible. Both views SHALL retain resolved links, unresolved entries, issues, and related-resource context. Links SHALL use the exact server-resolved URL with `target="_blank"` and `rel="noopener noreferrer"`. Discovered SHALL remain available on Ingress-only clusters and SHALL expose partial collection warnings when a supported source API cannot be read.

#### Scenario: Discovered endpoint has no resolved URL
- **WHEN** a discovered HTTPRoute has an unavailable Gateway parent
- **THEN** its card remains visible with its identity and resolution issue and no invented link

#### Scenario: Optional or inaccessible Gateway API
- **WHEN** the Gateway API is absent or HTTPRoute listing is forbidden
- **THEN** available discovered Ingresses remain visible
- **AND** a forbidden list request is reported as incomplete discovery while an absent optional API is tolerated

### Requirement: View-aware API and bounded caches

The Auto-Docs API SHALL accept an omitted or empty `view`, or `view=published`, for published results and `view=discovered` for discovered results. Both views SHALL retain the existing array schema and warnings mechanism. Unknown or repeated view parameters SHALL return HTTP 400 before resource collection or cache lookup. Payloads, warnings, and cache expiry SHALL be isolated by normalized view and from filtered-resources, with bounded cache partitions safe for concurrent requests. Filtered-resources SHALL remain published-only.

#### Scenario: Default API compatibility
- **WHEN** a client calls the Auto-Docs API without a view, with an empty view, or with `view=published`
- **THEN** it receives the existing published-only array schema

#### Scenario: Request order does not mix views
- **WHEN** published and discovered requests arrive in either order or concurrently within the cache interval
- **THEN** each response contains only its own view's entries and warnings
- **AND** a filtered-resources response never supplies either Auto-Docs view's payload

#### Scenario: Invalid view request
- **WHEN** a request supplies an unknown view or more than one view parameter
- **THEN** the API responds with HTTP 400 without collecting resources or accessing a view cache

### Requirement: Active-view search and navigation

The selected tab SHALL be represented in the page URL and restored through direct links, reloads, and browser back/forward navigation. An absent or unknown page view SHALL select Applications. Tabs SHALL support keyboard navigation and expose their selected state. Search SHALL operate within the active view, preserve the search text across switches, retain existing searchable fields, and include source group values. Counters SHALL describe the active loaded view and the search result message SHALL describe its filtered subset. The dashboard SHALL distinguish an empty published view, an empty discovered view, and no search matches, and SHALL omit empty group sections.

#### Scenario: Search by group across namespaces
- **WHEN** a user searches for `media` in Applications
- **THEN** matching group-labelled sources are found even when their names and namespaces do not contain `media`
- **AND** empty sections are omitted

#### Scenario: Switch tabs with a search
- **WHEN** a user enters a query and selects Discovered
- **THEN** the query is retained and applied only to discovered entries
- **AND** counters and empty-state messages refer to Discovered

#### Scenario: Restore a discovered link
- **WHEN** a user opens or reloads a page URL selecting Discovered, or navigates back to it
- **THEN** Discovered is selected and loads its own results

### Requirement: Safe transitions between views

Switching tabs SHALL clear prior cards and warnings and show the requested view's loading state. A late response for a previous selection SHALL NOT replace the active view. A failed request SHALL show the active view's error without presenting another view's stale results.

#### Scenario: Responses arrive out of order
- **WHEN** a published request is pending, the user selects Discovered, and the published response arrives last
- **THEN** the page remains in Discovered and displays only the discovered request's state

#### Scenario: Requested view fails
- **WHEN** Discovered fails to load after Applications was visible
- **THEN** the dashboard shows a Discovered loading error without the old Applications cards or warnings

### Requirement: Metadata refresh remains read-only

Group and publication changes SHALL be read from Kubernetes and reflected after the relevant cache expires without writing metadata or maintaining a separate publication database. Refresh guarantees SHALL be per view and SHALL NOT claim an atomic snapshot across both caches.

#### Scenario: Move from discovery to a named group
- **WHEN** a discovered source receives exact-true publication and group `media`, and both relevant caches expire
- **THEN** refreshed Applications places it in media and refreshed Discovered excludes it

#### Scenario: Remove group, remove publication, then hide
- **WHEN** a published source's group is removed and the published cache expires
- **THEN** it moves to Ungrouped
- **WHEN** its publication label is subsequently removed and both caches expire
- **THEN** it moves to Discovered
- **WHEN** its publication label is then set to `"false"` and both caches expire
- **THEN** it appears in neither view
