# M0 技术垂直切片计划

> **状态说明**：historical / superseded
> 本文件是早期 M0 切片计划，已不再定义当前产品与后端主线。当前主规范见：
> - [docs/vision/gdd_v3_backend_design.md](/Users/mawei/MyWork/SlgGame/docs/vision/gdd_v3_backend_design.md)
> - [docs/plans/gdd-v3-authority-core-bootstrap.md](/Users/mawei/MyWork/SlgGame/docs/plans/gdd-v3-authority-core-bootstrap.md)

**状态**: 草案 (Draft)
**最后更新**: 2026-04-15
**依赖**:
- `docs/vision/design-decisions.md` §8 M0 章节
- `docs/decisions/0007-hollywood-actor-framework.md`
- `docs/decisions/0008-save-protocol-and-pgstore-schema.md`
- `docs/decisions/0009-actor-id-conventions.md`
- `docs/decisions/0010-offline-deterministic-simulation.md`
- `AGENTS.md`
- `docs/process/engineering-standards.md`
- `docs/features/F-002-sect-map-playability-validation.md`

---

## Context

V1 范围已经在多轮讨论中锁定（见 `design-decisions.md` V1 锁定范围总表）。4 份核心 ADR 已经把架构原则和硬约束固化。**M0 的目的是用最少的代码打通从 Cocos 客户端到 Hollywood actor 到 PostgreSQL 的完整链路，并在真机上压测技术红线**。

Phase 1 的大地图可玩性验证切片是 M0 的前置门槛。2026-04-16 已完成一次 prototype 级探索，但该探索不等于门槛通过。真正的通过标准仍然是基于 `Cocos Creator + TypeScript + Tiled / Cocos TiledMap` 与 `Go + Hollywood` 的真实切片验证。

> 原则：M0 不追求功能丰富，追求**架构全链路通**。每个模块只做最小闭环，但所有模块都必须存在。

这个阶段最怕两件事：
1. **先写内容后写骨架**：策划写 10 条因果链但底层引擎还是 `TODO` → 全废
2. **只做单端**：客户端先跑通 demo 跑完又发现服务端模型不对 → 重做

所以 M0 的任务排序遵循 **"底层先行、全链路优先"**：先把 `advance()` 的最小纯函数 + 最小 actor 树 + 最小 PGStore + 最小客户端渲染打通，再填肉。

---

## M0 产出定义（什么叫"M0 完成"）

**一个能在真机上玩 10 分钟的技术 demo**，必须满足以下全部条件：

### 功能闭环
- [ ] 玩家能登录（本地 dev 账号）
- [ ] 能看到 32×32 tilemap 宗门，1 个弟子大头在地图上
- [ ] 弟子有 3 种情绪状态可切换（愉悦 / 平静 / 受伤）
- [ ] 玩家能建造 1 种建筑（`building.alchemy_room`）并看到进度推进
- [ ] 弟子能自主选择去砍柴（1 种工作：`job.gather_wood`）并累积资源
- [ ] 触发 1 条完整因果链的第一个 stage（`karma.first_disciple_arrival`）
- [ ] 玩家登出后 1 分钟再登录，看到"弟子日记"显示离线期间发生的事件
- [ ] 服务端重启后玩家重新登录能看到相同状态（持久化验证）

### 技术指标
- [ ] 红米 Note 8 上 30 弟子 + 建筑同屏渲染 ≥ **45 fps**（弟子数量注水测试）
- [ ] 常规操作（建造 / 情绪切换 / 事件触发）同步包 ≤ **200 字节**
- [ ] 登录首屏渲染 ≤ **5 秒**
- [ ] 服务端单机 100 并发 p99 ≤ **200ms**（5000 DAU 目标的 1/50 抽样）
- [ ] `simulation.Advance()` 单 tick p99 ≤ **50μs**（离线补偿预算的前提）

### 架构完整性
- [ ] Hollywood actor 树完整（Gateway / PlayerSupervisor / Simulation / Tick / Persistence / Sync）
- [ ] `simulation` package 通过 `simlint` 零告警
- [ ] PGStore 实现 Hollywood Storer 接口 + 自建 SaveRepo
- [ ] Protobuf 工具链跑通，客户端 / 服务端 / State 各自生成类型
- [ ] 确定性测试三条全部通过（ADR 0010 §测试必备三条）
- [ ] 离线补偿原型跑通（登出 → 等待 → 登录 → advance → 弟子日记）

