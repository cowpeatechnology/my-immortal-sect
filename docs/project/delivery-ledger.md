# My Immortal Sect 项目总台账

**状态**: 活文档  
**最后更新**: 2026-04-16  
**项目名**: My Immortal Sect（《我的宗门》）  
**当前阶段**: Phase 1 / accepted, ready for user playtest  

## Context

本文件是项目级持续更新台账。

它不记录某个功能的全部细节，而是记录：

- 当前里程碑是什么
- 目前哪些循环正在推进
- 已实现 / 未实现 / 暂缓项有哪些
- 当前最主要的冲突和风险是什么

详细过程进入 `docs/features/`。

## 1. 当前里程碑

- 当前主里程碑：`Phase 1 / sect-map validation slice`
- 当前目标：
  - 验证大地图是否真的可用、可读、可玩
  - 收敛大地图交互规则、blueprint 反馈和弟子任务可见性
  - 为 full M0 提供经过验证的地图交互前提

## 2. 活跃循环

| Loop ID | Feature / Topic | Area | Owner | Status | Notes |
|---|---|---|---|---|---|
| L-001 | `F-001` 项目治理基础设施 | process / architecture | supervisor | accepted | 已固化主管制、角色模板、agent team model |
| L-002 | `F-001` 正式命名与循环模板 | process / docs | supervisor | accepted | 已完成模板、总台账、Git 工作流与项目名同步 |
| L-003 | `F-002` 大地图可玩性验证切片 | client / design / qa | supervisor | accepted | 已完成地图闭环实现与浏览器验收，可交给项目主管体验 |

## 3. 已实现 / 已固化

### 3.1 规则与组织

- [x] 主线程作为 `supervisor`
- [x] 常驻角色模板与临时角色模板划分
- [x] hub-and-spoke 调度原则
- [x] pre-M0 阶段保护规则

### 3.2 文档与架构

- [x] 核心愿景文档
- [x] ADR 0007 ~ 0011
- [x] M0 vertical slice 草案
- [x] 智能体团队运作模型草案

### 3.3 工具与资产工作流

- [x] ChatGPT 图像生成工作流已具备基本可用性
- [x] 相关工具文档已存在于 `tools/` 范围

### 3.4 Phase 1 原型验证

- [x] `hifi-prototype/home-immersive.html` 已具备大地图可玩验证闭环
- [x] 已验证地块选择、蓝图落位、备料、施工、启用产出
- [x] 已验证重复交互：南坡完工后可继续推进西侧施工
- [x] 已修复 Phase 1 面板遮挡南坡/西侧 lot 的问题

## 4. 未实现 / 待推进

### 4.1 设计与策划

- [ ] Storylet DSL 正式文档
- [ ] 因果系统文档
- [ ] 弟子系统文档
- [ ] 建筑系统文档
- [x] Phase 1 大地图可玩性验证完成并验收

### 4.2 工程与实现

- [x] 本地 git 仓库初始化并绑定远程
- [ ] `client/`, `server/`, `shared/` 正式进入 M0
- [x] 第一个 feature 文档按模板启动（`F-001-project-governance-foundation.md`）
- [x] `F-002-sect-map-playability-validation.md` 进入执行、验证并通过首轮验收

## 5. 暂缓 / Deferred

- [ ] PM 智能体单独设岗
- [ ] 细分 `client_feature_worker` / `server_feature_worker`
- [ ] 细分“提示词设计”和“出图执行”两个美术角色
- [ ] full M0 技术垂直切片在 Phase 1 完成前展开

## 6. 当前冲突与风险

| ID | Topic | Type | Status | Notes |
|---|---|---|---|---|
| R-001 | 本地 git 仓库 | process | closed | 已初始化并绑定到 `git@github.com:cowpeatechnology/my-immortal-sect.git` |
| R-002 | 设计文档仍多于实现文档 | stage | expected | 目前仍是 pre-M0，属合理现象 |
| R-003 | 图像工作流可用但仍非项目核心 | scope | watch | 要防止工具工作继续挤占玩法 / 架构主线 |
| R-004 | 大地图读图性与可玩性未实测 | product | closed | Phase 1 已完成 prototype 级验收，风险下降为后续平衡与扩展问题 |
| R-005 | Phase 1 已通过但仍缺少用户主管体验反馈 | product | watch | 需由项目主管亲手体验后再决定 M0 的具体落点 |

## 7. 下一步建议

1. 由项目主管体验 `hifi-prototype/home-immersive.html`，确认 Phase 1 是否满足“真的可玩”
2. 基于 Phase 1 验证结果，拆出 M0 技术垂直切片的第一个研发 feature
3. 用新模板启动下一个 “Plan -> Design -> Execute -> Verify -> Record” 循环

## 相关文档

- [docs/process/development-loop.md](/Users/mawei/MyWork/SlgGame/docs/process/development-loop.md)
- [docs/process/github-workflow.md](/Users/mawei/MyWork/SlgGame/docs/process/github-workflow.md)
- [docs/templates/feature-loop-template.md](/Users/mawei/MyWork/SlgGame/docs/templates/feature-loop-template.md)
