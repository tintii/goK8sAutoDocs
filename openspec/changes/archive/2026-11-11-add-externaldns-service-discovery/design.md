## Context

The active directory is served from `inventoryService` snapshots. `collectTarget` in `go/inventory.go` currently produces Ingress and HTTPRoute partitions; Services are fetched as related resources and independently collected as published metadata for filtered-resources. `go/autodocs.go` models directory entries and web endpoints. `go/static/directory-model.js` uses cluster/kind/namespace/name source identities and accepts only server-resolved HTTP(S) links. `go/static/inventory-ui.js` explicitly names the directory's source partitions when calculating freshness.

The root specs predate implemented, unarchived multicluster and UI changes. This change's modified requirements preserve their shared snapshots, classification, personal groups and search behavior. Archive earlier overlapping changes before this change so older deltas cannot overwrite these requirements. No application changes are included in this planning change.

## Goals / Non-Goals

**Goals:**

- Document Services with ExternalDNS hostnames, including ExternalName declarations for destinations outside Kubernetes.
- Retain exact opt-in publication and the separate Discovered view.
- Support DNS-only declarations and explicitly configured browser links without fabricating a protocol or target.
- Integrate with multicluster classification, stable preferences, independent stale-data retention and existing read-only RBAC.

**Non-Goals:**

- DNS reconciliation, reachability checks, DNS lookups, HTTP probing, traffic forwarding or assertions that a DNS provider has applied a declaration.
- General DNS inventory: DNSEndpoint CRDs, arbitrary record types, `internal-hostname`, FQDN templates, custom annotation prefixes and controller/provider-specific filtering are deferred.
- Reproducing NodePort/headless target discovery, EndpointSlice or Pod enumeration for standalone Service entries.
- Arbitrary URL overrides, URL paths, multiple web protocols/ports per Service, or changing Ingress/HTTPRoute URL resolution.
- Deployment or publishing as part of this change.

## Decisions

### 1. Services are independent directory sources

Use `kind: "Service"` consistently. A Service is eligible when its own publication label is exactly `"true"`, or when that label is absent and either supported ExternalDNS `hostname` annotation key is present. Key presence keeps malformed/empty declarations visible for diagnosis. A present non-true publication label excludes it. Target-only, group-only and plain unlabelled Services are not discovered.

All Service types follow the same selection rule. Published Services with missing hostnames remain visible and unresolved; `spec.externalName`, cluster-local names and target annotations never substitute for an advertised hostname. One Service produces one card, including all of its hostnames and links. An Ingress or HTTPRoute sharing a hostname remains a separate source; do not deduplicate across resource identities or clusters.

Alternative: include every Service or require both publication and ExternalDNS metadata. The first floods Discovered with internal infrastructure; the second loses useful discovery and hides errors in explicitly published declarations.

### 2. Parse two annotation prefixes without silently preferring one

Recognize the `hostname` and `target` suffixes under `external-dns.alpha.kubernetes.io/` and `external-dns.kubernetes.io/`. Resolve each logical field independently. Split commas, trim surrounding whitespace, remove one terminal DNS dot, lowercase DNS names, canonicalize IP targets, deduplicate and sort deterministically. Ignore empty list members; a wholly empty declared field is invalid.

Advertised names must be ASCII DNS names, including already encoded IDNA names, with valid DNS label/total lengths. A leading `*.` is retained for display but never linked. Reject schemes, ports, paths, userinfo, embedded whitespace, IP literals as advertised names, and malformed wildcard placement. Targets accept DNS names or IPv4/IPv6 literals, without wildcard, URL or port syntax. Original annotations remain available for inspection and must be rendered as text.

For a single prefix, retain valid tokens and diagnose invalid tokens. When both prefixes declare a field, accept it only if both lists are valid and their normalized sets match; otherwise report a conflict and leave that effective field empty. An invalid or conflicting explicit target must not fall back to another target source. Hostname conflict prevents links; target conflict does not prevent a link to an independently valid hostname with explicit web metadata. Equivalent duplicate values do not produce issues.

Alternative: always prefer the newer prefix. AutoDocs does not know which prefix the running ExternalDNS controller uses; reporting ambiguity is more honest and avoids a new controller-configuration dependency.

### 3. Separate advertised hosts, declared targets and browser URLs

Keep existing `hostnames` and `endpoints` response fields. Add Service-only `serviceType`, optional `externalName`, and `dns` metadata containing `targets` and an optional `targetSource`. Target sources are `annotation`, `externalIPs`, `externalName`, or `loadBalancer`. Values describe declarations, not verified DNS records or provider record types.

Determine targets in this bounded order:

1. If either target annotation key exists, use its effective values or diagnose invalidity/conflict without fallback.
2. For ExternalName and LoadBalancer Services, use non-empty `spec.externalIPs` when present.
3. For ExternalName, otherwise use `spec.externalName`.
4. For LoadBalancer, otherwise use non-empty IP/hostname values in `status.loadBalancer.ingress`.
5. For other types, leave targets empty when no explicit target annotation exists. Do not infer controller flags, enumerate Nodes/Pods or resolve DNS.

Expose the raw declared `externalName` separately when an annotation or externalIPs overrides it. Validate selected target values; retain safe valid values with issues for malformed values. Unknown targets do not invalidate an advertised hostname. Do not infer a provider's final A/AAAA/CNAME record or health from these fields.

Alternative: implement the full ExternalDNS Service source algorithm. That would introduce controller flags, endpoint watches and provider semantics unrelated to a directory's read-only purpose.

### 4. Explicit annotations opt into web links

