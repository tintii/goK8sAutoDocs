## ADDED Requirements

### Requirement: Readable content-aware column defaults

The versions table SHALL size its eight columns using the full loaded inventory and headings, keep short kind/type values intact, show long text on one line with ellipsis and full accessible text, and contain horizontal overflow within the table region. Filtering SHALL NOT unexpectedly resize columns.

#### Scenario: Representative inventory at desktop width
- **WHEN** workloads with long resource names, image paths, tags and digests are loaded
- **THEN** columns use content-aware widths, Deployment and Main remain unbroken, and long values retain their complete text

#### Scenario: Narrow viewport and filtering
- **WHEN** the table is opened at 375px and a search is performed
- **THEN** the table scrolls internally without widening the page and column widths remain stable during filtering

### Requirement: Adjustable accessible columns

Each column SHALL support pointer dragging and keyboard resizing within bounded widths, independent of sorting. Double-clicking its divider or pressing Home on that divider SHALL fit that column to its contents. Fit columns SHALL reset all manual widths and restore content-aware defaults. Canceled dragging SHALL restore the prior width without saving it.

#### Scenario: Resize without sorting
- **WHEN** a user drags a divider or presses ArrowRight on its focused separator
- **THEN** only that column changes width and the selected sort keys and row order remain unchanged

#### Scenario: Reset or cancel
- **WHEN** a user cancels a drag, auto-fits one divider, or selects Fit columns
- **THEN** the drag is restored, the individual column fits its values, or all explicit widths are cleared respectively

### Requirement: Two-level heading sorting

Every column SHALL be sortable using ordinary heading activation. The first distinct heading SHALL be primary and the second secondary. A third distinct heading SHALL replace secondary while retaining primary. Activating a selected heading SHALL cycle ascending, descending and removal; removing primary SHALL promote secondary. Clear sort SHALL restore original API order. Sorting SHALL compare numeric text naturally, keep missing values last in both directions, and retain original order for equal rows.

#### Scenario: Namespace then resource
- **WHEN** a user clicks Namespace then Resource
- **THEN** rows are sorted by namespace ascending and resource ascending within each namespace, with visible priorities 1 and 2

#### Scenario: Reverse, remove and replace keys
- **WHEN** active headings are clicked again, removed, or a third distinct heading is clicked
- **THEN** directions cycle, secondary is promoted when primary is removed, and at most two unique keys remain

#### Scenario: Container sorting and search
- **WHEN** a user sorts by a container field and filters resources
- **THEN** all containers of each matching resource remain present and the visible rows follow the selected sort without changing resource totals

### Requirement: Accessible sort feedback

The table SHALL provide native keyboard-operable heading buttons, visible direction/priority indicators, a live textual summary, and primary-column aria-sort. Resize separators SHALL expose their label, orientation and current/minimum/maximum width and retain focus through interaction.

#### Scenario: Keyboard operation
- **WHEN** a keyboard user activates headings and resizes a focused divider
- **THEN** the requested sorting/resizing occurs, sort priority is announced, and focus remains usable

### Requirement: Personal table settings

The application SHALL persist only valid table widths and up to two sort keys in the existing browser preference document, restore them across reload/navigation, and react to same-origin tab changes. Older preference documents SHALL use table defaults without losing theme/directory state; invalid table settings SHALL reset only table settings. Blocked storage and unsupported future schemas SHALL preserve usable session controls and existing notices without overwriting future data. Fit columns and Clear sort SHALL leave unrelated settings intact.

#### Scenario: Reload and existing settings
- **WHEN** a user resizes and sorts, navigates to the directory, changes theme, and returns or reloads
- **THEN** table settings and existing directory/theme choices are preserved

#### Scenario: Invalid or unavailable storage
- **WHEN** table settings are malformed, storage is unavailable, or a future preference version is present
- **THEN** the table remains usable and valid existing settings or future stored data are preserved as applicable

#### Scenario: A second tab updates table preferences
- **WHEN** another same-origin tab saves widths or sorting
- **THEN** the open versions table reflects those valid changes without refetching the inventory
