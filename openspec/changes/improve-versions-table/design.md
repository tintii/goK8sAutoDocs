## Context

The versions page is a vanilla JavaScript table backed by an array of workloads and containers. Shared CSS currently allows arbitrary word breaking and the browser chooses widths. Shared version-1 browser preferences already handle theme, directory state, blocked storage, newer schemas, and cross-tab updates. The existing image pipeline validates Go, browsers, and a native amd64 container before publication.

## Goals / Non-Goals

**Goals:** Readable defaults, direct column sizing, ordinary-click two-key sorting, accessible controls, saved personal settings, and a verified GitOps rollout.

**Non-Goals:** API changes, server-side sorting, directory behavior changes, column reordering/hiding, additional dependencies, or chart/RBAC upgrades.

## Decisions

1. **Content-aware explicit widths.** Measure loaded text using the rendered column font and establish bounded preferred widths. Short badge columns and headings retain enough room for their complete labels. Distribute spare desktop space and shrink flexible long-text columns to readable minima when possible; overflow remains inside the focusable table region on small screens. Use a colgroup and fixed table layout so dragging a divider changes that column predictably. Long cell text is single-line with ellipsis and its complete value in a title, the DOM and an accessible label. Browser auto layout alone cannot provide predictable manual sizing and caused the reported wrapping.
2. **Independent resize controls.** Each heading has a sort button and a focusable vertical separator. Pointer capture supports mouse/touch/pen dragging. Arrow keys resize, Shift increases the step, Home auto-fits, and double-click auto-fits one column. Clamp widths and save only completed interactions; Escape or pointer cancellation restores the prior width. Fit columns clears all explicit widths and recomputes defaults from the full inventory. Sorting is never triggered by resizing.
3. **Two ordered keys without modifier keys.** Start with API order and no active sort. A new heading becomes primary, the next becomes secondary, and a third distinct heading replaces secondary while retaining primary. Clicking an active heading cycles ascending, descending, then removal; removing primary promotes secondary. Clear sort restores API order. Compare flattened container rows lexicographically with a numeric, case-insensitive collator; missing values sort last in either direction and original row order breaks complete ties. This permits all eight columns to sort and preserves every container. Search keeps its current resource-level semantics, then the selected sort orders visible rows. Counts remain resource/container totals, independent of row order.
4. **Visible and accessible sort state.** Heading indicators show arrow and priority 1/2. A live summary names both keys and directions. Only the primary header uses aria-sort; both buttons describe current priority/direction and next action. Native buttons support keyboard activation. This avoids requiring Shift-click or representing two simultaneous aria-sort columns.
5. **Additive personal preferences.** Add optional `versions: {widths, sort}` to the existing version-1 document. Older documents receive defaults. Validate only known column IDs, bounded numeric widths, unique keys, two levels, and asc/desc directions. Invalid table preferences reset just that section, preserving valid directory/theme state. Existing future-schema and unavailable-storage handling applies. Preferences and cross-tab updates never persist Kubernetes inventory or make writes to Kubernetes.

## Risks / Trade-offs

- Long values require truncation to fit useful desktop widths → preserve full text for hover, selection/copy and assistive technology; allow generous individual auto-fit and manual enlargement.
- Two-level click behavior differs from common single-sort tables → show concise instructions, visible priorities, and Clear sort.
- Stored sizes may not suit a smaller window → keep the page responsive with internal scrolling and a Fit columns reset; never silently overwrite user widths on window resize.
- Shared preferences can affect other dashboards → additive defaults, isolated validation and regression coverage for existing personalisation.

## Migration Plan

Validate the proposal and local Go/race/browser checks, publish source through the existing workflow, verify the image revision/platform and anonymous digest pull, then pin the digest in GitOps with README/CHANGELOG/runbook updates. Retain chart 0.4.0 and unrelated local edits. Verify one ready replica, exact HTTPS assets and real browser sorting/resizing/reload behavior. Roll back by restoring the prior verified digest. The completed proposal remains available with implementation and deployment tasks checked.

## Open Questions

None; the user's Namespace → Resource example defines sort priority. Clear sort starts a new primary key.
