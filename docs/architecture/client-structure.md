# Client Structure

**状态**: 草案  
**最后更新**: 2026-04-16  
**依赖**: `AGENTS.md`, `docs/vision/design-decisions.md`, `docs/plans/m0-vertical-slice.md`

## Context

This document defines the repository contract for the active Cocos Creator client.

The initial bootstrap assumption was "create the project directly under `client/`". The actual project now lives at `client/my-immortal-sect/`, which becomes the authoritative client root for implementation work.

## Bootstrap Rule

- The client workspace container is `client/`.
- The active Cocos project root is `client/my-immortal-sect/`.
- Cocos-generated runtime cache directories remain untracked through the root `.gitignore`.
- Project-local MCP settings live at `client/my-immortal-sect/settings/mcp-server.json`.
- The current MCP server port is `9527`.
- The MCP extension source-of-truth lives at `client/my-immortal-sect/extensions/cocos-mcp-server/`.
- Under the current repository policy, that directory is part of the single root repository at `~/MyWork/SlgGame/.git`, not an independent nested Git repository.

## Expected Owned Areas After Cocos Bootstrap

- `client/my-immortal-sect/assets/`
  - game scenes
  - tilemap assets
  - gameplay scripts
  - resources bundled with the client
- `client/my-immortal-sect/extensions/`
  - editor-side extensions that belong to this repo
- `client/my-immortal-sect/settings/`
  - committed editor project settings
- `client/my-immortal-sect/packages/`
  - package-level project extensions if the final Cocos layout uses them

## Expected Script Ownership Inside `assets/`

- `assets/scenes/`: Cocos scenes and map entry scenes
- `assets/scripts/app/`: app bootstrap and scene wiring
- `assets/scripts/net/`: preview-time authority adapter, protocol client, and other client-side gateway glue
- `assets/scripts/modules/`: gameplay-facing client modules
- `assets/scripts/ui/`: HUD and map overlays
- `assets/tiled/`: Tiled map sources and related data

## Runtime Asset Boundary

- `client/my-immortal-sect/assets/` is a runtime asset root, not a general-purpose workspace.
- Keep only content that Cocos is expected to import, compile, or load at runtime:
  - scenes
  - runtime scripts
  - tilemaps
  - final importable art assets
- Do not place Node or Python generators, preview collages, one-off export scripts, or human-only asset specs under `assets/`.
- If a tool generates runtime assets for the client, keep the tool under root `tools/` and emit only its final outputs into `client/my-immortal-sect/assets/`.

## Reserved Non-Goals

- Do not put authoritative economy or simulation decisions in the client.
- Do not treat the client as the source of truth for build progress, resource settlement, or karma triggers.
- Do not use `client/` as a dumping ground for tool output or prototype HTML.

## Next Manual Step

Current bootstrap status:

1. The Cocos Creator project already exists at `client/my-immortal-sect/`.
2. The Cocos MCP plugin is already installed and serving on port `9527`.

Next collaboration rule:

1. Codex edits `client/my-immortal-sect/extensions/cocos-mcp-server/` directly when plugin changes are needed, but those edits must remain tracked by the root repository rather than a nested `.git`.
2. The project owner refreshes the loaded extension inside Cocos after those changes if the runtime has not auto-reloaded.
