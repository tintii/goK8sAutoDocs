# Documentation migration and maintenance record

## Scope and baseline

Implementation started on 2026-11-11 with existing manual-link application edits, README/CHANGELOG changes, and structured-logging/versions archival work. Those changes are preserved. No application, container workflow or Dockerfile changes are part of this documentation work.

The source README had 1,026 lines. The following inventory maps every original heading, including nested headings, to its destination. Examples were kept and checked against the source; generic names replace homelab-specific examples. Release history remains in CHANGELOG.

| Original README heading | Destination / disposition |
| --- | --- |
| goK8sAutoDocs | README introduction/features/prerequisites; docs/index.md and getting-started.md |
| Features | README introduction/features/prerequisites; docs/index.md and getting-started.md |
| Prerequisites | README introduction/features/prerequisites; docs/index.md and getting-started.md |
| Installation | docs/installation.md; build/release internals retained below |
| From Source | docs/installation.md; build/release internals retained below |
| Container image | docs/installation.md; build/release internals retained below |
| Configuration | docs/configuration.md |
| Environment Variables | docs/configuration.md |
| Home page content and status | docs/configuration.md |
| Multiple clusters and label classification | docs/cluster-discovery.md |
| Rancher cluster discovery | docs/cluster-discovery.md |
| Sidebar navigation | docs/using-autodocs.md |
| Remote cluster Secret contract | docs/cluster-discovery.md; corrected exact default credential lookup in getting-started.md |
| Kubernetes Connection | docs/cluster-discovery.md; corrected exact default credential lookup in getting-started.md |
| Usage | docs/getting-started.md and using-autodocs.md |
| Running the Application | docs/getting-started.md and using-autodocs.md |
| Accessing the Dashboards | docs/getting-started.md and using-autodocs.md |
| Arrange the versions table | docs/getting-started.md and using-autodocs.md |
| Personalise your directory | docs/getting-started.md and using-autodocs.md |
| Browser settings | docs/getting-started.md and using-autodocs.md |
| API Endpoints | docs/api.md |
| GET /api/v1/health | docs/api.md |
| GET /api/v1/home | docs/api.md |
| GET /api/v1/service-status | docs/api.md |
| GET /api/v1/cluster-info | docs/api.md |
| GET /api/v1/versions-dashboard | docs/api.md |
| GET /api/v1/discovery-status | docs/api.md |
| POST /api/v1/discovery-refresh | docs/api.md |
| GET /api/v1/auto-docs-dashboard | docs/api.md |
| GET /api/v1/filtered-resources | docs/api.md |
| Resource Types Supported | docs/publishing-endpoints.md |
| Versions Dashboard | docs/publishing-endpoints.md |
| Auto-Docs Dashboard | docs/publishing-endpoints.md |
| Filtered Resources | docs/publishing-endpoints.md |
| Publication and grouping labels | docs/publishing-endpoints.md |
| URL derivation | docs/publishing-endpoints.md |
| ExternalDNS Service declarations | docs/publishing-endpoints.md |
| Migration from annotations | docs/publishing-endpoints.md |
| Kubernetes permissions | docs/installation.md#kubernetes-permissions |
| Caching | docs/operations.md |
| Development | OpenSpec migration notes below and linked existing implementation records |
| Project Structure | OpenSpec migration notes below and linked existing implementation records |
| Building | OpenSpec migration notes below and linked existing implementation records |
| Testing | OpenSpec migration notes below and linked existing implementation records |
| Security Considerations | docs/operations.md; obsolete blanket CORS claim removed (current snapshot APIs do not allow all origins) |
| Troubleshooting | docs/troubleshooting.md |
| Common Issues | docs/troubleshooting.md |
| Logging | docs/troubleshooting.md |
| Contributing | README Contributing and OpenSpec migration notes |
| License | Obsolete broken LICENSE link removed: no LICENSE exists; original GPLv3 assertion retained as unverified historical text below |
| Release history | README CHANGELOG link; CHANGELOG remains the sole release history |

## Implementation migration notes

These notes preserve README implementation context as documentation migration material, not newly completed historical work. Runtime algorithms and rationale remain in the existing [shared inventory design](../archive/2026-11-11-add-multicluster-discovery-snapshots/design.md), [Rancher design](../archive/2026-11-11-add-rancher-cluster-discovery/design.md), [ExternalDNS design](../archive/2026-11-11-add-externaldns-service-discovery/design.md), [logging design](../add-structured-level-logging/design.md), [Versions design](../improve-versions-table/design.md) and [manual-link guide](../../../docs/manual-directory-links.md).

