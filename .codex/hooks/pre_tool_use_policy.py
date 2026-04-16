from __future__ import annotations

import re

from common import emit_json, load_payload


payload = load_payload()
command = str(payload.get("tool_input", {}).get("command", "")).strip()

forbidden_patterns = [
    (r"\bgit\s+reset\s+--hard\b", "Do not discard workspace history or edits with git reset --hard."),
    (r"\bgit\s+checkout\s+--\b", "Do not discard tracked file changes with git checkout --."),
    (r"\bgit\s+clean\s+-fdx?\b", "Do not delete untracked workspace files with git clean."),
    (r"\brm\s+-rf\s+(\./)?tools(/\b|$)", "tools/ is protected. Only change it when the user explicitly asks for tools work."),
    (r"\brm\s+-rf\s+(\./)?hifi-prototype(/\b|$)", "hifi-prototype/ is protected. Only change it when the user explicitly asks for prototype work."),
    (r"\brm\s+-rf\s+/\b", "Destructive root-level deletion is forbidden."),
]

for pattern, reason in forbidden_patterns:
    if re.search(pattern, command):
        emit_json(
            {
                "hookSpecificOutput": {
                    "hookEventName": "PreToolUse",
                    "permissionDecision": "deny",
                    "permissionDecisionReason": reason,
                },
                "systemMessage": reason,
            }
        )
        raise SystemExit(0)
