#!/usr/bin/env python3
"""Mirror WeChat Mini Game docs into local Markdown files.

The site does not expose source Markdown directly. This script crawls the
rendered HTML docs under ``/minigame/``, extracts the main article content,
rewrites internal links to local relative Markdown paths, downloads referenced
content assets, and emits a local Markdown knowledge set.
"""

from __future__ import annotations

import argparse
import json
import os
import posixpath
import re
import sys
import time
from collections import Counter, deque
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Iterable
from urllib.parse import urljoin, urlparse

import requests
from lxml import etree, html


DEFAULT_SITE_ROOT = "https://developers.weixin.qq.com"
DEFAULT_ALLOWED_PREFIX = "/minigame/"
DEFAULT_START_URLS = [
    "https://developers.weixin.qq.com/minigame/introduction/",
    "https://developers.weixin.qq.com/minigame/dev/guide/",
    "https://developers.weixin.qq.com/minigame/dev/api/",
    "https://developers.weixin.qq.com/minigame/dev/api-backend/",
]
DEFAULT_OUTPUT_ROOT = Path(
    "/Users/mawei/MyWork/我的知识库/Projects/我的宗门 Wiki/raw/sources/weixin-minigame-docs"
)
REQUEST_TIMEOUT = 30
USER_AGENT = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
    "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)
DOC_EXTENSIONS = {"", ".html", ".htm"}
ASSET_EXTENSIONS = {
    ".png",
    ".jpg",
    ".jpeg",
    ".gif",
    ".svg",
    ".webp",
    ".bmp",
    ".ico",
    ".mp4",
    ".webm",
    ".ogg",
    ".mp3",
    ".wav",
    ".pdf",
    ".zip",
    ".rar",
    ".7z",
}
STRIP_CLASS_SUBSTRINGS = (
    "Breadcrumb",
    "breadcrumb",
    "page-edit",
    "page-nav",
    "table-of-contents",
    "catalog",
    "copy-btn",
    "copy-code-button",
)
INDEX_PAGES = [
    ("小游戏指引", "introduction/README.md"),
    ("开发指南", "dev/guide/README.md"),
    ("客户端 API", "dev/api/README.md"),
    ("服务端 API", "dev/api-backend/README.md"),
    ("设计", "design/README.md"),
    ("运营", "product/README.md"),
    ("数据", "analysis/README.md"),
]


class MirrorError(RuntimeError):
    """Raised when the mirror script cannot continue."""


@dataclass(frozen=True)
class SidebarNode:
    title: str
    href: str | None
    children: tuple["SidebarNode", ...]


def make_session() -> requests.Session:
    session = requests.Session()
    session.headers.update({"User-Agent": USER_AGENT})
    return session


def fetch(
    session: requests.Session,
    url: str,
    *,
    binary: bool = False,
    retries: int = 3,
) -> requests.Response:
    last_error: Exception | None = None
    for attempt in range(1, retries + 1):
        try:
            response = session.get(url, timeout=REQUEST_TIMEOUT)
            response.raise_for_status()
            if not binary and (
                not response.encoding or response.encoding.lower() == "iso-8859-1"
            ):
                response.encoding = response.apparent_encoding or "utf-8"
            return response
        except Exception as exc:  # pragma: no cover - best effort retries
            last_error = exc
            if attempt == retries:
                break
            time.sleep(0.6 * attempt)
    raise MirrorError(f"Failed to fetch {url}: {last_error}")


def canonicalize_path(path: str) -> str:
    clean = re.sub(r"/+", "/", path or "/")
    if not clean.startswith("/"):
        clean = "/" + clean
    if clean.endswith("/index.html"):
        clean = clean[: -len("index.html")]
    elif clean.endswith("/index"):
        clean = clean[: -len("index")]
    if clean != "/" and clean.endswith("/") and clean.count("/") > 1:
        return clean
    return clean


def normalize_doc_url(site_root: str, allowed_prefix: str, current_url: str, href: str) -> str | None:
    raw = (href or "").strip()
    if not raw or raw.startswith(("#", "mailto:", "javascript:", "tel:")):
        return None

    full = urljoin(current_url, raw)
    parsed = urlparse(full)
    site = urlparse(site_root)
    if parsed.scheme not in {"http", "https"}:
        return None
    if parsed.netloc != site.netloc:
        return None

    path = canonicalize_path(parsed.path)
    if not path.startswith(allowed_prefix):
        return None

    suffix = Path(path).suffix.lower()
    if suffix and suffix not in DOC_EXTENSIONS:
        return None

    return f"{site.scheme}://{site.netloc}{path}"


