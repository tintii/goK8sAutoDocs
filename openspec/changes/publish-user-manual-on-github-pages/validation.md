# Validation: user manual and Pages workflow

Date: 2026-11-11. Local implementation is complete through task 5.1:
**16/18 tasks complete**. At this initial validation stage, hosted publication and its README public link
were pending (5.2, 5.3). The later authorized publication is recorded below.

## Local results

| Check | Result |
| --- | --- |
| Pinned build environment | Python 3.12.12; MkDocs 1.6.1; complete transitive/hash lock installed successfully |
| Shared acceptance command | `.venv-docs/bin/python scripts/check-docs.py` passes strict build and generated HTML/CSS page, anchor and asset checks |
| Negative emitted-site fixtures | Missing page, missing fragment, missing asset, CSS missing asset and two escaped project paths all rejected; valid fragment accepted |
| Source Markdown | 72 local file/directory/fragment links checked across README, docs and implementation record; all resolve |
| Migration coverage | Every original README heading mapped in implementation.md; operational examples retained, implementation detail moved to OpenSpec, release history stays in CHANGELOG |
| README length | 56 lines (limit 150); verified prerequisites, source quick start, first-use result and source-manual links |
| Workflow syntax | `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 .github/workflows/docs.yaml` passes |
| OpenSpec | `openspec validate publish-user-manual-on-github-pages --strict` passes |
| Working-tree whitespace | `git diff --check` passes |
| Runtime scope | Baseline and final diffs for `go/`, Dockerfile, `.dockerignore` and container workflow match byte-for-byte; pre-existing application changes preserved |
| Docker context | Existing deny-all allowlist admits only Dockerfile and runtime go inputs; documentation and its tooling/output are excluded |
| Output inspection | Eleven manual pages plus 404; only theme CSS/JS/fonts/images, search files and sitemaps; no application code, config files, OpenSpec files, credentials or raw inventory |
| Example inspection | Generic hostnames/placeholders; no PEM bodies, private homelab IP/domain values, user paths or fixture tokens in manual sources |

Action SHAs were verified against official GitHub tag refs; the pinned Pages
composite source was inspected. The MkDocs configuration and GitHub workflow
behavior were checked against upstream documentation linked in implementation.md.
External services are not probed by the CI link checker, so external link
availability remains outside its guarantee.

## Browser verification under the project path

The exact generated `site/` directory was served at a loopback origin under
`/goK8sAutoDocs/`, without rewriting its generated paths. Real headless Chrome
checked **1440×1000** and **390×1000** viewports:

- Landing page and direct `/cluster-discovery/` loads work.
- Styles apply; desktop navigation and mobile expansion work.
- Setup dropdown → Installation navigates to the generated nested page.
- Search for `credential` returns real Lunr results on both sampled pages at
  both widths; result destinations remain under `/goK8sAutoDocs/`.
- Search worker and search index load from the subpath. No requested asset
  returned 404. No horizontal document overflow at the sampled widths.
- Desktop landing and mobile nested screenshots were inspected. Grouped top
  navigation keeps the desktop bar to 56 pixels; long content is readable on
  mobile. Screenshots and the temporary CDP harness are outside the repository.

MkDocs preview also started successfully at
`http://127.0.0.1:8000/goK8sAutoDocs/`. This is local preview evidence, not hosted
verification.

## Source quick-start exercise

Used a fresh temporary copy of the current Go module and static assets, an
isolated child-process home containing only a generated `~/.kube/config`, and a
local TLS Kubernetes fixture with embedded CA and bearer credentials. Production
credentials and user kubeconfigs were neither read nor modified. Existing public
Go module/build caches were reused; the application toolchain is Go 1.24.5.

The documented `go mod download`, `go build -o goK8sAutoDocs .` and
`./goK8sAutoDocs` commands succeeded from the temporary `go/` directory. The
application served health, HTML and JS/CSS assets on `localhost:8080`, and used
the default `local` target from `~/.kube/config`. Collected one workload, three
published endpoint declarations and one discovered Service, through 14 HTTPS
fixture calls. The process and fixture were stopped and temporary credentials
removed. This tests the command, static-asset, credential-lookup and first-use
contracts; it does not establish permissions or connectivity for a user's real
cluster. Network Git clone itself was not needed for the isolated source test.

## Workflow event/permission checks

Evaluated the actual YAML's configure/upload/deploy expressions for these cases:

| Event/ref | Expected and observed eligibility |
| --- | --- |
| Pull request / fork PR merge ref | Validation only; no publication |
| Pull request with master ref | No publication |
| Push to master | Checked artifact upload and dependent deployment |
| Manual dispatch on master | Checked artifact upload and dependent deployment |
| Manual dispatch on another branch | Validation only; no publication |
| Push tag | Not triggered; deploy expression also false |
| Push non-master | Not triggered; deploy expression also false |
| Failed build | Normal step success gating prevents upload; `needs: build` with no `always()` prevents deployment |

Assertions also checked workflow-default `contents: read`, no build-job write
permissions, deploy-only `pages: write`/`id-token: write`, `github-pages`
environment and URL, serialized deploy concurrency, identical eligibility gates
and upload of `site/` after the sole build/check command. These are local static
and expression checks; no GitHub workflow run has been dispatched for this
uncommitted change. The container workflow is unchanged, preserving its tests,
tag validation, exact-image artifact handoff and GHCR publication permissions.

## Hosted verification still required

The public repository API reports `has_pages: false`; Pages and github-pages
environment API requests return 404. Public-repository eligibility and expected
project URL are established, but no settings or real deployment URL have been
verified. Follow the first-publication checklist in implementation.md after
commit/push authorization, configure Actions as the Pages source, enforce master
in the environment, publish and record the returned URL and hosted checks here.
Only then complete tasks 5.2/5.3 and add the public manual link to README. Keep
the source-manual link working throughout.
