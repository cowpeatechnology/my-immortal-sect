# F-004-sect-map-m1-gameplay-foundation

**状态**: active  
**Area**: design / client / server / qa  
**Owner**: `supervisor`  
**Reviewer**: `engineer`  
**最后更新**: 2026-04-21

## Context

`F-002` 解决的是“大地图是否能作为可用游戏表面”。

在当前阶段，这个问题已经收敛到一个更具体的后续问题：

> 如果宗门大地图不仅要“能看、能拖、能点”，还要真正承担经营玩法，那么第一阶段到底要让玩家在这张图上做什么？

这份文档定义 `M1` 的大地图玩法基础盘。

它的职责不是追求内容量，也不是一次性做成完整修仙模拟，而是把《我的宗门》的第一条真实地图主循环锁定为一个可开发、可验证、可扩展的最小系统：

- 玩家在宗门地图上放置建筑
- 弟子按规则自动工作
- 资源在地图上被采集、搬运、消耗
- 一次局部威胁打断稳态，迫使玩家调整布局和人事
- 结果重新回到宗门地图，而不是跳到另一套玩法壳

方向参考了 RimWorld 的“角色驱动经营 + 事件促变 + 系统叙事”，但严格遵守本项目的边界：

- 宗门是家，不是求生殖民地
- 情绪重点在宗门内部对象，而不是外部 spectacle
- 地图是主经营盘，不是开放世界
- 建造、资源、战斗、事件最终都要能回收到同一张宗门图上

## Goal

定义一个可执行的 `M1` 大地图玩法计划，使团队在审核通过后可以直接进入实现，而不需要再次重做玩法收敛。

`M1` 的目标不是“功能多”，而是“形成第一条稳定主循环”：

**看图 -> 放建筑 -> 弟子自动工作 -> 资源积累 -> 升级/扩张 -> 应对第一次敌袭 -> 修复并继续经营**

## Non-Goals

本轮明确不做：

- 无缝超大世界地图
- 程序化随机地图生成
- 完整 RimWorld 式需求系统（饥饿、睡眠、器官、精神崩溃链）
- 全量自由地形改造
- 手操即时战斗
- 多势力外交与复杂宗门关系网
- 多人同步经营
- 完整 Storylet / Karma 纵深内容
- 完整经济大链条和工业化生产网
- 为了“先跑起来”而把资源、建造、战斗结算长期留在客户端做权威逻辑

## Authority Docs

- `AGENTS.md`
- `docs/vision/design-decisions.md`
- `docs/process/engineering-standards.md`
- `docs/plans/phase-1-sect-map-validation.md`
- `docs/plans/m0-vertical-slice.md`
- `docs/features/F-002-sect-map-playability-validation.md`

## Acceptance Criteria

- [ ] `M1` 地图有一张固定宗门主盘，支持拖拽、缩放、点击和稳定选中
- [x] 地图最少包含四类逻辑地块：`buildable`、`blocked`、`road`、`resource`
- [x] 玩家可以在有效地块上执行三类建筑操作：`place`、`upgrade`、`demolish`
- [x] `M1` 最少有五类建筑：`main_hall`、`disciple_quarters`、`warehouse`、`herb_garden`、`guard_tower`
- [x] `M1` 最少有三类地图资源：`spirit_wood`、`spirit_stone`、`herb`
- [x] 至少一名弟子能根据规则自动执行：`gather`、`haul`、`build_or_repair`、`guard`
- [x] 地图上存在可见的建筑状态链：`blueprint -> constructing -> active -> damaged`
- [x] 路径系统能够避开阻挡格与占用建筑，并稳定到达资源点、仓库、工地和防守点
- [x] 至少有一次“外敌从边缘进入 -> 逼近主殿 -> 护山台/弟子自动应对 -> 战后修复”的敌袭闭环
- [x] 玩家可以从新档开始，在 5~10 分钟内清楚完成一次“扩张后守住第一波敌袭”的短会话
- [x] 地图上的关键状态变化可以直接观察，不依赖隐藏日志解释
- [ ] 实现路径必须为真实项目栈：客户端 `Cocos Creator + TypeScript + Tiled / Cocos TiledMap`；资源、建造、战斗等关键状态的最终接受态必须保留到 `Go + Hollywood` 权威路径

## Current Decision Summary

### 1. `M1` 的产品定义

`M1` 不是“大地图技术样板”，而是《我的宗门》的第一块经营底盘。

这块底盘只承载三根玩法支柱：

- **建**：在地图上放置、升级、拆除宗门建筑
- **运**：让弟子按规则自动采集、搬运、建造和修复
- **守**：在一次轻量敌袭里验证布局、响应和恢复能力

## 2026-04-17 Runtime Snapshot

当前客户端实现已经进入 `M1` 早期运行态，不再只是“能看见 TileMap”：

- 固定宗门主盘已由真实 `TiledMap` 驱动
- 逻辑层已接入 `road / blocked / resource_wood / resource_stone / resource_herb`
- 地图上可见：
  - 主殿占格
  - 资源点标记
  - 弟子头像 token
  - 工具栏模式按钮
- 运行时系统已具备：
  - 采集标记
  - 建筑蓝图落位
  - 自动搬运
  - 自动施工
  - 建筑状态从 `planned -> supplied -> active` 的推进
- 已用专用 Chrome 的现有 `http://localhost:7456` 预览页做过真实烟测，确认：
  - 标记灵木后，弟子会自动寻路并进入采集循环
  - 给定库存后，下达 `药圃` 蓝图会经历备料与完工，并在地图上转为激活建筑

这说明 `F-004` 已从“玩法计划”进入“真实实现中的基础盘搭建”阶段。

## 2026-04-20 M1-B Local Loop Update

本轮在真实 Cocos 客户端里把 `M1-A` 的最小 `建 + 运` 基线，推进到了更适合玩法调校的 `M1-B` 本地短闭环：

- 新增弟子与外敌共用的本地单位模型，统一承载：
  - 基础属性
  - 衍生移动/采集/搬运/建造/修复速度
  - HP、攻防、攻击间隔与警戒参数
- 建筑运行态已接入：
  - `maxHp`
  - `damaged`
  - `repairCost`
  - `structureDefense`
  - 受击闪烁反馈
- 地图循环已补齐：
  - `guard_tower` 自动守御
  - 外敌边缘刷新、逼近、索敌与攻击
  - 弟子 `guard` / `repair` 任务分配
  - 受损建筑优先资源采集与修复回正
- 运行时调试快照已扩展：
  - `buildings[]` 含状态与 HP
  - `hostiles[]` 含目标、HP 与可见状态
  - `disciple.model` / `disciple.hp`

专用 Chrome `http://localhost:7456/` 预览页已验证：

- 外敌能从边缘进入并攻击非主殿建筑
- 护山台与弟子都会对外敌产生命中反馈
- 建筑会掉血并进入 `damaged`
- 当库存不足时，弟子会优先采集修复所需资源
- 资源到位后，弟子会执行 `repair` 并把建筑恢复到 `active`

当前仍未完成的项：

- `upgrade` 仍未进入本地闭环
- `5~10 分钟短会话` 还未做真实节奏调校
- `Go + Hollywood` 权威侧仍未接入，本轮仍属 client-local 玩法深化

## 2026-04-20 M1-B High-Readability Map Asset Package

本轮把当前版本宗门地图的高可读资源包，冻结为一套可直接供后续接图的 runtime SVG 主文件集。

当前 canonical runtime path：

- `client/my-immortal-sect/assets/resources/generated-buildings/sect-map-svg/`

当前资源包范围：

- 弟子头像 token：
  - `sect_disciple_normal.svg`
  - `sect_disciple_injured.svg`
  - `sect_disciple_dying.svg`
  - `sect_disciple_dead.svg`
