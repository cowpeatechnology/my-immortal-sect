# ADR 0011: V1 同步模型与多人扩展模式

**状态**: 已确认 (Accepted)
**日期**: 2026-04-15
**决策者**: 项目负责人 + Codex
**相关文档**:
- `docs/vision/design-decisions.md` §4 技术栈与后端架构
- `docs/decisions/0007-hollywood-actor-framework.md`
- `docs/decisions/0010-offline-deterministic-simulation.md`
- `AGENTS.md` §4 服务端权威 / Hollywood 相关硬约束

---

## Context

项目早期讨论中多次使用"多人在线"这个表述，但含义一直模糊。这直接导致早期 ADR（0007 / 0008 / 0010）里混入了为"未来某天的多玩家实时同步"写的约束，结果 V1 的架构比实际需要复杂了一个量级。

2026-04-15 的评审明确了一件事：

> **项目负责人眼中的"多人在线"** = 多个玩家同时在线各自玩各自的宗门
> **不是** 多个玩家同时观察同一份游戏状态

这和 MMO / PvP 对战 / 共享世界的"多人"完全不是一个东西。V1 的"宗门经营"本质上是**有云存档的单人游戏**，只是大家共用一个账号系统和一套配置热更新通道。

同时，项目负责人对未来多人扩展的形态也给出了具体模型：

1. 大多数"多人功能"其实是**排行榜 / 聊天 / 活动 / 共同地图**，不是共享游戏世界
2. 共同地图 / 活动等需要实时一致的玩法，用"房间式服务端 actor"处理：加入 = 提交数据 + 订阅变动
3. 跨宗门互动（如派弟子访问朋友宗门）用"**host 全模拟 + guest 收日志**"的异步模型

这是行业成熟做法（Colyseus / Nakama / Orleans / Akka Cluster Sharding 都是这个模式），本 ADR 把这个做法固化为项目的多人架构参考。

---

## Decision

### Part A: V1 的同步范围（单玩家云存档）

V1 **没有任何**跨玩家实时同步。所有玩家各自拥有独立的 actor 树和 State，互不可见、互不干扰。

形式化定义：

```
1 个玩家 == 1 份 player_id == 1 份 State == 1 棵 actor 子树 == 1 条 WebSocket 连接
```

**V1 存在的同步**:
- Client ↔ Server 增量状态推送（服务端权威，客户端渲染）
- 云存档持久化（为跨设备续玩 + 离线补偿）

**V1 不存在的同步**:
- ❌ Player ↔ Player 实时状态同步
- ❌ 广播给观察者
- ❌ 帧同步 / Lockstep
- ❌ 共享世界 / 公共地图
- ❌ 实时聊天
- ❌ 排行榜（V2+）
- ❌ 跨宗门互动（V2+）

### Part B: V2+ 多人扩展采用的两种模式

当未来需要加入真正的多人玩法时，所有新玩法**必须**落到以下两种模式之一。新模式的引入需要新 ADR 论证为什么现有模式不够。

#### 模式 1: Room Actor（共享状态 + 订阅广播）

适用于"多个玩家同时观察 / 操作同一份状态"的场景：
- 全服排行榜
- 公共聊天频道
- 宗门间公告
- 活动副本 / 组队副本
- 临时竞技场 / 匹配对局
- 共享地图 / 共同 boss 战

##### 抽象结构

```
RoomActor (某类活动的一个实例)
  ├─ RoomState               # 这个房间的权威状态
  ├─ Subscribers Set<PID>    # 当前订阅者列表
  ├─ Receive JoinMsg{player, payload}:
  │     1. 校验可加入（容量 / 资格）
  │     2. 把 payload 整合进 RoomState
  │     3. 把玩家的 PID 加入 Subscribers
  │     4. Respond InitialStateMsg{full_state}
  │
  ├─ Receive ActionMsg{player, action}:
  │     1. 校验动作合法
  │     2. 应用到 RoomState
  │     3. 计算 StateDelta
  │     4. Broadcast StateDeltaMsg{delta} → Subscribers
  │
  └─ Receive LeaveMsg{player}:
        1. 从 Subscribers 移除
        2. 可选：把"结算快照"发给离开的玩家
        3. 如果最后一人离开 → 触发生命周期策略
```

##### Room 生命周期三类

| 类型 | 例子 | 生命周期 | 持久化 | 崩溃恢复 |
|---|---|---|---|---|
| **常驻型** | 全服排行榜 / 公共聊天大厅 / 世界公告 | Engine 启动时 Spawn，永不 Poison | ✅ PGStore | 从快照重建 |
| **按需型** | 宗门事件副本 / 活动期 / 组队副本 | 首次 Join 时 Spawn；最后 Leave + N 分钟宽限后 Poison | 可选 | 取决于是否支持重入 |
| **短时型** | 对局房间 / 临时竞技 / 匹配实例 | 匹配开始时 Spawn；匹配结束时 Poison | ❌ 只存结果 | 不恢复，失败即重匹配 |

