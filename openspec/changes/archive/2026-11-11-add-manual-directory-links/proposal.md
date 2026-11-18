## Why

AutoDocs cannot currently include arbitrary destinations such as GitLab, Confluence, Proxmox or external documentation in its application directory unless they are represented by a supported Kubernetes endpoint resource. These links need declarative ownership and the same search, grouping and favourites as discovered applications, with optional explicit filtering metadata and meaningful behaviour when they are shared across scopes.

## What Changes

- Discover explicitly labelled ConfigMaps containing a versioned `links.yaml` document, using the existing inventory refresh coordinator across configured targets. Use ConfigMaps initially; no CRD or operator is introduced.
- Define each link with stable `id`, `title`, HTTP(S) `url`, optional `group`, and optional `scope` containing singular `clusterId`, `namespace`, `environment` and `datacentre` values.
- Present links as Manual link entries in Applications, including search, grid/list layouts, personal groups, group/link favourites and overview favourites. Do not add them to Discovered, workload/version inventory or the Kubernetes filtered-resources API.
- Treat omitted scope dimensions as intentionally unrestricted for manual links. Fully unscoped links remain visible across scope selections; partially scoped links match every declared restriction. Existing discovered-resource unknown/unclassified semantics remain unchanged.
- Group omitted dimensions into typed Shared across clusters/environments/datacentres/namespaces sections, distinct from unknown classifications and literal values with the same names. Display only supplied relevance metadata, with declaration provenance separately available.
- Keep source identity and collection provenance independent of declared scope. Preserve personal assignments and independent favourites when title, group or scope changes; exact-URL changes continue to have distinct link identities.
- Include manual-link refresh outcomes in snapshots, diagnostics and scope-aware freshness, including when declarations are stored in a different cluster from their declared relevance.

## Capabilities

### New Capabilities

- `manual-directory-links`: ConfigMap discovery, declaration schema, validation, normalization, stable provenance, failure handling and optional-scope semantics for manually authored links.

### Modified Capabilities

- `endpoint-directory-organization`: Published manual-link entries, display/search metadata and dimension-aware namespace filtering.
- `multicluster-inventory-experience`: Optional relevance scope, shared grouping, manual identities and provenance-aware freshness.
- `browser-directory-personalization`: Manual-link source identities independent of editable relevance metadata.
- `shared-inventory-snapshots`: Manual-link collection partitions, per-document validation retention and derived-cache contents.

## Impact

Application changes belong in `goK8sAutoDocs`: collection/validation, directory and status projections, snapshot compatibility, shared browser scope/grouping, directory cards and overview favourites. Existing Kubernetes resources and browser preferences retain their identities and publication rules. Manual environment/datacentre values reuse classification normalization without acquiring labels from the storage namespace or cluster.

ConfigMap listing is already part of the documented discovery RBAC. No new API kind, write permission, watch or runtime service is required. Existing Helm ConfigMap support can supply declarations; an actual homelab values change and rollout are separate follow-up work. Documentation and examples must coordinate with the active user-manual proposal.

This change contains planning artifacts only. Implementation, publication, deployment, commits and pushes are not part of writing this proposal. Multi-value scope, browser authoring, destination health checks, custom URL schemes and CRD support are out of scope for this first version.
