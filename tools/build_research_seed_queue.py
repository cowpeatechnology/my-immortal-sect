#!/usr/bin/env python3
"""Build or append a seed queue for the knowledge harvesting workflow."""

from __future__ import annotations

import argparse
import hashlib
import json
from copy import deepcopy
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, Iterable, List
from urllib.parse import quote_plus


DEFAULT_PROTOCOL_VERSION = "kc.v1"
DEFAULT_RUN_ID = "2026-04-17-sect-sim-cocos"
DEFAULT_OUTPUT = f"runtime/knowledge-harvest/{DEFAULT_RUN_ID}/seed-queue.json"
DEFAULT_PROFILE = "sect-sim-cocos"
DEFAULT_TARGET_COUNT = 250
DEFAULT_YEAR_FROM = 2024
DEFAULT_YEAR_TO = 2026

DEFAULT_REQUIRED_FIELDS = [
    "title",
    "url",
    "published_at",
    "source_domain",
    "query_id",
    "snippet",
    "language",
    "dedupe_key",
]

DEFAULT_QUERY_STRATEGY = {
    "domain_mode": "allow_any",
    "include_domains": [],
    "exclude_domains": [],
    "skip_index_pages": True,
}

ANALYSIS_EXCLUDE_DOMAINS = [
    "bbs.3dmgame.com",
    "bing.com",
    "zhidao.baidu.com",
    "steamcommunity.com",
    "zhihu.com",
    "wikipedia.org",
]

LOW_QUALITY_CONTENT_DOMAINS = [
    "zhihu.com",
    "zhidao.baidu.com",
    "baidu.com",
    "bbs.3dmgame.com",
    "douyin.com",
    "bilibili.com",
    "v.qq.com",
    "iqiyi.com",
    "huya.com",
    "douyu.com",
    "play.google.com",
]

