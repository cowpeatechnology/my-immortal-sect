# Authority-First Runtime Rebuild（历史基线）

**状态**: accepted / historical baseline
**最后更新**: 2026-04-23
**适用范围**: `F-004` 宗门地图 authority 收口历史基线；不再定义当前项目主线

> 当前项目的产品闭环与后端实现主规范已切换到 [docs/vision/gdd_v3_backend_design.md](/Users/mawei/MyWork/SlgGame/docs/vision/gdd_v3_backend_design.md)。
> 本文档保留已接受的 authority-only 迁移经验、薄客户端边界与验收门禁经验，但不再作为当前产品主线。

## Why This Rebuild Exists

当前宗门地图短会话已经证明：

- 真实玩法主循环是可做的
- Go/Hollywood authority 路径是可接通的
- shared config、save/load、preview restore/reset 都已经有了有效基线

但当前运行模式仍然存在根本性结构问题：

- authority 和 client 同时在推进玩法状态
- client 仍然负责弟子任务分配、建造/修复完工推进、局部阶段切换
- client 发出的很多命令不是玩家意图，而是“替 authority 宣布内部步骤已完成”
- 浏览器预览中的 bounded acceptance 只能证明单个子功能通过，不能证明主链路已经 authority-only 且稳定

因此，当前计划不再继续沿着 hybrid 模式做补丁，而是整体切换为：

**authority-only gameplay state + thin client renderer/input shell**

## Target Runtime Model

### Authority owns

服务端必须成为以下内容的唯一真实来源：

- session phase / outcome / timers
- stockpile 与 resource node 全部结算
- building state、upgrade、damage、repair、demolition
- disciple 当前 assignment、work target、work kind、work progress
- raid trigger、damage result、repair closure、post-raid continuity
- save/load、restore/reset、deterministic replay

### Client owns

客户端只保留以下职责：

- 渲染地图、建筑、弟子、外敌、HUD、提示和视觉反馈
- 收集玩家输入并提交高层意图
- 根据 authority 下发的 worker / building / session state 做运动插值和动画表现
- 暴露 bounded debug surface，帮助读取 authority/client 对齐状态

### Client must not own

客户端后续不得再直接拥有以下玩法职责：

- 弟子下一份工作该做什么
- 建筑什么时候从 `planned -> supplied -> constructing -> active`
- 修复什么时候完成
- 建筑是否 damaged / repaired
- session 是否进入 `recover` / `second_cycle_ready` / `victory` / `defeat`
- 任何 authority 已拒绝的旧命令重试闭环

## Command Model Reset

后续命令分为两类：

### Player intent commands

这些命令可以继续由 client 发起：

- `place_building`
- `request_upgrade`
- `toggle_demolition`
- future: 其他明确的玩家指令

### Bounded fact commands

只有在当前 slice 无法避免本地表现层参与时，才允许上报有限事实，而且必须满足：

- 事实字段可验证
- authority 可拒绝
- 拒绝后 client 只能清理本地表现态并重新同步
- client 不能用这类命令宣布业务闭环完成

### Commands to remove from client-owned progression

后续不能再由 client 负责宣布完成的命令包括：

- `deliver_build_resource` 的错误重试闭环
- `start_building_work`
- `complete_building_work`
- `complete_repair`

这些步骤应逐步改为 authority 根据 assignment、progress、tick、snapshot 自行推进。

## Worker Model Reset

`M1` 后续不引入“每个弟子一个独立 actor”的新架构。继续遵守当前项目规则：

- disciple、building、active storylet 仍属于 authority big-state
- Go/Hollywood `SessionActor` / `SimulationActor` 统一推进该 big-state

但 disciple runtime state 必须 authority-owned，至少包括：

- current assignment kind
- target building / resource / tile
- carrying state
- work progress
- expected next transition

client 只消费这些字段做视觉移动和动作，不再自己分派工作。

## Acceptance Model Reset

后续任何子功能，不再允许只凭“定向点测 + 代码阅读 + 单个错误修复”通过。

### Required test layers

1. authority unit tests
2. gateway / contract tests
3. dedicated-browser end-to-end replay
4. human-run smoke flow

### Fail-closed conditions

以下任一条件成立，则该轮不能 accept：

- `authority.lastError != null` 出现在主链 checkpoint
- preview snapshot 与 authority snapshot 在关键字段不一致
- client-local fallback 参与了正式主链
- 同一 authority reject 被重复触发
- 需要刷新或 reset 才能继续主流程
- engineer 只提交代码或日志，没有 replay 证据
- supervisor 只复述 engineer 结论，没有独立 replay

## Mandatory Replay Paths

后续 engineer 与 supervisor 都必须能独立执行以下 replay：

### Mainline reset replay

`reset` 新档从头跑完：

- bootstrap
- clear ruin
- place guard tower
- build guard tower
- upgrade guard tower to Lv.2
- raid countdown
- defend
- recover
- repair closure
- second-cycle-ready continuity

### Restore replay

`restore_latest` 从已有 authority session 恢复后验证：

- snapshot shape 一致
- current phase / buildings / stockpile 一致
- 同一会话可继续推进而不是回退到 client-local 兜底

### Rejection recovery replay

至少保留一条 authority reject 恢复验证：

- authority 拒绝旧命令
- client 清理表现态
- 重新拉取 snapshot
- 主链继续推进

## Dual-Role Acceptance Gate

### Engineer gate

每次 submit 至少包含：

- authority / gateway tests
- dedicated browser replay
- bounded snapshot checkpoints
- 若修复回归 bug，则提供“修复前失败点 + 修复后同路径不再失败”的对照

### Supervisor gate

每次 accept 前至少独立执行：

- `reset` mainline replay 一次
- `restore_latest` replay 一次
- authority / preview bounded field 对照
- 核心 checkpoint 上 `authority.lastError = null`

## Milestone Sequence

### 1. Authority Boundary Reset

先冻结新的 authority-only client/server contract，并清除旧 hybrid acceptance language。

### 2. Authority-Owned Disciple Orchestration

把 disciple assignment、haul/build/repair intent 与 work progression 收回 authority。

### 3. Authority-Owned Raid And Repair Closure

把首袭伤害、recover repair closure、post-raid continuity 从 client 表现逻辑中拆出。

### 4. Authority Session Persistence And Replay

在 authority-owned worker state 基础上完成 deterministic save/load/replay。

### 5. Authority Thin Client Cutover

移除主预览中的 client-local gameplay fallback，仅保留 render / input / debug surface。

### 6. Authority E2E Verification Gate

把 engineer 自测和 supervisor 独立验收都写入强制流程。

### 7. Formal Protocol And Platform Hardening

authority mainline 稳定后，再推进 formal protobuf gateway、mini-game shell、observability。

### 8. Sect-Home Expansion On Authority Baseline

在 authority-only 主循环稳定之后，再继续 disciple arrival、sect institutions、karma hook。
