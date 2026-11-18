## 1. Declaration and normalization

- [x] 1.1 Add strict bounded version-1 document parsing with required fields, optional scope, unique IDs, exact URL handling and sanitized validation diagnostics.
- [x] 1.2 Introduce manual source type, provenance and stable source identity separately from editable relevance/display fields.
- [x] 1.3 Normalize explicit environment/datacentre values through shared classification rules and documented unconfigured defaults; resolve cluster names without inferring restrictions.
- [x] 1.4 Cover schema failures, URL boundaries, full paths/query/fragments, arbitrary scope combinations and unresolved cluster references with focused Go tests.

## 2. Collection and snapshots

- [x] 2.1 Collect exactly marked ConfigMaps per target through the existing coordinator and implement per-document last-good retention, deletion, marker removal and empty-list handling.
- [x] 2.2 Merge manual entries into published directory responses only; keep Discovered, Versions, workload status and Kubernetes filtered-resources membership unchanged.
- [x] 2.3 Persist normalized entries and document freshness/provenance with compatibility handling, preserving target identity/removal protections and excluding raw documents.
- [x] 2.4 Extend status dimensions, partial outcomes and provenance-based scope freshness, including shared links from another or never-successful source target.
- [x] 2.5 Test independent valid/invalid documents, failed lists, retained timestamps, successful removals, cache restore and snapshot-only API reads.

## 3. Directory and preferences

- [x] 3.1 Extend common scope matching for omitted manual dimensions, including namespace and synthetic unknown selections, without changing discovered-resource matching.
- [x] 3.2 Add typed shared grouping buckets and composite paths with deterministic order and collision-safe group identities.
- [x] 3.3 Render Manual link titles, exact URLs, supplied scope and separate provenance; include relevance/title/group in search and accurate counters/options.
- [x] 3.4 Adapt browser source/link identity handling for manual entries, preserving existing preference identities and exact-URL favourite semantics.
- [x] 3.5 Apply the same scope, identity and freshness rules to overview favourites; preserve view switching, group selection and late-response guards.

## 4. Validation and documentation

- [x] 4.1 Add browser coverage for unscoped/partial/full scope, namespace/kind/search/group/favourites intersections, unknown-versus-shared grouping, and URL-restored scope.
- [x] 4.2 Verify title/scope edits preserve assignments and independent favourites, URL edits do not transfer link stars, duplicate destinations remain independent, and provenance never supplies relevance.
- [x] 4.3 Verify cross-cluster declaration freshness, retained stale cards, empty states, keyboard access and responsive grid/list presentation alongside discovered-resource regressions.
- [x] 4.4 Document the schema, examples, normalization defaults, shared filtering/grouping, source identity, refresh/error/removal behaviour and ConfigMap permissions; coordinate location with the user-manual change and update release notes.
- [x] 4.5 Run the repository's Go/race and browser checks plus strict OpenSpec validation; record evidence before proposing a separate image publication or homelab rollout.
