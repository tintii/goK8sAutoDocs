## Why

Applications offers personal favourites and groups, but every visitor has to find the same commonly used links and important groups for themselves. Opening the directory also takes an extra click or shortcut before a visitor can start searching.

## What Changes

- Add a shared **Featured** section as the first group in Applications. Deployment admins choose and order existing published links, including manual links.
- Let admins set the shared order of Applications source-label groups so groups such as Observability and Media Services can appear near the top. Apply that order to the directory sections and the sidebar.
- Focus the directory search field on initial desktop entry. As soon as the user types a query, show quick results with keyboard navigation and direct link opening.
- Present quick results in the approved **Floating palette**, with the search field and results centered above the dimmed directory.

## Capabilities

### New Capabilities

- `shared-directory-curation` covers the operator-managed Featured selection, its interaction with directory filters, and the shared order for source-label groups.

### Modified Capabilities

- `endpoint-directory-organization` changes how Applications orders label groups and how search receives focus, presents quick results, and opens destinations.

## Impact

- The Go server gains an optional operator-managed YAML file and adds curation metadata to the published directory response.
- The directory model and UI apply shared curation to Applications and add quick search to both directory views.
- Deployment and user documentation cover the new settings, their defaults, and keyboard search.
