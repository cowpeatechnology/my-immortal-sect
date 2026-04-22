#!/usr/bin/env python3
"""Local bridge for a Chrome extension that automates ChatGPT with an existing login."""

from __future__ import annotations

import argparse
import base64
import json
import re
import shutil
import sys
import threading
import time
import uuid
from dataclasses import dataclass
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any, Optional
from urllib.parse import parse_qs, urlparse


DEFAULT_HOST = "127.0.0.1"
DEFAULT_PORT = 8765
DEFAULT_QUEUE_ROOT = Path.home() / ".codex" / "chatgpt-chrome-bridge"
CLAIM_TIMEOUT_SECONDS = 300
RESULT_WAIT_INTERVAL_SECONDS = 1.0


class BridgeError(RuntimeError):
    """Raised when the Chrome bridge operation fails."""


def now_epoch_ms() -> int:
    return int(time.time() * 1000)


def iso_timestamp() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def json_dump(data: Any) -> str:
    return json.dumps(data, ensure_ascii=False, indent=2) + "\n"


@dataclass
class QueuePaths:
    root: Path

    @property
    def pending(self) -> Path:
        return self.root / "pending"

    @property
    def inflight(self) -> Path:
        return self.root / "inflight"

    @property
    def results(self) -> Path:
        return self.root / "results"

    @property
    def logs(self) -> Path:
        return self.root / "logs"

    @property
    def event_log(self) -> Path:
        return self.logs / "bridge-events.ndjson"

    def ensure(self) -> None:
        self.pending.mkdir(parents=True, exist_ok=True)
        self.inflight.mkdir(parents=True, exist_ok=True)
        self.results.mkdir(parents=True, exist_ok=True)
        self.logs.mkdir(parents=True, exist_ok=True)

    def pending_file(self, command_id: str, created_at: int) -> Path:
        return self.pending / f"{created_at:013d}__{command_id}.json"

    def inflight_file(self, command_id: str) -> Path:
        return self.inflight / f"{command_id}.json"

    def result_file(self, command_id: str) -> Path:
        return self.results / f"{command_id}.json"