### 文档与工具
- [ ] `simlint` 最小可用版本（至少能检测 `time.Now` / 包级 `rand` / `for range map`）
- [ ] 配置表工具链：Excel → JSON → Go struct + TS .d.ts
- [ ] CI 流水线：lint → test → determinism test → build
- [ ] `README.md` 有 "如何本地跑起来" 步骤，非原作者能按说明跑通

**未达到全部条件不进 M1。必要时回到 M0 再打磨一周，不要往前堆功能。**

---

## 时间预算

**总计：4~5 周（20~25 工作日）**

这是"人 + Codex"混合估计，**不是"Codex 独立完成"的估计**。按小批次、高频循环推进的节奏估算。

如果 Week 3 末尾（actor 树 + PGStore 完成时）发现延期超过 3 天，立即重新评估剩余任务，宁可砍内容也不延迟 M0 的架构验收。

---

## 任务拆分（按周 → 按天）

### Week 0：前置准备（0~3 天，计划阶段结束时完成）

这些不算 M0 工作量，但 Week 1 第一天开工必须先就位。

- [ ] P0-1：Hollywood fork 仓库确认可访问，版本锁在 fork 的某个 commit
- [ ] P0-2：本地开发环境：Go ≥ 1.22 / PostgreSQL 15 / Cocos Creator / protoc / Node.js
- [ ] P0-3：项目目录结构定下来：
  ```
  my-immortal-sect/
  ├── client/
  ├── server/
  │   ├── cmd/gameserver/
  │   ├── internal/slggame/
  │   │   ├── simulation/
  │   │   ├── actors/
  │   │   ├── storage/
  │   │   ├── config/
  │   │   └── simlint/
  │   ├── internal/proto/
  │   ├── migrations/
  │   └── tests/
  ├── shared/configs/
  └── tools/
  ```
- [ ] P0-4：`go mod init` / `npm init` / `protoc` 脚本就位
- [ ] P0-5：美术最小套件：1 张大头底板 + 3 种情绪覆盖（先手绘，不要 AI 糊）
- [ ] P0-6：起草 `shared/configs/storylets/karma.first_disciple_arrival.json` 一条完整因果链文本

---

### Week 1：服务端骨架 + 纯函数模拟层最小版

**目标**：`server/internal/slggame/simulation/` 可以独立运行一个最小 State 的 100 tick 推进，且通过确定性测试。

#### Day 1–2：工程搭建
- [ ] S1-1：`server/` Go module 初始化，依赖 Hollywood fork（走 `replace` 指向本地或 fork URL）
- [ ] S1-2：CI 流水线搭建：`go vet` / `golangci-lint` / `go test`
- [ ] S1-3：PostgreSQL 本地 docker-compose，建库 `slggame_dev`
- [ ] S1-4：Protobuf 工具链：`buf` 或原生 `protoc` + `vtprotobuf` 插件

#### Day 3–4：Proto schema 初稿
- [ ] S1-5：`proto/state/player_state.proto`（顶层 State）
- [ ] S1-6：`proto/state/sect.proto`（资源 × 5、境界、解锁建筑列表）
- [ ] S1-7：`proto/state/disciple.proto`（6 属性 + 3 情绪 + 当前位置 + 当前工作意图）
- [ ] S1-8：`proto/state/building.proto`（类型 + 等级 + 位置 + 建造进度）
- [ ] S1-9：`proto/state/storylet.proto`（instance_id + template_id + current_stage + variables）
- [ ] S1-10：`proto/state/rng.proto`（PRNG 状态）
- [ ] S1-11：`proto/state/events.proto`（DiscipleCreated / ResourceChanged / BuildingCompleted / KarmaStageEntered 等 8~10 种最小集）

#### Day 5：simulation package 骨架
- [ ] S1-12：`simulation/state.go`：State struct + `SortedDiscipleIDs()` / `SortedBuildingIDs()` 等辅助
- [ ] S1-13：`simulation/rng.go`：封装 `*rand.Rand`，seed 持久化
- [ ] S1-14：`simulation/advance.go`：`Advance(state, inputs, ticks)` 纯函数入口
- [ ] S1-15：`simulation/tick.go`：`tickOnce(state)` 单 tick 推进骨架（此时只推进 `GameTick++`）
- [ ] S1-16：`simulation/events.go`：Event 类型 + 产出接口

