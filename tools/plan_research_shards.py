#!/usr/bin/env python3
"""Plan minimal-context query shards from a seed queue."""

from __future__ import annotations

import argparse
import json
from collections import defaultdict
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, List


DEFAULT_QUEUE = "runtime/knowledge-harvest/2026-04-17-sect-sim-cocos/seed-queue.json"
DEFAULT_MAX_SEEDS_PER_SHARD = 4


def now_iso() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


def load_queue(path: Path) -> Dict:
    return json.loads(path.read_text(encoding="utf-8"))


def ensure_parent(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


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


def build_shards(queue_path: Path, queue: Dict, max_seeds_per_shard: int) -> Dict:
    grouped: Dict[str, List[Dict]] = defaultdict(list)
    for item in sorted(queue.get("items", []), key=priority_rank):
        grouped[shard_group(item)].append(item)

    shard_index = 1
    shards = []
    for group_name in sorted(grouped.keys()):
        items = grouped[group_name]
        for offset in range(0, len(items), max_seeds_per_shard):
            chunk = items[offset : offset + max_seeds_per_shard]
            shard_name = f"{group_name}-{shard_index:02d}"
            worker_id = f"{queue['run_id']}-{shard_name}"
            summary_path = queue_path.parent / "control" / "summaries" / f"{worker_id}.json"
            seed_ids = [item.get("seed_id", item["id"]) for item in chunk]
            command = [
                "python3",
                "tools/query_research_candidates.py",
                "--queue",
                str(queue_path),
                "--worker-id",
                worker_id,
                "--max-results-per-query",
                "8",
                "--sleep-seconds",
                "0.3",
                "--summary-json",
                str(summary_path),
            ]
            for seed_id in seed_ids:
                command.extend(["--seed-id", seed_id])
            shards.append(
                {
                    "shard_id": shard_name,
                    "group": group_name,
                    "worker_id": worker_id,
                    "seed_ids": seed_ids,
                    "summary_json": str(summary_path),
                    "command": command,
                }
            )
            shard_index += 1

    return {
        "protocol_version": queue.get("protocol_version", "kc.v1"),
        "run_id": queue["run_id"],
        "planned_at": now_iso(),
        "max_seeds_per_shard": max_seeds_per_shard,
        "shards": shards,
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Plan query shards from a knowledge-harvest seed queue.")
    parser.add_argument("--queue", default=DEFAULT_QUEUE, help=f"Seed queue path. Default: {DEFAULT_QUEUE}")
    parser.add_argument("--output", help="Optional output JSON path. Defaults to <queue_dir>/query-shards.json")
    parser.add_argument("--max-seeds-per-shard", type=int, default=DEFAULT_MAX_SEEDS_PER_SHARD, help=f"Maximum seeds per shard. Default: {DEFAULT_MAX_SEEDS_PER_SHARD}")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    queue_path = Path(args.queue).resolve()
    output_path = Path(args.output).resolve() if args.output else (queue_path.parent / "query-shards.json")

    queue = load_queue(queue_path)
    payload = build_shards(queue_path, queue, args.max_seeds_per_shard)

    ensure_parent(output_path)
    output_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"Shard plan: {output_path}")
    print(f"Shards: {len(payload['shards'])}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
