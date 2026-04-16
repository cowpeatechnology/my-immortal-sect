#!/usr/bin/env python3
"""Serial JSON-driven batch image generation on top of the ChatGPT CDP observer V3."""

from __future__ import annotations

import argparse
import json
import re
import sys
import time
from pathlib import Path
from typing import Any

from PIL import Image, ImageOps

from chatgpt_cdp_client_v3 import (
    ClientError,
    DEFAULT_SERVER_BASE,
    generate_and_capture,
)


try:
    RESAMPLE_LANCZOS = Image.Resampling.LANCZOS
except AttributeError:  # Pillow < 10
    RESAMPLE_LANCZOS = Image.LANCZOS


SUPPORTED_FORMATS = {"png", "jpeg", "webp"}


def utc_timestamp() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def read_json(path: Path) -> dict[str, Any]:
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise ClientError("Batch manifest root must be a JSON object.")
    return data


def write_json(path: Path, payload: dict[str, Any]) -> None:
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def normalize_format(value: Any, *, default: str | None) -> str | None:
    raw = value if value is not None else default
    if raw in (None, "", "original", "keep"):
        return default
    result = str(raw).strip().lower()
    if result == "jpg":
        result = "jpeg"
    if result not in SUPPORTED_FORMATS:
        raise ClientError(f"Unsupported format: {raw}")
    return result


def parse_size(value: Any) -> tuple[int, int] | None:
    if value in (None, "", "original", "keep"):
        return None
    match = re.fullmatch(r"\s*(\d+)x(\d+)\s*", str(value))
    if not match:
        raise ClientError(f"Invalid size value: {value!r}. Expected WIDTHxHEIGHT, e.g. 512x512.")
    width = int(match.group(1))
    height = int(match.group(2))
    if width <= 0 or height <= 0:
        raise ClientError(f"Invalid size value: {value!r}.")
    return (width, height)


def format_to_suffix(image_format: str | None, fallback: Path) -> str:
    if image_format == "jpeg":
        return ".jpg"
    if image_format == "png":
        return ".png"
    if image_format == "webp":
        return ".webp"
    return fallback.suffix or ".png"


def slugify_filename(value: str, *, default: str) -> str:
    result = re.sub(r"[^a-zA-Z0-9._-]+", "-", str(value).strip()).strip("-")
    return result or default


def resolve_output_dir(manifest_path: Path, defaults: dict[str, Any], item: dict[str, Any]) -> Path:
    raw = item.get("output_dir", defaults.get("output_dir"))
    if not raw:
        raise ClientError("Missing output_dir. Set defaults.output_dir or item.output_dir in the manifest.")
    candidate = Path(str(raw)).expanduser()
    if not candidate.is_absolute():
        candidate = (manifest_path.parent / candidate).resolve()
    else:
        candidate = candidate.resolve()
    candidate.mkdir(parents=True, exist_ok=True)
    return candidate


def resolve_basename(item: dict[str, Any], index: int) -> str:
    raw = item.get("basename") or item.get("id") or item.get("name")
    if raw:
        return slugify_filename(str(raw), default=f"item-{index:03d}")
    return f"item-{index:03d}"


def resolve_bool(item: dict[str, Any], defaults: dict[str, Any], key: str, *, fallback: bool) -> bool:
    value = item.get(key, defaults.get(key, fallback))
    if isinstance(value, str):
        normalized = value.strip().lower()
        if normalized in {"1", "true", "yes", "on"}:
            return True
        if normalized in {"0", "false", "no", "off"}:
            return False
    return bool(value)


def resolve_int(item: dict[str, Any], defaults: dict[str, Any], key: str, *, fallback: int) -> int:
    value = item.get(key, defaults.get(key, fallback))
    try:
        return int(value)
    except (TypeError, ValueError) as exc:
        raise ClientError(f"Invalid integer for {key}: {value!r}") from exc


def derive_output_base(manifest_path: Path, defaults: dict[str, Any], item: dict[str, Any], index: int) -> Path:
    return resolve_output_dir(manifest_path, defaults, item) / resolve_basename(item, index)


def ensure_items(manifest: dict[str, Any]) -> list[dict[str, Any]]:
    items = manifest.get("items")
    if not isinstance(items, list) or not items:
        raise ClientError("Batch manifest must contain a non-empty items array.")
    normalized: list[dict[str, Any]] = []
    for index, item in enumerate(items, start=1):
        if not isinstance(item, dict):
            raise ClientError(f"items[{index - 1}] must be an object.")
        normalized.append(item)
    return normalized


def suffix_to_format(path: Path) -> str:
    suffix = path.suffix.lower()
    if suffix in {".jpg", ".jpeg"}:
        return "jpeg"
    if suffix == ".webp":
        return "webp"
    return "png"


