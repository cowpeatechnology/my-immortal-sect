# F-003-knowledge-harvest-pipeline

**状态**: active  
**Area**: tools / process  
**Owner**: `supervisor`  
**Reviewer**: `engineer`  
**最后更新**: 2026-04-19

## Context

项目接下来会持续收集与真实开发相关的外部知识。

如果直接在 `supervisor` 协调上下文里做搜索、阅读、摘要和入库，很容易出现两个问题：

- 上下文快速膨胀，压缩后难以恢复状态
- 长跑采集过程中，`worker` 是否卡死不容易判断

因此需要一个“最小上下文增量”的知识采集管线，把 `supervisor` 的职责收敛为：

- 生成任务清单
- 分发 shard
- 监控 heartbeat
- 抽查结果质量

## Goal

建立一个可恢复的知识采集第一阶段，支持：

- 以关键词为单位构建 search queue
- 为后续并行 worker 预留 candidate / ingest / heartbeat / claim 文件路径
- 让 `supervisor` 只依赖文件和计数而不是长文本

本文里的 `worker` 指知识采集执行单元或常驻进程，不是项目的 durable role roster。项目级长期协作仍以 `supervisor / engineer / art_asset_producer` 三角色可见线程为准。

## Non-Goals

- 当前轮不实现完整大规模抓取调度器
- 当前轮不实现数据库或 Web UI
- 当前轮不跑 200-300 篇/关键词的全量采集

## Authority Docs

- `AGENTS.md`
- `docs/process/engineering-standards.md`
- `tools/AGENTS.md`
- `docs/process/knowledge-harvest-workflow.md`

## Acceptance Criteria

- [x] 已定义 `supervisor` / `worker` 的最小上下文采集协议
- [x] 已定义 seed queue / claim / heartbeat / manifest 的分层
- [x] 已明确运行态数据独立写入 `runtime/knowledge-harvest/`
- [x] 已提供第一阶段 queue 生成脚本
- [x] 已提供 query worker 脚本，用于写入 candidate manifest + claim + heartbeat
- [x] 已提供 ingest worker 脚本，用于把 candidate URL 写入 Obsidian，并维护 ingest claim + heartbeat + manifest
- [x] 已提供 heartbeat 状态检查脚本
- [x] 已提供 shard planner，用于生成最小上下文的 worker work orders
- [x] 已提供正式 execution plan 生成脚本，用于维护主 queue、历史补采 queue 和运行阶段
- [x] 已提供 ingest shard planner，用于为 Obsidian 入库阶段生成最小上下文 work orders
- [x] 已生成一份初始 seed queue 文件
- [x] 已完成 1 到 2 个关键词的小样本 query 验证
- [x] 已完成 1 个真实 Obsidian ingest smoke
- [x] 已完成 1 个 wiki-first Obsidian source ingest，并落下 `raw + wiki + index + log`
- [x] 已完成第一轮小规模多 worker query fan-out 验证
- [x] 已完成主 run 的第一轮真实 query fan-out
- [ ] 已完成并行 worker 的第一轮真实 ingest fan-out
- [x] 已完成第一轮 Google 浏览器驱动搜索 + 常驻 Wiki 入库长跑验证
- [x] 已完成“三常驻进程”浏览器采集包装：`supervisor + browser search worker + wiki ingest worker`
- [x] 已验证浏览器搜索 worker 在专用 Chrome 下只管理两个受管 Tab：`search` 与 `preview`
- [x] 已完成一轮三常驻进程 supervisor smoke，并确认 `candidate -> ingest -> summary` 全链路闭环

## Current Decision Summary

