#!/usr/bin/env python3
"""Supervisor for the dedicated-browser knowledge harvest flow."""

from __future__ import annotations

import argparse
import json
import logging
import os
import signal
import subprocess
import sys
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Optional

from google_browser_queue import build_run, build_status, load_json, now_iso


DEFAULT_POLL_SECONDS = 10
DEFAULT_STALE_SECONDS = 180
DEFAULT_BROWSER_VERSION_URL = "http://127.0.0.1:9333/json/version"
DEFAULT_SEARCH_WORKER_ID = "search-agent-01"
DEFAULT_INGEST_WORKER_ID = "ingest-agent-01"


class SupervisorError(RuntimeError):
    """Raised when the supervisor cannot continue."""


@dataclass
class WorkerProcess:
    name: str
    command: List[str]
    log_path: Path
    process: Optional[subprocess.Popen] = None
    started_at: str = ""
    last_restart_reason: str = ""


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run the dedicated-browser knowledge harvest supervisor.")
    parser.add_argument("--run-file", default="")
    parser.add_argument("--seed-queue", default="")
    parser.add_argument("--run-id", default="")
    parser.add_argument("--output-dir", default="")
    parser.add_argument("--seed-id", action="append", default=[])
    parser.add_argument("--max-queries", type=int)
    parser.add_argument("--vault", required=True)
    parser.add_argument("--project-folder", default="Projects/我的宗门 Wiki")
    parser.add_argument("--project-name", default="我的宗门")
    parser.add_argument("--browser-version-url", default=DEFAULT_BROWSER_VERSION_URL)
    parser.add_argument("--poll-seconds", type=int, default=DEFAULT_POLL_SECONDS)
    parser.add_argument("--stale-seconds", type=int, default=DEFAULT_STALE_SECONDS)
    parser.add_argument("--stop-when-idle", action="store_true")
    parser.add_argument("--max-runtime-seconds", type=int, default=0)
    parser.add_argument("--search-worker-id", default=DEFAULT_SEARCH_WORKER_ID)
    parser.add_argument("--ingest-worker-id", default=DEFAULT_INGEST_WORKER_ID)
    parser.add_argument("--search-results-min", type=int, default=5)
    parser.add_argument("--search-results-max", type=int, default=8)
    parser.add_argument("--search-max-pages", type=int, default=2)
    parser.add_argument("--search-preview-limit", type=int, default=12)
    parser.add_argument("--search-model", default="")
    parser.add_argument("--ingest-model", default="")
    parser.add_argument("--search-skip-llm-review", action="store_true")
    parser.add_argument("--ingest-skip-llm", action="store_true")
    parser.add_argument("--force-refresh", action="store_true")
    parser.add_argument("--ingest-poll-seconds", type=int, default=20)
    parser.add_argument("--ingest-idle-seconds", type=int, default=300)
    parser.add_argument("--log-level", default="INFO")
    return parser.parse_args()


def configure_logging(level: str) -> None:
    logging.basicConfig(
        level=getattr(logging, level.upper(), logging.INFO),
        format="%(asctime)s %(levelname)s %(message)s",
    )


def resolve_run_file(args: argparse.Namespace) -> Path:
    if args.run_file:
        run_file = Path(args.run_file).expanduser().resolve()
        if run_file.exists():
            return run_file
        raise SupervisorError(f"Run file does not exist: {run_file}")

    if not args.seed_queue or not args.run_id or not args.output_dir:
        raise SupervisorError("Either --run-file or (--seed-queue, --run-id, --output-dir) is required.")

    output_dir = Path(args.output_dir).expanduser().resolve()
    run_file = output_dir / "browser-run.json"
    if run_file.exists():
        return run_file

    build_run(
        seed_queue_path=Path(args.seed_queue).expanduser().resolve(),
        output_dir=output_dir,
        run_id=args.run_id,
        seed_ids=args.seed_id,
        max_queries=args.max_queries,
    )
    return run_file


