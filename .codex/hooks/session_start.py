from __future__ import annotations

from common import compact_project_context, emit_json


emit_json(
    {
        "hookSpecificOutput": {
            "hookEventName": "SessionStart",
            "additionalContext": compact_project_context(),
        }
    }
)
