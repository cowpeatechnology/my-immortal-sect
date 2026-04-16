# Phase 1 Sect-Map Validation Plan

**状态**: accepted  
**最后更新**: 2026-04-16  
**依赖**:
- `AGENTS.md`
- `docs/process/development-loop.md`
- `docs/process/engineering-standards.md`
- `docs/architecture/agent-team-operating-model.md`
- `docs/vision/design-decisions.md`
- `docs/plans/m0-vertical-slice.md`
- `docs/features/F-002-sect-map-playability-validation.md`

## Context

Phase 1 exists to answer one question before full M0 begins:

> Is the sect map actually usable and playable as a game surface?

This phase is intentionally narrower than M0. It is a validation slice, not a content-complete vertical slice.

The project has already established the long-term direction for a xianxia sect-management sim. However, if the big map cannot be read, understood, and interacted with in a moment-to-moment loop, then the later actor / persistence / storylet work is built on the wrong foundation.

Phase 1 therefore treats the map as the first product truth to prove:

- the player can understand what the map is showing
- the player can place blueprints on the map without guesswork
- the player can see what disciples are doing
- the player can make at least one satisfying in-map decision loop before any full M0 systems are expanded

## Goal

Prove that the sect map is a playable surface, not just a pretty background.

## Non-Goals

- 不做完整 M0 架构闭环
- 不做离线补偿、PGStore、完整 actor 树
- 不做完整 storylet / karma 系统
- 不做大规模内容填充
- 不做“先好看再可玩”的纯插画方向

## Phase Definition

Phase 1 is the playable sect-map validation slice.

The required emphasis is:

- map readability
- blueprint placement clarity
- disciple task visibility
- moment-to-moment playability

Visual polish is still important, but it must serve readability and interaction.

## Milestones

### P1-1: Map Readability Gate

Deliver a map state where a new tester can identify:

- main hall / core buildings
- buildable vs blocked areas
- road / path hierarchy
- disciple presence and general state
- current map scale and navigation affordances

### P1-2: Blueprint Placement Gate

Deliver a blueprint interaction that makes placement obvious:

- blueprint footprint visible before commit
- valid / invalid placement state readable at a glance
- cost or requirement hints are visible
- rotation / alignment does not feel arbitrary

### P1-3: Disciple Task Visibility Gate

Deliver a disciple presentation state where the player can tell:

- who is idle
- who is moving
- who is working
- where each disciple is going
- what current task is blocking or completing progress

### P1-4: Moment-to-Moment Playability Gate

Deliver a short playable loop on the big map:

- inspect map
- place or confirm a blueprint
- observe disciple task assignment or completion
- see a visible result in the map state

The tester should understand what happened without reading implementation notes.

### P1-5: QA / Exit Gate

Run a short structured playtest and verify:

- the map remains readable after repeated interactions
- no important UI element fights with the map surface
- the loop still feels like a game, not a static scene
- there is no major conflict between playability and visual composition

## Acceptance Criteria

Phase 1 is complete only when all of the following are true:

- [x] New tester can identify the map’s main play objects within 5 seconds
- [x] Blueprint placement can be understood without external instruction
- [x] Disciple task state is visible during normal play
- [x] At least one map loop feels immediate and legible
- [x] The map can be played for a short session without confusion or UI overload
- [x] No later M0 assumption is made if the map validation slice has not passed

## Exit Condition

When Phase 1 is accepted:

- freeze the validated map interaction rules
- roll forward only the validated assumptions into M0
- move the project into the full M0 technical slice with map interaction risks reduced

## Validation Result

Phase 1 passed prototype-level QA on 2026-04-16.

Validated outcomes:

- 玩家可以直接在地图上选择东坪 / 南坡 / 西侧地块
- 蓝图落位后会进入清晰的 `备料 -> 施工 -> 启用` 状态链
- 弟子任务、资源变化、当前营造和地图标注能同步说明系统正在发生什么
- 地图不仅能完成单次建造，也能在一块地启用后继续切到另一块地推进下一条施工链

Captured and fixed during validation:

- Phase 1 中部面板在桌面浏览器里高度过高，曾遮挡南坡与西侧 lot
- 通过压缩任务流摘要和限制面板高度，已恢复重复交互可达性

## Authority Docs

- `AGENTS.md`
- `docs/process/development-loop.md`
- `docs/process/engineering-standards.md`
- `docs/architecture/agent-team-operating-model.md`
- `docs/vision/design-decisions.md`
- `docs/plans/m0-vertical-slice.md`

## Open Questions

- [ ] What is the minimum camera / navigation model that still keeps the map readable?
- [ ] Which blueprint interactions must be immediate, and which can remain deferred?
- [ ] How much disciple detail is enough for visibility without visual clutter?
- [ ] What is the minimum play loop that still feels like a sect-management game?

## Related Docs

- [F-002-sect-map-playability-validation.md](/Users/mawei/MyWork/SlgGame/docs/features/F-002-sect-map-playability-validation.md)
- [m0-vertical-slice.md](/Users/mawei/MyWork/SlgGame/docs/plans/m0-vertical-slice.md)
