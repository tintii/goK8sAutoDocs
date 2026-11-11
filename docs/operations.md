# Operating AutoDocs

## Refresh and freshness

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
Services have their own partition and freshness/status diagnostics. One standalone
Service list supplies both directory views and published filtered metadata;
the generic metadata collector skips Services. The whole filtered partition is
retained if any of its required source lists fail, even when other directory
partitions successfully update.

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
unsupported. Rollback can retain the PVC; select compatible configuration and
cache data as described below. Home content has its separate live reread.

For direct-only configuration, the additive `services` partition keeps snapshot format version 1. Upgrading
from a compatible v1 cache without Services preserves existing data and marks
Services never collected until their first successful refresh. New caches restore
Service rows with original timestamps, including after failed target connections.
For these version-1 caches, rolling back to the preceding v1 snapshot reader preserves existing source data:
it ignores the additional partition in its directory projections. No DNS records
are changed by upgrade or rollback, and existing browser source/link preferences
remain compatible.

## Upgrade and roll back

1. Read the [release history](https://github.com/tintii/goK8sAutoDocs/blob/master/CHANGELOG.md) and select a verified image digest for your runtime architecture.
2. Save your configuration and a compatible cache backup; keep credentials in your secret supplier.
3. Stop the active instance before starting the replacement. Retain mounted configuration, credentials and cache volume.
4. Check health, per-component success times and directory/Versions content. A running HTTP server alone does not prove successful collection.
5. To roll back, restore the previous digest and compatible configuration/cache, again without overlapping processes.

Rancher/binding configurations write **version-2 snapshots**, which older binaries cannot read. For rollback to the preceding direct-only reader, remove Rancher/binding configuration and restore a compatible saved version-1 cache or start with an empty cache. Changing a bound primary invalidates cached inventory from the preceding primary while retaining canonical IDs for preferences. See [Connecting clusters](cluster-discovery.md) for source removal and identity rules. The optional manual-link component is additive; older binaries ignore it.

## Protect access

The application has no built-in authentication. Use your existing proxy/ingress access control when exposing it. Browser filters do not restrict API data or collector permissions. Grant read-only collection permissions and mount credentials read-only. Treat cache files as derived inventory metadata; keep them on the intended private volume. AutoDocs does not renew tokens or certificates.

For logs and recovery steps, see [Troubleshooting](troubleshooting.md).
