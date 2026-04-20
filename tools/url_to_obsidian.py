#!/usr/bin/env python3
"""Fetch a webpage, summarize it with Codex OAuth, and write an Obsidian note.

Default flow:
1. Fetch URL / local file URL
2. Extract the main readable content with lxml
3. Send the extracted text to the ChatGPT Codex responses backend using the
   local OAuth session from ``~/.codex/auth.json``
4. Write a Markdown note into an Obsidian vault

The tool intentionally writes to the vault via the filesystem instead of
depending on the Obsidian CLI for note creation. That makes the workflow more
robust when Obsidian CLI is not installed. If requested, it can still open the
note through the CLI or the official ``obsidian://`` URI.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, Iterable, List, Optional, Sequence, Tuple
from urllib.parse import quote, urlparse

import lxml.html
import requests

from gpt_image_1_5_codex_oauth import (
    AUTH_FILE,
    DEFAULT_TEXT_MODEL,
    CodexOAuthImageError as CodexOAuthError,
    ensure_fresh_tokens,
    load_codex_auth_file,
    post_codex_response_stream,
)


DEFAULT_FETCH_TIMEOUT = 30
DEFAULT_LLM_TIMEOUT = 300
DEFAULT_MAX_SOURCE_CHARS = 20000
DEFAULT_FOLDER = "Inbox/Web Research"
DEFAULT_NOTE_TAGS = ("web-ingest", "reference")
DEFAULT_USER_AGENT = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
    "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"
)
DEFAULT_SYSTEM_INSTRUCTIONS = """You are a precise research assistant.

Transform extracted webpage content into a clean Obsidian-ready Markdown note in Chinese.
Priorities:
- extract durable knowledge, workflows, constraints, and implementation guidance
- distinguish facts from inference
- preserve important commands, flags, file paths, API names, and version numbers
- paraphrase instead of quoting; if quoting is necessary, keep quotes very short
- write Markdown only, without wrapping the whole answer in a code fence

Use this structure:
# 摘要
- 5 到 8 条高信息量要点
## 核心信息
## 实操建议
## 风险与限制
## 后续可追踪的问题

