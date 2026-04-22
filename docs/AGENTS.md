# Docs Area Rules

This directory contains authority docs, ADRs, plans, and controlled legacy references.

## Scope

- Default write scope for design and architecture work lives here.
- Prefer adding or updating docs here instead of inventing behavior in code.

## Rules

- `vision/` is the North Star. Edit only when the user is intentionally changing project direction.
- `decisions/` is for concrete architecture decisions. Use ADRs when a choice changes code structure, data flow, ownership, or constraints.
- `plans/` is for sequencing and milestones, not for redefining architecture.
- `legacy/` is historical context. Do not rewrite it to retroactively reflect new decisions.
- `docs/README.md` is the default docs entry, not a second North Star.
- Do not create a second active planning surface when `docs/project/development-plan.json` already carries the current route.

## Writing discipline

- One document per topic.
- State the doc status.
- Link back to authority docs.
- Keep open questions explicit.
- If a missing authority doc blocks implementation, draft the doc before code.
- If a doc is not part of the default startup path, say so plainly instead of assuming every reader should open it first.
