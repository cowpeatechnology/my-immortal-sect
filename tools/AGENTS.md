# Tools Area Rules

`tools/` exists to support the main game project. It is not the main product.

## Default behavior

- Treat this directory as protected unless the user explicitly asks for tools, automation, or asset-pipeline work.
- Keep tool changes isolated from gameplay, client, and server planning unless the task explicitly spans them.

## Rules

- Do not break existing workflows just to clean things up aesthetically.
- Prefer additive changes, documentation, and targeted fixes.
- Keep inputs, outputs, and runtime assumptions explicit.
- Avoid coupling tool scripts to future `client/` or `server/` layouts that do not exist yet.
- When a tool drives content creation, document how it serves the game roadmap rather than letting the tool become the roadmap.
