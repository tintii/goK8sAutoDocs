# Implementation validation

## Results

- Full backend suite and Go race detector passed (`go test -race ./...`).
- `go vet ./...` passed.
- Existing full browser suite: 92 passed, zero failed/skipped. The updated
  inventory suite then passed all 11 tests, including the two new Rancher
  cases (94 distinct browser tests covered across those runs).
- The actual Go adapter's explicit live test passed against development
  Rancher 2.15.2: complete seven-cluster enumeration and an authenticated,
  verified-TLS downstream identity probe. See verification.md for measurements
  and the distinction between live evidence and source/fixture qualification.
- A representative 100-cluster/two-page/20 KB-per-object fixture benchmark
  passed; limits fail whole source listings rather than truncating them.
- Linux amd64 binary cross-compilation passed (`CGO_ENABLED=0`, `-trimpath`), separately from Docker.
- Docker container smoke checks could not run because the Docker daemon is
  unavailable. No container-runtime success is claimed.
- Strict change validation and `git diff --check` passed.

## Requirement and scenario review

| Requirement/scenarios | Evidence |
| --- | --- |
| Mixed, Rancher-only, empty and excluded configuration | `TestRancherConfigBindings`, `TestRancherOnlyEmptyHTTP`, `TestFailedEmptyDiscoveryIsNotSuccessfulEmptyInventory`, credential/list fixtures |
| Supported management resource, imported/provisioned IDs, API incompatibility | `TestRancherImportedAndProvisionedFixture`, `TestRancherInvalidListingsAndUntrustedNavigation`, live RK-API check, pinned upstream controller evidence |
| Missing/unsupported credentials, rotation and coherent refresh versions | `TestRancherPaginationAndCredentialRotation`, `TestRefreshCredentialVersionIsCoherent`; no executable or certificate authentication for sources |
| Proxy-only collection, scoped denial and trusted origin | `TestRancherProxyReusesCollectors`, redirect/route fixtures, live downstream read checks and existing credential's enumeration 403 |
| Stable canonical IDs, source namespaces and rename/rotation | Tagged JSON connection references, base64url tuple IDs, config/rotation tests, canonical browser fixtures; display names and bearer bytes do not determine IDs |
| Matching, denied/missing and stale identity observations | `TestIdentityDirectAndProxyWithDeniedProbe`, `TestIdentityProbeDeniedAndCloneEvidence`, `TestOverlapCurrentEvidenceAndIdentityReplacement`; no stale comparison or automatic equivalence |
| Invalid bindings/primaries, temporarily undiscovered members | `TestRancherConfigBindings`, `TestUndiscoveredRancherOnlyBindingRemainsVisible` |
| Primary-only collection, conflict, outage, switch and absent primary | `TestBindingPrimaryOnlyConflictAndPrimarySwitch`; observed marker changes discard incompatible inventory |
| Unresolved overlaps stay separate; discarded preferences never merge | Overlap backend fixture and canonical-primary browser fixture retain separate IDs, warn, and preserve the chosen canonical favourites |
| Complete authoritative reconciliation and partial failure retention | `TestDynamicMembershipFailureEmptyAndRecovery`, late-page fixtures, `TestSourceOverflowRetainsWholeMembership` |
| Direct-primary isolation from a failed source | Mixed-source recovery fixture plus binding primary-only tests; a direct client never depends on source enumeration success |
| Coherent generation, coalescing/cooldown and concurrent readers | `TestRegistryReadersSeeOneGeneration`, original shared coordinator/HTTP fixtures and full race suite |
| Versioned recovery, source removal/exclusion, classification/trust/primary incompatibility | `TestCacheExclusionsTrustAndBindings`, `TestSourceCacheTrustCompatibility`, `TestRancherClassificationInvalidatesDynamicCache`, original version-1 direct/cache-write fixtures |
| Source versus collection freshness, roles, unavailable probes, conflict guidance | Additive discovery-status projections and Rancher browser diagnostics fixture; original directory/overview/versions fixtures remain passing |
| Credentials, paths and raw kubeconfigs never enter logs/snapshots/status | `TestRancherFailureDisclosure`, `TestDynamicMembershipFailureEmptyAndRecovery`, original structured-log tests; persisted source metadata contains fingerprints, sanitized membership and owned errors only |
| Homelab migration and rollback | README retains `homelab` direct primary and Rancher `local` secondary; separate credential/image/GitOps rollout, version-1/version-2 compatibility and rollback instructions |

The multi-cluster and ExternalDNS changes are already archived; current
`openspec/specs/` is the implementation baseline. Publication labels, Service
endpoint interpretation, classification and array-based inventory responses
remain covered by the existing suites. Direct-only snapshots stay at version 1;
source/binding snapshots use version 2. Explicit bindings stay visible with
unavailable connection status even when every referenced dynamic member is
absent. Successful complete lists remove unbound dynamic members.

## External limits

No independent live direct/proxy pair, live provisioned sample, newly created
minimum-role account or container runtime was available. Source-derived minimum
operations and fixture-based behavior are documented as such. No broader
Rancher release support, per-object list filtering, live minimal-role proof or
container runtime qualification is implied. No cluster resources, permissions,
credentials, images or deployment pins were changed, and nothing was committed
or pushed.
