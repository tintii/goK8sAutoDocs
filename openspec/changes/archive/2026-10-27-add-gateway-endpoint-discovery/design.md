## Context

The Auto-Docs collector in `go/main.go` lists Ingresses, includes resources based on annotation presence, and returns hostnames and related resources. The browser builds every link with `http://`. The dynamic Kubernetes client already exists, but the Auto-Docs collector does not use it. The filtered-resources and Auto-Docs handlers also share cache variables despite dedicated Auto-Docs variables being declared.

The agreed product is an application endpoint directory: document declared endpoints and provide links that open a new tab. Publication is selected through labels. Networking resources supply destinations; no FQDN override is needed.

## Goals / Non-Goals

**Goals:**

- Document explicitly published Ingresses and Gateway API v1 HTTPRoutes together.
- Derive URLs with the appropriate scheme, hostname, and port.
- Keep publication state in Kubernetes metadata and keep the application read-only.
- Preserve useful resource identity and related-resource information.
- Explain unresolved endpoints without disrupting other entries.

**Non-Goals:**

- Individual API operation documentation, OpenAPI generation, or exhaustive rule/filter visualization.
- GRPCRoute, TCPRoute, TLSRoute, UDPRoute, mesh Service parents, or ListenerSet support in this change.
- A metadata editing UI, separate database, URL overrides, DNS management, endpoint probing, or deployment changes.
- A general rewrite of workload discovery or the versions dashboard.

## Decisions

### 1. Use one exact publication label

Both published-resource API paths select resources with `k8sautodocs.io/publish=true`. Only the string `"true"` opts in; missing, `"false"`, empty, and other values do not. Apply selectors in Kubernetes list requests where possible. Labels on an Ingress or HTTPRoute control that resource alone: a Gateway label does not implicitly publish its children.

Remove the legacy publication annotation and all handling of `k8sautodocs.io/fqdn-override`. No fallback is retained, and no replacement override label is introduced. This is a deliberate breaking metadata change that keeps selection unambiguous. Annotations with these names can remain on Kubernetes resources but have no effect on the application.

### 2. Add a Gateway API collector using the existing dynamic client

List published HTTPRoutes across namespaces. Resolve Gateway parent references, including defaulted namespace/group/kind and explicit `sectionName` or `port` restrictions. Read referenced Gateways as dependencies regardless of their publication labels; cache parent lookups within a collection pass. Use relevant HTTP/HTTPS listeners and intersect their hostname constraints with route hostnames.

Gateway API is optional. A missing HTTPRoute API leaves Ingress discovery working. A forbidden list request or inaccessible parent is a diagnostic, not evidence that no routes exist; retain available entries and report the incomplete collection. Do not introduce access to Secret contents for URL derivation.

The alternative of adding HTTPRoutes only to a GVR list would expose metadata without resolving URLs or feeding the actual Auto-Docs collector. A new typed Gateway API dependency is unnecessary for the initial v1 resource subset.

### 3. Resolve destinations on the server

Produce one directory entry per source resource with stable identity, `hostnames`, labels, annotations, related resources, and an `endpoints` array. Each endpoint contains the resolved `url`, `host`, `scheme`, and `port`; Gateway-derived endpoints also identify the parent Gateway and listener. Use a per-resource `issues` array for resolution diagnostics. Retain the existing array response shape and existing Ingress-specific fields where applicable. The browser consumes `endpoints[].url` rather than constructing URLs itself.

- Ingress: use concrete rule hosts; HTTPS applies to a host covered by the Ingress TLS declaration, otherwise HTTP. Use standard ports 443/80; controller-specific external port overrides are outside scope.
- HTTPRoute: use concrete hostnames from the route/listener intersection, HTTP or HTTPS from the listener protocol, and the listener's external port. Never infer external scheme or port from a backend Service.
- Deduplicate destinations within each resource. Keep separate resources separate even when they share a URL.
- A wildcard-only or unconstrained hostname cannot become a browser destination. A concrete listener hostname can resolve a route whose hostname is omitted or wildcarded when they intersect.
- Missing parents, no compatible listeners, incompatible hostnames, and explicit rejected attachments produce issues rather than invented URLs. Missing/stale controller status is reported as unknown; it is not a live connectivity check. When status is used, match the relevant parent and current resource generation.

The directory documents configured destinations. It does not guarantee DNS, certificate, or application health. Path/header/method matches can be added as documentation later; this change provides application origin links.

### 4. Keep resource context and presentation consistent

Retain Ingress related-resource behavior. For HTTPRoutes, include referenced Gateways and Service backends as related resources and follow supported Service selectors to Pods using the existing relationship approach. Respect backend namespaces; unresolved or unsupported backend references must not prevent documenting an otherwise resolvable origin. This is descriptive association, not reimplementation of reference authorization.

Use endpoint/resource terminology in dashboard counts, search, loading messages, and cards. Show kind and Gateway/listener context where applicable, render unresolved entries even with no links, and tolerate empty arrays. Open resolved links with `target="_blank"` and `rel="noopener noreferrer"`.

### 5. Isolate response caching

Give Auto-Docs its own cache state so requests to filtered-resources cannot populate its payload or vice versa. Preserve the existing refresh interval. Cache access must be safe for concurrent requests. No persistent storage or Kubernetes watches are required for this change.

## Risks / Trade-offs

- Existing annotation-only entries disappear → Document the breaking change and move publication intent to labels before upgrading.
- Gateway API attachment and hostname rules have edge cases → Exercise parent defaults, listener restrictions, wildcard intersections, and multiple listeners in focused fixture tests.
- Permissions or missing parents yield partial data → Preserve available results and expose collection/resource diagnostics; do not silently equate access failures with an empty cluster.
- Declared links can still be unreachable → Describe status as configuration/controller observations, without claiming end-to-end health.
- Argo CD may reconcile manual metadata edits → Document managing publication labels through the source Helm/Git configuration.

## Migration Plan

1. Set `metadata.labels.k8sautodocs.io/publish: "true"` on intended Ingresses and HTTPRoutes in their owning configuration. Remove legacy Auto-Docs annotations when convenient.
2. Grant the application read access to HTTPRoutes and parent Gateways where Gateway API is used, retaining existing related-resource permissions.
3. Upgrade the application after implementation and validation; verify existing Ingress URLs and representative HTTPS HTTPRoutes.
4. Rollback requires restoring the previous application version and publication annotations for Ingresses. The previous version cannot document HTTPRoutes.

This proposal performs none of these deployment steps.

## Open Questions

None blocking the agreed scope. Broader route types and individual path documentation remain future discussions.

## References

- Gateway API HTTPRoute: https://gateway-api.sigs.k8s.io/reference/api-types/httproute/
- Gateway API hostname intersection: https://gateway-api.sigs.k8s.io/docs/concepts/hostnames/
- Kubernetes labels: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
