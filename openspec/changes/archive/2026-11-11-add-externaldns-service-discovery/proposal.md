## Why

AutoDocs currently creates directory entries from Ingresses and HTTPRoutes, so Services carrying ExternalDNS declarations cannot appear as standalone entries. Supporting these Services, including `ExternalName`, lets one Kubernetes declaration drive ExternalDNS-managed DNS and document an endpoint hosted inside or outside the cluster.

## What Changes

- Add Services as directory sources with their own cluster, namespace, name, kind, publication and grouping metadata. Include ordinary Service types as well as selectorless `ExternalName` Services; discovery must not require Pods, an Ingress, a Gateway or a load-balancer address.
- Discover advertised hostnames from ExternalDNS annotations, supporting both `external-dns.alpha.kubernetes.io/` and `external-dns.kubernetes.io/`. Handle comma-separated hostnames and retain one directory entry per Service with distinct destinations.
- Preserve publication semantics: an exact `k8sautodocs.io/publish: "true"` label places the Service in Applications; an absent publication label permits Discovered only when an ExternalDNS hostname declaration exists; any present non-true value excludes it from both views. A group label alone never publishes a resource.
- Retain explicitly published Services without a usable hostname as unresolved entries with an explanation. Do not enumerate all ordinary unlabelled Services in Discovered.
- Distinguish advertised names from DNS targets. For `ExternalName`, expose the declared external destination and account for ExternalDNS target overrides. Any browser link uses the advertised hostname rather than silently substituting its target.
- Support copyable DNS-only entries without inventing web URLs. Opt into browser links with the Service annotations `k8sautodocs.io/scheme: http|https` and optional `k8sautodocs.io/port`; use the scheme's default port when omitted. Service ports, port names and DNS targets do not implicitly establish web protocols.
- Accept equivalent values across the two ExternalDNS prefixes, but report conflicting values and leave the affected hostname or target field unresolved. Do not silently choose a prefix or merge conflicting declarations.
- Integrate Services into the shared inventory refresh, independent stale-data retention, discovery status, classification, search and kind filters. Preserve source-based identities and published-only filtered resources without duplicating Service metadata rows.

## Capabilities

### New Capabilities

- `externaldns-service-discovery`: Service eligibility, supported ExternalDNS annotations, advertised hostname and target interpretation, ExternalName support, and diagnostics for missing or conflicting declarations.

### Modified Capabilities

- `published-endpoint-directory`: Extend directory sources to Services, retaining explicit publication, read-only discovery, source identity and honest unresolved destinations.
- `endpoint-directory-organization`: Extend Applications/Discovered eligibility, grouping, search and presentation to Service entries and their DNS context.

## Impact

- Go endpoint models/resolution, inventory collection/status/projection, filtered-resource integration, directory presentation, browser fixtures and documentation.
- Follow the implemented `add-multicluster-discovery-snapshots` design for refresh and persistence. Base specifications still describe older per-view caches; subsequent delta specs must preserve current shared-snapshot behavior.
- Existing deployment RBAC already permits Service reads. No DNS-provider credentials, Kubernetes write permissions or new controller are required.
- API consumers must tolerate an additional source kind and additive DNS metadata. Existing Ingress/HTTPRoute destination rules and browser preference identities remain intact.

## Non-goals

AutoDocs will not create DNS records, proxy traffic, verify DNS propagation or claim endpoint health. ExternalDNS retains DNS reconciliation ownership. This change does not add DNSEndpoint CRD discovery, a general DNS-management interface, arbitrary record types, or a replacement for ExternalDNS controller filtering/provider behavior.

## Proposed Defaults

The design resolves the earlier open questions by supporting DNS-only entries alongside explicitly configured web links, and by diagnosing conflicting annotation prefixes. These are proposed product choices for review, not claims about existing behavior. No separate DNS controller configuration or network probing is introduced.

## References

- [ExternalDNS ExternalName tutorial](https://kubernetes-sigs.github.io/external-dns/latest/docs/tutorials/externalname/)
- [ExternalDNS Service source semantics](https://kubernetes-sigs.github.io/external-dns/latest/docs/sources/service/)
