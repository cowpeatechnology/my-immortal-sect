# ADR 0010: SectActor Big-State And Offline Simulation

**状态**: 已确认 (Accepted)
**日期**: 2026-04-15，2026-04-23 按 GDD v3.1 收敛重写
**相关文档**:
- [docs/vision/gdd_v3_backend_design.md](/Users/mawei/MyWork/SlgGame/docs/vision/gdd_v3_backend_design.md)
- [docs/decisions/0007-hollywood-actor-framework.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0007-hollywood-actor-framework.md)
- [docs/decisions/0008-save-protocol-and-pgstore-schema.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0008-save-protocol-and-pgstore-schema.md)

## Context

项目已经明确：

- V1 是单玩家云存档，不是共享世界实时同步
- 运行时主实体是一个 `SectActor`
- 离线补偿、在线推进、命令处理必须落在同一份权威状态上

## Decision

采用：

- `SectActor` 持有单份 `SectState big-state`
- 在线推进与离线补偿都基于同一套 simulation 逻辑
- 弟子、建筑、任务、库存、事件、storylet 都是 `SectState` 内部结构

## Hard Rules

1. `advance()`、`AdvanceDay()`、离线追赶逻辑内禁止 I/O。
2. wall clock 只能在 actor shell 转换为 `game_tick / elapsed time` 后再进入 simulation。
3. RNG 状态必须保存在 `SectState` 中，不能使用包级随机数。
4. domain command handler 和离线推进都可以产生 `DomainEvent`。
5. `SectState` 的实际修改必须通过 `ApplyEvent`。

## Explicit Non-Decision

本 ADR 不要求：

- 在线 vs 离线字节级相等
- pure function 教条式实现
- 为 V1 提前拆出 `DiscipleActor / BuildingActor / StoryletActor`

它要求的是：

- 单一权威状态路径
- 离线补偿可恢复、可回放、可解释
- 同一条命令/事件/快照链路能覆盖在线与离线运行
