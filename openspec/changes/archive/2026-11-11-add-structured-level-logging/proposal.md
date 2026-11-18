## Why

AutoDocs currently mixes unlevelled standard-library log messages with a global `DEBUG` flag and resource dumps, while its active multi-cluster inventory coordinator produces no operational logs. Operators need consistent, safe messages that explain startup, refresh outcomes and failures without filling logs with browser polling or exposing Kubernetes data.

## What Changes

- Introduce structured JSON application logs on stderr with `debug`, `info`, `warn` and `error` severity and a configurable minimum `LOG_LEVEL`, defaulting to `info`.
- Preserve `DEBUG=true` as a compatibility fallback when `LOG_LEVEL` is unset; explicit `LOG_LEVEL` takes precedence and invalid values fail startup with a clear diagnostic.
- Add contextual lifecycle, refresh, per-target/component failure, recovery and snapshot restore/persistence events to the active runtime paths.
- Use debug events for collection summaries and refresh scheduling decisions; keep successful health checks, status polling and ordinary snapshot reads quiet.
- Replace legacy `log.Print*`, `log.Fatal` and debug-only resource dumps with consistent levelled events and safe, bounded attributes.
- Document severity selection and configuration, and verify filtering, failure/recovery coverage, credential safety and concurrent refresh behaviour with fixtures.

## Capabilities

### New Capabilities

- `application-logging`: Configurable severity, structured output, operational event coverage, noise control and safe log contents across application paths.

### Modified Capabilities

None. Existing discovery, API, snapshot and browser behaviour remains governed by its current specifications; this change adds observability around those operations.

## Impact

Changes are scoped to goK8sAutoDocs: a small logging module using Go's standard `log/slog`, startup and legacy helpers in `go/main.go`, the inventory coordinator and snapshot persistence in `go/inventory.go`, configuration/client failure boundaries, HTTP handlers, fixture tests and README configuration/troubleshooting/changelog sections. No new logging dependency, frontend change, Kubernetes resource, image publication or GitOps rollout is required to prepare or implement this change. Logs must preserve the existing credential-sanitization boundary and must not change response schemas, refresh scheduling or stale-data retention.
