# Manual directory links

This feature is **unreleased** on `master`; use a build containing it. See the
[release history](https://github.com/tintii/goK8sAutoDocs/blob/master/CHANGELOG.md).

Add arbitrary web destinations to **Applications** using a labelled ConfigMap in
any configured collection target. This is useful for a wiki, Proxmox console or
runbook that does not have a supported Kubernetes endpoint. No CRD, mounted file,
restart or endpoint publication label is needed. AutoDocs does not contact these
destinations or report their health.

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
        url: https://wiki.example.net/runbooks/media?mode=read#recovery
        group: media
        scope:
          clusterId: homelab
          namespace: media
          environment: production
          datacentre: home
```

Replace the example destinations and the `homelab` cluster ID with your values.
Manage this ConfigMap through your existing chart/GitOps workflow. The collector
uses the exact selector `k8sautodocs.io/link-source=true`; `True`, `false` and
unmarked ConfigMaps do not declare links. Only `data["links.yaml"]` is read for
this feature. Other data keys and ConfigMap classification/group labels do not
supply link metadata.

## Document schema and validation

`version: 1` and `links` are required. Each entry requires nonempty **string**
`id`, `title` and `url` fields. Quote numeric-looking strings. Entry IDs are
case-sensitive and unique within one ConfigMap; list order is irrelevant.
`group` is an optional nonempty string, with exact case-sensitive identity.
Omitting it puts the entry in the existing **Ungrouped** section.

`scope` is an optional map with independently optional nonempty string
`clusterId`, `namespace`, `environment` and `datacentre` fields. Omitted scope
and `scope: {}` both mean fully shared. Null values, empty optional values,
arrays in place of maps, unknown fields, duplicate YAML keys/IDs, multiple YAML
documents and unsupported versions invalidate the entire document. Documents
are limited to 256 KiB and 500 entries; oversized input is rejected, not truncated.
`links: []` is valid and intentionally clears the declaration.

URLs must be absolute HTTP(S) URLs with a hostname, without credentials,
backslashes, whitespace/control characters or invalid ports. Explicit ports must
be between 1 and 65535. Accepted URLs retain the exact path, query, fragment and
port in the link and its favourite identity. Titles and groups are plain text;
links open with `noopener noreferrer`. This is syntax validation, not a
restriction to trusted hosts. Do not put secrets in these published destinations.
Validation diagnostics identify the ConfigMap and field/index without echoing
raw documents or URL queries.

## Relevance and shared scope

Each supplied scope field restricts relevance. All supplied restrictions must
match the selected scope. Omitted dimensions impose no restriction, even when
an explicit unclassified/unknown selection is active. For example:

| Declared scope | Selected scope | Visible by scope? |
| --- | --- | --- |
| None | Any cluster, environment, datacentre or namespace | Yes |
| Datacentre `home` | Datacentre `home`, any other dimension | Yes |
| Datacentre `home` | Datacentre `office` | No |
| Namespace `media`, no cluster | Namespace `media` in any cluster | Yes |
| Cluster `homelab` | Another cluster | No |

Environment/datacentre values use the configured classification matching and
display rules, without requiring label keys. With no rule for a dimension,
identity is case-sensitive and display spelling is preserved. With a configured
case-insensitive rule, values share canonical identity; preserved display spelling
is selected deterministically from the same pool as discovered inventory.
Explicit values populate the scope pickers even without discovered peers.

Cluster IDs resolve against the current registry for display. An unresolved ID
remains an exact restriction, displays as its ID and has a diagnostic; it never
registers a target or becomes shared. Namespace scope does not require namespace
lookup or a cluster restriction. Storage cluster, namespace and labels never
supply relevance.

Search includes title, exact URL, group and declared relevance. Kind, query,
personal group selection and Favourites still restrict shared entries. Select
**Manual link** in the kind filter to see only these entries. Manual entries do
not enter **Discovered**, **Versions**, workload status or the Kubernetes
filtered-resources API. A manual-only environment may have no Versions results.

Grouping an omitted dimension creates **Shared across clusters**, **Shared
across namespaces**, **Shared across environments** or **Shared across
datacentres**. These buckets have separate typed identities from unknown values
and literal strings with the same names. Concrete groups sort before shared,
then unknown. Composite and namespace paths preserve shared dimensions, such as
`Production / Shared across clusters` or `Homelab / Shared across namespaces`.
A shared entry appears once, rather than in every concrete group. Selecting a
concrete group excludes the separate shared bucket even if a scope filter permits
it. Cards omit missing relevance rows; expand **Declaration source** to inspect
where an entry was collected.

## Personalisation and identity

The source identity is the tuple of `manual-link`, declaration cluster ID,
ConfigMap namespace/name and entry ID. Title, group, URL and relevance are not
source identity fields. Editing title/group/scope preserves personal assignments
and independent link stars; moving or renaming the ConfigMap or changing the
entry ID creates a different source. ConfigMap recreation with the same location
and IDs preserves identity.

Independent link favourites append the exact current URL. A URL edit leaves the
old star dormant and does not transfer it. Two declarations of the same URL have
independent favourites. Group stars follow current membership. Overview favourites
use the same identity and shared-scope matching. Saved preferences never supply
clickable destinations without a current snapshot entry.

## Refresh, failures, removal and permissions

Normal automatic refresh and **Update now** collect declarations through the
shared coordinator. API reads only use snapshots. Each ConfigMap is validated
independently: valid documents replace their entries; an invalid edit retains
only that document's previous entries and original success time, with a stale
notice and sanitized diagnostic. A newly invalid document contributes no links.
A failed ConfigMap list retains the entire previous manual component and its
timestamps. Other documents/targets can still update.

A successful list removes entries when a ConfigMap is deleted, its marker is
removed/changed, or its document becomes a valid `links: []`. Invalid YAML is
not a removal mechanism. Collection details expose per-document attempts and
success times. A shared link viewed under cluster B can still be stale because
its declaration in cluster A could not refresh. Never-successful collectors stay
visible as incomplete, because unseen entries might be shared.

The optional derived snapshot cache stores normalized entries, declaration
provenance, times and sanitized diagnostics, never raw ConfigMap documents. Old
caches without this component show it as never collected. Restore and target
identity/removal checks follow the declaration target, independently of relevance.

The existing discovery ClusterRole already lists ConfigMaps. Every collection
connection needs core API group `""`, resource `configmaps`, verb `list` across
the namespaces being collected (currently cluster-wide). No write permission,
watch or namespace lookup is added for manual declarations. The marker expresses
publication intent: restrict who can create/edit marked ConfigMaps using your
normal Kubernetes access controls.

[Back to manual](index.md)
