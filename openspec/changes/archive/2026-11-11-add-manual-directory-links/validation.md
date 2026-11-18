# Implementation and validation

Implemented in `goK8sAutoDocs` on 2026-11-11. This record covers local source
implementation; image publication and a homelab rollout are separate work.

## Scope

- Strict bounded YAML parsing, exact HTTP(S) URLs, optional independent relevance,
  declaration provenance and stable source identities are in `go/manual_links.go`.
- The existing coordinator collects a separate `manual-links` component with
  last-good data per ConfigMap. Invalid edits retain only their own prior results;
  successful removals and empty lists withdraw entries. Failed lists retain the
  component. Per-document success times and the last fully valid component time
  survive failures.
- Applications alone publishes normalized manual rows. Discovered, Versions,
  workload status and Kubernetes filtered resources retain their membership.
- Snapshot restoration follows declaration target identity/removal rules, rebuilds
  manual projections from derived document data and marks restored data stale.
  Older caches without this component leave it never collected.
- Shared scope/grouping and browser identities preserve existing discovered
  identities. Overview uses the same manual matching, favourites and retained
  declaration notices. Scope freshness uses each contributing document's source,
  including cross-cluster declarations and never-successful collectors.

## Checks

- `cd go && go test ./...`: passed.
- `cd go && go test -race ./...`: passed.
- `node --test go/static/manual-links.test.cjs`: 10 passed, no skips.
- `node --test go/static/*.test.cjs`: 104 passed, 0 failed, 0 skipped (68.3 seconds).
- `openspec validate add-manual-directory-links --strict`: passed.
- `git diff --check`: passed.

Go fixtures cover schema failures and exact bounds (256 KiB / 500 entries), URL
boundaries and exact destinations, all 16 relevance combinations, unresolved
clusters, shared classification spelling, per-document mixed outcomes, failed
lists, removal/marker changes/empty lists, snapshot-only reads, raw-document
exclusion, target identity changes and legacy-cache restore.

Real Chrome fixtures cover scope/filter intersections, shared/unknown/literal
identity separation, composite and namespace groups, preference persistence,
exact URL stars and duplicate destinations, provenance isolation, cross-cluster
and partially invalid document freshness, Overview, keyboard controls, and
375px/1440px grid/list layouts. The existing browser suite supplies discovered
resource, navigation, late-response, Service, theme and Versions regressions.
An initial new freshness assertion hit the nine-to-ten-minute display boundary;
the assertion now permits either adjacent minute without weakening stale-state
or provenance checks.

## Documentation handoff

At implementation time, `publish-user-manual-on-github-pages` was still in planning.
The complete feature guide lives at `docs/manual-directory-links.md`, linked
from README, so it can be linked from the planned endpoint-publication page and
added to manual navigation without moving or duplicating its content. README and
Unreleased release notes include the implemented capability. Existing unrelated
README/CHANGELOG edits and archival work were preserved.

## Operational boundary

No live resources were created or changed. No image was published, no commit or
push was made, and no runtime deployment was made. The example ConfigMap is
documentation only. Publication and rollout need their own explicit request;
existing Helm ConfigMap support and discovery ConfigMap list permissions suffice.

## Archive

On 2026-11-11, after approval to sync and archive, synced five added requirements
into the new `manual-directory-links` main spec and twelve modified requirements
across the browser-personalization, endpoint-directory, multicluster-experience
and snapshot specs. Existing scenarios and unrelated requirements were preserved.
All 19 tasks and all planning artifacts were complete before archive.

Archived to `openspec/changes/archive/2026-11-11-add-manual-directory-links/`,
including `.openspec.yaml`, source deltas and this validation record. Updated
README/specification links, Unreleased notes and the user-manual implementation
record's design link. No commit, push, image publication or deployment was made.


Archive verification passed: `openspec validate --all --strict --no-interactive`
validated all 17 active change/spec items; `git diff --check` passed; and
`.venv-docs/bin/python scripts/check-docs.py` passed the strict documentation
build, broken-link failure fixtures and generated page/fragment/asset checks.
`openspec list --json` confirms this change is no longer active.
