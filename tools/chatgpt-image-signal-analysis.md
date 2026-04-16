# ChatGPT 生图信号分析记录

这份文档用于沉淀 ChatGPT 生图流程里的真实信号观察结果。

目标不是立即下最终结论，而是把每次 HAR、控制台录制、CDP 观察到的现象按样本编号追加下来，逐步判断：

- 哪些信号只代表“进入生图流程”
- 哪些信号只代表“中间预览图已出现”
- 哪些信号最接近“最终图已完成并可下载”
- 双候选图、失败重试、异常中断时分别会出现什么差异

当前策略：

- 先记录事实
- 再给出本次样本的暂定结论
- 明确哪些点还需要后续样本验证

---

## 样本 1

### 基本信息

- 分析日期：2026-04-16
- 输入文件：
  `/Users/mawei/MyWork/SlgGame/output/chatgpt.com.har`
- 结论状态：暂定结论，待后续样本继续验证

### 样本背景

这是一次手动触发的真实生图流程 HAR 录制。

这次样本的价值在于：

- 它完整覆盖了从提交 prompt 到最终图片出现的网络过程
- 它能解释此前“自动下载到未完成图，手动下载到完成图”的现象
- 它没有覆盖“双候选图 + 跳过”分支
- 它也没有覆盖“生成失败/报错”分支

因此，这个样本适合用来回答：

- 为什么之前会过早下载
- 哪个下载请求是中间图
- 哪个下载请求才是最终图

但还不能单独回答：

- 双候选图时应该如何判定完成
- 报错重试时应该如何判定失败或继续等待

### 关键时间线

以下时间相对基准点：

- 基准点：`POST /backend-api/f/conversation`
- 时间偏移：相对于真正提交生图请求的时间

#### 0.000s

- 提交生图请求：
  `POST /backend-api/f/conversation`

这一步代表用户 prompt 已真正送入 ChatGPT 的生图流程。

#### 0.047s

- 埋点：
  `Composer Create Image Button Disabled`

这说明前端进入“提交后不可再次点击”的忙碌状态，但不代表生图完成。

#### 0.673s 到 0.675s

- 埋点：
  `chatgpt_conversation_turn_turn_exchange_started`
- 埋点：
  `Generate Completion`

这两个更接近“会话轮次开始”和“前端发起生成”的确认信号，仍然不代表出图完成。

#### 约 6.27s

- 埋点：
  `chatgpt_conversation_turn_turn_exchange_complete`
- 埋点：
  `ChatGPT Convo Stream: Stream Completed`

这一步很关键。

它只表示：

- SSE 文本流结束
- 工具调用前半段已经结束
- 前端拿到了“正在处理图片”的提示卡片

它不表示最终图片已经完成。

根据本样本，SSE 结束时距离最终图出现仍有大约 13 秒。

#### 约 10.26s

- 请求：
  `GET /backend-api/files/download/file_...`

返回 JSON 中的文件名为：

```text
user-Mfe4C2K78MIy99SrTnPuXbPL/f19eaffa-7df8-43d7-84f9-af463abc0242.part0.png
```

这是本次样本里第一次明确出现的图片下载地址。

这里最重要的信息不是 `download_url`，而是 `file_name` 中带有：

```text
.part0.png
```

这说明它大概率不是最终图，而是中间阶段产物。

#### 约 11.69s

- 请求：
  `GET /backend-api/estuary/content?id=...`

成功取回上一条 `.part0.png` 的 PNG 内容。

结合用户之前肉眼观察到的现象，本次样本强烈支持下面这个判断：

- 第一张被自动流程抓到的图，实际上就是中间预览图
- 这也是之前“output 目录图未完成，但手动下载图已完成”的主要原因

#### 约 19.06s

- 再次出现：
  `GET /backend-api/files/download/file_...`

返回 JSON 中的文件名为：

```text
user-Mfe4C2K78MIy99SrTnPuXbPL/b147e67f-0b6b-497d-a568-eeb14b950aec.png
```

注意这里已经不再带 `.part0` 或 `.partN`。

这一步是本次样本里最接近“最终图已准备好”的网络信号。

#### 约 19.93s

- 请求：
  `GET /backend-api/estuary/content?id=...`

成功取回最终 PNG 内容。

从“可实际下载、可实际落盘”的角度看，本次样本里最稳妥的完成点是这一刻，而不是更早的 SSE 结束。

#### 约 19.13s