The old README asserted “GNU General Public License v3.0” but linked to a nonexistent LICENSE. The repository currently supplies no license artifact, so this migration makes no verified licensing claim. The bundled Noto Sans font retains its separate SIL Open Font License link in the manual.

### Container build and release implementation (migrated)

Build from the repository root so the image includes the full Go module and
dashboard assets:

```bash
docker buildx build --platform linux/amd64 --load -t k8sautodocs:ci .
python3 scripts/smoke-container.py k8sautodocs:ci
```

The multi-stage build pins Go 1.24.5 and a distroless runtime by digest.
The runtime serves on 8080 as UID/GID 65532, includes CA certificates and
supports a read-only root filesystem. Its working directory is `/app`, with
the binary and `static/` tree together. The smoke check verifies the exact
packaged HTML/JS content using a temporary fixture kubeconfig, without real
cluster credentials. Kubernetes deployments use a projected ServiceAccount
token and the application's in-cluster authentication path.

GitHub Actions validates pull requests, `master` pushes, `v<semver>` tags
and manual runs. Required checks include Go tests, race tests, browser tests
with the Ubuntu runner's Chrome executable (absence fails CI), and the image
smoke check. Only successful pushes to `master` or valid release tags publish
to `ghcr.io/tintii/k8sautodocs`. The separate publication job loads and pushes
the exact tested image artifact using `GITHUB_TOKEN`; it does not rebuild or
deploy. Actions are pinned by commit and package write permission is confined
to that job.

Images have a full-commit `sha-<commit>` tag and OCI source/revision/version
labels. Releases also receive their exact `v<semver>` tag; prereleases are
accepted, build metadata and numeric prerelease identifiers with leading
zeroes are rejected. No moving `latest` tag is published. Use the digest from
the **Publish the exact tested image and report its digest** job summary to
pin a deployment; publication alone does not update GitOps values.

Push a `v<semver>` Git tag to publish a versioned image after validation. The
workflow consumes existing Git tags; it does not create tags or GitHub Release
entries. Release images currently target `linux/amd64`.

After the first publication, set the package visibility to public if cluster
pulls should be anonymous. [GHCR defaults new packages to private](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry).
Verify the selected digest can be pulled for `linux/amd64` before rollout.
If keeping it private, supply a read-only pull credential through your secret
manager and reference it with `imagePullSecrets`; never embed it in the image
or commit it in values. A package previously created outside this repository
may need its Actions repository access configured before `GITHUB_TOKEN` can
push to it.

