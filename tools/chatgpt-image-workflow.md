# ChatGPT Chrome 生图工作流

## 目标

这套工作流的目标不是“绕过 ChatGPT 登录”，而是：

- 复用你已经在正常 Chrome 中登录好的 `chatgpt.com` 会话
- 通过本地 Python 桥接脚本向 Chrome 扩展下发命令
- 由扩展在真实 ChatGPT 页面里发起生图
- 等待页面出现最终图片
- 打开图片详情弹层并点击 `Save`
- 等待 Chrome 的真实下载完成
- 把下载好的最终文件复制到你指定的输出目录

它解决的是“如何稳定复用真人已登录 Chrome，会控页面出图并自动收图”的问题。

## 适用范围

- 适合已经能在正常 Chrome 中使用 ChatGPT 生图的场景
- 适合单张或串行批量出图
- 适合需要把图片自动落到固定目录、供游戏资产流程继续消费的场景
- 适合在浏览器里继续看别的网页时，让一个专门绑定的 ChatGPT 标签页在后台跑生图
- 不负责切换 ChatGPT 模型
- 不负责自动通过真人验证
- 不保证模型一定只返回 1 张候选图
- 不保证模型一定产出透明背景或完全符合提示词审美

最后两点属于模型能力和产品策略，不属于桥接流程本身。

## 目录结构

当前工作流相关文件如下：

- `tools/chatgpt_chrome_bridge.py`
  本地桥服务与命令行入口
- `tools/chatgpt_batch_generate.py`
  基于 JSON 清单的串行批量生图工具
- `tools/chatgpt_batch_jobs.template.json`
  JSON 批处理模版
- `tools/chatgpt-chrome-bridge/manifest.json`
  Chrome MV3 扩展声明
- `tools/chatgpt-chrome-bridge/content.js`
  页面侧自动化逻辑
- `tools/chatgpt-chrome-bridge/background.js`
  扩展后台逻辑，负责下载捕获等能力
- `tools/chatgpt-chrome-bridge/popup.html`
  扩展弹窗 UI
- `tools/chatgpt-chrome-bridge/popup.js`
  扩展弹窗交互逻辑

## 架构概览

### 1. Python 桥服务

`chatgpt_chrome_bridge.py` 提供三个命令：

- `serve`
  启动本地 HTTP 服务和文件队列
- `status`
  请求扩展返回当前 ChatGPT 页面状态
- `generate`
  请求扩展执行一次真实生图并把结果复制到本地输出路径

桥服务默认监听：

```text
http://127.0.0.1:8765
```

默认队列目录：

```text
~/.codex/chatgpt-chrome-bridge
```

默认追加日志文件：

```text
~/.codex/chatgpt-chrome-bridge/logs/bridge-events.ndjson
```

### 2. Chrome 扩展

扩展只注入到：

```text
https://chatgpt.com/*
```

扩展的职责：

- 只让“被绑定的那个 ChatGPT 工作页”轮询本地桥服务，领取待执行命令
- 判断当前页面是否已登录、输入框是否可用、是否仍在生成中
- 把提示词写入真实 ChatGPT 输入框并提交
- 先锁定本次新提交对应的 user turn
- 再只在这条 user turn 后面的 response sections 中等待图片卡、占位按钮或真实图片出现
- 识别这一次生成对应的图片卡/占位按钮/下载动作，而不是全页最后一张图
- 打开图片详情弹层
- 点击 `Save`
- 通过 downloads API 等待真实下载完成
- 回传最终下载文件路径

### 2.5. 工作页绑定

当前版本的关键改动是：

- 只有扩展 popup 里显式绑定的那个 `chatgpt.com` 标签页会启动轮询和执行生图
- 其他网页标签页不会被触碰
- 其他 `chatgpt.com` 标签页如果存在，也不会参与抢队列
- 因此你可以继续在 Chrome 中浏览别的网站，不需要让自动化不断切页

建议做法：