#### Day 6–7：确定性基础设施
- [ ] S1-17：`simlint/main.go`：最小静态检查（禁用 imports / for range map / float / time / 包级 rand）
- [ ] S1-18：CI 把 simlint 加入阻塞检查
- [ ] S1-19：`tests/determinism_test.go`：
  - 测试 1：`Advance(s, [], 100)` 跑两次 state bytes 相等
  - 测试 2：`Advance(s, [], 100)` 等于 `for i in 100 { Advance(s, [], 1) }`
  - 测试 3：property-based 随机 seed × 100 个 input 序列都满足 1 和 2
- [ ] S1-20：跑通三条测试

**Week 1 验收**: `simulation` package 独立可运行，`simlint` 通过，三条确定性测试绿。**不需要涉及任何 actor 或 DB**。

---

### Week 2：Utility AI 最小版 + 1 种工作 + 1 种建筑 + 因果触发骨架

**目标**：一个 State 里有 1 个弟子自主砍柴、1 个建筑能建造，因果事件能被触发。所有逻辑仍然在 `simulation` package 里。

#### Day 8–9：Utility AI 最小版
- [ ] S2-1：`simulation/disciple_think.go`：utility AI 评估 4 种行为（砍柴 / 空闲 / 吃饭 / 休息）
- [ ] S2-2：配置表 `shared/configs/ai_weights.json`：utility 权重默认值
- [ ] S2-3：弟子意图状态机：`Idle → MovingTo → Working → Returning → Idle`
- [ ] S2-4：测试：1 个弟子在空地图上能进入 Working 状态并产出木材
- [ ] S2-5：property test：1000 次随机 tick 下弟子状态机不会卡死

#### Day 10–11：建筑建造
- [ ] S2-6：`simulation/building_progress.go`：tick 推进建筑进度
- [ ] S2-7：配置表 `shared/configs/buildings.json`：`building.alchemy_room` 定义（L1: 30 游戏分钟建成 = 1800 tick，消耗 20 木材）
- [ ] S2-8：`CreateBuildingInput` 处理（玩家下建造命令）
- [ ] S2-9：`BuildingCompleted` event 产出
- [ ] S2-10：测试：发 CreateBuilding 输入 → 推进 1800 tick → 建筑完成 + 事件产出

#### Day 12–13：Storylet 引擎最小版
- [ ] S2-11：`simulation/storylet_runner.go`：storylet 实例状态机
- [ ] S2-12：`simulation/karma_trigger.go`：触发条件评估器
- [ ] S2-13：配置表 `shared/configs/storylets/karma.first_disciple_arrival.json`：
  - 触发条件：`disciple_count == 1 && game_tick > 600`
  - Stage 1：叙事文本 + 2 个分支选择
  - Stage 2（接纳分支）：生成第二个弟子
  - Stage 2（拒绝分支）：获得少量资源 + 留下"遗憾"标记
- [ ] S2-14：测试：State 满足触发条件时，新的 storylet instance 被 spawn 进 State

#### Day 14：Week 2 整合测试
- [ ] S2-15：`tests/vertical_slice_test.go`：从零初始化 State，跑 3600 tick，断言：
  - 弟子累积了木材
  - 建筑建造完成
  - 因果 storylet 被触发进入 Stage 1
  - 整个过程确定性测试依然通过

**Week 2 验收**: 整个 V1 simulation 的"神经系统"已经在纯函数层可见。此时还没有 actor、没有 DB、没有客户端，但游戏的核心确定性循环已经跑通。

---

### Week 3：Hollywood Actor 树 + PGStore + 离线补偿

**目标**：把 Week 2 的纯函数 `Advance` 包进 actor 层，接上 PostgreSQL 持久化，实现离线补偿。

