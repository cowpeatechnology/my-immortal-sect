# My Immortal Sect 项目总台账

**状态**: 活文档  
**最后更新**: 2026-04-16  
**项目名**: My Immortal Sect（《我的宗门》）  
**当前阶段**: pre-M0 / 设计与核心流程收敛  

## Context

本文件是项目级持续更新台账。

它不记录某个功能的全部细节，而是记录：

- 当前里程碑是什么
- 目前哪些循环正在推进
- 已实现 / 未实现 / 暂缓项有哪些
- 当前最主要的冲突和风险是什么

详细过程进入 `docs/features/`。

## 1. 当前里程碑

- 当前主里程碑：`pre-M0`
- 当前目标：
  - 固化项目根规则与智能体协作方式
  - 收敛玩法 / 架构 / 研发循环文档体系
  - 保留并服务于既有美术与图像工作流

## 2. 活跃循环

| Loop ID | Feature / Topic | Area | Owner | Status | Notes |
|---|---|---|---|---|---|
| L-001 | `F-001` 项目治理基础设施 | process / architecture | supervisor | accepted | 已固化主管制、角色模板、agent team model |
| L-002 | `F-001` 正式命名与循环模板 | process / docs | supervisor | accepted | 已完成模板、总台账、Git 工作流与项目名同步 |

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

## 4. 未实现 / 待推进

### 4.1 设计与策划

- [ ] Storylet DSL 正式文档
- [ ] 因果系统文档
- [ ] 弟子系统文档
- [ ] 建筑系统文档

### 4.2 工程与实现

- [ ] 本地 git 仓库初始化并绑定远程
- [ ] `client/`, `server/`, `shared/` 正式进入 M0
- [x] 第一个 feature 文档按模板启动（`F-001-project-governance-foundation.md`）

## 5. 暂缓 / Deferred

- [ ] PM 智能体单独设岗
- [ ] 细分 `client_feature_worker` / `server_feature_worker`
- [ ] 细分“提示词设计”和“出图执行”两个美术角色

## 6. 当前冲突与风险

| ID | Topic | Type | Status | Notes |
|---|---|---|---|---|
| R-001 | 本地仍非 git 仓库 | process | open | 远程已建，本地尚未执行 `git init` 和 remote 绑定 |
| R-002 | 设计文档仍多于实现文档 | stage | expected | 目前仍是 pre-M0，属合理现象 |
| R-003 | 图像工作流可用但仍非项目核心 | scope | watch | 要防止工具工作继续挤占玩法 / 架构主线 |

## 7. 下一步建议

1. 初始化本地 git 仓库并绑定远程
2. 选定第一个真正进入产品/技术研发循环的 feature，创建 `F-002-...`
3. 用新模板跑一次完整的“Plan -> Design -> Execute -> Verify -> Record”

## 相关文档

- [docs/process/development-loop.md](/Users/mawei/MyWork/SlgGame/docs/process/development-loop.md)
- [docs/process/github-workflow.md](/Users/mawei/MyWork/SlgGame/docs/process/github-workflow.md)
- [docs/templates/feature-loop-template.md](/Users/mawei/MyWork/SlgGame/docs/templates/feature-loop-template.md)
