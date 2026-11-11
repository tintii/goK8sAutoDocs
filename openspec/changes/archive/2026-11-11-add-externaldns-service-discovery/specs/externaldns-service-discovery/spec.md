## ADDED Requirements

### Requirement: Service source eligibility

The system SHALL support all core/v1 Service types as directory sources. A Service SHALL enter Applications only when its own `k8sautodocs.io/publish` label is exactly `"true"`. A Service with no publication label SHALL enter Discovered only when either supported ExternalDNS `hostname` annotation key is present, including an empty or malformed value. Any present non-true publication value SHALL exclude it from both views. Group-only, target-only and ordinary unlabelled Services without a hostname annotation SHALL NOT enter Discovered. Selection SHALL NOT require selectors, Pods, ports, ingress objects, Gateway objects or load-balancer addresses.

#### Scenario: ExternalName without workloads
- **WHEN** an ExternalName Service has a hostname annotation and `publish: "true"`, with no selector, Pods or ports
- **THEN** it appears once in Applications with kind `Service` and its source cluster, namespace and name

#### Scenario: Unpublished DNS declarations
- **WHEN** a Service has an ExternalDNS hostname annotation and no publication label
- **THEN** it appears in Discovered even if its hostname value is empty or invalid
- **AND** invalid declarations carry an explanatory issue

#### Scenario: Ordinary Services stay outside discovery
- **WHEN** Services have only a group label, only a target annotation, or no relevant metadata, and none has a publication label
- **THEN** none appears in either directory view

#### Scenario: Explicit publication without a hostname
- **WHEN** a Service has `publish: "true"` but no supported hostname annotation
- **THEN** it remains in Applications with a missing-hostname issue and no web link
- **AND** externalName, target annotations and cluster-local names do not become advertised hostnames

#### Scenario: Explicit exclusion wins
- **WHEN** a Service's publication label is `"false"`, `"True"`, an empty string or any other non-true value
- **THEN** it appears in neither directory view regardless of its DNS annotations

### Requirement: ExternalDNS annotation normalization and conflicts

The system SHALL interpret `hostname` and `target` under both `external-dns.alpha.kubernetes.io/` and `external-dns.kubernetes.io/`. It SHALL split comma-separated lists, trim whitespace, discard empty list members, remove one trailing DNS dot, lowercase DNS names, canonicalize IP targets, deduplicate and sort values. Advertised names SHALL be valid ASCII DNS names, including encoded IDNA names and optional leading `*.`, respecting DNS label and total lengths; IP literals, URL syntax, ports, paths, userinfo, embedded whitespace and other wildcard placements SHALL be rejected. Targets SHALL accept valid DNS names or IPv4/IPv6 literals without wildcard, URL or port syntax. Invalid members SHALL produce issues while valid members of a single-prefix list remain usable. A wholly empty declared field SHALL be invalid.

When both prefixes declare a logical field, the system SHALL accept it only when both lists are valid and normalize to the same set. Otherwise it SHALL report a conflict and leave that effective field empty. Hostname and target fields SHALL resolve independently. Original annotation values SHALL remain inspectable as text.

#### Scenario: Multiple equivalent hostnames
- **WHEN** a hostname annotation contains ` App.Example.com., app.example.com, second.example.com, `
- **THEN** one Service entry contains `app.example.com` and `second.example.com` once each in deterministic order

#### Scenario: Equivalent prefixes
- **WHEN** both prefixes declare the same valid hostname set in different order, case or trailing-dot spelling
- **THEN** the set is accepted without a conflict issue

#### Scenario: Conflicting hostname prefixes
- **WHEN** the two prefixes declare different hostname sets, or one is invalid
- **THEN** the entry remains visible with a conflict issue, no effective advertised names and no generated links
- **AND** neither prefix silently wins

#### Scenario: Mixed valid and unsafe values
- **WHEN** a single hostname annotation contains `good.example.com,https://bad.example.com/path,user@bad.example.com`
- **THEN** only `good.example.com` is accepted as a hostname and the invalid members are diagnosed

