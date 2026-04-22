# Development Event Protocol

Use this exact JSON shape for durable role outputs in the V2 workflow.

These role events are no longer taken from the chat reply itself.

Coordex expects the active role to write the event through the local helper before the turn ends:

- `node .coordex-v2/bin/coordex-event.mjs --actor <role>`

The helper reads one JSON object from stdin, validates it against `development.active.json`, and writes the durable event file under `.coordex-v2/runtime/role-events/`.

Coordex then advances the workflow after the role turn reaches `turn/completed` and the helper-written event file is present.

## Protocol Name

- `protocol_version: "coordex-development-event.v1"`

## Required Keys

- `protocol_version`
- `task_id`
- `action`
- `evidence`
- `note`

## Allowed Actions

Worker roles:

- `submit`
- `block`

Supervisor:

- `accept`
- `reject`

## Output Rule

When asked to emit a protocol event:

- run the helper before the turn ends
- pass exactly one JSON object to the helper on stdin
- after the helper succeeds, reply in plain text for the human if needed
- do not rely on the chat reply itself as the machine-readable event source

## Example

```json
{
  "protocol_version": "coordex-development-event.v1",
  "task_id": "sf-001",
  "action": "submit",
  "evidence": [
    "file:src/main.ts",
    "command:npm run build"
  ],
  "note": "Scoped implementation is ready for supervisor review."
}
```
