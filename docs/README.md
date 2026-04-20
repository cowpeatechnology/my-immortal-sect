# My Immortal Sect 文档索引

这里是 My Immortal Sect（《我的宗门》）项目的**设计与架构文档中心**。所有设计决策、技术架构、实施计划与研发台账都在这里沉淀。

> 在任何开发活动开始前，请先确保你已读过项目根目录的 `AGENTS.md` 和本索引下标记为 ⭐ 的权威文档。

---

## 目录速览

```
docs/
├── README.md                    # 本文件
├── vision/                      # 顶层愿景与核心决策
├── design/                      # 游戏设计细节
├── architecture/                # 技术架构细节
├── decisions/                   # ADR（架构决策记录）
├── process/                     # 研发流程 / Git 工作流 / 主管调度规则
├── project/                     # 项目总台账（持续更新）
├── features/                    # 单功能循环文档
├── templates/                   # 派单 / 回传 / feature loop 模板
├── plans/                       # 实施计划
└── legacy/                      # 早期文档（仅供参考）
```

---

## 当前权威文档

### ⭐ `vision/design-decisions.md`

**这是整个项目的 North Star**。包含：

- 核心设计原则（弟子即因果 / 东方式宿命感 / 宗门是家）
- 系统架构（因果 + 神降共享 Storylet 引擎）
- 美术与表现层决策（大头 + Spine 仪式性资源）
- 技术栈与后端架构（Cocos + Go + 云端存档）
- Codex 协作纪律
- 风险与缓解
- M0 / M1 / M2 / M3 里程碑
- 长期规划（UGC 神降第二增长曲线）

**任何重大改动前必读。** 如果本文档与其他文档冲突，以这份为准。

---

## 待创建的关键文档

以下文档尚未创建，将在对应设计议题推进时补上。Codex 如果被要求执行相关任务，应先进入 Plan 模式或向用户确认。

### `design/systems/`
- `storylet-engine.md` — Storylet 引擎的数据结构、DSL、触发器、前端绑定方式
- `karma-system.md` — 因果前端的编年史 UI、预告机制、因果契约模板
- `divine-descent.md` — 神降前端，V1 最小形态，长期深度解锁路径
- `disciple-system.md` — 弟子实体、标签、成长、工作
- `building-system.md` — 建筑配置、升级、工作位（沿用 legacy `02`/`03` 基础）
- `resource-system.md` — 资源定义、生产链、库存、物流 Reservation
- `battle-symbols.md` — 符号化战斗语言清单（飘字 / buff / 特效规范）

### `design/content/`
- `storylet-dsl.md` — 事件编写规范，策划友好的表格格式
- `karma-chain-template.md` — 因果链写作模板
- `naming-conventions.md` — 配置表 ID 命名约定（已在 `process/engineering-standards.md` 中定义）

### `design/art-direction/`
- `big-head-visual-system.md` — 大头底板 / 发型 / 表情层规范 + 因果弟子独占标识指南
- `spine-usage-guide.md` — Spine 使用范围 + 规格 + 解锁节奏
- `battle-fx-library.md` — 20~30 个符号化特效清单

### `architecture/`
- `tech-stack.md` — 技术栈综述（继承并更新 legacy `04`）
- ✅ `client-structure.md` — Cocos 工程目录、模块划分、事件总线（已起草 2026-04-16）
- ✅ `server-structure.md` — Go 服务端模块、API 约定、数据库 schema（已起草 2026-04-16）
- `save-sync-protocol.md` — 云端存档结构、增量同步协议、离线时间处理
- `config-pipeline.md` — Excel → JSON → 双端类型生成
- `version-migration.md` — 配置与存档的版本迁移铁律
- ✅ `agent-team-operating-model.md` — 当前 Coordex 三角色可见线程体系：`supervisor / engineer / art_asset_producer` 的职责边界与协作规则（已更新 2026-04-19）

### `process/`
- ✅ `development-loop.md` — 标准研发循环：Plan → Design → Execute → Verify → Record
- ✅ `github-workflow.md` — GitHub issue / branch / commit / merge 约定
- ✅ `engineering-standards.md` — 项目级 Codex 工程规范：命名、禁令、TS / Go / commit 约束
- ✅ `cocos-mcp-workflow.md` — 项目内 Cocos MCP 的运行端口、编辑源与同步规则（已起草 2026-04-16）
- ✅ `dedicated-browser-workflow.md` — 浏览器相关任务的首选运行模式：专用 Chrome + 固定调试端口 + 会话复用（已确认 2026-04-17）
- ✅ `thread-conversation-protocol.md` — 多个可见角色线程之间的共享对话协议：单 owner 子功能、激活后受限直连、结构化消息与 ledger 留痕（已更新 2026-04-19）

