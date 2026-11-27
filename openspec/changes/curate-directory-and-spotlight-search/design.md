## Context

The Go server exposes a read-only published directory as an array of inventory sources. The directory model projects that array into groups and applies search, scope, and browser-local preferences. AutoDocs deployments already mount operator-managed YAML for homepage content. Deployment permissions provide the admin boundary for shared directory curation.

## Goals / Non-Goals

**Goals**

- Share a curated set of existing launch links and an ordered list of source-label groups across browsers.
- Give desktop users a ready-to-type field and a keyboard-first way to open results in either directory view.
- Keep directory curation and search compatible with retained and live inventory, manual links, and existing filters.

**Out of scope**

- In-app accounts, roles, or administrative editing.
- Personal reordering of source-label groups, and shared reordering of other grouping modes.
- A new publication mechanism or a new search service.

## Decisions

### Put curation in deployment configuration

Read an optional mounted YAML file selected by `AUTODOCS_DIRECTORY_CONFIG`. A deployment admin commits or edits the file through the existing deployment configuration. Read it on published directory requests, following the mounted-file reload pattern used for home content. For Kubernetes, mount the containing directory so atomic ConfigMap updates are visible. A page reload or the next published directory fetch picks up changes.

The file carries an ordered list of exact link references and an ordered list of exact source-group values. Identify a Kubernetes source by cluster ID, kind, namespace, and name. Identify a manual source by the declaration cluster ID, ConfigMap namespace and name, and entry ID. Use the exact live URL as part of either reference so a feature follows one destination on a multi-link source and can distinguish sources that share a URL.

```yaml
featured:
  - source:
      clusterId: homelab
      kind: Ingress
      namespace: monitoring
      name: grafana
    url: https://grafana.example.net/
  - manualLink:
      clusterId: homelab
      namespace: autodocs
      configMapName: directory-links
      entryId: service-guide
    url: https://guide.example.net/
groupOrder:
  - observability
  - media-services
```

Accept one bounded YAML document with known fields. Validate the selector shape, required identity fields, absolute HTTP(S) URLs, and duplicate references or exact group values. Each feature entry uses one selector form. An unset file path enables ordinary directory behavior. A configured missing or invalid file yields default curation and a sanitized warning alongside otherwise successful directory data.

### Add curation metadata to the existing published read

Project curation from the same retained rows that back the published response. Emit optional one-based `featuredLinkRanks` (an exact URL to rank map on the owning source) and `directoryGroupRank` (an integer on sources in a ranked source-label group). Only matched references contribute metadata. Copy rows that receive metadata so concurrent requests and persisted snapshots keep the original inventory records. This preserves the response array and gives the browser settings and data in a single fetch.

Build **Featured** from the same filtered rows and safe endpoints as the rest of the directory, using the configured rank. Reuse the link display name and destination from inventory. Keep the original card, and count each source and URL using the existing ordinary projection. Treat Featured as a heading in the results rather than a new label group or sidebar filter. It appears with any Applications grouping mode whenever a feature matches the current filters.

In Applications under Source label grouping, sort scoped sections and sidebar choices with the same comparator. Listed groups precede unlisted groups, which retain their exact-value order. Put the synthetic Ungrouped fallback last. A literal group called `Ungrouped` is eligible for ranking. Ranks apply independently of the current filters, so a group keeps its relative position when it reappears.

### Reuse the active result set for quick search

Keep the existing query, matcher, and directory filters as the source of truth. Derive quick results from the final matching rows after the sidebar selection. Show a result for each distinct source/URL pair and a result that jumps to the source card when a matching source has no web destination. Render a bounded preview in existing source and endpoint order while the regular directory continues to surface every match. Include enough source and URL context to disambiguate equal titles or different destinations on the same source.

Focus search once on initial desktop entry without scrolling. On touch-first entry, a tap activates search and opens the software keyboard. Keep the input value and caret as the palette opens. Open quick results on a non-empty trimmed query while search is active. Arrow keys move the active result and Enter opens the selected web link using the existing safe new-tab behavior or focuses the matching source card. Escape clears the query and closes quick results while keeping search focused. Ctrl/Cmd+K continues to focus search outside editable controls. Render loading, unavailable, and no-match states from the active view, with the current request-version guard for any data transitions.

### Use the Floating palette

With an empty or whitespace-only query, keep the field in the directory header and leave the directory undimmed, even when the field has focus. In the empty desktop search field, use the muted placeholder “Just type. Look, no clicks.” Keep the field’s accessible label independent of the placeholder. Use “Search directory” on touch-first devices. The **Floating palette** opens above the dimmed directory when the active query contains its first non-whitespace character. Keep keyboard focus within the palette while it is open. On dismissal by Escape or clearing the query, return the field and focus to the directory header. Use the directory’s existing light and dark themes and fit the palette to the available space on narrow screens.

## Risks / Trade-offs

- A featured source can disappear, change URL, or move between views. Exact references become dormant and can render again if the same source/URL pair returns in Applications.
- A failed settings update can remove the curated positions temporarily. The directory still loads with normal ordering and displays a warning for the configured file.
- Quick results can go stale when filters or a request change. Recompute them from the same current projection and clear the active result when its identity leaves that projection.
- Automatic focus can interrupt browser scroll restoration or open a touch keyboard. Use one desktop focus attempt on initial entry and preserve focus during subsequent renders, modal use, and history navigation.

## Migration Plan

Deploy the new version with the optional file unset to use the existing group order. Mount and populate the file when ready to enable curation. Remove the variable or supply empty lists to return to the original directory order. Existing snapshots and browser preferences require no migration.