The homelab integration, chart publication steps, anonymous pull check and
rollback commands live in the sibling
[GitOps runbook](https://github.com/tintii/homelab-gitops/blob/master/docs/k8sautodocs.md).

### Application development (migrated)

The main specs describe the [endpoint directory](../../../openspec/specs/published-endpoint-directory/spec.md),
[directory organization](../../../openspec/specs/endpoint-directory-organization/spec.md),
[shared inventory](../../../openspec/specs/shared-inventory-snapshots/spec.md),
[dashboard personalisation](../../../openspec/specs/browser-directory-personalization/spec.md),
[directory interface](../../../openspec/specs/directory-launchpad/spec.md) and
[ExternalDNS Services](../../../openspec/specs/externaldns-service-discovery/spec.md).
The main specs also cover [versions table interactions](../improve-versions-table/specs/versions-table-interaction/spec.md)
and [application logging](../add-structured-level-logging/specs/application-logging/spec.md). Historical proposals are kept in the
[OpenSpec archive](../../../openspec/changes/archive/).

### Project Structure

```
goK8sAutoDocs/
├── go/
│   ├── main.go                # Startup, versions, and existing resource relationships
│   ├── autodocs.go            # Endpoint discovery and URL resolution
│   ├── services.go            # ExternalDNS Service declarations and explicit web hints
│   ├── services_test.go       # Publication, DNS normalization, targets and web rules
│   ├── services_inventory_test.go # Service partitions, failure isolation and v1 caches
│   ├── resource_handlers.go   # Publication helpers and legacy isolated-handler tests
│   ├── inventory_config.go    # Registry, credential contract and classification
│   ├── inventory.go           # Shared coordinator, collection and persistent snapshots
│   ├── inventory_registry.go  # Dynamic reconciliation, identity probes and resolved targets
│   ├── rancher.go             # Source configuration, RK-API enumeration and proxy clients
│   ├── inventory_http.go      # Snapshot, freshness and shared update APIs
│   ├── inventory_test.go      # Multi-target, concurrency, recovery and credential fixtures
│   ├── home.go                # Configurable homepage, application health and workload status
│   ├── logging.go             # JSON severity filtering and safe response diagnostics
│   ├── logging_test.go        # Level, concurrency, quiet-reader and disclosure fixtures
│   ├── autodocs_test.go       # Kubernetes fixture and cache tests
│   ├── directory_views_test.go # Publication-state, view, and cache tests
│   ├── go.mod                # Go module file
│   └── static/               # Dashboard HTML, shared CSS/preferences, directory model and browser tests
├── scripts/                 # Container smoke check and same-origin headless browser fixtures
├── CHANGELOG.md             # Versioned release history and upgrade notes
└── README.md
```

### Building

```bash
cd go
go build -o goK8sAutoDocs .
```

### Testing

```bash
cd go
go test ./...
go test -race ./...
node --test static/*.test.cjs
```

The browser checks use headless Chrome/Chromium, a temporary loopback HTTP server and fixture responses, without contacting a cluster. They exercise real origin-scoped storage, reloads and tab updates, plus desktop/mobile light/dark layouts and contrast. Set `CHROME_BIN` to its executable if it is not found in a standard location. Browser checks are skipped when no executable is available; CI requires the executable. Node 22 or newer supplies the browser harness's built-in WebSocket client.

Optionally capture review screenshots outside the checkout:

```bash
AUTODOCS_SCREENSHOTS=/tmp/k8sautodocs-ui-review node --test go/static/autodocs-dashboard.test.cjs
```

## Documentation maintenance

Use **Python 3.12.12**, the same interpreter as `.github/workflows/docs.yaml`.
MkDocs **1.6.1** and every transitive dependency are locked with package hashes
in `requirements-docs.txt`. The direct input is `requirements-docs.in`.
The versions were resolved from PyPI and installed successfully on 2026-11-11.

From the repository root:

```bash
python3.12 -m venv .venv-docs
. .venv-docs/bin/activate
python -m pip install --require-hashes -r requirements-docs.txt
python scripts/check-docs.py
python -m mkdocs serve --dev-addr 127.0.0.1:8000
```

Verify `python --version` reports 3.12.12. If that interpreter is unavailable,
`uv venv --python 3.12.12 .venv-docs` supplies the pinned interpreter, then use
the same activation/install/check commands. Preview at
`http://127.0.0.1:8000/goK8sAutoDocs/`, including direct nested URLs such as
`http://127.0.0.1:8000/goK8sAutoDocs/cluster-discovery/`.

`python scripts/check-docs.py` is the local and CI acceptance command. It runs
negative fixtures for missing pages, anchors, assets, CSS assets and escaped
project paths, builds once with `mkdocs build --strict`, then inspects generated
HTML links/anchors and HTML/CSS assets under the configured project path. Strict
MkDocs warnings are failures. External links are not fetched in CI. A change to
the project URL must update both `mkdocs.yml` and the checker's `BASE`.

To update tooling, change `requirements-docs.in`, regenerate the whole lock,
review the versions/hashes and rerun checks and desktop/mobile preview:

```bash
uv pip compile --python-version 3.12 --universal --generate-hashes requirements-docs.in -o requirements-docs.txt
```

`site/`, `.venv-docs/` and Python bytecode are ignored. The existing
`.dockerignore` begins with `**` and allows only Dockerfile and runtime `go/`
inputs: docs, output, config, requirements and scripts are already excluded.
No Dockerfile or container-workflow changes were necessary.

## GitHub Pages setup and publication

Read-only public GitHub API checks on 2026-11-11 reported repository visibility
`public`, default branch `master`, and `has_pages: false`. Both `/pages` and
`/environments/github-pages` returned 404. There is no confirmed Pages site or
environment configuration; private administration settings were not inspected.
Public repositories are eligible for Pages on GitHub Free. The expected project
URL is `https://tintii.github.io/goK8sAutoDocs/`, configured for future deployment;
it is not a verified live URL. No repository settings have been changed.

The workflow uses the [official custom Pages workflow contract](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages)
and [MkDocs project-path configuration](https://www.mkdocs.org/user-guide/configuration/#site_url).
Only the deploy job receives `pages: write` and `id-token: write`. It depends on
successful build and upload, uses environment `github-pages`, reports the action's
`page_url`, and serializes deployments in concurrency group `github-pages` with
`cancel-in-progress: false`. Configure/upload and deployment have the same
master-only push/dispatch gate. PR validation has no Pages calls. Configure Pages
uses `enablement: false` so a missing setup fails rather than attempting to change
repository settings from the read-only build job.

Verified action tag refs from the GitHub API on 2026-11-11:

| Action | Tag | Commit SHA |
| --- | --- | --- |
| actions/checkout | v4 | 11d5960a326750d5838078e36cf38b85af677262 |
| actions/setup-python | v5 | a26af69be951a213d495a4c3e4e4022e16d87065 |
| actions/configure-pages | v5 | 983d7736d9b0ae728b81ab479565c72886d7745b |
| actions/upload-pages-artifact | v4 | 7b1f4a764d45c48632c6b24a0339c27f5614fb0b |
| actions/deploy-pages | v4 | d6db90164ac5ed86f2b6aed7e0febac5b3c0c03e |

The upload action's pinned composite source also pins its nested upload-artifact
action. The checked `site/` is uploaded without a rebuild. Documentation never
requires image publication, Kubernetes access or deployment secrets.

### First authorized publication checklist

1. Review the documentation changes independently of the pre-existing application
   and archive work. Obtain authorization before committing/pushing; no commit or
   push has been performed for this change.
2. In repository **Settings → Pages → Build and deployment**, set **Source** to
   **GitHub Actions**. Confirm the `github-pages` environment permits deployments
   only from `master` (selected branch rule). Retain any required reviewers.
3. Push the reviewed revision to `master`, or dispatch **Validate and publish user
   manual** on `master` after the workflow is on that branch. Check the build,
   artifact upload and deployment result. Missing Pages settings should fail at
   the named setup step; environment rejection should fail/wait at deployment.
4. Open the deploy action's returned `page_url`. Verify the landing page and a
   direct `/cluster-discovery/` URL; check navigation, a real search query,
   desktop/mobile layout and CSS/JS/font/search assets under `/goK8sAutoDocs/`.
5. Record run URL, revision, returned URL and hosted verification in `validation.md`.
   Only after this succeeds, add the public manual link to README while retaining
   `docs/index.md`, and complete tasks 5.2 and 5.3. A build or guessed URL is not
   evidence of publication.

To roll back documentation, revert the documentation/workflow changes through a
reviewed `master` revision and rerun the docs workflow on `master`. To restore a
known-good manual, restore those docs/config/lock/workflow files to `master` and
publish that revision. Dispatching an old tag or non-master branch cannot deploy.
The application image, GitOps pins and running cluster are unaffected.

## Publication authorization and confirmed setup

On 2026-11-11 the user confirmed that Pages can publish from Actions and
explicitly authorized commit, tag and push. Authenticated GitHub API checks
confirmed `build_type: workflow`, source branch `master`, project URL
`https://tintii.github.io/goK8sAutoDocs/`, and a `github-pages` environment with
one custom deployment branch policy permitting `master`. The earlier public
404 response did not reflect the authenticated configuration and is retained
above only as the initial observation.

The publication commit is scoped to documentation and tooling. The uncommitted
manual-link implementation and unrelated archival/specification changes remain
in the workspace. The manual-link guide is included because the manual links to
it, and is explicitly marked unreleased. Its design/spec archive belongs to the
separate feature change. Existing tagged Rancher commit v0.6.0-RC1 is an ancestor
of this publication, while remote master initially pointed at v0.5.2. Publication
will fast-forward master; it will not rewrite history. The next release tag is
v0.6.0-RC2. Hosted evidence follows in validation.md after deployment succeeds.

## Verified first publication

The initial documentation commit `98a0902` was pushed to master after explicit
user authorization. [Pages workflow 43050456155](https://github.com/tintii/goK8sAutoDocs/actions/runs/43050456155)
completed build and deployment successfully and reported
<https://tintii.github.io/goK8sAutoDocs/>. Live desktop/mobile navigation, nested
loads, search and assets were verified before adding the public link to README.
Tasks are now 18/18 complete; detailed evidence and limitations are in
[validation.md](validation.md). The final evidence/link update is released as
`v0.6.0-RC2`. Manual-link application and archival changes remain separate.
