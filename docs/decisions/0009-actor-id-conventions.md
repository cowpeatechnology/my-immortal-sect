# ADR 0009: Actor ID / Player ID 命名与持久化规范

**状态**: 已确认 (Accepted)
**日期**: 2026-04-15
**决策者**: 项目负责人 + Codex
**相关文档**:
- `docs/decisions/0007-hollywood-actor-framework.md`
- `docs/decisions/0008-save-protocol-and-pgstore-schema.md`
- `docs/decisions/0010-offline-deterministic-simulation.md`
- `AGENTS.md` §4 / §11

---

## Context

SlgGame 有三种 ID 体系互相纠缠，必须先理清：

1. **业务 ID**：玩家账号 ID、弟子在游戏里的身份 ID、建筑实例 ID、storylet 实例 ID
2. **Actor PID**：Hollywood 框架用来寻址 actor 的标识，格式 `name/id`，具备 location transparency
3. **配置表 ID**：弟子模板 ID、建筑模板 ID、storylet 模板 ID，永不修改（已在 `AGENTS.md` 和 `design-decisions.md §4.4` 中规定）

这三套 ID 有以下关键差异：

| 维度 | 业务 ID | Actor PID | 配置表 ID |
|---|---|---|---|
| 生命周期 | 玩家 / 实体存在期间 | Actor 存活期间（可短于业务实体） | 配置生效期间（接近永久） |
| 唯一性范围 | 全局 | 单 Engine 或集群 | 全局 |
| 可见性 | 存档 / 日志 / 客服 | 服务端内部 | 代码 / 配置表 |
| 生成时机 | 玩家注册 / 实体创建 | Actor spawn | 策划配置 |
| 重启后是否变化 | 不变 | **可能变化** | 不变 |

特别危险的是 **Actor PID 在重启后可能变化**：同一个业务实体（一个玩家的宗门）如果被 Poison 后重 Spawn，PID 可能不是同一个字符串——这和 Erlang / Akka 的 PID 语义一致。如果我们的代码假设"PID = 业务身份",重启后会出现"找不到自己"的 bug。

另一个危险点：ADR 0010 定义弟子 / 建筑 / storylet **不是独立 actor**，但它们依然需要业务 ID（在 State 里互相引用）。这些 ID 不能和 PID 混为一谈。

最后，`design-decisions.md` 技术栈包含"微信 / 抖音小游戏"，这意味着 player_id 必须能从平台 openid 映射过来，不能是完全自由生成。

本 ADR 把这三套 ID 的格式、生成、持久化、跨重启恢复规则钉死。

---

## Decision

三种 ID 各自遵守以下规则。

### 1. 业务 ID

#### 1.1 `player_id`（玩家账号）

**格式**: `p_<platform>_<opaque>`

**例子**:
- `p_wx_oAbCdEfG1234567890abc` （微信 openid 映射）
- `p_dy_abcdef1234567890` （抖音 openid 映射）
- `p_dev_0000000001` （本地开发 / 测试账号）

**生成规则**:
- `platform` ∈ `{wx, dy, dev, test}`，长度固定 2
- `opaque` 是平台 openid 经过 **hash + base62** 后的结果，不直接暴露原始 openid
- hash 用 SHA-256，取前 16 字节 base62 编码 → 约 22 字符
- dev/test 账号用自增数字，格式 `p_dev_<10 位数字>`

**存储**:
- 主键字段 `player_state.player_id TEXT PRIMARY KEY`
- openid → player_id 的映射存 `platform_bindings` 表（不在本 ADR 定义，属于 auth 模块）

**约束**:
- **永不复用**。玩家注销后 player_id 进入 `player_deleted`（见 ADR 0008），30 天后只清数据不清 ID
- **永不修改**。即使同一人换号也是新 player_id

#### 1.2 `disciple_id`（弟子实例）

**格式**: `d_<player_id_suffix>_<seq>`

**例子**: `d_oAbCdEfG_0001`, `d_oAbCdEfG_0042`