Introduce Service annotations `k8sautodocs.io/scheme` and `k8sautodocs.io/port`. Trim values; scheme must then be exactly `http` or `https`. An omitted port defaults to 80 or 443; an explicit port must be decimal digits in 1..65535. Empty or invalid values produce an issue and no web links. A port without a scheme is incomplete and also produces an issue.

For each valid concrete advertised hostname, build an HTTP(S) URL using the explicit scheme and selected port; omit the standard port from URL text. One annotation pair applies to all advertised names. Neither Service ports, targetPort, port names nor appProtocol supplies a fallback. This works for an ExternalName Service without any ports. The target is informational and never replaces the advertised host.

If neither web annotation exists, a Service with a valid advertised name is a DNS-only entry. Absence of web intent is not a resolution failure. Render its hostname and any known target as copyable text, label its content as DNS information, and provide no Open action. Report actual declaration problems through `issues`; do not generate a missing-scheme warning for intentional DNS-only entries. Wildcard-only entries remain visible, with an explanation that no concrete browser destination exists.

Alternative: assume HTTPS or infer from port 443. DNS declarations can refer to databases and arbitrary TCP services, and a Service's port need not describe the external endpoint. Explicit annotations keep the existing no-invented-URL contract.

The proposed authoring contract can be reviewed with this illustrative declaration:

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
    k8sautodocs.io/scheme: https
spec:
  type: ExternalName
  externalName: monitoring.vendor.example
```

With compatible ExternalDNS configuration, that controller manages the DNS alias. AutoDocs proposes `https://monitoring.example.com` as the browser destination and displays `monitoring.vendor.example` as the declared target. Removing the scheme annotation yields a DNS-only entry. Removing the publication label moves either form into Discovered. Adding an ExternalDNS target annotation can describe an explicit IP or hostname destination without making that target the browser hostname. This example is documentation of the proposed interface, not a deployed manifest.

### 5. Collect once per refresh and retain failures independently

Add a `services` partition to `collectTarget`, target connection-failure handling, status output, directory projection and frontend freshness keys. List Services once for the standalone directory and filtered metadata within each target run, then use the same eligible rows for both. Reuse the list for related lookups where practical; existing source-specific backend behavior must not change.

Published Service metadata enters filtered-resources through the directory partition; skip its separate generic Services collection to avoid duplicate rows. Preserve the current whole-filtered-partition retention policy when any required source fails. A Service list error retains the previous Services partition and timestamps while successful Ingress/HTTPRoute partitions update. Successful empty Service results remove prior entries. Publication changes use the same retained partition for both directory views.

Keep the additive map-based snapshot schema compatible with existing version-1 snapshots. A missing Services partition means never collected, not successfully empty; preserve older partitions and expose Services as pending until a successful refresh. A newly failed Service collection must appear in status/warnings even when other data is usable. Restore saved Service rows and original timestamps after restart. Older readers ignore the additional partition; verify this assumption against the current reader in compatibility tests.

Alternative: discover Services in an HTTP handler or keep a second cache. That would bypass the current coordinator, duplicate Kubernetes reads and make refresh/failure semantics inconsistent.

### 6. Preserve the existing directory interaction model

Use the current cluster-aware source tuple and exact resolved URL link identity. Group labels come from the Service itself; classification uses existing configured resource/namespace/cluster precedence. Index advertised names, selected targets, externalName and Service type for search. Kind filters, grid/list layouts, personal groups and group favourites work for DNS-only sources; individual link favourites remain web-only. No new DNS-hostname favourite identity is introduced.

Render all metadata safely as text. Copy controls must be keyboard accessible, copy the exact displayed normalized value and report clipboard failure without pretending success. DNS-only content must not be represented as a broken web application. Keep current safe link attributes, scoped counters, view transitions, legacy overview favourites and endpoint health disclaimers.

## Risks / Trade-offs

- [Declarations differ from deployed DNS] → Describe DNS intent only; document ExternalDNS source/filter/provider prerequisites and avoid active-status language.
- [Services become newly visible] → Keep ordinary unannotated Services out of Discovered and honor explicit exclusion labels; document additive `Service` kind for API consumers.
- [A web link requires extra metadata] → Provide a minimal ExternalName example with only `scheme: https`; no ports block is needed for this directory use case.
- [Alias points to an HTTPS backend without suitable virtual-host/certificate configuration] → Document that DNS does not configure the destination application or TLS; AutoDocs does not rewrite hosts to targets.
- [Stale cache and filtered-resource duplication regressions] → Exercise partition retention, upgrade/restart and one-row-per-source cases with fixtures.
- [Old spec deltas overwrite newer behavior during archive] → Preserve current scenarios in this change and archive overlapping implemented changes first.

## Migration Plan

1. Implement and test parsing/models, then snapshot collection/projections, then DNS/web presentation. Keep all Ingress/HTTPRoute behavior covered by regression checks.
2. Update README/CHANGELOG with both prefixes, publication matrix, explicit web annotations, target limitations, DNS-only examples and read-only Service permissions. Documentation must distinguish proposed annotations from ExternalDNS annotations.
3. Validate an old cache without Services, a new cache with Services, failed refresh after restore and API payload compatibility using local fixtures. Run the repository's Go, race, browser and container smoke checks during implementation.
4. Review the resulting change before a separately requested release/deployment. No existing Kubernetes manifests need mutation for DNS-only discovery; optional web hints are opt-in.
5. Rollback uses the previous application version, which ignores the new partition and annotations. Existing source IDs and browser preferences remain intact; no DNS records are changed by either version.

## Open Questions

No blocking questions remain for the proposed initial scope. Scheme/port annotation names and DNS-only presentation are explicit proposed defaults that can be adjusted during review. Custom prefixes, internal-hostname support, per-host web settings and non-web protocol links are deferred extensions.
