from __future__ import annotations

import json

from common import emit_json, load_payload, repo_is_git


payload = load_payload()
command = str(payload.get("tool_input", {}).get("command", "")).strip()
tool_response = payload.get("tool_response")

if isinstance(tool_response, str):
    response_text = tool_response
else:
    response_text = json.dumps(tool_response, ensure_ascii=False)

response_lower = response_text.lower()

if "not a git repository" in response_lower and not repo_is_git():
    message = (
        "This workspace is not a git repository. Do not rely on git add/commit/reset flows here; "
        "use direct file inspection and local path references instead."
    )
    emit_json(
        {
            "decision": "block",
            "reason": message,
            "hookSpecificOutput": {
                "hookEventName": "PostToolUse",
                "additionalContext": message,
            },
            "systemMessage": message,
        }
    )
    raise SystemExit(0)

if command.startswith("git ") and "fatal:" in response_lower:
    message = "Git command failed in this workspace. Re-evaluate whether git is actually available before continuing with git-based assumptions."
    emit_json(
        {
            "decision": "block",
            "reason": message,
            "hookSpecificOutput": {
                "hookEventName": "PostToolUse",
                "additionalContext": message,
            },
            "systemMessage": message,
        }
    )
    raise SystemExit(0)
