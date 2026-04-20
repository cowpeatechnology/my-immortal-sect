#!/usr/bin/env python3
"""Dedicated-browser Google search worker for the project knowledge harvest flow."""

from __future__ import annotations

import argparse
import asyncio
import json
import logging
import re
import sys
import time
from pathlib import Path
from typing import Any, Dict, List, Optional, Sequence, Tuple
from urllib.parse import parse_qs, urlencode, urlparse, urlunparse

from browser_cdp import BrowserCdpClient, BrowserCdpError, TargetSession, fetch_browser_ws_url, normalize_url
from browser_content_filters import probe_url, skip_reason_for_url
from google_browser_queue import (
    append_candidate_batch,
    claim_next_query,
    load_json,
    mark_query,
    now_iso,
    write_heartbeat,
)
from url_to_obsidian import (
    AUTH_FILE,
    DEFAULT_LLM_TIMEOUT,
    DEFAULT_TEXT_MODEL,
    build_text_request_body,
    collect_text_result,
    ensure_fresh_tokens,
    load_codex_auth_file,
)


DEFAULT_BROWSER_VERSION_URL = "http://127.0.0.1:9333/json/version"
DEFAULT_RESULTS_MIN = 5
DEFAULT_RESULTS_MAX = 8
DEFAULT_MAX_SEARCH_PAGES = 2
DEFAULT_PREVIEW_LIMIT = 12
DEFAULT_NAV_SETTLE_SECONDS = 2.0
DEFAULT_PAGE_READY_TIMEOUT = 12
DEFAULT_IDLE_SECONDS = 120
DEFAULT_PREVIEW_PROBE_TIMEOUT = 10

SEARCH_EXTRACTION_JS = r"""
(() => {
  const normalize = (value) => (value || '').replace(/\s+/g, ' ').trim();
  const isGoogleHost = (hostname) => /(^|\.)google\./.test(hostname) || hostname === 'webcache.googleusercontent.com';
  const rows = [];
  const seen = new Set();

  for (const anchor of Array.from(document.querySelectorAll('a[href]'))) {
    const href = anchor.href || '';
    if (!href.startsWith('http')) continue;
    let hostname = '';
    try {
      hostname = new URL(href).hostname || '';
    } catch (error) {
      continue;
    }
    if (isGoogleHost(hostname)) continue;
    const text = normalize(anchor.innerText || anchor.textContent || '');
    if (text.length < 8) continue;

    const container =
      anchor.closest('div.g, div.tF2Cxc, div.MjjYud, div[data-hveid], div[data-snc]') ||
      anchor.closest('div');
    const contextText = normalize(container ? container.innerText || '' : '');
    const snippet = normalize(contextText.replace(text, '')).slice(0, 320);
    const key = href.replace(/#.*$/, '').replace(/\/$/, '');
    if (seen.has(key)) continue;
    seen.add(key);

    rows.push({
      href: key,
      hostname,
      title: text,
      snippet,
      contextText: contextText.slice(0, 420),
    });
  }

  return {
    title: document.title,
    href: location.href,
    bodyPreview: (document.body?.innerText || '').slice(0, 1600),
    results: rows.slice(0, 40),
  };
})()
"""

PREVIEW_EXTRACTION_JS = r"""
(() => {
  const normalize = (value) => (value || '').replace(/\s+/g, ' ').trim();
  const bodyText = normalize(document.body?.innerText || '');
  const metaDescription = document.querySelector('meta[name="description"]')?.content ||
    document.querySelector('meta[property="og:description"]')?.content || '';
  return {
    href: location.href,
    title: document.title,
    ready: document.readyState,
    metaDescription: normalize(metaDescription),
    bodyTextExcerpt: bodyText.slice(0, 2400),
    bodyTextLength: bodyText.length,
  };
})()
"""

