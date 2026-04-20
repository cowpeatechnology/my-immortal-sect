#!/usr/bin/env python3
"""Mirror documentation sites for local reading with an official-like layout.

This tool does not convert docs back from Markdown. Instead it mirrors the
rendered HTML site inside a constrained prefix and serves the mirrored files
locally. This keeps the official navigation shell, styles, and page layout.
"""

from __future__ import annotations

import argparse
import json
import mimetypes
import os
import subprocess
import sys
from dataclasses import dataclass
from datetime import datetime, timezone
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Iterable
from urllib.parse import urlparse


DEFAULT_LIBRARY_ROOT = Path(
    "/Users/mawei/MyWork/我的知识库/Projects/我的宗门 Wiki/raw/assets/docsite-mirrors"
)
DEFAULT_HOST = "127.0.0.1"
DEFAULT_PORT = 8768


class DocsiteMirrorError(RuntimeError):
    """Raised when mirror operations fail."""


@dataclass
class MirrorRecord:
    slug: str
    title: str
    start_url: str
    output_dir: Path
    public_root: Path
    local_entry: Path | None
    created_at: str
    include_prefixes: list[str]

    def to_json(self) -> dict:
        return {
            "slug": self.slug,
            "title": self.title,
            "start_url": self.start_url,
            "output_dir": str(self.output_dir),
            "public_root": str(self.public_root),
            "local_entry": str(self.local_entry) if self.local_entry else None,
            "created_at": self.created_at,
            "include_prefixes": self.include_prefixes,
        }


