# ADR 0008: 云存档协议与 PGStore Schema

**状态**: 已确认 (Accepted)
**日期**: 2026-04-15
**决策者**: 项目负责人 + Codex
**相关文档**:
- `docs/vision/design-decisions.md` §4.3 云存档 + 增量同步协议 / §4.4 配置表驱动 + 版本迁移
- `docs/decisions/0007-hollywood-actor-framework.md` §使用策略
- `docs/decisions/0010-offline-deterministic-simulation.md`
- `AGENTS.md` §4 / §11

---

## Context

ADR 0010 已经定义了 `State` 作为"每玩家单个确定性游戏状态容器"。但 State 只是内存中的数据结构。要让它成为真正的云存档，我们必须解决四件事：

1. **持久化格式**：State 如何在磁盘 / 网络上表达（二进制 / JSON / protobuf）
2. **表结构**：PostgreSQL 里用什么 schema 存，哪些字段要可查询、哪些打包成 blob
3. **快照时机**：什么时候把内存 State 写回数据库（每 tick / 每事件 / 每次登出）
4. **版本治理**：schema_version / simulation_version / config_version 三者如何独立演进又协调一致

这四件事的任何一件设计不好都会在上线后撕裂数据。典型事故：
- 某次更新后老存档读不回来（schema 不兼容）
- 玩家下线时"时间没停在正确位置",登录后弟子凭空消失（快照时机错）
- 新版本模拟逻辑对老存档行为不同（版本锁缺失）
- 配置表删除了一个 storylet ID，老存档里有人正在跑这个 storylet → 崩溃

`design-decisions.md` §4.3 只给了方向（"瘦存档 + 胖配置"），没有定义具体的 schema 和协议。本 ADR 把这层钉死。

---

## Decision

**采用"单行 protobuf blob + 可查询索引表"的混合方案**，在 PostgreSQL 中为每玩家存储一份权威 State 快照，同时维护少量查询索引表。

核心数据结构：

```
player_state       ─ 主表，每玩家一行，State protobuf blob
player_events      ─ 事件流表，用于"弟子日记"和离线补偿后的事件回放
player_versions    ─ 版本元数据表，记录三个版本号和迁移历史
player_deleted     ─ 软删除表（账号删除走此表，不物理 DELETE）
```

**序列化**：protobuf（vtprotobuf 无反射变体，与 Hollywood 生态一致）

**快照时机**：登出必写 + 关键事件触发写 + 10 分钟兜底写，不按 tick 写

**三版本独立演进**：schema_version（存储布局）/ simulation_version（模拟逻辑 + State 结构）/ config_version（游戏内容），各自有独立迁移链

**Hollywood Storer 接口**：实现 `PGStore`，映射 Storer 的 KV 语义到上述表，业务代码通过 Storer 接口访问，不直接写 SQL

---

## Schema 设计

### 表 1：`player_state`（主表）

```sql
CREATE TABLE player_state (
    player_id           TEXT PRIMARY KEY,         -- 业务主键，非自增
    schema_version      INTEGER NOT NULL,         -- 存储格式版本
    simulation_version  INTEGER NOT NULL,         -- 模拟逻辑版本（see ADR 0010）
    config_version      INTEGER NOT NULL,         -- 内容配置版本
    game_tick           BIGINT  NOT NULL,         -- 当前游戏 tick
    last_seen_wall_ms   BIGINT  NOT NULL,         -- 上次 advance() 发生的现实时间（毫秒）
    state_blob          BYTEA   NOT NULL,         -- State protobuf 序列化结果
    state_size_bytes    INTEGER NOT NULL,         -- blob 大小（监控用）
    rng_seed            BIGINT  NOT NULL,         -- PRNG 初始种子（调试用，非权威）
    sect_name           TEXT    NOT NULL,         -- 索引字段，用于显示
    disciple_count      INTEGER NOT NULL,         -- 索引字段，用于监控
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_player_state_updated ON player_state(updated_at);
CREATE INDEX idx_player_state_versions ON player_state(simulation_version, config_version);
```

