## MODIFIED Requirements

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
