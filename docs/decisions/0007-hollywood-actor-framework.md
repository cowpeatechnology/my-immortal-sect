# ADR 0007: 采用 Hollywood Actor Framework 作为服务端基础

**状态**: 已确认 (Accepted)
**日期**: 2026-04-15
**决策者**: 项目负责人 + Codex
**相关文档**:
- `docs/vision/design-decisions.md` §4 技术栈与后端架构
- `AGENTS.md` §4 服务端权威与非妥协架构规则

---

## Context

SlgGame 是一款多人在线的修仙宗门经营小游戏，服务端需承载以下核心职责：

1. **服务端权威**（`AGENTS.md` §4）：所有关键状态（资源、弟子属性、因果触发、战斗结算）必须在服务端计算，客户端只做展示和预测
2. **Storylet 引擎**（`AGENTS.md` §4）：因果 + 神降共享同一套底层，支持触发条件、叙事节点、分支、结果、可种下新 storylet
3. **云端存档 + 增量同步**（`design-decisions.md` §4.3）：每个玩家的宗门状态、弟子、活跃 storylet、历史事件需要结构化持久化
4. **多人在线**（未来集群化）：初期单机，长期应支持横向扩展到多节点
5. **Codex 作为主力实现者**：代码结构必须对 AI 友好，模块边界硬约束（`docs/process/engineering-standards.md`）

服务端技术选型已约定为 Go（见 `docs/legacy/04_技术选型文档.md` 与 `design-decisions.md` §4.1），但需要决定:

- 从零造轮子，还是基于成熟框架？
- 如果用框架，哪一个最契合我们的设计哲学？

---

## Decision

