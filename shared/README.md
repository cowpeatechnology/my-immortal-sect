# Shared Workspace

This directory is the shared source-of-truth area between client and server.

Current intended usage:
- `configs/`: content/config source files that eventually feed both runtimes
- `contracts/`: protocol or contract documents that must stay aligned across client and server

This directory is intentionally small at bootstrap time. It should hold shared sources, not duplicated runtime code.

## Current Config Surface

The first concrete shared config source for the sect-map M1 short session is now:

- `configs/m1/sect_map_short_session.v1.json`
- schema: `configs/m1/sect_map_short_session.schema.json`

This file is the canonical shared source for:

- M1 building template values used by the short-session loop
- M1 resource-node rules and bootstrap seed tiles
- M1 session phase-driving values such as target duration, first-raid prep timing, and post-raid recovery / second-cycle-ready progression

Until a generated pipeline exists, client and server work should treat this JSON file as the one place to freeze those gameplay values. Runtime-local copies are transitional and should be converged toward this source rather than extended independently.

## Current Contract Surface

`M1-D` now freezes the minimum authority-backed short-session contract in:

- `contracts/m1-authority-short-session-v1.md`
- `../server/internal/proto/slggame/protocol/v1/authority_runtime.proto`

That contract is the current source of truth for:

- the formal protobuf-first `ClientCommand / CommandResult / StatePatch` runtime boundary
- the preview-only HTTP JSON shim used by the current Cocos local validation loop
- snapshot fields consumed by the Cocos preview runtime
- the explicit in-scope / out-of-scope boundary for the first `shared + Go + Hollywood` bridge

Interpretation rule:

- `authority_runtime.proto` is the formal authority protocol to extend
- `contracts/m1-authority-short-session-v1.md` documents the current preview shim and feature-slice payload semantics
- new authority work must not introduce a second JSON-first runtime protocol alongside the protobuf path
