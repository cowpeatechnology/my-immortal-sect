#!/usr/bin/env python3
"""Inspect research worker heartbeat freshness with minimal output."""

from __future__ import annotations

import argparse
import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, List, Tuple


DEFAULT_STALE_SECONDS = 120
DEFAULT_QUEUE = "runtime/knowledge-harvest/2026-04-17-sect-sim-cocos/seed-queue.json"


def now_utc() -> datetime:
    return datetime.now(timezone.utc)


def parse_iso(timestamp: str) -> datetime | None:
    if not timestamp:
        return None
    try:
        return datetime.fromisoformat(timestamp.replace("Z", "+00:00"))
    except ValueError:
        return None


def resolve_from_queue(queue_path: Path, relative_or_absolute: str) -> Path:
    path = Path(relative_or_absolute)
    if path.is_absolute():
        return path
    return (queue_path.parent / path).resolve()


def heartbeat_field_for_stage(stage: str) -> str:
    if stage == "ingest":
        return "ingest_heartbeat_file"
    return "heartbeat_file"


def fallback_ingest_heartbeat_path(queue_path: Path, seed_id: str) -> Path:
    return (queue_path.parent / f"control/ingest-heartbeats/{seed_id}.json").resolve()


def heartbeat_status(heartbeat_path: Path, stale_seconds: int) -> Tuple[str, Dict]:
    if not heartbeat_path.exists():
        return "missing", {}
    try:
        payload = json.loads(heartbeat_path.read_text(encoding="utf-8"))
    except Exception:
        return "invalid", {}

    updated_at = parse_iso(payload.get("updated_at", ""))
    if updated_at is None:
        return "invalid", payload

    age = (now_utc() - updated_at).total_seconds()
    payload["_age_seconds"] = int(age)
    if age > stale_seconds:
        return "stale", payload
    return "fresh", payload


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Check heartbeat freshness for knowledge harvesting workers.")
    parser.add_argument("--queue", default=DEFAULT_QUEUE, help=f"Seed queue path. Default: {DEFAULT_QUEUE}")
    parser.add_argument("--stage", choices=("query", "ingest"), default="query", help="Heartbeat stage to inspect. Default: query")
    parser.add_argument("--stale-seconds", type=int, default=DEFAULT_STALE_SECONDS, help=f"Heartbeat stale threshold in seconds. Default: {DEFAULT_STALE_SECONDS}")
    parser.add_argument("--json", action="store_true", help="Print machine-readable JSON instead of a text table.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    queue_path = Path(args.queue).resolve()
    queue = json.loads(queue_path.read_text(encoding="utf-8"))

    rows: List[Dict] = []
    for item in queue.get("items", []):
        outputs = item.get("outputs", {})
        seed_id = item.get("seed_id", item["id"])
        heartbeat_field = heartbeat_field_for_stage(args.stage)
        heartbeat_value = outputs.get(heartbeat_field, "")
        if heartbeat_value:
            heartbeat_path = resolve_from_queue(queue_path, heartbeat_value)
        elif args.stage == "ingest":
            heartbeat_path = fallback_ingest_heartbeat_path(queue_path, seed_id)
        else:
            heartbeat_path = resolve_from_queue(queue_path, outputs.get("heartbeat_file", ""))
        status, heartbeat = heartbeat_status(heartbeat_path, args.stale_seconds)
        rows.append(
            {
                "seed_id": seed_id,
                "keyword": item["keyword"],
                "status": status,
                "worker_stage": args.stage,
                "heartbeat_file": str(heartbeat_path),
                "stage": heartbeat.get("stage", ""),
                "processed_count": heartbeat.get("processed_count", 0),
                "saved_count": heartbeat.get("saved_count", 0),
                "error_count": heartbeat.get("error_count", 0),
                "age_seconds": heartbeat.get("_age_seconds"),
                "current_url": heartbeat.get("current_url", ""),
            }
        )

    if args.json:
        print(json.dumps(rows, ensure_ascii=False, indent=2))
        return 0

    print(f"Queue: {queue_path}")
    print(f"Run ID: {queue.get('run_id', 'unknown')}")
    print(f"Worker stage: {args.stage}")
    print(f"Items: {len(rows)}")
    print("Status  Age   Processed Saved Errors Seed ID")
    for row in rows:
        age = "-" if row["age_seconds"] is None else str(row["age_seconds"])
        print(
            f"{row['status']:<7} {age:<5} {row['processed_count']:<9} "
            f"{row['saved_count']:<5} {row['error_count']:<6} {row['seed_id']}"
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
