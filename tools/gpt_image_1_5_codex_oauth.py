#!/usr/bin/env python3
"""Generate images through Codex OAuth using the ChatGPT Codex responses backend.

This tool reuses the local Codex desktop/CLI OAuth session stored in
``~/.codex/auth.json`` and sends a Responses-style request to the same backend
family Codex uses for authenticated runs.

It is intended for personal experimentation and local automation on the same
machine where Codex is already logged in.
"""

from __future__ import annotations

import argparse
import base64
import json
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Dict, Iterable, Iterator, List, Optional, Sequence

AUTH_FILE = Path.home() / ".codex" / "auth.json"
CLIENT_ID = "app_EMoamEEZ73f0CkXaXp7hrann"
TOKEN_URL = "https://auth.openai.com/oauth/token"
CODEX_RESPONSES_URL = "https://chatgpt.com/backend-api/codex/responses"
JWT_CLAIM_PATH = "https://api.openai.com/auth"

DEFAULT_TEXT_MODEL = "gpt-5.4"
DEFAULT_SIZE = "1024x1024"
DEFAULT_QUALITY = "high"
DEFAULT_BACKGROUND = "transparent"
DEFAULT_ACTION = "generate"
DEFAULT_FORMAT = "png"
DEFAULT_TIMEOUT = 600
DEFAULT_INSTRUCTIONS = "You are a concise image generation assistant."
REFRESH_SKEW_SECONDS = 300


class CodexOAuthImageError(RuntimeError):
    """Raised when authentication or generation fails."""


def decode_jwt_payload(token: str) -> Dict:
    parts = token.split(".")
    if len(parts) != 3:
        return {}

    payload_b64 = parts[1] + "=" * (-len(parts[1]) % 4)
    try:
        raw = base64.urlsafe_b64decode(payload_b64.encode("utf-8"))
        return json.loads(raw.decode("utf-8"))
    except Exception:
        return {}


def jwt_expiry_epoch(token: str) -> Optional[int]:
    payload = decode_jwt_payload(token)
    exp = payload.get("exp")
    return exp if isinstance(exp, int) else None


def extract_account_id(access_token: str) -> Optional[str]:
    payload = decode_jwt_payload(access_token)
    auth = payload.get(JWT_CLAIM_PATH, {})
    account_id = auth.get("chatgpt_account_id")
    return account_id if isinstance(account_id, str) and account_id else None


def load_codex_auth_file(path: Path) -> Dict:
    if not path.exists():
        raise CodexOAuthImageError(
            f"Codex auth file not found: {path}. Run `codex login` first."
        )

    data = json.loads(path.read_text(encoding="utf-8"))
    tokens = data.get("tokens")
    if not isinstance(tokens, dict):
        raise CodexOAuthImageError(f"Unexpected auth file format in {path}")

    access_token = tokens.get("access_token")
    refresh_token = tokens.get("refresh_token")
    id_token = tokens.get("id_token")
    if not all(isinstance(value, str) and value for value in (access_token, refresh_token, id_token)):
        raise CodexOAuthImageError(
            f"Missing access_token / refresh_token / id_token in {path}"
        )

    account_id = tokens.get("account_id")
    if not isinstance(account_id, str) or not account_id:
        account_id = extract_account_id(access_token)
    if not account_id:
        raise CodexOAuthImageError("Failed to determine chatgpt account id from auth file")

    return {
        "access_token": access_token,
        "refresh_token": refresh_token,
        "id_token": id_token,
        "account_id": account_id,
    }


def oauth_post_form(params: Dict[str, str]) -> Dict:
    body = urllib.parse.urlencode(params).encode("utf-8")
    req = urllib.request.Request(
        TOKEN_URL,
        method="POST",
        data=body,
        headers={"Content-Type": "application/x-www-form-urlencoded"},
    )
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise CodexOAuthImageError(
            f"Token endpoint HTTP {exc.code}: {detail}"
        ) from exc


