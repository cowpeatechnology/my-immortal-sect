# Decision Log

Record only decisions that change future work.

## Entries

- 2026-04-23: 目录级文档权威重置已执行：`docs/vision/design-decisions.md` 降级为 superseded shim，旧 `F-002` / `F-004` / `authority-first-runtime-rebuild` / `m0-vertical-slice` / sect-map 资产规格迁入 `docs/legacy/`，`features/` 改为 historical-only，`design/` 改为 inactive，`development-plan.json` 重建为从 GDD v3.1 派生的五阶段路线。
- 2026-04-23: 项目 North Star 已正式切换到 `docs/vision/gdd_v3_backend_design.md`，并由 `docs/decisions/0012-adopt-gdd-v3-authoritative-backend-gdd.md` 固定为采纳裁决。默认阅读顺序调整为：根 `AGENTS.md` -> `docs/vision/gdd_v3_backend_design.md` -> `docs/decisions/0012-adopt-gdd-v3-authoritative-backend-gdd.md` -> `docs/project/development-plan.json` -> `docs/project/development.active.json`。`docs/vision/design-decisions.md` 降级为 superseded shim。
- 2026-04-21: 旧 `.coordex/current-plan.md` / `.coordex/plan-history.md` / `.coordex/project-board.json` 规划表面不再作为当前项目的活跃真相。当前唯一活跃计划体系固定为 `docs/project/development-plan.json` + `docs/project/development.active.json` + `docs/project/development.log.jsonl`。
- 2026-04-20: `M1-A` 的偏验收只接受宗门地图在真实 Cocos 客户端路径中的 client-local `建 + 运` 基线；不得把该结论表述为已完成 `Go + Hollywood` / `shared` 权威闭环。后续里程碑入口应先补权威路径与最小 `守` 分支，而不是把当前状态误判为前后端主线已通。
- 2026-04-20: 人类已明确覆盖上一条“先补权威路径”的下一步建议。下一里程碑改为 `M1-B` 本地优先深化：先在真实 Cocos 客户端中补通用单位属性、建筑 HP / 受损 / 修复与更高可读资源包，用更丰富的本地流程验证玩法，再决定后续服务端权威接入顺序。
- 2026-04-21: `M1-C` 已接受为 client-local 可完成短会话里程碑。后续规划不得把该结论表述为小游戏宿主验证已通过，或 `Go + Hollywood` / `shared` 权威路径已接入；下一阶段应在“补小游戏容器侧有效 smoke/产物链路”与“开始 authority-boundary 接入”之间，由人类按产品目标继续裁定优先级。
- 2026-04-21: 当前阶段的验证顺序调整为“先确认 `Cocos Creator` 编译/预览链路，再决定是否切换到平台宿主”。微信/抖音开发者工具与真实小游戏容器测试后置，不再作为默认下一步或当前里程碑 blocker；只有在人类明确要求切换到对应工具验证时，才启动单独计划。
- 2026-04-21: `M1-D` 的首轮 authority 竖切范围已冻结为“建造状态与资源结算”最小贯通：只覆盖 `place / upgrade / demolish`、资源扣减/返还与短会话关键快照的最小 `shared + Go + Hollywood` 路径；微信/抖音开发者工具、宿主 smoke、平台适配与新玩法扩写全部保持 out-of-scope。后续 `engineer` handoff 必须带回文档回写范围、提交 SHA 与远程分支信息，才可进入主管验收。
- 2026-04-21: Git 拓扑已由人类重新裁定：`~/MyWork/SlgGame/.git` 是唯一 canonical 仓库，根目录下任何独立子仓库都视为配置错误，需被清理并归并到根仓。当前根仓已存在可用 `origin=git@github.com:cowpeatechnology/my-immortal-sect.git`，因此优先路径是删除嵌套 `.git` 并继续在根仓提交/推送，而不是默认整仓重建。
- 2026-04-21: `M1-D` 已接受为最小 authority-backed 短会话里程碑。下一轮不再围绕“是否进入 authority 路径”争论，而应在“优先把敌袭/守御结算继续迁入 authority”与“优先把采集节点/资源刷新继续迁入 authority”之间，由人类按产品目标裁定优先级。
- 2026-04-21: Git 提交节奏已由人类明确：默认只在一次完整计划/里程碑执行收口时做一笔统一 commit/push，不把每个子功能完成视为单独的自动提交边界。若要拆成多笔提交，必须由人类显式要求。
- 2026-04-21: `M1-E` 的范围已冻结为 authority 战斗收口：只继续推进 `defend / recover / victory / defeat` 的最小 authority 结算链，不进入微信/抖音宿主验证、真实小游戏容器、平台适配、采集节点/资源刷新 authority 化、protobuf、持久化、正式网关协议或新玩法扩写；本轮提交节奏继续保持整轮收口时统一 commit/push 一次。
