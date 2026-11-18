## Context

`README.md` currently has 1,021 lines, including installation, detailed environment/configuration examples, Rancher discovery, dashboard usage, API payloads, publication labels, caching, source structure, build/test commands and troubleshooting. There is no `docs/` manual or Pages workflow. `.github/workflows/container.yaml` validates PRs, `master` pushes, release tags and manual runs, then publishes tested container artifacts on eligible pushes. Existing application specifications and archived changes already capture much of the implementation rationale.

The README and CHANGELOG have local edits, and two unrelated OpenSpec changes are being archived. Preserve that work. Only this application's documentation and documentation delivery are in scope.

## Goals / Non-Goals

**Goals:** Make first use straightforward; preserve operational detail in a navigable user manual; keep design history in OpenSpec; validate the manual before publishing it through GitHub Actions.

**Non-Goals:** Change application behavior or UI, deploy AutoDocs, change container release semantics, add a custom domain, host private cluster data, or create a versioned documentation portal. The initial manual follows `master` and identifies unreleased behavior where applicable.

## Decisions

### 1. Give the README one short path to first use

Target at most 150 lines, including commands. Include a brief description, a few core features, prerequisites for the recommended path, one copyable local-source quick start, the URL to open and what users should expect to see. Derive the Go requirement from `go/go.mod`, verify the actual kubeconfig lookup, and run from `go/` so relative static assets work. Link container and Kubernetes installation alternatives in the manual instead of duplicating full instructions. Do not make the quick start run the development test suite or require the homelab repositories.

Keep short links to configuration, troubleshooting, CHANGELOG, contribution information and OpenSpec changes. Verify existing links, including the current LICENSE link, before carrying them forward; do not invent a missing license artifact. Add the public manual link once its URL is verified, with a working repository `docs/index.md` link available from the initial migration.

Alternative considered: a table of contents over the current README. It would improve navigation but retain the mixed audience and duplication.

### 2. Organize the manual around user tasks

Use Markdown under `docs/` with an explicit navigation tree:

| Manual area | Content and existing material to migrate |
|---|---|
| Start here (`index.md`, `getting-started.md`) | Purpose, prerequisites, recommended first run and first successful inventory |
| Installation (`installation.md`) | Local source, published container selection, Kubernetes ServiceAccount/RBAC, credential mounts and persistence |
| Using AutoDocs (`using-autodocs.md`) | Applications/Discovered, search, scope, grouping, favourites, browser settings, Versions sorting/resizing and optional Overview |
| Configuration (`configuration.md`) | Supported environment variables, defaults, home content, logging and restart requirements |
| Connecting clusters (`cluster-discovery.md`) | Direct registry entries, Rancher sources, canonical bindings, permissions, credential rotation and diagnostics |
| Publishing endpoints (`publishing-endpoints.md`) | Labels, Ingress/HTTPRoute and ExternalDNS Service examples, DNS-only entries, explicit web hints and grouping |
| Operating AutoDocs (`operations.md`) | Refresh/freshness, persistent cache, single-active-instance requirement, upgrades, compatibility and rollback |
| Troubleshooting (`troubleshooting.md`) | Symptoms, diagnostic steps, expected outcomes, connectivity, RBAC, stale inventory and logs |
| API reference (`api.md`) | Existing endpoints, parameters, response examples and observable failure behavior |

Split a page further only when its size impedes finding a task. Readers must be able to complete setup and normal operation without reading OpenSpec. Operational constraints and externally visible behavior belong in the manual even when technical; source-file walkthroughs, algorithms, design alternatives and test execution history do not.

Create a section-by-section migration inventory in this change's implementation record. Map every existing README section to a manual page, CHANGELOG, an appropriate OpenSpec change or an explicit obsolete-content removal. Preserve rationale already recorded in archived changes by linking it; add any missing historical context as an identified migration note rather than rewriting completed history as new work. Record documentation tooling and publishing internals in this change's design/validation record. Do not create a developer-internals chapter in the manual.

Alternative considered: moving the README wholesale into `docs/index.md`. That would preserve the same usability problem and fail to separate the audiences.

### 3. Use a small static Markdown build

Add root `mkdocs.yml`, an exact-version documentation dependency lock/requirements file, and the standard MkDocs theme with search. Avoid a frontend application or custom theme. Set `docs_dir` to `docs`, generate only `site/`, and ignore generated output. Pin and verify the Python/tool versions during implementation. Provide a single local validation command and document preview/build commands in this OpenSpec change's maintainer record.

