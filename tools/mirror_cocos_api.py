#!/usr/bin/env python3
"""Mirror Cocos Creator 3.8 Chinese API docs into local Markdown files."""

from __future__ import annotations

import argparse
import json
import os
import posixpath
import re
import sys
from collections import Counter
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Iterable
from urllib.parse import urljoin

import requests
from lxml import html


DEFAULT_API_ROOT = "https://docs.cocos.com/creator/3.8/api/zh/"
DEFAULT_OUTPUT_ROOT = Path(
    "/Users/mawei/MyWork/我的知识库/Projects/我的宗门 Wiki/raw/sources/cocos-creator-3.8-api-zh"
)
REQUEST_TIMEOUT = 30
ASSET_EXTENSIONS = {
    ".png",
    ".jpg",
    ".jpeg",
    ".gif",
    ".svg",
    ".webp",
    ".bmp",
    ".css",
    ".js",
    ".ico",
    ".json",
}

API_PREFIXES = (
    "/creator/3.8/api/zh/",
    "creator/3.8/api/zh/",
)
MARKDOWN_LINK_RE = re.compile(r"(!?\[[^\]]*\]\()([^)]+)(\))")
HTML_ATTR_RE = re.compile(r'((?:href|src)=["\'])([^"\']+)(["\'])', re.I)


def fetch_text(url: str) -> str:
    response = requests.get(url, timeout=REQUEST_TIMEOUT)
    response.raise_for_status()
    if not response.encoding or response.encoding.lower() == "iso-8859-1":
        response.encoding = response.apparent_encoding or "utf-8"
    return response.text