- 建筑图标：
  - `main_hall.svg`
  - `disciple_quarters.svg`
  - `warehouse.svg`
  - `herb_garden.svg`
  - `guard_tower.svg`
- 资源图标：
  - `spirit_wood.svg`
  - `spirit_stone.svg`
  - `herb.svg`

尺寸与锚点合同：

- 弟子头像：
  - master size: `96x96`
  - node anchor: `0.5, 0.5`
  - intended in-token display: 置入当前 `88x96` token root 时，主体脸部保持 `56~64 px` 直径可读范围
- 建筑图标：
  - `main_hall`: `384x336`
  - `disciple_quarters`: `320x272`
  - `warehouse`: `256x224`
  - `herb_garden`: `256x192`
  - `guard_tower`: `192x288`
  - node anchor: `0.5, 0.5`
  - 视觉基线约束：底部 footprint 中线对齐 tile footprint 中心，不靠偏移 pivot 补偿
- 资源图标：
  - `spirit_wood`: `128x128`
  - `spirit_stone`: `128x128`
  - `herb`: `128x112`
  - node anchor: `0.5, 0.5`

命名与导出规格：

- runtime basename 一律使用 snake_case，并与 `BUILDING_DEFINITIONS.id` / `ResourceKind` 保持一一对应
- 当前提交形态为 `SVG` master；若后续进入 atlas / SpriteFrame 路径，PNG fallback 必须沿用同名 basename，不另起第二套命名
- `.meta` 继续使用 Cocos `importer: "*"` 的 SVG 导入路径
- 当前子任务只交付资源包与规格，不把 `sect-map-bootstrap` 现有 `Graphics` token / building / resource 渲染静默改成贴图消费

## 2026-04-20 Supervisor Acceptance Note

主管本轮接受的范围仅限于 `M1-B` 中的“本地通用单位属性与损伤/修复闭环”子任务，而不是整个 `F-004` 的最终完成。

已接受内容：

- 弟子与外敌共用本地单位模型，且已进入真实 Cocos 客户端运行态
- 建筑 HP、`damaged`、`repairCost`、受击反馈与修复回正已进入同一地图循环
- `guard_tower` 自动守御、外敌边缘刷新与弟子 `guard / repair` 已构成最小本地 `守 -> 修` 验证链
- 运行证据已回写到本节的 runtime snapshot，且验证路径保持为既有 `http://localhost:7456/` 预览页

仍保持 open 的范围：

- `upgrade`
- 更长时长的节奏与压力调校
- `Go + Hollywood` / `shared` 权威接入

## 2026-04-20 Supervisor Acceptance Note - Asset Package

主管已接受 `M1-B` 中的“高可读地图资源包”子任务完成。

本轮 accepted 内容：

- `sect-map-svg/` 目录下的 canonical runtime SVG 集已冻结：
  - 建筑 5 个
  - 资源 3 个
  - 弟子 HP 状态头像 4 个
- `F-004` 中的命名、尺寸、锚点与导出规格已可作为后续接图合同
- 新增弟子状态资源的交付形态为 SVG master + Cocos `.meta`

本轮不外推的范围：

- 不将当前结果表述为 `sect-map-bootstrap` 已切换为贴图消费
- 不将当前结果表述为最终美术风格已锁定
- 不将当前结果表述为整个 `F-004` 已完成

## 2026-04-20 M1-B Visual Integration Update

本轮在真实 Cocos 客户端里，把 `sect-map-svg/` 的 canonical 资源真正接入了当前本地宗门地图 runtime。

运行时接图合同冻结为：

- canonical source 继续保留在：
  - `client/my-immortal-sect/assets/resources/generated-buildings/sect-map-svg/`
- 实际供 `SpriteFrame` 动态加载的稳定导入态改为：
  - `client/my-immortal-sect/assets/resources/generated-buildings/sect-map-raster/`

本轮采用 `SVG master -> PNG runtime` 的原因：

- 本地 Cocos 3.8 官方文档的图像资源链路以 `ImageAsset / SpriteFrame` 为正式支持路径，未把 SVG 作为标准图片导入合同
- 当前工程里 `.svg` 导入后会落成 `cc.Asset`，而不是可直接给 `Sprite` 使用的 `SpriteFrame`
- 因此运行时不直接依赖 `SVG -> Sprite` 的非冻结假设，而是保留 `sect-map-svg/` 作为 canonical 源，并生成同名 PNG 供 `resources.load(.../spriteFrame)` 使用

当前接入范围：

- 弟子 token：
  - 使用 `sect_disciple_normal / injured / dying / dead` 四态头像替换原圆形占位
  - 仍保留 HP 与任务 badge，保证 `build / haul / guard / repair` 可读
- 建筑表现：
  - `main_hall / disciple_quarters / warehouse / herb_garden / guard_tower` 进入地图运行态
  - `planned / supplied / constructing / damaged / disabled` 仍通过底板、色调与短状态条反馈，不扩写新玩法
- 资源点表现：
  - `spirit_wood / spirit_stone / herb` 进入运行态
  - `designated / regenerating` 继续用描边、明暗与剩余次数状态字反馈

专用 Chrome `http://localhost:7456/` 预览页已验证：

- 启动后 12 个地图贴图资源可全部完成 `SpriteFrame` 加载
- 默认初始盘已经显示主殿贴图、弟子头像贴图和资源贴图
- 人工注入 `planned / active / damaged` 建筑组合后，地图可正确显示不同建筑资源与状态反馈
- 贴图版地图下，`guard_tower attack`、`disciple attack`、`build.repaired` 等本地闭环事件仍然发生，说明本轮没有破坏既有 `建 -> 运 -> 守 -> 修` 逻辑

本轮仍保持的边界：

- 没有新增服务端范围
- 没有新增建筑种类、资源种类或玩法规则
- hostile 仍使用现有轻量 token，占位刷新优先级让位给弟子、建筑与资源点

## 2026-04-20 Experience Refresh Validation

本轮在专用 Chrome 的既有 `http://localhost:7456/` 预览页上，对贴图版宗门地图做了一轮基于现有闭环的短流程回归。

本轮已确认：

- 地图默认启动态已经不再主要依赖 `Graphics` 占位来表达主殿、弟子与资源点
- 注入 `planned / active / damaged` 建筑组合后，建筑、资源、弟子三类核心对象的视觉区分度明显高于旧版
- 在贴图版地图下，`guard_tower attack`、`disciple attack`、`build.repaired` 仍持续出现，说明视觉替换没有破坏本地 `建 -> 运 -> 守 -> 修` 循环

本轮已做的体验收口：

- 将建筑状态标签从大面积上浮信息板收缩为更短的状态条，降低对建筑本体的遮挡
- 把弟子 token 改为四态头像优先，保留 HP 与任务 badge，继续支持状态判断
- 资源点从纯色圆点升级为资源图标 + 状态数值，保留 `designated / regenerating` 识别

本轮刷新后仍阻碍体验的剩余问题：

- `hostile` 仍是旧的轻量 token，与弟子/建筑/资源的视觉语言不一致，威胁识别仍偏弱
- 顶部状态栏仍承载过多系统汇总信息，当前“下一步该做什么”的主提示容易被总览文本稀释
- 建筑受损、待施工、蓝图状态在多建筑密集摆放时仍较依赖文字条，而不是足够强的远距图形信号
- 弟子为何在 `运 / 守 / 修` 间切换，仍主要依赖顶部消息与日志推断，地图内缺少更强的任务目标指向反馈
- 当前回归仍是在浏览器预览中完成，未覆盖小游戏容器侧输入、性能与资源加载差异

## 2026-04-20 Supervisor Acceptance Note - Experience Refresh Validation

主管已接受当前“体验刷新与可玩验证”子任务完成。

本轮 accepted 内容：

