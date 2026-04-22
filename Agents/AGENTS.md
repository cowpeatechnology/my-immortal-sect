# Agents Directory

These instructions apply to every Codex thread started under this directory. The project root `AGENTS.md` still applies first.

- Directory: `Agents`
- Template: `game-development-v2`
- Purpose: Visible role threads for a game project. Keep one role per subdirectory under this folder.
- Use English directory names that are stable and machine-safe.
- Put role-local behavior in `AGENTS.md` inside the role directory instead of overloading this shared layer.

## Shared Coordination Rules

- Durable role threads under this directory are visible project assets, not disposable hidden workers.
- Coordex owns activation, routing, and state-file updates. Roles do not decide who should be activated next.
- Read `docs/project/development.active.json` first after startup, resume, or compaction.
- If `owner_role` in `development.active.json` does not match your role, stop and wait.
- If `owner_role` matches your role, read only the files listed in `must_read` before acting.
- Do not replay large project history by default. Jump into `docs/project/development-plan.json` only when the active file is insufficient.
- Use the event contract in `docs/process/development-event-protocol.md` when writing a durable result.
- Do not widen task scope. Act only inside the current subfunction objective.
- Do not decide acceptance or the next role unless your role explicitly owns that responsibility.

## Shared Read Policy

This project uses a lightweight resident set plus on-demand reads.

Resident set:

- `docs/project/development.active.json`
- `docs/process/development-execution-manual.md`
- `docs/process/development-event-protocol.md`

On-demand set:

- `docs/project/development-plan.json`
- feature docs
- architecture docs
- browser or engine workflow docs
- ADRs

Only read on-demand files when:

- `must_read` explicitly lists them
- the active subfunction cannot be executed or reviewed without them
- a bounded validation or architecture question requires them

## Shared Startup Docs

Read these before non-trivial work if they exist:

- `docs/project/development.active.json`
- `docs/process/development-execution-manual.md`
- `docs/process/development-event-protocol.md`

If a listed file is missing, continue with the files that do exist.
