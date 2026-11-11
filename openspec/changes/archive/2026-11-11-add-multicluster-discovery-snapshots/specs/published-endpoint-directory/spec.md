## MODIFIED Requirements

### Requirement: Combined endpoint discovery

The Auto-Docs API SHALL return Ingresses and Gateway API v1 HTTPRoutes across namespaces and all configured eligible clusters in the requested published or discovered view. An omitted view SHALL select published resources. Each entry SHALL identify its stable cluster ID, cluster display context, source name, namespace, kind and available environment/datacentre classification and preserve available related-resource context. Source identity SHALL distinguish equal names in different clusters. All dependency lookups SHALL use the source's own target clients. Gateway parents SHALL be readable dependencies without requiring publication labels. Existing endpoint resolution and collection diagnostics SHALL apply to both views.

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

#### Scenario: Same source identity in different clusters
- **WHEN** two configured targets contain a source with the same kind, namespace and name
- **THEN** both entries remain distinct using cluster ID and each resolves Gateways/Services only within its own target

### Requirement: Kubernetes-backed read-only data and isolated caches

The system SHALL read publication metadata and endpoint declarations from Kubernetes without maintaining a separate publication database or writing to Kubernetes. Auto-Docs and filtered-resources SHALL expose separate response projections over a shared server-refreshed inventory snapshot and SHALL never exchange response formats, view payloads or unrelated warnings. Periodic/manual coordinator runs SHALL reflect successfully collected metadata changes; inventory GETs SHALL NOT trigger collection or expiry-based request blocking. A derived persistent snapshot SHALL be permitted for startup recovery, with preserved source success timestamps and explicit stale/failure diagnostics. Published/discovered classification SHALL use the same source partition within an application generation without claiming a transactional Kubernetes snapshot.

#### Scenario: Publication label removed
- **WHEN** a publication label is removed from a previously included resource and its source partition refreshes successfully
- **THEN** the next published snapshot excludes that resource from the published view
- **AND** the discovered projection includes it according to the existing absent-key rule

#### Scenario: API requests do not exchange cached payloads
- **WHEN** Auto-Docs and filtered-resources are requested in either order within a snapshot generation
- **THEN** each endpoint returns only its own response format and data without querying Kubernetes

#### Scenario: Refresh fails after restart
- **WHEN** a compatible disk snapshot is restored and a target's initial collection fails
- **THEN** its previous derived endpoints remain available with their original success times and visible stale diagnostics
