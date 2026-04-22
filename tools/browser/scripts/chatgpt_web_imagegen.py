#!/usr/bin/env python3
"""Generate and download ChatGPT web images through Playwright CLI automation."""

from __future__ import annotations

import argparse
import base64
import json
import os
import re
import subprocess
import sys
import time
from pathlib import Path
from typing import Any


DEFAULT_SESSION = "chatgpt-imagegen"
DEFAULT_URL = "https://chatgpt.com/"
DEFAULT_LOGIN_TIMEOUT = 300
DEFAULT_GENERATION_TIMEOUT = 600


class ChatGPTWebImageGenError(RuntimeError):
    """Raised when ChatGPT web generation fails."""


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Generate images through the ChatGPT web UI using Playwright CLI."
    )
    parser.add_argument("--prompt", required=True, help="Prompt to send to ChatGPT.")
    parser.add_argument("--output", required=True, help="Output image path.")
    parser.add_argument(
        "--session",
        default=DEFAULT_SESSION,
        help=f"Playwright browser session name. Default: {DEFAULT_SESSION}",
    )
    parser.add_argument(
        "--url",
        default=DEFAULT_URL,
        help=f"ChatGPT URL to open. Default: {DEFAULT_URL}",
    )
    parser.add_argument(
        "--login-timeout",
        type=int,
        default=DEFAULT_LOGIN_TIMEOUT,
        help=f"Seconds to wait for manual login. Default: {DEFAULT_LOGIN_TIMEOUT}",
    )
    parser.add_argument(
        "--generation-timeout",
        type=int,
        default=DEFAULT_GENERATION_TIMEOUT,
        help=f"Seconds to wait for generated images. Default: {DEFAULT_GENERATION_TIMEOUT}",
    )
    parser.add_argument(
        "--headless",
        action="store_true",
        help="Run the browser headlessly. By default the browser is visible.",
    )
    parser.add_argument(
        "--keep-open",
        action="store_true",
        help="Keep the browser session open after completion.",
    )
    return parser


def pwcli_path() -> str:
    codex_home = os.environ.get("CODEX_HOME") or str(Path.home() / ".codex")
    path = Path(codex_home) / "skills" / "playwright" / "scripts" / "playwright_cli.sh"
    if not path.exists():
        raise ChatGPTWebImageGenError(f"Playwright wrapper not found: {path}")
    return str(path)


def run_pwcli(
    session: str,
    *args: str,
    raw: bool = False,
    check: bool = True,
) -> str:
    command = [pwcli_path(), "--session", session]
    if raw:
        command.append("--raw")
    command.extend(args)

    result = subprocess.run(command, capture_output=True, text=True)
    output = result.stdout.strip()
    error = result.stderr.strip()

    if check and result.returncode != 0:
        detail = output or error or f"exit code {result.returncode}"
        raise ChatGPTWebImageGenError(f"Playwright CLI failed for {' '.join(args)}: {detail}")

    return output or error


def close_session(session: str) -> None:
    run_pwcli(session, "close", check=False)


def open_session(session: str, url: str, headless: bool) -> None:
    args = ["open", url]
    if not headless:
        args.append("--headed")
    run_pwcli(session, *args)


def goto_url(session: str, url: str) -> None:
    run_pwcli(session, "goto", url)


def snapshot(session: str) -> str:
    return run_pwcli(session, "snapshot")


def eval_json(session: str, func: str) -> Any:
    raw = run_pwcli(session, "eval", func, raw=True)
    try:
        return json.loads(raw)
    except json.JSONDecodeError as exc:
        raise ChatGPTWebImageGenError(f"Failed to parse eval output as JSON: {raw}") from exc


def click(session: str, ref: str) -> None:
    run_pwcli(session, "click", ref)


def fill(session: str, ref: str, text: str) -> None:
    run_pwcli(session, "fill", ref, text)


def press(session: str, key: str) -> None:
    run_pwcli(session, "press", key)


def regex_ref(snapshot_text: str, patterns: list[str]) -> str | None:
    for pattern in patterns:
        match = re.search(pattern, snapshot_text, re.MULTILINE)
        if match:
            return match.group(1)
    return None


def login_button_ref(snapshot_text: str) -> str | None:
    return regex_ref(
        snapshot_text,
        [
            r'button "登录" \[ref=(e\d+)\]',
            r'button "Log in" \[ref=(e\d+)\]',
            r'link "登录" \[ref=(e\d+)\]',
            r'link "Log in" \[ref=(e\d+)\]',
        ],
    )


def composer_ref(snapshot_text: str) -> str | None:
    return regex_ref(
        snapshot_text,
        [
            r'textbox "[^"]*" \[active\] \[ref=(e\d+)\]',
            r'textbox "[^"]*" \[ref=(e\d+)\]',
        ],
    )


def login_required(snapshot_text: str) -> bool:
    if login_button_ref(snapshot_text):
        return True
    return "登录以获取基于已保存聊天的回答" in snapshot_text or "Log in to get smarter responses" in snapshot_text


def wait_for_login(session: str, timeout_seconds: int) -> None:
    page = snapshot(session)
    if not login_required(page):
        return

    ref = login_button_ref(page)
    if ref:
        try:
            click(session, ref)
        except ChatGPTWebImageGenError:
            pass

    print("ChatGPT web session is not logged in.", file=sys.stderr)
    print("Please complete login in the opened browser window.", file=sys.stderr)

    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        time.sleep(2)
        page = snapshot(session)
        if not login_required(page):
            return

    raise ChatGPTWebImageGenError("Login was not completed before timeout.")


