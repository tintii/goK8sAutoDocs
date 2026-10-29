# goK8sAutoDocs

Experimental Kubernetes inventory and endpoint directory. Current source release: **v0.5.1**.

See [CHANGELOG.md](CHANGELOG.md) for release history and upgrade notes.

A Kubernetes resource monitoring and documentation tool that provides consolidated access to cluster resources, versions dashboard, and auto-documentation capabilities.

## Features

- **Versions Dashboard**: Monitor deployment, statefulset, and daemonset versions across namespaces
- **Auto-Docs Dashboard**: Organise published Ingress and HTTPRoute endpoints into application groups, and find undeclared endpoints in Discovered
- **Personal Directory**: Switch grouping on the fly, create personal groups, star groups or individual links, and combine search with namespace/kind filters
- **Shared Interface**: Collapsible navigation, grid/list layouts and browser-saved light/dark themes across all three pages
- **Cluster Overview**: Configurable heading and MOTD in a neon hero, the original orbit graphic, and your starred directory favourites
- **Resource Filtering**: Dynamic filtering of Kubernetes resources by GVR (Group, Version, Resource)
- **Application Status**: A compact Running/Unavailable indicator beside the theme toggle on every page
- **Shared inventory**: Background discovery, coordinated updates and persistent startup snapshots
- **Multiple clusters**: Explicit independent in-cluster or mounted-kubeconfig targets, with environment/datacentre grouping and filters
- **Operational logging**: Structured JSON logs with configurable debug/info/warn/error levels, refresh outcomes and recovery context

## Prerequisites

- Go 1.24.0 or higher (module toolchain: Go 1.24.5)
- Kubernetes cluster access (via kubeconfig or service account)
- Access to the following Kubernetes resources:
  - Deployments, StatefulSets, DaemonSets
  - Services, ConfigMaps
  - Ingresses
  - Pods
  - Gateway API v1 HTTPRoutes and parent Gateways when installed
  - Namespaces when Gateway listeners select allowed namespaces by label or namespace classification is configured

## Installation

### From Source

```bash
git clone https://github.com/tintii/goK8sAutoDocs.git
cd goK8sAutoDocs/go
go mod tidy
go build -o goK8sAutoDocs .
```

### Container image

Build from the repository root so the image includes the full Go module and
dashboard assets:

```bash
docker buildx build --platform linux/amd64 --load -t k8sautodocs:ci .
python3 scripts/smoke-container.py k8sautodocs:ci
```

The multi-stage build pins Go 1.24.5 and a distroless runtime by digest.
The runtime serves on 8080 as UID/GID 65532, includes CA certificates and
supports a read-only root filesystem. Its working directory is `/app`, with
the binary and `static/` tree together. The smoke check verifies the exact
packaged HTML/JS content using a temporary fixture kubeconfig, without real
cluster credentials. Kubernetes deployments use a projected ServiceAccount
token and the application's in-cluster authentication path.

GitHub Actions validates pull requests, `master` pushes, `v<semver>` tags
and manual runs. Required checks include Go tests, race tests, browser tests
with the Ubuntu runner's Chrome executable (absence fails CI), and the image
smoke check. Only successful pushes to `master` or valid release tags publish
to `ghcr.io/tintii/k8sautodocs`. The separate publication job loads and pushes
the exact tested image artifact using `GITHUB_TOKEN`; it does not rebuild or
deploy. Actions are pinned by commit and package write permission is confined
to that job.

Images have a full-commit `sha-<commit>` tag and OCI source/revision/version
labels. Releases also receive their exact `v<semver>` tag; prereleases are
accepted, build metadata and numeric prerelease identifiers with leading
zeroes are rejected. No moving `latest` tag is published. Use the digest from
the **Publish the exact tested image and report its digest** job summary to
pin a deployment; publication alone does not update GitOps values.

Push a `v<semver>` Git tag to publish a versioned image after validation. The
workflow consumes existing Git tags; it does not create tags or GitHub Release
entries. Release images currently target `linux/amd64`.

After the first publication, set the package visibility to public if cluster
pulls should be anonymous. [GHCR defaults new packages to private](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry).
Verify the selected digest can be pulled for `linux/amd64` before rollout.
If keeping it private, supply a read-only pull credential through your secret
manager and reference it with `imagePullSecrets`; never embed it in the image
or commit it in values. A package previously created outside this repository
may need its Actions repository access configured before `GITHUB_TOKEN` can
push to it.

The homelab integration, chart publication steps, anonymous pull check and
rollback commands live in the sibling
[GitOps runbook](../homelab-gitops/docs/k8sautodocs.md).