PROFILE_TOPICS = {
    "sect-sim-cocos": [
        {
            "id": "cocos-creator-tilemap",
            "keyword": "Cocos Creator TileMap",
            "category": "client.tilemap",
            "priority": "p0",
            "zh_terms": ["Cocos Creator TileMap", "Cocos Creator 瓦片图", "Cocos Creator TiledMap"],
            "en_terms": ["Cocos Creator tilemap", "Cocos Creator tiled map"],
            "prefer_domains": ["docs.cocos.com", "forum.cocos.org", "blog.csdn.net", "juejin.cn"],
            "notes": "基础瓦片地图能力，覆盖文档、教程、踩坑与版本变更。",
        },
        {
            "id": "cocos-creator-isometric-tilemap",
            "keyword": "Cocos Creator 等角 TileMap",
            "category": "client.tilemap",
            "priority": "p0",
            "zh_terms": ["Cocos Creator 等角 TileMap", "Cocos Creator 等距地图", "Cocos Creator isometric tilemap"],
            "en_terms": ["Cocos Creator isometric tilemap", "Cocos Creator isometric map"],
            "prefer_domains": ["docs.cocos.com", "forum.cocos.org", "blog.csdn.net", "juejin.cn"],
            "notes": "对当前宗门主地图最直接相关。",
        },
        {
            "id": "cocos-creator-tilemap-performance",
            "keyword": "Cocos Creator TileMap 性能优化",
            "category": "client.performance",
            "priority": "p0",
            "zh_terms": ["Cocos Creator TileMap 性能优化", "Cocos Creator 大地图 优化", "Cocos Creator 瓦片地图 性能"],
            "en_terms": ["Cocos Creator tilemap performance", "Cocos Creator large map optimization"],
            "prefer_domains": ["docs.cocos.com", "forum.cocos.org", "blog.csdn.net", "juejin.cn"],
            "notes": "大地图、移动端、小游戏运行负载相关。",
        },
        {
            "id": "cocos-creator-map-camera-input",
            "keyword": "Cocos Creator 地图拖拽 缩放 输入",
            "category": "client.input",
            "priority": "p0",
            "zh_terms": ["Cocos Creator 地图拖拽 缩放", "Cocos Creator 摄像机 拖拽", "Cocos Creator 触摸 地图 输入"],
            "en_terms": ["Cocos Creator map drag zoom", "Cocos Creator touch camera input"],
            "prefer_domains": ["forum.cocos.org", "blog.csdn.net", "juejin.cn"],
            "notes": "地图交互、相机拖动、触摸响应。",
        },
        {
            "id": "cocos-creator-wechat-mini-game-optimization",
            "keyword": "Cocos Creator 微信小游戏 优化",
            "category": "client.platform",
            "priority": "p0",
            "zh_terms": ["Cocos Creator 微信小游戏 优化", "Cocos Creator 小游戏 性能", "微信小游戏 Cocos Creator"],
            "en_terms": ["Cocos Creator WeChat mini game optimization"],
            "prefer_domains": ["docs.cocos.com", "forum.cocos.org", "blog.csdn.net"],
            "notes": "目标平台强相关。",
        },
        {
            "id": "wechat-mini-game-development",
            "keyword": "微信小游戏 开发",
            "category": "platform.wechat",
            "priority": "p0",
            "zh_terms": ["微信小游戏 开发", "微信小游戏 技术", "微信小游戏 性能 优化"],
            "en_terms": ["WeChat mini game development", "WeChat mini game optimization"],
            "prefer_domains": ["developers.weixin.qq.com", "blog.csdn.net", "juejin.cn"],
            "notes": "平台规则、包体、性能和工程约束。",
            "custom_query_variants": [
                {"suffix": "zh-official", "query": "微信小游戏 开发 site:developers.weixin.qq.com {year_from} {year_to}", "language": "zh"},
                {"suffix": "zh-official-performance", "query": "微信小游戏 性能 优化 site:developers.weixin.qq.com {year_from} {year_to}", "language": "zh"},
                {"suffix": "zh-juejin", "query": "微信小游戏 开发 site:juejin.cn {year_from} {year_to}", "language": "zh"},
                {"suffix": "zh-csdn", "query": "微信小游戏 开发 site:blog.csdn.net {year_from} {year_to}", "language": "zh"},
            ],
            "query_strategy_overrides": {
                "include_domains": ["developers.weixin.qq.com", "blog.csdn.net", "juejin.cn"],
                "exclude_domains": LOW_QUALITY_CONTENT_DOMAINS,
            },
        },
        {
            "id": "cocos-creator-douyin-mini-game",
            "keyword": "Cocos Creator 抖音小游戏",
            "category": "client.platform",
            "priority": "p1",
            "zh_terms": ["Cocos Creator 抖音小游戏", "Cocos Creator 抖音 小游戏 适配", "抖音小游戏 Cocos Creator"],
            "en_terms": ["Cocos Creator Douyin mini game"],
            "prefer_domains": ["docs.cocos.com", "blog.csdn.net", "juejin.cn"],
            "notes": "目标平台适配和发布流程。",
        },
        {
            "id": "douyin-mini-game-development",
            "keyword": "抖音小游戏 开发",
            "category": "platform.douyin",
            "priority": "p1",
            "zh_terms": ["抖音小游戏 开发", "抖音小游戏 技术", "抖音小游戏 性能 适配"],
            "en_terms": ["Douyin mini game development", "TikTok mini game development China"],
            "prefer_domains": ["developer.open-douyin.com", "blog.csdn.net", "juejin.cn"],
            "notes": "平台能力、审核、适配和性能要求。",
        },
        {
            "id": "cocos-creator-general",
            "keyword": "Cocos Creator",
            "category": "client.engine",
            "priority": "p1",
            "zh_terms": ["Cocos Creator", "Cocos Creator 教程", "Cocos Creator 实战"],
            "en_terms": ["Cocos Creator", "Cocos Creator tutorial"],
            "prefer_domains": ["docs.cocos.com", "forum.cocos.org", "blog.csdn.net", "juejin.cn"],
            "notes": "通用引擎使用、版本迁移、常见工程实践。",
        },
        {
            "id": "cocos-creator-asset-bundle-loading",
            "keyword": "Cocos Creator Asset Bundle 加载",
            "category": "client.assets",
            "priority": "p1",
            "zh_terms": ["Cocos Creator Asset Bundle 加载", "Cocos Creator 资源分包", "Cocos Creator 动态加载 资源"],
            "en_terms": ["Cocos Creator asset bundle loading", "Cocos Creator dynamic asset loading"],
            "prefer_domains": ["docs.cocos.com", "forum.cocos.org", "blog.csdn.net"],
            "notes": "大地图与建筑资源动态加载相关。",
        },
        {
            "id": "cocos-creator-spine-animation",
            "keyword": "Cocos Creator Spine 动画",
            "category": "client.animation",
            "priority": "p1",
            "zh_terms": ["Cocos Creator Spine 动画", "Cocos Creator 使用 Spine", "Cocos Creator Spine 性能"],
            "en_terms": ["Cocos Creator Spine animation", "Spine in Cocos Creator"],
            "prefer_domains": ["docs.cocos.com", "forum.cocos.org", "blog.csdn.net"],
            "notes": "角色表现、性能预算和工作流。",
        },
        {
            "id": "cocos-creator-2d-lighting",
            "keyword": "Cocos Creator 2D 光效",
            "category": "client.rendering",
            "priority": "p1",
            "zh_terms": ["Cocos Creator 2D 光效", "Cocos Creator 2D 灯光", "Cocos Creator 2D 光照 效果"],
            "en_terms": ["Cocos Creator 2D lighting", "2D light effect in Cocos Creator"],
            "prefer_domains": ["forum.cocos.org", "blog.csdn.net", "juejin.cn"],
            "notes": "氛围渲染、性能成本与实现方式。",
        },
        {
            "id": "battle-vfx-production",
            "keyword": "战斗特效 制作",
            "category": "client.vfx",
            "priority": "p1",
            "zh_terms": ["战斗特效 制作", "游戏 战斗特效", "技能特效 设计 制作"],
            "en_terms": ["battle vfx production", "game combat visual effects"],
            "prefer_domains": ["blog.csdn.net", "juejin.cn", "80.lv"],
            "notes": "战斗视觉反馈、资源流程与性能折中。",
        },
        {
            "id": "tilemap-pathfinding-occlusion",
            "keyword": "TileMap 寻路 遮挡",
            "category": "client.tilemap",
            "priority": "p0",
            "zh_terms": ["TileMap 寻路 遮挡", "瓦片地图 寻路 遮挡", "Cocos Creator TileMap 寻路"],
            "en_terms": ["tilemap pathfinding occlusion", "tilemap pathfinding visibility"],
            "prefer_domains": ["docs.cocos.com", "forum.cocos.org", "blog.csdn.net", "juejin.cn"],
            "notes": "建筑遮挡、角色寻路、地图层与碰撞。",
        },
        {
            "id": "colony-sim-job-system",
            "keyword": "colony sim job system",
            "category": "design.simulation",
            "priority": "p0",
            "zh_terms": ["模拟经营 工作系统", "殖民模拟 工作分配", "RimWorld 工作系统"],
            "en_terms": ["colony sim job system", "RimWorld work system", "colony sim task assignment"],
            "prefer_domains": ["gamasutra.com", "gamedeveloper.com", "medium.com", "substack.com"],
            "notes": "弟子派工、任务切换、岗位系统。",
        },
        {
            "id": "colony-sim-work-priority",
            "keyword": "colony sim work priority",
            "category": "design.simulation",
            "priority": "p0",
            "zh_terms": ["模拟经营 优先级 系统", "RimWorld 工作优先级", "殖民模拟 优先级"],
            "en_terms": ["colony sim work priority", "RimWorld work priority", "colony sim priority system"],
            "prefer_domains": ["gamedeveloper.com", "medium.com", "reddit.com"],
            "notes": "与 RimWorld 风格的核心循环强相关。",
        },
        {
            "id": "game-development-patterns",
            "keyword": "游戏开发模式",
            "category": "architecture.patterns",
            "priority": "p1",
            "zh_terms": ["游戏开发模式", "游戏开发 架构 模式", "游戏项目 设计模式"],
            "en_terms": ["game development patterns", "game architecture patterns"],
            "prefer_domains": ["gamedeveloper.com", "medium.com", "substack.com"],
            "notes": "系统组织、模块边界和通用工程模式。",
        },
        {
            "id": "management-game-design",
            "keyword": "模拟经营 游戏设计",
            "category": "design.simulation",
            "priority": "p0",
            "zh_terms": ["模拟经营 游戏设计", "经营游戏 设计思路", "模拟经营 核心循环"],
            "en_terms": ["management game design", "simulation management game design"],
            "prefer_domains": ["gamedeveloper.com", "medium.com", "substack.com"],
            "notes": "核心循环、数值节奏和玩家反馈。",
        },
        {
            "id": "deterministic-simulation-game-architecture",
            "keyword": "deterministic simulation game architecture",
            "category": "architecture.simulation",
            "priority": "p0",
            "zh_terms": ["游戏 确定性模拟 架构", "模拟经营 确定性", "游戏 事件驱动 模拟 架构"],
            "en_terms": ["deterministic simulation game architecture", "deterministic colony sim architecture"],
            "prefer_domains": ["gamedeveloper.com", "medium.com", "substack.com"],
            "notes": "服务端权威模拟和可重放性相关。",
        },
        {
            "id": "actor-model-game-engine",
            "keyword": "actor 模式 游戏引擎",
            "category": "architecture.simulation",
            "priority": "p0",
            "zh_terms": ["actor 模式 游戏引擎", "Actor 模式 游戏开发", "游戏服务器 Actor 架构"],
            "en_terms": ["actor model game engine", "actor model game architecture", "actor model game server"],
            "prefer_domains": ["getakka.net", "gamedeveloper.com", "medium.com", "substack.com"],
            "notes": "对应后续 Hollywood / actor 化服务端设计。",
        },
        {
            "id": "multiplayer-online-game-development",
            "keyword": "多人在线 游戏开发",
            "category": "architecture.online",
            "priority": "p1",
            "zh_terms": ["多人在线 游戏开发", "在线游戏 服务器 架构", "多人同步 游戏开发"],
            "en_terms": ["multiplayer online game development", "online game server architecture", "multiplayer game sync"],
            "prefer_domains": ["gamedeveloper.com", "medium.com", "substack.com"],
            "notes": "虽然当前不是 full MMO，但联机与状态同步经验有参考价值。",
        },
        {
            "id": "storylet-narrative-system",
            "keyword": "storylet narrative system",
            "category": "design.narrative",
            "priority": "p1",
            "zh_terms": ["Storylet 系统", "动态叙事 Storylet", "游戏 叙事 事件系统"],
            "en_terms": ["storylet narrative system", "storylet design game", "emergent narrative storylet"],
            "prefer_domains": ["gamedeveloper.com", "substack.com", "inklestudios.com"],
            "notes": "对应宗门因果事件与命运线。",
        },
        {
            "id": "ai-roleplay-game-characters",
            "keyword": "使用 AI 来扮演游戏角色",
            "category": "design.ai",
            "priority": "p1",
            "zh_terms": ["使用 AI 扮演 游戏角色", "AI NPC 角色扮演", "LLM 游戏角色"],
            "en_terms": ["AI roleplay game characters", "LLM NPC roleplay", "AI NPC game design"],
            "prefer_domains": ["openai.com", "gamedeveloper.com", "medium.com", "substack.com"],
            "notes": "用于未来弟子/NPC 对话与行为生成参考。",
            "custom_query_variants": [
                {"suffix": "en-openai", "query": "AI NPC roleplay site:openai.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-gamedeveloper", "query": "AI NPC game design site:gamedeveloper.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-substack", "query": "LLM NPC roleplay site:substack.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-medium", "query": "AI game characters site:medium.com {year_from} {year_to}", "language": "en"},
            ],
            "query_strategy_overrides": {
                "include_domains": ["openai.com", "gamedeveloper.com", "medium.com", "substack.com"],
                "exclude_domains": LOW_QUALITY_CONTENT_DOMAINS,
            },
        },
        {
            "id": "resource-chain-management-game-design",
            "keyword": "resource chain management game design",
            "category": "design.economy",
            "priority": "p1",
            "zh_terms": ["资源链 模拟经营 设计", "生产链 游戏设计", "资源循环 模拟经营"],
            "en_terms": ["resource chain management game design", "production chain simulation design"],
            "prefer_domains": ["gamedeveloper.com", "medium.com", "substack.com"],
            "notes": "宗门资源建筑、供给链、加工链。",
            "custom_query_variants": [
                {"suffix": "en-gamedeveloper", "query": "resource chain management game design site:gamedeveloper.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-substack", "query": "production chain simulation design site:substack.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-medium", "query": "resource chain game design site:medium.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-deconstructor", "query": "economy design game site:deconstructoroffun.com {year_from} {year_to}", "language": "en"},
            ],
            "query_strategy_overrides": {
                "include_domains": ["gamedeveloper.com", "medium.com", "substack.com", "deconstructoroffun.com"],
                "exclude_domains": LOW_QUALITY_CONTENT_DOMAINS,
            },
        },
        {
            "id": "colony-sim-ux-readability",
            "keyword": "colony sim UX readability",
            "category": "design.ux",
            "priority": "p1",
            "zh_terms": ["模拟经营 可读性 设计", "殖民模拟 UI 可读性", "RimWorld UI 设计"],
            "en_terms": ["colony sim UX readability", "colony sim readability", "RimWorld UI readability"],
            "prefer_domains": ["gamedeveloper.com", "medium.com", "substack.com"],
            "notes": "地图信息密度、任务状态与建筑可读性。",
            "custom_query_variants": [
                {"suffix": "en-gamedeveloper", "query": "colony sim readability site:gamedeveloper.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-substack", "query": "colony sim UX site:substack.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-medium", "query": "colony sim UI readability site:medium.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-rps", "query": "RimWorld UI readability site:rockpapershotgun.com {year_from} {year_to}", "language": "en"},
            ],
            "query_strategy_overrides": {
                "include_domains": ["gamedeveloper.com", "medium.com", "substack.com", "rockpapershotgun.com"],
                "exclude_domains": LOW_QUALITY_CONTENT_DOMAINS,
            },
        },
        {
            "id": "game-2d-development",
            "keyword": "2D 游戏开发",
            "category": "client.2d",
            "priority": "p1",
            "zh_terms": ["2D 游戏开发", "2D 游戏 程序 开发", "2D 游戏 制作 技术"],
            "en_terms": ["2D game development", "2D game programming", "2D game production"],
            "prefer_domains": ["docs.cocos.com", "gamedeveloper.com", "80.lv", "medium.com"],
            "notes": "扩展到通用 2D 游戏研发实践，补充引擎之外的方法论与流程。",
            "custom_query_variants": [
                {"suffix": "en-gamedeveloper", "query": "2D game development site:gamedeveloper.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-80lv", "query": "2D game development site:80.lv {year_from} {year_to}", "language": "en"},
                {"suffix": "en-medium", "query": "2D game programming site:medium.com {year_from} {year_to}", "language": "en"},
                {"suffix": "zh-cocos", "query": "2D 游戏开发 site:docs.cocos.com {year_from} {year_to}", "language": "zh"},
            ],
            "query_strategy_overrides": {
                "include_domains": ["docs.cocos.com", "gamedeveloper.com", "80.lv", "medium.com"],
                "exclude_domains": LOW_QUALITY_CONTENT_DOMAINS,
            },
        },
        {
            "id": "game-design-general",
            "keyword": "游戏设计",
            "category": "design.general",
            "priority": "p1",
            "zh_terms": ["游戏设计", "游戏设计 方法论", "游戏设计 核心循环"],
            "en_terms": ["game design", "game design framework", "game design core loop"],
            "prefer_domains": ["gamedeveloper.com", "substack.com", "medium.com"],
            "notes": "补充通用设计方法、核心循环、反馈与迭代框架。",
            "custom_query_variants": [
                {"suffix": "en-gamedeveloper", "query": "game design core loop site:gamedeveloper.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-substack", "query": "game design framework site:substack.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-medium", "query": "game design framework site:medium.com {year_from} {year_to}", "language": "en"},
                {"suffix": "zh-gamedeveloper", "query": "游戏设计 site:gamedeveloper.com {year_from} {year_to}", "language": "zh"},
            ],
            "query_strategy_overrides": {
                "include_domains": ["gamedeveloper.com", "substack.com", "medium.com"],
                "exclude_domains": LOW_QUALITY_CONTENT_DOMAINS,
            },
        },
        {
            "id": "game-commercialization",
            "keyword": "游戏商业化",
            "category": "business.monetization",
            "priority": "p1",
            "zh_terms": ["游戏商业化", "游戏 变现 设计", "手游 商业化"],
            "en_terms": ["game monetization", "game commercialization", "mobile game monetization"],
            "prefer_domains": ["gameworldobserver.com", "mobilegamer.biz", "gamedeveloper.com", "deconstructoroffun.com"],
            "notes": "补充商业模式、变现结构与项目可持续性参考。",
            "custom_query_variants": [
                {"suffix": "en-deconstructor", "query": "game monetization site:deconstructoroffun.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-gwo", "query": "game monetization site:gameworldobserver.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-mobilegamer", "query": "mobile game monetization site:mobilegamer.biz {year_from} {year_to}", "language": "en"},
                {"suffix": "en-gamedeveloper", "query": "game monetization site:gamedeveloper.com {year_from} {year_to}", "language": "en"},
            ],
            "query_strategy_overrides": {
                "include_domains": ["gameworldobserver.com", "mobilegamer.biz", "gamedeveloper.com", "deconstructoroffun.com", "businessofapps.com"],
                "exclude_domains": LOW_QUALITY_CONTENT_DOMAINS,
            },
        },
        {
            "id": "free-to-play-economy-design",
            "keyword": "免费游戏 货币化 设计",
            "category": "business.monetization",
            "priority": "p1",
            "zh_terms": ["免费游戏 货币化 设计", "F2P 经济 设计", "手游 付费 设计"],
            "en_terms": ["free to play economy design", "F2P monetization design", "mobile game economy design"],
            "prefer_domains": ["deconstructoroffun.com", "gamedeveloper.com", "gameworldobserver.com", "medium.com"],
            "notes": "补充 F2P 经济、付费点和数值节奏设计参考。",
            "custom_query_variants": [
                {"suffix": "en-deconstructor", "query": "free to play economy design site:deconstructoroffun.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-gwo", "query": "mobile game economy design site:gameworldobserver.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-gamedeveloper", "query": "F2P monetization design site:gamedeveloper.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-businessofapps", "query": "mobile game monetization site:businessofapps.com {year_from} {year_to}", "language": "en"},
            ],
            "query_strategy_overrides": {
                "include_domains": ["deconstructoroffun.com", "gamedeveloper.com", "gameworldobserver.com", "businessofapps.com"],
                "exclude_domains": LOW_QUALITY_CONTENT_DOMAINS,
            },
        },
        {
            "id": "game-retention-liveops",
            "keyword": "游戏 留存 运营",
            "category": "business.liveops",
            "priority": "p1",
            "zh_terms": ["游戏 留存 运营", "游戏 LiveOps", "手游 长线运营"],
            "en_terms": ["game retention liveops", "mobile game live ops", "game retention design"],
            "prefer_domains": ["deconstructoroffun.com", "gameworldobserver.com", "mobilegamer.biz", "gamedeveloper.com"],
            "notes": "补充长期留存、活动运营与内容节奏设计参考。",
            "custom_query_variants": [
                {"suffix": "en-deconstructor", "query": "game retention live ops site:deconstructoroffun.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-gwo", "query": "mobile game live ops site:gameworldobserver.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-mobilegamer", "query": "game retention design site:mobilegamer.biz {year_from} {year_to}", "language": "en"},
                {"suffix": "en-gamedeveloper", "query": "game retention design site:gamedeveloper.com {year_from} {year_to}", "language": "en"},
            ],
            "query_strategy_overrides": {
                "include_domains": ["deconstructoroffun.com", "gameworldobserver.com", "mobilegamer.biz", "gamedeveloper.com"],
                "exclude_domains": LOW_QUALITY_CONTENT_DOMAINS,
            },
        },
        {
            "id": "rimworld-success-analysis",
            "keyword": "RimWorld 游戏成功原因",
            "category": "design.reference",
            "priority": "p0",
            "zh_terms": ["RimWorld 游戏成功原因", "RimWorld 设计分析", "环世界 成功 原因"],
            "en_terms": ["why RimWorld succeeded", "RimWorld design analysis", "RimWorld success factors"],
            "prefer_domains": ["gamedeveloper.com", "medium.com", "substack.com", "reddit.com"],
            "notes": "用于提炼成功要素，而不是照搬表层系统。",
            "custom_query_variants": [
                {"suffix": "zh-analysis", "query": "RimWorld 设计分析 {year_from} {year_to}", "language": "zh"},
                {"suffix": "zh-success", "query": "RimWorld 成功 原因 {year_from} {year_to}", "language": "zh"},
                {"suffix": "en-analysis", "query": "RimWorld design analysis {year_from} {year_to}", "language": "en"},
                {"suffix": "en-postmortem", "query": "RimWorld postmortem analysis {year_from} {year_to}", "language": "en"},
                {"suffix": "en-success", "query": "RimWorld success factors {year_from} {year_to}", "language": "en"},
                {"suffix": "en-gamedeveloper", "query": "RimWorld design analysis site:gamedeveloper.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-medium", "query": "RimWorld design analysis site:medium.com {year_from} {year_to}", "language": "en"},
                {"suffix": "en-substack", "query": "RimWorld design analysis site:substack.com {year_from} {year_to}", "language": "en"},
            ],
            "query_strategy_overrides": {
                "include_domains": [
                    "gamedeveloper.com",
                    "substack.com",
                    "medium.com",
                    "pcgamer.com",
                    "rockpapershotgun.com",
                    "reddit.com",
                ],
                "exclude_domains": ANALYSIS_EXCLUDE_DOMAINS,
            },
        },
    ]
}


