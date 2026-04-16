# My Immortal Sect 智能体团队运作模型

**状态**: 草案  
**最后更新**: 2026-04-16  
**依赖**: 根 `AGENTS.md`, `vision/design-decisions.md` §6, `docs/process/engineering-standards.md`

## Context

My Immortal Sect（《我的宗门》）的核心目标不是“把图片工作流跑起来”，而是做一款适合微信 / 抖音小游戏环境的修仙宗门经营游戏。AI 协作必须服务于这个核心目标。

本项目的 AI 协作模式不是“一个超级助手包打天下”，而是**用户作为项目负责人，主线程作为项目主管，专门岗位智能体按职责执行**。这更接近真实游戏团队：

- 用户：项目负责人 / 产品 owner / 最终拍板人
- 主线程：项目主管 / 交付负责人
- 子智能体：设计、架构、前端、后端、美术、QA 等岗位执行者

本文件回答四个问题：

1. 哪些智能体应该被视为项目的常驻编制
2. 哪些智能体应该按需临时启动
3. 主管如何调度、验收、销毁它们
4. 智能体之间如何沟通，避免上下文污染和职责重叠

## 1. 基本原则

### 1.1 常驻的是“岗位定义”，不是永不关闭的线程

对 Codex 来说，真正应该常驻的是：

- 角色模板
- 项目规则
- 目录级文档约束
- 调度流程

而不是长期挂着很多不销毁的运行线程。

原因：

- 长时间不关闭的 worker 很容易上下文污染
- 支撑工具、临时脚本、设计讨论会混在一个线程里，后续判断会变钝
- 对这个项目来说，清晰职责比“保留一切历史对话”更重要

因此本项目采用：

- **主线程常驻**
- **角色模板常驻**
- **执行线程短命**

除了主线程外，绝大多数 worker 都应当在单个工作单完成后关闭。

### 1.2 用户是真正的产品 owner，不另设 PM 智能体

本项目当前不建议单独设立 `product_manager` 智能体。

原因：

- 用户本身就是产品 owner 和最终拍板人
- 当前阶段的核心压力仍然是把设计原则、架构边界和执行顺序收敛清楚，再把这些原则推进到真实技术栈里
- 若再加一个 PM 智能体，容易出现“主管和 PM 互相转述”的管理层重复

替代方案：

- 玩法与功能定义，由 `gameplay_designer` 承担
- 技术方案与边界，由 `technical_architect` 承担
- 最终优先级和取舍，由用户 + 主线程决定

## 2. 常驻编制

这里的“常驻”指**项目长期保留的角色模板**，不是永远不关闭的运行线程。

### 2.1 `supervisor`（主线程，唯一真正常驻）

这不是 `.codex/agents/` 里的子代理，而是当前对话主线程。

职责：

- 接收用户任务
- 判断任务属于哪个域
- 拆分工作单
- 决定是否需要多智能体并行
- 指定 owner、reviewer、write scope、no-touch scope
- 整合输出
- 对最终结果负责

不应做的事：

- 不要在跨域复杂任务里既当主管又偷偷当所有执行者
- 不要绕开权威文档直接拍板架构
- 不要把未验收的 worker 输出直接当最终答案

### 2.2 `gameplay_designer`

职责：

- 核心玩法循环
- 弟子、因果、宗门、资源、神降的系统设计
- 玩法文档、策划文档、内容模板
- 把用户模糊想法整理成可执行设计

边界：

- 以设计文档和 feature 定义为主
- 不默认承担代码实现

为什么常驻：

- 这个项目的灵魂是玩法与叙事结构，不是技术框架本身
- 长期会反复出现系统设计问题，保留稳定角色很有价值

### 2.3 `technical_architect`

职责：

- 架构边界
- ADR
- Storylet / save / sync / simulation / actor 拆分
- 执行计划和阶段性落地路径

边界：

- 优先写决策文档、方案、计划
- 不默认承担大段实现

为什么常驻：

- 该项目硬约束很多，必须有专门角色持续守住边界

### 2.4 `client_engineer`

职责：

- Cocos Creator 客户端
- TypeScript strict 约束
- Tilemap、事件总线、UI、同步接入
- 客户端性能与渲染实现

边界：

- 不定义服务端权威逻辑
- 不替代 gameplay_designer 做功能定义

为什么常驻：

- 未来客户端工作会持续存在，且与服务端边界必须长期清晰

### 2.5 `server_engineer`

职责：

- Go 服务端
- Hollywood actor 组织方式
- big-State simulation
- PGStore / protobuf / save-sync / deterministic discipline

