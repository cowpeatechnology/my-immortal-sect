# My Immortal Sect 可见角色协作模型

**状态**: 已确认  
**最后更新**: 2026-04-19  
**依赖**: 根 `AGENTS.md`, `docs/process/development-loop.md`, `docs/process/thread-conversation-protocol.md`

## Context

`SlgGame` 现在以 Coordex 的三角色可见线程体系为准，而不是旧的六角色 Codex 子智能体体系。

项目当前的长期协作表面是：

- `Agents/supervisor/`
- `Agents/engineer/`
- `Agents/art_asset_producer/`

这些角色线程是：

- 透明可见的
- 可人工切换和审查的
- 带稳定目录级上下文的
- 比隐藏式子线程更适合当前项目节奏的

本文件回答四个问题：

1. 为什么当前以可见角色线程为准
2. 三个角色分别负责什么
3. 它们如何协作而不失控
4. 哪些文档承担计划、沟通和验收职责

## 1. 为什么以可见角色线程为准

当前模型优先保留：

- 角色职责边界清晰
- 人类可见的上下文容器
- 可审计的任务路由与交接
- 在长周期开发中可持续维护的计划与历史

不再把旧的六角色 `.codex` specialist roster 当作项目协作的默认真相，原因是：

- 角色过多会让当前项目的真实执行链路变复杂
- 旧模型把技术架构、客户端、服务端、QA 拆得过细，不适合当前实际节奏
- 当前项目更需要稳定的“主管-工程-美术”三角，而不是形式上更完整但切换成本更高的岗位树

因此：

- `supervisor` 是当前 Coordex 体系里的产品 owner 与项目协调者
- `engineer` 同时承担技术架构与实现责任
- `art_asset_producer` 负责资源方向与资源产出

## 2. 当前角色集合

### 2.1 `supervisor`

职责：

- 接收用户目标
- 维护当前计划
- 把目标拆成单一 owner 的 subfunction
- 指定 scope / no-touch / validation
- 判断何时需要工程或美术执行
- 做最终验收并更新项目记录

边界：

- 不默认承担 `engineer` 或 `art_asset_producer` 的实施工作
- 不在未明确 owner 的情况下让任务自动扩散
- 不把未验收的执行结果直接当最终完成

### 2.2 `engineer`

职责：

- 技术架构
- 客户端与服务端实现
- 集成、调试、运行验证
- 技术风险识别与技术方案取舍

边界：

- 不替代 `supervisor` 做产品取舍或最终验收
- 不把 scope change、owner change、milestone change 当成可自行决定的事项
- 不把临时技术判断写成新的项目方向

项目决议：

- 旧模型里的 `technical_architect`、`client_engineer`、`server_engineer`，当前都归并到 `engineer`

### 2.3 `art_asset_producer`

职责：

- 资源需求拆分
- 画风与用途约束整理
- 生成工作流执行
- 输出命名、目录、规格、交付包装

边界：

- 不负责技术架构与代码实现
- 不自行拍板最终产品方向
- 对接集成或范围变化时，应回到 `supervisor` 或当前激活 subfunction 的协调链路

## 3. 协作规则

### 3.1 单一 owner

每个激活中的 subfunction 必须只有一个 owner。

也就是说：

- 一个子功能要么归 `engineer`
- 要么归 `art_asset_producer`
- 不允许同时把同一个子功能派给两个实施角色

### 3.2 谁可以启动任务

以下动作只能由人类或 `supervisor` 触发：

- 新建 subfunction
- 指定 owner
- 修改 milestone
- 修改 scope
- 最终验收完成

### 3.3 何时允许角色直连

在一个 subfunction 已经被明确激活后，角色之间允许直接协调。

允许的直连内容包括：

- question
- blocker
- handoff
- result
- decision

但前提是：

- 仍然在当前 subfunction 范围内
- 不改变 owner
- 不扩大 write scope
- 不把“临时协调”升级为“新任务启动”

### 3.4 什么时候必须回到 `supervisor`

以下情况必须回到 `supervisor` 或人类：

- scope 需要变更
- 需要换 owner
- 需要新开子功能
- 需要做产品取舍
- 需要做最终 acceptance

## 4. 文档与载体

### 4.1 角色规则

- 根规则：`AGENTS.md`
- 共享角色层：`Agents/AGENTS.md`
- 角色局部规则：`Agents/<role>/AGENTS.md`

### 4.2 当前计划

- 当前计划：`.coordex/current-plan.md`
- 历史计划：`.coordex/plan-history.md`

计划要求：

- 一个清晰 goal
- 若干单一 owner 的 subfunctions
- 不把整个项目历史塞进每个角色线程

### 4.3 执行与验收记录

- 项目总台账：`docs/project/delivery-ledger.md`
- 角色状态：`docs/project/role-state/<role>.md`
- 单功能循环：`docs/features/F-xxx-<slug>.md`

### 4.4 协调消息

- 结构化协议：`docs/process/structured-agent-communication-protocol.md`
- 线程协作规则：`docs/process/thread-conversation-protocol.md`
- 可见对话账本：`docs/project/thread-conversation-ledger.md`
- 消息模板：`docs/templates/thread-message-template.md`

原则：

- in-chat 协调尽量结构化
- 只有高价值协调事件才镜像进 ledger
- ledger 是可见性工具，不是第二套任务系统

## 5. 已淘汰的默认假设

以下假设不再是当前项目的默认协作模型：

- “主线程是唯一 supervisor，本体外都是短命 worker”
- “默认六角色常驻编制”
- “默认严格 hub-and-spoke，角色之间不能直连”
- “独立 `qa_verifier` 负责默认验收”
- “独立 `technical_architect` 负责默认技术边界”

如果未来项目规模变化，需要重新拆回更多角色，应单独立项并更新本文件，而不是在当前文档体系里混用旧规则。

## 相关文档

- [AGENTS.md](/Users/mawei/MyWork/SlgGame/AGENTS.md)
- [docs/process/development-loop.md](/Users/mawei/MyWork/SlgGame/docs/process/development-loop.md)
- [docs/process/thread-conversation-protocol.md](/Users/mawei/MyWork/SlgGame/docs/process/thread-conversation-protocol.md)
- [docs/project/project-method.md](/Users/mawei/MyWork/SlgGame/docs/project/project-method.md)
