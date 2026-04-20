#!/usr/bin/env python3
"""Consume discovered URLs from the browser queue into the project wiki."""

from __future__ import annotations

import argparse
import json
import time
from pathlib import Path
from typing import Dict, Iterable, List, Optional

import requests

from browser_content_filters import probe_url
from google_browser_queue import iter_jsonl
from llm_wiki_maintainer import (
    DEFAULT_PROJECT_FOLDER,
    DEFAULT_PROJECT_NAME,
    WikiMaintainerError,
    ingest_url,
)
from url_to_obsidian import (
    AUTH_FILE,
    DEFAULT_FETCH_TIMEOUT,
    DEFAULT_LLM_TIMEOUT,
    DEFAULT_MAX_SOURCE_CHARS,
    DEFAULT_TEXT_MODEL,
    CodexOAuthError,
    UrlToObsidianError,
    resolve_vault_path,
)


DEFAULT_RUN_FILE = "runtime/knowledge-harvest/2026-04-17-google-browser-wiki/browser-run.json"
DEFAULT_POLL_SECONDS = 20
DEFAULT_IDLE_SECONDS = 1200


def now_iso() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%S%z")


def append_jsonl(path: Path, rows: Iterable[Dict]) -> int:
    path.parent.mkdir(parents=True, exist_ok=True)
    count = 0
    with path.open("a", encoding="utf-8") as handle:
        for row in rows:
            handle.write(json.dumps(row, ensure_ascii=False) + "\n")
            count += 1
    return count


