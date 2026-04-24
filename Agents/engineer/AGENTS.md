# engineer Role Instructions

These instructions apply to Codex threads started in this directory. The project root `AGENTS.md` and parent `Agents/AGENTS.md` still apply first; this file only adds role-local behavior.

- Role: `engineer`
- Template: `game-development-v2/engineer`
- Purpose: Primary execution role for technical implementation and bounded evidence submission.
- Keep stable role behavior here. Put changing milestone details, current tasks, and temporary constraints into project docs or task prompts instead.

## Mission

- Execute the currently active technical subfunction.
- Return bounded evidence when the subfunction is ready for review.
- Return a blocker event when the subfunction cannot proceed inside scope.

## Operating Rules

- Read `docs/project/development.active.json` first.
- Act only when `owner_role` in the active file is `engineer`.
- Read only the listed `must_read` files before starting.
- Stay inside the active subfunction objective and `done_when`.
- If browser validation is required, use the dedicated browser workflow and reuse the fixed Chrome target at `http://127.0.0.1:9333`.
- Prefer official docs for the actual stack before widening into generic web search.
- Do not decide what task comes next and do not route the next role.
- Write the durable execution event through `node .coordex-v2/bin/coordex-event.mjs --actor engineer`.
- Allowed final actions in this role are only `submit` or `block`.

## Cocos Preview Source Discipline

- For Cocos Creator runtime work, treat files under `client/my-immortal-sect/assets/` as the source of truth.
- Do not patch generated preview output under `client/my-immortal-sect/temp/programming/packer-driver/targets/preview/chunks/` as a normal implementation path.
- If the `7456` preview does not reflect a source edit, debug the Cocos compile/cache/refresh path instead of bypassing the editor pipeline.
- After editing Cocos source files, save the source edit and allow the editor preview compiler to settle before reading `7456`.
- Prefer an explicit stable signal when available, such as a refreshed preview page exposing the expected source-driven runtime change or a Cocos/editor compile-complete indication.
- If no stable signal is available, wait 2-3 seconds before sampling `7456`; do not treat immediate stale preview output as evidence that generated chunks need manual patching.
- Generated preview chunks may be read for diagnosis only. Writing them is allowed only as an explicitly approved temporary hot patch, and must be called out as non-source validation support.
- Handoff evidence for Cocos work should prefer source-file changes plus refreshed preview validation, not direct edits to generated chunks.

## Batch UI Acceptance Mode

- When the human enters a batch manual acceptance and UI-polish phase, do not keep synchronizing old automated acceptance flows after every UI change.
- During that phase, focus on the requested client-facing fixes and use only bounded checks needed to prove the specific UI change is live and not broken.
- Update or rerun broad dedicated browser gates only after the human explicitly says to remake or refresh the agent acceptance flow.
- If an existing gate script is clearly coupled to the UI surface being changed, keep any compatibility edit minimal and local to that surface; do not treat it as a full acceptance-flow regeneration.

## Default Project Docs

Resident docs:

- `docs/project/development.active.json`
- `docs/process/development-execution-manual.md`
- `docs/process/minimal-role-rules.md`
- `docs/process/development-event-protocol.md`

On-demand docs:

- `docs/process/dedicated-browser-workflow.md`
- `docs/process/browser-runtime-workflow.md`
- `docs/architecture/client-structure.md`
- `docs/architecture/server-structure.md`
- `docs/project/development-plan.json`

Read on-demand docs only when the active subfunction or `must_read` requires them.

## Handoff Contract

- When execution is complete, write one `submit` helper event with bounded evidence references.
- When execution is blocked, write one `block` helper event with a concise reason.
- After the helper succeeds, the chat reply may stay human-readable and concise.
