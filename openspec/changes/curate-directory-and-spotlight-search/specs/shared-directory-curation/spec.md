## ADDED Requirements

### Requirement: Deployment-managed shared directory settings

AutoDocs SHALL load shared directory settings from optional deployment configuration writable by deployment admins. The settings SHALL define an ordered list of feature references and an ordered list of source-label group values. Feature references SHALL identify an exact HTTP(S) URL on a specific source, including published Kubernetes sources and manual link declarations. Published directory reads SHALL reflect file updates without a server restart. An unset configuration SHALL use default curation. An unreadable or invalid configured file SHALL use default curation and provide a sanitized warning while preserving usable directory results. Curation SHALL leave publication decisions to the existing inventory rules.

#### Scenario: Apply settings across browsers
- **WHEN** an admin updates the mounted settings with a different featured link and group order
- **THEN** subsequent Applications loads across browsers use the same updated selection and order

#### Scenario: Invalid deployment configuration
- **WHEN** the configured file is missing, unreadable, or has invalid selections such as duplicate references
- **THEN** Applications renders usable inventory with the default order and a directory settings warning

### Requirement: Featured links lead Applications

Applications SHALL display matching featured links in configuration order under a distinct Featured heading before other groups or a flat list. Featured links SHALL resolve only from current published response sources with an exact matching source and URL. Apply the same scope, search, resource, group and Favourites filters to Featured using the exact matched links. Omit the section when no featured links qualify. The directory SHALL retain the ordinary source cards, link identities and counts, and SHALL exclude Featured from Discovered and the group filter choices.

#### Scenario: Exact feature on a multi-link source
- **WHEN** an admin features one of two URLs on a published source and only one of two sources that share a URL
- **THEN** only the configured source and URL pairs appear in Featured
- **AND** the ordinary source cards still show their eligible links without inflating the counts

#### Scenario: A feature no longer resolves
- **WHEN** a configured manual link retains its declaration identity but changes URL, or a configured Kubernetes source leaves Applications
- **THEN** the stale feature is omitted
- **AND** an exact source and URL pair can appear again if it returns to the published response

#### Scenario: Directory filters apply
- **WHEN** the active scope, group, query or Favourites filter excludes the last eligible featured link
- **THEN** the Featured heading disappears
- **AND** the ordinary directory continues to show its matching results

### Requirement: Shared ordering of source-label groups

With Source label grouping selected in Applications, the directory and its sidebar SHALL put present configured group values first in the specified order and then unlisted group values in their existing order. Matching SHALL use exact group values and SHALL omit any configured value with no corresponding group. The synthetic Ungrouped section SHALL follow all named groups. Other grouping modes and Discovered SHALL retain their existing order.

#### Scenario: Prioritise operator-selected groups
- **WHEN** an admin ranks `observability` then `media-services` and Applications also contains an unranked `automation` group and unlabelled sources
- **THEN** the ordinary section and sidebar orders are observability, media-services, automation, then the Ungrouped fallback

#### Scenario: Preserve group identity
- **WHEN** the order includes a literal group value `Ungrouped` and a presently absent group value
- **THEN** the literal group uses its configured position
- **AND** the absent group produces no placeholder and the synthetic fallback remains last