SELECTION_SYSTEM_INSTRUCTIONS = """You are triaging web search results for a long-lived research wiki for a Chinese xianxia sect-management sim project.

Return strict JSON only. No markdown fences, no commentary.

Selection rules:
- Choose pages that are distinct, readable, and likely to add durable knowledge.
- Prefer official docs, engine/manual pages, source-code issues, concrete technical forum threads, and substantial articles.
- Reject homepages, tag/index pages, reposts, mirrors, thin content, obvious SEO sludge, login walls, download landing pages, and short video-only pages.
- Avoid selecting multiple near-duplicates that say the same thing with minor wording changes.
- If a result looks useful but only as a secondary reference, it can be rejected.

JSON shape:
{
  "selected": [
    {"index": 0, "reason": "short reason"}
  ],
  "rejected": [
    {"index": 1, "reason": "short reason"}
  ]
}
"""

LOW_VALUE_TEXT_PATTERNS = (
    "登录",
    "注册",
    "subscribe",
    "watch video",
    "app download",
)


class SearchWorkerError(RuntimeError):
    """Raised when the search worker cannot complete its loop."""


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run the dedicated-browser Google search worker.")
    parser.add_argument("--run-file", required=True)
    parser.add_argument("--worker-id", required=True)
    parser.add_argument("--browser-version-url", default=DEFAULT_BROWSER_VERSION_URL)
    parser.add_argument("--managed-state-file", default="")
    parser.add_argument("--results-min", type=int, default=DEFAULT_RESULTS_MIN)
    parser.add_argument("--results-max", type=int, default=DEFAULT_RESULTS_MAX)
    parser.add_argument("--max-search-pages", type=int, default=DEFAULT_MAX_SEARCH_PAGES)
    parser.add_argument("--preview-limit", type=int, default=DEFAULT_PREVIEW_LIMIT)
    parser.add_argument("--nav-settle-seconds", type=float, default=DEFAULT_NAV_SETTLE_SECONDS)
    parser.add_argument("--page-ready-timeout", type=int, default=DEFAULT_PAGE_READY_TIMEOUT)
    parser.add_argument("--idle-seconds", type=int, default=DEFAULT_IDLE_SECONDS)
    parser.add_argument("--model", default=DEFAULT_TEXT_MODEL)
    parser.add_argument("--auth-file", default=str(AUTH_FILE))
    parser.add_argument("--llm-timeout", type=int, default=DEFAULT_LLM_TIMEOUT)
    parser.add_argument("--force-refresh", action="store_true")
    parser.add_argument("--skip-llm-review", action="store_true")
    parser.add_argument("--log-level", default="INFO")
    return parser.parse_args()


def configure_logging(level: str) -> None:
    logging.basicConfig(
        level=getattr(logging, level.upper(), logging.INFO),
        format="%(asctime)s %(levelname)s %(message)s",
    )


def google_url_for_page(base_url: str, page_index: int) -> str:
    if page_index <= 0:
        return base_url
    parsed = urlparse(base_url)
    query = parse_qs(parsed.query, keep_blank_values=True)
    query["start"] = [str(page_index * 10)]
    query.setdefault("num", ["10"])
    return urlunparse(parsed._replace(query=urlencode(query, doseq=True)))


def state_file_for_run(run_file: Path, explicit: str, worker_id: str) -> Path:
    if explicit:
        return Path(explicit).expanduser().resolve()
    return (run_file.parent / "control" / f"{worker_id}-browser-state.json").resolve()


def load_state(path: Path) -> Dict[str, Any]:
    if not path.exists():
        return {}
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        return {}