- 贴图版宗门地图已完成一轮基于既有 `http://localhost:7456/` 预览页的短流程回归
- 最严重的建筑标签遮挡已在 live runtime 中收口，默认盘与注入状态盘都有可追溯验证
- `build -> haul -> guard -> repair` 本地闭环在贴图接入后仍可运行
- 当前版本阻碍体验的剩余问题已集中记录，可直接作为下一轮规划输入

本轮不外推的范围：

- 不将当前结果表述为所有体验问题都已解决
- 不将当前结果表述为 hostile 视觉刷新已完成
- 不将当前结果表述为 `F-004` 已整体完成，或 `Go + Hollywood` / `shared` 权威路径已完成

## 2026-04-20 Supervisor Acceptance Note - Visual Integration

主管已接受当前“资源接图并替换关键占位表现”子任务完成。

本轮 accepted 内容：

- canonical `sect-map-svg/` 已被稳定消费为 `sect-map-raster/` 下的 12 个 runtime PNG `SpriteFrame`
- 弟子 token、建筑表现、资源点表现已从纯占位 `Graphics` 切到状态感知的贴图优先路径
- `Graphics` fallback 仍保留，因此当前接图不会把现有本地闭环与调试路径一起破坏
- 专用 Chrome `http://localhost:7456/` 预览页已验证贴图接入后，`build -> haul -> guard -> repair` 本地循环仍可运行

本轮不外推的范围：

- 不将当前结果表述为整轮“体验刷新与可玩验证”已完成
- 不将当前结果表述为 hostile 视觉资源也已完成刷新
- 不将当前结果表述为新增玩法、节奏调校或服务端权威接入已完成

## 2026-04-20 M1-C Local Short Session Closure

本轮在真实 Cocos 客户端里，把贴图版宗门地图进一步收口为一轮可从新档跑完的本地短会话。

当前已补齐的最小闭环：

- `place / upgrade / demolish` 已进入同一张宗门地图的可玩路径
- 新档开局会生成一个 `ruined warehouse`，玩家需先拆除旧仓，再在同地块放置护山台
- 护山台支持 `Lv.1 -> Lv.2` 升级，升级物资沿用既有 `gather -> haul -> construct` 自动执行链
- 首波敌袭由短会话阶段机驱动：`clear_ruin -> place_guard_tower -> upgrade_guard_tower -> raid_countdown -> defend -> recover -> victory / defeat`
- 敌袭后，建筑受损会触发弟子优先补齐修复资源并执行 `repair`
- HUD 已接入短会话阶段、目标文本、计时与敌袭倒计时，运行时快照已暴露 `session` 结构与建筑 `level / pendingAction`

专用 Chrome `http://localhost:7456/` 预览页验证结果：

- 通过真实 runtime 组件跑完一轮新档短会话，结果为：
  - `victory`
  - 总时长约 `82.4s`
  - 阶段推进完整经过 `clear_ruin -> place_guard_tower -> upgrade_guard_tower -> raid_countdown -> defend -> recover -> victory`
- 运行证据显示：
  - `combat.hostile_spawned = 1`
  - `combat.hostile_defeated = 1`
  - `combat.building_damaged = 3`
  - `build.repaired = 1`
- 本轮验证截图：
  - `runtime/sect-map-m1c-short-session-victory.png`

当前仍保留的边界：

- 当前短会话仍是 client-local 调校闭环，不外推为 `Go + Hollywood` / `shared` 权威接入已完成
- 当前仍未补齐真正的缩放交互与小游戏容器侧平台验证
- 当前 hostile 仍沿用轻量 token 表现，视觉一致性问题仍待后续轮次处理

## 2026-04-21 M1-C Threat And State Signal Asset Package

为 `M1-C` 当前短会话，新增一批最小必要的可读资源，用于收口两个仍明显影响体验的问题：

- `hostile` 威胁识别弱
- 建筑在远距密集摆放时，`planned / supplied / constructing / damaged / disabled` 状态仍过度依赖文字条

canonical source path 继续保持：

- `client/my-immortal-sect/assets/resources/generated-buildings/sect-map-svg/`

runtime raster path 继续保持：

- `client/my-immortal-sect/assets/resources/generated-buildings/sect-map-raster/`

本轮新增 canonical basenames：

- hostile 头像：
  - `bandit_scout_normal`
  - `bandit_scout_injured`
- 建筑状态信号：
  - `building_signal_planned`
  - `building_signal_supplied`
  - `building_signal_constructing`
  - `building_signal_damaged`
  - `building_signal_disabled`

尺寸与锚点合同：

- hostile 头像：
  - master size: `96x96`
  - node anchor: `0.5, 0.5`
  - intended usage: 与现有 disciple portrait 同级，置入当前 token portrait 槽位时保持 `56~64 px` 主体脸部可读范围
- 建筑状态信号：
  - master size: `72x72`
  - node anchor: `0.5, 0.5`
  - intended usage: 作为 building art 顶部或标签近邻的小型 overlay，不替代建筑本体贴图

导出与命名合同：

- canonical 主文件仍为 `SVG`
- runtime 同名 `PNG` fallback 继续落在 `sect-map-raster/`
- basename 一律保持 snake_case，不复用已有 `building.* / resource.* / disciple.*` 资源名
- 当前子任务只补资源与合同，不把现有 runtime 静默改写为 hostile portrait 或 building signal overlay 已接入完成

## 2026-04-21 Supervisor Acceptance Note - M1-C Threat And State Signal Assets

主管已接受 `M1-C` 中“威胁识别与状态信号资源”子任务完成。

本轮 accepted 内容：

- `bandit_scout_normal / injured` 的 canonical SVG master 与 runtime PNG `SpriteFrame` 已落盘
- `building_signal_planned / supplied / constructing / damaged / disabled` 的 canonical SVG master 与 runtime PNG `SpriteFrame` 已落盘
- `F-004` 已冻结 hostile 头像 `96x96`、building signal `72x72`、anchor `0.5 / 0.5`、`SVG canonical + PNG runtime` 的命名与导出合同

本轮不外推的范围：

- 不将当前结果表述为 hostile portrait 或 building signal 已完成 live runtime 接图
- 不将当前结果表述为整个 `M1-C` 已完成
- 不将当前结果表述为小游戏容器侧 smoke 或服务端权威路径已完成

## 2026-04-21 Objective Guidance And Host Smoke

本轮把 `M1-C` 的目标导向与关键状态提示进一步收口到地图内可见反馈，并对小游戏宿主做了一次 smoke 尝试。

浏览器预览内已完成：

- 顶部状态栏收口为导向式 HUD，按 `阶段 / 目标 / 局势 / 提示` 驱动短会话推进
- 地图内已补入 `objective marker`
- 建筑状态徽记与 hostile 头像贴图已进入当前 runtime
- 专用 Chrome `http://localhost:7456/` 预览页已验证 guidance 会随
  - `clear_ruin`
  - `place_guard_tower`
  - `upgrade_guard_tower`
  - `raid_countdown`
  - `defend`
  - `victory`
  切换，且短会话仍可跑通 `victory`

小游戏容器 smoke 当前结论为 `partial`：

- 已尝试推进到抖音开发者工具宿主
- 当前受两项 blocker 限制，尚未形成可接受宿主样本：
  - 宿主停在登录页，未进入可执行地图闭环的运行态
  - 当前工作区缺少可直接送入抖音宿主的构建产物

因此，本轮可以确认：

- 浏览器预览结论未被当前代码改动推翻
- 宿主 smoke 已推进到真实 blocker，而不是完全未启动

但本轮不能确认：

- 抖音小游戏宿主中的最终输入、资源加载与 `safeArea` 结论

## 2026-04-21 Supervisor Acceptance Note - Objective Guidance And Host Smoke