**生成规则**:
- `player_id_suffix` = player_id 的最后 8 位（避免 disciple_id 过长）
- `seq` 是该玩家的弟子自增序号，从 1 开始，**永不回收**（死亡弟子的 ID 不会被新弟子复用）
- `seq` 存在 State 里（`state.next_disciple_seq`），每次创建弟子递增
- `seq` 用 4 位补零打印（当 seq > 9999 时扩展到 5 位）

**约束**:
- 全局唯一（跨玩家不冲突，因为前缀含 player_id）
- 死亡弟子的 ID 继续存在于存档历史中，**永不复用**
- 弟子 ID 会出现在客服日志 / 因果事件引用 / 弟子日记里，必须稳定

#### 1.3 `building_id`（建筑实例）

**格式**: `b_<player_id_suffix>_<seq>`

**例子**: `b_oAbCdEfG_0001`

**生成规则**: 同 disciple_id，用独立的 `state.next_building_seq`

**约束**:
- 拆除建筑后 ID 归档到 `state.history.buildings_demolished`，不复用
- 建筑升级不变 ID（L1 → L2 → L3 是同一个 building_id）

#### 1.4 `storylet_instance_id`（storylet 运行实例）

**格式**: `s_<category>_<player_id_suffix>_<seq>`

**例子**:
- `s_k_oAbCdEfG_0007` （因果）
- `s_d_oAbCdEfG_0003` （神降，V2 才用）

**生成规则**:
- `category` ∈ `{k, d}`（karma / descent）
- seq 来自 `state.next_storylet_seq`
- 这是"某次具体运行的 storylet"，不是 storylet 模板

**约束**:
- 完成后归档进 `state.history.storylets_completed`
- 不复用
- 与配置表 ID（`karma.xxx_yyy`）严格区分：instance_id 是运行时身份，template_id 是模板引用

### 2. Actor PID

#### 2.1 核心原则

- Hollywood PID 是**框架内部地址**，不是业务身份
- 业务代码**不得**把 PID 当成持久化 key 写入 State 或存档
- 业务代码**不得**用 PID 比较实体相等
- Actor 层可以缓存 PID 做消息派发，但必须支持 "PID 失效后通过业务 ID 重新查找" 的恢复路径

#### 2.2 Actor 命名规范（Hollywood `WithID` 传入的值）

| Actor 类型 | PID Name | 示例 |
|---|---|---|
| GatewayActor | `gateway` | `gateway/0` |
| PlayerSupervisor | `player_supervisor/<player_id>` | `player_supervisor/p_wx_oAbCdEfG...` |
| SimulationActor | `simulation/<player_id>` | `simulation/p_wx_oAbCdEfG...` |
| TickActor | `tick/<player_id>` | `tick/p_wx_oAbCdEfG...` |
| PersistenceActor | `persistence/<player_id>` | `persistence/p_wx_oAbCdEfG...` |
| SyncActor | `sync/<player_id>` | `sync/p_wx_oAbCdEfG...` |

**约束**:
- Actor name 里**必须包含 player_id**，这样同名 actor 不会误发到别人身上
- Hollywood 的 `WithID(name)` 在相同名字重复 Spawn 时返回 `ErrActorAlreadyExists`，这是我们依赖的"幂等 Spawn"行为
- **禁止**在 PID 里塞非业务 ID 信息（如时间戳、随机 suffix），那会让幂等失效

#### 2.3 重启后的 PID 恢复

玩家登出后 PlayerSupervisor Poison，所有子 actor 回收。再次登录时：

```
GatewayActor.OnLogin(player_id):
    name := "player_supervisor/" + player_id
    pid, err := engine.SpawnChild(NewPlayerSupervisor, name, WithID(name))
    if err == ErrActorAlreadyExists:
        pid = engine.Registry.Get(name)  # 复用已有的
    return pid
```

因为 PID name 是确定性的（基于 player_id），登录后能稳定对回同一棵 actor 树。重启后的 PID **字节级可能不同**（内存地址不一样），但**业务上等价**。

**业务代码应当**：用 name 而不是 PID 结构做持久化引用。例如 Gateway 的路由表应当是 `map[player_id]*actor.PID`，每次登录重建，不是持久化保存。

