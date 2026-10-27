## Why

goK8sAutoDocs currently documents only annotated Ingresses and constructs HTTP links even when endpoints use HTTPS. Applications exposed through Kubernetes Gateway API HTTPRoutes should appear in the same directory, with Kubernetes holding publication metadata and networking resources supplying the destination.

## What Changes

- Discover Gateway API `gateway.networking.k8s.io/v1` HTTPRoutes alongside existing Ingresses.
- **BREAKING**: Select published resources using the exact label `k8sautodocs.io/publish: "true"`. Missing labels and all other values exclude the resource; the legacy publish annotation no longer enables publication.
- **BREAKING**: Remove `k8sautodocs.io/fqdn-override` entirely. Neither annotations nor labels override endpoint destinations.
- Derive concrete clickable URLs from Ingress hosts/TLS or HTTPRoute hostnames and their parent Gateway listeners, including non-default listener ports.
- Present both resource kinds in the published endpoint dashboard, opening links in a new tab with `target="_blank"` and explaining endpoints that cannot be resolved.
- Keep Kubernetes as the live data source, without a separate publication database or writes to Kubernetes.
- Isolate the Auto-Docs response cache from the filtered-resources cache so both API endpoints return their own payloads.

## Capabilities

### New Capabilities

- `published-endpoint-directory`: Label-based publication, Ingress and HTTPRoute discovery, derived URLs, and a shared clickable endpoint directory.

### Modified Capabilities

None. This repository has no existing capability specs.

## Impact

- Backend: `go/main.go` discovery, metadata filtering, URL resolution, related resources, and response caching.
- Frontend: `go/static/autodocs-dashboard.html` links, resource descriptions, counts, search, and unresolved endpoint display; any supporting landing-page wording.
- APIs: `/api/v1/auto-docs-dashboard` gains HTTPRoute entries and resolved URL data; `/api/v1/filtered-resources` uses the new publication label and includes HTTPRoutes when supported.
- Kubernetes access: read access to HTTPRoutes and their parent Gateways, including parents in other namespaces, in addition to existing resource permissions. Gateway API remains optional on Ingress-only clusters.
- Documentation: explain the label contract, removal of legacy annotations, required permissions, and GitOps-managed publication metadata in the README and its changelog.
- Existing users must move publication intent from annotations to labels. No cluster, Helm, deployment, commit, or push actions are part of this proposal.