## Configuration

### Environment Variables

- `AUTODOCS_CONFIG=/etc/autodocs/discovery/autodocs.yaml` - Optional registry/classification/refresh/cache configuration, loaded at startup
- `LOG_LEVEL=info` - Minimum application log level: `debug`, `info`, `warn` or `error` (default `info`)
- `DEBUG=true` - Legacy debug fallback when `LOG_LEVEL` is unset or blank
- `HOMEPAGE_CONFIG=/etc/autodocs/home/home.yaml` - Optional home page content file, reread on each request

The server listens on port 8080.

### Home page content and status

The overview places the MOTD inside its hero alongside the original orbit
graphic, keeping the green/neon palette. Start with
[config/home.example.yaml](config/home.example.yaml):

```yaml
title: Cluster overview
description: Application links and container versions.
environment: Homelab
motd: |
  Maintenance window: Saturday 10:00–11:00.
  Backups run as usual.
footer: AutoDocs · Homelab
```

All fields are optional; omitted fields use defaults. An empty `motd` hides the
message. The title must be non-empty. Content is plain text, including multiline
MOTD text; no HTML, JavaScript or template syntax is evaluated. Unknown fields, malformed YAML,
multiple documents and files over 64 KiB report a configuration error. Keep
secrets out of this file: its content is available to everyone who can view the UI.

For Kubernetes, put that YAML in a ConfigMap key named `home.yaml`, mount the
**whole ConfigMap directory** read-only at `/etc/autodocs/home`, and set
`HOMEPAGE_CONFIG=/etc/autodocs/home/home.yaml`. Do not use `subPath`, which
prevents projected file updates. No extra RBAC or Reloader is needed. The app
rereads the file per request; open pages refresh content and status every 60
seconds while visible or when returning to the tab. The shared Update now control refreshes inventory; home content keeps its own automatic reread. Allow
Kubernetes time to project an updated ConfigMap. With GitOps, edit the owning
Helm values so reconciliation preserves the content. The homelab wiring is in
the sibling [values/k8sAutoDocs.yaml](../homelab-gitops/values/k8sAutoDocs.yaml).

Without `HOMEPAGE_CONFIG`, built-in content works normally. A configured but
unreadable or invalid file shows a notice and retains the last loaded content
in an open page, or defaults on first load; status and navigation remain usable.
The home page readiness probe remains independent of configuration and cluster
status.

The small header indicator beside the theme toggle reports **Running** when
AutoDocs responds to `/api/v1/health`, or **Unavailable** when that request fails.
Hover or focus it for an explanation. It checks application availability without
querying Kubernetes or inferring cluster/workload health. Inventory collection
age, shared updates and per-cluster diagnostics live in the Directory and
Versions sidebar submenu.

**Favourites** shows your existing directory stars across Applications and
Discovered. A starred group includes its current members; a starred link keeps
only that source's exact URL. Personal groups and unresolved group members use
the same directory rules. Destinations always come from the live API, never
saved preference strings. Removed sources disappear, while their saved stars
remain available if they return. Preferences update the home page immediately
and across tabs, without fetching for local group changes. The directory APIs
are requested only when stars exist, then refreshed every 60 seconds while
visible or on returning to the tab. Collection failures retain available live
matches and display a brief notice. With no stars, the section points to the
directory. Browser storage restrictions have the same behavior as the directory.

The legacy ConfigMap `links` field is still accepted by the API, but is not
rendered as personal favourites. Remove it from new configurations.

### Multiple clusters and label classification

Set `AUTODOCS_CONFIG` to a mounted YAML file. Configuration is read at startup;
registry/rule edits require a controlled restart. Credential files reload at the
start of every refresh. An example registry is:

```yaml
clusters:
  - id: homelab
    name: Homelab
    inCluster: true
  - id: another-cluster
    kubeconfig:
      path: /etc/autodocs/clusters/another-cluster/kubeconfig
      context: autodocs
    labels:
      location: pgt
legacyClusterID: homelab
classification:
  environment:
    labelKeys: [your.example/environment]
    sources: [resource, namespace]
    caseSensitive: false
    displayCase: upper
  datacentre:
    labelKeys: [your.example/datacentre, location]
    sources: [resource, namespace, cluster]
    caseSensitive: false
    displayCase: lower
refresh:
  interval: 5m
  cooldown: 30s
  clusterTimeout: 30s
cache:
  path: /var/cache/autodocs/snapshot.json
```

