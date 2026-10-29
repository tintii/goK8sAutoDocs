# Changelog

All notable changes to goK8sAutoDocs are recorded here, grouped by release.

## [Unreleased]

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

[Unreleased]: https://github.com/tintii/goK8sAutoDocs/compare/v0.5.1...HEAD
[v0.5.1]: https://github.com/tintii/goK8sAutoDocs/compare/v0.5.0...v0.5.1
[v0.5.0]: https://github.com/tintii/goK8sAutoDocs/tree/v0.5.0
