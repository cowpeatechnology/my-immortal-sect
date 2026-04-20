import fs from 'node:fs';
import pathModule from 'node:path';
import { fileURLToPath } from 'node:url';

const SCRIPT_FILE = fileURLToPath(import.meta.url);
const SCRIPT_DIR = pathModule.dirname(SCRIPT_FILE);
const REPO_ROOT = pathModule.resolve(SCRIPT_DIR, '..', '..', '..');
const OUT_DIR = pathModule.join(
  REPO_ROOT,
  'client',
  'my-immortal-sect',
  'assets',
  'resources',
  'generated-buildings',
  'sect-map-svg',
);
const PREVIEW_OUT = pathModule.join(SCRIPT_DIR, 'sect-map-svg-preview.svg');

const HALF_W = 64;
const HALF_H = 32;

const C = {
  ink: '#4d392a',
  inkSoft: '#7c6047',
  roofTop: '#7b9aa0',
  roofMid: '#5f7d84',
  roofFront: '#425960',
  roofSide: '#2b3d44',
  roofDeep: '#24343a',
  wallTop: '#f5ebd7',
  wallFront: '#e8d1ac',
  wallSide: '#c39a6b',
  wallSideDeep: '#a47b4e',
  stoneTop: '#eee1c5',
  stoneFront: '#cfa676',
  stoneSide: '#9b7049',
  timber: '#96604a',
  timberDeep: '#6b4435',
  gold: '#d9b06a',
  goldDeep: '#a8793f',
  herb: '#7ea660',
  herbDark: '#55763f',
  herbLight: '#a9c77d',
  jade: '#84bcc5',
  jadeSoft: '#a3d1d6',
  water: '#8cb7c6',
  shadow: '#3c2a1b',
  blossom: '#e6bcc7',
  stoneRock: '#8992a1',
  stoneRockDark: '#5f6778',
  bark: '#8c6544',
  barkDark: '#65452f',
  basket: '#b98b4a',
};

const assets = [
  {
    id: 'main_hall',
    label: '主殿',
    cols: 3,
    rows: 3,
    footprintWidth: 384,
    footprintHeight: 192,
    width: 384,
    height: 336,
    render: renderMainHall,
  },
  {
    id: 'disciple_quarters',
    label: '弟子居',
    cols: 2,
    rows: 3,
    footprintWidth: 320,
    footprintHeight: 160,
    width: 320,
    height: 272,
    render: renderDiscipleQuarters,
  },
  {
    id: 'warehouse',
    label: '仓库',
    cols: 2,
    rows: 2,
    footprintWidth: 256,
    footprintHeight: 128,
    width: 256,
    height: 224,
    render: renderWarehouse,
  },
  {
    id: 'herb_garden',
    label: '药圃',
    cols: 2,
    rows: 2,
    footprintWidth: 256,
    footprintHeight: 128,
    width: 256,
    height: 192,
    render: renderHerbGarden,
  },
  {
    id: 'guard_tower',
    label: '护山台',
    cols: 1,
    rows: 2,
    footprintWidth: 192,
    footprintHeight: 96,
    width: 192,
    height: 288,
    render: renderGuardTower,
  },
  {
    id: 'spirit_wood',
    label: '灵木',
    cols: 1,
    rows: 1,
    footprintWidth: 128,
    footprintHeight: 64,
    width: 128,
    height: 128,
    render: renderSpiritWood,
  },
  {
    id: 'spirit_stone',
    label: '灵石',
    cols: 1,
    rows: 1,
    footprintWidth: 128,
    footprintHeight: 64,
    width: 128,
    height: 128,
    render: renderSpiritStone,
  },
  {
    id: 'herb',
    label: '药草',
    cols: 1,
    rows: 1,
    footprintWidth: 128,
    footprintHeight: 64,
    width: 128,
    height: 112,
    render: renderHerb,
  },
];

fs.mkdirSync(OUT_DIR, { recursive: true });

const renderedAssets = assets.map((asset) => {
  const content = asset.render(asset);
  const svg = wrapSvg(asset, content);
  fs.writeFileSync(pathModule.join(OUT_DIR, `${asset.id}.svg`), svg);
  return { ...asset, ...content };
});

fs.writeFileSync(PREVIEW_OUT, buildPreview(renderedAssets));

function wrapSvg(asset, { defs, body, description }) {
  return [
    `<svg width="${asset.width}" height="${asset.height}" viewBox="0 0 ${asset.width} ${asset.height}" fill="none" xmlns="http://www.w3.org/2000/svg">`,
    `  <title>${asset.label}</title>`,
    `  <desc>${description}</desc>`,
    `  <defs>`,
    indent(defs, 4),
    `  </defs>`,
    indent(body, 2),
    `</svg>`,
    '',
  ].join('\n');
}

