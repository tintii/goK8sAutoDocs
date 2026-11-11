"""Build once, then check emitted HTML/CSS links against the Pages project path.

Run from any directory with the pinned documentation environment's Python.
No network requests are made by this checker.
"""

from html.parser import HTMLParser
from pathlib import Path
import re
import subprocess
import sys
import tempfile
from urllib.parse import unquote, urljoin, urlsplit

ROOT = Path(__file__).resolve().parents[1]
BASE = "https://tintii.github.io/goK8sAutoDocs/"


class Page(HTMLParser):
    def __init__(self, source):
        super().__init__()
        self.ids = set()
        self.links = []
        self.feed(source)

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if attrs.get("id"):
            self.ids.add(attrs["id"])
        if tag == "a" and attrs.get("name"):
            self.ids.add(attrs["name"])
        for key in ("href", "src", "poster", "data"):
            if attrs.get(key):
                self.links.append(attrs[key])
        if attrs.get("srcset"):
            self.links.extend(item.strip().split()[0] for item in attrs["srcset"].split(",") if item.strip())
        self.links.extend(css_links(attrs.get("style", "")))


def css_links(source):
    return [match[1].strip() for match in re.findall(r"url\(\s*(['\"]?)(.*?)\1\s*\)", source)]


def check_site(site):
    """Return all broken internal file/fragment links, including CSS assets."""
    site = site.resolve()
    files = list(site.rglob("*"))
    pages = {p: Page(p.read_text()) for p in files if p.suffix == ".html"}
    errors = []
    for source in files:
        if source.is_symlink():
            errors.append(f"Symlink in publication: {source.relative_to(site)}")
            continue
        if source in pages:
            links = pages[source].links
        elif source.suffix == ".css":
            links = css_links(source.read_text())
        else:
            continue
        relative = source.relative_to(site).as_posix()
        for link in links:
            url = urlsplit(urljoin(BASE + relative, link))
            base = urlsplit(BASE)
            if (url.scheme, url.netloc) != (base.scheme, base.netloc):
                continue
            path = unquote(url.path)
            if not path.startswith(base.path):
                errors.append(f"{relative}: outside project path: {link}")
                continue
            target = (site / path[len(base.path):]).resolve()
            if not target.is_relative_to(site):
                errors.append(f"{relative}: outside site: {link}")
                continue
            if target.is_dir():
                target /= "index.html"
            if not target.is_file():
                errors.append(f"{relative}: missing file: {link}")
            elif url.fragment and target in pages and unquote(url.fragment) not in pages[target].ids:
                errors.append(f"{relative}: missing fragment: {link}")
    return errors


def check_failure_fixtures():
    # These exercise the emitted-output checker, independently of MkDocs.
    with tempfile.TemporaryDirectory() as directory:
        site = Path(directory)
        (site / "index.html").write_text('<h1 id="good">Good</h1><a href="#good">Good</a>')
        assert not check_site(site), "valid fixture rejected"
        for link in ('missing/', '#absent', 'missing.png', '/outside/', '../outside/'):
            (site / "index.html").write_text(f'<a href="{link}">Broken</a>')
            assert check_site(site), f"broken-link fixture passed: {link}"
        (site / "index.html").write_text('<link rel="stylesheet" href="test.css">')
        (site / "test.css").write_text('body { background: url("missing.svg"); }')
        assert check_site(site), "broken CSS asset fixture passed"
    print("Broken page, fragment, asset and project-path fixtures rejected.", flush=True)


def main():
    check_failure_fixtures()
    subprocess.run([sys.executable, "-m", "mkdocs", "build", "--strict"], cwd=ROOT, check=True)
    errors = check_site(ROOT / "site")
    if errors:
        raise SystemExit("\n".join(errors))
    print("Generated HTML pages, fragments and HTML/CSS assets are valid under /goK8sAutoDocs/.")


if __name__ == "__main__":
    main()
