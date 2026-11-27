## 1. Shared directory settings

- [x] 1.1 Load and validate the optional `AUTODOCS_DIRECTORY_CONFIG` YAML on published directory reads and use default curation with a warning when the configured file fails.
- [x] 1.2 Match feature references and exact source-label values against the published snapshot rows. Add optional ranks without mutating retained inventory or changing the response shape.
- [x] 1.3 Verify configuration reload, rejection and fallback, exact Kubernetes and manual link identities, and compatibility with Discovered and restored snapshots.

## 2. Featured and group order

- [x] 2.1 Apply the shared source-label ordering to Applications sections and sidebar choices, including the fallback and groups that appear or disappear under filters.
- [x] 2.2 Render the Featured links before ordinary Applications results from the existing filtered rows. Verify link-only favourites, active scope and group selection, and stable counts and ordinary cards.

## 3. Quick search

- [x] 3.1 Add initial desktop search focus and the idle placeholder, then derive quick results from the final active-view projection while retaining the query, caret, and existing filters.
- [x] 3.2 Implement the approved Floating palette with a centered overlay, dimmed directory, responsive layout, and keyboard, pointer, and assistive-technology interactions.
- [x] 3.3 Verify first typing, destination opening, sources without web links, Escape and shortcut behavior, touch entry, and quick-result invalidation across view changes, refreshes, and errors.

## 4. Documentation and regression checks

- [x] 4.1 Document the settings and mount/reload behavior, provide an example configuration, and describe Featured, group order and keyboard search in the user manual.
- [x] 4.2 Run the existing Go and directory JavaScript suites, plus a desktop and narrow-viewport smoke check on both directory routes in light and dark appearances.