function buildPreview(items) {
  const width = 1320;
  const height = 860;
  const cellWidth = 300;
  const cellHeight = 330;
  const gapX = 20;
  const gapY = 24;
  const startX = 30;
  const startY = 104;

  const defs = items.map((item) => item.defs).join('\n');
  const cards = items
    .map((item, index) => {
      const col = index % 4;
      const row = Math.floor(index / 4);
      const x = startX + col * (cellWidth + gapX);
      const y = startY + row * (cellHeight + gapY);
      const scale = Math.min((cellWidth - 36) / item.width, (cellHeight - 86) / item.height);
      const drawX = x + cellWidth / 2 - (item.width * scale) / 2;
      const drawY = y + cellHeight - 30 - item.height * scale;

      return [
        `<g transform="translate(${x} ${y})">`,
        `  <rect x="0" y="0" width="${cellWidth}" height="${cellHeight}" rx="26" fill="rgba(255,248,236,0.94)" stroke="rgba(140,98,54,0.16)" stroke-width="2"/>`,
        `  <text x="24" y="34" font-family="'PingFang SC','Hiragino Sans GB',sans-serif" font-size="16" fill="#4d392a">${item.label}</text>`,
        `  <text x="24" y="58" font-family="'SFMono-Regular',monospace" font-size="12" fill="#8a6a49">${item.id}</text>`,
        `  <g transform="translate(${drawX - x} ${drawY - y}) scale(${round(scale, 4)})">`,
        indent(item.body, 4),
        `  </g>`,
        `</g>`,
      ].join('\n');
    })
    .join('\n');

  return [
    `<svg width="${width}" height="${height}" viewBox="0 0 ${width} ${height}" fill="none" xmlns="http://www.w3.org/2000/svg">`,
    `  <defs>`,
    indent(defs, 4),
    `  </defs>`,
    `  <rect width="${width}" height="${height}" fill="#efe5d3"/>`,
    `  <rect x="18" y="18" width="${width - 36}" height="${height - 36}" rx="34" fill="#f8f1e4" stroke="rgba(124,88,52,0.18)" stroke-width="2"/>`,
    `  <text x="42" y="54" font-family="'PingFang SC','Hiragino Sans GB',sans-serif" font-size="24" fill="#4d392a">Sect Map SVG Asset Pack v1</text>`,
    `  <text x="42" y="82" font-family="'PingFang SC','Hiragino Sans GB',sans-serif" font-size="14" fill="#7b6148">2.5D / 45° / transparent background / footprint-bottom-center anchor</text>`,
    indent(cards, 2),
    `</svg>`,
    '',
  ].join('\n');
}

function renderMainHall(asset) {
  const ctx = assetContext(asset);
  const parts = [];
  parts.push(shadow(ctx, 0, 0, 104, 18, 0.18));
  parts.push(mist(ctx, -30, -22, 64, 16, 0.24));

  parts.push(box(ctx, { x: 0.18, y: 0.22, w: 2.64, d: 2.46, z: 18, depth: 20, top: C.stoneTop, front: C.stoneFront, side: C.stoneSide }));
  parts.push(courtyardLines(ctx, 0.48, 0.52, 2.04, 1.86, 20, 4, 4));
  parts.push(stairs(ctx, { x: 1.08, y: 2.02, w: 0.84, d: 0.74, topZ: 18, steps: 4 }));
  parts.push(walkway(ctx, { x: 1.28, y: 1.36, w: 0.48, d: 1.3, z: 20, fill: '#e8cd92' }));

  parts.push(box(ctx, { x: 0.52, y: 1.18, w: 0.64, d: 0.82, z: 66, depth: 40, top: C.wallTop, front: C.wallFront, side: C.wallSide }));
  parts.push(frontPosts(ctx, { x: 0.58, y: 2.0, count: 2, spacing: 0.28, zTop: 90, zBottom: 30, color: C.timber }));
  parts.push(roof(ctx, { x: 0.44, y: 1.08, w: 0.84, d: 1.0, z: 102, rise: 20, eave: 0.08 }));

  parts.push(box(ctx, { x: 1.92, y: 1.18, w: 0.64, d: 0.82, z: 66, depth: 40, top: C.wallTop, front: C.wallFront, side: C.wallSide }));
  parts.push(frontPosts(ctx, { x: 1.98, y: 2.0, count: 2, spacing: 0.28, zTop: 90, zBottom: 30, color: C.timber }));
  parts.push(roof(ctx, { x: 1.84, y: 1.08, w: 0.84, d: 1.0, z: 102, rise: 20, eave: 0.08 }));

  parts.push(box(ctx, { x: 0.98, y: 0.84, w: 1.14, d: 1.04, z: 108, depth: 58, top: C.wallTop, front: C.wallFront, side: C.wallSideDeep }));
  parts.push(frontPosts(ctx, { x: 1.06, y: 1.88, count: 4, spacing: 0.28, zTop: 142, zBottom: 50, color: C.timber }));
  parts.push(door(ctx, { x: 1.34, y: 1.84, w: 0.42, h: 78 }));
  parts.push(roof(ctx, { x: 0.86, y: 0.74, w: 1.4, d: 1.28, z: 156, rise: 28, eave: 0.12 }));

  parts.push(box(ctx, { x: 1.26, y: 1.02, w: 0.56, d: 0.54, z: 150, depth: 34, top: C.wallTop, front: C.wallFront, side: C.wallSide }));
  parts.push(roof(ctx, { x: 1.18, y: 0.96, w: 0.74, d: 0.7, z: 182, rise: 18, eave: 0.08 }));

  parts.push(goldRidge(ctx, 0.98, 0.74, 1.4, 1.28, 156, 28));
  parts.push(goldRidge(ctx, 1.18, 0.96, 0.74, 0.7, 182, 18));
  parts.push(lantern(ctx, 0.9, 2.24, 18));
  parts.push(lantern(ctx, 2.1, 2.24, 18));
  parts.push(shrub(ctx, 0.36, 2.16, 18, C.blossom, C.blossom));
  parts.push(shrub(ctx, 2.56, 2.02, 18, C.jadeSoft, C.jade));

  return assetPayload(asset, ctx, parts.join('\n'), '木石主殿、三层屋顶、中轴台阶、底部对齐 3x3 footprint。');
}

