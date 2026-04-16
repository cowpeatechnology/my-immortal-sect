# F-001-project-governance-foundation

**状态**: accepted  
**Area**: process / architecture  
**Owner**: `supervisor`  
**Reviewer**: `qa_verifier`  
**最后更新**: 2026-04-16

## Context

项目已经从“散点讨论”进入“需要持续多轮协作”的阶段。

如果没有统一的主管派单、worker 回传、循环台账和 Git 工作流，本项目会在真正进入 M0 时迅速丢失上下文与边界。

因此需要先固化一套项目治理基础设施。

## Goal

- 明确项目正式名称
- 固化主管 / 常驻角色 / 临时角色的协作模型
- 固化标准研发循环
- 固化 work order / handoff / feature loop 模板
- 建立 delivery ledger
- 明确后续 GitHub 工作流约定

## Non-Goals

- 不启动 `client/` / `server/` / `shared/`
- 不进入 M0 实际编码
- 不处理某个具体玩法功能的实现

## Authority Docs

- `AGENTS.md`
- `docs/process/engineering-standards.md`
- `docs/vision/design-decisions.md`
- `docs/architecture/agent-team-operating-model.md`

## Acceptance Criteria

- [x] 正式项目名确定为 My Immortal Sect（《我的宗门》）
- [x] 主线程主管制与智能体编制规则已落盘
- [x] 标准研发循环文档已落盘
- [x] supervisor work order / worker handoff / feature loop 模板已落盘
- [x] delivery ledger 已落盘
- [x] GitHub 工作流约定已落盘

## Current Decision Summary

- 正式项目名采用 `My Immortal Sect`，中文名为《我的宗门》
- 本地目录代号暂保留 `SlgGame`，避免已有脚本与路径被误伤
- 主线程继续担任 `supervisor`
- `product_manager` 不独立设岗
- 研发采用 `Plan -> Design -> Execute -> Verify -> Record` 循环
- 每个通过验收的循环，未来在 git 启用后至少落一次可追踪 commit

## Conflict And Impact

- 冲突对象：旧的“每天 ≥ 5 commits”表述
  - 冲突原因：它约束的是频率，不足以约束“每个循环必须留痕”
  - 当前裁决：改为“每个通过验收的循环至少留下一次可追踪 commit”
  - 后续动作：等本地 git 启用后正式执行

- 冲突对象：项目正式名 vs 本地目录名
  - 冲突原因：项目品牌应统一，但本地目录和部分脚本仍依赖 `SlgGame`
  - 当前裁决：品牌与文档统一用 `My Immortal Sect`，本地目录代号暂不改
  - 后续动作：未来若需要，再做一次受控迁移

## Implementation Status

### Done

- [x] 更新顶层文档项目名
- [x] 更新 agent templates 中的项目标识
- [x] 新建 process / project / features / templates 文档体系
- [x] 新建 delivery ledger

### In Progress

- [x] 复核新文档之间的引用与一致性

### Not Started

- [ ] 本地 git 初始化
- [ ] 远程仓库绑定

### Deferred

- [ ] 目录名从 `SlgGame` 迁移到 `my-immortal-sect`

## Loop History

| Loop | Date | Stage | Summary | Output | Decision |
|---|---|---|---|---|---|
| L-001 | 2026-04-16 | Design | 固化主管制、角色编制、常驻/临时智能体边界 | `AGENTS.md`、`agent-team-operating-model.md`、agent templates | continue |
| L-002 | 2026-04-16 | Execute | 固化研发循环、台账、模板、Git 工作流，并切换正式项目名 | `docs/process/*`、`docs/project/delivery-ledger.md`、`docs/templates/*` | accepted |

## Open Questions

- [ ] 本地 git 仓库何时初始化
- [ ] 第一个真正进入研发循环的产品/技术 feature 是什么

## Related Issues / ADRs / Plans

- `docs/architecture/agent-team-operating-model.md`
- `docs/process/development-loop.md`
- `docs/process/github-workflow.md`
- `docs/project/delivery-ledger.md`
