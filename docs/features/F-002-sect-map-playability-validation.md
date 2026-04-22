# F-002-sect-map-playability-validation

**状态**: historical  
**Area**: client / validation  
**Owner**: `supervisor`  
**Reviewer**: `human`  
**最后更新**: 2026-04-21

## Context

`F-002` 是宗门地图可玩性验证基线。

它回答的不是“整个宗门地图玩法是否完成”，而是更早期的问题：

> 这张宗门地图，能不能先被证明是一个真正可读、可点、可操作、可形成短反馈的游戏表面？

这个问题现在已经被后续工作吸收。
当前主功能线已经转到 `F-004`。

## Read Policy

This is not a default startup document.

Read it when:

- the active subfunction explicitly references `F-002`
- you need the historical baseline for sect-map playability
- you need to explain why earlier client-local evidence was accepted only as a bounded milestone

For live execution, prefer:

- `docs/project/development-plan.json`
- `docs/project/development.active.json`
- `docs/features/F-004-sect-map-m1-gameplay-foundation.md`

## Historical Goal

验证宗门地图在真实项目栈中不是“静态背景”或“HTML 探索原型”，而是可作为第一块真实游戏表面的切片。

## Historical Acceptance Focus

`F-002` 关注过的核心判断只有四类：

- 地图第一眼是否可读
- 蓝图落位是否能被理解
- 弟子当前工作是否可见
- 地图上是否存在一个短而清楚的即时反馈循环

## Historical Accepted Baseline

当前可以视为已经沉淀完成的历史基线是：

- 宗门地图已在真实 `Cocos Creator + TypeScript + Tiled / Cocos TiledMap` 路径中建立，而不是停留在 HTML mockup
- 地图契约快照、有限资源节点和基本逻辑层已经跑通
- 地图内存在最小 `建 + 运` client-local 闭环
- 地图上的可读性、选择反馈和任务反馈已经达到“可以继续向主玩法推进”的门槛

这些结论已经被后续文档吸收：

- 当前主功能文档：`F-004`
- 已验收记录：`docs/project/delivery-ledger.md`
- 未来工作决策：`docs/project/decision-log.md`

## Out-Of-Scope Boundary

`F-002` 从来不负责：

- 完整 authority 路径
- 完整战斗闭环
- 完整故事、因果、神降、存档纵深
- 完整 M0 或 V1 内容量

因此，它不应再被当作当前主线功能文档使用。

## Current Use

今天保留 `F-002` 的原因只有两个：

1. 作为宗门地图“第一性验证”基线
2. 作为解释早期 client-local 里程碑为什么可以被接受、但又不能被夸大成完整前后端闭环的历史依据

## Current Source Of Truth

如果问题是“现在到底要做什么”，请不要从本文件开始，而是从下面这些文件开始：

- `AGENTS.md`
- `docs/vision/design-decisions.md`
- `docs/project/development-plan.json`
- `docs/project/development.active.json`
- `docs/features/F-004-sect-map-m1-gameplay-foundation.md`

## Related Docs

- `docs/features/F-004-sect-map-m1-gameplay-foundation.md`
- `docs/project/delivery-ledger.md`
- `docs/project/decision-log.md`
- `docs/plans/m0-vertical-slice.md`
