from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Any


REPO_ROOT = Path(__file__).resolve().parents[2]
AGENTS_MD = REPO_ROOT / "AGENTS.md"
VISION_DOC = REPO_ROOT / "docs" / "vision" / "design-decisions.md"


def load_payload() -> dict[str, Any]:
    raw = sys.stdin.read().strip()
    if not raw:
        return {}
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return {}


def emit_json(payload: dict[str, Any]) -> None:
    sys.stdout.write(json.dumps(payload, ensure_ascii=False))


def repo_is_git() -> bool:
    return (REPO_ROOT / ".git").exists()


def is_planning_stage() -> bool:
    if not AGENTS_MD.exists():
        return False
    text = AGENTS_MD.read_text(encoding="utf-8")
    return "planning / pre-M0" in text or "Do not initialize code scaffolding" in text


def compact_project_context() -> str:
    lines = [
        "My Immortal Sect is a portrait 2.5D xianxia sect-management sim for WeChat/Douyin mini-games.",
        "Main thread acts as project supervisor; it owns intake, routing, integration, and acceptance instead of acting as a general-purpose worker.",
        "Default persistent role templates are gameplay_designer, technical_architect, client_engineer, server_engineer, art_asset_producer, and qa_verifier.",
        "feature_worker and tools_engineer are temporary workers for bounded execution and should normally be closed after handoff.",
        "Do not create a separate product_manager agent by default; the user is the real product owner and gameplay_designer covers ongoing feature-definition work.",
        "Current repo stage is planning / pre-M0. Do not initialize client/server/shared scaffolding unless the user explicitly starts M0 or explicitly requests initialization.",
        "Protected by default: hifi-prototype/ and tools/. docs/ is authoritative.",
    ]
    if not repo_is_git():
        lines.append("This workspace is not currently a git repository. Do not assume git commit/reset workflows are available.")
    return " ".join(lines)


def contains_any(text: str, needles: list[str]) -> bool:
    lower = text.lower()
    return any(needle.lower() in lower for needle in needles)
