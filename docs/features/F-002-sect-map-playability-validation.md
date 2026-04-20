# F-002-sect-map-playability-validation

**状态**: active  
**Area**: client / server / design / qa  
**Owner**: `supervisor`  
**Reviewer**: `human`  
**最后更新**: 2026-04-20

## Context

The project’s long-term vision is a xianxia sect-management sim. Before broader M0 work can be justified, the big sect map itself needs to prove that it is actually usable as a game surface.

This feature exists to validate the map as a playable system:

- readable at first glance
- interactable through blueprint placement
- able to communicate disciple work status
- able to support a short moment-to-moment loop

Important clarification after supervisor review:

- `hifi-prototype/` work is useful exploratory evidence, but it does not close this feature.
- This feature is only accepted when the sect-map slice exists in the real project stack.
- For this repository, "really playable" means the documented stack:
  - client: `Cocos Creator + TypeScript + Tiled / Cocos TiledMap`
  - server: `Go + Hollywood`, with real runtime boundaries where applicable

## Goal

Build and verify a sect-map slice in the real project stack that a tester can understand and interact with without needing to mentally fill in missing structure.

## Non-Goals

- Full M0 actor / persistence / offline compensation
- Full storylet / karma system
- Full economy depth
- Full content library of buildings or disciples
- Final art polish that sacrifices readability
- HTML-only prototype delivery standing in for the real client/server slice

## Authority Docs

- `AGENTS.md`
- `docs/process/development-loop.md`
- `docs/process/engineering-standards.md`
- `docs/architecture/agent-team-operating-model.md`
- `docs/plans/phase-1-sect-map-validation.md`
- `docs/plans/m0-vertical-slice.md`

## Acceptance Criteria

- [x] The map’s main structures and play areas are legible within a short first look
- [ ] Blueprint placement shows footprint, validity, and orientation clearly
- [x] Disciple task visibility makes current work state obvious during play
- [x] A player can perform a short interaction loop and see an immediate result on the map
- [ ] No HUD element obscures the map to the point that play becomes ambiguous
- [x] The map remains understandable during repeated interactions, not only on the first frame
- [ ] Phase 1 can be judged playable by the supervisor or human acceptance path without relying on implementation context
- [ ] The slice runs on the real project stack: Cocos Creator + TypeScript client, Go + Hollywood-backed runtime path where applicable
- [x] The main map uses the intended client map tech path, centered on Tiled / Cocos TiledMap rather than HTML canvas mockup

## Current Decision Summary

- Phase 1 is a validation slice that precedes broader M0 expansion
- Readability and playability outrank scenic illustration if they conflict
- Blueprint and disciple feedback must live on the map surface, not only in hidden logs
- Prototype exploration may inform the slice, but cannot replace the real-stack milestone

## 2026-04-20 Audit Snapshot

本轮基于代码审计对 `F-002` 做了偏验收收口。

当前已可接受为 **done** 的，是宗门地图在真实 Cocos 客户端路径中的最小可玩性基线：

- `TiledMap` 驱动的真实主盘，而不是 HTML mockup
- 地图契约快照、有限资源节点规则与可见 lot/logic 分层
- 蓝图落位、合法性校验与 `planned -> supplied -> constructing -> active` 的 client-local 建造闭环
- 弟子 token、采集/搬运/施工任务与不依赖隐藏日志的地图反馈
- 基于 tile 的寻路、资源入库、工地供料与地图状态变化

当前仍保持 **open** 的项：

- 更细的蓝图方向/rotation 表达
- 不依赖实现上下文的主管或人类短运行再验收
- `Go + Hollywood` 权威接入与 `shared` 合同源

结论：`F-002` 现在已经不再是“只有建筑预览的早期尝试”，但也不能被表述成“真实前后端权威切片已通过”。

## Conflict And Impact

- 冲突对象：把 prototype 当 milestone 完成态
  - 冲突原因：会误导主管和 worker，把真实前后端切片错判为已完成
  - 当前裁决：`hifi-prototype/` 只保留为探索资产，不作为 Phase 1 通过依据
  - 后续动作：后续 Phase 1 验收必须依赖真实客户端与服务端切片