**说明**：
- `state_blob` 是 `simulation.State` 的 protobuf 序列化，包含**所有权威游戏数据**
- 其他列都是"冗余索引列"，从 blob 里同步出来，只用于监控 / 管理后台 / 未来社交功能，不是数据源
- `last_seen_wall_ms` 用 `BIGINT` 毫秒时间戳而非 `TIMESTAMPTZ`，避免时区歧义和确定性污染
- `rng_seed` 只是创建时的初始种子，**真正的运行时 RNG 状态在 state_blob 里**
- `state_size_bytes` 作为监控字段，超过阈值（如 100KB）告警

### 表 2：`player_events`（事件流）

```sql
CREATE TABLE player_events (
    player_id       TEXT    NOT NULL,
    game_tick       BIGINT  NOT NULL,
    event_seq       INTEGER NOT NULL,             -- 同 tick 内的序号
    event_type      SMALLINT NOT NULL,            -- 枚举：见 events.proto
    event_blob      BYTEA   NOT NULL,             -- 事件 payload protobuf
    acknowledged    BOOLEAN NOT NULL DEFAULT FALSE,  -- 玩家是否已经看过
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (player_id, game_tick, event_seq)
);

CREATE INDEX idx_player_events_unack
    ON player_events(player_id, game_tick)
    WHERE acknowledged = FALSE;
```

**说明**：
- `advance()` 每次调用产生的 events 落这张表
- 离线补偿时的弟子日记就是这张表的 `WHERE acknowledged = FALSE ORDER BY game_tick`
- 玩家确认看过后批量 UPDATE `acknowledged = TRUE`
- 已确认的事件**保留不删**，支持未来的"宗门大事记"翻阅功能（见 §2.1.3 "族谱 / 大事记"）
- 老事件（> 6 个月）可归档到冷存储，V1 先不做

### 表 3：`player_versions`（版本迁移历史）

```sql
CREATE TABLE player_versions (
    player_id               TEXT NOT NULL,
    migration_id            TEXT NOT NULL,        -- 如 "sim_v1_to_v2"
    from_version            INTEGER NOT NULL,
    to_version              INTEGER NOT NULL,
    version_kind            SMALLINT NOT NULL,    -- 1=schema 2=simulation 3=config
    applied_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (player_id, migration_id)
);
```

**说明**：
- 每次迁移执行都留痕，便于回溯"这个玩家的存档走过哪些迁移"
- 如果迁移有 bug，可以针对性地找到"所有走过 v3→v4 迁移的玩家"做修复
- 这张表会越来越大，V1 不做清理，M3 上线前再决定是否归档

### 表 4：`player_deleted`（软删除）

```sql
CREATE TABLE player_deleted (
    player_id       TEXT PRIMARY KEY,
    deleted_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason          TEXT,
    state_blob      BYTEA,                        -- 删除时的最后快照，便于恢复
    expire_at       TIMESTAMPTZ NOT NULL          -- 到期真正清理（合规要求）
);
```

**说明**：
- 账号注销走软删，30 天内可恢复
- 到期后由后台 job 真正清除 state_blob（只留 player_id 和 deleted_at 作审计）
- 符合"数据删除权"合规

---

## 序列化格式

### 选型：protobuf（vtprotobuf 变体）

候选方案对比：

| 格式 | 大小 | 速度 | 可读性 | 兼容性 | 裁定 |
|---|---|---|---|---|---|
| JSON | 大（3~5x） | 慢 | 人类可读 | 宽松 | ❌ |
| Protobuf | 小 | 快 | 需工具 | 严格 field number | ✅ |
| MessagePack | 中 | 快 | 需工具 | 弱 | ❌ |
| Gob (Go native) | 中 | 快 | 不可读 | 仅 Go | ❌ |
| FlatBuffers | 小 | 极快 | 需工具 | 严格 | ❌ |

**选 protobuf 的理由**：
- 与 Hollywood 生态一致（Hollywood Remote 就用 protobuf）
- 客户端也要用 protobuf 做增量同步（见未来的 `save-sync-protocol.md`），双端共享 schema
- field number 强制向后兼容，契合"ID 永不修改"铁律
- vtprotobuf 的无反射实现性能接近原生 struct 直接序列化
- protoc 工具链成熟，类型生成可纳入 CI

### .proto 文件组织