#### 2.4 禁止事项

- ❌ 不要把 PID 序列化到 protobuf 存档
- ❌ 不要把 PID 作为 SQL 主键
- ❌ 不要跨节点传递 PID 的 Go 指针（只传 name 字符串）
- ❌ 不要在日志里打印 PID 结构，打 name 就够了

### 3. 配置表 ID

已经在 `docs/process/engineering-standards.md` 规定，本 ADR 只做复述和补充：

| 类别 | 格式 | 示例 |
|---|---|---|
| 建筑模板 | `building.<name>` | `building.main_hall` |
| 资源 | `res.<name>` | `res.herb` |
| 岗位 | `job.<name>` | `job.farm_herb` |
| 境界 | `realm.<name>` | `realm.foundation` |
| 因果 storylet | `karma.<chain>_<stage>` | `karma.blood_moon_intro` |
| 神降 storylet | `descent.<world>_<scene>` | `descent.furnace_gate_0` |
| 弟子模板 | `disciple.<archetype>` | `disciple.rare_sword_prodigy` |
| 敌人模板 | `enemy.<name>` | `enemy.wild_boar` |
| 特质 | `trait.<name>` | `trait.hot_blooded` |
| 事件触发器 | `trigger.<name>` | `trigger.dawn_raid` |

**约束（重申 + 补充）**:
- **永不修改**：已进主分支的 ID 不得改拼写、不得改语义
- **永不删除**：下线走 `deprecated: true` 标记 + 迁移脚本处理在途实例，6 个月后才从配置里移除
- **前缀是物理命名空间**：不同类别的 ID 不得混用前缀
- **下划线分词**：一律 `snake_case`，禁止 camelCase 或 kebab-case
- **全局唯一**：同类别内 ID 不得重复
- **CI 校验**：配置表加载时校验所有 ID 符合格式，并检测重复

---

## 三套 ID 的协作

### 从 PID 到存档的完整调用链

```
客户端发 CreateDiscipleMsg{template: "disciple.rare_sword_prodigy"}
    ↓
GatewayActor (PID: gateway/0)
    ↓ 根据 session 查 player_id = "p_wx_oAbCdEfG..."
    ↓ 找到 PlayerSupervisor (PID: player_supervisor/p_wx_oAbCdEfG...)
    ↓
SimulationActor (PID: simulation/p_wx_oAbCdEfG...)
    ↓ advance(state, CreateDiscipleInput{template: "disciple.rare_sword_prodigy"})
    ↓
simulation.Advance():
    ├─ 读 config.Disciples["disciple.rare_sword_prodigy"]
    ├─ seq := state.NextDiscipleSeq; state.NextDiscipleSeq++
    ├─ new_id := fmt.Sprintf("d_%s_%04d", state.PlayerIDSuffix, seq)
    ├─ state.Disciples[new_id] = NewDiscipleFromTemplate(template, new_id)
    ├─ state.Events = append(state.Events, DiscipleCreated{id: new_id, template: ...})
    └─ return state, events
    ↓
SimulationActor 把 state 和 events 交给 PersistenceActor 写入 PG
    ↓ player_state 表（state_blob 包含 new_id）
    ↓ player_events 表（记录 DiscipleCreated 事件）
```

可以看到三套 ID 在一次调用里**各司其职**：
- **PID**（`simulation/p_wx_oAbCdEfG...`）是"消息送到哪里"
- **player_id**（`p_wx_oAbCdEfG...`）是"这是哪个玩家"
- **disciple_id**（`d_oAbCdEfG_0001`）是"这是哪个弟子"
- **template ID**（`disciple.rare_sword_prodigy`）是"从哪个配置模板创建"

任何把其中两个混用的代码都是错的。

---

## Consequences

### 正面影响

1. **重启和集群化天然友好**
   - PID name 确定性 → 重启后对回同一棵树
   - 未来分片后 `player_supervisor/<player_id>` 直接映射到某个节点