##### 订阅者生命周期约束

- **订阅者掉线** → Hollywood dead letter 机制触发隐式 Leave
- **Room 必须监听 dead letter** 并清理死订阅者，否则广播列表会无限增长
- **Room 支持心跳超时**：N 秒未收到任何消息的订阅者视为掉线
- **订阅者不得跨 Room 持有强引用**：玩家 A 在 Room X 的订阅状态不得被 Room Y 的逻辑访问

##### 广播的两个反模式

1. **广播全量 state** —— 对大 Room 灾难性。必须使用增量 `StateDelta` 消息
2. **不按 tick 节流** —— 1000 个玩家的聊天，每人每秒 1 条消息 = 每秒 100 万次 broadcast。必须有 tick 级聚合

#### 模式 2: 异步状态转移（不对称跨世界）

适用于"某个实体临时参与另一个玩家的世界，结束后带结果回来"的场景：
- 派弟子去朋友宗门历练（项目负责人原话案例）
- 弟子被邀请参加他人庆典
- 收养其他玩家的遗孤
- 跨宗门书信

##### 抽象结构

```
玩家 A 发起"出差":
  A.SimulationActor.State
    → 提取访客数据 {visitor: Disciple, payload: Context}
    → 打包为 DispatchMsg
    → 发给 GatewayActor (or DispatchRouterActor)
    → 路由到 B.SimulationActor
    → B 的 State.VisitingDisciples 添加此访客
    → A 的 State 标记"弟子外出"状态

B 的正常 advance tick:
  → 访客弟子按正常 utility AI 参与 B 的宗门活动
  → B 的事件流里可能产生访客相关事件（战斗 / 奇遇 / 死亡）
  → B 只需要把访客当作"临时多了一个弟子",advance() 逻辑无需特殊化

出差结束条件（时间到 / 事件触发 / 主动召回）:
  B.SimulationActor
    → 打包 SettlementMsg {
        visitor_final_state: Disciple,
        journey_log: []Event,
        rewards_or_losses: ...
      }
    → 发给 A.SimulationActor
    → A 的下一个 advance tick 里应用：
       * VisitingDisciple 状态回写为常驻弟子
       * journey_log 追加到 A 的 player_events（弟子日记）
       * 奖励 / 损失进 A 的资源
```

##### 关键约束（违反 = 架构腐化）

1. **访客弟子只有一个权威 State**：出差期间权威在 B，出差前后权威在 A。**绝对禁止"两份 State 同时存在"**
2. **访客进出必须走消息传递**，不得跨 actor 直接访问对方 State
3. **访客在 B 期间的变更必须批量化**，B 不得对 A 的 State 做高频操作
4. **Settlement 失败必须可重试**：网络不稳定时消息可能丢失，Settlement 必须幂等 + 可重发
5. **访客离线不影响宿主**：玩家 A 即使下线，弟子依然在 B 的宗门里参与活动，结束后 Settlement 会等 A 上线时投递

##### 为什么这不是 Room Actor

Room Actor 的本质是"**N 个玩家看同一份共享状态**"。
异步状态转移的本质是"**一个实体的所有权从 A 的 State 临时转移到 B 的 State**"。

两者的关键差异：
- Room 的参与者退出后，Room 继续存在给其他人用；异步转移结束后，访客权威回到 A，没有"共享状态留存"
- Room 需要考虑"多个参与者同时操作的冲突"；异步转移只有 B 在操作，无冲突
- Room 需要广播 diff；异步转移只在结束时发一次 Settlement

混用这两种模式是架构腐化的第一步。**必须严格区分**。

---

## V1 的实际影响

### 不做的事（明确砍掉）

基于本 ADR，以下内容从 V1 完全移除：

- ❌ Hollywood cluster 模式相关的所有配置（未来用到再启用）
- ❌ dRPC 节点间通信（V1 单机部署，无需节点间协议）
- ❌ Room Actor 相关的代码 / proto / schema（V2+ 才开始写）
- ❌ 异步状态转移的 Dispatch / Settlement 协议（V2+）
- ❌ 订阅机制 / EventStream 的跨玩家使用（V1 的 EventStream 只在单玩家内使用）
- ❌ 任何形式的"广播给其他玩家"逻辑

### 做的事（简化明确）