function renderDiscipleQuarters(asset) {
  const ctx = assetContext(asset);
  const parts = [];
  parts.push(shadow(ctx, 0, 0, 90, 15, 0.16));
  parts.push(box(ctx, { x: 0.16, y: 0.28, w: 1.66, d: 2.36, z: 10, depth: 14, top: C.stoneTop, front: '#c79c6f', side: '#966847' }));
  parts.push(courtyardLines(ctx, 0.34, 0.54, 1.3, 1.86, 12, 3, 5));

  parts.push(box(ctx, { x: 0.34, y: 0.62, w: 1.08, d: 0.72, z: 72, depth: 40, top: C.wallTop, front: C.wallFront, side: C.wallSide }));
  parts.push(frontPosts(ctx, { x: 0.42, y: 1.34, count: 3, spacing: 0.3, zTop: 98, zBottom: 32, color: C.timber }));
  parts.push(roof(ctx, { x: 0.24, y: 0.52, w: 1.28, d: 0.88, z: 104, rise: 20, eave: 0.08 }));

  parts.push(box(ctx, { x: 0.52, y: 1.54, w: 1.08, d: 0.72, z: 66, depth: 40, top: C.wallTop, front: C.wallFront, side: C.wallSide }));
  parts.push(frontPosts(ctx, { x: 0.6, y: 2.26, count: 3, spacing: 0.3, zTop: 92, zBottom: 24, color: C.timber }));
  parts.push(roof(ctx, { x: 0.42, y: 1.44, w: 1.28, d: 0.88, z: 98, rise: 18, eave: 0.08 }));

  parts.push(box(ctx, { x: 1.22, y: 0.9, w: 0.34, d: 1.18, z: 48, depth: 26, top: C.wallTop, front: '#dab98f', side: '#b48d60' }));
  parts.push(leanRoof(ctx, { x: 1.16, y: 0.84, w: 0.46, d: 1.3, z: 70, lift: 10 }));

  parts.push(clothesline(ctx, [iso(ctx, 0.24, 1.78, 28), iso(ctx, 0.72, 1.48, 54), iso(ctx, 1.1, 1.86, 28)], ['#e6bfc7', '#a3c7cf']));
  parts.push(barrel(ctx, 1.52, 2.12, 14));
  parts.push(shrub(ctx, 0.2, 2.1, 10, C.herbLight, C.herb));
  parts.push(lantern(ctx, 1.58, 2.26, 12));

  return assetPayload(asset, ctx, parts.join('\n'), '长向宿舍、生活感连廊、双屋身，对齐 2x3 footprint。');
}

function renderWarehouse(asset) {
  const ctx = assetContext(asset);
  const parts = [];
  parts.push(shadow(ctx, 0, 0, 72, 14, 0.16));
  parts.push(box(ctx, { x: 0.14, y: 0.18, w: 1.72, d: 1.58, z: 8, depth: 14, top: C.stoneTop, front: '#c79d6e', side: '#936746' }));
  parts.push(box(ctx, { x: 0.38, y: 0.44, w: 1.04, d: 0.86, z: 76, depth: 44, top: C.wallTop, front: C.wallFront, side: C.wallSideDeep }));
  parts.push(frontPosts(ctx, { x: 0.48, y: 1.3, count: 3, spacing: 0.24, zTop: 102, zBottom: 34, color: C.timber }));
  parts.push(door(ctx, { x: 0.72, y: 1.26, w: 0.34, h: 54 }));
  parts.push(roof(ctx, { x: 0.28, y: 0.34, w: 1.28, d: 1.02, z: 110, rise: 22, eave: 0.1 }));

  parts.push(box(ctx, { x: 1.12, y: 0.74, w: 0.4, d: 0.46, z: 48, depth: 28, top: C.wallTop, front: '#d8b689', side: '#b58d60' }));
  parts.push(roof(ctx, { x: 1.04, y: 0.68, w: 0.56, d: 0.58, z: 74, rise: 12, eave: 0.05 }));

  parts.push(crate(ctx, 0.26, 1.44, 10, '#9f7346'));
  parts.push(crate(ctx, 0.52, 1.58, 8, '#b0814d'));
  parts.push(crate(ctx, 1.38, 1.28, 12, '#9d6d3f'));
  parts.push(bundle(ctx, 1.58, 1.58, 10));
  parts.push(goldSeal(ctx, 1.04, 1.02, 12));

  return assetPayload(asset, ctx, parts.join('\n'), '厚重储藏建筑，正屋加侧棚，前侧堆货，对齐 2x2 footprint。');
}

function renderHerbGarden(asset) {
  const ctx = assetContext(asset);
  const parts = [];
  parts.push(shadow(ctx, 0, 0, 68, 12, 0.12));
  parts.push(lowBorder(ctx, { x: 0.18, y: 0.24, w: 1.66, d: 1.54, z: 4, depth: 8 }));
  parts.push(planter(ctx, { x: 0.28, y: 0.42, w: 0.54, d: 0.4, z: 10, height: 8, foliage: [C.herbLight, C.herb] }));
  parts.push(planter(ctx, { x: 0.94, y: 0.54, w: 0.56, d: 0.42, z: 10, height: 8, foliage: [C.herbLight, C.herb] }));
  parts.push(planter(ctx, { x: 0.42, y: 1.14, w: 0.54, d: 0.42, z: 10, height: 8, foliage: [C.herbLight, C.herbDark] }));
  parts.push(planter(ctx, { x: 1.08, y: 1.16, w: 0.5, d: 0.38, z: 10, height: 8, foliage: [C.herbLight, C.herb] }));
  parts.push(pergola(ctx, { x: 0.98, y: 0.28, w: 0.62, d: 0.58, z: 42, height: 26 }));
  parts.push(dryingRack(ctx, { x: 0.48, y: 0.92, z: 18 }));
  parts.push(basket(ctx, 1.48, 1.56, 8));
  parts.push(waterJar(ctx, 0.22, 1.46, 10));

  return assetPayload(asset, ctx, parts.join('\n'), '低矮药圃、种植床、棚架与晾架，对齐 2x2 footprint。');
}