- 第一阶段只生成查询清单，不抓正文
- query 阶段已有独立 worker 入口，职责仅限“发现 URL -> 写 candidate manifest”
- ingest 阶段已有独立 worker 入口，职责仅限“读取 candidate manifest -> OAuth 整理 -> 写入 Obsidian + ingest manifest”
- 对于需要长期沉淀的高价值来源，优先使用 wiki-first 入口，而不是把 URL 只写入扁平 Inbox
- `worker` 默认只回传路径、计数和错误签名
- liveness 以 heartbeat 文件为准，不以聊天回复为准
- candidate 与 ingest 采用 `JSONL`，seed queue 与 heartbeat 采用 `JSON`
- 脚本源代码放在 `tools/`，运行态队列和结果放在 `runtime/knowledge-harvest/`
- 小样本 smoke test 已验证控制面闭环：`claim -> heartbeat -> candidate manifest -> status checker`
- 真实 ingest smoke 已验证第二阶段闭环：`candidate manifest -> ingest worker -> Obsidian note + ingest manifest + ingest heartbeat`
- 主 run 已生成 `query-shards.json`，`supervisor` 后续可以直接按 shard 调度，而不是手工重组 seed ids
- query worker 当前默认采用开放搜索 queue：不预设来源白名单、不预设 `site:` 定向域名，并同时生成 `zh/en` 查询
- query worker 只保留 `include_domains` / `exclude_domains` / `skip_index_pages` 作为 rerun 旋钮，不再把 provider 暴露到控制面
- 正式运行以 `execution-plan.json` 为控制面入口，`seed-queue-relaxed.json` 为低产主题的历史补采入口
- 扩展主题已经并入主 queue，但执行顺序仍保持“先已总结关键词、后广义游戏主题”
- 当前已验证：`forum.cocos.org` 这类技术社区源适合持续 ingest，但它们不再作为 queue 默认白名单，而是 rerun/ingest 时的显式过滤选项
- 当前已验证：`design` 主题宽抓取虽然能出量，但大量 URL 会落到 `zhihu/百度`，后续 ingest 容易被 `403` 拦截
- 当前已验证：环境可访问 `google.com`，但 Google 搜索结果页对纯 HTTP 抓取返回强 JS 页面，因此后续如需稳定接入 Google，应走专用 Chrome 复用的浏览器执行链路
- 当前已验证：可以用“一个浏览器搜索 worker + 一个 Wiki 入库 worker”的双常驻模式，把 `supervisor` 职责收敛为 heartbeat / manifest / 计数检查
- 当前已验证：浏览器搜索侧需要支持查询重写，否则 `actor` 这类词会被 Google 拉向 `acting` 噪声页
- 当前已验证：浏览器搜索侧需要批量候选落盘入口，避免标题/摘要在 shell quoting 上卡死；`append-batch` 已作为正式控制面入口补齐
- 当前已验证：浏览器驱动运行不应默认依赖“每次新会话重新授权”的链路；后续浏览器任务应优先复用专用 Chrome、固定调试端口与现有登录态/现有 Tab
- 当前已验证：专用 Chrome 模式已能被新的独立进程重复附着，并已真实覆盖 Google 搜索、ChatGPT Web 生图和本地网页调试三类需求
- 当前已验证：正式包装应采用三个常驻进程，主进程只做拉起、心跳监控、异常重启和收尾；搜索与入库保持职责隔离
- 当前已验证：搜索 worker 自身只允许接管两个受管 Tab，一个搜索 Tab、一个预览 Tab；其他业务 Tab 只能复用，不能被搜索流程扩张占用
- 当前已验证：Wiki 入库必须处理“不同 URL 但相同标题”的文件名冲突，否则批量采集时会出现稳定的拒写错误
- 当前已确认：浏览器采集默认应硬跳过纯视频/直播站，以及文件型文档链接/文档内容类型，避免无效预览和无意义入库
- 当前仍未解决的重点不是“worker 能否工作”，而是“开放搜索之后，如何把低质量候选在 rerun / ingest 阶段有纪律地收窄”

## Conflict And Impact

- 冲突对象：把搜索结果正文直接带回 `supervisor` 协调上下文
  - 冲突原因：会导致协调上下文不可控增长
  - 当前裁决：正文只允许落盘，不允许进入 `supervisor` 自然语言往返
  - 后续动作：后续 worker work order 必须明确“不得粘贴正文”

- 冲突对象：在第一轮就实现复杂调度器
  - 冲突原因：会把当前重点从“协议可恢复”偏到“系统过度工程”
  - 当前裁决：当前只做 queue + heartbeat + status checker
  - 后续动作：等 seed queue 和小样本验证稳定后再决定是否补调度器

