# Development Execution Manual

This project uses the V2 execution model.

## Runtime Files

- `docs/project/development-plan.json`
  - full execution snapshot
- `docs/project/development.active.json`
  - small active-work pointer
- `docs/project/development.log.jsonl`
  - append-only durable event log

## Display Language Rule

- Keep the machine-readable plan structure unchanged.
- When the human operator is primarily working in Chinese, write the human-facing plan content in Chinese:
  - `phase.title`
  - `milestone.title`
  - `milestone.summary`
  - `subfunction.title`
  - `subfunction.objective`
  - `subfunction.done_when`
- Keep these machine fields in English exactly as defined by the schema:
  - JSON field names
  - `id`
  - `status`
  - `owner_role`
  - helper arguments
  - protocol fields and event actions
- Do not translate or rename machine-readable tokens just to localize the plan text.

## Read Order

After startup, resume, or compaction:

1. read `docs/project/development.active.json`
2. if `owner_role` is not your role, stop
3. if `owner_role` is your role, read only `must_read`
4. only if needed, jump into `docs/project/development-plan.json` by id
5. inspect `docs/project/development.log.jsonl` only when bounded historical evidence is required

## Execution Flow

1. Human chooses a subfunction and starts it through Coordex.
2. Coordex updates:
   - `development-plan.json`
   - `development.active.json`
   - `development.log.jsonl` with a `start` event
3. The current owner executes the subfunction.
4. The current owner emits exactly one durable event through the local helper:
   - `submit`
   - or `block`
5. If submitted, Coordex waits for the turn to finish, reads the helper-written event file, then switches the active owner to `supervisor`.
6. Supervisor reviews bounded evidence and emits exactly one event:
   - `accept`
   - or `reject`
7. If the supervisor emits `reject`, Coordex keeps the same subfunction active, switches the active owner back to the original worker, and includes the reject note/evidence in the worker rework prompt.
8. The worker must then re-check whether the reject reason actually needs a bounded fix, and respond with either:
   - `submit` after the bounded fix or stronger validation
   - or `block` if the reject reason cannot be safely resolved inside scope
9. Coordex updates the plan, clears or advances the active pointer only after `accept`, `block`, or the reject rework limit is exceeded.

## Durable Event Helper

The role currently holding the active pointer must write its machine-readable event through:

- `node .coordex-v2/bin/coordex-event.mjs --actor <role>`

The helper:

- accepts either repeated flags or one protocol JSON object on stdin
- validates that `development.active.json` currently points at the same task and role
- writes the durable event file under `.coordex-v2/runtime/role-events/`

The chat reply remains visible for humans, but Coordex no longer advances the workflow by parsing protocol JSON out of the chat transcript.

## Status Rules

Allowed subfunction status values:

- `idle`
- `running`
- `blocked`
- `submitted`
- `accepted`
- `rejected`
- `archived`

## Responsibility Split

- Coordex owns workflow mechanics and state-file updates.
- Worker roles own execution plus `submit` or `block`.
- Supervisor owns review plus `accept` or `reject`.
- A `reject` is not terminal by itself; it becomes a worker rework loop inside the same subfunction.
- Roles do not decide the next owner or rewrite broad plan structure.
- Ordinary plan subfunctions should stay worker-owned; supervisor review is triggered by the active pointer after `submit`.
