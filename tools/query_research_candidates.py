#!/usr/bin/env python3
"""Query search engines for seed topics and append candidate article manifests."""

from __future__ import annotations

import argparse
import base64
import json
import re
import socket
import tempfile
import time
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Dict, Iterable, List, Sequence, Tuple
from urllib.parse import parse_qs, unquote, urlencode, urlparse

import lxml.html
import requests


DEFAULT_QUEUE = "runtime/knowledge-harvest/2026-04-17-sect-sim-cocos/seed-queue.json"
DEFAULT_TIMEOUT = 30
DEFAULT_MAX_RESULTS_PER_QUERY = 20
DEFAULT_LEASE_SECONDS = 1800
DEFAULT_SLEEP_SECONDS = 1.0
DEFAULT_USER_AGENT = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
    "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"
)
ASCII_TOKEN_RE = re.compile(r"[A-Za-z][A-Za-z0-9_-]{3,}")
COMMON_ANCHOR_TOKENS = {
    "best",
    "cocos",
    "community",
    "creator",
    "design",
    "development",
    "forum",
    "game",
    "games",
    "general",
    "guide",
    "mini",
    "mobile",
    "online",
    "performance",
    "practice",
    "practices",
    "site",
    "tutorial",
}


class QueryWorkerError(RuntimeError):
    """Raised when the query worker cannot complete its assigned work."""


def now_utc() -> datetime:
    return datetime.now(timezone.utc)


def now_iso() -> str:
    return now_utc().replace(microsecond=0).isoformat()


def resolve_from_queue(queue_path: Path, relative_or_absolute: str) -> Path:
    path = Path(relative_or_absolute)
    if path.is_absolute():
        return path
    return (queue_path.parent / path).resolve()


