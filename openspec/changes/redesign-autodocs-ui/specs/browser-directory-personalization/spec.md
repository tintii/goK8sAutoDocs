## ADDED Requirements

### Requirement: Runtime grouping selection

Applications SHALL offer Label, Namespace, Kind, Custom and None grouping modes. Switching modes SHALL regroup already loaded sources immediately without a network request, Kubernetes edit or redeploy. Label SHALL be the initial default. Every mode SHALL preserve source namespace/kind context, diagnostics and publication boundaries. Discovered SHALL remain flat. Group identities SHALL distinguish modes, exact values and synthetic fallback sections. Grid/list density and group collapse choices SHALL be adjustable and persisted.

#### Scenario: Regroup without cluster edits
- **WHEN** a user switches Applications from Label to Namespace and then None
- **THEN** the same eligible sources appear under namespace headings and then in a flat presentation
- **AND** the change sends no API write or additional directory request

#### Scenario: Distinguish group identities
- **WHEN** a label group, namespace group and custom group have the same displayed name, and a label group is literally named Ungrouped
- **THEN** their stars and collapse settings remain independent
- **AND** the synthetic Ungrouped fallback remains distinct from the literal group

### Requirement: Editable personal groups

Users SHALL be able to create, rename, reorder and delete custom groups, and assign, move or unassign a source through keyboard-operable controls. Names SHALL be trimmed, non-empty and unique by exact value. Custom groups SHALL use stable IDs so rename preserves membership, ordering, collapse state and favourites. Each source SHALL belong to at most one custom group, with unassigned sources in a distinct fallback. Empty groups SHALL remain editable but SHALL NOT create empty result sections. Deletion SHALL unassign members and remove that group's star and presentation records while preserving independent link stars. All edits SHALL apply immediately and SHALL NOT modify source labels or publication state.

#### Scenario: Rename and reorder a starred custom group
- **WHEN** a user creates a custom group, assigns sources, stars it, renames it and changes its order
- **THEN** its membership and star remain associated with the same group
- **AND** Custom mode reflects the new name and order immediately and after reload

#### Scenario: Delete a personal group
- **WHEN** a user deletes a custom group containing a source with an independently starred link
- **THEN** the source becomes unassigned in Custom mode and the deleted group star is removed
- **AND** the link remains starred and the source's Kubernetes labels are unchanged

#### Scenario: Reject unusable names
- **WHEN** a user submits an empty, whitespace-only or exact-duplicate custom group name
- **THEN** an inline validation message identifies the problem without modifying the existing groups

### Requirement: Independent group and link favourites

The directory SHALL provide independent star toggles for Applications groups and each distinct resolved source URL in either view. Link identity SHALL include source kind, namespace, name and exact URL; group identity SHALL include its type and exact value or stable custom ID. Source/URL ordering, group renaming and unrelated grouping changes SHALL NOT transfer stars to another entity. Equal URLs on different sources SHALL remain independent. Starring a group SHALL NOT mutate member link stars, and un-starring a member link SHALL NOT remove the group star. Group stars SHALL follow current membership even when another grouping mode is displayed.

#### Scenario: Same URL from different sources
- **WHEN** two sources expose the same URL and a user stars only the first source's link
- **THEN** only that source/link identity is independently starred
- **AND** reordering responses or endpoint arrays does not change the selection

#### Scenario: Group membership changes
- **WHEN** a source leaves a starred label group and another source joins it in refreshed Applications data
- **THEN** the group favourite includes the joining source and stops including the departing source
- **AND** independent link favourites remain unchanged

### Requirement: View-scoped favourites filtering

The Favourites filter SHALL combine live members of starred groups and individually starred live links using union, then intersect with the active view, search and namespace/kind filters. Group memberships SHALL be evaluated independently of the displayed grouping mode and SHALL apply only to Applications. A group match SHALL retain all of the source's resolved links and unresolved context. A link-only match SHALL retain the source identity, issues and only its individually starred URLs. Stored stars SHALL NOT expose sources absent from the active response or invent links missing from current data. Dormant stars SHALL be retained for restoration when their exact identity returns. An individual link favourite SHALL survive a source moving between Applications and Discovered while respecting each tab's current response.

#### Scenario: Combine group and individual favourites
- **WHEN** a starred group contains one resolved and one unresolved source, another source has one of two URLs starred, and Favourites is enabled
- **THEN** both group members appear with their available links or unresolved explanation
- **AND** the other source displays only its individually starred URL
- **AND** search and namespace/kind selectors further restrict those results

#### Scenario: Missing or reclassified favourite
- **WHEN** a saved link is absent, explicitly hidden from both views, or moves to Discovered
- **THEN** it is never rendered from storage alone or leaked into Applications
- **AND** the star reattaches only when the exact source/URL appears in an eligible active response

### Requirement: Resilient browser-only persistence

The client SHALL persist theme, grouping, density, collapse choices, custom groups, memberships, ordering and favourites in a namespaced, versioned localStorage document scoped to browser profile and origin. It SHALL validate stored shapes and values, handle malformed JSON and read/write failures without preventing browsing, and retain session changes in memory when persistence fails. It SHALL display a non-blocking notice when settings cannot persist. Unsupported future versions SHALL remain untouched. Supported same-origin storage events SHALL refresh preferences in other open tabs. The store SHALL NOT contain API payloads, credentials or search history, and preference actions SHALL NOT write to the server or Kubernetes.

#### Scenario: Restore personal settings
- **WHEN** a user changes grouping, custom membership, density, theme and stars, then reloads the same origin
- **THEN** those supported saved choices are restored without a backend preference request

#### Scenario: Corrupt or blocked storage
- **WHEN** the preference document contains malformed data or browser storage throws on access
- **THEN** the directory remains usable with safe defaults or session settings
- **AND** it shows a persistence notice rather than an uncaught initialization error

#### Scenario: Newer schema and another tab
- **WHEN** the browser reads an unsupported future schema version
- **THEN** it uses session defaults with a notice and does not overwrite that document
- **WHEN** another tab subsequently writes supported valid preferences
- **THEN** the current tab applies that supported update

### Requirement: Separate layout and favourite resets

The settings UI SHALL offer separate Reset layout and Clear favourites actions. Reset layout SHALL restore default grouping, density and collapse settings and remove custom groups/memberships/order, while retaining independent link stars and derived-group stars. Stars referring to deleted custom groups SHALL be removed. Clear favourites SHALL remove all group/link stars while preserving layout, memberships and theme.

#### Scenario: Reset only layout
- **WHEN** a user resets layout while custom-group, label-group and link stars exist
- **THEN** Label mode and default presentation return, and custom groups and their stars are removed
- **AND** independent link and label-group stars and theme remain

#### Scenario: Clear only favourites
- **WHEN** a user clears favourites
- **THEN** all stars are removed while custom groups, membership, ordering, grouping and theme remain
