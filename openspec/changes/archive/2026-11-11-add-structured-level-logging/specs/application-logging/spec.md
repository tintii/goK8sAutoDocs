## ADDED Requirements

### Requirement: Configurable minimum severity with legacy compatibility

The application SHALL support `debug`, `info`, `warn` and `error` through `LOG_LEVEL`, normalized by trimming whitespace and ignoring case. An unset or blank `LOG_LEVEL` SHALL select `debug` when `DEBUG` is exactly `true`, and `info` otherwise. A nonblank explicit `LOG_LEVEL` SHALL take precedence. Only records at or above the selected severity SHALL be emitted. An unsupported level SHALL produce a structured error explaining the allowed levels, without echoing the supplied value, and terminate startup nonzero before inventory collection or listener startup.

#### Scenario: Default and legacy configuration
- **WHEN** `LOG_LEVEL` is unset or blank
- **THEN** the effective level is `info` unless `DEBUG=true`, in which case it is `debug`

#### Scenario: Explicit level overrides legacy debug
- **WHEN** `DEBUG=true` and `LOG_LEVEL` is ` WARN `
- **THEN** warning and error records are emitted and debug and info records are suppressed

#### Scenario: Invalid level fails early
- **WHEN** a nonblank unsupported `LOG_LEVEL` is supplied
- **THEN** startup exits nonzero with a structured error listing the supported levels
- **AND** the supplied value is absent from output and no inventory job or HTTP listener starts

### Requirement: Consistent structured application records

Application-owned logs SHALL use one JSON object per line on stderr, including timestamp, severity, message and a stable application-owned `event` identifier. Operational records SHALL include applicable cluster ID, component, generation, trigger, duration and aggregate counts as structured attributes. Initialization SHALL precede configuration loading and snapshot restore. Application-owned legacy print/fatal calls and raw debug dumps SHALL be migrated to this logging policy.

#### Scenario: Refresh record is machine-readable
- **WHEN** an accepted refresh completes
- **THEN** stderr contains a parseable JSON completion record with severity, event, generation, trigger, elapsed duration and aggregate outcome counts
- **AND** no application log record is written to stdout

#### Scenario: Early failure uses the same format
- **WHEN** application configuration is invalid before inventory service construction
- **THEN** an error record with a fixed safe reason is emitted using the same JSON structure before nonzero termination

### Requirement: Meaningful lifecycle and refresh events

At `info` severity the application SHALL report successful startup configuration summary, listener startup, accepted refresh start/completion, usable snapshot restore and recovery from previously failing collection or persistence. Refresh events SHALL distinguish startup, automatic and manual triggers. At `debug` severity it SHALL provide per-target/component success summaries, refresh scheduling decisions, first-start snapshot absence and successful persistence without per-resource dumps.

#### Scenario: Successful multi-cluster refresh
- **WHEN** a scheduled run successfully collects several configured targets
- **THEN** one run-start and one run-completion info record identify the generation and automatic trigger
- **AND** debug collection summaries identify their target/component and counts without logging individual inventory objects

#### Scenario: Concurrent manual refresh is coalesced
- **WHEN** a manual refresh request arrives while a run is active, during cooldown or with an outdated observed generation
- **THEN** a debug decision record identifies the applicable reason
- **AND** no extra refresh start/completion records imply a run that was not accepted

#### Scenario: Collection recovers
- **WHEN** a component that failed on its previous attempt succeeds
- **THEN** an info recovery event identifies its cluster and component
- **AND** an already healthy component does not emit a recovery event

### Requirement: Recoverable failures and fatal errors use distinct levels

Recoverable target connection, collection, partial-result, snapshot restore and snapshot persistence failures SHALL emit warnings with safe reason categories and applicable retention context. Target connection failure SHALL produce at most one target warning per run rather than one identical warning per partition. Component outcomes SHALL produce at most one warning per component per run with a warning count rather than one record per affected resource. Invalid startup configuration, unexpected listener failure and internal HTTP handler failures SHALL emit errors. Expected absence of the optional Gateway API SHALL NOT emit warning/error records.

#### Scenario: Failed target retains previous inventory
- **WHEN** credentials cannot be loaded for a target with previously successful inventory
- **THEN** one target warning identifies the target, safe failure reason and whether previous data is retained
- **AND** healthy targets continue refreshing under the existing failure-isolation contract

#### Scenario: Component collection fails
- **WHEN** workload listing fails while endpoint collection succeeds
- **THEN** a component warning identifies the affected target/component and retention outcome
- **AND** the run summary reports degraded collection without classifying the entire application as fatally failed

#### Scenario: Cache cannot be restored or persisted
- **WHEN** a saved snapshot is invalid or a new snapshot cannot be written
- **THEN** a warning identifies the restore or persistence operation and a safe reason
- **AND** the existing in-memory/startup fallback behaviour remains available

#### Scenario: Optional API is absent
- **WHEN** the HTTPRoute API is not installed and other collection succeeds
- **THEN** no warning/error record reports its expected absence
- **AND** a debug record can explain that the optional source was skipped

#### Scenario: Homepage configuration cannot be served
- **WHEN** the homepage handler returns HTTP 500 for invalid mounted YAML
- **THEN** exactly one application error record identifies the operation and a fixed configuration-failure reason without YAML contents or raw parser errors

### Requirement: Routine requests do not amplify logs

Successful health/status polling, static requests and ordinary snapshot reads SHALL emit no application log records, including at debug level. Known stale collection failures SHALL be logged at collection boundaries rather than for every snapshot reader. Expected request validation failures and initial inventory-not-ready 503 responses SHALL remain quiet at info and above. Instrumentation SHALL preserve HTTP status/payload contracts, refresh coalescing, deadlines and stale-partition retention and SHALL NOT initiate additional Kubernetes requests.

#### Scenario: Several browsers poll stale inventory
- **WHEN** multiple browsers repeatedly read health, discovery status and retained stale snapshots
- **THEN** these reads do not repeat the collection warning or generate successful-request records
- **AND** they do not trigger extra Kubernetes collection

#### Scenario: Inventory is not yet available
- **WHEN** a reader requests inventory before any usable initial snapshot exists
- **THEN** the existing HTTP 503 behaviour is preserved without an info, warning or error record for each request

### Requirement: Safe bounded context at every severity

Records SHALL use application-owned messages and safe categorized reasons rather than arbitrary upstream error strings. Logs SHALL exclude kubeconfig/configuration contents, tokens, certificates, private keys, authorization headers, bodies, ConfigMap values, raw Kubernetes objects, full endpoint URLs and query strings at every severity. Debug output SHALL use bounded counts or necessary resource identity instead of complete Pod/container/related-resource dumps. Logging SHALL remain safe under concurrent collection and SHALL NOT require new cluster permissions or external services.

#### Scenario: Upstream error contains a credential marker
- **WHEN** a fixture error contains a unique token, credential-bearing URL or parser payload
- **THEN** captured output at debug and all higher thresholds contains only a safe reason and allowed operational context
- **AND** none of the injected sensitive markers appear in the records

#### Scenario: Collected objects contain sensitive values
- **WHEN** inventory fixtures include ConfigMap values or object metadata containing secret markers
- **THEN** collection and failure logs report counts and owned categories without serializing those object values

#### Scenario: Concurrent operations emit valid records
- **WHEN** refreshes and HTTP handlers log concurrently
- **THEN** each emitted line remains valid JSON and logger/context handling has no data race
