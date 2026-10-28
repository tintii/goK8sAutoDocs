# Apply validation

- Go 1.24.5 race tests: passed, including registry/credential loading, classification, request coalescing, stale retention, persisted compatibility and actual multi-target fake-client collection.
- Complete JavaScript/browser suite: 75 passed, zero failures/skips. The focused inventory suite also passed after checking that same-origin application endpoint destinations remain unchanged by scope navigation.
- Native ARM image build and runtime smoke test: passed with two independent self-contained TLS kubeconfigs, same internal context names, namespace classification, zero Kubernetes requests from inventory GETs, coalesced refresh, restart hydration after fixture API failure, original collection timestamps and no token in the saved file.
- Real mobile browser inspection: scope controls, freshness, grouping and cards fit 375px without document overflow. Existing desktop/light/dark/version table/home-favourites fixtures passed.
- Strict OpenSpec validation and Git whitespace checks: passed.

Publication/native amd64 pipeline and live rollout evidence will be appended after they complete. Only synthetic clusters were used for local authentication tests. Real additional target credentials and production label keys remain external onboarding inputs.