主管已接受当前“地图内目标指向收口与容器 smoke”子任务完成。

本轮 accepted 内容：

- 顶部导向式 HUD、objective marker、建筑状态徽记与 hostile 头像贴图已在浏览器预览验证中收口
- 贴图版地图在导向式 HUD 下仍可跑通 `victory`
- 小游戏容器 smoke 已推进到真实宿主 blocker，并明确记录为 `partial`

本轮不外推的范围：

- 不将当前结果表述为抖音或微信小游戏宿主验证已通过
- 不将当前结果表述为 `safeArea / statusBarHeight / 资源加载` 的容器侧结论已最终确认
- 不将当前结果表述为整个 `M1-C`、`F-004` 或服务端权威路径已完成

## 2026-04-21 Supervisor Sequencing Note - Cocos Compile First

根据人类最新要求，当前阶段的验证顺序调整为：

- 当前能力首先看 `Cocos Creator` 工程内的编译与预览链路是否成立
- 专用 Chrome 预览与 runtime snapshot 继续作为当前阶段的主要验证证据
- 微信 / 抖音开发者工具与真实小游戏容器测试后置到最后
- 只有当人类明确要求切换到微信 / 抖音工具或真实容器验证时，才单独启动对应计划

因此，当前 `F-004` 的阶段口径改为：

- 已记录的小游戏宿主 smoke 尝试保留为历史探索证据
- 宿主侧 `safeArea / input / 资源加载` 不再作为当前阶段的进行中项
- 在未收到平台切换指令前，不把微信 / 抖音开发者工具测试写成默认下一步

## 2026-04-21 Supervisor Acceptance Note - M1-C Milestone

主管确认：`M1-C` 已达到“可从新档跑完的本地短会话”门槛，可作为当前 client-local 阶段的已接受里程碑。

本轮接受依据：

- 宗门地图已在真实 Cocos 客户端内补齐 `place / upgrade / demolish`、首波敌袭、repair 恢复与 `victory / defeat` 节奏
- 贴图版地图已完成高可读资源接图、目标导向 HUD、objective marker、建筑状态信号与 hostile 识别资源补强
- 专用 Chrome `http://localhost:7456/` 预览页已跑通一轮从新档开始的 `victory` 短会话，时长约 `82.4s`
- 当前版本阻碍体验的剩余问题已收敛到明确列表，并未再阻塞“短会话可完成”这一门槛

本轮接受但保留为后续问题的项：

- 小游戏容器 smoke 目前仅到抖音宿主登录门槛，结论为 `partial`
- hostile 视觉、building signal 与目标导向虽然已补入当前 runtime，但仍可继续优化一致性与远距可读性
- `Go + Hollywood` / `shared` 权威路径仍未接入

因此，本轮 acceptance 的准确表述是：

- `M1-C` 作为 **client-local 可完成短会话里程碑** 已通过
- `F-004` 作为整个大地图玩法基础盘文档仍保持 `active`

## 2026-04-21 Supervisor Acceptance Note - M1-C Local Short Session

主管已接受 `M1-C` 中“本地短会话核心循环”子任务完成。

本轮 accepted 内容：

- 真实 Cocos 客户端已补齐 `place / upgrade / demolish` 的最小可玩路径
- 一轮从新档开始的短会话已在专用 Chrome `http://localhost:7456/` 预览页跑通，结果为 `victory`，总时长约 `82.4s`
- 短会话阶段机已完成 `clear_ruin -> place_guard_tower -> upgrade_guard_tower -> raid_countdown -> defend -> recover -> victory / defeat` 收口
- `session` 结构与建筑 `level / pendingAction` 已进入 runtime debug，可作为后续验证与调参证据

本轮不外推的范围：

- 不将当前结果表述为整个 `M1-C` 已完成
- 不将当前结果表述为小游戏容器侧 smoke、hostile 视觉刷新或顶部目标指向问题已完成
- 不将当前结果表述为 `Go + Hollywood` / `shared` 权威路径已完成

### 1.1 `M1` 的输入哲学：玩家下达标记，不直接操控单位

`M1` 明确采用 **designation-driven** 设计，而不是直接点弟子下命令。

玩家在地图上的主要行为是：

- 标记采集区域
- 标记建筑蓝图
- 标记建筑拆除

弟子根据这些标记和系统规则自动轮询、自动寻路、自动执行。

这条规则是 `M1` 的核心 UX 约束：

- 玩家表达意图
- 系统生成任务
- 弟子执行任务
- 地图反馈结果

这比“玩家逐单位手操”更符合手机端视野限制，也更符合项目要借鉴的 RimWorld 式经营调度方向。

### 2. 向 RimWorld 借什么，不借什么

强借鉴部分：

- 角色/岗位驱动，而不是玩家逐单位手操
- 优先级和规则调度，而不是脚本式演出
- 事件打破稳态，而不是纯静态资源累积
- 经营结果自然转化为叙事

明确不借鉴的部分：

- 高压求生气质
- 过重的个体需求模拟
- 失控复杂度优先于可读性
- “模拟万物”式的大而全

### 3. `M1` 的地图规则包

地图采用固定中小尺寸宗门主盘，不做超大无缝地图。

逻辑层先锁定四种地块：

- `buildable`：可建造格
- `blocked`：不可通行、不可建造格
- `road`：可通行且移动效率更高
- `resource`：资源点所在格

地图同时需要以下固定逻辑锚点：

- 玩家初始主殿位置
- 资源点分布
- 外敌出生边缘
- 首波敌袭默认目标
- 护山塔或弟子默认防守集结点

### 3.1 `M1` 的移动端交互约束

`M1` 默认按手机端优先设计地图交互：

- 单指按住并拖动：移动地图
- 双指：缩放地图
- 不把“长按拖动地图”和“框选区域”绑定到同一手势

因此，采集、建造、拆除等行为不依赖“直接在地图上切换成鼠标式编辑器模式拖选”，而是通过 UI 工具态进入。

默认 UX 方案如下：

- 常驻一组地图操作按钮
- 玩家点击某个按钮后，进入对应的“标记模式”
- 在标记模式下，地图点击行为临时改写为该模式的输入语义
- 退出模式后，地图点击恢复为普通查看/选中语义

### 3.2 `M1` 的标记模式

#### 采集模式

- 玩家点击 `采集` 按钮后，进入采集标记模式
- 第一版以“单点落区”或“固定小范围落区”为主，不要求玩家用手势拖矩形框
- 如果点击点附近存在可采集资源，则生成采集标记
- 如果点击点不含可采集资源，则本次点击无效

说明：

这比在手机上强行做“拖矩形区域”更稳，因为单指拖动已经天然被用户理解为“拖地图”。

#### 建造模式

- 玩家点击 `建造` 按钮后，弹出二级建筑选择面板
- 玩家选择某个建筑后，在屏幕中心附近出现待放置蓝图
- 蓝图采用简化矩形占格，不要求所有建筑都是正方形
- 玩家在蓝图本体上按下并拖动时，视为尝试调整蓝图位置
- 拖动过程中，合法位置与非法位置必须有明显变色反馈
- 玩家点击地图其他位置，视为确认蓝图落位

#### 拆除模式

- 玩家点击 `拆除` 按钮后，点击某个现有建筑，视为添加拆除标记
- 拆除标记不会立刻销毁建筑，而是等待弟子执行

### 3.3 长按环形快捷操作

除常驻工具按钮外，`M1` 额外支持一种轻量快捷交互：

- 玩家长按某个格子
- 且长按期间没有发生明显位移
- 系统根据该格子上的对象弹出环形快捷操作

快捷操作的设计目的不是替代主工具栏，而是降低常用操作路径成本。

第一版上下文规则如下：