class CommandQueue:
    def __init__(self, root: Path):
        self.paths = QueuePaths(root=root)
        self.paths.ensure()
        self._lock = threading.Lock()

    def _append_event_log(self, payload: dict[str, Any]) -> None:
        record = {
            "appended_at": now_epoch_ms(),
            "appended_at_iso": iso_timestamp(),
            **payload,
        }
        with self.paths.event_log.open("a", encoding="utf-8") as handle:
            handle.write(json.dumps(record, ensure_ascii=False) + "\n")

    def enqueue(self, command_type: str, payload: dict[str, Any]) -> dict[str, Any]:
        command_id = str(uuid.uuid4())
        created_at = now_epoch_ms()
        command = {
            "id": command_id,
            "type": command_type,
            "payload": payload,
            "status": "pending",
            "created_at": created_at,
            "updated_at": created_at,
            "attempt": 0,
        }
        path = self.paths.pending_file(command_id, created_at)
        path.write_text(json_dump(command), encoding="utf-8")
        with self._lock:
            self._append_event_log({
                "kind": "command_enqueued",
                "command_id": command_id,
                "command_type": command_type,
                "payload": payload,
            })
        return command

    def _list_json_files(self, folder: Path) -> list[Path]:
        return sorted(path for path in folder.glob("*.json") if path.is_file())

    def _read_json(self, path: Path) -> dict[str, Any]:
        return json.loads(path.read_text(encoding="utf-8"))

    def _write_json(self, path: Path, data: dict[str, Any]) -> None:
        path.write_text(json_dump(data), encoding="utf-8")

    def reclaim_expired(self) -> None:
        now_ms = now_epoch_ms()
        for inflight_file in self._list_json_files(self.paths.inflight):
            command = self._read_json(inflight_file)
            claimed_at = int(command.get("claimed_at", 0))
            if claimed_at <= 0:
                continue
            if (now_ms - claimed_at) < CLAIM_TIMEOUT_SECONDS * 1000:
                continue

            command["status"] = "pending"
            command["updated_at"] = now_ms
            command["reclaimed_at"] = now_ms
            command["attempt"] = int(command.get("attempt", 0)) + 1
            created_at = int(command.get("created_at", now_ms))
            pending_path = self.paths.pending_file(command["id"], created_at)
            self._write_json(pending_path, command)
            inflight_file.unlink(missing_ok=True)

    def claim_next(self, client_id: str) -> Optional[dict[str, Any]]:
        with self._lock:
            self.reclaim_expired()
            pending_files = self._list_json_files(self.paths.pending)
            if not pending_files:
                return None

            command_path = pending_files[0]
            command = self._read_json(command_path)
            command["status"] = "inflight"
            command["claimed_by"] = client_id
            command["claimed_at"] = now_epoch_ms()
            command["updated_at"] = command["claimed_at"]
            inflight_path = self.paths.inflight_file(command["id"])
            self._write_json(inflight_path, command)
            command_path.unlink(missing_ok=True)
            self._append_event_log({
                "kind": "command_claimed",
                "command_id": command["id"],
                "command_type": command.get("type"),
                "client_id": client_id,
                "claimed_at": command["claimed_at"],
            })
            return command

    def record_result(self, result: dict[str, Any]) -> dict[str, Any]:
        command_id = result.get("command_id")
        if not isinstance(command_id, str) or not command_id:
            raise BridgeError("Result payload missing command_id.")

        with self._lock:
            inflight_path = self.paths.inflight_file(command_id)
            command: dict[str, Any] = {}
            if inflight_path.exists():
                command = self._read_json(inflight_path)
                inflight_path.unlink(missing_ok=True)

            record = {
                "command": command,
                "result": result,
                "received_at": now_epoch_ms(),
                "received_at_iso": iso_timestamp(),
            }
            result_path = self.paths.result_file(command_id)
            self._write_json(result_path, record)
            self._append_event_log({
                "kind": "result",
                "command_id": command_id,
                "client_id": result.get("client_id"),
                "status": result.get("status"),
                "error": result.get("error"),
                "payload": result.get("payload"),
            })
            return record

    def update_progress(self, progress: dict[str, Any]) -> dict[str, Any]:
        command_id = progress.get("command_id")
        if not isinstance(command_id, str) or not command_id:
            raise BridgeError("Progress payload missing command_id.")

        with self._lock:
            inflight_path = self.paths.inflight_file(command_id)
            if not inflight_path.exists():
                raise BridgeError(f"Inflight command not found for progress update: {command_id}")

            command = self._read_json(inflight_path)
            command["progress"] = progress
            command["updated_at"] = now_epoch_ms()
            self._write_json(inflight_path, command)
            self._append_event_log({
                "kind": "progress",
                "command_id": command_id,
                "client_id": progress.get("client_id"),
                "stage": progress.get("stage"),
                "timestamp": progress.get("timestamp"),
                "details": progress.get("details"),
            })
            return command

    def append_log_event(self, event: dict[str, Any]) -> dict[str, Any]:
        with self._lock:
            self._append_event_log({
                "kind": str(event.get("kind") or "event"),
                **event,
            })
        return event

    def wait_for_result(self, command_id: str, timeout_seconds: int) -> dict[str, Any]:
        deadline = time.time() + timeout_seconds
        result_path = self.paths.result_file(command_id)
        while time.time() < deadline:
            if result_path.exists():
                return self._read_json(result_path)
            time.sleep(RESULT_WAIT_INTERVAL_SECONDS)
        raise BridgeError(f"Timed out waiting for result of command {command_id}.")


class BridgeHandler(BaseHTTPRequestHandler):
    queue: CommandQueue

    def _set_headers(self, status_code: int = 200, content_type: str = "application/json") -> None:
        self.send_response(status_code)
        self.send_header("Content-Type", content_type)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.end_headers()

    def _write_json(self, status_code: int, payload: dict[str, Any]) -> None:
        self._set_headers(status_code)
        self.wfile.write(json.dumps(payload, ensure_ascii=False).encode("utf-8"))

    def _read_json_body(self) -> dict[str, Any]:
        content_length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(content_length) if content_length > 0 else b"{}"
        if not body:
            return {}
        return json.loads(body.decode("utf-8"))

    def do_OPTIONS(self) -> None:  # noqa: N802
        self._set_headers(204)

    def do_GET(self) -> None:  # noqa: N802
        parsed = urlparse(self.path)
        if parsed.path == "/api/health":
            self._write_json(200, {"ok": True, "now": iso_timestamp()})
            return

        if parsed.path == "/api/next":
            query = parse_qs(parsed.query)
            client_id = query.get("client_id", ["chrome-extension"])[0]
            command = self.queue.claim_next(client_id)
            self._write_json(200, {"ok": True, "command": command})
            return

        self._write_json(404, {"ok": False, "error": "not_found"})

    def do_POST(self) -> None:  # noqa: N802
        parsed = urlparse(self.path)
        if parsed.path == "/api/progress":
            try:
                body = self._read_json_body()
                command = self.queue.update_progress(body)
                self._write_json(200, {"ok": True, "updated": True, "command_id": body.get("command_id"), "command": command})
            except Exception as exc:
                self._write_json(400, {"ok": False, "error": str(exc)})
            return

        if parsed.path == "/api/log":
            try:
                body = self._read_json_body()
                event = self.queue.append_log_event(body)
                self._write_json(200, {"ok": True, "logged": True, "event": event})
            except Exception as exc:
                self._write_json(400, {"ok": False, "error": str(exc)})
            return

        if parsed.path == "/api/result":
            try:
                body = self._read_json_body()
                record = self.queue.record_result(body)
                self._write_json(200, {"ok": True, "recorded": True, "command_id": body.get("command_id"), "record": record})
            except Exception as exc:
                self._write_json(400, {"ok": False, "error": str(exc)})
            return

        self._write_json(404, {"ok": False, "error": "not_found"})

    def log_message(self, format: str, *args: Any) -> None:  # noqa: A003
        return


