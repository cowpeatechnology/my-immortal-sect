#!/usr/bin/env python3
"""Bootstrap and maintain a wiki-first Obsidian knowledge base.

This tool follows the "LLM Wiki" pattern:
- raw sources are immutable
- the wiki is the maintained synthesis layer
- schema files tell future agents how to operate on the wiki

Current implementation focuses on the highest-value loop:
1. bootstrap the wiki structure inside an Obsidian vault
2. ingest a URL into raw + wiki
3. rebuild index.md and append log.md
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
from datetime import datetime
from pathlib import Path
from typing import Dict, Iterable, List, Optional, Sequence, Tuple
from urllib.parse import urlparse

import requests

from url_to_obsidian import (
    AUTH_FILE,
    DEFAULT_FETCH_TIMEOUT,
    DEFAULT_LLM_TIMEOUT,
    DEFAULT_MAX_SOURCE_CHARS,
    DEFAULT_TEXT_MODEL,
    CodexOAuthError,
    ExtractedPage,
    UrlToObsidianError,
    build_fallback_markdown,
    build_text_request_body,
    collect_text_result,
    ensure_fresh_tokens,
    extract_page,
    fetch_url,
    load_codex_auth_file,
    normalize_text,
    resolve_vault_path,
    sanitize_filename,
    truncate_for_model,
)


DEFAULT_PROJECT_FOLDER = "Projects/我的宗门 Wiki"
DEFAULT_PROJECT_NAME = "我的宗门"
DEFAULT_TIMEZONE_LABEL = "Asia/Shanghai"
DEFAULT_TAGS = ("my-immortal-sect", "llm-wiki")
DEFAULT_MAX_TITLE_LENGTH = 96
MARKDOWN_H1_RE = re.compile(r"^\s*#\s+(.+?)\s*$", re.MULTILINE)

WIKI_SOURCE_SYSTEM_INSTRUCTIONS = """You are maintaining a long-lived Obsidian research wiki for a game project.

Write in Chinese.
Do not quote the source at length.
Separate source claims from project-specific implications.
Preserve durable workflows, architecture ideas, constraints, and operational guidance.
Prefer concise, high-signal sections over generic prose.

Output Markdown only with this structure:
# 标题
## 一句话结论
## 摘要
- 5 到 8 条高信息量要点
## 核心观点
## 对《我的宗门》的直接启发
## 可执行动作
## 待验证问题
"""

SCHEMA_TEMPLATE = """# {project_name} LLM Wiki 维护规则

本目录采用 wiki-first 知识维护模式：

- `raw/`：原始资料层。网页提取、PDF 转文本、附件索引等。只读，不覆盖。
- `wiki/`：知识层。由 LLM 维护的总结、分析、概念页、实体页、对比页。
- `schema/`：规则层。定义目录约定、入库流程、查询流程、lint 流程。

## 基本规则

1. 每次 ingest 新来源时，必须同时完成：
   - 保存 `raw/sources/` 原始提取稿
   - 生成或更新 `wiki/sources/` 来源页
   - 重建 `wiki/index.md`
   - 追加 `wiki/log.md`

2. `raw/` 是证据层：
   - 允许新增
   - 不允许覆盖历史来源
   - 不允许把总结反写到 `raw/`

3. `wiki/` 是可演化层：
   - 允许跨多个页面更新
   - 要显式区分“来源事实”和“对项目的推断/建议”
   - 回答用户问题时优先读取 `wiki/index.md` 和相关页面，而不是重新从原始网页开始

4. 如果一次问答产生了 durable insight，应优先沉淀到：
   - `wiki/analyses/`
   - 或相关主题页，而不是只留在聊天记录里

5. lint 时优先检查：
   - 孤儿页
   - 过时结论
   - 缺失的主题页
   - 明显冲突的页面陈述
   - `TODO: verify` 类待验证点

## 当前项目重点

- 平台：微信小游戏、抖音小游戏
- 引擎：Cocos Creator 3.8.8
- 类型：修仙宗门模拟经营，RimWorld 风格叙事与任务系统
- 当前重点：Tilemap、小游戏工程约束、系统设计、知识采集工作流
"""

OVERVIEW_TEMPLATE = """# {project_name} Wiki Overview

这个知识库用于沉淀《{project_name}》的长期外部研究与内部分析。

当前约定：

- 原始资料放在 `../raw/`
- 知识页放在本目录
- `index.md` 是导航入口
- `log.md` 是时间线入口