```
server/internal/proto/
├── state/
│   ├── player_state.proto       # 顶层 State 消息
│   ├── sect.proto               # 宗门子结构
│   ├── disciple.proto           # 弟子子结构
│   ├── building.proto           # 建筑子结构
│   ├── storylet.proto           # 活跃 storylet 状态
│   ├── rng.proto                # PRNG 状态序列化
│   └── events.proto             # 事件类型定义
├── sync/
│   └── sync_proto.proto         # 增量同步协议（ADR 0008 不涉及，留给后续 ADR）
└── gateway/
    └── gateway_proto.proto      # 客户端 ↔ GatewayActor 协议
```

**约束**：
- 每个 .proto 文件 package 独立
- 所有消息用 `syntax = "proto3";`
- field number 一旦进主分支**永不复用**，哪怕字段被 reserved
- 使用 `reserved` 关键字明确标记废弃 field number

---

## 快照时机策略

### 核心原则：按事件写，不按 tick 写

V1 每玩家 1 Hz tick、10 弟子、估算 State 约 10KB。如果按 tick 写：
- 1 写 / 秒 / 玩家 × 10KB = 10KB/s/玩家
- 5000 DAU → 50 MB/s 写负载
- PostgreSQL 会被打爆

按事件写的负载粗估：
- 关键事件发生率 ≈ 0.01 /tick/玩家 = 0.01 写 / 秒 / 玩家
- 5000 DAU → 50 写/s，Trivial

### 快照触发条件（四选一）

1. **玩家登出**（`LogoutMsg` 到达 GatewayActor）→ 同步写一次
2. **关键事件发生**（弟子死亡 / 境界突破 / 因果 stage 切换 / 建筑升级完成 / 战斗结束）→ 异步写
3. **兜底计时器**：距上次快照超过 600 个 game_tick（10 游戏分钟 = 600 现实秒）→ 异步写
4. **主动容灾点**（服务端发起的紧急持久化，如优雅关机信号）→ 同步写

### 写入流程

```
SimulationActor 检测到快照条件
  ↓
构造 SnapshotRequest (from State)
  ↓
发送给 PersistenceActor（per player）
  ↓
PersistenceActor:
  ├─ 序列化 State → blob
  ├─ BEGIN TRANSACTION
  ├─ UPSERT player_state (blob, versions, updated_at)
  ├─ INSERT player_events (new events since last snapshot)
  ├─ COMMIT
  └─ 如果失败 → 重试策略（指数退避，最多 3 次）→ 仍失败则 dead letter 报警
```

**关键约束**：
- PersistenceActor 是**独立于 SimulationActor** 的 actor，防止持久化阻塞模拟
- SimulationActor 发完 SnapshotRequest 立刻继续 tick，不等 DB 回执
- PersistenceActor 的 mailbox 保证同一玩家的写入顺序
- 写入失败不影响内存 State，只影响"下次登录看到的状态"；必须监控并告警

### 登出时的同步写

登出时必须**等 DB 写入完成**再关闭 actor 树，否则玩家状态会"回退"：

```
GatewayActor 收到 LogoutMsg
  ↓
向 SimulationActor 发 FinalSnapshotMsg（Request/Respond）
  ↓
SimulationActor 向 PersistenceActor 发 SnapshotRequestSync
  ↓
PersistenceActor 写完后 Respond
  ↓
SimulationActor Respond 给 GatewayActor
  ↓
GatewayActor Poison PlayerSupervisor
```

Hollywood 的 Request/Respond + 超时机制恰好支持这个模式。超时阈值建议 **5 秒**，超时则强制 Poison 并记录告警。

---

## 加载与恢复

### 登录流程

```
客户端发 LoginMsg 到 GatewayActor
  ↓
GatewayActor:
  ├─ 查 player_state WHERE player_id = ?
  ├─ 如果不存在 → 走 NewPlayerFlow（创建初始 State）
  └─ 如果存在：
       ├─ 检查三个 version，如有不匹配 → 走 Migration
       ├─ 反序列化 state_blob → State
       ├─ 计算 elapsed_ticks = (now - last_seen_wall_ms) * TICKS_PER_SECOND
       ├─ 处理 elapsed_ticks 上限（见下）
       ├─ Spawn PlayerSupervisor + SimulationActor
       ├─ SimulationActor.Receive(LoadStateMsg{state})
       ├─ SimulationActor.Receive(AdvanceMsg{ticks: elapsed_ticks})
       ├─ 收集产生的 events → 推送给客户端作为"弟子日记"
       └─ 切换到在线 tick 模式
```

