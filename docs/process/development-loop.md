# Development Loop

Use this as the short human-readable summary of the V2 loop.

For the exact runtime contract, read `docs/process/development-execution-manual.md`.

## Default Loop

1. Read the active pointer.
2. Stop if the active owner is not your role.
3. Read only the `must_read` files for the current subfunction.
4. Execute inside the current objective and `done_when`.
5. Validate with the real project path or the documented preview surface.
6. Return one bounded result event.
7. Let `supervisor` review and accept or reject.

## Guardrails

- Do not widen scope silently.
- Do not reopen broad planning while executing one subfunction.
- Do not default to re-reading the entire project history.
- Do not treat every subfunction as its own mandatory commit boundary.