The expected project URL is `https://tintii.github.io/goK8sAutoDocs/`; verify it against repository Pages settings and deployment output before declaring it live. Configure the `/goK8sAutoDocs/` base path, use relative internal page/asset links, and use absolute GitHub URLs for links outside the manual, such as CHANGELOG or OpenSpec. No sibling checkout paths may appear in published navigation.

Run `mkdocs build --strict` with link-validation settings that treat missing pages and anchors as failures. Add a generated-site check where needed to cover emitted internal URLs, assets and fragments; strict mode alone must not be assumed to check everything. Inspect the output inventory to ensure only the manual and intended assets are shipped. External links are reviewed without making CI depend on every third-party service being online.

Alternative considered: Jekyll or a bespoke site. MkDocs directly supports Markdown navigation and search with a small configuration; the application has no existing documentation generator to preserve.

### 4. Add an independent docs workflow

Create `.github/workflows/docs.yaml`. Run a read-only build/check job on all PRs, pushes to `master` and manual dispatches, avoiding path filters initially so documentation-affecting source changes cannot silently skip validation. This job installs pinned tooling, builds once and checks the output without cluster credentials, a container image or Pages setup. Fork PR validation must work without deployment permissions. Do not use `pull_request_target`.

After successful checks, eligible `master` pushes and manual dispatches on `refs/heads/master` configure Pages and upload that exact checked `site/` directory with the official Pages artifact action. A separate deploy job depends on the build/artifact job and has the same event/ref restriction. Only deploy receives `pages: write` and `id-token: write`; default permissions are `contents: read`. Use the `github-pages` environment, its deployment URL output and serialized publication concurrency so deployments cannot race. PRs, non-default branches, tags and failed builds never deploy. Pin actions to verified commit SHAs during implementation.

Keep the container workflow's checks, tag rules, artifact handoff and GHCR permissions intact. Exclude `docs/`, `site/`, documentation configuration/dependencies and tooling from the Docker build context as appropriate, without changing runtime contents. A docs build must not depend on successful image publication or introduce Kubernetes access.

Alternative considered: appending Pages publication to the container release job. Independent workflows allow documentation publication without coupling it to image builds or granting Pages rights to container jobs. Branch-based `mkdocs gh-deploy` is unnecessary because Actions can deploy the built artifact directly.

### 5. Treat first publication as a verified setup step

Record a maintainer checklist here: confirm repository Pages eligibility, set Settings → Pages → Build and deployment → Source to GitHub Actions, and configure the `github-pages` environment to allow `master`. These repository settings have not been inspected or changed while preparing this proposal. The workflow must report missing setup clearly; a local build is not proof of a hosted deployment.

After implementation is pushed and publication is authorized, verify the Actions deployment result and open the returned URL. Check the landing page, nested-page direct loading, navigation, search, mobile layout and assets under the project base path. Only then record the site as published. Do not publish runtime inventory, credentials, private kubeconfigs or captured homelab data as examples; use generic sample names and placeholders.

## Risks / Trade-offs

- Lost instructions or broken README anchors → Use the migration inventory, fix repository references and retain concise compatibility links for commonly referenced old headings where practical.
- Existing documentation may describe stale behavior → Verify commands, defaults and examples against current source/specifications instead of blindly copying them.
- Master documentation can lead the latest release → Label unreleased instructions and keep upgrade/release links visible; versioned manuals remain future work.
- A green build can still produce broken project-site links → Exercise generated pages and assets under `/goK8sAutoDocs/`, then verify the actual deployment URL.
- Pages may need repository/account setup → Verify eligibility and settings before the first publication and track hosted verification separately from implementation completion.

## Migration Plan

1. Inventory README sections and preserve unrelated work; verify configuration and setup commands.
2. Write the manual, move implementation material to the relevant OpenSpec records, simplify README and update CHANGELOG.
3. Add pinned MkDocs tooling, local validation, output exclusions and the docs workflow; check publication conditions and unchanged container behavior.
4. Preview the manual and run acceptance checks. After an authorized push, configure Pages as needed, publish and verify the returned URL before adding a live-site claim.
5. To roll back, revert the documentation/workflow change and rerun the last known-good docs revision through `master`; the application deployment remains unaffected.

## Open Questions

No product decision blocks implementation. Verify tool versions/action SHAs, repository Pages availability/settings, the actual public URL and the supported source quick-start path during implementation. These are checks, not claims of completed configuration.

## References

- [GitHub: custom Pages workflows](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages) — artifact deployment, job permissions and environment requirements.
- [MkDocs configuration](https://www.mkdocs.org/user-guide/configuration/) — docs directory, navigation, project-subpath URL and validation settings.
- [MkDocs deployment](https://www.mkdocs.org/user-guide/deploying-your-docs/) — generated static output; this design uses Actions artifacts rather than `gh-deploy`.