function renderGuardTower(asset) {
  const ctx = assetContext(asset);
  const parts = [];
  parts.push(shadow(ctx, 0, 0, 56, 12, 0.16));
  parts.push(box(ctx, { x: 0.08, y: 0.22, w: 0.86, d: 1.56, z: 10, depth: 14, top: C.stoneTop, front: C.stoneFront, side: C.stoneSide }));
  parts.push(box(ctx, { x: 0.28, y: 0.66, w: 0.36, d: 0.4, z: 130, depth: 112, top: '#ead9b8', front: '#c59b69', side: '#996e49' }));
  parts.push(frontPosts(ctx, { x: 0.31, y: 1.06, count: 2, spacing: 0.18, zTop: 156, zBottom: 36, color: C.timber }));
  parts.push(box(ctx, { x: 0.1, y: 0.42, w: 0.72, d: 0.72, z: 152, depth: 20, top: C.wallTop, front: C.wallFront, side: C.wallSide }));
  parts.push(frontPosts(ctx, { x: 0.16, y: 1.14, count: 3, spacing: 0.21, zTop: 176, zBottom: 114, color: C.timber }));
  parts.push(roof(ctx, { x: 0.04, y: 0.36, w: 0.84, d: 0.84, z: 188, rise: 18, eave: 0.06 }));
  parts.push(flag(ctx, { x: 0.58, y: 0.6, z: 210 }));
  parts.push(bell(ctx, 0.46, 0.84, 168));

  return assetPayload(asset, ctx, parts.join('\n'), '塔状警戒建筑，石基木塔、旗帜与小铃，对齐 1x2 footprint。');
}

function renderSpiritWood(asset) {
  const ctx = assetContext(asset);
  const parts = [];
  parts.push(shadow(ctx, 0, 0, 32, 8, 0.12));
  parts.push(lowBorder(ctx, { x: 0.2, y: 0.22, w: 0.6, d: 0.58, z: 4, depth: 6, top: '#dac8a4', front: '#b89265', side: '#8f6844' }));
  parts.push(box(ctx, { x: 0.3, y: 0.32, w: 0.22, d: 0.22, z: 42, depth: 30, top: C.bark, front: C.barkDark, side: '#503726' }));
  parts.push(box(ctx, { x: 0.52, y: 0.5, w: 0.24, d: 0.16, z: 20, depth: 10, top: '#9d7348', front: '#7c5637', side: '#63432b' }));
  parts.push(box(ctx, { x: 0.18, y: 0.54, w: 0.18, d: 0.14, z: 18, depth: 8, top: '#aa8050', front: '#7f5738', side: '#65442d' }));
  parts.push(leafSprig(ctx, 0.36, 0.28, 60));
  parts.push(auraRing(64, 88, 22, 8, 0.18));

  return assetPayload(asset, ctx, parts.join('\n'), '灵木资源点，树桩与木料堆，对齐 1x1 footprint。');
}

function renderSpiritStone(asset) {
  const ctx = assetContext(asset);
  const parts = [];
  parts.push(shadow(ctx, 0, 0, 30, 8, 0.12));
  parts.push(lowBorder(ctx, { x: 0.22, y: 0.24, w: 0.56, d: 0.54, z: 4, depth: 6, top: '#d2d4db', front: '#8f96a5', side: '#646b7a' }));
  parts.push(crystal(ctx, { baseX: 64, baseY: 74, height: 34, width: 18, fill: '#9cb3d8', side: '#7185ad', stroke: '#586987' }));
  parts.push(crystal(ctx, { baseX: 48, baseY: 88, height: 26, width: 14, fill: '#c2cedf', side: '#8996ab', stroke: '#667284' }));
  parts.push(crystal(ctx, { baseX: 84, baseY: 92, height: 22, width: 12, fill: '#a8c3cf', side: '#708e9c', stroke: '#56727f' }));
  parts.push(rock(ctx, 46, 96, 12));
  parts.push(rock(ctx, 90, 100, 10));
  parts.push(auraRing(64, 92, 18, 7, 0.14));

  return assetPayload(asset, ctx, parts.join('\n'), '灵石资源点，石簇与矿脉裂面，对齐 1x1 footprint。');
}

function renderHerb(asset) {
  const ctx = assetContext(asset);
  const parts = [];
  parts.push(shadow(ctx, 0, 0, 28, 7, 0.1));
  parts.push(lowBorder(ctx, { x: 0.2, y: 0.22, w: 0.58, d: 0.52, z: 4, depth: 5, top: '#d5caa9', front: '#b69467', side: '#8b6543' }));
  parts.push(herbTuft(ctx, 52, 74, 1));
  parts.push(herbTuft(ctx, 72, 70, 1.1));
  parts.push(herbTuft(ctx, 88, 82, 0.9));
  parts.push(basket(ctx, 44, 92, 6));

  return assetPayload(asset, ctx, parts.join('\n'), '药草资源点，草药丛与采药篮，对齐 1x1 footprint。');
}