def url_to_local_doc_path(url: str, allowed_prefix: str) -> Path:
    parsed = urlparse(url)
    path = canonicalize_path(parsed.path)
    relative = path[len(allowed_prefix) :].lstrip("/")
    if not relative:
        return Path("README.md")
    suffix = Path(relative).suffix.lower()
    if path.endswith("/"):
        return Path(relative) / "README.md"
    if suffix in {".html", ".htm"}:
        return Path(relative).with_suffix(".md")
    if not suffix:
        return Path(relative + ".md")
    return Path(relative)


def url_to_local_asset_path(url: str) -> Path:
    parsed = urlparse(url)
    path = canonicalize_path(parsed.path)
    relative = path.lstrip("/") or "index"
    return Path("_assets") / parsed.netloc / relative


def relative_path(from_path: Path, to_path: Path) -> str:
    return Path(os.path.relpath(to_path, from_path.parent)).as_posix()


def clean_text(text: str) -> str:
    return " ".join(text.split())


def escape_markdown_heading_text(text: str) -> str:
    return text.replace("<", "&lt;").replace(">", "&gt;").strip()


def escape_markdown_link_text(text: str) -> str:
    escaped = escape_markdown_heading_text(text)
    escaped = escaped.replace("\\", "\\\\")
    escaped = escaped.replace("[", "\\[").replace("]", "\\]")
    return escaped


def find_content_node(root: html.HtmlElement) -> html.HtmlElement:
    candidates = [
        '//div[@id="docContent"]//div[contains(concat(" ", normalize-space(@class), " "), " content ")]',
        '//main[contains(concat(" ", normalize-space(@class), " "), " page ")]//div[contains(concat(" ", normalize-space(@class), " "), " content ")]',
        '//main[contains(concat(" ", normalize-space(@class), " "), " page ")]',
    ]
    for xpath in candidates:
        nodes = root.xpath(xpath)
        if nodes:
            return nodes[0]
    raise MirrorError("Could not locate page content node")


def element_has_strip_class(node: etree._Element) -> bool:
    cls = node.get("class", "")
    return any(part in cls for part in STRIP_CLASS_SUBSTRINGS)


def strip_unwanted_nodes(content: html.HtmlElement) -> None:
    for node in content.xpath('.//a[contains(concat(" ", normalize-space(@class), " "), " header-anchor ")]'):
        parent = node.getparent()
        if parent is None:
            continue
        tail = node.tail or ""
        previous = node.getprevious()
        if previous is not None:
            previous.tail = (previous.tail or "") + tail
        else:
            parent.text = (parent.text or "") + tail
        parent.remove(node)

    for xpath in (
        ".//script",
        ".//style",
        './/*[contains(@class, "copy-code-button")]',
        './/*[contains(@class, "copy-btn")]',
        ".//button",
    ):
        for node in content.xpath(xpath):
            parent = node.getparent()
            if parent is not None:
                parent.remove(node)

    for node in list(content.iter()):
        if not isinstance(node.tag, str):
            continue
        if element_has_strip_class(node):
            parent = node.getparent()
            if parent is not None:
                parent.remove(node)


def rewrite_internal_link(
    href: str,
    *,
    current_url: str,
    current_local_path: Path,
    site_root: str,
    allowed_prefix: str,
) -> str:
    raw = (href or "").strip()
    if not raw or raw.startswith(("#", "mailto:", "javascript:", "tel:")):
        return raw

    anchor = ""
    if "#" in raw:
        raw, frag = raw.split("#", 1)
        anchor = "#" + frag
    if "?" in raw:
        raw = raw.split("?", 1)[0]

    target_url = normalize_doc_url(site_root, allowed_prefix, current_url, raw)
    if target_url is None:
        return href

    target_local = url_to_local_doc_path(target_url, allowed_prefix)
    return relative_path(current_local_path, target_local) + anchor


