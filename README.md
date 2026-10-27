# goK8sAutoDocs

!THIS PROJECT IS UNFINISHED AND EXPERIMENTAL! I work on it when time permits :)

A Kubernetes resource monitoring and documentation tool that provides consolidated access to cluster resources, versions dashboard, and auto-documentation capabilities.

## Features

- **Versions Dashboard**: Monitor deployment, statefulset, and daemonset versions across namespaces
- **Auto-Docs Dashboard**: Organise published Ingress and HTTPRoute endpoints into application groups, and find undeclared endpoints in Discovered
- **Resource Filtering**: Dynamic filtering of Kubernetes resources by GVR (Group, Version, Resource)
- **Cluster Information**: Display current cluster context and connection details
- **Caching**: Built-in caching for improved performance
- **Flexible Authentication**: Supports both in-cluster and out-of-cluster Kubernetes connections

## Prerequisites

- Go 1.24.0 or higher (module toolchain: Go 1.24.5)
- Kubernetes cluster access (via kubeconfig or service account)
- Access to the following Kubernetes resources:
  - Deployments, StatefulSets, DaemonSets
  - Services, ConfigMaps
  - Ingresses
  - Pods
  - Gateway API v1 HTTPRoutes and parent Gateways when installed
  - Namespaces when Gateway listeners select allowed namespaces by label

## Installation

### From Source

```bash
git clone https://github.com/tintii/goK8sAutoDocs.git
cd goK8sAutoDocs/go
go mod tidy
go build -o goK8sAutoDocs .
```

### Container image

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
[GitOps runbook](../homelab-gitops/docs/k8sautodocs.md).

## Configuration

### Environment Variables

- `DEBUG=true` - Enable debug logging

The server listens on port 8080.

### Kubernetes Connection

The application automatically detects the connection method:

- **In-Cluster**: When running inside a Kubernetes pod, uses service account authentication
- **Out-of-Cluster**: When running locally, uses your current kubeconfig context

## Usage

### Running the Application

Before running, if using your local k8s context make sure you set it up `kubectl config use-context <context-name>`

```bash
# Run locally
./goK8sAutoDocs

# Run with debug mode
DEBUG=true ./goK8sAutoDocs

# For Kubernetes, use the container image and a read-only ServiceAccount.
# The sibling homelab-gitops repo supplies the Helm values/ApplicationSet.
```

### Accessing the Dashboards

- **Main Dashboard**: http://localhost:8080/
- **Versions Dashboard**: http://localhost:8080/versions-dashboard-page
- **Auto-Docs Dashboard**: http://localhost:8080/autodocs-dashboard-page

The Auto-Docs dashboard opens on **Applications**, with one section per group and an **Ungrouped** section for published resources without a group. Groups can span namespaces; each card retains its namespace and kind. Named groups are sorted by exact label value, followed by Ungrouped. Cards are sorted by namespace, kind, and name.

Select **Discovered** to see a flat list of Ingresses and HTTPRoutes that have no publication label. The selected tab is included in the URL, so direct links, reloads, and browser back/forward navigation restore it. Search includes group names and applies within the active tab; resource/namespace/URL counters describe that tab's complete loaded results, with the matching subset shown separately. Both tabs preserve unresolved entries and display collection warnings.

## API Endpoints

### GET /api/v1/cluster-info
Returns current cluster information including context and server details.

### GET /api/v1/versions-dashboard
Returns version information for deployments, statefulsets, and daemonsets.

### GET /api/v1/auto-docs-dashboard
Returns an array of Ingress and HTTPRoute entries selected by `k8sautodocs.io/publish=true` labels. Entries include `name`, `namespace`, `kind`, `hostnames`, `labels`, `annotations`, `relatedResources`, `endpoints`, and `issues`. Ingress entries retain the existing `kind: "ingress"`; HTTPRoute entries use `kind: "HTTPRoute"`.

The optional `view` parameter selects the resource set:

- Omitted, empty, or `view=published`: published resources, preserving the default API behavior.
- `view=discovered`: resources with no `k8sautodocs.io/publish` label key, using the same array schema and URL resolution.
- Unknown or repeated `view` parameters return HTTP 400.

For example, `/api/v1/auto-docs-dashboard?view=discovered` supplies the Discovered tab. `/api/v1/filtered-resources` remains published-only.

Each `endpoints` item includes `url`, `host`, `scheme`, and `port`. HTTPRoute endpoints also identify their `gateway` (`name` and `namespace`) and `listener`. Empty collections are JSON arrays. Per-resource `issues` explain unresolved destinations or unknown/stale controller status.

