## ADDED Requirements

### Requirement: Reproducible documentation validation
Documentation SHALL build using pinned tooling and a documented local command shared with CI. Validation SHALL fail for build errors, missing internal pages, broken anchors and missing local assets. The build SHALL need no production credentials or running cluster and SHALL include only the manual and intended static assets.

#### Scenario: Pull request breaks a manual link
- **WHEN** a pull request introduces a missing internal page or anchor
- **THEN** documentation validation fails and no Pages deployment runs

#### Scenario: Validate a fork contribution
- **WHEN** a fork pull request runs the docs workflow
- **THEN** it builds and validates with read-only repository permissions and without deployment secrets or Pages administration access

### Requirement: Publish validated master artifacts
GitHub Actions SHALL publish the exact checked documentation output after a successful push to `master` or manual dispatch on `master`. Deployment SHALL depend on successful build and artifact upload, use the `github-pages` environment, report the deployment URL and serialize competing publications.

#### Scenario: Publish a manual update
- **WHEN** a push to `master` passes documentation validation and Pages is configured
- **THEN** the deploy job publishes the same output that was validated and reports its public URL

#### Scenario: Reject ineligible publication
- **WHEN** a run originates from a PR, a non-master branch, a tag or a failed build
- **THEN** it cannot publish to GitHub Pages

#### Scenario: Manually republish master
- **WHEN** a maintainer dispatches the docs workflow on `master`
- **THEN** it rebuilds and validates that revision before deploying its checked artifact

### Requirement: Limit publication permissions and preserve application delivery
Only the Pages deploy job SHALL receive `pages: write` and `id-token: write`; workflow defaults SHALL be read-only. Actions SHALL be pinned to verified commit SHAs. Documentation delivery SHALL preserve existing container tests, release conditions and GHCR artifact publication, and SHALL NOT require cluster access or bundle documentation tooling into the runtime image.

#### Scenario: Review workflow boundaries
- **WHEN** a maintainer reviews the implemented workflows and Docker context
- **THEN** PR jobs have no Pages deployment rights, container publication retains its existing rules and runtime contents, and docs output/tooling are excluded from the runtime build context

### Requirement: Verify initial Pages setup and delivery
The change's maintainer record SHALL document repository eligibility/settings, the GitHub Actions publishing source, allowed deployment branch, local preview/build commands and first-deployment verification. Documentation SHALL NOT claim the site is live until its actual deployment URL has been checked.

#### Scenario: First hosted publication
- **WHEN** the workflow is enabled and the first authorized deployment completes
- **THEN** the maintainer records a successful public landing page, nested-page navigation, search and asset check at the returned project URL

#### Scenario: Repository Pages setup is missing
- **WHEN** a validated build cannot publish because Pages is not enabled or its environment rejects the branch
- **THEN** the run exposes the deployment failure and the maintainer checklist identifies the settings needed without treating the build as a successful publication
