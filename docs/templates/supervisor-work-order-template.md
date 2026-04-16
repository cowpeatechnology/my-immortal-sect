# Supervisor Work Order Template

**Work Order ID**: `WO-xxx`  
**关联 Feature**: `F-xxx-<slug>`  
**状态**: draft / active / handed-off / closed  
**Owner**: `<agent role>`  
**Reviewer / Verifier**: `<agent role>`  
**最后更新**: `YYYY-MM-DD`

## Objective

一句话说明本次工作单要完成什么。

## Context

这次工作为什么要做，它属于哪个研发循环，解决什么问题。

## Authority Docs

- `AGENTS.md`
- `AGENTS.md`
- `docs/...`

## Write Scope

明确允许修改的文件或目录：

- `...`

## No-Touch Scope

明确禁止修改的文件或目录：

- `...`

## Deliverable

这次交付物具体是什么：

- 文档
- 代码
- 资源
- 工具

## Runtime Contract

如果本次工作会产生可运行结果，主管下发时必须明确：

- 启动命令
- 依赖服务或环境前提
- 测试入口地址 / 页面 / 场景
- 期望的最小验收流程
- 主管或 QA 应观察到的关键结果

## Validation Required

交付前必须完成的验证：

- 测试
- QA review
- 人工验收

## Record Targets

本轮必须回写的文档：

- `docs/project/delivery-ledger.md`
- `docs/features/F-xxx-<slug>.md`

## Notes For Worker

给 worker 的额外约束或提醒。
