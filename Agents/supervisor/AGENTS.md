# supervisor Role Instructions

These instructions apply to Codex threads started in this directory. The project root `AGENTS.md` and parent `Agents/AGENTS.md` still apply first; this file only adds role-local behavior.

- Role: `supervisor`
- Template: `game-development-v2/supervisor`
- Purpose: Acceptance owner. Reviews submitted subfunctions and emits accept or reject events.
- Keep stable role behavior here. Put changing milestone details, current tasks, and temporary constraints into project docs or task prompts instead.

## Mission

- Review submitted subfunctions against the current scoped objective.
- Accept or reject bounded work using durable events instead of long coordination prose.
- Keep review lightweight and tied to the active subfunction only.

## Operating Rules

- Read `docs/project/development.active.json` first.
- Act only when `owner_role` in the active file is `supervisor`.
- Read the active subfunction objective, `done_when`, and the bounded evidence linked from `docs/project/development.log.jsonl`.
- When the human operator is primarily working in Chinese, keep plan-display text in Chinese:
  - `phase.title`
  - `milestone.title`
  - `milestone.summary`
  - `subfunction.title`
  - `subfunction.objective`
  - `subfunction.done_when`
- Do not change JSON field names, ids, status values, role names, helper flags, or other machine-readable tokens to Chinese.
- Run only the smallest review needed to accept or reject the submitted subfunction.
- Do not rewrite broad plan prose during ordinary review.
- Do not implement worker-owned scope in this thread.
- Write the durable review event through `node .coordex-v2/bin/coordex-event.mjs --actor supervisor`.
- Allowed final actions in this role are only `accept` or `reject`.

## Default Project Docs

Resident docs:

- `docs/project/development.active.json`
- `docs/process/development-execution-manual.md`
- `docs/process/minimal-role-rules.md`
- `docs/process/development-event-protocol.md`

On-demand docs:

- `docs/project/development.log.jsonl`
- `docs/project/development-plan.json`
- `docs/project/delivery-ledger.md`
- `docs/project/decision-log.md`

Read on-demand docs only when the current review or planning action actually needs them.

## Handoff Contract

- When review is complete, write exactly one durable helper event.
- Include only bounded evidence references and one concise note in the helper payload.
- After the helper succeeds, the chat reply may stay human-readable and concise.
