## Why

The AutoDocs README is over 1,000 lines and mixes getting started, everyday use, configuration reference and implementation details. Users need a short route to a working application and a searchable user manual, while development decisions belong in OpenSpec changes.

## What Changes

- Reduce `README.md` to a brief introduction, core features, prerequisites, one working quick start, first-use guidance and links to the manual, releases and contribution information.
- Create a task-based user manual in `docs/` covering installation, configuration, dashboards, endpoint publication, cluster discovery, operation, troubleshooting and API reference.
- Preserve useful user-facing detail during the move. Keep implementation rationale, source structure, build/pipeline internals and validation history in the relevant OpenSpec changes and archive; keep release history in `CHANGELOG.md`.
- Build the Markdown manual with MkDocs and publish it as a GitHub Pages project site with navigation, search and working repository-subpath links.
- Add GitHub Actions documentation validation for pull requests and trusted publication from `master`, plus manual publication from `master`. Record the one-time Pages settings and verify the first hosted deployment.

## Capabilities

### New Capabilities

- `user-documentation`: Concise README and a complete, task-oriented AutoDocs user manual, with an explicit boundary between user guidance and OpenSpec implementation records.
- `documentation-pages-pipeline`: Reproducible documentation builds, validation and trusted GitHub Pages publication.

### Modified Capabilities

None. Application behavior and existing capability contracts remain unchanged.

## Impact

Scoped entirely to `goK8sAutoDocs`: README, `docs/`, MkDocs configuration, pinned documentation dependencies, a documentation workflow/check script, generated-output ignore rules, CHANGELOG and relevant OpenSpec records. The existing container workflow retains its tests and publication behavior. Documentation tooling is build-time only and is excluded from the runtime container build context.

This proposal does not implement the documentation migration, enable Pages, publish a site, commit or push. It requires no changes to homelab-gitops, Helm charts, application APIs or the live cluster.