#### Day 15–16：Actor 骨架
- [ ] S3-1：`actors/gateway_actor.go`：接收 LoginMsg / LogoutMsg，根据 player_id 路由
- [ ] S3-2：`actors/player_supervisor.go`：Spawn 子 actor 树
- [ ] S3-3：`actors/tick_actor.go`：1 Hz 定时器发送 AdvanceMsg
- [ ] S3-4：`actors/simulation_actor.go`：持有 State，调用 `simulation.Advance()`，产出 events
- [ ] S3-5：`actors/sync_actor.go`：订阅 events，打包发给客户端（此时客户端还是 stub）
- [ ] S3-6：Hollywood Engine 启动入口 `cmd/gameserver/main.go`
- [ ] S3-7：测试：本地启动 Engine，发 LoginMsg，看到弟子自动砍柴推进 10 秒

#### Day 17–18：PGStore + SaveRepo
- [ ] S3-8：`storage/pgstore/pgstore.go`：Hollywood Storer 接口实现
- [ ] S3-9：`migrations/schema/001_initial.sql`：创建 ADR 0008 四张表
- [ ] S3-10：`storage/saves.go`：SaveRepo 接口
- [ ] S3-11：`storage/saves_pg.go`：
  - `LoadPlayerState` / `SavePlayerState`
  - `AppendEvents` / `MarkEventsAcknowledged`
  - `SoftDeletePlayer`（skeleton，逻辑留空）
- [ ] S3-12：`actors/persistence_actor.go`：接收 SnapshotRequest，调用 SaveRepo
- [ ] S3-13：测试：启动 Engine → 跑 10 秒 → 登出 → 查 DB 确认 state_blob + events 写入正确
- [ ] S3-14：测试：重启 Engine → 再登录 → 反序列化 State 正确 → 继续跑

#### Day 19–20：离线补偿原型
- [ ] S3-15：登录流程补齐：查 `last_seen_wall_ms` → 计算 elapsed_ticks → 发 AdvanceMsg{ticks: elapsed_ticks}
- [ ] S3-16：实现 7 天离线上限
- [ ] S3-17：事件聚合：advance 产出的 events 进入 DB events 表
- [ ] S3-18：GetUnacknowledgedEvents API（为客户端日记 UI 准备）
- [ ] S3-19：测试：登出 → 手动跳时间 1 小时 → 登录 → 补偿模拟跑 3600 tick → 弟子状态正确推进 + events 落地
- [ ] S3-20：确定性测试扩展：在线跑 1 小时 vs 离线补偿 1 小时，两份 state_blob 字节级相等（**这是 M0 最严苛的测试**）

**Week 3 验收**: 服务端全链路可运行。确定性黄金测试通过。任何客户端不在场的情况下，服务端都能模拟一个宗门跑起来。

---

### Week 4：客户端 Tilemap + 大头渲染 + WebSocket Gateway

**目标**：Cocos 客户端能连上服务端，显示宗门和弟子，看到实时状态推送。

#### Day 21–22：WebSocket Gateway 扩展
- [ ] C4-1：`actors/gateway_actor.go` 扩展：接收 WebSocket 连接 + protobuf 消息
- [ ] C4-2：Gateway 协议最小集：
  - ClientLogin / ClientLogout
  - ClientInput{kind: CreateBuilding | ChangeAIWeight | ChooseStoryletBranch}
  - ServerStateSnapshot（初始全量）
  - ServerStateDelta（增量）
  - ServerDiary（弟子日记）
- [ ] C4-3：测试：curl / ts-node 脚本能连 WebSocket 收发 protobuf

#### Day 23–24：Cocos Creator 工程
- [ ] C4-4：Cocos 项目初始化，TypeScript strict
- [ ] C4-5：`client/assets/scripts/systems/SyncClient.ts`：WebSocket 客户端 + protobuf 解码
- [ ] C4-6：`client/assets/scripts/render/HeadRenderer.ts`：大头组件化渲染（底板 + 情绪层）
- [ ] C4-7：`client/assets/scripts/render/TilemapHost.ts`：Tiled 导入 + 2.5D 菱形渲染
- [ ] C4-8：`client/assets/scripts/systems/EventBus.ts`：模块间通信
- [ ] C4-9：最小 UI：左上角资源栏 + 右下角建造按钮

#### Day 25：端到端联调
- [ ] C4-10：客户端连服务端，显示 1 个弟子在 tilemap 上移动
- [ ] C4-11：点击建造按钮 → 发送 CreateBuildingInput → 服务端 advance → 推送 delta → 客户端显示建筑
- [ ] C4-12：情绪切换演示（手动 mock 或通过事件触发）
- [ ] C4-13：登出重登 → 显示弟子日记 UI（哪怕只是原始文本）

