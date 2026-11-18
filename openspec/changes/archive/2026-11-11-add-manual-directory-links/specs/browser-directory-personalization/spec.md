## MODIFIED Requirements

### Requirement: Independent group and link favourites

The directory SHALL provide independent star toggles for groups in either view and each distinct resolved source URL in either view. Discovered link identity SHALL include stable cluster ID, source kind, namespace, name and exact URL. Manual link identity SHALL instead include typed declaration cluster ID, ConfigMap namespace/name, entry ID and exact URL; editable relevance, title and group SHALL NOT be identity components. Namespace-group identity SHALL include concrete cluster ID or an explicit shared-cluster component as applicable, while group identity SHALL otherwise include its type and exact value or stable custom ID. Shared, unknown and literal-value groups SHALL have distinct typed identities. Source/URL ordering, group renaming and unrelated grouping changes SHALL NOT transfer stars to another entity. Equal URLs on different sources SHALL remain independent. Starring a group SHALL NOT mutate member link stars, and un-starring a member link SHALL NOT remove the group star. Group stars SHALL follow current membership even when another grouping mode is displayed.

#### Scenario: Same URL from different sources
- **WHEN** two sources expose the same URL and a user stars only the first source's link
- **THEN** only that source/link identity is independently starred
- **AND** reordering responses or endpoint arrays does not change the selection

#### Scenario: Group membership changes
- **WHEN** a source leaves a starred label group and another source joins it in refreshed Applications data
- **THEN** the group favourite includes the joining source and stops including the departing source
- **AND** independent link favourites remain unchanged

#### Scenario: Manual scope edit retains independent favourites
- **WHEN** a manual link changes title or declared scope without changing declaration location, entry ID or URL
- **THEN** its independent link star and custom assignment remain attached while current filter and group membership update

#### Scenario: Manual URL edit does not transfer an exact-URL star
- **WHEN** a manual entry retains its source identity but changes its URL
- **THEN** its custom assignment remains and an independent star for the old URL stays dormant
- **AND** a current group favourite can still include the source through live membership