def ensure_parent(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def atomic_write_json(path: Path, payload: Dict) -> None:
    ensure_parent(path)
    with tempfile.NamedTemporaryFile("w", encoding="utf-8", dir=path.parent, delete=False) as handle:
        json.dump(payload, handle, ensure_ascii=False, indent=2)
        handle.write("\n")
        temp_name = handle.name
    Path(temp_name).replace(path)


def append_jsonl(path: Path, records: Iterable[Dict]) -> int:
    ensure_parent(path)
    count = 0
    with path.open("a", encoding="utf-8") as handle:
        for record in records:
            handle.write(json.dumps(record, ensure_ascii=False) + "\n")
            count += 1
    return count


def load_queue(path: Path) -> Dict:
    return json.loads(path.read_text(encoding="utf-8"))


def canonicalize_result_url(url: str) -> str:
    if not url:
        return ""
    parsed = urlparse(url)
    if parsed.netloc.endswith("duckduckgo.com") and parsed.path.startswith("/l/"):
        query = parse_qs(parsed.query)
        uddg = query.get("uddg", [])
        if uddg:
            return unquote(uddg[0])
    if parsed.netloc.endswith("bing.com") and parsed.path.startswith("/ck/a"):
        query = parse_qs(parsed.query)
        raw_u = query.get("u", [])
        if raw_u:
            encoded = raw_u[0]
            if encoded.startswith("a1"):
                encoded = encoded[2:]
            padding = "=" * (-len(encoded) % 4)
            try:
                decoded = base64.urlsafe_b64decode(encoded + padding).decode("utf-8")
                if decoded.startswith(("http://", "https://")):
                    return decoded
            except Exception:  # noqa: BLE001
                pass
    return url


def extract_anchor_tokens(keyword: str) -> List[str]:
    tokens: List[str] = []
    for token in ASCII_TOKEN_RE.findall(keyword):
        lowered = token.lower()
        if lowered not in tokens:
            tokens.append(lowered)
    return tokens


def passes_anchor_filter(*, keyword: str, title: str, snippet: str, url: str) -> bool:
    anchor_tokens = extract_anchor_tokens(keyword)
    if not anchor_tokens:
        return True
    haystack = f"{title}\n{snippet}\n{url}".lower()
    matched_tokens = [token for token in anchor_tokens if token in haystack]
    if not matched_tokens:
        return False
    distinctive_tokens = [token for token in anchor_tokens if token not in COMMON_ANCHOR_TOKENS]
    if distinctive_tokens:
        return any(token in haystack for token in distinctive_tokens)
    return True


def normalize_domain(domain: str) -> str:
    return domain.lower().lstrip(".")


def domain_matches(domain: str, patterns: Sequence[str]) -> bool:
    normalized = normalize_domain(domain)
    for pattern in patterns:
        candidate = normalize_domain(pattern)
        if normalized == candidate or normalized.endswith(f".{candidate}"):
            return True
    return False


def is_index_like_url(url: str) -> bool:
    parsed = urlparse(url)
    path = parsed.path or ""
    query = parsed.query or ""
    if path in {"", "/"} and not query:
        return True
    if path.lower() in {"/top", "/latest"}:
        return True
    lowered = path.lower()
    if lowered.endswith("/forum") or lowered.endswith("/forums"):
        return True
    if parsed.netloc.endswith("forum.cocos.org") and lowered.startswith("/c/"):
        return True
    if "forumdisplay" in query:
        return True
    return False


def load_seen_dedupe_keys(path: Path) -> set[str]:
    if not path.exists():
        return set()
    seen: set[str] = set()
    with path.open("r", encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if not line:
                continue
            try:
                payload = json.loads(line)
            except json.JSONDecodeError:
                continue
            dedupe_key = payload.get("dedupe_key")
            if dedupe_key:
                seen.add(dedupe_key)
    return seen


def search_duckduckgo(query: str, timeout: int, max_results: int) -> List[Dict]:
    response = requests.get(
        "https://html.duckduckgo.com/html/",
        params={"q": query},
        headers={"User-Agent": DEFAULT_USER_AGENT},
        timeout=timeout,
    )
    response.raise_for_status()
    doc = lxml.html.fromstring(response.text, base_url=str(response.url))

    items: List[Dict] = []
    nodes = doc.xpath(
        "//div[contains(@class,'result')]"
        "[.//a[contains(@class,'result__a')]]"
    )
    for node in nodes:
        anchor = node.xpath(".//a[contains(@class,'result__a')][1]")
        if not anchor:
            continue
        anchor = anchor[0]
        title = " ".join(anchor.itertext()).strip()
        url = canonicalize_result_url(anchor.get("href", "").strip())
        if not title or not url:
            continue
        snippet_nodes = node.xpath(".//*[contains(@class,'result__snippet')][1]")
        snippet = ""
        if snippet_nodes:
            snippet = " ".join(snippet_nodes[0].itertext()).strip()
        items.append({"title": title, "url": url, "snippet": snippet})
        if len(items) >= max_results:
            break
    return items


def search_bing(query: str, timeout: int, max_results: int) -> List[Dict]:
    response = requests.get(
        "https://www.bing.com/search",
        params={"q": query},
        headers={"User-Agent": DEFAULT_USER_AGENT},
        timeout=timeout,
    )
    response.raise_for_status()
    doc = lxml.html.fromstring(response.text, base_url=str(response.url))

    items: List[Dict] = []
    nodes = doc.xpath("//li[contains(@class,'b_algo')]")
    for node in nodes:
        anchor = node.xpath(".//h2/a[1]")
        if not anchor:
            continue
        anchor = anchor[0]
        title = " ".join(anchor.itertext()).strip()
        url = anchor.get("href", "").strip()
        if not title or not url:
            continue
        snippet_nodes = node.xpath(".//*[contains(@class,'b_caption')]//p[1]")
        snippet = ""
        if snippet_nodes:
            snippet = " ".join(snippet_nodes[0].itertext()).strip()
        items.append({"title": title, "url": url, "snippet": snippet})
        if len(items) >= max_results:
            break
    return items


BUILTIN_SEARCHERS = (
    search_duckduckgo,
    search_bing,
)


def write_claim(
    *,
    path: Path,
    protocol_version: str,
    run_id: str,
    worker_id: str,
    seed_id: str,
    candidate_manifest: Path,
    heartbeat_file: Path,
    lease_seconds: int,
    state: str,
) -> None:
    claimed_at = now_utc()
    payload = {
        "protocol_version": protocol_version,
        "run_id": run_id,
        "claim_id": f"{seed_id}:{worker_id}",
        "seed_id": seed_id,
        "worker_id": worker_id,
        "state": state,
        "claimed_at": claimed_at.replace(microsecond=0).isoformat(),
        "lease_expires_at": (claimed_at + timedelta(seconds=lease_seconds)).replace(microsecond=0).isoformat(),
        "candidate_manifest": str(candidate_manifest),
        "heartbeat_file": str(heartbeat_file),
        "hostname": socket.gethostname(),
    }
    atomic_write_json(path, payload)


def write_heartbeat(
    *,
    path: Path,
    protocol_version: str,
    run_id: str,
    worker_id: str,
    seed_id: str,
    stage: str,
    current_query_id: str,
    current_url: str,
    processed_count: int,
    saved_count: int,
    skipped_count: int,
    error_count: int,
    last_error: str,
    last_meaningful_event: str,
) -> None:
    timestamp = now_iso()
    payload = {
        "protocol_version": protocol_version,
        "run_id": run_id,
        "worker_id": worker_id,
        "seed_id": seed_id,
        "stage": stage,
        "updated_at": timestamp,
        "last_progress_at": timestamp,
        "processed_count": processed_count,
        "saved_count": saved_count,
        "skipped_count": skipped_count,
        "error_count": error_count,
        "current_query_id": current_query_id,
        "current_url": current_url,
        "last_error": last_error,
        "last_meaningful_event": last_meaningful_event,
    }
    atomic_write_json(path, payload)


def build_candidate_records(
    *,
    protocol_version: str,
    run_id: str,
    seed_id: str,
    keyword: str,
    prefer_domains: Sequence[str],
    query_strategy: Dict,
    query_id: str,
    query: str,
    language: str,
    results: Sequence[Dict],
    seen_keys: set[str],
) -> Tuple[List[Dict], int]:
    records: List[Dict] = []
    skipped = 0
    include_domains = query_strategy.get("include_domains", [])
    exclude_domains = query_strategy.get("exclude_domains", [])
    domain_mode = query_strategy.get("domain_mode", "allow_any")
    skip_index_pages = bool(query_strategy.get("skip_index_pages", True))
    for result in results:
        dedupe_key = canonicalize_result_url(result["url"]).rstrip("/")
        if not dedupe_key:
            skipped += 1
            continue
        source_domain = urlparse(dedupe_key).netloc.lower()
        if skip_index_pages and is_index_like_url(dedupe_key):
            skipped += 1
            continue
        if include_domains and not domain_matches(source_domain, include_domains):
            skipped += 1
            continue
        if exclude_domains and domain_matches(source_domain, exclude_domains):
            skipped += 1
            continue
        if domain_mode == "prefer_only" and prefer_domains and not domain_matches(source_domain, prefer_domains):
            skipped += 1
            continue
        if not passes_anchor_filter(
            keyword=keyword,
            title=result["title"],
            snippet=result.get("snippet", ""),
            url=dedupe_key,
        ):
            skipped += 1
            continue
        if dedupe_key in seen_keys:
            skipped += 1
            continue
        seen_keys.add(dedupe_key)
        records.append(
            {
                "protocol_version": protocol_version,
                "run_id": run_id,
                "seed_id": seed_id,
                "query_id": query_id,
                "query": query,
                "title": result["title"],
                "url": dedupe_key,
                "published_at": "",
                "source_domain": source_domain,
                "snippet": result.get("snippet", ""),
                "language": language,
                "dedupe_key": dedupe_key,
                "source_tier": "preferred" if prefer_domains and domain_matches(source_domain, prefer_domains) else "general",
                "discovered_at": now_iso(),
            }
        )
    return records, skipped


def run_seed(
    *,
    queue_path: Path,
    queue_payload: Dict,
    item: Dict,
    worker_id: str,
    timeout: int,
    max_results_per_query: int,
    lease_seconds: int,
    sleep_seconds: float,
) -> Dict:
    protocol_version = queue_payload["protocol_version"]
    run_id = queue_payload["run_id"]
    seed_id = item.get("seed_id", item["id"])

    outputs = item["outputs"]
    query_strategy = item.get("query_strategy", {})
    candidate_manifest = resolve_from_queue(queue_path, outputs["candidate_manifest"])
    claim_file = resolve_from_queue(queue_path, outputs["claim_file"])
    heartbeat_file = resolve_from_queue(queue_path, outputs["heartbeat_file"])

    seen_keys = load_seen_dedupe_keys(candidate_manifest)
    processed_count = 0
    saved_count = 0
    skipped_count = 0
    error_count = 0
    last_error = ""

    write_claim(
        path=claim_file,
        protocol_version=protocol_version,
        run_id=run_id,
        worker_id=worker_id,
        seed_id=seed_id,
        candidate_manifest=candidate_manifest,
        heartbeat_file=heartbeat_file,
        lease_seconds=lease_seconds,
        state="running",
    )
    write_heartbeat(
        path=heartbeat_file,
        protocol_version=protocol_version,
        run_id=run_id,
        worker_id=worker_id,
        seed_id=seed_id,
        stage="starting",
        current_query_id="",
        current_url="",
        processed_count=processed_count,
        saved_count=saved_count,
        skipped_count=skipped_count,
        error_count=error_count,
        last_error=last_error,
        last_meaningful_event="seed-started",
    )

    for query_payload in item.get("search_queries", []):
        query_id = query_payload["id"]
        query = query_payload["query"]
        language = query_payload.get("language", "")
        write_heartbeat(
            path=heartbeat_file,
            protocol_version=protocol_version,
            run_id=run_id,
            worker_id=worker_id,
            seed_id=seed_id,
            stage="querying",
            current_query_id=query_id,
            current_url="",
            processed_count=processed_count,
            saved_count=saved_count,
            skipped_count=skipped_count,
            error_count=error_count,
            last_error=last_error,
            last_meaningful_event=f"query-start:{query_id}",
        )

        query_results: List[Dict] = []
        search_errors: List[str] = []
        for searcher in BUILTIN_SEARCHERS:
            searcher_name = searcher.__name__
            try:
                query_results.extend(searcher(query, timeout, max_results_per_query))
            except Exception as exc:  # noqa: BLE001
                search_error = f"{searcher_name}:{type(exc).__name__}"
                search_errors.append(search_error)
                last_error = search_error
                error_count += 1

        if not query_results and search_errors:
            write_heartbeat(
                path=heartbeat_file,
                protocol_version=protocol_version,
                run_id=run_id,
                worker_id=worker_id,
                seed_id=seed_id,
                stage="querying",
                current_query_id=query_id,
                current_url="",
                processed_count=processed_count,
                saved_count=saved_count,
                skipped_count=skipped_count,
                error_count=error_count,
                last_error=last_error,
                last_meaningful_event=f"query-error:{query_id}",
            )
            time.sleep(sleep_seconds)
            continue

        if not query_results:
            write_heartbeat(
                path=heartbeat_file,
                protocol_version=protocol_version,
                run_id=run_id,
                worker_id=worker_id,
                seed_id=seed_id,
                stage="querying",
                current_query_id=query_id,
                current_url="",
                processed_count=processed_count,
                saved_count=saved_count,
                skipped_count=skipped_count,
                error_count=error_count,
                last_error=last_error,
                last_meaningful_event=f"query-empty:{query_id}",
            )
            time.sleep(sleep_seconds)
            continue

        processed_count += len(query_results)
        records, skipped = build_candidate_records(
            protocol_version=protocol_version,
            run_id=run_id,
            seed_id=seed_id,
            keyword=query,
            prefer_domains=item.get("prefer_domains", []),
            query_strategy=query_strategy,
            query_id=query_id,
            query=query,
            language=language,
            results=query_results,
            seen_keys=seen_keys,
        )
        skipped_count += skipped
        saved_now = append_jsonl(candidate_manifest, records) if records else 0
        saved_count += saved_now
        current_url = records[-1]["url"] if records else ""
        write_heartbeat(
            path=heartbeat_file,
            protocol_version=protocol_version,
            run_id=run_id,
            worker_id=worker_id,
            seed_id=seed_id,
            stage="querying",
            current_query_id=query_id,
            current_url=current_url,
            processed_count=processed_count,
            saved_count=saved_count,
            skipped_count=skipped_count,
            error_count=error_count,
            last_error=last_error,
            last_meaningful_event=f"query-saved:{query_id}",
        )
        time.sleep(sleep_seconds)

    write_claim(
        path=claim_file,
        protocol_version=protocol_version,
        run_id=run_id,
        worker_id=worker_id,
        seed_id=seed_id,
        candidate_manifest=candidate_manifest,
        heartbeat_file=heartbeat_file,
        lease_seconds=lease_seconds,
        state="completed",
    )
    write_heartbeat(
        path=heartbeat_file,
        protocol_version=protocol_version,
        run_id=run_id,
        worker_id=worker_id,
        seed_id=seed_id,
        stage="completed",
        current_query_id="",
        current_url="",
        processed_count=processed_count,
        saved_count=saved_count,
        skipped_count=skipped_count,
        error_count=error_count,
        last_error=last_error,
        last_meaningful_event="seed-completed",
    )

    return {
        "seed_id": seed_id,
        "candidate_manifest": str(candidate_manifest),
        "claim_file": str(claim_file),
        "heartbeat_file": str(heartbeat_file),
        "processed_count": processed_count,
        "saved_count": saved_count,
        "skipped_count": skipped_count,
        "error_count": error_count,
        "last_error": last_error,
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Query open web sources for assigned seed topics and append candidate manifests.")
    parser.add_argument("--queue", default=DEFAULT_QUEUE, help=f"Seed queue path. Default: {DEFAULT_QUEUE}")
    parser.add_argument("--worker-id", required=True, help="Stable worker identifier used in claim and heartbeat files.")
    parser.add_argument("--seed-id", action="append", default=[], help="Seed id to process. Repeatable. Defaults to all seeds in the queue.")
    parser.add_argument("--max-results-per-query", type=int, default=DEFAULT_MAX_RESULTS_PER_QUERY, help=f"Maximum search results to keep per query. Default: {DEFAULT_MAX_RESULTS_PER_QUERY}")
    parser.add_argument("--timeout", type=int, default=DEFAULT_TIMEOUT, help=f"HTTP timeout in seconds. Default: {DEFAULT_TIMEOUT}")
    parser.add_argument("--lease-seconds", type=int, default=DEFAULT_LEASE_SECONDS, help=f"Claim lease duration in seconds. Default: {DEFAULT_LEASE_SECONDS}")
    parser.add_argument("--sleep-seconds", type=float, default=DEFAULT_SLEEP_SECONDS, help=f"Sleep between queries in seconds. Default: {DEFAULT_SLEEP_SECONDS}")
    parser.add_argument("--summary-json", help="Optional path to save a run summary JSON file.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    queue_path = Path(args.queue).resolve()
    queue_payload = load_queue(queue_path)
    requested_seed_ids = set(args.seed_id)

    items = queue_payload.get("items", [])
    if requested_seed_ids:
        items = [item for item in items if item.get("seed_id", item["id"]) in requested_seed_ids]
    if not items:
        raise QueryWorkerError("No matching seed ids found in queue.")

    results = []
    for item in items:
        results.append(
            run_seed(
                queue_path=queue_path,
                queue_payload=queue_payload,
                item=item,
                worker_id=args.worker_id,
                timeout=args.timeout,
                max_results_per_query=args.max_results_per_query,
                lease_seconds=args.lease_seconds,
                sleep_seconds=args.sleep_seconds,
            )
        )

    payload = {
        "protocol_version": queue_payload["protocol_version"],
        "run_id": queue_payload["run_id"],
        "worker_id": args.worker_id,
        "completed_at": now_iso(),
        "results": results,
    }
    if args.summary_json:
        summary_path = Path(args.summary_json).resolve()
        atomic_write_json(summary_path, payload)
        print(f"Summary JSON: {summary_path}")

    print(json.dumps(payload, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