def rewrite_asset_url(
    value: str,
    *,
    current_url: str,
    current_local_path: Path,
    session: requests.Session,
    downloaded_assets: dict[str, Path],
    output_root: Path,
) -> str:
    raw = (value or "").strip()
    if not raw or raw.startswith(("data:", "blob:", "#")):
        return raw

    full = urljoin(current_url, raw)
    parsed = urlparse(full)
    if parsed.scheme not in {"http", "https"}:
        return value

    if full not in downloaded_assets:
        target_path = output_root / url_to_local_asset_path(full)
        target_path.parent.mkdir(parents=True, exist_ok=True)
        try:
            response = fetch(session, full, binary=True)
        except MirrorError:
            return full
        target_path.write_bytes(response.content)
        downloaded_assets[full] = target_path.relative_to(output_root)

    asset_local = downloaded_assets[full]
    return Path(os.path.relpath(asset_local.as_posix(), current_local_path.parent.as_posix())).as_posix()


def rewrite_content(
    content: html.HtmlElement,
    *,
    current_url: str,
    current_local_path: Path,
    site_root: str,
    allowed_prefix: str,
    session: requests.Session,
    downloaded_assets: dict[str, Path],
    output_root: Path,
) -> str:
    strip_unwanted_nodes(content)

    for anchor in content.xpath(".//a[@href]"):
        href = anchor.get("href")
        anchor.set(
            "href",
            rewrite_internal_link(
                href or "",
                current_url=current_url,
                current_local_path=current_local_path,
                site_root=site_root,
                allowed_prefix=allowed_prefix,
            ),
        )

    for tag_name, attr in (("img", "src"), ("source", "src"), ("video", "src")):
        for node in content.xpath(f".//{tag_name}[@{attr}]"):
            node.set(
                attr,
                rewrite_asset_url(
                    node.get(attr) or "",
                    current_url=current_url,
                    current_local_path=current_local_path,
                    session=session,
                    downloaded_assets=downloaded_assets,
                    output_root=output_root,
                ),
            )
        if tag_name == "img":
            for node in content.xpath(".//img[@data-src]"):
                local = rewrite_asset_url(
                    node.get("data-src") or "",
                    current_url=current_url,
                    current_local_path=current_local_path,
                    session=session,
                    downloaded_assets=downloaded_assets,
                    output_root=output_root,
                )
                node.set("src", local)
                node.attrib.pop("data-src", None)

    return etree.tostring(content, encoding="unicode", method="html")


def extract_title(root: html.HtmlElement, content: html.HtmlElement, fallback_url: str) -> str:
    h1_nodes = content.xpath(".//h1[1]")
    if h1_nodes:
        title = clean_text(h1_nodes[0].text_content())
        title = re.sub(r"^\s*#\s*", "", title).strip()
        if title:
            return title
    title = clean_text(root.xpath("string(//title)"))
    if title:
        return re.sub(r"\s*\|\s*微信开放文档\s*$", "", title).strip()
    return fallback_url


def parse_sidebar_node(li: html.HtmlElement) -> SidebarNode:
    link = li.xpath("./a[1] | ./p/a[1] | .//a[1]")
    href = None
    title = ""
    if link:
        href = link[0].get("href")
        title = clean_text(link[0].text_content())
    if not title:
        title = clean_text(li.text_content())
    children = tuple(parse_sidebar_node(child) for child in li.xpath("./ul/li"))
    return SidebarNode(title=title or "(untitled)", href=href, children=children)


def extract_sidebar_tree(root: html.HtmlElement) -> list[SidebarNode]:
    aside_nodes = root.xpath('//aside[contains(concat(" ", normalize-space(@class), " "), " sidebar ")]')
    if not aside_nodes:
        return []
    aside = aside_nodes[0]
    root_items = aside.xpath("./ul/li | .//ul[1]/li")
    nodes = [parse_sidebar_node(li) for li in root_items]
    return nodes


