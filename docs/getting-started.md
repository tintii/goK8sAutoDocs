# Getting started

## Run from source

Install Git and Go **1.24.0 or later**; the module selects toolchain **1.24.5** (Go can download it automatically). You need read-only Kubernetes credentials and a free local port **8080**.

Without `AUTODOCS_CONFIG`, AutoDocs reads the current context from **`~/.kube/config`** when outside Kubernetes. It does **not** use `KUBECONFIG` or merge several kubeconfig files. The selected context must use verified HTTPS and self-contained credentials: an embedded bearer token or embedded client certificate/key, with embedded CA data when required. External file references, exec/auth-provider login helpers, impersonation and insecure TLS are unsupported. Prepare a dedicated read-only context if your normal context uses one of these methods; see [Connecting clusters](cluster-discovery.md#remote-cluster-secret-contract).

```bash
git clone https://github.com/tintii/goK8sAutoDocs.git
cd goK8sAutoDocs/go
go mod download
go build -o goK8sAutoDocs .
./goK8sAutoDocs
```

Keep the working directory at `go/`: HTML and static files are served relative to it. Open **<http://localhost:8080/>**. Stop with Ctrl+C.

## Check your first inventory

The header should show **Running**, which means AutoDocs responds, independently of Kubernetes collection. Startup discovery runs in the background. Open **Scope & updates → Collection details** to inspect target/component results. **Applications** can be empty until you publish endpoints or add manual links; **Discovered** shows eligible unlabelled routes and ExternalDNS Services. Open **Versions** to see Deployments, StatefulSets and DaemonSets and their containers. Successful empty collections are valid; a 503 means no usable inventory yet.

If a component reports access forbidden, grant the [collector permissions](installation.md#kubernetes-permissions), then use **Update now** after the cooldown. See [Troubleshooting](troubleshooting.md) for unsupported credentials or connection failures. Browser polling reads snapshots and does not make Kubernetes requests.

Next: [publish endpoints](publishing-endpoints.md), [add manual links](manual-directory-links.md) or [connect another cluster](cluster-discovery.md). For container and Kubernetes deployment, follow [Installation](installation.md).
