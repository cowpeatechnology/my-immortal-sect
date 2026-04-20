#!/usr/bin/env python3
"""Mirror the published Cocos Creator manual pages into a local Markdown tree.

Primary strategy:
1. Read the public manual home page and enumerate published URLs under one prefix.
2. Map each published page URL to its official Markdown source in cocos/cocos-docs.
3. Copy only published Markdown pages plus the assets they reference.
4. If a published page has no matching source file in the repo, fall back to
   extracting the rendered HTML page and saving it as Markdown.

The output is meant for local knowledge-vault use, not for re-hosting.
"""

from __future__ import annotations

import argparse
import html as html_lib
import json
import os
import posixpath
import re
import shutil
import subprocess
import sys
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Iterable
from urllib.parse import urljoin, urlparse

import requests
from lxml import html


DEFAULT_SITE_ROOT = "https://docs.cocos.com/creator/3.8/manual/zh/"
DEFAULT_REPO_URL = "https://github.com/cocos/cocos-docs.git"
DEFAULT_REPO_REF = "master"
DEFAULT_REPO_SUBDIR = "versions/3.8/zh"
DEFAULT_REPO_CACHE = Path("/tmp/cocos-docs-tree")
DEFAULT_OUTPUT_ROOT = Path(
    "/Users/mawei/MyWork/我的知识库/Projects/我的宗门 Wiki/raw/sources/cocos-creator-3.8-manual-zh"
)
REQUEST_TIMEOUT = 30

PUBLISHED_LINK_RE = re.compile(r'href=["\'](/creator/3\.8/manual/zh/[^"\']+)["\']')
MARKDOWN_LINK_RE = re.compile(r"!\[[^\]]*\]\(([^)]+)\)|\[[^\]]*\]\(([^)]+)\)")
HTML_LINK_RE = re.compile(r"""<(?:a|img|source|video)\b[^>]+?(?:href|src)=["']([^"']+)["']""", re.I)


def run(cmd: list[str], cwd: Path | None = None) -> None:
    subprocess.run(cmd, cwd=str(cwd) if cwd else None, check=True)


def fetch_text(url: str) -> str:
    response = requests.get(url, timeout=REQUEST_TIMEOUT)
    response.raise_for_status()
    if not response.encoding or response.encoding.lower() == "iso-8859-1":
        response.encoding = response.apparent_encoding or "utf-8"
    return response.text


def fetch_binary(url: str) -> bytes:
    response = requests.get(url, timeout=REQUEST_TIMEOUT)
    response.raise_for_status()
    return response.content


def ensure_repo_checkout(repo_cache: Path, repo_url: str, repo_ref: str, repo_subdir: str) -> Path:
    if not repo_cache.exists():
        run(
            [
                "git",
                "clone",
                "--depth",
                "1",
                "--filter=blob:none",
                "--sparse",
                "--branch",
                repo_ref,
                repo_url,
                str(repo_cache),
            ]
        )
    else:
        run(["git", "fetch", "origin", repo_ref, "--depth", "1"], cwd=repo_cache)
        run(["git", "checkout", "-f", "FETCH_HEAD"], cwd=repo_cache)

    run(["git", "sparse-checkout", "set", repo_subdir], cwd=repo_cache)
    source_root = repo_cache / repo_subdir
    if not source_root.exists():
        raise FileNotFoundError(f"Repo subdir does not exist after sparse checkout: {source_root}")
    return source_root


def enumerate_published_pages(site_root: str) -> list[str]:
    home_html = fetch_text(site_root)
    published: list[str] = []
    seen: set[str] = set()
    for raw_href in PUBLISHED_LINK_RE.findall(home_html):
        full_url = urljoin(site_root, raw_href)
        if full_url not in seen:
            seen.add(full_url)
            published.append(full_url)
    if site_root not in seen:
        published.insert(0, site_root)
    return published


def dedupe_page_aliases(site_root: str, page_urls: Iterable[str]) -> list[str]:
    unique_urls: list[str] = []
    seen_paths: set[Path] = set()
    for page_url in page_urls:
        relative_md = site_url_to_repo_path(site_root, page_url)
        if relative_md in seen_paths:
            continue
        seen_paths.add(relative_md)
        unique_urls.append(page_url)
    return unique_urls


