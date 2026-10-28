#!/usr/bin/env python3
"""Check the actual runtime image without real Kubernetes credentials or an API."""

from html.parser import HTMLParser
import base64
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
from pathlib import Path
import ssl
import subprocess
import sys
import tempfile
import threading
import time
from urllib.parse import urljoin, urlparse
from urllib.request import Request, urlopen


ROOT = Path(__file__).resolve().parents[1]


def docker(*args):
    return subprocess.check_output(["docker", *args], text=True).strip()


class KubernetesFixture(BaseHTTPRequestHandler):
    failed = threading.Event()
    calls = 0

    def log_message(self, *_):
        pass

    def do_GET(self):
        type(self).calls += 1
        if self.failed.is_set():
            self.send_response(503)
            self.end_headers()
            return
        cluster, _, resource_path = self.path.lstrip("/").partition("/")
        path = "/" + resource_path.split("?")[0]
        resource = path.rsplit("/", 1)[-1]
        if resource == "httproutes":
            self.send_response(404)
            self.end_headers()
            return
        api_version = "apps/v1" if "/apps/" in path else "networking.k8s.io/v1" if "/networking.k8s.io/" in path else "v1"
        kind = {"deployments": "Deployment", "statefulsets": "StatefulSet", "daemonsets": "DaemonSet",
                "ingresses": "Ingress", "services": "Service", "configmaps": "ConfigMap", "pods": "Pod"}.get(resource)
        items = []
        if resource == "deployments":
            items = [{"metadata": {"name": "same", "namespace": "apps", "generation": 1},
                      "spec": {"replicas": 1, "template": {"spec": {"containers": [{"name": "app", "image": "example/app:v1"}]}}},
                      "status": {"availableReplicas": 1, "observedGeneration": 1}}]
        if resource == "ingresses":
            items = [{"metadata": {"name": "same", "namespace": "apps", "labels": {"k8sautodocs.io/publish": "true"}},
                      "spec": {"rules": [{"host": cluster + ".example.invalid"}]}}]
        payload = {"apiVersion": api_version, "kind": (kind or "List") + "List", "items": items}
        if "/namespaces/" in path and resource == "apps":
            payload = {"apiVersion": "v1", "kind": "Namespace", "metadata": {"name": "apps", "labels": {"fixture/environment": "NPE00" if cluster == "a" else "npe00"}}}
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(payload).encode())


def read_json(origin, path, method="GET", body=None):
    data = None if body is None else json.dumps(body).encode()
    request = Request(origin + path, data=data, method=method, headers={"Content-Type": "application/json"})
    with urlopen(request, timeout=5) as response:
        return json.load(response)


def wait_for(check, message):
    for _ in range(400):
        try:
            result = check()
            if result:
                return result
        except OSError:
            pass
        time.sleep(0.1)
    raise RuntimeError(message)


class Assets(HTMLParser):
    def __init__(self):
        super().__init__()
        self.paths = set()

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        target = attrs.get("src") if tag in {"script", "img"} else None
        if tag == "link" and "stylesheet" in attrs.get("rel", "").split():
            target = attrs.get("href")
        if target and not urlparse(target).netloc and not urlparse(target).scheme:
            self.paths.add(target)