- 长按空白可建格：显示与建造相关的快捷入口
- 长按资源格：显示采集相关快捷入口
- 长按现有建筑：显示建筑相关快捷入口（例如拆除）

这套环形菜单必须满足两个约束：

- 不抢占“单指拖地图”的默认语义
- 只有在“长按且不移动”时才触发，避免与地图浏览手势冲突

### 4. `M1` 的最小内容包

#### 资源

- `res.spirit_wood`
- `res.spirit_stone`
- `res.herb`

三种资源只承担两类职责：

- 建筑放置、升级、修复的成本
- 局部功能解锁与防守准备

本轮不引入更长的炼丹工业链，也不做复杂市场交易。

#### 建筑

- `building.main_hall`
  - 宗门核心与失败判定中心
- `building.disciple_quarters`
  - 提供弟子容量
- `building.warehouse`
  - 提供库存容量与交付节点
- `building.herb_garden`
  - 提供稳定产出或资源补充能力
- `building.guard_tower`
  - 在敌袭中提供自动防御

第一版建筑不要求全部同尺寸，允许矩形占格。

推荐首版占格如下：

- `main_hall`: `3x3`
- `disciple_quarters`: `2x3`
- `warehouse`: `2x2`
- `herb_garden`: `2x2`
- `guard_tower`: `1x2` 或 `1x1`

建筑统一采用以下生命周期：

- `planned`
- `supplied`
- `constructing`
- `active`
- `damaged`

状态语义固定如下：

- `planned`：灰色蓝图，表示玩家已下达建造意图，但资源尚未补齐
- `supplied`：绿色蓝图，表示资源已填充完成，等待施工
- `constructing`：有施工中的可视状态
- `active`：正常建筑图标
- `damaged`：受损，需要修复

`upgrade` 与 `demolish` 只对 `active` 建筑开放。

#### 弟子

`M1` 不做完整人物系统，只做“可工作的宗门单位”。

最小行为集合：

- `idle`
- `move`
- `gather`
- `haul`
- `build`
- `repair`
- `guard`

弟子在地图上以 **头像 token** 形式出现，而不是全身角色。

原因：

- 手机屏幕可视面积有限
- 大地图上弟子首先是经营单位，不是动作主角
- 头像更适合叠加状态信息

本轮不做复杂情绪链，但需要保留最低限度的可见状态，并通过头像表达：

- `idle`
- `working`
- `injured`
- `carrying`

第一版不单独设计“守卫”头像态；敌袭出现时，默认所有可战斗弟子都把战斗响应视作最高优先事项。

#### 敌袭

`M1` 只做一类敌人原型：

- 外敌从地图边缘生成
- 沿可通行路径逼近主殿
- 护山台自动攻击
- 默认所有可战斗弟子尝试寻路参战
- 主殿或其他建筑可受损
- 战后进入修复状态

这次敌袭的作用是验证：

- 路径是否可靠
- 布局是否有意义
- 防守建筑是否有价值
- 地图状态是否会因为战斗而永久变化

### 4.1 `M1` 的寻路与占格规则

所有寻路必须严格基于 `tilemap` 逻辑格。

本项目不采用：

- 像素级自由寻路
- 运行时 navmesh
- 脱离 tile 坐标体系的“视觉走位”

第一版推荐规则：

- 基于 `tilemap` 的格子 A* 寻路
- 以地块类型和建筑占格决定可通行性
- 道路格拥有更低移动代价
- 阻挡格和建筑实体格不可穿越
- 资源点、仓库、工地、敌袭目标点都必须映射到格子坐标

如果后续需要扩展更复杂的成本场或分层阻挡，也必须建立在同一套 tile 逻辑网格之上，而不是另起第二套移动系统。

### 5. `M1` 的短会话目标

`M1` 的首个玩家目标定义为：

**把草创宗门扩成能扛住第一波袭扰的小宗门。**

推荐 5~10 分钟短会话流程：

1. 观察地图，理解资源点与可建区域
2. 放置弟子居和仓库，形成基础生产区
3. 派弟子自动采集与搬运
4. 放置或升级护山台
5. 承受一次首波敌袭
6. 修复受损区域并继续经营

如果这条链在没有实现上下文说明的情况下仍可理解，`M1` 方向就成立。

## Conflict And Impact

- 冲突对象：把大地图继续当成“只验证交互的技术样板”
  - 冲突原因：会导致地图能拖、能点、能放，但没有真正的玩法压力与运营逻辑
  - 当前裁决：`F-002` 之后必须进入 `M1` 玩法盘定义，地图要承载资源、建造、工作和敌袭
  - 后续动作：实现顺序围绕主循环，而不是围绕单点技术功能堆砌

- 冲突对象：过早把 M1 做成“大而全的修仙版 RimWorld”
  - 冲突原因：范围会立刻失控，客户端和服务端都没有稳定落点
  - 当前裁决：`M1` 只保留“建、运、守”三根主支柱
  - 后续动作：复杂人物需求、外交、长故事链和多人玩法全部后移

- 冲突对象：为了快速出效果，把资源、建造、战斗长期留在客户端做权威结算
  - 冲突原因：直接违反项目的服务端权威边界，后续回迁成本高
  - 当前裁决：允许短期 client-local smoke adapter 用于视觉验证，但不作为 accepted 方案；`M1` 验收态必须预留或接入最小 `Go + Hollywood` 权威路径
  - 后续动作：实现计划中单列“authoritative runtime hook”阶段

- 冲突对象：把手机端地图交互做成 PC 编辑器式拖框和单位直控
  - 冲突原因：会与“单指拖地图 / 双指缩放”的约定俗成冲突，导致输入语义混乱
  - 当前裁决：采集、建造、拆除全部收敛为 UI 驱动的标记模式
  - 后续动作：实现时优先保证模式切换和蓝图拖拽反馈稳定，不追求复杂手势

## Implementation Plan

### Slice 1. 地图契约与坐标底座

目标：

- 锁定 Tile 坐标、世界坐标、建筑占格、阻挡判定
- 固化地图图层命名和数据来源
- 确定资源点、道路、可建格、阻挡格的运行时读取方式

本阶段交付：

- 固定地图数据约定
- 建筑占格与选中框规则
- 路径网格生成规则
- raid spawn / target 逻辑锚点

### Slice 2. 建筑放置、升级、拆除

目标：

- 地图上稳定执行 `place / upgrade / demolish`
- 建筑状态从 `planned` 进入 `supplied`、`constructing` 再到 `active`
- 让建筑从“预览贴图”升级成“有状态实体”

本阶段交付：

- 建筑配置表
- 建筑成本规则
- 建筑状态机
- 占格与碰撞刷新
- 建造模式 UI 与蓝图拖拽确认

### Slice 3. 弟子工作循环与寻路

目标：

- 让弟子从“地图装饰”变成“自动执行任务的经营单位”
- 打通采集、搬运、施工、修理四类任务
- 让所有任务都以 tile 格寻路为基础

本阶段交付：

- 最小任务选择规则
- 地图格子 A* 寻路
- 资源点 -> 仓库 -> 工地的工作链
- 任务状态可视反馈
- 弟子头像 token 状态反馈

### Slice 4. 资源流与建筑成长

目标：

- 让资源成为地图上的真实约束
- 让建筑升级、修复和容量变化具备经营意义

本阶段交付：

- 资源库存与容量规则
- 仓库交付节点
- 建筑升级成本
- 损坏与修复成本

### Slice 5. 首次敌袭与自动防守

目标：

- 给地图经营引入第一次系统级压力
- 验证“布局是有代价和收益的”

本阶段交付：

- 外敌出生与寻路
- 护山台自动攻击
- 弟子防守响应
- 建筑受损、修复与战后恢复

### Slice 6. 权威运行时挂钩与验收

目标：

- 把 `M1` 从 client-local 演示推进到真实项目边界
- 保证资源、建造、战斗等关键状态最终不悬空在客户端

