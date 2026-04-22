#!/usr/bin/env python3
"""Client for the ChatGPT CDP observer V3 service."""

from __future__ import annotations

import argparse
import json
import sys
import time
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import quote, unquote
from urllib.request import Request, urlopen


DEFAULT_SERVER_BASE = "http://127.0.0.1:8776"
DEFAULT_WAIT_TIMEOUT_SECONDS = 360
DEFAULT_WAIT_POLL_TIMEOUT_MS = 30000
DEFAULT_OUTPUT_DIR = Path.cwd() / "workspace" / "output" / "browser" / "chatgpt-capture-v3-client"
DEFAULT_IDLE_TIMEOUT_SECONDS = 120
DEFAULT_IDLE_POLL_INTERVAL_SECONDS = 2


class ClientError(RuntimeError):
    """Raised when the V3 client fails."""


def json_dump(payload: Any) -> bytes:
    return (json.dumps(payload, ensure_ascii=False) + "\n").encode("utf-8")


def now_slug() -> str:
    return time.strftime("%Y-%m-%dT%H-%M-%SZ", time.gmtime())


def safe_stem(value: str, default: str) -> str:
    result = "".join(char if char.isalnum() or char in "._-" else "-" for char in value.strip())
    result = result.strip("-")
    return result or default


def request_json(
    method: str,
    url: str,
    *,
    payload: dict[str, Any] | None = None,
    timeout_seconds: int = 30,
) -> dict[str, Any]:
    body = json_dump(payload) if payload is not None else None
    request = Request(url, method=method.upper(), data=body)
    request.add_header("Accept", "application/json")
    if body is not None:
        request.add_header("Content-Type", "application/json")

    try:
        with urlopen(request, timeout=timeout_seconds) as response:
            content = response.read().decode("utf-8")
    except HTTPError as exc:
        try:
            content = exc.read().decode("utf-8")
        except Exception:  # pragma: no cover - defensive
            content = exc.reason or str(exc)
        raise ClientError(f"HTTP {exc.code} calling {url}: {content}") from exc
    except URLError as exc:
        raise ClientError(f"Failed to reach {url}: {exc.reason}") from exc

    try:
        data = json.loads(content)
    except json.JSONDecodeError as exc:
        raise ClientError(f"Invalid JSON from {url}: {content[:200]!r}") from exc

    if not isinstance(data, dict):
        raise ClientError(f"Unexpected payload from {url}: {type(data)!r}")
    return data


def download_result(url: str, *, timeout_seconds: int = 60) -> tuple[bytes, dict[str, str]]:
    request = Request(url, method="GET")
    try:
        with urlopen(request, timeout=timeout_seconds) as response:
            body = response.read()
            headers = {key.lower(): value for key, value in response.headers.items()}
    except HTTPError as exc:
        try:
            content = exc.read().decode("utf-8")
        except Exception:  # pragma: no cover - defensive
            content = exc.reason or str(exc)
        raise ClientError(f"HTTP {exc.code} downloading result from {url}: {content}") from exc
    except URLError as exc:
        raise ClientError(f"Failed to download result from {url}: {exc.reason}") from exc

    return body, headers


def suffix_from_headers(headers: dict[str, str], fallback_name: str | None) -> str:
    mime_type = headers.get("x-observer-mime-type") or headers.get("content-type") or ""
    file_name = headers.get("x-observer-file-name")
    if file_name:
        try:
            decoded = unquote(file_name)
        except Exception:  # pragma: no cover - defensive
            decoded = file_name
        suffix = Path(decoded).suffix
        if suffix:
            return suffix
    if fallback_name:
        suffix = Path(fallback_name).suffix
        if suffix:
            return suffix
    if "png" in mime_type:
        return ".png"
    if "jpeg" in mime_type or "jpg" in mime_type:
        return ".jpg"
    if "webp" in mime_type:
        return ".webp"
    return ".bin"


def write_bytes(target: Path, body: bytes) -> None:
    target.parent.mkdir(parents=True, exist_ok=True)
    temp_path = target.with_name(f"{target.name}.tmp")
    temp_path.write_bytes(body)
    temp_path.replace(target)


def register_job(server_base: str, *, label: str) -> dict[str, Any]:
    data = request_json(
        "POST",
        f"{server_base}/jobs/register",
        payload={"label": label},
    )
    job = data.get("job")
    if not data.get("ok") or not isinstance(job, dict):
        raise ClientError(f"Unexpected register response: {data!r}")
    return job