### 离线时间上限

长时间离线的玩家补偿模拟会很慢：
- 离线 7 天 = 604800 tick，按 10μs/tick 估算 ≈ 6 秒
- 离线 30 天 ≈ 26 秒（不可接受）

**V1 策略**：
- 最大补偿时长 **7 天**（604800 tick）
- 超过部分直接跳过，弹"久违重逢"剧情（这本身是个 storylet）
- 跳过时 `last_seen_wall_ms` 仍然推进到当前，下次从当前开始
- 这符合 §2.3 "宿命感"的叙事（长期失联有惩罚但不暴击）

M1 末可以评估是否引入"跳跃式补偿"（见 ADR 0010 未决问题）把上限提到 30 天。

### 恢复失败降级

如果反序列化失败（schema 破坏 / blob corrupted）：

```
1. 尝试从 player_events 表重建最近 N tick 的状态
2. 如果失败 → 从最近的 manual_backup 加载（如有）
3. 如果仍失败 → 上报严重告警，让该玩家登录失败，提示"存档损坏请联系客服"
```

**严禁**"遇到损坏就初始化新存档"。这会让玩家丢失所有弟子，是最严重的事故。

---

## 三版本治理

### 定义

| 版本号 | 含义 | 变更触发 |
|---|---|---|
| `schema_version` | 数据库 / blob 存储布局 | 新增 / 重命名 / 删除 `player_state` 表列；proto 消息新增 field（field number 新增） |
| `simulation_version` | `advance()` 函数行为 + State 结构 | 调整数值、修 bug、改变行为逻辑、改变 State 字段含义（不是 proto 层面的变更） |
| `config_version` | 游戏内容配置 | 新增 / 修改 / 下线 storylet、弟子模板、建筑、资源等配置 |

### 独立演进，但组合触发迁移

每个版本号独立递增。加载存档时按顺序检查：

```
load_state(player_id):
    row = SELECT * FROM player_state WHERE player_id = ?

    # 1. schema 迁移（存储层）
    if row.schema_version < CURRENT_SCHEMA_VERSION:
        for v in range(row.schema_version, CURRENT_SCHEMA_VERSION):
            run schema_migrations[v]  # SQL + blob 解码

    # 2. state blob 反序列化
    state = Proto.Unmarshal(row.state_blob)

    # 3. simulation 迁移（逻辑层）
    if row.simulation_version < CURRENT_SIMULATION_VERSION:
        for v in range(row.simulation_version, CURRENT_SIMULATION_VERSION):
            state = simulation_migrations[v](state)

    # 4. config 迁移（内容层）
    if row.config_version < CURRENT_CONFIG_VERSION:
        for v in range(row.config_version, CURRENT_CONFIG_VERSION):
            state = config_migrations[v](state, config_diff[v])

    # 5. 所有迁移结果写回
    UPDATE player_state SET
        schema_version = CURRENT_SCHEMA_VERSION,
        simulation_version = CURRENT_SIMULATION_VERSION,
        config_version = CURRENT_CONFIG_VERSION,
        state_blob = Proto.Marshal(state),
        ...

    return state
```

### 迁移脚本规范

三种迁移脚本放三个独立目录：

```
server/migrations/
├── schema/             ← SQL + blob 结构迁移
│   ├── 001_initial.sql
│   ├── 002_add_disciple_count_column.sql
│   └── ...
├── simulation/         ← Go 代码迁移
│   ├── v1_to_v2.go
│   └── ...
└── config/             ← Go 代码迁移，消费 config diff
    ├── v1_to_v2.go
    └── ...
```

**每个迁移脚本的约束**：
- 必须是**幂等的**（跑两次结果一样）
- 必须是**确定性的**（纯函数，不依赖 wall clock / 随机数种子）
- 必须有**对应的单元测试**（老 state 进入 → 新 state 出来 → 用新 advance 跑 100 tick 不崩）
- 必须写 `player_versions` 表留痕

### config_version 的特殊性：ID 不可删除

`design-decisions.md` §4.4 已经规定"事件 / 弟子 / 建筑的 ID 永不修改，只能新增"。这条铁律是 config_version 能独立演进的前提。

