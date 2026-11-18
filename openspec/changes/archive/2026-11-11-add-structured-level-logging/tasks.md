## 1. Logging configuration and record format

- [x] 1.1 Add a small standard-library `slog` module with JSON stderr output, stable event identifiers, consistent scalar attributes and an injectable writer/logger for fixture tests.
- [x] 1.2 Implement normalized `LOG_LEVEL` parsing, info default, exact `DEBUG=true` fallback and explicit-level precedence; initialize before configuration/restore and fail nonzero with a safe structured diagnostic for invalid levels.
- [x] 1.3 Add meaningful tests covering every severity threshold, unset/blank/case/whitespace inputs, legacy precedence, invalid-value non-disclosure, valid JSON lines and stderr-only output; verify invalid configuration stops before collection/listener startup.

## 2. Startup and shared inventory instrumentation

- [x] 2.1 Replace startup/listener print/fatal messages with contextual info/error events while preserving existing process exit behaviour.
- [x] 2.2 Carry startup/automatic/manual trigger context through accepted refresh work and add one start/completion summary with generation, duration and aggregate outcome counts per actual run.
- [x] 2.3 Add debug scheduling decisions and per-target/component successful summaries; warn once per failed target connection or affected component, including safe reason, warning count and retention outcome; report recovery transitions at info.
- [x] 2.4 Instrument compatible snapshot restore, first-start absence, invalid/unavailable restore, successful save, persistence failure and persistence recovery at the designed levels without changing snapshot formats or memory fallback.
- [x] 2.5 Add coordinator/restore/persistence fixture tests for successful/degraded runs, failed credentials, failed partitions, recovery, optional API absence, manual coalescing/cooldown/outdated generations and failure-record bounds; verify existing request counts and retention semantics.

## 3. Handler diagnostics and legacy migration

- [x] 3.1 Add safe error events for application-owned internal handler failures, including homepage configuration and JSON encoding failures, without duplicate logging in shared response helpers or raw parser/transport errors.
- [x] 3.2 Migrate remaining application-owned `log.Print*`/`log.Fatal` calls and `debugMode` checks to levelled events; replace Pod/container/related-resource/image dumps with bounded counts and necessary operation context.
- [x] 3.3 Keep successful health/status/static/snapshot reads silent even at debug and expected validation/initial-503 responses quiet at info and above; add tests showing polling does not repeat stale warnings or trigger collection.
- [x] 3.4 Add captured-output fixtures with unique credential, URL, query, ConfigMap and raw-object markers in errors/data; verify every severity excludes them while preserving safe target/component context and concurrent JSON integrity.

## 4. Validation and documentation

- [x] 4.1 Run `go test ./...` and `go test -race ./...` from `go/`, resolve regressions and verify concurrent logger use without modifying global loggers in parallel tests.
- [x] 4.2 Run the existing required browser checks and fixture container smoke check using the repository's documented commands; inspect structured container output and unchanged startup, health and refresh behaviour. Record unavailable tooling explicitly rather than claiming a check passed.
- [x] 4.3 After implementation and validation, update README configuration and troubleshooting examples with `LOG_LEVEL`, severity meanings, JSON stderr output, `DEBUG` precedence, safe diagnostic limits and quiet polling; record the actual logging behaviour in the versioned changelog for the authorized release.
- [x] 4.4 Review implemented event coverage and fixture evidence against the logging specification; reconcile artifacts with the final behaviour and record validation evidence and the authorized release plan; cluster deployment remains outside this change.