def wait_for_job(server_base: str, job: dict[str, Any], *, timeout_seconds: int, poll_timeout_ms: int) -> dict[str, Any]:
    deadline = time.time() + timeout_seconds
    current = job

    while time.time() < deadline:
        remaining = max(1, int(deadline - time.time()))
        timeout_ms = min(poll_timeout_ms, remaining * 1000)
        data = request_json(
            "GET",
            f"{server_base}/jobs/{quote(current['id'])}/wait?since_version={current['version']}&timeout_ms={timeout_ms}",
            timeout_seconds=max(30, remaining + 5),
        )
        updated_job = data.get("job")
        if isinstance(updated_job, dict):
            current = updated_job
        if data.get("shuttingDown"):
            raise ClientError("Observer server is shutting down.")
        if current.get("state") in {"ready", "failed", "acked"}:
            return current

    raise ClientError(f"Timed out waiting for job {current.get('id')} to complete.")


def ack_job(server_base: str, job_id: str) -> None:
    data = request_json(
        "POST",
        f"{server_base}/jobs/{quote(job_id)}/ack",
        payload={},
    )
    if not data.get("ok"):
        raise ClientError(f"Ack failed for job {job_id}: {data!r}")


def cancel_job(server_base: str, job_id: str) -> None:
    data = request_json(
        "POST",
        f"{server_base}/jobs/{quote(job_id)}/cancel",
        payload={},
    )
    if not data.get("ok"):
        raise ClientError(f"Cancel failed for job {job_id}: {data!r}")


def page_status(server_base: str) -> dict[str, Any]:
    data = request_json("GET", f"{server_base}/page/status")
    page = data.get("page")
    if not data.get("ok") or not isinstance(page, dict):
        raise ClientError(f"Unexpected page status response: {data!r}")
    return page


def debug_events(server_base: str, *, limit: int, event: str) -> dict[str, Any]:
    query = [f"limit={max(1, limit)}"]
    if event:
        query.append(f"event={quote(event)}")
    return request_json("GET", f"{server_base}/debug/events?{'&'.join(query)}")


def resolve_candidate_choice(
    server_base: str,
    *,
    timeout_seconds: int = 30,
    post_clear_settle_ms: int = 1500,
) -> dict[str, Any]:
    data = request_json(
        "POST",
        f"{server_base}/actions/resolve-candidate-choice",
        payload={
            "timeoutMs": max(1000, int(timeout_seconds * 1000)),
            "postClearSettleMs": max(0, int(post_clear_settle_ms)),
        },
        timeout_seconds=max(30, timeout_seconds + 5),
    )
    result = data.get("result")
    if not data.get("ok") or not isinstance(result, dict):
        raise ClientError(f"Unexpected resolve-candidate-choice response: {data!r}")
    return result


def page_status_is_idle(status: dict[str, Any]) -> bool:
    return (
        bool(status.get("composerFound"))
        and not bool(status.get("loginRequired"))
        and not bool(status.get("busyGenerating"))
        and not bool(status.get("candidateChoiceVisible"))
    )


def wait_for_idle_page(
    server_base: str,
    *,
    timeout_seconds: int,
    poll_interval_seconds: int,
) -> dict[str, Any]:
    deadline = time.time() + timeout_seconds
    last_status: dict[str, Any] | None = None
    previous_signature = ""
    stable_hits = 0

    while time.time() < deadline:
        status = page_status(server_base)
        last_status = status
        ready = page_status_is_idle(status)
        signature = json.dumps(
            {
                "url": status.get("url"),
                "title": status.get("title"),
                "composerFound": status.get("composerFound"),
                "loginRequired": status.get("loginRequired"),
                "busyGenerating": status.get("busyGenerating"),
                "candidateChoiceVisible": status.get("candidateChoiceVisible"),
                "progressText": status.get("progressText"),
            },
            ensure_ascii=False,
            sort_keys=True,
        )

        if ready and signature == previous_signature:
            stable_hits += 1
        else:
            previous_signature = signature
            stable_hits = 1 if ready else 0

        if ready and stable_hits >= 2:
            return status

        time.sleep(max(0.2, poll_interval_seconds))

    raise ClientError(
        "Timed out waiting for the ChatGPT worker page to become idle. "
        f"Last status: {json.dumps(last_status or {}, ensure_ascii=False)}"
    )