但**删除不是修改**——有时候我们需要下线某个 storylet。处理方式：

1. 标记 `deprecated: true`，新玩家不触发
2. 老玩家如果正在跑这个 storylet，config 迁移脚本负责"平滑结束"（推进到了结 stage 或跳过）
3. 6 个月 deprecation window 后真正从配置里移除，但 ID 仍然 reserved

---

## Hollywood Storer 接口集成

Hollywood 的 Storer 是简单的 KV 接口（来自 ADR 0007 源码阅读）：

```go
// Hollywood upstream
type Storer interface {
    Set(key string, value []byte) error
    Get(key string) ([]byte, bool, error)
    Delete(key string) error
}
```

我们自建的 `PGStore` 实现这个接口：

```go
// server/internal/slggame/storage/pgstore/pgstore.go
package pgstore

type PGStore struct {
    db *sql.DB
}

func (p *PGStore) Set(key string, value []byte) error {
    // key 格式："player:{player_id}:state"
    // value 是 protobuf-serialized State
    return p.upsertPlayerState(parseKey(key), value)
}
```

**但 PGStore 只是最底层的 KV 适配**，我们的业务不会直接用它。真正的业务入口是：

```go
// server/internal/slggame/storage/saves.go
type SaveRepo interface {
    LoadPlayerState(ctx, playerID) (*State, error)
    SavePlayerState(ctx, playerID, state *State, events []Event) error
    AppendEvents(ctx, playerID, events []Event) error
    MarkEventsAcknowledged(ctx, playerID, upToTick int64) error
    SoftDeletePlayer(ctx, playerID, reason string) error
}
```

`SaveRepo` 实现时可以内部用 PGStore 做 blob 写入 + 直接 SQL 做索引列和 events 表。**业务层只依赖 `SaveRepo` 接口**，不知道底层是 Hollywood Storer + PostgreSQL 还是别的什么。

这让我们享受 Hollywood 的 `WithPersistence` 中间件生态，又不被它的 KV 抽象束缚。

---

## 大小与性能估算

### 单玩家 State 大小（V1 规模）

| 子结构 | 估算 |
|---|---|
| 宗门基础（名字 / 等级 / 资源 × 6） | ~500 bytes |
| 弟子 × 10（6 属性 + 特质 + 因果状态 + 位置） | ~5 KB |
| 建筑 × 10（类型 + 等级 + 进度 + 位置） | ~1.5 KB |
| 活跃 storylet × 5（stage + 变量快照） | ~1 KB |
| Utility AI 权重矩阵（~50 格） | ~500 bytes |
| PRNG 状态 | ~200 bytes |
| 已完成 storylet 索引（最多 100 条） | ~1 KB |
| 元数据 + 预留 | ~500 bytes |
| **合计** | **~10 KB** |

protobuf 压缩后实际落盘预估 **6~8 KB**。

### 5000 DAU 场景写负载

- 快照频率：平均 0.01 次/秒/玩家（事件驱动）
- 写入负载：5000 × 0.01 = **50 写/秒**
- 数据量：50 × 8 KB = **400 KB/s**
- PostgreSQL 轻松处理

### 5000 DAU 热数据

- 内存中 actor 持有的 State：5000 × 10 KB = **50 MB**
- PG player_state 表总大小：5000 × 8 KB = **40 MB**
- 加索引表和事件表总计 < **200 MB**
- 单机 16GB 服务器绰绰有余

### 登录耗时预算

- player_state 查询：~2ms
- Protobuf 反序列化 8KB：~0.1ms
- 离线补偿（平均 1 小时 = 3600 tick × 10μs）：~36ms
- Hollywood 启动 actor 树：~5ms
- 推送初始状态到客户端：~10ms
- **合计 ~55ms**，符合"登录首屏 ≤ 5s"的 §8 压测目标

---

## Consequences

### 正面影响

1. **与 Hollywood 生态零冲突**
   - Storer 接口契合，享受 WithPersistence 中间件
   - 持久化逻辑通过 actor 隔离不阻塞模拟

2. **三版本独立演进**
   - schema / simulation / config 各自可以独立发布
   - 减少耦合导致的回滚代价

3. **瘦存档，符合 §4.3 哲学**
   - 10 KB / 玩家远小于常见云存档限制
   - 配置依然走 CDN，不进 blob

