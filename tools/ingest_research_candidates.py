#!/usr/bin/env python3
"""Ingest candidate article URLs into Obsidian with minimal control-plane files."""

from __future__ import annotations

import argparse
import json
import tempfile
import time
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Dict, Iterable, List, Tuple

import requests

from query_research_candidates import DEFAULT_QUEUE, QueryWorkerError, resolve_from_queue
from query_research_candidates import domain_matches
from url_to_obsidian import (
    AUTH_FILE,
    DEFAULT_FETCH_TIMEOUT,
    DEFAULT_FOLDER,
    DEFAULT_LLM_TIMEOUT,
    DEFAULT_MAX_SOURCE_CHARS,
    DEFAULT_TEXT_MODEL,
    CodexOAuthError,
    UrlToObsidianError,
    build_fallback_markdown,
    build_note_filename,
    build_note_markdown,
    extract_page,
    resolve_vault_path,
    summarize_page_with_oauth,
    write_note,
)


DEFAULT_LEASE_SECONDS = 1800
DEFAULT_SLEEP_SECONDS = 0.5


def now_utc() -> datetime:
    return datetime.now(timezone.utc)


def now_iso() -> str:
    return now_utc().replace(microsecond=0).isoformat()


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


def load_successful_urls(path: Path) -> set[str]:
    urls: set[str] = set()
    for row in iter_jsonl(path):
        if row.get("status") == "saved" and row.get("url"):
            urls.add(row["url"])
    return urls


def resolve_ingest_control_paths(queue_path: Path, item: Dict) -> Tuple[Path, Path, Path]:
    outputs = item.get("outputs", {})
    seed_id = item.get("seed_id", item["id"])
    candidate_manifest = resolve_from_queue(queue_path, outputs["candidate_manifest"])
    ingest_manifest = resolve_from_queue(queue_path, outputs["ingest_manifest"])

    ingest_claim_value = outputs.get("ingest_claim_file")
    if ingest_claim_value:
        ingest_claim_file = resolve_from_queue(queue_path, ingest_claim_value)
    else:
        ingest_claim_file = (queue_path.parent / f"control/ingest-claims/{seed_id}.json").resolve()

    ingest_heartbeat_value = outputs.get("ingest_heartbeat_file")
    if ingest_heartbeat_value:
        ingest_heartbeat_file = resolve_from_queue(queue_path, ingest_heartbeat_value)
    else:
        ingest_heartbeat_file = (queue_path.parent / f"control/ingest-heartbeats/{seed_id}.json").resolve()

    return candidate_manifest, ingest_manifest, ingest_claim_file, ingest_heartbeat_file


def write_claim(
    *,
    path: Path,
    protocol_version: str,
    run_id: str,
    worker_id: str,
    seed_id: str,
    candidate_manifest: Path,
    ingest_manifest: Path,
    heartbeat_file: Path,
    lease_seconds: int,
    state: str,
) -> None:
    claimed_at = now_utc()
    payload = {
        "protocol_version": protocol_version,
        "run_id": run_id,
        "claim_id": f"ingest:{seed_id}:{worker_id}",
        "seed_id": seed_id,
        "worker_id": worker_id,
        "state": state,
        "claimed_at": claimed_at.replace(microsecond=0).isoformat(),
        "lease_expires_at": (claimed_at + timedelta(seconds=lease_seconds)).replace(microsecond=0).isoformat(),
        "candidate_manifest": str(candidate_manifest),
        "ingest_manifest": str(ingest_manifest),
        "heartbeat_file": str(heartbeat_file),
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
        "current_url": current_url,
        "last_error": last_error,
        "last_meaningful_event": last_meaningful_event,
    }
    atomic_write_json(path, payload)


def build_record_base(*, protocol_version: str, run_id: str, worker_id: str, seed_id: str, candidate: Dict) -> Dict:
    return {
        "protocol_version": protocol_version,
        "run_id": run_id,
        "worker_id": worker_id,
        "seed_id": seed_id,
        "url": candidate.get("url", ""),
        "title": candidate.get("title", ""),
        "source_domain": candidate.get("source_domain", ""),
        "query_id": candidate.get("query_id", ""),
        "attempted_at": now_iso(),
    }


