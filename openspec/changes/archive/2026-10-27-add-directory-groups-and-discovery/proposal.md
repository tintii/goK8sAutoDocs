## Why

The endpoint directory currently groups published applications by namespace and omits endpoints without AutoDocs publication metadata. Users need purpose-based groups for their chosen applications and a separate place to find endpoints that have not declared publication intent.

## What Changes

- Add Applications and Discovered tabs to the Auto-Docs dashboard, with Applications as the default.
- Group Applications by the source Ingress or HTTPRoute's `k8sautodocs.io/group` label. Published entries without a non-empty group appear under Ungrouped; groups can span namespaces.
- Show supported Ingresses and HTTPRoutes with no `k8sautodocs.io/publish` label in Discovered, even when they carry a group label or legacy annotations.
- Treat `publish: "false"` as an explicit hide from both tabs. Other present values besides exact `"true"` also stay out of both views; only an absent publication label qualifies for discovery.
- Preserve the default published-only Auto-Docs API and add an explicit discovered view with the same entry schema, URL resolution, and diagnostic behavior.
- Keep response caches separate by view, and make group/search/count/empty-state behavior match the active tab.
- Keep Kubernetes metadata as the source of truth; no editing UI, cluster writes, or separate database.

## Capabilities

### New Capabilities

- `endpoint-directory-organization`: Applications grouped by a dedicated label, an Ungrouped section, and a separate Discovered view classified by publication-label presence, with view-aware API and caching behavior.

### Modified Capabilities

- `published-endpoint-directory`: Scope publication-only selection to the default/published Auto-Docs view and filtered-resources, and extend combined discovery to the explicitly requested view.

The canonical `openspec/specs/` tree is currently empty. The modified requirements use the full baseline from the completed, unarchived `add-gateway-endpoint-discovery` change. Archive that prerequisite first when promoting these changes to canonical specs; do not rewrite its completed artifacts.

## Impact

- Backend: `go/autodocs.go` collection/classification and shared entry resolution; `go/resource_handlers.go` request view validation and cache separation.
- Frontend: `go/static/autodocs-dashboard.html` and `.js` tabs, group sections, namespace context, active-view search/counts, and empty/loading/error states.
- API: `/api/v1/auto-docs-dashboard` gains `view=discovered` and an explicit `view=published` equivalent to its existing default. The array response shape remains unchanged. `/api/v1/filtered-resources` remains published-only; the versions dashboard is unaffected.
- Tests: backend publication-state/view/cache fixtures and dashboard grouping/navigation/rendering fixtures.
- Documentation: README usage, label examples, discovery visibility, and its changelog during implementation.
- Permissions/dependencies: reuse the existing read permissions and Gateway API support; discovery broadens which already-readable endpoint resources are displayed. Deployment, Helm configuration, commits, and pushes are outside this change.
