# 修仙宗门模拟经营游戏设计文档 v3.1：产品与权威后端主规范 / Authoritative Backend GDD

> 版本：v3.1
> 状态：Active / Authoritative
> 日期：2026-04-23
> 用途：本文件用于指导产品拆解、后端智能体、代码生成智能体与工程实现者按同一套主规范推进《我的宗门》。`development-plan.json` 是对本文的执行拆解，不得反向改写本文。
> 架构核心：**服务器权威状态 + Hollywood Actor big-state + 事件应用流水线 + protobuf 快照 blob + 有界事件日志 + 低频模拟 Tick + 离线追赶结算**。
> 玩法核心：**宗门供养弟子，弟子反哺宗门，贡献分配资源，弟子成长，宗门扩张，扩张带来风险，风险要求更强制度与更强弟子。**

---

## 0. 给实现智能体的硬性约束

本节是后端实现的最高优先级约束。任何代码实现、模块拆分、数据库设计、接口设计都不得违背这些约束。

### 0.1 服务端权威原则

1. 客户端永远不能直接修改游戏状态。
2. 客户端只能提交“意图命令”，例如 `BuildBuilding`、`AssignDiscipleTask`、`StartCultivation`。
3. 服务端必须校验命令是否合法，合法后才产生领域事件并变更状态。
4. 状态变更必须由服务端产生的 `DomainEvent` 驱动。
5. 客户端收到的是 `Snapshot`、`StatePatch`、`EventLog` 或 `CommandResult`，只负责渲染。
6. 客户端提交的资源变化、弟子成长、任务完成结果都不可信。
7. 前端不能提交“我获得了 100 灵石”，只能提交“我要领取任务奖励”，奖励由服务端根据任务状态计算。

### 0.2 Actor 边界原则

1. **一个玩家宗门 = 一个 `SectActor`。**
2. `SectActor` 是该宗门内存状态的唯一权威持有者。
3. 同一时间，同一个 `sect_id` 只能有一个活跃 `SectActor` owner。
4. 第一版不要把每个弟子、每个建筑、每个任务都做成 Actor。
5. 弟子、建筑、任务、库存、贡献、事件都是 `SectActor` 内部状态结构。
6. 只有跨宗门、全局、异步、共享资源系统才考虑独立 Actor，例如 `MarketActor`、`RankingActor`、`ExpeditionActor`、`WorldClockActor`。
7. 任何跨 Actor 交互必须通过消息完成，不能直接共享可变内存。

### 0.3 状态变更原则

每个玩家命令的处理必须遵循以下顺序：

```text
Receive ClientCommand
  → ValidateAuthAndOwnership
  → RouteToSectActor
  → CheckIdempotency
  → ValidateAgainstSectState
  → ProduceDomainEvents
  → PersistCommandAndEvents
  → ApplyEventsToMemoryState
  → ProduceStatePatchOrSnapshotDelta
  → SendCommandResult
  → PushToCurrentPlayerSession
```

推荐强一致命令在命令记录与事件成功写入数据库后再返回成功。普通 UI 查询不得修改状态。定时模拟也必须走事件和状态应用逻辑，不允许绕过。

冻结解释：

1. `ValidateAgainstSectState` 只能读取当前 `SectState`，不能直接修改它。
2. 命令处理器、离线追赶、低频 tick 推进都只能产出 `DomainEvent`，不能直接写 `SectState`。
3. `ApplyEvent` 是唯一允许改写玩法状态的入口；任何“先改状态，再补事件”的做法都视为违反主架构。
4. 该流水线服务于 snapshot-first 权威持久化：命令与事件先形成可审计增量，再由 `ApplyEvent` 推进内存真相，最终落入 `snapshot_blob` 作为恢复主源。

V1 默认只有“当前玩家自己的会话”这一条推送面，不存在跨玩家广播给观察者的主路径。

### 0.4 持久化原则

1. 内存状态不是最终存储。
2. 每个宗门需要 `snapshot_blob + query indexes + event_log + command_log`。
3. `snapshot_blob` 是权威持久化主源，用于快速加载宗门状态。
4. `query indexes` 是可重建的 bounded projections，用于查询与排障，不是权威恢复主源。
5. `event_log` 用于审计、回放、debug、弟子日记、补偿排查，不作为唯一持久化数据源。
6. 每个命令必须有 `cmd_id`，防止网络重试导致重复执行。
7. 每个宗门状态必须有单调递增的 `version`。
8. 每个领域事件必须带 `version`。
9. 加载宗门时，必须加载最新 `snapshot_blob`，再回放 snapshot 之后的 bounded events 或执行 offline catch-up。
10. `SectActor` 停止前必须尝试保存 `snapshot_blob`。
11. 离线追赶结算也必须写事件并更新快照。
12. 正式权威持久化与协议默认使用 Protobuf，而不是 JSON。

### 0.5 时间推进原则

本游戏不是动作游戏，不采用高频实时同步。

推荐：

```text
在线 UI Tick：1 秒或 5 秒一次，只做轻量进度刷新
游戏日结算：按游戏时间推进，每日结算弟子行为、生产、任务、修炼
月结算：俸禄、义务、贡献兑付、满意与忠诚
年结算：收徒、大比、年度考核、年度事件
离线结算：玩家上线时根据 last_simulated_at 做批量追赶
```

禁止在 10Hz、20Hz、60Hz 的动作游戏 match-loop 思维下实现本游戏后端。

### 0.6 玩法边界原则

1. 不设置“宗门灵气库存”。
2. 灵气只作为环境、建筑、修炼效率加成存在。
3. 灵石是货币、修炼燃料、阵法供能资源。
4. 贡献点不是宗门库存，而是弟子对宗门资源的索取权。
5. 不新增大量细碎资源，例如各种灵草、各种矿石，早期只保留灵植、矿材、妖材、阵材等抽象资源。
6. 每个系统必须进入核心闭环：资源供养弟子、弟子反哺宗门、贡献分配资源、弟子成长、宗门扩张、风险升级。
7. 若一个功能无法进入核心闭环，MVP 不实现。

---

## 1. 项目定位与重构目标

### 1.1 游戏定位

本游戏在 V1 的真实形态是：**单人宗门经营 + 云存档 + 服务端权威**。玩家不是单个主角，而是一个宗门的经营者。

多个玩家可以同时在线，但 V1 中每位玩家只经营自己的宗门，不共享同一份实时状态。未来的跨宗门访问、房间式活动、排行榜与共享地图属于后续阶段，不属于 V1 默认同步模型。

玩家需要经营：弟子来源、宗门资源、弟子成长、建筑生产、任务堂、功勋宝库、月例与义务、晋升与大比、修炼与突破、宗门事件、坊市、外务、秘境、宗门战等系统。

玩家的个人宗门场景本质上是一个服务器托管的单玩家权威实例：

```text
玩家前端 = 宗门场景观察者 + 操作提交者
服务端 SectActor = 宗门状态权威持有者
```

### 1.2 技术重构目标

本版本 GDD 不只是玩法文档，而是“玩法 + 技术实现规格”。目标是让后端智能体可以直接按本文档实现：

1. Hollywood Actor 架构。
2. 宗门状态数据结构。
3. 领域命令。
4. 领域事件。
5. 状态补丁。
6. 游戏日/月/年模拟。
7. 各玩法系统的内部状态、命令、事件、Actor 归属。
8. MVP 与后续阶段范围。

### 1.3 推荐技术栈

```text
语言：Go 1.21+
Actor 引擎：github.com/anthdm/hollywood
网关：WebSocket + HTTP
数据协议：Protobuf-first
数据库：PostgreSQL
缓存/分布式锁：Redis 或 PostgreSQL advisory lock
任务调度：SectActor 低频 Tick + Scheduler jobs
日志：slog + structured logs
测试：Go test + golden replay tests
```

### 1.4 系统总览

```text
Client
  ↓ WebSocket / HTTP (Protobuf)
Gateway / Session Layer
  ↓
Command Router
  ↓
SectActor
  ↓
Domain Modules
  ↓
StateBlob + EventLog + CommandLog
  ↓
Current-Session Push / SnapshotDelta
```

基础支撑系统：

```text
SessionRegistryActor
PersistenceActor
ClientPushActor
WorldClockActor
AdminActor
```

未来扩展系统（非 V1 默认主路径）：

```text
MarketActor
RankingActor
EventSchedulerActor
ExpeditionActor
CompetitionActor
WarActor
```

---

## 2. 架构思想：Nakama Authoritative 思路 + Hollywood Actor 实现

### 2.1 概念映射

| Nakama authoritative 概念 | 本项目 Hollywood 实现 |
|---|---|
| Match | SectActor |
| Match state | SectState |
| Match Init | SectActor Started → LoadSnapshot → ReplayEvents |
| Match Join Attempt | JoinScene 权限校验 |
| Match Join | AddPresence / AddWatcher |
| Match Leave | RemovePresence / IdleStopTimer |
| Match Loop | Tick / AdvancePhase / AdvanceDay |
| Match Terminate | SaveSnapshot / ReleaseLease / StopActor |
| Match Signal | AdminSignal / SystemSignal |
| Op code | CommandType / EventType |
| Broadcast | ClientPushActor 向当前玩家会话推送 StatePatch / SnapshotDelta |
| Match Label | SectPublicProfile / SceneIndex |

### 2.2 为什么选择 Actor

Actor 模型适合本游戏的原因：

1. 每个宗门天然是独立状态边界。
2. 同一宗门内的命令可以串行处理，避免资源并发扣减错误。
3. 玩家操作、定时结算、事件触发都可以统一为消息。
4. 跨宗门系统可以通过独立 Actor 进行协调。
5. 在线时加载 Actor，离线时保存并停止，适合大量玩家不同时在线的场景。

### 2.3 不采用传统 MMO 大世界模型

本游戏不是：

```text
所有玩家在同一张大地图实时移动
```

而是：

```text
每个玩家有自己的服务器权威宗门实例；
其他玩家可以访问、交易、挑战、协作，但不能直接改其状态；
跨玩家玩法通过独立 Actor 或请求目标 SectActor 实现。
```

---

## 3. 核心 ID、版本与时间模型

### 3.1 ID 规范

所有 ID 使用字符串，推荐格式：

```text
user_id:      usr_xxx
sect_id:      sect_xxx
disciple_id:  dis_xxx
building_id:  bld_xxx
task_id:      task_xxx
event_id:     evt_xxx
cmd_id:       cmd_xxx 或 UUID
item_id:      item_xxx
formation_id: form_xxx
```

### 3.2 版本规范

每个 `SectState` 必须包含：

```go
type Version int64
```

规则：

1. 每个成功改变宗门状态的领域事件都使 `version + 1`。
2. 一个命令可能产生多个事件，每个事件占用一个版本号。
3. `CommandResult.scene_version` 返回命令处理后的最终版本。
4. 客户端提交 `base_version`，可用于检测客户端是否过旧。
5. 客户端过旧时仍可处理命令，但返回结果必须包含最新 patch 或提示重新拉取 snapshot。

### 3.3 游戏时间

```go
type GameTime struct {
    Year  int `json:"year"`
    Month int `json:"month"` // 1-12；每月 30 日
    Day   int `json:"day"`   // 1-30
    Phase DayPhase `json:"phase"`
    TotalDays int64 `json:"total_days"`
}

type DayPhase string

const (
    PhaseMorning DayPhase = "morning"
    PhaseDaytime DayPhase = "daytime"
    PhaseEvening DayPhase = "evening"
    PhaseNight   DayPhase = "night"
)
```

时间单位：

| 单位 | 规则 |
|---|---|
| 1 日 | 弟子行为与任务进度结算 |
| 10 日 | 小生产周期 |
| 30 日 | 月例、义务、贡献兑付、满意忠诚 |
| 90 日 | 季节、灵田、事件权重 |
| 360 日 | 收徒、大比、年度考核 |

### 3.4 真实时间与游戏时间

推荐 MVP 配置：

```go
type TimeConfig struct {
    RealtimeSecondsPerGameDay int // 默认 300，即在线 5 分钟推进 1 游戏日
    MaxOfflineCatchupDays     int // 默认 30，单次上线最多结算 30 游戏日
    MaxDailyStepsPerTick      int // 默认 3，避免一次消息阻塞过久
}
```

如果偏放置，可配置为真实 1 小时 = 游戏 1 日，真实 1 天 = 游戏 30 日。该设定属于数值配置，不改变架构。

---

## 4. Actor 总体设计

### 4.1 Actor 列表

| Actor | 阶段 | 作用 |
|---|---|---|
| `GatewaySessionActor` | V1 核心 | 管理单个连接、鉴权、转发命令、返回结果 |
| `SessionRegistryActor` | V1 核心 | 管理 `player_id/sect_id` 到 `SectActor` PID 的映射与单 owner 保护 |
| `SectActor` | V1 核心 | 玩家宗门的唯一权威状态机 |
| `PersistenceActor` | V1 核心 | 读写 snapshot blob、event_log、command_log |
| `ClientPushActor` | V1 核心 | 向当前玩家在线连接推送 snapshot、delta、通知 |
| `WorldClockActor` | V1 核心 | 向在线 `SectActor` 发送低频 Tick / AdvanceDay |
| `EventSchedulerActor` | V1 后续 | 生成宗门事件、拜山事件、危机事件 |
| `MarketActor` | V2+ | 处理坊市、交易、外部委托 |
| `RankingActor` | V2+ | 排行榜、名望榜、战力榜 |
| `CompetitionActor` | V2+ | 大比、演武、赛程 |
| `ExpeditionActor` | V2+ | 秘境、联合外务、跨宗门探索 |
| `WarActor` | V3+ | 宗门战 |
| `AdminActor` | V1 核心 / 后续扩展 | GM 指令、调试、补偿、回放 |

### 4.2 Actor 粒度规则

必须坚持：

```text
SectActor 内部聚合弟子、建筑、任务、资源、贡献、事件。
```

不要这样做：

```text
DiscipleActor
BuildingActor
TaskActor
ResourceActor
InventoryActor
```

除非后续性能和玩法证明某个系统必须独立，否则都作为 `SectState` 内部模块。

### 4.3 SessionRegistryActor

职责：

1. 根据 `sect_id` 找到对应 `SectActor`。
2. 若不存在，则获取分布式 lease 并创建。
3. 维护本节点内存映射。
4. 处理 Actor 停止后的映射清理。
5. 防止同一个宗门多节点双开。