def main():
    image = sys.argv[1] if len(sys.argv) == 2 else "k8sautodocs:ci"
    image_info = json.loads(docker("image", "inspect", image))[0]
    config = image_info["Config"]
    if config.get("User", "").split(":")[0] != "65532":
        raise RuntimeError("Expected the image's default non-root user 65532")
    container = None
    with tempfile.TemporaryDirectory(prefix="k8sautodocs-smoke-") as directory:
        fixture = Path(directory) / "kube"
        fixture.mkdir(mode=0o755)
        cache = Path(directory) / "cache"
        cache.mkdir(mode=0o777)
        cache.chmod(0o777)  # Fixture-only writable bind mount for image UID 65532.
        cert, key = Path(directory) / "ca.crt", Path(directory) / "ca.key"
        subprocess.check_call(["openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "1",
                               "-keyout", str(key), "-out", str(cert), "-subj", "/CN=host.docker.internal",
                               "-addext", "subjectAltName=DNS:host.docker.internal"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        server = ThreadingHTTPServer(("0.0.0.0", 0), KubernetesFixture)
        tls = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        tls.load_cert_chain(cert, key)
        server.socket = tls.wrap_socket(server.socket, server_side=True)
        threading.Thread(target=server.serve_forever, daemon=True).start()
        ca = base64.b64encode(cert.read_bytes()).decode()
        clusters = []
        for target in ["a", "b"]:
            kubeconfig = {
                "apiVersion": "v1", "kind": "Config",
                "clusters": [{"name": "same", "cluster": {"server": f"https://host.docker.internal:{server.server_port}/{target}", "certificate-authority-data": ca}}],
                "users": [{"name": "same", "user": {"token": "fixture-token-never-persist"}}],
                "contexts": [{"name": "same", "context": {"cluster": "same", "user": "same"}}], "current-context": "same"}
            (fixture / (target + ".json")).write_text(json.dumps(kubeconfig))
            clusters.append({"id": target, "kubeconfig": {"path": "/fixtures/" + target + ".json", "context": "same"}})
        configuration = {"clusters": clusters, "legacyClusterID": "a", "classification": {"environment": {"labelKeys": ["fixture/environment"], "sources": ["namespace"], "caseSensitive": False, "displayCase": "upper"}},
                         "cache": {"path": "/cache/snapshot.json"}, "refresh": {"interval": "1m", "cooldown": "1s", "clusterTimeout": "3s"}}
        (fixture / "autodocs.json").write_text(json.dumps(configuration))
        for file in fixture.iterdir():
            file.chmod(0o644)
        host_arguments = ["--add-host", "host.docker.internal:host-gateway"] if sys.platform.startswith("linux") else []
        try:
            container = docker(
                "run", "--detach", "--read-only", "--cap-drop=ALL",
                "--platform", f"{image_info['Os']}/{image_info['Architecture']}",
                "--security-opt=no-new-privileges", "--publish", "127.0.0.1::8080",
                "--mount", f"type=bind,source={fixture},target=/fixtures,readonly",
                "--mount", f"type=bind,source={cache},target=/cache",
                "--env", "AUTODOCS_CONFIG=/fixtures/autodocs.json", *host_arguments, image,
            )
            inspect = json.loads(docker("inspect", container))[0]
            ports = inspect["NetworkSettings"]["Ports"].get("8080/tcp")
            if not ports:
                raise RuntimeError("Container exited before publishing HTTP port 8080")
            port = ports[0]["HostPort"]
            origin = f"http://127.0.0.1:{port}"
            for attempt in range(60):
                try:
                    with urlopen(origin + "/", timeout=1) as response:
                        response.read()
                    break
                except OSError:
                    if docker("inspect", container, "--format", "{{.State.Running}}") != "true":
                        raise RuntimeError("Container exited before becoming ready")
                    if attempt == 59:
                        raise RuntimeError("Container did not become ready")
                    time.sleep(0.5)
            pages = {
                "/": "index.html",
                "/versions-dashboard-page": "versions-dashboard.html",
                "/autodocs-dashboard-page": "autodocs-dashboard.html",
            }
            paths = {path: ROOT / "go/static" / file for path, file in pages.items()}
            for path, file in list(paths.items()):
                parser = Assets()
                parser.feed(file.read_text())
                for asset in parser.paths:
                    asset_path = urlparse(urljoin(path, asset)).path
                    if not asset_path.startswith("/static/"):
                        raise RuntimeError(f"Unexpected local asset path: {asset_path}")
                    paths[asset_path] = ROOT / "go" / asset_path.lstrip("/")
            for path, source in paths.items():
                with urlopen(origin + path, timeout=5) as response:
                    if response.read() != source.read_bytes():
                        raise RuntimeError(f"Image serves incorrect or missing content at {path}")
                print(f"PASS {path}")
            with urlopen(origin + "/api/v1/health", timeout=5) as response:
                if json.load(response) != {"status": "running"} or response.headers.get("Cache-Control") != "no-store":
                    raise RuntimeError("Expected uncached application health independent of Kubernetes")
            print("PASS application health independent of fixture Kubernetes API")
            def collected():
                status = read_json(origin, "/api/v1/discovery-status")
                return status if status["generation"] >= 1 and not status["running"] else None
            status = wait_for(collected, "Fixture discovery did not finish")
            inventory = read_json(origin, "/api/v1/auto-docs-dashboard")
            if len(inventory) != 2 or {row["clusterId"] for row in inventory} != {"a", "b"}:
                raise RuntimeError("Expected separate source identities from both fixture clusters")
            if {row["classification"]["environment"]["display"] for row in inventory} != {"NPE00"}:
                raise RuntimeError("Expected canonical namespace environment classification")
            count = KubernetesFixture.calls
            for _ in range(5):
                read_json(origin, "/api/v1/auto-docs-dashboard")
                read_json(origin, "/api/v1/versions-dashboard")
            if KubernetesFixture.calls != count:
                raise RuntimeError("Inventory GETs triggered Kubernetes collection")
            time.sleep(1.1)
            generation = status["generation"]
            read_json(origin, "/api/v1/discovery-refresh", "POST", {"generation": generation})
            read_json(origin, "/api/v1/discovery-refresh", "POST", {"generation": generation})
            status = wait_for(lambda: (value if (value := collected()) and value["generation"] == generation + 1 else None), "Shared refresh did not complete")
            print("PASS two TLS kubeconfig targets, classification, GET isolation and coalesced manual update")
            saved_at = status["clusters"][0]["parts"]["ingresses"]["lastSuccessAt"]
            KubernetesFixture.failed.set()
            docker("restart", container)
            # Docker may reassign an ephemeral published host port on restart.
            inspect = json.loads(docker("inspect", container))[0]
            port = inspect["NetworkSettings"]["Ports"]["8080/tcp"][0]["HostPort"]
            origin = f"http://127.0.0.1:{port}"
            status = wait_for(lambda: (value if (value := collected()) and value["generation"] > generation + 1 else None), "Restart refresh did not complete")
            if status["cacheError"] or status["clusters"][0]["parts"]["ingresses"]["lastSuccessAt"] != saved_at:
                raise RuntimeError("Persistent hydration lost original collection timestamp")
            if len(read_json(origin, "/api/v1/auto-docs-dashboard")) != 2:
                raise RuntimeError("Failed refresh discarded restored inventory")
            snapshot = Path(directory) / "saved-snapshot.json"
            docker("cp", f"{container}:/cache/snapshot.json", str(snapshot))
            if "fixture-token-never-persist" in snapshot.read_text():
                raise RuntimeError("Credential leaked into snapshot")
            print("PASS persistent startup recovery, stale retention and credential-free snapshot")
            bundle = Path(directory) / "ca-certificates.crt"
            docker("cp", f"{container}:/etc/ssl/certs/ca-certificates.crt", str(bundle))
            if "BEGIN CERTIFICATE" not in bundle.read_text():
                raise RuntimeError("Runtime CA certificates missing")
            print("PASS default non-root user, read-only runtime and CA certificates")
        except Exception:
            if container:
                print(docker("logs", container), file=sys.stderr)
            raise
        finally:
            server.shutdown()
            server.server_close()
            if container:
                docker("rm", "--force", container)


if __name__ == "__main__":
    main()
