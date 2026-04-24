# F-004-sect-map-m1-gameplay-foundation

> **状态说明**：superseded / historical
> 本文件记录 F-004 宗门地图 authority rebuild 历史。当前产品与后端主线已迁移到：
> - [docs/vision/gdd_v3_backend_design.md](/Users/mawei/MyWork/SlgGame/docs/vision/gdd_v3_backend_design.md)
> - [docs/decisions/0012-adopt-gdd-v3-authoritative-backend-gdd.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0012-adopt-gdd-v3-authoritative-backend-gdd.md)
>
> 本文件不得再作为 active implementation spec。

**状态**: accepted / historical baseline
**Area**: design / client / server
**Owner**: `supervisor`
**Reviewer**: `engineer`
**最后更新**: 2026-04-22

## Context

`F-004` 是宗门地图 M1 authority 基线文档，也是当前仓库里已经沉淀完成的一段历史玩法/运行时基线。

它承接了 `F-002` 留下的结论：

- 宗门地图已经证明自己是一个可用游戏表面
- 下一步不再是证明“能不能玩”
- 下一步是把《我的宗门》的第一条真实地图主循环，做成可扩展、可验收、可持续推进的主线

这条主循环当前仍然明确限定为：

**看图 -> 放建筑 -> 弟子自动工作 -> 资源积累 -> 扩张 / 升级 -> 第一次敌袭 -> 修复并继续经营**

## Read Policy

This is the main sect-map feature document, but it is still on-demand rather than universal startup reading.

Read it when:

- the active subfunction belongs to the sect-map M1 line
- `development.active.json` lists this file in `must_read`
- you need the current sect-map milestone boundary, acceptance language, or main product loop

Use it in a bounded way:

- prefer the sections relevant to the active subfunction
- use `docs/project/delivery-ledger.md` and `docs/project/decision-log.md` for detailed accepted history
- do not treat this file as a second full-project plan

## Current Role In The Project

`F-004` 仍然重要，但它不再定义整个项目的产品主线。

当前应按下面的层级理解它：

- `gdd_v3_backend_design.md`：当前产品闭环与后端实现主规范
- `development-plan.json`：从主规范导出的当前执行拆解
- `F-004`：宗门地图 authority baseline、薄客户端经验与历史验收边界

换句话说，`F-004` 现在服务于“宗门地图 authority 基线可复用”，而不是继续充当全项目的唯一玩法主线。

## Historical Goal

把宗门地图 M1 做成当前仓库中最可信的一条真实玩法主线：

- 玩家通过地图标记而不是单位直控来表达意图
- 建筑、资源、工作和敌袭都回收到同一张宗门图上
- 当前关键状态的最终接受态逐步从 client-local 迁入 authority path

## Thin-Client Runtime Responsibilities

在 authority-only runtime rebuild 下，客户端保留的正式职责只分为四类：

- render state：地图、建筑、弟子、敌人、HUD、提示、特效与可视反馈
- animation / interpolation state：根据 authority snapshot 做移动平滑、动作播放与表现层过渡
- debug state：bounded snapshot 对照、authority 连接/报错显示、reset / restore 调试入口
- player-input submission：把点击、长按、按钮等输入映射为允许的玩家意图命令或合同允许的 bounded fact

这四类职责都服务于“看见 authority truth、表达玩家意图、检查对齐状态”，而不是推进 gameplay truth 本身。

### Not gameplay truth

后续 `F-004` 的客户端不再允许直接决定：

- disciple assignment / haul / build / repair 下一步做什么
- building `planned -> supplied -> constructing -> active`
- damaged building repair closure
- `recover`、`second_cycle_ready`、`victory`、`defeat` 等阶段收口
- authority reject 之后继续用旧命令死循环重试

这些内容属于 authority snapshot 与 authority runtime progression。

## Product Definition

`M1` 当前只承载三根玩法支柱：