- 埋点：
  `Share Post: Share Button Shown`

它的出现时间与最终图的出现非常接近。

这说明前端 UI 侧也在接近同一时刻认定：

- 图片已经达到可展示、可分享、可操作状态

这个埋点可以作为辅助信号，但暂时不建议单独依赖它作为唯一完成条件。

#### 约 21.08s

- 请求：
  `POST /backend-api/conversation/<id>/async-status`
- 返回：
  `{"status":"OK"}`

这个信号出现得比最终图还晚。

因此它不适合作为“最早完成判定”。

### 样本 1 的核心结论

#### 结论 1

SSE 结束不等于最终图片完成。

在本样本里：

- SSE 在约 `6.27s` 结束
- 最终图在约 `19.93s` 才真正取回

两者之间有明显时间差。

#### 结论 2

`files/download` 可能会出现不止一次。

在本样本里出现了两次：

- 第一次对应 `.part0.png`
- 第二次对应最终 `.png`

因此“只要看见第一张图就下载”是错误策略。

#### 结论 3

`.part0.png` 很可能是中间预览图，而不是最终图。

这与此前实际观察到的现象一致：

- 自动下载的图像不完整
- 手动下载的最终图正常

#### 结论 4

当前最可靠的网络级候选完成规则是：

1. 等待 `files/download`
2. 读取返回 JSON 中的 `file_name`
3. 如果文件名包含 `.part`，忽略该结果
4. 只接受不带 `.part` 的最终文件
5. 再等待对应 `estuary/content` 成功返回 PNG 内容
6. 之后才允许真正保存到输出目录

### 样本 1 中不可靠或不足够的信号

以下信号在本样本里不适合直接拿来判定“最终图已完成”：

- `conversation_async_status: 7`
  只能说明进入异步生图阶段
- `ChatGPT Convo Stream: Stream Completed`
  只说明 SSE 阶段结束
- `chatgpt_conversation_turn_turn_exchange_complete`
  只说明会话流结束，不代表最终图片已落地
- `/conversation/<id>/async-status`
  出现太晚，不是最早完成信号

### 样本 1 中可作为辅助信号的内容

- `Composer Create Image Button Disabled`
  可视为开始进入生图忙碌态
- `Generate Completion`
  可视为提交成功后的前端埋点
- `Share Post: Share Button Shown`
  可视为最终图已接近可操作态的 UI 侧信号

但这些都更适合作为辅助验证，不适合单独决定下载时机。

### 样本 1 暂定工作假设

在没有更多异常样本之前，先采用以下工作假设：

- 图片下载最早可判定点不在 SSE 流结束时
- 需要等待 `files/download`
- 并且要区分中间文件与最终文件
- `.part0.png` 属于中间产物，不能直接当最终输出

### 样本 1 未覆盖的场景

后续必须继续补样本验证以下情况：

- 双候选图出现后，网络上是否会出现两个“最终文件”
- 用户点击“跳过”前后，是否会重新触发新的最终文件下载
- 生图失败时，是否仍会出现 `.part0` 但永远没有最终 `.png`
- 失败重试时，是否会出现多轮 `.part0 -> 最终图` 序列
- 右上角动画消失、UI 可点击变化、网络下载完成之间的先后关系是否稳定

---

## 样本 2

### 基本信息

- 分析日期：2026-04-16
- 输入文件：
  `/Users/mawei/MyWork/SlgGame/output/chatgpt.com1.har`
- 结论状态：暂定结论，作为样本 1 的补强验证

### 样本背景

这是第二次手动触发的真实生图流程 HAR 录制。

本样本与样本 1 的相同点：

- 都是 `picture_v2` 生图流程
- 都出现了两次 `files/download`
- 都先出现一个 `.part0.png`
- 都在之后才出现最终 `.png`
- 都在最终文件附近出现 `Share Post: Share Button Shown`

本样本与样本 1 的不同点：

- 这次 prompt 为“生成一张男性图片”
- SSE 流持续时间明显更长
- 最终目标尺寸不同，这次在 SSE 中可见尺寸参数为 `1024x1792`

这意味着：

- “先出 `.part0`，后出最终 `.png`”并不是某一次偶发现象
- 至少在两次不同请求中，这条规律都成立
- 但 SSE 的结束时间并不稳定，不能用固定时长推断图片完成

### 关键时间线

以下时间相对基准点：

- 基准点：`POST /backend-api/f/conversation`
- 时间偏移：相对于真正提交生图请求的时间

