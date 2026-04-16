# My Immortal Sect GitHub 工作流

**状态**: 草案  
**最后更新**: 2026-04-16  
**依赖**: `AGENTS.md`, `docs/process/development-loop.md`, `docs/process/engineering-standards.md`

## Context

项目已经确定正式名称和远程仓库：

- 中文名：`我的宗门`
- 英文名：`My Immortal Sect`
- GitHub 仓库：`git@github.com:cowpeatechnology/my-immortal-sect.git`

本文件定义当本地 git 工作流启用后，主管和智能体应如何留下可追踪的研发记录。

## 1. 核心原则

- issue 用来记录要做什么
- feature 文档用来记录为什么这样做
- commit 用来记录实际改了什么
- PR 用来记录为什么现在合并

它们职责不同，不应互相替代。

## 2. 推荐工作项映射

### 2.1 中高复杂度功能

建议同时具备：

- 1 个 GitHub issue
- 1 份 `docs/features/F-xxx-<slug>.md`
- 1 个或多个 commit

### 2.2 小修复

可只保留：

- 1 个 issue 或在已有 issue 下继续
- 1 个 commit

### 2.3 架构级决策

建议具备：

- 1 个 issue 或 discussion
- 1 个 ADR
- 相关实现 commit

## 3. 分支策略

推荐策略：

- `main`：始终代表当前可接受主线
- 短生命周期分支：仅用于中高风险循环或多人并行时

命名建议：

- `feature/f-012-sect-map-loop`
- `fix/f-021-save-sync-edge-case`
- `tools/f-034-image-batch-pipeline`
- `docs/f-005-storylet-engine-rfc`

如果任务很小，也可以直接在主线完成，但前提是：

- 影响范围小
- 验证简单
- 不会引入长时间未完成状态

## 4. Commit 规则

### 4.1 基本规则

- 每个通过验收的循环，至少有 1 个 commit
- 每个 commit 尽量只做一件事
- 优先小 commit，而不是超大 commit

### 4.2 推荐格式

`<area>: <subject>`

示例：

- `docs: define supervisor work order template`
- `design: draft first sect map readability loop`
- `server: add simulation advance smoke test`
- `client: render first disciple mood badge`
- `tools: add image batch json parser`

## 5. Issue 字段建议

建议在 GitHub Project 或 issue 模板中保留以下字段：

- `Area`: design / architecture / client / server / tools / art / qa
- `Status`: proposed / planned / designing / implementing / verifying / accepted / blocked / deferred
- `Priority`: P0 / P1 / P2
- `Iteration`: 当前属于哪一轮
- `Needs ADR`: yes / no

## 6. 合并前检查

在合并或视为“本轮结束”前，至少检查：

- feature 文档已更新
- delivery ledger 已更新
- 验收结果已写明
- 相关风险已标注
- commit 信息可读

## 7. 当前状态说明

截至 2026-04-16：

- 远程仓库已创建
- 本地目录尚未初始化为 git repository

因此本文件当前主要起“工作流约定”作用。等本地 `git init` 并绑定远程后，再正式执行。

## 相关文档

- [docs/process/development-loop.md](/Users/mawei/MyWork/SlgGame/docs/process/development-loop.md)
- [docs/project/delivery-ledger.md](/Users/mawei/MyWork/SlgGame/docs/project/delivery-ledger.md)