1. 保留一个专门用于自动化的 `chatgpt.com` 标签页
2. 在这个标签页中手动切好模型
3. 打开扩展 popup，点击 `绑定工作页`
4. 确认 popup 中出现工作页编号和标题
5. 后续所有 `status` / `generate` / 批量任务都只会打到这个工作页

### 3. ChatGPT 页面

真实生图发生在你当前打开的 `chatgpt.com` 页面里，因此：

- 实际使用哪个模型，由当前页面决定
- 是否有真人验证，由当前页面决定
- 是否出现 2 张候选图，也由当前产品行为决定

## 运行日志与关键信号

当前版本会把以下内容统一追加到一个 NDJSON 日志中：

- 命令入队
- 工作页领取命令
- 页面阶段进度
- 关键网络事件
- 最终结果或错误

日志文件：

```text
~/.codex/chatgpt-chrome-bridge/logs/bridge-events.ndjson
```

设计原则：

- 只追加，不修改历史行
- 一行一个 JSON，方便后续筛选和脚本分析
- 只在实际执行命令的窗口期记录关键网络事件
- 会过滤扩展自身请求本地桥服务、以及桥接自己回抓图片 payload 时造成的噪声

当前重点观测的网络类别：

- `conversation`
  对应 `/backend-api/f/conversation`
- `conversation_prepare`
  对应 `/backend-api/f/conversation/prepare`
- `async_status`
  对应 `/backend-api/conversation/<id>/async-status`
- `file_download`
  对应 `/backend-api/files/download/<file_id>`
- `estuary_content`
  对应 `/backend-api/estuary/content?id=<file_id>`

日志里常见的 `kind`：

- `command_enqueued`
- `command_claimed`
- `progress`
- `network`
- `result`

其中：

- `progress` 代表页面自动化阶段
- `network` 代表关键网络信号
- `result` 代表这次命令的最终成功或失败

快速查看最近日志：

```bash
tail -n 80 ~/.codex/chatgpt-chrome-bridge/logs/bridge-events.ndjson
```

## 当前已经验证通过的完整链路

当前版本已经验证过下面这条链路是通的：

1. Python 发起 `generate`
2. 扩展在真实 ChatGPT 页面输入提示词
3. 页面开始生成
4. 扩展等待最终图片稳定出现
5. 扩展打开最新图片详情弹层
6. 扩展点击 `Save`
7. 后台脚本等待 Chrome 真实下载完成
8. Python 把下载结果复制到 `--output` 或 `--output-dir`
9. 扩展自动关闭图片详情弹层
10. 页面回到主对话生图界面，便于下一张继续生成

第 9、10 步是当前版本修过的重要稳定性补丁。

## 一次性准备

### 1. 启动本地桥服务

```bash
python3 /Users/mawei/MyWork/SlgGame/tools/chatgpt_chrome_bridge.py serve
```

如果你的项目路径不同，把脚本路径替换成自己的绝对路径即可。

### 2. 在 Chrome 中加载未打包扩展

打开：

```text
chrome://extensions
```

操作：

1. 打开“开发者模式”
2. 点击“加载已解压的扩展程序”
3. 选择目录：

```text
/Users/mawei/MyWork/SlgGame/tools/chatgpt-chrome-bridge
```

### 3. 确认扩展 popup 配置

在扩展弹窗中确认 `Server Base` 为：

```text
http://127.0.0.1:8765
```

然后：

1. 打开你准备用作自动化生图的 `chatgpt.com` 页面
2. 在扩展 popup 中点一次 `绑定工作页`
3. 确认 popup 显示了对应的工作页信息
4. 需要时再点一次 popup 中的 `页面状态`

### 4. 使用正常 Chrome 登录 ChatGPT

要求：

- 必须是在你平时正常使用的 Chrome 中完成登录
- 必须能手动在这个页面里正常生图
- 最好保留一个专门用于自动化的 `chatgpt.com` 标签页
- 最好在这个标签页中手动切好要用的模型
- 绑定完成后，后续不要把这个标签页手动切到别的站点

