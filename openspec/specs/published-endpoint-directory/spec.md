# published-endpoint-directory Specification

## Purpose

Define read-only discovery of published and discovered Kubernetes Ingress and
HTTPRoute endpoints, including publication labels, networking-derived URLs,
related resources, diagnostics and dashboard links.

## Requirements
### Requirement: Explicit publication through labels

The system SHALL include supported source resources in the default or explicitly published Auto-Docs view and in filtered-resources only when their own `k8sautodocs.io/publish` label equals the string `"true"`. The explicitly requested discovered Auto-Docs view SHALL include only supported Ingresses and HTTPRoutes for which this label key is absent. A present label with any other value SHALL exclude the source resource from both Auto-Docs views and filtered-resources. The system SHALL ignore the legacy publication annotation and SHALL NOT use `k8sautodocs.io/fqdn-override` in either labels or annotations to select resources or change destinations.

#### Scenario: Label opts a resource in
- **WHEN** an Ingress or HTTPRoute has `k8sautodocs.io/publish: "true"` under `metadata.labels`
- **THEN** it is selected once for the default/published Auto-Docs and corresponding filtered-resource results
- **AND** it is excluded from the discovered Auto-Docs view

#### Scenario: Missing or non-true label
- **WHEN** the publication label is missing, empty, `"false"`, or any value other than `"true"`
- **THEN** the resource is excluded from the default/published Auto-Docs view and filtered-resources even if a legacy Auto-Docs annotation exists
- **AND** only a missing publication-label key permits inclusion in the discovered Auto-Docs view

#### Scenario: Legacy metadata does not override a destination
- **WHEN** a published or discovered resource has `k8sautodocs.io/fqdn-override` in labels or annotations
- **THEN** its destinations are derived exclusively from networking resource fields

#### Scenario: Parent publication does not cascade
- **WHEN** a Gateway is labelled for publication but an attached HTTPRoute has no publication label
- **THEN** the HTTPRoute is excluded from the default/published Auto-Docs view and filtered-resources
- **AND** it qualifies for the discovered Auto-Docs view based on its own label absence

### Requirement: Combined endpoint discovery

The Auto-Docs API SHALL return Ingresses and Gateway API v1 HTTPRoutes across namespaces in the requested published or discovered view. An omitted view SHALL select published resources. Each entry SHALL identify its source name, namespace, and kind and preserve available related-resource context. Gateway parents SHALL be readable dependencies without requiring publication labels. Existing endpoint resolution and collection diagnostics SHALL apply to both views.

#### Scenario: Mixed resource directory
- **WHEN** the cluster contains a published Ingress and a published HTTPRoute
- **THEN** both appear once with distinct source identities in the default/published Auto-Docs view

#### Scenario: Cross-namespace Gateway
- **WHEN** a published HTTPRoute references a Gateway in another namespace
- **THEN** the system resolves the named Gateway in that namespace and includes its identity as related context

#### Scenario: Backend association
- **WHEN** a published HTTPRoute references a supported Service backend
- **THEN** related-resource discovery uses the backend's effective namespace and name
- **AND** a missing backend does not discard the source entry

#### Scenario: Discovered resource resolution
- **WHEN** an Ingress or HTTPRoute without a publication label is requested through the discovered view
- **THEN** it receives the same endpoint URL resolution, related-resource handling, and issues as a published resource
- **AND** unresolved entries remain visible without invented URLs

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

The system SHALL retain published source entries when it cannot derive a concrete destination, SHALL supply an explanatory issue, and SHALL NOT invent a URL. Explicit current controller rejection SHALL be reported against its relevant parent without treating unrelated or stale status as authoritative. Derived URLs SHALL NOT be described as proof of live endpoint health.

#### Scenario: Wildcard-only destination
- **WHEN** the effective hostname remains `*.example.com` without a concrete intersecting hostname
- **THEN** the entry explains that no concrete hostname is available and supplies no wildcard link

#### Scenario: Parent unavailable or incompatible
- **WHEN** a referenced Gateway is missing, inaccessible, has no matching listener, or has incompatible hostname constraints
- **THEN** the entry remains visible with an issue and no URL from that unresolved attachment

#### Scenario: Attachment rejected
- **WHEN** current status explicitly rejects the relevant parent attachment
- **THEN** the system reports the rejection and does not present that attachment as an accepted endpoint

### Requirement: Optional Gateway API and visible collection failures

The system SHALL continue documenting Ingresses when the HTTPRoute API is not installed. Other collection failures SHALL preserve available results and SHALL be exposed as diagnostics rather than silently presented as an empty successful discovery.

#### Scenario: Ingress-only cluster
- **WHEN** Kubernetes reports the HTTPRoute resource API is not installed
- **THEN** the Auto-Docs endpoint returns the available published Ingress entries successfully

#### Scenario: HTTPRoute access forbidden
- **WHEN** listing HTTPRoutes is forbidden but Ingress discovery succeeds
- **THEN** available Ingress entries remain accessible and the API/dashboard communicates incomplete Gateway API discovery

### Requirement: Clickable directory presentation

The dashboard SHALL present both source kinds using endpoint terminology, search their identities and destinations, and use server-resolved URLs for links with `target="_blank"` and `rel="noopener noreferrer"`. It SHALL display unresolved entries and handle an empty directory without a rendering error.

#### Scenario: Open a resolved endpoint
- **WHEN** a user selects a resolved HTTPRoute or Ingress link
- **THEN** the browser opens the exact server-resolved URL in a new tab

#### Scenario: No link available
- **WHEN** an entry has no resolved endpoints
- **THEN** the dashboard still displays its identity and resolution issues

### Requirement: Kubernetes-backed read-only data and isolated caches

The system SHALL read publication metadata and endpoint declarations from Kubernetes without maintaining a separate publication database or writing to Kubernetes. Auto-Docs and filtered-resources SHALL use independent response caches and reflect changed metadata after cache expiry.

#### Scenario: Publication label removed
- **WHEN** a publication label is removed from a previously included resource and the directory cache expires
- **THEN** the next successful refresh excludes that resource

#### Scenario: API requests do not exchange cached payloads
- **WHEN** Auto-Docs and filtered-resources are requested in either order within the cache interval
- **THEN** each endpoint returns only its own response format and data
