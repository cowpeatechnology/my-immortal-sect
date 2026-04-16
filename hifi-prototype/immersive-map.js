const immersiveCanvas = document.getElementById("immersiveMapCanvas");

if (immersiveCanvas) {
  const ctx = immersiveCanvas.getContext("2d");
  ctx.lineCap = "round";
  ctx.lineJoin = "round";

  const palette = {
    paperLight: "#f6f0e2",
    paperDark: "#eadfc6",
    ink: "#3d2f23",
    inkSoft: "rgba(64, 50, 37, 0.42)",
    cliff: "#b28e63",
    cliffDark: "#88694a",
    stoneTop: "#f1ead9",
    stoneLeft: "#d8c4a2",
    stoneRight: "#bea07a",
    stoneShade: "#907258",
    jadeRoof: "#445d63",
    jadeRoofLight: "#6c8890",
    jadeRoofDark: "#2f4047",
    jadeRoofMint: "#7ea4a5",
    timber: "#8f5f49",
    timberDark: "#6d4636",
    lacquer: "#b86d56",
    gold: "#d2af72",
    goldDeep: "#b78449",
    canal: "#a5c7d5",
    canalDeep: "#7199a8",
    grass: "#92a572",
    grassDeep: "#65784f",
    moss: "#7c8d60",
    blossom: "#edc6c7",
    blossomDeep: "#d9a4a8",
    pine: "#4d654a",
    pineDark: "#324632",
    blueprint: "#4fc6d9",
    blueprintGlow: "rgba(109, 220, 239, 0.28)",
    taskRed: "#9f4a38",
    white: "rgba(255, 252, 246, 0.9)",
  };

  let lastWidth = 0;
  let lastHeight = 0;

  function mulberry32(seed) {
    return function random() {
      let t = (seed += 0x6d2b79f5);
      t = Math.imul(t ^ (t >>> 15), t | 1);
      t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
      return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
  }

  function iso(originX, originY, tileW, tileH, x, y, z = 0) {
    return {
      x: originX + (x - y) * tileW * 0.5,
      y: originY + (x + y) * tileH * 0.5 - z,
    };
  }

  function pointLerp(a, b, t) {
    return {
      x: a.x + (b.x - a.x) * t,
      y: a.y + (b.y - a.y) * t,
    };
  }

  function offsetPoint(point, dx, dy) {
    return { x: point.x + dx, y: point.y + dy };
  }

  function pathPolygon(points, options = {}) {
    if (!points.length) {
      return;
    }

    ctx.save();
    ctx.beginPath();
    ctx.moveTo(points[0].x, points[0].y);
    for (let i = 1; i < points.length; i += 1) {
      ctx.lineTo(points[i].x, points[i].y);
    }

    if (options.close !== false) {
      ctx.closePath();
    }

    if (options.shadowColor) {
      ctx.shadowColor = options.shadowColor;
      ctx.shadowBlur = options.shadowBlur || 14;
      ctx.shadowOffsetY = options.shadowOffsetY || 8;
    }

    if (options.fill) {
      ctx.fillStyle = options.fill;
      ctx.fill();
    }

    if (options.stroke) {
      if (options.lineDash) {
        ctx.setLineDash(options.lineDash);
      }
      ctx.lineWidth = options.lineWidth || 1.5;
      ctx.strokeStyle = options.stroke;
      ctx.stroke();
    }

    ctx.restore();
  }

  function polygonBounds(points) {
    return points.reduce(
      (bounds, point) => ({
        minX: Math.min(bounds.minX, point.x),
        minY: Math.min(bounds.minY, point.y),
        maxX: Math.max(bounds.maxX, point.x),
        maxY: Math.max(bounds.maxY, point.y),
      }),
      { minX: Infinity, minY: Infinity, maxX: -Infinity, maxY: -Infinity },
    );
  }

  function clipPolygon(points, draw) {
    ctx.save();
    ctx.beginPath();
    ctx.moveTo(points[0].x, points[0].y);
    for (let i = 1; i < points.length; i += 1) {
      ctx.lineTo(points[i].x, points[i].y);
    }
    ctx.closePath();
    ctx.clip();
    draw();
    ctx.restore();
  }

  function watercolorTexture(points, color, density, random, lengthScale = 16) {
    const bounds = polygonBounds(points);
    clipPolygon(points, () => {
      ctx.strokeStyle = color;
      for (let i = 0; i < density; i += 1) {
        const x = bounds.minX + random() * (bounds.maxX - bounds.minX);
        const y = bounds.minY + random() * (bounds.maxY - bounds.minY);
        const length = 4 + random() * lengthScale;
        const angle = -0.8 + random() * 1.6;
        ctx.lineWidth = 0.5 + random() * 1.2;
        ctx.beginPath();
        ctx.moveTo(x, y);
        ctx.lineTo(x + Math.cos(angle) * length, y + Math.sin(angle) * length);
        ctx.stroke();
      }
    });
  }

  function ellipse(x, y, rx, ry, fill, alpha = 1) {
    ctx.save();
    ctx.globalAlpha = alpha;
    ctx.beginPath();
    ctx.ellipse(x, y, rx, ry, 0, 0, Math.PI * 2);
    ctx.fillStyle = fill;
    ctx.fill();
    ctx.restore();
  }

  function drawSoftShadow(x, y, rx, ry, alpha = 0.16) {
    ctx.save();
    ctx.filter = "blur(10px)";
    ellipse(x, y, rx, ry, `rgba(54, 38, 26, ${alpha})`);
    ctx.restore();
  }

  function drawPaperBackground(width, height, random) {
    const paper = ctx.createLinearGradient(0, 0, 0, height);
    paper.addColorStop(0, "#f2ebdb");
    paper.addColorStop(0.56, "#eadfcb");
    paper.addColorStop(1, "#ddc6a3");
    ctx.fillStyle = paper;
    ctx.fillRect(0, 0, width, height);

    for (let i = 0; i < 90; i += 1) {
      const x = random() * width;
      const y = random() * height;
      const radius = 20 + random() * 60;
      const alpha = 0.02 + random() * 0.03;
      ellipse(x, y, radius, radius * (0.35 + random() * 0.35), `rgba(112, 88, 53, ${alpha})`);
    }
  }

  function drawMountainLayer(points, fillA, fillB, ridgeColor, random, blur = 0) {
    ctx.save();
    if (blur) {
      ctx.filter = `blur(${blur}px)`;
    }
    const bounds = polygonBounds(points);
    const grad = ctx.createLinearGradient(bounds.minX, bounds.minY, bounds.maxX, bounds.maxY);
    grad.addColorStop(0, fillA);
    grad.addColorStop(1, fillB);
    pathPolygon(points, { fill: grad });
    watercolorTexture(points, "rgba(76, 95, 82, 0.1)", 80, random, 30);
    ctx.restore();

    ctx.save();
    ctx.strokeStyle = ridgeColor;
    ctx.lineWidth = 3;
    ctx.globalAlpha = 0.35;
    ctx.beginPath();
    ctx.moveTo(points[0].x, points[0].y);
    for (let i = 1; i < points.length - 1; i += 1) {
      ctx.lineTo(points[i].x, points[i].y);
    }
    ctx.stroke();
    ctx.restore();
  }

  function drawBackdrop(width, height, random) {
    drawMountainLayer(
      [
        { x: -50, y: height * 0.28 },
        { x: width * 0.1, y: height * 0.18 },
        { x: width * 0.24, y: height * 0.24 },
        { x: width * 0.36, y: height * 0.14 },
        { x: width * 0.52, y: height * 0.22 },
        { x: width * 0.68, y: height * 0.15 },
        { x: width * 0.84, y: height * 0.24 },
        { x: width + 40, y: height * 0.2 },
        { x: width + 40, y: height * 0.44 },
        { x: -50, y: height * 0.46 },
      ],
      "rgba(158, 174, 160, 0.8)",
      "rgba(118, 138, 126, 0.64)",
      "rgba(61, 76, 67, 0.34)",
      random,
      5,
    );

    drawMountainLayer(
      [
        { x: -60, y: height * 0.39 },
        { x: width * 0.12, y: height * 0.3 },
        { x: width * 0.3, y: height * 0.38 },
        { x: width * 0.42, y: height * 0.31 },
        { x: width * 0.62, y: height * 0.4 },
        { x: width * 0.78, y: height * 0.32 },
        { x: width + 40, y: height * 0.4 },
        { x: width + 40, y: height * 0.54 },
        { x: -60, y: height * 0.56 },
      ],
      "rgba(127, 147, 133, 0.72)",
      "rgba(95, 117, 106, 0.6)",
      "rgba(58, 71, 65, 0.34)",
      random,
      2,
    );

    const pagoda = [
      { x: width * 0.62, y: height * 0.27 },
      { x: width * 0.635, y: height * 0.24 },
      { x: width * 0.65, y: height * 0.27 },
      { x: width * 0.645, y: height * 0.33 },
      { x: width * 0.625, y: height * 0.33 },
    ];
    pathPolygon(pagoda, { fill: "rgba(68, 76, 70, 0.24)" });

    ctx.save();
    ctx.filter = "blur(18px)";
    ellipse(width * 0.24, height * 0.35, width * 0.16, height * 0.045, "rgba(255, 255, 255, 0.54)", 0.56);
    ellipse(width * 0.66, height * 0.34, width * 0.2, height * 0.05, "rgba(252, 247, 236, 0.54)", 0.5);
    ellipse(width * 0.5, height * 0.56, width * 0.28, height * 0.045, "rgba(255, 248, 236, 0.3)", 0.24);
    ctx.restore();
  }

  function drawWaterways(width, height, originX, originY, tileW, tileH, random) {
    const canalOne = [
      iso(originX, originY, tileW, tileH, -0.8, 4.9, 0),
      iso(originX, originY, tileW, tileH, 1.3, 4.9, 0),
      iso(originX, originY, tileW, tileH, 3.5, 7.0, 0),
      iso(originX, originY, tileW, tileH, 1.4, 7.0, 0),
    ];
    const canalTwo = [
      iso(originX, originY, tileW, tileH, 3.5, 7.0, 0),
      iso(originX, originY, tileW, tileH, 5.7, 7.0, 0),
      iso(originX, originY, tileW, tileH, 7.9, 9.1, 0),
      iso(originX, originY, tileW, tileH, 5.8, 9.1, 0),
    ];

    const canalBounds = polygonBounds(canalOne.concat(canalTwo));
    const grad = ctx.createLinearGradient(canalBounds.minX, canalBounds.minY, canalBounds.maxX, canalBounds.maxY);
    grad.addColorStop(0, palette.canal);
    grad.addColorStop(1, palette.canalDeep);
    pathPolygon(canalOne, { fill: grad, shadowColor: "rgba(89, 115, 127, 0.16)", shadowBlur: 20, shadowOffsetY: 10 });
    pathPolygon(canalTwo, { fill: grad });
    watercolorTexture(canalOne, "rgba(255, 255, 255, 0.12)", 32, random, 22);
    watercolorTexture(canalTwo, "rgba(255, 255, 255, 0.12)", 32, random, 22);

    const bankColor = "rgba(255, 248, 236, 0.55)";
    pathPolygon(
      canalOne.map((point, index) => offsetPoint(point, index < 2 ? -6 : 6, index < 2 ? -3 : 3)),
      { stroke: bankColor, lineWidth: 2.2, close: false },
    );
    pathPolygon(
      canalTwo.map((point, index) => offsetPoint(point, index < 2 ? -6 : 6, index < 2 ? -3 : 3)),
      { stroke: bankColor, lineWidth: 2.2, close: false },
    );

    const bridgeA = iso(originX, originY, tileW, tileH, 1.36, 5.5, 12);
    const bridgeB = iso(originX, originY, tileW, tileH, 2.46, 6.6, 12);
    drawArchedBridge(bridgeA, bridgeB, 26);

    const bridgeC = iso(originX, originY, tileW, tileH, 5.08, 7.12, 12);
    const bridgeD = iso(originX, originY, tileW, tileH, 6.12, 8.14, 12);
    drawArchedBridge(bridgeC, bridgeD, 22);

    const pondCenter = iso(originX, originY, tileW, tileH, 0.5, 7.46, 0);
    ellipse(pondCenter.x, pondCenter.y, tileW * 0.22, tileH * 0.24, "#96c5d0", 0.9);
  }

  function drawArchedBridge(start, end, width) {
    const mid = pointLerp(start, end, 0.5);
    const normal = { x: -(end.y - start.y), y: end.x - start.x };
    const length = Math.hypot(normal.x, normal.y) || 1;
    const unit = { x: (normal.x / length) * width * 0.18, y: (normal.y / length) * width * 0.18 };

    const a = offsetPoint(start, unit.x, unit.y);
    const b = offsetPoint(end, unit.x, unit.y);
    const c = offsetPoint(end, -unit.x, -unit.y);
    const d = offsetPoint(start, -unit.x, -unit.y);

    pathPolygon([a, b, c, d], {
      fill: "#efe4cb",
      stroke: "rgba(114, 89, 60, 0.42)",
      lineWidth: 1.4,
      shadowColor: "rgba(82, 66, 47, 0.16)",
      shadowBlur: 14,
      shadowOffsetY: 8,
    });

    ctx.save();
    ctx.strokeStyle = "rgba(111, 89, 65, 0.62)";
    ctx.lineWidth = 1.6;
    ctx.beginPath();
    ctx.moveTo(a.x, a.y);
    ctx.quadraticCurveTo(mid.x, mid.y - width * 0.38, b.x, b.y);
    ctx.stroke();
    ctx.beginPath();
    ctx.moveTo(d.x, d.y);
    ctx.quadraticCurveTo(mid.x, mid.y - width * 0.38, c.x, c.y);
    ctx.stroke();
    ctx.restore();
  }

  function drawBox(originX, originY, tileW, tileH, x, y, w, d, z, depth, colors) {
    const nw = iso(originX, originY, tileW, tileH, x, y, z);
    const ne = iso(originX, originY, tileW, tileH, x + w, y, z);
    const se = iso(originX, originY, tileW, tileH, x + w, y + d, z);
    const sw = iso(originX, originY, tileW, tileH, x, y + d, z);

    const swDown = iso(originX, originY, tileW, tileH, x, y + d, z - depth);
    const seDown = iso(originX, originY, tileW, tileH, x + w, y + d, z - depth);
    const neDown = iso(originX, originY, tileW, tileH, x + w, y, z - depth);
    const nwDown = iso(originX, originY, tileW, tileH, x, y, z - depth);

    const top = [nw, ne, se, sw];
    const left = [sw, se, seDown, swDown];
    const right = [ne, se, seDown, neDown];
    const front = [nw, sw, swDown, nwDown];

    pathPolygon(left, { fill: colors.left, stroke: "rgba(86, 65, 45, 0.12)", lineWidth: 1 });
    pathPolygon(right, { fill: colors.right, stroke: "rgba(86, 65, 45, 0.12)", lineWidth: 1 });
    pathPolygon(front, { fill: colors.front || colors.left, stroke: "rgba(86, 65, 45, 0.12)", lineWidth: 1 });
    pathPolygon(top, {
      fill: colors.top,
      stroke: colors.stroke || "rgba(96, 71, 49, 0.16)",
      lineWidth: 1.2,
      shadowColor: "rgba(65, 49, 34, 0.08)",
      shadowBlur: 8,
      shadowOffsetY: 4,
    });

    return { nw, ne, se, sw, top };
  }

  function drawRoof(originX, originY, tileW, tileH, x, y, w, d, z, rise, eave, colors) {
    const nw = offsetPoint(iso(originX, originY, tileW, tileH, x - eave, y - eave, z), -6, -2);
    const ne = offsetPoint(iso(originX, originY, tileW, tileH, x + w + eave, y - eave, z), 6, -2);
    const se = offsetPoint(iso(originX, originY, tileW, tileH, x + w + eave, y + d + eave, z), 10, 8);
    const sw = offsetPoint(iso(originX, originY, tileW, tileH, x - eave, y + d + eave, z), -10, 8);
    const ridgeW = iso(originX, originY, tileW, tileH, x + w * 0.14, y + d * 0.56, z + rise);
    const ridgeE = iso(originX, originY, tileW, tileH, x + w * 0.86, y + d * 0.56, z + rise);

    const backPlane = [nw, ne, ridgeE, ridgeW];
    const frontPlane = [sw, se, ridgeE, ridgeW];
    const leftPlane = [nw, sw, ridgeW];
    const rightPlane = [ne, se, ridgeE];

    pathPolygon(backPlane, { fill: colors.topBack });
    pathPolygon(frontPlane, {
      fill: colors.topFront,
      shadowColor: "rgba(39, 35, 31, 0.14)",
      shadowBlur: 16,
      shadowOffsetY: 8,
    });
    pathPolygon(leftPlane, { fill: colors.sideLeft });
    pathPolygon(rightPlane, { fill: colors.sideRight });

    clipPolygon(frontPlane, () => {
      ctx.strokeStyle = "rgba(32, 44, 49, 0.22)";
      ctx.lineWidth = 1;
      for (let i = 0; i <= 8; i += 1) {
        const t = i / 8;
        const left = pointLerp(sw, ridgeW, t);
        const right = pointLerp(se, ridgeE, t);
        ctx.beginPath();
        ctx.moveTo(left.x, left.y);
        ctx.lineTo(right.x, right.y);
        ctx.stroke();
      }
    });

    clipPolygon(backPlane, () => {
      ctx.strokeStyle = "rgba(52, 67, 70, 0.16)";
      ctx.lineWidth = 0.9;
      for (let i = 0; i <= 6; i += 1) {
        const t = i / 6;
        const left = pointLerp(nw, ridgeW, t);
        const right = pointLerp(ne, ridgeE, t);
        ctx.beginPath();
        ctx.moveTo(left.x, left.y);
        ctx.lineTo(right.x, right.y);
        ctx.stroke();
      }
    });

    ctx.save();
    ctx.strokeStyle = colors.ridge;
    ctx.lineWidth = 1.5;
    ctx.beginPath();
    ctx.moveTo(ridgeW.x, ridgeW.y);
    ctx.lineTo(ridgeE.x, ridgeE.y);
    ctx.stroke();

    [sw, se].forEach((point, index) => {
      const ridge = index === 0 ? ridgeW : ridgeE;
      ctx.beginPath();
      ctx.moveTo(point.x, point.y);
      ctx.lineTo(ridge.x, ridge.y);
      ctx.stroke();
    });

    [sw, se, ridgeW, ridgeE].forEach((point) => {
      ellipse(point.x, point.y, 2.5, 2, palette.gold, 0.9);
    });
    ctx.restore();
  }

  function drawSouthFacade(originX, originY, tileW, tileH, x, y, w, d, z, depth, columns) {
    ctx.save();
    ctx.strokeStyle = "rgba(96, 64, 47, 0.44)";
    ctx.lineWidth = 1.2;
    for (let i = 1; i < columns; i += 1) {
      const t = i / columns;
      const top = iso(originX, originY, tileW, tileH, x + w * t, y + d, z - 3);
      const bottom = iso(originX, originY, tileW, tileH, x + w * t, y + d, z - depth + 8);
      ctx.beginPath();
      ctx.moveTo(top.x, top.y);
      ctx.lineTo(bottom.x, bottom.y);
      ctx.stroke();
    }
    for (let i = 0; i < columns - 1; i += 1) {
      const left = iso(originX, originY, tileW, tileH, x + (w / columns) * (i + 0.18), y + d, z - depth * 0.42);
      const right = iso(originX, originY, tileW, tileH, x + (w / columns) * (i + 0.82), y + d, z - depth * 0.42);
      const lowerRight = iso(originX, originY, tileW, tileH, x + (w / columns) * (i + 0.82), y + d, z - depth * 0.7);
      const lowerLeft = iso(originX, originY, tileW, tileH, x + (w / columns) * (i + 0.18), y + d, z - depth * 0.7);
      pathPolygon([left, right, lowerRight, lowerLeft], { fill: "rgba(62, 85, 91, 0.28)" });
    }

    const beamTopLeft = iso(originX, originY, tileW, tileH, x, y + d, z - depth * 0.18);
    const beamTopRight = iso(originX, originY, tileW, tileH, x + w, y + d, z - depth * 0.18);
    const beamBottomRight = iso(originX, originY, tileW, tileH, x + w, y + d, z - depth * 0.28);
    const beamBottomLeft = iso(originX, originY, tileW, tileH, x, y + d, z - depth * 0.28);
    pathPolygon([beamTopLeft, beamTopRight, beamBottomRight, beamBottomLeft], { fill: "rgba(147, 92, 71, 0.58)" });
    ctx.restore();
  }

  function drawSouthDoor(originX, originY, tileW, tileH, x, y, w, d, z, depth, startT, endT) {
    const topLeft = iso(originX, originY, tileW, tileH, x + w * startT, y + d, z - depth * 0.18);
    const topRight = iso(originX, originY, tileW, tileH, x + w * endT, y + d, z - depth * 0.18);
    const bottomRight = iso(originX, originY, tileW, tileH, x + w * endT, y + d, z - depth * 0.82);
    const bottomLeft = iso(originX, originY, tileW, tileH, x + w * startT, y + d, z - depth * 0.82);
    pathPolygon([topLeft, topRight, bottomRight, bottomLeft], {
      fill: "rgba(59, 48, 39, 0.6)",
      stroke: "rgba(199, 172, 126, 0.24)",
      lineWidth: 1,
    });
  }

  function drawWallRun(originX, originY, tileW, tileH, x, y, w, d, z) {
    drawBox(originX, originY, tileW, tileH, x, y, w, d, z, 14, {
      top: "#eee4cd",
      left: "#ccb796",
      right: "#b3926a",
    });
    drawRoof(originX, originY, tileW, tileH, x - 0.02, y - 0.02, w + 0.04, d + 0.04, z + 14, 10, 0.04, {
      topBack: "#7a9698",
      topFront: "#536f73",
      sideLeft: "#4d666b",
      sideRight: "#34454a",
      ridge: palette.gold,
    });
  }

  function drawStairs(originX, originY, tileW, tileH, x, y, w, d, topZ, steps) {
    const stepDepth = d / steps;
    for (let i = 0; i < steps; i += 1) {
      const z = topZ - i * 7;
      drawBox(originX, originY, tileW, tileH, x, y + i * stepDepth, w, stepDepth + 0.02, z, 7, {
        top: "#f6ecd4",
        left: "#d7c2a0",
        right: "#bea17a",
      });
    }

    const runner = [
      iso(originX, originY, tileW, tileH, x + w * 0.36, y, topZ + 1),
      iso(originX, originY, tileW, tileH, x + w * 0.64, y, topZ + 1),
      iso(originX, originY, tileW, tileH, x + w * 0.64, y + d, topZ - steps * 7 + 1),
      iso(originX, originY, tileW, tileH, x + w * 0.36, y + d, topZ - steps * 7 + 1),
    ];
    pathPolygon(runner, { fill: "rgba(213, 176, 103, 0.9)" });
  }

  function drawCourtyardGrid(originX, originY, tileW, tileH, x, y, w, d, z) {
    const top = [
      iso(originX, originY, tileW, tileH, x, y, z),
      iso(originX, originY, tileW, tileH, x + w, y, z),
      iso(originX, originY, tileW, tileH, x + w, y + d, z),
      iso(originX, originY, tileW, tileH, x, y + d, z),
    ];
    pathPolygon(top, { fill: "rgba(241, 232, 213, 0.92)" });
    ctx.save();
    ctx.strokeStyle = "rgba(124, 95, 66, 0.16)";
    ctx.lineWidth = 1;
    for (let i = 0; i <= 5; i += 1) {
      const start = iso(originX, originY, tileW, tileH, x + (w / 5) * i, y, z + 0.5);
      const end = iso(originX, originY, tileW, tileH, x + (w / 5) * i, y + d, z + 0.5);
      ctx.beginPath();
      ctx.moveTo(start.x, start.y);
      ctx.lineTo(end.x, end.y);
      ctx.stroke();
    }
    for (let i = 0; i <= 4; i += 1) {
      const start = iso(originX, originY, tileW, tileH, x, y + (d / 4) * i, z + 0.5);
      const end = iso(originX, originY, tileW, tileH, x + w, y + (d / 4) * i, z + 0.5);
      ctx.beginPath();
      ctx.moveTo(start.x, start.y);
      ctx.lineTo(end.x, end.y);
      ctx.stroke();
    }
    ctx.restore();
  }

  function drawHall(originX, originY, tileW, tileH) {
    drawBox(originX, originY, tileW, tileH, 2.4, 1.7, 3.42, 2.88, 132, 54, {
      top: palette.stoneTop,
      left: palette.stoneLeft,
      right: palette.stoneRight,
    });
    drawCourtyardGrid(originX, originY, tileW, tileH, 2.62, 1.96, 2.94, 2.34, 134);
    drawStairs(originX, originY, tileW, tileH, 3.46, 4.16, 1.22, 1.18, 132, 6);

    drawBox(originX, originY, tileW, tileH, 2.96, 2.12, 0.76, 1.06, 178, 44, {
      top: "#f3ead7",
      left: "#ccb89a",
      right: "#b99b75",
    });
    drawBox(originX, originY, tileW, tileH, 4.52, 2.12, 0.76, 1.06, 178, 44, {
      top: "#f3ead7",
      left: "#ccb89a",
      right: "#b99b75",
    });
    drawBox(originX, originY, tileW, tileH, 3.38, 2.04, 1.42, 1.44, 190, 58, {
      top: "#f6eee0",
      left: "#d2bf9f",
      right: "#bf9f78",
    });
    drawSouthFacade(originX, originY, tileW, tileH, 3.38, 2.04, 1.42, 1.44, 190, 58, 4);
    drawSouthFacade(originX, originY, tileW, tileH, 2.96, 2.12, 0.76, 1.06, 178, 44, 3);
    drawSouthFacade(originX, originY, tileW, tileH, 4.52, 2.12, 0.76, 1.06, 178, 44, 3);
    drawSouthDoor(originX, originY, tileW, tileH, 3.38, 2.04, 1.42, 1.44, 190, 58, 0.36, 0.64);

    drawRoof(originX, originY, tileW, tileH, 2.84, 1.9, 1.0, 1.28, 228, 42, 0.18, {
      topBack: palette.jadeRoofMint,
      topFront: palette.jadeRoofLight,
      sideLeft: "#4f6871",
      sideRight: "#34484f",
      ridge: palette.gold,
    });
    drawRoof(originX, originY, tileW, tileH, 4.44, 1.9, 1.0, 1.28, 228, 42, 0.18, {
      topBack: palette.jadeRoofMint,
      topFront: palette.jadeRoofLight,
      sideLeft: "#4f6871",
      sideRight: "#34484f",
      ridge: palette.gold,
    });
    drawRoof(originX, originY, tileW, tileH, 3.18, 1.78, 1.82, 1.78, 252, 52, 0.22, {
      topBack: "#6c8b90",
      topFront: palette.jadeRoof,
      sideLeft: "#4e676f",
      sideRight: palette.jadeRoofDark,
      ridge: palette.gold,
    });
    drawBox(originX, originY, tileW, tileH, 3.7, 2.38, 0.78, 0.7, 236, 30, {
      top: "#f6eee2",
      left: "#d2bf9d",
      right: "#b99872",
    });
    drawRoof(originX, originY, tileW, tileH, 3.56, 2.26, 1.06, 0.98, 266, 34, 0.12, {
      topBack: "#84a6a5",
      topFront: "#58767a",
      sideLeft: "#496165",
      sideRight: "#324548",
      ridge: palette.gold,
    });

    drawWallRun(originX, originY, tileW, tileH, 2.66, 3.72, 0.42, 0.92, 144);
    drawWallRun(originX, originY, tileW, tileH, 5.18, 3.72, 0.42, 0.92, 144);
    drawWallRun(originX, originY, tileW, tileH, 2.88, 1.9, 2.48, 0.26, 144);

    drawLantern(originX, originY, tileW, tileH, 3.14, 4.12, 142);
    drawLantern(originX, originY, tileW, tileH, 5.04, 4.12, 142);
  }

  function drawWarehouse(originX, originY, tileW, tileH) {
    drawBox(originX, originY, tileW, tileH, 6.16, 4.42, 1.86, 1.62, 84, 36, {
      top: palette.stoneTop,
      left: "#d0bb99",
      right: "#b89a70",
    });
    drawCourtyardGrid(originX, originY, tileW, tileH, 6.34, 4.58, 1.5, 1.28, 86);
    drawBox(originX, originY, tileW, tileH, 6.54, 4.8, 1.06, 0.86, 116, 42, {
      top: "#f6ede0",
      left: "#ceb899",
      right: "#ba9b72",
    });
    drawSouthFacade(originX, originY, tileW, tileH, 6.54, 4.8, 1.06, 0.86, 116, 42, 3);
    drawSouthDoor(originX, originY, tileW, tileH, 6.54, 4.8, 1.06, 0.86, 116, 42, 0.34, 0.62);
    drawRoof(originX, originY, tileW, tileH, 6.36, 4.66, 1.42, 1.08, 158, 38, 0.16, {
      topBack: "#6f9092",
      topFront: palette.jadeRoof,
      sideLeft: "#507073",
      sideRight: "#324649",
      ridge: palette.gold,
    });
    drawBox(originX, originY, tileW, tileH, 7.78, 4.66, 0.48, 0.56, 108, 32, {
      top: "#efe4cf",
      left: "#c8b28e",
      right: "#af9066",
    });
    drawRoof(originX, originY, tileW, tileH, 7.68, 4.58, 0.64, 0.7, 140, 20, 0.08, {
      topBack: "#7b9d9d",
      topFront: "#567276",
      sideLeft: "#4e6669",
      sideRight: "#36474b",
      ridge: palette.gold,
    });
    drawPropStack(originX, originY, tileW, tileH, 7.9, 5.92, 74);
  }

  function drawAlchemyBlueprint(originX, originY, tileW, tileH) {
    const top = [
      iso(originX, originY, tileW, tileH, 7.78, 3.1, 76),
      iso(originX, originY, tileW, tileH, 9.04, 3.1, 76),
      iso(originX, originY, tileW, tileH, 9.04, 4.28, 76),
      iso(originX, originY, tileW, tileH, 7.78, 4.28, 76),
    ];
    const glow = ctx.createLinearGradient(top[0].x, top[0].y, top[2].x, top[2].y);
    glow.addColorStop(0, "rgba(104, 221, 239, 0.26)");
    glow.addColorStop(1, "rgba(62, 176, 202, 0.16)");
    pathPolygon(top, {
      fill: glow,
      stroke: palette.blueprint,
      lineWidth: 2.5,
      lineDash: [10, 8],
      shadowColor: "rgba(90, 194, 214, 0.2)",
      shadowBlur: 20,
      shadowOffsetY: 6,
    });

    const frameA = [
      iso(originX, originY, tileW, tileH, 8.04, 3.34, 130),
      iso(originX, originY, tileW, tileH, 8.82, 3.34, 130),
      iso(originX, originY, tileW, tileH, 8.82, 4.02, 130),
      iso(originX, originY, tileW, tileH, 8.04, 4.02, 130),
    ];
    pathPolygon(frameA, {
      stroke: "rgba(94, 213, 233, 0.92)",
      lineWidth: 2,
      lineDash: [8, 6],
      close: true,
    });

    frameA.forEach((point) => {
      const foot = offsetPoint(point, 0, 48);
      pathPolygon([point, foot], { stroke: "rgba(94, 213, 233, 0.72)", lineWidth: 1.6, close: false });
    });
  }

  function drawGateTower(originX, originY, tileW, tileH) {
    drawBox(originX, originY, tileW, tileH, 5.46, 7.28, 1.52, 1.28, 64, 52, {
      top: "#d9c7a5",
      left: "#b89466",
      right: "#9d774f",
    });

    const face = [
      iso(originX, originY, tileW, tileH, 5.94, 7.94, 90),
      iso(originX, originY, tileW, tileH, 6.46, 7.94, 90),
      iso(originX, originY, tileW, tileH, 6.46, 8.56, 40),
      iso(originX, originY, tileW, tileH, 5.94, 8.56, 40),
    ];
    pathPolygon(face, { fill: "rgba(72, 58, 46, 0.18)" });

    drawBox(originX, originY, tileW, tileH, 5.68, 7.48, 1.08, 0.92, 130, 44, {
      top: "#f6ede0",
      left: "#d2bf9b",
      right: "#be9d76",
    });
    drawRoof(originX, originY, tileW, tileH, 5.52, 7.32, 1.38, 1.08, 174, 36, 0.16, {
      topBack: "#6e8a90",
      topFront: palette.jadeRoof,
      sideLeft: "#4f6870",
      sideRight: "#324348",
      ridge: palette.gold,
    });
    drawWallRun(originX, originY, tileW, tileH, 4.86, 7.66, 0.42, 0.84, 92);
    drawWallRun(originX, originY, tileW, tileH, 7.08, 7.66, 0.42, 0.84, 92);
  }

  function drawPavilion(originX, originY, tileW, tileH) {
    drawBox(originX, originY, tileW, tileH, 8.18, 1.92, 0.78, 0.76, 78, 38, {
      top: "#efe4cf",
      left: "#ccb58f",
      right: "#b7986e",
    });
    drawRoof(originX, originY, tileW, tileH, 8.08, 1.82, 0.96, 0.92, 116, 24, 0.12, {
      topBack: "#7ba1a2",
      topFront: "#537074",
      sideLeft: "#4e676e",
      sideRight: "#36474d",
      ridge: palette.gold,
    });
  }

  function drawSecondaryHall(originX, originY, tileW, tileH, x, y, z, scale = 1) {
    const w = 1.02 * scale;
    const d = 0.92 * scale;
    drawBox(originX, originY, tileW, tileH, x, y, w, d, z, 26 * scale, {
      top: "#efe5d2",
      left: "#ccb692",
      right: "#b4956c",
    });
    drawBox(originX, originY, tileW, tileH, x + 0.12 * scale, y + 0.1 * scale, 0.74 * scale, 0.58 * scale, z + 22 * scale, 30 * scale, {
      top: "#f6ede0",
      left: "#cfb899",
      right: "#bb9b72",
    });
    drawSouthFacade(originX, originY, tileW, tileH, x + 0.12 * scale, y + 0.1 * scale, 0.74 * scale, 0.58 * scale, z + 22 * scale, 30 * scale, 3);
    drawSouthDoor(originX, originY, tileW, tileH, x + 0.12 * scale, y + 0.1 * scale, 0.74 * scale, 0.58 * scale, z + 22 * scale, 30 * scale, 0.34, 0.64);
    drawRoof(originX, originY, tileW, tileH, x + 0.02 * scale, y + 0.02 * scale, 0.92 * scale, 0.74 * scale, z + 54 * scale, 24 * scale, 0.12 * scale, {
      topBack: "#78989a",
      topFront: "#516b70",
      sideLeft: "#4b6368",
      sideRight: "#35464b",
      ridge: palette.gold,
    });
  }

  function drawPropStack(originX, originY, tileW, tileH, x, y, z) {
    drawBox(originX, originY, tileW, tileH, x, y, 0.24, 0.24, z, 12, {
      top: "#b88953",
      left: "#896340",
      right: "#6f5034",
    });
    drawBox(originX, originY, tileW, tileH, x + 0.28, y - 0.06, 0.22, 0.22, z + 4, 10, {
      top: "#c79d63",
      left: "#8d6a42",
      right: "#755539",
    });
    drawBox(originX, originY, tileW, tileH, x + 0.12, y + 0.3, 0.34, 0.18, z, 8, {
      top: "#6d7d50",
      left: "#4f603e",
      right: "#415033",
    });
  }

  function drawLantern(originX, originY, tileW, tileH, x, y, z) {
    const post = iso(originX, originY, tileW, tileH, x, y, z);
    ctx.save();
    ctx.strokeStyle = "#694f3c";
    ctx.lineWidth = 2.4;
    ctx.beginPath();
    ctx.moveTo(post.x, post.y);
    ctx.lineTo(post.x, post.y - 22);
    ctx.stroke();
    ellipse(post.x, post.y - 26, 4.5, 5.5, palette.gold, 0.95);
    ctx.restore();
  }

  function drawPine(x, y, scale) {
    ctx.save();
    ctx.strokeStyle = "#5f4835";
    ctx.lineWidth = 2.8 * scale;
    ctx.beginPath();
    ctx.moveTo(x, y);
    ctx.lineTo(x + 2 * scale, y - 20 * scale);
    ctx.stroke();
    ellipse(x - 8 * scale, y - 22 * scale, 18 * scale, 10 * scale, palette.pineDark, 0.96);
    ellipse(x + 4 * scale, y - 32 * scale, 22 * scale, 11 * scale, palette.pine, 0.98);
    ellipse(x - 6 * scale, y - 42 * scale, 16 * scale, 9 * scale, palette.pineDark, 0.96);
    ctx.restore();
  }

  function drawBlossomTree(x, y, scale) {
    ctx.save();
    ctx.strokeStyle = "#755241";
    ctx.lineWidth = 2.4 * scale;
    ctx.beginPath();
    ctx.moveTo(x, y);
    ctx.lineTo(x + 2 * scale, y - 18 * scale);
    ctx.lineTo(x - 6 * scale, y - 28 * scale);
    ctx.stroke();
    [
      [0, -32, 18, 10, palette.blossom],
      [-12, -26, 12, 8, palette.blossomDeep],
      [10, -20, 14, 9, palette.blossom],
      [8, -36, 12, 8, palette.blossomDeep],
    ].forEach(([dx, dy, rx, ry, fill]) => {
      ellipse(x + dx * scale, y + dy * scale, rx * scale, ry * scale, fill, 0.94);
    });
    ctx.restore();
  }

  function drawRock(x, y, scale) {
    pathPolygon(
      [
        { x: x - 10 * scale, y: y + 6 * scale },
        { x: x - 2 * scale, y: y - 8 * scale },
        { x: x + 10 * scale, y: y - 4 * scale },
        { x: x + 14 * scale, y: y + 8 * scale },
        { x: x - 2 * scale, y: y + 12 * scale },
      ],
      { fill: "rgba(107, 97, 88, 0.72)", stroke: "rgba(73, 64, 58, 0.38)", lineWidth: 1.1 },
    );
  }

  function drawDisciple(originX, originY, tileW, tileH, x, y, robe, accent) {
    const point = iso(originX, originY, tileW, tileH, x, y, 12);
    drawSoftShadow(point.x, point.y + 8, 10, 4, 0.12);
    ellipse(point.x, point.y - 2, 4.2, 4.6, "#f5d9b4", 1);
    ellipse(point.x, point.y + 9, 8.8, 12.4, robe, 1);
    ellipse(point.x, point.y + 12, 5.6, 7.4, accent, 0.88);
  }

  function drawForegroundScenery(width, height, originX, originY, tileW, tileH) {
    drawPine(width * 0.11, height * 0.88, 1.12);
    drawPine(width * 0.82, height * 0.92, 1.04);
    drawBlossomTree(width * 0.28, height * 0.56, 1.02);
    drawBlossomTree(width * 0.73, height * 0.58, 0.94);
    drawRock(width * 0.2, height * 0.64, 1);
    drawRock(width * 0.84, height * 0.7, 0.9);

    const garden = drawBox(originX, originY, tileW, tileH, 0.12, 8.08, 1.34, 1.08, 30, 10, {
      top: "#c7d5a6",
      left: "#8aa06f",
      right: "#6e8557",
    });
    watercolorTexture(garden.top, "rgba(77, 111, 58, 0.18)", 38, mulberry32(9), 10);
  }

  function drawTerrainPatches(originX, originY, tileW, tileH) {
    [
      [0.4, 6.3, 1.2, 0.88, 26, palette.grass, palette.grassDeep, "#5d7347"],
      [1.76, 7.38, 1.02, 0.78, 24, "#d5c79d", "#b49769", "#937754"],
      [6.86, 6.42, 0.92, 0.84, 30, palette.grass, palette.grassDeep, "#5e7547"],
      [7.54, 5.46, 0.7, 0.64, 28, "#d4c499", "#b99966", "#95784f"],
    ].forEach(([x, y, w, d, z, top, left, right]) => {
      drawBox(originX, originY, tileW, tileH, x, y, w, d, z, 8, { top, left, right });
    });
  }

  function drawGroundWash(width, height, originX, originY, tileW, tileH, random) {
    const ground = [
      iso(originX, originY, tileW, tileH, -2.0, 2.5, 0),
      iso(originX, originY, tileW, tileH, 8.6, 2.2, 0),
      iso(originX, originY, tileW, tileH, 11.0, 7.8, 0),
      iso(originX, originY, tileW, tileH, 7.2, 11.0, 0),
      iso(originX, originY, tileW, tileH, -1.8, 9.7, 0),
      iso(originX, originY, tileW, tileH, -3.0, 5.8, 0),
    ];
    const grad = ctx.createLinearGradient(width * 0.2, height * 0.44, width * 0.74, height * 0.96);
    grad.addColorStop(0, "rgba(235, 223, 198, 0.98)");
    grad.addColorStop(1, "rgba(214, 189, 145, 0.98)");
    pathPolygon(ground, { fill: grad });
    watercolorTexture(ground, "rgba(143, 114, 73, 0.08)", 220, random, 28);
  }

  function drawScene(width, height, random) {
    const originX = width * 0.42;
    const originY = height * 0.42;
    const tileW = width * 0.13;
    const tileH = tileW * 0.5;

    drawGroundWash(width, height, originX, originY, tileW, tileH, random);
    drawWaterways(width, height, originX, originY, tileW, tileH, random);
    drawTerrainPatches(originX, originY, tileW, tileH);

    drawPavilion(originX, originY, tileW, tileH);
    drawHall(originX, originY, tileW, tileH);
    drawWarehouse(originX, originY, tileW, tileH);
    drawAlchemyBlueprint(originX, originY, tileW, tileH);
    drawSecondaryHall(originX, originY, tileW, tileH, 7.36, 8.26, 36, 1.12);
    drawSecondaryHall(originX, originY, tileW, tileH, 0.28, 8.32, 28, 0.96);
    drawGateTower(originX, originY, tileW, tileH);

    drawDisciple(originX, originY, tileW, tileH, 2.44, 5.58, "#8ba9c9", "#58779c");
    drawDisciple(originX, originY, tileW, tileH, 7.72, 5.82, "#d6b37b", "#9b7549");
    drawDisciple(originX, originY, tileW, tileH, 5.18, 8.5, "#cab9e2", "#8570aa");

    drawForegroundScenery(width, height, originX, originY, tileW, tileH);

    ctx.save();
    ctx.filter = "blur(12px)";
    ellipse(width * 0.34, height * 0.58, width * 0.12, height * 0.02, "rgba(255, 253, 244, 0.34)", 0.36);
    ellipse(width * 0.72, height * 0.54, width * 0.1, height * 0.018, "rgba(255, 249, 239, 0.28)", 0.32);
    ctx.restore();
  }

  function drawFrame(width, height) {
    const random = mulberry32(width * 31 + height * 17 + 7);
    ctx.clearRect(0, 0, width, height);
    drawPaperBackground(width, height, random);
    drawBackdrop(width, height, random);
    drawScene(width, height, random);

    const vignette = ctx.createLinearGradient(0, 0, 0, height);
    vignette.addColorStop(0, "rgba(255,255,255,0)");
    vignette.addColorStop(0.8, "rgba(255,255,255,0)");
    vignette.addColorStop(1, "rgba(83, 61, 38, 0.1)");
    ctx.fillStyle = vignette;
    ctx.fillRect(0, 0, width, height);
  }

  function resizeAndDraw() {
    const rect = immersiveCanvas.getBoundingClientRect();
    const width = Math.max(1, Math.round(rect.width));
    const height = Math.max(1, Math.round(rect.height));
    const ratio = Math.min(window.devicePixelRatio || 1, 2);

    if (width === lastWidth && height === lastHeight) {
      return;
    }

    lastWidth = width;
    lastHeight = height;

    immersiveCanvas.width = Math.round(width * ratio);
    immersiveCanvas.height = Math.round(height * ratio);
    ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
    drawFrame(width, height);
  }

  const resizeObserver = new ResizeObserver(() => resizeAndDraw());
  resizeObserver.observe(immersiveCanvas);

  window.addEventListener("load", resizeAndDraw, { once: true });
  window.addEventListener("resize", resizeAndDraw);
  resizeAndDraw();
}
