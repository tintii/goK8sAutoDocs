# Using AutoDocs

## Open the dashboards

- **Application Directory (default)**: http://localhost:8080/
- **Versions Dashboard**: http://localhost:8080/versions-dashboard-page
- **Existing directory URL**: http://localhost:8080/autodocs-dashboard-page
- **Optional overview**: http://localhost:8080/overview-page

The Auto-Docs dashboard opens on **Applications**, grouped by source label by default, with an **Ungrouped** section for published resources without a group. Groups can span namespaces; each card retains its namespace and kind. Label groups are sorted by exact value, followed by the distinct Ungrouped fallback. Cards are sorted by namespace, kind, and name.

Select **Discovered** to see Ingresses and HTTPRoutes that have no publication label, plus Services without that label that declare an ExternalDNS hostname. Both tabs support the same grouping and personalisation controls. The selected tab is included in the URL, so direct links, reloads, and browser back/forward navigation restore it. Both tabs preserve unresolved entries and display collection warnings. DNS-only Services count as sources with zero web URLs; Service type, advertised names, targets and declared externalName are searchable. Names and targets have keyboard-accessible Copy buttons with success/failure feedback. Group favourites and personal groups include DNS-only entries; individual link stars remain web-only.

## Arrange the versions table

Columns fit the loaded contents by default. Short kind/type labels stay on one line; long names, images and digests use ellipsis with their complete value available on hover and for selection/copy. The table scrolls horizontally on smaller screens.

Drag a heading divider to resize its column. Double-click a divider to fit that column's full contents, or use **Fit columns** to restore all content-aware defaults. A focused divider supports Left/Right arrows, Shift for larger steps, and Home to fit; Escape cancels a drag.

Click **Namespace**, then **Resource** to sort by namespace and name within it. The first heading is primary, the second secondary; another distinct heading replaces the secondary key. Click an active heading again to reverse its direction, then again to remove it. Numbered arrows and the sort summary show the priority. **Clear sort** restores API order so the next heading starts a new primary sort. All eight columns support sorting, including container fields; numeric text sorts naturally, missing values stay last, and search retains every container of each matching resource.

Widths and sort keys are saved in this browser alongside the existing directory/theme settings, survive reload/navigation and update other open tabs. Fit columns and Clear sort reset only their respective table choices. No inventory data is stored in preferences.

## Personalise your directory

**Group by** switches either tab between source label, namespace, resource kind, personal groups, cluster, environment, datacentre, Environment → Cluster, Cluster → Environment and no grouping. Composite headings show the ordered group path. Namespace groups distinguish clusters; environment and datacentre groups intentionally span them. The grouping choice is shared between tabs and saved in the browser. Changes apply immediately to loaded data. **Layout** selects grid or list, and group headings can collapse sections.

Open **Personalise** to create, rename, reorder or delete personal groups. Group names are trimmed and must be non-empty and unique by exact spelling. Creating a group selects Personal groups mode. Assign published or discovered endpoints with the personal-group selector on each card; each source has at most one assignment. Unassigned applications have a fallback section. Empty groups stay in the editor until assigned. Rename preserves membership and stars; deletion unassigns members and removes that group's star. Personal grouping never edits Kubernetes labels or publishes a discovered resource.

Star a group heading in either tab to follow its current membership in the active view, including unresolved sources. Star a link to save only that source and exact URL; a group star and its member link stars are independent. **Favourites** shows their combined live matches within the active tab, even if you have since changed the displayed grouping mode. A group match shows its whole source card; a link-only match shows only the individually starred URLs. Identical URLs from different sources have independent stars. Missing sources and links remain saved but are never rendered from browser storage alone; their stars return if the exact identity reappears. Individual link stars follow a source moving between Applications and Discovered; group stars apply to the same group identity in either tab, always using only that tab’s loaded sources.

Search matches case-insensitive, whitespace-separated literal terms across source names, namespaces, kinds, source/personal groups, hosts, URLs, Gateway/listener details and related-resource metadata. All terms must match somewhere on the source. Namespace/kind filters and Favourites further narrow the active view. Search and filters stay selected across tab switches; a namespace with no matches remains visible and clearable. Use **Ctrl/Cmd+K** outside an editable control to focus search, **Escape** in search or **Clear** to clear its query. Loaded resource/namespace/URL totals stay separate from the matching resource and displayed-URL count.

