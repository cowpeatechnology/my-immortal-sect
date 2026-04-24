# Docs Area Rules

This directory contains authority docs, ADRs, plans, and controlled legacy references.

## Scope

- Default write scope for design and architecture work lives here.
- Prefer adding or updating docs here instead of inventing behavior in code.

## Rules

- `vision/` is the North Star. The active product/backend implementation spec is `vision/gdd_v3_backend_design.md`. `vision/design-decisions.md` is only a superseded shim that points at the active GDD and archived history.
- `decisions/` is for concrete architecture decisions. Use ADRs when a choice changes code structure, data flow, ownership, or constraints.
- `plans/` is for sequencing and milestones, not for redefining architecture.
- `legacy/` is historical context. Do not rewrite it to retroactively reflect new decisions.
- `docs/README.md` is the default docs entry, not a second North Star.
- Do not create a second competing execution plan. Product/backend direction comes from the authoritative GDD; `docs/project/development-plan.json` is the routed execution surface derived from it.
- `features/` and `design/` are not active sources of product truth. Keep them historical or narrowly scoped.

## Writing discipline

- One document per topic.
- State the doc status.
- Link back to authority docs.
- Keep open questions explicit.
- If a missing authority doc blocks implementation, draft the doc before code.
- If a doc is not part of the default startup path, say so plainly instead of assuming every reader should open it first.