def create_server(queue: CommandQueue, host: str, port: int) -> ThreadingHTTPServer:
    handler = type("BoundBridgeHandler", (BridgeHandler,), {})
    handler.queue = queue
    return ThreadingHTTPServer((host, port), handler)


def mime_to_extension(mime_type: str, fallback: Path) -> str:
    suffix = fallback.suffix.lower()
    if suffix:
        return suffix
    if "jpeg" in mime_type or "jpg" in mime_type:
        return ".jpg"
    if "webp" in mime_type:
        return ".webp"
    return ".png"


def slugify_filename(text: str, *, default: str = "image") -> str:
    slug = re.sub(r"[^a-zA-Z0-9_-]+", "-", text.strip().lower()).strip("-_")
    return slug[:80] or default


def build_output_base(*, prompt: str, output: str | None, output_dir: str | None, basename: str | None) -> Path:
    if output:
        output_path = Path(output).expanduser().resolve()
        if output_path.exists() and output_path.is_dir():
            directory = output_path
            stem = basename or slugify_filename(prompt)
            return directory / stem
        if output_path.suffix:
            return output_path.with_suffix("")
        return output_path

    if output_dir:
        directory = Path(output_dir).expanduser().resolve()
        stem = basename or f"{slugify_filename(prompt)}-{time.strftime('%Y%m%d-%H%M%S')}"
        return directory / stem

    raise BridgeError("Either --output or --output-dir must be provided.")


def save_images_from_result(result_record: dict[str, Any], output_base: Path) -> list[Path]:
    result = result_record.get("result", {})
    payload = result.get("payload", {})
    downloads = payload.get("downloads", [])
    if isinstance(downloads, list) and downloads:
        output_base.parent.mkdir(parents=True, exist_ok=True)
        saved_paths: list[Path] = []
        multiple = len(downloads) > 1
        for index, download in enumerate(downloads, start=1):
            if not isinstance(download, dict):
                continue
            source_value = download.get("path")
            if not isinstance(source_value, str) or not source_value:
                continue
            source_path = Path(source_value).expanduser()
            if not source_path.exists():
                raise BridgeError(f"Downloaded file not found: {source_path}")
            extension = mime_to_extension(str(download.get("mimeType", "")), source_path)
            target = output_base.with_suffix(extension) if not multiple else output_base.with_name(f"{output_base.name}-{index}{extension}")
            shutil.copy2(source_path, target)
            saved_paths.append(target)
        if saved_paths:
            return saved_paths

    images = payload.get("images", [])
    if not isinstance(images, list) or not images:
        raise BridgeError("Result did not contain any images.")

    output_base.parent.mkdir(parents=True, exist_ok=True)
    saved_paths: list[Path] = []
    multiple = len(images) > 1
    for index, image in enumerate(images, start=1):
        if not isinstance(image, dict):
            continue
        base64_data = image.get("base64")
        mime_type = image.get("mimeType", "image/png")
        if not isinstance(base64_data, str) or not base64_data:
            continue
        extension = mime_to_extension(str(mime_type), output_base)
        target = output_base.with_suffix(extension) if not multiple else output_base.with_name(f"{output_base.name}-{index}{extension}")
        target.write_bytes(base64.b64decode(base64_data))
        saved_paths.append(target)

    if not saved_paths:
        raise BridgeError("No decodable images were returned.")
    return saved_paths


