# Server Structure

**状态**: 草案  
**最后更新**: 2026-04-16  
**依赖**: `AGENTS.md`, `docs/vision/design-decisions.md`, `docs/decisions/0007-hollywood-actor-framework.md`, `docs/plans/m0-vertical-slice.md`

## Context

This document defines the initial repository structure for the real Go server runtime.

The goal at this stage is not to pre-implement business logic. The goal is to establish a stable write scope so later Go and Hollywood work can land without re-laying out the repository.

## Directory Contract

```text
server/
├── cmd/gameserver/
├── configs/
├── internal/
│   ├── proto/
│   └── slggame/
│       ├── gateway/
│       ├── player/
│       ├── simulation/
│       ├── sect/
│       ├── disciple/
│       ├── storylet/
│       ├── karma/
│       ├── descent/
│       ├── sync/
│       ├── anticheat/
│       └── storage/
├── migrations/
│   ├── db/
│   └── config/
├── tests/
└── tools/
    ├── config/
    └── simlint/
```

## Ownership Notes

- `cmd/gameserver/`: only the actual runtime entrypoint and wiring code
- `internal/proto/`: project protobuf definitions
- `internal/slggame/simulation/`: deterministic simulation core
- `internal/slggame/gateway/`: HTTP / WebSocket edge translation
- `internal/slggame/storage/`: PGStore and storage-facing adapters
- `tools/simlint/`: static checks that protect deterministic simulation rules

## Hard Constraints

- Business code stays under `server/internal/slggame/`.
- Protobuf definitions stay under `server/internal/proto/`.
- Hollywood upstream is not a business-code write target.
- Actor boundaries must remain compatible with the project ADRs.

## Bootstrap Status

Current state:
- directory scaffold exists
- `go.mod` 已初始化
- `cmd/gameserver/` 已落地本地 authority HTTP 入口
- `internal/slggame/authority/` 已落地 Hollywood actor-backed `M1-D` session state
- `internal/slggame/gateway/` 已落地最小 HTTP -> actor command translation

Current implementation is still intentionally narrow. It only covers the `M1-D` authority slice for建造状态、资源结算与关键短会话快照；完整 persistence、protobuf 和更广的 simulation authority 仍待后续循环推进。
