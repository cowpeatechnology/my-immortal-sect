# ChatGPT Chrome Bridge

这是 ChatGPT Chrome 生图工作流里的浏览器扩展部分。

权威文档在这里：

```text
/Users/mawei/MyWork/SlgGame/tools/chatgpt-image-workflow.md
```

## 这个目录里有什么

- `manifest.json`
  MV3 扩展声明
- `content.js`
  ChatGPT 页面自动化逻辑
- `background.js`
  下载捕获与工作页绑定逻辑
- `popup.html`
  扩展弹窗
- `popup.js`
  扩展弹窗交互

## 运行日志

桥服务现在会把命令、阶段进度、关键网络事件和最终结果，统一追加到一个 NDJSON 文件：

```text
~/.codex/chatgpt-chrome-bridge/logs/bridge-events.ndjson
```

日志特点：

- 只追加，不回写旧记录
- 每行一个 JSON，适合后续脚本分析
- 会过滤桥接自身发往 `127.0.0.1` 的请求噪声
- 重点记录 ChatGPT 生图相关的 `conversation`、`async-status`、`files/download`、`estuary/content` 等信号

快速查看最近日志：

```bash
tail -n 50 ~/.codex/chatgpt-chrome-bridge/logs/bridge-events.ndjson
```

## 最小启动步骤

1. 启动桥服务

```bash
python3 /Users/mawei/MyWork/SlgGame/tools/chatgpt_chrome_bridge.py serve
```

2. 在 Chrome 扩展页加载这个目录

```text
chrome://extensions
```

3. 在正常 Chrome 中打开并登录 `https://chatgpt.com/`

4. 扩展 popup 中确认：

```text
http://127.0.0.1:8765
```

5. 打开准备用作自动化的 `chatgpt.com` 页面，在 popup 中点击 `绑定工作页`

6. 手动执行一次：

```bash
python3 /Users/mawei/MyWork/SlgGame/tools/chatgpt_chrome_bridge.py status
```

## 工作页原则

- 只有 popup 里绑定的那个 `chatgpt.com` 页面会执行队列命令
- 其他网站标签页不会被自动切换或操作
- 如果你关闭或跳转走这个工作页，需要重新绑定
- 当前版本会把每次生成锁定到“刚提交的那条 user turn”后面的回复区域里处理，尽量避免误抓旧图或误点历史消息

## 批处理入口

JSON 批处理工具：

```text
/Users/mawei/MyWork/SlgGame/tools/chatgpt_batch_generate.py
```

模版文件：

```text
/Users/mawei/MyWork/SlgGame/tools/chatgpt_batch_jobs.template.json
```

## 修改扩展后的注意事项

只要你改了这个目录里的任何脚本文件，都建议按这个顺序操作：

1. `chrome://extensions` 中点“重新加载”
2. 手动刷新已经打开的 `chatgpt.com` 标签页
3. 再运行 `status`
4. `status` 正常后再运行 `generate`

否则很容易遇到：

```text
Extension context invalidated.
```

## 说明

这份 README 只保留入口信息，详细的开发说明、使用方式、批量建议、已知限制和排障说明，请统一看：

```text
/Users/mawei/MyWork/SlgGame/tools/chatgpt-image-workflow.md
```
