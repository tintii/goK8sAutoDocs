## Why

The versions table breaks short values such as Deployment and Main across lines while assigning unnecessary space to other columns. Users cannot resize columns or order the cluster inventory by namespace and then resource name, which makes comparison difficult.

## What Changes

- Give columns content-aware default widths; keep short values intact, truncate long values with their full text available, and scroll inside the table on narrow screens.
- Add pointer and keyboard column resizing, double-click auto-fit for an individual column, and a Fit columns reset.
- Make every column heading sortable with up to two ordered keys. Clicking Namespace and then Resource sorts by namespace, then resource within each namespace. Show priority and direction and provide Clear sort.
- Save column widths and sorting in browser preferences, preserving existing directory and theme settings.
- Add browser regression coverage, publish the tested image, update the GitOps digest, and verify the deployed table.

## Capabilities

### New Capabilities

- `versions-table-interaction`: Content-aware column sizing, accessible resizing, two-level sorting, and personal table preferences.

### Modified Capabilities

None. Existing directory requirements remain unchanged.

## Impact

Changes affect the versions HTML/JavaScript, shared CSS/preferences, browser fixtures/tests, and user documentation. No API, RBAC, chart, or dependency changes are required. Deployment uses the existing application image pipeline and a verified digest in the sibling homelab-gitops repository, retaining its published workload chart.