def load_heartbeat(path: Path) -> Dict:
    if not path.exists():
        return {}
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        return {}


def parse_iso_epoch(value: str) -> float:
    if not value:
        return 0.0
    try:
        from datetime import datetime

        return datetime.fromisoformat(value.replace("Z", "+00:00")).timestamp()
    except Exception:
        return 0.0


def heartbeat_is_stale(path: Path, stale_seconds: int) -> bool:
    payload = load_heartbeat(path)
    updated_at = parse_iso_epoch(str(payload.get("updated_at") or ""))
    if not updated_at:
        return False
    return (time.time() - updated_at) > stale_seconds


def ensure_parent(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def write_json(path: Path, payload: Dict) -> None:
    ensure_parent(path)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def start_worker(worker: WorkerProcess) -> None:
    ensure_parent(worker.log_path)
    handle = worker.log_path.open("a", encoding="utf-8")
    worker.process = subprocess.Popen(
        worker.command,
        stdout=handle,
        stderr=subprocess.STDOUT,
        cwd=Path(__file__).resolve().parent.parent,
        text=True,
    )
    worker.started_at = now_iso()
    logging.info("Started %s pid=%s", worker.name, worker.process.pid)


def stop_worker(worker: WorkerProcess, *, reason: str) -> None:
    if worker.process is None:
        return
    proc = worker.process
    worker.last_restart_reason = reason
    if proc.poll() is None:
        logging.warning("Stopping %s pid=%s reason=%s", worker.name, proc.pid, reason)
        proc.terminate()
        try:
            proc.wait(timeout=10)
        except subprocess.TimeoutExpired:
            proc.kill()
            proc.wait(timeout=5)
    worker.process = None


def worker_running(worker: WorkerProcess) -> bool:
    return worker.process is not None and worker.process.poll() is None


def build_worker_commands(args: argparse.Namespace, run_file: Path, outputs: Dict[str, str], control_dir: Path) -> Dict[str, WorkerProcess]:
    python_bin = sys.executable
    tools_dir = Path(__file__).resolve().parent

    search_cmd = [
        python_bin,
        str(tools_dir / "browser_google_search_worker.py"),
        "--run-file",
        str(run_file),
        "--worker-id",
        args.search_worker_id,
        "--browser-version-url",
        args.browser_version_url,
        "--results-min",
        str(args.search_results_min),
        "--results-max",
        str(args.search_results_max),
        "--max-search-pages",
        str(args.search_max_pages),
        "--preview-limit",
        str(args.search_preview_limit),
    ]
    if args.search_model:
        search_cmd.extend(["--model", args.search_model])
    if args.force_refresh:
        search_cmd.append("--force-refresh")
    if args.search_skip_llm_review:
        search_cmd.append("--skip-llm-review")

    ingest_cmd = [
        python_bin,
        str(tools_dir / "wiki_ingest_queue_worker.py"),
        "--run-file",
        str(run_file),
        "--worker-id",
        args.ingest_worker_id,
        "--vault",
        args.vault,
        "--project-folder",
        args.project_folder,
        "--project-name",
        args.project_name,
        "--poll-seconds",
        str(args.ingest_poll_seconds),
        "--idle-seconds",
        str(args.ingest_idle_seconds),
    ]
    if args.ingest_model:
        ingest_cmd.extend(["--model", args.ingest_model])
    if args.force_refresh:
        ingest_cmd.append("--force-refresh")
    if args.ingest_skip_llm:
        ingest_cmd.append("--skip-llm")

    return {
        "search": WorkerProcess(
            name="search",
            command=search_cmd,
            log_path=control_dir / "logs" / f"{args.search_worker_id}.log",
        ),
        "ingest": WorkerProcess(
            name="ingest",
            command=ingest_cmd,
            log_path=control_dir / "logs" / f"{args.ingest_worker_id}.log",
        ),
    }


def should_start_search(status: Dict[str, int]) -> bool:
    return int(status.get("query_pending", 0)) > 0 or int(status.get("query_active", 0)) > 0


def should_start_ingest(status: Dict[str, int]) -> bool:
    return int(status.get("candidate_total", 0)) > int(status.get("ingest_total", 0))


def main() -> int:
    args = parse_args()
    configure_logging(args.log_level)
    run_file = resolve_run_file(args)
    run_payload = load_json(run_file)
    outputs = run_payload["outputs"]
    control_dir = run_file.parent / "control"
    heartbeat_path = control_dir / "supervisor-heartbeat.json"
    state_path = control_dir / "supervisor-state.json"
    summary_path = control_dir / "supervisor-summary.json"

    workers = build_worker_commands(args, run_file, outputs, control_dir)
    started_at = time.time()

    try:
        while True:
            status = build_status(run_file)
            search_hb_path = Path(outputs["search_heartbeat"]).resolve()
            ingest_hb_path = Path(outputs["ingest_heartbeat"]).resolve()

            if should_start_search(status):
                if not worker_running(workers["search"]):
                    start_worker(workers["search"])
                elif heartbeat_is_stale(search_hb_path, args.stale_seconds):
                    stop_worker(workers["search"], reason="stale-search-heartbeat")
                    start_worker(workers["search"])
            elif worker_running(workers["search"]):
                stop_worker(workers["search"], reason="no-search-work-left")

            if should_start_ingest(status):
                if not worker_running(workers["ingest"]):
                    start_worker(workers["ingest"])
                elif heartbeat_is_stale(ingest_hb_path, args.stale_seconds):
                    stop_worker(workers["ingest"], reason="stale-ingest-heartbeat")
                    start_worker(workers["ingest"])
            elif worker_running(workers["ingest"]):
                ingest_hb = load_heartbeat(ingest_hb_path)
                if str(ingest_hb.get("stage") or "") == "idle-timeout":
                    stop_worker(workers["ingest"], reason="idle-timeout")

            state_payload = {
                "run_file": str(run_file),
                "updated_at": now_iso(),
                "status": status,
                "workers": {
                    name: {
                        "running": worker_running(worker),
                        "pid": worker.process.pid if worker.process else None,
                        "started_at": worker.started_at,
                        "log_path": str(worker.log_path),
                        "last_restart_reason": worker.last_restart_reason,
                    }
                    for name, worker in workers.items()
                },
            }
            write_json(state_path, state_payload)
            write_json(
                heartbeat_path,
                {
                    "role": "supervisor",
                    "updated_at": now_iso(),
                    "run_file": str(run_file),
                    "status": status,
                    "search_worker_running": worker_running(workers["search"]),
                    "ingest_worker_running": worker_running(workers["ingest"]),
                },
            )

            if args.stop_when_idle:
                if (
                    int(status.get("query_pending", 0)) == 0
                    and int(status.get("query_active", 0)) == 0
                    and int(status.get("candidate_total", 0)) == int(status.get("ingest_total", 0))
                    and not worker_running(workers["search"])
                    and not worker_running(workers["ingest"])
                ):
                    write_json(
                        summary_path,
                        {
                            "completed_at": now_iso(),
                            "run_file": str(run_file),
                            "status": status,
                        },
                    )
                    logging.info("Supervisor exiting: run is idle and fully processed.")
                    return 0

            if args.max_runtime_seconds and (time.time() - started_at) >= args.max_runtime_seconds:
                write_json(
                    summary_path,
                    {
                        "completed_at": now_iso(),
                        "run_file": str(run_file),
                        "status": status,
                        "reason": "max-runtime-seconds",
                    },
                )
                logging.warning("Supervisor exiting due to max-runtime-seconds.")
                return 0

            time.sleep(args.poll_seconds)
    except KeyboardInterrupt:
        return 130
    finally:
        for worker in workers.values():
            stop_worker(worker, reason="supervisor-exit")


if __name__ == "__main__":
    raise SystemExit(main())