def refresh_tokens(refresh_token: str) -> Dict:
    data = oauth_post_form(
        {
            "grant_type": "refresh_token",
            "refresh_token": refresh_token,
            "client_id": CLIENT_ID,
        }
    )
    if not data.get("access_token") or not data.get("refresh_token"):
        raise CodexOAuthImageError(f"Invalid refresh response: {data}")
    return data


def ensure_fresh_tokens(auth_state: Dict, force_refresh: bool = False) -> Dict:
    access_token = auth_state["access_token"]
    refresh_token_value = auth_state["refresh_token"]
    exp = jwt_expiry_epoch(access_token)
    now = int(time.time())
    needs_refresh = force_refresh or exp is None or now >= (exp - REFRESH_SKEW_SECONDS)

    if not needs_refresh:
        return auth_state

    refreshed = refresh_tokens(refresh_token_value)
    access_token = refreshed["access_token"]
    account_id = extract_account_id(access_token) or auth_state["account_id"]

    return {
        "access_token": access_token,
        "refresh_token": refreshed["refresh_token"],
        "id_token": refreshed.get("id_token") or auth_state["id_token"],
        "account_id": account_id,
    }


def iter_sse_data_chunks(raw_stream) -> Iterator[str]:
    buffer = ""
    while True:
        chunk = raw_stream.read(4096)
        if not chunk:
            break

        buffer += chunk.decode("utf-8", errors="replace")
        while "\n\n" in buffer:
            event_block, buffer = buffer.split("\n\n", 1)
            data_lines: List[str] = []
            for line in event_block.splitlines():
                if line.startswith("data:"):
                    data_lines.append(line[5:].strip())

            if not data_lines:
                continue

            data = "\n".join(data_lines).strip()
            if data and data != "[DONE]":
                yield data


def build_image_request_body(
    *,
    prompt: str,
    instructions: str,
    text_model: str,
    size: str,
    quality: str,
    background: str,
    image_format: str,
    compression: Optional[int],
    action: str,
) -> Dict:
    tool: Dict[str, object] = {
        "type": "image_generation",
        "size": size,
        "quality": quality,
        "background": background,
        "format": image_format,
        "action": action,
    }

    if compression is not None:
        tool["compression"] = compression

    return {
        "model": text_model,
        "store": False,
        "stream": True,
        "instructions": instructions,
        "input": [
            {
                "role": "user",
                "content": [
                    {
                        "type": "input_text",
                        "text": prompt,
                    }
                ],
            }
        ],
        "tools": [tool],
        "text": {"verbosity": "low"},
        "include": ["reasoning.encrypted_content"],
    }


def post_codex_response_stream(*, access_token: str, account_id: str, body: Dict, timeout: int) -> Iterator[Dict]:
    req = urllib.request.Request(
        CODEX_RESPONSES_URL,
        method="POST",
        data=json.dumps(body).encode("utf-8"),
        headers={
            "Authorization": f"Bearer {access_token}",
            "chatgpt-account-id": str(account_id),
            "OpenAI-Beta": "responses=experimental",
            "originator": "pi",
            "accept": "text/event-stream",
            "content-type": "application/json",
            "User-Agent": "codex-oauth-gpt-image-1-5-python/0.1",
        },
    )

    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            for data in iter_sse_data_chunks(resp):
                yield json.loads(data)
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise CodexOAuthImageError(
            f"Codex backend HTTP {exc.code}: {detail}"
        ) from exc


def extract_image_calls_from_output(output: Sequence[Dict]) -> List[Dict]:
    image_calls: List[Dict] = []
    for item in output:
        if isinstance(item, dict) and item.get("type") == "image_generation_call":
            image_calls.append(item)
    return image_calls