function assetPayload(asset, ctx, body, description) {
  return {
    defs: baseDefs(asset.id, asset.width, asset.height),
    body,
    description,
    width: asset.width,
    height: asset.height,
  };
}

function assetContext(asset) {
  return {
    ox: asset.rows * HALF_W,
    oy: asset.height - asset.footprintHeight,
    id: asset.id,
  };
}

function baseDefs(id, width, height) {
  return [
    `<filter id="${id}-drop" x="0" y="0" width="${width}" height="${height}" filterUnits="userSpaceOnUse" color-interpolation-filters="sRGB">`,
    `  <feOffset dy="10"/>`,
    `  <feGaussianBlur stdDeviation="8"/>`,
    `  <feColorMatrix type="matrix" values="0 0 0 0 0.211765 0 0 0 0 0.152941 0 0 0 0 0.105882 0 0 0 0.18 0"/>`,
    `  <feBlend in2="SourceGraphic" result="shape"/>`,
    `</filter>`,
    `<filter id="${id}-mist" x="0" y="0" width="${width}" height="${height}" filterUnits="userSpaceOnUse" color-interpolation-filters="sRGB">`,
    `  <feGaussianBlur stdDeviation="6"/>`,
    `</filter>`,
  ].join('\n');
}

function iso(ctx, x, y, z = 0) {
  return [ctx.ox + (x - y) * HALF_W, ctx.oy + (x + y) * HALF_H - z];
}

function round(value, digits = 2) {
  return Number(value.toFixed(digits));
}

function attrs(map = {}) {
  const entries = Object.entries(map).filter(([, value]) => value !== undefined && value !== null);
  if (!entries.length) {
    return '';
  }

  return ' ' + entries.map(([key, value]) => `${key}="${value}"`).join(' ');
}

function points(pointsList) {
  return pointsList.map(([x, y]) => `${round(x)},${round(y)}`).join(' ');
}

function polygon(pointsList, map = {}) {
  return `<polygon points="${points(pointsList)}"${attrs(map)}/>`;
}

function ellipse(cx, cy, rx, ry, map = {}) {
  return `<ellipse cx="${round(cx)}" cy="${round(cy)}" rx="${round(rx)}" ry="${round(ry)}"${attrs(map)}/>`;
}

function line(a, b, map = {}) {
  return `<line x1="${round(a[0])}" y1="${round(a[1])}" x2="${round(b[0])}" y2="${round(b[1])}"${attrs(map)}/>`;
}

function path(d, map = {}) {
  return `<path d="${d}"${attrs(map)}/>`;
}

function shadow(ctx, dx, dy, rx, ry, opacity) {
  return ellipse(ctx.ox + dx, ctx.oy + 96 + dy, rx, ry, { fill: C.shadow, opacity: round(opacity, 3), filter: `url(#${ctx.id}-mist)` });
}

function mist(ctx, dx, dy, rx, ry, opacity) {
  return ellipse(ctx.ox + dx, ctx.oy + 54 + dy, rx, ry, { fill: '#fff8ef', opacity: round(opacity, 3), filter: `url(#${ctx.id}-mist)` });
}

function box(ctx, options) {
  const { x, y, w, d, z, depth, top, front, side, stroke = C.ink, sw = 1.5 } = options;
  const nw = iso(ctx, x, y, z);
  const ne = iso(ctx, x + w, y, z);
  const se = iso(ctx, x + w, y + d, z);
  const swp = iso(ctx, x, y + d, z);
  const nwd = iso(ctx, x, y, z - depth);
  const ned = iso(ctx, x + w, y, z - depth);
  const sed = iso(ctx, x + w, y + d, z - depth);
  const swd = iso(ctx, x, y + d, z - depth);
  return [
    polygon([swp, se, sed, swd], { fill: front, stroke, 'stroke-opacity': 0.28, 'stroke-width': sw }),
    polygon([ne, se, sed, ned], { fill: side, stroke, 'stroke-opacity': 0.3, 'stroke-width': sw }),
    polygon([nw, ne, se, swp], { fill: top, stroke, 'stroke-opacity': 0.36, 'stroke-width': sw }),
  ].join('\n');
}

function roof(ctx, options) {
  const { x, y, w, d, z, rise, eave = 0.08 } = options;
  const nw = iso(ctx, x - eave, y - eave, z);
  const ne = iso(ctx, x + w + eave, y - eave, z);
  const se = iso(ctx, x + w + eave, y + d + eave, z);
  const swp = iso(ctx, x - eave, y + d + eave, z);
  const ridgeW = iso(ctx, x + w * 0.16, y + d * 0.54, z + rise);
  const ridgeE = iso(ctx, x + w * 0.84, y + d * 0.54, z + rise);

  const ribs = [];
  for (let i = 1; i <= 4; i += 1) {
    const t = i / 5;
    ribs.push(
      line(lerpPoint(swp, ridgeW, t), lerpPoint(se, ridgeE, t), {
        stroke: '#26373d',
        'stroke-opacity': 0.24,
        'stroke-width': 1,
      }),
    );
  }

  return [
    polygon([nw, ne, ridgeE, ridgeW], { fill: C.roofMid, stroke: C.ink, 'stroke-opacity': 0.42, 'stroke-width': 1.6 }),
    polygon([swp, se, ridgeE, ridgeW], { fill: C.roofFront, stroke: C.ink, 'stroke-opacity': 0.42, 'stroke-width': 1.6 }),
    polygon([ne, se, ridgeE], { fill: C.roofSide, stroke: C.ink, 'stroke-opacity': 0.36, 'stroke-width': 1.4 }),
    polygon([nw, swp, ridgeW], { fill: C.roofDeep, stroke: C.ink, 'stroke-opacity': 0.28, 'stroke-width': 1.2 }),
    ...ribs,
  ].join('\n');
}

