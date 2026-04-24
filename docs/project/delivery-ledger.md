# Delivery Ledger

Record only accepted work here.

## Current Release Or Milestone

- Name: `GDD v3.1 Authority Core Bootstrap`
- Goal: 以 GDD v3.1 为唯一主规范，建立 protobuf-first、snapshot-first、SectActor big-state 的最小权威后端骨架，并让后续资源、建筑、弟子、任务、贡献与事件系统都落在同一套命令、事件、存档与同步路径上。
- Acceptance owner: `supervisor`

### Current Acceptance Boundary

- 文档、ADR、计划、激活指针之间不得再出现并行真相。
- `ClientCommand / CommandResult / StatePatch`、`SectState`、`DomainEvent / ApplyEvent`、`sect_snapshots / sect_events / command_log` 必须都沿同一条 GDD v3.1 路径定义。
- Fail closed when：仍保留 JSON 权威协议、仍保留共享房间式 V1 同步表述、仍把 `DiscipleActor / BuildingActor / StoryletActor` 当作 V1 默认 actor 粒度、或仍允许 `F-004` / sect-map 历史文档作为 active 主线。
- 当前里程碑的实现验收以 `development-plan.json` 的 Phase A 子功能为准；本台账只保留 accepted 历史，不再把旧 sect-map authority rebuild 继续当作当前 release 说明。

## Accepted Deliveries

以下记录保留为已接受历史，尤其包含一段完整的 sect-map authority rebuild 轨迹；它们不再定义当前产品主线，只作为可追溯的基线证据。

| Date | Scope | Evidence | Accepted By |
| --- | --- | --- | --- |
| 2026-04-20 | `M1-A` client-local `建 + 运` 偏验收基线 | `F-002` 与 `F-004` 已回写代码审计结论：accepted 为地图契约快照、有限资源节点、`gather -> haul -> build`、资源入库、可见状态变化与最小可读反馈；`safeArea` 原生证据保持 `partial`，`Go + Hollywood` / `shared` 权威路径保持 `missing` | `supervisor` |
| 2026-04-20 | `M1-B` 本地通用单位属性与损伤/修复闭环 | `F-004` 已回写真实 Cocos 客户端中的共享单位模型、建筑 HP / `damaged` / `repairCost`、外敌边缘刷新、`guard_tower` 自动守御、弟子 `guard / repair`、修复缺口优先采集与 `http://localhost:7456/` 预览页验证结果；该接受态仅覆盖 client-local 玩法深化，不外推为 `Go + Hollywood` / `shared` 权威闭环 | `supervisor` |
| 2026-04-20 | `M1-B` 高可读地图资源包 | `F-004` 已冻结 `client/my-immortal-sect/assets/resources/generated-buildings/sect-map-svg/` 下的 canonical SVG 资源包：建筑 5 个、资源 3 个、弟子四态头像 4 个，并记录命名、尺寸、锚点与导出合同；该接受态仅覆盖资源包与规格交付，不外推为运行时贴图接入已完成 | `supervisor` |
| 2026-04-20 | `M1-B` 资源接图与关键占位刷新 | `F-004` 已回写 `sect-map-svg -> sect-map-raster -> SpriteFrame` 的稳定消费路径、12 个 runtime PNG 贴图、弟子/建筑/资源关键占位替换，以及 `http://localhost:7456/` 预览页下 `build -> haul -> guard -> repair` 本地闭环未被破坏的验证结果；该接受态仅覆盖关键地图表现刷新，不外推为整轮体验验证已完成 | `supervisor` |
| 2026-04-20 | `M1-B` 体验刷新与可玩验证 | `F-004` 已回写贴图版地图在 `http://localhost:7456/` 预览页上的短流程回归、建筑标签遮挡收口、默认盘/注入盘验证与剩余体验 blocker 清单；该接受态仅覆盖体验刷新后的回归验证与问题收敛，不外推为所有体验问题、敌方视觉刷新或服务端权威路径已完成 | `supervisor` |
| 2026-04-21 | `M1-C` 本地短会话核心循环 | `F-004` 已回写 `place / upgrade / demolish`、首波敌袭、repair 恢复、短会话阶段机与 `victory / defeat` 收口；专用 Chrome `http://localhost:7456/` 预览页已跑通一轮新档 `victory` 短会话，约 `82.4s`，阶段推进为 `clear_ruin -> place_guard_tower -> upgrade_guard_tower -> raid_countdown -> defend -> recover -> victory`；该接受态仅覆盖 client-local 短会话核心循环，不外推为整个 `M1-C`、小游戏容器 smoke 或服务端权威路径已完成 | `supervisor` |
| 2026-04-21 | `M1-C` 威胁识别与状态信号资源 | `F-004` 已回写 `bandit_scout_normal / injured` hostile 头像与 `building_signal_planned / supplied / constructing / damaged / disabled` 远距状态信号的 canonical SVG + runtime PNG/SpriteFrame 资源合同；该接受态仅覆盖资源与规格交付，不外推为 live runtime 接图、小游戏容器 smoke 或服务端权威路径已完成 | `supervisor` |
| 2026-04-21 | `M1-C` 目标指向收口与宿主 smoke 结论 | `F-004` 已回写导向式 HUD、objective marker、建筑状态徽记与 hostile 头像贴图进入当前 runtime，并在专用 Chrome `http://localhost:7456/` 预览页确认短会话仍可跑通 `victory`；小游戏宿主 smoke 已推进到抖音开发者工具登录门槛，但受登录页与缺少抖音构建产物限制，当前结论保持 `partial`。该接受态仅覆盖目标指向收口与 blocker 明确化，不外推为宿主验证通过 | `supervisor` |
| 2026-04-21 | `M1-C` client-local 可完成短会话里程碑 | `F-004` 已由主管补记里程碑验收结论：贴图版宗门地图在真实 Cocos 客户端路径中，已具备从新档完成一轮本地短会话的门槛，包含 `place / upgrade / demolish`、首波敌袭、防守、修复恢复、目标导向与胜负收口；专用 Chrome `http://localhost:7456/` 预览页已有约 `82.4s` 的 `victory` 证据。该接受态仅表述为 client-local 里程碑通过，小游戏宿主 smoke 仍为 `partial`，`Go + Hollywood` / `shared` 权威路径仍未完成 | `supervisor` |
| 2026-04-21 | `M1-D` 最小 authority 短会话接轨 | `F-004` 已回写 `shared/contracts/m1-authority-short-session-v1.md`、`server/` 下的 Hollywood actor-backed authority session 与 HTTP gateway、客户端 authority snapshot/command 适配层，以及专用 Chrome 预览下 `clear_ruin -> ... -> raid_countdown` 的 authority 命令链验证；主管已确认该结果达到“最小 authority-backed 短会话”门槛，但不外推为完整敌袭/采集/守御链路已全部服务端权威化 | `supervisor` |
| 2026-04-21 | `M1-D` 单仓清理与统一提交推送 | 项目已按单仓策略清理 `client/my-immortal-sect/.git` 与 `client/my-immortal-sect/extensions/cocos-mcp-server/.git` 两个错误嵌套仓，统一归并到根仓 `~/MyWork/SlgGame/.git`，并已将提交 `dfecf42ef80da2b54ff77d64da71c57480d06246` 与 `db028d3789eaab74b4f4d791526a9fbabd037191` 推送到 `origin/main` | `supervisor` |
