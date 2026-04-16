# ADR 0010: 离线补偿模拟（Offline Compensation Simulation）

**状态**: 已确认 (Accepted, revision 2)
**日期**: 2026-04-15
**决策者**: 项目负责人 + Codex
**相关文档**:
- `docs/vision/design-decisions.md` §4.3 云存档 + 增量同步协议
- `docs/decisions/0007-hollywood-actor-framework.md`
- `docs/decisions/0008-save-protocol-and-pgstore-schema.md`
- `docs/decisions/0011-v1-sync-model-and-multiplayer-pattern.md`
- `AGENTS.md` §4 / `docs/process/engineering-standards.md`

> **📌 注**：本 ADR 文件名保留原始 `0010-offline-deterministic-simulation.md`，但**标题和内容已重写**。为了保留讨论痕迹，变更历史在下面的 Revision History 中记录。

---

## Revision History

### 2026-04-15 rev 2（当前生效版本）

初版（rev 1）以"**字节级确定性**"为核心论证，要求模拟逻辑无论在线 tick 驱动还是离线批量快进都必须产生相等的字节输出。评审后发现这条要求对 V1 项目是**过度约束**：

- 项目是**单玩家云存档游戏**（见 ADR 0011），不是多玩家同步游戏
- 玩家不可能同时处于"在线"和"离线"两种状态，不存在可观察的反事实差异
- 服务端在某一时刻运行一次模拟，产出的结果就是权威真相
- "在线 vs 离线字节相等" 实质是在要求"**两个不存在的反事实结果必须相等**",纯粹是过度约束

被砍掉的过度约束：
- ❌ "在线 vs 离线补偿结果字节级相等"黄金测试
- ❌ `for range map` 必须排序
- ❌ 禁用 `float32/float64`
- ❌ advance() 必须字节级纯函数
- ❌ simlint 作为 CI 阻塞项（降级为告警）
- ❌ "六条铁律"的戏剧化措辞

本 rev 2 保留了核心架构（big-State + 纯 advance() 函数 + actor 层外壳），但把动机从"确定性"改为"**离线补偿的功能正确性 + 随机流连续性 + 补偿性能 + 架构简化**",约束从 6 条降到 4 条，每条用新理由论证。

### 2026-04-15 rev 1（已废弃）

初版。以"Strict Determinism + Byte-Equality"为主框架。详见 git 历史。

---

## Context

V1 范围锁定（`design-decisions.md` 2026-04-15 评审）的时间尺度：

> 1 现实小时 = 1 游戏日，弟子在玩家离线时继续生活

这一条强制要求服务端能够"**补算**"玩家离线期间应发生的事情。有两种实现方式：

### 方案 A：常驻 actor

玩家下线后 Hollywood actor 继续运行，模拟持续推进，登录时读取当前状态。

- 实现简单
- ❌ **服务器成本 ∝ 注册用户数**
- ❌ 10 万注册 / 5000 DAU 场景：9.5 万个离线玩家的 actor 树常驻，资源利用率极低
- ❌ 无法做"弟子日记"回放

### 方案 B：补偿模拟

玩家下线时快照整个 State，登录时按 `current_time - last_seen` 的差值一次性快进模拟，生成"离线期间发生的事件流"。

- ✅ 成本 ∝ DAU（5000 DAU 场景相比方案 A 节省 ~95% 资源）
- ✅ 补偿过程天然产出"弟子日记"事件流
- ❌ 要求模拟代码有能力**在不依赖外部时间和 I/O 的前提下批量快进**

V1 锁定范围讨论中已经决定采用**方案 B**。本 ADR 定义方案 B 的最小工程约束。

> **注**：ADR 0011 已明确 V1 是单玩家云存档。本 ADR 所说的"离线补偿"是单玩家世界对自己时间线的快进，**不涉及多玩家一致性**。

---

## Decision

**所有 per-player 游戏模拟逻辑封装在一个纯函数 `advance()` 中**：

```
advance : (State, Inputs, Ticks) → (State', Events)
```

其中：
- `State` 是该玩家的完整游戏状态（含 PRNG 状态 + game_tick）
- `Inputs` 是该段时间内玩家输入的有序列表
- `Ticks` 是要推进的 tick 数
- `State'` 是推进后的状态
- `Events` 是这段时间产生的观察事件（用于弟子日记 / 客户端 UI 推送）