## Implementation Status

### Done

- [x] 已创建知识采集流程文档
- [x] 已创建 seed queue 生成脚本
- [x] 已创建 query worker 脚本
- [x] 已创建 ingest worker 脚本
- [x] 已创建 heartbeat 状态检查脚本
- [x] 已创建 shard planner 脚本
- [x] 已创建正式 execution plan 生成脚本
- [x] 已创建 ingest shard planner 脚本
- [x] 已生成 `runtime/knowledge-harvest/<run_id>/seed-queue.json`
- [x] 已生成 `runtime/knowledge-harvest/<run_id>/execution-plan.json`
- [x] 已生成 `runtime/knowledge-harvest/<run_id>/seed-queue-relaxed.json`
- [x] 已对 `cocos-creator-tilemap` 与 `rimworld-success-analysis` 做过真实联网 smoke test
- [x] 已对 `cocos-creator-tilemap` 做过真实 Obsidian ingest smoke
- [x] 已创建 `tools/llm_wiki_maintainer.py`，支持 `bootstrap` 与 `ingest-url`
- [x] 已在 `/Users/mawei/MyWork/我的知识库/Projects/我的宗门 Wiki/` 下创建项目 wiki 骨架
- [x] 已将 Karpathy 的 `LLM Wiki` 文章真实写入 `raw/sources/` 与 `wiki/sources/`，并同步更新 `wiki/index.md` 与 `wiki/log.md`
- [x] 已对技术 shard 与设计 shard 各做过一次真实多 worker fan-out 验证
- [x] 已对 main run 的 3 个 shard 做过一次真实最小上下文 fan-out 验证
- [x] 已创建 `tools/google_browser_queue.py`，用于 Google 浏览器驱动运行的 query claim / candidate / heartbeat 控制面
- [x] 已创建 `tools/wiki_ingest_queue_worker.py`，用于持续消费浏览器候选队列并写入项目 Wiki
- [x] 已创建 `runtime/knowledge-harvest/2026-04-17-google-browser-wiki/browser-run.json`，并从主 seed queue 初始化出 `183` 条浏览器搜索查询
- [x] 已以 `search-agent-01 + ingest-agent-01` 双常驻 worker 真实跑通：Google 浏览器搜索 -> candidate queue -> Wiki ingest
- [x] 已补 `append-batch` 批量候选写入入口，避免浏览器 worker 在长参数命令上失速
- [x] 已新增 `docs/process/dedicated-browser-workflow.md`，并把“专用 Chrome + 固定调试端口 + 会话复用”设为后续浏览器任务的默认模式
- [x] 已真实验证新进程复用专用 Chrome：ChatGPT 图片请求成功出图，本地 `localhost:7456` 页面成功接收输入事件并完成截图
- [x] 已新增 `tools/browser_cdp.py`，提供专用 Chrome 复用所需的最小 CDP 连接与 target/session 操作
- [x] 已新增 `tools/browser_google_search_worker.py`，以专用 Chrome 的两个受管 Tab 执行 `claim -> 搜索 -> 预览 -> 候选落盘`
- [x] 已新增 `tools/browser_harvest_supervisor.py`，把现有 CLI 包装为 `supervisor + search worker + ingest worker` 三常驻进程
- [x] 已新增 `tools/browser_content_filters.py`，统一纯视频站与文件型文档过滤规则，并接到搜索/入库双层链路
- [x] 已修复 `tools/llm_wiki_maintainer.py` 的同标题来源命名冲突，撞名时追加稳定 URL 哈希后缀
- [x] 已完成 `2026-04-17-browser-supervisor-smoke` 与 `2026-04-17-browser-supervisor-smoke-2` 两轮真实 smoke；第二轮结果为 `candidate=6 / ingest_saved=6 / ingest_error=0`

### In Progress

- [ ] 收敛第一批关键词集合
- [ ] 收敛基础参考主题与近两年资讯主题的分流策略
- [ ] 扩大 browser-driven Google 运行的覆盖面，并抽查不同主题下的候选质量

### Not Started

