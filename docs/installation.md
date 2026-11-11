# Installation

This manual follows `master`. Rancher discovery is available in **v0.6.0-RC1**; manual directory links are **unreleased**. Check the [release history](https://github.com/tintii/goK8sAutoDocs/blob/master/CHANGELOG.md) before using these features with an older image.

Choose source for local evaluation, a published container for an existing runtime, or Kubernetes for persistent operation. All paths need reachable cluster APIs and the [read permissions below](#kubernetes-permissions).

## Local source

Follow [Getting started](getting-started.md). Keep the built binary next to `static/` and launch from that directory. An explicit `AUTODOCS_CONFIG` can select a different mounted kubeconfig/context; see [Connecting clusters](cluster-discovery.md).

## Container

Select an exact release or full-commit tag from [GHCR](https://github.com/tintii/goK8sAutoDocs/pkgs/container/k8sautodocs), verify that it exists and resolves for `linux/amd64`, then record its digest. No moving `latest` tag is published. Replace `sha256:REPLACE_WITH_VERIFIED_DIGEST` below with that digest.

Create a local `autodocs.yaml` (use an absolute host path for mounts):

```yaml
clusters:
  - id: evaluation
    kubeconfig:
      path: /etc/autodocs/credentials/kubeconfig
      context: autodocs
```

Place a self-contained kubeconfig with context `autodocs` in a dedicated `credentials/` directory. Make the files readable to container UID/GID **65532**, without opening access unnecessarily on the host.

```bash
docker run --rm --read-only --cap-drop ALL --security-opt no-new-privileges \
  -p 127.0.0.1:8080:8080 \
  --mount type=bind,src="$PWD/autodocs.yaml",dst=/etc/autodocs/autodocs.yaml,readonly \
  --mount type=bind,src="$PWD/credentials",dst=/etc/autodocs/credentials,readonly \
  -e AUTODOCS_CONFIG=/etc/autodocs/autodocs.yaml \
  ghcr.io/tintii/k8sautodocs@sha256:REPLACE_WITH_VERIFIED_DIGEST
```

The image serves on 8080 from `/app`, includes CA certificates and runs as 65532:65532. It needs no writable root filesystem. Open <http://localhost:8080/> and check collection details. A local cluster's `localhost` API endpoint is usually unreachable from a container; use an API address reachable from that runtime.

If the package is private, authenticate the runtime using a read-only pull credential. In Kubernetes supply it with `imagePullSecrets`; never embed it in the image or commit it in values. Public images should be verified with an anonymous digest pull before rollout.

## Kubernetes

Use your deployment chart to create a dedicated ServiceAccount, the ClusterRole below and a ClusterRoleBinding to that account. Set `serviceAccountName` on the pod, allow its projected token, pin the image digest and expose container port 8080 through a Service. Use **one replica** with **Recreate** or another strategy that prevents overlap. Set `runAsUser`/`runAsGroup` to 65532, `runAsNonRoot: true` and `readOnlyRootFilesystem: true` through the chart's supported security contexts.

With no `AUTODOCS_CONFIG`, the projected ServiceAccount token selects in-cluster authentication. With an explicit registry, declare an `inCluster: true` target; no implicit local target is added. Use `/api/v1/health` on port 8080 for process liveness/readiness independently of collection results. After rollout, port-forward the Service and check **Scope & updates** and **Versions**.

For multiple targets, mount [remote credentials](cluster-discovery.md#remote-cluster-secret-contract) as directories. Mount configuration and optional [home content](configuration.md#home-page-content-and-status) read-only. For persistence, mount a writable volume at `/var/cache/autodocs` and configure `cache.path`; arrange volume permissions for 65532. Keep the rest of the filesystem read-only. See [Operations](operations.md) for cache and rollback constraints.

Chart interfaces differ: use your chart's documented ServiceAccount, RBAC, volume, security-context and strategy fields. The [example GitOps runbook](https://github.com/tintii/homelab-gitops/blob/master/docs/k8sautodocs.md) is specific to a separate homelab and is not required for installation.

## Kubernetes permissions

A ClusterRole for the current collectors is:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: autodocs-reader
rules:
  - apiGroups: [apps]
    resources: [deployments, statefulsets, daemonsets]
    verbs: [list]
  - apiGroups: [""]
    resources: [pods, configmaps, services]
    verbs: [list]
  - apiGroups: [""]
    resources: [services, namespaces]
    verbs: [get]
  - apiGroups: [networking.k8s.io]
    resources: [ingresses]
    verbs: [list]
  - apiGroups: [gateway.networking.k8s.io]
    resources: [httproutes]
    verbs: [list]
  - apiGroups: [gateway.networking.k8s.io]
    resources: [gateways]
    verbs: [get]
```

Bind this role to each collection identity. Namespace gets support classification, Gateway listener selectors and optional `kube-system` identity observations. No Secret reads, writes or watches are needed. A kubeconfig namespace does not restrict all-namespace collection. Gateway API absence is optional; forbidden APIs are collection failures. Rancher needs additional [management enumeration permissions](cluster-discovery.md#rancher-cluster-discovery).