def save_state(path: Path, payload: Dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def normalize_candidate_url(url: str) -> str:
    cleaned = normalize_url(url)
    if not cleaned:
        return ""
    parsed = urlparse(cleaned)
    if parsed.scheme not in {"http", "https"}:
        return ""
    return cleaned


def candidate_key(url: str, title: str) -> str:
    normalized_title = re.sub(r"\s+", " ", (title or "").strip().lower())
    return f"{normalize_candidate_url(url)}::{normalized_title}"


def looks_low_value(card: Dict[str, Any]) -> bool:
    host = str(card.get("hostname") or "")
    if skip_reason_for_url(str(card.get("url") or ""), hostname=host):
        return True
    text = " ".join(
        [
            str(card.get("title") or ""),
            str(card.get("snippet") or ""),
            str(card.get("preview_title") or ""),
            str(card.get("preview_excerpt") or ""),
        ]
    ).lower()
    return any(pattern in text for pattern in LOW_VALUE_TEXT_PATTERNS)


def extract_json_object(text: str) -> Dict[str, Any]:
    candidate = text.strip()
    if candidate.startswith("```"):
        candidate = re.sub(r"^```(?:json)?", "", candidate).strip()
        candidate = re.sub(r"```$", "", candidate).strip()
    try:
        payload = json.loads(candidate)
        if isinstance(payload, dict):
            return payload
    except json.JSONDecodeError:
        pass
    start = candidate.find("{")
    end = candidate.rfind("}")
    if start >= 0 and end > start:
        payload = json.loads(candidate[start : end + 1])
        if isinstance(payload, dict):
            return payload
    raise SearchWorkerError(f"Could not parse JSON from model output: {text[:400]!r}")


def fallback_select(cards: Sequence[Dict[str, Any]], *, results_min: int, results_max: int) -> List[Dict[str, Any]]:
    selected: List[Dict[str, Any]] = []
    seen_hosts: set[str] = set()
    for card in cards:
        if looks_low_value(card):
            continue
        if int(card.get("preview_text_length") or 0) < 160:
            continue
        host = str(card.get("hostname") or "")
        if host and host in seen_hosts and len(selected) >= results_min:
            continue
        if host:
            seen_hosts.add(host)
        selected.append({"index": int(card["index"]), "reason": "fallback-heuristic"})
        if len(selected) >= results_max:
            break
    return selected


def review_cards_with_model(
    *,
    cards: Sequence[Dict[str, Any]],
    query_item: Dict[str, Any],
    model: str,
    auth_file: Path,
    llm_timeout: int,
    force_refresh: bool,
    results_min: int,
    results_max: int,
) -> List[Dict[str, Any]]:
    auth_state = load_codex_auth_file(auth_file.expanduser())
    auth_state = ensure_fresh_tokens(auth_state, force_refresh=force_refresh)

    serialized_cards = []
    for card in cards:
        serialized_cards.append(
            {
                "index": card["index"],
                "url": card["url"],
                "hostname": card["hostname"],
                "search_title": card["title"],
                "search_snippet": card["snippet"],
                "preview_title": card.get("preview_title", ""),
                "preview_excerpt": card.get("preview_excerpt", ""),
                "preview_text_length": card.get("preview_text_length", 0),
            }
        )

    prompt = (
        f"Query ID: {query_item['query_id']}\n"
        f"Query: {query_item['query']}\n"
        f"Language: {query_item.get('language', '')}\n"
        f"Category: {query_item.get('category', '')}\n"
        f"Select between {results_min} and {results_max} distinct URLs when enough worthwhile pages exist.\n\n"
        "Candidate cards JSON:\n"
        f"{json.dumps(serialized_cards, ensure_ascii=False, indent=2)}\n"
    )

    body = build_text_request_body(prompt=prompt, instructions=SELECTION_SYSTEM_INSTRUCTIONS, model=model)
    result = collect_text_result(
        access_token=auth_state["access_token"],
        account_id=auth_state["account_id"],
        body=body,
        timeout=llm_timeout,
    )
    payload = extract_json_object(result["assistant_text"])
    selected = payload.get("selected", [])
    if not isinstance(selected, list):
        raise SearchWorkerError(f"Model output missing selected list: {payload!r}")
    normalized: List[Dict[str, Any]] = []
    for item in selected:
        if not isinstance(item, dict):
            continue
        try:
            index = int(item["index"])
        except Exception:
            continue
        normalized.append({"index": index, "reason": str(item.get("reason") or "model-selected")})
    return normalized


async def ensure_managed_tabs(client: BrowserCdpClient, state_path: Path, worker_id: str) -> Tuple[TargetSession, TargetSession]:
    state = load_state(state_path)
    page_targets = await client.list_page_targets()
    target_map = {target.target_id: target for target in page_targets}

    search_target_id = state.get("search_target_id") if isinstance(state, dict) else None
    preview_target_id = state.get("preview_target_id") if isinstance(state, dict) else None

    if not isinstance(search_target_id, str) or search_target_id not in target_map:
        search_target_id = await client.create_target("about:blank", background=True)
    if not isinstance(preview_target_id, str) or preview_target_id not in target_map or preview_target_id == search_target_id:
        preview_target_id = await client.create_target("about:blank", background=True)

    save_state(
        state_path,
        {
            "worker_id": worker_id,
            "updated_at": now_iso(),
            "search_target_id": search_target_id,
            "preview_target_id": preview_target_id,
        },
    )

    search_session = await client.attach(search_target_id)
    preview_session = await client.attach(preview_target_id)
    await search_session.enable_basics()
    await preview_session.enable_basics()
    return search_session, preview_session


async def collect_search_results(search_session: TargetSession) -> Dict[str, Any]:
    payload = await search_session.evaluate(SEARCH_EXTRACTION_JS, timeout_seconds=10)
    if not isinstance(payload, dict):
        raise SearchWorkerError(f"Unexpected search extraction payload: {payload!r}")
    return payload


async def preview_candidate(
    preview_session: TargetSession,
    candidate: Dict[str, Any],
    *,
    settle_seconds: float,
    ready_timeout_seconds: int,
) -> Optional[Dict[str, Any]]:
    url = candidate["url"]
    try:
        await preview_session.navigate(url, settle_seconds=settle_seconds, ready_timeout_seconds=ready_timeout_seconds)
        payload = await preview_session.evaluate(PREVIEW_EXTRACTION_JS, timeout_seconds=10)
    except (BrowserCdpError, asyncio.TimeoutError) as exc:
        logging.warning("Preview failed for %s: %s", url, exc)
        return None
    if not isinstance(payload, dict):
        return None
    return {
        "preview_url": str(payload.get("href") or url),
        "preview_title": str(payload.get("title") or ""),
        "preview_excerpt": str(payload.get("bodyTextExcerpt") or payload.get("metaDescription") or ""),
        "preview_text_length": int(payload.get("bodyTextLength") or 0),
    }


def merge_preview(candidate: Dict[str, Any], preview: Optional[Dict[str, Any]]) -> Dict[str, Any]:
    merged = dict(candidate)
    if preview:
        merged.update(preview)
    else:
        merged.setdefault("preview_url", merged["url"])
        merged.setdefault("preview_title", merged["title"])
        merged.setdefault("preview_excerpt", merged.get("snippet", ""))
        merged.setdefault("preview_text_length", len(str(merged.get("snippet", ""))))
    return merged


def build_batch_rows(
    *,
    selected: Sequence[Dict[str, Any]],
    cards_by_index: Dict[int, Dict[str, Any]],
    query_item: Dict[str, Any],
    worker_id: str,
) -> List[Dict[str, Any]]:
    rows: List[Dict[str, Any]] = []
    seen_urls: set[str] = set()
    for item in selected:
        index = int(item["index"])
        card = cards_by_index.get(index)
        if not card:
            continue
        preferred_url = str(card.get("preview_url") or card["url"])
        url = normalize_candidate_url(preferred_url)
        skip_reason = skip_reason_for_url(url, hostname=str(urlparse(url).hostname or ""))
        if skip_reason:
            continue
        if not url or url in seen_urls:
            continue
        seen_urls.add(url)
        rows.append(
            {
                "seed_id": query_item["seed_id"],
                "query_id": query_item["query_id"],
                "query": query_item["query"],
                "title": card.get("preview_title") or card.get("title") or url,
                "url": url,
                "snippet": card.get("preview_excerpt") or card.get("snippet") or "",
                "discovered_by": worker_id,
            }
        )
    return rows


async def process_query(
    *,
    client: BrowserCdpClient,
    search_session: TargetSession,
    preview_session: TargetSession,
    query_item: Dict[str, Any],
    args: argparse.Namespace,
    search_heartbeat: Path,
    run_file: Path,
) -> Dict[str, Any]:
    results_seen: set[str] = set()
    previewed_cards: List[Dict[str, Any]] = []

    for page_index in range(args.max_search_pages):
        search_url = google_url_for_page(query_item["google_url"], page_index)
        write_heartbeat(
            path=search_heartbeat,
            role="search-agent",
            worker_id=args.worker_id,
            stage="searching",
            current_query_id=query_item["query_id"],
            current_url=search_url,
            last_event=f"loading search page {page_index + 1}",
        )
        await search_session.navigate(
            search_url,
            settle_seconds=args.nav_settle_seconds,
            ready_timeout_seconds=args.page_ready_timeout,
        )
        search_payload = await collect_search_results(search_session)
        raw_results = search_payload.get("results", [])
        if not isinstance(raw_results, list):
            raw_results = []

        for result in raw_results:
            if not isinstance(result, dict):
                continue
            url = normalize_candidate_url(str(result.get("href") or ""))
            title = str(result.get("title") or "")
            if not url or not title:
                continue
            hostname = str(result.get("hostname") or urlparse(url).hostname or "")
            skip_reason = skip_reason_for_url(url, hostname=hostname)
            if skip_reason:
                logging.info("Skipping candidate before preview %s reason=%s", url, skip_reason)
                continue
            key = candidate_key(url, title)
            if key in results_seen:
                continue
            results_seen.add(key)
            previewed_cards.append(
                {
                    "index": len(previewed_cards),
                    "url": url,
                    "hostname": hostname,
                    "title": title,
                    "snippet": str(result.get("snippet") or result.get("contextText") or ""),
                }
            )
            if len(previewed_cards) >= args.preview_limit:
                break
        if len(previewed_cards) >= args.preview_limit:
            break

    enriched_cards: List[Dict[str, Any]] = []
    for card in previewed_cards:
        probe = await asyncio.to_thread(
            probe_url,
            card["url"],
            timeout_seconds=DEFAULT_PREVIEW_PROBE_TIMEOUT,
        )
        if probe.skip_reason:
            logging.info(
                "Skipping candidate before browser preview %s reason=%s",
                probe.final_url or card["url"],
                probe.skip_reason,
            )
            continue
        candidate_for_preview = dict(card)
        if probe.final_url:
            candidate_for_preview["url"] = probe.final_url
            candidate_for_preview["hostname"] = str(urlparse(probe.final_url).hostname or candidate_for_preview["hostname"])
        write_heartbeat(
            path=search_heartbeat,
            role="search-agent",
            worker_id=args.worker_id,
            stage="previewing",
            current_query_id=query_item["query_id"],
            current_url=candidate_for_preview["url"],
            last_event=f"previewing candidate {card['index'] + 1}/{len(previewed_cards)}",
        )
        preview = await preview_candidate(
            preview_session,
            candidate_for_preview,
            settle_seconds=args.nav_settle_seconds,
            ready_timeout_seconds=args.page_ready_timeout,
        )
        merged = merge_preview(candidate_for_preview, preview)
        merged_skip_reason = skip_reason_for_url(
            str(merged.get("preview_url") or merged["url"]),
            hostname=str(urlparse(str(merged.get("preview_url") or merged["url"])).hostname or ""),
        )
        if merged_skip_reason:
            logging.info(
                "Skipping candidate after preview %s reason=%s",
                str(merged.get("preview_url") or merged["url"]),
                merged_skip_reason,
            )
            continue
        enriched_cards.append(merged)

    write_heartbeat(
        path=search_heartbeat,
        role="search-agent",
        worker_id=args.worker_id,
        stage="reviewing",
        current_query_id=query_item["query_id"],
        current_url=query_item["google_url"],
        last_event=f"reviewing {len(enriched_cards)} candidates",
    )

    if args.skip_llm_review:
        selected = fallback_select(enriched_cards, results_min=args.results_min, results_max=args.results_max)
    else:
        try:
            selected = review_cards_with_model(
                cards=enriched_cards,
                query_item=query_item,
                model=args.model,
                auth_file=Path(args.auth_file),
                llm_timeout=args.llm_timeout,
                force_refresh=args.force_refresh,
                results_min=args.results_min,
                results_max=args.results_max,
            )
        except Exception as exc:
            logging.warning("Model review failed for %s: %s; falling back to heuristics.", query_item["query_id"], exc)
            selected = fallback_select(enriched_cards, results_min=args.results_min, results_max=args.results_max)

    cards_by_index = {int(card["index"]): card for card in enriched_cards}
    rows = build_batch_rows(selected=selected, cards_by_index=cards_by_index, query_item=query_item, worker_id=args.worker_id)
    batch_result = append_candidate_batch(run_file=run_file, rows=rows)
    return {
        "previewed_count": len(enriched_cards),
        "selected_count": len(rows),
        "saved_count": int(batch_result.get("saved", 0)),
        "skipped_count": int(batch_result.get("skipped", 0)),
        "selected_rows": rows,
    }


async def async_main(args: argparse.Namespace) -> int:
    run_file = Path(args.run_file).resolve()
    run_payload = load_json(run_file)
    search_heartbeat = Path(run_payload["outputs"]["search_heartbeat"]).resolve()
    browser_ws_url = fetch_browser_ws_url(args.browser_version_url)
    managed_state_file = state_file_for_run(run_file, args.managed_state_file, args.worker_id)

    async with BrowserCdpClient(browser_ws_url) as client:
        search_session, preview_session = await ensure_managed_tabs(client, managed_state_file, args.worker_id)
        while True:
            try:
                query_item = claim_next_query(run_file=run_file, worker_id=args.worker_id)
            except SystemExit:
                write_heartbeat(
                    path=search_heartbeat,
                    role="search-agent",
                    worker_id=args.worker_id,
                    stage="idle",
                    current_query_id="",
                    current_url="",
                    last_event="no pending queries",
                )
                logging.info("No pending queries left.")
                return 0

            logging.info("Claimed query %s", query_item["query_id"])
            try:
                result = await process_query(
                    client=client,
                    search_session=search_session,
                    preview_session=preview_session,
                    query_item=query_item,
                    args=args,
                    search_heartbeat=search_heartbeat,
                    run_file=run_file,
                )
                next_state = "completed" if result["saved_count"] > 0 or result["selected_count"] > 0 else "skipped"
                mark_query(
                    run_file=run_file,
                    query_id=query_item["query_id"],
                    state=next_state,
                    worker_id=args.worker_id,
                )
                write_heartbeat(
                    path=search_heartbeat,
                    role="search-agent",
                    worker_id=args.worker_id,
                    stage="searching",
                    current_query_id=query_item["query_id"],
                    current_url=query_item["google_url"],
                    last_event=(
                        f"completed query; previewed={result['previewed_count']} "
                        f"selected={result['selected_count']} saved={result['saved_count']}"
                    ),
                )
                logging.info(
                    "Completed %s: previewed=%s selected=%s saved=%s",
                    query_item["query_id"],
                    result["previewed_count"],
                    result["selected_count"],
                    result["saved_count"],
                )
            except Exception as exc:
                mark_query(
                    run_file=run_file,
                    query_id=query_item["query_id"],
                    state="skipped",
                    worker_id=args.worker_id,
                )
                write_heartbeat(
                    path=search_heartbeat,
                    role="search-agent",
                    worker_id=args.worker_id,
                    stage="searching",
                    current_query_id=query_item["query_id"],
                    current_url=query_item["google_url"],
                    last_event=f"query failed: {type(exc).__name__}: {exc}",
                )
                logging.exception("Query %s failed", query_item["query_id"])
                await asyncio.sleep(1)


def main() -> int:
    args = parse_args()
    configure_logging(args.log_level)
    try:
        return asyncio.run(async_main(args))
    except KeyboardInterrupt:
        return 130
    except Exception as exc:
        logging.exception("Search worker crashed")
        print(json.dumps({"ok": False, "error": type(exc).__name__, "message": str(exc)}, ensure_ascii=False, indent=2))
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