2. **客服和日志友好**
   - 客服拿到玩家投诉时，player_id + disciple_id 就能定位
   - 日志里的 ID 稳定、可追溯、不会"重启后消失"

3. **存档不依赖运行时状态**
   - PID 不进 State，重启后存档仍然完整
   - 迁移脚本不需要重写 PID 引用

4. **防止"混用 ID 的隐蔽 bug"**
   - 命名前缀（`p_` / `d_` / `b_` / `s_` / `building.` / `res.`）一眼可辨
   - 误用会在 CI 校验时暴露

5. **业务 ID 永不复用的保证**
   - 死亡弟子的 ID 保留 → 因果系统可以在若干年后引用"那位陨落的剑痴"
   - 配置 ID 永不复用 → 老存档始终能引用到正确的模板或 deprecated 标记

### 负面影响

1. **player_id 含 platform 前缀后不能跨平台合并账号**
   - 微信账号和抖音账号即使是同一个人也是两个 player_id
   - V1 不支持账号合并，未来需要做"账号关联"功能而不是"ID 统一"

2. **disciple_seq 只增不减**
   - 长时间游玩的玩家 seq 可能增长到 5 位、6 位
   - 实际影响小（字符串长度增加 2~3 字节）

3. **PID name 里含 player_id 导致长字符串**
   - `simulation/p_wx_oAbCdEfGhIjKlMnOpQrStUv` ≈ 40 字符
   - Hollywood mailbox key 走字符串 map，性能影响极小
   - 日志稍微冗长，可接受

4. **配置表 ID 命名规范严格**
   - 策划新增内容必须遵守命名，需要工具校验
   - CI 加 linter 的一次性成本

### 中性影响

- **与 Hollywood 的 `ErrActorAlreadyExists` 语义强耦合**：这是 Hollywood 的设计，我们顺水推舟
- **openid hash 用 SHA-256**：不是因为安全，而是为了"不把原始 openid 写入日志"的合规考虑

---

## Alternatives Considered

### A. 使用纯 UUID 作为所有 ID

- ✅ 生成简单，无需全局协调
- ❌ 可读性差，客服定位困难
- ❌ 无前缀区分，类别混淆风险高
- ❌ 长度固定 36 字符，浪费
- **裁定**: ❌ 不采用

### B. 使用数据库自增 bigint

- ✅ 紧凑、性能好
- ❌ 要一次数据库往返才能生成 ID
- ❌ 在 State 里拿不到 ID 之前无法推进 advance()，破坏纯函数约束
- ❌ 不利于跨服迁移
- **裁定**: ❌ 不采用

### C. 雪花 ID（Snowflake）

- ✅ 全局唯一无需协调
- ❌ 含时间戳 → 确定性模拟里是污染源（ADR 0010 禁用 wall clock）
- ❌ 要为每个 shard 分配 worker_id，增加运维负担
- **裁定**: ❌ 不采用

### D. 把 PID 当业务 ID

- ✅ 少一个映射层
- ❌ 重启后 PID 变化，业务引用全部失效
- ❌ 存档里存 PID 是绝对不能碰的错误
- **裁定**: ❌ 不采用，这是 actor 新手最容易犯的错

### E. player_id 用原始 openid

- ✅ 零映射成本
- ❌ 原始 openid 进日志和存档有合规问题
- ❌ openid 格式由平台定义，不同平台结构不一致
- ❌ 平台更新 openid 规则时会冲击我们的 ID 空间
- **裁定**: ❌ 不采用

---

## Risks

| 风险 | 严重度 | 缓解 |
|---|---|---|
| 业务代码意外序列化 PID | 🟡 中 | CI 检查 protobuf schema 中不得出现 PID 类型；code review 清单 |
| openid → player_id 映射丢失 | 🔴 高 | `platform_bindings` 表双写 + 备份；hash 可验证但不可逆 |
| disciple_seq 超过 int32 范围 | 🟢 低 | 用 int64 存，天文数字才会溢出 |
| 配置 ID 拼写错误 / 重复 | 🟡 中 | CI 校验 + 配置加载器运行时检查 + 单元测试 |
| 跨平台账号用户抱怨"两个号不合并" | 🟢 低 | V1 明确公告：账号按平台独立；未来做关联功能 |
| 重启后 PID 引用失效导致消息丢失 | 🟡 中 | Gateway 路由表每次登录重建，不做持久化；使用 name 查找，不存 PID |
| test/dev 账号泄漏到生产 | 🟡 中 | 启动时检查环境，生产环境禁止创建 `p_dev_*` ID |

