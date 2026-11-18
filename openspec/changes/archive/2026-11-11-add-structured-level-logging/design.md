## Context

`go/main.go` uses the standard `log` package and global `debugMode`, including complete Pod/container/related-resource dumps. `go/resource_handlers.go` logs legacy cache failures, but the running application serves inventory through `inventoryService` and handlers in `inventory_http.go`. That service coordinates refreshes, retains stale partitions, restores snapshots and persists new generations without logging those events. `home.go` can return configuration errors without an operator-facing diagnostic.

Existing `safeCollectionIssue` and snapshot cache messages deliberately hide upstream URLs and credential parser contents. Logging must preserve that boundary. Inventory reads and status/health polling are frequent; Kubernetes collection belongs to the shared coordinator, not individual requests. The module already targets a Go version with `log/slog` available.

## Goals / Non-Goals

**Goals:**

- Make startup, inventory updates, degraded operation and recovery understandable from container logs.
- Provide predictable severity filtering and structured fields without a new dependency.
- Preserve `DEBUG=true` compatibility and existing API/discovery behaviour.
- Keep event volume proportional to refresh operations rather than users or inventory size.
- Make output capturable in tests and safe at every severity.

**Non-Goals:**

- Log aggregation infrastructure, metrics, distributed tracing, audit logs or frontend logging.
- Request access logs, configurable formatters, runtime log-level changes or changes to Kubernetes client-library verbosity.
- New shutdown/signal handling, collection retries, discovery scheduling, deployment changes or image publication.

## Decisions

### 1. One application logger using standard-library slog

Add a small `go/logging.go` module configuring `slog.NewJSONHandler` on stderr. Each record has the standard `time`, `level` and `msg` keys plus a stable application-owned `event` value. Use consistent attributes where relevant: `component`, `cluster_id`, `generation`, `trigger`, `duration_ms`, counts, `reason` and `retained_previous`. Attribute values must be scalars or deliberately bounded summaries, not serialized objects.

Initialize logging before configuration loading, snapshot restore or client creation. Application helpers use `slog` consistently; tests can provide a capture writer and isolated logger without concurrently mutating global defaults. Use service-owned logger references or constructor injection where needed for concurrent coordinator tests. Keep the module small rather than introducing a logging interface hierarchy. Remove application-owned legacy print/fatal calls; emit a final error before exiting nonzero for startup/listener failures. Do not globally redirect dependency logs, which could bypass application sanitization.

Alternatives: keeping string prefixes would not supply real filtering or structured fields; adding a third-party logger duplicates facilities already in Go. A selectable text formatter is unnecessary for this scope.

### 2. Explicit level configuration with a legacy fallback

`LOG_LEVEL` accepts `debug`, `info`, `warn` and `error`, ignoring surrounding whitespace and letter case. An unset or blank value uses `debug` only when the existing `DEBUG` value is exactly `true`, otherwise `info`. A nonblank explicit value always wins. An unsupported value causes one structured error and nonzero exit before inventory work or HTTP startup; report the allowed values without echoing the supplied value.

The threshold includes the selected severity and all more severe events. There is no hot reload. Remove `debugMode` and replace its guarded messages with `Debug` calls; check `Enabled` before any costly summary construction. This preserves the old debug switch while making verbosity controllable in deployments.

Alternatives: removing `DEBUG` would break existing commands; silently accepting invalid levels could hide diagnostics or unexpectedly flood logs.

### 3. Log at the boundary that owns the outcome

| Level | Events and context |
| --- | --- |
| `info` | Successful startup configuration summary (target count and refresh timing), listener startup, refresh start/completion with generation, trigger, duration and aggregate success/failure counts, compatible snapshot restore, and recovery of a previously failing partition or persistence operation. |
| `debug` | Per-target/component successful collection counts and duration; accepted/manual refresh decisions and coalescing/cooldown/outdated-generation reasons; missing snapshot on first start; successful persistence; optional Gateway API absence. |
| `warn` | Target connection failures; component collection failures or partial-result warning counts; stale-data retention; invalid/unreadable/incompatible snapshot restore; snapshot persistence failure while memory remains usable. |
| `error` | Invalid startup configuration, unexpected HTTP listener failure, and application-owned handler failures that return an internal-server error, using fixed safe reasons. |

