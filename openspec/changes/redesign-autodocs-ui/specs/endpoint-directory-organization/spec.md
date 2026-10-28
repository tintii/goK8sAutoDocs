## MODIFIED Requirements

### Requirement: Label-based application groups and Ungrouped

Applications SHALL default to grouping resources by their own non-empty `k8sautodocs.io/group` label, with one group per source resource. Missing or empty group values SHALL place published entries in a distinct Ungrouped fallback section. Label group identity SHALL preserve exact spelling and case, SHALL allow membership across namespaces and source kinds, and SHALL NOT inherit labels from related resources. In Label mode, group headings SHALL be ordered by group value, followed by the fallback section. Cards SHALL be ordered by namespace, kind, and name and show namespace and kind. Each source SHALL appear once with all its resolved links, except that a link-only Favourites match SHALL show only its individually starred URLs. Users SHALL be able to select Namespace, Kind, Custom or None as a browser-only presentation alternative without changing source group labels or publication eligibility. Custom groups SHALL follow saved user ordering followed by their unassigned fallback; other derived groups SHALL use deterministic value ordering.

#### Scenario: Group spans namespaces
- **WHEN** a published Ingress and published HTTPRoute in different namespaces share group `media`
- **THEN** Label mode renders both under the same media section with their own namespace and kind visible

#### Scenario: Published without a usable group
- **WHEN** published sources have missing or empty group labels
- **THEN** Label mode places them under Ungrouped and they remain absent from Discovered

#### Scenario: Group comes from the source resource
- **WHEN** a published source has no group label but its namespace, Gateway, Service, Deployment, or Argo Application has one
- **THEN** the source remains in Ungrouped in Label mode

#### Scenario: Exact group identity and fallback collision
- **WHEN** published sources have group values `media`, `Media`, and `Ungrouped`, and another source has no group
- **THEN** Label mode gives each exact group value its own section
- **AND** the unlabelled source belongs to a separate fallback section without merging into the literal `Ungrouped` group

#### Scenario: Personal layout preserves original metadata
- **WHEN** a published source labelled `media` is assigned to a custom group and the user selects Custom or Namespace
- **THEN** it appears once according to the selected presentation mode
- **AND** its original media label remains unchanged and searchable

### Requirement: Active-view search and navigation

The selected tab SHALL be represented in the page URL and restored through direct links, reloads, and browser back/forward navigation. An absent or unknown page view SHALL select Applications. Tabs SHALL support keyboard navigation and expose their selected state. Search SHALL operate within the active loaded view and preserve its text across switches. Search SHALL trim input, match case-insensitively and require every whitespace-separated literal term to match at least one searchable field. Fields SHALL include source identity, ingress class, source group value, assigned personal group name, hosts, endpoint URLs, Gateway names/namespaces, listeners, and existing related-resource names, namespaces, kinds and label values. Exact namespace/kind selectors and Favourites SHALL compose with search using intersection, with the query and selector values retained across view switches. Counters SHALL describe the active complete loaded view and the result message SHALL separately describe matching resources and displayed URLs. The dashboard SHALL distinguish an empty published view, an empty discovered view, no favourites and no search/filter matches, and SHALL omit empty result group sections. Ctrl/Cmd+K outside editable controls SHALL focus search; Escape in search and an explicit clear button SHALL clear its query without changing other filters. Search input and personal names SHALL be treated as text rather than markup or executable patterns.

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
