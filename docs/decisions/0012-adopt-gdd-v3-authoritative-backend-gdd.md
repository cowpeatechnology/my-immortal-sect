# ADR 0012: 采用 GDD v3.1 作为产品与权威后端主规范

**状态**: 已确认 (Accepted)
**日期**: 2026-04-23
**决策者**: 项目负责人 + Codex
**相关文档**:
- `docs/vision/gdd_v3_backend_design.md`
- `docs/vision/design-decisions.md`
- `docs/decisions/0008-save-protocol-and-pgstore-schema.md`
- `docs/decisions/0010-offline-deterministic-simulation.md`
- `docs/decisions/0011-v1-sync-model-and-multiplayer-pattern.md`
- `docs/project/development-plan.json`

---

## Context

项目在 `F-004` 宗门地图与 authority runtime rebuild 阶段沉淀出了一批有效的 authority-only 运行时经验，但当前执行系统逐渐出现了一个结构问题：

1. `docs/vision/gdd_v3_backend_design.md` 已经是更完整的产品闭环与后端实现蓝图。
2. `docs/project/development-plan.json` 曾以宗门地图 authority 债务清理为当前主线。
3. `docs/README.md`、`docs/vision/design-decisions.md`、`F-004`、`authority-first-runtime-rebuild.md` 与早期 `m0-vertical-slice` 之间对“谁是 North Star、谁是主规范、谁只是执行拆解”存在漂移。

继续把旧的 `F-004` / authority debt clean-up 路线当作默认主线，会导致项目长期围绕局部 sect-map runtime 收口打转，而不是按真正的宗门经营产品闭环推进。

---

## Decision

采用 `docs/vision/gdd_v3_backend_design.md` 的 v3.1 版本作为当前项目的：

- 产品主规范
- 权威后端实现主规范
- 智能体执行拆解的上游来源

同时明确以下约束：

1. `development-plan.json` 是从权威 GDD 导出的执行路线，不得反向改写产品方向。
2. `docs/vision/design-decisions.md` 保留为高层原则摘要，不再与权威 GDD 竞争实现级主规范地位。
3. `F-004`、`docs/plans/authority-first-runtime-rebuild.md` 与早期 `m0-vertical-slice` 降级为历史基线与迁移经验文档，不再定义当前主线。
4. GDD v3.1 必须吸收以下已接受 ADR 约束：
   - ADR 0008：`protobuf-first`、`state_blob + index tables`、非 JSON 权威持久化
   - ADR 0010：`big-state deterministic simulation`
   - ADR 0011：`V1 单玩家云存档边界`，不是共享房间式多人同步
5. GDD v3.1 采用事件应用流水线来约束状态修改入口，但不采用“只存事件、由事件完全重建状态”的纯 event sourcing。

---

## Consequences

### 正面影响

1. 项目重新获得单一的产品闭环与后端实现北极针。
2. 执行计划可以从“sect-map authority debt cleanup”切换到“宗门经营 authority core bootstrap”。
3. 后续所有客户端、服务端、协议、存档、回放、机构、因果扩展，都能落在同一份主规范上推进。

### 约束与代价

1. `gdd_v3_backend_design.md` 必须被修订为 v3.1，删除与 ADR 0008/0011 冲突的 JSON、多人房间、纯 event-sourcing 误读与过宽 MVP 表述。
2. `development-plan.json` 必须重建当前 phase / milestone / subfunction 路线，使旧 F-004 authority runtime rebuild 不再作为当前主线。
3. 后续若出现新的架构争议，不得再通过新增平行“主计划”文档来绕过权威 GDD；应修订 GDD 或新增 ADR。
