# published-endpoint-directory Specification

## Purpose

Define read-only discovery of published and discovered Kubernetes Ingress, HTTPRoute and Service declarations, including publication labels, honest resolved URLs or DNS-only content, related resources and snapshot diagnostics.

## Requirements

### Requirement: Explicit publication through labels

The system SHALL include supported source resources in the default or explicitly published Auto-Docs view and in filtered-resources only when their own `k8sautodocs.io/publish` label equals the string `"true"`. The explicitly requested discovered Auto-Docs view SHALL include supported Ingresses and HTTPRoutes for which this label key is absent, and Services with an absent publication label and a supported ExternalDNS hostname annotation key. A present label with any other value SHALL exclude the source resource from both Auto-Docs views and filtered-resources. The system SHALL ignore the legacy publication annotation and SHALL NOT use `k8sautodocs.io/fqdn-override` in either labels or annotations to select resources or change destinations.

#### Scenario: Label opts a resource in
- **WHEN** an Ingress, HTTPRoute or Service has `k8sautodocs.io/publish: "true"` under `metadata.labels`
- **THEN** it is selected once for the default/published Auto-Docs and corresponding filtered-resource results
- **AND** it is excluded from the discovered Auto-Docs view

#### Scenario: Missing or non-true label
- **WHEN** the publication label is missing, empty, `"false"`, or any value other than `"true"`
- **THEN** the resource is excluded from the default/published Auto-Docs view and filtered-resources even if a legacy Auto-Docs annotation exists
- **AND** only a missing publication-label key permits inclusion in the discovered Auto-Docs view, with Services additionally requiring an ExternalDNS hostname annotation key

#### Scenario: Legacy metadata does not override a destination
- **WHEN** a published or discovered resource has `k8sautodocs.io/fqdn-override` in labels or annotations
- **THEN** the legacy override is ignored; Ingress/HTTPRoute destinations retain their networking-field rules, and Service destinations use supported ExternalDNS hostname declarations plus explicit scheme/port annotations

#### Scenario: Parent publication does not cascade
- **WHEN** a Gateway is labelled for publication but an attached HTTPRoute has no publication label
- **THEN** the HTTPRoute is excluded from the default/published Auto-Docs view and filtered-resources
- **AND** it qualifies for the discovered Auto-Docs view based on its own label absence

#### Scenario: Published Service missing its advertised name
- **WHEN** a Service has the exact-true publication label but no supported ExternalDNS hostname annotation
- **THEN** it remains in Applications with an issue and no invented web URL
- **AND** filtered-resources includes its published metadata once

### Requirement: Combined endpoint discovery

The Auto-Docs API SHALL return Ingresses, Gateway API v1 HTTPRoutes and eligible core/v1 Services across namespaces and all configured eligible clusters in the requested published or discovered view. An omitted view SHALL select published resources. Each entry SHALL identify its stable cluster ID, cluster display context, source name, namespace, kind and available environment/datacentre classification and preserve available related-resource context. Source identity SHALL distinguish equal names in different clusters. All dependency lookups SHALL use the source's own target clients. Gateway parents SHALL be readable dependencies without requiring publication labels. Existing endpoint resolution and collection diagnostics SHALL apply to both views.

#### Scenario: Mixed resource directory
- **WHEN** the cluster contains a published Ingress, a published HTTPRoute and a published Service
- **THEN** all three appear once with distinct source identities in the default/published Auto-Docs view

#### Scenario: Cross-namespace Gateway
- **WHEN** a published HTTPRoute references a Gateway in another namespace
- **THEN** the system resolves the named Gateway in that namespace and includes its identity as related context

#### Scenario: Backend association
- **WHEN** a published HTTPRoute references a supported Service backend
- **THEN** related-resource discovery uses the backend's effective namespace and name
- **AND** a missing backend does not discard the source entry

