# My Immortal Sect 工程规范

**状态**: 草案  
**最后更新**: 2026-04-16  
**依赖**: 根 `AGENTS.md`, `docs/vision/design-decisions.md`, 相关 ADR 与 feature 文档

## Context

本文件是 My Immortal Sect 的项目级 Codex 工程规范。

它用于承接原本散落在旧根规范文档中、但对 Codex 研发仍然必要的内容：

- 编码纪律
- 任务启动检查
- 禁止事项
- 命名规范
- TypeScript / Go 约束
- commit 规范

根 `AGENTS.md` 负责仓库级行为和多智能体组织；
本文件负责更细的工程执行标准。

## 1. 六条工程纪律

1. **每个模块必须有测试**
   重点：Storylet 引擎、存档序列化、战斗符号系统、因果触发器

2. **TypeScript strict + Go 强类型**
   禁止 `any`，禁止空 `interface{}`，配置表必须有生成的类型定义

3. **小批次、频繁提交**
   每个通过验收的研发循环至少留下一次可追踪 commit；大循环可拆成多个小 commit

4. **大改动先 plan 后执行**
   范围超过 1 个模块、或会影响架构 / 协议 / 数据模型的改动，必须先有计划和明确 work order

5. **模块边界硬约束**
   模块间通过接口、事件总线或协议交互；不要跨模块偷改内部状态

6. **不要越界修改**
   只修改本次 work order 明确授权的文件和目录

## 2. 任务启动检查

在开始中高复杂度任务前，至少完成：

1. 读取 `AGENTS.md`
2. 读取相关权威文档：
   - `docs/vision/design-decisions.md`
   - 相关 ADR
   - 相关 feature 文档
3. 确认本次任务的 write scope / no-touch scope
4. 如果已有测试，先确认基线
5. 如果是跨域任务，确认本轮需要回写：
   - `docs/project/delivery-ledger.md`
   - 对应 `docs/features/F-xxx-<slug>.md`

## 3. 禁止事项

- 不要引入 `any` 作为偷懒方案
- 不要绕过 Storylet 引擎另起第二套事件系统
- 不要在客户端做资源扣减、战斗结算、因果触发等权威决策
- 不要修改已经进入线上配置体系的配置 ID
- 不要把 UI 逻辑和 simulation 逻辑混在一起
- 不要为了 demo 方便写死本应配置驱动的内容
- 不要把 `console.log` 当正式日志体系
- 不要把 `TODO: 临时` / `FIXME: 以后修` 当成主线交付完成态
- 不要同屏渲染超过 1 个 Spine
- 不要让客户端和服务端各自实现不同版本的 Storylet 逻辑

## 4. 命名规范

### 4.1 通用

- 文件：`kebab-case.ts` / `snake_case.go`
- 类：`PascalCase`
- 变量 / 函数：`camelCase`
- 常量：`SCREAMING_SNAKE_CASE`

### 4.2 配置表 ID

格式：

`category.subcategory.name`

示例：

- 建筑：`building.main_hall`
- 资源：`res.wood`
- 岗位：`job.gather_wood`
- 境界：`realm.qi_refining`
- Storylet：`karma.xxx_yyy` / `descent.world_scene`

规则：

- 一旦进入线上配置体系，ID 永不修改
- 只能新增，不可复用

## 5. TypeScript 规范

- `strict: true`
- 禁止 `any`
- 配置表生成 `.d.ts`
- 模块通过 `index.ts` 暴露公共 API
- UI 与 simulation 严格分离
- 不要把所有场景逻辑塞进 ECS

## 6. Go 规范

- 统一 `gofmt`
- 使用 `golangci-lint`
- 错误显式返回，不以 panic 处理业务分支
- 接口定义在使用方，避免提前抽象
- 不用 `init()` 承载业务逻辑
- 优先单二进制部署，不为未来过早拆微服务

## 7. Git 规范

- commit 格式：`<area>: <subject>`
- 一次 commit 尽量只做一件事
- 主分支应保持可接受状态
- 不 amend 已推送 commit
- 不 force push 到 `main`

## 8. 人工主导的事项

以下事项可以由 Codex 辅助，但不应完全交给 Codex 拍板：

- 美术最终审美判断
- 音乐音效
- 文案终稿
- 真机性能结论
- 手感与节奏调优
- 产品方向取舍

## 相关文档

- [AGENTS.md](/Users/mawei/MyWork/SlgGame/AGENTS.md)
- [docs/process/development-loop.md](/Users/mawei/MyWork/SlgGame/docs/process/development-loop.md)
- [docs/process/github-workflow.md](/Users/mawei/MyWork/SlgGame/docs/process/github-workflow.md)