def now_iso() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


def make_custom_id(keyword: str) -> str:
    digest = hashlib.sha1(keyword.encode("utf-8")).hexdigest()[:10]
    return f"custom-{digest}"


def search_urls(query: str) -> Dict[str, str]:
    encoded = quote_plus(query)
    return {
        "google": f"https://www.google.com/search?q={encoded}",
        "bing": f"https://www.bing.com/search?q={encoded}",
        "duckduckgo": f"https://duckduckgo.com/?q={encoded}",
    }


def build_query_strategy(topic: Dict, *, respect_topic_overrides: bool = False) -> Dict:
    strategy = deepcopy(DEFAULT_QUERY_STRATEGY)
    if not respect_topic_overrides:
        return strategy

    category = topic.get("category", "")

    if category == "design.reference":
        strategy["exclude_domains"] = ANALYSIS_EXCLUDE_DOMAINS[:]

    allowed_override_keys = {
        "domain_mode",
        "include_domains",
        "exclude_domains",
        "skip_index_pages",
    }
    overrides = topic.get("query_strategy_overrides") or {}
    for key, value in overrides.items():
        if key not in allowed_override_keys:
            continue
        strategy[key] = deepcopy(value)
    return strategy


def dedupe_terms(terms: Iterable[str]) -> List[str]:
    unique_terms: List[str] = []
    for raw_term in terms:
        term = (raw_term or "").strip()
        if not term or term in unique_terms:
            continue
        unique_terms.append(term)
    return unique_terms


