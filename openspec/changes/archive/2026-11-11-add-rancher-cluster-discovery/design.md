## Context

The pre-change baseline validated 1–100 explicit targets and persisted version-1 snapshots. The implemented registry now supports optional Rancher sources, canonical bindings and dynamic targets; `inventory_registry.go` resolves one registry per refresh and `inventory_http.go` projects that committed registry. Direct-only operation retains version-1 compatibility. Resource and browser preference identities continue to include the canonical AutoDocs cluster ID.

The implemented `add-multicluster-discovery-snapshots` change was archived on 2026-11-11; its baseline now lives in the current `openspec/specs/` specifications. ExternalDNS Service discovery is also an archived, implemented baseline; Rancher must reuse its collectors without changing publication or endpoint interpretation. The user approved visible overlap warnings, explicit primary selection, no automatic merging/failover, and retaining the direct hosting-cluster connection as primary when it overlaps Rancher's `local` cluster.

The [2026-11-11 verification record](verification.md) distinguishes live
Rancher 2.15.2 checks, source evidence and automated fixtures. The accepted
contract requires an unscoped bearer token with management inventory read
permissions, including Rancher's `local` access gate. It does not promise
per-object filtering for a Kubernetes list permission or discovery for arbitrary
downstream-only credentials. No legacy API fallback is implemented.

## Goals / Non-Goals

**Goals:** One configured Rancher source discovers its accessible clusters; collection travels through Rancher; direct connections coexist; dynamic membership survives transient failures; overlap is visible and explicitly resolvable; stable canonical identities preserve preferences.

**Non-Goals:** Credential entry in the browser, account/token management, cluster writes, automatic primary election or failover, automatic permission-union inventories, per-browser-user authorization, project-scoped discovery, or deployment as part of this proposal.

## Decisions

### 1. Add configured Rancher sources, reuse mounted credentials

Add `ranchers` beside existing `clusters`. Each source has a stable `id`, optional display `name`, a kubeconfig path and explicit context, and an optional list of excluded Rancher cluster IDs. Discover all credential-visible clusters, including `local`, unless excluded. Permit Rancher-only configuration and a successfully discovered empty inventory; preserve today's implicit local target only when no configuration file is supplied.

Read one coherent credential version per source per refresh, using the existing self-contained HTTPS kubeconfig validation. The kubeconfig server denotes the Rancher management endpoint. Reuse its verified TLS trust and supported bearer-token authentication for enumeration and downstream proxy clients. Do not assume that Kubernetes client certificates work as Rancher user credentials; the initial Rancher contract is bearer-token authentication. Invalid credentials affect their source, not direct targets or other sources. No credential bytes enter derived metadata or persistence. Optional Secret directory mounts, without `subPath`, support rotation without a process restart; source and binding configuration changes require restart.

Implemented optional configuration:

```yaml
clusters:
  - id: homelab
    name: Homelab
    inCluster: true
ranchers:
  - id: homelab-rancher
    name: Homelab Rancher
    kubeconfig:
      path: /etc/autodocs/rancher/kubeconfig
      context: rancher
    excludeClusterIDs: []
clusterBindings:
  - id: homelab
    name: Homelab
    connections:
      - direct: homelab
      - rancher: homelab-rancher
        clusterID: local
    primary:
      direct: homelab
```

One or more sources use the same interface; default deployment needs only one. Source IDs are unique. A connection reference contains exactly one method, references a configured direct target or Rancher source, and is assigned to at most one binding. A binding has exactly one primary listed among its members. A referenced Rancher cluster may be temporarily undiscovered; report its availability rather than rejecting an otherwise valid configuration. Reject contradictory exclusions/bindings, ambiguous canonical IDs, and collisions with unmapped targets. Explicit bindings do not broaden source exclusions or permissions.

Alternative: a URL/token-specific credential schema. Reusing kubeconfig keeps trust and file handling consistent with direct targets. A browser connection editor would add credential storage and application authorization concerns beyond this change.

### 2. Verify the supported management API, then build proxy clients

Prefer Rancher's supported RK-API for enumerating managed clusters. The release contract is Rancher 2.15.2, using `GET /apis/management.cattle.io/v3/clusters` with Kubernetes `limit` and opaque `metadata.continue`. The source token must have management inventory access: the upstream handler requires `get` on management Cluster `local`, followed by management Cluster `list` authorization. A Kubernetes list grant is not a per-cluster visibility filter; configure an identity permitted to read that inventory. The existing downstream credential was denied enumeration while the supplied management credential succeeded. Check imported as well as provisioned clusters: a provisioning-only collection is insufficient. Do not substitute a provisioning object name for the management ID used by the proxy.

The downstream route `/k8s/clusters/<management-cluster-id>` and bearer/TLS compatibility were verified live and through typed/dynamic collector fixtures. Construct downstream `rest.Config` values from the validated source trust/authentication and verified route contract, then reuse typed/dynamic collectors. No downloaded per-cluster kubeconfig, Rancher CLI or executable authentication helper is required. Keep credentials on the configured trusted Rancher origin; reject cross-origin authenticated redirects/pagination and invalid route identifiers.

