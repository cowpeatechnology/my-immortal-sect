#!/usr/bin/env python3
"""Prepare a formal knowledge-harvest execution plan from an existing queue."""

from __future__ import annotations

import argparse
import json
from copy import deepcopy
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, Iterable, List, Tuple

from build_research_seed_queue import PROFILE_TOPICS, build_topic_item


DEFAULT_QUEUE = "runtime/knowledge-harvest/2026-04-17-sect-sim-cocos/seed-queue.json"
DEFAULT_RELAXED_QUEUE = "runtime/knowledge-harvest/2026-04-17-sect-sim-cocos/seed-queue-relaxed.json"
DEFAULT_PLAN = "runtime/knowledge-harvest/2026-04-17-sect-sim-cocos/execution-plan.json"
DEFAULT_RELAX_THRESHOLD = 12
DEFAULT_RELAXED_YEAR_FROM = 2021
DEFAULT_FOUNDATION_YEAR_FROM = 2016
DEFAULT_DO_NOT_CLOSE_BEFORE = "2026-04-17T14:00:00+08:00"

EXPANSION_TOPIC_IDS = [
    "game-2d-development",
    "game-design-general",
    "game-commercialization",
    "free-to-play-economy-design",
    "game-retention-liveops",
]


def now_iso() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


def ensure_parent(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def load_json(path: Path) -> Dict:
    return json.loads(path.read_text(encoding="utf-8"))


def write_json(path: Path, payload: Dict) -> None:
    ensure_parent(path)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def count_lines(path: Path) -> int:
    if not path.exists():
        return 0
    with path.open("r", encoding="utf-8") as handle:
        return sum(1 for line in handle if line.strip())


def count_ingest_saved(path: Path) -> Tuple[int, int]:
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


def topic_lookup(profile: str) -> Dict[str, Dict]:
    return {topic["id"]: topic for topic in PROFILE_TOPICS.get(profile, [])}


def priority_key(item: Dict) -> Tuple[int, str, str]:
    rank = {"p0": 0, "p1": 1, "p2": 2}.get(item.get("priority", "p9"), 9)
    return (rank, item.get("category", ""), item.get("seed_id", item.get("id", "")))


def merge_items(existing_items: Iterable[Dict], new_items: Iterable[Dict]) -> List[Dict]:
    merged = {item.get("seed_id", item["id"]): item for item in existing_items}
    for item in new_items:
        merged[item.get("seed_id", item["id"])] = item
    return sorted(merged.values(), key=priority_key)


def outputs_for_item(queue_path: Path, item: Dict) -> Dict[str, Path]:
    root = queue_path.parent
    outputs = item["outputs"]
    return {
        "candidate_manifest": (root / outputs["candidate_manifest"]).resolve(),
        "ingest_manifest": (root / outputs["ingest_manifest"]).resolve(),
        "claim_file": (root / outputs["claim_file"]).resolve(),
        "heartbeat_file": (root / outputs["heartbeat_file"]).resolve(),
    }


def inventory_row(queue_path: Path, item: Dict) -> Dict:
    paths = outputs_for_item(queue_path, item)
    candidate_count = count_lines(paths["candidate_manifest"])
    ingest_saved_count, ingest_error_count = count_ingest_saved(paths["ingest_manifest"])
    pending_ingest = max(candidate_count - ingest_saved_count, 0)
    has_query_evidence = (
        paths["candidate_manifest"].exists()
        or paths["claim_file"].exists()
        or paths["heartbeat_file"].exists()
    )
    return {
        "seed_id": item.get("seed_id", item["id"]),
        "keyword": item.get("keyword", ""),
        "category": item.get("category", ""),
        "priority": item.get("priority", ""),
        "candidate_count": candidate_count,
        "ingest_saved_count": ingest_saved_count,
        "ingest_error_count": ingest_error_count,
        "pending_ingest_count": pending_ingest,
        "has_query_evidence": has_query_evidence,
    }


def fallback_topic_from_item(item: Dict, *, new_id: str) -> Dict:
    return {
        "id": new_id,
        "keyword": item["keyword"],
        "category": item["category"],
        "priority": item["priority"],
        "zh_terms": [item["keyword"]],
        "en_terms": [item["keyword"]],
        "prefer_domains": item.get("prefer_domains", []),
        "notes": item.get("notes", ""),
    }


def relaxed_years_for_item(item: Dict, *, relaxed_year_from: int, foundation_year_from: int) -> Tuple[int, str]:
    category = item.get("category", "")
    if category.startswith("architecture.") or category in {"design.reference", "design.narrative"}:
        return foundation_year_from, "foundation-history"
    return relaxed_year_from, "history-plus"


def build_relaxed_item(
    *,
    queue: Dict,
    base_item: Dict,
    inventory: Dict,
    source_topics: Dict[str, Dict],
    relaxed_year_from: int,
    foundation_year_from: int,
) -> Dict:
    base_seed_id = base_item.get("seed_id", base_item["id"])
    new_seed_id = f"{base_seed_id}-historical"
    historical_year_from, relax_mode = relaxed_years_for_item(
        base_item,
        relaxed_year_from=relaxed_year_from,
        foundation_year_from=foundation_year_from,
    )

    source_topic = deepcopy(source_topics.get(base_seed_id) or fallback_topic_from_item(base_item, new_id=new_seed_id))
    source_topic["id"] = new_seed_id
    relaxed_item = build_topic_item(
        source_topic,
        run_id=queue["run_id"],
        year_from=historical_year_from,
        year_to=base_item.get("year_to", queue.get("generated_at", "")[:4] or "2026"),
        target_count=base_item.get("target_article_count", 250),
        protocol_version=queue.get("protocol_version", "kc.v1"),
    )
    relaxed_item["keyword"] = base_item["keyword"]
    relaxed_item["notes"] = (
        f"{base_item.get('notes', '').strip()} "
        f"[historical pass: {relax_mode}, existing_candidates={inventory['candidate_count']}]"
    ).strip()
    relaxed_item["source_seed_id"] = base_seed_id
    relaxed_item["query_expansion"] = {
        "mode": relax_mode,
        "trigger": "candidate_count_below_threshold",
        "existing_candidate_count": inventory["candidate_count"],
        "historical_year_from": historical_year_from,
        "historical_year_to": relaxed_item["year_to"],
    }
    relaxed_item["status"]["phase"] = "seeded-historical"
    return relaxed_item


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Prepare a formal knowledge-harvest execution plan.")
    parser.add_argument("--queue", default=DEFAULT_QUEUE, help=f"Base seed queue. Default: {DEFAULT_QUEUE}")
    parser.add_argument("--relaxed-queue", default=DEFAULT_RELAXED_QUEUE, help=f"Historical fallback queue output. Default: {DEFAULT_RELAXED_QUEUE}")
    parser.add_argument("--output-plan", default=DEFAULT_PLAN, help=f"Execution plan output path. Default: {DEFAULT_PLAN}")
    parser.add_argument("--relax-threshold", type=int, default=DEFAULT_RELAX_THRESHOLD, help=f"Create a historical pass when candidate_count is below this threshold after a real query attempt. Default: {DEFAULT_RELAX_THRESHOLD}")
    parser.add_argument("--relaxed-year-from", type=int, default=DEFAULT_RELAXED_YEAR_FROM, help=f"Historical pass start year for standard topics. Default: {DEFAULT_RELAXED_YEAR_FROM}")
    parser.add_argument("--foundation-year-from", type=int, default=DEFAULT_FOUNDATION_YEAR_FROM, help=f"Historical pass start year for foundational narrative / architecture topics. Default: {DEFAULT_FOUNDATION_YEAR_FROM}")
    parser.add_argument("--append-expansion-topics", action="store_true", help="Append the broader 2D/design/monetization topics into the base queue when missing.")
    parser.add_argument("--do-not-close-before", default=DEFAULT_DO_NOT_CLOSE_BEFORE, help=f"Operational guardrail for the active plan. Default: {DEFAULT_DO_NOT_CLOSE_BEFORE}")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    queue_path = Path(args.queue).resolve()
    relaxed_queue_path = Path(args.relaxed_queue).resolve()
    plan_path = Path(args.output_plan).resolve()

    queue = load_json(queue_path)
    source_topics = topic_lookup(queue.get("profile", ""))
    year_from = min(item.get("year_from", 2024) for item in queue.get("items", []) or [{"year_from": 2024}])
    year_to = max(item.get("year_to", 2026) for item in queue.get("items", []) or [{"year_to": 2026}])
    target_count = max(item.get("target_article_count", 250) for item in queue.get("items", []) or [{"target_article_count": 250}])

    appended_ids: List[str] = []
    if args.append_expansion_topics:
        existing_ids = {item.get("seed_id", item["id"]) for item in queue.get("items", [])}
        new_items: List[Dict] = []
        for topic_id in EXPANSION_TOPIC_IDS:
            if topic_id in existing_ids:
                continue
            topic = source_topics.get(topic_id)
            if not topic:
                continue
            new_items.append(
                build_topic_item(
                    topic,
                    run_id=queue["run_id"],
                    year_from=year_from,
                    year_to=year_to,
                    target_count=target_count,
                    protocol_version=queue.get("protocol_version", "kc.v1"),
                )
            )
            appended_ids.append(topic_id)
        if new_items:
            queue["items"] = merge_items(queue.get("items", []), new_items)
            queue["generated_at"] = now_iso()
            queue["notes"] = (
                "Seed queue for supervisor -> worker knowledge harvesting. "
                "Paths are relative to this queue file unless absolute."
            )
            write_json(queue_path, queue)

    inventory = [inventory_row(queue_path, item) for item in queue.get("items", [])]
    inventory_by_seed = {row["seed_id"]: row for row in inventory}

    core_seed_ids = [row["seed_id"] for row in inventory if row["seed_id"] not in EXPANSION_TOPIC_IDS]
    expansion_seed_ids = [row["seed_id"] for row in inventory if row["seed_id"] in EXPANSION_TOPIC_IDS]

    relaxed_items: List[Dict] = []
    item_by_seed = {item.get("seed_id", item["id"]): item for item in queue.get("items", [])}
    for row in inventory:
        if not row["has_query_evidence"]:
            continue
        if row["candidate_count"] >= args.relax_threshold:
            continue
        base_item = item_by_seed[row["seed_id"]]
        relaxed_items.append(
            build_relaxed_item(
                queue=queue,
                base_item=base_item,
                inventory=row,
                source_topics=source_topics,
                relaxed_year_from=args.relaxed_year_from,
                foundation_year_from=args.foundation_year_from,
            )
        )

    relaxed_queue = {
        "protocol_version": queue.get("protocol_version", "kc.v1"),
        "version": 1,
        "profile": f"{queue.get('profile', 'custom')}-historical",
        "run_id": queue["run_id"],
        "generated_at": now_iso(),
        "notes": (
            "Historical fallback queue for sparse seeds. "
            "Generated only for seeds with real query evidence and low candidate counts."
        ),
        "items": sorted(relaxed_items, key=priority_key),
    }
    write_json(relaxed_queue_path, relaxed_queue)

    plan_payload = {
        "protocol_version": queue.get("protocol_version", "kc.v1"),
        "run_id": queue["run_id"],
        "status": "active",
        "planned_at": now_iso(),
        "do_not_close_before": args.do_not_close_before,
        "base_queue": str(queue_path),
        "historical_queue": str(relaxed_queue_path),
        "appended_expansion_topic_ids": appended_ids,
        "inventory": inventory,
        "phases": [
            {
                "id": "phase-01-core-query",
                "status": "active",
                "queue": str(queue_path),
                "seed_ids": core_seed_ids,
                "goal": "Run the summarized keyword set first.",
            },
            {
                "id": "phase-02-core-ingest",
                "status": "active",
                "queue": str(queue_path),
                "seed_ids": [row["seed_id"] for row in inventory if row["pending_ingest_count"] > 0],
                "goal": "Continuously ingest finished candidate backlogs into Obsidian.",
            },
            {
                "id": "phase-03-core-historical-query",
                "status": "planned",
                "queue": str(relaxed_queue_path),
                "seed_ids": [item.get("source_seed_id", item["seed_id"]) for item in relaxed_items if item.get("source_seed_id") in core_seed_ids],
                "goal": "If summarized keywords are sparse after a real run, expand beyond the recent three-year window.",
            },
            {
                "id": "phase-04-expansion-query",
                "status": "planned",
                "queue": str(queue_path),
                "seed_ids": expansion_seed_ids,
                "goal": "After the summarized keywords, continue with broader game-related topics: 2D dev, design, monetization, retention.",
            },
            {
                "id": "phase-05-expansion-historical-query",
                "status": "planned",
                "queue": str(relaxed_queue_path),
                "seed_ids": [item.get("source_seed_id", item["seed_id"]) for item in relaxed_items if item.get("source_seed_id") in expansion_seed_ids],
                "goal": "Historical fallback for the broader game-related topics when result counts stay low.",
            },
        ],
        "next_commands": {
            "plan_base_query_shards": [
                "python3",
                "tools/plan_research_shards.py",
                "--queue",
                str(queue_path),
            ],
            "plan_historical_query_shards": [
                "python3",
                "tools/plan_research_shards.py",
                "--queue",
                str(relaxed_queue_path),
                "--output",
                str((queue_path.parent / "query-shards-historical.json").resolve()),
            ],
            "plan_ingest_shards": [
                "python3",
                "tools/plan_ingest_shards.py",
                "--queue",
                str(queue_path),
            ],
        },
    }
    write_json(plan_path, plan_payload)

    print(f"Base queue: {queue_path}")
    print(f"Historical queue: {relaxed_queue_path}")
    print(f"Execution plan: {plan_path}")
    print(f"Inventory rows: {len(inventory)}")
    print(f"Historical fallback items: {len(relaxed_items)}")
    if appended_ids:
        print("Appended expansion topics:")
        for topic_id in appended_ids:
            print(f"  - {topic_id}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
