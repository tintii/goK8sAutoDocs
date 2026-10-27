## Context

The completed `add-gateway-endpoint-discovery` change is present in the local working tree. `go/autodocs.go` lists Ingresses and HTTPRoutes with an exact publication selector and resolves URLs, related resources, and issues. `go/resource_handlers.go` owns an independent response cache per handler, but that cache currently has no request-view dimension. `go/static/autodocs-dashboard.js` fetches one published array and renders namespace sections.

The user wants purpose-based grouped and ungrouped applications, plus a separate Discovered tab for resources without a publication label. Explicit false means hidden. This proposal builds on the completed change without modifying its artifacts or existing implementation.

## Goals / Non-Goals

**Goals:** One group per published source resource; groups spanning namespaces; a distinct discovery view; stable default API behavior; correct caching and navigation between views.

**Non-Goals:** Multiple groups/tags, nested groups, custom ordering, arbitrary grouping selectors, metadata editing, bulk publication actions, new resource kinds, a hidden-items tab, endpoint probing, or deployment changes. The versions dashboard and filtered-resources selection remain unchanged.

## Decisions

### 1. Classify publication independently of grouping

Inspect labels on the source Ingress or HTTPRoute itself:

| Publication label | Group label | Result |
|---|---|---|
| Exact `"true"` | Non-empty | Applications, named group |
| Exact `"true"` | Missing or empty | Applications, Ungrouped |
| Absent | Any | Discovered |
| Exact `"false"` | Any | Hidden from both views |
| Present with any other value, including empty or `"True"` | Any | Excluded from both views |

Only an absent key is undeclared intent. This preserves the existing exact-true contract and avoids treating malformed explicit intent as an invitation to display an endpoint. A group label alone never publishes anything. Legacy annotations do not affect classification. Labels on namespaces, Gateways, Services, Deployments, and Argo Applications are not inherited.

The alternative of using every non-true resource for discovery would make explicit hiding ineffective. Requiring both labels for Applications would incorrectly move intentionally published, ungrouped applications into Discovered.

### 2. Add one explicit API view while preserving the default

Use `/api/v1/auto-docs-dashboard` with an optional `view` parameter:

- Omitted, `view=`, or `view=published`: existing published-only response.
- `view=discovered`: resources with an absent publication label.
- Unknown or repeated view parameters: HTTP 400 before collection/cache lookup.

Both successful views retain the existing array entry schema and `X-AutoDocs-Warnings` diagnostic mechanism. Retain source labels so the browser can read `k8sautodocs.io/group`; no new publication database or redundant stored group field is required.

Select each view server-side. Published requests retain the exact-true list selector; discovered requests use label-key nonexistence selection for both Ingresses and HTTPRoutes, with presence-aware checks after listing. Reuse the existing resolver for both. Hidden resources do not enter either response or require endpoint/related-resource resolution.

This provides discovery without making the default API return unpublished resources or fetching a combined inventory into the browser. `/api/v1/filtered-resources` remains published-only for all its supported resource kinds.

### 3. Partition cache state by normalized view

The Auto-Docs handler needs two bounded cache partitions: published and discovered, each including its own payload, warnings, expiry, and concurrency protection. Normalize the request before choosing a partition; do not use arbitrary raw query strings as cache keys. Keep the existing cache lifetime and separation from filtered-resources. Two handlers/loaders or a bounded keyed handler are both acceptable implementation structures.

Responses describe their collection snapshot. A label change becomes visible as each relevant cache expires; no transactional snapshot across the two views is promised. Neither view may return the other view's cached payload, even under concurrent requests.

### 4. Present groups and discovery as separate tabs

Use the existing Auto-Docs page with Applications and Discovered tabs. Applications is the initial default. Store the selected tab in the page URL using `view=published` or `view=discovered` so refreshes, direct links, and browser back/forward navigation select the same tab. Unknown page values fall back to Applications. Use accessible tab selection and keyboard navigation.

Applications renders one section per exact non-empty group label, sorted by label value, followed by Ungrouped when needed. Preserve group spelling/case; display values as text. Represent the fallback bucket separately from real group strings, so a literal group value `Ungrouped` cannot merge with it. Every card shows namespace and kind because a group can span both. Sort cards by namespace, kind, then name. A source has one card containing its existing resolved links, not a duplicate per group or URL.

Discovered uses one flat card list with the same identity ordering and namespace/kind context. A group-only resource stays here. Both views reuse existing endpoint links, unresolved-entry presentation, related resources, and diagnostics.

Search applies within the active view, retaining existing searchable fields and adding the source group value. Preserve the search text when switching tabs. Resource/namespace/URL counters describe the active loaded view; the search result message describes its filtered subset. Empty messages distinguish no published entries, no discovered entries, and no search matches. Only show non-empty sections after filtering.

On a tab switch, show the new view's loading state and clear the old cards/warnings. Guard against out-of-order responses so a completed request for the previous tab cannot overwrite the active tab. On failure, show an error for the requested view without displaying the other view's stale data.

### 5. Keep publication changes in GitOps

The UI reads metadata and links only. Document setting publication and grouping labels on the owning Ingress/HTTPRoute through its source configuration. No label edits or deployment are part of this change. Existing URL derivation, optional Gateway API support, and partial-discovery diagnostics apply to Discovered as well.

## Risks / Trade-offs

- Broader discovery can resolve more resources and dependencies → Use view-specific selectors, reuse per-pass Gateway lookups, and retain caching; avoid collecting discovered entries for default published requests.
- A shared handler cache currently ignores query parameters → Partition payloads and warnings by normalized view and test both request orders and concurrency.
- Group names may collide with fallback wording or contain unexpected display characters → Keep fallback identity separate and use text-node rendering, preserving exact group identity.
- Independent cache ages can briefly show an old classification after label edits → Document refresh latency and test transitions after cache expiry; no cross-view atomicity is claimed.
- The prerequisite is complete but unarchived → Archive it before this change so the modified requirement baseline exists.

## Migration Plan

1. Implement and validate this proposal after the completed Gateway discovery change.
2. Existing exact-true resources remain in Applications, initially under Ungrouped unless a group is supplied. Existing default API clients keep their published-only results.
3. Add group labels through the owning GitOps configuration when desired. Resources with no publication label appear only when Discovered is selected. Use `publish: "false"` for explicit exclusion.
4. Rollback to the preceding implementation restores namespace grouping and removes the discovery view; group labels can remain harmless metadata.

This planning change performs no implementation, deployment, label edits, commits, or pushes.

## Open Questions

None blocking the agreed scope. Multiple group membership, display-name overrides, and custom ordering remain future discussions.