def current_large_images(session: str) -> list[dict[str, Any]]:
    result = eval_json(
        session,
        "() => Array.from(document.querySelectorAll('img')).map((img) => ({"
        "src: img.currentSrc || img.src || '',"
        "width: img.naturalWidth || img.width || 0,"
        "height: img.naturalHeight || img.height || 0,"
        "visibleWidth: Math.round(img.getBoundingClientRect().width || 0),"
        "visibleHeight: Math.round(img.getBoundingClientRect().height || 0),"
        "alt: img.alt || ''"
        "})).filter((img) => img.src && !img.src.startsWith('data:image/svg') && "
        "((img.width >= 256 || img.visibleWidth >= 256) && (img.height >= 256 || img.visibleHeight >= 256)))",
    )
    if not isinstance(result, list):
        return []

    unique: list[dict[str, Any]] = []
    seen: set[str] = set()
    for item in result:
        if not isinstance(item, dict):
            continue
        src = item.get("src")
        if not isinstance(src, str) or not src or src in seen:
            continue
        seen.add(src)
        unique.append(item)
    return unique


def wait_for_images(session: str, before_sources: set[str], timeout_seconds: int) -> list[dict[str, Any]]:
    deadline = time.time() + timeout_seconds
    previous_sources: list[str] = []
    stable_hits = 0

    while time.time() < deadline:
        images = current_large_images(session)
        fresh = [item for item in images if item["src"] not in before_sources]
        current_sources = [item["src"] for item in fresh]

        if current_sources:
            if current_sources == previous_sources:
                stable_hits += 1
            else:
                previous_sources = current_sources
                stable_hits = 1

            if stable_hits >= 2:
                return fresh

        time.sleep(2)

    raise ChatGPTWebImageGenError("Timed out waiting for generated images.")


def fetch_image_payload(session: str, src: str) -> dict[str, str]:
    func = (
        "() => fetch("
        + json.dumps(src)
        + ").then((response) => response.blob()).then(async (blob) => {"
        "const bytes = new Uint8Array(await blob.arrayBuffer());"
        "const chunkSize = 0x8000;"
        "let binary = '';"
        "for (let index = 0; index < bytes.length; index += chunkSize) {"
        "binary += String.fromCharCode(...bytes.subarray(index, index + chunkSize));"
        "}"
        "return { mime: blob.type || 'image/png', base64: btoa(binary) };"
        "})"
    )
    result = eval_json(session, func)
    if not isinstance(result, dict):
        raise ChatGPTWebImageGenError("Image payload fetch returned an invalid response.")
    base64_value = result.get("base64")
    mime = result.get("mime")
    if not isinstance(base64_value, str) or not base64_value:
        raise ChatGPTWebImageGenError("Image payload did not include base64 data.")
    if not isinstance(mime, str) or not mime:
        mime = "image/png"
    return {"mime": mime, "base64": base64_value}


def extension_for_mime(mime: str, fallback: str) -> str:
    suffix = Path(fallback).suffix.lower()
    if suffix:
        return suffix
    if "jpeg" in mime or "jpg" in mime:
        return ".jpg"
    if "webp" in mime:
        return ".webp"
    return ".png"


def save_images(images: list[dict[str, Any]], output_path: Path, session: str) -> list[Path]:
    output_path.parent.mkdir(parents=True, exist_ok=True)
    saved: list[Path] = []

    for index, image in enumerate(images, start=1):
        payload = fetch_image_payload(session, image["src"])
        extension = extension_for_mime(payload["mime"], str(output_path))
        if len(images) == 1:
            target = output_path.with_suffix(extension)
        else:
            target = output_path.with_name(f"{output_path.stem}-{index}{extension}")
        target.write_bytes(base64.b64decode(payload["base64"]))
        saved.append(target)

    return saved


def debug_screenshot(session: str, output_path: Path) -> None:
    target = output_path.with_name(f"{output_path.stem}-debug.png")
    target.parent.mkdir(parents=True, exist_ok=True)
    run_pwcli(session, "screenshot", "--filename", str(target), check=False)
    print(f"Debug screenshot hint: {target}", file=sys.stderr)


def main() -> None:
    args = build_parser().parse_args()
    output_path = Path(args.output).expanduser().resolve()

    close_session(args.session)
    open_session(args.session, args.url, args.headless)

    try:
        wait_for_login(args.session, args.login_timeout)
        goto_url(args.session, args.url)

        before_sources = {item["src"] for item in current_large_images(args.session)}
        page = snapshot(args.session)
        ref = composer_ref(page)
        if not ref:
            raise ChatGPTWebImageGenError("Could not find the ChatGPT input textbox.")

        fill(args.session, ref, args.prompt)
        press(args.session, "Enter")

        images = wait_for_images(args.session, before_sources, args.generation_timeout)
        saved = save_images(images, output_path, args.session)

        print("Generation succeeded.")
        for path in saved:
            print(f"Saved: {path}")
    except Exception:
        debug_screenshot(args.session, output_path)
        raise
    finally:
        if not args.keep_open:
            close_session(args.session)


if __name__ == "__main__":
    try:
        main()
    except ChatGPTWebImageGenError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        raise SystemExit(1)