边界：

- 不把弟子 / 建筑 / storylet拆成错误的 actor 粒度
- 不越权定义产品规则

为什么常驻：

- 服务端约束是项目成败关键，必须长期由一个稳定角色守住

### 2.6 `art_asset_producer`

职责：

- 根据设计需求整理美术资源需求单
- 使用既有工作流产出大头、建筑、地图、图标等资源
- 统一命名、输出目录、规格说明
- 对图像结果做“是否符合本项目画风和用途”的初步筛选

边界：

- 不负责重写工具链底层
- 不负责玩法定义
- 不直接决定最终美术方向，最终由用户拍板

为什么常驻：

- 美术资源生成是长期持续动作
- 它服务于核心开发，不是一次性工具实验
- 把“出图执行”从“工具脚本开发”中独立出来，对项目更清晰

### 2.7 `qa_verifier`

职责：

- 验收标准检查
- 风险与回归检查
- 对文档、代码、流程、资源做交付前审视

边界：

- 默认只读
- 不自己顺手改实现

为什么常驻：

- 主管需要稳定的验收者，而不是让每个实现者自己说自己没问题

## 3. 临时智能体

临时智能体用于**短期、高噪声、强范围隔离**的任务。完成后应销毁，不应保留为长期上下文容器。

### 3.1 `feature_worker`

用途：

- 某个单点功能的实现
- 范围清晰、写集明确、可验收

典型场景：

- 写一个 UI 面板
- 接一个 protobuf 消息
- 落地某个小型子系统
- 在现有模块内完成一次局部重构

何时启动：

- 任务写集清晰
- 不需要长期保留上下文
- 主管已经给出明确 work order

何时销毁：

- 代码已提交给主管审阅
- QA 或 reviewer 已给出结论
- 需要的上下文已经体现在文件里

### 3.2 `tools_engineer`

用途：

- 临时工具开发
- 调试脚本
- 资产流水线修补
- 支持性自动化

为什么不设为常驻执行线程：

- 这类任务高噪声、上下文污染重
- 如果长期保留，会把主产品脉络和工具脉络混在一起

适用场景：

- 修一个生成脚本
- 加一个批处理能力
- 做一段临时迁移脚本
- 调一个工作流桥接器

### 3.3 其他按需临时角色

如果后续出现明确需求，可以再加短命模板，例如：

- `bug_reproducer`
- `migration_worker`
- `content_batch_worker`

但原则是：**没有稳定复用价值，不要把它升级成常驻编制。**

## 4. 推荐团队编制

### 4.1 长期保留的角色模板

- `supervisor`
- `gameplay_designer`
- `technical_architect`
- `client_engineer`
- `server_engineer`
- `art_asset_producer`
- `qa_verifier`

### 4.2 只按需调用的模板

- `feature_worker`
- `tools_engineer`

### 4.3 当前不建议独立设立的角色

- `product_manager`
- `release_manager`
- `scrum_master`

原因不是这些角色不重要，而是当前项目规模和阶段下，它们的职责已经被：

- 用户
- 主线程主管
- `gameplay_designer`
- `technical_architect`

共同覆盖了。单独拆出来只会增加汇报链条。

## 5. 调度方式

### 5.1 默认采用 hub-and-spoke

所有 worker 都向主管汇报，**不走 worker 与 worker 的自由直连协作**。

也就是说：

- worker A 需要 worker B 的输入
- 不应自己假设 B 已理解上下文
- 应回到主管，请主管转发或重新派单

这样做的原因：

- 保证信息源单一
- 防止两个 worker 用不同假设并行推进
- 保证用户只需要对齐主管一个入口

### 5.2 工作单格式

每个工作单至少包含：

- Objective
- Owner
- Write scope
- No-touch scope
- Authority docs
- Expected deliverable
- Validation required

没有这六项，不应开工复杂任务。

### 5.3 常见调度链路

#### 设计驱动型任务

`user -> supervisor -> gameplay_designer -> technical_architect -> relevant engineer -> qa_verifier -> supervisor`

适用于：

- 新系统定义
- 新功能落地
- 玩法和架构共同变化

#### 纯实现型任务

`user -> supervisor -> client_engineer/server_engineer/feature_worker -> qa_verifier -> supervisor`

适用于：

- 已有方案明确
- 只差实现

#### 美术资源任务

`user or gameplay_designer -> supervisor -> art_asset_producer -> qa_verifier or human visual review -> supervisor`

适用于：

- 大头资源
- 建筑资源
- 地图底图
- 图标和按钮

#### 工具链任务

