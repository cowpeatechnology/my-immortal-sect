# Current Plan

**Created**: `2026-04-20T17:48:09.000Z`  
**Updated**: `2026-04-20T18:24:30.000Z`

## Goal

启动 `M1-D` 最小权威短会话接轨里程碑，在保持 `Cocos Creator` 编译/预览为主要验证面的前提下，把当前 `M1-C` 的短会话从纯 client-local 推进到最小 `shared + Go + Hollywood` 权威路径，优先让 `place / upgrade / demolish`、资源扣减/返还与短会话核心快照不再悬空在客户端；本轮完成后必须回写必要文档，并由工程侧提交代码、推送远程。

## Subfunctions

- [x] 冻结 `M1-D` 最小权威范围与交付契约 (`supervisor`)
  - Description: 明确本轮只做“建造状态与资源结算”的最小 authority 竖切，不进入微信/抖音开发者工具验证，不扩写新玩法；冻结验收必须以 `Cocos Creator` 编译链路、专用 Chrome 预览与本地 authority runtime 证据为准，并要求工程 handoff 附带文档回写范围、提交 SHA 与远程分支信息。
  - Coordination:
  - human -> supervisor [dispatch/open]
    - Summary: 冻结 `M1-D` 最小权威范围与交付契约
    - Input: 明确本轮只做“建造状态与资源结算”的最小 authority 竖切，不进入微信/抖音开发者工具验证，不扩写新玩法；冻结验收必须以 `Cocos Creator` 编译链路、专用 Chrome 预览与本地 authority runtime 证据为准，并要求工程 handoff 附带文档回写范围、提交 SHA 与远程分支信息。
    - Expected Output: A structured coordination response that keeps the subfunction moving until completion or failure.
  - supervisor -> human [decision/done]
    - Summary: `M1-D` 范围、验收证据与工程交付契约已冻结。
    - Input: 明确本轮只做“建造状态与资源结算”的最小 authority 竖切，不进入微信/抖音开发者工具验证，不扩写新玩法；冻结验收必须以 `Cocos Creator` 编译链路、专用 Chrome 预览与本地 authority runtime 证据为准，并要求工程 handoff 附带文档回写范围、提交 SHA 与远程分支信息。
    - Expected Output: `M1-D` 的 in-scope / out-of-scope、验证面与工程 handoff 合同被固定，可直接用于后续 `engineer` dispatch。
    - Output: 已冻结本轮只覆盖 `place / upgrade / demolish`、资源扣减/返还与短会话关键快照的最小 authority 竖切；微信/抖音开发者工具、宿主 smoke、容器适配与新玩法全部保持 out-of-scope。验收以 `Cocos Creator` 编译链路、专用 Chrome `http://127.0.0.1:9333` 预览复用与本地 `shared + Go + Hollywood` authority runtime 证据为准；工程 handoff 必须附带文档回写范围、提交 SHA 与远程分支信息。