## 日常使用

### 1. 先确认页面状态

```bash
python3 /Users/mawei/MyWork/SlgGame/tools/chatgpt_chrome_bridge.py status
```

注意：

- 这个命令会发到“当前绑定的工作页”
- 如果你还没有在 popup 里绑定工作页，它会一直等到超时

正常时通常会看到这些关键信号：

- `composerFound: true`
- `loginRequired: false`
- `busyGenerating: false`

如果不是这个状态，不要直接批量发图。

### 2. 发起单次生图

输出到固定文件：

```bash
python3 /Users/mawei/MyWork/SlgGame/tools/chatgpt_chrome_bridge.py generate \
  --prompt "你的提示词" \
  --output /absolute/path/to/result.png \
  --post-completion-settle-ms 30000
```

输出到目录并指定基名：

```bash
python3 /Users/mawei/MyWork/SlgGame/tools/chatgpt_chrome_bridge.py generate \
  --prompt "你的提示词" \
  --output-dir /absolute/path/to/output-dir \
  --basename asset-name \
  --post-completion-settle-ms 30000
```

### 3. 推荐参数

一般推荐：

- `--wait-timeout 1200`
  给整次生成足够时间
- `--generation-timeout-ms 600000`
  给页面生成足够时间
- `--post-completion-settle-ms 30000`
  给页面在“看起来完成后”再多留一点稳定时间

尤其在 ChatGPT 会继续做“最后微调一下”或者慢速收尾时，`post-completion-settle-ms` 很有用。

## 批量生成建议

当前最稳的方式是串行批量，而不是并行批量。

建议：

- 一次只让一个 `generate` 命令在跑
- 等当前命令完全结束，再发下一张
- 保持同一个 `chatgpt.com` 标签页
- 中间不要手动切模型、切对话或打开其他图片弹层

推荐批量策略：

1. 先手动 `status`
2. 串行执行多次 `generate`
3. 每次都指定 `--basename`
4. 生成后再统一做筛选与缩放

## JSON 批量工具

### 文件

- 工具脚本：

```text
/Users/mawei/MyWork/SlgGame/tools/chatgpt_batch_generate.py
```

- 模版文件：

```text
/Users/mawei/MyWork/SlgGame/tools/chatgpt_batch_jobs.template.json
```

### 用法

```bash
python3 /Users/mawei/MyWork/SlgGame/tools/chatgpt_batch_generate.py \
  /Users/mawei/MyWork/SlgGame/tools/chatgpt_batch_jobs.template.json
```

### 行为

- 串行读取 `items`
- 每完成一个资源，就立刻把 `status`、`output_path`、`output_paths`、`error_message` 回写到原 JSON
- 如果中途停掉，下次再跑时会默认跳过已经 `done` 且产物仍存在的项
- 输出尺寸由 `size` 控制，例如 `512x512`
- 默认会统一转成 `png`

### JSON 约定

顶层结构：

```json
{
  "version": 1,
  "defaults": {},
  "items": []
}
```

推荐字段：

- `defaults.output_dir`
  批量输出目录。相对路径按 manifest 文件所在目录解析。
- `defaults.size`
  统一目标尺寸，例如 `512x512`
- `defaults.format`
  统一输出格式。当前推荐 `png`
- `defaults.wait_timeout`
  单项任务总等待秒数
- `defaults.generation_timeout_ms`
  页面侧等待生成完成的毫秒数
- `defaults.post_completion_settle_ms`
  页面显示“完成”后额外等待的稳定时间
- `defaults.continue_on_error`
  出错后是否继续下一个。当前建议默认 `false`，人工盯流程时更安全。
- `defaults.skip_done`
  再次执行时是否跳过已完成项
- `items[].name`
  资源名称
- `items[].prompt`
  该资源的提示词
- `items[].size`
  该项自定义尺寸，可覆盖默认值
- `items[].basename`
  输出文件基名