**Week 4 验收**: 能在电脑上看到一个会动的 demo。

---

### Week 5（1~2 天）：真机压测 + 整合 + 文档

#### Day 26–27：压测与打磨
- [ ] T5-1：注水 30 个弟子 / 建筑测试客户端帧率
- [ ] T5-2：测量实际同步包大小
- [ ] T5-3：Apache Bench 测服务端 p99
- [ ] T5-4：measure `simulation.Advance()` 单 tick 耗时
- [ ] T5-5：红米 Note 8 或等价真机跑 30 分钟无崩溃

#### Day 28：收尾
- [ ] T5-6：`README.md` 写本地运行步骤
- [ ] T5-7：把验收清单逐条打钩
- [ ] T5-8：总结 M0 学到的教训（进 `docs/plans/m0-retrospective.md`）
- [ ] T5-9：决定是否进 M1：如果验收有未达标项，延期 M0

---

## 依赖关系图

```
Week 1 (server skeleton)
  ↓ (simulation 可独立运行)
Week 2 (utility AI + building + storylet 骨架)
  ↓ (确定性 core 完成)
Week 3 (actor + PGStore + 离线补偿)
  ↓ (服务端全链路)
Week 4 (client + WebSocket)
  ↓ (demo 可视)
Week 5 (真机压测)
```

Week 1 / 2 / 3 严格串行（底层依赖），Week 4 可以在 Week 3 Day 17 之后部分并行开始（先做 Cocos 工程初始化和 HeadRenderer 等不依赖服务端的部分）。

---

## 并行机会

以下任务 Codex 可以拆成独立 agent 并发跑：

| 主任务 | 可独立的子任务 | 并发性 |
|---|---|---|
| Week 1 | Proto 定义 / simulation 骨架 / simlint | 3 并发 |
| Week 2 | utility AI / building / storylet | 3 并发 |
| Week 3 | actor 骨架 / PGStore / 确定性测试 | 2 并发（actor + PGStore 并行，最后合并） |
| Week 4 | Gateway / Cocos 工程 / HeadRenderer | 3 并发 |

但**集成点必须串行**：Week 1 末尾的确定性测试、Week 3 末尾的"在线 vs 离线补偿"黄金测试、Week 5 的真机压测，都是一夫当关的检查点。

---

## 风险与应急

| 风险 | 概率 | 影响 | 应急方案 |
|---|---|---|---|
| simlint 实现拖延 | 中 | 中 | 降级为人工 code review 清单 + unit test；不卡住 Week 1 推进 |
| Hollywood 源码有坑要绕过 | 中 | 中 | 记录到 fork 仓库 issue，一时不能解决的先 patch fork，M1 再 upstream |
| Protobuf 工具链双端生成复杂 | 中 | 中 | 先手写类型，工具链用 M1 优化；不让它卡住 Week 1 |
| 确定性黄金测试失败 | 低 | 🔴 高 | **立即停下**，定位 non-determinism 源头，哪怕延期一周 |
| 真机帧率不达标 | 中 | 🔴 高 | 减少同屏弟子数 / 简化大头组件 / 关闭特效 / 使用更小 tilemap |
| 离线补偿跑 1 小时超时 | 低 | 中 | 调低 tick 频率到 0.5 Hz，或简化 utility AI 计算 |
| 美术资源没到位 | 高 | 低 | 用占位图跑，不卡住代码进度；M1 美术管线时替换 |
| PostgreSQL 本地环境问题 | 低 | 低 | 用 SQLite 作为 M0 过渡（Storer 接口抽象允许） |

---

## 不做的事（M0 范围外）

这些明确不进 M0，避免范围蔓延：

- ❌ 弟子属性的完整 6 项（M0 只做到"砍柴需要体魄"的最小子集）
- ❌ 所有 12~15 个特质（M0 只做 1~2 个用于因果触发）
- ❌ 境界系统 / 突破（M0 弟子不进阶）
- ❌ 战斗系统（M0 无敌袭）
- ❌ 多个建筑种类（M0 只 1 种）
- ❌ 多种资源（M0 只 `res.wood`）
- ❌ 完整的工作类型（M0 只 `job.gather_wood`）
- ❌ 完整的存档协议（M0 只做到能存能读，暂不做加密 / 压缩）
- ❌ 完整因果链（M0 只 1 条链的 Stage 1 和 Stage 2）
- ❌ 神降（V2 才做）
- ❌ 宗门外大地图（V2 才做）
- ❌ 新手引导（V2 才做）
- ❌ 集群化（V3 才评估）
- ❌ Redis 缓存（M1 加）
- ❌ 监控 / metrics（M1 加）
- ❌ 跳跃式离线补偿（M2 评估）

