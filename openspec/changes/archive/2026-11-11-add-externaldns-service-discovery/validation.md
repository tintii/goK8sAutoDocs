# Implementation validation

Validated locally on 2026-11-06. No release, publication, archive or application
deployment was performed. Authentication and collection checks used synthetic
Kubernetes APIs only.

- `go test ./...` and `go test -race ./...` from `go/`: passed.
- `node --test static/*.test.cjs` from `go/` with the installed Google Chrome:
  92 passed, zero failures or skips. Existing Ingress/HTTPRoute, classification,
  scope, personalisation, overview favourites and Versions checks remain green.
- Native `linux/arm64` image build through the installed Rancher Desktop Docker
  runtime: passed, tag `k8sautodocs:externaldns-local`. This follows the earlier
  local ARM validation setup; the release workflow still builds `linux/amd64`.
- `python3 scripts/smoke-container.py k8sautodocs:externaldns-local`: passed.
  The image serves the exact packaged assets, runs as non-root with a read-only
  filesystem, and authenticates to two synthetic TLS Kubernetes targets.
  Checks cover DNS-only and explicit HTTPS Services, advertised names versus
  targets, separate publication views, unique filtered Service rows, zero
  Kubernetes reads from inventory GETs, coordinated refresh and persisted
  Service timestamps after restart with failed API access.
- `openspec validate add-externaldns-service-discovery --strict`: passed.
- `git diff --check`: passed.
- Inspected the real 375px browser screenshot at
  `/tmp/autodocs-services-mobile.png`: Service metadata, DNS values/copy controls,
  actual conflicts and web links remain distinct without document overflow.

## Capability review

- **externaldns-service-discovery**: resolver fixtures cover exact publication
  across Service types; hostname-key eligibility; absent/empty/group-only and
  target-only declarations; both-prefix normalization and independent conflicts;
  partial-invalid lists, unsafe/wildcard/IDNA names and IPv4/IPv6 targets;
  browser-style shortened/integer/octal/hex IP spellings are rejected as DNS names;
  bounded target precedence without fallback on explicit errors; HTTP(S)-only
  web hints, port validation and DNS-only behavior. Inventory fixtures verify
  source isolation, old v1 caches pending Services, new-cache retention, failed
  lists/connections, independent successful deletion, and the previous reader's
  map-based envelope/named-partition projection pattern. Browser fixtures verify
  text rendering, exact safe links, native keyboard copy and honest failure.
- **published-endpoint-directory**: mixed Ingress/HTTPRoute/Service fixtures
  retain separate backend/source identities and equal names across clusters;
  source classification and filtered metadata share the Service collection;
  active HTTP handlers only read snapshots. The container fixture also covers
  Services with the optional HTTPRoute API absent. Existing Gateway resolution
  and per-view regression tests pass.
- **endpoint-directory-organization**: browser fixtures cover searchable Service
  type, advertised names, selected targets and externalName; Service kind filters;
  resource versus URL counts; group and personal membership; DNS-only group
  favourites and web-only individual stars; grid/list/mobile presentation;
  view-specific warnings/transitions; pending and restored stale Services in
  scoped freshness. Existing preference migration and overview tests pass.

## Archive order

The base directory specs still predate completed, unarchived changes. Archive
the earlier overlapping `redesign-autodocs-ui` and
`add-multicluster-discovery-snapshots` changes in their dependency order before
this change, so older deltas cannot overwrite the shared-snapshot and Service
requirements. `adopt-warm-launchpad-directory` also changes directory behavior
and should be reviewed in that earlier-change sequence. This implementation
does not archive any of them or change a release version.

## Release preparation — 2026-11-11

- Re-ran Go tests and race checks: passed. Re-ran the full Chrome suite:
  92 passed, zero failures or skips. The application sources are unchanged
  from the implementation validated above, including its native ARM container
  build and two-target runtime smoke checks.
- Synced and archived the completed UI, multicluster snapshot, launchpad and
  ExternalDNS Service changes in the dependency order above. All four changes
  passed strict validation before archiving; all 10 resulting main specs pass
  strict validation. Original archive metadata and completed tasks are retained.
- Prepared stable `v0.5.2` release notes covering both ExternalDNS Service
  discovery and the UI improvements from the prerelease series. Main specs
  describe the current Signal/Noto Sans interface, shared snapshots and
  cluster-aware identities. The unfinished Rancher discovery proposal is
  excluded from the release.
- README links and Git whitespace checks pass. Native `linux/amd64` build,
  runtime checks and image publication remain enforced by the existing GitHub
  Actions workflow for the pushed branch/tag.