**核心约束**：`advance()` 必须能在两种上下文下正确运行：
1. 在线 tick 驱动：每秒被调用一次，`Ticks = 1`
2. 离线补偿：登录时被调用一次，`Ticks = 登出到现在的 tick 差`

两种上下文的结果**不要求字节相等**，但都必须**逻辑正确且对玩家体验合理**。

---

## 四条工程纪律

这四条是让 `advance()` 在两种上下文下都能工作的最小纪律。每条都有**具体的功能性理由**，不是纯粹的代码洁癖。

### 纪律 1: 禁用 wall clock，一切时间走 `game_tick`

```go
// ❌ 禁止
if time.Since(disciple.LastMealTime) > 8*time.Hour { ... }

// ✅ 允许
if state.GameTick - disciple.LastMealTick > TICKS_PER_GAME_DAY/3 { ... }
```

**理由（不是确定性，是补偿功能正确性）**：

离线补偿的工作原理是"**用 game_tick 计数器快进**"：

```go
for i := 0; i < elapsedTicks; i++ {
    state, events = advance(state, nil, 1)
}
```

这个循环在**同一现实时刻**连续调用 `advance()` 60 万次。如果模拟层里写 `time.Now()`，每次循环里 `time.Now()` 的值都几乎相等，代码会判断"几乎没有时间流逝",弟子在这 60 万次 advance 里既不吃饭也不休息——**补偿根本没有发生**。

这是**功能性 bug**，不是字节差异问题。必须禁。

**唯一合法的 wall clock 使用**：Actor 外壳层入口（GatewayActor 接收玩家登录消息时）将 `time.Now()` 转换为 `game_tick`，计算 `elapsedTicks`，然后喂给 `advance()`。模拟层内部只看 `state.GameTick`。

### 纪律 2: 每玩家独立 PRNG，状态在 State 里

```go
// ❌ 禁止
if rand.Float64() < 0.1 { ... }   // 全局 rand，每次 seed 不一样

// ✅ 允许
if state.RNG.Float64() < 0.1 { ... }  // 玩家独立 PRNG，状态持久化
```

**理由（不是字节相等，是随机流连续性）**：

如果 RNG 是全局的：
- 玩家登出时 RNG 内部状态丢失
- 玩家登录时 RNG 重新初始化（通常用 `time.Now()` 作 seed）
- 玩家会观察到：每次登录后的第一次随机事件总是"重开一条序列",例如"刚上线战斗总暴击"、"每次登录后第一次采药总是稀有"等规律

这是玩家可以**感知到**的体验 bug。把 RNG 状态放 State 里并持久化，保证了"**存档前后的随机流是一条连续河流**"。

**实现规格**：
- V1 使用 Go 标准库 `*rand.Rand`（Mersenne Twister）
- PRNG seed 在玩家首次创建时生成，存 `player_state.rng_seed`
- 运行时 PRNG 状态（internal state bytes）存 `State.RNG` 字段，随 protobuf 序列化
- 反序列化后 RNG 从中断点继续生成

### 纪律 3: `advance()` 内禁止 I/O

```go
// ❌ 禁止
func advance(state State, inputs []Input, ticks int) State {
    log.Printf("...")                   // 日志库可能有缓冲、时间戳
    db.Exec("INSERT ...")               // 数据库写入
    http.Get("...")                     // 外部 HTTP
    go recomputeBackground()            // goroutine
    return state
}

// ✅ 允许
func advance(state State, inputs []Input, ticks int) (State, []Event) {
    var events []Event
    for i := 0; i < ticks; i++ {
        state, newEvents = tickOnce(state)
        events = append(events, newEvents...)
    }
    return state, events
}
```

**理由（不是纯函数教条，是补偿性能 + 架构解耦）**：

1. **性能**：离线补偿最坏情况要跑 604800 次 advance（7 天 × 86400 tick）。如果 advance 内部每次写 DB 或记 log，60 万次 I/O 是不可接受的——5000 DAU 场景下登录会秒级卡顿，数据库会被打爆
2. **架构解耦**：advance 只负责"把 State 向前推进",I/O 由外层 actor（`PersistenceActor` / `SyncActor`）消费 `events` 来执行。这让 advance 的单测极其简单（传 State 进，看 State 和 events 出），也让未来迁移到批量处理、快照式推进等优化成为可能

**允许的例外**：
- Debug / test 场景下可以注入 `Tracer` 接口做 instrumentation，但 V1 不做
- 读配置（config 加载一次后常驻内存）不算 I/O