状态：

```go
type SessionRegistryState struct {
    Scenes map[SectID]*SceneEntry
}

type SceneEntry struct {
    SectID      SectID
    PID         string
    NodeID      string
    OwnerUserID UserID
    Status      SceneStatus
    LastActiveAt time.Time
    LeaseExpireAt time.Time
}

type SceneStatus string

const (
    SceneLoading SceneStatus = "loading"
    SceneReady   SceneStatus = "ready"
    SceneStopping SceneStatus = "stopping"
)
```

消息：

```go
type GetOrSpawnScene struct {
    SectID SectID
    UserID UserID
    ReplyTo PID
}

type SceneReady struct {
    SectID SectID
    PID PID
}

type ReleaseScene struct {
    SectID SectID
    PID PID
}

type RenewSceneLease struct {
    SectID SectID
    PID PID
}
```

处理规则：

```text
GetOrSpawnScene:
  if local registry has ready scene:
      reply PID
  else:
      acquire distributed lease for sect_id
      if success:
          spawn SectActor
          mark loading
      else:
          ask remote owner or return retry
```

### 4.4 SectActor

职责：

1. 持有 `SectState`。
2. 处理玩家命令。
3. 处理定时推进。
4. 生成领域事件。
5. 应用领域事件。
6. 生成状态补丁。
7. 管理当前玩家在线 session。
8. 触发持久化。
9. 空闲停止。

状态：

```go
type SectActorState struct {
    SectID SectID
    State *SectState
    Loaded bool
    LoadingError error

    Sessions map[SessionID]SessionInfo
    RecentCommands LRUMap[CommandID, CommandResult]
    PendingPersist []DomainEvent

    LastActiveAt time.Time
    IdleStopScheduled bool
}
```

生命周期：

```text
Started
  → Send LoadSectSnapshot to PersistenceActor
  → Receive SnapshotLoaded
  → ReplayEventsAfterSnapshot
  → ApplyOfflineCatchup
  → Mark Ready

JoinScene
  → Auth owner or admin
  → Add session
  → Send snapshot to current session

SubmitCommand
  → Check loaded
  → Check ownership/permission
  → Idempotency
  → Validate command
  → Produce events
  → Persist events
  → Apply events
  → Push patch to current session

Tick
  → Only light progress, no heavy DB IO

AdvanceDay
  → Run daily simulation in fixed order

Stopped
  → Save snapshot
  → Release registry lease
```

主要消息：

```go
type JoinScene struct {
    UserID UserID
    SessionID SessionID
    ViewMode ViewMode // owner, admin
    ReplyTo PID
}

type LeaveScene struct {
    UserID UserID
    SessionID SessionID
}

type SubmitCommand struct {
    Command ClientCommand
    ReplyTo PID
}

type TickScene struct {
    Now time.Time
}

type AdvanceDay struct {
    Reason AdvanceReason
    MaxDays int
}

type SaveSceneSnapshot struct {
    Reason string
}

type StopIfIdle struct{}
```

### 4.5 PersistenceActor

职责：加载 snapshot、加载 event log、写入 domain events、写入 command log、保存 snapshot、提供回放数据。

消息：

```go
type LoadSectState struct {
    SectID SectID
    ReplyTo PID
}

type AppendSectEvents struct {
    SectID SectID
    CommandID CommandID
    ExpectedBaseVersion Version
    Events []DomainEvent
    ReplyTo PID
}

type SaveSectSnapshot struct {
    SectID SectID
    Version Version
    StateBlob []byte
    Reason string
    ReplyTo PID
}
```

推荐实现：先生成 events，不应用；持久化成功后 apply；若持久化失败，内存不变，返回 `PERSIST_FAILED`。

### 4.6 ClientPushActor

职责：管理当前玩家 `session_id` 到连接的映射；向该玩家推送私有 snapshot、delta、通知、事件日志；处理断线。

```go
type PushToSession struct {
    SessionID SessionID
    Message ServerPush
}

type PushSectPatch struct {
    SessionID SessionID
    Patch StatePatch
}
```

### 4.7 WorldClockActor

职责：定期向在线 `SectActor` 发送 `TickScene`；达到游戏日推进条件时发送 `AdvanceDay`；避免所有宗门同一时间集中结算；支持暂停、加速、运维控制。

```go
type WorldClockState struct {
    OnlineScenes map[SectID]PID
    TimeConfig TimeConfig
    NextTickAt time.Time
}
```

规则：`WorldClockActor` 不直接修改宗门状态，只发消息给 `SectActor`。

### 4.8 EventSchedulerActor

职责：根据宗门状态生成候选事件；维护全局事件权重；生成拜山、危机、机缘、内部事件；不直接修改宗门状态，只返回事件候选。

```go
type RequestSectEventCandidates struct {
    SectID SectID
    Snapshot EventContextSnapshot
    Count int
    ReplyTo PID
}
```

### 4.9 MarketActor / ExpeditionActor / CompetitionActor

这些 Actor 属于二期或三期。核心原则：它们不直接修改 `SectState`，只能给 `SectActor` 返回交易、远征、比赛结果，最终由 `SectActor` 生成并应用本宗门事件。

---

## 5. 通信协议设计

### 5.1 ClientCommand Envelope

正式权威协议统一使用 Protobuf。下面展示的是语义结构，而不是 JSON 传输层：

```proto
message ClientCommand {
  string cmd_id = 1;
  string user_id = 2;
  string sect_id = 3;
  CommandType type = 4;
  bytes payload = 5;
  int64 client_seq = 6;
  uint64 base_version = 7;
  int64 sent_at_wall_ms = 8;
}
```

### 5.2 CommandResult

```proto
message CommandResult {
  string cmd_id = 1;
  bool accepted = 2;
  string error_code = 3;
  string error_message = 4;
  string sect_id = 5;
  uint64 scene_version = 6;
  repeated ClientEvent events = 7;
  StatePatch patch = 8;
  bool need_snapshot = 9;
}
```

### 5.3 ServerPush

```proto
message ServerPush {
  ServerPushType type = 1; // command_result, snapshot, patch, notification, event_log
  bytes data = 2;
}
```

### 5.4 StatePatch

正式权威路径中的 `StatePatch` 是一个 Protobuf 增量消息，而不是 JSON Merge Patch。JSON 形式只允许保留给调试工具或 GM 界面，不构成权威传输协议。

```proto
message StatePatch {
  string sect_id = 1;
  uint64 from_version = 2;
  uint64 to_version = 3;
  repeated PatchOp ops = 4;
}

message PatchOp {
  PatchOpType op = 1; // set, inc, remove, append
  string path = 2;
  bytes value = 3;
}
```

### 5.5 DomainEvent

```go
type DomainEvent struct {
    EventID   EventID
    SectID    SectID
    Version   Version
    Type      EventType
    Payload   []byte
    CmdID     CommandID
    CreatedAt time.Time
    GameTime  GameTime
}
```

### 5.6 错误码

| ErrorCode | 含义 |
|---|---|
| `AUTH_REQUIRED` | 未鉴权 |
| `NOT_SCENE_OWNER` | 非宗门主人，无权执行 |
| `SCENE_NOT_READY` | Actor 未加载完 |
| `STALE_CLIENT_VERSION` | 客户端版本太旧，需要刷新 |
| `INVALID_COMMAND` | 命令格式错误 |
| `INSUFFICIENT_RESOURCE` | 资源不足 |
| `BUILDING_LOCKED` | 建筑未解锁 |
| `DISCIPLE_NOT_FOUND` | 弟子不存在 |
| `DISCIPLE_BUSY` | 弟子已有任务 |
| `TASK_NOT_FOUND` | 任务不存在 |
| `TASK_REQUIREMENT_NOT_MET` | 任务条件不满足 |
| `MONTHLY_LIMIT_REACHED` | 兑换限购达到上限 |
| `CONTRIBUTION_NOT_ENOUGH` | 弟子贡献不足 |
| `COOLDOWN_NOT_READY` | 冷却未结束 |
| `PERSIST_FAILED` | 持久化失败 |
| `INTERNAL_ERROR` | 内部错误 |

---

## 6. SectState 核心数据结构

### 6.1 根状态

```go
type SectState struct {
    Meta        SectMeta             `json:"meta"`
    Runtime     SectRuntime          `json:"runtime"`
    Time        GameTime             `json:"time"`

    Resources   ResourceState        `json:"resources"`
    Inventory   InventoryState       `json:"inventory"`
    Contribution ContributionState   `json:"contribution"`

    Disciples   map[DiscipleID]DiscipleState `json:"disciples"`
    Buildings   map[BuildingID]BuildingState `json:"buildings"`
    Tasks       map[TaskID]TaskState         `json:"tasks"`
    Productions map[ProductionID]ProductionJob `json:"productions"`
    Formations  map[FormationID]FormationState `json:"formations"`

    Policies    PolicyState          `json:"policies"`
    Admissions  AdmissionState       `json:"admissions"`
    Events      SectEventState       `json:"events"`
    Relations   RelationState        `json:"relations"`

    Monthly     MonthlyState         `json:"monthly"`
    Stats       SectStats            `json:"stats"`
    Flags       map[string]bool      `json:"flags"`
}
```

### 6.2 SectMeta

```go
type SectMeta struct {
    SectID      SectID    `json:"sect_id"`
    OwnerUserID UserID    `json:"owner_user_id"`
    Name        string    `json:"name"`
    CreatedAt   time.Time `json:"created_at"`

    Level       int       `json:"level"`
    Reputation  int       `json:"reputation"` // 名望
    Order       int       `json:"order"`      // 宗门秩序，0-100
    WealthScore int       `json:"wealth_score"`

    SpiritVein  SpiritVeinState `json:"spirit_vein"`
}
```

### 6.3 SectRuntime

只用于运行时，不一定全部入库：

```go
type SectRuntime struct {
    Version              Version   `json:"version"`
    LastSimulatedAt      time.Time `json:"last_simulated_at"`
    LastSnapshotVersion  Version   `json:"last_snapshot_version"`
    LastSnapshotAt       time.Time `json:"last_snapshot_at"`
    Online               bool      `json:"online"`
    Dirty                bool      `json:"dirty"`
}
```

### 6.4 ResourceState

```go
type ResourceState struct {
    SpiritStone int64 `json:"spirit_stone"` // 灵石
    SpiritGrain int64 `json:"spirit_grain"` // 灵谷
    Herb        int64 `json:"herb"`         // 灵植
    Ore         int64 `json:"ore"`          // 矿材
    BeastMat    int64 `json:"beast_mat"`    // 妖材
    FormationMat int64 `json:"formation_mat"` // 阵材
}
```

注意：这里没有 `Aura` 或 `QiStock`。

### 6.5 InventoryState

丹药、法器、功法、典籍等非纯数量资源放入 Inventory。

```go
type InventoryState struct {
    Pills map[PillType]int64 `json:"pills"`
    Artifacts map[ItemID]ArtifactState `json:"artifacts"`
    Manuals map[ManualID]ManualState `json:"manuals"`
    Tools map[ItemID]ToolState `json:"tools"`
}

type PillType string

const (
    PillFasting PillType = "fasting_pill"
    PillHealing PillType = "healing_pill"
    PillCultivation PillType = "cultivation_pill"
    PillBreakthrough PillType = "breakthrough_pill"
    PillCalmMind PillType = "calm_mind_pill"
)
```

### 6.6 ContributionState

贡献点属于弟子账户，不属于宗门库存。

```go
type ContributionState struct {
    Accounts map[DiscipleID]ContributionAccount `json:"accounts"`
    TreasuryRules map[ExchangeItemID]ExchangeRule `json:"treasury_rules"`
    MonthlyPurchases map[DiscipleID]map[ExchangeItemID]int `json:"monthly_purchases"`
    RedeemabilityRatio float64 `json:"redeemability_ratio"`
    OutstandingContribution int64 `json:"outstanding_contribution"`
    TreasuryValue int64 `json:"treasury_value"`
}

type ContributionAccount struct {
    DiscipleID DiscipleID `json:"disciple_id"`
    Balance int64 `json:"balance"`
    EarnedTotal int64 `json:"earned_total"`
    SpentTotal int64 `json:"spent_total"`
}
```

### 6.7 SectStats

```go
type SectStats struct {
    TotalDisciples int `json:"total_disciples"`
    OuterDisciples int `json:"outer_disciples"`
    InnerDisciples int `json:"inner_disciples"`
    FoundationDisciples int `json:"foundation_disciples"`

    FoodPressure int `json:"food_pressure"`
    ResourcePressure int `json:"resource_pressure"`
    Security int `json:"security"`
    ProductionScore int `json:"production_score"`
    CombatScore int `json:"combat_score"`
    TeachingScore int `json:"teaching_score"`
}
```

---

## 7. 持久化设计

### 7.1 数据表：sect_snapshots

```sql
CREATE TABLE sect_snapshots (
    sect_id TEXT PRIMARY KEY,
    owner_player_id TEXT NOT NULL,
    schema_version INTEGER NOT NULL,
    simulation_version INTEGER NOT NULL,
    config_version INTEGER NOT NULL,
    state_version BIGINT NOT NULL,
    game_tick BIGINT NOT NULL,
    last_simulated_wall_ms BIGINT NOT NULL,
    state_blob BYTEA NOT NULL,
    state_size_bytes INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 7.2 数据表：sect_events

```sql
CREATE TABLE sect_events (
    sect_id TEXT NOT NULL,
    state_version BIGINT NOT NULL,
    event_seq INTEGER NOT NULL,
    cmd_id TEXT,
    event_type INTEGER NOT NULL,
    event_blob BYTEA NOT NULL,
    event_size_bytes INTEGER NOT NULL,
    game_tick BIGINT NOT NULL,
    acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (sect_id, state_version, event_seq)
);

CREATE INDEX idx_sect_events_sect_version
ON sect_events (sect_id, state_version);