The legacy `/v3/clusters` approach is attractive for simplicity, but current upstream documentation calls v3 unsupported. Do not silently fall back. If RK-API does not expose required inventory with suitable permissions on the target version, document that evidence and revise the API compatibility decision before implementation continues.

### 3. Separate connection IDs from canonical cluster IDs

Internally identify direct connections by a tagged direct-target reference and Rancher connections by a tagged `(source ID, management cluster ID)` tuple. Unmapped direct connections retain their existing AutoDocs cluster IDs. Unmapped Rancher connections receive deterministic, unambiguously encoded IDs derived from their tuple. Validate collisions instead of relying on display names or delimiter concatenation.

Bindings assign multiple connections to a canonical cluster ID and designate one primary. Bindings involving an existing direct target retain that target's canonical ID for the initial migration. Newly bound Rancher-only connections should retain the chosen primary's existing canonical ID; do not promise automatic migration of preferences belonging to a discarded duplicate ID. Display names, credentials and connection selection never become part of source identity.

Collect resource inventory only through the primary. Secondary connections can perform a bounded identity probe; they do not run full collectors. No fallback occurs on primary failure, absence or restricted permissions. Where the primary is absent but a secondary remains present, show the canonical cluster as unavailable/stale, not as successfully refreshed. Canonical labels use explicit binding labels when supplied, otherwise existing direct-target labels where a direct target owns the canonical ID. Defer automatic import of Rancher labels until their semantics and precedence are specified; connection switching must not change classification unexpectedly.

Changing primary preserves canonical source/link IDs and preferences for unchanged resources/URLs. Invalidate prior inventory on a primary change before serving data from the new configuration: different credentials may expose different resources. Include primary connection identity and relevant source configuration in cache compatibility checks. Routine credential rotation keeps identity stable and follows the existing refresh credential contract. Observed cluster identity changes invalidate old inventory rather than transferring it to a replacement cluster.

### 4. Treat identity markers as overlap evidence

Use a hash of the UID of the `kube-system` Namespace as the observed marker. Live proxy reads and direct/proxy fixture comparisons verified this API shape; an independent live direct/proxy pair was unavailable. The additional operation is `get` on Namespace `kube-system`; denied probes are covered by fixtures. This is an observed marker, not a guaranteed global cluster identifier: restored/cloned clusters can share metadata. Names and endpoint fingerprints alone never establish equivalence.

Compare successful observations from the same refresh generation. Equal markers across unbound connections produce `possible overlap`; missing/forbidden/unreachable checks produce `identity check unavailable`; contradictory markers in an explicit binding produce `binding identity conflict`. Never use a stale cached observation to claim a fresh match or conflict. A failed probe does not block otherwise permitted resource collection.

An explicit binding remains the operator's selection even when the probe is unavailable. A conflict warns prominently while collecting only the chosen primary; do not merge or switch. Unresolved connections retain distinct canonical identities, continue independent collection, and can therefore show duplicate resources. Explain this consequence next to the warning. Do not persist automatic equivalence decisions or choose a winner by list order.

### 5. Reconcile dynamic membership within the shared refresh

Use one immutable resolved registry per refresh. First enumerate each source completely, then resolve bindings and probe identities, then collect primaries and unresolved standalone connections. Keep existing coalescing/cooldown behavior and bounded source/cluster operations. A source failure cannot prevent independent sources or direct targets from refreshing.

Only a complete, validated listing is authoritative. Handle all pages before replacing source membership. A page failure, invalid response, exceeded size/count limit or timeout marks discovery failed and retains the prior membership; it is not an empty cluster list. A complete successful empty listing removes that source's discovered members. Do not quietly truncate at the existing 100-target limit: define and enforce bounded aggregate/source budgets and report overflow as discovery failure.

On transient discovery failure, retain the previous source membership and last successful data as stale; skip that source's proxy collection for that run. A direct primary in a mixed binding still refreshes. On a complete successful listing, newly visible members join immediately; absent members leave active inventory. This is loss of visibility, not necessarily physical deletion. Remove a canonical dynamic cluster only when it has no remaining configured/present members. A configured direct target keeps its canonical cluster present independently of Rancher membership.

Publish reconciled membership and inventory together as one snapshot generation. Data/status handlers read the resolved registry, not `config.Clusters`; progress accounts for source discovery and resolved collection work. Never expose a half-reconciled list with inventory from an unrelated generation.

### 6. Persist sanitized membership and report source health

Version the snapshot envelope to include source membership, canonical/connection associations, primary selection, collection provenance, identity-probe status and source discovery timestamps/errors. Persist only the sanitized fields required for recovery, not the source configuration object or credentials. Bound the file and validate entries on restore.

Read existing version-1 snapshots for unchanged direct targets. Restore new-format discovered membership only when source identity, endpoint/trust compatibility, bindings and collection configuration still match. Mark restored discovery and inventory stale until refreshed. Removed/excluded sources and reassigned primaries cannot resurrect old inventory. Retain atomic write behavior and isolate persistence failures from in-memory service.