- 冲突对象：纯 scenic / illustration-first map direction
  - 冲突原因：过度追求画面美感会降低地图读图性和操作确认度
  - 当前裁决：Phase 1 优先可玩性，必要时牺牲部分装饰密度
  - 后续动作：在真实切片通过后再考虑更高层次的美术强化

## Implementation Status

### Done

- [x] Phase 1 的目标与验收方向已定义
- [x] Phase 1 与 broader M0 的边界已明确
- [x] 已完成一次 `hifi-prototype/` 探索性地图闭环验证
- [x] 已确认主管与 worker 的交接需要更严格的 runtime handoff 结构
- [x] 已为真实运行栈建立初始仓库骨架：`server/`、`shared/` 已落盘，`client/` 已预留为干净 Cocos 根目录
- [x] 已确认真实 Cocos 项目路径 `client/my-immortal-sect/` 与 MCP 端口 `9527`
- [x] 已完成 Cocos MCP live smoke test：读取项目、读取场景、创建临时节点并删除
- [x] 已创建真实客户端入口场景 `db://assets/scenes/sect-map-main.scene`
- [x] 已创建最小客户端入口脚本 `db://assets/scripts/app/sect-map-bootstrap.ts`
- [x] 已建立最小场景层级：`Canvas` / `MainCamera` / `AppRoot` / `MapRoot` / `LotRoot` / `DiscipleRoot` / `OverlayRoot`
- [x] 已补入最小 isometric `TiledMap` 资源：`assets/resources/tilemaps/sect-map/*`
- [x] 已补入 3 张当前可用的建筑资源：`main-hall` / `alchemy-room` / `herb-garden`
- [x] 已把入口脚本收敛为“可拖拽地图 + 点击选中地块”的最小交互闭环
- [x] 已按官方 `TiledMap -> Tmx Asset` 路径把 `GroundMap + cc.TiledMap` 配回场景，并恢复 Canvas / camera 的可见性约束
- [x] 已确认当前 Cocos 预览工作流可简化为“手动首次点击播放 -> 常驻 `http://localhost:7456` 页面 -> 后续保存自动刷新”
- [x] 已把当前 Cocos 编辑器 / 预览常见问题追加记录到 `docs/process/cocos-mcp-workflow.md`
- [x] 已修正等距 TileMap 选中光标的坐标锚点与渲染顺序，黄色选中框现已覆盖单个目标地块，而不是落在四格交点
- [x] 已从现有透明建筑资源中选用 `alchemy-room`，并完成“点击地块 -> 在该格绘制缩放后的建筑预览”最小闭环
- [x] 已把 `main-hall` 生成严格 `128x64` 的派生透明图片，并按“与黄色高亮相同的 overlay 定位方式”稳定绘制到目标地块
- [x] 已为 `sect-map` runtime 接入结构化日志、预览页 debug bridge 与 MCP 增量 tail，形成“截图看界面 + 日志看输入/状态迁移”的联调闭环
- [x] 已在真实客户端路径中形成 `Map contract snapshot + finite resource nodes + build + haul + visible state change` 的最小可玩性基线
- [x] 已确认当前地图可读性不再依赖隐藏日志：HUD、建筑标签、资源标签与弟子角标已构成最小解释链

### Exploratory Evidence

- [x] HTML prototype 证明了“大地图可玩”至少需要地块选择、蓝图落位、施工链、弟子可见任务状态
- [x] HTML prototype 暴露了真实 UX 问题：中部信息面板会压住可交互 lot
- [x] prototype 的交互结论可以转译为真实 client/server 切片输入

### Not Yet Done

- [x] `client/` 已真实建立并进入 Cocos Creator + TypeScript 开发
- [ ] `server/` 真实建立并进入 Go + Hollywood 开发
- [x] tilemap / TiledMap 路径已进入第一版实现
- [ ] 至少一条前后端联动的大地图命令路径
- [ ] 基于真实运行栈的主管验收
- [ ] 蓝图 orientation / rotation 的更细表达