def ensure_idle_page(
    server_base: str,
    *,
    timeout_seconds: int,
    poll_interval_seconds: int,
) -> dict[str, Any]:
    status = page_status(server_base)
    if bool(status.get("candidateChoiceVisible")):
        resolve_candidate_choice(
            server_base,
            timeout_seconds=min(30, timeout_seconds),
        )
    return wait_for_idle_page(
        server_base,
        timeout_seconds=timeout_seconds,
        poll_interval_seconds=poll_interval_seconds,
    )


def send_prompt(server_base: str, prompt: str) -> dict[str, Any]:
    data = request_json(
        "POST",
        f"{server_base}/actions/send-prompt",
        payload={"prompt": prompt},
        timeout_seconds=60,
    )
    result = data.get("result")
    if not data.get("ok") or not isinstance(result, dict):
        raise ClientError(f"Unexpected send-prompt response: {data!r}")
    return result


def build_output_path(output_dir: Path, basename: str, suffix: str) -> Path:
    stem = safe_stem(basename, default=f"capture-{now_slug()}")
    return output_dir / f"{stem}{suffix}"


def capture_registered_job(
    server_base: str,
    job: dict[str, Any],
    *,
    output_dir: Path,
    basename: str,
    wait_timeout_seconds: int,
    wait_poll_timeout_ms: int,
) -> Path:
    job = wait_for_job(
        server_base,
        job,
        timeout_seconds=wait_timeout_seconds,
        poll_timeout_ms=wait_poll_timeout_ms,
    )

    if job.get("state") == "failed":
        error = job.get("error") or {}
        raise ClientError(f"Observer marked job failed: {error.get('code')}: {error.get('message')}")
    if job.get("state") != "ready":
        raise ClientError(f"Unexpected terminal job state: {job.get('state')}")

    body, headers = download_result(
        f"{server_base}/jobs/{quote(job['id'])}/result",
        timeout_seconds=max(60, wait_timeout_seconds),
    )
    result = job.get("result") or {}
    suffix = suffix_from_headers(headers, result.get("fileName"))
    target = build_output_path(output_dir, basename, suffix)
    write_bytes(target, body)
    ack_job(server_base, job["id"])
    return target


def generate_and_capture(
    *,
    server_base: str,
    prompt: str,
    output_dir: Path,
    basename: str,
    label: str,
    wait_timeout_seconds: int,
    wait_poll_timeout_ms: int,
    idle_timeout_seconds: int,
    idle_poll_interval_seconds: int,
) -> tuple[dict[str, Any], Path]:
    ensure_idle_page(
        server_base,
        timeout_seconds=idle_timeout_seconds,
        poll_interval_seconds=idle_poll_interval_seconds,
    )

    job = register_job(server_base, label=label)
    job_id = str(job["id"])

    try:
        send_prompt(server_base, prompt)
        target = capture_registered_job(
            server_base,
            job,
            output_dir=output_dir,
            basename=basename,
            wait_timeout_seconds=wait_timeout_seconds,
            wait_poll_timeout_ms=wait_poll_timeout_ms,
        )
        return job, target
    except Exception:
        try:
            cancel_job(server_base, job_id)
        except Exception:
            pass
        raise


def command_capture(args: argparse.Namespace) -> int:
    server_base = args.server_base.rstrip("/")
    output_dir = args.output_dir.resolve()
    label = args.label or args.basename or "capture-job"

    job = register_job(server_base, label=label)
    print(f"Registered job: {job['id']}")
    print(f"Waiting for next run on observer: {server_base}")

    target = capture_registered_job(
        server_base,
        job,
        output_dir=output_dir,
        basename=args.basename,
        wait_timeout_seconds=args.wait_timeout_seconds,
        wait_poll_timeout_ms=args.wait_poll_timeout_ms,
    )

    print(f"Saved: {target}")
    print(f"Job: {job['id']}")
    return 0


def command_generate(args: argparse.Namespace) -> int:
    server_base = args.server_base.rstrip("/")
    output_dir = args.output_dir.resolve()
    label = args.label or args.basename or "generate-job"
    job, target = generate_and_capture(
        server_base=server_base,
        prompt=args.prompt,
        output_dir=output_dir,
        basename=args.basename,
        label=label,
        wait_timeout_seconds=args.wait_timeout_seconds,
        wait_poll_timeout_ms=args.wait_poll_timeout_ms,
        idle_timeout_seconds=args.idle_timeout_seconds,
        idle_poll_interval_seconds=args.idle_poll_interval_seconds,
    )
    print(f"Saved: {target}")
    print(f"Job: {job['id']}")
    return 0