**M0 里看到"顺手加一个" 必须拒绝**。

---

## 交付物清单

完成 M0 时，仓库里应该有：

```
server/
├── cmd/gameserver/main.go
├── internal/slggame/
│   ├── simulation/
│   │   ├── state.go / advance.go / tick.go / rng.go / events.go
│   │   ├── disciple_think.go
│   │   ├── building_progress.go
│   │   ├── storylet_runner.go
│   │   └── karma_trigger.go
│   ├── actors/
│   │   ├── gateway_actor.go / player_supervisor.go
│   │   ├── simulation_actor.go / tick_actor.go
│   │   ├── persistence_actor.go / sync_actor.go
│   ├── storage/
│   │   ├── pgstore/pgstore.go
│   │   ├── saves.go / saves_pg.go
│   ├── simlint/main.go
│   └── config/ (loader)
├── internal/proto/
│   ├── state/*.proto (generated .go)
│   └── gateway/*.proto (generated .go)
├── migrations/
│   ├── schema/001_initial.sql
│   └── ...
└── tests/
    ├── determinism_test.go
    ├── vertical_slice_test.go
    └── online_vs_offline_test.go

client/
├── assets/scripts/
│   ├── render/HeadRenderer.ts
│   ├── render/TilemapHost.ts
│   ├── systems/SyncClient.ts
│   ├── systems/EventBus.ts
│   └── ui/*
├── assets/resources/ (占位美术)
└── tsconfig.json (strict: true)

shared/configs/
├── buildings.json (只 1 种)
├── ai_weights.json
├── emotions.json
└── storylets/
    └── karma.first_disciple_arrival.json

tools/
├── gen_proto.sh
└── excel_to_json.py (最小版本)

docs/plans/
├── m0-vertical-slice.md (本文件)
└── m0-retrospective.md (M0 结束时写)
```

---

## 验收评审

M0 结束时，需要对照以下清单逐项打钩。全绿才能进 M1。

```
功能闭环
[ ] 登录 / 登出 / 重登
[ ] 1 个弟子大头 + 情绪切换
[ ] 建造 1 种建筑
[ ] 弟子自主砍柴
[ ] 触发因果链 Stage 1
[ ] 离线补偿 + 弟子日记

技术指标
[ ] 30 弟子同屏 ≥ 45 fps (红米 Note 8)
[ ] 同步包 ≤ 200 字节
[ ] 登录 ≤ 5 秒
[ ] 服务端 p99 ≤ 200ms
[ ] simulation.Advance() tick p99 ≤ 50μs

架构完整性
[ ] Actor 树六件套就位
[ ] simlint 零告警
[ ] PGStore + SaveRepo 就位
[ ] Protobuf 工具链
[ ] 确定性三条测试通过
[ ] 离线补偿黄金测试通过

文档
[ ] README 可复现
[ ] m0-retrospective.md 完成
```

---

## 相关文档

- `docs/vision/design-decisions.md`（North Star）
- `docs/decisions/0007-hollywood-actor-framework.md`
- `docs/decisions/0008-save-protocol-and-pgstore-schema.md`
- `docs/decisions/0009-actor-id-conventions.md`
- `docs/decisions/0010-offline-deterministic-simulation.md`
- `AGENTS.md`（仓库级总章程）
- `docs/process/engineering-standards.md`（工程规范）

## 未决问题

- [ ] 美术最小套件何时就位？（P0-5 的实际负责人和时间）
- [ ] 红米 Note 8 真机获取路径？或替代设备？
- [ ] 部署环境：M0 结束时要不要推到云服务器？还是留在本地？
- [ ] 是否需要在 M0 阶段引入 git flow？（建议：main + feature/* 简单流）
- [ ] commit 粒度的具体约定写进哪？（当前以根 `AGENTS.md` 与 `docs/process/engineering-standards.md` 为准）