### 纪律 4: Big-State + 单一 SimulationActor

弟子 / 建筑 / 活跃 storylet **不是独立 actor**，而是同一个 `SimulationActor` 内部 `State` 结构的子字段。

**理由（不是并发顺序，是工程复杂度）**：

1. **快照简单**：一个 struct 一次序列化 vs 10+ actor 异步收集状态后拼装。特别是在 PGStore 持久化时，一次 `protobuf.Marshal(state)` 搞定一切

2. **资源仲裁简单**：10 个弟子抢同一棵树怎么处理？
   - **独立 actor 方案**：需要引入 JobBoardActor 做协调，定义 ReservationMsg / CancelMsg，处理 timeout 回滚，写死锁检测
   - **Big-State 方案**：`advance()` 里按 `SortedDiscipleIDs()` 遍历，先到先得，一个 if 搞定

3. **离线补偿是一次函数调用**：`advance(state, elapsedTicks)` 一行代码 vs 让 10 个 actor 异步协调跑 60 万次 tick 并同步推进

4. **原子性天然**：整个玩家世界在每个 tick 有明确的快照点（tick 开始 / tick 结束），没有"中间状态"

**这条约束是对 ADR 0007 最初描述的修正**。ADR 0007 初稿说"每个弟子 = 1 actor / 每个 storylet = 1 actor"，这是 actor 模型初学者常见的反模式——把 actor 当成"对象"用，结果并发复杂度爆炸。本 ADR 明确：**actor 是进程边界 / 故障隔离边界 / 消息串行化边界，不是对象边界**。V1 规模下每玩家一个 SimulationActor 就够。

> **actor 模型的正确使用**：actor 层负责"何时调用 advance()"、"advance() 的输出派发给谁"；业务逻辑本身是 advance() 这个大纯函数。这是 Erlang/Elixir 社区的 **"functional core, imperative shell"** 模式。

---

## 砍掉的（原 rev 1 约束，现在不做）

以下约束在 rev 1 存在，rev 2 移除。保留这个清单是为了**防止有人看到旧代码 / 旧文档时困惑"为什么没遵守"**。

| 原约束 | 移除理由 |
|---|---|
| `for range map` 必须排序 | 只为字节相等测试，移除后 Go 随机迭代顺序不影响 gameplay 观察 |
| 禁用 `float32/float64` | 浮点漂移只在字节相等测试下才会暴露；普通游戏逻辑用 float 没问题 |
| advance() 必须字节级纯函数 | 降级为"无 I/O 的函数"；不强制不可变数据结构、不强制值传递 |
| 在线跑 N tick 与离线补偿 N tick 字节相等黄金测试 | 错误 motivation 的产物，rev 2 明确玩家不可能同时经历两种路径 |
| simlint 作为 CI 阻塞项 | 降级为告警工具，只检查纪律 1 和 2 |
| "六条铁律 / 违反 = 必须回滚" 措辞 | 戏剧化表述，换成"四条工程纪律" |

---

## Architecture

### Actor 拓扑

```
Hollywood Engine
└── GatewayActor
    └── PlayerSupervisor (per online player, ADR 0011 明确单玩家)
        ├── TickActor
        │   └── 每秒向 SimulationActor 发 AdvanceMsg{ticks: 1}
        ├── SimulationActor          ← 持有 State，调用 advance()
        │   ├── State (含 RNG / GameTick / 宗门 / 弟子 / 建筑 / storylet)
        │   └── Receive AdvanceMsg{N}:
        │       1. newState, events = advance(state, inputs[since_last], N)
        │       2. state = newState
        │       3. for e in events: publish to local EventStream
        ├── SyncActor                 (消费 events，增量推送客户端)
        └── PersistenceActor          (消费 events，按策略写 PGStore)
```

### 离线补偿流程

```
玩家登录 → GatewayActor 收到 LoginMsg
  ↓
  查询 player_state.last_seen_wall_ms
  ↓
  计算 elapsed_ms = now() - last_seen_wall_ms
       elapsed_ticks = elapsed_ms / 1000  (V1 tick = 1 Hz)
       elapsed_ticks = min(elapsed_ticks, 7 * 86400)  // 7 天上限
  ↓
  SpawnChild PlayerSupervisor
  ↓
  PersistenceActor.LoadStateMsg → 反序列化 state_blob → State
  ↓
  SimulationActor.Receive(LoadStateMsg{state})
  ↓
  SimulationActor.Receive(AdvanceMsg{ticks: elapsed_ticks})
    → 一次性调用 advance(state, [], elapsed_ticks)
    → 产生的 events 批量投递给 PersistenceActor 写入 player_events 表
  ↓
  客户端请求"弟子日记" → 读 player_events WHERE acknowledged=false → UI 展示
  ↓
  切换到在线 1 Hz 模式（TickActor 启动定时器）
```

