# My Immortal Sect 文档入口

这里不是“大而全文档目录”，而是当前项目的简洁入口。

当前项目的默认原则是：

- 平时先读少量高权威文档，不默认通读整棵 `docs/`
- 产品与后端实现主规范只保留一套，以 `docs/vision/gdd_v3_backend_design.md` 为准
- `docs/project/development-plan.json` 是从主规范导出的当前执行拆解，不反向定义产品方向
- 其他文档按需进入，不把每次任务都变成长篇补课

## 默认阅读顺序

普通任务默认只读这几份：

1. 根 [AGENTS.md](/Users/mawei/MyWork/SlgGame/AGENTS.md)
   - 项目身份、权威顺序、角色体系、硬约束
2. [vision/gdd_v3_backend_design.md](/Users/mawei/MyWork/SlgGame/docs/vision/gdd_v3_backend_design.md)
   - 当前产品闭环与权威后端实现主规范
3. [decisions/0012-adopt-gdd-v3-authoritative-backend-gdd.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0012-adopt-gdd-v3-authoritative-backend-gdd.md)
   - 采纳 GDD v3.1 为唯一主规范的正式裁决
4. [project/development-plan.json](/Users/mawei/MyWork/SlgGame/docs/project/development-plan.json)
   - 从主规范导出的当前执行路线、phase / milestone / subfunction
5. [project/development.active.json](/Users/mawei/MyWork/SlgGame/docs/project/development.active.json)
   - 当前激活子功能和必须先读的文件

如果当前没有激活子功能，通常不需要先读 `development.active.json` 以外的大量流程文档。

## 当前最高权威

- 项目章程：根 [AGENTS.md](/Users/mawei/MyWork/SlgGame/AGENTS.md)
- Product / Backend North Star： [vision/gdd_v3_backend_design.md](/Users/mawei/MyWork/SlgGame/docs/vision/gdd_v3_backend_design.md)
- Adoption Decision： [decisions/0012-adopt-gdd-v3-authoritative-backend-gdd.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0012-adopt-gdd-v3-authoritative-backend-gdd.md)
- Protocol / Persistence Constraint： [decisions/0008-save-protocol-and-pgstore-schema.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0008-save-protocol-and-pgstore-schema.md)
- Runtime Simulation Constraint： [decisions/0010-offline-deterministic-simulation.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0010-offline-deterministic-simulation.md)
- V1 Sync Boundary： [decisions/0011-v1-sync-model-and-multiplayer-pattern.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0011-v1-sync-model-and-multiplayer-pattern.md)
- 当前执行计划： [project/development-plan.json](/Users/mawei/MyWork/SlgGame/docs/project/development-plan.json)
- 当前激活指针： [project/development.active.json](/Users/mawei/MyWork/SlgGame/docs/project/development.active.json)
- 已验收交付： [project/delivery-ledger.md](/Users/mawei/MyWork/SlgGame/docs/project/delivery-ledger.md)

## 按需阅读

只有任务真的需要时再进入下面这些分区。

### `decisions/`

用于不可轻易反转的架构决策，例如：

- Hollywood actor 选型
- 存档协议
- Actor ID 规则
- 离线补偿模拟
- V1 同步模型

### `architecture/`

用于当前真实代码结构和协作结构的边界说明，例如：

- 客户端目录与资产边界
- 服务端目录与责任边界
- 可见角色线程协作模型

### `features/`

用于持续迭代中的功能主文档，不是默认启动文档。

当前作为历史功能循环目录保留，不再承载 active 主线。

知识采集与本地知识库整理工具已迁出主仓库，当前外部项目路径为：

- `/Users/mawei/MyWork/知识采集工具/`

### `plans/`

用于阶段性拆解或早期实施计划。

注意：

- `plans/` 只保存阶段拆解或专题实施计划
- 历史计划已迁入 `legacy/`
- 当前活跃执行状态以 `project/development-plan.json` 为准，但其路线必须服从 `vision/gdd_v3_backend_design.md`

### `process/`

用于流程约束和专项工作流。

只在以下情况按需进入：

- 需要执行 Coordex V2 子功能流转
- 需要 Cocos MCP / 编辑器规则
- 需要专用浏览器验证
- 需要结构化线程通信协议

## 历史与参考

### `legacy/`

这里只保留历史思考，不作为当前默认阅读集。

用途：

- 追溯项目早期判断
- 回收仍有价值的范围清单或概念
- 对照哪些旧假设已经被后续文档修正

如果 `legacy/` 与当前 North Star 或 ADR 冲突，以当前权威文档为准。

## 文档维护规则

- 不再维护第二套并行“产品主规范”或“当前计划”文档
- 不再让索引页列出大量尚不存在的“未来文档”作为默认入口
- 短文档如果只是在复述根规则，应缩短并回链到权威文档
- 功能文档负责局部循环；ADR 负责约束与补丁；`development-plan.json` 负责当前执行路线；GDD 负责产品与后端主规范
