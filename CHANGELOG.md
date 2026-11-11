# Changelog

All notable changes to goK8sAutoDocs are recorded here, grouped by release.

## [Unreleased]

## [v0.6.0-RC2] - 2026-11-11

### Added

- Task-oriented user manual with source installation, configuration, cluster discovery, dashboard use, endpoint publication, operations, troubleshooting and API reference.
- Pinned MkDocs tooling, strict generated-site link checks and an independent GitHub Actions workflow for validated GitHub Pages publication.

### Changed

- Shorten the README to a verified quick start and manual links; preserve implementation context and documentation maintenance records in OpenSpec.

## [v0.6.0-RC1] - 2026-11-11

### Added

- Optional Rancher RK-API sources with mounted bearer kubeconfigs, complete bounded pagination, exclusions, credential rotation and downstream proxy collection, verified against Rancher 2.15.2.
- Canonical cluster bindings with one explicit primary, identity-probe overlap/conflict diagnostics and primary-only collection without failover or inventory merging.
- Atomic dynamic membership reconciliation, isolated source failures, compatible stale recovery and version-2 membership/provenance persistence while retaining direct version-1 support.
- Shared source/connection diagnostics and browser coverage preserving canonical resource identities and favourites across primary changes. Documented RBAC prerequisites, directory Secret mounts, homelab binding, discovery budgets and rollback behavior.

### Changed

- Sync and archive the completed Rancher discovery, canonical connection binding and dynamic membership specifications.

## [v0.5.2] - 2026-11-11

### Added

- Standalone core/v1 Service directory sources, including selectorless ExternalName Services, with exact publication labels and ExternalDNS hostname-key discovery. Ordinary unlabelled, group-only and target-only Services stay outside Discovered.
- Support for `hostname` and `target` under both `external-dns.alpha.kubernetes.io/` and `external-dns.kubernetes.io/`, normalized DNS/IP lists, partial-invalid diagnostics and independent prefix-conflict handling without silent precedence or fallback.
- Copyable DNS-only entries, distinct Service type/advertised names/declared externalName/selected targets, safe annotation inspection and DNS metadata search. Explicit AutoDocs `k8sautodocs.io/scheme` and optional `k8sautodocs.io/port` annotations opt into HTTP(S) links using advertised names; Service ports never infer web intent.
- Independent per-cluster Services collection/status/freshness with failure retention, deletion, restart recovery and additive version-1 snapshot compatibility. Published filtered Service metadata comes from the same collection without duplicate generic lists.
- Resolver, shared-inventory, browser and runtime fixtures covering DNS/web Service declarations, publication transitions, conflicts, keyboard copy/failure, identity, preferences and old/new snapshots. Documented ExternalName examples, read-only RBAC, ExternalDNS prerequisites and DNS/TLS limitations.

### Changed

- UI improvements bring the application directory to the default page, with compact application tiles, aligned metadata, locally bundled Noto Sans, Signal light/dark themes, green accents and coloured application initials. The desktop sidebar, keyboard-accessible mobile drawer and live group navigation preserve search, scope, favourites and personalisation.
- Refine control spacing, hairline borders, charcoal dark surfaces, plain Overview styling and Versions totals while preserving grouping, refresh, table resizing/sorting and saved preferences. This stable release includes the UI improvements validated through the v0.5.2 prerelease series.
- Sync and archive the completed directory UI, multicluster snapshot, launchpad and ExternalDNS Service changes in dependency order, preserving Service discovery and current interface requirements in the main specifications.

## [v0.5.2-RC6] - 2026-11-04

### Changed

- Apply the approved local UI polish: hairline borders, compact navigation and control spacing, charcoal dark surfaces, restrained group markers and direct Versions/personalisation copy. Preserve green accents, Noto Sans, plain metadata and name-coloured tile initials.
- Replace the Overview background grid with a plain surface and simplify the versions totals to numbers and labels. Preserve directory, discovery, personalisation, scope, refresh and table interactions. Measure table fonts from explicit font properties so tabular numerals do not clip metadata labels.

## [v0.5.2-RC5] - 2026-11-03

### Changed

- Organise directory tile metadata into aligned, labeled namespace, cluster, environment, datacentre and group rows. Remove the ingress-class detail.

## [v0.5.2-RC4] - 2026-11-03

### Changed

- Use locally bundled Noto Sans, medium-weight headings, larger application names and sans-serif supporting labels while preserving the green Signal palette.
- Replace outlined metadata tags with plain labels; keep image names and versions in monospace. Measure version-table columns after the bundled font loads.