def render_image_variant(source: Path, target: Path, *, size: tuple[int, int] | None, image_format: str | None) -> None:
    target.parent.mkdir(parents=True, exist_ok=True)
    requested_format = image_format or suffix_to_format(source)
    if requested_format not in SUPPORTED_FORMATS:
        raise ClientError(f"Unsupported target format for {target.name}: {requested_format}")

    with Image.open(source) as original:
        use_alpha = requested_format in {"png", "webp"}
        working = original.convert("RGBA" if use_alpha else "RGB")
        if size:
            fitted = ImageOps.contain(working, size, RESAMPLE_LANCZOS)
            canvas = Image.new("RGBA" if use_alpha else "RGB", size, (0, 0, 0, 0) if use_alpha else (255, 255, 255))
            offset = ((size[0] - fitted.width) // 2, (size[1] - fitted.height) // 2)
            if use_alpha and fitted.mode == "RGBA":
                canvas.alpha_composite(fitted, dest=offset)
            else:
                canvas.paste(fitted, offset)
            output_image = canvas
        else:
            output_image = working

        if requested_format == "jpeg" and output_image.mode != "RGB":
            flattened = Image.new("RGB", output_image.size, (255, 255, 255))
            flattened.paste(output_image, mask=output_image.getchannel("A") if "A" in output_image.getbands() else None)
            output_image = flattened

        save_kwargs: dict[str, Any] = {}
        if requested_format == "jpeg":
            save_kwargs["quality"] = 95
        elif requested_format == "webp":
            save_kwargs["quality"] = 95
        elif requested_format == "png":
            save_kwargs["compress_level"] = 6

        temp_target = target.with_name(f"{target.name}.tmp")
        output_image.save(temp_target, format=requested_format.upper(), **save_kwargs)
        temp_target.replace(target)


def normalize_outputs(
    saved_paths: list[Path],
    *,
    output_base: Path,
    size: tuple[int, int] | None,
    image_format: str | None,
) -> list[Path]:
    multiple = len(saved_paths) > 1
    final_paths: list[Path] = []

    for index, source in enumerate(saved_paths, start=1):
        source = source.resolve()
        stem = output_base.name if not multiple else f"{output_base.name}-{index}"
        suffix = format_to_suffix(image_format, source)
        target = output_base.with_name(f"{stem}{suffix}")

        if size or image_format or source != target:
            render_image_variant(source, target, size=size, image_format=image_format)
            if source != target and source.exists():
                source.unlink()
        final_paths.append(target)

    return final_paths


def item_is_done(item: dict[str, Any]) -> bool:
    if item.get("status") != "done":
        return False
    output_path = item.get("output_path")
    return isinstance(output_path, str) and Path(output_path).expanduser().exists()


def update_item_state(item: dict[str, Any], *, status: str, error_message: str | None = None, extra: dict[str, Any] | None = None) -> None:
    item["status"] = status
    item["error_message"] = error_message
    item["updated_at"] = utc_timestamp()
    if extra:
        item.update(extra)


def execute_item(
    manifest_path: Path,
    defaults: dict[str, Any],
    item: dict[str, Any],
    *,
    index: int,
) -> dict[str, Any]:
    prompt = str(item.get("prompt") or "").strip()
    if not prompt:
        raise ClientError(f"items[{index - 1}] is missing prompt.")

    output_base = derive_output_base(manifest_path, defaults, item, index)
    server_base = str(item.get("server_base", defaults.get("server_base", DEFAULT_SERVER_BASE))).rstrip("/")
    wait_timeout_seconds = resolve_int(item, defaults, "wait_timeout_seconds", fallback=900)
    wait_poll_timeout_ms = resolve_int(item, defaults, "wait_poll_timeout_ms", fallback=30000)
    idle_timeout_seconds = resolve_int(item, defaults, "idle_timeout_seconds", fallback=120)
    idle_poll_interval_seconds = resolve_int(item, defaults, "idle_poll_interval_seconds", fallback=2)
    size = parse_size(item.get("size", defaults.get("size")))
    image_format = normalize_format(item.get("format"), default=normalize_format(defaults.get("format"), default="png"))
    label = str(item.get("label") or item.get("basename") or item.get("id") or item.get("name") or output_base.name)

    job, saved_path = generate_and_capture(
        server_base=server_base,
        prompt=prompt,
        output_dir=output_base.parent,
        basename=output_base.name,
        label=label,
        wait_timeout_seconds=wait_timeout_seconds,
        wait_poll_timeout_ms=wait_poll_timeout_ms,
        idle_timeout_seconds=idle_timeout_seconds,
        idle_poll_interval_seconds=idle_poll_interval_seconds,
    )
    final_paths = normalize_outputs(
        [saved_path],
        output_base=output_base,
        size=size,
        image_format=image_format,
    )
    primary_path = str(final_paths[0]) if final_paths else None
    return {
        "job_id": job["id"],
        "output_path": primary_path,
        "output_paths": [str(path) for path in final_paths],
    }


def run(args: argparse.Namespace) -> None:
    manifest_path = Path(args.manifest).expanduser().resolve()
    manifest = read_json(manifest_path)
    items = ensure_items(manifest)
    defaults = manifest.get("defaults", {})
    if not isinstance(defaults, dict):
        raise ClientError("defaults must be a JSON object when provided.")

    continue_on_error = resolve_bool({}, defaults, "continue_on_error", fallback=False)
    skip_done = resolve_bool({}, defaults, "skip_done", fallback=True)

    for index, item in enumerate(items, start=1):
        if skip_done and item_is_done(item):
            continue

        update_item_state(item, status="running", error_message=None)
        write_json(manifest_path, manifest)
        display_name = str(item.get("name") or item.get("id") or f"item-{index:03d}")
        print(f"[{index}/{len(items)}] generating {display_name}")

        try:
            result = execute_item(manifest_path, defaults, item, index=index)
            update_item_state(
                item,
                status="done",
                error_message=None,
                extra={
                    "output_path": result["output_path"],
                    "output_paths": result["output_paths"],
                    "last_job_id": result["job_id"],
                },
            )
            print(f"  saved: {result['output_path']}")
        except Exception as exc:
            update_item_state(item, status="error", error_message=str(exc))
            write_json(manifest_path, manifest)
            print(f"  error: {exc}", file=sys.stderr)
            if not continue_on_error:
                raise
        else:
            write_json(manifest_path, manifest)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Run serial JSON batch generation through the ChatGPT CDP observer V3."
    )
    parser.add_argument("manifest", help="Path to the batch manifest JSON file.")
    return parser


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()
    run(args)


if __name__ == "__main__":
    try:
        main()
    except ClientError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        raise SystemExit(1)