`user -> supervisor -> tools_engineer -> qa_verifier -> supervisor`

适用于：

- 一次性脚本
- 工作流改进
- 批处理 / 数据转换 / 自动化桥接

## 6. 通讯协议

### 6.1 主管下发

主管下发给 worker 的信息必须包含：

- 任务目标
- 约束
- 只准改哪些文件
- 明确不要碰哪些文件
- 先读哪些文档
- 成果以什么形式回传

### 6.2 worker 回传

worker 回传必须包含：

- 改了什么
- 改了哪些文件
- 做了哪些验证
- 还有什么风险
- 建议谁来下一棒

### 6.3 禁止的通讯方式

- 不要让 worker 自己决定跨域任务的主流程
- 不要让 worker 之间私下形成新的 authority chain
- 不要让 worker 在未通知主管的情况下扩大写集

## 7. 什么时候必须用临时智能体

符合以下任一条件时，优先用临时智能体：

- 任务上下文噪声很高
- 任务只会做一次或极少复用
- 任务写集明确且可封闭
- 需要并行吞吐，但不值得污染常驻角色线程
- 是工具链、调试、实验性质工作

示例：

- 做一个只服务当前批量资源生产的脚本
- 调一个浏览器自动化问题
- 写一个临时数据迁移器
- 对某个模块做一次局部重构

## 8. 什么时候不该新开智能体

不要为了“看起来像团队”而机械分代理。

以下情况不建议新开：

- 单文件小改动
- 主管自己在当前上下文已经足够清楚
- 额外智能体不会降低认知负担
- 任务无法明确 write scope

## 9. 与本项目阶段的关系

当前仓库已从纯 planning / pre-M0 过渡到“允许真实 M0 实现启动”的阶段。

因此默认优先级是：

1. 玩法与架构收敛
2. 文档和计划补齐
3. 按既定技术栈推进真实前后端切片
4. 美术资产规格与生成工作流

在这个阶段，最常用的角色组合通常是：

- `supervisor + gameplay_designer + technical_architect`
- `supervisor + client_engineer + server_engineer + qa_verifier`
- `supervisor + art_asset_producer`
- `supervisor + tools_engineer`

额外澄清：

- `hifi-prototype/` 仍可用于视觉和交互草图，但不能被当作“第一阶段可玩大地图”已经完成的证据
- 若里程碑写的是“大地图可玩”，默认应理解为真实技术栈切片，而不是 HTML 原型
- 对可运行工作，主管和 worker 的通讯必须包含启动命令、测试地址、验收流程，不能只给抽象结论

## 10. 实施建议

短结论：

- **主线程常驻**
- **角色模板常驻**
- **执行线程短命**
- **不设独立 PM 智能体**
- **美术资源智能体应为常驻角色模板**
- **工具开发类智能体默认临时**

这是目前最适合 My Immortal Sect 的智能体团队结构。

## 11. 规则落点

为了避免这些规则只停留在说明层，本项目把它们分别落在以下位置：

- 根 `AGENTS.md`
  作用：给 Codex 提供仓库级总章程、主管模式、角色总表、工作单格式、交接要求
- `.codex/agents/*.toml`
  作用：定义各岗位模板，包括常驻角色模板与临时角色模板
- `.codex/hooks/common.py`
  作用：把“主管制、常驻/临时角色、无独立 PM”压缩进会话公共上下文
- `.codex/hooks/user_prompt_submit.py`
  作用：在用户提交任务时提醒按主管模式路由，并对跨域任务、美术任务做角色建议
- `.codex/rules/*.rules`
  作用：承接仓库阶段和危险动作约束，例如保护目录、禁止 destructive 操作，以及阶段性初始化策略

换句话说：

- 原则在文档里
- 角色在模板里
- 会话提醒在 hooks 里
- 危险边界在 rules 里

## 相关文档

- [AGENTS.md](/Users/mawei/MyWork/SlgGame/AGENTS.md)
- [docs/process/engineering-standards.md](/Users/mawei/MyWork/SlgGame/docs/process/engineering-standards.md)
- [docs/README.md](/Users/mawei/MyWork/SlgGame/docs/README.md)
- [vision/design-decisions.md](/Users/mawei/MyWork/SlgGame/docs/vision/design-decisions.md)

## 未决问题

- [ ] M0 正式开始后，是否需要把 `feature_worker` 再细分为 `client_feature_worker` / `server_feature_worker`
- [ ] M1 美术产能扩大后，是否需要把 `art_asset_producer` 拆成“提示词设计”和“出图执行”两个角色