- `items[].status`
  运行状态，通常是 `pending` / `running` / `done` / `error`
- `items[].output_path`
  主产物路径
- `items[].output_paths`
  全部产物路径
- `items[].error_message`
  最近一次失败信息

## 输出规则

当前桥接脚本最终拿到的是“真实下载的原图”，然后复制到你的目标位置。

因此：

- `--output /a/b/c.png`
  会尽量保存为该基名，对应扩展名由实际下载文件类型决定
- `--output-dir + --basename`
  会保存为 `basename.ext`
- 如果本次返回多张图，脚本会按 `basename-1.ext`、`basename-2.ext` 命名

当前项目里常见用法是：

- 先保存原始 `1024x1024`
- 再单独复制一份缩放为 `512x512`

批量工具和单次 `generate` 的区别是：

- 单次 `generate` 更接近“拿到真实下载原图”
- 批量工具更接近“直接产出指定尺寸、指定格式的资产文件”

## 开发文档

### 核心状态机

页面侧生成逻辑大致分为几段：

1. `waitForStablePageBaseline`
  等待页面处于稳定、可输入、非生成状态
2. `setComposerValue + submitPrompt`
  写入提示词并提交
3. `waitForGenerationCompletion`
  等待新图出现且页面退出忙碌状态
4. `settleImageCandidates`
  给页面一点额外稳定时间
5. `captureFinalAsset`
  进入最终抓图逻辑
6. `captureDialogSaveDownload`
  优先走详情弹层里的 `Save`
7. `closeAllVisibleDialogs`
  收尾时关闭详情弹层，回到主界面

### 当前优先抓图策略

当前优先策略是：

1. 找到“最新生成图片”的真实可点击容器
2. 点击进入详情弹层
3. 找到 `Save`
4. 调用后台下载捕获
5. 等 Chrome 报告下载完成

只有这条链路失败时，才会退回旧的下载捕获逻辑。

### 最近修过的两个关键稳定性问题

#### 问题 1：下载后残留在图片详情弹层

旧行为：

- 一张图保存完后，页面仍停留在图片详情层
- 下一张图开始生成时，脚本很容易误用旧弹层

当前修复：

- `generate` 开始前先清掉残留弹层
- `generate` 保存成功后再主动关闭所有可见弹层
- 这样下一张图会从主对话界面开始

#### 问题 2：点错图片外层，无法打开最新详情层

旧行为：

- 某些新图片的 DOM 里，`cursor-pointer` 那层只是视觉容器
- 真正有点击处理的是更外层 `role="button"` 节点
- 旧逻辑有时会点到错误层级

当前修复：

- 现在会优先向上查找真正的交互节点：
  - `button`
  - `[role='button']`
  - `a[href]`
- 只有找不到时，才回退到 `.cursor-pointer`

### 修改页面脚本后的正确刷新顺序

只要你改了下面任意文件：

- `content.js`
- `background.js`
- `popup.js`
- `manifest.json`

都建议执行这个顺序：

1. 打开 `chrome://extensions`
2. 对 `SlgGame ChatGPT Bridge` 点“重新加载”
3. 刷新已经打开的 `chatgpt.com` 页面
4. 再执行 `status`
5. `status` 正常后再执行 `generate`

如果你只做了第 2 步，没刷新 `chatgpt.com` 页面，页面里仍可能是旧 content script，常见报错是：

```text
Extension context invalidated.
```

### 推荐的调试手段

最有效的调试方法是三路一起看：

- 命令行输出
  看 `Generation succeeded` 或具体错误
- 扩展 popup 的 `status`
  看页面当前是不是可输入、是否仍在生成
- Chrome DevTools MCP 或手动 DevTools
  看最新图片对应 DOM 到底是什么结构

### 对其他项目复用时需要替换的部分

这套工具本身可以复用，但下面这些路径通常要换成你自己的：

- Python 脚本绝对路径
- Chrome 扩展加载目录
- 输出目录
- 项目内批量命名规则