function leanRoof(ctx, options) {
  const { x, y, w, d, z, lift } = options;
  const a = iso(ctx, x, y, z + lift);
  const b = iso(ctx, x + w, y, z + lift);
  const c = iso(ctx, x + w, y + d, z);
  const dPoint = iso(ctx, x, y + d, z);
  return polygon([a, b, c, dPoint], { fill: C.roofFront, stroke: C.ink, 'stroke-opacity': 0.32, 'stroke-width': 1.3 });
}

function stairs(ctx, options) {
  const { x, y, w, d, topZ, steps } = options;
  const pieces = [];
  const stepDepth = d / steps;
  for (let i = 0; i < steps; i += 1) {
    pieces.push(
      box(ctx, {
        x,
        y: y + i * stepDepth,
        w,
        d: stepDepth + 0.02,
        z: topZ - i * 5,
        depth: 5,
        top: '#f7ecd7',
        front: '#d8b583',
        side: '#ad8152',
        sw: 1.1,
      }),
    );
  }
  return pieces.join('\n');
}

function walkway(ctx, options) {
  const { x, y, w, d, z, fill } = options;
  const pts = [iso(ctx, x, y, z), iso(ctx, x + w, y, z), iso(ctx, x + w, y + d, z), iso(ctx, x, y + d, z)];
  return polygon(pts, { fill, opacity: 0.92, stroke: '#a37a49', 'stroke-opacity': 0.26, 'stroke-width': 1.1 });
}

function courtyardLines(ctx, x, y, w, d, z, cols, rows) {
  const lines = [];
  for (let i = 0; i <= cols; i += 1) {
    const a = iso(ctx, x + (w / cols) * i, y, z);
    const b = iso(ctx, x + (w / cols) * i, y + d, z);
    lines.push(line(a, b, { stroke: '#a68256', 'stroke-opacity': 0.2, 'stroke-width': 1 }));
  }
  for (let i = 0; i <= rows; i += 1) {
    const a = iso(ctx, x, y + (d / rows) * i, z);
    const b = iso(ctx, x + w, y + (d / rows) * i, z);
    lines.push(line(a, b, { stroke: '#a68256', 'stroke-opacity': 0.2, 'stroke-width': 1 }));
  }
  return lines.join('\n');
}

function frontPosts(ctx, options) {
  const { x, y, count, spacing, zTop, zBottom, color } = options;
  const lines = [];
  for (let i = 0; i < count; i += 1) {
    const top = iso(ctx, x + i * spacing, y, zTop);
    const bottom = iso(ctx, x + i * spacing, y, zBottom);
    lines.push(line(top, bottom, { stroke: color, 'stroke-width': 6, 'stroke-linecap': 'round' }));
  }
  return lines.join('\n');
}

function door(ctx, options) {
  const { x, y, w, h } = options;
  const topLeft = iso(ctx, x, y, h);
  const topRight = iso(ctx, x + w, y, h);
  const bottomRight = iso(ctx, x + w, y, 0);
  const bottomLeft = iso(ctx, x, y, 0);
  return polygon([topLeft, topRight, bottomRight, bottomLeft], {
    fill: '#59453a',
    stroke: '#7b6148',
    'stroke-opacity': 0.3,
    'stroke-width': 1.2,
  });
}

function lantern(ctx, x, y, z) {
  const base = iso(ctx, x, y, z);
  return [
    line(base, [base[0], base[1] - 16], { stroke: C.timberDeep, 'stroke-width': 2.2 }),
    ellipse(base[0], base[1] - 19, 4.5, 5.5, { fill: C.gold, opacity: 0.95 }),
  ].join('\n');
}

function shrub(ctx, x, y, z, fillA, fillB) {
  const center = iso(ctx, x, y, z);
  return [
    ellipse(center[0] - 6, center[1], 12, 8, { fill: fillA, opacity: 0.94 }),
    ellipse(center[0] + 6, center[1] - 4, 10, 7, { fill: fillB, opacity: 0.88 }),
  ].join('\n');
}

function goldRidge(ctx, x, y, w, d, z, rise) {
  const a = iso(ctx, x + w * 0.16, y + d * 0.54, z + rise);
  const b = iso(ctx, x + w * 0.84, y + d * 0.54, z + rise);
  const finial = iso(ctx, x + w * 0.5, y + d * 0.54, z + rise + 12);
  return [
    line(a, b, { stroke: C.gold, 'stroke-width': 2.4 }),
    polygon([finial, [finial[0] + 6, finial[1] + 10], [finial[0], finial[1] + 18], [finial[0] - 6, finial[1] + 10]], {
      fill: C.gold,
      stroke: C.goldDeep,
      'stroke-opacity': 0.3,
      'stroke-width': 1,
    }),
  ].join('\n');
}