#### 0.000s

- 提交生图请求：
  `POST /backend-api/f/conversation`

#### 0.065s

- 埋点：
  `Composer Create Image Button Disabled`

与样本 1 一样，这只能说明进入忙碌态，不代表最终图完成。

#### 0.933s 到 0.949s

- 埋点：
  `chatgpt_conversation_turn_turn_exchange_started`
- 埋点：
  `Generate Completion`

与样本 1 一样，这更接近“已成功发起会话轮次”的起点信号。

#### 约 28.06s

- 埋点：
  `chatgpt_conversation_turn_turn_exchange_complete`
- 埋点：
  `ChatGPT Convo Stream: Stream Completed`

这里和样本 1 有一个重要差异：

- 样本 1 中 SSE 大约在 `6.27s` 结束
- 样本 2 中 SSE 大约在 `28.06s` 才结束

这说明 SSE 持续时间本身波动很大。

但即便如此，本样本仍然说明：

- SSE 结束依然不等于最终图片完成

因为最终图片还要更晚才出现。

#### 约 37.36s

- 请求：
  `GET /backend-api/files/download/file_...`

返回 JSON 中的文件名为：

```text
user-Mfe4C2K78MIy99SrTnPuXbPL/9d4eb738-d629-42df-9224-a7cb3e50833a.part0.png
```

这再次验证了样本 1 的关键现象：

- 第一张可下载图片不是最终图
- 仍然是带 `.part0` 的中间产物

#### 约 38.05s

- 请求：
  `GET /backend-api/estuary/content?id=...`

成功取回上一条 `.part0.png` 的 PNG 内容。

这说明：

- 中间图不仅有下载地址
- 前端也真的会把这个 `.part0` 对应的 PNG 内容拉下来

因此，如果自动化只盯“第一张图片内容成功返回”，仍然会过早下载。

#### 约 47.14s

- 再次出现：
  `GET /backend-api/files/download/file_...`

返回 JSON 中的文件名为：

```text
user-Mfe4C2K78MIy99SrTnPuXbPL/60f48c66-f127-436b-b3f6-9e18e9aac48e.png
```

这一步再次对应最终文件，而不是 `.part0`。

#### 约 47.21s

- 埋点：
  `Share Post: Share Button Shown`

和样本 1 一样，它与最终图出现时间高度接近。

这进一步支持：

- `Share Post: Share Button Shown` 可以作为“最终图可操作态”的辅助信号
- 但它仍然更适合做旁证，而不是唯一完成条件

#### 约 47.98s

- 请求：
  `GET /backend-api/estuary/content?id=...`

成功取回最终 PNG 内容。

这仍然是本样本里最稳妥的网络级完成点。

#### 约 48.21s

- 请求：
  `POST /backend-api/conversation/<id>/async-status`
- 返回：
  `{"status":"OK"}`

和样本 1 一样，这个信号依然比最终图片落地更晚。

### 样本 2 的核心结论

#### 结论 1

样本 1 中观察到的 `.part0 -> 最终 .png` 两阶段下载，在样本 2 中再次出现。

这使得该规律从“单次现象”升级为“已有两次样本支持的工作假设”。

#### 结论 2

SSE 流结束时间波动很大，但无论它早还是晚，都不能等同于最终图完成。

在本样本里：

- SSE 结束时间约 `28.06s`
- 最终 PNG 内容真正返回约 `47.98s`

二者之间仍有接近 20 秒差距。

#### 结论 3

不能用文件大小判断最终图。

本样本中：

- `.part0.png` 的 `file_size_bytes` 为 `2324594`
- 最终 `.png` 的 `file_size_bytes` 为 `2119031`

也就是说，中间图甚至可能比最终图更大。

因此：

- 文件大小不是可靠判定依据
- 文件名里的 `.part` 标记比文件大小更可靠

#### 结论 4

“第一张图片内容已成功拉取”仍然不是完成条件。

因为本样本再次证明：

- 第一张成功拉取的 PNG 内容对应的还是 `.part0.png`

### 样本 2 对当前工作假设的影响

样本 2 没有推翻样本 1 的工作假设，反而强化了它：

1. 不能用 SSE 结束判定完成
2. 不能用第一张图片出现判定完成
3. 不能用文件大小判定完成
4. 目前最可靠的网络级规则仍然是：

   - 等待 `files/download`
   - 读取返回中的 `file_name`
   - 忽略所有带 `.part` 的文件
   - 只接受不带 `.part` 的最终文件
   - 再等待对应 `estuary/content` 成功返回 PNG 内容

