# GDD v3.1 Authority Core Bootstrap

**状态**: active plan
**最后更新**: 2026-04-23
**上游权威**:
- [docs/vision/gdd_v3_backend_design.md](/Users/mawei/MyWork/SlgGame/docs/vision/gdd_v3_backend_design.md)
- [docs/decisions/0012-adopt-gdd-v3-authoritative-backend-gdd.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0012-adopt-gdd-v3-authoritative-backend-gdd.md)
- [docs/decisions/0008-save-protocol-and-pgstore-schema.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0008-save-protocol-and-pgstore-schema.md)
- [docs/decisions/0010-offline-deterministic-simulation.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0010-offline-deterministic-simulation.md)
- [docs/decisions/0011-v1-sync-model-and-multiplayer-pattern.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0011-v1-sync-model-and-multiplayer-pattern.md)

> 本文档只做阶段拆解，不重新定义产品方向或架构真相。产品与后端主规范以 GDD v3.1 为准。

## Phase A：Authority Core Bootstrap

第一阶段只做最小权威后端骨架，不把招生、任务、炼丹、修炼、贡献、月结全部一起压入。

### 必做范围

1. `ClientCommand / CommandResult / StatePatch` protobuf 合同
2. `SectState` 根结构与版本边界
3. `DomainEvent / ApplyEvent` 事件应用流水线
4. `ResourceState` 最小资源模型
5. `BuildBuilding` 第一条权威玩法命令
6. `sect_snapshots / sect_events / command_log` 持久化路径
7. 最小 `SectActor`、`JoinSect`、`SubmitCommand` 路由闭环

### 非目标

- 不在第一阶段同时完成招生、任务堂、炼丹、修炼、贡献月结
- 不把 sect-map 历史短会话继续当作当前产品主线
- 不为了兼容旧 preview 而恢复 client-local fallback

## 交付门槛

- 权威协议为 protobuf-first
- 快照主源为 `snapshot blob`
- `event_log` 只作为审计、回放、debug、弟子日记与幂等辅助，不是 pure event sourcing
- V1 同步边界保持单玩家云存档，不引入共享房间或跨玩家实时同步

## 与执行计划的关系

当前详细的 phase / milestone / subfunction 拆解以：

- [docs/project/development-plan.json](/Users/mawei/MyWork/SlgGame/docs/project/development-plan.json)

为准。