def queue_root_from_args(args: argparse.Namespace) -> Path:
    return Path(args.queue_root).expanduser().resolve()


def run_serve(args: argparse.Namespace) -> None:
    queue = CommandQueue(queue_root_from_args(args))
    server = create_server(queue=queue, host=args.host, port=args.port)
    print(f"ChatGPT Chrome bridge listening on http://{args.host}:{args.port}")
    print(f"Queue root: {queue.paths.root}")
    print(f"Event log: {queue.paths.event_log}")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nShutting down bridge server.")
    finally:
        server.server_close()


def run_status(args: argparse.Namespace) -> None:
    queue = CommandQueue(queue_root_from_args(args))
    command = queue.enqueue("status", {})
    record = queue.wait_for_result(command["id"], timeout_seconds=args.timeout)
    print(json.dumps(record, ensure_ascii=False, indent=2))


def run_generate(args: argparse.Namespace) -> None:
    queue = CommandQueue(queue_root_from_args(args))
    command = queue.enqueue(
        "generate_image",
        {
            "prompt": args.prompt,
            "timeout_ms": args.generation_timeout_ms,
            "post_completion_settle_ms": args.post_completion_settle_ms,
        },
    )
    record = queue.wait_for_result(command["id"], timeout_seconds=args.wait_timeout)
    result = record.get("result", {})
    status = result.get("status")
    if status != "ok":
        raise BridgeError(result.get("error") or "Extension reported an error.")
    output_base = build_output_base(
        prompt=args.prompt,
        output=args.output,
        output_dir=args.output_dir,
        basename=args.basename,
    )
    saved = save_images_from_result(record, output_base)
    payload = result.get("payload", {})
    generation = payload.get("generation", {})
    print("Generation succeeded.")
    if isinstance(generation, dict):
        if generation.get("completed_at"):
            print(f"Completed at: {generation['completed_at']}")
        if generation.get("final_image_count") is not None:
            print(f"Image count: {generation['final_image_count']}")
        if generation.get("capture_method"):
            print(f"Capture method: {generation['capture_method']}")
        if generation.get("capture_url"):
            print(f"Capture URL: {generation['capture_url']}")
        if generation.get("downloaded_file"):
            print(f"Downloaded file: {generation['downloaded_file']}")
    for path in saved:
        print(f"Saved: {path}")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Local queue and HTTP bridge for a ChatGPT Chrome extension."
    )
    parser.add_argument(
        "--queue-root",
        default=str(DEFAULT_QUEUE_ROOT),
        help=f"Queue directory root. Default: {DEFAULT_QUEUE_ROOT}",
    )

    sub = parser.add_subparsers(dest="command", required=True)

    serve = sub.add_parser("serve", help="Run the localhost bridge server.")
    serve.add_argument("--host", default=DEFAULT_HOST, help=f"Bind host. Default: {DEFAULT_HOST}")
    serve.add_argument("--port", type=int, default=DEFAULT_PORT, help=f"Bind port. Default: {DEFAULT_PORT}")

    status = sub.add_parser("status", help="Ask the extension for current ChatGPT page status.")
    status.add_argument("--timeout", type=int, default=30, help="Wait timeout in seconds.")

    generate = sub.add_parser("generate", help="Ask the extension to generate images and save them locally.")
    generate.add_argument("--prompt", required=True, help="Prompt text to send to ChatGPT.")
    generate.add_argument("--output", help="Output image path or existing directory.")
    generate.add_argument("--output-dir", help="Directory to save generated images into.")
    generate.add_argument("--basename", help="Base filename to use with --output-dir or directory output.")
    generate.add_argument(
        "--wait-timeout",
        type=int,
        default=600,
        help="How long to wait for the extension to finish, in seconds.",
    )
    generate.add_argument(
        "--generation-timeout-ms",
        type=int,
        default=600000,
        help="How long the content script should wait for new images, in milliseconds.",
    )
    generate.add_argument(
        "--post-completion-settle-ms",
        type=int,
        default=12000,
        help="Extra settle time after the page appears finished, in milliseconds.",
    )

    return parser


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()

    if args.command == "generate" and not (args.output or args.output_dir):
        parser.error("generate requires --output or --output-dir")

    if args.command == "serve":
        run_serve(args)
    elif args.command == "status":
        run_status(args)
    elif args.command == "generate":
        run_generate(args)
    else:
        parser.error("Unknown command")


if __name__ == "__main__":
    try:
        main()
    except BridgeError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        raise SystemExit(1)
