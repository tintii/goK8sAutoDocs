## Context

The Go service serves three static HTML pages, with the directory's behavior in `go/static/autodocs-dashboard.js`. The directory already has case-insensitive substring search, source-label groups, Applications/Discovered tabs, theme persistence and protection against out-of-order requests. The canonical `endpoint-directory-organization` and `published-endpoint-directory` specs describe important compatibility guarantees.

Directory responses already contain source identity, labels, endpoints, issues and related resources. Browser grouping and favourites can therefore be implemented entirely in the static client. The existing browser tests inject fixture responses into HTML files; shared asset extraction will require adapting that harness. The image smoke script checks page assets against repository content.

## Goals / Non-Goals

**Goals:**

- Make AutoDocs a visually distinctive, usable daily directory with a consistent landing page and versions dashboard.
- Support immediate grouping changes, editable personal groups, group/link stars and composable search/filters.
- Restore personal settings across reloads on the same origin, without server-side state.
- Preserve publication boundaries, resolved destinations, source context, diagnostics, caches and bookmarkable tabs.
- Keep implementation compatible with the existing Go/static/container workflow.

**Non-Goals:**

- Kubernetes metadata editing, arbitrary user-created URLs, custom endpoint resolution, accounts or cross-device preference sync.
- A frontend framework migration, a new backend/API, a versions data-model redesign, or image publication/deployment as part of planning.
- Inferring live application health from a resolved URL or cluster display name.

## Decisions

### 1. A shared visual system, with the directory as the centrepiece

Use a compact application shell: a branded navigation rail on desktop, a top navigation on narrow screens, a cluster/context header and a wide content canvas. Keep all existing page routes. The landing page becomes a designed overview with strong directory/versions navigation; the versions page gains the same shell and a readable table while preserving existing functionality.

The direction is a calm infrastructure console: graphite `#101719` dark surfaces, warm `#f4f5f1` light canvas, teal `#35c9b0` accents and amber stars. Define theme-specific contrast-safe text and interactive colors rather than using the same accent for every role. Use system sans-serif typography and monospace for namespaces, hosts and image identifiers, fine borders, generous spacing and small local SVG icons. A restrained header gradient and short entrance/hover transitions provide polish without competing with endpoint information. Respect reduced motion. No remote fonts, images or icon requests are required.

Directory hierarchy: page title and real loaded totals; prominent search; Applications/Discovered tabs; grouping/filter controls; group sections with counts, collapse controls and independent star buttons; responsive resource cards with clearly separated link rows and star controls. Diagnostics stay visible outside collapsed details. Mobile controls wrap or use an accessible settings panel. Grid/list density is a saved presentation choice. A group editor provides labelled fields and keyboard-operable move-up/down controls; drag-and-drop is optional and never required.

Alternative: restyle only the directory. A small shared stylesheet and shell script keep all three entry points coherent with less duplicated theme code. A SPA/framework rewrite adds a build system without improving the required data flow.

### 2. Explicit view state and pure directory derivation

Keep request/view state separate from preferences. Derive presentation through:

`active response -> source index -> live group memberships -> search + namespace/kind + favourites predicates -> ordering -> sections/cards`.

Compute the complete-view totals before filters and announce matched resource/link totals separately. Build search text and membership maps once per data/preference update; ordinary keystrokes filter that index. Debounce briefly without delaying star or grouping changes. Preserve the existing request-version guard, clearing of old cards/warnings, URL-selected view and accessible tabs. Show refresh controls honestly: a request can still receive the server's existing 60-second cached response.

Use plain JavaScript helpers separated by responsibility, for example shared theme/preferences handling, directory model derivation and DOM rendering, all served under `/static/`. Retain text-only metadata rendering and exact server-resolved HTTP(S) destinations with the existing safe new-tab attributes. Never build markup by interpolating resource metadata or personal group names.

Alternative: derive everything while creating DOM nodes. A separate model makes membership, identity and filtering verifiable without duplicating DOM logic, and avoids repeated scans per card.

### 3. Grouping is a personal projection over live source data

Applications offers Label (default), Namespace, Kind, Custom and None. Label mode retains exact source group spelling/case, cross-namespace membership, a distinct fallback, existing ordering and one source card containing all resolved URLs. Namespace and Kind derive sections from current payloads. Custom mode uses user-created groups and one optional assignment per source; unassigned sources appear in its distinct fallback. None is a flat view. Discovered remains flat; custom assignments never publish a discovered or explicitly hidden source.

Custom group names are trimmed, non-empty and unique by exact value. Groups receive opaque stable IDs, so renaming preserves membership, ordering, collapse state and stars. Users can create/rename/delete/reorder groups and assign/reassign/remove a source through controls. Deleting a group unassigns its sources and removes its group star/collapse/order records; independent link stars survive. Empty custom groups remain available in the editor but are omitted from directory results. Reordering applies to custom groups; derived groups retain deterministic ordering. Unassigned/fallback sections sort last.

Settings apply immediately to loaded data without a network request. Label metadata remains searchable and visible even when another presentation mode is selected. Reset layout restores default grouping, custom assignments, ordering, collapsed state and density; it preserves independent link stars and derived-group stars but necessarily removes stars belonging to deleted custom groups. Clearing favourites leaves the layout intact.

Alternative: write group edits to Kubernetes. Browser projections satisfy on-the-fly personalization while respecting the service's read-only permissions and GitOps ownership.

### 4. Stable typed identities define favourite behavior

Encode IDs as serialized tuples, never delimiter-joined display text:

| Entity | Identity |
| --- | --- |
| Source | `[kind, namespace, name]` using the API's source kind |
| Link | `["link", sourceTuple, exactResolvedURL]` |
| Label group | `["group", "label", exactLabelValue]` |
| Label fallback | `["group", "label-fallback"]` |
| Namespace / Kind group | `["group", mode, exactValue]` |
| Custom group / fallback | `["group", "custom", opaqueID]` / `["group", "custom-fallback"]` |

Deduplicate identical URLs within a source for display/starring while preserving available diagnostic/attachment context. Equal URLs from different sources remain independent. Custom group IDs survive rename; source IDs survive source ordering, endpoint reordering and label changes. The API has no source UID: deletion/recreation with the same tuple inherits personal settings, a documented limitation.

Group and link stars are independent. Group stars mean current membership, not a frozen list of links; leaving a group stops inclusion through that star. The Favourites filter is the union of sources/links covered by saved group stars and individually starred live links, intersected with the active view, search and other filters. Group membership is evaluated even when a different grouping mode is displayed. Group stars apply to Applications only; individual link stars work in both tabs and follow a source if its publication classification changes. A group match includes the full source card, including unresolved sources. A link-only match includes only individually starred URLs on that source card, with its identity and issues preserved. Stars never change unrelated stars.

Missing saved sources, links or derived groups stay dormant in storage and never render as invented or clickable entries. They reattach only if their exact identity returns in an eligible loaded view. The page can clear all favourites explicitly. Never resolve a saved URL independently of the current response.

Alternative: favourite by resource name, URL alone or group display text. These collide across namespaces, sources, modes and the literal/fallback Ungrouped cases.

### 5. A resilient browser preference envelope

Use a namespaced localStorage key such as `k8sautodocs.preferences`, with schema `version: 1`, a shared theme and directory grouping/density/collapse/custom-group/membership/favourite state. Read and validate once at startup, validate object shapes and allowed enum values, and catch read/parse/write failures. Import the existing `theme` key when no supported envelope exists, without deleting it. An existing explicit theme wins over the system preference; otherwise follow the system default. Theme applies before content paint where practical.

Unknown future versions remain untouched and use session defaults with a visible persistence notice. Malformed known-version data falls back safely; a subsequent user edit can save a clean supported envelope. Quota/blocked-storage failures retain usable in-memory settings and show a non-blocking notice that changes will not survive reload. Only genuine preference changes write storage; no fetches, payloads, tokens or search history are persisted. Same-origin tabs consume supported storage updates; simultaneous edits use last successful write semantics.

Scope is the current origin/browser profile. `/api/v1/cluster-info` supplies a display name and can be unknown in-cluster; it is not a trustworthy cluster ID. Do not key preferences off that name. A future installation identifier could enable finer partitioning, but needs a separate API decision.

Alternative: IndexedDB or server persistence. The small preference document fits localStorage and needs neither asynchronous database management nor an account/backend.

### 6. Search composes with layout and favourites

Trim and case-fold input, split whitespace into literal terms and require every term to match at least one indexed field. Search name, namespace, kind, ingress class, source group label, personal group name, hosts, URLs, Gateway/listener details and currently searched related-resource names/kinds/namespaces/label values. Do not interpret user input as HTML or regular expressions. Exact namespace/kind selectors combine with the query and Favourites using AND; favourite groups/links combine with one another using OR.

Preserve the query and selected filters across tab switches, so an unavailable selected namespace can honestly produce zero matches rather than silently clearing intent. Keep that selected value available to clear. Search is within the active loaded view, never a cross-view fetch. `Ctrl/Cmd+K` focuses search outside editable controls, Escape clears it, and the clear button restores results. Separate no data, no matches, no favourites, incomplete collection, loading and failed-request states. Announce results politely; do not hijack typing in the group editor.

Alternative: global search across both APIs or a query language. Both expand scope and complicate existing view semantics; literal term matching covers the requested daily navigation.

## Risks / Trade-offs

- [A visual redesign can obscure diagnostics] → Keep warning banners and unresolved-entry messages first-class, preserve related-resource details, and test partial responses and failed view transitions.
- [Browser data may disappear or be shared by installations behind one origin] → Document origin scope, tolerate missing storage, provide resets and avoid promises of device sync or authoritative cluster partitioning.
- [Favourite semantics become confusing after regrouping] → Use typed IDs, display group type, derive live membership independently of displayed mode and test rename/move/reclassification cases.
- [Large responses make DOM work expensive] → Index once, debounce search and render filtered results; assess a representative large fixture before considering virtualization.
- [Shared asset extraction breaks file-based tests or images] → Update the fixture harness to load/inject local dependencies and check all new shared assets in the packaged-image smoke check.
- [Parallel browser edits overwrite each other] → Consume storage events and document last-write semantics; conflict-free multi-device editing is outside scope.

## Migration Plan

1. Implement against fixtures and the existing API contracts; import the legacy theme preference and default new directory settings to today's label layout.
2. Verify browser behavior, theme/responsive visuals and local asset loading, then run existing Go/race and container checks. Update README and its embedded changelog with the actual shipped behavior.
3. Release through the existing image workflow when separately requested. A later GitOps change pins the verified image digest; this proposal needs no Helm or cluster configuration change.
4. Rollback uses the previous image digest. Older code ignores the new namespaced preferences; keep the legacy theme key and leave unknown future preference versions intact.

## Open Questions

No blocking decisions remain for implementation. This proposal assumes personal grouping applies to Applications while Discovered remains flat, and that a shared visual refresh covers all three existing pages. Confirm the subjective visual quality during implementation with representative desktop/mobile screenshots, not solely automated assertions.