function clothesline(ctx, pointsList, clothColors) {
  const [a, b, c] = pointsList;
  return [
    line(a, b, { stroke: C.timberDeep, 'stroke-width': 2.2 }),
    line(b, c, { stroke: C.timberDeep, 'stroke-width': 2.2 }),
    polygon([[b[0] - 16, b[1] + 10], [b[0] - 5, b[1] + 4], [b[0] + 2, b[1] + 12], [b[0] - 10, b[1] + 18]], {
      fill: clothColors[0],
      opacity: 0.94,
    }),
    polygon([[c[0] - 10, c[1] + 6], [c[0] + 4, c[1] - 1], [c[0] + 10, c[1] + 8], [c[0] - 4, c[1] + 15]], {
      fill: clothColors[1],
      opacity: 0.9,
    }),
  ].join('\n');
}

function barrel(ctx, x, y, z) {
  const center = iso(ctx, x, y, z);
  return [
    ellipse(center[0], center[1] - 4, 7, 4, { fill: '#b8874a' }),
    path(`M ${round(center[0] - 7)} ${round(center[1] - 4)} L ${round(center[0] - 7)} ${round(center[1] + 10)} Q ${round(center[0])} ${round(center[1] + 16)} ${round(center[0] + 7)} ${round(center[1] + 10)} L ${round(center[0] + 7)} ${round(center[1] - 4)}`, {
      fill: '#9a6c3e',
      stroke: '#6f4b2e',
      'stroke-opacity': 0.3,
      'stroke-width': 1.2,
    }),
  ].join('\n');
}

function crate(ctx, x, y, z, fill) {
  return box(ctx, { x, y, w: 0.22, d: 0.22, z: z + 6, depth: 8, top: fill, front: darken(fill, -12), side: darken(fill, -24), sw: 1 });
}

function bundle(ctx, x, y, z) {
  const center = iso(ctx, x, y, z);
  return ellipse(center[0], center[1], 12, 7, { fill: '#d9c39a', stroke: '#9b7a52', 'stroke-opacity': 0.26, 'stroke-width': 1.2 });
}

function goldSeal(ctx, x, y, z) {
  const c = iso(ctx, x, y, z);
  return ellipse(c[0], c[1], 5.2, 5.2, { fill: C.gold, opacity: 0.96 });
}

function lowBorder(ctx, options) {
  const { x, y, w, d, z, depth, top = C.stoneTop, front = C.stoneFront, side = C.stoneSide } = options;
  return box(ctx, { x, y, w, d, z, depth, top, front, side, sw: 1.1 });
}

function planter(ctx, options) {
  const { x, y, w, d, z, height, foliage } = options;
  const body = box(ctx, { x, y, w, d, z, depth: height, top: '#8f6a43', front: '#6f5033', side: '#5b412b', sw: 1 });
  const center = iso(ctx, x + w * 0.52, y + d * 0.5, z + height + 10);
  const leaves = [
    ellipse(center[0] - 10, center[1] + 4, 12, 8, { fill: foliage[0], opacity: 0.94 }),
    ellipse(center[0] + 8, center[1], 14, 9, { fill: foliage[1], opacity: 0.94 }),
    ellipse(center[0], center[1] - 10, 10, 7, { fill: foliage[1], opacity: 0.88 }),
  ];
  return [body, ...leaves].join('\n');
}

function pergola(ctx, options) {
  const { x, y, w, d, z, height } = options;
  const parts = [];
  const posts = [
    [x, y + d, z + height],
    [x + w, y + d, z + height],
    [x, y, z + height],
    [x + w, y, z + height],
  ];
  const bottoms = [
    [x, y + d, z],
    [x + w, y + d, z],
    [x, y, z],
    [x + w, y, z],
  ];
  for (let i = 0; i < 4; i += 1) {
    parts.push(line(iso(ctx, ...posts[i]), iso(ctx, ...bottoms[i]), { stroke: C.timber, 'stroke-width': 4, 'stroke-linecap': 'round' }));
  }
  parts.push(leanRoof(ctx, { x: x - 0.02, y: y - 0.02, w: w + 0.04, d: d + 0.04, z: z + height + 4, lift: 8 }));
  return parts.join('\n');
}

function dryingRack(ctx, options) {
  const { x, y, z } = options;
  const leftTop = iso(ctx, x, y, z + 18);
  const rightTop = iso(ctx, x + 0.34, y, z + 18);
  const leftBottom = iso(ctx, x, y, z);
  const rightBottom = iso(ctx, x + 0.34, y, z);
  const beam = line(leftTop, rightTop, { stroke: C.timberDeep, 'stroke-width': 3 });
  const legs = [
    line(leftTop, leftBottom, { stroke: C.timberDeep, 'stroke-width': 2.2 }),
    line(rightTop, rightBottom, { stroke: C.timberDeep, 'stroke-width': 2.2 }),
  ];
  const cloth = polygon(
    [
      [leftTop[0] + 3, leftTop[1] + 5],
      [rightTop[0] - 2, rightTop[1] + 3],
      [rightTop[0] + 4, rightTop[1] + 12],
      [leftTop[0] + 8, leftTop[1] + 14],
    ],
    { fill: '#d7c497', opacity: 0.88 },
  );
  return [beam, ...legs, cloth].join('\n');
}

function basket(ctx, x, y, z) {
  const c = iso(ctx, x, y, z);
  return [
    ellipse(c[0], c[1], 10, 6, { fill: C.basket }),
    line([c[0] - 6, c[1] - 2], [c[0] + 6, c[1] - 2], { stroke: '#8a6332', 'stroke-width': 1.4 }),
  ].join('\n');
}