def site_url_to_repo_path(site_root: str, page_url: str) -> Path:
    if not page_url.startswith(site_root):
        raise ValueError(f"Page URL is outside site root: {page_url}")
    relative = page_url[len(site_root) :]
    if not relative:
        relative = "index.html"
    if relative.endswith("/"):
        relative += "index.html"
    if relative.endswith(".html"):
        relative = relative[:-5] + ".md"
    return Path(relative)


def ensure_parent(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def copy_file(src: Path, dst: Path) -> None:
    ensure_parent(dst)
    shutil.copy2(src, dst)


def normalize_md_target(raw_target: str) -> str:
    target = raw_target.strip()
    if target.startswith("<") and target.endswith(">"):
        target = target[1:-1]
    if " " in target:
        target = target.split(None, 1)[0]
    return target.split("#", 1)[0].split("?", 1)[0].strip()


def collect_repo_assets(markdown_path: Path, source_root: Path) -> set[Path]:
    text = markdown_path.read_text(encoding="utf-8")
    assets: set[Path] = set()
    for pattern in (MARKDOWN_LINK_RE, HTML_LINK_RE):
        for match in pattern.finditer(text):
            target = normalize_md_target(match.group(1) or match.group(2) or "")
            if not target or target.startswith(("http://", "https://", "#", "mailto:", "javascript:")):
                continue
            resolved = (markdown_path.parent / target).resolve()
            try:
                rel_path = resolved.relative_to(source_root.resolve())
            except ValueError:
                continue
            if rel_path.suffix.lower() != ".md" and resolved.exists() and resolved.is_file():
                assets.add(rel_path)
    return assets


def slug_from_page_url(site_root: str, page_url: str) -> str:
    parsed = urlparse(page_url)
    relative = parsed.path.removeprefix(urlparse(site_root).path)
    if relative.endswith("/"):
        relative += "index.html"
    relative = relative or "index.html"
    if relative.endswith(".html"):
        relative = relative[:-5] + ".md"
    return relative


def rewrite_site_link(site_root: str, href: str, current_rel_md: Path) -> str:
    if not href:
        return href
    if href.startswith("#"):
        return href

    absolute = urljoin(site_root, href)
    if absolute.startswith(site_root):
        target_rel = site_url_to_repo_path(site_root, absolute)
        return posixpath.relpath(target_rel.as_posix(), current_rel_md.parent.as_posix())
    return href


def extract_main_content(doc: html.HtmlElement) -> html.HtmlElement:
    candidates = doc.xpath('//*[contains(@class,"content-container")]')
    if candidates:
        return candidates[0]
    mains = doc.xpath("//main")
    if mains:
        return mains[0]
    raise RuntimeError("Unable to find main content container in HTML fallback page.")


def inline_text(node: html.HtmlElement, site_root: str, current_rel_md: Path) -> str:
    pieces: list[str] = []
    if node.text:
        pieces.append(node.text)

    for child in node:
        tag = child.tag.lower() if isinstance(child.tag, str) else ""
        if tag == "br":
            pieces.append("  \n")
        elif tag in {"strong", "b"}:
            pieces.append(f"**{inline_text(child, site_root, current_rel_md).strip()}**")
        elif tag in {"em", "i"}:
            pieces.append(f"*{inline_text(child, site_root, current_rel_md).strip()}*")
        elif tag == "code":
            pieces.append(f"`{inline_text(child, site_root, current_rel_md).strip()}`")
        elif tag == "a":
            text = inline_text(child, site_root, current_rel_md).strip() or child.get("href", "")
            href = rewrite_site_link(site_root, child.get("href", ""), current_rel_md)
            pieces.append(f"[{text}]({href})")
        elif tag == "img":
            alt = (child.get("alt") or "").strip()
            src = child.get("src", "")
            rewritten = rewrite_fallback_asset_url(src)
            pieces.append(f"![{alt}]({rewritten})")
        else:
            pieces.append(inline_text(child, site_root, current_rel_md))

        if child.tail:
            pieces.append(child.tail)

    return "".join(pieces)


def block_to_markdown(node: html.HtmlElement, site_root: str, current_rel_md: Path, depth: int = 0) -> str:
    tag = node.tag.lower() if isinstance(node.tag, str) else ""
    if tag in {"script", "style", "button", "nav"}:
        return ""

    if tag in {"h1", "h2", "h3", "h4", "h5", "h6"}:
        level = int(tag[1])
        return f"{'#' * level} {inline_text(node, site_root, current_rel_md).strip()}\n\n"

    if tag == "p":
        text = inline_text(node, site_root, current_rel_md).strip()
        return f"{text}\n\n" if text else ""

    if tag == "pre":
        code_text = "".join(node.itertext()).strip("\n")
        return f"```\n{code_text}\n```\n\n"

    if tag == "blockquote":
        body = "".join(block_to_markdown(child, site_root, current_rel_md, depth) for child in node)
        lines = [line for line in body.strip().splitlines() if line.strip()]
        return "".join(f"> {line}\n" for line in lines) + "\n"

    if tag in {"ul", "ol"}:
        lines: list[str] = []
        for index, child in enumerate(node, start=1):
            if getattr(child, "tag", "").lower() != "li":
                continue
            prefix = f"{index}. " if tag == "ol" else "- "
            text = inline_text(child, site_root, current_rel_md).strip()
            if text:
                lines.append(("  " * depth) + prefix + text)
            for grandchild in child:
                grand_tag = grandchild.tag.lower() if isinstance(grandchild.tag, str) else ""
                if grand_tag in {"ul", "ol"}:
                    nested = block_to_markdown(grandchild, site_root, current_rel_md, depth + 1).rstrip()
                    if nested:
                        lines.append(nested)
        return "\n".join(lines) + "\n\n" if lines else ""

    if tag == "img":
        alt = (node.get("alt") or "").strip()
        src = rewrite_fallback_asset_url(node.get("src", ""))
        return f"![{alt}]({src})\n\n"

    if tag in {"table"}:
        # Preserve table text in a simple readable form instead of dropping it.
        rows = node.xpath(".//tr")
        rendered_rows: list[str] = []
        for row in rows:
            cols = [inline_text(col, site_root, current_rel_md).strip() for col in row.xpath("./th|./td")]
            if cols:
                rendered_rows.append(" | ".join(cols))
        if not rendered_rows:
            return ""
        if len(rendered_rows) == 1:
            return rendered_rows[0] + "\n\n"
        header = rendered_rows[0]
        separator = " | ".join(["---"] * len(header.split(" | ")))
        body = "\n".join(rendered_rows[1:])
        return f"{header}\n{separator}\n{body}\n\n"

    if tag == "div":
        # Ignore pure anchor wrappers but keep text nodes.
        parts: list[str] = []
        text = (node.text or "").strip()
        if text:
            parts.append(text + "\n\n")
        for child in node:
            parts.append(block_to_markdown(child, site_root, current_rel_md, depth))
            if child.tail and child.tail.strip():
                parts.append(child.tail.strip() + "\n\n")
        return "".join(parts)

    if tag == "a":
        text = inline_text(node, site_root, current_rel_md).strip() or node.get("href", "")
        href = rewrite_site_link(site_root, node.get("href", ""), current_rel_md)
        return f"[{text}]({href})\n\n"

    parts: list[str] = []
    text = (node.text or "").strip()
    if text:
        parts.append(text + "\n\n")
    for child in node:
        parts.append(block_to_markdown(child, site_root, current_rel_md, depth))
        if child.tail and child.tail.strip():
            parts.append(child.tail.strip() + "\n\n")
    return "".join(parts)


def cleanup_markdown(markdown: str) -> str:
    markdown = html_lib.unescape(markdown)
    markdown = markdown.replace("\r\n", "\n")
    markdown = re.sub(r"\n{3,}", "\n\n", markdown)
    return markdown.strip() + "\n"


def load_sidebar_summary(source_root: Path) -> list[dict]:
    summary_path = source_root / "summary.json"
    return json.loads(summary_path.read_text(encoding="utf-8"))


def render_summary_items(items: list[dict], depth: int = 0) -> list[str]:
    lines: list[str] = []
    indent = "  " * depth
    for item in items:
        text = item.get("text", "").strip() or "(untitled)"
        link = item.get("link")
        if isinstance(link, str) and link:
            lines.append(f"{indent}- [{text}]({link})")
        else:
            lines.append(f"{indent}- {text}")
        child_items = item.get("items")
        if isinstance(child_items, list) and child_items:
            lines.extend(render_summary_items(child_items, depth + 1))
    return lines


def write_navigation_artifacts(
    output_root: Path,
    source_root: Path,
    site_root: str,
    *,
    page_count: int,
    asset_count: int,
    html_fallback_pages: list[str],
) -> None:
    summary = load_sidebar_summary(source_root)

    summary_copy = output_root / "_sidebar_summary.json"
    summary_copy.write_text(json.dumps(summary, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    lines: list[str] = [
        "# Cocos Creator 3.8 中文手册本地入口说明",
        "",
        "这个目录是对官方中文手册 `https://docs.cocos.com/creator/3.8/manual/zh/` 的本地镜像，范围只包含该手册树内真实公开的页面。",
        "",
        "## 为什么本地看起来和线上不一样",
        "",
        "- 线上左侧导航不是每个正文页面自己带出来的，而是由 VitePress 根据官方 `summary.json` 动态渲染的。",
        "- 本地镜像里的 `504` 个正文页面本身是对的，但如果只打开单个 `.md`，你看到的是“正文内容”，看不到线上那层站点外壳。",
        "- 为了补这个差异，本目录额外生成了这份入口说明页和一份原始导航数据 `[_sidebar_summary.json](_sidebar_summary.json)`。",
        "",
        "## 本地镜像范围",
        "",
        f"- 公开页面数：`{page_count}`",
        f"- 本地资源数：`{asset_count}`",
        "- 官方手册首页：[index.md](index.md)",
        "- 镜像清单：[_mirror_manifest.json](_mirror_manifest.json)",
    ]

    if html_fallback_pages:
        lines.extend(
            [
                "",
                "## 特殊回退页",
                "",
                "以下页面在线上仍可访问，但官方源码仓库当前版本已不存在原始 Markdown，因此本地镜像使用了 HTML 回退提取：",
                "",
            ]
        )
        for page_url in html_fallback_pages:
            rel = site_url_to_repo_path(site_root, page_url).as_posix()
            lines.append(f"- [{rel}]({rel})")

    lines.extend(
        [
            "",
            "## 官方导航镜像",
            "",
            "下面的结构按官方线上左侧导航生成，方便在本地直接跳转：",
            "",
        ]
    )

    for group in summary:
        title = group.get("text", "").strip() or "(untitled)"
        lines.append(f"## {title}")
        lines.append("")
        items = group.get("items")
        if isinstance(items, list) and items:
            lines.extend(render_summary_items(items))
        else:
            lines.append("- （无条目）")
        lines.append("")

    navigation_path = output_root / "00-本地入口说明.md"
    navigation_path.write_text("\n".join(lines).rstrip() + "\n", encoding="utf-8")


def rewrite_fallback_asset_url(src: str) -> str:
    if src.startswith("/creator/3.8/manual/assets/"):
        return "./" + posixpath.basename(src)
    return src


def extract_fallback_assets(doc: html.HtmlElement, site_root: str) -> dict[str, bytes]:
    assets: dict[str, bytes] = {}
    for node in doc.xpath('//img[@src]'):
        src = node.get("src", "")
        if not src.startswith("/creator/3.8/manual/assets/"):
            continue
        name = posixpath.basename(src)
        if name not in assets:
            assets[name] = fetch_binary(urljoin(site_root, src))
    return assets


def convert_html_page_to_markdown(page_url: str, site_root: str) -> tuple[str, dict[str, bytes]]:
    page_html = fetch_text(page_url)
    doc = html.fromstring(page_html)
    content = extract_main_content(doc)
    parts: list[str] = []
    for child in content:
        parts.append(block_to_markdown(child, site_root, site_url_to_repo_path(site_root, page_url)))
    markdown = cleanup_markdown("".join(parts))
    assets = extract_fallback_assets(content, site_root)
    return markdown, assets


@dataclass
class MirrorResult:
    output_root: Path
    page_count: int
    copied_markdown_files: int
    copied_asset_files: int
    html_fallback_pages: list[str]
    manifest_path: Path


def mirror_manual(
    site_root: str,
    source_root: Path,
    output_root: Path,
) -> MirrorResult:
    published_urls = dedupe_page_aliases(site_root, enumerate_published_pages(site_root))
    output_root.mkdir(parents=True, exist_ok=True)

    copied_assets: set[Path] = set()
    fallback_pages: list[str] = []
    copied_markdown_files = 0

    for page_url in published_urls:
        relative_md = site_url_to_repo_path(site_root, page_url)
        repo_md = source_root / relative_md
        output_md = output_root / relative_md
        if repo_md.exists():
            copy_file(repo_md, output_md)
            copied_markdown_files += 1
            for asset_rel in collect_repo_assets(repo_md, source_root):
                if asset_rel in copied_assets:
                    continue
                src_asset = source_root / asset_rel
                dst_asset = output_root / asset_rel
                copy_file(src_asset, dst_asset)
                copied_assets.add(asset_rel)
        else:
            markdown, assets = convert_html_page_to_markdown(page_url, site_root)
            ensure_parent(output_md)
            output_md.write_text(markdown, encoding="utf-8")
            copied_markdown_files += 1
            fallback_pages.append(page_url)
            for name, payload in assets.items():
                asset_path = output_md.parent / name
                ensure_parent(asset_path)
                asset_path.write_bytes(payload)
                copied_assets.add(asset_path.relative_to(output_root))

    manifest_path = output_root / "_mirror_manifest.json"
    manifest = {
        "site_root": site_root,
        "source_root": str(source_root),
        "generated_at_utc": datetime.now(timezone.utc).isoformat(),
        "published_page_count": len(published_urls),
        "copied_markdown_files": copied_markdown_files,
        "copied_asset_files": len(copied_assets),
        "html_fallback_pages": fallback_pages,
    }
    manifest_path.write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    write_navigation_artifacts(
        output_root,
        source_root,
        site_root,
        page_count=len(published_urls),
        asset_count=len(copied_assets),
        html_fallback_pages=fallback_pages,
    )

    return MirrorResult(
        output_root=output_root,
        page_count=len(published_urls),
        copied_markdown_files=copied_markdown_files,
        copied_asset_files=len(copied_assets),
        html_fallback_pages=fallback_pages,
        manifest_path=manifest_path,
    )


def parse_args(argv: Iterable[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Mirror the published Cocos Creator manual into local Markdown.")
    parser.add_argument("--site-root", default=DEFAULT_SITE_ROOT, help=f'Site root to mirror. Default: "{DEFAULT_SITE_ROOT}"')
    parser.add_argument("--repo-url", default=DEFAULT_REPO_URL, help=f'Git repository URL. Default: "{DEFAULT_REPO_URL}"')
    parser.add_argument("--repo-ref", default=DEFAULT_REPO_REF, help=f'Git ref to checkout. Default: "{DEFAULT_REPO_REF}"')
    parser.add_argument("--repo-subdir", default=DEFAULT_REPO_SUBDIR, help=f'Repo subdir with manual source. Default: "{DEFAULT_REPO_SUBDIR}"')
    parser.add_argument("--repo-cache", default=str(DEFAULT_REPO_CACHE), help=f'Local git cache directory. Default: "{DEFAULT_REPO_CACHE}"')
    parser.add_argument("--output-root", default=str(DEFAULT_OUTPUT_ROOT), help=f'Output directory. Default: "{DEFAULT_OUTPUT_ROOT}"')
    return parser.parse_args(list(argv))


def main(argv: Iterable[str]) -> int:
    args = parse_args(argv)
    repo_cache = Path(args.repo_cache).expanduser()
    output_root = Path(args.output_root).expanduser()

    source_root = ensure_repo_checkout(repo_cache, args.repo_url, args.repo_ref, args.repo_subdir)
    result = mirror_manual(args.site_root, source_root, output_root)
    print(json.dumps(
        {
            "output_root": str(result.output_root),
            "page_count": result.page_count,
            "copied_markdown_files": result.copied_markdown_files,
            "copied_asset_files": result.copied_asset_files,
            "html_fallback_pages": result.html_fallback_pages,
            "manifest_path": str(result.manifest_path),
        },
        ensure_ascii=False,
        indent=2,
    ))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main(sys.argv[1:]))
    except KeyboardInterrupt:
        raise SystemExit(130)