def collect_generation_result(
    *,
    access_token: str,
    account_id: str,
    body: Dict,
    timeout: int,
) -> Dict:
    completed_response: Optional[Dict] = None
    image_calls: List[Dict] = []
    text_parts: List[str] = []
    event_types: List[str] = []
    failed_message: Optional[str] = None

    for event in post_codex_response_stream(
        access_token=access_token,
        account_id=account_id,
        body=body,
        timeout=timeout,
    ):
        event_type = event.get("type")
        if isinstance(event_type, str):
            event_types.append(event_type)

        if event_type == "response.output_text.delta":
            delta = event.get("delta")
            if isinstance(delta, str):
                text_parts.append(delta)
            continue

        if event_type in ("response.completed", "response.done"):
            response = event.get("response")
            if isinstance(response, dict):
                completed_response = response
                image_calls.extend(extract_image_calls_from_output(response.get("output", [])))
            continue

        if event_type == "response.output_item.done":
            item = event.get("item")
            if isinstance(item, dict) and item.get("type") == "image_generation_call":
                image_calls.append(item)
            continue

        if event_type == "response.failed":
            err = event.get("response", {}).get("error", {})
            failed_message = err.get("message") or json.dumps(event, ensure_ascii=False)
            continue

        if event_type == "error":
            failed_message = event.get("message") or json.dumps(event, ensure_ascii=False)

    if completed_response is None and not image_calls and not failed_message:
        raise CodexOAuthImageError("The backend returned no completed response and no image output.")

    return {
        "response": completed_response or {},
        "image_calls": image_calls,
        "assistant_text": "".join(text_parts).strip(),
        "error_message": failed_message,
        "event_types": event_types,
    }


def guess_extension(image_format: str) -> str:
    normalized = image_format.lower()
    if normalized == "jpeg":
        return ".jpg"
    if normalized == "webp":
        return ".webp"
    return ".png"


def write_images(image_calls: Sequence[Dict], output_path: Path, image_format: str) -> List[Path]:
    output_path.parent.mkdir(parents=True, exist_ok=True)
    extension = guess_extension(image_format)

    if output_path.suffix.lower() != extension:
        output_path = output_path.with_suffix(extension)

    saved: List[Path] = []
    multiple = len(image_calls) > 1

    for index, image_call in enumerate(image_calls, start=1):
        result_b64 = image_call.get("result")
        if not isinstance(result_b64, str) or not result_b64:
            continue

        if multiple:
            target = output_path.with_name(f"{output_path.stem}-{index}{output_path.suffix}")
        else:
            target = output_path

        target.write_bytes(base64.b64decode(result_b64))
        saved.append(target)

    return saved