def build_query_variants(
    topic: Dict,
    year_from: int,
    year_to: int,
    *,
    use_topic_custom_variants: bool = False,
    use_prefer_domains: bool = False,
) -> List[Dict]:
    custom_variants = topic.get("custom_query_variants") or []
    if use_topic_custom_variants and custom_variants:
        variants = []
        for variant in custom_variants:
            query = variant["query"].format(year_from=year_from, year_to=year_to)
            variants.append(
                {
                    "id": f"{topic['id']}-{variant['suffix']}",
                    "query": query,
                    "language": variant["language"],
                    "search_urls": search_urls(query),
                }
            )
        return variants

    zh_terms = dedupe_terms(topic.get("zh_terms") or [topic["keyword"]])
    en_terms = dedupe_terms(topic.get("en_terms") or [topic["keyword"]])
    domains = dedupe_terms(topic.get("prefer_domains") or []) if use_prefer_domains else []
    year_window = f"{year_from} {year_to}"

    queries: List[tuple[str, str, str]] = []
    if zh_terms:
        queries.append(("zh-core", f"{zh_terms[0]} {year_window}", "zh"))
        queries.append(("zh-practice", f"{zh_terms[0]} 教程 实战 {year_window}", "zh"))
    if len(zh_terms) > 1:
        queries.append(("zh-alt", f"{zh_terms[1]} {year_window}", "zh"))
    if en_terms:
        queries.append(("en-core", f"{en_terms[0]} {year_window}", "en"))
        queries.append(("en-practice", f"{en_terms[0]} tutorial guide best practices {year_window}", "en"))
    if len(en_terms) > 1:
        queries.append(("en-alt", f"{en_terms[1]} {year_window}", "en"))

    if domains:
        queries.append(("zh-domain", f"{zh_terms[0]} site:{domains[0]} {year_window}", "zh"))
    if len(domains) > 1:
        queries.append(("community-domain", f"{zh_terms[0]} site:{domains[1]} {year_window}", "zh"))

    variants = []
    seen_queries: set[str] = set()
    for suffix, query, language in queries:
        normalized_query = " ".join(query.split())
        if not normalized_query or normalized_query in seen_queries:
            continue
        seen_queries.add(normalized_query)
        variants.append(
            {
                "id": f"{topic['id']}-{suffix}",
                "query": normalized_query,
                "language": language,
                "search_urls": search_urls(normalized_query),
            }
        )
    return variants