- **建**：放置、升级、拆除宗门建筑
- **运**：弟子自动采集、搬运、建造、修复
- **守**：轻量敌袭验证布局、应对与恢复

如果某个新需求不直接服务这三根支柱，它就不该默认进入当前里程碑。

## Current Scope

当前已经被接受并沉淀为基线的内容：

- `M1-A`：真实 Cocos 客户端中的 client-local `建 + 运` 基线
- `M1-B`：本地损伤/修复/可读性深化
- `M1-C`：client-local 可完成短会话
- `M1-D`：最小 authority-backed 短会话接轨

当前真正打开的工作，不是重新讨论这些是否成立，而是继续推进：

- `M1-F`：authority 资源刷新与多轮短会话收口
- 共享配置源收口：冻结 buildings / resources / phases 的第一份 shared config

## Historical Milestone Trail

`F-004` 关联过的 authority runtime rebuild 已经形成历史基线，主要包括：

- 冻结 authority-only runtime contract，停止继续维护 hybrid 主线
- 把 disciple assignment、build/repair completion、raid damage closure、post-raid continuity 全部收回 authority
- 把验收从“子功能 bounded 点测”提升到“authority-only 整链 replay + engineer/supervisor 双重独立验证”

如果你要确认当前真正的第一条应执行子功能，请读：

- `docs/project/development-plan.json`
- `docs/project/development.active.json`
- `docs/vision/gdd_v3_backend_design.md`

## Transitional Client Debt Vs Permanent Design

当前预览里仍然可能存在部分 client-side runtime glue，但必须明确区分：

### Permanent thin-client design

以下内容属于长期保留的客户端职责：

- render state
- animation / interpolation state
- debug state
- player-intent submission

### Transitional debt

以下内容如果仍然存在，只能作为 authority cutover 期间的过渡债务：

- 为保持 dedicated preview 可跑通而保留的本地 gather / dropoff 表现桥接
- authority reject 后用于清理陈旧表现态的本地恢复胶水
- 因 authority worker snapshot 字段尚未完全齐备而保留的临时镜像或兼容层
- 任何为了兼容旧 hybrid preview 命令而保留的 client-local fallback

这些债务不属于目标架构；后续子功能应继续把它们从正式主链中剥离。

## Shared Config Source

`F-004` 后续的 client / server 收口，不应继续在各自运行时里复制一份 buildings、resources、session phase timing 规则。

当前这条主线的第一份共享配置源固定为：

- `shared/configs/m1/sect_map_short_session.v1.json`
- schema: `shared/configs/m1/sect_map_short_session.schema.json`

当前约束：

- 该文件冻结 authority short-session 当前采用的 M1 建筑模板值、资源节点刷新规则与 phase-driving 数值
- client 现存本地副本只允许作为过渡生成物存在，不再允许承担独立规则演进
- server 现存 authority 常量应在后续实现环节逐步迁入该 shared source，而不是继续加新常量
- `shared/contracts/m1-authority-short-session-v1.md` 负责传输面，不再单独成为第二份规则表

## Acceptance Boundary

当前对 `F-004` 的正确表述必须保持克制：

- 它已经不再只是玩法草图
- 它也不再只是 client-local 演示
- 但它还没有完成 authority-only runtime cutover、正式协议、平台容器验证与后续宗门扩展

因此，任何结论都必须明确写出它属于哪一层：

- 历史 client-local 基线
- authority bridge / first authority closure
- authority-only runtime rebuild 历史基线
- 或更后续的 formal protocol、平台壳与宗门扩展

在当前里程碑下，以下内容都**不再**构成主玩法循环成立的充分证据：

- hybrid preview 路径看起来可以继续推进
- 某条 fallback 路径把主链补跑通
- 单个子功能在局部点测中通过，但没有 authority-only 整链 replay
- 只证明某个 authority 命令能返回成功，而没有证明 preview / authority 在主链 checkpoint 上持续对齐