#### Scenario: Discovered resource resolution
- **WHEN** an Ingress or HTTPRoute without a publication label, or a Service with a hostname annotation and no publication label, is requested through the discovered view
- **THEN** it receives the same endpoint URL resolution, related-resource handling, and issues as a published resource
- **AND** unresolved entries remain visible without invented URLs

#### Scenario: Same source identity in different clusters
- **WHEN** two configured targets contain a source with the same kind, namespace and name
- **THEN** both entries remain distinct using cluster ID and each resolves Gateways/Services only within its own target

#### Scenario: Service is both a backend and a source
- **WHEN** an independently published Service is also referenced by an Ingress or HTTPRoute
- **THEN** the Service has one standalone entry while remaining related context on its referring source
- **AND** sharing a hostname does not merge their source identities

### Requirement: Derive Ingress destinations

The system SHALL derive Ingress URLs from concrete rule hostnames and TLS host coverage, using HTTPS port 443 for TLS-covered hosts and HTTP port 80 otherwise. It SHALL NOT construct clickable destinations from wildcard-only hosts.

#### Scenario: TLS-covered Ingress
- **WHEN** a published Ingress declares a concrete rule host covered by its TLS hosts
- **THEN** its endpoint URL uses `https://` and that host

#### Scenario: Plain HTTP Ingress
- **WHEN** a published Ingress declares a concrete rule host without matching TLS coverage
- **THEN** its endpoint URL uses `http://` and that host

### Requirement: Derive HTTPRoute destinations from Gateway listeners

The system SHALL resolve Gateway parents with API defaults and listener restrictions, intersect route and listener hostnames, and derive URLs from the compatible listener's HTTP/HTTPS protocol and port. It SHALL omit default ports from URL text and include non-default ports. It SHALL NOT use backend Service ports as external endpoint ports.

#### Scenario: Homelab HTTPS route
- **WHEN** a route for `sonarr.k8s.sgw00.attards.top` references listener `https` on Gateway `envoy-gateway-system/homelab`, whose protocol is HTTPS, port is 443, and hostname is `*.k8s.sgw00.attards.top`
- **THEN** the endpoint URL is `https://sonarr.k8s.sgw00.attards.top`
- **AND** backend Service port 8989 does not appear in that URL

#### Scenario: Non-default port
- **WHEN** a compatible HTTP listener uses port 8080 for concrete hostname `example.com`
- **THEN** the endpoint URL is `http://example.com:8080`

#### Scenario: Default parent fields
- **WHEN** a Gateway parent reference omits group, kind, and namespace
- **THEN** it resolves to a Gateway in the HTTPRoute's namespace

#### Scenario: Listener restrictions
- **WHEN** a parent reference specifies a section name, a port, or both
- **THEN** only listeners matching all specified restrictions contribute destinations

#### Scenario: Concrete listener resolves an unspecified route hostname
- **WHEN** a route has no hostname and attaches to a compatible listener for `example.com`
- **THEN** `example.com` is used as the concrete destination host

#### Scenario: Multiple listeners
- **WHEN** multiple relevant listeners yield distinct HTTP and HTTPS URLs
- **THEN** each distinct URL appears once within the source entry

### Requirement: Explain unresolved endpoints

The system SHALL retain published source entries when it cannot derive a concrete destination, SHALL supply an explanatory issue, and SHALL NOT invent a URL. Explicit current controller rejection SHALL be reported against its relevant parent without treating unrelated or stale status as authoritative. Derived URLs SHALL NOT be described as proof of live endpoint health or DNS propagation. A Service with a valid advertised DNS name but no web intent SHALL be retained as a DNS-only entry without an error solely for lacking a browser URL. Missing or invalid advertised names and explicit invalid web hints SHALL carry issues.

#### Scenario: Wildcard-only destination
- **WHEN** the effective hostname remains `*.example.com` without a concrete intersecting hostname
- **THEN** the entry explains that no concrete hostname is available and supplies no wildcard link

