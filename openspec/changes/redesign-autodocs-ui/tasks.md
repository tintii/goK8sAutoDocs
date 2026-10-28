## 1. Establish shared client foundations

- [x] 1.1 Inspect the current working tree, static server, directory/version handlers, browser fixture harness and container asset checks; preserve existing API and publication behavior.
- [x] 1.2 Extract shared local theme/preferences helpers and visual tokens into `/static/` assets, with explicit page initialization and no frontend framework or remote asset dependency.
- [x] 1.3 Implement and verify versioned preference parsing/validation, legacy theme import, system-theme fallback, supported storage events and session fallback with notices for malformed, blocked, quota-exceeded or future-version storage.
- [x] 1.4 Adapt the browser fixture harness to load shared scripts/styles and support a real same-origin persistence/reload fixture rather than relying solely on file-URL localStorage behavior.

## 2. Build the dashboard visual system

- [x] 2.1 Implement graphite/warm-neutral light/dark tokens, teal accents, local SVG controls, typography, focus styles and reduced-motion behavior with contrast-safe theme variants.
- [x] 2.2 Build shared desktop/mobile navigation and context headers across the existing landing, directory and versions routes; preserve the versions dashboard's current data and filtering functions.
- [x] 2.3 Redesign the directory header, search/toolbar, group headings and endpoint cards with responsive grid/list layouts, separate link/star actions and readable source metadata.
- [x] 2.4 Add distinct loading/error/empty/no-match/no-favourite states and retain collection warnings, unresolved-entry explanations and related-resource detail controls.

## 3. Implement dynamic grouping and personal group editing

- [x] 3.1 Implement typed source/group/link IDs and pure membership/order helpers; cover namespace/kind/name collisions, exact group case, literal/fallback Ungrouped and identical URLs from different sources.
- [x] 3.2 Add immediate Label, Namespace, Kind, Custom and None mode controls for Applications, preserving deterministic default ordering, one card per source and flat Discovered presentation without refetches.
- [x] 3.3 Add an accessible custom-group editor with stable IDs, validated create/rename, keyboard reordering and delete actions; keep empty custom groups editable without empty directory sections.
- [x] 3.4 Add source assign/move/unassign controls and a Custom fallback; verify rename preserves state and deletion cleans group-specific records while leaving labels and independent link stars intact.
- [x] 3.5 Persist grouping, density, ordering and per-group collapse choices; implement Reset layout with the specified favourite preservation and custom-group cleanup.

## 4. Implement group and link favourites

- [x] 4.1 Add labelled independent group/link star buttons with visible and accessible pressed state; deduplicate repeated source URLs for display while preserving attachment/issue context.
- [x] 4.2 Implement Favourites as live group membership plus individual source/URL matches, independent of the current display grouping and scoped to the active view.
- [x] 4.3 Render full cards for group matches and only starred URLs for link-only matches; retain unresolved group members and never construct a visible link from stored preferences alone.
- [x] 4.4 Persist stars, retain dormant identities across disappearance/reclassification, and implement Clear favourites without changing groups, layout or theme.
- [x] 4.5 Verify collisions, endpoint reorder, group rename, membership moves, missing URLs, return of dormant links and transitions between Applications/Discovered do not transfer or leak favourites.

## 5. Improve search and compose filters

- [x] 5.1 Build a reusable search index covering all existing searchable metadata plus personal group names; implement case-insensitive literal multi-term AND matching without regex or HTML interpretation.
- [x] 5.2 Add exact namespace/kind selectors and compose them with query and Favourites; retain query/filter intent across tab switches, including visible clearable selections with zero available matches.
- [x] 5.3 Add Ctrl/Cmd+K focus outside editable controls, Escape/clear-query actions and polite resource/displayed-URL result announcements; retain complete-view counters separately.
- [x] 5.4 Preserve URL tab navigation, keyboard tabs, safe link attributes, warning isolation and request-version guards through filtering, refresh and failed or out-of-order responses.

## 6. Verify behavior, presentation and packaging

- [x] 6.1 Extend browser fixtures for grouping/editing/stars, safe custom names, reset isolation and composition of search/filters/favourites; keep existing publication, diagnostics and race-transition regression coverage.
- [x] 6.2 Exercise persistence through actual same-origin reloads and page navigation; cover legacy theme migration, corrupt/future data, storage exceptions, quota failures and supported updates from another tab.
- [x] 6.3 Inspect and capture representative 375-pixel mobile and 1440-pixel desktop screenshots in both themes, covering all three pages, populated/empty directory, long names, unresolved entries and the group editor; verify keyboard/focus, contrast and reduced motion.
- [x] 6.4 Assess search/regroup/star responsiveness with a representative large fixture and verify preference changes cause no additional directory requests, remote asset requests or Kubernetes/API writes.
- [x] 6.5 Run `go test ./...` and `go test -race ./...` from `go/`; run `node --test static/autodocs-dashboard.test.cjs` with an available Chrome/Chromium executable and ensure browser checks execute without skips.
- [x] 6.6 Update asset checks if needed and run the existing root container build and `python3 scripts/smoke-container.py <test-image>` to confirm all three pages and their shared local assets ship in the non-root read-only image.

## 7. Document and hand over

- [x] 7.1 Update README usage and its embedded Unreleased changelog for the implemented UI, grouping modes, custom-group lifecycle, group-versus-link stars, search shortcuts and reset controls.
- [x] 7.2 Document browser/origin scope, clearing-data behavior, no device sync, same-source-tuple recreation behavior, legacy theme handling and the existing per-view cache delay.
- [x] 7.3 Run `openspec validate redesign-autodocs-ui --strict`, review the implementation diff and report actual validation outcomes; leave image publication and any GitOps digest update for separately authorized release work.