本阶段交付：

- 最小 shared state / message 定义
- 最小 `Go + Hollywood` 权威处理入口
- 客户端显示层与权威状态同步边界
- 验收脚本与最小操作路径

## Recommended Build Order

审核通过后，开发顺序固定为：

1. `Slice 1` 地图契约
2. `Slice 2` 建筑状态链
3. `Slice 3` 弟子任务与寻路
4. `Slice 4` 资源与升级
5. `Slice 5` 敌袭与防守
6. `Slice 6` 权威运行时挂钩

理由很直接：

- 没有地图契约，后续所有坐标和占格都会返工
- 没有建筑状态链，地图上不会出现真正的经营对象
- 没有弟子任务，资源流只是静态数值
- 没有敌袭，布局与升级没有压力测试
- 没有权威运行时挂钩，`M1` 只能算演示，不算项目主线资产

## Runtime Contract For Future Implementation

当前计划通过后，实际开发应默认使用以下运行约束：

- 客户端观察入口：常驻 `http://localhost:7456`
- 当前场景：`sect-map-main`
- 每次编辑后必须保存，依赖浏览器自动刷新
- 不新开重复调试页；优先复用现有业务 tab
- 若进入浏览器自动化验证，优先使用已验证的“专用 Chrome + 固定调试端口 + 会话复用”模式

## 2026-04-20 Portrait Baseline Validation Matrix

本轮把宗门地图竖屏基线从“猜一个通用手机分辨率”收敛为“编辑器作者分辨率 + 小游戏平台窗口/安全区规则 + 运行时归一化适配”。

结论先明确：

- `750x1314` 现在只作为当前 Cocos 竖屏场景的 authoring / observation baseline，不再被视为真实设备分辨率契约
- 小游戏目标输出的方向约束来自各平台 `game.json.deviceOrientation`
- 运行时布局基线来自 `windowWidth`、`windowHeight`、`safeArea`、`statusBarHeight`
- 只有当容器不给 `safeArea` 时，才允许在运行时构造 fallback safe area

### Official Rule References

- WeChat:
  - `https://developers.weixin.qq.com/minigame/dev/api/base/system/wx.getWindowInfo.html`
  - `https://developers.weixin.qq.com/minigame/dev/api/base/system/wx.getSystemInfoSync.html`
- Douyin:
  - `https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/api/device/system-information/tt-get-system-info-sync`
  - `https://developer.open-douyin.com/docs/resource/zh-CN/mini-game/develop/framework/mini-game-configuration`

### Validation Matrix

| Check | Expected rule | Evidence | Result | Notes |
|---|---|---|---|---|
| Cocos authoring baseline | 当前场景保持竖屏 authoring 分辨率 | `client/my-immortal-sect/settings/v2/packages/project.json` -> `designResolution.width=750`, `designResolution.height=1314` | pass | 这是作者分辨率，不是设备真值 |
| WeChat target output | 小游戏构建输出保持竖屏 | `client/my-immortal-sect/build-templates/wechatgame/game.json` -> `deviceOrientation=portrait` | pass | 与官方 `game.json` 方向规则一致 |
| Douyin target output | 小游戏构建输出保持竖屏 | `client/my-immortal-sect/build-templates/bytedance-mini-game/game.json` -> `deviceOrientation=portrait` | pass | 与官方 `game.json` 方向规则一致 |
| Runtime metrics adapter | 运行时提供 `windowWidth`、`windowHeight`、`safeArea`、`statusBarHeight`，且 `safeArea` 缺失时有 fallback | `client/my-immortal-sect/assets/scripts/app/runtime-screen-metrics.ts` | pass | 优先小游戏窗口信息，预览环境退回 browser / Cocos view |
| Sect-map portrait baseline | HUD 位置由安全区换算后的 view-space inset 驱动，而不是写死设备高度 | `client/my-immortal-sect/assets/scripts/app/sect-map-bootstrap.ts` | pass | 运行态已暴露 `portraitBaseline` 快照 |
| Live preview snapshot | 当前常驻预览页能读到 `screenMetrics` 与 `portraitBaseline` | 专用 Chrome 现有 `http://localhost:7456/`，`window.__MIS_RUNTIME_DEBUG__.getSnapshot()` | pass | 本地预览当前走 `browser-window` fallback |
| Safe-area fallback path | 当容器未提供原生 `safeArea` 时，debug snapshot 能明确标记 fallback 生效 | 同一 live snapshot 中 `safeAreaFallbackApplied=true` | pass | 当前本地浏览器预览不等同小游戏真实容器 |

### Current Live Evidence

2026-04-20 在专用 Chrome 复用现有 `http://localhost:7456/` 页签读取到的当前快照为：

- `viewport.designWidth=750`
- `viewport.designHeight=1314`
- `viewport.visibleWidth=750`
- `viewport.visibleHeight=1314`
- `screenMetrics.windowWidth=2880`
- `screenMetrics.windowHeight=1393`
- `screenMetrics.safeArea={ left: 0, right: 2880, top: 0, bottom: 1393, width: 2880, height: 1393 }`
- `screenMetrics.statusBarHeight=0`
- `screenMetrics.safeAreaFallbackApplied=true`
- `screenMetrics.source=browser-window`
- `portraitBaseline.safeInsets={ left: 0, right: 0, top: 0, bottom: 0 }`
- `portraitBaseline.safeFrameWidth=750`
- `portraitBaseline.safeFrameHeight=1314`
- `portraitBaseline.hud.statusY=601`
- `portraitBaseline.hud.toolbarY=-585`
- `portraitBaseline.hud.buildPanelY=-469`
- `mapReady=true`

### Boundary

- 当前记录已经证明：
  - 竖屏方向输出已固定到 WeChat / Douyin 目标模板
  - 宗门地图的运行时布局基线已改为“窗口/安全区驱动”
  - 本地预览环境下的 fallback 路径可稳定工作并可观测
- 当前记录尚未证明：
  - 微信开发者工具、抖音开发者工具或真实小游戏容器返回的原生 `safeArea` 数值
  - 刘海屏 / 打孔屏 / 异形屏在小游戏容器内的最终 UI 占位结果

## 2026-04-20 M1-A Validation Matrix

本轮把 `M1` 的当前可验收范围明确收敛为 `M1-A`：

- 只验收 `建 + 运` 两根玩法支柱
- 敌袭 / 自动防守不属于本轮必达项
- 野外资源必须是有限资源节点，而不是无限供给点
- 结果必须能从画面直接读懂，而不是依赖日志推理

### Scope Boundary

- 当前通过的是“宗门地图最小经营闭环”：
  - 看图
  - 下标记
  - 弟子自动执行
  - 资源积累
  - 建筑生效
  - 地图状态改变
- 当前尚未通过的是：
  - 首次敌袭 / 自动防守闭环
  - `Go + Hollywood` 权威接入后的验收态
  - 微信 / 抖音小游戏真实容器中的最终运行证据

### Validation Matrix