### `project/`
- ✅ `delivery-ledger.md` — 项目总台账：当前里程碑、活跃循环、已实现 / 未实现 / 暂缓功能、主要冲突
- ✅ `thread-conversation-ledger.md` — 项目内线程对话账本：不同角色线程之间的 request / handoff / blocker / status 记录（已起草 2026-04-18）

### `features/`
- `F-xxx-<slug>.md` — 单功能循环文档，记录每轮计划、设计、执行、验收、冲突和当前状态

### `templates/`
- ✅ `supervisor-work-order-template.md` — 主管派单模板
- ✅ `worker-handoff-template.md` — worker 回传模板
- ✅ `feature-loop-template.md` — 单功能循环文档模板
- ✅ `thread-message-template.md` — 线程对话消息模板

### `decisions/`（ADR，每个决策一个文件）
- `0001-big-head-over-spine.md` — 为什么用大头替代全身 Spine
- `0002-shared-storylet-engine.md` — 为什么因果 + 神降共享引擎
- `0003-go-backend.md` — 为什么选 Go 作为服务端
- `0004-cloud-save.md` — 为什么走云端存档而非本地
- `0005-server-authoritative.md` — 为什么服务端权威
- `0006-rimworld-inspiration.md` — 为什么 RimWorld 是灵感来源而不是《最强祖师》
- ✅ `0007-hollywood-actor-framework.md` — 为什么采用 Hollywood Actor Framework 作为服务端基础（已确认 2026-04-15）
- ✅ `0008-save-protocol-and-pgstore-schema.md` — 云存档协议与 PGStore Schema：单行 protobuf blob + 事件表 + 三版本治理（已确认 2026-04-15）
- ✅ `0009-actor-id-conventions.md` — Actor ID / PID 命名与持久化规范：三套 ID 体系（业务 / PID / 配置）的格式、生成、恢复规则（已确认 2026-04-15）
- ✅ `0010-offline-deterministic-simulation.md` — 离线补偿模拟：big-State + 纯函数 `advance()` + 四条工程纪律（禁 wall clock / 每玩家 PRNG / 无 I/O / 单 SimulationActor）。rev 2 已撤销 rev 1 的"字节级确定性"过度约束（已确认 2026-04-15, rev 2）

### `plans/`
- ✅ `m0-vertical-slice.md` — M0 技术垂直切片的具体任务拆解：4~5 周按天任务清单（已确认 2026-04-15）
- `m1-art-pipeline.md` — 美术管线验证
- `m2-storylet-content.md` — Storylet 内容流水线
- `m3-differentiation.md` — 差异化一句话

---

## Legacy 文档

`legacy/` 下的 5 份文档是项目早期设计思考，创建于 2026 年 4 月上旬。**部分结论已被后来的讨论修正**（尤其是神降 / 战斗 / 存档 / 因果）。

| 文件 | 状态 | 有效部分 |
|---|---|---|
| `01_修仙宗门经营要素与完整度审视.md` | 部分有效 | 资源 / 建筑 / 岗位清单仍是 V1 基础 |
| `02_完整开发计划.md` | 部分有效 | ID 命名规范已吸收到 `process/engineering-standards.md`；核心循环仍成立 |
| `03_第一版本开发计划.md` | 部分有效 | V1 范围清单仍可用；神降定位需重写 |
| `04_技术选型文档.md` | 大部分有效 | Cocos / Tiled / Spine / ESEngine / Go 已确认 |
| `05_第一版本验收标准与测试清单.md` | 部分有效 | 验收清单需补充 Storylet / 因果 / 大头相关项 |

**修改任何系统前，如 legacy 与 vision 冲突，以 vision 为准。**

---

## 文档写作约定

新增文档时请遵守：

1. **一个文档一个主题**：不要一个文档讨论多个互不相关的系统
2. **开头必有 Context 段**：说明这份文档为什么存在
3. **明确状态**：在头部标注 `状态: 草案 / 评审中 / 已确认 / 已过时`
4. **交叉引用**：链接到其他相关文档，保持导航网
5. **可执行**：包含足够细节让 Codex 或人能据此实现
6. **不写废话**：删去"根据多年经验""众所周知"等无信息量的句子

## 研发记录约定

从 2026-04-16 起，项目默认按“循环式研发”记账。

- 非 trivial 的任务，要先有 work order
- 每个持续迭代的功能，要维护一份 `docs/features/F-xxx-<slug>.md`
- 每轮结束，要回写 `docs/project/delivery-ledger.md`
- 重大且难以反转的技术决策，进入 `docs/decisions/ADR-xxxx-*.md`

模板骨架：

```markdown
# <标题>

**状态**: 草案 / 评审中 / 已确认
**最后更新**: YYYY-MM-DD
**依赖**: `vision/design-decisions.md` §X.Y

## Context
为什么这份文档存在，它解决什么问题。

## <核心章节>
...

## 相关文档
- ...

## 未决问题
- [ ] ...
```
