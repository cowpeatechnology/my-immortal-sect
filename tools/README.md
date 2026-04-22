# Tools Directory

`tools/` only contains project-support tooling. It is not part of the game runtime.

## Current layout

- `asset-pipeline/`: asset generation and transformation workflows that produce game-ready resources
- `browser/`: browser automation, ChatGPT web helpers, Chrome bridge, CDP capture, and related operator docs

## Rules

- Keep new tools inside a named subdirectory with a clear domain boundary.
- Do not add new loose scripts directly under `tools/` unless there is a strong temporary reason.
- Keep runtime outputs out of `tools/`; write them under `workspace/`.
- When a tool needs operator guidance, place the doc next to the tool entrypoint in the same subarea.
