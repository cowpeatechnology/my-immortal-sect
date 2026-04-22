# F-001-project-governance-foundation

**状态**: accepted  
**Area**: process / architecture  
**Owner**: `supervisor`  
**Reviewer**: `human`  
**最后更新**: 2026-04-19

## Context

项目已经从“散点讨论”进入“需要持续多轮协作”的阶段。

如果没有统一的主管派单、worker 回传、循环台账和 Git 工作流，本项目会在真正进入 M0 时迅速丢失上下文与边界。

因此需要先固化一套项目治理基础设施。

## Goal

- 明确项目正式名称
- 固化主管 / 常驻角色 / 临时角色的协作模型
- 固化标准研发循环
- 固化 work order / handoff / feature loop 模板
- 固化跨线程可见对话账本协议
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
- `docs/project/project-method.md`

## Acceptance Criteria

- [x] 正式项目名确定为 My Immortal Sect（《我的宗门》）
- [x] 可见角色主管制与智能体编制规则已落盘
- [x] 标准研发循环文档已落盘
- [x] supervisor work order / worker handoff / feature loop 模板已落盘
- [x] 线程对话账本协议、live ledger 与消息模板已落盘
- [x] delivery ledger 已落盘
- [x] GitHub 工作流约定已落盘

## Current Decision Summary

- 正式项目名采用 `My Immortal Sect`，中文名为《我的宗门》
- 本地目录代号暂保留 `SlgGame`，避免已有脚本与路径被误伤
- 当前协作模型以 Coordex 三角色可见线程体系为准：`supervisor / engineer / art_asset_producer`
- `supervisor` 在当前体系内承担 product owner、计划、路由和验收职责
- `product_manager` 不独立设岗
- 研发采用 `Plan -> Design -> Execute -> Verify -> Record` 循环
- 每个通过验收的循环至少落一次可追踪 commit
- 长跑 worker 默认采用 `60s` heartbeat / `120s` silence handling
- observer / network / runtime logs 默认按增量、过滤、限量方式读取，不允许再把整份大日志直接塞进上下文
- 允许用户在同一项目下维护多个可见角色聊天；新任务启动和最终 acceptance 仍归 `supervisor` 或人类，已激活子功能内允许角色受限直连协调

## Conflict And Impact

- 冲突对象：旧的“每天 ≥ 5 commits”表述
  - 冲突原因：它约束的是频率，不足以约束“每个循环必须留痕”
  - 当前裁决：改为“每个通过验收的循环至少留下一次可追踪 commit”
  - 后续动作：按当前 git 工作流持续执行

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
- [x] 补强运行期治理规则：长日志读取预算、worker 心跳、失联处理
- [x] 重构 hooks 分层：SessionStart 负责稳定基线，UserPromptSubmit 只负责 prompt 级动态提醒
- [x] 新增线程对话协议、project ledger 与消息模板，支持按角色保留可见聊天而不破坏主管制

### In Progress

- [x] 复核新文档之间的引用与一致性

### Not Started

- _None._

### Deferred

- [ ] 目录名从 `SlgGame` 迁移到 `my-immortal-sect`

## Loop History

| Loop | Date | Stage | Summary | Output | Decision |
|---|---|---|---|---|---|
| L-001 | 2026-04-16 | Design | 固化主管制、角色编制、常驻/临时智能体边界 | `AGENTS.md`、角色规则文档、agent templates | continue |
| L-002 | 2026-04-16 | Execute | 固化研发循环、台账、模板、Git 工作流，并切换正式项目名 | `docs/process/*`、`docs/project/delivery-ledger.md`、`docs/templates/*` | accepted |
| L-003 | 2026-04-16 | Execute | 补强运行期治理：限制大日志读取、要求增量观测，并给 worker 增加 heartbeat / silence handling 默认规则 | `AGENTS.md`、`docs/process/engineering-standards.md`、角色协作规则、`docs/templates/*`、`docs/project/delivery-ledger.md` | accepted |
| L-004 | 2026-04-16 | Execute | 重构 Codex hooks 分层，避免 SessionStart 与 UserPromptSubmit 重复注入完整项目上下文 | `.codex/hooks/*.py`、角色协作规则 | accepted |
| L-005 | 2026-04-18 | Execute | 新增“可见线程对话账本”治理层；该轮仍采用“默认 `Via: supervisor`”口径，后续已在 2026-04-19 被当前三角色直连规则取代 | `AGENTS.md`、结构化协调协议、`docs/project/thread-conversation-ledger.md`、`docs/templates/thread-message-template.md`、`docs/README.md`、角色协作规则 | accepted |
| L-006 | 2026-04-19 | Record | 将旧六角色 / 主线程主管模型收敛到当前 Coordex 三角色可见线程体系，并同步计划、协作协议、hooks 与本地 agent 配置 | `AGENTS.md`、`Agents/*`、`docs/process/*`、`docs/project/*`、`.codex/hooks/*` | accepted |

## Open Questions

- [ ] 第一个真正进入研发循环的产品/技术 feature 是什么
- [ ] 是否要把 worker 心跳与长日志预算进一步自动化为 watchdog / hook
- [ ] thread conversation ledger 是否需要后续增加归档 / watchdog / 状态提醒脚本

## Related Issues / ADRs / Plans

- `AGENTS.md`
- `docs/project/project-method.md`
- `docs/process/development-loop.md`
- `docs/process/engineering-standards.md`
- `docs/project/delivery-ledger.md`