| Check | Expected rule | Evidence | Result | Notes |
|---|---|---|---|---|
| Map contract snapshot | 运行时能稳定给出宗门主盘逻辑契约 | `window.__MIS_RUNTIME_DEBUG__.getSnapshot().mapContract` | pass | 当前快照为 `14x14`、`tile=128x64`、`road=28`、`blocked=12`、`resource=6`、`buildable=150` |
| Finite resource node rule | 单个资源点存在 `remainingCharges`、`state`、`regenSeconds`，并可从有限 -> 枯竭 -> 刷新 | `resourceSummary` + `resourceNodes` + live preview tile `2,4` | pass | `spirit_wood` 节点从 `3/3` 采至 `0/3`，进入 `regenerating(9s)`，再恢复 `3/3` |
| Build + haul loop | 玩家下蓝图后，弟子会自动采集、搬运、施工直至建筑启用 | live preview `herb_garden` at `4,8` | pass | `planned -> supplied -> constructing -> active` 在一次连续观察中完成 |
| Resource accumulation | 采集资源会先进入库存，再被送往工地，而不是直接跳过库存 | HUD `库存` + runtime snapshot `stockpile` | pass | `wood`、`herb` 都先出现入库，再被搬运清空 |
| Map state change | 建筑启用后地图状态必须有直接可见变化 | 建筑标签 + `buildingCounts.active` | pass | `active` 建筑数从 `1` 增至 `2`，`药圃` 标签从 `灰图` / `施工中` 变为 `Lv.1` |
| Readability without logs | 玩家可仅从 HUD、建筑标签、资源标签和弟子角标理解状态推进 | 专用 Chrome 现有 `http://localhost:7456/` 观察结果 | pass | 顶部状态栏、建筑标签、资源余量/待刷新、弟子角标已构成最小可读反馈链 |

### Current Live Evidence

2026-04-20 在专用 Chrome 复用现有 `http://localhost:7456/` 页签完成两条本地 live 验证：

1. 有限资源节点验证
   - 目标节点：`spirit_wood @ 2,4`
   - 初始：`state=available`，`remainingCharges=3/3`
   - 枯竭：约 `10.7s` 后进入 `state=regenerating`，`remainingCharges=0/3`，UI 标签变为 `待9s`
   - 刷新：约 `9.1s` 后恢复 `state=available`，`remainingCharges=3/3`

2. `建 + 运` 最小闭环验证
   - 建筑样本：`herb_garden @ 4,8`
   - 资源标记：`spirit_wood @ 2,4`、`herb @ 3,10`
   - 观察到的建筑状态链：
     - `planned`
     - `supplied`
     - `constructing`
     - `active`
   - 建筑在约 `12.1s` 内达到 `active`
   - `buildingCounts.active` 从 `1` 变为 `2`

3. 画面内可读反馈验证
   - 顶部状态栏可显示：
     - 当前模式
     - 库存 `木 / 石 / 药`
     - 建筑统计 `已启用 / 待备料`
     - 资源统计 `可采 / 待刷 / 标记`
     - 当前消息
   - 建筑标签可显示：
     - `药圃 灰图`
     - `木0/1 药0/1`
     - `药圃 施工中`
     - `药圃 Lv.1`
   - 资源点标签可显示：
     - `3/3`
     - `待9s`
   - 弟子角标可显示：
     - `闲`
     - `行`
     - `工`
     - 资源简称，如 `木`

### Residual Boundary

- 当前 `M1-C` 结果只证明本地 client runtime 已形成可从新档跑完的一轮短会话，不等于完整 `M1`
- 当前结论仍是 client-local 节奏收口，不等于完整小游戏平台验收
- `M1-D` 当前只把“建造状态 + 资源结算 + 关键短会话快照”接入最小 authority 竖切，不等于完整战斗/采集/AI 都已服务端权威
- 当前证据主要来自专用 Chrome 下的本地预览 + 本地 authority runtime，不代表小游戏容器的完整平台验收

## 2026-04-21 M1-D Minimal Authority Bridge

本轮已把 `M1-C` 的“建造状态与资源结算”从纯 client-local 推进到了最小 `shared + Go + Hollywood` authority 路径。

当前 authority contract：

- canonical shared contract:
  - `shared/contracts/m1-authority-short-session-v1.md`
- local authority base URL:
  - `http://127.0.0.1:8787`
- current command surface:
  - `place_building`
  - `request_upgrade`
  - `toggle_demolition`
  - `collect_stockpile`
  - `deliver_build_resource`
  - `start_building_work`
  - `complete_building_work`
  - `complete_demolition`
  - `complete_repair`
  - `sync_session_progress`

当前服务端实现边界：

- `server/cmd/gameserver/` 提供本地 authority HTTP 入口
- `server/internal/slggame/authority/` 提供 Hollywood actor-backed session state
- `server/internal/slggame/gateway/` 提供 HTTP -> actor request translation

当前客户端接入边界：

- `client/my-immortal-sect/assets/scripts/net/` 新增 authority contract/client 适配层
- `sect-map-bootstrap` 启动时先 bootstrap authority snapshot，再把建筑与库存写回本地 runtime
- `gather -> dropoff` 现在通过 `collect_stockpile` 回写 authority，而不是直接改本地库存
- `place / upgrade / demolish` 与 `build/repair` 的结算节点现在通过 authority command 返回 snapshot
- `sect-map` runtime snapshot 已额外暴露 `authority.mode / connected / sessionId / lastEvent / lastError`

当前已验证的 authority 结果：

- bootstrap 后预览页建筑 id 已切到 authority 生成的 `building-*`
- 自动采集卸货会触发 `authority.snapshot_applied(reason=collect_stockpile)`，库存增长不再悬空在客户端
- 通过当前客户端 authority 适配层执行 `complete_demolition -> place_building -> deliver_build_resource -> complete_building_work -> request_upgrade -> complete_building_work`，可把短会话从 `clear_ruin` 推进到 `raid_countdown`

当前仍保持 out-of-scope：

- 外敌移动/攻击与路径搜索仍主要是 client runtime
- `defend / recover / victory / defeat` 中的战斗判定尚未整体迁入 authority；当前只通过 `sync_session_progress` 做最小关键快照镜像
- protobuf、持久化和正式网关协议仍未启动

## 2026-04-20 Code Audit Alignment

基于本轮代码审计，`F-004` 的当前实现边界应明确写成：

### Done

- 真实 Cocos 客户端中已经存在 `M1-C` 的 client-local 短会话闭环
- `Map contract snapshot`、有限资源节点规则、`gather -> haul -> build / upgrade / repair / demolish`、资源入库和地图状态变化已具备代码证据
- `clear_ruin -> place_guard_tower -> upgrade_guard_tower -> raid_countdown -> defend -> recover -> victory / defeat` 已进入同一张宗门地图的阶段机
- 当前 HUD、建筑标签、资源标签和弟子角标已足够支撑最小可读反馈，并可直接提示短会话目标与阶段

### Partial

- 地图交互中的 `拖拽 + 点击 + 稳定选中` 已有，但 `缩放` 仍未完成
- 建筑状态链已跑通 `planned -> supplied -> constructing -> active -> damaged -> repair -> active`，但更强的远距图形反馈仍待继续强化
- `warehouse`、`guard_tower`、`disciple_quarters` 当前主要还是 footprint/cost 差异，尚未形成完整专属玩法语义
- 小游戏平台原生 `safeArea` 证据仍待补样本
- authority 当前只收住 `建造状态 / 资源结算 / 关键短会话快照`；敌袭与采集节点本体仍是 client 驱动后再回写

### Missing

- 完整敌袭 / 采集 / 守御链路的服务端权威化
- protobuf / persistence / 正式网关协议
- `shared` 配置源和内容驱动资产仍未接到 authority runtime

## Implementation Status

### Done