4. **protobuf 双端对称**
   - 服务端 Go + 客户端 TypeScript 共享 .proto
   - 配合后续 save-sync-protocol.md 的增量同步，架构一致

5. **事件流天然支持弟子日记**
   - 离线补偿产出的事件直接进 player_events 表
   - 下次登录 SELECT 即可构造日记 UI

6. **数据安全合规基础**
   - 软删除 + 30 天恢复窗口
   - 明确的数据归档路径

### 负面影响

1. **blob 非关系化，查询受限**
   - 想查"哪些玩家的弟子叫 XX"需要 scan 全表
   - 缓解：关键字段（sect_name / disciple_count）冗余到索引列；未来可加专用索引表
   - V1 不是问题，社交功能上线时再处理

2. **版本治理心智负担**
   - 三个版本号任何一个升级都要写迁移
   - 每次迁移要写测试和文档
   - 这是**必要的复杂度**，砍掉会在 3 个月后引起事故

3. **事件表会持续增长**
   - 每玩家每天约 1000 事件，10KB
   - 5000 DAU × 365 天 = 18 GB/年
   - V1 不清理，M3 前决定归档策略

4. **protobuf 工具链初始成本**
   - 需要 protoc + vtprotobuf + 代码生成脚本
   - 这本来就要做（Hollywood Remote 也用）
   - 纳入 M0 前置工作

5. **登出同步写 5 秒超时是硬性 SLA**
   - 数据库慢的时候玩家感知"登出卡顿"
   - 必须监控 p99 登出时长

### 中性影响

- **不使用 JSONB 列**：虽然 PG 支持 JSONB 且可以部分查询，但和 protobuf 混用会引入两种序列化格式，增加心智负担。V1 不用。
- **不使用 MongoDB / 其他 NoSQL**：`legacy/04` 已经选定 PostgreSQL，本 ADR 不挑战这个决策。
- **不分片**：V1 单实例 PG 足够，M3 前不讨论分片。

---

## Alternatives Considered

### A. 完全归一化存储

把 State 的每个子结构（disciple / building / storylet）都拆成独立表，用外键关联。

- ✅ 可查询性最好，支持任意 SQL 分析
- ❌ 写入一次快照 = 几十个 INSERT/UPDATE，负载暴涨
- ❌ 事务复杂
- ❌ 和 "actor 里拿的是内存 State" 的心智模型不符
- ❌ Schema 迁移代价是 N 倍
- **裁定**: ❌ 不采用

### B. 纯 JSONB

用 `JSONB` 列替代 `BYTEA`，保留部分查询能力。

- ✅ 可读性好
- ✅ PG 原生支持 JSON 路径查询
- ❌ 尺寸比 protobuf 大 2~3 倍
- ❌ 失去 schema 强制
- ❌ 和增量同步协议分道扬镳
- **裁定**: ❌ 不采用

### C. 纯 KV（Redis 作主存）

Redis 作主存，PG 仅作备份。

- ✅ 性能极高
- ❌ 持久化不保证
- ❌ 事务语义弱
- ❌ 版本迁移困难
- ❌ 成本结构不合适（内存贵）
- **裁定**: ❌ 不采用。Redis 用作会话缓存 / 热数据，不作权威

### D. Event Sourcing

只存事件流，State 从事件重建。

- ✅ 完美的审计日志
- ✅ 时光回溯天然支持
- ❌ 重建代价高（特别对离线回归玩家）
- ❌ schema 演进极其复杂
- ❌ 工程复杂度远超 V1 需要
- **裁定**: ❌ 不采用。V1 混合方案（state snapshot + events table）已经吸收了 ES 的优点

---

## Risks