If the page is mostly marketing or narrative rather than guidance, say so clearly and still extract any durable takeaways.
"""

DROP_XPATH = (
    "//script|//style|//noscript|//svg|//canvas|//iframe|//form|//button|"
    "//nav|//footer|//header|//aside"
)
BLOCK_XPATH = (
    ".//*[self::h1 or self::h2 or self::h3 or self::h4 or self::h5 or self::h6 "
    "or self::p or self::li or self::pre or self::blockquote or self::table]"
)
YAML_UNSAFE = re.compile(r'["\\]')
FILENAME_UNSAFE = re.compile(r'[\\/:*?"<>|]+')
MULTISPACE = re.compile(r"\s+")


class UrlToObsidianError(RuntimeError):
    """Raised when the ingest flow cannot complete."""


@dataclass
class ExtractedPage:
    source_url: str
    final_url: str
    canonical_url: str
    title: str
    site_name: str
    description: str
    published_at: str
    extracted_markdown: str
    content_type: str


def now_utc_iso() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


def normalize_text(text: str) -> str:
    return MULTISPACE.sub(" ", text.replace("\xa0", " ")).strip()


def yaml_escape(text: str) -> str:
    return YAML_UNSAFE.sub(lambda match: "\\" + match.group(0), text)


def sanitize_filename(text: str, fallback: str = "web-note") -> str:
    cleaned = FILENAME_UNSAFE.sub("-", text).strip().strip(".")
    cleaned = MULTISPACE.sub(" ", cleaned)
    return cleaned or fallback


def decode_http_text(response: requests.Response) -> str:
    """Decode HTTP bodies with a safer fallback for mislabelled HTML pages."""

    encoding = response.encoding
    apparent = getattr(response, "apparent_encoding", None)
    if not encoding or encoding.lower() in {"iso-8859-1", "latin-1", "ascii"}:
        encoding = apparent or encoding or "utf-8"
    try:
        return response.content.decode(encoding, errors="replace")
    except LookupError:
        fallback = apparent or "utf-8"
        return response.content.decode(fallback, errors="replace")


def fetch_url(url: str, timeout: int) -> Tuple[str, str, str]:
    parsed = urlparse(url)
    if parsed.scheme == "file":
        file_path = Path(parsed.path)
        if not file_path.exists():
            raise UrlToObsidianError(f"Local file URL does not exist: {file_path}")
        return url, file_path.read_text(encoding="utf-8"), "text/html"

    headers = {"User-Agent": DEFAULT_USER_AGENT}
    response = requests.get(url, headers=headers, timeout=timeout)
    response.raise_for_status()
    content_type = response.headers.get("content-type", "")
    return response.url, decode_http_text(response), content_type


def meta_content(doc, *, xpath_expr: str) -> str:
    values = doc.xpath(xpath_expr)
    for value in values:
        text = normalize_text(value)
        if text:
            return text
    return ""


def pick_best_root(doc) -> object:
    candidates = doc.xpath("//article|//main|//*[@role='main']|//body")
    best = None
    best_score = -1
    for node in candidates:
        text_len = len(normalize_text(node.text_content()))
        block_count = len(node.xpath(BLOCK_XPATH))
        score = text_len + (block_count * 80)
        if score > best_score:
            best = node
            best_score = score
    return best if best is not None else doc


def extract_blocks(root) -> List[str]:
    lines: List[str] = []
    seen = set()
    for node in root.xpath(BLOCK_XPATH):
        text = normalize_text(node.text_content())
        if not text:
            continue
        if text in seen:
            continue
        seen.add(text)

        tag = node.tag.lower() if isinstance(node.tag, str) else ""
        if tag.startswith("h") and len(tag) == 2 and tag[1].isdigit():
            level = min(int(tag[1]), 6)
            lines.append(f"{'#' * level} {text}")
            continue
        if tag == "li":
            lines.append(f"- {text}")
            continue
        if tag == "blockquote":
            quoted = "\n".join(f"> {line}" for line in text.splitlines() if line.strip())
            lines.append(quoted or f"> {text}")
            continue
        if tag == "pre":
            lines.append(f"```text\n{text}\n```")
            continue
        if tag == "table":
            lines.append(f"Table: {text}")
            continue
        lines.append(text)
    return lines


def extract_page(url: str, timeout: int) -> ExtractedPage:
    final_url, html_text, content_type = fetch_url(url, timeout)
    doc = lxml.html.fromstring(html_text, base_url=final_url)
    lxml.html.etree.strip_elements(doc, "script", "style", "noscript", with_tail=False)
    for node in doc.xpath(DROP_XPATH):
        parent = node.getparent()
        if parent is not None:
            parent.remove(node)

    title = (
        meta_content(doc, xpath_expr="//meta[@property='og:title']/@content")
        or meta_content(doc, xpath_expr="//meta[@name='twitter:title']/@content")
        or normalize_text("".join(doc.xpath("//title/text()")))
        or meta_content(doc, xpath_expr="//h1[1]/text()")
        or "Untitled Page"
    )
    canonical = (
        meta_content(doc, xpath_expr="//link[@rel='canonical']/@href")
        or final_url
    )
    description = (
        meta_content(doc, xpath_expr="//meta[@name='description']/@content")
        or meta_content(doc, xpath_expr="//meta[@property='og:description']/@content")
    )
    site_name = meta_content(doc, xpath_expr="//meta[@property='og:site_name']/@content")
    published_at = (
        meta_content(doc, xpath_expr="//meta[@property='article:published_time']/@content")
        or meta_content(doc, xpath_expr="//time/@datetime")
    )

    root = pick_best_root(doc)
    blocks = extract_blocks(root)
    extracted_markdown = "\n\n".join(blocks).strip()
    if not extracted_markdown:
        extracted_markdown = normalize_text(root.text_content())

    return ExtractedPage(
        source_url=url,
        final_url=final_url,
        canonical_url=canonical,
        title=title,
        site_name=site_name,
        description=description,
        published_at=published_at,
        extracted_markdown=extracted_markdown,
        content_type=content_type,
    )


def truncate_for_model(text: str, max_chars: int) -> str:
    if len(text) <= max_chars:
        return text
    return text[:max_chars].rstrip() + "\n\n[TRUNCATED]"


def build_text_request_body(*, prompt: str, instructions: str, model: str) -> Dict:
    return {
        "model": model,
        "store": False,
        "stream": True,
        "instructions": instructions,
        "input": [
            {
                "role": "user",
                "content": [
                    {
                        "type": "input_text",
                        "text": prompt,
                    }
                ],
            }
        ],
        "text": {"verbosity": "medium"},
        "include": ["reasoning.encrypted_content"],
    }


def extract_text_from_response_output(output: Sequence[Dict]) -> str:
    texts: List[str] = []
    for item in output:
        if not isinstance(item, dict):
            continue
        if item.get("type") == "message":
            for content in item.get("content", []):
                if isinstance(content, dict) and content.get("type") == "output_text":
                    text = content.get("text")
                    if isinstance(text, str) and text.strip():
                        texts.append(text.strip())
        if item.get("type") == "output_text":
            text = item.get("text")
            if isinstance(text, str) and text.strip():
                texts.append(text.strip())
    return "\n\n".join(texts).strip()


def collect_text_result(
    *,
    access_token: str,
    account_id: str,
    body: Dict,
    timeout: int,
) -> Dict:
    completed_response: Optional[Dict] = None
    text_parts: List[str] = []
    event_types: List[str] = []
    failed_message: Optional[str] = None

    for event in post_codex_response_stream(
        access_token=access_token,
        account_id=account_id,
        body=body,
        timeout=timeout,
    ):
        event_type = event.get("type")
        if isinstance(event_type, str):
            event_types.append(event_type)

        if event_type == "response.output_text.delta":
            delta = event.get("delta")
            if isinstance(delta, str):
                text_parts.append(delta)
            continue

        if event_type in ("response.completed", "response.done"):
            response = event.get("response")
            if isinstance(response, dict):
                completed_response = response
            continue

        if event_type == "response.failed":
            err = event.get("response", {}).get("error", {})
            failed_message = err.get("message") or json.dumps(event, ensure_ascii=False)
            continue

        if event_type == "error":
            failed_message = event.get("message") or json.dumps(event, ensure_ascii=False)

    assistant_text = "".join(text_parts).strip()
    if not assistant_text and completed_response:
        assistant_text = extract_text_from_response_output(completed_response.get("output", []))

    if not assistant_text and failed_message:
        raise UrlToObsidianError(f"Model request failed: {failed_message}")
    if not assistant_text:
        raise UrlToObsidianError("Model request completed but returned no text.")

    return {
        "assistant_text": assistant_text,
        "response": completed_response or {},
        "event_types": event_types,
        "error_message": failed_message,
    }


def summarize_page_with_oauth(
    extracted: ExtractedPage,
    *,
    auth_file: Path,
    model: str,
    timeout: int,
    max_source_chars: int,
    force_refresh: bool,
) -> Dict:
    auth_state = load_codex_auth_file(auth_file.expanduser())
    auth_state = ensure_fresh_tokens(auth_state, force_refresh=force_refresh)

    source_excerpt = truncate_for_model(extracted.extracted_markdown, max_source_chars)
    prompt = (
        f"Source URL: {extracted.source_url}\n"
        f"Final URL: {extracted.final_url}\n"
        f"Canonical URL: {extracted.canonical_url}\n"
        f"Title: {extracted.title}\n"
        f"Site Name: {extracted.site_name or 'N/A'}\n"
        f"Description: {extracted.description or 'N/A'}\n"
        f"Published At: {extracted.published_at or 'N/A'}\n"
        f"Content Type: {extracted.content_type or 'N/A'}\n\n"
        "Below is the extracted page content in Markdown-like form. "
        "Use it as the sole source of truth.\n\n"
        "<source>\n"
        f"{source_excerpt}\n"
        "</source>\n"
    )

    body = build_text_request_body(
        prompt=prompt,
        instructions=DEFAULT_SYSTEM_INSTRUCTIONS,
        model=model,
    )
    result = collect_text_result(
        access_token=auth_state["access_token"],
        account_id=auth_state["account_id"],
        body=body,
        timeout=timeout,
    )
    result["account_id"] = auth_state["account_id"]
    return result


def build_fallback_markdown(extracted: ExtractedPage) -> str:
    blocks = [block for block in extracted.extracted_markdown.split("\n\n") if block.strip()]
    summary = blocks[:8]
    bullets = "\n".join(f"- {normalize_text(item)}" for item in summary[:6])
    return (
        "# 摘要\n"
        f"{bullets or '- 提取成功，但未抽取到足够的文本块。'}\n\n"
        "## 核心信息\n"
        f"{extracted.description or normalize_text(extracted.extracted_markdown[:800])}\n\n"
        "## 实操建议\n"
        "- 当前为 `--skip-llm` 生成的占位整理稿。\n"
        "- 后续使用 OAuth 模型调用后，可得到结构化总结版本。\n"
    )


def resolve_vault_path(value: Optional[str]) -> Path:
    candidate = value or os.environ.get("OBSIDIAN_VAULT_PATH")
    if not candidate:
        raise UrlToObsidianError(
            "Missing vault path. Pass --vault or set OBSIDIAN_VAULT_PATH."
        )
    vault = Path(candidate).expanduser().resolve()
    if not vault.exists() or not vault.is_dir():
        raise UrlToObsidianError(f"Vault path is not a directory: {vault}")
    return vault


def build_note_filename(
    *,
    extracted: ExtractedPage,
    explicit_name: Optional[str],
    timestamp_prefix: bool,
) -> str:
    if explicit_name:
        base = sanitize_filename(explicit_name)
    else:
        base = sanitize_filename(extracted.title)
    if timestamp_prefix:
        return f"{datetime.now().strftime('%Y-%m-%d')} {base}.md"
    return f"{base}.md"


def uniquify_path(path: Path) -> Path:
    if not path.exists():
        return path
    stem = path.stem
    suffix = path.suffix
    parent = path.parent
    index = 2
    while True:
        candidate = parent / f"{stem}-{index}{suffix}"
        if not candidate.exists():
            return candidate
        index += 1


def build_note_markdown(
    *,
    extracted: ExtractedPage,
    model_output: str,
    model_name: str,
    fetched_at: str,
) -> str:
    tags_yaml = "\n".join(f"  - {tag}" for tag in DEFAULT_NOTE_TAGS)
    frontmatter = (
        "---\n"
        f'title: "{yaml_escape(extracted.title)}"\n'
        f'source_url: "{yaml_escape(extracted.source_url)}"\n'
        f'final_url: "{yaml_escape(extracted.final_url)}"\n'
        f'canonical_url: "{yaml_escape(extracted.canonical_url)}"\n'
        f'content_type: "{yaml_escape(extracted.content_type or "unknown")}"\n'
        f'published_at: "{yaml_escape(extracted.published_at or "")}"\n'
        f'fetched_at: "{fetched_at}"\n'
        f'model: "{yaml_escape(model_name)}"\n'
        f"tags:\n{tags_yaml}\n"
        "---\n\n"
    )
    metadata_block = (
        "## Source Metadata\n"
        f"- Title: {extracted.title}\n"
        f"- Source URL: {extracted.source_url}\n"
        f"- Canonical URL: {extracted.canonical_url}\n"
        f"- Site: {extracted.site_name or 'N/A'}\n"
        f"- Published: {extracted.published_at or 'N/A'}\n"
        f"- Fetched At: {fetched_at}\n"
    )
    body = model_output.strip()
    if not body.startswith("# "):
        body = f"# {extracted.title}\n\n{body}"
    return frontmatter + body + "\n\n" + metadata_block + "\n"


def write_note(
    *,
    vault: Path,
    folder: str,
    filename: str,
    content: str,
    overwrite: bool,
) -> Tuple[Path, Path]:
    relative = Path(folder) / filename if folder else Path(filename)
    target = (vault / relative).resolve()
    if vault not in target.parents and target != vault:
        raise UrlToObsidianError(f"Refusing to write outside the vault: {target}")
    target.parent.mkdir(parents=True, exist_ok=True)
    if target.exists() and not overwrite:
        target = uniquify_path(target)
        relative = target.relative_to(vault)
    target.write_text(content, encoding="utf-8")
    return target, relative


def open_note_in_obsidian(vault: Path, relative: Path, absolute: Path, prefer_cli: bool) -> None:
    if prefer_cli:
        obsidian_cli = shutil.which("obsidian")
        if obsidian_cli:
            subprocess.run(
                [
                    obsidian_cli,
                    f"vault={vault.name}",
                    "open",
                    f"path={relative.as_posix()}",
                ],
                check=False,
            )
            return

    uri = "obsidian://open?path=" + quote(str(absolute))
    if sys.platform == "darwin":
        subprocess.run(["open", uri], check=False)
        return
    if sys.platform.startswith("linux"):
        opener = shutil.which("xdg-open")
        if opener:
            subprocess.run([opener, uri], check=False)
        return
    if os.name == "nt":
        os.startfile(uri)  # type: ignore[attr-defined]


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Fetch a webpage, summarize it with Codex OAuth, and write an Obsidian note."
    )
    parser.add_argument("url", help="HTTP(S) URL or file:// URL to ingest")
    parser.add_argument(
        "--vault",
        help="Absolute path to the Obsidian vault. Defaults to OBSIDIAN_VAULT_PATH.",
    )
    parser.add_argument(
        "--folder",
        default=DEFAULT_FOLDER,
        help=f'Folder inside the vault. Default: "{DEFAULT_FOLDER}"',
    )
    parser.add_argument("--note-name", help="Explicit note file name without folder")
    parser.add_argument(
        "--auth-file",
        default=str(AUTH_FILE),
        help=f"Codex auth file path. Default: {AUTH_FILE}",
    )
    parser.add_argument(
        "--model",
        default=DEFAULT_TEXT_MODEL,
        help=f"Text model used through the OAuth-backed Codex responses backend. Default: {DEFAULT_TEXT_MODEL}",
    )
    parser.add_argument(
        "--fetch-timeout",
        type=int,
        default=DEFAULT_FETCH_TIMEOUT,
        help=f"HTTP fetch timeout in seconds. Default: {DEFAULT_FETCH_TIMEOUT}",
    )
    parser.add_argument(
        "--llm-timeout",
        type=int,
        default=DEFAULT_LLM_TIMEOUT,
        help=f"Model request timeout in seconds. Default: {DEFAULT_LLM_TIMEOUT}",
    )
    parser.add_argument(
        "--max-source-chars",
        type=int,
        default=DEFAULT_MAX_SOURCE_CHARS,
        help=f"Maximum extracted source characters to send to the model. Default: {DEFAULT_MAX_SOURCE_CHARS}",
    )
    parser.add_argument(
        "--force-refresh",
        action="store_true",
        help="Refresh the OAuth token before sending the model request",
    )
    parser.add_argument(
        "--overwrite",
        action="store_true",
        help="Overwrite an existing note if the target path already exists",
    )
    parser.add_argument(
        "--skip-llm",
        action="store_true",
        help="Skip the OAuth model call and write a deterministic fallback summary instead",
    )
    parser.add_argument(
        "--timestamp-prefix",
        action="store_true",
        help="Prefix the note file name with YYYY-MM-DD",
    )
    parser.add_argument(
        "--open-note",
        action="store_true",
        help="Open the resulting note in Obsidian after writing",
    )
    parser.add_argument(
        "--prefer-cli",
        action="store_true",
        help="When --open-note is set, prefer the Obsidian CLI if it is installed",
    )
    parser.add_argument(
        "--dump-extract",
        help="Optional path to save the extracted source markdown for debugging",
    )
    parser.add_argument(
        "--dump-response-json",
        help="Optional path to save the completed model response JSON",
    )
    return parser


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()

    fetched_at = now_utc_iso()
    vault = resolve_vault_path(args.vault)
    extracted = extract_page(args.url, args.fetch_timeout)

    if args.dump_extract:
        dump_path = Path(args.dump_extract).expanduser()
        dump_path.parent.mkdir(parents=True, exist_ok=True)
        dump_path.write_text(extracted.extracted_markdown, encoding="utf-8")

    if args.skip_llm:
        model_output = build_fallback_markdown(extracted)
        response = None
    else:
        result = summarize_page_with_oauth(
            extracted,
            auth_file=Path(args.auth_file),
            model=args.model,
            timeout=args.llm_timeout,
            max_source_chars=args.max_source_chars,
            force_refresh=args.force_refresh,
        )
        model_output = result["assistant_text"]
        response = result["response"]

        if args.dump_response_json:
            dump_path = Path(args.dump_response_json).expanduser()
            dump_path.parent.mkdir(parents=True, exist_ok=True)
            dump_path.write_text(
                json.dumps(response, ensure_ascii=False, indent=2) + "\n",
                encoding="utf-8",
            )

    filename = build_note_filename(
        extracted=extracted,
        explicit_name=args.note_name,
        timestamp_prefix=args.timestamp_prefix,
    )
    note_text = build_note_markdown(
        extracted=extracted,
        model_output=model_output,
        model_name=args.model if not args.skip_llm else "skip-llm",
        fetched_at=fetched_at,
    )
    absolute_path, relative_path = write_note(
        vault=vault,
        folder=args.folder,
        filename=filename,
        content=note_text,
        overwrite=args.overwrite,
    )

    if args.open_note:
        open_note_in_obsidian(
            vault=vault,
            relative=relative_path,
            absolute=absolute_path,
            prefer_cli=args.prefer_cli,
        )

    print(f"Source URL   : {extracted.source_url}")
    print(f"Final URL    : {extracted.final_url}")
    print(f"Title        : {extracted.title}")
    print(f"Vault        : {vault}")
    print(f"Note         : {absolute_path}")
    print(f"Model        : {args.model if not args.skip_llm else 'skip-llm'}")
    print(f"Extract chars: {len(extracted.extracted_markdown)}")


if __name__ == "__main__":
    try:
        main()
    except (UrlToObsidianError, CodexOAuthError, requests.RequestException) as exc:
        print(f"ERROR: {exc}")
        raise SystemExit(1)
