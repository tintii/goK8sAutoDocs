# Apply validation

- Go 1.24.5 race tests: passed, including registry/credential loading, classification, request coalescing, stale retention, persisted compatibility and actual multi-target fake-client collection.
- Complete JavaScript/browser suite: 75 passed, zero failures/skips. The focused inventory suite also passed after checking that same-origin application endpoint destinations remain unchanged by scope navigation.
- Native ARM image build and runtime smoke test: passed with two independent self-contained TLS kubeconfigs, same internal context names, namespace classification, zero Kubernetes requests from inventory GETs, coalesced refresh, restart hydration after fixture API failure, original collection timestamps and no token in the saved file.
- Real mobile browser inspection: scope controls, freshness, grouping and cards fit 375px without document overflow. Existing desktop/light/dark/version table/home-favourites fixtures passed.
- Strict OpenSpec validation and Git whitespace checks: passed.

Only synthetic clusters were used for local authentication tests. Real additional target credentials and production label keys remain external onboarding inputs.


## Publication and live deployment

- Final executable source: 386cb1835e56e0fb615bc63b91a0ad50fb1a72d8. Native amd64 pipeline https://github.com/tintii/goK8sAutoDocs/actions/runs/40716254138 completed successfully, including Go/race, all 75 browser checks and the authenticated two-target TLS/runtime recovery smoke test.
- Published image: ghcr.io/tintii/k8sautodocs@sha256:5e130fc2639a3d63add107a2d477c18ca7786803df656195b5a135aa2bbf0fcc. Anonymous pull verified both amd64 architecture and OCI source revision.
- GitOps release 830303d and registry revision/recovery rollout 6227850 deployed through Argo using published workload chart 0.4.0. An explicitly documented, dry-run-verified one-time strategy migration removed previous RollingUpdate defaults; desired state remains chart-managed Recreate.
- Live HTTPS: all three pages and seven assets match source. Inventory: 1 published source, 5 discovered sources, 43 workloads and 63 containers. Two simultaneous requests yielded exactly one new generation.
- Real browser: availability, shared progress, composite grouping, favourites on the overview, and 375px layout verified. Same-origin application links retain their exact resolved destination.
- Persistent 1 GiB TrueNAS iSCSI/ext4 claim is bound and writable by the non-root app. A GitOps pod replacement continued generation 2 to 3 without cache/collection errors; original-timestamp retention during API failure was verified by the container recovery fixture.
- One pod is ready; route Accepted and ResolvedRefs are true. Required reads permitted; Secret reads and Deployment writes denied. Argo operation succeeded and health is Healthy; pre-existing HTTPRoute backend-default drift remains OutOfSync.
- Only the hosting target homelab is configured. Real environment/datacentre keys were absent, so classification remains unconfigured with best-effort fallbacks. Remote onboarding uses the documented one-Secret-per-cluster contract.