| 风险 | 严重度 | 缓解 |
|---|---|---|
| 登出同步写超时导致数据回退 | 🔴 高 | 超时 5s + 硬性告警 + 超时后用 async persistence 继续尝试 + 监控 p99 登出时长 |
| Blob 损坏无法反序列化 | 🟡 中 | 写入时计算 CRC32 冗余存储 + 从 events 表重建 + manual backup |
| 版本迁移脚本有 bug | 🟡 中 | 所有迁移必须有单元测试 + CI 跑"旧存档 × 新逻辑"兼容性测试 + player_versions 表留痕便于回溯 |
| 事件表膨胀影响查询性能 | 🟡 中 | 按 (player_id, game_tick) 分区 + 6 个月后归档冷存储 + 只查 acknowledged=false 的部分索引 |
| DB 写负载超预期 | 🟢 低 | 按事件驱动而非 tick 驱动，负载可控；监控 + 告警 |
| protobuf field number 冲突 | 🟢 低 | CI 检查 .proto 的 reserved 声明 + code review 清单 |
| GDPR 数据删除合规 | 🟢 低 | 软删表 + 定时清理 job + 审计日志 |
| 长期离线补偿卡顿 | 🟡 中 | 7 天上限 + "久违重逢" storylet + M1 末评估跳跃式补偿 |

---

## 使用策略（必须遵守）

### 代码组织

```
server/internal/slggame/storage/
├── pgstore/
│   ├── pgstore.go          # Hollywood Storer 适配器
│   └── pgstore_test.go
├── saves.go                # SaveRepo 接口
├── saves_pg.go             # SaveRepo 的 PG 实现
├── migrations.go           # 迁移执行器
└── saves_test.go
```

业务代码**只能通过 SaveRepo 接口访问持久化**。禁止在 simulation / actors 层直接写 SQL 或调用 PGStore。

### 协议演进纪律

1. **Proto field number 永不复用**，废弃字段用 `reserved`
2. **三个版本号独立递增**，每次发版 code review 必须确认是否需要 bump
3. **迁移脚本必有测试**，CI 阻塞测试项
4. **新增 State 字段必须有默认值**，使老存档反序列化不崩
5. **删除 State 字段必须先 deprecated 一个版本**，再走迁移移除

### 监控必备

上线前必须埋点：

- `save.snapshot.latency` (p50/p99)
- `save.load.latency` (p50/p99)
- `save.blob.size_bytes` (histogram)
- `save.events.backlog_per_player` (gauge)
- `save.migration.executed` (counter by version)
- `save.logout.sync_timeout` (counter)
- `save.corrupt.detected` (counter)

---

## 架构初稿

```
Hollywood Engine
└── GatewayActor
    └── PlayerSupervisor (per online player)
        ├── SimulationActor
        │   └── 持有 State，每 tick 调用 simulation.Advance()
        ├── PersistenceActor       ← 本 ADR 核心
        │   ├── 接收 SnapshotRequest{state, events}
        │   ├── 序列化 state → blob
        │   ├── 调用 SaveRepo.SavePlayerState()
        │   └── 失败重试 + 告警
        ├── TickActor
        └── SyncActor

存储层（不是 actor）:
    SaveRepo interface
        └── saves_pg.go implementation
              ├── 写 player_state（blob + 冗余列）
              ├── 写 player_events
              └── 写 player_versions（迁移时）
```

---

## 未决问题

- [ ] **事件表归档策略**：6 个月后冷数据放 S3？还是 PG 分区 + `cold` 分区？M3 上线前定
- [ ] **灾难恢复**：单机 PG 挂了怎么办？V1 依赖云 RDS 的自动备份；自建场景另起 ADR
- [ ] **跨服迁移**：未来分片后玩家怎么迁移到新 shard？留给 cluster ADR
- [ ] **Protobuf field number 分配策略**：是否需要为不同模块预留段（如 disciple 1-99 / building 100-199）？Proto 实现时定
- [ ] **CRC 校验字段**：是否在 state_blob 外加一列 CRC32 防静默损坏？倾向加，M0 实现时定
- [ ] **手动快照与玩家可见的"存档点"**：V1 不做，但未来"渡劫前手动存档"是合理需求
- [ ] **player_id 生成策略**：UUID v7？雪花 ID？小游戏 openid 映射？另起 ADR 0009 Actor ID 规范

---

## 引用

- PostgreSQL JSONB vs BYTEA 性能对比（`https://www.postgresql.org/docs/current/datatype-json.html`）
- vtprotobuf 项目（`https://github.com/planetscale/vtprotobuf`）—— 已是 Hollywood 的间接依赖
- Hollywood `actor/middleware/persistence/` 源码 —— Storer 接口定义
- Fallen London 技术分享：Storylet 系统的存档结构
- Factorio 存档格式文档：二进制 blob + 版本化字段