Extend discovery status and shared UI with Rancher source status/last successful enumeration, canonical cluster primary/secondary connections, overlap warnings, identity-check availability and action guidance. Source discovery failure and downstream collection failure are distinct. Show a primary outage even if another connection exists. Expose sanitized IDs/names, never endpoint credentials or kubeconfig paths. Existing data APIs remain arrays and use canonical IDs across directory, overview and versions.

The UI remains a shared inventory view under the configured service identity. Rancher credential permissions are not browser-user-specific authorization. Status warns and explains the configuration needed; it does not accept credentials or mutate bindings.

## Risks / Trade-offs

- **RK-API coverage and token behavior vary by release** → First implementation task records verified resource paths, token behavior, version scope and minimal RBAC; no assumed v3 fallback.
- **One Rancher outage affects many targets** → Retain stale inventory with source-level diagnostics; direct primaries remain independent.
- **Identity metadata is unavailable or duplicated by cloning** → Treat it as evidence; require explicit binding and never automatically collapse results.
- **Primary switching changes permission scope** → Preserve canonical identity but invalidate previous-primary data; no inventory union or automatic failover.
- **Dynamic deletion can follow lost visibility** → Remove only after a complete successful listing; communicate absence without claiming physical deletion.
- **Discovery increases refresh duration** → Bound requests, pagination, counts and identity probes; measure worst-case behavior before deciding whether bounded parallel collection is necessary.
- **Previously duplicated IDs have separate preferences** → Retain the chosen canonical ID; document that preferences on discarded duplicate IDs are not automatically merged.

## Migration Plan

1. Verify API/identity assumptions and record evidence in this change before implementing the integration.
2. Add the source/resolved-registry model while preserving direct-only configuration and version-1 recovery behavior. Implement bindings, reconciliation, snapshots and status behind optional Rancher configuration.
3. Validate fixture-based failure, overlap, identity, migration and UI scenarios, then run the existing backend/race/browser/container checks appropriate to changed paths.
4. Update README and CHANGELOG when behavior is implemented, including the verified Rancher compatibility/RBAC contract and illustrative Secret mounts. Do not advertise proposal fields as currently supported.
5. For a later homelab rollout, retain the current hosting-cluster ID/direct primary and bind Rancher's `local` connection explicitly. Mount credentials through the existing deployment chart; publish/pin the image through the established separate workflow. No live configuration or credentials are changed by this proposal.
6. Roll back by removing Rancher/binding configuration and returning to the previous image with a compatible saved version-1 cache or an empty cache. The old binary must not be expected to read the new snapshot format; a fresh direct inventory remains possible without it.

## Qualification limits

- Live verification covered Rancher 2.15.2, seven management Clusters including `local`, imported objects, complete continuation and collector read checks on one downstream cluster. Provisioned metadata/ID selection is supported by upstream controllers and fixtures, not a live provisioned sample.
- Minimum read operations are derived from the inspected handler and collectors. The supplied credential was not replaced with a newly created minimum-role identity. No permissions or cluster resources were changed during qualification.
- The independent live direct/proxy identity pair remains unqualified; fixtures cover matching, denied, unavailable and conflicting probes. Namespace UID is evidence only and can be duplicated by cloning/restoration.
- Limits are 10 sources, 100 listed clusters/source before exclusions, 200 configured/discovered references overall, 100 pages/source, 50 requested items/page, 4 MiB/page, 16 MiB/source, a 15-second source deadline and a 10-second request timeout. Identity work uses 2 seconds/connection within a 20-second generation budget. Exceeding a discovery budget fails the whole listing. The representative 100-object fixture uses about 20 KB/object, based on live response sizes, and averaged about 14.8 ms per complete two-page enumeration on this workstation. Collection remains sequential and retains the configured cluster timeout; worst-case refreshes may exceed the scheduling interval.
- Direct-only configurations keep version 1; source/binding configurations use version 2. Restore rebuilds canonical associations from current configuration and checks primary/configuration/trust compatibility. Reassigned primaries cannot serve old-primary rows. Routine bearer rotation does not change identity.
- Docker was unavailable, so container runtime smoke qualification is outstanding; backend, race, browser and opt-in live adapter checks provide the available validation.

## References

- [Rancher RK-API quickstart](https://ranchermanager.docs.rancher.com/api/quickstart): Rancher endpoint kubeconfig, bearer credential and client-go access.
- [Previous v3 API guide](https://ranchermanager.docs.rancher.com/api/v3-rancher-api-guide): current unsupported-v3 warning and RK-API guidance.
- [Rancher API keys](https://ranchermanager.docs.rancher.com/reference-guides/user-settings/api-keys): token scope and token resource migration.
- [Rancher access paths](https://ranchermanager.docs.rancher.com/v2.13/how-to-guides/new-user-guides/manage-clusters/access-clusters/authorized-cluster-endpoint): authentication proxy versus direct downstream access.