def command_status(args: argparse.Namespace) -> int:
    server_base = args.server_base.rstrip("/")
    data = request_json("GET", f"{server_base}/healthz")
    print(json.dumps(data, ensure_ascii=False, indent=2))
    return 0


def command_events(args: argparse.Namespace) -> int:
    server_base = args.server_base.rstrip("/")
    data = debug_events(
        server_base,
        limit=args.limit,
        event=args.event,
    )
    print(json.dumps(data, ensure_ascii=False, indent=2))
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="command", required=True)

    capture = subparsers.add_parser("capture", help="Register one capture job, wait, download result, then ack.")
    capture.add_argument("--server-base", default=DEFAULT_SERVER_BASE, help=f"Observer base URL. Default: {DEFAULT_SERVER_BASE}")
    capture.add_argument("--output-dir", type=Path, default=DEFAULT_OUTPUT_DIR, help=f"Directory for the final file. Default: {DEFAULT_OUTPUT_DIR}")
    capture.add_argument("--basename", default="capture-v3", help="Target output basename without suffix.")
    capture.add_argument("--label", default="", help="Optional observer-side label for logs/debug.")
    capture.add_argument("--wait-timeout-seconds", type=int, default=DEFAULT_WAIT_TIMEOUT_SECONDS, help=f"Overall timeout. Default: {DEFAULT_WAIT_TIMEOUT_SECONDS}")
    capture.add_argument("--wait-poll-timeout-ms", type=int, default=DEFAULT_WAIT_POLL_TIMEOUT_MS, help=f"Single long-poll timeout. Default: {DEFAULT_WAIT_POLL_TIMEOUT_MS}")
    capture.set_defaults(func=command_capture)

    generate = subparsers.add_parser("generate", help="Wait for idle page, register a job, submit a prompt, capture result, then ack.")
    generate.add_argument("--server-base", default=DEFAULT_SERVER_BASE, help=f"Observer base URL. Default: {DEFAULT_SERVER_BASE}")
    generate.add_argument("--prompt", required=True, help="Prompt to submit to the attached ChatGPT page.")
    generate.add_argument("--output-dir", type=Path, default=DEFAULT_OUTPUT_DIR, help=f"Directory for the final file. Default: {DEFAULT_OUTPUT_DIR}")
    generate.add_argument("--basename", default="generate-v3", help="Target output basename without suffix.")
    generate.add_argument("--label", default="", help="Optional observer-side label for logs/debug.")
    generate.add_argument("--wait-timeout-seconds", type=int, default=DEFAULT_WAIT_TIMEOUT_SECONDS, help=f"Overall timeout. Default: {DEFAULT_WAIT_TIMEOUT_SECONDS}")
    generate.add_argument("--wait-poll-timeout-ms", type=int, default=DEFAULT_WAIT_POLL_TIMEOUT_MS, help=f"Single long-poll timeout. Default: {DEFAULT_WAIT_POLL_TIMEOUT_MS}")
    generate.add_argument("--idle-timeout-seconds", type=int, default=DEFAULT_IDLE_TIMEOUT_SECONDS, help=f"Timeout for pre-submit idle wait. Default: {DEFAULT_IDLE_TIMEOUT_SECONDS}")
    generate.add_argument("--idle-poll-interval-seconds", type=int, default=DEFAULT_IDLE_POLL_INTERVAL_SECONDS, help=f"Page idle poll interval. Default: {DEFAULT_IDLE_POLL_INTERVAL_SECONDS}")
    generate.set_defaults(func=command_generate)

    status = subparsers.add_parser("status", help="Read observer status.")
    status.add_argument("--server-base", default=DEFAULT_SERVER_BASE, help=f"Observer base URL. Default: {DEFAULT_SERVER_BASE}")
    status.set_defaults(func=command_status)

    events = subparsers.add_parser("events", help="Read recent observer debug event summaries.")
    events.add_argument("--server-base", default=DEFAULT_SERVER_BASE, help=f"Observer base URL. Default: {DEFAULT_SERVER_BASE}")
    events.add_argument("--limit", type=int, default=20, help="Max events to return. Default: 20")
    events.add_argument("--event", default="", help="Optional event name filter, e.g. job_ready")
    events.set_defaults(func=command_events)

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ClientError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        raise SystemExit(1)
