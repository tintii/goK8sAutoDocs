# goK8sAutoDocs

Experimental, read-only Kubernetes inventory and application directory.

- Find published Ingress, HTTPRoute and ExternalDNS Service declarations.
- Search, scope, group and star applications across clusters.
- Compare Deployment, StatefulSet and DaemonSet container versions.
- Collect shared inventory in the background, with per-component freshness and optional persistent snapshots.
- Connect explicit targets or discover clusters through Rancher.

The [published user manual](https://tintii.github.io/goK8sAutoDocs/) follows `master`.
[Source Markdown](docs/index.md) is also available. Rancher discovery is available
in **v0.6.0-RC1**; manual directory links are **unreleased**. See
[CHANGELOG.md](CHANGELOG.md) for releases and upgrade notes.

## Quick start

You need Git, Go **1.24.0+** (module toolchain **1.24.5**, downloaded by Go when
needed), a free port **8080** and read-only Kubernetes credentials in
**`~/.kube/config`** with a current context. AutoDocs does not select credentials
through `KUBECONFIG`. The context must use verified HTTPS and an embedded bearer
token or embedded client certificate/key, with embedded CA data when needed.
Exec helpers, external credential files and insecure TLS are unsupported; see
[credential setup](docs/cluster-discovery.md#remote-cluster-secret-contract).

```bash
git clone https://github.com/tintii/goK8sAutoDocs.git
cd goK8sAutoDocs/go
go mod download
go build -o goK8sAutoDocs .
./goK8sAutoDocs
```

Run from `go/` so the server can find `static/`. Open **http://localhost:8080/**.
Startup collection runs in the background. **Running** means the application
responds; open **Scope & updates → Collection details** to verify collection.
**Applications** may be empty until endpoints are published; **Discovered**
shows eligible unlabelled sources. **Versions** shows collected workloads and
containers. Stop with Ctrl+C.

## Documentation

- [Getting started and first inventory](docs/getting-started.md)
- [Installation: source, container and Kubernetes](docs/installation.md)
- [Using the dashboards](docs/using-autodocs.md)
- [Configuration](docs/configuration.md) and [connecting clusters](docs/cluster-discovery.md)
- [Publishing endpoints](docs/publishing-endpoints.md) and [manual links](docs/manual-directory-links.md)
- [Operations](docs/operations.md), [troubleshooting](docs/troubleshooting.md) and [API reference](docs/api.md)
- [Container packages](https://github.com/tintii/goK8sAutoDocs/pkgs/container/k8sautodocs) and [release history](CHANGELOG.md)

## Contributing

Fork the repository, create a feature branch, make a focused change with relevant
tests and submit a pull request. The [OpenSpec specifications](openspec/specs/)
and [implementation archive](openspec/changes/archive/) hold behavior contracts
and design history. [Development and documentation maintenance](openspec/changes/publish-user-manual-on-github-pages/implementation.md)
records build, test and publishing instructions.