**采用 [Hollywood Actor Framework](https://github.com/anthdm/hollywood) 作为服务端基础**，基于项目负责人 fork 的版本迭代。

所有业务逻辑以 actor 为基本单元组织，监督树（supervision tree）作为故障隔离与状态持久化的主结构。

---

## Hollywood 定位

Hollywood 不是游戏服务器全家桶，而是**纯 actor model 引擎**，设计血统接近：

- Proto.Actor（Go 分支）—— 但更精简
- Akka.NET —— 但没那么重
- Erlang OTP —— 概念同源

提供的核心原语：

- **PID + Actor**（通过 mailbox 串行化处理消息的隔离状态单元）
- **Send / Request / Respond / Forward**（消息传递四件套）
- **SpawnChild + Parent/Children**（监督树）
- **MaxRestarts / RestartDelay**（故障恢复）
- **Middleware**（持久化、metrics、日志横切）
- **EventStream**（发布订阅）
- **Cluster + Consul/mDNS 服务发现**（横向扩展）
- **Remote via dRPC + protobuf**（跨节点通信，节点间用，不供客户端）

数据佐证：

- 9 个直接依赖（轻量）
- 单机 350 万 msg/s 测试基准（10 engines × 2000 actors）
- Go 1.22+，MIT 许可
- 生产部署：Sensora IoT、Market Monkey Terminal
- README 明确把"game servers / ad brokers / trading engines"列为首选场景

---

## 契合度评估

Hollywood 原生能力与 SlgGame 架构决策逐项对齐：

| SlgGame 决策 | Hollywood 原语 | 契合度 |
|---|---|---|
| 服务端权威（状态隔离无并发） | Actor mailbox 天然串行化 | ★★★★★ |
| Storylet 引擎（状态机 + 消息驱动） | Actor 本质就是状态机 + mailbox | ★★★★★ |
| 弟子即因果契约（状态隔离） | 每个弟子 = 1 actor | ★★★★★ |
| 宗门 = 家（玩家会话持久） | Supervisor actor + 子 actor 树 | ★★★★★ |
| 云端存档 | WithPersistence middleware + 自定义 Storer 接口 | ★★★★☆ |
| 模块边界硬约束 | Actor 隔离是**物理硬边界**，不可越过 | ★★★★★ |
| 防作弊 / 幂等 | Request/Respond + 单线程处理 | ★★★★★ |
| 横向扩展（UGC 第二曲线） | 原生 cluster + Consul/mDNS 服务发现 | ★★★★★ |
| 配置驱动 / ID 永不修改 | 支持 `WithID(string)` 明确命名 | ★★★★ |

### 关键洞察：Storylet ≡ Actor

Storylet 引擎和 actor 模型是**同构的**——两者都是"状态 + mailbox + 处理消息 + 状态转移 + 副作用"。

因此 Storylet 引擎**不是一个独立模块**，而是**一组 actor 类型**。这让实现成本降低一个量级，并且天然符合"因果和神降共享同一引擎"的设计决策（`AGENTS.md` §4）。

---

## Consequences

### 正面影响

1. **故障隔离粒度远超手写架构**
   - 玩家离线 → 整棵子树 Poison + 状态持久化
   - 单个 storylet 崩溃 → 只重启它，其他不受影响
   - 单个弟子 bug → 只恢复该 actor，不影响宗门

2. **Codex 协作友好**
   - Actor 隔离 = 物理模块边界，AI 不可能"越界修改"其他实体状态
   - Actor 类型 ≈ 模块，天然满足"每个模块有 index.ts"的纪律
   - 每个 actor 独立测试，符合"每个模块必须有测试"

3. **省去 4~6 周自造 runtime 的工作**
   - Mailbox / 调度 / 监督树 / 重启策略 / dead letters 现成
   - 单机 350 万 msg/s 的性能储备远超我们需要

4. **平滑的扩展路径**
   - V1 单机部署
   - 后期需要分片 → 切换到 cluster mode 不需要重写业务代码
   - 匹配 `design-decisions.md` §9 "UGC 作为第二增长曲线"的长期规划

5. **与"服务端权威"物理契合**
   - Actor mailbox 保证对单个实体的操作被串行化
   - 业务逻辑中**不会有并发 bug**——不是"小心写锁"，是"没有机会写错"

### 负面影响与成本

1. **必须自建 WebSocket/HTTP Gateway**（约 1 周）
   - Hollywood 的 Remote 是 Go-to-Go dRPC，Cocos 客户端不能直接用
   - 需要 `GatewayActor` 负责客户端 ↔ actor 系统的协议翻译
   - 这是所有 actor 框架的常态，不是 Hollywood 的缺陷

2. **必须自建 PostgreSQL Storer**（约 1 周）
   - 上游 `examples/persistance` 是 JSON 文件 toy，不能用于生产
   - `Storer` 是接口，自建 `PGStore` 实现即可
   - 标准 CRUD，难度低

3. **必须定义完整 protobuf schema**
   - Actor 消息类型、客户端协议都必须 protobuf 化
   - 这本来就要做，不是 Hollywood 增加的负担

4. **Actor 模型学习曲线**
   - 对传统 HTTP handler 开发者有一定上手成本
   - 项目负责人已有 Hollywood 使用经验，成本已摊销
   - Codex 需要在 `AGENTS.md` 与工程规范中明确 actor 模式习惯

5. **上游文档浅，依赖源码阅读**
   - 源码量小（<5k LoC），质量高
   - `examples/trade-engine` 结构与我们服务接近，可作参考模板

### 中性影响

- 依赖树：9 个直接依赖（dRPC、vtprotobuf、consul、zeroconf、prometheus 等），远少于典型 Java/C# actor 框架
- Go 版本要求：≥ 1.22
- 许可：MIT，可商用

---

## Alternatives Considered

### A. 从零自建 Go 服务

- **优点**：完全掌控，无第三方依赖
- **缺点**：
  - Mailbox / 调度 / 监督树 / 重启策略 / dead letters 全部要自己实现
  - 估计 4~6 周工作量
  - Bug 风险高（这些基础设施代码"容易写错且难以发现"）
  - 自建版本很难比 Hollywood 更好
- **裁定**: ❌ 不采用。典型的"造轮子陷阱"

### B. Proto.Actor Go 分支

- **优点**：更成熟，社区更大，文档丰富，与 Akka 血缘更近
- **缺点**：
  - 更重，依赖更多，学习曲线更陡
  - 很多功能是我们不需要的（MMO 特化的区域管理等）
  - Go 分支维护频率不如 Hollywood
- **裁定**: ❌ 不采用。体量与我们不匹配

### C. Nakama

- **优点**：
  - "游戏服务器全家桶"
  - 自带账号、匹配、聊天、好友、存档
  - Lua/TS/Go runtime 都可扩展
- **缺点**：
  - 我们不需要 MMO 功能（排行榜、好友、组队）
  - 自带账号 / 存档系统与"配置驱动 + Storylet"架构不契合
  - 定制成本高，很多能力用不上
  - 大一个量级，维护负担重
- **裁定**: ❌ 不采用。过度工程化，架构不契合

### D. 手写 Gin/Echo HTTP 服务

- **优点**：生态最成熟，任何人都能上手，起步最快
- **缺点**：
  - 所有并发、状态隔离、故障恢复都要手写
  - Storylet 引擎在 HTTP handler 模型下非常别扭（每次读写都经过数据库）
  - 最终会自己长出一个劣化版 actor 模型
  - 和"服务端权威"原则物理上不契合
- **裁定**: ❌ 不采用。短期快，长期痛

---

## Risks

| 风险 | 严重度 | 缓解 |
|---|---|---|
| 上游停更 | 🟡 中 | 已 fork，可自行维护。定制全走 middleware / 自定义 actor，不改上游源码，merge 成本可控 |
| 社区较小 | 🟡 中 | 代码量小，源码即文档。项目负责人有经验，Codex 可读源码分析 |
| 文档浅 | 🟢 低 | README + 示例覆盖主要场景；`examples/trade-engine` 接近我们的架构 |
| Actor 模式学习曲线 | 🟢 低 | 项目负责人已有经验；Codex 只要边界清楚就能写 |
| Persistence 示例简陋 | 🟢 低 | 自建 PGStore 实现 Storer 接口即可 |
| 集成 HTTP/WS 需自建 | 🟢 低 | Gateway Actor 工作量约 1 周，标准工作 |

**结论**：无红线级风险。最严重的"上游停更"风险已通过 fork 对冲。

---

## 使用策略（必须遵守）

### 硬约束

1. **不修改 `actor/` 包源码**（除非修 bug 要 PR 回上游）
2. **不修改 `cluster/` 包源码**
3. **我们的代码放在独立目录** `server/internal/slggame/`
4. **持久化**：实现 `Storer` 接口，不改 middleware 源码
5. **我们的 protobuf 独立目录** `server/internal/proto/`，不与 Hollywood 的 protos 混
6. **Gateway 是独立 actor 类型**，不嵌入 Hollywood 内部

### 升级策略

- 定期 merge 上游 `master` 到 fork（建议月度）
- 版本锁在 `go.mod`，重要升级走专门 PR
- 上游 API 若有 breaking change，先评估影响再升级

### Fork 维护原则

- Fork 仓库**只做 bug fix 和必要的 PR backport**
- 业务定制一律不进 fork，全部在 `slggame/` 内通过 middleware / 自定义 actor 实现
- Fork 的目的是"对冲上游停更风险"，不是"基于它做二次开发"

---

## 架构初稿

```
Hollywood Engine (root)
└── GatewayActor (WebSocket/HTTP 网关)
    └── PlayerSupervisor (per online player, supervised tree)
        ├── SectActor              (宗门状态)
        ├── DiscipleActor × N      (弟子)
        ├── KarmaFrontendActor     (编年史前端状态)
        ├── DescentFrontendActor   (活跃神降会话)
        ├── ActiveStoryletActor × N (进行中因果/神降，短命)
        └── SyncActor              (增量同步消息聚合)

持久化层:
- PGStore (自建) 实现 Hollywood Storer 接口
- 挂到 WithPersistence middleware
- PlayerSupervisor 级别持久化整棵子树状态
- Redis 做热缓存 / 会话索引
```

架构细节将在未来的 `docs/architecture/server-structure.md` 中详细展开。

---

## 引用与证据

- **Hollywood 仓库**：https://github.com/anthdm/hollywood
- **源码阅读记录**（2026-04-15）：
  - `actor/engine.go` — Engine / Spawn / SpawnFunc / SpawnProc / Send / Request / Poison
  - `actor/context.go` — Context / Receiver / Respond / Forward / SpawnChild
  - `actor/opts.go` — MaxRestarts=3 / RestartDelay=500ms / InboxSize=1024 / Middleware
  - `examples/request/main.go` — 标准 Request/Respond 模式
  - `examples/persistance/main.go` — Storer 接口 + WithPersistence middleware（file store toy）
  - `examples/trade-engine/readme.md` — 与我们架构最接近的真实示例
  - `cluster/` — activation.go / agent.go / consul_provider.go / selfmanaged.go（集群化能力完备）
- **依赖清单**（`go.mod`）：
  - `storj.io/drpc` — 高性能 RPC
  - `github.com/planetscale/vtprotobuf` — 无反射 protobuf
  - `github.com/hashicorp/consul/api` — 生产级服务发现
  - `github.com/grandcat/zeroconf` — 本地开发 mDNS
  - `github.com/prometheus/client_golang` — metrics
  - `google.golang.org/grpc` / `protobuf` — 传输层
  - `github.com/DataDog/gostackparse` — 异常堆栈解析（supervision 用）

---

## 未决问题（后续 ADR）

- [ ] Gateway 协议选择：WebSocket / HTTP / 两者兼有？（倾向：主 WS + 关键操作 HTTP）
- [ ] Cluster 集群化的触发阈值：单机多少 DAU 后切换到分片？
- [ ] PGStore 的具体 schema 设计（另起 **ADR 0008 存档协议**）
- [ ] Actor ID 命名规范：玩家 PID / 弟子 PID 如何生成和持久化（另起 **ADR 0009 Actor ID 规范**）
- [ ] 是否需要为 Storylet 引擎引入代码生成工具（双端对称）
