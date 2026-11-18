# manual-directory-links Specification

## Purpose

Define declarative manual directory links, optional relevance scope, stable identity and provenance-aware collection.

## Requirements

### Requirement: Explicit ConfigMap link declarations
AutoDocs SHALL list ConfigMaps with exact label `k8sautodocs.io/link-source: "true"` in each configured collection target through its shared refresh coordinator. Marked ConfigMaps SHALL supply `data["links.yaml"]` with one version-1 YAML document containing a required `links` list. Each link SHALL have nonempty string `id`, `title` and absolute HTTP(S) `url`, optional nonempty `group`, and optional `scope` with independently optional nonempty string `clusterId`, `namespace`, `environment` and `datacentre`. IDs SHALL be unique and case-sensitive within a document. Missing scope and an empty scope map SHALL mean fully shared. The parser SHALL reject unknown fields, duplicate keys/IDs, null fields, unsupported versions, multiple documents, scope arrays, files above 256 KiB and lists above 500 entries without truncation. Other ConfigMap data keys SHALL be ignored. An empty links list SHALL be valid.

#### Scenario: Explicit publication intent
- **WHEN** a valid ConfigMap has the exact link-source marker without a publication label
- **THEN** its links become eligible manual Applications entries after collection
- **AND** unmarked or differently valued marker ConfigMaps contribute no manual links

#### Scenario: Independent optional scope
- **WHEN** one declaration omits scope, another has only datacentre and another specifies all four dimensions
- **THEN** all three documents are accepted without requiring additional scope fields

#### Scenario: Invalid document or ambiguous identity
- **WHEN** a document contains duplicate IDs, an unsupported field or version, multiple documents, null/empty scope values, arrays or exceeds either bound
- **THEN** the entire document is rejected with a sanitized document diagnostic rather than partially accepted or truncated

### Requirement: Safe exact web destinations
Manual links SHALL support only absolute HTTP(S) destinations with a hostname, valid port syntax, no embedded credentials, no backslashes and no control characters. Accepted URLs SHALL preserve path, query, fragment and explicit port. Titles and groups SHALL render as text; external links SHALL use the existing safe new-tab attributes. Collection SHALL NOT fetch destinations or infer their availability. Diagnostics SHALL identify the source and invalid field without exposing raw document bodies or URL queries.

#### Scenario: Full web URL
- **WHEN** a valid URL includes port 8006, a path, query and fragment
- **THEN** the rendered destination preserves those parts rather than rebuilding a hostname-only URL

#### Scenario: Unsafe URL
- **WHEN** a URL uses a script scheme, lacks a hostname or contains credentials, a backslash, control character or invalid port
- **THEN** document validation fails and no unsafe new destination is rendered

### Requirement: Relevance metadata is independent of declaration provenance
Manual entries SHALL expose explicit manual source type, title, stable source identity, current endpoint and separate declaration provenance. Provenance SHALL identify the collection cluster, ConfigMap namespace/name and entry ID. Declared scope SHALL supply relevance fields without inheriting values from ConfigMap labels, storage namespace or collection target. Cluster names SHALL resolve from the registry; unresolved IDs SHALL remain exact restrictions with a diagnostic and ID display, without creating targets or becoming unrestricted. Namespace relevance SHALL NOT require namespace lookup or cluster scope. Environment/datacentre SHALL use the existing configured matching/display normalization, defaulting to case-sensitive preserved values if unconfigured, and SHALL contribute values to scope choices. Optional group SHALL project to existing exact group metadata, with omission using Ungrouped.

#### Scenario: Central storage does not imply central relevance
- **WHEN** a ConfigMap in cluster A declares a link scoped to cluster B and environment production
- **THEN** its relevance is B/production and its provenance remains A
- **AND** labels or namespace metadata from A do not add other restrictions

#### Scenario: Scope value without discovered peers
- **WHEN** a manual link declares environment core but no discovered resource has that value
- **THEN** core remains selectable using the configured normalization or exact preserved defaults

#### Scenario: Unresolved cluster reference
- **WHEN** a declared cluster ID cannot be resolved in the current registry
- **THEN** its exact restriction is retained with a diagnostic and ID display
- **AND** it does not become visible under a different concrete cluster selection

### Requirement: Optional dimensions express shared relevance
Manual links SHALL match each scope dimension when its filter is unset, the link omits that dimension, or the declared normalized value equals the selection. This SHALL apply to cluster, namespace, environment and datacentre with AND across declared restrictions. An omitted dimension SHALL impose no restriction even for explicit unknown/unclassified filter selections. Search, kind, selected group and favourites SHALL retain their ordinary restrictions. Discovered sources with missing classification SHALL retain existing unknown/unclassified matching and SHALL NOT become shared. Directory and overview favourites SHALL apply identical scope semantics.

#### Scenario: Shared Confluence link
- **WHEN** an unscoped Confluence entry is viewed under any combination of cluster, namespace, environment and datacentre filters
- **THEN** scope alone does not exclude it
- **AND** search, favourites, kind and selected-group predicates still apply

#### Scenario: Partially scoped Proxmox link
- **WHEN** a link declares datacentre home and no other scope
- **THEN** it matches datacentre home with any cluster, namespace or environment
- **AND** it does not match datacentre office

#### Scenario: Namespace across clusters
- **WHEN** a link declares namespace media without cluster scope
- **THEN** it matches namespace media across cluster selections and does not match another selected namespace

#### Scenario: Shared differs from unknown
- **WHEN** an environment filter selects production and a manual entry omits environment while a discovered source has unknown environment
- **THEN** the manual entry satisfies that dimension and the discovered source does not
- **WHEN** the filter instead explicitly selects unclassified
- **THEN** both satisfy that dimension while retaining separate shared and unknown grouping identities

### Requirement: Shared grouping and honest presentation
Omitted manual dimensions SHALL group into typed Shared across clusters/namespaces/environments/datacentres buckets, including typed components in composite groups. Shared identities SHALL differ from unknown/unclassified identities and real values with identical text. Concrete groups SHALL sort before shared groups, followed by unknown groups, retaining existing concrete/unknown ordering. Namespace grouping SHALL distinguish cluster-qualified namespace, shared-cluster namespace and cluster-qualified shared-namespace paths. Each manual source SHALL appear once rather than being copied into every concrete group. Cards SHALL show the title, URL, group and supplied relevance metadata, omit empty scope rows and expose declaration provenance separately. The kind SHALL display as Manual link. Label/custom/kind/none grouping SHALL retain existing behaviour.

#### Scenario: Global documentation grouped by cluster
- **WHEN** an unscoped documentation link is grouped by cluster
- **THEN** it appears once under Shared across clusters with no invented cluster row
- **AND** selecting a particular concrete cluster group excludes that separate shared group even though the cluster scope filter alone permits it

#### Scenario: Composite shared path
- **WHEN** a production link omits cluster scope and grouping is Environment then Cluster
- **THEN** it appears under production / Shared across clusters

#### Scenario: Literal shared label collision
- **WHEN** a real environment is named Shared across environments and another entry omits environment
- **THEN** their groups, favourites and collapse records remain distinct

#### Scenario: Namespace grouping without both dimensions
- **WHEN** one link supplies only namespace media and another supplies only cluster homelab
- **THEN** namespace grouping uses distinct Shared across clusters / media and Homelab / Shared across namespaces paths
