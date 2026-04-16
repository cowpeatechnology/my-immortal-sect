# My Immortal Sect 标准研发循环

**状态**: 草案  
**最后更新**: 2026-04-16  
**依赖**: 根 `AGENTS.md`, `docs/process/engineering-standards.md`, `docs/architecture/agent-team-operating-model.md`

## Context

My Immortal Sect 不是一次性冲刺型项目，而是一个需要长期反复澄清玩法、架构、资产和实现边界的项目。

如果没有固定循环，研发过程很容易退化为：

- 用户一句话
- 主线程或 worker 直接开始写
- 改完后口头总结
- 下一轮再从记忆里猜之前做过什么

这会导致三个问题：

1. 冲突无法累计记录
2. 功能状态无法持续追踪
3. 智能体协作会逐轮丢失上下文

因此本项目采用统一的循环：

`Plan -> Design -> Execute -> Verify -> Record`

## 1. 循环适用范围

以下任务必须进入标准研发循环：

- 跨域任务（设计 + 架构 + 前端 / 后端 / QA）
- 会持续多轮推进的功能
- 对既有设计或架构可能产生冲突的变更
- 需要调用临时智能体的实现任务
- 需要形成验收结果和历史记录的中高复杂度任务

以下任务通常不必单独开一轮完整循环：

- 单文件小修
- 拼写 / 文案 / 路径类修正
- 已有循环内附带的小补丁

## 2. 循环步骤

### 2.1 Plan

责任人：`supervisor`

目标：

- 明确任务目标
- 确认是否值得立项成独立循环
- 分配 owner / reviewer
- 指定 write scope / no-touch scope
- 指定本轮要更新的 feature 文档和总台账

输出：

- 一个明确的 work order
- 一个 feature id（如 `F-012`）
- 一个 feature 文档路径（如 `docs/features/F-012-sect-map-loop.md`）

### 2.2 Design

责任人：通常是 `gameplay_designer` 或 `technical_architect`

目标：

- 把目标转成可执行设计
- 确认与既有玩法 / 架构是否冲突
- 必要时拆分成多步实现
- 如果是重大不可逆决策，额外创建 ADR

输出：

- feature 文档中的设计结论
- 若需要，新增或更新 ADR / 计划文档

### 2.3 Execute

责任人：`client_engineer` / `server_engineer` / `feature_worker` / `tools_engineer` / `art_asset_producer`

目标：

- 按 write scope 实施
- 不越权扩散范围
- 逐步提交可验收成果

输出：

- 代码、文档、资源或工具变更
- worker handoff

### 2.4 Verify

责任人：`qa_verifier`，必要时由主管补充人工判断

目标：

- 检查验收标准
- 检查回归风险
- 检查是否遗漏文档和记录

输出：

- findings / residual risks / next validation step

### 2.5 Record

责任人：`supervisor`

目标：

- 回写本轮真实结果
- 更新项目级状态
- 关闭或推进下一轮

必须回写：

- `docs/project/delivery-ledger.md`
- 对应 `docs/features/F-xxx-<slug>.md`

若本地 git 已启用：

- 本轮通过验收后，至少要有一次可追踪 commit

## 3. 功能状态字段

每个 feature 文档至少维护以下状态之一：

- `proposed`
- `planned`
- `designing`
- `implementing`
- `verifying`
- `accepted`
- `blocked`
- `deferred`
- `dropped`

项目总台账只保留高层状态摘要，详细过程进入 feature 文档。

## 4. 必要文档载体

### 4.1 项目总台账

文件：`docs/project/delivery-ledger.md`

用途：

- 看当前项目在做什么
- 看当前里程碑的主要风险
- 看已实现 / 未实现 / 暂缓功能

### 4.2 单功能循环文档

文件：`docs/features/F-xxx-<slug>.md`

用途：

- 记录单个功能或单个专项的完整循环历史
- 记录与既有系统的冲突和当前裁决
- 让后续循环有地方继续写，而不是重新解释上下文

### 4.3 ADR

文件：`docs/decisions/ADR-xxxx-*.md`

用途：

- 只记录重大、难以反转、跨多轮都要遵守的决策

不是所有功能都需要 ADR。

## 5. 冲突处理

如果某轮发现与既有设计 / 架构冲突，不要直接覆盖旧内容。

应当：

1. 在 feature 文档中新增“冲突与影响”段
2. 写清冲突对象、冲突原因、临时裁决
3. 如属架构级冲突，再补 ADR 或更新相关权威文档

## 6. Git 落痕原则

当本地 git 工作流启用后，执行以下规则：

- 每个**通过验收**的循环，至少有 1 个 commit
- 大循环允许多个 commit，但每个 commit 应保持单一目的
- 文档更新与实现更新尽量在同一循环内成套提交
- 不允许“功能已经通过验收，但没有任何提交记录”

## 7. 最小合格循环

一个最小合格循环，至少应留下以下痕迹：

- 有明确 owner 的 work order
- 有 feature 文档
- 有实施或设计产物
- 有 QA 或等价验收结论
- 有 delivery ledger 更新
- 若 git 已启用，有至少一个可追踪 commit

## 相关文档

- [AGENTS.md](/Users/mawei/MyWork/SlgGame/AGENTS.md)
- [docs/process/engineering-standards.md](/Users/mawei/MyWork/SlgGame/docs/process/engineering-standards.md)
- [docs/architecture/agent-team-operating-model.md](/Users/mawei/MyWork/SlgGame/docs/architecture/agent-team-operating-model.md)
- [docs/project/delivery-ledger.md](/Users/mawei/MyWork/SlgGame/docs/project/delivery-ledger.md)
