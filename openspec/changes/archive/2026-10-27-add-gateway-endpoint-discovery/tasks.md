## 1. Publication contract and endpoint model

- [x] 1.1 Replace legacy Auto-Docs annotation selection with the exact `k8sautodocs.io/publish=true` label selector in Auto-Docs and filtered-resources; remove FQDN override handling and prevent duplicate source entries.
- [x] 1.2 Define shared directory entries with source identity, existing resource context, resolved endpoint URLs, and resolution issues; retain the Auto-Docs array response and represent empty collections as arrays.
- [x] 1.3 Add Ingress URL resolution using concrete rule hosts and matching TLS declarations, with no configurable hostname override.

## 2. Gateway API discovery

- [x] 2.1 Discover labelled Gateway API v1 HTTPRoutes using the existing dynamic client and include them in the relevant filtered-resource results.
- [x] 2.2 Resolve Gateway parent defaults and cross-namespace references, restrict listeners by section name and port, and reuse parent lookups within a collection pass.
- [x] 2.3 Resolve concrete hostname intersections and listener schemes/ports into deduplicated URLs; report wildcard-only, incompatible, missing-parent, and current rejected-attachment cases.
- [x] 2.4 Associate parent Gateways and supported Service backends with HTTPRoute entries, respecting backend namespaces and preserving entries when related resources cannot be resolved.
- [x] 2.5 Keep Gateway API optional and communicate collection/permission failures while retaining successfully collected entries; keep Kubernetes access read-only.

## 3. API caching and dashboard

- [x] 3.1 Isolate Auto-Docs and filtered-resources caches, retain the existing refresh interval, and make cache access safe under concurrent requests.
- [x] 3.2 Render server-resolved URLs with `target="_blank"` and `rel="noopener noreferrer"`; update cards, search, counters, and text to cover both resource kinds.
- [x] 3.3 Display unresolved entries, resolution issues, incomplete collection diagnostics, and empty results without rendering failures.

## 4. Verification and documentation

- [x] 4.1 Add focused tests for true/missing/false/other publication labels, ignored legacy annotations, removed overrides, and duplicate prevention.
- [x] 4.2 Test Ingress HTTP/TLS links and HTTPRoute parent defaults, cross-namespace parents, listener restrictions, hostname intersections, non-default ports, duplicate URLs, unavailable parents, and attachment status.
- [x] 4.3 Test mixed-resource discovery, absent Gateway API, forbidden discovery, API cache separation, and metadata refresh after expiry using fake clients or fixtures without changing a live cluster.
- [x] 4.4 Run the Go tests, race detector for cache coverage, and build; verify dashboard rendering with mixed, unresolved, and empty fixture data.
- [x] 4.5 Update the README and its changelog with Gateway API support, publication label examples, removal of both legacy annotations, Kubernetes permissions, and GitOps migration guidance.