def write_text(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


def run(cmd: list[str], *, cwd: Path | None = None) -> None:
    subprocess.run(cmd, cwd=str(cwd) if cwd else None, check=True)


def default_slug_from_url(start_url: str) -> str:
    parsed = urlparse(start_url)
    host = parsed.netloc.replace(":", "-")
    path = parsed.path.strip("/")
    if not path:
        return host
    parts = [part for part in path.split("/") if part]
    return "-".join([host] + parts)


def default_title_from_url(start_url: str) -> str:
    parsed = urlparse(start_url)
    return parsed.netloc + parsed.path


def default_include_prefixes(start_url: str) -> list[str]:
    parsed = urlparse(start_url)
    path = parsed.path.rstrip("/")
    if not path:
        return ["/"]
    parts = [part for part in path.split("/") if part]
    if parts and parts[-1] in {"zh", "en"}:
        base = "/" + "/".join(parts[:-1])
        return [path or "/", base + "/assets"]
    return [path or "/"]


def find_local_entry(public_root: Path, start_url: str) -> Path | None:
    parsed = urlparse(start_url)
    relative = parsed.path.lstrip("/")
    candidate = public_root / relative
    if candidate.is_dir():
        index = candidate / "index.html"
        if index.exists():
            return index
    if candidate.exists():
        return candidate
    html_candidate = public_root / (relative + ".html")
    if html_candidate.exists():
        return html_candidate
    return None


def mirror_site(
    *,
    start_url: str,
    output_dir: Path,
    slug: str,
    title: str,
    include_prefixes: list[str],
) -> MirrorRecord:
    output_dir.mkdir(parents=True, exist_ok=True)

    parsed = urlparse(start_url)
    if not parsed.scheme or not parsed.netloc:
        raise DocsiteMirrorError(f"Invalid start URL: {start_url}")

    include_args: list[str] = []
    if include_prefixes:
        include_args = ["--include-directories=" + ",".join(include_prefixes)]

    wget_cmd = [
        "wget",
        "--mirror",
        "--page-requisites",
        "--convert-links",
        "--adjust-extension",
        "--compression=auto",
        "--execute=robots=off",
        "--domains",
        parsed.netloc,
        "--no-verbose",
        "--directory-prefix",
        str(output_dir),
        *include_args,
        start_url,
    ]
    run(wget_cmd)

    public_root = output_dir / parsed.netloc
    local_entry = find_local_entry(public_root, start_url)
    created_at = datetime.now(timezone.utc).isoformat()

    record = MirrorRecord(
        slug=slug,
        title=title,
        start_url=start_url,
        output_dir=output_dir,
        public_root=public_root,
        local_entry=local_entry,
        created_at=created_at,
        include_prefixes=include_prefixes,
    )
    write_text(output_dir / "_docsite_mirror.json", json.dumps(record.to_json(), ensure_ascii=False, indent=2) + "\n")
    write_text(
        output_dir / "00-镜像说明.md",
        "\n".join(
            [
                f"# {title}",
                "",
                "这是一个本地 HTML 站点镜像，用于尽量保留官网阅读体验。",
                "",
                f"- 起始 URL：`{start_url}`",
                f"- 创建时间（UTC）：`{created_at}`",
                f"- 允许抓取前缀：`{', '.join(include_prefixes)}`",
                f"- 本地入口：`{local_entry}`" if local_entry else "- 本地入口：未自动识别",
                "",
                "请优先通过 `docsite_mirror.py serve` 提供的本地服务访问，而不是直接双击 HTML 文件。",
            ]
        )
        + "\n",
    )
    return record


def load_records(library_root: Path) -> list[MirrorRecord]:
    records: list[MirrorRecord] = []
    if not library_root.exists():
        return records

    for manifest in sorted(library_root.rglob("_docsite_mirror.json")):
        payload = json.loads(manifest.read_text(encoding="utf-8"))
        local_entry = payload.get("local_entry")
        record = MirrorRecord(
            slug=payload["slug"],
            title=payload["title"],
            start_url=payload["start_url"],
            output_dir=Path(payload["output_dir"]),
            public_root=Path(payload["public_root"]),
            local_entry=Path(local_entry) if local_entry else None,
            created_at=payload["created_at"],
            include_prefixes=list(payload.get("include_prefixes", [])),
        )
        records.append(record)
    return records


def html_index(records: list[MirrorRecord]) -> str:
    cards = []
    for record in records:
        entry_href = "/mirror/" + record.slug + "/"
        cards.append(
            f"""
            <article class="card">
              <div class="eyebrow">Mirror</div>
              <h2>{record.title}</h2>
              <p class="url">{record.start_url}</p>
              <p class="meta">Created {record.created_at}</p>
              <a class="open" href="{entry_href}">Open Site</a>
            </article>
            """
        )

    return f"""<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Local Docsite Mirrors</title>
  <style>
    :root {{
      --bg: #f4f6fb;
      --ink: #10203a;
      --muted: #5f6f86;
      --panel: rgba(255,255,255,0.9);
      --line: rgba(16,32,58,0.10);
      --accent: #2b6fff;
      --accent-soft: rgba(43,111,255,0.1);
      --shadow: 0 20px 60px rgba(17, 32, 61, 0.08);
      --radius: 22px;
      --font: "Avenir Next", "PingFang SC", "Noto Sans SC", "Helvetica Neue", sans-serif;
    }}
    * {{ box-sizing: border-box; }}
    body {{
      margin: 0;
      font-family: var(--font);
      background:
        radial-gradient(circle at top left, rgba(43,111,255,0.14), transparent 28%),
        radial-gradient(circle at right center, rgba(16,32,58,0.06), transparent 30%),
        var(--bg);
      color: var(--ink);
    }}
    .shell {{
      max-width: 1120px;
      margin: 0 auto;
      padding: 48px 24px 72px;
    }}
    .hero {{
      padding: 28px 30px;
      border: 1px solid var(--line);
      border-radius: 30px;
      background: linear-gradient(180deg, rgba(255,255,255,0.92), rgba(255,255,255,0.76));
      box-shadow: var(--shadow);
      backdrop-filter: blur(14px);
    }}
    .hero .eyebrow {{
      font-size: 12px;
      letter-spacing: 0.18em;
      text-transform: uppercase;
      color: var(--muted);
      margin-bottom: 10px;
    }}
    .hero h1 {{
      margin: 0;
      font-size: clamp(32px, 6vw, 58px);
      line-height: 0.95;
      letter-spacing: -0.04em;
    }}
    .hero p {{
      max-width: 760px;
      margin: 18px 0 0;
      color: var(--muted);
      font-size: 16px;
      line-height: 1.7;
    }}
    .grid {{
      margin-top: 28px;
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 18px;
    }}
    .card {{
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: var(--radius);
      padding: 22px;
      box-shadow: var(--shadow);
    }}
    .card .eyebrow {{
      color: var(--muted);
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: 0.14em;
    }}
    .card h2 {{
      margin: 10px 0 12px;
      font-size: 24px;
      line-height: 1.1;
      letter-spacing: -0.03em;
    }}
    .url {{
      font-size: 13px;
      color: var(--muted);
      word-break: break-all;
      margin: 0 0 14px;
    }}
    .meta {{
      margin: 0 0 18px;
      color: var(--muted);
      font-size: 13px;
    }}
    .open {{
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-width: 120px;
      padding: 12px 16px;
      border-radius: 999px;
      background: var(--accent);
      color: white;
      text-decoration: none;
      font-weight: 600;
      box-shadow: 0 14px 24px rgba(43,111,255,0.18);
    }}
    .empty {{
      margin-top: 28px;
      padding: 22px;
      border-radius: var(--radius);
      border: 1px dashed var(--line);
      color: var(--muted);
      background: rgba(255,255,255,0.72);
    }}
  </style>
</head>
<body>
  <main class="shell">
    <section class="hero">
      <div class="eyebrow">Local Reading</div>
      <h1>Docsite Mirrors</h1>
      <p>这里提供本地官方文档镜像入口。阅读时优先保留官网的导航、样式和页面组织，而不是只看裸 Markdown。</p>
    </section>
    {"<section class='grid'>" + "".join(cards) + "</section>" if cards else "<section class='empty'>当前还没有镜像站点。先运行 docsite_mirror.py mirror。</section>"}
  </main>
</body>
</html>
"""


class MirrorHttpHandler(BaseHTTPRequestHandler):
    server_version = "DocsiteMirror/0.1"

    def do_GET(self) -> None:  # noqa: N802
        server: MirrorHttpServer = self.server  # type: ignore[assignment]
        path = self.path.split("?", 1)[0]

        if path == "/":
            payload = html_index(server.records).encode("utf-8")
            self.send_response(HTTPStatus.OK)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.send_header("Content-Length", str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)
            return

        if path == "/api/mirrors":
            payload = json.dumps([record.to_json() for record in server.records], ensure_ascii=False, indent=2).encode("utf-8")
            self.send_response(HTTPStatus.OK)
            self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Content-Length", str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)
            return

        if path.startswith("/mirror/"):
            self._serve_mirror_file(server, path)
            return

        self.send_error(HTTPStatus.NOT_FOUND, "Not Found")

    def _serve_mirror_file(self, server: "MirrorHttpServer", path: str) -> None:
        suffix = path[len("/mirror/") :]
        slug, _, rest = suffix.partition("/")
        record = next((item for item in server.records if item.slug == slug), None)
        if record is None:
            self.send_error(HTTPStatus.NOT_FOUND, "Unknown mirror")
            return

        relative = rest or ""
        if not relative:
            target = record.local_entry or record.public_root
        else:
            target = record.public_root / relative

        if target.is_dir():
            index = target / "index.html"
            if index.exists():
                target = index
            else:
                self.send_error(HTTPStatus.NOT_FOUND, "Directory without index")
                return

        try:
            resolved = target.resolve(strict=True)
        except FileNotFoundError:
            self.send_error(HTTPStatus.NOT_FOUND, "Missing file")
            return

        try:
            resolved.relative_to(record.public_root.resolve())
        except ValueError:
            self.send_error(HTTPStatus.FORBIDDEN, "Path escapes mirror root")
            return

        mime, _ = mimetypes.guess_type(resolved.name)
        self.send_response(HTTPStatus.OK)
        self.send_header("Content-Type", mime or "application/octet-stream")
        self.send_header("Content-Length", str(resolved.stat().st_size))
        self.end_headers()
        with resolved.open("rb") as handle:
            self.wfile.write(handle.read())

    def log_message(self, fmt: str, *args) -> None:  # noqa: A003
        return


class MirrorHttpServer(ThreadingHTTPServer):
    def __init__(self, server_address: tuple[str, int], records: list[MirrorRecord]) -> None:
        super().__init__(server_address, MirrorHttpHandler)
        self.records = records


def serve_library(*, library_root: Path, host: str, port: int) -> None:
    records = load_records(library_root)
    server = MirrorHttpServer((host, port), records)
    print(json.dumps(
        {
            "host": host,
            "port": port,
            "library_root": str(library_root),
            "mirror_count": len(records),
            "url": f"http://{host}:{port}/",
        },
        ensure_ascii=False,
        indent=2,
    ))
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nShutting down docsite mirror server.")
    finally:
        server.server_close()


def parse_args(argv: Iterable[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Mirror and serve official-style docsites locally.")
    subparsers = parser.add_subparsers(dest="command", required=True)

    mirror = subparsers.add_parser("mirror", help="Mirror one documentation site.")
    mirror.add_argument("--start-url", required=True, help="Site entry URL.")
    mirror.add_argument("--slug", help="Mirror slug. Default: derived from URL.")
    mirror.add_argument("--title", help="Display title. Default: derived from URL.")
    mirror.add_argument(
        "--include-prefix",
        action="append",
        default=[],
        help="Allowed URL path prefix on the same host. Repeat for multiple prefixes.",
    )
    mirror.add_argument(
        "--output-dir",
        help="Target output directory. Default: <library-root>/<slug>",
    )
    mirror.add_argument(
        "--library-root",
        default=str(DEFAULT_LIBRARY_ROOT),
        help=f'Mirror library root. Default: "{DEFAULT_LIBRARY_ROOT}"',
    )

    serve = subparsers.add_parser("serve", help="Serve mirrored sites locally.")
    serve.add_argument(
        "--library-root",
        default=str(DEFAULT_LIBRARY_ROOT),
        help=f'Mirror library root. Default: "{DEFAULT_LIBRARY_ROOT}"',
    )
    serve.add_argument("--host", default=DEFAULT_HOST, help=f'Bind host. Default: "{DEFAULT_HOST}"')
    serve.add_argument("--port", type=int, default=DEFAULT_PORT, help=f"Bind port. Default: {DEFAULT_PORT}")

    return parser.parse_args(list(argv))


def main(argv: Iterable[str]) -> int:
    args = parse_args(argv)
    if args.command == "mirror":
        start_url = args.start_url.rstrip("/") + "/"
        slug = args.slug or default_slug_from_url(start_url)
        title = args.title or default_title_from_url(start_url)
        library_root = Path(args.library_root).expanduser()
        output_dir = Path(args.output_dir).expanduser() if args.output_dir else library_root / slug
        include_prefixes = args.include_prefix or default_include_prefixes(start_url)
        record = mirror_site(
            start_url=start_url,
            output_dir=output_dir,
            slug=slug,
            title=title,
            include_prefixes=include_prefixes,
        )
        print(json.dumps(record.to_json(), ensure_ascii=False, indent=2))
        return 0

    if args.command == "serve":
        serve_library(
            library_root=Path(args.library_root).expanduser(),
            host=args.host,
            port=args.port,
        )
        return 0

    raise DocsiteMirrorError(f"Unsupported command: {args.command}")


if __name__ == "__main__":
    try:
        raise SystemExit(main(sys.argv[1:]))
    except KeyboardInterrupt:
        raise SystemExit(130)
