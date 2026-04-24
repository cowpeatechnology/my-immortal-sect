# ADR 0009: Sect ID 与 Actor Name 规范

**状态**: 已确认 (Accepted)
**日期**: 2026-04-15，2026-04-23 按 GDD v3.1 收敛重写
**相关文档**:
- [docs/vision/gdd_v3_backend_design.md](/Users/mawei/MyWork/SlgGame/docs/vision/gdd_v3_backend_design.md)
- [docs/decisions/0007-hollywood-actor-framework.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0007-hollywood-actor-framework.md)
- [docs/decisions/0011-v1-sync-model-and-multiplayer-pattern.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0011-v1-sync-model-and-multiplayer-pattern.md)

## Context

V1 已裁决为：

- 一名玩家拥有一份宗门权威状态
- 一份宗门状态由一个 `SectActor` 持有
- 业务身份、actor name、配置 ID 不能混用

因此 ID 规范要围绕 `sect_id` 收口，而不是围绕旧的 `SimulationActor/player_id` 表述继续扩散。

## Decision

### 1. `player_id`

玩家账号身份仍保留平台绑定语义，例如：

- `p_wx_<opaque>`
- `p_dy_<opaque>`
- `p_dev_<opaque>`

### 2. `sect_id`

V1 一名玩家默认只有一份宗门。

推荐格式：

- `sect_<player_id_hash>`
- 或 `sect_<player_id_suffix>_0001`

`sect_id` 是：

- 权威快照主键
- `SectActor` 的业务身份
- 命令路由和恢复的主索引

### 3. Actor Names

V1 默认 actor name 规范：

- `gateway_session/<connection_id>`
- `sect/<sect_id>`
- `persistence`
- `world_clock`

不再把以下名称当作 V1 默认 actor 拓扑：

- `simulation/<player_id>`
- `disciple/<...>`
- `storylet/<...>`

### 4. Runtime Entity IDs

弟子、建筑、任务、storylet 实例都保留稳定业务 ID，但它们属于 `SectState` 内部结构，不自动对应 actor name。

### 5. Config IDs

配置表 ID 继续保持不可变。

配置 ID：

- 不是 actor name
- 不是运行时实例 ID
- 不能在 live content model 中被重命名