## [v0.5.2-RC3] - 2026-11-02

### Changed

- Replace the warm paper/olive themes with Signal light and dark themes: near-black or off-white surfaces, a sage-green accent, square geometry, 2px rules, monospace labels, underlined inline links and a 12px minimum text size. The Overview hero uses a grid pattern instead of the orbit artwork.

### Added

- Per-group colours from an 8-hue palette, assigned by position in the unfiltered group list so they hold while searching or filtering. They mark card edges, group headings and the sidebar group list; fallback groups stay neutral.
- Name-derived colours for card initials tiles: an FNV-1a hash of the resource name selects an OKLCH hue, so each application keeps the same colour across pages, themes and browsers.

## [v0.5.2-RC2] - 2026-11-02

### Changed

- Place sidebar Scope & updates controls above Browse groups, with matching visual and keyboard order on desktop and mobile.

## [v0.5.2-RC1] - 2026-11-02

### Changed

- Open the application directory directly at `/`, preserving `/autodocs-dashboard-page`, view/scope URLs and Versions. Keep configured home content and `/api/v1/home` on the optional `/overview-page`, linked from secondary navigation; unknown paths return 404.
- Replace the directory hero and large totals with compact tiles, search, visible filter/result summaries and warm paper/olive light and dark themes. Preserve resolved endpoint URLs, independent stars, unresolved sources, personal groups, grid/list layouts, diagnostics and refresh controls.
- Derive sidebar groups and matching counts from the active directory model across all grouping modes. Group selection combines with existing filters, retains zero-match choices and resets on view/grouping changes or removal during refresh.
- Keep a browser-saved desktop rail and add a separate mobile modal drawer with keyboard focus management, Escape/backdrop dismissal and breakpoint cleanup. Long group lists scroll independently of fixed navigation.

### Added

- Complete remote-cluster kubeconfig Secret example with embedded certificate authentication, bearer-token alternative, encoding guidance, Secret mount and matching explicit registry context.

## [v0.5.1] - 2026-10-29

### Added

- Structured JSON application logs on stderr using Go's standard `log/slog`, with debug, info, warn and error thresholds selected through `LOG_LEVEL`.
- Startup, refresh start/completion, collection failure/recovery and snapshot restore/persistence events with cluster/component context, generation, startup/automatic/manual trigger, duration and aggregate counts.
- Debug diagnostics for coordinated refresh decisions, optional Gateway API absence and successful collection/persistence; bounded warnings for retained stale data and partial metadata or endpoint resolution.
- Safe internal-handler diagnostics for homepage configuration and response encoding failures.
- Fixture tests for severity filtering, early startup failure, concurrent refresh decisions, quiet readers, recovery and credential/object non-disclosure; container smoke assertions for JSON stderr logs and refresh context.

### Changed

- Default application output is structured JSON at info level. `DEBUG=true` remains a fallback when `LOG_LEVEL` is unset or blank; explicit levels take precedence. Invalid levels fail startup before collection or listener initialization.
- Replace unlevelled print/fatal messages and raw Pod/container/related-resource/image dumps with owned messages, sanitized reasons and bounded summaries. Successful health/status/static/snapshot requests remain silent, including at debug level.
- Consolidate the accumulated release history into this changelog and update README configuration, troubleshooting, connection scope, project layout and release-pipeline documentation.

### Fixed

- JSON response encoding failures now return HTTP 500 before committing a successful response, with one safe error diagnostic.

## [v0.5.0] - 2026-10-29

### Added

