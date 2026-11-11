## ADDED Requirements

### Requirement: Cohesive dashboard shell

The landing page, endpoint directory and versions dashboard SHALL share navigation, typography, spacing, theme tokens and interactive-control styling. Existing page routes and versions-dashboard functionality SHALL remain available. The directory SHALL present a clear hierarchy of context, loaded totals, search, view/grouping controls, group headings and endpoint cards. Production pages SHALL use packaged local assets without requiring remote fonts or icon services.

#### Scenario: Navigate between dashboards
- **WHEN** a user opens the landing page and visits either dashboard
- **THEN** each page presents consistent branding, navigation and theme controls
- **AND** existing dashboard routes and functions remain usable

#### Scenario: External asset services are unavailable
- **WHEN** the browser cannot reach external font, icon or image services
- **THEN** dashboard styling and controls remain complete using packaged assets and system fonts

### Requirement: Responsive and accessible interaction

The interface SHALL remain usable at 375-pixel mobile and 1440-pixel desktop viewport widths without page-wide horizontal overflow. Wide versions tables SHALL have their own scrolling container. Search, tabs, stars, grouping, custom-group editing and theme controls SHALL be labelled and keyboard operable, with visible focus and exposed selected/pressed state. Dialogs or panels SHALL restore focus when closed. Text and controls SHALL meet WCAG AA contrast targets, stars SHALL communicate state beyond color, and transitions SHALL respect reduced-motion preferences.

#### Scenario: Personalise on a narrow screen using a keyboard
- **WHEN** a user at mobile width opens the group editor, edits a group and closes it using keyboard controls
- **THEN** all fields and actions are reachable without page overflow
- **AND** focus returns to the control that opened the editor

#### Scenario: Inspect a star without color cues
- **WHEN** a group or link star is toggled
- **THEN** its accessible pressed state and visible shape or text identify the new state
- **AND** toggling a link star does not open the destination

### Requirement: Shared persistent themes

All three pages SHALL support light and dark themes through the shared browser preference store. A supported saved choice SHALL override the system preference; otherwise the system preference SHALL determine the initial theme. The existing `theme` localStorage value SHALL be imported when no supported new preference document exists. A theme storage failure SHALL NOT prevent page initialization.

#### Scenario: Retain an existing dark preference
- **WHEN** a browser with the legacy dark theme value and no new preferences visits any dashboard
- **THEN** the page uses dark theme and retains that choice when navigating to another dashboard

#### Scenario: Storage is unavailable
- **WHEN** localStorage reads or writes throw
- **THEN** the page still loads and theme controls work for the current session

### Requirement: Honest directory states

The directory SHALL distinguish loading, failed loading, empty active view, zero search/filter matches and no matching favourites. Collection warnings, source issues and unresolved entries SHALL remain visible in their applicable states. Displayed totals SHALL describe actual loaded data, with filtered counts reported separately. The UI SHALL NOT invent health status, endpoints or successful refresh timestamps from unavailable data.

#### Scenario: Incomplete discovery
- **WHEN** a response contains available sources and collection warnings
- **THEN** the page displays those sources and a visible warning explaining incomplete collection
- **AND** it does not describe the response as proof of endpoint health

#### Scenario: Empty favourites differs from no sources
- **WHEN** sources are loaded but the Favourites filter has no live matching stars
- **THEN** the page explains that no favourites match and offers a way to clear the filter
- **AND** complete-view totals remain based on the loaded sources
