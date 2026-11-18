## 1. Inventory and verify documentation

- [x] 1.1 Preserve current README/CHANGELOG edits and unrelated OpenSpec archival work; map every README section to a manual page, CHANGELOG, OpenSpec record or explicit obsolete-content removal.
- [x] 1.2 Verify the local-source quick start, toolchain, credential lookup, working directory, port and first-use result against current code; identify stale claims and broken existing links.
- [x] 1.3 Check repository Pages eligibility, expected project URL and current settings without claiming publication; select and verify exact documentation dependency versions and action SHAs.

## 2. Write the user manual and simplify README

- [x] 2.1 Create `docs/index.md`, getting-started and installation guides with local, container and Kubernetes paths, permissions and expected results.
- [x] 2.2 Write task-based dashboard use, configuration, cluster discovery and endpoint-publication pages, preserving working examples and current operational constraints.
- [x] 2.3 Write operations, troubleshooting and API reference pages covering freshness, persistence, upgrades/rollback, logging and observable failures.
- [x] 2.4 Link existing implementation records and preserve missing rationale in identified OpenSpec migration notes; record documentation build/publishing maintenance instructions in this change.
- [x] 2.5 Reduce README to at most 150 lines with a verified quick start and source-manual links; update repository references to moved sections and CHANGELOG without duplicating release history.

## 3. Build and validate the site

- [x] 3.1 Add MkDocs configuration, explicit navigation/search, project base URL and pinned documentation dependencies; ignore generated output and exclude docs/tooling from Docker context.
- [x] 3.2 Add one local/CI validation command using strict build and checks for generated internal page, fragment and asset links; confirm broken-link fixtures fail the check.
- [x] 3.3 Preview the generated manual under `/goK8sAutoDocs/`; check desktop/mobile navigation, search and nested-page direct loading, and inspect the output for unintended files or private example data.

## 4. Add GitHub Pages delivery

- [x] 4.1 Create the documentation workflow with read-only validation for PRs, `master` pushes and manual runs, using verified SHA-pinned actions and pinned build tooling.
- [x] 4.2 Upload the exact checked site artifact only for eligible master runs and add a dependent deploy job with Pages-only write permissions, the `github-pages` environment, deployment URL and serialized publication concurrency.
- [x] 4.3 Validate workflow syntax and the PR/fork, master push, manual master, manual non-master, tag and failed-build cases; verify the container workflow's test/release behavior and runtime content remain unchanged.
- [x] 4.4 Record one-time Pages settings, branch/environment rules, local maintenance commands and publication/rollback steps in this change's implementation record.

## 5. Verify and publish

- [x] 5.1 Run documentation acceptance checks, check all migrated sections and execute the documented quick start in a clean environment with suitable test credentials; record results and limitations.
- [x] 5.2 After an authorized push, enable or confirm GitHub Actions as the Pages source, publish through the workflow and verify landing/nested pages, navigation, search and assets at its returned URL.
- [x] 5.3 Add the verified public manual link to README, retain the source-manual link and record hosted verification evidence before marking publication complete.
