# features/ 目录说明

本目录用于维护“单功能或单专项”的循环文档。

它不是：

- North Star
- 当前整体开发计划
- 项目总台账

当前这些职责分别属于：

- North Star: `docs/vision/design-decisions.md`
- 当前整体路线: `docs/project/development-plan.json`
- 已验收台账: `docs/project/delivery-ledger.md`

## 命名规则

命名格式：

`F-xxx-<slug>.md`

示例：

- `F-001-project-governance-foundation.md`
- `F-002-sect-map-playability-validation.md`
- `F-004-sect-map-m1-gameplay-foundation.md`

## 当前使用建议

默认不要把所有 feature 文档都当启动文档来读。

通常只在以下情况进入：

- `development.active.json` 的 `must_read` 明确要求
- 当前 subfunction 直接属于某个 feature
- 需要追溯某个功能循环的历史设计、验收或边界

## 当前项目内的功能文档角色

- `F-001`：治理与协作地基，已接受
- `F-002`：宗门地图可玩性验证基线，偏历史基线
- `F-004`：当前宗门地图 M1 主功能文档

独立支持线说明：

- 知识采集与知识库整理工具已迁出到 `/Users/mawei/MyWork/知识采集工具/`

## 推荐内容

每个 feature 文档应优先记录：

- 这个功能为什么存在
- 当前目标和非目标
- 验收标准
- 已接受结论
- 当前仍未解决的冲突或 open questions

如果某部分内容已经进入：

- ADR
- `development-plan.json`
- `delivery-ledger.md`
- `decision-log.md`

就不要在 feature 文档里再维护第二份平行真相。
