## Why

AutoDocs already discovers and groups endpoints, but its separate dashboard pages lack a cohesive visual identity and its label-only layout cannot be personalised while browsing. A polished, responsive interface with adjustable grouping, favourites and stronger search will make it useful as a daily homelab launchpad while keeping Kubernetes discovery read-only.

## What Changes

- Give the landing page, endpoint directory and versions dashboard a shared visual system: graphite and warm neutral surfaces, teal accents, clear typography, responsive navigation, light/dark themes and deliberate loading, empty and error states. Make the endpoint directory the visual centrepiece.
- Add immediate grouping controls in both Applications and Discovered for source label, namespace, resource kind, custom groups or no grouping. Let users create, rename, reorder and delete personal groups and move sources between them without changing Kubernetes labels or redeploying.
- Add independent star controls for whole groups and individual resolved links, plus a Favourites filter. A starred group follows its current membership; a starred link identifies one source and exact URL.
- Persist favourites, custom membership, grouping, ordering and display preferences in versioned browser localStorage. Handle unavailable or invalid storage gracefully and provide separate reset controls for layout and favourites.
- Improve existing active-view search with multiple terms, personal group names, namespace/kind filters, a keyboard shortcut, clear actions and accurate result counts. Preserve current searchable metadata, view URLs, diagnostics and safe links.
- Keep Applications and Discovered classification, API schemas, endpoint resolution and cache behavior compatible.

## Capabilities

### New Capabilities

- `dashboard-experience`: A cohesive, accessible and responsive visual shell across all dashboard pages, including themes and honest data states.
- `browser-directory-personalization`: Runtime grouping choices, editable personal groups, group/link favourites and resilient browser-only persistence.

### Modified Capabilities

- `endpoint-directory-organization`: Extend grouping in both views beyond the label-based default and expand active-view search/filter behavior while preserving publication boundaries, navigation and Discovered presentation.

## Impact

- Primary implementation is in `go/static/autodocs-dashboard.html` and `go/static/autodocs-dashboard.js`; shared local CSS/JavaScript will also serve `index.html` and `versions-dashboard.html`.
- Extend the existing headless-browser fixture tests and asset packaging checks; update the application README and its embedded changelog when implementing.
- Use the existing Go static server and API payloads, with no frontend framework, remote asset service, backend database, Kubernetes write permission or new chart requirement.
- Preferences belong to the browser profile and origin. They do not sync between devices, and clearing browser data removes them. The current API supplies no authoritative cluster identifier, so isolation is per origin rather than inferred from the display name.
- This change creates planning artifacts only. Implementation, image publication and any later GitOps digest update are separate steps; no homelab deployment files need changing for this proposal.
