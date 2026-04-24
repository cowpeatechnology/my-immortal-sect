# ADR 0007: 采用 Hollywood Actor Framework 作为运行时外壳

**状态**: 已确认 (Accepted)
**日期**: 2026-04-15，2026-04-23 按 GDD v3.1 收敛重写
**相关文档**:
- [docs/vision/gdd_v3_backend_design.md](/Users/mawei/MyWork/SlgGame/docs/vision/gdd_v3_backend_design.md)
- [docs/decisions/0010-offline-deterministic-simulation.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0010-offline-deterministic-simulation.md)
- [docs/decisions/0011-v1-sync-model-and-multiplayer-pattern.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0011-v1-sync-model-and-multiplayer-pattern.md)
- [AGENTS.md](/Users/mawei/MyWork/SlgGame/AGENTS.md)

## Context

项目已经裁决为：

- V1 是单玩家宗门经营 + 云存档 + 服务端权威
- 运行时核心是 `SectActor big-state`
- 不允许把弟子、建筑、active storylet 拆成默认独立 actor

因此，服务端需要的是一套可靠的 actor runtime shell，而不是一套实体对象模型。

## Decision

采用 [Hollywood Actor Framework](https://github.com/anthdm/hollywood) 作为服务端运行时外壳。

Hollywood 在本项目中的定位是：

- mailbox 串行化
- supervision / 生命周期管理
- actor 路由与消息边界
- 远程节点与 shell 级基础设施

它不是：

- 业务实体拆分蓝图
- 每个弟子 / 每个建筑 / 每个 storylet 一个 actor 的默认方案

## V1 运行时拓扑

```text
Hollywood Engine
└── GatewaySessionActor
    └── SectActor
        ├── SectState big-state
        ├── Domain command handlers
        ├── ApplyEvent
        └── Snapshot / Patch / Event output

PersistenceActor / storage adapter 负责 I/O
WorldClockActor 负责低频 tick / offline catch-up 驱动
```

## Explicit Non-Decision

以下做法不是 V1 默认架构：

- `DiscipleActor × N`
- `BuildingActor × N`
- `StoryletActor × N`
- 基于房间或广播的多人同步主路径

如后续确需引入独立 actor 粒度，必须以性能或玩法证据为基础新增 ADR。

## Consequences

正面影响：

- 与 `SectActor big-state` 完全一致
- 保留服务端权威、监督树、消息边界和 shell 可扩展性
- 不需要为未来多人想象提前拆碎业务实体

约束：

- 不改 Hollywood upstream 业务逻辑
- 客户端仍需通过网关提交命令，不能直接使用 Hollywood remote
- 持久化、协议、命令幂等等仍由项目自己的 `SectActor + storage` 路径定义