#### Scenario: Wildcard declaration
- **WHEN** the only advertised name is `*.example.com`
- **THEN** it remains visible as DNS information but never becomes a clickable URL
- **AND** the entry explains that no concrete browser destination is available

### Requirement: Bounded declared target interpretation

The system SHALL expose Service type, the declared externalName when present, and normalized declared targets with their source. If either supported target annotation key is present, its effective values SHALL take precedence; an invalid or conflicting explicit target SHALL NOT fall back to Service fields. Otherwise ExternalName and LoadBalancer Services SHALL use non-empty `spec.externalIPs` first. Without externalIPs, ExternalName SHALL use `spec.externalName` and LoadBalancer SHALL use its status ingress IP/hostname values. Other Service types without explicit target annotations SHALL have no inferred targets. Invalid selected values SHALL produce issues while valid selected values remain usable. Targets SHALL NOT replace advertised names or be described as verified records, record types, propagation or endpoint health. Target resolution SHALL NOT perform DNS queries, provider calls or standalone Pod/Node enumeration.

#### Scenario: External destination
- **WHEN** an ExternalName Service advertises `monitoring.example.com` and declares externalName `monitoring.vendor.example`
- **THEN** its advertised hostname and declared target are separate fields with target source `externalName`

#### Scenario: Explicit target override
- **WHEN** the same Service declares target annotation `192.0.2.10`
- **THEN** the selected target is `192.0.2.10` with source `annotation`
- **AND** the declared externalName remains inspectable but is not substituted for the override

#### Scenario: Conflicting targets do not change valid hostnames
- **WHEN** target annotations conflict but the advertised hostname and explicit web hints are valid
- **THEN** targets remain unresolved with an issue and no fallback to externalName
- **AND** the independently derived browser link remains available

#### Scenario: ExternalIPs precede type-derived targets
- **WHEN** an ExternalName or LoadBalancer Service has externalIPs and no target annotation
- **THEN** externalIPs supply the declared targets with source `externalIPs`

#### Scenario: LoadBalancer and unsupported target inference
- **WHEN** a LoadBalancer Service has status ingress addresses but no explicit target or externalIPs
- **THEN** those addresses supply targets with source `loadBalancer`
- **WHEN** a ClusterIP, headless or NodePort Service has an advertised name but no explicit target annotation
- **THEN** it remains a valid DNS declaration with no inferred target and no Node or Pod enumeration

### Requirement: Explicit web intent and DNS-only entries

Service annotations `k8sautodocs.io/scheme` and `k8sautodocs.io/port` SHALL control optional web links. After trimming, scheme SHALL be exactly `http` or `https`; an omitted port SHALL default to 80 or 443 respectively, and an explicit port SHALL contain decimal digits representing 1..65535. Invalid or empty explicit values and a port without a scheme SHALL produce an issue and no web links. With valid web hints, the system SHALL generate one deduplicated URL per concrete advertised hostname, omit default ports, and include non-default ports. Service ports, targetPort, port names, appProtocol, targets and legacy fqdn-override metadata SHALL NOT infer or override these URLs. The annotation pair SHALL apply to every advertised hostname of the Service.

When neither web annotation is present, an entry with valid DNS declarations SHALL remain DNS-only with no links and no missing-scheme error. No host-target mismatch, missing target, or absence of Pods SHALL alone invalidate an explicitly configured hostname-based link. Ingress and HTTPRoute resolution SHALL remain unchanged.

#### Scenario: HTTPS ExternalName without ports
- **WHEN** a Service advertises `monitoring.example.com` and has scheme annotation `https` without any port annotation or Service ports
- **THEN** its URL is `https://monitoring.example.com` regardless of the target's spelling

#### Scenario: Custom port
- **WHEN** the scheme is `http` and the port annotation is `8080`
- **THEN** every concrete advertised hostname receives an HTTP URL with port 8080