Replace example IDs/label keys with your own. IDs must be stable, unique and
nonempty. Each target requires exactly one connection method. The selected
kubeconfig context must exist; files are loaded independently, so their internal
context/user/cluster names can repeat. Explicit registry failures never fall
back to another cluster. A kubeconfig namespace does not restrict all-namespace
inventory. Missing/invalid target credentials affect that target, not startup of
the whole service. Invalid registry structure is a startup error.

Environment and cluster are independent: each can span several of the other.
Classification checks configured sources in order, then keys within each source.
`resource` means the Ingress/HTTPRoute or workload itself; `namespace` reads its
namespace; `cluster` reads configured target labels. Labels on arbitrary related
resources and cluster-name components are not inferred. Empty/missing values
fall through; missing labels never exclude a resource. Namespace read failures
produce diagnostics. Case folding applies to values, not label keys. No NPE,
PRD, CORE, location vocabulary, regex or prefix parsing is hardcoded.

`caseSensitive: false` makes `npe00` and `NPE00` one canonical identity.
`displayCase` independently accepts `preserve`, `upper` or `lower`; preserve
mode chooses a deterministic observed spelling. Distinct case-sensitive values
remain independent even if display formatting makes them look alike. Unknown
environments/datacentres have distinct fallback groups, separate from literal
labels with the same display name. Classification can be omitted entirely.
Refresh durations accept Go duration strings from `1s` through `24h`; registry
files are bounded to 1 MiB and kubeconfigs to 4 MiB.

Each page provides URL-restorable cluster/environment/datacentre scope. Filters
combine with AND, including existing query/kind/namespace/favourite filters.
Changing scope or grouping uses loaded inventory without querying Kubernetes.
The sidebar’s Scope & updates submenu on Directory and Versions shows actual
collection age, next scheduled attempt, shared refresh progress/cooldown and
per-cluster/component diagnostics. The overview has no inventory control panel;
background refresh, URL scope and favourites keep working. The
Running indicator measures AutoDocs availability independently.

### Sidebar navigation

