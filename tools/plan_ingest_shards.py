#!/usr/bin/env python3
"""Plan minimal-context ingest shards from a seed queue."""

from __future__ import annotations

import argparse
import json
from collections import defaultdict
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, List, Tuple


DEFAULT_QUEUE = "runtime/knowledge-harvest/2026-04-17-sect-sim-cocos/seed-queue.json"
DEFAULT_MAX_SEEDS_PER_SHARD = 4
DEFAULT_MAX_ITEMS_PER_SEED = 8
DEFAULT_FOLDER = "Inbox/Web Research"
DEFAULT_VAULT = "/Users/mawei/MyWork/我的知识库"


def now_iso() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


def load_queue(path: Path) -> Dict:
    return json.loads(path.read_text(encoding="utf-8"))


def ensure_parent(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def count_jsonl(path: Path) -> int:
    if not path.exists():
        return 0
    with path.open("r", encoding="utf-8") as handle:
        return sum(1 for line in handle if line.strip())


def count_saved(path: Path) -> Tuple[int, int]:
    if not path.exists():
        return (0, 0)
    saved = 0
    errors = 0
    with path.open("r", encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if not line:
                continue
            try:
                payload = json.loads(line)
            except json.JSONDecodeError:
                continue
            status = payload.get("status", "")
            if status == "saved":
                saved += 1
            elif status == "error":
                errors += 1
    return (saved, errors)


def shard_group(item: Dict) -> str:
    category = item.get("category", "")
    if category.startswith("client.tilemap"):
        return "client-map"
    if category.startswith("client.") or category.startswith("platform."):
        return "client-platform"
    if category.startswith("architecture."):
        return "architecture"
    if category.startswith("design."):
        return "design"
    if category.startswith("business."):
        return "business"
    return "misc"


def priority_rank(item: Dict) -> tuple[int, str, str]:
    rank = {"p0": 0, "p1": 1, "p2": 2}.get(item.get("priority", "p9"), 9)
    return (rank, item.get("category", ""), item.get("seed_id", item.get("id", "")))


def backlog_row(queue_path: Path, item: Dict) -> Dict:
    outputs = item["outputs"]
    candidate_manifest = (queue_path.parent / outputs["candidate_manifest"]).resolve()
    ingest_manifest = (queue_path.parent / outputs["ingest_manifest"]).resolve()
    candidate_count = count_jsonl(candidate_manifest)
    ingest_saved_count, ingest_error_count = count_saved(ingest_manifest)
    pending_count = max(candidate_count - ingest_saved_count, 0)
    return {
        "seed_id": item.get("seed_id", item["id"]),
        "candidate_count": candidate_count,
        "ingest_saved_count": ingest_saved_count,
        "ingest_error_count": ingest_error_count,
        "pending_count": pending_count,
    }


def build_shards(
    queue_path: Path,
    queue: Dict,
    *,
    max_seeds_per_shard: int,
    max_items_per_seed: int,
    folder: str,
    vault: str,
    preferred_only: bool,
    min_pending: int,
) -> Dict:
    grouped: Dict[str, List[Dict]] = defaultdict(list)
    backlog_rows: List[Dict] = []
    backlog_by_seed: Dict[str, Dict] = {}
    for item in sorted(queue.get("items", []), key=priority_rank):
        row = backlog_row(queue_path, item)
        backlog_rows.append({**row, "category": item.get("category", ""), "priority": item.get("priority", "")})
        backlog_by_seed[row["seed_id"]] = row
        if row["pending_count"] < min_pending:
            continue
        grouped[shard_group(item)].append(item)

    shard_index = 1
    shards = []
    for group_name in sorted(grouped.keys()):
        items = grouped[group_name]
        for offset in range(0, len(items), max_seeds_per_shard):
            chunk = items[offset : offset + max_seeds_per_shard]
            shard_name = f"ingest-{group_name}-{shard_index:02d}"
            worker_id = f"{queue['run_id']}-{shard_name}"
            summary_path = queue_path.parent / "control" / "summaries" / f"{worker_id}.json"
            seed_ids = [item.get("seed_id", item["id"]) for item in chunk]
            command = [
                "python3",
                "tools/ingest_research_candidates.py",
                "--queue",
                str(queue_path),
                "--worker-id",
                worker_id,
                "--vault",
                vault,
                "--folder",
                folder,
                "--seed-subfolder",
                "--max-items-per-seed",
                str(max_items_per_seed),
                "--sleep-seconds",
                "0.3",
                "--summary-json",
                str(summary_path),
            ]
            if preferred_only:
                command.append("--preferred-only")
            for seed_id in seed_ids:
                command.extend(["--seed-id", seed_id])
            shards.append(
                {
                    "shard_id": shard_name,
                    "group": group_name,
                    "worker_id": worker_id,
                    "seed_ids": seed_ids,
                    "summary_json": str(summary_path),
                    "max_items_per_seed": max_items_per_seed,
                    "pending_count": sum(backlog_by_seed[seed_id]["pending_count"] for seed_id in seed_ids),
                    "command": command,
                }
            )
            shard_index += 1

    return {
        "protocol_version": queue.get("protocol_version", "kc.v1"),
        "run_id": queue["run_id"],
        "planned_at": now_iso(),
        "max_seeds_per_shard": max_seeds_per_shard,
        "max_items_per_seed": max_items_per_seed,
        "preferred_only": preferred_only,
        "min_pending": min_pending,
        "vault": vault,
        "folder": folder,
        "backlog": backlog_rows,
        "shards": shards,
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Plan ingest shards from a knowledge-harvest seed queue.")
    parser.add_argument("--queue", default=DEFAULT_QUEUE, help=f"Seed queue path. Default: {DEFAULT_QUEUE}")
    parser.add_argument("--output", help="Optional output JSON path. Defaults to <queue_dir>/ingest-shards.json")
    parser.add_argument("--max-seeds-per-shard", type=int, default=DEFAULT_MAX_SEEDS_PER_SHARD, help=f"Maximum seeds per ingest shard. Default: {DEFAULT_MAX_SEEDS_PER_SHARD}")
    parser.add_argument("--max-items-per-seed", type=int, default=DEFAULT_MAX_ITEMS_PER_SEED, help=f"Maximum ingest items per seed for one shard execution. Default: {DEFAULT_MAX_ITEMS_PER_SEED}")
    parser.add_argument("--vault", default=DEFAULT_VAULT, help=f'Obsidian vault path. Default: "{DEFAULT_VAULT}"')
    parser.add_argument("--folder", default=DEFAULT_FOLDER, help=f'Folder inside the vault. Default: "{DEFAULT_FOLDER}"')
    parser.add_argument("--min-pending", type=int, default=1, help="Minimum pending ingest backlog before a seed enters an ingest shard.")
    parser.add_argument("--preferred-only", action="store_true", help="Only ingest candidates marked as preferred by the query stage.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    queue_path = Path(args.queue).resolve()
    output_path = Path(args.output).resolve() if args.output else (queue_path.parent / "ingest-shards.json")

    queue = load_queue(queue_path)
    payload = build_shards(
        queue_path,
        queue,
        max_seeds_per_shard=args.max_seeds_per_shard,
        max_items_per_seed=args.max_items_per_seed,
        folder=args.folder,
        vault=args.vault,
        preferred_only=args.preferred_only,
        min_pending=args.min_pending,
    )

    ensure_parent(output_path)
    output_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"Ingest shard plan: {output_path}")
    print(f"Shards: {len(payload['shards'])}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
