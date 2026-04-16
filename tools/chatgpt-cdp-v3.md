# ChatGPT CDP Observer V3

## 目标

V3 解决的是“只连一次 Chrome/CDP，然后把图片完成信号分发给短生命周期客户端”的问题。

和 V2 相比，V3 有两个核心变化：

- `observer server` 常驻，只在启动时 attach 一次 ChatGPT 页面
- `client` 不再直接连接 Chrome，只向本地 observer 注册 job、等待、下载、落盘、ack

这样做的目的：

- 避免每次抓图都重复建立 CDP 外连
- 把 Chrome 监听和业务落盘解耦
- 让后续批处理直接复用同一个 observer

## 版本复盘

### V1 的价值

V1 指的是 `chatgpt_chrome_bridge.py` 和基于扩展轮询的那套桥接方案。

它现在仍然有 3 个价值：

- 证明“复用用户已登录的 Chrome 会话”这条路线是可行的，不需要强行在 Playwright 里重新过登录和真人验证
- 先把最小闭环跑通了：提交 prompt、等待页面结果、下载文件、批处理 manifest 回写
- 积累了页面状态字段和批处理输入结构，很多命令行参数和 manifest 约定后来直接沿用了

但 V1 的上限也很明确：

- 依赖扩展轮询和前端 DOM 观察，完成判定过于靠页面表象
- 业务脚本、扩展状态、下载动作绑得太紧，调试时很难拆开看
- 很难区分“页面正在忙”“前端还没写完结果”“图片其实已经在 network 层完成”这几种状态

### V2 的价值

V2 指的是 `chatgpt_cdp_capture_v2.mjs` 这类直接监听 CDP network 的单次捕获脚本。

它最大的价值不是“直接可商用”，而是把真正关键的信号挖出来了：

- `POST /backend-api/f/conversation`
- `POST /backend-api/conversation/<id>/async-status`
- `GET /backend-api/files/download/<file_id>`
- `GET /backend-api/estuary/content?id=<file_id>`

V2 还验证了两件事：

- 只看 DOM 动画或按钮状态不够，最终还是要落到 network 信号
- `async-status == OK` 也不能单独作为完成条件，后面还要等最终图收敛

但 V2 仍然有明显缺点：

- 它本质上是“单次观察脚本”，不是稳定的服务
- 每次运行都重新 attach CDP，成本高，也容易让测试和业务逻辑混在一起
- 它解决了“怎么看”，没有彻底解决“如何可靠地分发给业务客户端”

### V3 为什么是当前正确形态

V3 把前两版的经验拆成了两层：

- 常驻 `observer` 只负责监听、判定、缓存和分发
- 短生命周期 `client` 只负责发起 job、等待、落盘、ack

这让三个问题第一次被真正分开：

- Chrome/CDP 连接是否稳定
- 某一轮生成的完成信号是否可信
- 业务客户端有没有正确拿到并落盘

## 为什么之前会花很长时间但没有直接找到正确调试路径

核心原因不是“某个实现写错了”，而是前期把不同层级的问题混在了一起。

### 1. 一开始盯的是页面现象，不是状态机

前期更容易注意到的是这些现象：

- 右上角动画
- 页面是否还在转圈
- 是否出现图片
- 是否能点下载

这些现象对人类判断很直观，但对程序来说不稳定。它们是 UI 表层，不是最底层的完成信号。

### 2. 把“真正双候选 UI”和“network 层多图片信号”混成了同一个问题

这是本项目里最容易犯错、而且确实已经发生过的一类误判。

必须明确区分：

- `network 多图片信号`
  指一个 run 内出现多个 `files/download` 或多个 `estuary/content`，包括 `.part0/.part1`
- `单图内双方案排版`
  指提示词要求两个方案，最终模型把两个建筑或两个变体合成在一张图里
- `真正双候选 UI`
  指 ChatGPT 前端真的返回两张候选图，并出现“你更喜欢哪张图片 / 跳过 / 图片 1 更佳 / 图片 2 更佳”这类交互

这三件事不是一回事。

### 3. 早期缺少“只追加日志”的观察阶段

在找到正确路径之前，脚本一边尝试控制页面，一边尝试判定结果，一边尝试下载，这会导致：

- 一旦判定错了，很难知道是页面控制错、网络解析错，还是落盘错
- 每次失败都像是“整个流程失败”，而不是某一个环节失败

后来转向“先做 observer / 先打 jsonl 日志 / 先不抢着控制页面”，路径才明显清晰。

### 4. `async-status` 是关键节点，但不是完整答案

前面花时间的另一个原因，是我们一直在问“哪个单一信号才是完成”。

实际答案不是单个信号，而是一个小状态机：

1. 新的 `f/conversation` 开始一个 run
2. 期间可能出现多个 `file_download` / `estuary_content`
3. 看到 `async-status == OK`
4. 进入 settle / grace 窗口
5. 选择当前 run 内最后一个“非 `.part` 的有效最终图”