def render_sidebar_tree(
    nodes: list[SidebarNode],
    *,
    current_local_path: Path,
    current_url: str,
    site_root: str,
    allowed_prefix: str,
    depth: int = 0,
) -> list[str]:
    lines: list[str] = []
    indent = "  " * depth
    for node in nodes:
        link_title = escape_markdown_link_text(node.title)
        if node.href:
            target = rewrite_internal_link(
                node.href,
                current_url=current_url,
                current_local_path=current_local_path,
                site_root=site_root,
                allowed_prefix=allowed_prefix,
            )
            lines.append(f"{indent}- [{link_title}]({target})")
        else:
            lines.append(f"{indent}- {link_title}")
        if node.children:
            lines.extend(
                render_sidebar_tree(
                    list(node.children),
                    current_local_path=current_local_path,
                    current_url=current_url,
                    site_root=site_root,
                    allowed_prefix=allowed_prefix,
                    depth=depth + 1,
                )
            )
    return lines


def write_text(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


def write_page(
    *,
    output_root: Path,
    local_path: Path,
    title: str,
    source_url: str,
    content_html: str,
) -> None:
    path = output_root / local_path
    write_text(
        path,
        "\n".join(
            [
                f"# {escape_markdown_heading_text(title)}",
                "",
                f"- 来源：`{source_url}`",
                "",
                "<!-- Weixin Mini Game doc body below keeps official HTML structure for fidelity. -->",
                "",
                content_html.strip(),
                "",
            ]
        ),
    )


def group_key(local_path: Path) -> str:
    parts = local_path.parts
    if not parts:
        return "root"
    if parts[0] == "README.md":
        return "root"
    if len(parts) >= 2:
        return "/".join(parts[:2])
    return parts[0]


def write_index_pages(
    *,
    output_root: Path,
    page_map: dict[str, Path],
    title_map: dict[str, str],
    sidebar_map: dict[str, tuple[str, list[SidebarNode]]],
    asset_count: int,
) -> None:
    entry_lines = [
        "# 微信小游戏文档本地入口说明",
        "",
        "这个目录是对 `https://developers.weixin.qq.com/minigame/` 官方文档站的本地 Markdown 镜像。",
        "",
        f"- 镜像页面数：`{len(page_map)}`",
        f"- 镜像资源数：`{asset_count}`",
        "",
        "## 快速入口",
        "",
    ]
    for title, rel in INDEX_PAGES:
        target = output_root / rel
        if target.exists():
            entry_lines.append(f"- [{title}]({rel})")
    entry_lines.append("")
    entry_lines.append("## 导航页")
    entry_lines.append("")
    entry_lines.append("- [01-完整页面索引.md](01-完整页面索引.md)")
    for filename, (seed_title, _) in sorted(sidebar_map.items()):
        entry_lines.append(f"- [{escape_markdown_link_text(seed_title)} 导航]({filename})")
    entry_lines.append("")
    write_text(output_root / "00-本地入口说明.md", "\n".join(entry_lines))

    grouped: dict[str, list[tuple[str, Path]]] = {}
    for url, path in page_map.items():
        grouped.setdefault(group_key(path), []).append((title_map[url], path))
    index_lines = [
        "# 微信小游戏文档完整页面索引",
        "",
        "以下索引按本地相对路径分组，便于在 Markdown 模式下快速跳转。",
        "",
    ]
    for key in sorted(grouped):
        index_lines.append(f"## {key}")
        index_lines.append("")
        for title, path in sorted(grouped[key], key=lambda item: item[1].as_posix()):
            index_lines.append(f"- [{escape_markdown_link_text(title)}]({path.as_posix()})")
        index_lines.append("")
    write_text(output_root / "01-完整页面索引.md", "\n".join(index_lines).rstrip() + "\n")


def crawl_and_mirror(
    *,
    site_root: str,
    allowed_prefix: str,
    start_urls: list[str],
    output_root: Path,
) -> dict:
    session = make_session()
    output_root.mkdir(parents=True, exist_ok=True)

    queue: deque[str] = deque()
    queued: set[str] = set()
    page_map: dict[str, Path] = {}
    title_map: dict[str, str] = {}
    sidebar_map: dict[str, tuple[str, list[SidebarNode]]] = {}
    downloaded_assets: dict[str, Path] = {}
    crawled_order: list[str] = []
    skipped_pages: dict[str, str] = {}

    for url in start_urls:
        normalized = normalize_doc_url(site_root, allowed_prefix, url, url)
        if normalized and normalized not in queued:
            queue.append(normalized)
            queued.add(normalized)

    while queue:
        current_url = queue.popleft()
        try:
            response = fetch(session, current_url)
        except MirrorError as exc:
            skipped_pages[current_url] = str(exc)
            continue
        if "text/html" not in response.headers.get("content-type", ""):
            continue

        root = html.fromstring(response.text)
        try:
            content = find_content_node(root)
        except MirrorError as exc:
            skipped_pages[current_url] = str(exc)
            continue
        local_path = url_to_local_doc_path(current_url, allowed_prefix)
        title = extract_title(root, content, current_url)
        content_html = rewrite_content(
            content,
            current_url=current_url,
            current_local_path=local_path,
            site_root=site_root,
            allowed_prefix=allowed_prefix,
            session=session,
            downloaded_assets=downloaded_assets,
            output_root=output_root,
        )
        write_page(
            output_root=output_root,
            local_path=local_path,
            title=title,
            source_url=current_url,
            content_html=content_html,
        )

        page_map[current_url] = local_path
        title_map[current_url] = title
        crawled_order.append(current_url)

        if current_url in start_urls:
            sidebar_nodes = extract_sidebar_tree(root)
            if sidebar_nodes:
                local_filename = {
                    "/minigame/introduction/": "10-小游戏指引导航.md",
                    "/minigame/dev/guide/": "11-开发指南导航.md",
                    "/minigame/dev/api/": "12-客户端 API 导航.md",
                    "/minigame/dev/api-backend/": "13-服务端 API 导航.md",
                }.get(urlparse(current_url).path)
                if local_filename:
                    sidebar_path = Path(local_filename)
                    lines = [
                        f"# {title} 导航",
                        "",
                        f"- 来源：`{current_url}`",
                        "",
                    ]
                    lines.extend(
                        render_sidebar_tree(
                            sidebar_nodes,
                            current_local_path=sidebar_path,
                            current_url=current_url,
                            site_root=site_root,
                            allowed_prefix=allowed_prefix,
                        )
                    )
                    write_text(output_root / sidebar_path, "\n".join(lines).rstrip() + "\n")
                    sidebar_map[local_filename] = (title, sidebar_nodes)

        for anchor in root.xpath("//a[@href]"):
            href = anchor.get("href")
            normalized = normalize_doc_url(site_root, allowed_prefix, current_url, href or "")
            if normalized and normalized not in queued and normalized not in page_map:
                queue.append(normalized)
                queued.add(normalized)

    category_counts = Counter()
    for path in page_map.values():
        parts = path.parts
        category_counts[parts[0] if parts else "root"] += 1

    write_index_pages(
        output_root=output_root,
        page_map=page_map,
        title_map=title_map,
        sidebar_map=sidebar_map,
        asset_count=len(downloaded_assets),
    )

    manifest = {
        "site_root": site_root,
        "allowed_prefix": allowed_prefix,
        "generated_at_utc": datetime.now(timezone.utc).isoformat(),
        "page_count": len(page_map),
        "asset_count": len(downloaded_assets),
        "skipped_page_count": len(skipped_pages),
        "output_root": str(output_root),
        "start_urls": start_urls,
        "category_counts": dict(category_counts),
        "crawled_order_sample": crawled_order[:50],
        "skipped_pages": skipped_pages,
    }
    write_text(
        output_root / "_mirror_manifest.json",
        json.dumps(manifest, ensure_ascii=False, indent=2) + "\n",
    )
    return manifest


def parse_args(argv: Iterable[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Mirror Weixin minigame docs to local Markdown.")
    parser.add_argument("--site-root", default=DEFAULT_SITE_ROOT)
    parser.add_argument("--allowed-prefix", default=DEFAULT_ALLOWED_PREFIX)
    parser.add_argument(
        "--start-url",
        action="append",
        dest="start_urls",
        help="Seed page URL. May be passed multiple times.",
    )
    parser.add_argument("--output-root", default=str(DEFAULT_OUTPUT_ROOT))
    return parser.parse_args(list(argv))


def main(argv: Iterable[str]) -> int:
    args = parse_args(argv)
    manifest = crawl_and_mirror(
        site_root=args.site_root,
        allowed_prefix=args.allowed_prefix,
        start_urls=args.start_urls or list(DEFAULT_START_URLS),
        output_root=Path(args.output_root).expanduser(),
    )
    print(json.dumps(manifest, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main(sys.argv[1:]))
    except KeyboardInterrupt:
        raise SystemExit(130)
