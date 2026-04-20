# Shared Workspace

This directory is the shared source-of-truth area between client and server.

Current intended usage:
- `configs/`: content/config source files that eventually feed both runtimes
- `contracts/`: protocol or contract documents that must stay aligned across client and server

This directory is intentionally small at bootstrap time. It should hold shared sources, not duplicated runtime code.

## Current Contract Surface

`M1-D` now freezes the minimum authority-backed short-session contract in:

- `contracts/m1-authority-short-session-v1.md`

That contract is the current source of truth for:

- authority session endpoints
- command names and payload shapes
- snapshot fields consumed by the Cocos preview runtime
- the explicit in-scope / out-of-scope boundary for the first `shared + Go + Hollywood` bridge
