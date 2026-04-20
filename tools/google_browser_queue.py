#!/usr/bin/env python3
"""Shared control-plane helpers for browser-driven Google discovery."""

from __future__ import annotations

import argparse
import json
import tempfile
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, Iterable, List, Optional, Sequence
from urllib.parse import quote_plus


DEFAULT_PROTOCOL_VERSION = "kb.browser.v1"
DEFAULT_SEED_QUEUE = "runtime/knowledge-harvest/2026-04-17-sect-sim-cocos/seed-queue.json"
DEFAULT_RUN_ID = "2026-04-17-google-browser-wiki"
DEFAULT_OUTPUT_DIR = f"runtime/knowledge-harvest/{DEFAULT_RUN_ID}"


PRIORITY_RANK = {
    "p0": 0,
    "p1": 1,
    "p2": 2,
}


def now_iso() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


def ensure_parent(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def atomic_write_json(path: Path, payload: Dict) -> None:
    ensure_parent(path)
    with tempfile.NamedTemporaryFile("w", encoding="utf-8", dir=path.parent, delete=False) as handle:
        json.dump(payload, handle, ensure_ascii=False, indent=2)
        handle.write("\n")
        temp_name = handle.name
    Path(temp_name).replace(path)


def append_jsonl(path: Path, rows: Iterable[Dict]) -> int:
    ensure_parent(path)
    count = 0
    with path.open("a", encoding="utf-8") as handle:
        for row in rows:
            handle.write(json.dumps(row, ensure_ascii=False) + "\n")
            count += 1
    return count


def load_json(path: Path) -> Dict:
    return json.loads(path.read_text(encoding="utf-8"))


def iter_jsonl(path: Path) -> Iterable[Dict]:
    if not path.exists():
        return []
    rows: List[Dict] = []
    with path.open("r", encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if not line:
                continue
            try:
                rows.append(json.loads(line))
            except json.JSONDecodeError:
                continue
    return rows


def google_search_url(query: str) -> str:
    return f"https://www.google.com/search?q={quote_plus(query)}"


def run_paths(output_dir: Path) -> Dict[str, Path]:
    return {
        "output_dir": output_dir,
        "run_file": output_dir / "browser-run.json",
        "candidate_manifest": output_dir / "control" / "browser-candidates.jsonl",
        "ingest_manifest": output_dir / "control" / "wiki-ingest.jsonl",
        "search_heartbeat": output_dir / "control" / "search-heartbeat.json",
        "ingest_heartbeat": output_dir / "control" / "ingest-heartbeat.json",
        "search_summary": output_dir / "control" / "search-summary.json",
        "ingest_summary": output_dir / "control" / "ingest-summary.json",
    }


def priority_key(query_item: Dict) -> tuple[int, str, str]:
    return (
        PRIORITY_RANK.get(query_item.get("priority", "p9"), 9),
        query_item.get("category", ""),
        query_item.get("query_id", ""),
    )


def build_run(
    *,
    seed_queue_path: Path,
    output_dir: Path,
    run_id: str,
    seed_ids: Sequence[str],
    max_queries: Optional[int],
) -> Dict:
    seed_queue = load_json(seed_queue_path)
    requested = set(seed_ids)
    items = seed_queue.get("items", [])
    if requested:
        items = [item for item in items if item.get("seed_id", item["id"]) in requested]

    queries: List[Dict] = []
    for item in items:
        seed_id = item.get("seed_id", item["id"])
        for query_payload in item.get("search_queries", []):
            queries.append(
                {
                    "seed_id": seed_id,
                    "query_id": query_payload["id"],
                    "query": query_payload["query"],
                    "google_url": query_payload.get("search_urls", {}).get("google") or google_search_url(query_payload["query"]),
                    "language": query_payload.get("language", ""),
                    "category": item.get("category", ""),
                    "priority": item.get("priority", ""),
                    "state": "pending",
                    "candidate_count": 0,
                    "claimed_by": "",
                    "updated_at": now_iso(),
                }
            )

    queries = sorted(queries, key=priority_key)
    if max_queries is not None:
        queries = queries[:max_queries]

    paths = run_paths(output_dir)
    payload = {
        "protocol_version": DEFAULT_PROTOCOL_VERSION,
        "run_id": run_id,
        "created_at": now_iso(),
        "source_seed_queue": str(seed_queue_path.resolve()),
        "outputs": {
            "candidate_manifest": str(paths["candidate_manifest"]),
            "ingest_manifest": str(paths["ingest_manifest"]),
            "search_heartbeat": str(paths["search_heartbeat"]),
            "ingest_heartbeat": str(paths["ingest_heartbeat"]),
            "search_summary": str(paths["search_summary"]),
            "ingest_summary": str(paths["ingest_summary"]),
        },
        "queries": queries,
    }
    atomic_write_json(paths["run_file"], payload)
    return payload


def claim_next_query(*, run_file: Path, worker_id: str) -> Dict:
    payload = load_json(run_file)
    queries = payload.get("queries", [])
    next_item = None
    for query_item in sorted(queries, key=priority_key):
        if query_item.get("state") == "pending":
            next_item = query_item
            break
    if next_item is None:
        raise SystemExit("No pending Google queries left.")

    for query_item in queries:
        if query_item["query_id"] == next_item["query_id"]:
            query_item["state"] = "active"
            query_item["claimed_by"] = worker_id
            query_item["updated_at"] = now_iso()
            next_item = dict(query_item)
            break
    atomic_write_json(run_file, payload)
    return next_item


def mark_query(*, run_file: Path, query_id: str, state: str, worker_id: str) -> Dict:
    payload = load_json(run_file)
    found = None
    for query_item in payload.get("queries", []):
        if query_item["query_id"] != query_id:
            continue
        query_item["state"] = state
        query_item["claimed_by"] = worker_id
        query_item["updated_at"] = now_iso()
        found = dict(query_item)
        break
    if found is None:
        raise SystemExit(f"Unknown query_id: {query_id}")
    atomic_write_json(run_file, payload)
    return found


def load_seen_urls(path: Path) -> set[str]:
    seen = set()
    for row in iter_jsonl(path):
        url = row.get("url")
        if isinstance(url, str) and url:
            seen.add(url.rstrip("/"))
    return seen


def append_candidate(
    *,
    run_file: Path,
    seed_id: str,
    query_id: str,
    query: str,
    title: str,
    url: str,
    discovered_by: str,
    snippet: str,
) -> Dict:
    payload = load_json(run_file)
    output_path = Path(payload["outputs"]["candidate_manifest"])
    seen = load_seen_urls(output_path)
    normalized_url = url.rstrip("/")
    saved = 0
    if normalized_url not in seen:
        append_jsonl(
            output_path,
            [
                {
                    "protocol_version": payload["protocol_version"],
                    "run_id": payload["run_id"],
                    "seed_id": seed_id,
                    "query_id": query_id,
                    "query": query,
                    "title": title.strip(),
                    "url": normalized_url,
                    "snippet": snippet.strip(),
                    "status": "pending",
                    "discovered_at": now_iso(),
                    "discovered_by": discovered_by,
                }
            ],
        )
        saved = 1

    for query_item in payload.get("queries", []):
        if query_item["query_id"] == query_id:
            query_item["candidate_count"] = int(query_item.get("candidate_count", 0)) + saved
            query_item["updated_at"] = now_iso()
            break
    atomic_write_json(run_file, payload)
    return {
        "saved": saved,
        "url": normalized_url,
        "query_id": query_id,
    }


def append_candidate_batch(*, run_file: Path, rows: Sequence[Dict]) -> Dict:
    saved = 0
    skipped = 0
    results: List[Dict] = []
    for row in rows:
        result = append_candidate(
            run_file=run_file,
            seed_id=row["seed_id"],
            query_id=row["query_id"],
            query=row["query"],
            title=row["title"],
            url=row["url"],
            discovered_by=row["discovered_by"],
            snippet=row.get("snippet", ""),
        )
        results.append(result)
        if result["saved"]:
            saved += 1
        else:
            skipped += 1
    return {
        "saved": saved,
        "skipped": skipped,
        "results": results,
    }


def write_heartbeat(
    *,
    path: Path,
    role: str,
    worker_id: str,
    stage: str,
    current_query_id: str,
    current_url: str,
    last_event: str,
) -> Dict:
    payload = {
        "protocol_version": DEFAULT_PROTOCOL_VERSION,
        "role": role,
        "worker_id": worker_id,
        "stage": stage,
        "current_query_id": current_query_id,
        "current_url": current_url,
        "updated_at": now_iso(),
        "last_event": last_event,
    }
    atomic_write_json(path, payload)
    return payload


def build_status(run_file: Path) -> Dict:
    payload = load_json(run_file)
    candidate_manifest = Path(payload["outputs"]["candidate_manifest"])
    ingest_manifest = Path(payload["outputs"]["ingest_manifest"])
    candidate_rows = list(iter_jsonl(candidate_manifest))
    ingest_rows = list(iter_jsonl(ingest_manifest))
    return {
        "run_id": payload["run_id"],
        "query_total": len(payload.get("queries", [])),
        "query_pending": sum(1 for item in payload.get("queries", []) if item.get("state") == "pending"),
        "query_active": sum(1 for item in payload.get("queries", []) if item.get("state") == "active"),
        "query_completed": sum(1 for item in payload.get("queries", []) if item.get("state") == "completed"),
        "candidate_total": len(candidate_rows),
        "ingest_total": len(ingest_rows),
        "ingest_saved": sum(1 for row in ingest_rows if row.get("status") == "saved"),
        "ingest_error": sum(1 for row in ingest_rows if row.get("status") == "error"),
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Manage the shared Google browser discovery queue.")
    subparsers = parser.add_subparsers(dest="command", required=True)

    init_cmd = subparsers.add_parser("init", help="Build a browser discovery run from the seed queue.")
    init_cmd.add_argument("--seed-queue", default=DEFAULT_SEED_QUEUE)
    init_cmd.add_argument("--run-id", default=DEFAULT_RUN_ID)
    init_cmd.add_argument("--output-dir", default=DEFAULT_OUTPUT_DIR)
    init_cmd.add_argument("--seed-id", action="append", default=[])
    init_cmd.add_argument("--max-queries", type=int)

    next_cmd = subparsers.add_parser("claim-next", help="Claim the next pending Google query.")
    next_cmd.add_argument("--run-file", required=True)
    next_cmd.add_argument("--worker-id", required=True)

    mark_cmd = subparsers.add_parser("mark-query", help="Mark a query state.")
    mark_cmd.add_argument("--run-file", required=True)
    mark_cmd.add_argument("--query-id", required=True)
    mark_cmd.add_argument("--state", required=True, choices=["pending", "active", "completed", "skipped"])
    mark_cmd.add_argument("--worker-id", required=True)

    append_cmd = subparsers.add_parser("append-candidate", help="Append one discovered URL to the shared candidate queue.")
    append_cmd.add_argument("--run-file", required=True)
    append_cmd.add_argument("--seed-id", required=True)
    append_cmd.add_argument("--query-id", required=True)
    append_cmd.add_argument("--query", required=True)
    append_cmd.add_argument("--title", required=True)
    append_cmd.add_argument("--url", required=True)
    append_cmd.add_argument("--snippet", default="")
    append_cmd.add_argument("--discovered-by", required=True)

    batch_cmd = subparsers.add_parser("append-batch", help="Append one or more discovered URLs from a JSON file.")
    batch_cmd.add_argument("--run-file", required=True)
    batch_cmd.add_argument("--json-file", required=True)

    hb_cmd = subparsers.add_parser("heartbeat", help="Write a role heartbeat file.")
    hb_cmd.add_argument("--path", required=True)
    hb_cmd.add_argument("--role", required=True)
    hb_cmd.add_argument("--worker-id", required=True)
    hb_cmd.add_argument("--stage", required=True)
    hb_cmd.add_argument("--current-query-id", default="")
    hb_cmd.add_argument("--current-url", default="")
    hb_cmd.add_argument("--last-event", default="")

    status_cmd = subparsers.add_parser("status", help="Show a compact run status.")
    status_cmd.add_argument("--run-file", required=True)

    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if args.command == "init":
        payload = build_run(
            seed_queue_path=Path(args.seed_queue).resolve(),
            output_dir=Path(args.output_dir).resolve(),
            run_id=args.run_id,
            seed_ids=args.seed_id,
            max_queries=args.max_queries,
        )
        print(json.dumps({"run_file": str(run_paths(Path(args.output_dir).resolve())["run_file"]), "query_total": len(payload["queries"])}, ensure_ascii=False, indent=2))
        return 0

    if args.command == "claim-next":
        result = claim_next_query(run_file=Path(args.run_file).resolve(), worker_id=args.worker_id)
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return 0

    if args.command == "mark-query":
        result = mark_query(
            run_file=Path(args.run_file).resolve(),
            query_id=args.query_id,
            state=args.state,
            worker_id=args.worker_id,
        )
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return 0

    if args.command == "append-candidate":
        result = append_candidate(
            run_file=Path(args.run_file).resolve(),
            seed_id=args.seed_id,
            query_id=args.query_id,
            query=args.query,
            title=args.title,
            url=args.url,
            discovered_by=args.discovered_by,
            snippet=args.snippet,
        )
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return 0

    if args.command == "append-batch":
        batch_path = Path(args.json_file).resolve()
        batch_payload = json.loads(batch_path.read_text(encoding="utf-8"))
        rows = batch_payload if isinstance(batch_payload, list) else [batch_payload]
        result = append_candidate_batch(
            run_file=Path(args.run_file).resolve(),
            rows=rows,
        )
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return 0

    if args.command == "heartbeat":
        result = write_heartbeat(
            path=Path(args.path).resolve(),
            role=args.role,
            worker_id=args.worker_id,
            stage=args.stage,
            current_query_id=args.current_query_id,
            current_url=args.current_url,
            last_event=args.last_event,
        )
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return 0

    if args.command == "status":
        result = build_status(Path(args.run_file).resolve())
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return 0

    raise SystemExit(f"Unsupported command: {args.command}")


if __name__ == "__main__":
    raise SystemExit(main())