### 样本 2 未覆盖的场景

本样本依然没有覆盖以下情况：

- 双候选图出现后，是否会出现两个不带 `.part` 的最终文件
- 用户点击“跳过”后，最终文件是否会重新生成或重新下载
- 生成失败时，是否只停留在 `.part0` 阶段
- 错误恢复时，是否会出现多轮中间图和最终图序列

---

## 样本 3

### 基本信息

- 分析日期：2026-04-16
- 输入文件：
  `/Users/mawei/MyWork/SlgGame/output/chatgpt.com2.har`
- 结论状态：继续补强正常路径样本

### 样本背景

这是第三次手动触发的真实生图 HAR 录制。

本次 prompt 是：

```text
生成一个小狗
```

这次样本的主要价值不是引入新分支，而是验证你刚才的肉眼观察：

- 你看到完整图片后，几乎立刻看到了图片请求记录
- 然后又看到了 `async-status`

从 HAR 时间线看，这个观察是成立的。

### 关键时间线

以下时间相对基准点：

- 基准点：`POST /backend-api/f/conversation`

#### 0.000s

- 提交生图请求：
  `POST /backend-api/f/conversation`

#### 0.066s

- 埋点：
  `Composer Create Image Button Disabled`

#### 0.555s 到 0.610s

- 埋点：
  `chatgpt_conversation_turn_turn_exchange_started`
- 埋点：
  `Generate Completion`

#### 约 11.81s

- 埋点：
  `chatgpt_conversation_turn_turn_exchange_complete`
- 埋点：
  `ChatGPT Convo Stream: Stream Completed`

这次 SSE 时长介于样本 1 和样本 2 之间，再次说明：

- SSE 结束时间本身不稳定
- 不能用 SSE 结束当图片完成条件

#### 约 19.38s

- 请求：
  `GET /backend-api/files/download/file_...`

返回文件名：

```text
user-Mfe4C2K78MIy99SrTnPuXbPL/c01db4c6-9003-4af3-9a03-054d9a834bd3.part0.png
```

第三次确认：

- 第一张下载地址仍然是 `.part0.png`
- 正常路径下，中间图先出现的模式具有重复性

#### 约 20.77s

- 请求：
  `GET /backend-api/estuary/content?id=...`

成功取回 `.part0.png` 的 PNG 内容。

#### 约 22.35s

- 请求：
  `GET /backend-api/files/download/file_...`

返回文件名：

```text
user-Mfe4C2K78MIy99SrTnPuXbPL/db140860-fac9-4c8d-ab64-10e011a19dd9.png
```

这一步对应最终图。

#### 约 22.41s

- 埋点：
  `Share Post: Share Button Shown`

与最终图下载地址出现几乎同步。

#### 约 23.46s

- 请求：
  `GET /backend-api/estuary/content?id=...`

成功取回最终 PNG 内容。

这一步仍然是网络级最稳妥的完成点。

#### 约 23.69s

- 请求：
  `POST /backend-api/conversation/<id>/async-status`
- 返回：
  `{"status":"OK"}`

这一步发生在最终 PNG 内容取回之后约 `0.23s`。

### 样本 3 的核心结论

#### 结论 1

你刚才的肉眼观察与 HAR 一致：

- 先看到最终图片内容请求
- 然后几乎立刻看到 `async-status`

从网络时序看，这个“几乎立刻”大约是几百毫秒量级，而不是领先于最终图。

#### 结论 2

第三次样本继续支持同一条主假设：

- `.part0.png` 是中间图
- 正常 `.png` 才是最终图
- `async-status` 更像最终图出现后的收尾确认

#### 结论 3

在正常路径中，`async-status` 目前的定位可以进一步收敛为：

- 它很可能是“最终图已展示后”的后验状态同步
- 它不是“最早完成判定信号”
- 但它可以作为一个比较靠后的辅助确认点

### 三个正常样本合并后的阶段判断

截至样本 3，三次正常路径都呈现出同一阶段序列：

1. 提交 `POST /backend-api/f/conversation`
2. SSE 流返回“正在处理图片”相关内容
3. SSE 结束
4. 第一轮 `files/download`，文件名带 `.part0.png`
5. 第一轮 `estuary/content`，拉取中间图内容
6. 第二轮 `files/download`，文件名为最终 `.png`
7. 第二轮 `estuary/content`，拉取最终图内容
8. `Share Post: Share Button Shown`
9. `async-status -> {"status":"OK"}`