def run_seed(
    *,
    queue_path: Path,
    queue_payload: Dict,
    item: Dict,
    worker_id: str,
    vault: Path,
    folder: str,
    seed_subfolder: bool,
    auth_file: Path,
    model: str,
    fetch_timeout: int,
    llm_timeout: int,
    max_source_chars: int,
    lease_seconds: int,
    sleep_seconds: float,
    max_items_per_seed: int,
    include_domains: List[str],
    preferred_only: bool,
    force_refresh: bool,
    overwrite: bool,
    skip_llm: bool,
    timestamp_prefix: bool,
) -> Dict:
    protocol_version = queue_payload["protocol_version"]
    run_id = queue_payload["run_id"]
    seed_id = item.get("seed_id", item["id"])
    candidate_manifest, ingest_manifest, claim_file, heartbeat_file = resolve_ingest_control_paths(queue_path, item)

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
        ingest_manifest=ingest_manifest,
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
        current_url="",
        processed_count=processed_count,
        saved_count=saved_count,
        skipped_count=skipped_count,
        error_count=error_count,
        last_error=last_error,
        last_meaningful_event="seed-started",
    )

    if not candidate_manifest.exists():
        write_claim(
            path=claim_file,
            protocol_version=protocol_version,
            run_id=run_id,
            worker_id=worker_id,
            seed_id=seed_id,
            candidate_manifest=candidate_manifest,
            ingest_manifest=ingest_manifest,
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
            current_url="",
            processed_count=0,
            saved_count=0,
            skipped_count=0,
            error_count=0,
            last_error="",
            last_meaningful_event="candidate-manifest-missing",
        )
        return {
            "seed_id": seed_id,
            "candidate_manifest": str(candidate_manifest),
            "ingest_manifest": str(ingest_manifest),
            "claim_file": str(claim_file),
            "heartbeat_file": str(heartbeat_file),
            "processed_count": 0,
            "saved_count": 0,
            "skipped_count": 0,
            "error_count": 0,
            "last_error": "",
        }

    successful_urls = load_successful_urls(ingest_manifest)
    candidates = [row for row in iter_jsonl(candidate_manifest) if row.get("url")]

    pending_candidates = []
    for candidate in candidates:
        if candidate["url"] in successful_urls:
            skipped_count += 1
            continue
        candidate_domain = candidate.get("source_domain", "")
        if include_domains and not domain_matches(candidate_domain, include_domains):
            skipped_count += 1
            continue
        if preferred_only and candidate.get("source_tier") != "preferred":
            skipped_count += 1
            continue
        pending_candidates.append(candidate)

    pending_candidates.sort(
        key=lambda candidate: (
            candidate.get("source_tier") != "preferred",
            candidate.get("source_domain", ""),
            candidate.get("query_id", ""),
            candidate.get("url", ""),
        )
    )

    if max_items_per_seed > 0:
        pending_candidates = pending_candidates[:max_items_per_seed]

    target_folder = folder
    if seed_subfolder:
        target_folder = str(Path(folder) / seed_id)

    for candidate in pending_candidates:
        url = candidate["url"]
        processed_count += 1
        write_heartbeat(
            path=heartbeat_file,
            protocol_version=protocol_version,
            run_id=run_id,
            worker_id=worker_id,
            seed_id=seed_id,
            stage="ingesting",
            current_url=url,
            processed_count=processed_count,
            saved_count=saved_count,
            skipped_count=skipped_count,
            error_count=error_count,
            last_error=last_error,
            last_meaningful_event="ingest-start",
        )
        try:
            extracted = extract_page(url, fetch_timeout)
            fetched_at = now_iso()
            if skip_llm:
                model_output = build_fallback_markdown(extracted)
                model_name = "skip-llm"
            else:
                result = summarize_page_with_oauth(
                    extracted,
                    auth_file=auth_file,
                    model=model,
                    timeout=llm_timeout,
                    max_source_chars=max_source_chars,
                    force_refresh=force_refresh,
                )
                model_output = result["assistant_text"]
                model_name = model

            filename = build_note_filename(
                extracted=extracted,
                explicit_name=None,
                timestamp_prefix=timestamp_prefix,
            )
            note_text = build_note_markdown(
                extracted=extracted,
                model_output=model_output,
                model_name=model_name,
                fetched_at=fetched_at,
            )
            absolute_path, relative_path = write_note(
                vault=vault,
                folder=target_folder,
                filename=filename,
                content=note_text,
                overwrite=overwrite,
            )
            append_jsonl(
                ingest_manifest,
                [
                    {
                        **build_record_base(
                            protocol_version=protocol_version,
                            run_id=run_id,
                            worker_id=worker_id,
                            seed_id=seed_id,
                            candidate=candidate,
                        ),
                        "status": "saved",
                        "obsidian_note_path": str(absolute_path),
                        "obsidian_relative_path": str(relative_path),
                        "error_signature": "",
                    }
                ],
            )
            saved_count += 1
            last_error = ""
            write_heartbeat(
                path=heartbeat_file,
                protocol_version=protocol_version,
                run_id=run_id,
                worker_id=worker_id,
                seed_id=seed_id,
                stage="ingesting",
                current_url=url,
                processed_count=processed_count,
                saved_count=saved_count,
                skipped_count=skipped_count,
                error_count=error_count,
                last_error=last_error,
                last_meaningful_event="ingest-saved",
            )
        except (UrlToObsidianError, CodexOAuthError, requests.RequestException) as exc:
            error_count += 1
            last_error = f"{type(exc).__name__}:{str(exc)[:200]}"
            append_jsonl(
                ingest_manifest,
                [
                    {
                        **build_record_base(
                            protocol_version=protocol_version,
                            run_id=run_id,
                            worker_id=worker_id,
                            seed_id=seed_id,
                            candidate=candidate,
                        ),
                        "status": "error",
                        "obsidian_note_path": "",
                        "obsidian_relative_path": "",
                        "error_signature": last_error,
                    }
                ],
            )
            write_heartbeat(
                path=heartbeat_file,
                protocol_version=protocol_version,
                run_id=run_id,
                worker_id=worker_id,
                seed_id=seed_id,
                stage="ingesting",
                current_url=url,
                processed_count=processed_count,
                saved_count=saved_count,
                skipped_count=skipped_count,
                error_count=error_count,
                last_error=last_error,
                last_meaningful_event="ingest-error",
            )
        time.sleep(sleep_seconds)

    write_claim(
        path=claim_file,
        protocol_version=protocol_version,
        run_id=run_id,
        worker_id=worker_id,
        seed_id=seed_id,
        candidate_manifest=candidate_manifest,
        ingest_manifest=ingest_manifest,
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
        "ingest_manifest": str(ingest_manifest),
        "claim_file": str(claim_file),
        "heartbeat_file": str(heartbeat_file),
        "processed_count": processed_count,
        "saved_count": saved_count,
        "skipped_count": skipped_count,
        "error_count": error_count,
        "last_error": last_error,
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Ingest candidate article URLs into Obsidian.")
    parser.add_argument("--queue", default=DEFAULT_QUEUE, help=f"Seed queue path. Default: {DEFAULT_QUEUE}")
    parser.add_argument("--worker-id", required=True, help="Stable worker identifier used in claim and heartbeat files.")
    parser.add_argument("--seed-id", action="append", default=[], help="Seed id to process. Repeatable. Defaults to all seeds in the queue.")
    parser.add_argument("--vault", required=True, help="Absolute Obsidian vault path.")
    parser.add_argument("--folder", default=DEFAULT_FOLDER, help=f'Folder inside the vault. Default: "{DEFAULT_FOLDER}"')
    parser.add_argument("--seed-subfolder", action="store_true", help="Store notes under <folder>/<seed_id>/ instead of a flat folder.")
    parser.add_argument("--auth-file", default=str(AUTH_FILE), help=f"Codex auth file path. Default: {AUTH_FILE}")
    parser.add_argument("--model", default=DEFAULT_TEXT_MODEL, help=f"OAuth-backed text model. Default: {DEFAULT_TEXT_MODEL}")
    parser.add_argument("--fetch-timeout", type=int, default=DEFAULT_FETCH_TIMEOUT, help=f"HTTP fetch timeout in seconds. Default: {DEFAULT_FETCH_TIMEOUT}")
    parser.add_argument("--llm-timeout", type=int, default=DEFAULT_LLM_TIMEOUT, help=f"Model request timeout in seconds. Default: {DEFAULT_LLM_TIMEOUT}")
    parser.add_argument("--max-source-chars", type=int, default=DEFAULT_MAX_SOURCE_CHARS, help=f"Maximum extracted source chars sent to the model. Default: {DEFAULT_MAX_SOURCE_CHARS}")
    parser.add_argument("--lease-seconds", type=int, default=DEFAULT_LEASE_SECONDS, help=f"Claim lease duration in seconds. Default: {DEFAULT_LEASE_SECONDS}")
    parser.add_argument("--sleep-seconds", type=float, default=DEFAULT_SLEEP_SECONDS, help=f"Sleep between candidate ingests in seconds. Default: {DEFAULT_SLEEP_SECONDS}")
    parser.add_argument("--max-items-per-seed", type=int, default=0, help="Maximum candidate URLs to ingest per seed in this run. Default: 0 (no limit).")
    parser.add_argument("--include-domain", action="append", default=[], help="Only ingest candidates whose source_domain matches one of these domains. Repeatable.")
    parser.add_argument("--preferred-only", action="store_true", help="Only ingest candidates marked as preferred by the query stage.")
    parser.add_argument("--force-refresh", action="store_true", help="Refresh the OAuth token before sending the model request.")
    parser.add_argument("--overwrite", action="store_true", help="Overwrite an existing note if the target file already exists.")
    parser.add_argument("--skip-llm", action="store_true", help="Write deterministic fallback notes without the OAuth model call.")
    parser.add_argument("--timestamp-prefix", action="store_true", help="Prefix note filenames with YYYY-MM-DD.")
    parser.add_argument("--summary-json", help="Optional path to save a run summary JSON file.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    queue_path = Path(args.queue).resolve()
    queue_payload = load_queue(queue_path)
    vault = resolve_vault_path(args.vault)
    auth_file = Path(args.auth_file).expanduser().resolve()
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
                vault=vault,
                folder=args.folder,
                seed_subfolder=args.seed_subfolder,
                auth_file=auth_file,
                model=args.model,
                fetch_timeout=args.fetch_timeout,
                llm_timeout=args.llm_timeout,
                max_source_chars=args.max_source_chars,
                lease_seconds=args.lease_seconds,
                sleep_seconds=args.sleep_seconds,
                max_items_per_seed=args.max_items_per_seed,
                include_domains=args.include_domain,
                preferred_only=args.preferred_only,
                force_refresh=args.force_refresh,
                overwrite=args.overwrite,
                skip_llm=args.skip_llm,
                timestamp_prefix=args.timestamp_prefix,
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
