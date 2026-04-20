#!/usr/bin/env python3
"""Mirror selected Douyin mini-game docs into local Markdown files.

Scope:
- quick start: /docs/resource/zh-CN/mini-game/guide/
- development: /docs/resource/zh-CN/mini-game/develop/

Excluded:
- operation/community sections
- guide/minigame/operationalguidelines landing
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
import time
from collections import Counter, deque
from datetime import datetime, timezone
from pathlib import Path
from typing import Iterable
from urllib.parse import urljoin, urlparse

import requests
from lxml import etree, html


DEFAULT_SITE_ROOT = "https://developer.open-douyin.com"
DEFAULT_LOCAL_BASE_PREFIX = "/docs/resource/zh-CN/mini-game/"
DEFAULT_ALLOWED_PREFIXES = (
    "/docs/resource/zh-CN/mini-game/guide/",
    "/docs/resource/zh-CN/mini-game/develop/",
)
DEFAULT_EXCLUDED_PATHS = {
    "/docs/resource/zh-CN/mini-game/guide/minigame/operationalguidelines",
}
DEFAULT_START_URLS = [
    "https://developer.open-douyin.com/docs/resource/zh-CN/mini-game/guide/overview",
    "https://developer.open-douyin.com/docs/resource/zh-CN/mini-game/guide/minigame/introduction",
    "https://developer.open-douyin.com/docs/resource/zh-CN/mini-game/develop/guide/dev-guide/bytedance-mini-game",
]
DEFAULT_OUTPUT_ROOT = Path(
    "/Users/mawei/MyWork/我的知识库/Projects/我的宗门 Wiki/raw/sources/douyin-minigame-guide-develop"
)
REQUEST_TIMEOUT = 30
USER_AGENT = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
    "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)
DOC_EXTENSIONS = {"", ".html", ".htm"}
STRIP_XPATHS = (
    ".//script",
    ".//style",
    './/*[contains(@class, "copy-TYWbk1")]',
    './/*[contains(@class, "feelgood-wrapper")]',
    './/*[@id="non-content"]',
    './/*[@id="docs-heading"]',
)


class MirrorError(RuntimeError):
    """Raised when the mirror script cannot continue."""


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
        except Exception as exc:  # pragma: no cover
            last_error = exc
            if attempt == retries:
                break
            time.sleep(0.5 * attempt)
    raise MirrorError(f"Failed to fetch {url}: {last_error}")


def canonicalize_path(path: str) -> str:
    clean = re.sub(r"/+", "/", path or "/")
    if not clean.startswith("/"):
        clean = "/" + clean
    if clean.endswith("/index.html"):
        clean = clean[: -len("index.html")]
    elif clean.endswith("/index"):
        clean = clean[: -len("index")]
    if clean != "/" and clean.endswith("/"):
        clean = clean.rstrip("/")
    return clean


def is_allowed_path(path: str, allowed_prefixes: tuple[str, ...], excluded_paths: set[str]) -> bool:
    if path in excluded_paths:
        return False
    return any(path.startswith(prefix) for prefix in allowed_prefixes)


def normalize_doc_url(
    site_root: str,
    allowed_prefixes: tuple[str, ...],
    excluded_paths: set[str],
    current_url: str,
    href: str,
) -> str | None:
    raw = (href or "").strip()
    if not raw or raw.startswith(("#", "mailto:", "javascript:", "tel:")):
        return None
    if any(ch.isspace() for ch in raw):
        return None
    if raw.startswith(("- ", "•", "* ")):
        return None

    full = urljoin(current_url, raw)
    parsed = urlparse(full)
    site = urlparse(site_root)
    if parsed.scheme not in {"http", "https"}:
        return None
    if parsed.netloc != site.netloc:
        return None

    path = canonicalize_path(parsed.path)
    suffix = Path(path).suffix.lower()
    if suffix and suffix not in DOC_EXTENSIONS:
        return None
    if not is_allowed_path(path, allowed_prefixes, excluded_paths):
        return None
    return f"{site.scheme}://{site.netloc}{path}"


def url_to_local_doc_path(url: str, local_base_prefix: str) -> Path:
    parsed = urlparse(url)
    path = canonicalize_path(parsed.path)
    relative = path[len(local_base_prefix) :].lstrip("/")
    if not relative:
        return Path("README.md")
    suffix = Path(relative).suffix.lower()
    if suffix in {".html", ".htm"}:
        return Path(relative).with_suffix(".md")
    if not suffix:
        return Path(relative + ".md")
    return Path(relative)


def url_to_local_asset_path(url: str) -> Path:
    parsed = urlparse(url)
    path = canonicalize_path(parsed.path)
    relative = path.lstrip("/") or "index"
    path_obj = Path(relative)
    if parsed.query:
        digest = hashlib.sha1(parsed.query.encode("utf-8")).hexdigest()[:10]
        stem = path_obj.stem or "asset"
        suffix = path_obj.suffix
        filename = f"{stem}__q_{digest}{suffix}"
        path_obj = path_obj.with_name(filename)
    return Path("_assets") / parsed.netloc / path_obj


def relative_path(from_path: Path, to_path: Path) -> str:
    return Path(os.path.relpath(to_path, from_path.parent)).as_posix()


def clean_text(text: str) -> str:
    return " ".join(text.split())


def clean_title_candidate(text: str) -> str:
    value = clean_text(text or "")
    if re.match(r"^https?://", value):
        return ""
    value = re.sub(r"\s*[_\-|]\s*抖音开放平台$", "", value).strip()
    value = re.sub(r"(?:我的收藏|收藏)+$", "", value).strip()
    value = re.sub(r"^(?:上一篇：|下一篇：)\s*", "", value).strip()
    if value == "Untitled":
        return ""
    return value


def is_generic_title(text: str) -> bool:
    value = clean_title_candidate(text)
    return value in {"", "抖音开放平台", "开发", "API", "快速入门"}


def escape_markdown_heading_text(text: str) -> str:
    return clean_title_candidate(text).replace("<", "&lt;").replace(">", "&gt;").strip()


def escape_markdown_link_text(text: str) -> str:
    escaped = escape_markdown_heading_text(text)
    escaped = escaped.replace("\\", "\\\\")
    escaped = escaped.replace("[", "\\[").replace("]", "\\]")
    return escaped


def score_title_hint(title: str, count: int) -> tuple[int, int, int, int]:
    generic_penalty = 0 if is_generic_title(title) else 1
    specificity_bonus = 1 if any(ch in title for ch in ".()/:") else 0
    return (generic_penalty, specificity_bonus, count, len(title))


def extract_content_heading(content: html.HtmlElement) -> str:
    headings = content.xpath('.//*[self::h1 or self::h2][1]')
    if not headings:
        return ""
    return clean_title_candidate(headings[0].text_content())


def choose_best_title(
    page_title: str,
    content_heading: str,
    hint_counter: Counter[str] | None,
    local_path: Path,
) -> str:
    cleaned_page_title = clean_title_candidate(page_title)
    if cleaned_page_title and not is_generic_title(cleaned_page_title):
        return cleaned_page_title
    cleaned_content_heading = clean_title_candidate(content_heading)
    if cleaned_content_heading and not is_generic_title(cleaned_content_heading):
        return cleaned_content_heading

    best_hint = ""
    if hint_counter:
        candidates = {
            clean_title_candidate(title): count
            for title, count in hint_counter.items()
            if clean_title_candidate(title)
        }
        if candidates:
            best_hint = max(
                candidates.items(),
                key=lambda item: score_title_hint(item[0], item[1]),
            )[0]
    if best_hint and not is_generic_title(best_hint):
        return best_hint
    if cleaned_page_title:
        return cleaned_page_title

    stem = local_path.stem
    if stem.startswith("tt-"):
        parts = stem.split("-")
        if len(parts) >= 2:
            return parts[0] + "." + parts[1] + "".join(part.capitalize() for part in parts[2:])
    return stem.replace("-", " ") or "Untitled"


def get_title(root: html.HtmlElement) -> str:
    heading = root.xpath('//*[@id="docs-heading"]//h1[1]')
    if heading:
        title = clean_title_candidate(heading[0].text_content())
        if title:
            return title
    title = clean_title_candidate(root.xpath("string(//title)"))
    return title or "Untitled"


def get_sidebar_links(root: html.HtmlElement) -> list[tuple[str, str]]:
    links: list[tuple[str, str]] = []
    seen: set[str] = set()
    for anchor in root.xpath('//div[contains(@class, "aside-menu")]//a[@href]'):
        href = (anchor.get("href") or "").strip()
        text = clean_title_candidate(anchor.text_content())
        if not href or not text:
            continue
        if href in seen:
            continue
        seen.add(href)
        links.append((href, text))
    return links


def get_page_links(root: html.HtmlElement) -> list[tuple[str, str]]:
    links: list[tuple[str, str]] = []
    seen: set[str] = set()
    for anchor in root.xpath("//a[@href]"):
        href = (anchor.get("href") or "").strip()
        text = clean_title_candidate(anchor.text_content())
        if not href or not text:
            continue
        if href in seen:
            continue
        seen.add(href)
        links.append((href, text))
    return links


def find_content_node(root: html.HtmlElement) -> html.HtmlElement:
    nodes = root.xpath('//*[@id="open-doc"]//div[contains(@class, "doc-renderer-container")]')
    if nodes:
        return nodes[0]
    nodes = root.xpath('//*[@id="open-doc"]')
    if nodes:
        return nodes[0]
    raise MirrorError("Could not locate Douyin content node")


def strip_unwanted_nodes(content: html.HtmlElement) -> None:
    for xpath in STRIP_XPATHS:
        for node in content.xpath(xpath):
            parent = node.getparent()
            if parent is not None:
                parent.remove(node)

    for node in list(content.iter()):
        if not isinstance(node.tag, str):
            continue
        cls = node.get("class", "")
        if "copy-" in cls or "feelgood" in cls:
            parent = node.getparent()
            if parent is not None:
                parent.remove(node)

    for heading in content.xpath('.//*[contains(@class, "doc-heading")]//*[self::h1 or self::h2 or self::h3]'):
        for child in list(heading):
            text = clean_text(child.text_content())
            if not text:
                heading.remove(child)


def rewrite_internal_link(
    href: str,
    *,
    current_url: str,
    current_local_path: Path,
    site_root: str,
    allowed_prefixes: tuple[str, ...],
    excluded_paths: set[str],
    local_base_prefix: str,
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

    full = urljoin(current_url, raw)
    parsed = urlparse(full)
    site = urlparse(site_root)

    if parsed.netloc == site.netloc and parsed.scheme in {"http", "https"}:
        canonical_path = canonicalize_path(parsed.path)
        normalized = f"{site.scheme}://{site.netloc}{canonical_path}"
        if is_allowed_path(canonical_path, allowed_prefixes, excluded_paths):
            local_target = url_to_local_doc_path(normalized, local_base_prefix)
            return relative_path(current_local_path, local_target) + anchor
        return normalized + anchor

    return full + anchor if parsed.scheme in {"http", "https"} else href


def rewrite_asset_url(
    value: str,
    *,
    current_url: str,
    current_local_path: Path,
    output_root: Path,
    session: requests.Session,
    downloaded_assets: dict[str, Path],
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

    return relative_path(current_local_path, downloaded_assets[full])


def rewrite_content(
    content: html.HtmlElement,
    *,
    current_url: str,
    current_local_path: Path,
    site_root: str,
    allowed_prefixes: tuple[str, ...],
    excluded_paths: set[str],
    local_base_prefix: str,
    output_root: Path,
    session: requests.Session,
    downloaded_assets: dict[str, Path],
) -> str:
    strip_unwanted_nodes(content)

    for anchor in content.xpath(".//a[@href]"):
        anchor.set(
            "href",
            rewrite_internal_link(
                anchor.get("href") or "",
                current_url=current_url,
                current_local_path=current_local_path,
                site_root=site_root,
                allowed_prefixes=allowed_prefixes,
                excluded_paths=excluded_paths,
                local_base_prefix=local_base_prefix,
            ),
        )

    for attr in ("src", "poster"):
        for node in content.xpath(f'.//*[@{attr}]'):
            node.set(
                attr,
                rewrite_asset_url(
                    node.get(attr) or "",
                    current_url=current_url,
                    current_local_path=current_local_path,
                    output_root=output_root,
                    session=session,
                    downloaded_assets=downloaded_assets,
                ),
            )

    return etree.tostring(content, encoding="unicode", method="html")


def build_landing_html(
    *,
    title: str,
    current_url: str,
    current_local_path: Path,
    site_root: str,
    allowed_prefixes: tuple[str, ...],
    excluded_paths: set[str],
    local_base_prefix: str,
    link_candidates: list[tuple[str, str]],
) -> str:
    lines = [f"<div><p>{title} 是该分区的入口页，正文极少，建议从以下目录继续阅读：</p><ul>"]
    for href, text in link_candidates:
        local = rewrite_internal_link(
            href,
            current_url=current_url,
            current_local_path=current_local_path,
            site_root=site_root,
            allowed_prefixes=allowed_prefixes,
            excluded_paths=excluded_paths,
            local_base_prefix=local_base_prefix,
        )
        lines.append(f'<li><a href="{local}">{clean_title_candidate(text)}</a></li>')
    lines.append("</ul></div>")
    return "\n".join(lines)


def is_meaningful_content(content_html: str) -> bool:
    text = clean_text(re.sub(r"<[^>]+>", " ", content_html))
    return len(text) >= 24


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
    write_text(
        output_root / local_path,
        "\n".join(
            [
                f"# {escape_markdown_heading_text(title)}",
                "",
                f"- 来源：`{source_url}`",
                "",
                "<!-- Douyin Mini Game doc body below keeps official HTML structure for fidelity. -->",
                "",
                content_html.strip(),
                "",
            ]
        ),
    )


def write_index_pages(
    *,
    output_root: Path,
    page_map: dict[str, Path],
    title_map: dict[str, str],
    asset_count: int,
    skipped_count: int,
) -> None:
    entry_lines = [
        "# 抖音小游戏文档本地入口说明",
        "",
        "这个目录是对抖音开放平台小游戏文档中“快速入门 + 开发”部分的本地 Markdown 镜像。",
        "",
        f"- 镜像页面数：`{len(page_map)}`",
        f"- 镜像资源数：`{asset_count}`",
        f"- 跳过页面数：`{skipped_count}`",
        "",
        "## 快速入口",
        "",
    ]
    for label, rel in (
        ("快速入门总览", "guide/overview.md"),
        ("开发总览", "develop/guide/dev-guide/bytedance-mini-game.md"),
        ("快速入门索引", "10-快速入门页面索引.md"),
        ("开发索引", "11-开发页面索引.md"),
        ("完整页面索引", "01-完整页面索引.md"),
    ):
        if (output_root / rel).exists():
            entry_lines.append(f"- [{label}]({rel})")
    entry_lines.append("")
    write_text(output_root / "00-本地入口说明.md", "\n".join(entry_lines))

    grouped: dict[str, list[tuple[str, Path]]] = {"guide": [], "develop": []}
    for url, path in page_map.items():
        bucket = path.parts[0] if path.parts else "other"
        if bucket not in grouped:
            grouped[bucket] = []
        grouped[bucket].append((title_map[url], path))

    full_index = [
        "# 抖音小游戏文档完整页面索引",
        "",
        "以下索引只包含“快速入门 + 开发”范围内的页面。",
        "",
    ]
    for bucket in sorted(grouped):
        full_index.append(f"## {bucket}")
        full_index.append("")
        for title, path in sorted(grouped[bucket], key=lambda item: item[1].as_posix()):
            full_index.append(f"- [{escape_markdown_link_text(title)}]({path.as_posix()})")
        full_index.append("")
    write_text(output_root / "01-完整页面索引.md", "\n".join(full_index).rstrip() + "\n")

    for filename, bucket, heading in (
        ("10-快速入门页面索引.md", "guide", "快速入门"),
        ("11-开发页面索引.md", "develop", "开发"),
    ):
        lines = [f"# {heading} 页面索引", ""]
        for title, path in sorted(grouped.get(bucket, []), key=lambda item: item[1].as_posix()):
            lines.append(f"- [{escape_markdown_link_text(title)}]({path.as_posix()})")
        lines.append("")
        write_text(output_root / filename, "\n".join(lines))


def crawl_and_mirror(
    *,
    site_root: str,
    local_base_prefix: str,
    allowed_prefixes: tuple[str, ...],
    excluded_paths: set[str],
    start_urls: list[str],
    output_root: Path,
) -> dict:
    session = make_session()
    output_root.mkdir(parents=True, exist_ok=True)

    queue: deque[str] = deque()
    queued: set[str] = set()
    page_map: dict[str, Path] = {}
    title_map: dict[str, str] = {}
    title_hints: dict[str, Counter[str]] = {}
    skipped_pages: dict[str, str] = {}
    downloaded_assets: dict[str, Path] = {}
    crawled_order: list[str] = []

    for url in start_urls:
        normalized = normalize_doc_url(site_root, allowed_prefixes, excluded_paths, url, url)
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

        root = html.fromstring(response.text)
        sidebar_links = get_sidebar_links(root)
        page_links = get_page_links(root)
        discoverable_links = [
            ((anchor.get("href") or "").strip(), clean_title_candidate(anchor.text_content()))
            for anchor in root.xpath("//a[@href]")
        ]

        try:
            content = find_content_node(root)
        except MirrorError as exc:
            skipped_pages[current_url] = str(exc)
            continue

        local_path = url_to_local_doc_path(current_url, local_base_prefix)
        raw_title = get_title(root)
        content_heading = extract_content_heading(content)
        title = choose_best_title(
            raw_title,
            content_heading,
            title_hints.get(current_url),
            local_path,
        )
        content_html = rewrite_content(
            content,
            current_url=current_url,
            current_local_path=local_path,
            site_root=site_root,
            allowed_prefixes=allowed_prefixes,
            excluded_paths=excluded_paths,
            local_base_prefix=local_base_prefix,
            output_root=output_root,
            session=session,
            downloaded_assets=downloaded_assets,
        )
        if not is_meaningful_content(content_html):
            landing_links = sidebar_links
            if len(landing_links) <= 1:
                landing_links = page_links
            content_html = build_landing_html(
                title=title,
                current_url=current_url,
                current_local_path=local_path,
                site_root=site_root,
                allowed_prefixes=allowed_prefixes,
                excluded_paths=excluded_paths,
                local_base_prefix=local_base_prefix,
                link_candidates=landing_links,
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

        for href, hint_text in discoverable_links:
            normalized = normalize_doc_url(
                site_root,
                allowed_prefixes,
                excluded_paths,
                current_url,
                href,
            )
            if normalized and hint_text:
                title_hints.setdefault(normalized, Counter())[hint_text] += 1
            if normalized and normalized not in queued and normalized not in page_map:
                queue.append(normalized)
                queued.add(normalized)

    write_index_pages(
        output_root=output_root,
        page_map=page_map,
        title_map=title_map,
        asset_count=len(downloaded_assets),
        skipped_count=len(skipped_pages),
    )

    category_counts = Counter()
    for path in page_map.values():
        category_counts[path.parts[0] if path.parts else "root"] += 1

    manifest = {
        "site_root": site_root,
        "local_base_prefix": local_base_prefix,
        "allowed_prefixes": list(allowed_prefixes),
        "excluded_paths": sorted(excluded_paths),
        "generated_at_utc": datetime.now(timezone.utc).isoformat(),
        "page_count": len(page_map),
        "asset_count": len(downloaded_assets),
        "skipped_page_count": len(skipped_pages),
        "output_root": str(output_root),
        "start_urls": start_urls,
        "category_counts": dict(category_counts),
        "crawled_order_sample": crawled_order[:60],
        "skipped_pages": skipped_pages,
    }
    write_text(
        output_root / "_mirror_manifest.json",
        json.dumps(manifest, ensure_ascii=False, indent=2) + "\n",
    )
    return manifest


def parse_args(argv: Iterable[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Mirror selected Douyin mini-game docs to local Markdown.")
    parser.add_argument("--site-root", default=DEFAULT_SITE_ROOT)
    parser.add_argument("--local-base-prefix", default=DEFAULT_LOCAL_BASE_PREFIX)
    parser.add_argument(
        "--allowed-prefix",
        action="append",
        dest="allowed_prefixes",
        help="Allowed doc path prefix. Can be passed multiple times.",
    )
    parser.add_argument(
        "--exclude-path",
        action="append",
        dest="excluded_paths",
        help="Excluded doc path. Can be passed multiple times.",
    )
    parser.add_argument(
        "--start-url",
        action="append",
        dest="start_urls",
        help="Seed page URL. Can be passed multiple times.",
    )
    parser.add_argument("--output-root", default=str(DEFAULT_OUTPUT_ROOT))
    return parser.parse_args(list(argv))


def main(argv: Iterable[str]) -> int:
    args = parse_args(argv)
    manifest = crawl_and_mirror(
        site_root=args.site_root,
        local_base_prefix=args.local_base_prefix,
        allowed_prefixes=tuple(args.allowed_prefixes or DEFAULT_ALLOWED_PREFIXES),
        excluded_paths=set(args.excluded_paths or DEFAULT_EXCLUDED_PATHS),
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
