# Configuration

## Environment variables

- `AUTODOCS_CONFIG=/etc/autodocs/discovery/autodocs.yaml` - Optional registry/classification/refresh/cache configuration, loaded at startup
- `LOG_LEVEL=info` - Minimum application log level: `debug`, `info`, `warn` or `error` (default `info`)
- `DEBUG=true` - Legacy debug fallback when `LOG_LEVEL` is unset or blank
- `HOMEPAGE_CONFIG=/etc/autodocs/home/home.yaml` - Optional home page content file, reread on each request

The server listens on port 8080.

## Home page content and status

The default page `/` opens the application directory. The optional overview at
`/overview-page` is linked from the sidebar footer and keeps configured content,
MOTD and favourites. `HOMEPAGE_CONFIG` and `/api/v1/home` retain
their behavior; configured home content appears on the overview. Start with
[config/home.example.yaml](https://github.com/tintii/goK8sAutoDocs/blob/master/config/home.example.yaml):

```yaml
title: Cluster overview
description: Application links and container versions.
environment: Homelab
motd: |
  Maintenance window: Saturday 10:00–11:00.
  Backups run as usual.
footer: AutoDocs · Homelab
```

All fields are optional; omitted fields use defaults. An empty `motd` hides the
message. The title must be non-empty. Content is plain text, including multiline
MOTD text; no HTML, JavaScript or template syntax is evaluated. Unknown fields, malformed YAML,
multiple documents and files over 64 KiB report a configuration error. Keep
secrets out of this file: its content is available to everyone who can view the UI.

For Kubernetes, put that YAML in a ConfigMap key named `home.yaml`, mount the
**whole ConfigMap directory** read-only at `/etc/autodocs/home`, and set
`HOMEPAGE_CONFIG=/etc/autodocs/home/home.yaml`. Do not use `subPath`, which
prevents projected file updates. No extra RBAC or Reloader is needed. The app
rereads the file per request; open pages refresh content and status every 60
seconds while visible or when returning to the tab. The shared Update now control refreshes inventory; home content keeps its own automatic reread. Allow
Kubernetes time to project an updated ConfigMap. With GitOps, edit the owning
Helm values so reconciliation preserves the content. The homelab wiring is in
the sibling [values/k8sAutoDocs.yaml](https://github.com/tintii/homelab-gitops/blob/master/values/k8sAutoDocs.yaml).

Without `HOMEPAGE_CONFIG`, built-in content works normally. A configured but
unreadable or invalid file shows a notice and retains the last loaded content
in an open page, or defaults on first load; status and navigation remain usable.
The home page readiness probe remains independent of configuration and cluster
status.

The small header indicator beside the theme toggle reports **Running** when
AutoDocs responds to `/api/v1/health`, or **Unavailable** when that request fails.
Hover or focus it for an explanation. It checks application availability without
querying Kubernetes or inferring cluster/workload health. Inventory collection
age, shared updates and per-cluster diagnostics live in the Directory and
Versions sidebar submenu.

**Favourites** shows your existing directory stars across Applications and
Discovered. A starred group includes its current members; a starred link keeps
only that source's exact URL. Personal groups and unresolved group members use
the same directory rules. Destinations always come from the live API, never
saved preference strings. Removed sources disappear, while their saved stars
remain available if they return. Preferences update the home page immediately
and across tabs, without fetching for local group changes. The directory APIs
are requested only when stars exist, then refreshed every 60 seconds while
visible or on returning to the tab. Collection failures retain available live
matches and display a brief notice. With no stars, the section points to the
directory. Browser storage restrictions have the same behavior as the directory.

The legacy ConfigMap `links` field is still accepted by the API, but is not
rendered as personal favourites. Remove it from new configurations.

## Restart and reload contract

| Setting | Takes effect |
| --- | --- |
| `AUTODOCS_CONFIG` registry, classification, refresh and cache settings | Controlled process restart |
| `LOG_LEVEL` / `DEBUG` | Process restart |
| Mounted target/Rancher credentials | Start of next refresh; directory mounts allow projected rotation |
| `HOMEPAGE_CONFIG` content | Each request; visible pages poll every 60 seconds |
| Personal groups, filters, theme, stars and table layout | Immediately in this browser; no collector restart |

For the complete registry/classification example, see [Connecting clusters](cluster-discovery.md). Refresh defaults are interval `5m`, cooldown `30s`, cluster timeout `30s`; cache persistence is disabled unless `cache.path` is set. Invalid registry structure or log levels prevent startup; a single unavailable target does not. See [Logging](troubleshooting.md#logging) for accepted levels and observable events.