def build_topic_item(
    topic: Dict,
    *,
    run_id: str,
    year_from: int,
    year_to: int,
    target_count: int,
    protocol_version: str,
    respect_topic_overrides: bool = False,
    use_topic_custom_variants: bool = False,
    emit_prefer_domains: bool = False,
    use_prefer_domains_in_queries: bool = False,
) -> Dict:
    topic_id = topic["id"]
    return {
        "protocol_version": protocol_version,
        "run_id": run_id,
        "seed_id": topic_id,
        "id": topic_id,
        "keyword": topic["keyword"],
        "category": topic["category"],
        "priority": topic["priority"],
        "target_article_count": target_count,
        "year_from": year_from,
        "year_to": year_to,
        "prefer_domains": topic.get("prefer_domains", []) if emit_prefer_domains else [],
        "required_fields": deepcopy(DEFAULT_REQUIRED_FIELDS),
        "notes": topic.get("notes", ""),
        "status": {
            "phase": "seeded",
            "claimed": False,
            "completed": False,
            "updated_at": now_iso(),
        },
        "query_strategy": build_query_strategy(topic, respect_topic_overrides=respect_topic_overrides),
        "search_queries": build_query_variants(
            topic,
            year_from,
            year_to,
            use_topic_custom_variants=use_topic_custom_variants,
            use_prefer_domains=use_prefer_domains_in_queries,
        ),
        "outputs": {
            "candidate_manifest": f"control/candidates/{topic_id}.jsonl",
            "ingest_manifest": f"control/ingest/{topic_id}.jsonl",
            "claim_file": f"control/claims/{topic_id}.json",
            "heartbeat_file": f"control/heartbeats/{topic_id}.json",
            "ingest_claim_file": f"control/ingest-claims/{topic_id}.json",
            "ingest_heartbeat_file": f"control/ingest-heartbeats/{topic_id}.json",
            "payload_root": f"payload/articles/{topic_id}",
        },
    }


