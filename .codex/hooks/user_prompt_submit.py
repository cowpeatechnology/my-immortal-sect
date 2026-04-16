from __future__ import annotations

from common import compact_project_context, emit_json, is_planning_stage, load_payload


payload = load_payload()
prompt = str(payload.get("prompt", "")).strip()

extra = []

if prompt:
    extra.append("Default operating model: keep the main thread in supervisor mode and assign explicit owners before substantial cross-domain work.")

cross_domain_keywords = [
    "前后端",
    "产品",
    "测试",
    "工作流",
    "团队",
    "架构",
    "系统",
    "pipeline",
    "workflow",
    "frontend",
    "backend",
    "qa",
    "design",
]

if any(token in prompt.lower() for token in [k.lower() for k in cross_domain_keywords]):
    extra.append(
        "For cross-functional work, route through the specialist roster: gameplay_designer, technical_architect, client_engineer, server_engineer, art_asset_producer, and qa_verifier. Use feature_worker or tools_engineer only for bounded short-lived execution."
    )
    extra.append(
        "For non-trivial iterative work, maintain docs/project/delivery-ledger.md and the relevant docs/features/F-xxx-<slug>.md record alongside the implementation."
    )
    extra.append(
        "If the work is runnable, require runtime handoff details in both work orders and worker handoffs: startup commands, environment prerequisites, QA URL/entry, validation steps, and expected results."
    )

art_keywords = [
    "美术",
    "资源",
    "立绘",
    "建筑",
    "角色图",
    "出图",
    "prompt",
    "asset",
    "art",
    "image",
]

if any(token.lower() in prompt.lower() for token in art_keywords):
    extra.append(
        "Prefer the persistent art_asset_producer role for recurring asset-production requests. Only use tools_engineer when the task is about the workflow or automation itself, not the art deliverable."
    )

if is_planning_stage():
    stage_keywords = [
        "初始化",
        "脚手架",
        "go mod init",
        "npm init",
        "pnpm init",
        "yarn init",
        "bun init",
        "client/",
        "server/",
        "shared/",
    ]
    if any(token.lower() in prompt.lower() for token in stage_keywords):
        extra.append(
            "Repository initialization should still be tied to an explicit milestone or supervisor work order, not done casually."
        )

extra.append(compact_project_context())

emit_json(
    {
        "hookSpecificOutput": {
            "hookEventName": "UserPromptSubmit",
            "additionalContext": " ".join(extra),
        }
    }
)
