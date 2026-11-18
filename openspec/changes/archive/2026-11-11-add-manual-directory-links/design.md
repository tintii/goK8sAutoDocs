## Context

The directory currently derives entries from Ingresses, HTTPRoutes and eligible Services. `go/inventory.go` collects per-target partitions, adds cluster and label-derived classification, and serves published snapshots. `go/static/inventory-ui.js` matches one cluster/environment/datacentre value per resource; `directory-model.js` adds namespace/kind/search/favourites predicates and grouping. Browser source identities currently include the displayed cluster, namespace, kind and name.

Manual destinations do not necessarily belong to a Kubernetes cluster. Reusing the current decoration and identity paths unchanged would make a centrally stored Confluence link appear to belong to the hosting cluster and would reset personal assignments when declared relevance changes. The legacy homepage `links` field is not a directory source and will not be repurposed.

## Goals / Non-Goals

**Goals:** Declarative arbitrary web links; normal directory presentation; optional independent scope; explicit shared versus unknown semantics; stable preferences; read-only collection with failure isolation and truthful freshness.

**Non-Goals:** CRDs/operators, browser editing, creating routing/DNS resources, destination probes, workload/version synthesis, automatic migration of legacy homepage links, multi-value scope, extra shared-link toggles or a separate bookmarks page. Publishing and deployment are separate work.

## Decisions

### 1. Discover labelled ConfigMaps through the existing collector

Use the exact selector `k8sautodocs.io/link-source=true` in each configured target. Require `data["links.yaml"]`; unrelated data keys are ignored. No publication-label requirement is added: the explicit source marker is publication intent for its valid link document. Keep source data separate from existing generic ConfigMap resource collection.

This supports both a central list and declarations next to workloads without additional volumes or application restarts. Existing chart ConfigMap support and list permission suffice. A mounted central file is simpler but does not support distributed declarations through the existing target registry; a CRD adds installation, schema lifecycle and link-specific RBAC benefits that this first version does not need. Revisit a CRD if those benefits become requirements.

Proposed declaration:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: directory-links
  namespace: k8sautodocs
  labels:
    k8sautodocs.io/link-source: "true"
data:
  links.yaml: |
    version: 1
    links:
      - id: confluence
        title: Confluence
        url: https://wiki.example.net/
        group: documentation
      - id: proxmox
        title: Proxmox
        url: https://pve.example.net:8006/
        group: infrastructure
        scope:
          datacentre: home
      - id: media-runbook
        title: Media runbook
        url: https://wiki.example.net/runbooks/media#recovery
        group: media
        scope:
          clusterId: homelab
          namespace: media
          environment: production
          datacentre: home