- ✅ 每个玩家启动时 Spawn 一棵独立 actor 子树
- ✅ 客户端 WebSocket 1:1 对应 GatewayActor 的某个路由项
- ✅ advance() 纯函数推进单玩家世界
- ✅ PGStore 持久化 per-player State
- ✅ 离线补偿（ADR 0010）

### 对现有 ADR 的修正

- **ADR 0007 Hollywood Framework**: 原文多次把"Cluster + Consul/mDNS 服务发现 / 横向扩展 / UGC 第二曲线扩展路径"作为选择 Hollywood 的卖点。这些部分对 V1 不构成决策依据，仅保留作为"未来可能的扩展"标注。选择 Hollywood 的 V1 理由足够独立：单 actor 串行化 + 监督树 + mailbox 已经值回票价
- **ADR 0008 存档协议**: 原文未决问题里的"跨服迁移 / 集群化灾备"明确标记为 V3+
- **ADR 0009 Actor ID**: 原文未决问题里的"集群化后的 PID 路由"明确标记为 V3+
- **ADR 0010 离线补偿模拟**: 本 ADR 明确其服务对象是"单玩家云存档"而非"多玩家一致性",进一步简化 0010 的论证（不再需要任何"跨玩家状态一致"的约束）

---

## 设计原则汇总

### 原则 1: 状态所有权单一

任何一个游戏状态片段（弟子 / 资源 / 建筑 / 活跃 storylet）**任何时刻只由一个 SimulationActor 持有权威**。跨 actor 的读写都必须走消息传递，不得持有指针。

### 原则 2: per-player 边界不可穿透

玩家 A 的 State 不得直接被玩家 B 的代码访问。所有跨玩家的信息流动都是**显式的消息 + 快照**。

### 原则 3: Room 和 per-player 状态严格隔离

Room 里的数据不得持久化到 per-player save，per-player 的数据进 Room 时必须显式 snapshot。

### 原则 4: "多人"先问哪种模式

新玩法设计时第一个问题必须是："**这是 Room Actor 模式还是异步状态转移模式，或是纯 per-player？**" 没回答这个问题就不允许开工。

### 原则 5: 现在不为未来写代码

V1 不得出现任何"为了未来 V2 的 Room Actor 做铺垫"的代码。铺垫代码是架构腐化的种子。V2 要用时再写。

---

## Consequences

### 正面影响

1. **V1 工作量大幅简化**
   - 不做任何同步层 / 广播层 / 订阅管理
   - 不用集成 Hollywood cluster
   - 不用设计多玩家协议
   - 估算节省 3~4 周 M0/M1 工作量

2. **架构边界清晰**
   - 每个玩家是独立的"小世界",心智负担最小
   - 不会出现"这个状态到底归谁管"的问题
   - 调试简单（单玩家问题复现不需要搭建多玩家环境）

3. **V2+ 扩展路径明确**
   - Room Actor 和异步状态转移两个模式覆盖已知需求
   - Hollywood 原生支持这两种模式
   - 未来新玩法按模板写即可，不会每次重新发明架构

4. **ADR 0010 的论证进一步简化**
   - 离线补偿不需要处理"和其他玩家状态一致"的问题
   - 单玩家世界的 advance() 是纯粹的工程问题

### 负面影响

1. **"多人在线"宣传口径需要小心**
   - 如果市场用"多人在线"做买量素材，玩家可能期待共享世界
   - 缓解：文案用"云端宗门 / 离线生长 / 跨设备续玩"等准确表达

2. **V2 引入 Room Actor 时是一次真正的架构升级**
   - 订阅管理 / dead letter 清理 / 增量广播 / 心跳超时 / 断线重连都要一次性到位
   - 估算 V2 引入多人玩法的门槛：**3~5 周**基础设施 + 具体玩法内容
   - 必须正视：V2 的多人不是"加一个按钮"

3. **某些玩家期待的"小交互"V1 做不了**
   - 不能点朋友宗门看一眼
   - 不能给朋友送礼物
   - 缓解：**异步转移模式**在 V1.5 可以补一个最简版本（纯礼物 / 纯书信），工作量约 1 周

### 中性影响

- **服务端成本模型依然是 DAU-linear**，不受多人模式影响
- **Hollywood cluster 功能被搁置**，这是正确的——V3+ 再启用
- **Gateway 协议简化**：V1 的 gateway 只处理 client ↔ 单个 per-player actor tree

---

## Alternatives Considered

### A. 从 V1 就做完整 Room Actor 基础设施

- ✅ 未来加玩法时没门槛
- ❌ V1 的 M0 范围膨胀 3~4 周
- ❌ 基础设施没有真实玩法驱动，容易过度抽象
- ❌ 违反"只为当前问题写约束"原则
- **裁定**: ❌ 不采用

