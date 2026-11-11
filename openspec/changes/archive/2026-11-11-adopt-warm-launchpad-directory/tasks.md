## 1. Routing and compact shell

- [x] 1.1 Serve the directory at `/` and its existing route with view/scope restoration; expose the existing overview at `/overview-page` and keep unknown paths from becoming directory aliases.
- [x] 1.2 Update shared brand/navigation links; retain Versions and a secondary overview link, home API/config behavior and availability semantics.
- [x] 1.3 Replace the directory hero and large totals with a compact heading, search, view controls, visible active-filter summary and honest counts/warnings.
- [x] 1.4 Apply the warm light palette and complementary dark tokens to the shared shell, preserving theme preferences and Versions table behavior.

## 2. Scalable group navigation

- [x] 2.1 Build sidebar groups/counts from the existing model for all supported grouping modes; preserve exact identities, ordering, fallbacks and personal groups.
- [x] 2.2 Implement transient group selection, AND filtering, zero-match handling and reset rules without additional inventory requests.
- [x] 2.3 Make long group lists independently scrollable; preserve readable long/composite names and fixed navigation.
- [x] 2.4 Reuse desktop collapse persistence and implement an accessible mobile drawer with separate transient state and breakpoint cleanup.

## 3. Cards and existing controls

- [x] 3.1 Implement compact tile styling and safe single/multiple/unresolved endpoint presentations with separate star/action controls.
- [x] 3.2 Retain grid/list layouts, section collapse, personal group editing, group/link stars, full search semantics and keyboard shortcuts.
- [x] 3.3 Retain resource/scope filters, URL navigation, collection diagnostics and refresh controls in compact, labelled panels with visible active state.

## 4. Verification and documentation

- [x] 4.1 Add focused route coverage for `/`, the existing directory URL, the secondary overview and unsupported paths.
- [x] 4.2 Add browser fixtures for at least 12 groups, long/duplicate names, synthetic fallbacks, composites, None mode and group-selection reset behavior.
- [x] 4.3 Exercise desktop collapse/reload, blocked storage, mobile drawer keyboard/focus behavior and breakpoint transitions at 375px and 1440px in both themes; verify contrast and no page-wide overflow.
- [x] 4.4 Exercise combined filters, multi-endpoint and unresolved cards, independent stars, loading/failure/partial inventory states, scope restoration and Versions regression behavior.
- [x] 4.5 Run appropriate Go and browser regression checks and asset checks; report actual results and any unavailable browser validation.
- [x] 4.6 Update README and CHANGELOG with the default page, retained overview/config behavior, dynamic grouping and responsive navigation. Keep local mockup files ignored.