A connection failure affecting all target partitions emits one target warning rather than four copies of the same connection error. Component-specific failures emit at most one outcome warning per component per run, with a warning count rather than a message per row. Record recovery only when previous state was failing. Repeated failures can appear once per affected operation/run; no additional retry or cross-run suppression subsystem is introduced.

Capture `startup`, `automatic` or `manual` trigger when accepting work so asynchronous logs retain the right context. Retain the existing public request/status contracts. Summarize state under existing locks, then emit outside write-locked sections where practical; do not hold locks while formatting whole snapshots or performing logging I/O.

Ordinary successful health/status polling, static requests and snapshot reads emit no events, including at debug level. Validation failures such as invalid view/method/origin and expected inventory-not-ready 503 responses are quiet at info and above; any debug diagnostic uses a fixed route/reason. Stale state is logged at collection time rather than repeated on each GET. Log internal handler failures at their owning handler, avoiding middleware duplicates.

Alternatives: request-wide access logging scales with browser activity and repeatedly reports known stale state; logging only at leaf helpers loses cluster/generation context and duplicates failures when callers also log them.

### 4. Allowlisted log context and safe error categories

Reuse `safeCollectionIssue` for collection errors or share its owned-category mapping with logging. Define equivalent fixed categories for configuration, file, homepage YAML and encoding failures. Do not pass arbitrary errors to `slog.Any`, interpolate raw error strings, or record collected warning text unless it has an explicitly safe owned category. Cluster IDs, component names, operation names and counts are sufficient for normal diagnosis; individual resource identity is allowed only for a necessary debug event and must be bounded.

Never log kubeconfig/configuration contents, tokens, certificates, private keys, authorization headers, request/response bodies, ConfigMap values, raw Kubernetes objects, full endpoint URLs, query strings, or configuration-file contents. JSON encoding provides correct escaping, not credential redaction; safe values must be chosen before logging. Replace existing raw object/image dumps with counts and fixed matching outcomes. Tests inject unique credential markers into upstream errors and fixtures, then inspect all captured severities.

Alternatives: regex scrubbing arbitrary strings cannot reliably recognize every credential, URL or parser payload. Preserving raw errors only at debug would still expose secrets when operators need more detail.

## Risks / Trade-offs

- [Default output changes from plain messages to JSON] → Document the format and level controls; stderr remains the output stream and `DEBUG=true` still works.
- [Sanitized reasons omit upstream detail] → Include safe operation/cluster/component context and distinguish timeout, forbidden, unauthorized and generic unavailable categories.
- [Debug volume on large inventories] → Prefer per-component summaries, remove object dumps and avoid per-row logs or expensive disabled-log work.
- [Logger writes extend critical sections or introduce test races] → Capture immutable event context, emit outside locks where practical and run concurrent fixture tests under the race detector.
- [Repeated bad homepage requests can repeat error events] → Emit one bounded record per failing request; do not include payloads or duplicate the event in shared response helpers.

## Migration Plan

Implement the logger and configuration first, instrument active runtime paths, migrate legacy messages, then run fixture/race checks and update README configuration, troubleshooting and Unreleased notes. Verify container logs and stdout/stderr behaviour using the existing fixture smoke harness if available. Existing deployments need no new configuration; `LOG_LEVEL` can be supplied through the workload's existing environment support when intentionally rolled out later. Reverting the application version restores prior logging; no persistent data or browser migration is involved.

## Open Questions

None blocking. JSON output and `info` as the default are deliberate implementation choices; the existing discovery and HTTP contracts remain unchanged.