## Non-Goals

当前不默认进入 `F-004` 的内容包括：

- 无缝大世界
- 完整 RimWorld 式需求系统
- 手操即时战斗
- 多势力外交和复杂关系网
- 多人同步经营
- 完整 Storylet / Karma 纵深
- 平台宿主验证优先于当前 authority 主线
- 为了“先跑起来”而长期把关键结算留在客户端做权威

## Implementation Constraints

- 客户端真实路径：`Cocos Creator + TypeScript + Tiled / Cocos TiledMap`
- 服务端真实路径：`Go + Hollywood`
- 当前项目不接受把 HTML-only prototype 当成主线完成态
- 当前浏览器验证只承认专用 Chrome `127.0.0.1:9333`
- 当前默认验证顺序仍然是：先 `Cocos Creator` 编译/预览链路，再决定是否切到平台宿主

## Authority-Only Mainline Validation Flow

当前 `F-004` 的主线验收不再接受 hybrid 成功路径。后续主管验收必须复用专用 Chrome 中的既有预览页与运行时调试桥，按固定 checkpoint 读取最小字段，并确认整条主链是在 authority-only 条件下推进，而不是被 client-local fallback 偷偷接管。

如果某轮验证只能证明“hybrid 还能跑”或“fallback 兜底后能跑通”，该轮必须判定为未通过，而不是弱接受。

### Fixed surfaces

- 专用 Chrome target：`http://127.0.0.1:9333`
- 复用既有预览页：`http://localhost:7456/`
- 本地 authority 入口：`go run ./cmd/gameserver`
- 运行时读取面：`window.__MIS_RUNTIME_DEBUG__.getSnapshot()`
- authority snapshot 面：`GET /v1/authority/m1/session/snapshot?sessionId=<id>`

### Required replay A: reset mainline

1. bootstrap
   - 预期：`session.phase=clear_ruin`
   - 预期：`authority.connected=true`
   - 预期：`authority.lastError=null`
   - 预期：preview / authority snapshot 的 `phase`、关键 `building`、`stockpile` 对齐
2. clear ruin
   - 预期：authority 推进到 `place_guard_tower`
   - 预期：`authority.lastEvent=build.demolished`
3. build guard tower
   - 预期：护山台在 authority 下完成 `planned -> supplied/constructing -> active`
   - 预期：`session.phase=upgrade_guard_tower`
4. upgrade guard tower to Lv.2
   - 预期：护山台在 authority 下完成升级并进入 `level=2`
   - 预期：`session.phase=raid_countdown`
5. trigger first raid
   - 预期：`session.phase=defend`
   - 预期：`session.firstRaidTriggered=true`
   - 预期：`authority.lastEvent=session.raid_triggered`
6. resolve defend into recover
   - 预期：`session.phase=recover`
   - 预期：`session.firstRaidResolved=true`
   - 预期：`authority.lastEvent=session.raid_resolved`
7. close the damaged-raid repair step
   - 预期：`session.phase=recover`
   - 预期：`session.outcome=in_progress`
   - 预期：`session.firstRaidResolved=true`
   - 预期：`authority.lastError=null`
   - 预期：不能出现 client 连续重试旧命令、也不能出现 preview / authority 对 damaged 状态不一致
8. continue into the next management cycle
   - 观察 authority snapshot / preview snapshot 直到至少一个已采空节点恢复可用
   - 再执行一次 post-raid `collect_stockpile`
   - 预期：`session.outcome` 仍为 `in_progress`
   - 预期：`stockpile` 与选定 `resourceNodes[*]` 字段在 preview / authority 快照中一致
   - 预期：不会出现 client-only reset、client-local fallback、或 authority reject 死循环

### Required replay B: restore_latest continuity

- 使用 `restore_latest` 重进同一 authority session
- 预期：preview / authority snapshot 在关键字段上与恢复前一致
- 预期：恢复后主链仍可继续推进，不会退回 client-local 兜底
- 预期：`authority.lastError=null`

