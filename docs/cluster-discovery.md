# Connecting clusters

This manual follows `master`. Rancher discovery is available in **v0.6.0-RC1**; manual directory links are **unreleased**. Check the [release history](https://github.com/tintii/goK8sAutoDocs/blob/master/CHANGELOG.md) before using these features with an older image.

Use direct targets for explicit connections and Rancher sources for dynamic management inventory. Use bindings to represent the same cluster through more than one connection while keeping one authoritative primary.

## Configure direct clusters and classification

Set `AUTODOCS_CONFIG` to a mounted YAML file. Configuration is read at startup;
registry/rule edits require a controlled restart. Credential files reload at the
start of every refresh. An example registry is:

```yaml
clusters:
  - id: primary-cluster
    name: Primary cluster
    inCluster: true
  - id: another-cluster
    kubeconfig:
      path: /etc/autodocs/clusters/another-cluster/kubeconfig
      context: autodocs
    labels:
      location: site-a
legacyClusterID: primary-cluster
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

## Rancher cluster discovery

Optional `ranchers` sources discover management Clusters through supported
RK-API (`/apis/management.cattle.io/v3/clusters`) and collect through
`/k8s/clusters/<management-id>`. Verified against Rancher **2.15.2**; an
unsupported API or denied enumeration fails that source without legacy-v3
fallback. Imported cluster coverage was verified; provisioned coverage uses the upstream management/provisioning controller contract and sanitized fixtures. This does not guarantee compatibility with every Rancher version.

Use a **Rancher API key with no cluster scope**, attached to an identity with
management inventory read permissions. An ordinary cluster-scoped kubeconfig
may collect through its downstream proxy while enumeration returns 403.
Rancher 2.15.2 gates management Cluster requests on `get` of
`clusters.management.cattle.io` named `local`; the identity also needs `list`
on management Clusters. These are the source-derived minimum enumeration
operations, not a claim that a minimum-role credential was exercised live.
Proxy authorization also requires access to each selected management Cluster
and its downstream resources. Kubernetes list authorization does not provide per-object filtering: use an
identity permitted to read the management inventory, then use explicit
exclusions to control collection. Exclusions and browser filters do not change
Rancher authorization. Downstream access requires the existing read-only
[collector permissions](installation.md#kubernetes-permissions), plus `get` on Namespace
`kube-system` for optional identity observations. Secret reads and resource
writes are not required.

For an existing direct cluster, retain its ID and direct primary when adding a Rancher secondary. This example uses `primary-cluster`:

```yaml
clusters:
  - id: primary-cluster
    name: Primary cluster
    inCluster: true
legacyClusterID: primary-cluster
ranchers:
  - id: management-rancher
    name: Management Rancher
    kubeconfig:
      path: /etc/autodocs/rancher/kubeconfig
      context: rancher
    excludeClusterIDs: []
clusterBindings:
  - id: primary-cluster
    name: Primary cluster
    connections:
      - direct: primary-cluster
      - rancher: management-rancher
        clusterID: local
    primary:
      direct: primary-cluster
```

A Rancher source's kubeconfig points to the **Rancher HTTPS origin**, without
a downstream `/k8s/clusters/...` path. Use a self-contained embedded bearer
token and an explicit context; embed CA data if system roots are insufficient.
Client certificates, insecure TLS, proxy URLs, executable/interactive helpers,
external credential files and impersonation are unsupported for sources. For
example, the mounted Secret's `kubeconfig` key contains:

```yaml
apiVersion: v1
kind: Config
clusters:
  - name: rancher
    cluster:
      server: https://rancher.example.com
users:
  - name: rancher
    user:
      token: REPLACE_LOCALLY
contexts:
  - name: rancher
    context:
      cluster: rancher
      user: rancher
current-context: rancher
```

Mount an existing credential Secret as a read-only **directory**, without
`subPath`, so Kubernetes can rotate it. This illustrative pod-spec fragment
requires no new chart interface:

```yaml
volumes:
  - name: rancher-credentials
    secret:
      secretName: autodocs-rancher
      optional: true
      defaultMode: 0444
containers:
  - name: autodocs
    volumeMounts:
      - name: rancher-credentials
        mountPath: /etc/autodocs/rancher
        readOnly: true
```

The non-root process must be able to read the mount. Missing mounts and invalid
replacement credentials leave that source unavailable; independent direct
connections keep working. Each refresh reads one credential version for all
source/proxy requests. Tokens never enter snapshots or public diagnostics.
Credential issuance and rotation remain external. Source/binding configuration
changes require restart. Creating Secrets, publishing an image and changing
GitOps deployment pins are separate deployment work.

Explicit configuration is authoritative: Rancher-only configuration is valid,
including a successful empty discovery, with no implicit local connection.
Each complete listing adds newly visible clusters and removes absent members.
Partial pages, timeouts, invalid responses and budget overflow retain compatible
previous membership/inventory as stale and skip that source's proxy collection.
A failed source never prevents a bound direct primary from refreshing.

A connection reference contains either `direct`, or `rancher` and `clusterID`.
Each belongs to at most one binding; each binding requires exactly one primary
among its members. Unknown sources, invalid references, conflicting assignments,
excluded bound members and canonical collisions are rejected. A temporarily
undiscovered member is valid and shown unavailable. Explicit bindings remain
visible as configured canonical targets even when all their members are absent;
unbound discovered targets leave active inventory when absent from a complete list. Bound primaries do not
fail over or union permission-dependent inventories. Secondary connections run
only identity probes. A missing primary remains unavailable even when a
secondary is healthy. Labels come from the binding or its owning direct target;
Rancher labels are not imported automatically.

Equal `kube-system` Namespace UIDs in the same refresh produce a **possible
overlap** warning. Cloned/restored clusters can share this marker; no automatic
binding follows. Denied, missing or timed-out probes report identity checks
unavailable without blocking collection. Contradictory bound observations
produce a conflict warning while the selected primary remains authoritative.
Scope & updates → Collection details distinguishes source enumeration, primary
and secondary connections, identity diagnostics and collection failures.

Unmapped direct IDs remain unchanged. Unmapped Rancher IDs are
`rancher:<base64url(source-id)>:<base64url(management-id)>`, without padding.
Bindings involving a direct connection retain a member's existing direct ID;
for Rancher-only migration, retain the chosen member's displayed canonical ID.
Changing the primary preserves canonical resource/link identities and favourites
for unchanged URLs, but invalidates cached inventory from the previous primary.
Preferences belonging to a discarded duplicate ID are not merged automatically.
An observed change of underlying cluster marker also discards old inventory.

Discovery budgets are 10 sources, 100 listed clusters per source (including
excluded entries), 200 discovered/configured connections overall, 100 pages,
50 requested items/page, 4 MiB/page and 16 MiB/source. Enumeration has a 15-second
source deadline and 10-second HTTP timeout. Identity probes have a 2-second
connection deadline and a shared 20-second budget per generation. Collection
uses the existing per-cluster timeout, sequentially; large inventories can
therefore take substantially longer than the configured refresh interval.
Overflow fails the listing; it never publishes a truncated membership.

Direct-only deployments continue writing version-1 snapshots. Rancher/binding
configuration writes version 2 with sanitized membership, primary provenance
and compatibility hashes. Unchanged, unbound direct version-1 targets restore
on upgrade. Version-2 targets restore only against matching source/binding and
collection configuration, and are explicitly stale until refreshed. Removed,
excluded or reassigned connections cannot resurrect unrelated cached rows.
To roll back to the previous binary, remove Rancher/binding configuration and
use a compatible saved version-1 cache or an empty cache; the previous binary
cannot read version 2. Directory, overview and versions retain canonical IDs.

## Remote cluster Secret contract

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
permissions documented in [installation](installation.md#kubernetes-permissions).

Use a directory mount without `subPath` so Kubernetes can project rotations.
Allow the non-root process to read it. An optional missing Secret is reported
as unavailable target credentials while other clusters continue. The hosting
ServiceAccount needs no Secret-read API permissions; each remote identity needs
the read-only collector permissions in [installation](installation.md#kubernetes-permissions). Bitwarden-backed ExternalSecrets
can supply the whole kubeconfig text as the `kubeconfig` key. Credential renewal
belongs to the supplier: AutoDocs reloads replacement files and does not renew
expired tokens/certificates. A malformed replacement is reported without
silently authenticating with superseded credentials.

## Default Kubernetes connection

Without `AUTODOCS_CONFIG`, the application uses in-cluster ServiceAccount
authentication when available, otherwise the current local kubeconfig context.
With an explicit registry, each target uses its declared in-cluster connection
or independently mounted kubeconfig and selected context; it never falls back
to another target. Remote credentials must meet the self-contained HTTPS
contract described above.