- [ ] 打通最小 `shared + Go + Hollywood` 权威短会话路径 (`engineer`)
  - Description: 在 `shared/` 与 `server/` 中建立最小 authority contract 与 `Go + Hollywood` 处理入口，把 `place / upgrade / demolish`、资源扣减/返还与短会话关键快照接入权威路径，并让 `client/my-immortal-sect/` 改为消费 authority snapshot / command 结果；完成后回写 `F-004` 与必要技术文档，并提交代码、推送远程。
  - Coordination:
  - human -> engineer [dispatch/open]
    - Summary: 打通最小 `shared + Go + Hollywood` 权威短会话路径
    - Input: 在 `shared/` 与 `server/` 中建立最小 authority contract 与 `Go + Hollywood` 处理入口，把 `place / upgrade / demolish`、资源扣减/返还与短会话关键快照接入权威路径，并让 `client/my-immortal-sect/` 改为消费 authority snapshot / command 结果；完成后回写 `F-004` 与必要技术文档，并提交代码、推送远程。
    - Expected Output: A structured coordination response that keeps the subfunction moving until completion or failure.
  - engineer -> supervisor [blocker/blocked]
    - Summary: 最小 `shared + Go + Hollywood` authority 短会话路径已本地打通并完成预览验证，但因 `client/my-immortal-sect` 是无远端的嵌套独立 git 仓库，无法按当前仓库拓扑完成统一 commit/push。
    - Input: 在 `shared/` 与 `server/` 中建立最小 authority contract 与 `Go + Hollywood` 处理入口，把 `place / upgrade / demolish`、资源扣减/返还与短会话关键快照接入权威路径，并让 `client/my-immortal-sect/` 改为消费 authority snapshot / command 结果；完成后回写 `F-004` 与必要技术文档，并提交代码、推送远程。
    - Expected Output: `shared` 中有最小 authority contract，`server` 中有可运行的 `Go + Hollywood` authority 入口，真实 Cocos 客户端预览改为消费 authority snapshot / command 结果并完成最小短会话回归，相关文档已回写，且代码已提交并推送远程。
    - Output: 已在本地完成 `M1-D` 最小 authority 竖切：`shared/contracts` 冻结了 `m1-authority-short-session-v1`，`server/` 新增 `go.mod`、Hollywood actor-backed session state 与 HTTP gateway，`client` 新增 authority client/contract 适配层并把 `place / upgrade / demolish`、`collect_stockpile`、`deliver_build_resource`、`complete_building_work`、`complete_demolition`、`complete_repair` 与 `sync_session_progress` 接到 authority command/snapshot。专用 Chrome 预览页已验证 authority bootstrap、生效后的 `building-*` authority id、自动 `collect_stockpile` 回写，以及通过当前客户端 authority 适配层把短会话从 `clear_ruin` 推进到 `raid_countdown`。`F-004`、`server/README`、`shared/README`、`docs/architecture/server-structure.md`、`docs/architecture/client-structure.md` 已回写。本轮未完成项是统一 commit/push：根仓库无法纳入 `client/my-immortal-sect` 内的代码变更，因此没有产生可接受的统一提交 SHA 或远端推送结果。
  - supervisor -> human [blocker/blocked]
    - Summary: `M1-D` 的最小 authority 路径已有本地实现与预览证据，但当前被仓库拓扑阻塞，无法满足既定的 commit/push 交付契约。
    - Input: 在 `shared/` 与 `server/` 中建立最小 authority contract 与 `Go + Hollywood` 处理入口，把 `place / upgrade / demolish`、资源扣减/返还与短会话关键快照接入权威路径，并让 `client/my-immortal-sect/` 改为消费 authority snapshot / command 结果；完成后回写 `F-004` 与必要技术文档，并提交代码、推送远程。
    - Expected Output: `shared` 中有最小 authority contract，`server` 中有可运行的 `Go + Hollywood` authority 入口，真实 Cocos 客户端预览改为消费 authority snapshot / command 结果并完成最小短会话回归，相关文档已回写，且代码已提交并推送远程。
    - Output: 已将该子功能在 durable records 中保留为 blocked：工程侧回报显示最小 `shared + Go + Hollywood` authority 竖切已本地打通并完成专用 Chrome 预览验证，相关文档也已回写；当前唯一未满足的交付项是 commit/push。阻塞原因不是 authority 实现本身，而是 `client/my-immortal-sect` 作为无远端的嵌套独立 git 仓库，无法按当前仓库拓扑产出可接受的统一提交 SHA 与远端推送结果。当前需要你裁定提交策略：1. 为 `client/my-immortal-sect` 补远端并允许分仓分别提交/推送；2. 调整本轮验收口径，允许分仓提交而非统一提交；3. 先整理仓库拓扑，再继续 `M1-D` 验收。
  - human -> supervisor [decision/answered]
    - Summary: 根仓是唯一项目仓库；嵌套子仓库是配置错误，可清理归并。
    - Input: 项目根目录是在 `~/MyWork/SlgGame`。如果这个根目录之下，还有独立仓库，说明设置错误，要删除掉错误消息。我可以接受清理掉所有Git记录，初始化一次之后创建并提交到远程仓库。
    - Expected Output: 明确 canonical git 拓扑与后续 commit/push 策略，解除“允许保留嵌套仓库”的歧义。
    - Output: 已确认根仓 `~/MyWork/SlgGame/.git` 是唯一 canonical 仓库，当前真正错误的是 `client/my-immortal-sect/.git` 与 `client/my-immortal-sect/extensions/cocos-mcp-server/.git` 两个嵌套 `.git`。根仓现有 `origin=git@github.com:cowpeatechnology/my-immortal-sect.git` 可继续作为远端，因此优先动作是清掉嵌套 `.git` 并归并到根仓，而不是默认整仓重建。
  - Notes: 当前 blocker 已从“仓库拓扑不明”收敛为“等待按单仓策略清理嵌套 `.git` 并恢复根仓统一 commit/push”。
- [ ] 验收 `M1-D` 并收口文档台账 (`supervisor`)
  - Description: 基于 engineer 回写证据判断 `M1-D` 是否达到“最小 authority-backed 短会话”门槛；验收通过后更新 `docs/project/delivery-ledger.md`、必要的 `docs/project/decision-log.md` 与当前计划，并确认提交/推送信息已进入 durable record。