function waterJar(ctx, x, y, z) {
  const c = iso(ctx, x, y, z);
  return [
    path(`M ${round(c[0] - 6)} ${round(c[1] + 4)} Q ${round(c[0] - 10)} ${round(c[1] - 8)} ${round(c[0])} ${round(c[1] - 12)} Q ${round(c[0] + 10)} ${round(c[1] - 8)} ${round(c[0] + 6)} ${round(c[1] + 4)} Q ${round(c[0])} ${round(c[1] + 10)} ${round(c[0] - 6)} ${round(c[1] + 4)} Z`, {
      fill: C.water,
      stroke: '#5f8594',
      'stroke-opacity': 0.34,
      'stroke-width': 1.2,
    }),
  ].join('\n');
}

function flag(ctx, options) {
  const { x, y, z } = options;
  const poleTop = iso(ctx, x, y, z);
  const poleBottom = iso(ctx, x, y, z - 44);
  return [
    line(poleTop, poleBottom, { stroke: C.timberDeep, 'stroke-width': 2.4 }),
    polygon(
      [
        [poleTop[0], poleTop[1] + 6],
        [poleTop[0] + 22, poleTop[1] + 12],
        [poleTop[0] + 10, poleTop[1] + 24],
        [poleTop[0], poleTop[1] + 18],
      ],
      { fill: '#a45a55', opacity: 0.92 },
    ),
  ].join('\n');
}

function bell(ctx, x, y, z) {
  const c = iso(ctx, x, y, z);
  return [
    line([c[0], c[1] - 12], [c[0], c[1]], { stroke: C.timberDeep, 'stroke-width': 1.4 }),
    ellipse(c[0], c[1] + 2, 4.5, 5.5, { fill: C.gold, opacity: 0.9 }),
  ].join('\n');
}

function leafSprig(ctx, x, y, z) {
  const start = iso(ctx, x, y, z);
  const end = [start[0] + 4, start[1] - 26];
  return [
    line(start, end, { stroke: '#64864d', 'stroke-width': 2.2 }),
    ellipse(end[0] - 6, end[1] + 6, 8, 5, { fill: C.herbLight, opacity: 0.94 }),
    ellipse(end[0] + 4, end[1], 8, 5, { fill: C.herb, opacity: 0.94 }),
  ].join('\n');
}

function auraRing(cx, cy, rx, ry, opacity) {
  return ellipse(cx, cy, rx, ry, { fill: C.jadeSoft, opacity: round(opacity, 3) });
}

function crystal(ctx, options) {
  const { baseX, baseY, height, width, fill, side, stroke } = options;
  const top = [baseX, baseY - height];
  const left = [baseX - width, baseY];
  const right = [baseX + width, baseY - 4];
  const lower = [baseX, baseY + 14];
  return [
    polygon([top, right, lower, [baseX + 2, baseY + 4]], { fill: side, stroke, 'stroke-opacity': 0.34, 'stroke-width': 1.2 }),
    polygon([top, left, [baseX - 2, baseY + 6], lower], { fill, stroke, 'stroke-opacity': 0.34, 'stroke-width': 1.2 }),
  ].join('\n');
}

function rock(ctx, x, y, scale) {
  return polygon(
    [
      [x - scale, y + scale * 0.5],
      [x - scale * 0.2, y - scale],
      [x + scale, y - scale * 0.4],
      [x + scale * 1.2, y + scale * 0.8],
      [x - scale * 0.1, y + scale * 1.3],
    ],
    { fill: C.stoneRockDark, stroke: '#525867', 'stroke-opacity': 0.34, 'stroke-width': 1 },
  );
}

function herbTuft(ctx, x, y, scale) {
  const s = scale;
  return [
    path(`M ${round(x)} ${round(y)} Q ${round(x - 10 * s)} ${round(y - 20 * s)} ${round(x - 5 * s)} ${round(y - 30 * s)}`, { stroke: C.herbDark, 'stroke-width': 2.2, fill: 'none' }),
    path(`M ${round(x)} ${round(y)} Q ${round(x + 2 * s)} ${round(y - 24 * s)} ${round(x + 4 * s)} ${round(y - 34 * s)}`, { stroke: C.herbDark, 'stroke-width': 2.4, fill: 'none' }),
    path(`M ${round(x)} ${round(y)} Q ${round(x + 12 * s)} ${round(y - 18 * s)} ${round(x + 14 * s)} ${round(y - 28 * s)}`, { stroke: C.herbDark, 'stroke-width': 2, fill: 'none' }),
    ellipse(x - 4 * s, y - 22 * s, 8 * s, 5 * s, { fill: C.herbLight, opacity: 0.9 }),
    ellipse(x + 8 * s, y - 16 * s, 8 * s, 5 * s, { fill: C.herb, opacity: 0.92 }),
  ].join('\n');
}

function darken(hex, amount) {
  const value = hex.replace('#', '');
  const channels = value.match(/.{1,2}/g).map((part) => parseInt(part, 16));
  const next = channels.map((channel) => Math.max(0, Math.min(255, channel + amount)));
  return `#${next.map((channel) => channel.toString(16).padStart(2, '0')).join('')}`;
}

function lerpPoint(a, b, t) {
  return [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t];
}

function indent(input, spaces) {
  const prefix = ' '.repeat(spaces);
  return input
    .split('\n')
    .map((line) => (line ? `${prefix}${line}` : line))
    .join('\n');
}