- [x] 已完成大地图可选玩法的第一轮系统枚举
- [x] 已从枚举中收敛出 `M1` 的三根玩法支柱：`建 / 运 / 守`
- [x] 已锁定 `M1` 的最小资源、建筑、弟子任务与敌袭范围
- [x] 已明确 `M1` 必须是地图主盘玩法，而非技术样板
- [x] 已锁定 `M1` 为“标记驱动而非单位直控”的输入模型
- [x] 已完成当前移动端优先交互基线：单指拖图、点击/长按输入与 UI 模式切换
- [x] 已形成当前建筑蓝图状态链前四态：`planned -> supplied -> constructing -> active`
- [x] 已锁定弟子在地图上以头像 token 表达最低限度状态
- [x] 已锁定所有寻路严格基于 tilemap 逻辑格
- [x] 已锁定“长按且不移动”可弹出环形快捷操作，作为工具栏外的次级快捷入口
- [x] 已固定当前宗门地图竖屏 authoring baseline 为 `750x1314`，并明确其仅作为编辑器/观察基线，不作为设备真值
- [x] 已固定 WeChat / Douyin 小游戏目标输出为 `deviceOrientation=portrait`
- [x] 已接入统一 runtime screen metrics adapter，并在 `sect-map` runtime debug snapshot 中暴露 `screenMetrics` 与 `portraitBaseline`
- [x] 已把本地竖屏验证收敛为“专用 Chrome 固定观察条件 + live snapshot + 结构化记录”的矩阵化证据
- [x] 已补入 `M1-A` 的地图契约快照、有限资源节点规则与 live preview 验收矩阵
- [x] 已确认 `药圃` 样本可在一次短会话内完成 `蓝图 -> 采集 -> 入库 -> 搬运 -> 施工 -> active`
- [x] 已确认当前 HUD / 建筑标签 / 资源标签 / 弟子角标 足以支撑 `M1-A` 的最小可读反馈
- [x] 已通过代码审计确认当前真实客户端中存在 `gather / haul / build / demolish` 的 client-local 经营闭环
- [x] 已补齐 `place / upgrade / demolish` 的最小可玩闭环，并让 `upgrade` 复用既有 `gather -> haul -> build` 自动执行链
- [x] 已把首波敌袭、建筑受损、修复恢复与 `victory / defeat` 节奏收口进同一轮本地短会话
- [x] 已通过专用 Chrome 预览页验证新档可在约 `82.4s` 内完成一次 `clear_ruin -> build -> upgrade -> defend -> recover -> victory` 的真实运行流程
- [x] 已把顶部状态栏收口为 `阶段 / 目标 / 局势 / 提示` 导向式 HUD，并在地图内补入 objective marker、建筑状态信号徽记与 hostile 头像贴图
- [x] 已通过专用 Chrome 预览页与 runtime snapshot 验证目标指向会随 `clear_ruin -> place_guard_tower -> upgrade_guard_tower -> raid_countdown -> defend -> recover -> victory` 实时切换
- [x] 已在 `shared/` 中冻结 `M1-D` 最小 authority contract，并在 `server/` 中落地 `Go + Hollywood` 本地 authority HTTP 入口
- [x] 已让客户端在预览启动时消费 authority snapshot，并把 `place / upgrade / demolish`、资源扣减/返还与关键短会话快照改为消费 authority command 结果
- [x] 已通过专用 Chrome 预览页 + 当前客户端 authority 适配层验证：自动 `collect_stockpile` 会回写 authority，且 `clear_ruin -> place_guard_tower -> upgrade_guard_tower -> raid_countdown` 可由 authority command 链推进

### In Progress

- [ ] 等待项目负责人审核本计划
- [ ] 等待把当前 `defend / recover / victory / defeat` 与更完整的采集/敌袭结算继续迁入 authority

### Not Started

- [ ] `Slice 6` 权威运行时挂钩与验收

### Deferred

- [ ] 多人同步经营
- [ ] 完整外交系统
- [ ] 完整人物需求系统
- [ ] 高自由地形编辑
- [ ] 复杂战斗技能与手操战斗
- [ ] 微信 / 抖音开发者工具或真实小游戏容器测试，等待人类明确要求切换平台验证时再启动
- [ ] 原生 `safeArea / statusBarHeight / input / 资源加载` 样本补录，跟随同一平台切换计划执行

## Loop History

| Loop | Date | Stage | Summary | Output | Decision |
|---|---|---|---|---|---|
| L-001 | 2026-04-17 | Plan | 对大地图可承载玩法做系统枚举，并收敛出 `M1` 基础盘范围 | 本文档 | review |
| L-002 | 2026-04-17 | Plan Refinement | 根据移动端交互约束，补入“标记驱动、UI 模式切换、头像 token、tile 格寻路”等细则 | 本文档 | review |
| L-003 | 2026-04-17 | Plan Refinement | 补入“长按且不移动 -> 环形快捷操作”的移动端快捷交互规则 | 本文档 | review |
| L-004 | 2026-04-20 | Verify / Record | 基于微信/抖音官方方向、窗口信息与安全区规则，补齐宗门地图竖屏基线验证矩阵，并把 live preview `screenMetrics + portraitBaseline` 证据落盘 | 本文档 / `docs/project/delivery-ledger.md` / `docs/project/decision-log.md` | continue |
| L-005 | 2026-04-20 | Verify / Record | 以 `药圃` 样本和 `spirit_wood @ 2,4` 有限资源节点，补齐 `M1-A` 的 `建 + 运` 验收矩阵与 durable record，确认本地预览已具备“看图 -> 下标记 -> 弟子自动执行 -> 资源积累 -> 建筑生效 -> 地图状态改变”的最小闭环 | 本文档 / `docs/project/delivery-ledger.md` / `docs/project/decision-log.md` | continue |
| L-006 | 2026-04-20 | Audit / Record | 对照真实代码路径收敛 `M1-A` 完成度：确认 client-local `建 + 运` 闭环已具备，但 `upgrade`、`repair/guard`、`Go + Hollywood` 权威接入与 `shared` 合同仍未完成，并回写文档边界 | 本文档 / `docs/features/F-002-sect-map-playability-validation.md` / `docs/project/delivery-ledger.md` | continue |
| L-007 | 2026-04-20 | Implement / Verify / Record | 在真实 Cocos 客户端中补齐 `place / upgrade / demolish`、首波敌袭、修复恢复与 `victory / defeat` 阶段机，并验证新档短会话可在本地预览页中完整跑通 | 本文档 / `docs/project/delivery-ledger.md` / `runtime/sect-map-m1c-short-session-victory.png` | continue |
| L-008 | 2026-04-21 | Implement / Verify / Record | 在真实 Cocos 客户端中补齐顶部目标导向 HUD、地图内 objective marker、建筑状态徽记与 hostile 头像接图，并以专用 Chrome 预览页回归目标切换；小游戏容器 smoke 已尝试，但当前受“无现成抖音构建产物 + 开发者工具停在登录页”限制，尚未形成可接受宿主样本 | 本文档 / `runtime/sect-map-m1c-objective-guidance-preview.png` / `runtime/sect-map-m1c-objective-guidance-defend-clamped.png` | continue |
| L-009 | 2026-04-21 | Implement / Verify / Record | 在 `shared/` 中冻结 `M1-D` 最小 authority contract，在 `server/` 中落地 `Go + Hollywood` authority session actor + HTTP gateway，并让真实 Cocos 客户端改为消费 authority snapshot / command 结果；专用 Chrome 预览页已验证自动 `collect_stockpile` 与 `拆 -> 建 -> 升 -> raid_countdown` authority 命令链 | 本文档 / `shared/contracts/m1-authority-short-session-v1.md` / `server/` / `runtime/sect-map-m1d-authority-raid-countdown.png` | continue |

## Open Questions

- [ ] `M1-D` 之后，下一轮 authority 优先扩到“敌袭结算”还是“采集节点/资源刷新”
- [ ] 采集标记首版采用“单点落区”还是“固定小范围落区”，以便既适配手机又便于玩家理解
- [ ] `M1-C` 之后，下一步是先把敌袭/建造扣减接入最小权威路径，还是先继续做地图内的可读性和新手引导强化

## Related Issues / ADRs / Plans

- `docs/features/F-002-sect-map-playability-validation.md`
- `docs/plans/phase-1-sect-map-validation.md`
- `docs/plans/m0-vertical-slice.md`
- `docs/vision/design-decisions.md`
- `docs/project/decision-log.md`