### SimulationActor 的关键洞察

SimulationActor **不是业务对象**，它只是 `advance()` 这个纯函数的"actor 壳"。它的责任：

1. 持有 State
2. Mailbox 串行化接收 AdvanceMsg / PlayerInputMsg / SaveTriggerMsg
3. 调用 `advance()` 更新 State
4. 把 events 转发给 SyncActor / PersistenceActor

业务并发是 State 数据结构的事（一个 tick 内要处理 10 弟子的行为 → for 循环），不是 actor 拓扑的事。

---

## Consequences

### 正面影响

1. **成本模型清晰**
   - 服务端活跃资源 ∝ DAU
   - 5000 DAU 场景比"常驻 actor"方案节省约 95% 资源

2. **弟子日记自然涌现**
   - 补偿模拟产生的 events 直接进 player_events 表
   - 不需要单独设计叙事回放系统

3. **架构简单**
   - 一个大 State + 一个 advance() + 一个 SimulationActor
   - 测试极其简单（纯函数传 State 进，看 State 出）
   - 没有跨 actor 协调复杂度

4. **Actor 模型的价值保留**
   - PlayerSupervisor 级故障隔离
   - mailbox 保证玩家输入串行处理
   - 持久化中间件接入点

### 负面影响

1. **开发者需要理解"纯函数 + State" 模式**
   - Go 开发者通常习惯命令式 + 指针
   - 新约束要求"传 State 进，返回新 State"
   - 缓解：在 code review 清单里明确 advance 层的模式，`simulation/` 包有 README

2. **调试不如命令式自然**
   - 看中间 tick 状态需要专门的 Tracer（V1 不做）
   - 缓解：埋点日志在 actor 外壳层做，不进 advance

3. **不能用 goroutine 加速 advance 内部**
   - V1 规模（10 弟子 × 少量建筑）单线程绰绰有余
   - 未来弟子数量提升到 50+ 时再评估

4. **长期离线玩家登录延迟**
   - 离线 7 天 × 86400 tick ≈ 60 万 advance 调用
   - 估算 10μs/advance → ~6 秒登录延迟
   - 缓解：loading UI 遮盖 + M1 末评估"跳跃式补偿"（无事件 tick 段解析式跳过）

### 中性影响

- **PRNG 选型**：V1 用 `math/rand`（Mersenne Twister）就够
- **State 序列化**：走 ADR 0008 定的 protobuf，和 Hollywood Storer 无缝集成
- **`simlint` 工具**：V1 最小版只检查纪律 1 和 2，是告警不是阻塞

---

## Alternatives Considered

### A. 常驻 actor（方案 A）

- ✅ 实现简单
- ❌ 成本 ∝ 注册用户，致命
- **裁定**: ❌ 不采用

### B. 补偿模拟（本 ADR）

### C. 近似补偿（非 tick-based）

玩家登录时按概率"估算"离线发生了什么，不逐 tick 跑。

- ✅ 登录快
- ❌ 弟子日记会很糊（估算不出具体时刻）
- ❌ 因果事件无法准确触发
- ❌ 不符合"宿命感 / 天机"的设计原则
- **裁定**: ❌ 不采用

### D. 半持久化 actor（LRU 休眠）

- ✅ 省一半成本
- ❌ 本质还是方案 A 的变种
- ❌ 还是需要补偿能力
- **裁定**: ❌ 不采用

### E. 原 rev 1 的"严格确定性 + 字节相等"方案

- ✅ 理论最严密
- ❌ 对单玩家云存档游戏是过度约束
- ❌ 六条铁律给开发者添加大量心智负担
- ❌ 要求自建 simlint 工具链
- **裁定**: ❌ 已于 rev 2 撤销

---

## Risks