## Browser settings

Theme, grouping, density, collapsed groups, custom ordering/membership and stars live in the versioned `k8sautodocs.preferences` localStorage document. They are personal to this browser profile and origin, with no account, backend storage or device sync. Clearing site data removes them. Open tabs on the same origin receive settings updates; simultaneous edits use the last successful write. Changing hostname/port uses a different store, and stable configured cluster IDs distinguish source/link/namespace identities across clusters. Version-1 source preferences migrate only to the configured original `legacyClusterID`; ambiguous records stay dormant. Version-2 preferences preserve theme, personal groups and table layout across the migration.

Source identity uses kind, namespace and name, plus the exact URL for individual link stars. Recreating a source with the same identity inherits its personal settings. Renaming a source or changing a URL creates a different identity.

**Reset layout** restores default grouping, density and collapse choices and removes personal groups/membership/order and their stars, while retaining source-label/namespace/kind group stars and independent link stars. **Clear favourites** removes all stars while preserving layout and theme. Blocked/full storage leaves controls usable for the session and displays a notice. Malformed saved data falls back to defaults; unsupported newer versions are preserved without overwriting them.

All pages share a light/dark theme. Existing `theme` preferences are imported; otherwise the system preference supplies the default until you choose one explicitly. The interface uses locally bundled Noto Sans with regular body text, medium-weight headings, plain metadata labels, hairline borders and compact controls. Dark mode separates charcoal cards from the page; app initials keep their name-derived colours. The overview uses a plain surface; image names and versions retain monospace. Assets are packaged locally with no external font or icon dependency. Noto Sans is distributed under the [SIL Open Font License](https://github.com/tintii/goK8sAutoDocs/blob/master/go/static/NotoSans-OFL.txt); the Latin and Latin Extended variable WOFF2 subsets come from [Google Fonts](https://fonts.google.com/specimen/Noto+Sans), with system-font fallback for other scripts.

## Navigate and scope inventory

The shared sidebar footer links to the [GitHub project](https://github.com/tintii/goK8sAutoDocs) in a new tab, including an accessible icon link in the collapsed desktop rail and the expanded mobile menu.

Use the sidebar chevron to collapse navigation into an icon rail on desktop.
The desktop preference and Scope & updates submenu state persist across pages,
reloads and tabs. With no desktop preference, navigation starts expanded.
Scope & updates sits below Directory and Versions and above Browse groups,
so scope filters precede group choices on desktop and in the mobile drawer.
On mobile, **Menu** opens a closed-by-default modal drawer; Escape, the backdrop,
a destination or a group selection closes it and returns focus to Menu.
Mobile drawer state is transient and never changes the desktop preference.
Resizing clears modal focus/inert state. Blocked browser storage leaves controls
usable for the current session.

Directory groups come from the active Applications/Discovered inventory and the
selected grouping mode: source label, namespace, kind, personal, cluster,
environment, datacentre or composite paths. Configure source labels in the owning
Helm/Git resources; no separate sidebar category configuration is required.
The group list scrolls independently while fixed navigation stays reachable.
Long names wrap; collapsed desktop navigation can be expanded to read groups.
No grouping shows only **All groups**.

Selecting a group combines with search, favourites, scope and resource filters
using AND, without an inventory request. Counts show matching sources, while
choices remain present during filtering, including zero-match groups. Group
selection resets on view/grouping changes, reload, or refresh removing that
group. Resource filters and layout live in a labelled expandable panel; active
filters, matching/loaded source counts and inventory warnings remain visible.
Scope & updates retains collection details and refresh controls; the visible
freshness summary is separate from AutoDocs availability. The optional overview
retains its compact availability/theme header, hero and favourites.

Add arbitrary web destinations through [Manual directory links](manual-directory-links.md). Their optional relevance differs from their declaration source; shared links can appear under several selected scopes without becoming duplicates.
