# directory-launchpad Specification

## Purpose

Define directory-first routing, compact application tiles, scalable group navigation and responsive desktop/mobile navigation while preserving directory behavior.

## Requirements

### Requirement: Directory-first routing

The application SHALL serve the application directory at `/` and `/autodocs-dashboard-page` using the same implementation. Both routes SHALL retain view and scope URL restoration. The brand SHALL link to `/`. Versions SHALL remain available at its existing route. The former overview SHALL remain reachable at `/overview-page` through secondary navigation, retaining its home API and configuration behavior. Unknown paths SHALL NOT render the directory as a fallback.

#### Scenario: Open the default page
- **WHEN** a user visits `/` without view parameters
- **THEN** Applications, search and directory controls are available directly without an intermediate overview or dashboard chooser

#### Scenario: Use existing and secondary routes
- **WHEN** a user opens an existing directory URL selecting Discovered, or the Versions URL
- **THEN** the requested view and existing functions remain available
- **AND** `/overview-page` retains configured overview content without inserting it into the directory

### Requirement: Compact Signal visual hierarchy

The directory SHALL use compact sidebar-and-tile structure, Signal light/dark themes, off-white light and charcoal dark surfaces, green accents and locally bundled Noto Sans typography with hairline borders and readable plain metadata. It SHALL omit the directory hero, promotional copy and large statistics cards. Counts, active filter state and applicable warnings SHALL remain visible. Shared navigation and theme tokens SHALL apply to Versions without changing table functionality. Both themes SHALL preserve existing preference precedence, local assets and accessible contrast.

#### Scenario: Browse loaded applications
- **WHEN** inventory is available
- **THEN** a compact heading, search, directory controls and application cards form the primary content
- **AND** the result summary distinguishes matching sources from loaded sources without suggesting endpoint health

#### Scenario: Retain dark theme
- **WHEN** a user with a saved dark preference opens the directory or Versions
- **THEN** the chosen theme remains active and controls and text remain readable

### Requirement: Dynamic scalable group navigation

The directory SHALL derive sidebar group choices from the active view and selected existing grouping mode without a new category registry. Source labels and browser-local personal groups SHALL retain their current configuration semantics. The sidebar SHALL distinguish fixed destinations from group filters and support at least 12 groups and long names without page-wide horizontal overflow. Group lists SHALL scroll independently on desktop while fixed destinations and the collapse control remain reachable. None grouping SHALL expose All groups only.

Scope and update controls SHALL appear below fixed navigation and above Browse groups, with matching visual and keyboard order on desktop and mobile.

#### Scenario: Discover more groups
- **WHEN** loaded resources contain 12 distinct source-label groups
- **THEN** all 12 groups are reachable in the sidebar without creating horizontal category tabs or editing menu configuration
- **AND** resources without a group retain a distinct fallback choice

#### Scenario: Preserve group identity
- **WHEN** displayed names overlap across cluster-qualified namespaces, composite paths or synthetic fallbacks
- **THEN** each choice retains the directory model's distinct identity and appropriate membership

### Requirement: Group selection composes with directory filters

Selecting a sidebar group SHALL filter loaded sources using AND with existing search, favourites and scope/resource filters without an additional inventory request. Group membership choices SHALL derive from the active loaded view before these filters, with counts showing matching sources. Zero-match selections SHALL remain visible and clearable. Changing view or grouping mode, or removing the selected group during refresh, SHALL reset selection to All groups. Group selection SHALL be transient and independent of saved grouping and section-collapse preferences.

#### Scenario: Search within a group
- **WHEN** a user selects a group and applies a namespace filter and search query
- **THEN** results satisfy all three conditions and sidebar counts reflect those conditions
- **AND** a zero-result group remains identifiable with an empty state and a way to clear filters

#### Scenario: Change grouping mode
- **WHEN** a user switches from Label to Namespace with a group selected
- **THEN** group selection resets to All groups and navigation is rebuilt from namespace identities without fetching inventory

### Requirement: Persistent desktop rail and transient mobile drawer

Desktop navigation SHALL start expanded when no preference exists, support collapse to a narrow rail and reuse the existing browser collapse preference. The rail SHALL retain accessible fixed navigation and a control to expand group names. Mobile navigation SHALL start closed behind a labelled menu button and open as a modal drawer without overwriting the desktop preference. The drawer SHALL manage focus, inert background content, Escape/backdrop dismissal, selection dismissal and focus restoration. Breakpoint changes SHALL remove stale modal state. Storage failures SHALL leave navigation functional for the session.

#### Scenario: Restore desktop preference
- **WHEN** a user collapses the desktop sidebar and reloads with storage available
- **THEN** the rail remains collapsed and the user can expand it to browse group names

#### Scenario: Operate mobile drawer
- **WHEN** a keyboard user opens the mobile drawer and dismisses it with Escape
- **THEN** focus stays within the drawer while open and returns to its trigger after closing
- **AND** desktop collapse preferences are unchanged

#### Scenario: Resize while drawer is open
- **WHEN** a mobile viewport becomes desktop width with the drawer open
- **THEN** desktop navigation follows its saved preference and the main content is no longer inert or subject to a drawer focus trap

### Requirement: Preserve real directory behavior

The redesigned directory SHALL retain Applications/Discovered boundaries, all supported grouping and scope modes, existing search semantics and shortcuts, personalisation, favourites, grid/list layouts, diagnostics and refresh behavior. Cards SHALL use exact server-resolved URLs with existing safe-link attributes. Multiple endpoints SHALL have distinct links; unresolved sources SHALL remain visible without invented links. Action controls SHALL NOT be nested inside links or cause navigation when operated. The UI SHALL distinguish loading, error, empty inventory, zero matches and no matching favourites without fabricating health or freshness.

#### Scenario: Open a source with multiple endpoints
- **WHEN** a source has two resolved URLs and a user activates the second endpoint
- **THEN** that exact URL opens with safe external-link behavior
- **AND** toggling a star does not open either URL

#### Scenario: Incomplete inventory
- **WHEN** collection returns available sources, unresolved entries and warnings
- **THEN** usable links remain available and applicable issues and warnings remain visible
- **AND** no link or healthy status is invented for unresolved entries