#### Scenario: Parent unavailable or incompatible
- **WHEN** a referenced Gateway is missing, inaccessible, has no matching listener, or has incompatible hostname constraints
- **THEN** the entry remains visible with an issue and no URL from that unresolved attachment

#### Scenario: Attachment rejected
- **WHEN** current status explicitly rejects the relevant parent attachment
- **THEN** the system reports the rejection and does not present that attachment as an accepted endpoint

#### Scenario: DNS-only is intentional
- **WHEN** a Service has a valid advertised hostname and neither scheme nor port annotations
- **THEN** it remains visible as DNS information with no web link and no missing-scheme error

### Requirement: Optional Gateway API and visible collection failures

The system SHALL continue documenting Ingresses and eligible Services when the HTTPRoute API is not installed. Other collection failures SHALL preserve available results and SHALL be exposed as diagnostics rather than silently presented as an empty successful discovery.

#### Scenario: Ingress-only cluster
- **WHEN** Kubernetes reports the HTTPRoute resource API is not installed
- **THEN** the Auto-Docs endpoint returns the available published Ingress and Service entries successfully

#### Scenario: HTTPRoute access forbidden
- **WHEN** listing HTTPRoutes is forbidden but Ingress discovery succeeds
- **THEN** available Ingress entries remain accessible and the API/dashboard communicates incomplete Gateway API discovery

#### Scenario: Service access forbidden
- **WHEN** Service listing is forbidden but other directory sources can be collected
- **THEN** the directory preserves available data and exposes Service-specific incomplete or stale discovery
- **AND** missing Service permission is not treated as optional successful absence

### Requirement: Clickable directory presentation

The dashboard SHALL present Ingress, HTTPRoute and Service sources using endpoint terminology, search their identities and destinations, and use server-resolved URLs for links with `target="_blank"` and `rel="noopener noreferrer"`. It SHALL display unresolved entries and handle an empty directory without a rendering error.

#### Scenario: Open a resolved endpoint
- **WHEN** a user selects a resolved HTTPRoute, Ingress or Service link
- **THEN** the browser opens the exact server-resolved URL in a new tab

#### Scenario: No link available
- **WHEN** an entry has no resolved endpoints
- **THEN** the dashboard still displays its identity, available DNS metadata and any resolution issues; valid DNS-only Service entries have no Open action

### Requirement: Kubernetes-backed read-only data and isolated caches

The system SHALL read publication metadata and endpoint declarations from Kubernetes without maintaining a separate publication database or writing to Kubernetes. Auto-Docs and filtered-resources SHALL expose separate response projections over a shared server-refreshed inventory snapshot and SHALL never exchange response formats, view payloads or unrelated warnings. Periodic/manual coordinator runs SHALL reflect successfully collected metadata changes; inventory GETs SHALL NOT trigger collection or expiry-based request blocking. A derived persistent snapshot SHALL be permitted for startup recovery, with preserved source success timestamps and explicit stale/failure diagnostics. Published/discovered classification SHALL use the same source partition within an application generation without claiming a transactional Kubernetes snapshot.

#### Scenario: Publication label removed
- **WHEN** a publication label is removed from a previously included resource and its source partition refreshes successfully
- **THEN** the next published snapshot excludes that resource from the published view
- **AND** the discovered projection includes it according to the absent-key rule, with Services additionally requiring a supported hostname annotation

#### Scenario: API requests do not exchange cached payloads
- **WHEN** Auto-Docs and filtered-resources are requested in either order within a snapshot generation
- **THEN** each endpoint returns only its own response format and data without querying Kubernetes

#### Scenario: Refresh fails after restart
- **WHEN** a compatible disk snapshot is restored and a target's initial collection fails
- **THEN** its previous derived endpoints remain available with their original success times and visible stale diagnostics

#### Scenario: Removing publication from a label-only Service
- **WHEN** a Service without an ExternalDNS hostname annotation loses its publication label and its source partition refreshes successfully
- **THEN** it disappears from both directory views and filtered-resources