建议工作流：

1. 新资料进入 `raw/sources/`
2. LLM 生成或更新 `wiki/sources/`、`wiki/analyses/` 等页面
3. `index.md` 和 `log.md` 同步刷新
4. 问答产生的新分析，优先沉淀回 wiki
"""


class WikiMaintainerError(RuntimeError):
    """Raised when the wiki maintainer cannot complete its task."""


def now_local() -> datetime:
    return datetime.now().astimezone()


def now_iso() -> str:
    return now_local().replace(microsecond=0).isoformat()


def bounded_title(title: str) -> str:
    cleaned = sanitize_filename(title or "untitled-source")
    if len(cleaned) <= DEFAULT_MAX_TITLE_LENGTH:
        return cleaned
    return cleaned[:DEFAULT_MAX_TITLE_LENGTH].rstrip(" -_.")


def improve_extracted_title(extracted: ExtractedPage) -> ExtractedPage:
    title = normalize_text(extracted.title)
    if title and title.lower() not in {"untitled page", "untitled"}:
        return extracted

    match = MARKDOWN_H1_RE.search(extracted.extracted_markdown)
    if match:
        extracted.title = normalize_text(match.group(1))
    return extracted


def build_plaintext_page(url: str, timeout: int) -> ExtractedPage:
    final_url, body_text, content_type = fetch_url(url, timeout)
    extracted_markdown = body_text.strip()
    title = ""
    for line in extracted_markdown.splitlines():
        stripped = line.strip()
        if stripped.startswith("# "):
            title = normalize_text(stripped[2:])
            break
        if stripped:
            title = normalize_text(stripped)
            break
    if not title:
        title = Path(urlparse(final_url).path).name or "Untitled Page"

    return ExtractedPage(
        source_url=url,
        final_url=final_url,
        canonical_url=final_url,
        title=title,
        site_name=urlparse(final_url).netloc,
        description="",
        published_at="",
        extracted_markdown=extracted_markdown,
        content_type=content_type,
    )


def extract_source_page(url: str, timeout: int) -> ExtractedPage:
    final_url, _, content_type = fetch_url(url, timeout)
    parsed = urlparse(final_url)
    path_lower = parsed.path.lower()
    if content_type.startswith("text/plain") or path_lower.endswith(".md"):
        return improve_extracted_title(build_plaintext_page(url, timeout))
    return improve_extracted_title(extract_page(url, timeout))


def yaml_escape(text: str) -> str:
    return text.replace("\\", "\\\\").replace('"', '\\"')


def relative_link(from_path: Path, to_path: Path) -> str:
    return Path(os.path.relpath(to_path, from_path.parent)).as_posix()


def project_paths(vault: Path, project_folder: str) -> Dict[str, Path]:
    root = (vault / project_folder).resolve()
    return {
        "root": root,
        "raw_root": root / "raw",
        "raw_sources": root / "raw" / "sources",
        "raw_assets": root / "raw" / "assets",
        "wiki_root": root / "wiki",
        "wiki_sources": root / "wiki" / "sources",
        "wiki_concepts": root / "wiki" / "concepts",
        "wiki_entities": root / "wiki" / "entities",
        "wiki_analyses": root / "wiki" / "analyses",
        "schema_root": root / "schema",
        "schema_agents": root / "schema" / "AGENTS.md",
        "overview": root / "wiki" / "overview.md",
        "index": root / "wiki" / "index.md",
        "log": root / "wiki" / "log.md",
    }


def ensure_dirs(paths: Dict[str, Path]) -> None:
    for key, path in paths.items():
        if key.endswith("_root") or key.endswith("_sources") or key.endswith("_assets") or key.endswith("_concepts") or key.endswith("_entities") or key.endswith("_analyses"):
            path.mkdir(parents=True, exist_ok=True)
    paths["root"].mkdir(parents=True, exist_ok=True)


def write_if_missing(path: Path, content: str) -> bool:
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists():
        return False
    path.write_text(content, encoding="utf-8")
    return True


def bootstrap_wiki(*, vault: Path, project_folder: str, project_name: str) -> Dict[str, Path]:
    paths = project_paths(vault, project_folder)
    ensure_dirs(paths)

    write_if_missing(paths["schema_agents"], SCHEMA_TEMPLATE.format(project_name=project_name))
    write_if_missing(paths["overview"], OVERVIEW_TEMPLATE.format(project_name=project_name))
    write_if_missing(paths["index"], "# Index\n\n## Sources\n\n暂无来源页。\n")
    write_if_missing(paths["log"], "# Log\n\n")
    return paths


def summarize_for_wiki(
    extracted: ExtractedPage,
    *,
    auth_file: Path,
    model: str,
    timeout: int,
    max_source_chars: int,
    force_refresh: bool,
    project_name: str,
    source_url: str,
) -> str:
    auth_state = load_codex_auth_file(auth_file.expanduser())
    auth_state = ensure_fresh_tokens(auth_state, force_refresh=force_refresh)

    source_excerpt = truncate_for_model(extracted.extracted_markdown, max_source_chars)
    prompt = (
        f"Project: {project_name}\n"
        f"Original Source URL: {source_url}\n"
        f"Fetch URL: {extracted.source_url}\n"
        f"Final URL: {extracted.final_url}\n"
        f"Canonical URL: {extracted.canonical_url}\n"
        f"Title: {extracted.title}\n"
        f"Site Name: {extracted.site_name or 'N/A'}\n"
        f"Description: {extracted.description or 'N/A'}\n"
        f"Published At: {extracted.published_at or 'N/A'}\n"
        f"Content Type: {extracted.content_type or 'N/A'}\n\n"
        "Task: produce a durable wiki source note for this source, with direct implications for the game project.\n\n"
        "<source>\n"
        f"{source_excerpt}\n"
        "</source>\n"
    )
    body = build_text_request_body(
        prompt=prompt,
        instructions=WIKI_SOURCE_SYSTEM_INSTRUCTIONS,
        model=model,
    )
    result = collect_text_result(
        access_token=auth_state["access_token"],
        account_id=auth_state["account_id"],
        body=body,
        timeout=timeout,
    )
    return result["assistant_text"]


def extract_summary_line(markdown_text: str) -> str:
    lines = [line.strip() for line in markdown_text.splitlines() if line.strip()]
    for line in lines:
        if line.startswith("- "):
            return normalize_text(line[2:])
    for line in lines:
        if not line.startswith("#"):
            return normalize_text(line)
    return "No summary extracted."


def build_frontmatter(metadata: Dict[str, str], *, tags: Sequence[str]) -> str:
    lines = ["---"]
    for key, value in metadata.items():
        lines.append(f'{key}: "{yaml_escape(value)}"')
    lines.append("tags:")
    for tag in tags:
        lines.append(f"  - {tag}")
    lines.append("---")
    return "\n".join(lines) + "\n\n"


def build_raw_note(
    *,
    extracted: ExtractedPage,
    source_url: str,
    fetched_at: str,
    note_title: str,
) -> str:
    frontmatter = build_frontmatter(
        {
            "title": note_title,
            "note_type": "raw-source",
            "source_url": source_url,
            "fetch_url": extracted.source_url,
            "final_url": extracted.final_url,
            "canonical_url": extracted.canonical_url,
            "content_type": extracted.content_type or "unknown",
            "published_at": extracted.published_at or "",
            "ingested_at": fetched_at,
        },
        tags=[*DEFAULT_TAGS, "raw-source"],
    )
    metadata_block = (
        "## Metadata\n"
        f"- Original Source URL: {source_url}\n"
        f"- Fetch URL: {extracted.source_url}\n"
        f"- Final URL: {extracted.final_url}\n"
        f"- Canonical URL: {extracted.canonical_url}\n"
        f"- Site: {extracted.site_name or 'N/A'}\n"
        f"- Published At: {extracted.published_at or 'N/A'}\n"
        f"- Fetched At: {fetched_at}\n\n"
        "## Extracted Content\n\n"
    )
    return frontmatter + f"# {note_title}\n\n" + metadata_block + extracted.extracted_markdown.strip() + "\n"


def build_source_note(
    *,
    extracted: ExtractedPage,
    source_url: str,
    fetch_url: str,
    fetched_at: str,
    note_title: str,
    raw_note_relative: str,
    model_name: str,
    model_output: str,
) -> str:
    summary_line = extract_summary_line(model_output)
    frontmatter = build_frontmatter(
        {
            "title": note_title,
            "note_type": "source-note",
            "source_url": source_url,
            "fetch_url": fetch_url,
            "final_url": extracted.final_url,
            "canonical_url": extracted.canonical_url,
            "source_domain": urlparse(extracted.final_url or source_url).netloc.lower(),
            "published_at": extracted.published_at or "",
            "ingested_at": fetched_at,
            "model": model_name,
            "summary_line": summary_line,
            "raw_note": raw_note_relative,
        },
        tags=[*DEFAULT_TAGS, "source-note"],
    )
    source_meta = (
        "## Source Metadata\n"
        f"- Original Source URL: {source_url}\n"
        f"- Fetch URL: {fetch_url}\n"
        f"- Final URL: {extracted.final_url}\n"
        f"- Canonical URL: {extracted.canonical_url}\n"
        f"- Raw Note: [{raw_note_relative}]({raw_note_relative})\n"
        f"- Site: {extracted.site_name or 'N/A'}\n"
        f"- Published At: {extracted.published_at or 'N/A'}\n"
        f"- Ingested At: {fetched_at}\n"
    )
    body = model_output.strip()
    if not body.startswith("# "):
        body = f"# {note_title}\n\n{body}"
    return frontmatter + body + "\n\n" + source_meta + "\n"


def parse_frontmatter(path: Path) -> Dict[str, str]:
    text = path.read_text(encoding="utf-8")
    if not text.startswith("---\n"):
        return {}
    try:
        _, rest = text.split("---\n", 1)
        fm_text, _ = rest.split("\n---\n", 1)
    except ValueError:
        return {}

    metadata: Dict[str, str] = {}
    for line in fm_text.splitlines():
        if not line or line.startswith("tags:") or line.startswith("  - "):
            continue
        if ":" not in line:
            continue
        key, raw_value = line.split(":", 1)
        value = raw_value.strip().strip('"')
        metadata[key.strip()] = value
    return metadata


def iter_markdown_files(directory: Path) -> Iterable[Path]:
    if not directory.exists():
        return []
    return sorted(path for path in directory.glob("*.md") if path.is_file())


def build_index(paths: Dict[str, Path], *, project_name: str) -> str:
    source_lines: List[str] = []
    for path in iter_markdown_files(paths["wiki_sources"]):
        metadata = parse_frontmatter(path)
        title = metadata.get("title") or path.stem
        summary = metadata.get("summary_line") or "No summary."
        rel = path.relative_to(paths["wiki_root"]).as_posix()
        source_lines.append(f"- [{title}]({rel}) - {summary}")

    concept_lines: List[str] = []
    for path in iter_markdown_files(paths["wiki_concepts"]):
        metadata = parse_frontmatter(path)
        title = metadata.get("title") or path.stem
        summary = metadata.get("summary_line") or "No summary."
        rel = path.relative_to(paths["wiki_root"]).as_posix()
        concept_lines.append(f"- [{title}]({rel}) - {summary}")

    analysis_lines: List[str] = []
    for path in iter_markdown_files(paths["wiki_analyses"]):
        metadata = parse_frontmatter(path)
        title = metadata.get("title") or path.stem
        summary = metadata.get("summary_line") or "No summary."
        rel = path.relative_to(paths["wiki_root"]).as_posix()
        analysis_lines.append(f"- [{title}]({rel}) - {summary}")

    return (
        f"# {project_name} Index\n\n"
        f"- [overview.md](overview.md)\n"
        f"- [log.md](log.md)\n\n"
        "## Sources\n\n"
        + ("\n".join(source_lines) if source_lines else "暂无来源页。")
        + "\n\n## Concepts\n\n"
        + ("\n".join(concept_lines) if concept_lines else "暂无概念页。")
        + "\n\n## Analyses\n\n"
        + ("\n".join(analysis_lines) if analysis_lines else "暂无分析页。")
        + "\n"
    )


def append_log_entry(
    *,
    log_path: Path,
    ingested_at: str,
    title: str,
    source_url: str,
    source_note_rel: str,
    raw_note_rel: str,
    summary_line: str,
) -> None:
    entry = (
        f"## [{ingested_at}] ingest | {title}\n"
        f"- Source URL: {source_url}\n"
        f"- Source Note: [{source_note_rel}]({source_note_rel})\n"
        f"- Raw Note: [{raw_note_rel}]({raw_note_rel})\n"
        f"- Summary: {summary_line}\n\n"
    )
    log_path.parent.mkdir(parents=True, exist_ok=True)
    with log_path.open("a", encoding="utf-8") as handle:
        handle.write(entry)


def write_note(path: Path, content: str, *, overwrite: bool = False) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists() and not overwrite:
        raise WikiMaintainerError(f"Refusing to overwrite existing note: {path}")
    path.write_text(content, encoding="utf-8")


def build_collision_suffix(*, source_url: str, extracted: ExtractedPage) -> str:
    uniqueness_key = extracted.canonical_url or extracted.final_url or source_url
    return hashlib.sha1(uniqueness_key.encode("utf-8")).hexdigest()[:8]


def resolve_note_pair_paths(
    *,
    paths: Dict[str, Path],
    base_stem: str,
    collision_suffix: str,
    overwrite: bool,
) -> Tuple[Path, Path]:
    def pair_for_stem(stem: str) -> Tuple[Path, Path]:
        return (
            paths["raw_sources"] / f"{stem}.md",
            paths["wiki_sources"] / f"{stem}.md",
        )

    raw_note_path, source_note_path = pair_for_stem(base_stem)
    if overwrite or (not raw_note_path.exists() and not source_note_path.exists()):
        return raw_note_path, source_note_path

    candidate_stem = f"{base_stem} [{collision_suffix}]"
    raw_note_path, source_note_path = pair_for_stem(candidate_stem)
    if overwrite or (not raw_note_path.exists() and not source_note_path.exists()):
        return raw_note_path, source_note_path

    counter = 2
    while True:
        candidate_stem = f"{base_stem} [{collision_suffix}-{counter}]"
        raw_note_path, source_note_path = pair_for_stem(candidate_stem)
        if overwrite or (not raw_note_path.exists() and not source_note_path.exists()):
            return raw_note_path, source_note_path
        counter += 1


def ingest_url(
    *,
    vault: Path,
    project_folder: str,
    project_name: str,
    source_url: str,
    fetch_url: Optional[str],
    auth_file: Path,
    model: str,
    fetch_timeout: int,
    llm_timeout: int,
    max_source_chars: int,
    force_refresh: bool,
    skip_llm: bool,
    overwrite: bool,
) -> Dict[str, Path]:
    paths = bootstrap_wiki(vault=vault, project_folder=project_folder, project_name=project_name)
    effective_fetch_url = fetch_url or source_url
    extracted = extract_source_page(effective_fetch_url, fetch_timeout)
    ingested_at = now_iso()

    title = bounded_title(extracted.title or "untitled-source")
    date_prefix = now_local().strftime("%Y-%m-%d")
    base_stem = f"{date_prefix} {title}"
    raw_note_path, source_note_path = resolve_note_pair_paths(
        paths=paths,
        base_stem=base_stem,
        collision_suffix=build_collision_suffix(source_url=source_url, extracted=extracted),
        overwrite=overwrite,
    )

    raw_note_rel = relative_link(source_note_path, raw_note_path)
    source_note_rel = source_note_path.relative_to(paths["wiki_root"]).as_posix()
    log_raw_note_rel = relative_link(paths["log"], raw_note_path)
    log_source_note_rel = relative_link(paths["log"], source_note_path)

    if skip_llm:
        model_output = build_fallback_markdown(extracted)
        model_name = "skip-llm"
    else:
        model_output = summarize_for_wiki(
            extracted,
            auth_file=auth_file,
            model=model,
            timeout=llm_timeout,
            max_source_chars=max_source_chars,
            force_refresh=force_refresh,
            project_name=project_name,
            source_url=source_url,
        )
        model_name = model

    raw_note_content = build_raw_note(
        extracted=extracted,
        source_url=source_url,
        fetched_at=ingested_at,
        note_title=title,
    )
    source_note_content = build_source_note(
        extracted=extracted,
        source_url=source_url,
        fetch_url=effective_fetch_url,
        fetched_at=ingested_at,
        note_title=title,
        raw_note_relative=raw_note_rel,
        model_name=model_name,
        model_output=model_output,
    )

    write_note(raw_note_path, raw_note_content, overwrite=overwrite)
    write_note(source_note_path, source_note_content, overwrite=overwrite)

    index_text = build_index(paths, project_name=project_name)
    paths["index"].write_text(index_text, encoding="utf-8")
    append_log_entry(
        log_path=paths["log"],
        ingested_at=ingested_at,
        title=title,
        source_url=source_url,
        source_note_rel=log_source_note_rel,
        raw_note_rel=log_raw_note_rel,
        summary_line=extract_summary_line(model_output),
    )

    return {
        "project_root": paths["root"],
        "raw_note": raw_note_path,
        "source_note": source_note_path,
        "index": paths["index"],
        "log": paths["log"],
        "schema": paths["schema_agents"],
    }


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Bootstrap and maintain a wiki-first Obsidian knowledge base.")
    parser.add_argument(
        "--vault",
        help="Absolute path to the Obsidian vault. Defaults to OBSIDIAN_VAULT_PATH.",
    )
    parser.add_argument(
        "--project-folder",
        default=DEFAULT_PROJECT_FOLDER,
        help=f'Folder inside the vault for the project wiki. Default: "{DEFAULT_PROJECT_FOLDER}"',
    )
    parser.add_argument(
        "--project-name",
        default=DEFAULT_PROJECT_NAME,
        help=f'Project name used in generated wiki files. Default: "{DEFAULT_PROJECT_NAME}"',
    )

    subparsers = parser.add_subparsers(dest="command", required=True)

    subparsers.add_parser("bootstrap", help="Create the wiki-first knowledge base skeleton.")

    ingest = subparsers.add_parser("ingest-url", help="Ingest one URL into raw + wiki + index + log.")
    ingest.add_argument("source_url", help="The original source URL that should appear in the wiki.")
    ingest.add_argument("--fetch-url", help="Optional alternate URL to fetch. Useful for raw markdown mirrors of the same source.")
    ingest.add_argument(
        "--auth-file",
        default=str(AUTH_FILE),
        help=f"Codex auth file path. Default: {AUTH_FILE}",
    )
    ingest.add_argument(
        "--model",
        default=DEFAULT_TEXT_MODEL,
        help=f"OAuth-backed text model. Default: {DEFAULT_TEXT_MODEL}",
    )
    ingest.add_argument(
        "--fetch-timeout",
        type=int,
        default=DEFAULT_FETCH_TIMEOUT,
        help=f"HTTP fetch timeout in seconds. Default: {DEFAULT_FETCH_TIMEOUT}",
    )
    ingest.add_argument(
        "--llm-timeout",
        type=int,
        default=DEFAULT_LLM_TIMEOUT,
        help=f"Model request timeout in seconds. Default: {DEFAULT_LLM_TIMEOUT}",
    )
    ingest.add_argument(
        "--max-source-chars",
        type=int,
        default=DEFAULT_MAX_SOURCE_CHARS,
        help=f"Maximum source characters sent to the model. Default: {DEFAULT_MAX_SOURCE_CHARS}",
    )
    ingest.add_argument("--force-refresh", action="store_true", help="Refresh the OAuth token before the model request.")
    ingest.add_argument("--skip-llm", action="store_true", help="Write a deterministic fallback source note without the OAuth call.")
    ingest.add_argument("--overwrite", action="store_true", help="Overwrite existing raw/source notes with the same generated file name.")

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    vault = resolve_vault_path(args.vault)

    if args.command == "bootstrap":
        paths = bootstrap_wiki(vault=vault, project_folder=args.project_folder, project_name=args.project_name)
        print(f"Vault       : {vault}")
        print(f"Project Root: {paths['root']}")
        print(f"Schema      : {paths['schema_agents']}")
        print(f"Overview    : {paths['overview']}")
        print(f"Index       : {paths['index']}")
        print(f"Log         : {paths['log']}")
        return 0

    if args.command == "ingest-url":
        paths = ingest_url(
            vault=vault,
            project_folder=args.project_folder,
            project_name=args.project_name,
            source_url=args.source_url,
            fetch_url=args.fetch_url,
            auth_file=Path(args.auth_file),
            model=args.model,
            fetch_timeout=args.fetch_timeout,
            llm_timeout=args.llm_timeout,
            max_source_chars=args.max_source_chars,
            force_refresh=args.force_refresh,
            skip_llm=args.skip_llm,
            overwrite=args.overwrite,
        )
        print(f"Project Root: {paths['project_root']}")
        print(f"Raw Note    : {paths['raw_note']}")
        print(f"Source Note : {paths['source_note']}")
        print(f"Index       : {paths['index']}")
        print(f"Log         : {paths['log']}")
        print(f"Schema      : {paths['schema']}")
        return 0

    raise WikiMaintainerError(f"Unsupported command: {args.command}")


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (WikiMaintainerError, UrlToObsidianError, CodexOAuthError, requests.RequestException) as exc:
        print(f"ERROR: {exc}")
        raise SystemExit(1)