def ensure_parent(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def normalize_target_text(target: str) -> str:
    clean = target.strip()
    if clean.startswith("<") and clean.endswith(">"):
        clean = clean[1:-1]
    if " " in clean:
        clean = clean.split(None, 1)[0]
    return clean.split("#", 1)[0].split("?", 1)[0].strip()


def site_link_to_local_path(target: str) -> Path:
    clean = normalize_target_text(target)
    for prefix in API_PREFIXES:
        if clean.startswith(prefix):
            remainder = clean[len(prefix) :]
            if not remainder:
                return Path("README.md")
            return Path(remainder + ".md")
    if clean in {"", "/"}:
        return Path("README.md")
    if clean.endswith(".md"):
        return Path(clean)
    return Path(clean + ".md")


def site_link_to_remote_md_url(api_root: str, target: str) -> str:
    clean = normalize_target_text(target)
    for prefix in API_PREFIXES:
        if clean.startswith(prefix):
            remainder = clean[len(prefix) :]
            if not remainder:
                return urljoin(api_root, "README.md")
            return urljoin(api_root, remainder + ".md")
    raise ValueError(f"Unsupported API link target: {target}")


def resolve_internal_page_path(raw_target: str, current_page_path: Path) -> Path | None:
    target = normalize_target_text(raw_target)
    if not target or target.startswith(("#", "mailto:", "javascript:")):
        return None

    if target.startswith(("http://", "https://")):
        if target.startswith(DEFAULT_API_ROOT):
            target = target[len(DEFAULT_API_ROOT) :]
        else:
            return None
    elif any(target.startswith(prefix) for prefix in API_PREFIXES):
        for prefix in API_PREFIXES:
            if target.startswith(prefix):
                target = target[len(prefix) :]
                break
    elif target.startswith("/"):
        return None
    else:
        target = posixpath.normpath(posixpath.join(current_page_path.parent.as_posix(), target))

    if target in {"", "."}:
        return Path("README.md")

    if target.endswith("/"):
        target = target.rstrip("/") + "/README.md"

    suffix = Path(target).suffix.lower()
    if suffix in ASSET_EXTENSIONS:
        return None
    if suffix != ".md":
        target += ".md"
    return Path(target)


def rewrite_internal_target(raw_target: str, current_local_path: Path) -> str:
    target = raw_target.strip()
    if not target or target.startswith(("#", "mailto:", "javascript:")):
        return raw_target

    anchor = ""
    if "?id=" in target:
        target, anchor = target.split("?id=", 1)
        anchor = "#" + anchor
    elif "#" in target:
        target, hash_value = target.split("#", 1)
        anchor = "#" + hash_value

    local_target = resolve_internal_page_path(target, current_local_path)
    if local_target is not None:
        rel_text = Path(os.path.relpath(local_target.as_posix(), current_local_path.parent.as_posix() or ".")).as_posix()
        return rel_text + anchor

    return raw_target


def rewrite_markdown_content(markdown_text: str, current_local_path: Path) -> str:
    def replace_md(match: re.Match[str]) -> str:
        prefix, target, suffix = match.groups()
        return prefix + rewrite_internal_target(target, current_local_path) + suffix

    text = MARKDOWN_LINK_RE.sub(replace_md, markdown_text)

    def replace_html_attr(match: re.Match[str]) -> str:
        prefix, target, suffix = match.groups()
        return prefix + rewrite_internal_target(target, current_local_path) + suffix

    return HTML_ATTR_RE.sub(replace_html_attr, text)


@dataclass
class SidebarNode:
    title: str
    href: str | None
    children: list["SidebarNode"]

    @property
    def is_group(self) -> bool:
        return self.href is None


def parse_sidebar_node(li: html.HtmlElement) -> SidebarNode:
    link_nodes = li.xpath("./p/a[1]")
    if link_nodes:
        title = " ".join(link_nodes[0].text_content().split())
        href = link_nodes[0].get("href")
    else:
        title = " ".join("".join(li.xpath("./p[1]//text()")).split())
        href = None

    children = [parse_sidebar_node(child) for child in li.xpath("./ul/li")]
    return SidebarNode(title=title or "(untitled)", href=href, children=children)


def fetch_sidebar_tree(api_root: str) -> tuple[list[SidebarNode], str]:
    sidebar_html = fetch_text(urljoin(api_root, "_sidebar.md"))
    doc = html.fromstring(sidebar_html)
    if doc.tag.lower() == "ul":
        root_list_items = doc.xpath("./li")
    else:
        root_list_items = doc.xpath("./ul[1]/li")
    nodes = [parse_sidebar_node(li) for li in root_list_items]
    return nodes, sidebar_html


def collect_sidebar_links(nodes: list[SidebarNode]) -> list[str]:
    links: list[str] = []

    def walk(node: SidebarNode) -> None:
        if node.href and any(node.href.startswith(prefix) for prefix in API_PREFIXES):
            links.append(node.href)
        for child in node.children:
            walk(child)

    for node in nodes:
        walk(node)

    # Deduplicate while preserving order.
    seen: set[str] = set()
    unique: list[str] = []
    for link in links:
        if link not in seen:
            seen.add(link)
            unique.append(link)
    return unique


def collect_internal_page_paths(markdown_text: str, current_page_path: Path) -> list[Path]:
    found: list[Path] = []
    for pattern in (MARKDOWN_LINK_RE, HTML_ATTR_RE):
        for match in pattern.finditer(markdown_text):
            if pattern is MARKDOWN_LINK_RE:
                target = match.group(2)
            else:
                target = match.group(2)
            page_path = resolve_internal_page_path(target, current_page_path)
            if page_path is not None:
                found.append(page_path)
    seen: set[Path] = set()
    unique: list[Path] = []
    for path in found:
        if path not in seen:
            seen.add(path)
            unique.append(path)
    return unique


def render_sidebar_tree(nodes: list[SidebarNode], current_local_path: Path, depth: int = 0) -> list[str]:
    lines: list[str] = []
    indent = "  " * depth
    for node in nodes:
        if node.href:
            local_target = rewrite_internal_target(node.href, current_local_path)
            lines.append(f"{indent}- [{node.title}]({local_target})")
        else:
            lines.append(f"{indent}- {node.title}")
        if node.children:
            lines.extend(render_sidebar_tree(node.children, current_local_path, depth + 1))
    return lines


def write_navigation_pages(
    output_root: Path,
    nodes: list[SidebarNode],
    *,
    page_count: int,
    category_counts: Counter[str],
) -> None:
    entry_path = output_root / "00-本地入口说明.md"
    lines = [
        "# Cocos Creator 3.8 中文 API 本地入口说明",
        "",
        "这个目录是对官方 `https://docs.cocos.com/creator/3.8/api/zh/` 的本地 Markdown 镜像。",
        "",
        "## 本地范围",
        "",
        f"- 镜像页面数：`{page_count}`",
        f"- 类：`{category_counts.get('class', 0)}`",
        f"- 函数：`{category_counts.get('function', 0)}`",
        f"- 变量：`{category_counts.get('variable', 0)}`",
        f"- 枚举：`{category_counts.get('enumeration', 0)}`",
        f"- 接口：`{category_counts.get('interface', 0)}`",
        f"- 命名空间：`{category_counts.get('namespace', 0)}`",
        "",
        "## 阅读入口",
        "",
        "- [README.md](README.md)",
        "- [01-完整导航索引.md](01-完整导航索引.md)",
        "",
        "## 顶层模块",
        "",
    ]

    entry_rel_path = Path("00-本地入口说明.md")
    for node in nodes:
        if node.href:
            target = rewrite_internal_target(node.href, entry_rel_path)
            lines.append(f"- [{node.title}]({target})")
        else:
            lines.append(f"- {node.title}")

    entry_path.write_text("\n".join(lines).rstrip() + "\n", encoding="utf-8")

    full_index_path = output_root / "01-完整导航索引.md"
    full_index_rel_path = Path("01-完整导航索引.md")
    full_lines = [
        "# Cocos Creator 3.8 中文 API 完整导航索引",
        "",
        "以下索引根据官方 `_sidebar.md` 生成，保留了官方导航层级。",
        "",
    ]
    full_lines.extend(render_sidebar_tree(nodes, full_index_rel_path))
    full_index_path.write_text("\n".join(full_lines).rstrip() + "\n", encoding="utf-8")


def mirror_api(api_root: str, output_root: Path) -> dict:
    output_root.mkdir(parents=True, exist_ok=True)

    nodes, sidebar_html = fetch_sidebar_tree(api_root)
    page_links = collect_sidebar_links(nodes)

    # Root README is not always included as a sidebar href in non-expanded cases; ensure it exists.
    if "/creator/3.8/api/zh/" not in page_links:
        page_links.insert(0, "/creator/3.8/api/zh/")

    category_counts: Counter[str] = Counter()
    for link in page_links:
        suffix = link.split("/creator/3.8/api/zh/", 1)[1]
        category = suffix.split("/", 1)[0] if suffix else "root"
        category_counts[category] += 1

    # Save raw sidebar HTML for debugging/reference.
    (output_root / "_sidebar_source.html").write_text(sidebar_html, encoding="utf-8")

    initial_paths = [site_link_to_local_path(link) for link in page_links]
    queue: list[Path] = []
    seen_paths: set[Path] = set()
    for path in initial_paths:
        if path not in seen_paths:
            seen_paths.add(path)
            queue.append(path)

    fetched_pages: dict[Path, str] = {}
    while queue:
        current_path = queue.pop(0)
        remote_url = urljoin(api_root, current_path.as_posix())
        markdown_text = fetch_text(remote_url)
        fetched_pages[current_path] = markdown_text
        for linked_path in collect_internal_page_paths(markdown_text, current_path):
            if linked_path not in seen_paths:
                seen_paths.add(linked_path)
                queue.append(linked_path)

    for local_path, markdown_text in fetched_pages.items():
        rewritten = rewrite_markdown_content(markdown_text, local_path)
        target_path = output_root / local_path
        ensure_parent(target_path)
        target_path.write_text(rewritten, encoding="utf-8")

    write_navigation_pages(
        output_root,
        nodes,
        page_count=len(fetched_pages),
        category_counts=category_counts,
    )

    manifest = {
        "api_root": api_root,
        "generated_at_utc": datetime.now(timezone.utc).isoformat(),
        "page_count": len(fetched_pages),
        "sidebar_page_count": len(page_links),
        "output_root": str(output_root),
        "category_counts": dict(category_counts),
    }
    (output_root / "_mirror_manifest.json").write_text(
        json.dumps(manifest, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    return manifest


def parse_args(argv: Iterable[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Mirror Cocos Creator 3.8 Chinese API Markdown locally.")
    parser.add_argument("--api-root", default=DEFAULT_API_ROOT, help=f'API root URL. Default: "{DEFAULT_API_ROOT}"')
    parser.add_argument("--output-root", default=str(DEFAULT_OUTPUT_ROOT), help=f'Output directory. Default: "{DEFAULT_OUTPUT_ROOT}"')
    return parser.parse_args(list(argv))


def main(argv: Iterable[str]) -> int:
    args = parse_args(argv)
    manifest = mirror_api(args.api_root, Path(args.output_root).expanduser())
    print(json.dumps(manifest, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main(sys.argv[1:]))
    except KeyboardInterrupt:
        raise SystemExit(130)
