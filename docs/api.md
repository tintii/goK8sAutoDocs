# API reference

The APIs serve shared snapshots. They do not query Kubernetes on reads. Inventory clients must accept additive fields/kinds and inspect warnings and success times. No built-in API authentication is supplied; protect access through your proxy.

## GET /api/v1/health
Returns `{"status":"running"}` while AutoDocs is serving requests. It does not
query Kubernetes or read home-page configuration. Responses are not cached.

## GET /api/v1/home
Returns the home page content from `HOMEPAGE_CONFIG`, or defaults when unset.
Responses are not cached. Invalid or unreadable configured files return HTTP 500.

## GET /api/v1/service-status
Returns cached workload availability (`ready`, `attention`, `inactive`) and workload rows from the same inventory snapshot. `checkedAt` is the oldest contributing successful workload collection time. Collection failures retain available prior data with warnings; no usable initial data returns HTTP 503. This API never initiates discovery.

## GET /api/v1/cluster-info
Returns sanitized registry-backed cluster identities/display names, with a compatible single-target display name. Credentials, kubeconfig paths and server authentication details are not returned.

## GET /api/v1/versions-dashboard
Returns Deployments, StatefulSets and DaemonSets across configured clusters/namespaces, including main/init containers and additive cluster/classification metadata. Successful empty inventories return `[]`; no usable initial snapshot returns HTTP 503. Failed collection retains previous component data with original timestamps and diagnostics. GETs never query Kubernetes and responses use no-store with `X-AutoDocs-Generation`.

## GET /api/v1/discovery-status
Returns no-store shared generation, server time, progress, next automatic/manual times, configured cluster identities, available classification values and per-component last attempt/last success/error/warnings. A never-successful component has no successful timestamp. This endpoint does not trigger collection.

## POST /api/v1/discovery-refresh
Requests one server-owned refresh, returning HTTP 202 with the shared state immediately. Optional JSON `{"generation": 1}` supplies the viewer's observed generation. Concurrent requests, obsolete generations and cooldown clicks do not queue extra scans. Browser requests must be same-origin. Closing the browser does not cancel the job.

## GET /api/v1/auto-docs-dashboard
Returns an array of Ingress, HTTPRoute and Service entries selected by `k8sautodocs.io/publish=true` labels. Entries include `name`, `namespace`, `kind`, `hostnames`, `labels`, `annotations`, `relatedResources`, `endpoints`, and `issues`. Ingress entries retain the existing `kind: "ingress"`; HTTPRoute entries use `kind: "HTTPRoute"`. Services add `kind: "Service"`, `serviceType`, optional raw `externalName`, and `dns: {targets: [...], targetSource?: "annotation"|"externalIPs"|"externalName"|"loadBalancer"}`. Clients must accept the additive source kind and metadata. Cluster-aware source/link identities are unchanged; equal hostnames never merge different sources.

The optional `view` parameter selects the resource set:

- Omitted, empty, or `view=published`: published resources, preserving the default API behavior.
- `view=discovered`: resources with no `k8sautodocs.io/publish` label key, using the same array schema and URL resolution. Services additionally require either supported ExternalDNS hostname annotation key, even if its value is empty or invalid.
- Unknown or repeated `view` parameters return HTTP 400.

For example, `/api/v1/auto-docs-dashboard?view=discovered` supplies the Discovered tab. `/api/v1/filtered-resources` remains published-only.

Each Kubernetes-derived `endpoints` item includes `url`, `host`, `scheme`, and `port`. HTTPRoute endpoints also identify their `gateway` (`name` and `namespace`) and `listener`. Empty collections are JSON arrays. Per-resource `issues` explain unresolved destinations or unknown/stale controller status.

Collection failures preserve available entries and are reported as a JSON array of strings in the `X-AutoDocs-Warnings` response header, which the dashboard displays. A missing HTTPRoute API is normal on an Ingress-only cluster.

## GET /api/v1/filtered-resources
Returns metadata for supported GVRs selected by the same publication label. HTTPRoutes are included when their API is available; partial collection warnings use `X-AutoDocs-Warnings`.

## Manual links (unreleased)

The published directory also includes `kind: "manual-link"`, `sourceType: "manual-link"` rows from marked ConfigMaps. Optional relevance and declaration provenance follow the [manual-link contract](manual-directory-links.md). These entries are excluded from Discovered, Versions, service status and filtered resources.

## Observe a snapshot and request refresh

```bash
curl -i http://localhost:8080/api/v1/versions-dashboard
curl -s http://localhost:8080/api/v1/discovery-status
curl -i -X POST http://localhost:8080/api/v1/discovery-refresh \
  -H 'Content-Type: application/json' -d '{"generation":1}'
```

The generation above is illustrative: use the generation you observed. Success returns 202 with shared state, including whether a run is already active. Invalid JSON/unknown fields return 400; cross-site browser refresh requests return 403; unsupported methods on data/status/refresh return 405. Data responses use `Cache-Control: no-store`; successful inventory responses expose `X-AutoDocs-Generation` and optional `X-AutoDocs-Warnings` (a JSON string array). Partial failures can return 200 with retained stale data. A successful empty inventory is `[]`, distinct from 503 before a usable snapshot exists. UI scope/filter choices are client-side; `view` is the directory API selector.
