## ADDED Requirements

### Requirement: Configurable independent classification dimensions
AutoDocs SHALL resolve environment and datacentre independently using configured label keys and ordered lookup sources. Supported sources SHALL be the documented source resource's labels, its namespace's labels and explicitly configured target labels. For each dimension, source order SHALL take precedence, followed by key order within that source; empty or absent values SHALL permit fallback. Directory source resources SHALL be Ingresses/HTTPRoutes and workload source resources SHALL be the workloads themselves. No built-in environment vocabulary, prefix/suffix parsing, cluster-name extraction or arbitrary related-resource inheritance SHALL determine classification. An environment SHALL be able to span targets and a target SHALL be able to contain multiple environments.

#### Scenario: Many environments and clusters
- **WHEN** resources in two targets carry the same configured environment value and another resource in the first target carries a different value
- **THEN** classification groups the matching resources across targets and keeps the second environment distinct within the first target

#### Scenario: Ordered label fallback
- **WHEN** a source lacks the first configured key but has another configured key or its namespace has a usable value
- **THEN** the documented source/key precedence selects one deterministic value for that dimension

#### Scenario: Arbitrary environment names
- **WHEN** the configured environment label contains `NPE00`, `PRD02`, `CORE01` or an unrelated value
- **THEN** each value is accepted without hardcoded interpretation or prefix removal

### Requirement: Matching identity is independent of display casing
Each dimension SHALL support case-sensitive or case-insensitive value equality and an independent preserve/upper/lower display format. Label keys SHALL use exact lookup. The backend SHALL provide a deterministic canonical identity and display value used consistently for grouping, filters, counts, search and preference identities. Case-insensitive mode SHALL identify `npe00`, `NPE00` and `Npe00` as the same environment while keeping `NPE01` separate. Preserve display mode SHALL choose a deterministic observed spelling independent of collection order. Source labels SHALL retain their original values. Matching SHALL NOT imply fuzzy, substring or regex classification.

#### Scenario: Case-insensitive equality with uppercase display
- **WHEN** environment matching is case-insensitive with uppercase display and sources contain `npe00` and `NPE00`
- **THEN** both sources share one canonical environment and one displayed `NPE00` group

#### Scenario: Case-sensitive equality
- **WHEN** matching is case-sensitive and sources contain `npe00` and `NPE00`
- **THEN** their canonical identities remain distinct even if a configured presentation format renders equal-looking labels

#### Scenario: Stable preserve mode
- **WHEN** the same observed spellings arrive in a different collection order
- **THEN** the chosen preserved display spelling and canonical identities do not change

### Requirement: Missing metadata never removes discovered resources
Missing, empty or unreadable classification metadata SHALL NOT remove otherwise eligible resources. Each available dimension SHALL remain independently usable. Unavailable namespace metadata SHALL yield a diagnostic while allowing configured fallback. Grouping SHALL provide typed fallback sections for unclassified values and filters SHALL make that state clearable/selectable. Synthetic fallback identities SHALL differ from real label values with equal display text.

#### Scenario: Partial classification
- **WHEN** a source has cluster and datacentre context but no resolvable environment
- **THEN** it remains visible, participates in cluster/datacentre grouping and appears in the environment fallback when that grouping is selected

#### Scenario: Namespace metadata forbidden
- **WHEN** namespace label lookup is forbidden but the source is readable
- **THEN** AutoDocs retains the source, reports the lookup issue and uses any permitted configured fallback without inventing an environment

#### Scenario: Literal fallback label
- **WHEN** one source has a real environment value `Unclassified` and another has no environment
- **THEN** they retain distinct group identities and preference records
