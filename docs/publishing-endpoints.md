# Publishing endpoints

Label the owning resource in your Helm/Git configuration, wait for a successful refresh, then check Applications. This publishes directory metadata; it does not deploy a route, create DNS records or test endpoint reachability.

## Publish and group a source

Add this metadata to each Ingress, HTTPRoute or Service you want to document:

```yaml
metadata:
  labels:
    k8sautodocs.io/publish: "true"
    k8sautodocs.io/group: media
```

Only the exact string `"true"` publishes a resource in Applications. The optional group label assigns one group to the whole source resource, including all its resolved URLs. Missing or empty group values use Ungrouped. Group names preserve spelling and case; `Media` and `media` are separate groups.

| Publication label | Group label | Ingress / HTTPRoute | Service |
|---|---|---|---|
| `"true"` | Non-empty | Applications → named group | Applications → named group, even without a hostname |
| `"true"` | Missing or empty | Applications → Ungrouped | Applications → Ungrouped, even without a hostname |
| Absent key | Any | Discovered | Discovered only with an ExternalDNS hostname annotation key |
| `"false"` | Any | Hidden | Hidden |
| Present with any other value, including empty or `"True"` | Any | Hidden | Hidden |

The absence of the publication key is significant: an empty value does not qualify for Discovered. A group label alone never publishes a resource. Target-only, group-only and ordinary unlabelled Services stay outside Discovered. Labels are read from the source itself; they are not inherited from Gateways, namespaces, workloads, or Argo Applications. A literal group named `Ungrouped` remains distinct from the fallback section, whose description identifies applications without a group.

Kubernetes holds the live publication state. With GitOps, declare labels in the resource's owning Helm/Git configuration so reconciliation preserves them. The application only reads Kubernetes; its personal-group editor changes browser presentation only, with no publication database or Kubernetes metadata editing.

## Ingress and HTTPRoute examples

For an existing Ingress controller, backend Service `example-web:80` and TLS
Secret `example-web-tls` in namespace `web`, publish this Ingress through its
owning chart. Replace the class, names and hostname with your configuration:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example-web
  namespace: web
  labels:
    k8sautodocs.io/publish: "true"
    k8sautodocs.io/group: applications
spec:
  ingressClassName: nginx
  tls:
    - hosts: [app.example.com]
      secretName: example-web-tls
  rules:
    - host: app.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: example-web
                port:
                  number: 80
```

For Gateway API, use an existing `shared-gateway` in namespace `web` with an
HTTPS listener named `https` that permits HTTPRoutes and intersects
`app.example.com`. The listener owns TLS and the external port:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: example-web
  namespace: web
  labels:
    k8sautodocs.io/publish: "true"
    k8sautodocs.io/group: applications
spec:
  parentRefs:
    - name: shared-gateway
      sectionName: https
  hostnames: [app.example.com]
  rules:
    - backendRefs:
        - name: example-web
          port: 80
```