```

`version` and `links` are required. Each link requires nonempty string `id`, `title` and `url`; `group` and `scope` are optional. IDs are case-sensitive, unique within a document and independent of list order. Present optional scalar values must be nonempty strings; omitted scope and `scope: {}` both mean fully shared. Nulls, arrays, duplicate YAML keys, unknown fields, multiple documents and unsupported versions are invalid. Bound parsing to 256 KiB and 500 entries per document; do not truncate oversized documents. A valid `links: []` explicitly clears the document.

Only absolute HTTP(S) URLs with a hostname and no embedded credentials or control characters are supported. Reject backslashes and invalid ports. Preserve valid path, query, fragment and explicit port in the clickable URL. Treat titles/groups as text. Do not fetch destinations during collection. Diagnostics identify the document and offending field/entry without echoing its raw body or URL query. URL validation limits navigation syntax; it is not a destination allowlist.

### 2. Normalize relevance independently of provenance

Create a manual-link directory source with explicit source type, stable source identity, title, endpoint, group metadata, optional relevance fields and declaration provenance. Use `kind: manual-link` internally and display Manual link. The serializer must distinguish omitted dimensions from ordinary missing discovered-resource metadata; the browser must not infer sharedness solely from absent fields.

Map `scope.clusterId` to relevance cluster ID and resolve its display name against the current registry. An unresolved ID retains its exact restriction, displays the ID and reports a diagnostic; it must not register a target, guess a name or silently become shared. This accommodates unavailable/dynamically enumerated clusters. A namespace is a relevance string, not a request to read or verify that namespace. Without cluster scope, a declared namespace applies across clusters.

Explicit environment/datacentre values reuse existing equality/display normalization. When no rule exists for a dimension, use exact case-sensitive identity and preserve display spelling. Do not require label keys merely to express a manual value. Include normalized manual values in scope choices, even when no discovered resource has them. Preserve-mode spelling selection uses the same deterministic pool as discovered values. Do not inherit scope or group from ConfigMap labels, its namespace or its source cluster.

Keep declaration cluster ID, namespace, ConfigMap name and entry ID in provenance. Display relevance on cards and make provenance available in source details. An unscoped card shows no empty scope rows. Group metadata uses the existing group-label projection with exact case-sensitive identity; omission gives the existing Ungrouped group.

### 3. Omitted dimensions are unrestricted only for manual links

For cluster, namespace, environment and datacentre, a manual entry matches when the filter is unset, the entry omits that dimension, or its declared canonical value equals the selected value. Apply all declared restrictions with AND. Unspecified dimensions also match an explicit unknown/unclassified filter selection: they impose no restriction, but are never reclassified as unknown.

| Declaration | Selection | Result |
|---|---|---|
| No scope | Any cluster/environment/datacentre/namespace | Visible |
| Datacentre home | Datacentre home, environment production | Visible |
| Datacentre home | Datacentre office | Hidden |
| Cluster homelab | Cluster another | Hidden |
| Namespace media, no cluster | Namespace media in any cluster | Visible |
| Namespace media | Namespace infrastructure | Hidden |

Kind, search, selected group and favourites retain their usual restrictions. Selecting Ingress excludes Manual link. Sharedness does not bypass these controls. Existing discovered-resource matching and unknown fallbacks remain unchanged. The directory and overview use the same matching helper; Versions keeps its existing workload rules and receives no manual rows.

### 4. Group shared dimensions explicitly

Add typed shared buckets for cluster, namespace, environment and datacentre, displayed as Shared across clusters/namespaces/environments/datacentres. Keep these separate from unknown/unclassified buckets, custom unassigned groups and real values with identical display text. Composite group paths represent each omitted dimension with a typed shared component; for example, a production link without cluster scope appears under `production / Shared across clusters`.

Concrete groups sort first, shared buckets next, unknown buckets last; preserve the established ordering among concrete and unknown groups. Namespace groups retain cluster-qualified identity when cluster is supplied; a declared namespace without a cluster uses a distinct shared-cluster identity. A supplied cluster without a namespace groups under that cluster / Shared across namespaces. Never duplicate a shared card into every concrete group. Label/custom/kind/none grouping works as for other sources. Group selection remains an ordinary restriction, so selecting a particular cluster group does not include the separate Shared across clusters group even though a cluster scope filter permits shared links.

### 5. Stable identities use the declaration, not relevance

Use a typed tuple equivalent to `[manual-link, declarationClusterId, declarationNamespace, configMapName, entryId]`. Do not use ConfigMap UID, list index, title, URL or declared scope for source identity. Link favourites append the exact current URL to that source identity. Existing discovered identities remain unchanged.

Title/group/scope edits retain personal assignments and independent URL stars. A changed URL is a different link favourite, consistent with existing semantics. Moving/renaming the declaration or changing entry ID produces a new source. Equal URLs in different documents remain distinct. Group favourites follow current membership and may change as a consequence of scope/group edits; they are not frozen per-source favourites. Persisted preferences remain references, never a source of destinations.

### 6. Refresh and retain by source provenance

Add a manual-link component per collection target, with last-good document results keyed by namespace/name. List failure retains the previous component and timestamps. A successful list handles documents independently: valid documents replace previous results, invalid documents retain only their own last-good results with diagnostics and original success times, and newly invalid documents contribute no cards. Other valid documents still update. Successful deletion, marker removal or a valid empty list removes prior entries. Invalid documents do not silently withdraw all links or prevent unrelated declarations from updating.

Partition metadata must expose mixed outcomes and per-document ages; a successful list alone cannot claim all documents are fresh. Snapshot persistence stores normalized entries, provenance, document identities/times and sanitized diagnostics, not raw ConfigMap bodies. Absent manual components in older caches are never-successful, not successful empty data. Preserve existing target-removal and connection-identity protections using declaration provenance.

The published directory merges manual rows; Discovered, Versions, workload status and the Kubernetes filtered-resources projection do not gain synthetic manual resources. All reads remain snapshot-only. Optional manual fields are additive to directory responses.

Freshness selection must consider declarations contributing to the selected relevance scope, even when stored in another cluster. A shared link stored in cluster A and viewed under cluster B carries A's collection age and failures. Never-successful manual collectors must remain visible as incomplete discovery because their unseen links could be shared. Filter choices and counters derive from relevance, not storage namespaces; shared links do not create fictitious clusters or namespaces. Manual values in the shared scope picker can legitimately yield no matching workloads on Versions.

## Risks / Trade-offs

- Shared links may remain visible under a narrow scope → show only actual relevance and retain ordinary search/kind/group/favourites restrictions; document the intentional semantics.
- Existing code conflates provenance, relevance and identity → add an explicit source discriminator and stable identity adapter, and regression-test discovered resources and existing preferences.
- Invalid edits can leave old destinations visible → retain them visibly stale with document diagnostics; a valid empty list or deletion remains an explicit removal mechanism.
- Multiple declarations can point to the same URL → retain independent ownership/identity rather than silently deduplicating across sources.
- Reference clusters can be temporarily unavailable → preserve the declared restriction and expose unresolved names without broadening scope.
- ConfigMaps have no API-level validation of embedded YAML → strict bounded parsing and visible per-document diagnostics; CRD remains a possible future alternative.

## Migration Plan

1. Implement and test backend normalization, collection, snapshots and status alongside shared browser identity/filtering/grouping changes.
2. Verify installations with no marked ConfigMaps retain existing views, classification and preference identities. Exercise cache upgrade/rollback compatibility; incompatible caches can be discarded with a diagnostic and recollected.
3. Add examples and user documentation in the README or the manual, depending on the state of `publish-user-manual-on-github-pages`; update release notes during implementation.
4. After normal release validation, a separate authorized deployment change can add the image and ConfigMap declarations through existing Helm values. No live resources are created by this proposal.
5. Rollback to the previous application release removes manual directory support while leaving ConfigMaps inert. Existing Kubernetes endpoint discovery remains available; manual preference records remain dormant.

## Open Questions

None blocking this proposal. Singular scope values, ConfigMap storage and matching shared links across all omitted dimensions are the proposed first-version decisions. Supporting a bounded list of clusters/environments or migrating declarations between storage backends without changing identity would require a subsequent design.
