# ADR 0011: V1 单玩家云存档同步边界

**状态**: 已确认 (Accepted)
**日期**: 2026-04-15，2026-04-23 按 GDD v3.1 收敛重写
**相关文档**:
- [docs/vision/gdd_v3_backend_design.md](/Users/mawei/MyWork/SlgGame/docs/vision/gdd_v3_backend_design.md)
- [docs/decisions/0007-hollywood-actor-framework.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0007-hollywood-actor-framework.md)
- [docs/decisions/0012-adopt-gdd-v3-authoritative-backend-gdd.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0012-adopt-gdd-v3-authoritative-backend-gdd.md)

## Context

项目早期文档里多次使用“多人在线”这个词，但 V1 真正需要的同步模型其实是：

- 多个玩家可以同时在线
- 每个玩家只经营自己的宗门
- 不共享一份实时状态

如果不把这件事钉死，GDD 和实现很容易再次滑向房间、广播、观察者、共享地图等过早设计。

## Decision

V1 只实现：

- `Client ↔ Server`
- 单玩家 `SectActor`
- 云存档
- 离线补偿

形式化表达：

```text
1 个玩家 == 1 个 player_id == 1 个 sect_id == 1 份 SectState == 1 个活跃 SectActor owner
```

## V1 明确不做

- 房间式共享状态
- 跨玩家实时广播
- 观察者同步
- 共享世界地图
- 排行榜实时推送
- 跨宗门实时互动

## Relation To GDD v3.1

GDD v3.1 中所有“多人在线”“房间”“访问”“广播”“跨宗门”相关表述，在 V1 中均解释为未来扩展说明，不构成当前实现范围。

这些能力最多属于：

- `V2+`
- 单独新增 ADR 后再进入实现计划

## Consequence

`SessionRegistryActor / SectActor / ClientPushActor` 就足以覆盖 V1 的同步需求。

任何重新引入 `RoomActor / BroadcasterActor / Watchers` 作为 V1 主路径的提案，都必须被视为越界。
