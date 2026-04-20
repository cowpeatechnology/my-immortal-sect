# Sect Map SVG Asset Pack v1

本目录用于维护《我的宗门》宗门地图第一批 SVG 资产的**工具侧规格与生成脚本**。

运行时真正给 Cocos 导入和小游戏加载的 SVG 输出，仍然落到：

- `client/my-immortal-sect/assets/resources/generated-buildings/sect-map-svg/`

这里不再放会被 Cocos 当成资源导入的 Node 工具脚本、预览拼图或仅供人阅读的规格文档。

## 固定规则

- 输出格式：`SVG`
- 背景：透明
- 朝向：固定单朝向 2.5D / 斜 45 度俯视
- 锚点：`底边中心` 对齐 footprint `底边中心`
- 接触要求：底部必须有清晰接地感，不允许漂浮
- footprint：严格遵守当前项目规则，不做 L 形、不做圆形、不做斜切
- 状态：本批只做 `active / 正常` 版
- 地面装饰：不自带大面积公共地台，只允许建筑自身基础、台阶、石沿、种植床等与 footprint 强相关的局部结构

## Footprint 公式

```text
footprintWidth = (tileCols + tileRows) * 64
footprintHeight = (tileCols + tileRows) * 32
```

## 资产清单

| ID | 中文名 | 占格 | footprint | SVG 画布 | 视觉关键词 | 高度级别 | 主材质 / 识别元素 |
|---|---|---:|---:|---:|---|---|---|
| `main_hall` | 主殿 | `3x3` | `384x192` | `384x336` | 木石宗门主殿、三层屋顶、中轴台阶 | 中高 | 木构、石基、青瓦、金饰、牌匾感 |
| `disciple_quarters` | 弟子居 | `2x3` | `320x160` | `320x272` | 长向宿舍、生活感、连廊 | 中 | 木构、灰青瓦、灯笼、生活器具 |
| `warehouse` | 仓库 | `2x2` | `256x128` | `256x224` | 厚重储藏、箱笼、侧棚 | 中 | 木石混合、重檐、小货堆 |
| `herb_garden` | 药圃 | `2x2` | `256x128` | `256x192` | 药田、棚架、晾架、草木感 | 低 | 土床、棚架、药草、晾晒盘 |
| `guard_tower` | 护山台 | `1x2` | `192x96` | `192x288` | 警戒塔楼、眺望台、塔铃 | 高 | 石基、木塔、瓦顶、旗帜 |
| `spirit_wood` | 灵木 | `1x1` | `128x64` | `128x128` | 灵木桩、木堆、灵叶 | 低 | 木料、嫩叶、轻灵气 |
| `spirit_stone` | 灵石 | `1x1` | `128x64` | `128x128` | 石簇、矿脉、灵光裂面 | 低 | 青灰岩、晶簇、石纹 |
| `herb` | 药草 | `1x1` | `128x64` | `128x112` | 草药丛、药叶、采药篮 | 低 | 草叶、花穗、竹篮 |

## 文件约定

- 运行时输出：`client/my-immortal-sect/assets/resources/generated-buildings/sect-map-svg/*.svg`
- 工具脚本：`tools/asset-pipeline/sect-map-svg/generate-sect-map-svg-assets.mjs`
- 预览总览：`tools/asset-pipeline/sect-map-svg/sect-map-svg-preview.svg`
- 如果后续 `supervisor` 确认要接入运行态，可以再决定是否复制到 `tilemaps/sect-map/` 或转成导入态 PNG / 图集

## 注意事项

- Node/Python 生成器、预览拼图、资产规格文档不得放进 `client/.../assets/`，否则 Cocos 会把它们当项目资源导入。
- 当前仓库的 Cocos Editor 尚未为这些新 SVG 生成 `.meta`，后续如需正式挂到运行时资源，需要在编辑器中导入一次。
- 本批重点是“远看能懂”和 footprint 准确，不追求第一版就把细节堆到 UI 图标级复杂度。