### Required replay C: rejection recovery

- 至少保留一条 authority reject 恢复验证
- 预期：client 清理本地表现态、拉取最新 snapshot、主链继续推进
- 预期：reject 不会形成无限重试

### Secondary spot-check: authority-driven defeat

- 从新 bootstrap 开始，不需要重跑整轮守御
- authority command: `expire_session`
- 预期：`session.phase=defeat`
- 预期：`session.outcome=defeat`
- 预期：`authority.lastEvent=session.expired`

## Authority-Only Acceptance Evidence Boundary

主管更新 `F-004` 或 `docs/project/delivery-ledger.md` 时，后续轮次只需要保留以下有界证据：

- 浏览器面：说明复用的 dedicated preview tab 是 `http://localhost:7456/`
- authority 面：`authority.mode`、`authority.connected`、`authority.sessionId`、`authority.lastEvent`、`authority.lastError`、`authority.pendingCommands`
- session 面：`session.phase`、`session.outcome`、`session.objective`、`session.firstRaidTriggered`、`session.firstRaidResolved`
- building / worker 面：至少一个关键 `buildings[*].state/level/hp` 与当前 worker / disciple assignment 对照
- resource / stockpile 面：`stockpile` 与至少一个关键资源节点的 `state`、`remainingCharges`、`regenTimerSeconds`
- 结论面：
  - `reset` mainline replay 已跑通
  - `restore_latest` replay 已跑通
  - 如当前任务涉及 authority reject，rejection recovery replay 已跑通
  - preview / authority bounded snapshots 在这些 checkpoint 上保持一致
  - 主链上没有 `authority.lastError != null`

以下情况必须直接判定为 fail-closed，而不是写成“局部通过”或“仍可接受”：

- 任一主链 checkpoint 出现 authority / preview drift
- 同一 authority reject 在主链上重复触发
- client-local fallback 参与正式主链推进
- 需要刷新页面、重置会话或切回本地兜底才能继续
- replay 证据只能证明 hybrid success，不能证明 authority-only mainline

当前轮次不需要纳入 acceptance note 的内容：

- 全量 runtime logs
- 全量 network traces
- 弟子逐帧移动或 hostile 路径细节
- 微信 / 抖音小游戏容器结论
- protobuf、正式网关协议或平台壳扩写的细节实现

## Historical Summary

详细 loop history、逐轮 acceptance note、运行时证据和已接受产物，已经主要沉淀在：

- `docs/project/delivery-ledger.md`
- `docs/project/decision-log.md`

本文件不再重复维护完整长历史。

## Current Open Questions

- disciple assignment、work progress、raid damage 这些 authority-owned worker fields 用什么最小 snapshot shape 暴露给 thin client
- 哪些当前命令应该保留为玩家意图，哪些必须降级为 bounded fact，哪些应该彻底从 client progression 中移除
- authority-only 主链稳定到什么程度后，再值得切入正式 protobuf gateway 与小游戏宿主 smoke

## Current Working Set

执行 `F-004` 相关任务时，通常只需要按需进入以下文件：

- `AGENTS.md`
- `docs/vision/design-decisions.md`
- `docs/project/development-plan.json`
- `docs/project/development.active.json`
- `docs/project/decision-log.md`
- `docs/project/delivery-ledger.md`
- `shared/contracts/m1-authority-short-session-v1.md`

其他文档只有在当前 subfunction 的 `must_read` 明确要求时再进入。

## Related Docs

- `docs/features/F-002-sect-map-playability-validation.md`
- `docs/plans/m0-vertical-slice.md`
- `docs/project/development-plan.json`
- `docs/project/development.active.json`
- `docs/project/decision-log.md`
- `docs/project/delivery-ledger.md`
- `shared/contracts/m1-authority-short-session-v1.md`
