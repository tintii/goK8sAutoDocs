# AutoDocs user manual

AutoDocs is a read-only Kubernetes inventory and application directory. Find published applications, inspect discovered endpoints and compare container versions across clusters. AutoDocs displays declarations; it does not test destination health or reconcile DNS.

This manual follows `master`. Rancher discovery is available in **v0.6.0-RC1**; manual directory links are **unreleased**. Check the [release history](https://github.com/tintii/goK8sAutoDocs/blob/master/CHANGELOG.md) before using these features with an older image.

Start with [Getting started](getting-started.md) for your first local run.

| Task | Guide |
| --- | --- |
| Install from source, run a container or deploy to Kubernetes | [Installation](installation.md) |
| Search, filter, group and star applications; arrange Versions | [Using AutoDocs](using-autodocs.md) |
| Set home content, refresh intervals and log levels | [Configuration](configuration.md) |
| Connect direct clusters or discover them through Rancher | [Connecting clusters](cluster-discovery.md) |
| Publish Ingress, HTTPRoute or ExternalDNS Service declarations | [Publishing endpoints](publishing-endpoints.md) |
| Add a wiki, console or runbook to Applications | [Manual directory links](manual-directory-links.md) (unreleased) |
| Manage freshness, persistence, upgrades and rollback | [Operations](operations.md) |
| Diagnose startup, permissions or stale inventory | [Troubleshooting](troubleshooting.md) |
| Integrate with the read-only snapshot APIs | [API reference](api.md) |

The UI has no built-in authentication. Put access control at your ingress/proxy when needed. Filters and publication labels control presentation; they do not limit the collector's Kubernetes permissions.
