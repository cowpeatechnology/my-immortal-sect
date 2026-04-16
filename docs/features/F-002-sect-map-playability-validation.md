# F-002-sect-map-playability-validation

**状态**: accepted  
**Area**: client / design / qa  
**Owner**: `supervisor`  
**Reviewer**: `qa_verifier`  
**最后更新**: 2026-04-16

## Context

The project’s long-term vision is a xianxia sect-management sim. Before the full M0 slice can be justified, the big sect map itself needs to prove that it is actually usable as a game surface.

This feature exists to validate the map as a playable system:

- readable at first glance
- interactable through blueprint placement
- able to communicate disciple work status
- able to support a short moment-to-moment loop

This is not a content-complete feature. It is a validation feature.

## Goal

Build and verify a sect-map slice that a tester can understand and interact with without needing to mentally fill in missing structure.

## Non-Goals

- Full M0 actor / persistence / offline compensation
- Full storylet / karma system
- Full economy depth
- Full content library of buildings or disciples
- Final art polish that sacrifices readability

## Authority Docs

- `AGENTS.md`
- `docs/process/development-loop.md`
- `docs/process/engineering-standards.md`
- `docs/architecture/agent-team-operating-model.md`
- `docs/plans/phase-1-sect-map-validation.md`
- `docs/plans/m0-vertical-slice.md`

## Acceptance Criteria

- [x] The map’s main structures and play areas are legible within a short first look
- [x] Blueprint placement shows footprint, validity, and orientation clearly
- [x] Disciple task visibility makes current work state obvious during play
- [x] A player can perform a short interaction loop and see an immediate result on the map
- [x] No HUD element obscures the map to the point that play becomes ambiguous
- [x] The map remains understandable during repeated interactions, not only on the first frame
- [x] Phase 1 can be judged playable by QA without relying on implementation context

## Current Decision Summary

- Phase 1 is a validation slice that precedes full M0
- Readability and playability outrank scenic illustration if they conflict
- Blueprint and disciple feedback must live on the map surface, not only in hidden logs
- The target is “usable and playable,” not “fully content complete”

## Conflict And Impact

- 冲突对象：纯 scenic / illustration-first map direction
  - 冲突原因：过度追求画面美感会降低地图读图性和操作确认度
  - 当前裁决：Phase 1 优先可玩性，必要时牺牲部分装饰密度
  - 后续动作：在 Phase 1 通过后再考虑更高层次的美术强化

- 冲突对象：M0 技术垂直切片直接前置
  - 冲突原因：如果大地图本身不可玩，后续 actor / persistence 工作缺少产品验证基础
  - 当前裁决：先完成 Phase 1，再进入 M0
  - 后续动作：Phase 1 通过后冻结验证过的地图交互假设

## Implementation Status

### Done

- [x] Phase 1 的目标与验收方向已定义
- [x] Phase 1 与 full M0 的边界已明确
- [x] `hifi-prototype/home-immersive.html` 接入 Phase 1 地图可玩验证交互
- [x] 地块选择、蓝图选择、落位、备料、施工、启用产出形成闭环
- [x] 弟子状态、任务流、资源、昼夜时段、施工状态同屏更新
- [x] QA 已验证至少一条单地块闭环和一条多地块重复交互闭环

### Verified

- [x] 新玩家可在首屏辨认地块、蓝图入口和当前营造状态
- [x] 南坡药园落位并完工后，仍可继续在西侧地块推进炼器房
- [x] 中部验证面板压住南坡/西侧 lot 的问题已修复
- [x] QA 可仅凭页面反馈理解当前施工阶段与弟子任务

### Deferred

- [ ] 施工取消 / 拆除 / 重排队列
- [ ] 更细的 blueprint footprint / rotation 表达
- [ ] 更长时段的平衡性验证
- [ ] 完整 M0 技术切片中的 actor / persistence / offline 相关内容

## Loop History

| Loop | Date | Stage | Summary | Output | Decision |
|---|---|---|---|---|---|
| L-001 | 2026-04-16 | Plan | 定义 Phase 1 为大地图可玩性验证切片 | `docs/plans/phase-1-sect-map-validation.md` | continue |
| L-002 | 2026-04-16 | Execute | 在 `hifi-prototype` 中实现地图交互闭环与动态反馈 | `home-immersive.html` / `immersive-loop-phase1.js` / `styles.css` | continue |
| L-003 | 2026-04-16 | Verify | 浏览器内完成单地块与多地块重复交互验收，修复 panel 遮挡地块问题 | 本文档 + delivery ledger | accepted |

## QA Summary

- QA 路径 1：东坪/南坡地块选择 -> 蓝图落位 -> 备料 -> 施工 -> 启用产出
- QA 路径 2：南坡药园施工中切换到西侧地块继续落位炼器房，验证重复交互与并行施工可读性
- 关键修复：Phase 1 面板在桌面浏览器中垂直高度过高，曾遮挡南坡/西侧 lot；已通过压缩任务流和限制面板高度修复
- 剩余风险：当前仍是 prototype 级切片，玩法节奏参数和高级交互尚未进入长期平衡阶段

## Open Questions

- [ ] 什么样的地图提示密度最能保持可读性
- [ ] blueprint 预览和 disciple 路径提示的最小表达形式是什么
- [ ] moment-to-moment loop 的最低验收时长是多少

## Related Plans / Docs

- [docs/plans/phase-1-sect-map-validation.md](/Users/mawei/MyWork/SlgGame/docs/plans/phase-1-sect-map-validation.md)
- [docs/project/delivery-ledger.md](/Users/mawei/MyWork/SlgGame/docs/project/delivery-ledger.md)