def build_custom_topic(keyword: str, *, year_from: int, year_to: int, target_count: int) -> Dict:
    topic_id = make_custom_id(keyword)
    return {
        "id": topic_id,
        "keyword": keyword,
        "category": "custom",
        "priority": "p1",
        "zh_terms": [keyword],
        "en_terms": [keyword],
        "prefer_domains": [],
        "notes": "User-supplied custom keyword.",
        "target_article_count": target_count,
        "year_from": year_from,
        "year_to": year_to,
    }


def ensure_parent(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def load_existing(path: Path) -> Dict:
    if not path.exists():
        return {}
    return json.loads(path.read_text(encoding="utf-8"))


def merge_items(existing_items: Iterable[Dict], new_items: Iterable[Dict], replace_existing: bool) -> List[Dict]:
    merged = {item["id"]: item for item in existing_items}
    for item in new_items:
        if item["id"] in merged and not replace_existing:
            continue
        merged[item["id"]] = item
    return [merged[key] for key in sorted(merged.keys())]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Build or append a knowledge harvest seed queue.")
    parser.add_argument("--profile", default=DEFAULT_PROFILE, help=f"Built-in topic profile. Default: {DEFAULT_PROFILE}")
    parser.add_argument("--output", default=DEFAULT_OUTPUT, help=f"Output JSON path. Default: {DEFAULT_OUTPUT}")
    parser.add_argument("--run-id", default=DEFAULT_RUN_ID, help=f"Run identifier. Default: {DEFAULT_RUN_ID}")
    parser.add_argument("--keyword", action="append", default=[], help="Custom keyword to append. Repeatable.")
    parser.add_argument("--append", action="store_true", help="Append new items into an existing queue file.")
    parser.add_argument("--replace-existing", action="store_true", help="Replace existing items with the same id.")
    parser.add_argument("--target-count", type=int, default=DEFAULT_TARGET_COUNT, help=f"Target article count per keyword. Default: {DEFAULT_TARGET_COUNT}")
    parser.add_argument("--year-from", type=int, default=DEFAULT_YEAR_FROM, help=f"Search window start year. Default: {DEFAULT_YEAR_FROM}")
    parser.add_argument("--year-to", type=int, default=DEFAULT_YEAR_TO, help=f"Search window end year. Default: {DEFAULT_YEAR_TO}")
    parser.add_argument("--respect-topic-overrides", action="store_true", help="Apply topic-level source filters from the built-in topic profile. Default is open search without source restrictions.")
    parser.add_argument("--use-topic-custom-query-variants", action="store_true", help="Use topic-authored custom query variants when present. Default uses balanced open zh/en queries.")
    parser.add_argument("--emit-prefer-domains", action="store_true", help="Write topic prefer_domains into the queue metadata. Default omits source preference metadata.")
    parser.add_argument("--use-prefer-domains-in-queries", action="store_true", help="Generate site: queries from topic prefer_domains. Default disables preset site targeting.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    output_path = Path(args.output)
    ensure_parent(output_path)

    profile_topics = PROFILE_TOPICS.get(args.profile)
    if not profile_topics and not args.keyword:
        raise SystemExit(f"Unknown profile: {args.profile}")

    generated_items = []
    for topic in profile_topics or []:
        generated_items.append(
            build_topic_item(
                topic,
                run_id=args.run_id,
                year_from=args.year_from,
                year_to=args.year_to,
                target_count=args.target_count,
                protocol_version=DEFAULT_PROTOCOL_VERSION,
                respect_topic_overrides=args.respect_topic_overrides,
                use_topic_custom_variants=args.use_topic_custom_query_variants,
                emit_prefer_domains=args.emit_prefer_domains,
                use_prefer_domains_in_queries=args.use_prefer_domains_in_queries,
            )
        )
    for keyword in args.keyword:
        generated_items.append(
            build_topic_item(
                build_custom_topic(
                    keyword,
                    year_from=args.year_from,
                    year_to=args.year_to,
                    target_count=args.target_count,
                ),
                run_id=args.run_id,
                year_from=args.year_from,
                year_to=args.year_to,
                target_count=args.target_count,
                protocol_version=DEFAULT_PROTOCOL_VERSION,
                respect_topic_overrides=args.respect_topic_overrides,
                use_topic_custom_variants=args.use_topic_custom_query_variants,
                emit_prefer_domains=args.emit_prefer_domains,
                use_prefer_domains_in_queries=args.use_prefer_domains_in_queries,
            )
        )

    if args.append:
        existing = load_existing(output_path)
        merged_items = merge_items(existing.get("items", []), generated_items, args.replace_existing)
        payload = {
            "protocol_version": DEFAULT_PROTOCOL_VERSION,
            "version": 1,
            "profile": args.profile,
            "run_id": args.run_id,
            "generated_at": now_iso(),
            "notes": "Seed queue for supervisor -> worker knowledge harvesting. Paths are relative to this queue file unless absolute.",
            "items": merged_items,
        }
    else:
        payload = {
            "protocol_version": DEFAULT_PROTOCOL_VERSION,
            "version": 1,
            "profile": args.profile,
            "run_id": args.run_id,
            "generated_at": now_iso(),
            "notes": "Seed queue for supervisor -> worker knowledge harvesting. Paths are relative to this queue file unless absolute.",
            "items": generated_items,
        }

    output_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"Wrote queue: {output_path}")
    print(f"Items: {len(payload['items'])}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