def run_generate(args: argparse.Namespace) -> None:
    auth_state = load_codex_auth_file(Path(args.auth_file).expanduser())
    auth_state = ensure_fresh_tokens(auth_state, force_refresh=args.force_refresh)

    body = build_image_request_body(
        prompt=args.prompt,
        instructions=args.instructions,
        text_model=args.text_model,
        size=args.size,
        quality=args.quality,
        background=args.background,
        image_format=args.format,
        compression=args.compression,
        action=args.action,
    )

    result = collect_generation_result(
        access_token=auth_state["access_token"],
        account_id=auth_state["account_id"],
        body=body,
        timeout=args.timeout,
    )

    image_calls = result["image_calls"]
    if not image_calls:
        response = result.get("response", {})
        if (
            isinstance(response, dict)
            and response.get("status") == "completed"
            and response.get("tools") == []
        ):
            detail = (
                "The Codex ChatGPT-backed responses surface accepted the request but did not expose "
                "the image_generation tool on this account/session. No image output was returned."
            )
        else:
            detail = result["error_message"] or result["assistant_text"] or "No image output returned."
        raise CodexOAuthImageError(detail)

    saved_files = write_images(
        image_calls=image_calls,
        output_path=Path(args.output).expanduser(),
        image_format=args.format,
    )

    if not saved_files:
        raise CodexOAuthImageError("Image generation completed but no decodable image payload was returned.")

    print("Generation succeeded.")
    print(f"Text model     : {args.text_model}")
    print(f"Account ID     : {auth_state['account_id']}")
    print(f"Image count    : {len(saved_files)}")
    print(f"Event count    : {len(result['event_types'])}")
    for path in saved_files:
        print(f"Saved          : {path}")

    for index, image_call in enumerate(image_calls, start=1):
        revised_prompt = image_call.get("revised_prompt")
        if isinstance(revised_prompt, str) and revised_prompt:
            print(f"Revised Prompt {index}: {revised_prompt}")

    if args.dump_response_json:
        dump_path = Path(args.dump_response_json).expanduser()
        dump_path.parent.mkdir(parents=True, exist_ok=True)
        dump_path.write_text(
            json.dumps(result["response"], ensure_ascii=False, indent=2) + "\n",
            encoding="utf-8",
        )
        print(f"Response JSON  : {dump_path}")

    if args.dump_events_json:
        dump_path = Path(args.dump_events_json).expanduser()
        dump_path.parent.mkdir(parents=True, exist_ok=True)
        dump_path.write_text(
            json.dumps(result["event_types"], ensure_ascii=False, indent=2) + "\n",
            encoding="utf-8",
        )
        print(f"Event Types    : {dump_path}")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Generate images through Codex OAuth and the Codex responses backend."
    )
    sub = parser.add_subparsers(dest="command", required=True)

    generate = sub.add_parser("generate", help="Generate one or more images")
    generate.add_argument("--prompt", required=True, help="The image prompt")
    generate.add_argument(
        "--output",
        required=True,
        help="Target image path. If multiple images are returned, numbered siblings are created.",
    )
    generate.add_argument(
        "--auth-file",
        default=str(AUTH_FILE),
        help=f"Codex auth file path. Defaults to {AUTH_FILE}",
    )
    generate.add_argument(
        "--text-model",
        default=DEFAULT_TEXT_MODEL,
        help=f"Main text model used with the image_generation tool. Default: {DEFAULT_TEXT_MODEL}",
    )
    generate.add_argument("--size", default=DEFAULT_SIZE, help=f"Image size. Default: {DEFAULT_SIZE}")
    generate.add_argument(
        "--instructions",
        default=DEFAULT_INSTRUCTIONS,
        help=f"Codex backend instructions string. Default: {DEFAULT_INSTRUCTIONS}",
    )
    generate.add_argument(
        "--quality",
        default=DEFAULT_QUALITY,
        choices=["auto", "low", "medium", "high"],
        help=f"Image quality. Default: {DEFAULT_QUALITY}",
    )
    generate.add_argument(
        "--background",
        default=DEFAULT_BACKGROUND,
        choices=["auto", "transparent", "opaque"],
        help=f"Background mode. Default: {DEFAULT_BACKGROUND}",
    )
    generate.add_argument(
        "--format",
        default=DEFAULT_FORMAT,
        choices=["png", "jpeg", "webp"],
        help=f"Output format. Default: {DEFAULT_FORMAT}",
    )
    generate.add_argument(
        "--compression",
        type=int,
        default=None,
        help="Optional JPEG/WebP compression level 0-100",
    )
    generate.add_argument(
        "--action",
        default=DEFAULT_ACTION,
        choices=["auto", "generate", "edit"],
        help=f"Image generation action. Default: {DEFAULT_ACTION}",
    )
    generate.add_argument(
        "--timeout",
        type=int,
        default=DEFAULT_TIMEOUT,
        help=f"HTTP timeout in seconds. Default: {DEFAULT_TIMEOUT}",
    )
    generate.add_argument(
        "--force-refresh",
        action="store_true",
        help="Refresh the OAuth token before sending the request",
    )
    generate.add_argument(
        "--dump-response-json",
        help="Optional path to save the completed response JSON",
    )
    generate.add_argument(
        "--dump-events-json",
        help="Optional path to save the observed SSE event type list",
    )

    return parser


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()

    if args.command == "generate":
        run_generate(args)
    else:
        parser.error("Unknown command")


if __name__ == "__main__":
    try:
        main()
    except CodexOAuthImageError as exc:
        print(f"ERROR: {exc}")
        raise SystemExit(1)