def atomic_write_json(path: Path, payload: Dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def load_json(path: Path) -> Dict:
    return json.loads(path.read_text(encoding="utf-8"))


def load_processed_urls(path: Path) -> set[str]:
    seen = set()
    for row in iter_jsonl(path):
        url = row.get("url")
        if isinstance(url, str) and url:
            seen.add(url.rstrip("/"))
    return seen


def write_heartbeat(
    *,
    path: Path,
    worker_id: str,
    stage: str,
    current_url: str,
    processed_count: int,
    saved_count: int,
    error_count: int,
    last_error: str,
    last_event: str,
) -> None:
    atomic_write_json(
        path,
        {
            "role": "ingest",
            "worker_id": worker_id,
            "stage": stage,
            "current_url": current_url,
            "processed_count": processed_count,
            "saved_count": saved_count,
            "error_count": error_count,
            "last_error": last_error,
            "last_event": last_event,
            "updated_at": now_iso(),
        },
    )


def pending_candidates(run_payload: Dict, processed_urls: set[str]) -> List[Dict]:
    candidate_manifest = Path(run_payload["outputs"]["candidate_manifest"])
    rows = list(iter_jsonl(candidate_manifest))
    pending: List[Dict] = []
    for row in rows:
        url = (row.get("url") or "").rstrip("/")
        if not url or url in processed_urls:
            continue
        pending.append(row)
    return pending


def process_candidate(
    *,
    candidate: Dict,
    vault: Path,
    project_folder: str,
    project_name: str,
    auth_file: Path,
    model: str,
    fetch_timeout: int,
    llm_timeout: int,
    max_source_chars: int,
    force_refresh: bool,
    skip_llm: bool,
) -> Dict:
    probe = probe_url(candidate["url"], timeout_seconds=fetch_timeout)
    if probe.skip_reason:
        return {
            "status": "skipped",
            "url": candidate["url"],
            "final_url": probe.final_url,
            "seed_id": candidate.get("seed_id", ""),
            "query_id": candidate.get("query_id", ""),
            "title": candidate.get("title", ""),
            "attempted_at": now_iso(),
            "skip_reason": probe.skip_reason,
            "content_type": probe.content_type,
        }

    result_paths = ingest_url(
        vault=vault,
        project_folder=project_folder,
        project_name=project_name,
        source_url=candidate["url"],
        fetch_url=probe.final_url if probe.final_url and probe.final_url != candidate["url"] else None,
        auth_file=auth_file,
        model=model,
        fetch_timeout=fetch_timeout,
        llm_timeout=llm_timeout,
        max_source_chars=max_source_chars,
        force_refresh=force_refresh,
        skip_llm=skip_llm,
        overwrite=False,
    )
    return {
        "status": "saved",
        "url": candidate["url"],
        "seed_id": candidate.get("seed_id", ""),
        "query_id": candidate.get("query_id", ""),
        "title": candidate.get("title", ""),
        "attempted_at": now_iso(),
        "raw_note": str(result_paths["raw_note"]),
        "source_note": str(result_paths["source_note"]),
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Continuously ingest discovered browser URLs into the project wiki.")
    parser.add_argument("--run-file", default=DEFAULT_RUN_FILE)
    parser.add_argument("--worker-id", required=True)
    parser.add_argument("--vault", required=True)
    parser.add_argument("--project-folder", default=DEFAULT_PROJECT_FOLDER)
    parser.add_argument("--project-name", default=DEFAULT_PROJECT_NAME)
    parser.add_argument("--auth-file", default=str(AUTH_FILE))
    parser.add_argument("--model", default=DEFAULT_TEXT_MODEL)
    parser.add_argument("--fetch-timeout", type=int, default=DEFAULT_FETCH_TIMEOUT)
    parser.add_argument("--llm-timeout", type=int, default=DEFAULT_LLM_TIMEOUT)
    parser.add_argument("--max-source-chars", type=int, default=DEFAULT_MAX_SOURCE_CHARS)
    parser.add_argument("--poll-seconds", type=int, default=DEFAULT_POLL_SECONDS)
    parser.add_argument("--idle-seconds", type=int, default=DEFAULT_IDLE_SECONDS)
    parser.add_argument("--force-refresh", action="store_true")
    parser.add_argument("--skip-llm", action="store_true")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    run_file = Path(args.run_file).resolve()
    run_payload = load_json(run_file)
    ingest_manifest = Path(run_payload["outputs"]["ingest_manifest"])
    heartbeat_path = Path(run_payload["outputs"]["ingest_heartbeat"])
    summary_path = Path(run_payload["outputs"]["ingest_summary"])
    vault = resolve_vault_path(args.vault)

    processed_count = 0
    saved_count = 0
    error_count = 0
    last_error = ""
    last_progress = time.time()

    write_heartbeat(
        path=heartbeat_path,
        worker_id=args.worker_id,
        stage="starting",
        current_url="",
        processed_count=processed_count,
        saved_count=saved_count,
        error_count=error_count,
        last_error=last_error,
        last_event="worker-started",
    )

    while True:
        run_payload = load_json(run_file)
        processed_urls = load_processed_urls(ingest_manifest)
        pending = pending_candidates(run_payload, processed_urls)
        if not pending:
            if time.time() - last_progress >= args.idle_seconds:
                write_heartbeat(
                    path=heartbeat_path,
                    worker_id=args.worker_id,
                    stage="idle-timeout",
                    current_url="",
                    processed_count=processed_count,
                    saved_count=saved_count,
                    error_count=error_count,
                    last_error=last_error,
                    last_event="idle-timeout",
                )
                atomic_write_json(
                    summary_path,
                    {
                        "worker_id": args.worker_id,
                        "processed_count": processed_count,
                        "saved_count": saved_count,
                        "error_count": error_count,
                        "completed_at": now_iso(),
                    },
                )
                print(json.dumps({"status": "idle-timeout", "saved_count": saved_count, "error_count": error_count}, ensure_ascii=False, indent=2))
                return 0

            write_heartbeat(
                path=heartbeat_path,
                worker_id=args.worker_id,
                stage="idle",
                current_url="",
                processed_count=processed_count,
                saved_count=saved_count,
                error_count=error_count,
                last_error=last_error,
                last_event="waiting-for-candidates",
            )
            time.sleep(args.poll_seconds)
            continue

        candidate = pending[0]
        current_url = candidate["url"]
        write_heartbeat(
            path=heartbeat_path,
            worker_id=args.worker_id,
            stage="ingesting",
            current_url=current_url,
            processed_count=processed_count,
            saved_count=saved_count,
            error_count=error_count,
            last_error=last_error,
            last_event="ingest-start",
        )

        try:
            record = process_candidate(
                candidate=candidate,
                vault=vault,
                project_folder=args.project_folder,
                project_name=args.project_name,
                auth_file=Path(args.auth_file),
                model=args.model,
                fetch_timeout=args.fetch_timeout,
                llm_timeout=args.llm_timeout,
                max_source_chars=args.max_source_chars,
                force_refresh=args.force_refresh,
                skip_llm=args.skip_llm,
            )
            append_jsonl(ingest_manifest, [record])
            if record.get("status") == "saved":
                saved_count += 1
            processed_count += 1
            last_progress = time.time()
            write_heartbeat(
                path=heartbeat_path,
                worker_id=args.worker_id,
                stage="ingesting",
                current_url=current_url,
                processed_count=processed_count,
                saved_count=saved_count,
                error_count=error_count,
                last_error=last_error,
                last_event="ingest-saved" if record.get("status") == "saved" else "ingest-skipped",
            )
        except (WikiMaintainerError, UrlToObsidianError, CodexOAuthError, requests.RequestException) as exc:
            error_count += 1
            processed_count += 1
            last_error = f"{type(exc).__name__}: {exc}"
            append_jsonl(
                ingest_manifest,
                [
                    {
                        "status": "error",
                        "url": candidate["url"],
                        "seed_id": candidate.get("seed_id", ""),
                        "query_id": candidate.get("query_id", ""),
                        "title": candidate.get("title", ""),
                        "attempted_at": now_iso(),
                        "error": last_error,
                    }
                ],
            )
            last_progress = time.time()
            write_heartbeat(
                path=heartbeat_path,
                worker_id=args.worker_id,
                stage="ingesting",
                current_url=current_url,
                processed_count=processed_count,
                saved_count=saved_count,
                error_count=error_count,
                last_error=last_error,
                last_event="ingest-error",
            )


if __name__ == "__main__":
    raise SystemExit(main())