| 风险 | 严重度 | 缓解 |
|---|---|---|
| 开发者在 simulation 层意外使用 `time.Now()` / 全局 rand | 🟡 中 | simlint 告警 + code review 清单 + `simulation/` package 禁 import `time` / `math/rand` |
| 补偿模拟对 60 万 tick 太慢 | 🟡 中 | 登录 loading UI + M1 末评估跳跃式补偿 |
| Float 精度长期累积漂移 | 🟢 低 | 非字节相等模型下无害，必要时用 fixed-point 表达核心资源 |
| SimulationActor 崩溃导致 State 丢失 | 🟡 中 | PersistenceActor 按事件持久化 + Hollywood supervision 重启 + LoadState 恢复 |
| 弟子日记事件量巨大 | 🟢 低 | 事件合并 / 按类型汇总 / UI 分页 |

---

## 使用策略

### 代码组织

```
server/internal/slggame/simulation/   ← 纯函数模拟层
├── advance.go                         # advance() 入口
├── tick.go                            # tickOnce() 单 tick 推进
├── state.go                           # State 定义
├── rng.go                             # State.RNG 封装
├── disciple_think.go                  # utility AI
├── job_dispatch.go                    # 工作调度
├── building_progress.go               # 建筑进度
├── storylet_runner.go                 # storylet tick 推进
└── events.go                          # Event 类型
```

**`simulation/` package 禁止 import**：
- `time`（整个包，连 `time.Duration` 都不要用于模拟逻辑）
- `math/rand`（包级函数，只允许通过 `state.RNG` 访问）
- `log`（记录走外层 actor）
- `net/http` / `database/sql` / 任何 I/O

**允许 import**：
- `sort` / `slices` / `encoding/binary`
- 项目内的 `types` / `config` package
- `math`（计算用，不是时间）

### `simlint` 最小版

只检查两项：
1. `simulation/` 下禁止出现 `time.Now()` / `time.Since()` / `time.Sleep()` / `time.NewTimer()`
2. `simulation/` 下禁止出现 `rand.` 的包级调用（除非是 `state.RNG.` 成员调用）

其他检查（map 迭代 / float / goroutine）**不做**。

simlint 作为 `go vet`-style 告警，CI 报告但不阻塞合并。重复违反三次升级为阻塞。

### 必备测试

1. **功能正确性测试**：给定 State，跑 `advance(state, nil, 3600)` 一次，断言弟子按预期累积资源、建筑按预期推进进度、因果按预期触发
2. **补偿等价测试**（不是字节相等）：跑 `advance(s, [], 100)` 一次 vs 100 次 `advance(s, [], 1)`，断言**关键游戏指标相等**（资源总量 / 建筑进度 / 弟子数量 / 活跃 storylet 数量）。允许字节差异（如事件时间戳排列不同），但核心 gameplay 结果必须一致
3. **长期补偿测试**：跑 7 天 604800 tick，断言耗时 < 30 秒 + 最终 State 合理

上面第 2 条的"关键指标相等"代替原 rev 1 的"字节相等"，宽松得多但足够发现真正的 bug。

### 代码审查清单（`simulation/` 层改动必查）

- [ ] 没有 `time.*` 调用
- [ ] 随机数都走 `state.RNG`
- [ ] 没有 I/O（log / DB / http / file）
- [ ] 没有 `go` 关键字（goroutine）
- [ ] 新增 State 字段有默认值 + 迁移脚本
- [ ] 有对应的单元测试

---

## 未决问题

- [ ] **跳跃式补偿的算法**：无事件连续 tick 段解析式跳过，把 60 万 tick 砍到几千次关键 advance。V1 先不做，M1 末评估
- [ ] **最大补偿时长**：V1 锁定 7 天，M1 末可以评估是否用跳跃式把上限提到 30 天
- [ ] **Tracer 接口**：debug 用的中间状态记录机制。V1 不做，M2 视需要引入
- [ ] **战斗 tick 粒度**：1 Hz 对战斗节奏够吗？战斗期间客户端插值是否需要服务端提速到 10 Hz？V1 末评估
- [ ] **State 字段级 diff 算法**：SyncActor 推送增量时如何高效算 diff？ADR 0008 扩展时定

---

## 引用

- Erlang/OTP "Functional Core, Imperative Shell" 模式
- Hollywood `actor/engine.go` —— mailbox 串行化保证
- Hollywood `actor/middleware/persistence/` —— Storer 接口
- `design-decisions.md` §4.3 —— 瘦存档 + 胖配置哲学
- `docs/decisions/0011-v1-sync-model-and-multiplayer-pattern.md` —— V1 单玩家定位