### B. V1 用共享世界架构（类 MMO）

- ✅ 一次到位
- ❌ 项目负责人明确表达过不要这种形态
- ❌ 服务端成本爆炸
- ❌ 和"宗门是家"的情绪锚点冲突
- **裁定**: ❌ 不采用

### C. V1 只做 client-server，V2+ 临时拼装多人架构

- ✅ V1 最简
- ❌ 没有预先规划，V2 时每个新玩法都会重新发明架构
- ❌ 容易引入反模式（广播全量 state / 不清理订阅者）
- **裁定**: ❌ 不采用，本 ADR 采用 V1 简化 + V2+ 模式预先登记的混合方案

### D. 用现成的游戏服务器（Nakama / Colyseus）包装

- ✅ Room Actor 开箱即用
- ❌ 已经在 ADR 0007 中评估过并拒绝
- ❌ 需要维护两套 runtime
- **裁定**: ❌ 不采用

---

## Risks

| 风险 | 严重度 | 缓解 |
|---|---|---|
| V2 引入多人时低估了 Room Actor 基础设施工作量 | 🟡 中 | 本 ADR 明确登记"V2 多人门槛 3~5 周基础设施",避免产品侧过早承诺玩法 |
| V1 代码里偷偷出现"为未来铺垫"的 dead code | 🟡 中 | Code review 清单：检查是否有 V1 用不到的抽象 |
| 异步状态转移的 Settlement 消息丢失 | 🟡 中 | V2+ 时实现：持久化 pending settlement + 幂等应用 + 重试 |
| 玩家期待"好友系统"但 V1 没有 | 🟢 低 | 文案管理 + V1.5 补最简异步礼物 |
| Room 状态大时广播风暴 | 🟢 低 | V2+ 实现时强制使用 StateDelta，不允许 full state broadcast |

---

## 使用策略

### V1 代码约束

- `server/internal/slggame/` 下**不得出现**以下内容：
  - `room/` 目录
  - `RoomActor` 类型
  - `Subscribers`、`Broadcast`、`Publish` 等词汇（单玩家场景用不着）
  - 任何玩家 ID 的集合操作（`for _, pid := range allPlayers`）
  - Hollywood cluster 相关配置
- GatewayActor 的路由表是 `map[player_id] → *PlayerSupervisor PID`，**没有广播接口**

### V2+ 引入多人时的流程

新多人玩法必须走以下流程：

1. 分类：这是 Room Actor 还是异步状态转移？
2. 写 ADR：`00XX-玩法名.md`，引用本 ADR，说明属于哪种模式
3. 实现参考本 ADR 的"抽象结构"
4. code review 确认没有越界（per-player 不能持有 room 引用，反之亦然）

### V1 的 Gateway 协议

- 每个 WebSocket 连接 1:1 对应一个 player_id
- Gateway 的唯一路由逻辑：`session_token → player_id → PlayerSupervisor PID`
- 没有"房间"、"频道"、"订阅"概念
- 协议 proto 文件只有 `gateway/client_proto.proto`，不要做 `gateway/room_proto.proto`

---

## 未决问题

- [ ] **V1.5 的"最简异步交互"是否要做**（纯书信 / 纯礼物）？如果做，什么时机？留给 M1 末评估
- [ ] **V2 第一个 Room 玩法是什么**？排行榜？聊天？活动副本？顺序决定先做哪类基础设施
- [ ] **异步状态转移的消息路由**：Gateway 直接路由还是引入专门的 DispatchRouter？V2+ 实现时定
- [ ] **Settlement 消息的持久化机制**：Pending settlement 放 `player_state` 里还是独立表？ADR 0008 扩展时定
- [ ] **房间快照的粒度**：哪些字段常驻 PGStore，哪些只在 room 生命周期内内存存在？V2+ 设计时定
- [ ] **心跳超时阈值**：房间订阅者多久未活动算掉线？15s？60s？V2+ 定

---

## 引用

- **Colyseus 文档**（`https://docs.colyseus.io/`）—— Room 抽象的业界标准实现
- **Nakama Match 文档**（`https://heroiclabs.com/docs/nakama/concepts/server-framework/match-handler/`）
- **Microsoft Orleans Virtual Actors**（`https://learn.microsoft.com/en-us/dotnet/orleans/`）
- **Akka Cluster Sharding**（`https://doc.akka.io/docs/akka/current/typed/cluster-sharding.html`）
- **Hollywood `actor/eventstream.go`** —— V2+ Room 广播的底层原语
- **Hollywood `actor/context.go` Forward / Respond** —— 异步状态转移的消息传递基础
