## 1. Service parsing and endpoint model

- [x] 1.1 Add Service-only type, declared externalName and DNS target/source metadata to the directory model; preserve existing response fields and cluster-aware source/link identity.
- [x] 1.2 Implement exact publication eligibility for all Service types, including hostname-key discovery, explicit non-true exclusion and unresolved labelled Services without hostnames.
- [x] 1.3 Implement hostname/target parsing for both ExternalDNS prefixes with normalization, stable deduplication, DNS/IP validation, partial-invalid-list issues and independent conflict handling.
- [x] 1.4 Implement the bounded target precedence: explicit annotation, applicable externalIPs, ExternalName externalName, then applicable LoadBalancer status; retain the declared externalName and avoid DNS/provider/Pod/Node queries for standalone resolution.
- [x] 1.5 Implement optional Service scheme/port annotations with HTTP(S)-only links, default-port omission, explicit invalid-input diagnostics and intentional DNS-only behavior. Keep Service port inference and fqdn-override disabled.
- [x] 1.6 Add focused resolver tests covering all publication values and Service types, empty/target-only/group-only metadata, multiple/duplicate names, both-prefix equivalence/conflicts, unsafe/wildcard names, IPv4/IPv6 targets, target precedence, missing ports and web-hint failures.

## 2. Shared inventory and API integration

- [x] 2.1 Add a Services partition to the active per-cluster collector using one standalone Service list per target run; apply existing classification and keep selectorless Services independent of related-resource lookups.
- [x] 2.2 Project Services into both directory views and derive published Service metadata for filtered-resources from that same collection. Skip the generic Services metadata list and preserve the existing whole-filtered-partition failure policy.
- [x] 2.3 Include Services in target connection-failure results, discovery-status metadata, directory data/warnings and refresh logging; preserve previous rows/timestamps on failures and remove rows after successful empty collection.
- [x] 2.4 Preserve version-1 snapshot compatibility: old caches without Services remain usable with a never-collected Services state; new caches retain Service rows/timestamps across restart. Verify older readers ignore the additive partition without invalidating existing data.
- [x] 2.5 Check retained legacy collector/helper paths and their callers; share Service resolution where those paths are still supported, while keeping active inventory GET handlers free of Kubernetes reads.
- [x] 2.6 Add integration fixtures for same-named sources across clusters, a Service that is also an Ingress/HTTPRoute backend, publication/hostname removal, independent list failures, target connection failure, deletion, filtered-row deduplication and old/new snapshot restoration.

## 3. Directory presentation and freshness

- [x] 3.1 Render Service type, advertised names, declared externalName and selected target/source distinctly in existing grid/list details. Keep actual declaration issues visible and distinguish intentional DNS-only content from unresolved web entries.
- [x] 3.2 Add keyboard-accessible DNS copy controls with exact-value copy and honest clipboard success/failure feedback. Use text rendering for metadata and retain safe server-resolved HTTP(S) links and existing link attributes.
- [x] 3.3 Extend search and kind filters to Service type, advertised names, selected targets and externalName; preserve scope, personal groups, group favourites, web-only individual link favourites and source identity. Count DNS-only sources as resources with zero web links.
- [x] 3.4 Include the Services partition in frontend freshness and incomplete-discovery calculations, including pending first collection and stale restored data; preserve the current coordinator refresh controls and per-cluster scope.
- [x] 3.5 Add browser fixtures/tests for DNS-only and web Services, multiple hostnames, prefix conflicts, unsafe metadata, keyboard copy/failure, target search, grouping/favourites, grid/list layouts, view transitions and Services freshness. Verify legacy overview favourites remain compatible.

## 4. Documentation and validation

- [x] 4.1 Document the publication matrix, both supported prefixes, conflict policy and new Service web annotations in README/CHANGELOG. Include ExternalName examples for a DNS-only entry, HTTPS without Service ports, non-default port and explicit IP target override.
- [x] 4.2 Document ExternalDNS ownership and deployment prerequisites, the declared-versus-live distinction, DNS/TLS alias limitations, bounded target interpretation, read-only Service RBAC, additive API kind/metadata and snapshot upgrade/rollback behavior.
- [x] 4.3 Run Go tests and race checks from `go/`, then the repository browser suite using its documented Chrome setup; resolve failures and confirm existing Ingress/HTTPRoute, classification, personalisation and Versions behavior.
- [x] 4.4 Build the local application container and run `scripts/smoke-container.py` using the repository's established platform/runtime setup; verify packaged assets and API behavior without deploying or publishing.
- [x] 4.5 Run strict OpenSpec validation and review implemented behavior against all three capability specs. Record verification results and retain the archive-order note for earlier overlapping completed changes; do not archive or release as part of implementation validation.
