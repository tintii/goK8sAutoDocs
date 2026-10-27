## 1. Classification and collection

- [x] 1.1 Define presence-aware publication classification and a source-only group-label contract; cover exact true, false, absent, empty, other values, group-only resources, and ignored legacy annotations.
- [x] 1.2 Extend the Auto-Docs collector with published and discovered selection for Ingresses and HTTPRoutes, using view-specific list selectors and defensive checks while reusing existing URL resolution, related resources, and diagnostics.
- [x] 1.3 Verify mixed source kinds, cross-namespace dependencies, unresolved entries, absent Gateway API, and partial permission failures for Discovered; ensure hidden resources are not resolved and filtered-resources stays published-only.

## 2. API views and caches

- [x] 2.1 Parse and normalize the optional Auto-Docs API view; retain the existing default and array schema, support discovered selection, and reject unknown/repeated view parameters before collection or cache lookup.
- [x] 2.2 Partition the Auto-Docs payload, warnings, expiry, and concurrent access into bounded published/discovered caches, retaining the existing TTL and filtered-resources isolation.
- [x] 2.3 Test both request orders, concurrent view requests, default/empty/explicit published compatibility, invalid requests, per-view warnings, and label transitions after cache expiry.

## 3. Applications and Discovered UI

- [x] 3.1 Add accessible Applications and Discovered tabs with URL selection, reload/direct-link restoration, browser history handling, and keyboard navigation.
- [x] 3.2 Render Applications by exact group value with Ungrouped last, keeping the fallback bucket distinct from real values; order cards by namespace/kind/name and show namespace/kind on every source card.
- [x] 3.3 Render Discovered as a flat ordered card list with existing resolved links, unresolved issues, related resources, and collection warnings.
- [x] 3.4 Make search, counters, and empty states view-aware; search group values, preserve the query on tab changes, and omit empty sections.
- [x] 3.5 Handle view-specific loading/errors and prevent late responses from another selection replacing the active view.

## 4. Verification and documentation

- [x] 4.1 Extend dashboard fixtures for cross-namespace groups, missing/empty groups, case-distinct groups, fallback-name collisions, group-only discovery, multiple URLs per source, and text-safe group rendering.
- [x] 4.2 Test tab links/history/keyboard behavior, active-view search and counts, empty results, out-of-order responses, and failed view switches without stale cards or warnings.
- [x] 4.3 Run Go tests, the cache race checks, the existing JavaScript dashboard test command, and the build; inspect representative grouped, ungrouped, discovered, and unresolved fixture views without changing a live cluster.
- [x] 4.4 Update the README and its changelog with label examples, the publication-state table, discovered API selection, read-only GitOps usage, default API compatibility, and per-view refresh latency.
- [x] 4.5 Record prerequisite-first archive order and validate both changes and their modified requirement baseline; leave actual archival to the archive workflow.

Archive order: archive the completed `add-gateway-endpoint-discovery` change before this change so the canonical `published-endpoint-directory` requirements exist for these deltas. Applying this change does not archive either change.

Verification: `go test ./...`, `go test -race ./...`, all nine headless-browser dashboard fixtures, `go build`, `openspec validate --all --strict`, and `git diff --check` passed. Modified requirement names match the prerequisite baseline. No live-cluster changes were made.