After a successful collection, inspect the card's issues and attachment state.
Both examples can yield `https://app.example.com` when their declarations and
controller state permit it. For a listener on 8443, the HTTPRoute URL includes
`:8443`. AutoDocs does not create the prerequisite controller, Gateway, Service,
TLS Secret or DNS record. Grant the [collector permissions](installation.md#kubernetes-permissions)
to read the sources and referenced Gateway.

### URL derivation

- Ingress URLs use concrete rule hosts and matching TLS host declarations, with standard HTTP/HTTPS ports.
- HTTPRoute URLs use the concrete intersection of route and listener hostnames, the compatible Gateway listener protocol, and its external port. Non-default ports are included. Backend Service ports do not determine browser URLs.
- Service URLs use advertised ExternalDNS hostnames and explicit `k8sautodocs.io/scheme` / optional `k8sautodocs.io/port` annotations, as described below. DNS-only declarations do not imply HTTP or HTTPS.
- Gateway parent namespaces, listener names/ports, allowed route kinds, and allowed namespaces are respected. Namespace selectors require read access to namespace labels.
- Wildcard-only or unresolved endpoints remain visible with an explanation and no invented link. Unknown/stale attachment status is reported; current rejected attachments do not supply links.
- Links open the application origin in a new tab with `target="_blank"`. Individual path, header, and method matching is outside the directory's scope; links describe configuration and do not guarantee endpoint reachability.

### ExternalDNS Service declarations

AutoDocs reads `hostname` and `target` annotations under both
`external-dns.alpha.kubernetes.io/` and `external-dns.kubernetes.io/`.
Comma-separated values are trimmed, DNS names lowercased with one final dot
removed, IP targets canonicalized, and results deduplicated and sorted. Names
must be ASCII DNS names (encoded IDNA is supported); wildcards beginning `*.`
are visible but never linked. URL syntax, userinfo, ports, paths, malformed
names and IP literals as advertised names produce issues. Targets accept DNS
names or IPv4/IPv6 literals without wildcard or URL syntax. Valid members of
a single-prefix list remain usable when other members are invalid.

When both prefixes specify a field, both complete lists must be valid and
normalize to the same set. Otherwise that field is unresolved with a conflict
issue. Hostname and target conflicts are independent: a conflicting target
does not remove a valid hostname-based web link. An invalid/conflicting
explicit target never falls back to another target source. Raw annotations
remain inspectable as text in Service details.

An ExternalName declaration can describe an external DNS-only destination
without Pods, selectors or Service ports:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-monitoring
  namespace: monitoring
  labels:
    k8sautodocs.io/publish: "true"
    k8sautodocs.io/group: observability
  annotations:
    external-dns.alpha.kubernetes.io/hostname: monitoring.example.com
spec:
  type: ExternalName
  externalName: monitoring.vendor.example
```

Add these annotations to that same Service to opt into an HTTPS link without
adding a ports block:

```yaml
metadata:
  annotations:
    external-dns.alpha.kubernetes.io/hostname: monitoring.example.com
    k8sautodocs.io/scheme: https
```

AutoDocs links to `https://monitoring.example.com`; the target remains
`monitoring.vendor.example`. For a non-default web port and explicit IP target
override, use this annotation set instead:

```yaml
metadata:
  annotations:
    external-dns.alpha.kubernetes.io/hostname: monitoring.example.com, status.example.com
    external-dns.alpha.kubernetes.io/target: 192.0.2.10
    k8sautodocs.io/scheme: https
    k8sautodocs.io/port: "8443"
```

The links use both advertised names with port 8443. The declared externalName
remains visible alongside the selected IP target. These YAML blocks illustrate
authoring metadata; they are not deployed manifests.

`scheme` and `port` are **AutoDocs annotations**, independent of ExternalDNS.
After trimming, scheme must be exactly `http` or `https`. Omitted ports default
to 80/443 and are omitted from URL text; an explicit port must be decimal digits
in 1..65535. Empty or invalid hints, or a port without a scheme, produce issues
and no web links. Service ports, names, appProtocol, targetPort and DNS targets
never infer a protocol or replace an advertised hostname. Neither hint means
intentional DNS-only content with no missing-scheme error. Published Services
without advertised names remain visible with an issue. Legacy fqdn-override
metadata never supplies a destination.

Target interpretation is deliberately bounded: explicit target annotations
win; otherwise ExternalName and LoadBalancer use non-empty `spec.externalIPs`;
then ExternalName uses `spec.externalName`, and LoadBalancer uses status ingress
IP/hostname values. ClusterIP, headless and NodePort have no inferred targets
without explicit target annotations. Invalid selected values carry issues;
valid values remain available. AutoDocs does not query DNS, providers, Pods or
Nodes to resolve standalone Service entries, or claim a provider record type.

ExternalDNS owns DNS reconciliation. Deploy/configure it separately with the
Service source and compatible type, namespace, label and domain filters,
provider access, and annotation-prefix support for the declarations you use.
AutoDocs can display declarations even when ExternalDNS is absent or ignores
them; it does not verify propagation or health. See the upstream
[ExternalName tutorial](https://kubernetes-sigs.github.io/external-dns/latest/docs/tutorials/externalname/)
and [Service source semantics](https://kubernetes-sigs.github.io/external-dns/latest/docs/sources/service/).
A DNS alias does not configure the destination's virtual host or TLS certificate:
the destination must serve the advertised hostname, and AutoDocs does not
substitute the target to bypass that requirement. DNSEndpoint, internal-hostname,
FQDN templates, arbitrary record types and full controller/provider emulation
are outside this directory's scope.

### Migration from annotations

Move publication intent from the old `k8sautodocs.io/publish` annotation into `metadata.labels` before upgrading. Annotation-only Ingresses/HTTPRoutes appear in Discovered; Services additionally need an ExternalDNS hostname key. All are excluded from Applications and filtered-resources without the exact-true label. `k8sautodocs.io/fqdn-override` has been removed; neither annotations nor labels override destinations. Existing legacy annotations can be deleted, and retaining them has no effect.

## Supported sources

### Workload inventory

- Deployments
- StatefulSets
- DaemonSets

### Endpoint directory

- Ingresses (`networking.k8s.io/v1`)
- HTTPRoutes (`gateway.networking.k8s.io/v1`)
- Services (`v1`), including selectorless ExternalName, ClusterIP/headless, NodePort and LoadBalancer declarations

Services and Pods also appear as related resources. A Service that is both a backend and a directory source has one standalone entry and remains related context on its referring source. HTTPRoutes also show parent Gateways; existing Ingress relationship discovery includes Deployment and ConfigMap context. Gateways are dependencies rather than separately published directory entries.

### Filtered Resources

- Deployments
- StatefulSets
- DaemonSets
- ConfigMaps
- Services
- Ingresses
- HTTPRoutes

For arbitrary HTTP(S) destinations, use [Manual directory links](manual-directory-links.md) (unreleased). For collector access, see [Kubernetes permissions](installation.md#kubernetes-permissions).
