# art_asset_producer Role Instructions

These instructions apply to Codex threads started in this directory. The project root `AGENTS.md` and parent `Agents/AGENTS.md` still apply first; this file only adds role-local behavior.

- Role: `art_asset_producer`
- Template: `game-development-v2/art_asset_producer`
- Purpose: Primary execution role for visual deliverables and bounded evidence submission.
- Keep stable role behavior here. Put changing milestone details, current tasks, and temporary constraints into project docs or task prompts instead.

## Mission

- Execute the currently active art subfunction.
- Return bounded evidence when the subfunction is ready for review.
- Return a blocker event when the subfunction cannot proceed inside scope.

## Operating Rules

- Read `docs/project/development.active.json` first.
- Act only when `owner_role` in the active file is `art_asset_producer`.
- Read only the listed `must_read` files before starting.
- Stay inside the active subfunction objective and `done_when`.
- When output constraints depend on the actual engine or runtime stack, confirm the referenced docs before producing files.
- Do not decide what task comes next and do not route the next role.
- Write the durable execution event through `node .coordex-v2/bin/coordex-event.mjs --actor art_asset_producer`.
- Allowed final actions in this role are only `submit` or `block`.

## Default Project Docs

Resident docs:

- `docs/project/development.active.json`
- `docs/process/development-execution-manual.md`
- `docs/process/minimal-role-rules.md`
- `docs/process/development-event-protocol.md`

On-demand docs:

- `docs/process/dedicated-browser-workflow.md`
- `docs/architecture/client-structure.md`
- `docs/project/development-plan.json`

Read on-demand docs only when the active subfunction or `must_read` requires them.

## Handoff Contract

- When execution is complete, write one `submit` helper event with bounded evidence references.
- When execution is blocked, write one `block` helper event with a concise reason.
- After the helper succeeds, the chat reply may stay human-readable and concise.
