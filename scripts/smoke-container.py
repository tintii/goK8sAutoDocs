#!/usr/bin/env python3
"""Check the actual runtime image without real Kubernetes credentials or an API."""

from html.parser import HTMLParser
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import time
from urllib.parse import urljoin, urlparse
from urllib.request import urlopen


ROOT = Path(__file__).resolve().parents[1]


def docker(*args):
    return subprocess.check_output(["docker", *args], text=True).strip()


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
        # Docker supplies the distroless non-root user's /home/nonroot home.
        (fixture / "config").write_text("""apiVersion: v1
kind: Config
clusters:
  - name: fixture
    cluster:
      server: http://127.0.0.1:9
contexts:
  - name: fixture
    context:
      cluster: fixture
      user: fixture
current-context: fixture
users:
  - name: fixture
    user: {}
""")
        (fixture / "config").chmod(0o644)
        try:
            container = docker(
                "run", "--detach", "--read-only", "--cap-drop=ALL",
                "--platform", f"{image_info['Os']}/{image_info['Architecture']}",
                "--security-opt=no-new-privileges", "--publish", "127.0.0.1::8080",
                "--mount", f"type=bind,source={fixture},target=/home/nonroot/.kube,readonly", image,
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
            if container:
                docker("rm", "--force", container)


if __name__ == "__main__":
    main()