---

## 使用策略（必须遵守）

### 代码纪律

1. **持久化层禁止出现 PID 类型字段**
   - Protobuf .proto 文件里不得出现 `Pid pid = 1;` 这类定义
   - State 结构体里不得有 `*actor.PID` 字段

2. **Actor 层引用其他 actor 必须通过 name 查找**
   ```go
   // ❌ 禁止
   c.simulationPID = simPID  // 持久化缓存

   // ✅ 允许
   pid := c.Engine().Registry.Get("simulation/" + playerID)
   c.Send(pid, msg)
   ```

3. **所有业务 ID 生成走 `state` 的计数器**
   - `state.NextDiscipleSeq` / `state.NextBuildingSeq` / `state.NextStoryletSeq`
   - 禁止用 `time.Now()` / UUID / 随机数生成业务 ID（违反 ADR 0010 确定性）

4. **业务 ID 比较用字符串相等**
   - 禁止 PID 结构比较
   - 禁止把 PID 序列化成字符串再比较

### CI 校验项（必须在 M0 前置完成）

- [ ] 配置表 ID 格式正则校验（按类别）
- [ ] 配置表 ID 唯一性检查
- [ ] `.proto` 文件禁止出现 PID 类型字段
- [ ] 简单 grep：`state_blob` / `player_state` 所在 Go 文件不得 import `hollywood/actor` 里的 PID 相关类型
- [ ] 生产环境启动时检查：不得存在 `player_id` 以 `p_dev_` / `p_test_` 开头的记录

### 命名示例 cheat sheet

```
✅ player_id:        p_wx_oAbCdEfGhIjKlMnOpQrStUv
✅ disciple_id:      d_pQrStUv_0001
✅ building_id:      b_pQrStUv_0012
✅ storylet_inst_id: s_k_pQrStUv_0003
✅ disciple tmpl:    disciple.rare_sword_prodigy
✅ karma tmpl:       karma.blood_moon_intro
✅ building tmpl:    building.alchemy_room
✅ PID name:         player_supervisor/p_wx_oAbCdEfGhIjKlMnOpQrStUv

❌ player_id:        12345             ← 无前缀
❌ disciple_id:      Disciple_1        ← camelCase
❌ storylet_inst_id: karma.intro       ← 这是模板 ID，不是运行时实例
❌ PID name:         simulation_12345  ← 下划线错位，且无 player 前缀
```

---

## 未决问题

- [ ] **账号关联功能**：同一人多平台账号如何关联（不合并，但共享某些信息）？V2 之后考虑
- [ ] **player_id 的防枚举攻击**：hash 后的 opaque 理论上可枚举？需要安全审计
- [ ] **Gateway 路由表的 eviction 策略**：长时间未活跃的 player_id → PID 映射何时清理？
- [ ] **集群化后的 PID 路由**：Hollywood cluster 模式下 name 到 node 的映射策略？留给 cluster ADR
- [ ] **AnonymousID**：未登录玩家（试玩模式）是否需要临时 ID？V1 不做
- [ ] **ID 可观测性**：是否给每个业务 ID 加 trace tag 方便日志关联？M1 决定

---

## 引用

- Hollywood `actor/pid.go` —— PID 结构和 name 语义
- Hollywood `actor/registry.go` —— name → PID 查找机制
- Erlang/OTP `Process Registry` —— 相同设计模式的起源
- `docs/process/engineering-standards.md` —— 配置表 ID 命名规范的源头
- `docs/legacy/02_完整开发计划.md` —— 命名规范最初版本
- SHA-256 + base62：常见的"安全 opaque ID" 套路