The shared sidebar footer links to the [GitHub project](https://github.com/tintii/goK8sAutoDocs) in a new tab, including an accessible icon link in the collapsed desktop rail and the expanded mobile menu.

Use the sidebar chevron to collapse navigation into an icon rail on desktop,
or to show/hide navigation on mobile. Every destination keeps an accessible
name and a tooltip when its text is hidden. Mobile starts folded unless an
explicit preference is saved. Sidebar state and the Scope & updates submenu’s
open state persist in the existing browser preference envelope across pages,
reloads and tabs, without changing theme, favourites or table settings.
Collapsed controls are removed from keyboard focus. The overview retains its
compact availability/theme header, hero and favourites; inventory controls
appear only inside the Directory and Versions sidebar submenu.

### Remote cluster Secret contract

Use one `Opaque` Secret per target in the hosting namespace. Required key
`kubeconfig` contains a complete standard `apiVersion: v1`, `kind: Config`
document with its cluster endpoint, authentication and selected context. The
initial supported format is self-contained verified HTTPS, using an embedded
bearer token or embedded `client-certificate-data` plus `client-key-data`.
Embed `certificate-authority-data` when system trust is insufficient. External
credential-file paths, `exec`, auth-provider plugins, interactive login, insecure
TLS and impersonation are rejected. Extra unselected contexts are not scanned.
A file can be prepared from an existing trusted kubeconfig by flattening only
its intended context; store the result in the secret supplier, never Git.

For certificate authentication, the complete Secret has this structure. Replace
the placeholders with the values from your kubeconfig; never commit the populated
Secret. The namespace is the one hosting AutoDocs:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: autodocs-another-cluster
  namespace: k8sautodocs
type: Opaque
stringData:
  kubeconfig: |
    apiVersion: v1
    kind: Config
    clusters:
      - name: default
        cluster:
          server: https://your-cluster-api:6443
          certificate-authority-data: <base64 CA certificate>
    contexts:
      - name: default
        context:
          cluster: default
          user: default
    current-context: default
    users:
      - name: default
        user:
          client-certificate-data: <base64 client certificate>
          client-key-data: <base64 private key>
```

With `stringData`, the whole kubeconfig is plain YAML; its three `*-data` fields
retain their existing base64 values. If your secret supplier uses `data` instead,
the entire kubeconfig must additionally be base64-encoded as the `kubeconfig`
value. For bearer-token authentication, replace both client certificate/key
fields with `token: <bearer token>`; use exactly one authentication method.
For Bitwarden/External Secrets, store the complete kubeconfig text as the secret
value and map it into the target Secret's `kubeconfig` key.

The deployment references the Secret and mounts its `kubeconfig` key read-only;
the app registry references the mounted path and explicit context. The existing
workload chart supports native `pod.volumes` and `container.volumeMounts`:

```yaml
pod:
  volumes:
    - name: remote-cluster
      secret:
        secretName: autodocs-another-cluster
        optional: true
        items:
          - key: kubeconfig
            path: kubeconfig
container:
  volumeMounts:
    - name: remote-cluster
      mountPath: /etc/autodocs/clusters/another-cluster
      readOnly: true
```

The matching entry in the `AUTODOCS_CONFIG` registry explicitly selects the
context from that file, even when `current-context` is set:

```yaml
clusters:
  - id: another-cluster
    name: Another cluster
    kubeconfig:
      path: /etc/autodocs/clusters/another-cluster/kubeconfig
      context: default
```

Each file is loaded independently, so names such as `default` can repeat across
clusters. The API endpoint must be reachable from the AutoDocs pod, its TLS
certificate must validate, and the remote identity needs the discovery read
permissions documented below.

Use a directory mount without `subPath` so Kubernetes can project rotations.
Allow the non-root process to read it. An optional missing Secret is reported
as unavailable target credentials while other clusters continue. The hosting
ServiceAccount needs no Secret-read API permissions; each remote identity needs
the read-only collector permissions below. Bitwarden-backed ExternalSecrets
can supply the whole kubeconfig text as the `kubeconfig` key. Credential renewal
belongs to the supplier: AutoDocs reloads replacement files and does not renew
expired tokens/certificates. A malformed replacement is reported without
silently authenticating with superseded credentials.

### Kubernetes Connection

Without `AUTODOCS_CONFIG`, the application uses in-cluster ServiceAccount
authentication when available, otherwise the current local kubeconfig context.
With an explicit registry, each target uses its declared in-cluster connection
or independently mounted kubeconfig and selected context; it never falls back
to another target. Remote credentials must meet the self-contained HTTPS
contract described above.

## Usage

### Running the Application

Before running, if using your local k8s context make sure you set it up `kubectl config use-context <context-name>`

```bash
# Run locally
./goK8sAutoDocs

# Run with detailed collection and scheduling logs
LOG_LEVEL=debug ./goK8sAutoDocs

# For Kubernetes, use the container image and a read-only ServiceAccount.
# The sibling homelab-gitops repo supplies the Helm values/ApplicationSet.
```

### Accessing the Dashboards

- **Main Dashboard**: http://localhost:8080/
- **Versions Dashboard**: http://localhost:8080/versions-dashboard-page
- **Auto-Docs Dashboard**: http://localhost:8080/autodocs-dashboard-page

The Auto-Docs dashboard opens on **Applications**, grouped by source label by default, with an **Ungrouped** section for published resources without a group. Groups can span namespaces; each card retains its namespace and kind. Label groups are sorted by exact value, followed by the distinct Ungrouped fallback. Cards are sorted by namespace, kind, and name.

Select **Discovered** to see Ingresses and HTTPRoutes that have no publication label. Both tabs support the same grouping and personalisation controls. The selected tab is included in the URL, so direct links, reloads, and browser back/forward navigation restore it. Both tabs preserve unresolved entries and display collection warnings.

### Arrange the versions table

Columns fit the loaded contents by default. Short kind/type labels stay on one line; long names, images and digests use ellipsis with their complete value available on hover and for selection/copy. The table scrolls horizontally on smaller screens.

Drag a heading divider to resize its column. Double-click a divider to fit that column's full contents, or use **Fit columns** to restore all content-aware defaults. A focused divider supports Left/Right arrows, Shift for larger steps, and Home to fit; Escape cancels a drag.

Click **Namespace**, then **Resource** to sort by namespace and name within it. The first heading is primary, the second secondary; another distinct heading replaces the secondary key. Click an active heading again to reverse its direction, then again to remove it. Numbered arrows and the sort summary show the priority. **Clear sort** restores API order so the next heading starts a new primary sort. All eight columns support sorting, including container fields; numeric text sorts naturally, missing values stay last, and search retains every container of each matching resource.

Widths and sort keys are saved in this browser alongside the existing directory/theme settings, survive reload/navigation and update other open tabs. Fit columns and Clear sort reset only their respective table choices. No inventory data is stored in preferences.

### Personalise your directory

**Group by** switches either tab between source label, namespace, resource kind, personal groups, cluster, environment, datacentre, Environment → Cluster, Cluster → Environment and no grouping. Composite headings show the ordered group path. Namespace groups distinguish clusters; environment and datacentre groups intentionally span them. The grouping choice is shared between tabs and saved in the browser. Changes apply immediately to loaded data. **Layout** selects grid or list, and group headings can collapse sections.

Open **Personalise** to create, rename, reorder or delete personal groups. Group names are trimmed and must be non-empty and unique by exact spelling. Creating a group selects Personal groups mode. Assign published or discovered endpoints with the personal-group selector on each card; each source has at most one assignment. Unassigned applications have a fallback section. Empty groups stay in the editor until assigned. Rename preserves membership and stars; deletion unassigns members and removes that group's star. Personal grouping never edits Kubernetes labels or publishes a discovered resource.

Star a group heading in either tab to follow its current membership in the active view, including unresolved sources. Star a link to save only that source and exact URL; a group star and its member link stars are independent. **Favourites** shows their combined live matches within the active tab, even if you have since changed the displayed grouping mode. A group match shows its whole source card; a link-only match shows only the individually starred URLs. Identical URLs from different sources have independent stars. Missing sources and links remain saved but are never rendered from browser storage alone; their stars return if the exact identity reappears. Individual link stars follow a source moving between Applications and Discovered; group stars apply to the same group identity in either tab, always using only that tab’s loaded sources.

Search matches case-insensitive, whitespace-separated literal terms across source names, namespaces, kinds, source/personal groups, hosts, URLs, Gateway/listener details and related-resource metadata. All terms must match somewhere on the source. Namespace/kind filters and Favourites further narrow the active view. Search and filters stay selected across tab switches; a namespace with no matches remains visible and clearable. Use **Ctrl/Cmd+K** outside an editable control to focus search, **Escape** in search or **Clear** to clear its query. Loaded resource/namespace/URL totals stay separate from the matching resource and displayed-URL count.

### Browser settings

Theme, grouping, density, collapsed groups, custom ordering/membership and stars live in the versioned `k8sautodocs.preferences` localStorage document. They are personal to this browser profile and origin, with no account, backend storage or device sync. Clearing site data removes them. Open tabs on the same origin receive settings updates; simultaneous edits use the last successful write. Changing hostname/port uses a different store, and stable configured cluster IDs distinguish source/link/namespace identities across clusters. Version-1 source preferences migrate only to the configured original `legacyClusterID`; ambiguous records stay dormant. Version-2 preferences preserve theme, personal groups and table layout across the migration.

Source identity uses kind, namespace and name, plus the exact URL for individual link stars. Recreating a source with the same identity inherits its personal settings. Renaming a source or changing a URL creates a different identity.

**Reset layout** restores default grouping, density and collapse choices and removes personal groups/membership/order and their stars, while retaining source-label/namespace/kind group stars and independent link stars. **Clear favourites** removes all stars while preserving layout and theme. Blocked/full storage leaves controls usable for the session and displays a notice. Malformed saved data falls back to defaults; unsupported newer versions are preserved without overwriting them.

All pages share a light/dark theme. Existing `theme` preferences are imported; otherwise the system preference supplies the default until you choose one explicitly. Assets are packaged locally with no external font or icon dependency.

## API Endpoints

### GET /api/v1/health
Returns `{"status":"running"}` while AutoDocs is serving requests. It does not
query Kubernetes or read home-page configuration. Responses are not cached.

### GET /api/v1/home
Returns the home page content from `HOMEPAGE_CONFIG`, or defaults when unset.
Responses are not cached. Invalid or unreadable configured files return HTTP 500.

### GET /api/v1/service-status
Returns cached workload availability (`ready`, `attention`, `inactive`) and workload rows from the same inventory snapshot. `checkedAt` is the oldest contributing successful workload collection time. Collection failures retain available prior data with warnings; no usable initial data returns HTTP 503. This API never initiates discovery.

### GET /api/v1/cluster-info
Returns sanitized registry-backed cluster identities/display names, with a compatible single-target display name. Credentials, kubeconfig paths and server authentication details are not returned.

### GET /api/v1/versions-dashboard
Returns Deployments, StatefulSets and DaemonSets across configured clusters/namespaces, including main/init containers and additive cluster/classification metadata. Successful empty inventories return `[]`; no usable initial snapshot returns HTTP 503. Failed collection retains previous component data with original timestamps and diagnostics. GETs never query Kubernetes and responses use no-store with `X-AutoDocs-Generation`.

### GET /api/v1/discovery-status
Returns no-store shared generation, server time, progress, next automatic/manual times, configured cluster identities, available classification values and per-component last attempt/last success/error/warnings. A never-successful component has no successful timestamp. This endpoint does not trigger collection.

### POST /api/v1/discovery-refresh
Requests one server-owned refresh, returning HTTP 202 with the shared state immediately. Optional JSON `{"generation": 1}` supplies the viewer's observed generation. Concurrent requests, obsolete generations and cooldown clicks do not queue extra scans. Browser requests must be same-origin. Closing the browser does not cancel the job.

### GET /api/v1/auto-docs-dashboard
Returns an array of Ingress and HTTPRoute entries selected by `k8sautodocs.io/publish=true` labels. Entries include `name`, `namespace`, `kind`, `hostnames`, `labels`, `annotations`, `relatedResources`, `endpoints`, and `issues`. Ingress entries retain the existing `kind: "ingress"`; HTTPRoute entries use `kind: "HTTPRoute"`.

The optional `view` parameter selects the resource set:

- Omitted, empty, or `view=published`: published resources, preserving the default API behavior.
- `view=discovered`: resources with no `k8sautodocs.io/publish` label key, using the same array schema and URL resolution.
- Unknown or repeated `view` parameters return HTTP 400.

For example, `/api/v1/auto-docs-dashboard?view=discovered` supplies the Discovered tab. `/api/v1/filtered-resources` remains published-only.

Each `endpoints` item includes `url`, `host`, `scheme`, and `port`. HTTPRoute endpoints also identify their `gateway` (`name` and `namespace`) and `listener`. Empty collections are JSON arrays. Per-resource `issues` explain unresolved destinations or unknown/stale controller status.

Collection failures preserve available entries and are reported as a JSON array of strings in the `X-AutoDocs-Warnings` response header, which the dashboard displays. A missing HTTPRoute API is normal on an Ingress-only cluster.

### GET /api/v1/filtered-resources
Returns metadata for supported GVRs selected by the same publication label. HTTPRoutes are included when their API is available; partial collection warnings use `X-AutoDocs-Warnings`.

## Resource Types Supported

### Versions Dashboard

- Deployments
- StatefulSets
- DaemonSets

### Auto-Docs Dashboard

- Ingresses (`networking.k8s.io/v1`)
- HTTPRoutes (`gateway.networking.k8s.io/v1`)

Services and Pods appear as related resources. HTTPRoutes also show parent Gateways; existing Ingress relationship discovery includes Deployment and ConfigMap context. Gateways are dependencies rather than separately published directory entries.

### Filtered Resources

- Deployments
- StatefulSets
- DaemonSets
- ConfigMaps
- Services
- Ingresses
- HTTPRoutes

## Publication and grouping labels

Add this metadata to each Ingress or HTTPRoute you want to document:

```yaml
metadata:
  labels:
    k8sautodocs.io/publish: "true"
    k8sautodocs.io/group: media
```

Only the exact string `"true"` publishes a resource in Applications. The optional group label assigns one group to the whole source resource, including all its resolved URLs. Missing or empty group values use Ungrouped. Group names preserve spelling and case; `Media` and `media` are separate groups.

| Publication label | Group label | Dashboard location |
|---|---|---|
| `"true"` | Non-empty | Applications → named group |
| `"true"` | Missing or empty | Applications → Ungrouped |
| Absent key | Any | Discovered |
| `"false"` | Any | Hidden from both tabs |
| Present with any other value, including empty or `"True"` | Any | Excluded from both tabs |

The absence of the publication key is significant: an empty value does not qualify for Discovered. A group label alone never publishes a resource. Labels are read from the Ingress or HTTPRoute itself; they are not inherited from Gateways, namespaces, workloads, or Argo Applications. A literal group named `Ungrouped` remains distinct from the fallback section, whose description identifies applications without a group.

Kubernetes holds the live publication state. With GitOps, declare labels in the resource's owning Helm/Git configuration so reconciliation preserves them. The application only reads Kubernetes; its personal-group editor changes browser presentation only, with no publication database or Kubernetes metadata editing.

### URL derivation

- Ingress URLs use concrete rule hosts and matching TLS host declarations, with standard HTTP/HTTPS ports.
- HTTPRoute URLs use the concrete intersection of route and listener hostnames, the compatible Gateway listener protocol, and its external port. Non-default ports are included. Backend Service ports do not determine browser URLs.
- Gateway parent namespaces, listener names/ports, allowed route kinds, and allowed namespaces are respected. Namespace selectors require read access to namespace labels.
- Wildcard-only or unresolved endpoints remain visible with an explanation and no invented link. Unknown/stale attachment status is reported; current rejected attachments do not supply links.
- Links open the application origin in a new tab with `target="_blank"`. Individual path, header, and method matching is outside the directory's scope; links describe configuration and do not guarantee endpoint reachability.

### Migration from annotations

Move publication intent from the old `k8sautodocs.io/publish` annotation into `metadata.labels` before upgrading. Annotation-only resources appear in Discovered, and are excluded from Applications and filtered-resources. `k8sautodocs.io/fqdn-override` has been removed; neither annotations nor labels override destinations. Existing legacy annotations can be deleted, and retaining them has no effect.

### Kubernetes permissions

Use read-only RBAC. Existing dashboards require `list` on Deployments, StatefulSets, DaemonSets, ConfigMaps, Services, Ingresses, and Pods, plus `get` on referenced Services. Gateway API adds these ClusterRole rules when enabled:

```yaml
- apiGroups: [gateway.networking.k8s.io]
  resources: [httproutes]
  verbs: [list]
- apiGroups: [gateway.networking.k8s.io]
  resources: [gateways]
  verbs: [get]
- apiGroups: [""]
  resources: [namespaces]
  verbs: [get]
```

Namespace reads support listener namespace selectors and any configured namespace classification. Discovery spans namespaces, including cross-namespace Gateway and backend references. No Secret contents or Kubernetes write permissions are needed for endpoint discovery. Backend references describe associations; Auto-Docs does not validate ReferenceGrants itself.

## Caching

A single process-owned coordinator refreshes all targets sequentially at startup,
every five minutes by default, or through shared Update now controls. Each target
has a bounded timeout. Manual requests coalesce globally and have a thirty-second
cooldown by default; automatic scheduling resumes from run completion. Browser
polling reads AutoDocs only. Inventory GETs immediately project the last
published snapshot while collection runs. Published/discovered classification
uses the same source partition; filtered resources and versions retain their
own projection formats. This is an application generation, not a transactional
Kubernetes snapshot.

Successful source partitions replace prior contents, including empty results.
Failed partitions retain prior successful data/timestamps and diagnostics while
healthy clusters/partitions update. Optional absent Gateway API is supported;
forbidden/unreachable APIs are failures. Reading cached data, restoring a file,
checking liveness or completing a failed run never resets collection age.

With `cache.path`, restore a compatible versioned JSON snapshot immediately,
then refresh in the background. Snapshots contain derived inventory, metadata
and timestamps; they omit credentials, kubeconfigs, raw lists and ConfigMap
bodies. Files are bounded to 32 MiB and replaced atomically with mode 0600.
Removed/reconfigured targets and incompatible classifications are excluded.
Corrupt files trigger fresh discovery; write failures leave memory serving
available with a cache warning. Deleting the file forces a cold startup.

Mount a persistent writable cache volume for the non-root UID while keeping the
root filesystem read-only. A PVC survives ordinary pod replacement; claim
removal/reclaim behaviour depends on deployment/storage policy. Run exactly one
active process, including during upgrades: use one replica with `Recreate` or
another non-overlapping strategy. Multiple active replicas/shared writers are
unsupported. Rollback can retain the PVC; older releases ignore its file and
newer browser preference envelopes. Home content has its separate live reread.

## Development

The base [endpoint directory](openspec/specs/published-endpoint-directory/spec.md)
and [directory organization](openspec/specs/endpoint-directory-organization/spec.md)
specifications are extended by the implemented
[multi-cluster inventory](openspec/changes/add-multicluster-discovery-snapshots/specs/),
[dashboard personalisation](openspec/changes/redesign-autodocs-ui/specs/),
[versions table](openspec/changes/improve-versions-table/specs/) and
[application logging](openspec/changes/add-structured-level-logging/specs/application-logging/spec.md)
changes. Archived proposals are kept in the [OpenSpec archive](openspec/changes/archive/).

### Project Structure

```
goK8sAutoDocs/
├── go/
│   ├── main.go                # Startup, versions, and existing resource relationships
│   ├── autodocs.go            # Endpoint discovery and URL resolution
│   ├── resource_handlers.go   # Publication helpers and legacy isolated-handler tests
│   ├── inventory_config.go    # Registry, credential contract and classification
│   ├── inventory.go           # Shared coordinator, collection and persistent snapshots
│   ├── inventory_http.go      # Snapshot, freshness and shared update APIs
│   ├── inventory_test.go      # Multi-target, concurrency, recovery and credential fixtures
│   ├── home.go                # Configurable homepage, application health and workload status
│   ├── logging.go             # JSON severity filtering and safe response diagnostics
│   ├── logging_test.go        # Level, concurrency, quiet-reader and disclosure fixtures
│   ├── autodocs_test.go       # Kubernetes fixture and cache tests
│   ├── directory_views_test.go # Publication-state, view, and cache tests
│   ├── go.mod                # Go module file
│   └── static/               # Dashboard HTML, shared CSS/preferences, directory model and browser tests
├── scripts/                 # Container smoke check and same-origin headless browser fixtures
├── CHANGELOG.md             # Versioned release history and upgrade notes
└── README.md
```

### Building

```bash
cd go
go build -o goK8sAutoDocs .
```

### Testing

```bash
cd go
go test ./...
go test -race ./...
node --test static/*.test.cjs
```

The browser checks use headless Chrome/Chromium, a temporary loopback HTTP server and fixture responses, without contacting a cluster. They exercise real origin-scoped storage, reloads and tab updates, plus desktop/mobile light/dark layouts and contrast. Set `CHROME_BIN` to its executable if it is not found in a standard location. Browser checks are skipped when no executable is available; CI requires the executable. Node 22 or newer supplies the browser harness's built-in WebSocket client.

Optionally capture review screenshots outside the checkout:

```bash
AUTODOCS_SCREENSHOTS=/tmp/k8sautodocs-ui-review node --test go/static/autodocs-dashboard.test.cjs
```

## Security Considerations

- **CORS**: Currently allows all origins (`*`) - consider restricting in production
- **Authentication**: No built-in authentication - implement if exposing publicly
- **Read-only access**: Discovery spans namespaces within each configured target; browser filters change presentation, not Kubernetes permissions. Grant only the read permissions described above.
- **Runtime**: Run one active instance with read-only credential mounts, a non-root user and a writable cache volume when persistence is enabled.

## Troubleshooting

### Common Issues

1. **Connection Errors**: Verify kubeconfig or service account permissions
2. **Resource Access**: Ensure RBAC permissions for target resources
3. **Port Conflicts**: Check if port 8080 is available

### Logging

Application logs are one JSON object per line on **stderr**. The default level
is `info`; `LOG_LEVEL` accepts `debug`, `info`, `warn` or `error`, ignoring case
and surrounding whitespace. The chosen threshold includes all more severe
levels. An invalid nonblank value exits before discovery or HTTP startup.

```bash
LOG_LEVEL=debug ./goK8sAutoDocs
LOG_LEVEL=warn ./goK8sAutoDocs
```

`DEBUG=true` still selects debug logging when `LOG_LEVEL` is unset or blank.
An explicit level wins, so `LOG_LEVEL=warn DEBUG=true` emits warnings and errors.
Changes take effect on process restart.

| Level | What is logged |
| --- | --- |
| `debug` | Collection counts and duration, scheduling/coalescing/cooldown decisions, absent optional Gateway API and successful snapshot saves. |
| `info` | Startup summary, listener startup, refresh start/completion, snapshot restore and collection/storage recovery. |
| `warn` | Connection/collection failures, partial metadata or unresolved endpoints, retained stale inventory, and snapshot restore/write failures. |
| `error` | Invalid startup configuration, listener failure and internal handler failures such as invalid homepage configuration or response encoding. |

Records include `time`, `level`, `msg` and a stable `event`, with applicable
`cluster_id`, `component`, `generation`, `trigger`, `duration_ms` and counts.
Refresh triggers distinguish `startup`, `automatic` and `manual`; completion
counts distinguish successful and failed components. Recoverable cluster
failures produce warnings while other targets continue, and recovery events
appear only after a previous failure or partial result.

Successful health/status polling, static requests and snapshot reads stay quiet,
even at debug. Stale failures are reported during collection rather than on
every GET. A target connection failure produces one warning per run; partial
results use counts instead of one record per affected resource.

Logs use owned failure categories such as `collection timed out`, `access
forbidden` and `authentication rejected`. They exclude raw errors, full endpoint
URLs, query strings, request/response bodies, kubeconfigs, credentials, ConfigMap
values and object dumps at every level. Use cluster/component context together
with discovery-status diagnostics to check connectivity, credentials and read
permissions. This controls application logs; Kubernetes client-library verbosity
is unchanged.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

**GNU GPL v3.0**

The GNU General Public License is a free, copyleft license for software and other kinds of works. It ensures that the software remains free and open source, requiring that any derivative works also be licensed under the GPL.

## Release history

See [CHANGELOG.md](CHANGELOG.md) for all recorded releases and migration notes.
