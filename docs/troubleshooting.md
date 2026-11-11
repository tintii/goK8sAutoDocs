# Troubleshooting

Start with `/api/v1/health`, then `/api/v1/discovery-status` or **Scope & updates → Collection details**. A Running indicator proves only that AutoDocs responds. Diagnose the failing cluster/component without printing credentials.

| Symptom | Check and expected result |
| --- | --- |
| Process exits before listening | Confirm `~/.kube/config` exists or `AUTODOCS_CONFIG` is readable; check schema, explicit context and `LOG_LEVEL`. Invalid startup configuration exits with status 1. |
| HTML/static files return 404 | Run the binary from `go/`, beside `static/`, or use the packaged container working directory `/app`. |
| Port 8080 is occupied | Stop the conflicting local listener or remap the container's host port. The application listener is fixed at 8080. |
| Credentials invalid or unsupported | Use a verified HTTPS, self-contained embedded token or cert/key kubeconfig. `KUBECONFIG`, exec helpers and credential file references do not select/complete it. See [credential contract](cluster-discovery.md#remote-cluster-secret-contract). |
| Connection/authentication fails | Check API reachability from the runtime, TLS trust, credential expiry and the selected context. Repair the supplier's file and refresh after cooldown. |
| Access forbidden | Check [RBAC](installation.md#kubernetes-permissions) for that collector identity across namespaces. Test with `kubectl auth can-i list services --all-namespaces` using the same identity. Expect `yes`; do not grant writes or Secret reads. |
| Rancher enumeration is 403 but a proxy works | Use an unscoped API key with management inventory access, including local-Cluster get and Cluster list; see [Rancher permissions](cluster-discovery.md#rancher-cluster-discovery). |
| Applications is empty | Look in Discovered; publish with the exact label string `"true"`. Group labels alone never publish. A successful empty collection is valid. |
| Endpoint has no clickable URL | Read issues, listener/host attachment and explicit Service web hints. Wildcard-only or DNS-only declarations have no inferred browser URL. |
| Inventory is stale or 503 | Inspect per-component attempt/success times and warnings. 503 means no usable snapshot yet. Failed collection retains old successful rows; fix the underlying connection/API and use Update now. |
| Update now does not start another scan | Shared coalescing and cooldown prevent extra runs. Check next manual time and current progress. |
| Home content fails or does not update | Validate one YAML document under 64 KiB; mount the entire ConfigMap directory without subPath and wait for projection. The UI retains old content/defaults with a notice. |
| Cache cannot restore/write | Check `cacheError`, format compatibility, directory permissions for UID 65532 and disk capacity. Stop the instance before removing a corrupt cache; restart gives cold discovery. |
| Browser preferences disappear | Confirm the same origin/profile and allowed localStorage. Host/port changes use a different store. Clearing site data removes personal choices. |
| Manual link remains after invalid edit | Invalid YAML retains that document's previous links. Use valid `links: []`, remove the marker or delete the ConfigMap, then refresh successfully. |

## Logging

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