Collection failures preserve available entries and are reported as a JSON array of strings in the `X-AutoDocs-Warnings` response header, which the dashboard displays. A missing HTTPRoute API is normal on an Ingress-only cluster.

### GET /api/v1/filtered-resources
Returns metadata for supported GVRs selected by the same publication label. HTTPRoutes are included when their API is available; partial collection warnings use `X-AutoDocs-Warnings`.

## Resource Types Supported

### Versions Dashboard

- Deployments
- StatefulSets
- DaemonSets

### Auto-Docs Dashboard

- Ingresses (`networking.k8s.io/v1`)
- HTTPRoutes (`gateway.networking.k8s.io/v1`)

Services and Pods appear as related resources. HTTPRoutes also show parent Gateways; existing Ingress relationship discovery includes Deployment and ConfigMap context. Gateways are dependencies rather than separately published directory entries.

### Filtered Resources

- Deployments
- StatefulSets
- DaemonSets
- ConfigMaps
- Services
- Ingresses
- HTTPRoutes

## Publication and grouping labels

Add this metadata to each Ingress or HTTPRoute you want to document:

```yaml
metadata:
  labels:
    k8sautodocs.io/publish: "true"
    k8sautodocs.io/group: media
```

Only the exact string `"true"` publishes a resource in Applications. The optional group label assigns one group to the whole source resource, including all its resolved URLs. Missing or empty group values use Ungrouped. Group names preserve spelling and case; `Media` and `media` are separate groups.

| Publication label | Group label | Dashboard location |
|---|---|---|
| `"true"` | Non-empty | Applications → named group |
| `"true"` | Missing or empty | Applications → Ungrouped |
| Absent key | Any | Discovered |
| `"false"` | Any | Hidden from both tabs |
| Present with any other value, including empty or `"True"` | Any | Excluded from both tabs |

The absence of the publication key is significant: an empty value does not qualify for Discovered. A group label alone never publishes a resource. Labels are read from the Ingress or HTTPRoute itself; they are not inherited from Gateways, namespaces, workloads, or Argo Applications. A literal group named `Ungrouped` remains distinct from the fallback section, whose description identifies applications without a group.

Kubernetes holds the live publication state. With GitOps, declare labels in the resource's owning Helm/Git configuration so reconciliation preserves them. The application only reads Kubernetes; it has no publication database or metadata editing interface.

### URL derivation

- Ingress URLs use concrete rule hosts and matching TLS host declarations, with standard HTTP/HTTPS ports.
- HTTPRoute URLs use the concrete intersection of route and listener hostnames, the compatible Gateway listener protocol, and its external port. Non-default ports are included. Backend Service ports do not determine browser URLs.
- Gateway parent namespaces, listener names/ports, allowed route kinds, and allowed namespaces are respected. Namespace selectors require read access to namespace labels.
- Wildcard-only or unresolved endpoints remain visible with an explanation and no invented link. Unknown/stale attachment status is reported; current rejected attachments do not supply links.
- Links open the application origin in a new tab with `target="_blank"`. Individual path, header, and method matching is outside the directory's scope; links describe configuration and do not guarantee endpoint reachability.

### Migration from annotations

Move publication intent from the old `k8sautodocs.io/publish` annotation into `metadata.labels` before upgrading. Annotation-only resources appear in Discovered, and are excluded from Applications and filtered-resources. `k8sautodocs.io/fqdn-override` has been removed; neither annotations nor labels override destinations. Existing legacy annotations can be deleted, and retaining them has no effect.

### Kubernetes permissions

Use read-only RBAC. Existing dashboards require `list` on Deployments, StatefulSets, DaemonSets, ConfigMaps, Services, Ingresses, and Pods, plus `get` on referenced Services. Gateway API adds these ClusterRole rules when enabled:

```yaml
- apiGroups: [gateway.networking.k8s.io]
  resources: [httproutes]
  verbs: [list]
- apiGroups: [gateway.networking.k8s.io]
  resources: [gateways]
  verbs: [get]
- apiGroups: [""]
  resources: [namespaces]
  verbs: [get]
```

Namespace reads are needed only for listener namespace selectors. Discovery spans namespaces, including cross-namespace Gateway and backend references. No Secret contents or Kubernetes write permissions are needed for endpoint discovery. Backend references describe associations; Auto-Docs does not validate ReferenceGrants itself.

## Caching

