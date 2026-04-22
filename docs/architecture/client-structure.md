# Client Structure

**状态**: 草案  
**最后更新**: 2026-04-22  
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

## Thin-Client Runtime Contract

The sect-map runtime is now being rebuilt around:

**authority-only gameplay truth + thin client render/input shell**

For the active `M1` line, the client must separate permanent thin-client responsibilities from transitional preview debt.

### Render State

Render state is permanent client-owned state.

It includes:

- scene graph composition for map, buildings, disciples, hostiles, HUD, tips, and VFX
- sprite orientation, sorting, visibility, highlight, and selection presentation
- UI-local display caches derived from the latest authority snapshot
- presentation-only timers for fades, popups, shakes, and other visual effects

Render state must be disposable and reconstructable from authority snapshot plus current UI state.

### Animation And Interpolation State

Animation and interpolation state is permanent client-owned state.

It includes:

- smoothing movement between authority checkpoints
- animation playback phase, blend, facing, and speed multipliers
- path visualization or motion trails used to show the current authority assignment
- short-lived presentation anchors such as "walk to tile", "play build swing", or "show hit reaction"

Animation and interpolation state must never decide gameplay truth. It can only visualize authority-owned assignment, progress, and damage outcomes.

### Debug State

Debug state is permanent client-owned state.

It includes:

- preview-only overlays and inspector panels
- cached bounded snapshot comparisons between preview and authority
- local debug toggles, labels, and warning surfaces such as `authority.lastError`
- replay helpers that expose reset / restore controls without changing gameplay ownership

Debug state may help humans detect drift, but it must not become a second progression controller.

### Player-Input Submission

Player-input submission is permanent client-owned behavior.

It includes:

- collecting taps, drags, long-presses, and UI button intent
- mapping those interactions into allowed high-level authority commands
- submitting bounded fact reports only where the current contract explicitly permits them
- reacting to authority rejection by clearing stale local presentation state and re-syncing

Player-input submission must stop at intent or allowed bounded facts. It must not announce that build, repair, demolition, raid, or phase progression has completed.

## Transitional Preview Debt

The following client responsibilities may still exist temporarily during the authority rebuild, but they are debt and must not be treated as part of the permanent client design:

- local gather / dropoff choreography that still needs a bounded fact submit before authority updates stockpile
- preview-side temporary markers that help recover from rejected legacy commands or stale presentation tasks
- local animation glue that still mirrors incomplete authority worker fields because the snapshot shape is not fully cut over yet
- any compatibility layer that exists only to keep the dedicated preview runnable while old hybrid paths are being removed

These debt paths must obey two rules:

- they cannot become the source of truth for gameplay progression
- they must be removable once the authority snapshot exposes the required worker / building / session fields

## Permanent Thin-Client Design

The long-term client design for the sect-map runtime is limited to:

- render state
- animation / interpolation state
- debug state
- player-input submission

Anything outside those four buckets requires explicit justification as either:

- a contract-approved bounded fact surface, or
- temporary transitional debt scheduled for removal

## Next Manual Step

Current bootstrap status:

1. The Cocos Creator project already exists at `client/my-immortal-sect/`.
2. The Cocos MCP plugin is already installed and serving on port `9527`.

Next collaboration rule:

1. Codex edits `client/my-immortal-sect/extensions/cocos-mcp-server/` directly when plugin changes are needed, but those edits must remain tracked by the root repository rather than a nested `.git`.
2. The project owner refreshes the loaded extension inside Cocos after those changes if the runtime has not auto-reloaded.