- Explicit multi-cluster registry with stable IDs, independent in-cluster or mounted self-contained kubeconfig targets, credential rotation and read-only discovery.
- Configurable environment/datacentre label classification, ordered source precedence, case equality/display rules, missing-value groups and cross-page URL scope.
- Shared automatic/manual discovery with global cooldown, progress and per-component failure isolation; persistent startup snapshots with original freshness timestamps and stale-data retention.
- Cluster-aware browser identities and version-2 preference migration through an explicit original-cluster mapping.
- Gateway API v1 HTTPRoute discovery alongside Ingresses, listener-derived HTTP/HTTPS URLs and related Gateways, Services and Pods; unresolved entries and partial collection diagnostics.
- Applications groups through `k8sautodocs.io/group`, an Ungrouped fallback and a Discovered tab for sources without a publication label; bookmarked view selection and independent filtering.
- Browser-only grouping by source label, namespace, kind, personal groups, cluster, environment, datacentre and ordered composite paths; editable membership/order, grid/list layouts and collapsible groups.
- Independent group/link favourites, view-scoped filtering, separate layout/star resets and resilient versioned browser persistence with cross-tab updates.
- Multi-term directory search, namespace/kind selectors, personal-group search and Ctrl/Cmd+K navigation.
- Content-aware, resizable versions-table columns with keyboard controls, two-level sorting and saved browser preferences.
- Configurable homepage heading, description, environment, MOTD, useful links and footer through live directory-mounted `HOMEPAGE_CONFIG` YAML; homepage favourites shared with both directory views.
- Compact application-availability indicator beside the theme toggle, independent of cluster/workload health.
- Saved collapsible sidebar with desktop icon rail, mobile disclosure, a Scope & updates submenu on Directory and Versions, and a GitHub project footer link.
- Digest-pinned, non-root multi-stage container build supporting a read-only runtime with locally packaged assets and CA certificates.
- GitHub Actions Go/race/browser/container validation and GHCR publication of the exact tested linux/amd64 image with full-commit/release tags, OCI metadata and a reported digest; no moving latest tag.
- Kubernetes, browser and two-target TLS container fixtures covering persistence, authentication, coalescing, responsive layouts and preferences; OpenSpec requirements and archived endpoint-discovery changes.

### Changed

- Inventory GETs serve no-store snapshot projections without initiating Kubernetes collection. No usable initial snapshot returns HTTP 503; later collection failures retain previous successful partitions and diagnostics.
- Cluster information comes from the registry; Applications and Discovered classify the same collected source partitions.
- Landing, directory and versions pages share a responsive visual shell, local assets and persistent light/dark themes with legacy preference import.
- Overview presents the MOTD in its hero with the orbit graphic and directory favourites; inventory scope, refresh timing and detailed collection status live in sidebar controls.
- Directory counts distinguish matching sources, displayed URLs and complete-view totals; cards use server-resolved destinations, show namespace/source kind and open links in new tabs.
- **Breaking:** publication requires `metadata.labels.k8sautodocs.io/publish: "true"`. Legacy publish annotations no longer opt resources in; annotation-only sources qualify for Discovered when the publication label is absent. Any present non-true label hides the source from both views.

### Removed

- `k8sautodocs.io/fqdn-override`; destinations derive exclusively from networking resources.
- Homepage refresh button, overview inventory panel and detailed workload-status presentation; background discovery and the workload-status API remain available.

### Fixed

- Versions discovery across all namespaces, empty inventories and safe text rendering of metadata; failed HTTP responses show explicit loading errors.
- Discovered grouping, personal assignments and favourites retain preferences across tabs and reloads.
- In-cluster client initialization, pending Pod relationships and empty Service selectors.
- Shared legacy response-cache concurrency and isolated payloads; active runtime collection is superseded by the shared snapshot coordinator.
- Chrome fixture startup/readiness and owned-process cleanup races on Linux CI.

## [v0.1.0]

### Added

- Initial Kubernetes resource monitoring and auto-documentation dashboards.
- Deployment, StatefulSet and DaemonSet version inventory, container image/tag/digest extraction, GVR filtering and related Pod/Service/Ingress resources.
- In-cluster and out-of-cluster connections, REST APIs, static dashboard pages, configurable caching, debug output, CORS and graceful error handling.

[Unreleased]: https://github.com/tintii/goK8sAutoDocs/compare/v0.5.2...HEAD
[v0.5.2]: https://github.com/tintii/goK8sAutoDocs/compare/v0.5.1...v0.5.2
[v0.5.2-RC6]: https://github.com/tintii/goK8sAutoDocs/compare/v0.5.2-RC5...v0.5.2-RC6
[v0.5.2-RC5]: https://github.com/tintii/goK8sAutoDocs/compare/v0.5.2-RC4...v0.5.2-RC5
[v0.5.2-RC4]: https://github.com/tintii/goK8sAutoDocs/compare/v0.5.2-RC3...v0.5.2-RC4
[v0.5.2-RC3]: https://github.com/tintii/goK8sAutoDocs/compare/v0.5.2-RC2...v0.5.2-RC3
[v0.5.2-RC2]: https://github.com/tintii/goK8sAutoDocs/compare/v0.5.2-RC1...v0.5.2-RC2
[v0.5.2-RC1]: https://github.com/tintii/goK8sAutoDocs/compare/v0.5.1...v0.5.2-RC1
[v0.5.1]: https://github.com/tintii/goK8sAutoDocs/compare/v0.5.0...v0.5.1
[v0.5.0]: https://github.com/tintii/goK8sAutoDocs/tree/v0.5.0
