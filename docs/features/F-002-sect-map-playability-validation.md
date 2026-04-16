# F-002-sect-map-playability-validation

**状态**: active  
**Area**: client / server / design / qa  
**Owner**: `supervisor`  
**Reviewer**: `qa_verifier`  
**最后更新**: 2026-04-16

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

- [ ] The map’s main structures and play areas are legible within a short first look
- [ ] Blueprint placement shows footprint, validity, and orientation clearly
- [ ] Disciple task visibility makes current work state obvious during play
- [ ] A player can perform a short interaction loop and see an immediate result on the map
- [ ] No HUD element obscures the map to the point that play becomes ambiguous
- [ ] The map remains understandable during repeated interactions, not only on the first frame
- [ ] Phase 1 can be judged playable by QA without relying on implementation context
- [ ] The slice runs on the real project stack: Cocos Creator + TypeScript client, Go + Hollywood-backed runtime path where applicable
- [ ] The main map uses the intended client map tech path, centered on Tiled / Cocos TiledMap rather than HTML canvas mockup

## Current Decision Summary

- Phase 1 is a validation slice that precedes broader M0 expansion
- Readability and playability outrank scenic illustration if they conflict
- Blueprint and disciple feedback must live on the map surface, not only in hidden logs
- Prototype exploration may inform the slice, but cannot replace the real-stack milestone

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

### Exploratory Evidence

- [x] HTML prototype 证明了“大地图可玩”至少需要地块选择、蓝图落位、施工链、弟子可见任务状态
- [x] HTML prototype 暴露了真实 UX 问题：中部信息面板会压住可交互 lot
- [x] prototype 的交互结论可以转译为真实 client/server 切片输入

### Not Yet Done

- [ ] `client/` 真实建立并进入 Cocos Creator + TypeScript 开发
- [ ] `server/` 真实建立并进入 Go + Hollywood 开发
- [ ] tilemap / TiledMap 路径打通
- [ ] 至少一条前后端联动的大地图命令路径
- [ ] 基于真实运行栈的 QA 验收

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

## QA Summary

- 已完成的 QA 仅覆盖 HTML prototype，不构成 feature 验收通过
- 后续 QA 必须基于真实客户端与真实服务端切片
- prototype 中暴露的交互与信息密度问题，后续应作为真实实现时的重点回归项

## Open Questions

- [ ] 真实 Cocos tilemap 中，lot 可见性与 HUD 层级如何组织最稳
- [ ] blueprint footprint 和 disciple 路径提示的最小表达形式是什么
- [ ] 第一条真实前后端命令路径选“蓝图落位”还是“弟子派工”更合适

## Related Plans / Docs

- [docs/plans/phase-1-sect-map-validation.md](/Users/mawei/MyWork/SlgGame/docs/plans/phase-1-sect-map-validation.md)
- [docs/project/delivery-ledger.md](/Users/mawei/MyWork/SlgGame/docs/project/delivery-ledger.md)