下面这些一般不需要改：

- `127.0.0.1:8765`
- 扩展权限
- 队列机制

除非你确实需要多实例桥服务，才考虑改端口和 `--queue-root`。

## 已知限制

### 1. 模型选择不由脚本控制

脚本不会帮你切换 ChatGPT 页面上的模型。

实际用的模型，取决于：

- 你当前打开的会话页面
- 你手动切好的模型
- ChatGPT 产品当时的前端行为

### 2. 提示词无法 100% 禁止“双候选图”

即使你在提示词里写：

```text
只生成1张最终成图，不要2张或多张候选图
```

也只能降低概率，不能 100% 强制。

因为“返回 2 张候选图”可能是产品层策略，不完全受提示词约束。

### 3. 透明背景不能保证

即使提示词明确要求透明背景，某些模型仍然可能产出：

- 黑灰渐变背景
- 舞台光背景
- 假透明观感但实际不是透明 PNG

这属于模型出图质量问题，不是桥接流程问题。

### 4. 审美和资产化程度不能由流程保证

流程只负责：

- 发起
- 等待
- 下载
- 落盘

它不能保证模型一定画出你真正想要的资产风格。

## 故障排查

### `status` 卡住或无返回

优先检查：

- `serve` 是否还在运行
- 扩展是否已加载
- 扩展 popup 的 `Server Base` 是否正确
- 当前活动标签页是不是 `chatgpt.com`

### `Extension context invalidated`

基本就是：

- 扩展重新加载了
- 但 `chatgpt.com` 页面没有刷新

处理方式：

1. `chrome://extensions` 重新加载扩展
2. 刷新 `chatgpt.com`
3. 再跑 `status`

### `Could not find a final download action on the latest generated image`

常见原因：

- 页面里残留旧详情弹层
- 新图 DOM 结构变化
- ChatGPT 还没真正结束生成
- 当前页面不是最终图，而是候选图或过渡态

现在版本已经修复了两类常见误判，但如果未来 ChatGPT 前端大改，这个错误仍可能重新出现。

排查方式：

1. 先看页面里是否已经有最新图
2. 手动点最新图，确认能否进入详情层
3. 确认详情层里是否仍有 `Save`
4. 再决定是修点击目标识别，还是修保存按钮识别

### 生成成功但抓到的是上一张图

旧版本出现过，根因是：

- 下载后没有退出旧弹层
- 下一轮又复用了旧详情层

当前版本已修复这个问题；如果再次出现，优先检查是不是页面里还有新的多层 dialog 结构没有被识别到。

### 明明提示词要求单图，最后还是出现多图候选

这是产品行为或模型行为，不是桥接 bug。

处理思路：

- 在提示词里继续强调“只要唯一最终稿”
- 但不要把它当作强约束
- 真正要做到强约束，只能走支持显式 `n=1` 的 API 方案

## 建议的项目内使用规范

如果要在其他项目里长期复用，建议统一以下约定：

- 固定一个专门的 ChatGPT 自动化标签页
- 固定一个专门的输出目录
- 原图与缩放图分目录
- 所有批量任务串行执行
- 修改扩展后必须先 reload，再刷新页面，再 `status`

## 当前项目内已验证的输出示例

当前项目里已验证过的产出目录示例：

```text
/Users/mawei/MyWork/SlgGame/output/chrome-bridge/assets/batch-20260416
/Users/mawei/MyWork/SlgGame/output/chrome-bridge/assets/batch-20260416/final-512
```

这只是当前项目的约定，不是工具硬编码要求。

## 结论

这套工作流当前已经具备可持续使用的工程稳定性：

- 能复用真人已登录 Chrome
- 能真实控制 ChatGPT 页面发图
- 能判断生成完成
- 能下载最终图片
- 能落到指定目录
- 能在一张结束后自动回到下一张可继续生成的主界面

如果后续要继续做批量工作流、任务模板或插件化封装，建议直接以这份文档作为当前基线。