#### Scenario: Invalid web metadata
- **WHEN** a Service declares scheme `javascript`, an empty explicit scheme, a nonnumeric or out-of-range port, or a port without a scheme
- **THEN** it retains DNS information with an explanatory issue and no browser links

#### Scenario: DNS-only database entry
- **WHEN** a published Service advertises `db.example.com` with no web annotations, even if Service ports include 443 or a port named https
- **THEN** it is shown as a DNS-only entry and no HTTP/HTTPS scheme is inferred
- **AND** the absence of web intent is not reported as an error

### Requirement: Service inventory isolation and compatibility

The system SHALL collect standalone Service entries in a dedicated per-cluster `services` partition during the existing coordinator run. Directory GET requests SHALL only project snapshots. Service collection success, failure and freshness SHALL appear in discovery status and directory freshness calculations. Failed lists or target connections SHALL retain previous successful Service rows and timestamps with visible diagnostics; successful empty lists SHALL remove previous rows. Restoring a compatible old snapshot without Services SHALL preserve other partitions and represent Services as never collected until a successful refresh. Restored Service data SHALL retain original success timestamps. Both directory views SHALL classify the same Service partition. Filtered-resources SHALL project published Service metadata from the collected Service rows once and SHALL retain its current whole-partition failure policy when a required source fails.

#### Scenario: Service collection fails while other sources succeed
- **WHEN** a Service list is forbidden after a successful collection while Ingress discovery succeeds
- **THEN** prior Service entries and their old success timestamp remain visible with a Service collection warning
- **AND** newly collected Ingress entries are published normally

#### Scenario: Successful deletion
- **WHEN** the next successful Service list contains no eligible Services
- **THEN** the Services partition becomes successfully empty and previous Service entries disappear

#### Scenario: Upgrade from a cache without Services
- **WHEN** a compatible persisted snapshot contains Ingress/HTTPRoute data but no Services partition
- **THEN** those sources remain available and Services have no fabricated successful timestamp
- **AND** a pending or failed initial Service collection is visible in status

#### Scenario: Restart with Services followed by connection failure
- **WHEN** a snapshot containing Services is restored and its cluster cannot be reached
- **THEN** saved Service rows remain available with their original success timestamp and current failure diagnostics

#### Scenario: One filtered Service row
- **WHEN** a published Service is collected for the directory and filtered-resources
- **THEN** filtered-resources includes exactly one metadata row for that source identity without another generic Service list

### Requirement: Service DNS presentation and identity

The directory SHALL present Service type, advertised hostnames and available declared targets distinctly. DNS-only entries SHALL have keyboard-accessible copy controls for their displayed normalized names/targets, no Open action and no generic broken-web-endpoint presentation solely for lacking web intent. Clipboard failures SHALL be reported without claiming success. Actual issues SHALL remain visible. Metadata SHALL render as text, never executable markup. Search SHALL include Service type, advertised names, selected targets and declared externalName. Existing kind/scope filters, personal groups, group favourites and grid/list layouts SHALL support Service sources. Source identity SHALL remain cluster ID, kind, namespace and name; link identity SHALL additionally include the exact resolved URL. Individual link favourites SHALL remain web-only. Distinct sources sharing a DNS name SHALL remain distinct.

#### Scenario: Copy DNS-only content
- **WHEN** a user activates a copy control on a DNS-only Service by keyboard
- **THEN** the displayed normalized value is copied with success or failure feedback and no browser navigation

#### Scenario: Search target and filter kind
- **WHEN** a user searches an externalName or selected target while filtering kind to Service
- **THEN** matching Service entries in the active view and scope remain visible

#### Scenario: Shared names preserve source ownership
- **WHEN** two clusters or an Ingress and a Service advertise the same hostname
- **THEN** each source retains its own identity, grouping and preference assignments

#### Scenario: Service content is untrusted text
- **WHEN** annotations or diagnostics contain markup-like strings
- **THEN** they render as text and do not create executable content or unsafe links
