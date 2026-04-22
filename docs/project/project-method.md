# Coordex Project Method

This file explains the current project method for `SlgGame`.

It is intentionally small.
The project no longer uses two active planning systems in parallel.

## Current Truth

The current Coordex V2 execution truth is:

- root rules: `AGENTS.md`
- North Star: `docs/vision/design-decisions.md`
- full project route: `docs/project/development-plan.json`
- active pointer: `docs/project/development.active.json`
- durable event log: `docs/project/development.log.jsonl`
- accepted delivery history: `docs/project/delivery-ledger.md`
- future-shaping decisions: `docs/project/decision-log.md`

If another document still describes `.coordex/current-plan.md`, `.coordex/plan-history.md`, `.coordex/project-board.json`, or `docs/project/role-state/<role>.md` as the current planning surface, treat that description as obsolete.

## Core Loop

1. The human sets direction and priority.
2. `supervisor` maintains the current route in `docs/project/development-plan.json`.
3. Coordex sets the current executable pointer in `docs/project/development.active.json`.
4. The human opens the visible role thread for the current owner.
5. The owner reads only the active pointer and its `must_read` set.
6. The owner executes one scoped subfunction.
7. The owner emits one bounded result:
   - `submit`
   - or `block`
8. `supervisor` reviews bounded evidence and emits:
   - `accept`
   - or `reject`
9. Coordex advances the active pointer and records the durable state.

## Scope Rules

- One active subfunction has exactly one owner role.
- `supervisor` owns planning, routing, scope boundaries, and acceptance.
- Worker roles do not widen scope on their own.
- If a subfunction needs browser validation, it must use the dedicated browser workflow at `127.0.0.1:9333`.
- If a task depends on engine, editor, platform, build, or runtime behavior, read the relevant official docs and local project constraints before implementation.

## File Boundaries

### `docs/project/development-plan.json`

This is the only authoritative full-route plan.

Use it for:

- project phases
- milestone order
- subfunction ownership
- execution priority
- long-range route visibility

Do not maintain a second human-written current-plan file in parallel.

### `docs/project/development.active.json`

This is the only active-work pointer.

Use it for:

- current owner role
- current active subfunction
- current objective
- current `done_when`
- current `must_read`

Roles should stop here unless the active pointer explicitly requires deeper reading.

### `docs/project/development.log.jsonl`

This is the append-only event trail.

Use it for:

- `start`
- `submit`
- `block`
- `accept`
- `reject`

Do not use it as a second planning document.

### `docs/project/delivery-ledger.md`

This records only accepted work.

### `docs/project/decision-log.md`

This records only decisions that change future work.

## Default Reading Order

For ordinary project work:

1. `AGENTS.md`
2. `docs/vision/design-decisions.md`
3. `docs/project/development-plan.json`
4. `docs/project/development.active.json`
5. only the docs referenced by the active subfunction

This repository should not require every role to replay the full documentation tree on each task.

## Supervisor Rule

`supervisor` should plan first, route second, and implement only when the current subfunction genuinely has no more suitable worker owner.

The default implementation owners remain:

- `engineer`
- `art_asset_producer`

## Historical Note

This project previously carried an older `.coordex` planning surface.
That model is no longer the live planning truth for this repository.
Historical references may still exist in deleted files or old commits, but current execution should follow the V2 runtime files under `docs/project/`.
