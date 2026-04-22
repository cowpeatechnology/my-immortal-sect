# Structured Agent Communication Protocol

Use this JSON envelope when role-to-role or role-to-supervisor messages need to stay low-drift.

It replaces the older split between a free-form thread-conversation note and a separate structured protocol.

## When To Use

Use this protocol when:

- the coordination event is important enough to be durable
- free-form prose would create drift
- the message crosses role boundaries and needs explicit input/output framing

For simple human dispatches, ordinary concise chat is still allowed.

## Recommended Message Shape

Even when the event is small, keep these meanings clear:

- `summary`: what is being asked or reported
- `input`: the scoped problem or incoming handoff
- `expected_output`: what the receiving role should return
- `output`: the actual result, blocker, or clarification

## Protocol

- `protocol_version`: `coordex-agent-io.v1`
- `kind`: `dispatch | question | blocker | handoff | result | decision`
- `status`: `open | answered | blocked | done`

## Required Keys

- `protocol_version`
- `task_id`
- `from_role`
- `to_role`
- `kind`
- `status`
- `summary`
- `input`
- `expected_output`
- `output`

## Example

```json
{
  "protocol_version": "coordex-agent-io.v1",
  "task_id": "feature-001",
  "from_role": "engineer",
  "to_role": "supervisor",
  "kind": "result",
  "status": "answered",
  "summary": "Implemented the scoped UI change.",
  "input": "Build the current assigned subfunction.",
  "expected_output": "Working UI plus validation notes.",
  "output": "Updated the UI and ran the documented build command."
}
```

Keep one message equal to one coordination event.