- [ ] 多 worker 并行 ingest 阶段
- [ ] 结果质量抽检阶段

### Deferred

- [ ] 数据库化
- [ ] Web 控制台
- [ ] 自动重试调度器

## Loop History

| Loop | Date | Stage | Summary | Output | Decision |
|---|---|---|---|---|---|
| L-001 | 2026-04-17 | Plan | 定义最小上下文知识采集协议和文件分层 | `docs/process/knowledge-harvest-workflow.md` | continue |
| L-002 | 2026-04-17 | Bootstrap | 落第一阶段 queue 生成器与 heartbeat 状态检查器 | `tools/build_research_seed_queue.py` / `tools/research_worker_status.py` | continue |
| L-003 | 2026-04-17 | Seed | 生成第一版 `sect-sim-cocos` seed queue | `runtime/knowledge-harvest/<run_id>/seed-queue.json` | continue |
| L-004 | 2026-04-17 | Query Worker | 落 query-only worker，支持真实搜索并把候选 URL 追加到 candidate manifest，同时维护 claim 与 heartbeat | `tools/query_research_candidates.py` | continue |
| L-005 | 2026-04-17 | Live Smoke | 对 `cocos-creator-tilemap` 与 `rimworld-success-analysis` 做联网 smoke test，验证 claim / heartbeat / candidate manifest / status checker 闭环 | `runtime/knowledge-harvest/2026-04-17-sect-sim-cocos-smoke2/` | continue |
| L-006 | 2026-04-17 | Query Tuning | 为分析型 topic 增加 topic 级自定义查询模板，并把结果回写主 queue | `tools/build_research_seed_queue.py` / `runtime/knowledge-harvest/2026-04-17-sect-sim-cocos/seed-queue.json` | continue |
| L-007 | 2026-04-17 | Worker Fan-Out | 用两个最小上下文 worker 分别验证技术 shard 与设计 shard 的 query fan-out，确认 `supervisor` 可只依赖 summary / heartbeat / manifest | `runtime/knowledge-harvest/2026-04-17-sect-sim-cocos-fanout1/` / `runtime/knowledge-harvest/2026-04-17-sect-sim-cocos-fanout2/` | continue |
| L-008 | 2026-04-17 | Shard Planning | 生成主 run 的 `query-shards.json`，把后续 fan-out 所需的最小 work order 固化为文件 | `tools/plan_research_shards.py` / `runtime/knowledge-harvest/2026-04-17-sect-sim-cocos/query-shards.json` | continue |
| L-009 | 2026-04-17 | Ingest Worker | 增加 ingest worker、ingest heartbeat/claim 输出，并对 `docs.cocos.com` 页面完成一次真实 Obsidian 入库 smoke | `tools/ingest_research_candidates.py` / `runtime/knowledge-harvest/2026-04-17-sect-sim-cocos/control/ingest/` | continue |
| L-010 | 2026-04-17 | Main Run Fan-Out | 对主 run 的 `client-map-02`、`client-platform-03`、`design-07` 做并行最小上下文 query，验证 `supervisor` 侧 summary / heartbeat / candidate manifest 的主线可用性 | `runtime/knowledge-harvest/2026-04-17-sect-sim-cocos/control/summaries/2026-04-17-sect-sim-cocos-*.json` | continue |
| L-011 | 2026-04-17 | Formal Execution Plan | 把正式运行阶段固化为 `execution-plan.json`，补齐历史补采 queue、扩展主题和 ingest shard plan，并设置 `2026-04-17T14:00:00+08:00` 前不得关闭计划 | `tools/prepare_harvest_execution.py` / `tools/plan_ingest_shards.py` / `runtime/knowledge-harvest/2026-04-17-sect-sim-cocos/execution-plan.json` | continue |
| L-012 | 2026-04-17 | Source Quality Split | 对 `wechat / design / business / 2D` 扩展主题做来源质量修正，确认“技术源可持续 ingest、抽象主题需要宽抓取后筛选”的分流策略 | `tools/build_research_seed_queue.py` / `runtime/knowledge-harvest/2026-04-17-sect-sim-cocos/control/summaries/2026-04-17-sect-sim-cocos-quality-*.json` | continue |
| L-013 | 2026-04-17 | Open Search Reset | 按用户要求把 queue 默认策略改回开放搜索：去掉来源白名单、去掉 `site:` 预设、强制保留 `zh/en` 双语查询，并说明 Google 应走浏览器 / MCP 路线 | `tools/build_research_seed_queue.py` / `tools/query_research_candidates.py` / `runtime/knowledge-harvest/2026-04-17-sect-sim-cocos/seed-queue.json` | continue |
| L-014 | 2026-04-17 | Provider Removal | 去掉知识采集控制面中的 provider 概念：不再在 queue、worker CLI、candidate manifest 中暴露 `providers/provider_mode/search_provider` | `tools/build_research_seed_queue.py` / `tools/query_research_candidates.py` / `runtime/knowledge-harvest/2026-04-17-sect-sim-cocos/seed-queue.json` | continue |
| L-015 | 2026-04-17 | Wiki-First Bootstrap | 新增 `llm_wiki_maintainer.py`，在 Obsidian 中落地《我的宗门》项目级 `raw/wiki/schema` 三层结构，并完成 Karpathy `LLM Wiki` 文章的首条真实入库 | `tools/llm_wiki_maintainer.py` / `/Users/mawei/MyWork/我的知识库/Projects/我的宗门 Wiki/` | continue |
| L-016 | 2026-04-17 | Browser Google Ingest Loop | 新增 Google 浏览器驱动控制面与常驻 Wiki 入库 worker，启动 `search-agent-01 + ingest-agent-01` 双 worker 长跑，并以 `append-batch` 打通 `Google -> candidate queue -> Wiki ingest` 正式链路 | `tools/google_browser_queue.py` / `tools/wiki_ingest_queue_worker.py` / `runtime/knowledge-harvest/2026-04-17-google-browser-wiki/` | continue |
| L-017 | 2026-04-17 | Dedicated Browser Validation | 用专用 Chrome + 固定调试端口验证“新进程复用现有浏览器会话”链路，确认可覆盖 Google/ChatGPT/本地网页调试，并把该模式提升为后续浏览器任务默认策略 | `docs/process/dedicated-browser-workflow.md` / `runtime/browser-broker-tests/` | continue |
| L-018 | 2026-04-17 | Resident Process Packaging | 将现有 CLI 包装为 `browser_harvest_supervisor + browser_google_search_worker + wiki_ingest_queue_worker` 三常驻进程，确认搜索 worker 在专用 Chrome 下只管理两个 Tab，并在修复同标题来源撞名后完成 `6/6` 成功入库回归 | `tools/browser_cdp.py` / `tools/browser_google_search_worker.py` / `tools/browser_harvest_supervisor.py` / `runtime/knowledge-harvest/2026-04-17-browser-supervisor-smoke-2/` | continue |