CREATE INDEX idx_sect_events_cmd_id
ON sect_events (sect_id, cmd_id)
WHERE cmd_id IS NOT NULL;
```

### 7.3 数据表：command_log

```sql
CREATE TABLE command_log (
    sect_id TEXT NOT NULL,
    cmd_id TEXT NOT NULL,
    player_id TEXT NOT NULL,
    command_type INTEGER NOT NULL,
    command_blob BYTEA NOT NULL,
    command_size_bytes INTEGER NOT NULL,
    result_blob BYTEA,
    result_size_bytes INTEGER,
    result_status INTEGER NOT NULL,
    base_version BIGINT NOT NULL,
    state_version_after BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    PRIMARY KEY (sect_id, cmd_id)
);
```

用途：幂等、审计、排查玩家争议、回放。

### 7.4 数据表：query indexes

这些表都是从 `sect_snapshots.state_blob` 与 snapshot 后 bounded events 派生出来的查询投影。它们可以重建，不是权威状态。

```sql
CREATE TABLE sect_runtime_index (
    sect_id TEXT PRIMARY KEY REFERENCES sect_snapshots(sect_id) ON DELETE CASCADE,
    owner_player_id TEXT NOT NULL,
    sect_name TEXT NOT NULL,
    state_version BIGINT NOT NULL,
    game_tick BIGINT NOT NULL,
    active_storylet_count INTEGER NOT NULL,
    disciple_count INTEGER NOT NULL,
    building_count INTEGER NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sect_building_index (
    sect_id TEXT NOT NULL REFERENCES sect_snapshots(sect_id) ON DELETE CASCADE,
    building_id TEXT NOT NULL,
    building_type INTEGER NOT NULL,
    building_state INTEGER NOT NULL,
    level INTEGER NOT NULL,
    assigned_task_id TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (sect_id, building_id)
);

CREATE TABLE sect_disciple_index (
    sect_id TEXT NOT NULL REFERENCES sect_snapshots(sect_id) ON DELETE CASCADE,
    disciple_id TEXT NOT NULL,
    status INTEGER NOT NULL,
    current_task_id TEXT,
    current_building_id TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (sect_id, disciple_id)
);

CREATE TABLE sect_task_index (
    sect_id TEXT NOT NULL REFERENCES sect_snapshots(sect_id) ON DELETE CASCADE,
    task_id TEXT NOT NULL,
    task_type INTEGER NOT NULL,
    task_status INTEGER NOT NULL,
    assigned_disciple_id TEXT,
    target_building_id TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (sect_id, task_id)
);
```

用途：bounded query、运维排障、后台筛选、玩家列表页摘要。禁止把这些 index tables 当成权威恢复主源。

### 7.5 数据表：scene_leases

```sql
CREATE TABLE scene_leases (
    sect_id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    lease_until TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

单 owner 保护：spawn `SectActor` 前必须获取 `scene_leases`。lease 过期或 owner 节点失联后可抢占。

### 7.6 数据表：outbox

```sql
CREATE TABLE outbox (
    id BIGSERIAL PRIMARY KEY,
    topic TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    payload_blob BYTEA NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at TIMESTAMPTZ
);
```

用于跨服务事件、补偿、通知、排行榜更新等。

### 7.7 Snapshot 策略

保存 snapshot 的条件：

| 条件 | 说明 |
|---|---|
| Actor idle stop | 玩家离线一段时间后停止 actor 前 |
| 每 N 个事件 | 推荐 100-500 个事件 |
| 每 N 分钟 | 推荐 5-15 分钟 |
| 大结算后 | 月结、年结、秘境结算、大比结算 |
| 运维命令 | 手动保存 |

加载顺序：

```text
Load latest snapshot
  → Load events where version > snapshot.version
  → Apply events
  → If offline gap exists, run catch-up
```

---

## 8. 领域事件与状态应用

### 8.1 状态修改必须通过事件

这里采用的是**事件应用流水线**，不是“只存事件、完全靠事件重建状态”的纯 event sourcing。

冻结规则：

1. `DomainCommandHandler`、离线推进器、定时模拟器都只能返回 `[]DomainEvent`。
2. `ApplyEvent(state, event)` 是唯一允许修改 `SectState` 的玩法入口。
3. `ApplyEvent` 修改后的内存状态会被序列化进 `sect_snapshots.state_blob`；`sect_events` 只承担 bounded 增量、审计、debug、补偿追踪与 snapshot 后回放。
4. 正式权威加载路径是“读最新 snapshot -> 回放 snapshot 之后的 bounded events -> 执行 offline catch-up”，不是从零开始纯事件重放。

权威持久化主源仍然是 `sect_snapshots.state_blob`。`sect_events` 与 `command_log` 的职责是：

- 约束状态修改入口
- 审计 / debug / 回放
- 幂等与争议排查
- 弟子日记与离线补偿追踪

命令处理器只能生成事件：

```go
func HandleBuildBuilding(cmd BuildBuildingCommand, state *SectState) ([]DomainEvent, error)
```

事件应用器负责改状态：

```go
func ApplyEvent(state *SectState, event DomainEvent) error
```

不允许命令处理器直接修改 `SectState` 后再伪造事件。
不允许把 `ApplyEvent` 降级成“事件回放辅助函数”而在其他路径上直接改状态。

### 8.2 常用事件类型

```go
type EventType string

const (
    EventResourceChanged EventType = "resource_changed"
    EventBuildingQueued EventType = "building_queued"
    EventBuildingBuilt EventType = "building_built"
    EventBuildingUpgraded EventType = "building_upgraded"

    EventDiscipleRecruited EventType = "disciple_recruited"
    EventDiscipleStatusChanged EventType = "disciple_status_changed"
    EventDiscipleAssignedTask EventType = "disciple_assigned_task"
    EventDisciplePromoted EventType = "disciple_promoted"

    EventTaskCreated EventType = "task_created"
    EventTaskAccepted EventType = "task_accepted"
    EventTaskCompleted EventType = "task_completed"
    EventTaskFailed EventType = "task_failed"

    EventContributionEarned EventType = "contribution_earned"
    EventContributionSpent EventType = "contribution_spent"
    EventTreasuryExchanged EventType = "treasury_exchanged"

    EventCultivationAdvanced EventType = "cultivation_advanced"
    EventBreakthroughAttempted EventType = "breakthrough_attempted"
    EventBreakthroughSucceeded EventType = "breakthrough_succeeded"
    EventBreakthroughFailed EventType = "breakthrough_failed"

    EventProductionStarted EventType = "production_started"
    EventProductionCompleted EventType = "production_completed"

    EventPayrollPaid EventType = "payroll_paid"
    EventPayrollDelayed EventType = "payroll_delayed"

    EventSectOrderChanged EventType = "sect_order_changed"
    EventReputationChanged EventType = "reputation_changed"

    EventSectEventTriggered EventType = "sect_event_triggered"
    EventSectEventResolved EventType = "sect_event_resolved"

    EventDayAdvanced EventType = "day_advanced"
    EventMonthAdvanced EventType = "month_advanced"
    EventYearAdvanced EventType = "year_advanced"
)
```

### 8.3 Event Payload 示例

```go
type ResourceChangedPayload struct {
    Changes map[ResourceKind]int64 `json:"changes"`
    Reason string `json:"reason"`
}

type BuildingQueuedPayload struct {
    BuildingID BuildingID `json:"building_id"`
    BuildingType BuildingType `json:"building_type"`
    Level int `json:"level"`
    FinishAt GameTime `json:"finish_at"`
    Cost ResourceCost `json:"cost"`
}

type DiscipleAssignedTaskPayload struct {
    DiscipleID DiscipleID `json:"disciple_id"`
    TaskID TaskID `json:"task_id"`
    Forced bool `json:"forced"`
}
```


---

## 9. 命令总表

### 9.1 宗门基础命令

| CommandType | 作用 | Owner | MVP |
|---|---|---|---|
| `CreateSect` | 创建宗门 | Gateway/Admin | 是 |
| `JoinScene` | 加入宗门场景 | Gateway/SectActor | 是 |
| `LeaveScene` | 离开场景 | Gateway/SectActor | 是 |
| `RenameSect` | 改名 | SectActor | 是 |
| `SetPolicy` | 设置政策 | SectActor | 是 |
| `QuerySnapshot` | 拉取快照 | SectActor | 是 |

### 9.2 建筑命令

| CommandType | 作用 | MVP |
|---|---|---|
| `BuildBuilding` | 建造建筑 | 是 |
| `UpgradeBuilding` | 升级建筑 | 是 |
| `AssignBuildingManager` | 分配负责人 | 二期 |
| `AttachFormationToBuilding` | 阵法挂接 | 二期 |
| `RepairBuilding` | 维修建筑 | 二期 |

### 9.3 弟子命令

| CommandType | 作用 | MVP |
|---|---|---|
| `StartRecruitment` | 开启收徒 | 是 |
| `AcceptCandidate` | 接收候选弟子 | 是 |
| `RejectCandidate` | 拒绝候选弟子 | 是 |
| `SetDiscipleDevelopmentPlan` | 设置培养方向 | 是 |
| `PromoteDisciple` | 晋升弟子 | 是 |
| `PunishDisciple` | 处罚弟子 | 二期 |
| `ExpelDisciple` | 逐出弟子 | 二期 |

### 9.4 任务命令

| CommandType | 作用 | MVP |
|---|---|---|
| `PublishTask` | 发布任务 | 是 |
| `CancelTask` | 取消任务 | 是 |
| `AssignDiscipleTask` | 指派弟子 | 是 |
| `ClaimTaskCompletion` | 手动结算任务 | 可选 |
| `SetTaskPriority` | 设置任务优先级 | 是 |

### 9.5 生产命令

| CommandType | 作用 | MVP |
|---|---|---|
| `StartProduction` | 开始生产 | 是 |
| `CancelProduction` | 取消生产 | 是 |
| `CollectProduction` | 收取产物 | 是 |
| `SetProductionQueue` | 设置生产队列 | 二期 |

### 9.6 修炼命令

| CommandType | 作用 | MVP |
|---|---|---|
| `StartCultivation` | 安排修炼 | 是 |
| `UsePillForCultivation` | 使用丹药修炼 | 是 |
| `ReserveCave` | 预约洞府 | 是 |
| `AttemptBreakthrough` | 突破 | 是 |

### 9.7 贡献与宝库命令

| CommandType | 作用 | MVP |
|---|---|---|
| `SetExchangeRule` | 设置兑换规则 | 是 |
| `ExchangeContributionItem` | 弟子兑换资源 | 是 |
| `RestockTreasury` | 调整宝库库存 | 是 |
| `SetMonthlyLimit` | 设置限购 | 是 |

### 9.8 事件与多人命令

| CommandType | 作用 | MVP |
|---|---|---|
| `ChooseEventOption` | 选择事件处理方式 | 是 |
| `DismissEvent` | 忽略事件 | 是 |
| `VisitSect` | 访问他人宗门 | 二期 |
| `RequestTrade` | 请求交易 | 二期 |
| `ChallengeSect` | 挑战宗门 | 三期 |
| `JoinExpedition` | 加入秘境/联合任务 | 三期 |

---

## 10. 资源与库存系统

### 10.1 玩法目的

资源系统支撑所有宗门经营：

- 灵石：货币、修炼燃料、阵法供能。
- 灵谷：弟子生活。
- 灵植：炼丹和任务材料。
- 矿材：建筑、炼器、阵基。
- 妖材：战斗产物，连接战斗与生产。
- 阵材：阵法维护与阵盘。
- 丹药：一次性成长与恢复资源。
- 法器：长期装备与任务效率资源。
- 贡献点：弟子对宗门资源的索取权。

### 10.2 数据结构

```go
type ResourceKind string

const (
    ResSpiritStone ResourceKind = "spirit_stone"
    ResSpiritGrain ResourceKind = "spirit_grain"
    ResHerb ResourceKind = "herb"
    ResOre ResourceKind = "ore"
    ResBeastMat ResourceKind = "beast_mat"
    ResFormationMat ResourceKind = "formation_mat"
)

type ResourceCost map[ResourceKind]int64
type ResourceReward map[ResourceKind]int64
```

### 10.3 资源操作 API

所有资源变更必须走 domain helper：

```go
func CanAfford(stock ResourceState, cost ResourceCost) bool
func ApplyCost(stock *ResourceState, cost ResourceCost) error
func ApplyReward(stock *ResourceState, reward ResourceReward)
```

### 10.4 资源命令归属

资源本身不对客户端开放直接加减命令。资源只能由其他系统引发事件变化，例如：建筑建造、任务完成、生产完成、贡献兑换、月例、事件奖励。

### 10.5 不变量

1. 任何资源数量不得小于 0。
2. 贡献点不是 `ResourceState` 的字段。
3. 灵气不是资源字段。
4. 资源扣减必须在命令校验阶段检查。
5. 资源变化必须产生 `ResourceChanged` 事件。
6. 玩家不能通过客户端直接请求 `AddResource`，除非 Admin/GM 命令。

---

## 11. 灵气、灵脉与修炼环境系统

### 11.1 玩法目的

灵气不作为库存，而作为修炼环境系数。它影响弟子修炼、闭关、突破、阵法挂接建筑的效率。

### 11.2 数据结构

```go
type SpiritVeinLevel string

const (
    VeinNone SpiritVeinLevel = "none"
    VeinWeak SpiritVeinLevel = "weak"
    VeinSmall SpiritVeinLevel = "small"
    VeinMedium SpiritVeinLevel = "medium"
    VeinLarge SpiritVeinLevel = "large"
)

type SpiritVeinState struct {
    Level SpiritVeinLevel `json:"level"`
    BaseCultivationMultiplier float64 `json:"base_cultivation_multiplier"`
    Discovered bool `json:"discovered"`
}
```

### 11.3 环境系数

```go
func GetBaseEnvironmentMultiplier(vein SpiritVeinLevel) float64 {
    switch vein {
    case VeinNone: return 0.7
    case VeinWeak: return 1.0
    case VeinSmall: return 1.2
    case VeinMedium: return 1.5
    case VeinLarge: return 2.0
    default: return 1.0
    }
}
```

### 11.4 阵法挂接

聚灵阵等阵法通过 `FormationState` 挂接到建筑。

```go
type FormationState struct {
    FormationID FormationID `json:"formation_id"`
    Type FormationType `json:"type"`
    Level int `json:"level"`
    Durability int `json:"durability"` // 0-100
    AttachedBuildings []BuildingID `json:"attached_buildings"`
    DailySpiritStoneCost int64 `json:"daily_spirit_stone_cost"`
    MonthlyFormationMatCost int64 `json:"monthly_formation_mat_cost"`
    Maintainer DiscipleID `json:"maintainer,omitempty"`
    Active bool `json:"active"`
}
```

### 11.5 命令

| Command | 说明 | 阶段 |
|---|---|---|
| `BuildFormation` | 建造阵法 | 二期 |
| `AttachFormationToBuilding` | 挂接建筑 | 二期 |
| `DetachFormationFromBuilding` | 取消挂接 | 二期 |
| `AssignFormationMaintainer` | 分配阵师 | 二期 |
| `RepairFormation` | 修复阵盘 | 二期 |

Actor 归属：阵法状态属于 `SectActor` 内部 `Formations`，不创建 `FormationActor`。

---

## 12. 弟子系统

### 12.1 玩法目的

弟子是宗门社会的核心。他们消耗资源、完成任务、修炼成长、参与事件、影响秩序和名望。

### 12.2 DiscipleState

```go
type DiscipleState struct {
    DiscipleID DiscipleID `json:"disciple_id"`
    Name string `json:"name"`
    Gender string `json:"gender"`
    Age int `json:"age"`

    Identity IdentityRank `json:"identity"`
    Realm RealmState `json:"realm"`

    Attributes BaseAttributes `json:"attributes"`
    Skills SkillSet `json:"skills"`
    Personality PersonalitySet `json:"personality"`

    Status DiscipleStatus `json:"status"`
    Needs DiscipleNeeds `json:"needs"`
    Loyalty int `json:"loyalty"`             // 0-100
    Satisfaction int `json:"satisfaction"`   // 0-100

    Contribution int64 `json:"contribution"` // 冗余展示字段；权威账户在 ContributionState
    CurrentTaskID TaskID `json:"current_task_id,omitempty"`
    CurrentPlan DevelopmentPlan `json:"current_plan"`

    Equipment EquipmentState `json:"equipment"`
    Relations DiscipleRelations `json:"relations"`

    Flags map[string]bool `json:"flags"`
    CreatedAtGameDay int64 `json:"created_at_game_day"`
}
```

### 12.3 基础属性

```go
type BaseAttributes struct {
    SpiritRoot int `json:"spirit_root"`       // 灵根
    Comprehension int `json:"comprehension"` // 悟性
    Physique int `json:"physique"`           // 根骨
    DivineSense int `json:"divine_sense"`    // 神识
    Mind int `json:"mind"`                   // 心性
    Luck int `json:"luck"`                   // 气运
}
```

取值范围：1-20，普通人 5-8，优秀 9-12，天才 13+。

### 12.4 百艺

```go
type SkillKind string

const (
    SkillHerb SkillKind = "herb"
    SkillAlchemy SkillKind = "alchemy"
    SkillCrafting SkillKind = "crafting"
    SkillFormation SkillKind = "formation"
    SkillBeastTaming SkillKind = "beast_taming"
    SkillMedicine SkillKind = "medicine"
    SkillExternalAffairs SkillKind = "external_affairs"
    SkillCombat SkillKind = "combat"
)

type SkillSet map[SkillKind]SkillProgress

type SkillProgress struct {
    Level int `json:"level"` // 0-10
    Exp int64 `json:"exp"`
}
```

### 12.5 性格

```go
type PersonalityTrait string

const (
    TraitDiligent PersonalityTrait = "diligent"
    TraitLazy PersonalityTrait = "lazy"
    TraitAggressive PersonalityTrait = "aggressive"
    TraitCautious PersonalityTrait = "cautious"
    TraitGreedy PersonalityTrait = "greedy"
    TraitDaoSeeker PersonalityTrait = "dao_seeker"
    TraitEmotional PersonalityTrait = "emotional"
    TraitLoner PersonalityTrait = "loner"
    TraitRighteous PersonalityTrait = "righteous"
    TraitOpportunistic PersonalityTrait = "opportunistic"
)

type PersonalitySet struct {
    Traits []PersonalityTrait `json:"traits"`
}
```

### 12.6 弟子状态

```go
type DiscipleStatus struct {
    Health int `json:"health"` // 0-100
    Fatigue int `json:"fatigue"` // 0-100
    Pressure int `json:"pressure"` // 0-100
    InjuryLevel int `json:"injury_level"` // 0-5
    PillToxin int `json:"pill_toxin"` // 丹毒 0-100
    InnerDemon int `json:"inner_demon"` // 心魔 0-100
    IsInClosedCultivation bool `json:"is_in_closed_cultivation"`
    IsAway bool `json:"is_away"`
    IsDead bool `json:"is_dead"`
}
```

### 12.7 弟子命令

| Command | 校验 | 事件 |
|---|---|---|
| `AcceptCandidate` | 有居舍容量，候选存在 | `DiscipleRecruited` |
| `SetDiscipleDevelopmentPlan` | 弟子存在 | `DisciplePlanChanged` |
| `AssignDiscipleTask` | 弟子空闲，任务可接 | `DiscipleAssignedTask` |
| `PromoteDisciple` | 满足身份条件 | `DisciplePromoted` |
| `PunishDisciple` | 执法堂解锁 | `DisciplePunished` |
| `ExpelDisciple` | 非关键锁定角色 | `DiscipleExpelled` |

### 12.8 Actor 归属

所有弟子数据属于 `SectActor` 内部状态。第一版不实现 `DiscipleActor`。

### 12.9 每日弟子结算顺序

```text
1. 检查死亡、重伤、闭关、外出状态
2. 消耗灵谷或辟谷丹
3. 执行强制任务或当前任务
4. 若空闲，按 AI 决策选择行为
5. 根据行为增加修炼点、技能经验、任务进度
6. 更新疲劳、压力、满意、忠诚
7. 检查突破候选、冲突、心魔、事件触发
```

---

## 13. 弟子来源与招生系统

### 13.1 玩法目的

弟子不自动刷新。每个弟子来源必须有世界观逻辑和资源成本：开山收徒、拜山投靠、任务带回、外部推荐。

### 13.2 AdmissionState

```go
type AdmissionState struct {
    CurrentRecruitment *RecruitmentSession `json:"current_recruitment,omitempty"`
    Candidates map[CandidateID]CandidateState `json:"candidates"`
    LastAnnualRecruitmentYear int `json:"last_annual_recruitment_year"`
}

type RecruitmentSession struct {
    RecruitmentID string `json:"recruitment_id"`
    Type RecruitmentType `json:"type"`
    StartedAt GameTime `json:"started_at"`
    EndsAt GameTime `json:"ends_at"`
    InvestmentSpiritStone int64 `json:"investment_spirit_stone"`
    CandidateCount int `json:"candidate_count"`
}

type CandidateState struct {
    CandidateID CandidateID `json:"candidate_id"`
    Name string `json:"name"`
    Source CandidateSource `json:"source"`
    Attributes BaseAttributes `json:"attributes"`
    Personality PersonalitySet `json:"personality"`
    InitialIdentity IdentityRank `json:"initial_identity"`
    BackgroundTags []string `json:"background_tags"`
    RiskTags []string `json:"risk_tags"`
    ExpiresAt GameTime `json:"expires_at"`
}
```

### 13.3 StartRecruitment 命令

```go
type StartRecruitmentPayload struct {
    Type RecruitmentType `json:"type"`
    InvestmentSpiritStone int64 `json:"investment_spirit_stone"`
}
```

处理：

```text
validate 问灵台/山门是否满足条件
validate 灵石足够
deduct 灵石
create RecruitmentSession
generate CandidateState list
events:
  ResourceChanged
  RecruitmentStarted
  CandidatesGenerated
```

### 13.4 候选人数

```text
候选人数 = 基础人数 + 名望加成 + 山门等级加成 + 招生投入加成 + 地区人口加成
```

MVP：

```go
base = 6
reputationBonus = reputation / 100
gateBonus = gateLevel * 2
investmentBonus = min(investmentSpiritStone / 100, 5)
```

### 13.5 候选质量

```text
候选质量 = 地区潜力 + 宗门名望 + 问灵台等级 + 随机气运
```

实现时生成属性：

```go
qualityScore := regionPotential + reputation/50 + examHallLevel*5 + rng.Intn(30)
```

### 13.6 拜山投靠

拜山不是稳定来源，而是事件系统的一类事件。玩家选择接纳后产生：

```text
DiscipleRecruited
ReputationChanged
PossibleHiddenRiskFlagAdded
```

---

## 14. 身份、权限与晋升系统

### 14.1 身份枚举

```go
type IdentityRank string

const (
    IdentityServant IdentityRank = "servant" // 杂役
    IdentityOuter IdentityRank = "outer" // 外门
    IdentityInner IdentityRank = "inner" // 内门
    IdentityTrue IdentityRank = "true" // 真传
    IdentitySteward IdentityRank = "steward" // 执事
    IdentityElderCandidate IdentityRank = "elder_candidate"
    IdentityElder IdentityRank = "elder"
)
```

### 14.2 身份配置

```go
type IdentityConfig struct {
    Rank IdentityRank
    MonthlySpiritStone int64
    FoodConsumptionPerDay float64
    AllowedTaskGrades []TaskGrade
    AllowedBuildings []BuildingType
    AllowedExchangeTiers []int
    MonthlyObligationDays int
}
```

初始配置：

| 身份 | 月例 | 每月义务 | 可接任务 |
|---|---:|---:|---|
| 杂役 | 5 | 8 天 | 丁 |
| 外门 | 20 | 6 天 | 丁/丙 |
| 内门 | 80 | 4 天 | 丙/乙 |
| 真传 | 300 | 2 次高阶 | 乙/甲 |
| 执事 | 120 + 津贴 | 管理任务 | 管理类 |
| 长老 | 500+ | 传道/护法 | 战略类 |

### 14.3 晋升条件结构

```go
type PromotionRequirement struct {
    From IdentityRank `json:"from"`
    To IdentityRank `json:"to"`
    MinRealm RealmStage `json:"min_realm"`
    MinContribution int64 `json:"min_contribution"`
    MinLoyalty int `json:"min_loyalty"`
    MinSatisfaction int `json:"min_satisfaction"`
    RequiredTaskCompletions int `json:"required_task_completions"`
    RequiredExamPassed bool `json:"required_exam_passed"`
    RequiredBuilding BuildingType `json:"required_building,omitempty"`
}
```

### 14.4 PromoteDisciple 命令

```go
type PromoteDisciplePayload struct {
    DiscipleID DiscipleID `json:"disciple_id"`
    TargetRank IdentityRank `json:"target_rank"`
}
```

校验：弟子存在、未死亡/未外出、晋升路径合法、境界/贡献/考核/建筑条件满足。

事件：`ContributionSpent optional`、`DisciplePromoted`、`SatisfactionChanged`、`ReputationChanged optional`。

### 14.5 晋升不公

若玩家强行晋升低贡献、低修为弟子，或长期压制高贡献弟子：

```text
SectOrder -x
其他相关弟子 Satisfaction -x
可能触发 InternalConflictEvent
```

MVP 可先通过简单公式在月结时计算。

---

## 15. 修炼与突破系统

### 15.1 RealmState

```go
type RealmStage string

const (
    RealmMortal RealmStage = "mortal"
    RealmQiEntry RealmStage = "qi_entry"
    RealmQiEarly RealmStage = "qi_early"
    RealmQiMiddle RealmStage = "qi_middle"
    RealmQiLate RealmStage = "qi_late"
    RealmFoundation RealmStage = "foundation"
    RealmGoldenCore RealmStage = "golden_core"
)

type RealmState struct {
    Stage RealmStage `json:"stage"`
    CultivationPoints int64 `json:"cultivation_points"`
    RequiredPoints int64 `json:"required_points"`
    ReadyForBreakthrough bool `json:"ready_for_breakthrough"`
    FailedBreakthroughCount int `json:"failed_breakthrough_count"`
}
```

### 15.2 修炼公式

```text
修炼点 = 基础修炼 × 时间投入 × 灵根系数 × 功法匹配 × 修炼环境 × 心境系数 × 资源加成
```

```go
type CultivationContext struct {
    Disciple DiscipleState
    Sect SectState
    Building *BuildingState
    UsedPills []PillType
    UsedSpiritStone int64
    TimeRatio float64
}

func CalculateDailyCultivationPoints(ctx CultivationContext) int64
```

### 15.3 修炼环境来源

| 来源 | 实现 |
|---|---|
| 灵脉 | `SectMeta.SpiritVein` |
| 洞府 | `BuildingTypeCave` |
| 聚灵阵 | `FormationState` 挂接洞府/传功阁 |
| 传功阁 | 新弟子基础加成 |
| 藏经阁 | 功法匹配和技术上限 |
| 丹药 | `Inventory.Pills` 消耗 |

### 15.4 修炼命令

| Command | 说明 |
|---|---|
| `StartCultivation` | 设置弟子修炼行为 |
| `UsePillForCultivation` | 为弟子使用修炼丹 |
| `ReserveCave` | 租用洞府 |
| `AttemptBreakthrough` | 突破 |

### 15.5 突破配置

```go
type BreakthroughConfig struct {
    Stage RealmStage
    NextStage RealmStage
    RequiredPoints int64
    BaseSuccessRate float64
    MinCaveLevel int
    RecommendedPill PillType
}
```

| 突破 | 基础成功率 |
|---|---:|
| 引气 | 90% |
| 练气小阶段 | 75% |
| 筑基 | 45% |
| 金丹 | 25% |

### 15.6 AttemptBreakthrough 命令

```go
type AttemptBreakthroughPayload struct {
    DiscipleID DiscipleID `json:"disciple_id"`
    UsePills map[PillType]int64 `json:"use_pills"`
    UseSpiritStone int64 `json:"use_spirit_stone"`
    CaveBuildingID BuildingID `json:"cave_building_id,omitempty"`
    ProtectorDiscipleID DiscipleID `json:"protector_disciple_id,omitempty"`
}
```

校验：弟子存在、修炼点足够、伤病不过重、压力不过高、资源足够、洞府可用、丹药库存足够。

事件：`ResourceChanged`、`InventoryChanged`、`BreakthroughAttempted`、`BreakthroughSucceeded/Failed`、`DiscipleStatusChanged`、高阶时可 `SectEventTriggered`。

### 15.7 失败结果

| 失败程度 | 状态变化 |
|---|---|
| 轻微 | 修炼点损失 10%-20% |
| 普通 | 受伤 + 压力 |
| 严重 | 心魔 / 丹毒 / 境界进度大损失 |
| 极端 | 高阶才可能死亡 |

---

## 16. 百艺成长系统

### 16.1 玩法目的

弟子不仅通过境界成长，也通过工作获得百艺经验，从而支撑宗门路线分化。

### 16.2 百艺经验公式

```text
百艺经验 = 工作时长 × 工作难度 × 悟性系数 × 兴趣系数 × 师承加成
```

```go
type SkillExpGain struct {
    DiscipleID DiscipleID
    Skill SkillKind
    Exp int64
    Reason string
}
```

事件：`SkillExpGained`、`SkillLevelUp`。

### 16.3 百艺对应玩法

| 百艺 | 主要系统 |
|---|---|
| 灵植 | 灵田产量、采药、病害 |
| 炼丹 | 丹药成功率、品质、炸炉 |
| 炼器 | 法器品质、耐久、失败损耗 |
| 阵法 | 维护成本、事故率、覆盖槽 |
| 御兽 | 灵兽成长、灵兽任务 |
| 医术 | 疗伤、解毒、死亡率 |
| 外务 | 交易、谈判、外交 |
| 战斗 | 除妖、护送、秘境、宗门战 |

### 16.4 Actor 归属

百艺成长由 `SectActor` 的任务、生产、修炼结算触发，不创建 `SkillActor`。


---

## 17. 任务系统

### 17.1 玩法目的

任务堂是宗门劳动力调度中心。任务系统把资源缺口、建筑需求、事件危机、弟子成长、贡献点连接起来。

### 17.2 TaskState

```go
type TaskType string

const (
    TaskInternal TaskType = "internal"
    TaskProduction TaskType = "production"
    TaskExternal TaskType = "external"
    TaskExplore TaskType = "explore"
    TaskCombat TaskType = "combat"
    TaskSpecialized TaskType = "specialized"
)

type TaskGrade string

const (
    GradeDing TaskGrade = "ding"
    GradeBing TaskGrade = "bing"
    GradeYi TaskGrade = "yi"
    GradeJia TaskGrade = "jia"
    GradeSpecial TaskGrade = "special"
)

type TaskState struct {
    TaskID TaskID `json:"task_id"`
    Type TaskType `json:"type"`
    Grade TaskGrade `json:"grade"`
    Title string `json:"title"`
    Description string `json:"description"`

    Status TaskStatus `json:"status"`
    Priority int `json:"priority"`

    RequiredSkills map[SkillKind]int `json:"required_skills"`
    MinIdentity IdentityRank `json:"min_identity"`
    MinRealm RealmStage `json:"min_realm"`

    AssignedDisciples []DiscipleID `json:"assigned_disciples"`
    Progress int `json:"progress"` // 0-100
    DurationDays int `json:"duration_days"`
    RemainingDays int `json:"remaining_days"`

    Risk int `json:"risk"` // 0-100
    ContributionReward int64 `json:"contribution_reward"`
    SpiritStoneReward int64 `json:"spirit_stone_reward"`
    ResourceRewards ResourceReward `json:"resource_rewards"`

    Source TaskSource `json:"source"`
    CreatedAt GameTime `json:"created_at"`
    ExpiresAt *GameTime `json:"expires_at,omitempty"`
}
```

```go
type TaskStatus string

const (
    TaskOpen TaskStatus = "open"
    TaskAssigned TaskStatus = "assigned"
    TaskInProgress TaskStatus = "in_progress"
    TaskCompleted TaskStatus = "completed"
    TaskFailed TaskStatus = "failed"
    TaskCancelled TaskStatus = "cancelled"
)
```

### 17.3 任务生成

任务来源：宗门资源缺口、建筑需求、外部委托、地图资源点、随机事件、玩家手动发布。

任务优先级：

```text
任务优先级 = 宗门缺口权重 + 事件紧急度 + 建筑需求 + 战略目标
```

例：

```text
炼丹殿缺灵植 → 自动提高采药任务出现率
阵法阁缺阵材 → 自动提高采矿、采购、秘境任务权重
膳堂灵谷不足 → 自动发布灵田协助和采购任务
山门安全下降 → 自动发布巡山和除妖任务
```

### 17.4 PublishTask 命令

```go
type PublishTaskPayload struct {
    Type TaskType `json:"type"`
    Grade TaskGrade `json:"grade"`
    Title string `json:"title"`
    Priority int `json:"priority"`
    ContributionReward int64 `json:"contribution_reward"`
    SpiritStoneReward int64 `json:"spirit_stone_reward"`
    ResourceRewards ResourceReward `json:"resource_rewards"`
    RequiredSkills map[SkillKind]int `json:"required_skills"`
    DurationDays int `json:"duration_days"`
    Risk int `json:"risk"`
}
```

校验：任务堂存在、奖励不能为负、奖励预算不能导致贡献通胀过高、任务等级与宗门等级匹配。

事件：`TaskCreated`。

### 17.5 AssignDiscipleTask 命令

校验：任务存在且可接、弟子存在、身份满足、未重伤/闭关/外出、队伍人数不超上限。若强制指派，计算满意/忠诚惩罚。

事件：`TaskAccepted`、`DiscipleAssignedTask`、`DiscipleStatusChanged optional`。

### 17.6 每日任务推进

```go
func AdvanceTasksOneDay(state *SectState, rng RNG) []DomainEvent
```

步骤：

```text
for each task in progress:
  calculate team capability
  calculate success/progress
  reduce remaining days
  apply fatigue / injury risk
  if completed:
      grant rewards
      grant contribution
      grant skill exp
      evaluate task
  if failed:
      apply injury/loss/order penalty
```

### 17.7 任务成功率

```text
成功率 = 基础成功率 + 能力适配 + 队伍配合 + 法器加成 - 难度 - 风险
```

```go
func CalculateTaskSuccessRate(task TaskState, disciples []DiscipleState, sect SectState) float64
```

### 17.8 任务评价

```go
type TaskEvaluation string

const (
    EvalExcellent TaskEvaluation = "excellent"
    EvalGood TaskEvaluation = "good"
    EvalNormal TaskEvaluation = "normal"
    EvalPoor TaskEvaluation = "poor"
    EvalBad TaskEvaluation = "bad"
    EvalFailed TaskEvaluation = "failed"
)
```

评价因素：完成度、时间效率、资源损耗、伤亡情况、委托方满意度。

---

## 18. 贡献兑换与功勋宝库系统

### 18.1 玩法目的

功勋宝库负责资源分配。贡献点体现弟子对宗门资源的索取权，防止玩家无限压榨弟子。

### 18.2 ExchangeRule

```go
type ExchangeItemID string

type ExchangeRule struct {
    ExchangeItemID ExchangeItemID `json:"exchange_item_id"`
    Name string `json:"name"`
    ItemKind ExchangeItemKind `json:"item_kind"`
    ItemRef string `json:"item_ref"`
    ContributionCost int64 `json:"contribution_cost"`
    RequiredIdentity IdentityRank `json:"required_identity"`
    MonthlyLimit int `json:"monthly_limit"`
    StockLimit int64 `json:"stock_limit"` // -1 means by real inventory
    Enabled bool `json:"enabled"`
}
```

### 18.3 初始兑换规则

| 物品/权限 | 价格 | 身份 |
|---|---:|---|
| 辟谷丹 | 5 | 杂役 |
| 疗伤丹 | 15 | 杂役 |
| 修炼丹 | 30 | 外门 |
| 低阶法器 | 120 | 外门 |
| 功法借阅 7 日 | 60 | 外门 |
| 洞府使用 1 日 | 5 | 内门 |
| 闭关名额 | 80 | 内门 |
| 晋升申请 | 100 | 外门 |
| 大比报名 | 20 | 外门 |
| 秘境名额 | 200 | 真传 |

### 18.4 ExchangeContributionItem 命令

```go
type ExchangeContributionItemPayload struct {
    DiscipleID DiscipleID `json:"disciple_id"`
    ExchangeItemID ExchangeItemID `json:"exchange_item_id"`
    Quantity int64 `json:"quantity"`
}
```

校验：功勋宝库存在、弟子存在、身份满足、贡献余额足够、库存足够、未超过月限购、兑换规则启用。

事件：`ContributionSpent`、`TreasuryExchanged`、`InventoryChanged/ResourceChanged/PermissionGranted`、`SatisfactionChanged optional`。

### 18.5 贡献兑付率

```text
贡献兑付率 = 功勋宝库可兑换资源总价值 / 全体弟子未消费贡献点总额
```

```go
func RecalculateRedeemabilityRatio(state *SectState) float64
```

| 兑付率 | 效果 |
|---|---|
| >= 120% | 满意度缓慢上升 |
| 80%-120% | 正常 |
| 50%-80% | 满意下降、任务积极性下降 |
| <50% | 黑市、叛逃、抗议事件权重增加 |

### 18.6 Actor 归属

贡献和宝库属于 `SectActor` 内部。后期若做跨宗门交易，交易由 `MarketActor` 处理，但贡献账户仍归本宗门 `SectActor`。

---

## 19. 俸禄、义务与月结系统

### 19.1 玩法目的

月例形成宗门固定支出，义务任务形成弟子对宗门的最低劳动责任。二者共同防止玩家只堆修炼或只压榨弟子。

### 19.2 PayrollState

```go
type PayrollState struct {
    LastPaidMonth int64 `json:"last_paid_month"`
    Arrears map[DiscipleID]int `json:"arrears"` // 欠薪月份
}

type MonthlyObligationState struct {
    MonthIndex int64 `json:"month_index"`
    CompletedDays map[DiscipleID]int `json:"completed_days"`
    RequiredDays map[DiscipleID]int `json:"required_days"`
    Violations map[DiscipleID]int `json:"violations"`
}

type MonthlyState struct {
    Payroll PayrollState `json:"payroll"`
    Obligations MonthlyObligationState `json:"obligations"`
}
```

### 19.3 月结流程

```text
1. 计算每个弟子月例
2. 检查灵石是否足够
3. 足够则支付；不足则按身份优先级或玩家政策处理欠薪
4. 检查义务任务完成情况
5. 根据欠薪、义务、公平、兑付率更新满意与忠诚
6. 重置月限购
7. 生成月报
```

### 19.4 事件

```text
PayrollPaid
PayrollDelayed
MonthlyObligationChecked
DiscipleSatisfactionChanged
DiscipleLoyaltyChanged
MonthAdvanced
```

### 19.5 欠薪后果

| 欠薪 | 后果 |
|---|---|
| 1 月 | 满意下降 |
| 2 月 | 任务积极性下降 |
| 3 月 | 流失、叛逃、执法压力 |
| 长期 | 名望下降、招生质量下降 |

---

## 20. 弟子 AI 行为系统

### 20.1 玩法目的

玩家不逐个控制弟子，而是通过政策、任务、建筑、培养方向影响弟子行为。弟子每天自动决策。

### 20.2 DevelopmentPlan

```go
type DevelopmentPlanType string

const (
    PlanBalanced DevelopmentPlanType = "balanced"
    PlanCultivation DevelopmentPlanType = "cultivation"
    PlanProduction DevelopmentPlanType = "production"
    PlanCombat DevelopmentPlanType = "combat"
    PlanAlchemy DevelopmentPlanType = "alchemy"
    PlanCrafting DevelopmentPlanType = "crafting"
    PlanFormation DevelopmentPlanType = "formation"
    PlanExternalAffairs DevelopmentPlanType = "external_affairs"
)

type DevelopmentPlan struct {
    Type DevelopmentPlanType `json:"type"`
    PreferredSkills []SkillKind `json:"preferred_skills"`
    AvoidHighRisk bool `json:"avoid_high_risk"`
    ForcedByPlayer bool `json:"forced_by_player"`
}
```

### 20.3 行为优先级

```text
1. 生存危机：重伤、饥饿、走火、战斗
2. 宗门强制：义务任务、执事指派、处罚
3. 个人目标：修炼、突破、赚贡献、兑换资源
4. 建筑工作：按技能适配进行生产
5. 社交与恢复：休息、同门互动、降低压力
```

### 20.4 任务吸引力公式

```text
任务吸引力 =
贡献奖励 × 贡献需求权重
+ 灵石奖励 × 贪财权重
+ 技能经验 × 成长权重
+ 宗门优先级 × 忠诚权重
- 风险 × 谨慎权重
- 时间成本
- 受伤概率
- 与自身性格冲突
```

### 20.5 AI 结算函数

```go
func DecideDailyAction(d DiscipleState, state SectState, rng RNG) DiscipleAction

type DiscipleAction struct {
    Type DiscipleActionType
    TaskID TaskID
    BuildingID BuildingID
    Reason string
}
```

MVP 可简化：当前任务存在则执行；否则如果义务缺口存在，则选择最高优先级低风险任务；否则如果培养方向为修炼，则修炼；否则选择与技能匹配的生产/任务。

### 20.6 强制指派惩罚

当玩家强制指派不适合任务：满意 -1 到 -10，忠诚 -0 到 -5，压力 +5 到 +20，任务失败率上升。严重时触发 `DiscipleRefusedTask` 或 `InternalConflictEvent`。MVP 可以先不做拒绝，只做惩罚。

---

## 21. 建筑系统总设计

### 21.1 玩法目的

建筑是系统解锁、资源生产、弟子成长和事件承载的实体。每个建筑必须至少满足：产生资源、转化资源、消耗资源并提供成长、管理弟子、解锁系统、承载重大事件。

### 21.2 BuildingState

```go
type BuildingType string

type BuildingState struct {
    BuildingID BuildingID `json:"building_id"`
    Type BuildingType `json:"type"`
    Level int `json:"level"`
    Status BuildingStatus `json:"status"`

    BuiltAt GameTime `json:"built_at"`
    UpgradeFinishAt *GameTime `json:"upgrade_finish_at,omitempty"`

    Durability int `json:"durability"` // 0-100
    Manager DiscipleID `json:"manager,omitempty"`
    AssignedWorkers []DiscipleID `json:"assigned_workers"`

    Slots BuildingSlots `json:"slots"`
    AttachedFormations []FormationID `json:"attached_formations"`

    Config map[string]any `json:"config"`
}

type BuildingStatus string

const (
    BuildingPlanned BuildingStatus = "planned"
    BuildingConstructing BuildingStatus = "constructing"
    BuildingActive BuildingStatus = "active"
    BuildingUpgrading BuildingStatus = "upgrading"
    BuildingDamaged BuildingStatus = "damaged"
    BuildingDisabled BuildingStatus = "disabled"
)
```

### 21.3 BuildingConfig

配置数据不要硬编码在 handler 中，应放到 config：

```go
type BuildingConfig struct {
    Type BuildingType
    Name string
    MaxLevel int
    UnlockSectLevel int
    BuildCostByLevel map[int]ResourceCost
    BuildDaysByLevel map[int]int
    MaintenanceCostByLevel map[int]ResourceCost
    EffectsByLevel map[int]BuildingEffect
}
```

### 21.4 BuildBuilding 命令

```go
type BuildBuildingPayload struct {
    BuildingType BuildingType `json:"building_type"`
}
```

校验：建筑类型存在、宗门等级满足、前置建筑满足、资源足够、未超过数量限制。

事件：`ResourceChanged`、`BuildingQueued` 或 `BuildingBuilt`。

MVP：核心基础建筑可即时完成；二期：高级建筑走建造队列。

### 21.5 UpgradeBuilding 命令

```go
type UpgradeBuildingPayload struct {
    BuildingID BuildingID `json:"building_id"`
}
```

校验：建筑存在、未在升级、等级未满、资源足够、宗门等级满足。

事件：`ResourceChanged`、`BuildingUpgradeStarted`、`BuildingUpgraded`。

### 21.6 每日建筑维护

每日结算：扣维护资源；若灵石不足，建筑效率下降或耐久下降；若耐久过低，建筑进入 damaged。

事件：`BuildingMaintained`、`BuildingDamaged`、`ResourceChanged`。

---

## 22. 核心建筑具体设计

所有建筑均归 `SectActor` 管理，默认不创建单独 Actor。

### 22.1 山门 Gate

```go
const BuildingGate BuildingType = "gate"
```

功能：控制访客、招生入口、名望展示、安全第一道防线。

```go
type GateConfigState struct {
    OpenToVisitors bool `json:"open_to_visitors"`
    AllowWanderingCultivators bool `json:"allow_wandering_cultivators"`
    GuardDiscipleIDs []DiscipleID `json:"guard_disciple_ids"`
}
```

命令：`SetGatePolicy`、`AssignGateGuard`。事件：`GatePolicyChanged`、`GateGuardAssigned`、`VisitCandidateEventTriggered`、`SecurityChanged`。

### 22.2 宗门大殿 MainHall

```go
const BuildingMainHall BuildingType = "main_hall"
```

功能：宗门等级、政策槽、主线目标、对外宣告。

```go
type MainHallState struct {
    PolicySlots int `json:"policy_slots"`
    ActiveEdicts []SectPolicyID `json:"active_edicts"`
}
```

命令：`UpgradeSectLevel`、`SetPolicy`、`DeclareSectGoal`。

### 22.3 问灵台 / 考核殿 ExamHall

```go
const BuildingExamHall BuildingType = "exam_hall"
```

功能：检测灵根、入门考核、晋升审核、年度考核。

```go
type ExamHallState struct {
    ExaminerIDs []DiscipleID `json:"examiner_ids"`
    AnnualExamScheduled bool `json:"annual_exam_scheduled"`
}
```

命令：`StartRecruitment`、`ScheduleAnnualExam`、`RunPromotionExam`。

### 22.4 任务堂 TaskHall

```go
const BuildingTaskHall BuildingType = "task_hall"
```

功能：发布任务、接取任务、任务评价、贡献发放、义务追踪。

```go
type TaskHallState struct {
    MaxOpenTasks int `json:"max_open_tasks"`
    TaskRefreshWeight map[TaskType]int `json:"task_refresh_weight"`
}
```

命令：`PublishTask`、`CancelTask`、`SetTaskPriority`、`AssignDiscipleTask`。

### 22.5 功勋宝库 Treasury

```go
const BuildingTreasury BuildingType = "treasury"
```

功能：存储资源、贡献兑换、限购、兑付率计算。

```go
type TreasuryBuildingState struct {
    PublicExchangeEnabled bool `json:"public_exchange_enabled"`
    StewardID DiscipleID `json:"steward_id,omitempty"`
}
```

命令：`SetExchangeRule`、`ExchangeContributionItem`、`SetMonthlyLimit`、`RestockTreasury`。

### 22.6 执法堂 LawHall

```go
const BuildingLawHall BuildingType = "law_hall"
```

功能：私斗处理、处罚违规、追回偷盗、维持秩序。

```go
type LawHallState struct {
    EnforcerIDs []DiscipleID `json:"enforcer_ids"`
    Strictness int `json:"strictness"` // 0-100
}
```

命令：`AssignEnforcer`、`PunishDisciple`、`ResolveDispute`、`SetLawStrictness`。二期实现。

### 22.7 弟子居舍 Dormitory

```go
const BuildingDormitory BuildingType = "dormitory"
```

功能：人口上限、恢复、满意度。

```go
type DormitoryState struct {
    Capacity int `json:"capacity"`
    Comfort int `json:"comfort"`
}
```

效果：`capacity = level * 10`；comfort 影响疲劳恢复和满意。

### 22.8 膳堂 / 灵谷仓 Canteen

```go
const BuildingCanteen BuildingType = "canteen"
```

功能：灵谷储备、制作灵食、降疲劳、提满意。MVP 只做食物消耗和满意影响。二期加入 `ProduceSpiritMeal`。

### 22.9 传功阁 TeachingHall

```go
const BuildingTeachingHall BuildingType = "teaching_hall"
```

功能：基础功法、早课、新弟子成长速度。

```go
type TeachingHallState struct {
    TeacherIDs []DiscipleID `json:"teacher_ids"`
    ActiveCurriculum []ManualID `json:"active_curriculum"`
}
```

日结算影响：新弟子修炼加成、技能经验加成。

### 22.10 藏经阁 Library

```go
const BuildingLibrary BuildingType = "library"
```

功能：功法、丹方、器谱、阵图、技术解锁、贡献借阅。

```go
type LibraryState struct {
    Manuals map[ManualID]ManualState `json:"manuals"`
    BorrowRecords map[DiscipleID][]BorrowRecord `json:"borrow_records"`
}
```

二期实现。MVP 可仅作为修炼加成建筑。

### 22.11 闭关洞府 Cave

```go
const BuildingCave BuildingType = "cave"
```

功能：高效修炼、突破、洞府席位、消耗灵石和丹药。

```go
type CaveState struct {
    Slots []CaveSlot `json:"slots"`
}

type CaveSlot struct {
    SlotID string `json:"slot_id"`
    OccupiedBy DiscipleID `json:"occupied_by,omitempty"`
    ReservedUntil *GameTime `json:"reserved_until,omitempty"`
    EnvironmentMultiplier float64 `json:"environment_multiplier"`
}
```

命令：`ReserveCave`、`StartClosedCultivation`、`EndClosedCultivation`、`AttemptBreakthrough`。

### 22.12 演武场 / 大比场 Arena

```go
const BuildingArena BuildingType = "arena"
```

功能：战斗训练、切磋、大比、排名。

```go
type ArenaState struct {
    CurrentCompetitionID string `json:"current_competition_id,omitempty"`
    TrainingSchedule []DiscipleID `json:"training_schedule"`
}
```

二期实现。

### 22.13 灵田 / 灵植园 Farm

```go
const BuildingFarm BuildingType = "farm"
```

功能：产出灵谷、产出灵植、培养灵植百艺。

```go
type FarmState struct {
    Plots int `json:"plots"`
    CropMode FarmCropMode `json:"crop_mode"` // grain, herb, mixed
    DaysUntilHarvest int `json:"days_until_harvest"`
    AssignedWorkers []DiscipleID `json:"assigned_workers"`
}
```

命令：`SetFarmCropMode`、`AssignFarmWorker`、`CollectFarmHarvest`。

### 22.14 炼丹殿 / 药庐 AlchemyHall

```go
const BuildingAlchemyHall BuildingType = "alchemy_hall"
```

功能：炼丹、疗伤、控丹毒、生产突破丹。

```go
type AlchemyHallState struct {
    AlchemistIDs []DiscipleID `json:"alchemist_ids"`
    Queue []ProductionID `json:"queue"`
    MedicalBeds int `json:"medical_beds"`
}
```

命令：`StartProduction(recipe=alchemy)`、`AssignAlchemist`、`TreatDisciple`、`DetoxDisciple`。

### 22.15 炼器坊 CraftingHall

```go
const BuildingCraftingHall BuildingType = "crafting_hall"
```

功能：制作法器、修理法器、生产工具。二期实现。

### 22.16 阵法阁 / 阵枢 FormationHall

```go
const BuildingFormationHall BuildingType = "formation_hall"
```

功能：研究阵法、制作阵盘、维护阵法、挂接建筑。二期实现。

### 22.17 灵兽苑 BeastHall

```go
const BuildingBeastHall BuildingType = "beast_hall"
```

功能：培养灵兽、巡山、战斗辅助、妖材副产物。三期实现。

### 22.18 坊市 / 外务堂 MarketOffice

```go
const BuildingMarketOffice BuildingType = "market_office"
```

功能：买卖资源、外部委托、外交、拍卖。

```go
type MarketOfficeState struct {
    EnvoyIDs []DiscipleID `json:"envoy_ids"`
    UnlockedMarkets []string `json:"unlocked_markets"`
}
```

二期实现，需与 `MarketActor` 交互。

### 22.19 秘境台 / 传送阵 ExpeditionPlatform

```go
const BuildingExpeditionPlatform BuildingType = "expedition_platform"
```

功能：秘境探索、远行加速、高风险事件。三期实现，需与 `ExpeditionActor` 交互。

### 22.20 祖师祠 AncestralHall

```go
const BuildingAncestralHall BuildingType = "ancestral_hall"
```

功能：宗门道统、士气恢复、传承事件、大事件祭告。二期/三期实现。


---

## 23. 生产系统

### 23.1 ProductionJob

```go
type ProductionKind string

const (
    ProductionFarm ProductionKind = "farm"
    ProductionAlchemy ProductionKind = "alchemy"
    ProductionCrafting ProductionKind = "crafting"
    ProductionFormation ProductionKind = "formation"
)

type ProductionJob struct {
    ProductionID ProductionID `json:"production_id"`
    Kind ProductionKind `json:"kind"`
    BuildingID BuildingID `json:"building_id"`
    RecipeID RecipeID `json:"recipe_id"`
    Status ProductionStatus `json:"status"`

    AssignedDisciples []DiscipleID `json:"assigned_disciples"`
    InputCost ResourceCost `json:"input_cost"`
    OutputReward ResourceReward `json:"output_reward"`
    OutputItems []ItemReward `json:"output_items"`

    Progress int `json:"progress"`
    RequiredWork int `json:"required_work"`
    StartedAt GameTime `json:"started_at"`
    FinishAt *GameTime `json:"finish_at,omitempty"`
}
```

### 23.2 RecipeConfig

```go
type RecipeConfig struct {
    RecipeID RecipeID
    Kind ProductionKind
    Name string
    RequiredBuilding BuildingType
    RequiredBuildingLevel int
    RequiredSkill SkillKind
    RequiredSkillLevel int
    Input ResourceCost
    Output ResourceReward
    OutputItems []ItemReward
    BaseSuccessRate float64
    WorkRequired int
}
```

### 23.3 炼丹配方 MVP

| 丹药 | 消耗 | 产出 |
|---|---|---|
| 辟谷丹 | 2 灵植 + 1 灵石 | 2 |
| 疗伤丹 | 3 灵植 + 1 妖材 + 2 灵石 | 1 |
| 修炼丹 | 5 灵植 + 2 灵石 | 1 |
| 破境丹 | 20 灵植 + 5 妖材 + 20 灵石 | 1 |

### 23.4 炼器配方 二期

| 法器 | 消耗 |
|---|---|
| 下品飞剑 | 10 矿材 + 2 妖材 + 5 灵石 |
| 法袍 | 6 矿材 + 4 妖材 + 5 灵石 |
| 灵植工具 | 8 矿材 + 3 灵石 |
| 阵盘 | 10 矿材 + 5 阵材 + 10 灵石 |

### 23.5 StartProduction 命令

```go
type StartProductionPayload struct {
    BuildingID BuildingID `json:"building_id"`
    RecipeID RecipeID `json:"recipe_id"`
    AssignedDisciples []DiscipleID `json:"assigned_disciples"`
    Quantity int `json:"quantity"`
}
```

校验：建筑存在且 active、配方存在、建筑等级满足、弟子技能满足或允许低概率尝试、输入资源足够、生产队列未满。

事件：`ResourceChanged`、`ProductionStarted`、`DiscipleAssignedProduction`。

### 23.6 生产完成

每日推进：根据工作量、技能、建筑等级、阵法加成增加 progress；progress 达标时结算成功率；成功产出资源/物品；失败损失部分材料、可能压力/事故。

事件：`ProductionCompleted`、`ProductionFailed`、`InventoryChanged`、`ResourceChanged`、`SkillExpGained`。

---

## 24. 灵田生产系统

### 24.1 FarmState 细化

```go
type FarmCropMode string

const (
    FarmGrain FarmCropMode = "grain"
    FarmHerb FarmCropMode = "herb"
    FarmMixed FarmCropMode = "mixed"
)
```

### 24.2 产量公式

```text
产量 = 土地等级 × 劳动力质量 × 灵植技能 × 护植阵加成 × 季节系数
```

```go
func CalculateFarmYield(farm BuildingState, workers []DiscipleState, state SectState) ResourceReward
```

MVP 一级灵田：周期 10 天；灵谷 60；灵植 8；所需劳动力 4 人日；护植阵加成 +20%；灵植师 3 级 +15%。

### 24.3 每日流程

```text
1. 检查是否有 Farm 建筑
2. 根据分配弟子增加工作进度
3. 到周期则产生收成
4. 增加灵植百艺经验
5. 若无人维护，可能降低产量或触发病害事件
```

---

## 25. 阵法系统

### 25.1 FormationType

```go
type FormationType string

const (
    FormationGatherQi FormationType = "gather_qi"
    FormationProtectPlant FormationType = "protect_plant"
    FormationDefense FormationType = "defense"
    FormationFireCondense FormationType = "fire_condense"
    FormationTeleport FormationType = "teleport"
)
```

### 25.2 阵法消耗

| 阵法 | 灵石/日 | 阵材/月 |
|---|---:|---:|
| 小聚灵阵 | 2 | 1 |
| 护植阵 | 1 | 1 |
| 守御阵 | 3 | 2 |
| 凝火阵 | 2 | 1 |
| 传送阵 | 10/次 | 3 |

### 25.3 维护公式

```text
实际消耗 = 基础消耗 × (1 - 阵法等级 × 3%)
最低不低于 60%
```

### 25.4 每日维护流程

```text
1. 遍历 active formations
2. 扣除灵石
3. 若灵石不足，active=false 或效率下降
4. 每月扣阵材
5. 阵师技能降低消耗与事故率
6. 耐久过低触发事故
```

二期实现。

---

## 26. 事件系统

### 26.1 玩法目的

事件系统让宗门被世界推动，而不是纯数值循环。

事件类型：

| 类型 | 示例 |
|---|---|
| 机缘 | 灵脉显现、古卷出土、天才拜山 |
| 危机 | 妖兽袭击、邪修渗透、灵田病害 |
| 外交 | 友宗来访、商会拍卖、附属家族求援 |
| 内部 | 私斗、偷盗、师徒冲突、心魔 |
| 修炼 | 突破异象、走火入魔、天劫 |
| 经济 | 灵石涨价、材料短缺、坊市繁荣 |
| 剧情 | 古宗遗址、秘境开启、仇敌寻来 |

### 26.2 SectEventState

```go
type SectEventState struct {
    ActiveEvents map[EventID]SectEvent `json:"active_events"`
    ResolvedEvents []ResolvedEventSummary `json:"resolved_events"`
    Tension int `json:"tension"` // 张弛曲线
    LastMajorEventDay int64 `json:"last_major_event_day"`
}

type SectEvent struct {
    EventID EventID `json:"event_id"`
    Type SectEventType `json:"type"`
    Severity int `json:"severity"`
    Title string `json:"title"`
    Description string `json:"description"`
    Options []EventOption `json:"options"`
    ExpiresAt *GameTime `json:"expires_at,omitempty"`
    Tags []string `json:"tags"`
}

type EventOption struct {
    OptionID string `json:"option_id"`
    Text string `json:"text"`
    Requirements EventOptionRequirements `json:"requirements"`
    Preview string `json:"preview"`
}
```

### 26.3 事件强度

```text
事件强度 = 名望 + 财富 + 高阶弟子数量 + 敌对关系 + 时间推进
```

```go
func CalculateEventIntensity(state SectState) int
```

### 26.4 张弛曲线

```go
type EventTensionStage string

const (
    TensionDevelop EventTensionStage = "develop"
    TensionSmallEvent EventTensionStage = "small_event"
    TensionReward EventTensionStage = "reward"
    TensionCrisis EventTensionStage = "crisis"
    TensionRecovery EventTensionStage = "recovery"
    TensionMajor EventTensionStage = "major"
)
```

日结算时根据 tension、event intensity、随机种子生成事件候选。

### 26.5 ChooseEventOption 命令

```go
type ChooseEventOptionPayload struct {
    EventID EventID `json:"event_id"`
    OptionID string `json:"option_id"`
}
```

校验：事件存在、未过期、选项要求满足。

事件：`SectEventResolved`、`ResourceChanged optional`、`DiscipleStatusChanged optional`、`ReputationChanged optional`、`TaskCreated optional`。

---

## 27. 大比与考核系统

### 27.1 玩法目的

大比与考核提供身份跃迁、战力展示、宗门名望、弟子剧情冲突。

### 27.2 CompetitionState

二期实现：

```go
type CompetitionState struct {
    CompetitionID string `json:"competition_id"`
    Type CompetitionType `json:"type"`
    Year int `json:"year"`
    Status CompetitionStatus `json:"status"`
    Participants []DiscipleID `json:"participants"`
    Rounds []CompetitionRound `json:"rounds"`
    Results []CompetitionResult `json:"results"`
}
```

### 27.3 年度大比流程

```text
1. 年末或玩家手动安排
2. 报名
3. 初赛按战力筛选
4. 复赛考虑术法、法器、战斗技能
5. 决赛生成排名
6. 奖励贡献、丹药、法器、晋升资格、名望
```

### 27.4 考核流程

| 考核 | 评估 |
|---|---|
| 入门考核 | 灵根、心性、悟性 |
| 外门考核 | 修为、义务、贡献 |
| 内门考核 | 修为、战力或百艺 |
| 真传考核 | 筑基、特殊贡献、师承 |
| 执事考核 | 管理、忠诚、秩序贡献 |

### 27.5 Actor 归属

宗门内部年度大比可在 `SectActor` 内部结算。跨宗门大比后期使用 `CompetitionActor`。

---

## 28. 政策系统

### 28.1 玩法目的

玩家不逐个控制所有弟子，而是通过政策影响宗门倾向。

### 28.2 PolicyState

```go
type PolicyState struct {
    TaskPolicy TaskPolicy `json:"task_policy"`
    ResourcePolicy ResourcePolicy `json:"resource_policy"`
    RecruitmentPolicy RecruitmentPolicy `json:"recruitment_policy"`
    CustomFlags map[string]bool `json:"custom_flags"`
}

type TaskPolicy string

const (
    TaskPolicyStable TaskPolicy = "stable"
    TaskPolicyRewardExternal TaskPolicy = "reward_external"
    TaskPolicyProduction TaskPolicy = "production"
    TaskPolicyCombat TaskPolicy = "combat"
    TaskPolicyClosedCultivation TaskPolicy = "closed_cultivation"
)

type ResourcePolicy string

const (
    ResourcePolicySaving ResourcePolicy = "saving"
    ResourcePolicyGenerous ResourcePolicy = "generous"
    ResourcePolicyPillLimited ResourcePolicy = "pill_limited"
    ResourcePolicyOpenExchange ResourcePolicy = "open_exchange"
    ResourcePolicyWarPreparation ResourcePolicy = "war_preparation"
)

type RecruitmentPolicy string

const (
    RecruitmentBroad RecruitmentPolicy = "broad"
    RecruitmentSelective RecruitmentPolicy = "selective"
    RecruitmentWandering RecruitmentPolicy = "wandering"
    RecruitmentAffiliated RecruitmentPolicy = "affiliated"
)
```

### 28.3 SetPolicy 命令

```go
type SetPolicyPayload struct {
    PolicyCategory string `json:"policy_category"`
    PolicyValue string `json:"policy_value"`
}
```

事件：`PolicyChanged`。

### 28.4 政策效果

| 政策 | 实现 |
|---|---|
| 稳健经营 | 降低高风险任务生成和 AI 接取 |
| 重赏外务 | 外务任务奖励提高，灵石压力提高 |
| 强化生产 | 生产任务优先级提高 |
| 战斗历练 | 战斗任务生成率提高，伤病风险提高 |
| 闭门修炼 | 修炼时间提高，资源收入下降 |
| 节流 | 月例降低，满意下降 |
| 厚养弟子 | 月例提高，忠诚上升 |
| 丹药限购 | 月限购更严格 |
| 开放兑换 | 满意上升，库存压力上升 |
| 战备优先 | 法器优先给战斗弟子 |
| 广收门徒 | 候选更多，平均质量低 |
| 宁缺毋滥 | 候选更少，平均质量高 |

---

## 29. 外务、坊市与市场系统

### 29.1 玩法目的

外务系统负责灵石收入、交易、外部关系、委托、拍卖，是宗门与外部世界的接口。

### 29.2 MVP 简化

MVP 不实现全局 `MarketActor`，可以把外务作为任务类型：`TaskTypeExternal`。任务完成后产出灵石、名望、关系、稀有材料。

### 29.3 二期 MarketActor

```go
type MarketState struct {
    Markets map[string]MarketRegionState `json:"markets"`
    PriceTable map[ResourceKind]MarketPrice `json:"price_table"`
    Orders map[string]MarketOrder `json:"orders"`
}

type MarketPrice struct {
    Resource ResourceKind `json:"resource"`
    BuyPrice int64 `json:"buy_price"`
    SellPrice int64 `json:"sell_price"`
    Volatility int `json:"volatility"`
}
```

交互：

```text
SectActor → MarketActor: RequestQuote
MarketActor → SectActor: QuoteResult
SectActor → MarketActor: ExecuteTrade
MarketActor → SectActor: TradeCommitted
SectActor: Apply ResourceChanged
```

注意：交易最终资源变更必须在 `SectActor` 内应用，不能由 MarketActor 直接写宗门状态。

---

## 30. 秘境与远征系统

### 30.1 玩法目的

秘境是高风险高回报玩法，连接高阶弟子、法器、阵法、战斗、机缘事件。

### 30.2 MVP 简化

MVP 不实现独立秘境，仅通过探索/战斗任务表现。

### 30.3 三期 ExpeditionActor

```go
type ExpeditionParticipant struct {
    SectID SectID `json:"sect_id"`
    Team []DiscipleSnapshot `json:"team"`
    Contribution ResourceCost `json:"contribution"`
}

type DiscipleSnapshot struct {
    DiscipleID DiscipleID
    Realm RealmStage
    Combat int
    Skills map[SkillKind]int
    EquipmentScore int
}
```

流程：

```text
1. SectActor 发起 JoinExpedition
2. ExpeditionActor 创建实例
3. 多个宗门报名
4. 到期后模拟探索
5. 生成每个宗门的结算
6. 发送 ExpeditionResult 到各 SectActor
7. 各 SectActor 应用资源、伤病、名望变化
```

---

## 31. 多人访问与跨宗门交互

### 31.1 访问他人宗门

访客只能获取公开视图：

```go
type PublicSectSnapshot struct {
    SectID SectID
    Name string
    Level int
    Reputation int
    PublicBuildings []PublicBuildingInfo
    PublicDisciples []PublicDiscipleInfo
    VisitorActions []VisitorAction
}
```

不能暴露：全量库存、弟子完整属性、防御阵法细节、贡献账户、隐藏事件、高价值物品列表。

### 31.2 VisitSect 命令

```text
Gateway → SessionRegistryActor → Target SectActor JoinScene(view_mode=visitor)
```

Target SectActor 返回 public snapshot。

### 31.3 访客可提交的命令

| Command | 说明 | 阶段 |
|---|---|---|
| `LeaveMessage` | 留言 | 二期 |
| `RequestTrade` | 发起交易请求 | 二期 |
| `RequestChallenge` | 挑战 | 三期 |
| `AssistTask` | 协助公开任务 | 三期 |

所有命令由目标 `SectActor` 校验。

---

## 32. 战斗、法器与装备简化模型

### 32.1 MVP 战斗不做实时战斗

MVP 战斗只作为任务结算公式：

```text
战斗任务成功率 = 队伍战斗力 + 法器加成 + 境界加成 - 任务难度 - 风险
```

```go
func CalculateDiscipleCombatScore(d DiscipleState) int {
    return RealmScore(d.Realm.Stage) +
           d.Skills[SkillCombat].Level*10 +
           d.Attributes.Physique*2 +
           d.Attributes.DivineSense +
           EquipmentCombatScore(d.Equipment)
}
```

### 32.2 伤病

风险任务失败或低评价时：`InjuryLevel +1~3`、`Health -10~50`、`Pressure +10~30`，高风险高阶任务才可能死亡。

事件：`DiscipleInjured`、`DiscipleDied`、`TaskFailed`、`SectOrderChanged`。

### 32.3 ArtifactState

```go
type ArtifactType string

const (
    ArtifactSword ArtifactType = "sword"
    ArtifactRobe ArtifactType = "robe"
    ArtifactFarmTool ArtifactType = "farm_tool"
    ArtifactAlchemyFurnace ArtifactType = "alchemy_furnace"
    ArtifactFormationDisk ArtifactType = "formation_disk"
    ArtifactBeastBell ArtifactType = "beast_bell"
)

type ArtifactState struct {
    ItemID ItemID `json:"item_id"`
    Type ArtifactType `json:"type"`
    Quality int `json:"quality"` // 1-5
    Durability int `json:"durability"` // 0-100
    BoundDiscipleID DiscipleID `json:"bound_disciple_id,omitempty"`
    Stats map[string]int `json:"stats"`
}

type EquipmentState struct {
    Weapon ItemID `json:"weapon,omitempty"`
    Robe ItemID `json:"robe,omitempty"`
    Tool ItemID `json:"tool,omitempty"`
    Special ItemID `json:"special,omitempty"`
}
```

命令：`EquipArtifact`、`UnequipArtifact`、`RepairArtifact`、`CraftArtifact`。二期实现。MVP 可只使用“低阶法器数量”作为库存资源。

---

## 33. 灵兽系统

三期实现。

```go
type BeastState struct {
    BeastID string `json:"beast_id"`
    Species string `json:"species"`
    Level int `json:"level"`
    Loyalty int `json:"loyalty"`
    Health int `json:"health"`
    AssignedDisciple DiscipleID `json:"assigned_disciple,omitempty"`
    Traits []string `json:"traits"`
}
```

功能：巡山、战斗辅助、妖材副产物、御兽百艺成长。Actor 归属：`SectActor` 内部。

---

## 34. 离线追赶结算

### 34.1 目的

玩家离线后，不常驻 `SectActor`。下次上线时根据真实时间计算应推进的游戏天数。

### 34.2 算法

```go
func CalculateOfflineDays(lastSimulatedAt time.Time, now time.Time, config TimeConfig) int {
    elapsedSeconds := now.Sub(lastSimulatedAt).Seconds()
    days := int(elapsedSeconds) / config.RealtimeSecondsPerGameDay
    return min(days, config.MaxOfflineCatchupDays)
}
```

### 34.3 追赶流程

```text
1. 加载 snapshot + events
2. 计算 offlineDays
3. for day in offlineDays:
      AdvanceOneDay(reason=offline)
      if day reaches month/year:
          RunMonthly/Yearly
      if step count too high:
          split into batches
4. 生成 OfflineCatchupSummary
5. 写入事件
6. 推送给玩家
```

### 34.4 性能保护

1. 单次最多结算 30 天。
2. 超出部分使用压缩收益结算。
3. 复杂事件不在离线中无限触发，最多触发有限数量。
4. 离线期间不进行需要玩家选择的高风险事件，只生成待处理事件。

---

## 35. 每日/月度/年度模拟顺序

### 35.1 AdvanceOneDay

```text
1. Advance GameTime
2. ConsumeFood
3. MaintainBuildings
4. MaintainFormations
5. ProcessDiscipleActions
6. AdvanceTasks
7. AdvanceProductions
8. AdvanceCultivation
9. ProcessInjuriesAndMedical
10. UpdateSatisfactionAndLoyaltyLight
11. GenerateMinorEvents
12. RecalculateSectStats
13. EmitDayAdvanced
```

### 35.2 AdvanceMonth

```text
1. PayPayroll
2. CheckObligations
3. ResetMonthlyPurchaseLimits
4. RecalculateContributionRedeemability
5. UpdateSatisfactionAndLoyalty
6. CheckPromotionCandidates
7. GenerateMonthlyReport
8. EmitMonthAdvanced
```

### 35.3 AdvanceYear

```text
1. RunAnnualRecruitmentIfEnabled
2. ScheduleOrRunAnnualCompetition
3. RunAnnualExam
4. UpdateExternalRelations
5. IncreaseEventIntensity
6. EmitYearAdvanced
```

### 35.4 模拟函数要求

所有模拟函数必须是可测试的纯函数或近似纯函数：

```go
func AdvanceOneDay(state *SectState, rng RNG) ([]DomainEvent, error)
```

禁止在模拟函数中直接访问数据库、网络、WebSocket。

---

## 36. 安全、权限与反作弊

### 36.1 鉴权

Gateway 必须验证：JWT/session token、user_id、sect ownership、command permission。

### 36.2 权限矩阵

| 命令 | Owner | Visitor | Admin |
|---|---:|---:|---:|
| 查看私有快照 | 是 | 否 | 是 |
| 查看公开快照 | 是 | 是 | 是 |
| 建造/升级 | 是 | 否 | 是 |
| 指派任务 | 是 | 否 | 是 |
| 贡献兑换 | 是或模拟弟子 | 否 | 是 |
| 访问 | 是 | 是 | 是 |
| GM 补偿 | 否 | 否 | 是 |

### 36.3 幂等

每个命令以 `cmd_id + user_id + sect_id` 唯一。如果重复收到相同 `cmd_id`：已成功则返回之前的 `CommandResult`；处理中则返回 pending 或等待；失败则根据失败类型决定是否允许重试。

### 36.4 反作弊

1. 前端提交的数值全部忽略，只使用 ID 和选择。
2. 资源、奖励、成功率全部由服务端配置和状态计算。
3. 关键命令写 command_log。
4. 异常资源变化可由 event_log 审计。
5. Admin/Gm 命令必须单独权限和审计。

---

## 37. Go 代码包结构建议

```text
cmd/server/
  main.go

internal/gateway/
  websocket.go
  http.go
  auth.go

internal/actors/
  registry_actor.go
  sect_actor.go
  persistence_actor.go
  broadcaster_actor.go
  world_clock_actor.go
  event_scheduler_actor.go
  market_actor.go

internal/protocol/
  command.go
  result.go
  patch.go
  push.go

internal/domain/
  sect/
    state.go
    events.go
    apply.go
    commands.go
  resources/
    resources.go
  disciples/
    disciple.go
    ai.go
    cultivation.go
  buildings/
    building.go
    config.go
  tasks/
    task.go
    advance.go
  contribution/
    treasury.go
  production/
    production.go
    recipes.go
  events/
    sect_events.go
  simulation/
    daily.go
    monthly.go
    yearly.go
    offline.go

internal/persistence/
  postgres.go
  snapshots.go
  events.go
  commands.go
  leases.go

internal/config/
  building_config.yaml
  recipe_config.yaml
  identity_config.yaml
  task_config.yaml
  event_config.yaml

internal/testutil/
  fixtures.go
  rng.go
```

---

## 38. 测试要求

### 38.1 纯函数测试

必须覆盖：资源扣减不能为负、修炼点计算、突破概率计算、任务成功率计算、贡献兑付率计算、月例计算、离线天数计算。

### 38.2 Command Handler 测试

每个命令必须测试：合法输入产生正确事件、资源不足返回错误、身份不足返回错误、重复 cmd_id 幂等、事件应用后状态正确。

### 38.3 Event Replay 测试

测试从空状态 apply events、从 snapshot + events 恢复、恢复后状态等于原状态。

### 38.4 Actor 测试

测试 `SectActor load`、`JoinScene returns snapshot`、`SubmitCommand returns result`、`Tick does not panic`、`Idle stop saves snapshot`。

### 38.5 Golden Simulation 测试

为开局宗门准备固定随机种子：模拟 30 天，验证资源、弟子状态、任务结果、事件数量。该测试用于防止智能体改代码时破坏数值闭环。

---

## 39. MVP 实现范围

### 39.1 V1 Product Core Scope

当前主线不再用“一口气做完整 MVP”来表述，而是拆成分阶段落地。

### Phase A：Authority Core Bootstrap

| 系统 | 要求 |
|---|---|
| 协议 | Protobuf `ClientCommand` / `CommandResult` / `StatePatch` |
| 状态 | `SectState`、`DomainEvent`、`ApplyEvent`、版本边界 |
| Actor 基础 | `SessionRegistryActor`、`SectActor`、`PersistenceActor`、`ClientPushActor` |
| 持久化 | `state_blob + event_log + command_log + indexes` |
| 资源与建筑 | `ResourceState`、`BuildBuilding`、`UpgradeBuilding` 的第一条闭环 |
| 低频推进 | online tick、offline catch-up 的最小可运行路径 |

### Phase B：Core Sect Loops

| 系统 | 要求 |
|---|---|
| 弟子 | 属性、身份、状态、贡献账户 |
| 招生 | 开山收徒、候选、接纳 |
| 任务 | 发布、指派、推进、完成、贡献奖励 |
| 生产 | 灵田、炼丹 |
| 修炼 | 修炼点、洞府、丹药、突破 |
| 贡献 | 兑换规则、限购、兑付率 |
| 月结 | 月例、义务、满意、忠诚 |
| 事件 | 简单机缘/危机/内部事件 |
| 离线 | 最多 30 天追赶结算 |

### Phase C：Thin Client Vertical Slice And Hardening

| 系统 | 要求 |
|---|---|
| 薄客户端 | 客户端只消费 authority snapshot / patch，不拥有 gameplay truth |
| 会话同步 | 正式 protobuf 会话同步、可安全重连 |
| 验证 | deterministic replay、server/client mainline replay、restore continuity |
| 平台外壳 | 微信 / 抖音壳层有界冒烟与加固 |
| 可观测性 | bounded telemetry、drift/reject/runbook |

### 39.2 V2+ 扩展实现

| 系统 | 要求 |
|---|---|
| 炼器 | 法器、装备、修理 |
| 阵法 | 阵法阁、聚灵阵、护植阵、守御阵 |
| 执法堂 | 私斗、偷盗、处罚 |
| 外务堂/坊市 | MarketActor 简版 |
| 演武场/大比 | 年度大比、考核 |
| 弟子性格 AI | 更完整的行为倾向 |
| 藏经阁 | 功法/典籍/借阅 |

### 39.3 长线扩展实现

| 系统 | 要求 |
|---|---|
| 灵兽 | 灵兽苑、御兽 |
| 秘境 | ExpeditionActor |
| 外交 | 关系、附属势力 |
| 长老 | 长老职责、收徒、护法 |
| 天劫 | 高阶突破事件 |
| 宗门战 | WarActor |
| 跨宗门协作 | 访问、交易、挑战 |

---

## 40. 初始配置数据

### 40.1 开局资源

```json
{
  "spirit_stone": 1000,
  "spirit_grain": 300,
  "herb": 50,
  "ore": 80,
  "beast_mat": 10,
  "formation_mat": 10
}
```

库存：

```json
{
  "pills": {
    "cultivation_pill": 5
  },
  "low_tier_artifact_count": 2
}
```

### 40.2 开局弟子

| 角色 | 身份 | 特长 |
|---|---|---|
| 大师兄/大师姐 | 外门精英 | 战斗或管理 |
| 灵植弟子 | 杂役/外门 | 灵植 |
| 丹道弟子 | 外门 | 炼丹潜力 |
| 器道弟子 | 外门 | 炼器潜力 |
| 普通弟子 | 杂役 | 综合成长 |

### 40.3 开局宗门状态

```json
{
  "level": 1,
  "reputation": 20,
  "order": 70,
  "spirit_vein": "weak",
  "contribution_redeemability_ratio": 1.0
}
```

### 40.4 开局建筑

MVP 开局推荐：山门 1、宗门大殿 1、弟子居舍 1、膳堂 1、任务堂 1、灵田 1、传功阁 1。

30-90 天解锁：炼丹殿、功勋宝库、问灵台、洞府。

---

## 41. 实现智能体任务拆分建议

### 41.1 Task 1：基础协议与状态

交付：`protocol.ClientCommand`、`protocol.CommandResult`、`protocol.StatePatch`（protobuf）、`domain.sect.SectState`、`domain.sect.DomainEvent`、`domain.sect.ApplyEvent`。

验收：能创建初始 SectState；能 apply ResourceChanged；能生成 patch。

### 41.2 Task 2：持久化

交付：Postgres `state_blob + event_log + command_log + indexes` schema、`LoadSnapshot`、`SaveSnapshot`、`AppendEvents`、`LoadEventsAfterVersion`、`CommandLog`。

验收：snapshot + events 能恢复状态；duplicate cmd_id 能返回旧结果。

### 41.3 Task 3：Hollywood Actor 基础

交付：`SessionRegistryActor`、`SectActor`、`PersistenceActor`、`WorldClockActor`、`ClientPushActor` stub。

验收：`JoinScene` 可加载宗门；`SubmitCommand` 可路由到 `SectActor`；当前玩家连接可收到 snapshot / patch。

### 41.4 Task 4：资源 + 建筑

交付：Resource helpers、`BuildBuilding`、`UpgradeBuilding`、Building configs。

验收：资源不足不能建造；建造成功扣资源并新增建筑。

### 41.5 Task 5：弟子 + 招生

交付：`DiscipleState`、`StartRecruitment`、`AcceptCandidate`、Contribution account creation。

验收：收徒生成候选；接纳后增加弟子并占用居舍容量。

### 41.6 Task 6：任务系统

交付：`TaskState`、`PublishTask`、`AssignDiscipleTask`、`AdvanceTasksOneDay`。

验收：任务可发布、指派、每日推进、完成后奖励资源和贡献。

### 41.7 Task 7：生产 + 炼丹 + 灵田

交付：`ProductionJob`、Farm daily yield、Alchemy recipes、`StartProduction`、`CollectProduction`。

验收：灵田 10 日产出；炼丹消耗灵植灵石并产丹药。

### 41.8 Task 8：修炼 + 突破

交付：Cultivation calculation、`StartCultivation`、`UsePillForCultivation`、`AttemptBreakthrough`。

验收：每日增加修炼点；修炼点足够后可突破；失败会产生伤病/压力。

### 41.9 Task 9：贡献兑换 + 月结

交付：Exchange rules、`ExchangeContributionItem`、Payroll、Obligation check、Redeemability ratio。

验收：弟子贡献可兑换修炼丹；月结发月例；兑付率影响满意。

### 41.10 Task 10：事件 + 离线追赶

交付：EventScheduler、`ChooseEventOption`、Offline catch-up、Monthly/yearly summary。

验收：离线后上线可结算收益；事件可选择并产生后果。

---

## 42. 实现智能体必须遵守的代码风格

1. Domain 层不得依赖 Actor、DB、WebSocket。
2. Actor 层负责消息编排，不写复杂公式。
3. Persistence 层不得包含玩法逻辑。
4. Config 数据不要散落在 handler 中。
5. 所有随机必须可注入 seed，便于测试。
6. 所有 command handler 必须返回 events，不直接改状态。
7. ApplyEvent 必须是唯一修改 SectState 的入口。
8. 每个命令必须有单元测试。
9. 任何新增资源、建筑、系统必须说明如何进入核心闭环。
10. 不允许引入“宗门灵气库存”。

---

## 43. 关键不变量清单

实现过程中必须写测试保护：

```text
ResourceState never negative
Contribution account never negative
SectState version monotonically increases
One sect can only have one active SectActor owner
A repeated cmd_id does not apply twice
Disciple cannot be assigned to two active tasks
Dead disciple cannot work/cultivate
Building under construction cannot produce
Task rewards cannot be claimed twice
Monthly purchase limit resets only on month advance
Aura is never stored as inventory/resource
```

---

## 44. 最终架构总结

本项目的正确后端形态是：

```text
服务器权威宗门实例模拟器
```

不是：

```text
客户端本地模拟 + 后端存档
传统 MMO 大世界状态同步
```

最终实现应满足：

```text
Client submits intent
  → Gateway validates session
  → SessionRegistry routes to single SectActor
  → SectActor validates command against authoritative SectState
  → Domain command handler produces events
  → PersistenceActor appends events
  → SectActor updates memory only through ApplyEvent
  → PersistenceActor snapshots post-ApplyEvent state as the recovery source of truth
  → ClientPushActor sends patch
  → Offline/online simulation continues through low-frequency ticks
```

玩法上，所有系统都服务下面闭环：

```text
宗门用资源供养弟子
弟子用任务反哺宗门
宗门用贡献分配资源
弟子用资源提升自身
高阶弟子完成更高阶任务
更高阶任务带来更高阶资源
宗门因此扩张
扩张带来更大风险
风险要求更强弟子和更好制度
```

任何偏离这个闭环的系统，MVP 都不实现。

---

## 45. 给后端智能体的第一条实现指令模板

你可以将下面这段直接复制给代码智能体作为第一条任务：

```text
你要实现一个 Go + Hollywood Actor 的服务器权威修仙宗门模拟经营游戏后端。请严格按照《修仙宗门模拟经营游戏设计文档 v3.1：产品与权威后端主规范 / Authoritative Backend GDD》实现。

第一阶段只实现：
1. protobuf `protocol.ClientCommand / CommandResult / StatePatch`；
2. domain.sect.SectState / DomainEvent / ApplyEvent；
3. ResourceState 和 ResourceChanged 事件；
4. PostgreSQL `sect_snapshots.state_blob`、`sect_events.event_blob`、`command_log.command_blob` schema；
5. PersistenceActor 的 LoadSnapshot、AppendEvents、SaveSnapshot；
6. SessionRegistryActor 和 SectActor 的最小可运行版本；
7. JoinScene 返回 Snapshot；
8. SubmitCommand 能处理 BuildBuilding 命令。

必须遵守：
- 客户端只能提交意图；
- SectActor 是宗门状态唯一权威；
- 命令处理器只产事件，不直接改状态；
- 状态只能通过 ApplyEvent 修改；
- snapshot blob 是权威持久化主源，不做纯事件溯源；
- 每个命令必须有 cmd_id 幂等；
- 不得把每个弟子做成 Actor；
- 不得引入宗门灵气库存。
```
