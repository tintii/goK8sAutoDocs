# Validation and release handoff

Validated on 2026-10-29 using fixture data only; no live cluster or credentials were used.

- `go test ./...`: passed, including severity/legacy precedence, structured early startup failures, per-target/component failure bounds, recovery, concurrent scheduling, quiet readers, snapshot failures and sensitive-marker exclusion.
- `go test -race ./...`: passed.
- `go vet ./...`: passed.
- `node --test go/static/*.test.cjs`: 78 passed, zero failed/skipped.
- `docker buildx build --platform linux/amd64 --load -t k8sautodocs:ci .`: passed.
- Local emulated amd64 smoke run: failed in Go runtime `netpoll` during startup with SIGSEGV on the arm64 Docker host. No application workaround or change to production architecture was made.
- `docker buildx build --platform linux/arm64 --load -t k8sautodocs:ci-arm64 .` and `python3 scripts/smoke-container.py k8sautodocs:ci-arm64`: passed. Verified actual packaged assets, two TLS fixture targets, isolated GETs, manual coalescing, persistent recovery, stale retention, JSON stderr logs, trigger context, quiet successful reads, credential exclusion, CA certificates and non-root/read-only execution.
- `openspec validate add-structured-level-logging --strict`: passed.

The user authorized implementation, README/changelog updates, commit, tag and push. The existing remote `v0.5.0` points at the preceding source revision; the next release is `v0.5.1`. Accumulated history now lives in `CHANGELOG.md`, with v0.5.0 features recorded separately from the logging release. No history/tag rewrite or cluster deployment is planned.

Release sequence: commit and push reviewed changes to `master`, require successful native linux/amd64 GitHub Actions validation/publication, then create and push annotated `v0.5.1` at that same source revision. The tag workflow performs its own required checks before publishing the versioned image. Native CI results and publication must be verified from the actual workflow runs; the local emulation failure is not reported as a passing amd64 runtime check.