### Deferred

- [ ] 施工取消 / 拆除 / 重排队列
- [ ] 更细的 blueprint footprint / rotation 表达
- [ ] 更长时段的平衡性验证
- [ ] 完整 M0 技术切片中的 actor / persistence / offline 相关内容

## Loop History

| Loop | Date | Stage | Summary | Output | Decision |
|---|---|---|---|---|---|
| L-001 | 2026-04-16 | Plan | 定义 Phase 1 为大地图可玩性验证切片 | `docs/plans/phase-1-sect-map-validation.md` | continue |
| L-002 | 2026-04-16 | Explore | 在 `hifi-prototype` 中实现地图交互闭环与动态反馈 | `home-immersive.html` / `immersive-loop-phase1.js` / `styles.css` | continue |
| L-003 | 2026-04-16 | Review | 用户澄清“可玩大地图”必须是 Cocos + Go 真实切片，不是 HTML prototype | 本文档 + delivery ledger | reopen |
| L-004 | 2026-04-16 | Bootstrap | 建立真实代码目录骨架，避免后续 Cocos / Go 实现再反复改路径 | `server/` / `shared/` / `docs/architecture/*-structure.md` | continue |
| L-005 | 2026-04-16 | Runtime Check | 确认真实 Cocos 项目、MCP 端口与基础编辑能力已打通 | `client/my-immortal-sect/` / `client/my-immortal-sect/extensions/cocos-mcp-server/` | continue |
| L-006 | 2026-04-16 | Client Bootstrap | 创建真实入口场景、入口脚本与最小根层级 | `assets/scenes/sect-map-main.scene` / `assets/scripts/app/sect-map-bootstrap.ts` | continue |
| L-007 | 2026-04-16 | Tilemap Slice | 接入 isometric TiledMap、真实建筑资源、拖拽地图和长按落建筑的最小玩法闭环 | `assets/resources/tilemaps/sect-map/*` / `assets/resources/generated-buildings/*` / `sect-map-bootstrap.ts` | continue |
| L-008 | 2026-04-16 | Scene Recovery | 把 tilemap 从“纯运行时挂载”改回编辑器可见的静态场景结构，并修正竖版画布与 camera visibility | `assets/scenes/sect-map-main.scene` / `assets/scripts/app/sect-map-bootstrap.ts` | continue |
| L-009 | 2026-04-17 | Preview Workflow Simplification | 确认预览页 `http://localhost:7456` 在首次播放后可常驻，后续通过“修改并保存 -> 页面自动刷新”完成观察闭环 | 本文档 + delivery ledger | continue |
| L-010 | 2026-04-17 | TileMap Recovery | 记录编辑器 / 预览坑位，按官方 `TiledMap` 路径恢复 `GroundMap.tmxAsset`，并验证“拖拽地图 + 点击选中地块”最小闭环 | `sect-map-main.scene` / `sect-map-bootstrap.ts` / `docs/process/cocos-mcp-workflow.md` | continue |
| L-011 | 2026-04-17 | Selection Cursor Alignment | 改用 `TiledLayer.getPositionAt()` 作为等距地块定位基准，并把 `OverlayRoot` 调整到 `GroundMap` 之上，修正选中框错位与被遮挡问题 | `sect-map-bootstrap.ts` / `docs/process/cocos-mcp-workflow.md` | continue |
| L-012 | 2026-04-17 | Building Preview Placement | 选用透明 `alchemy-room` 资源作为当前最适合的最小建筑预览，并完成“点击地块 -> 缩放后建筑落到对应地块”的运行时验证 | `sect-map-bootstrap.ts` | continue |
| L-013 | 2026-04-17 | 128x64 Tile Image Placement | 将 `main-hall` 压成严格 `128x64` 且保持原非透明内容宽高比不失真，并按已验证的 overlay 定位方式绘制到被点击地块 | `main-hall-tile-128x64.png` / `sect-map-bootstrap.ts` | continue |
| L-014 | 2026-04-17 | Runtime Observability | 为 `sect-map` runtime 增加结构化日志、预览页 `window.__MIS_RUNTIME_DEBUG__` 与 MCP `get_project_logs_incremental`，把 Chrome 画面观察与事件时间线联动起来 | `sect-map-bootstrap.ts` / `debug-tools.ts` / `docs/process/cocos-mcp-workflow.md` | continue |
| L-015 | 2026-04-20 | Audit / Record | 基于代码审计收敛 `F-002` 的偏验收边界，确认真实 Cocos 客户端中的 `建 + 运` 最小可玩性基线已具备证据，但 `Go + Hollywood` / `shared` 权威路径仍未接入 | 本文档 / `docs/features/F-004-sect-map-m1-gameplay-foundation.md` / `docs/project/delivery-ledger.md` | continue |

