## Context

The current overview links to the directory and Versions and displays configurable homepage content. The directory adds a hero, three large totals, search, view controls and filters before the sources. Shared JavaScript already stores `sidebarCollapsed`, themes and directory preferences. Source label groups, personal groups and multi-cluster grouping already exist.

The user selected Launchpad's layout and Catalogue's colours and confirmed a collapsible sidebar. The sample top navigation and four-category board do not scale to real inventories. The local mockup establishes visual direction only: it has fictional data and simplified interactions, and has not been visually verified in a browser.

## Goals / Non-Goals

**Goals:** Make links the initial content; combine compact tiles with warm colours; support 10–30 groups and long names; preserve all existing discovery and personalization semantics; keep navigation usable with keyboard, touch and restricted browser storage.

**Non-goals:** Introduce a category registry, edit cluster metadata, add libraries, replace version-table interactions, redesign the legacy overview content, or deploy this change.

## Decisions

### 1. Directory-first entry with compatible routes

Serve the same directory document at `/` and `/autodocs-dashboard-page`; do not maintain a second directory implementation. Existing query-based view and scope restoration must work at either route. The brand and fixed Directory navigation point to `/`; Versions retains `/versions-dashboard-page`. Do not treat unknown paths as directory aliases.

Keep the existing overview document available at `/overview-page`, with a secondary link in an overflow/settings area rather than a primary navigation destination. Preserve `/api/v1/home`, `HOMEPAGE_CONFIG`, its validation and existing content behavior on that page. Do not transplant the hero, orbit, dashboard chooser or MOTD into the directory. Document that configured homepage content is now shown on the optional overview. Removing that compatibility page or changing homepage configuration is a separate decision.

### 2. Warm colours, compact hierarchy

Use Launchpad's sans-serif typography, compact rectangular tiles, sidebar and grouped grid. Do not carry over Catalogue's oversized serif heading or horizontal categories. Light-theme starting tokens are background `#f6f3ec`, surface `#fffcf7`, text `#3a4037`, muted text `#75786d`, border `#dcded2`, accent `#456848` and selected surface `#e5eadc`. Verify contrast and adjust tokens where required; prototype colours are not a waiver of accessibility requirements.

Keep a small application heading, search, Applications/Discovered selection, favourites and result count above the cards. Use compact controls for grouping and scope; put less frequent personalisation and inventory details behind labelled expandable controls. Active filters and collection warnings must remain visible when their control panel is closed. Keep the application's availability indicator distinct from endpoint/workload health. No demo markers, marketing copy, giant totals or fabricated availability indicators enter production.

Apply shared tokens and navigation to Versions while preserving its table. Preserve dark mode, saved/system preference precedence and local asset packaging; derive dark tokens with muted olive accents and validate both themes.

### 3. Group navigation derives from the existing model

The sidebar separates fixed destinations from directory-local group choices. It lists All groups followed by groups from the selected grouping mode in the active Applications/Discovered view. Source-label mode reads `k8sautodocs.io/group` from the Ingress or HTTPRoute itself; `k8sautodocs.io/publish: "true"` remains the publication rule. Labels are configured through the owning Helm/Git resources. No separate menu config is introduced.

Retain namespace, kind, personal, cluster, environment, datacentre, composite and ungrouped modes. Personal groups use the existing editor and browser preference store. Preserve existing group ordering and stable identities, including synthetic fallbacks, cluster-qualified namespaces and composite paths. Render names as text. None mode shows All groups only.

Selecting a group filters the loaded view and combines with search, favourites and other scope/resource filters using AND. Build navigation membership from the loaded active view before these filters so choices do not disappear during search; show matching source counts alongside each group. Keep a selected zero-match group visible with a clear empty state. Reset group selection to All groups when changing view or grouping mode, or when refresh removes the selected group. This new selection is transient and does not redefine saved grouping or collapse state.

An expanded desktop sidebar has a bounded viewport height and independently scrollable group list. Fixed destinations and the collapse control remain reachable. Wrap long names and composite paths rather than requiring hover to identify them. A collapsed rail retains labelled fixed-destination icons and an expand control; it does not replace arbitrary group names with indistinguishable icons.

### 4. Desktop rail and mobile drawer

Reuse `sidebarCollapsed`: desktop starts expanded when no preference exists and respects an existing choice. Storage failure falls back to functional session state. Collapsing returns width to the main content; group choices remain reachable by expanding.

Use the current responsive breakpoint where practical. Below it, the sidebar becomes a closed-by-default modal drawer opened by a labelled menu button. Mobile open/close state is transient and does not overwrite the desktop preference. Trap focus while open, make background content inert, support Escape and backdrop dismissal, restore focus to the trigger, and close on selection. Crossing the breakpoint must remove stale inert/focus-trap state. Reduced-motion preferences suppress animation.

### 5. Real endpoint cards, not prototype shortcuts

Preserve source context, exact resolved URLs, group/link stars, multiple endpoints and unresolved sources. A single-endpoint card can have a large anchor target with sibling buttons for actions. Multiple-endpoint sources expose distinct labelled anchors; the card must not arbitrarily open the first endpoint. Unresolved sources remain visible without a fake clickable destination. Keep metadata and diagnostics available without nested interactive elements.

Retain current search semantics and Ctrl/Cmd+K, grid/list density, collapse choices, personal group editing, inventory updates, scope URL restoration and Versions preferences. Prefer existing shared model/rendering helpers over reproducing them from the sample script.

## Risks / Trade-offs

- Root-route change moves configured home content out of the default journey. Mitigation: retain and document the secondary overview route and link, with no configuration deletion.
- Compact controls can hide operational context. Mitigation: keep active filters, result counts and warnings visible; test partial and stale inventory states.
- Existing unarchived specs describe the old shell. This is a focused successor; reconcile the dashboard-experience wording when archiving rather than erasing prior implementation history now.
- Dynamic group navigation can drift from card identities. Mitigation: use the same directory model and exercise duplicate names, fallbacks and composites.
- Shared preferences and responsive focus management can regress unrelated pages. Mitigation: preserve existing keys and test reload, cross-tab changes, blocked storage and viewport transitions.

## Migration Plan

Implement the shell and routing with local fixtures, then run focused browser and route checks plus existing Go/browser regression checks appropriate to the shared files. Update README and CHANGELOG after behavior is implemented. No image publication or GitOps rollout is authorized by this proposal. Reverting the future implementation restores the previous root page without changing cluster state or deleting stored preferences.

## Open Questions

None blocking proposal creation. Exact spacing, breakpoint and dark-theme tokens should be resolved against actual long-name and multi-endpoint fixtures during implementation, not inferred from the 12-app mockup.
