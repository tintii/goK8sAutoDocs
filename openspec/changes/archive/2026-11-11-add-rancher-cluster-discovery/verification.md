# Rancher contract investigation

Checked on 2026-11-11. Implementation resumed following the supplied management
credential. The accepted Rancher 2.15.2 contract and qualification limits are
recorded in design.md and the implementation results below. Early investigation
notes are retained as historical evidence, not current implementation status.

## Release and evidence

The sibling GitOps repository pins upstream Rancher chart `2.15.2`.
Inspected the upstream `v2.15.2` source archive; `git ls-remote` returned
`272e65580ca92b9dc51e0f0f0f9ae9d6d356e3b8` for that tag ref. This establishes
the source version examined, not the running homelab server version.

- [RK-API quickstart](https://ranchermanager.docs.rancher.com/api/quickstart)
  documents a Rancher-origin kubeconfig with bearer API-key authentication,
  client-go support and CA trust. It also warns that not all resources are
  available. This is documentation evidence, not a successful credential test.
- [Previous v3 guide](https://ranchermanager.docs.rancher.com/api/v3-rancher-api-guide)
  explicitly identifies the legacy API as unsupported. Neither `/v3/clusters`
  nor Steve `/v1` is an accepted fallback for this change.
- [Management Cluster types](https://github.com/rancher/rancher/blob/v2.15.2/pkg/apis/management.cattle.io/v3/cluster_types.go)
  define `spec.displayName` and `spec.importedConfig`.
- [Provisioning controller registration](https://github.com/rancher/rancher/blob/v2.15.2/pkg/controllers/provisioningv2/cluster/controller.go)
  generates management Cluster objects from provisioning Clusters and has a
  reverse legacy-to-provisioning path with eligibility conditions.
  [Import handling](https://github.com/rancher/rancher/blob/v2.15.2/pkg/controllers/provisioningv2/cluster/import.go)
  also creates a management Cluster with `ImportedConfig`. Management Cluster
  objects are therefore the candidate common resource; a provisioning-only
  listing is not established as complete. A provisioning name must not be
  substituted for its management `status.clusterName`.
- [Cluster route lookup](https://github.com/rancher/rancher/blob/v2.15.2/pkg/clusterrouter/lookup.go)
  recognizes `/k8s/clusters/{clusterID}/...`.
  [Proxy prefix removal](https://github.com/rancher/rancher/blob/v2.15.2/pkg/clusterrouter/proxy/proxy_server.go)
  uses the management Cluster's name. This supports the candidate management-ID
  proxy route, but does not verify this deployment's TLS, token or permissions.

## RK-API visibility blocker

The candidate enumeration request is
`GET /apis/management.cattle.io/v3/clusters`.

In the inspected release:

1. [AllowAll](https://github.com/rancher/rancher/blob/v2.15.2/pkg/api/steve/disallow/disallow.go)
   does not contain `clusters`.
2. [Proxy middleware](https://github.com/rancher/rancher/blob/v2.15.2/pkg/api/steve/proxy/proxy.go)
   sends management CRD requests outside that allow-list through
   `authLocalCluster`. That check requires authorization for `get` on
   `management.cattle.io` resource `clusters`, name `local`, and returns 403
   if the check fails. Passing that gate does not itself establish list RBAC.
3. [Upstream tests](https://github.com/rancher/rancher/blob/v2.15.2/pkg/api/steve/proxy/proxy_test.go)
   cover rejection of non-allow-listed management CRD requests without local
   access. These tests were inspected, not executed during this investigation.

Consequently, downstream visibility alone is insufficient for this candidate
endpoint. A Kubernetes list permission is also not evidence that the response
has Rancher's per-user accessible-cluster filtering. The actual visibility
semantics and minimum read-only role remain unverified. Do not advertise
arbitrary downstream-only Rancher identities as supported, broaden credentials,
or interpret a 403 as an empty cluster list.

This is a compatibility decision requiring resolution under design decision 2,
not proof that every RK-API deployment is unusable. One possible narrower
contract is an explicitly configured management inventory reader with the
necessary local-cluster gate and management list permissions, plus independently
authorized downstream read access. That contract requires verification and an
explicit design/spec revision before implementation.

## External verification unavailable

Read-only checks using the documented homelab kubeconfig and `k00m00` context
failed: the Kubernetes endpoint timed out and then refused connections. The
workstation's default context belongs to another cluster and is not evidence
about the homelab. No credentials were extracted, issued, renewed or changed.

Still required before the integration proceeds:

- Confirm the live Rancher release and an existing intended read-only Rancher
  kubeconfig/context without disclosing its contents.
- Verify candidate API discovery, imported/provisioned/local coverage,
  management IDs, names, restricted-user visibility and complete pagination.
  Kubernetes `limit`/`metadata.continue` is a candidate, not a recorded live
  result. Do not invent sanitized "live" fixtures from it.
- Verify bearer/TLS proxy access and the minimum enumeration/collector RBAC;
  no Secret reads or Rancher writes are part of this verification.
- Compare `GET /api/v1/namespaces/kube-system` through the direct and Rancher
  proxy routes with the intended credentials, including denied access.
  `metadata.uid` is only candidate overlap evidence; cloning/restoring metadata
  can reproduce it, and matching observations must never auto-bind clusters.
- Measure representative complete listings before selecting byte, page,
  source/aggregate count and timeout budgets. No measurements or limits have
  been fabricated in this investigation.

At this initial investigation stage, tasks 1.1–1.4 were incomplete and no application behavior had changed. Subsequent qualification and implementation are recorded below.

## Development Rancher follow-up

The user supplied an existing development Rancher instance on 2026-11-11.
Its HTTPS `/ping` returned `pong` with normal system certificate verification.
An existing workstation downstream context uses that origin and embedded bearer
authentication. Read-only requests with that existing credential produced:

- `GET /apis/management.cattle.io/v3/clusters?limit=2`: HTTP 403.
- `GET /version`: HTTP 401; no Rancher release was established by this request.
- Through the existing downstream proxy context,
  `GET /api/v1/namespaces/kube-system`: succeeded with a nonempty UID. The UID
  itself was not retained in this record.

This establishes working downstream proxy access and a usable candidate marker
on one connection, while enumeration is denied for the tested credential.
The denial alone does not distinguish a cluster-scoped token restriction, the
local-cluster access gate, or missing management list permissions. It does not
establish incompatibility for other credentials or the running release.
No token, raw kubeconfig, credential path or response body was retained here.
The remaining enumeration/RBAC, release, coverage, pagination and paired-marker
checks still require a suitable existing management credential.

### Supplied management credential

The user subsequently supplied a local credential file for this same instance.
Read-only verification using its bearer token established:

- `/rancherversion` reports Rancher **v2.15.2**, Git commit
  `272e65580ca92b9dc51e0f0f0f9ae9d6d356e3b8`, matching the inspected source.
- The candidate management enumeration endpoint returns HTTP 200 and
  `management.cattle.io/v3` `ClusterList`. Following Kubernetes
  `metadata.continue` tokens with `limit=2` completes **four pages / seven
  unique clusters**, including `local`. Page sizes were 38,280, 37,610,
  38,255 and 8,149 bytes; the complete listing took approximately 1.4 seconds.
- Six entries have `spec.importedConfig`; none has
  `status.provisioningClusterRef`. This instance therefore does not establish
  live provisioned-cluster coverage; the upstream controller evidence remains
  the available evidence for that path.
- `GET /k8s/clusters/local/api/v1/namespaces/kube-system` succeeds with a
  nonempty UID. A downstream proxy probe also succeeds and matches the
  existing workstation context's UID. That workstation context itself uses
  Rancher, so this is not an independent direct/proxy identity comparison.
- On that downstream cluster, bounded list checks (`limit=1`) returned HTTP
  200 for Services, Namespaces, Pods, ConfigMaps, Deployments, ReplicaSets,
  StatefulSets, DaemonSets, Ingresses, HTTPRoutes and Gateways. These checks
  establish working read access, not the minimum RBAC grant or completion of
  the application's full collectors.
- `/version` returns the management Kubernetes release
  `v1.36.5+rke2r1`; it must not be interpreted as the Rancher release.

The supplied credential clears the immediate enumeration-access blocker for
this instance. The previous credential's 403 is still valid evidence of denied
enumeration. The precise token-scope/RBAC difference was not inspected and is
not inferred. All requests used normal verified HTTPS without redirects,
legacy API fallback, Secret reads or resource writes. No credential contents
or raw Cluster response bodies were retained. The local credential file is
ignored by Git and its permissions were restricted to its owner.

Remaining acceptance checks include minimum-role/restricted visibility,
provisioned coverage, an independent direct/proxy identity pair and denied
probe fixtures, and measured implementation budgets. No verification task is
marked fully complete from these successful smoke checks alone.

## Baseline reconciliation

The multi-cluster baseline was archived on 2026-11-11 as
`openspec/changes/archive/2026-11-11-add-multicluster-discovery-snapshots`.
Its current requirements now live in `openspec/specs/`, including
`cluster-discovery-registry`, `shared-inventory-snapshots`,
`inventory-classification` and `multicluster-inventory-experience`.
The ExternalDNS Service baseline is also archived and has a current
`externaldns-service-discovery` specification. Implementation must preserve
these current specifications; task 7.4 still requires the final scenario review.


## Implementation verification

- The actual Go adapter's opt-in `TestRancherLiveContract` passed against the
  supplied instance: seven clusters from a complete RK-API listing, then a
  verified-TLS bearer-authenticated proxy Namespace probe. Temporary kubeconfig
  material was owner-readable only and removed after the test; it was never
  included in Git or test output.
- `go/testdata/rancher/clusters.json` is a hand-authored sanitized schema fixture,
  not a retained production response. It covers imported, management and
  provisioned records with deliberately different provisioning/management names.
- Direct/proxy fixture probes agree on the hash of the same Namespace UID;
  missing/forbidden probes, duplicate/clone evidence, stale observations,
  binding conflicts and replacement markers are covered without automatic joins.
- Source-derived enumeration requirements are management Cluster `get` on
  `local` plus management Cluster `list`. Existing collector operations are
  documented in README, with optional Namespace `get` for identity. Live checks
  used the supplied existing credential; a minimum-role identity was not
  created. The restricted existing credential's 403 remains a tested denial.
- `BenchmarkRancherBudget` enumerates 100 synthetic records over two pages, with
  20,000 bytes of unused payload/object approximating live record sizes. Three
  runs averaged 14,789,403 ns/op (~14.8 ms), 17,573,482 allocated bytes/op and
  4,174 allocations/op on Apple M2 Pro. These are local fixture measurements,
  not live latency or a performance guarantee. Counts/byte/time limits are
  fixed in `rancher.go`, fail complete listings on overflow and are documented
  in README/design.
- Backend tests cover pagination failure, overflow, trusted-origin routing,
  credential rotation/coherent versions, missing/unsupported credentials,
  source failure retention, successful empty discovery versus failed initial
  discovery, binding/primary behavior, identity invalidation, snapshot migration,
  exclusions/removal, concurrent readers and private-data disclosure.
- Browser fixtures distinguish source enumeration from component failure,
  expose connection roles/unavailable probes/overlap/conflict guidance, and
  preserve canonical favourites without merging discarded duplicate IDs.
- No Secret creation, permission changes, resource writes, image publication,
  GitOps rollout, commit or push was performed.

Unavailable external qualification remains explicit: an independent live
Kubernetes-direct/proxy pair, live provisioned-cluster coverage, a newly scoped
minimum-role account, and container runtime smoke checks (Docker daemon absent).
These do not change the implemented failure behavior or establish broader
Rancher release support.