也就是说，正确路径不是“找到一个神奇的包”，而是“找到一组有顺序关系的包”。

## 信号语义

这一节是当前文档最需要强调的部分。

### 什么能说明“内部过程复杂”

下面这些信号只能说明这一轮内部过程比较复杂，不能直接等价于“前端进入双候选模式”：

- 同一 run 内出现多个 `file_download`
- 同一 run 内出现多个 `estuary_content`
- 下载文件名里出现 `.part0`、`.part1`

### 什么能说明“有最终可落盘图”

下面这些信号组合起来，才说明这一轮已经收敛到了最终图：

- 当前 run 已经看到 `async-status == OK`
- settle 窗口内出现了非 `.part` 的最终图下载
- observer 把该图标记为 `job_ready`

### 什么才算“真正双候选 UI”

当前系统里，真正双候选应该优先按页面状态判定，而不是按 network 数量判定。

可靠判定口径：

- `candidateChoiceVisible = true`
- `preferenceButtonCount > 0`
- 页面文案出现“你更喜欢哪张图片 / 跳过 / 图片 1 更佳 / 图片 2 更佳”

注意：

- `partial` 文件不等于双候选 UI
- “一张最终图里画了两个建筑”也不等于双候选 UI

## 当前推荐排错路径

如果后面再出现“为什么看起来完成了，但脚本没有正确下载”之类问题，建议严格按下面顺序排：

1. 先看 `/page/status`
   用来判断当前是不是未登录、还在生成、还是进入了候选选择态
2. 再看 `/debug/state`
   用来判断当前 run 有没有拿到 `conversation / async-status / final image`
3. 最后看 `logs/*.jsonl`
   用来还原完整时间线，确认到底是没有 `async-status`，还是没有最终非 `.part` 图片

不建议再回到“先看页面动效是不是还在转”这种调试顺序。

## 文件

- `tools/chatgpt_cdp_observer_v3.mjs`
  常驻 Node observer server
- `tools/chatgpt_cdp_client_v3.py`
  短生命周期 Python client
- `tools/chatgpt_batch_generate_v3.py`
  基于 JSON manifest 的串行批处理脚本
- `tools/chatgpt_batch_jobs_v3.template.json`
  V3 批处理 manifest 模版

## 当前行为

observer 只监听 4 类关键消息：

- `/backend-api/f/conversation`
- `/backend-api/conversation/<id>/async-status`
- `/backend-api/files/download/<file_id>`
- `/backend-api/estuary/content?id=<file_id>`

判定策略：

1. 看到新的 `f/conversation`，定义为一个新 run
2. 把等待队列里的第一个 job 绑定到这个 run
3. 看到 `async-status == OK` 后，不立刻结束，而是进入 settle 窗口
4. 默认延迟 `3000ms`，然后在当前 run 内寻找“非 `.part` 的最终图”
5. 如果 settle 时还没齐，会继续等待后续图片事件，直到 `grace-ms` 超时
6. observer 把最终图写入自己的 cache
7. client 下载 cache 内容，落到目标目录，然后调用 ack

补充说明：

- 一个 run 内可能出现多个 `file_download`，这不必然代表失败
- 一个 run 内也可能先出现若干 `.part` 图片，最后才出现单张正式结果
- 当前完成判定依赖的是“`async-status OK` 之后是否收敛到非 `.part` 最终图”，不是“下载次数是否等于 1”

observer 现在还额外提供两类动作：

- 读取当前 ChatGPT 页面状态
- 直接向当前页面提交 prompt

## 默认路径

observer 默认状态目录：

```text
<当前工作目录>/output/chatgpt-capture-v3-server
```

其中：

- 日志目录：`logs/`
- 缓存目录：`cache/`

client 默认输出目录：

```text
<当前工作目录>/output/chatgpt-capture-v3-client
```

## 启动 observer

```bash
node /Users/mawei/MyWork/SlgGame/tools/chatgpt_cdp_observer_v3.mjs
```

常用参数：

```bash
node /Users/mawei/MyWork/SlgGame/tools/chatgpt_cdp_observer_v3.mjs \
  --host 127.0.0.1 \
  --port 8776 \
  --settle-ms 3000 \
  --grace-ms 30000
```

## 启动 client

```bash
python3 /Users/mawei/MyWork/SlgGame/tools/chatgpt_cdp_client_v3.py capture \
  --server-base http://127.0.0.1:8776 \
  --output-dir /Users/mawei/MyWork/SlgGame/output/v3-test \
  --basename test-image
```

它的流程是：

1. 注册一个 job
2. 等待 observer 把这个 job 绑定到“下一次 run”
3. 等待结果 ready
4. 从 observer 下载图片字节
5. 写入目标目录
6. 调用 ack
7. client 退出

如果要让 client 同时负责“发 prompt + 收图”，可以直接用：

```bash
python3 /Users/mawei/MyWork/SlgGame/tools/chatgpt_cdp_client_v3.py generate \
  --server-base http://127.0.0.1:8776 \
  --prompt "画一个红色风筝" \
  --output-dir /Users/mawei/MyWork/SlgGame/output/v3-test \
  --basename red-kite
```