Auto-Docs has separate, concurrency-safe 60-second in-memory caches for published and discovered responses. Filtered-resources has its own independent cache. Publication, group, and networking changes appear on the next request after the relevant cache expires; reload or revisit a tab to request fresh data. Collection warnings are cached with their matching view. Views refresh independently, so label edits do not produce an atomic transition across both caches.

## Development

Current behavior is specified in the [endpoint directory](openspec/specs/published-endpoint-directory/spec.md)
and [directory organization](openspec/specs/endpoint-directory-organization/spec.md)
specifications. Completed change proposals are kept in the
[OpenSpec archive](openspec/changes/archive/).

### Project Structure

```
goK8sAutoDocs/
├── go/
│   ├── main.go                # Startup, versions, and existing resource relationships
│   ├── autodocs.go            # Endpoint discovery and URL resolution
│   ├── resource_handlers.go   # Publication filtering and isolated response caches
│   ├── autodocs_test.go       # Kubernetes fixture and cache tests
│   ├── directory_views_test.go # Publication-state, view, and cache tests
│   ├── go.mod                # Go module file
│   └── static/               # Dashboard HTML, JavaScript, and browser fixture tests
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
node --test static/autodocs-dashboard.test.cjs
```

The browser checks use headless Chrome/Chromium and fixture responses, without contacting a cluster. Set `CHROME_BIN` to its executable if it is not found in a standard location. Browser checks are skipped when no executable is available.

## Security Considerations

- **CORS**: Currently allows all origins (`*`) - consider restricting in production
- **Authentication**: No built-in authentication - implement if exposing publicly
- **Namespace Filtering**: Some endpoints use hardcoded namespaces - make configurable for production

## Troubleshooting

### Common Issues

1. **Connection Errors**: Verify kubeconfig or service account permissions
2. **Resource Access**: Ensure RBAC permissions for target resources
3. **Port Conflicts**: Check if port 8080 is available

### Debug Mode

Enable debug logging to see detailed information:

```bash
DEBUG=true ./goK8sAutoDocs
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

**GNU GPL v3.0**

The GNU General Public License is a free, copyleft license for software and other kinds of works. It ensures that the software remains free and open source, requiring that any derivative works also be licensed under the GPL.

---

## Changelog

### [Unreleased]
#### Added

- Canonical OpenSpec specifications for endpoint discovery and directory
  organization, with both completed changes archived under `2026-10-27`.

- Digest-pinned multi-stage container build with a non-root, read-only runtime,
  packaged dashboard assets and an isolated fixture-based image smoke check.
- GitHub Actions validation and trusted GHCR publication of the exact tested
  image, with full-commit/release tags, OCI metadata and a reported digest.

- Gateway API v1 HTTPRoute discovery alongside Ingresses, with listener-derived HTTP/HTTPS URLs and related Gateways, Services, and Pods.
- Resolution issues, partial collection warnings, and browser/backend fixture tests.
- Applications groups through `k8sautodocs.io/group`, an Ungrouped fallback section, and a separate Discovered tab for sources without a publication label.
- Explicit API view selection, bookmarked tab navigation, group search, and independent published/discovered caches.

#### Changed

- **Breaking:** publication now requires `metadata.labels.k8sautodocs.io/publish: "true"`; legacy publish annotations no longer opt resources in.
- Published endpoint cards use server-resolved URLs, show both source kinds, and open links in a new tab.
- Application sections use source group labels across namespaces; cards retain namespace context. Explicit false and other present non-true publication values stay hidden from both views.

#### Removed

- `k8sautodocs.io/fqdn-override`; destinations come from networking resources.

#### Fixed

- Shared Auto-Docs/filtered-resources response caching and concurrent cache access.
- In-cluster Kubernetes client initialization and related-resource handling of pending Pods and empty Service selectors.

### [v0.1.0] - 2024-01-XX
#### Added

- Initial release of goK8sAutoDocs
- Versions dashboard for monitoring deployment versions
- Auto-docs dashboard for resource filtering
- Kubernetes resource discovery and filtering
- Caching system for improved performance
- Support for in-cluster and out-of-cluster connections
- RESTful API endpoints for all dashboards
- Static HTML dashboard pages
- Resource relationship mapping (pods, services, ingresses)
- Image parsing and SHA extraction
- Configurable resource filtering by GVR

#### Technical Features

- Go-based Kubernetes client implementation
- Dynamic client support for custom resources
- Structured logging with debug mode
- CORS support for web dashboard access
- Graceful error handling and fallbacks
- Modular handler architecture
- Configurable cache durations

#### Supported Resources

- Deployments, StatefulSets, DaemonSets
- Services, ConfigMaps, Ingresses
- Pods and container information
- Image metadata extraction