## Acceptance Summary

- 已完成的验收不再只覆盖 HTML prototype；当前已确认真实 Cocos 客户端路径中的最小 `建 + 运` 可玩性基线成立
- 该结论仍不构成 `F-002` 完整通过，因为真实服务端 / `shared` 权威路径尚未接入
- prototype 中暴露的交互与信息密度问题，后续应作为真实实现时的重点回归项
- 当前客户端最小验证入口应以常驻预览页 `http://localhost:7456` 为准，而不是依赖 MCP 触发“播放”动作
- 当前默认验证闭环：
  - 项目负责人在 Cocos 中手动首次点击播放
  - 浏览器保留 `http://localhost:7456`
  - worker 修改当前场景或脚本并保存
  - 等待约 1 秒后在该页面观察自动刷新结果
  - 截图用于判断 `TileMap`、选中框、HUD、建筑预览、弟子 token 等可见结果
  - 日志用于判断 `mode.change`、`pointer.tap_tile`、`task.assigned`、`task.phase_change`、`task.completed`、`build.state_change`、`resource.stockpile_*` 等事件序列
  - 首选通过 MCP `get_project_logs_incremental` 只读取新增日志；如需在固定 Chrome 里即时看最近状态，可在 DevTools 调用 `window.__MIS_RUNTIME_DEBUG__.getRecentLogs()` 与 `window.__MIS_RUNTIME_DEBUG__.getSnapshot()`
  - 如果发生“编辑器当前场景已更新但浏览器仍残留旧运行态”的异常，仅 reload 现有页，不新开 tab
  - 点击地块后，黄色选中框应包住单个菱形地块；如果光标落在四格交点，优先检查 `TiledLayer.getPositionAt()` 的中心换算和 `OverlayRoot` sibling 顺序
  - 当前最小建筑验证使用 `resources/generated-buildings/alchemy-room.png`；点击任意有效地块后，建筑预览应跟随切换到该地块
  - 当前偏验收已知可接受项包括：地图契约快照、有限资源节点、弟子 `gather/haul/build` 闭环、资源入库与可见状态变化
  - 当前偏验收仍保留为 open 的项包括：蓝图 orientation 表达、主管短运行再验收、`Go + Hollywood` / `shared` 权威路径

## Open Questions

- [ ] blueprint orientation / rotation 的最小表达形式是什么
- [ ] 第一条真实前后端命令路径选“蓝图落位”还是“弟子派工”更合适
- [ ] 主管是否要在专用 Chrome 上补一次短运行再验收，以把当前代码审计结论升级为更强的 acceptance evidence
- [x] 当前场景恢复后，Cocos 预览已恢复到“可见 TileMap + 可拖拽 + 点击选中”状态
- [x] 当前阶段不需要通过 MCP 触发播放；`localhost:7456` 常驻页已足够支撑快速验证闭环
- [x] Phase 1 地图入口场景命名为 `sect-map-main`
- [x] 第一版 tilemap 资源先走项目内最小 Tiled 文件

## Related Plans / Docs

- [docs/plans/phase-1-sect-map-validation.md](/Users/mawei/MyWork/SlgGame/docs/plans/phase-1-sect-map-validation.md)
- [docs/project/delivery-ledger.md](/Users/mawei/MyWork/SlgGame/docs/project/delivery-ledger.md)