## 状态查询

observer 健康检查：

```bash
curl http://127.0.0.1:8776/healthz
```

observer 调试状态：

```bash
curl http://127.0.0.1:8776/debug/state
```

Python 方式：

```bash
python3 /Users/mawei/MyWork/SlgGame/tools/chatgpt_cdp_client_v3.py status
```

## HTTP 接口

### `POST /jobs/register`

请求体：

```json
{
  "label": "main-hall"
}
```

返回：

```json
{
  "ok": true,
  "job": {
    "id": "job-...",
    "state": "waiting",
    "version": 1
  }
}
```

### `GET /jobs/<job_id>/wait?since_version=1&timeout_ms=30000`

长轮询等待 job 状态变化。

返回：

- `timedOut: false` 表示状态有变化
- `timedOut: true` 表示这次长轮询只是超时，没有变化

### `GET /jobs/<job_id>/result`

当 job 进入 `ready` 后，返回图片二进制内容。

响应头里会带：

- `X-Observer-Job-Id`
- `X-Observer-File-Name`
- `X-Observer-Mime-Type`

### `POST /jobs/<job_id>/ack`

表示 client 已经成功落盘，observer 可以清理 cache。

### `POST /jobs/<job_id>/cancel`

表示当前 job 不再继续等待消费。

典型场景：

- job 已注册，但 prompt 提交失败
- 批处理脚本决定中止当前 item

### `GET /page/status`

返回 observer 当前附着页面的状态，包含：

- `composerFound`
- `loginRequired`
- `busyGenerating`
- `candidateChoiceVisible`
- `preferenceButtonCount`
- `progressText`

字段解释：

- `busyGenerating`
  页面当前是否仍处于生成忙碌态
- `candidateChoiceVisible`
  是否出现“更喜欢哪张图片/跳过/候选偏好按钮”这类 UI 选择态
- `preferenceButtonCount`
  候选偏好按钮数量。这个值大于 0 时，基本可以认为页面进入了真正双候选交互
- `progressText`
  前端当前展示的生图进度文案，例如“正在创建图片”“最后微调一下”

### `POST /actions/send-prompt`

请求体：

```json
{
  "prompt": "画一个红色风筝"
}
```

observer 会在当前附着的 ChatGPT 页面中：

1. 尝试清掉遗留候选图选择态
2. 检查页面未登录/仍在生成时直接报错
3. 把 prompt 写入 composer
4. 触发发送

## 当前约束

- 设计目标是串行生图；虽然接口支持多 job 排队，但建议先按单页串行使用
- job 默认绑定“下一次 run”，不是当前已经开始的 run
- observer 只认当前匹配 `chatgpt.com/c/` 的那张页面
- prompt 提交仍建议走 observer 提供的 `send-prompt`，不要在外部另起新的 CDP 连接

## 当前已知边界

- V3 已经能稳定处理“同一 run 内出现多个图片下载事件”这种情况
- V3 还没有在自动批处理中稳定、大量地复现“真正双候选 UI”并自动选择偏好
- 因此，后续测试时应把“network 多下载”和“前端双候选选择模式”分开统计

## 批处理

V3 批处理入口：

```bash
python3 /Users/mawei/MyWork/SlgGame/tools/chatgpt_batch_generate_v3.py \
  /Users/mawei/MyWork/SlgGame/tools/chatgpt_batch_jobs_v3.template.json
```

### manifest 结构

根结构：

```json
{
  "version": 3,
  "defaults": {},
  "items": []
}
```

`defaults` 常用字段：

- `server_base`
- `output_dir`
- `size`
- `format`
- `wait_timeout_seconds`
- `wait_poll_timeout_ms`
- `idle_timeout_seconds`
- `idle_poll_interval_seconds`
- `continue_on_error`
- `skip_done`

`items[]` 常用字段：

- `id`
- `name`
- `basename`
- `prompt`
- `status`
- `output_path`
- `output_paths`
- `error_message`
- `updated_at`

### 批处理执行顺序

每个 item 会串行执行：

1. 等待页面 idle
2. 向 observer 注册 job
3. 通过 observer 提交 prompt
4. 等待 observer 根据 network 信号判定完成
5. 下载结果、落盘、ack
6. 回写 manifest 的 `status / output_path / output_paths / last_job_id`

### 批处理中断语义

- 默认 `continue_on_error = false`
- 任一 item 出错时，manifest 会先回写为 `error`，然后脚本退出
- 如果设为 `true`，则继续后续 item
- 如果 `skip_done = true`，已经完成且 `output_path` 仍存在的 item 会被跳过

## 推荐测试顺序

1. 启动 observer
2. 用 `status` 或 `/healthz` 确认服务正常
3. 先用单条 `generate` 跑一张图
4. 再用 batch manifest 跑 2 到 3 条串行任务
5. 看 manifest 是否正确回写
6. 再看 observer 的 `/debug/state` 和日志是否收敛