## Open Questions

- [ ] 第一轮 seed keyword 数量控制在多少最合理
- [ ] 每个 keyword 是按“单 worker 一关键词”还是按“单 worker 一 shard”更稳
- [x] 已按用户要求把 query 默认值改为开放搜索，不再默认先做 source whitelist
- [x] 已确认不能对所有主题统一使用强白名单；至少 `design/business` 需要与技术主题分开策略
- [ ] 是否要按 topic 类型区分 rerun 的来源过滤策略，例如技术文档优先官方文档/社区，分析类主题优先英文分析站点
- [ ] `rimworld-success-analysis` 这类分析型主题是否要在第二轮 rerun 时显式收紧来源，避免论坛/攻略噪声
- [x] 已为 Google 补 browser-driven 搜索执行器控制面，不再把纯 HTTP parser 当主方案
- [x] `storylet-narrative-system` 这类基础理论主题已放宽年份窗口，转为“长期参考型”补采策略
- [ ] ingest 阶段是否需要先做 URL canonicalization 的独立预处理
- [x] 已确认 ingest 阶段需要优先排除视频 / 直播 / 应用商店页，否则会稀释平台主题的知识密度

## Related Issues / ADRs / Plans

- `docs/project/delivery-ledger.md`
- `docs/process/knowledge-harvest-workflow.md`