需要注意：

- 第 8 步和第 9 步的相对顺序在细节上可能靠得很近
- 但第 9 步到目前为止始终没有早于最终图内容拉取

### 现阶段最稳的工作结论

在正常路径里，当前最稳的完成判断仍然是：

1. 等待 `files/download`
2. 读取 `file_name`
3. 忽略一切带 `.part` 的文件
4. 等到不带 `.part` 的最终 `.png`
5. 再等待对应 `estuary/content` 成功返回

`async-status -> {"status":"OK"}` 当前只建议作为辅助收尾信号，不建议作为主判定条件。

### 样本 3 仍未覆盖的场景

到目前为止，仍然缺失以下关键样本：

- 双候选图出现时的网络分支
- 点击“跳过”后的网络变化
- 生图报错时的网络变化
- 卡住不出图时，是否只停留在 `.part0` 或停留在 SSE 阶段

---

## 样本 4

### 基本信息

- 分析日期：2026-04-16
- 输入来源：
  `tools/chatgpt_cdp_capture_v2.mjs` 的事件驱动实时捕获日志
- 产出文件：
  `/Users/mawei/MyWork/SlgGame/output/chatgpt-capture-v2/manual-test.png`
- 结论状态：重要修正样本

### 样本背景

这次不是 HAR 离线分析，而是 V2 监听器在你手动生图时实时捕获到的网络链路。

本次的关键发现不是“成功保存图片”，而是：

- 这次正常生图流程中，没有出现 `.part0.png`
- 监听器捕获到的第一张图就是最终 `.png`
- 随后仍然出现了 `async-status -> {"status":"OK"}`

这说明此前“三次正常样本都出现 `.part0`”只能说明：

- `.part0` 是一种常见路径

但不能再写成：

- 正常路径一定先有 `.part0`

### 关键时间线

以下时间来自 V2 捕获日志：

- `23:57:42.841Z`
  检测到本轮 `POST /backend-api/f/conversation`
- `23:58:05.777Z`
  出现 `files/download`
- `23:58:06.402Z`
  `files/download` 响应解析完成，文件名为：

```text
user-Mfe4C2K78MIy99SrTnPuXbPL/c442745e-0ce4-4052-b98e-20f0dd05a69a.png
```

注意：

- 这里直接是不带 `.part` 的最终 `.png`

- `23:58:06.414Z`
  请求对应的 `estuary/content`
- `23:58:07.283Z`
  最终图片内容成功取回
- `23:58:06.858Z`
  发出 `async-status` 请求
- `23:58:08.112Z`
  监听器确认 `async-status -> {"status":"OK"}`
- `23:58:08.116Z`
  监听器将图片保存到：

```text
/Users/mawei/MyWork/SlgGame/output/chatgpt-capture-v2/manual-test.png
```

### 样本 4 的核心结论

#### 结论 1

正常路径不一定总是两阶段：

- 有些正常样本会先出现 `.part0.png`，再出现最终 `.png`
- 也有些正常样本会直接出现最终 `.png`

#### 结论 2

因此，当前更准确的规则应该是：

1. 如果出现 `.part` 文件，忽略它
2. 只接受不带 `.part` 的最终 `.png`
3. 再等待对应 `estuary/content` 成功返回
4. `async-status -> {"status":"OK"}` 作为辅助收尾信号

也就是说：

- `.part0` 是“可能出现的中间图”
- 不是“必然出现的前置阶段”

#### 结论 3

事件驱动版 V2 在这次手动测试中已经成功工作：

- 正确识别本轮生图开始
- 正确匹配最终文件
- 正确保存到指定目录
- 未误抓历史图片

### 对当前工作假设的修正

此前的工作假设需要从：

- 正常路径通常是 `.part0 -> 最终图`

修正为：

- 正常路径可能是 `.part0 -> 最终图`
- 也可能直接是最终图

不变的部分是：

- `async-status` 仍然出现在最终图内容请求之后
- 它仍更像后验确认，而不是最早完成信号

---

## 后续追加规则

后续每次分析都按以下结构继续追加：

```text
## 样本 N
### 基本信息
### 样本背景
### 关键时间线
### 样本结论
### 暂定工作假设
### 未覆盖场景
```

不要回写旧样本结论，只在新样本里补充、修正或推翻之前的暂定结论。
