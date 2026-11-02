## Why

AutoDocs is primarily a dashboard for opening application links. The separate overview, directory hero, promotional copy and large statistics blocks delay that task. The selected direction combines prototype 1's compact sidebar-and-tile layout with prototype 3's warm paper surfaces and olive accents. Navigation must work with ten or more groups rather than assuming a handful of horizontal category tabs.

## What Changes

- Make `/` open the application directory, with search, compact controls and clickable application tiles immediately available. Keep the existing directory and Versions URLs working; retain the old overview at `/overview-page` as a secondary compatibility route.
- Replace the directory hero and statistics cards with a compact heading and accurate result summary. Remove the intermediate dashboard chooser from the default journey.
- Apply the selected warm light palette to the existing shared shell; retain the saved/system theme behavior and a complementary accessible dark theme.
- Place dynamically derived directory groups in an independently scrollable sidebar, separate from fixed Directory and Versions navigation. Use the existing grouping modes and data identities, not a hardcoded category taxonomy.
- Make the desktop sidebar collapsible to a narrow rail, remembering its state in existing browser preferences. On mobile, use a closed-by-default accessible drawer with its own transient state.
- Preserve Applications/Discovered, scope and resource filters, personal groups, favourites, grid/list layouts, diagnostics, refresh controls and safe endpoint behavior. Keep advanced controls reachable without dominating the initial view.

## Capabilities

### New Capabilities

- `directory-launchpad`: Directory-first routing, warm visual hierarchy, scalable group navigation, responsive sidebar behavior and compatibility with existing directory functionality.

### Modified Capabilities

None. This change adds a presentation contract on top of existing directory and inventory semantics. It supersedes the landing-page-first presentation in the implemented, unarchived `redesign-autodocs-ui` change; it does not replace its personalization or accessibility requirements.

## Impact

- Update Go page routing, directory HTML/JavaScript, shared CSS/navigation/preferences, and related browser and route fixtures. Versions receives shared shell styling, not a table redesign.
- Preserve API contracts, publication labels, endpoint resolution, inventory coordination, RBAC and browser preference identities. No new frontend dependencies or backend configuration are required.
- Document default-route and legacy-overview behavior, group configuration, navigation and theme behavior in README/CHANGELOG during implementation.
- Local visual reference: `.nocommit/ui-options/warm-launchpad.html`. It is intentionally ignored and is not an implementation dependency; the tracked design and requirements describe the agreed direction independently.

## Non-goals

No Kubernetes changes, deployment, new category configuration language, cross-device preferences, replacement discovery logic, versions-table redesign, or copying prototype fixture data into production. This proposal does not implement the application changes.
