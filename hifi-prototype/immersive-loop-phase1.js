const phaseOneRoot = document.querySelector(".final-home");

if (phaseOneRoot) {
  const sceneClock = document.getElementById("sceneClock");
  const sceneShift = document.getElementById("sceneShift");
  const sceneObjective = document.getElementById("sceneObjective");
  const resourceWood = document.getElementById("resourceWood");
  const resourceStone = document.getElementById("resourceStone");
  const resourceHerb = document.getElementById("resourceHerb");
  const resourceFaith = document.getElementById("resourceFaith");
  const statusRule = document.getElementById("statusRule");
  const statusBuild = document.getElementById("statusBuild");
  const statusPlayable = document.getElementById("statusPlayable");
  const selectedLotReadout = document.getElementById("selectedLotReadout");
  const selectedBlueprintReadout = document.getElementById("selectedBlueprintReadout");
  const loopFeedback = document.getElementById("loopFeedback");
  const taskFlowList = document.getElementById("taskFlowList");
  const pawnLayer = document.getElementById("pawnLayer");
  const placeBlueprintBtn = document.getElementById("placeBlueprintBtn");
  const clearSelectionBtn = document.getElementById("clearSelectionBtn");
  const taskBadgeStore = document.getElementById("taskBadgeStore");
  const taskBadgeBuild = document.getElementById("taskBadgeBuild");

  const lotButtons = {
    east: document.getElementById("lotEast"),
    south: document.getElementById("lotSouth"),
    west: document.getElementById("lotWest"),
  };
  const blueprintButtons = Array.from(document.querySelectorAll(".phase1-blueprint-btn"));

  const lotConfig = {
    east: { label: "东坪地块", anchor: { x: 0.72, y: 0.42 } },
    south: { label: "南坡地块", anchor: { x: 0.55, y: 0.6 } },
    west: { label: "西侧地块", anchor: { x: 0.31, y: 0.52 } },
  };

  const blueprints = {
    alchemy: {
      name: "炼丹房",
      lotHint: "适合东坪，邻接主路时效率更高",
      cost: { wood: 80, stone: 45, faith: 12 },
      buildSeconds: 36,
      upkeep: { herb: 0.32 },
      produce: { faith: 0.28 },
    },
    herbGarden: {
      name: "药园",
      lotHint: "适合南坡，持续补足灵草",
      cost: { wood: 44, stone: 22, faith: 6 },
      buildSeconds: 30,
      upkeep: {},
      produce: { herb: 0.5 },
    },
    forge: {
      name: "炼器房",
      lotHint: "适合西侧，能稳步补足石料",
      cost: { wood: 62, stone: 68, faith: 10 },
      buildSeconds: 40,
      upkeep: { faith: 0.08 },
      produce: { stone: 0.32 },
    },
  };

  const clockNames = [
    "子时",
    "丑时",
    "寅时",
    "卯时",
    "辰时",
    "巳时",
    "午时",
    "未时",
    "申时",
    "酉时",
    "戌时",
    "亥时",
  ];

  const mapNodes = [];
  const mapDiscipleNodes = [];
  const state = {
    selectedLot: null,
    selectedBlueprint: "alchemy",
    resources: {
      wood: 268,
      stone: 154,
      herb: 82,
      faith: 48,
    },
    lots: {
      east: { id: "east", building: null, construction: null },
      south: { id: "south", building: null, construction: null },
      west: { id: "west", building: null, construction: null },
    },
    disciples: [
      {
        name: "顾青遥",
        color: "rgba(146, 185, 212, 0.95)",
        accent: "rgba(91, 130, 165, 0.95)",
        x: 0.58,
        y: 0.56,
        targetX: 0.58,
        targetY: 0.56,
        task: "待命",
        detail: "等待指令",
      },
      {
        name: "林采微",
        color: "rgba(229, 194, 145, 0.95)",
        accent: "rgba(167, 121, 75, 0.95)",
        x: 0.52,
        y: 0.7,
        targetX: 0.52,
        targetY: 0.7,
        task: "待命",
        detail: "巡视宗门",
      },
      {
        name: "沈见山",
        color: "rgba(205, 187, 228, 0.95)",
        accent: "rgba(133, 110, 170, 0.95)",
        x: 0.4,
        y: 0.62,
        targetX: 0.4,
        targetY: 0.62,
        task: "待命",
        detail: "巡视宗门",
      },
    ],
    feedbackType: "info",
    feedbackText: "请选择一个地块开始验证。",
    inGameMinutes: 12 * 24 * 60 + 8 * 60,
    completedBuildings: 0,
    totalPlaced: 0,
  };

  function clamp(value, min, max) {
    return Math.min(max, Math.max(min, value));
  }

  function formatInt(value) {
    return Math.max(0, Math.round(value));
  }

  function updateFeedback(text, type = "info") {
    state.feedbackText = text;
    state.feedbackType = type;
  }

  function getCurrentHour() {
    return Math.floor((state.inGameMinutes % (24 * 60)) / 60);
  }

  function getShiftLabel() {
    const hour = getCurrentHour();
    if (hour >= 6 && hour < 18) {
      return "白昼作业段";
    }
    if (hour >= 18 && hour < 22) {
      return "黄昏调整段";
    }
    return "夜间轮值段";
  }

  function getWorkMultiplier() {
    const hour = getCurrentHour();
    if (hour >= 6 && hour < 18) {
      return 1;
    }
    if (hour >= 18 && hour < 22) {
      return 0.82;
    }
    return 0.56;
  }

  function formatClock() {
    const day = Math.floor(state.inGameMinutes / (24 * 60));
    const hour = getCurrentHour();
    const branch = clockNames[Math.floor(hour / 2) % clockNames.length];
    return `第 ${day} 日 · ${branch}`;
  }

  function getConstructionLots() {
    return Object.values(state.lots).filter((lot) => lot.construction);
  }

  function selectLot(lotId) {
    state.selectedLot = lotId;
    const lot = lotConfig[lotId];
    const blueprint = blueprints[state.selectedBlueprint];
    selectedLotReadout.textContent = `当前地块：${lot.label}`;
    selectedBlueprintReadout.textContent = `当前蓝图：${blueprint.name}`;
    updateFeedback(`${lot.label} 已选中，可落位 ${blueprint.name}。`);
  }

  function selectBlueprint(blueprintId) {
    state.selectedBlueprint = blueprintId;
    const blueprint = blueprints[blueprintId];
    if (state.selectedLot) {
      selectedBlueprintReadout.textContent = `当前蓝图：${blueprint.name}`;
      updateFeedback(`${blueprint.name} 已选中，${blueprint.lotHint}。`);
    } else {
      selectedBlueprintReadout.textContent = `当前蓝图：${blueprint.name}`;
      updateFeedback(`${blueprint.name} 已选中，请先选择地块。`);
    }
  }

  function canAfford(cost) {
    return Object.entries(cost).every(([key, amount]) => state.resources[key] >= amount);
  }

  function placeBlueprint() {
    if (!state.selectedLot) {
      updateFeedback("请先点击地图中的一个地块。", "warn");
      return;
    }

    const lot = state.lots[state.selectedLot];
    const lotName = lotConfig[state.selectedLot].label;
    if (lot.construction || lot.building) {
      updateFeedback(`${lotName} 已有建筑任务，请换一块地。`, "warn");
      return;
    }

    const blueprint = blueprints[state.selectedBlueprint];
    if (!canAfford(blueprint.cost)) {
      updateFeedback(`${blueprint.name} 资源不足，无法落位。`, "warn");
      return;
    }

    lot.construction = {
      blueprintId: state.selectedBlueprint,
      phase: "delivery",
      delivered: { wood: 0, stone: 0, faith: 0 },
      progress: 0,
    };
    state.totalPlaced += 1;
    updateFeedback(`${lotName} 已落位 ${blueprint.name} 蓝图，弟子开始搬运。`, "good");
  }

  function clearSelection() {
    state.selectedLot = null;
    selectedLotReadout.textContent = "当前地块：未选择";
    updateFeedback("已清空地块选择。");
  }

  function updateConstruction(dt) {
    const workScale = getWorkMultiplier();
    Object.entries(state.lots).forEach(([lotId, lot]) => {
      const construction = lot.construction;
      if (!construction) {
        return;
      }

      const blueprint = blueprints[construction.blueprintId];
      if (construction.phase === "delivery") {
        Object.entries(blueprint.cost).forEach(([resource, cost]) => {
          const delivered = construction.delivered[resource] || 0;
          const remaining = Math.max(0, cost - delivered);
          if (remaining <= 0) {
            return;
          }
          const maxDelivery = cost * 0.11 * dt * workScale;
          const taken = Math.min(remaining, maxDelivery, state.resources[resource]);
          if (taken > 0) {
            construction.delivered[resource] = delivered + taken;
            state.resources[resource] -= taken;
          }
        });

        const allDelivered = Object.entries(blueprint.cost).every(
          ([resource, cost]) => (construction.delivered[resource] || 0) >= cost - 0.01,
        );
        if (allDelivered) {
          construction.phase = "build";
          construction.progress = 0;
          updateFeedback(`${lotConfig[lotId].label} 备料完成，转入施工。`, "good");
        }
        return;
      }

      construction.progress += ((dt * workScale) / blueprint.buildSeconds) * 100;
      if (construction.progress >= 100) {
        lot.building = {
          blueprintId: construction.blueprintId,
          onlineSeconds: 0,
        };
        lot.construction = null;
        state.completedBuildings += 1;
        updateFeedback(`${lotConfig[lotId].label} ${blueprint.name} 已启用。`, "good");
      }
    });
  }

  function runProduction(dt) {
    Object.values(state.lots).forEach((lot) => {
      if (!lot.building) {
        return;
      }
      const blueprint = blueprints[lot.building.blueprintId];
      lot.building.onlineSeconds += dt;

      const canRun = Object.entries(blueprint.upkeep).every(([resource, rate]) => state.resources[resource] >= rate * dt);
      if (!canRun) {
        return;
      }

      Object.entries(blueprint.upkeep).forEach(([resource, rate]) => {
        state.resources[resource] -= rate * dt;
      });
      Object.entries(blueprint.produce).forEach(([resource, rate]) => {
        state.resources[resource] += rate * dt;
      });
    });
  }

  function updateDisciples(dt) {
    const constructions = getConstructionLots();
    const firstConstruction = constructions[0];
    const secondConstruction = constructions[1];

    const warehouse = { x: 0.66, y: 0.56 };
    const hall = { x: 0.48, y: 0.35 };
    const patrolWest = { x: 0.32, y: 0.62 };
    const patrolSouth = { x: 0.57, y: 0.76 };

    if (firstConstruction) {
      const anchor = lotConfig[firstConstruction.id].anchor;
      const firstBlueprint = blueprints[firstConstruction.construction.blueprintId];
      if (firstConstruction.construction.phase === "delivery") {
        state.disciples[0].task = "搬运";
        state.disciples[0].detail = `${firstBlueprint.name} 备料`;
        const phase = Math.sin(state.inGameMinutes * 0.03) > 0 ? warehouse : anchor;
        state.disciples[0].targetX = phase.x;
        state.disciples[0].targetY = phase.y;

        state.disciples[1].task = "放样";
        state.disciples[1].detail = `${lotConfig[firstConstruction.id].label} 划线`;
        state.disciples[1].targetX = anchor.x - 0.03;
        state.disciples[1].targetY = anchor.y + 0.02;
      } else {
        state.disciples[0].task = "施工";
        state.disciples[0].detail = `${firstBlueprint.name} 主体搭建`;
        state.disciples[0].targetX = anchor.x;
        state.disciples[0].targetY = anchor.y;

        state.disciples[1].task = "施工";
        state.disciples[1].detail = `${firstBlueprint.name} 收边`;
        state.disciples[1].targetX = anchor.x - 0.04;
        state.disciples[1].targetY = anchor.y - 0.02;
      }
    } else {
      state.disciples[0].task = "巡山";
      state.disciples[0].detail = "检查道路";
      state.disciples[0].targetX = hall.x;
      state.disciples[0].targetY = hall.y + 0.18;

      state.disciples[1].task = "待命";
      state.disciples[1].detail = "等待建造任务";
      state.disciples[1].targetX = patrolSouth.x;
      state.disciples[1].targetY = patrolSouth.y;
    }

    if (secondConstruction) {
      const anchor = lotConfig[secondConstruction.id].anchor;
      state.disciples[2].task = "支援";
      state.disciples[2].detail = `${lotConfig[secondConstruction.id].label} 支援`;
      state.disciples[2].targetX = anchor.x + 0.02;
      state.disciples[2].targetY = anchor.y + 0.02;
    } else {
      const activeGarden = Object.entries(state.lots).find(
        ([, lot]) => lot.building && lot.building.blueprintId === "herbGarden",
      );
      if (activeGarden) {
        const anchor = lotConfig[activeGarden[0]].anchor;
        state.disciples[2].task = "照料";
        state.disciples[2].detail = "药园维护";
        state.disciples[2].targetX = anchor.x + 0.02;
        state.disciples[2].targetY = anchor.y - 0.01;
      } else {
        state.disciples[2].task = "巡山";
        state.disciples[2].detail = "西侧警戒";
        state.disciples[2].targetX = patrolWest.x;
        state.disciples[2].targetY = patrolWest.y;
      }
    }

    state.disciples.forEach((disciple) => {
      disciple.x += (disciple.targetX - disciple.x) * clamp(dt * 2.5, 0, 1);
      disciple.y += (disciple.targetY - disciple.y) * clamp(dt * 2.5, 0, 1);
    });
  }

  function renderLots() {
    Object.entries(lotButtons).forEach(([lotId, button]) => {
      const lotState = state.lots[lotId];
      const isSelected = lotId === state.selectedLot;
      button.classList.toggle("is-selected", isSelected);
      button.classList.toggle("is-occupied", !!lotState.building);
      button.classList.toggle("is-building", !!lotState.construction);

      let detail = "空地";
      if (lotState.construction) {
        const blueprint = blueprints[lotState.construction.blueprintId];
        if (lotState.construction.phase === "delivery") {
          detail = `${blueprint.name} · 备料`;
        } else {
          detail = `${blueprint.name} · 施工 ${Math.min(99, Math.floor(lotState.construction.progress))}%`;
        }
      } else if (lotState.building) {
        detail = `${blueprints[lotState.building.blueprintId].name} · 运行中`;
      }
      button.querySelector("span").textContent = `${lotConfig[lotId].label} · ${detail}`;
    });
  }

  function renderBlueprintButtons() {
    blueprintButtons.forEach((button) => {
      button.classList.toggle("is-selected", button.dataset.blueprint === state.selectedBlueprint);
    });
  }

  function renderPawns() {
    state.disciples.forEach((disciple, index) => {
      const node = mapDiscipleNodes[index];
      node.style.left = `${disciple.x * 100}%`;
      node.style.top = `${disciple.y * 100}%`;
      node.style.setProperty("--disciple-primary", disciple.color);
      node.style.setProperty("--disciple-accent", disciple.accent);
      node.querySelector(".map-pawn__name").textContent = disciple.name;
      node.querySelector(".map-pawn__task").textContent = `${disciple.task} · ${disciple.detail}`;
    });
  }

  function renderMapNodes() {
    mapNodes.forEach((node) => node.remove());
    mapNodes.length = 0;

    Object.entries(state.lots).forEach(([lotId, lot]) => {
      const anchor = lotConfig[lotId].anchor;
      if (!lot.construction && !lot.building) {
        return;
      }

      const node = document.createElement("div");
      node.className = "map-site";
      node.style.left = `${anchor.x * 100}%`;
      node.style.top = `${anchor.y * 100}%`;

      const title = document.createElement("strong");
      const sub = document.createElement("span");

      if (lot.construction) {
        const blueprint = blueprints[lot.construction.blueprintId];
        node.classList.add("is-construction");
        title.textContent = blueprint.name;
        if (lot.construction.phase === "delivery") {
          const totalCost = Object.values(blueprint.cost).reduce((sum, value) => sum + value, 0);
          const delivered = Object.entries(blueprint.cost).reduce(
            (sum, [resource]) => sum + (lot.construction.delivered[resource] || 0),
            0,
          );
          sub.textContent = `备料 ${Math.floor((delivered / totalCost) * 100)}%`;
        } else {
          sub.textContent = `施工 ${Math.floor(lot.construction.progress)}%`;
        }
      } else {
        const blueprint = blueprints[lot.building.blueprintId];
        node.classList.add("is-active");
        title.textContent = blueprint.name;
        sub.textContent = "运行中";
      }

      node.append(title, sub);
      mapNodes.push(node);
      pawnLayer.appendChild(node);
    });
  }

  function renderTaskFlow() {
    const constructionRows = [];
    Object.entries(state.lots).forEach(([lotId, lot]) => {
      if (!lot.construction) {
        return;
      }
      const blueprint = blueprints[lot.construction.blueprintId];
      if (lot.construction.phase === "delivery") {
        constructionRows.push(`${lotConfig[lotId].label}：${blueprint.name} 正在搬运备料`);
      } else {
        constructionRows.push(`${lotConfig[lotId].label}：${blueprint.name} 施工 ${Math.floor(lot.construction.progress)}%`);
      }
    });

    const productionRows = [];
    Object.entries(state.lots).forEach(([lotId, lot]) => {
      if (!lot.building) {
        return;
      }
      const blueprint = blueprints[lot.building.blueprintId];
      productionRows.push(`${lotConfig[lotId].label}：${blueprint.name} 持续产出中`);
    });

    const rows = [
      ...constructionRows,
      ...productionRows,
      ...state.disciples.map((disciple) => `${disciple.name}：${disciple.task}（${disciple.detail}）`),
    ].slice(0, 4);

    taskFlowList.innerHTML = rows.map((row) => `<li>${row}</li>`).join("");
  }

  function renderSelections() {
    if (state.selectedLot) {
      selectedLotReadout.textContent = `当前地块：${lotConfig[state.selectedLot].label}`;
    } else {
      selectedLotReadout.textContent = "当前地块：未选择";
    }

    selectedBlueprintReadout.textContent = `当前蓝图：${blueprints[state.selectedBlueprint].name}`;
  }

  function renderHud() {
    selectedBlueprintReadout.textContent = `当前蓝图：${blueprints[state.selectedBlueprint].name}`;
    if (!state.selectedLot) {
      selectedLotReadout.textContent = "当前地块：未选择";
    }

    resourceWood.textContent = `${formatInt(state.resources.wood)}`;
    resourceStone.textContent = `${formatInt(state.resources.stone)}`;
    resourceHerb.textContent = `${formatInt(state.resources.herb)}`;
    resourceFaith.textContent = `${formatInt(state.resources.faith)}`;

    sceneClock.textContent = formatClock();
    sceneShift.textContent = getShiftLabel();

    const constructionLots = getConstructionLots();
    if (constructionLots.length) {
      const lotName = lotConfig[constructionLots[0].id].label;
      sceneObjective.textContent = `${lotName} 正在推进施工`;
      statusBuild.textContent = `${lotName} ${blueprints[constructionLots[0].construction.blueprintId].name} 进行中`;
    } else {
      sceneObjective.textContent = state.completedBuildings ? "可继续扩建或观察产线反馈" : "请选择地块并落位蓝图";
      statusBuild.textContent = state.completedBuildings ? `已启用 ${state.completedBuildings} 栋建筑` : "暂无施工";
    }

    if (state.completedBuildings > 0) {
      statusPlayable.textContent = `可玩循环达成：已完成 ${state.completedBuildings} 栋建造并进入产出`;
    } else {
      statusPlayable.textContent = "请完成至少 1 栋建筑的落位与启用";
    }

    statusRule.textContent = getCurrentHour() >= 6 && getCurrentHour() < 18 ? "白昼优先建造与搬运" : "夜间优先修整与产线";
    taskBadgeStore.textContent = `搬运 ${state.disciples.filter((d) => d.task === "搬运").length}`;
    taskBadgeBuild.textContent = `施工 ${state.disciples.filter((d) => d.task === "施工").length}`;
  }

  function renderFeedback() {
    loopFeedback.textContent = state.feedbackText;
    loopFeedback.dataset.type = state.feedbackType;
  }

  function update(dt) {
    state.inGameMinutes += dt * 12;
    updateConstruction(dt);
    runProduction(dt);
    updateDisciples(dt);
  }

  function render() {
    renderSelections();
    renderLots();
    renderBlueprintButtons();
    renderMapNodes();
    renderPawns();
    renderTaskFlow();
    renderHud();
    renderFeedback();
  }

  function initPawnNodes() {
    state.disciples.forEach(() => {
      const node = document.createElement("div");
      node.className = "map-pawn";
      node.innerHTML = `
        <i class="map-pawn__dot"></i>
        <div class="map-pawn__meta">
          <strong class="map-pawn__name"></strong>
          <span class="map-pawn__task"></span>
        </div>
      `;
      pawnLayer.appendChild(node);
      mapDiscipleNodes.push(node);
    });
  }

  Object.entries(lotButtons).forEach(([lotId, button]) => {
    button.addEventListener("click", () => selectLot(lotId));
  });
  blueprintButtons.forEach((button) => {
    button.addEventListener("click", () => selectBlueprint(button.dataset.blueprint));
  });
  placeBlueprintBtn.addEventListener("click", placeBlueprint);
  clearSelectionBtn.addEventListener("click", clearSelection);

  initPawnNodes();
  render();

  let lastTimestamp = performance.now();
  function frame(timestamp) {
    const dt = clamp((timestamp - lastTimestamp) / 1000, 0, 0.08);
    lastTimestamp = timestamp;
    update(dt);
    render();
    window.requestAnimationFrame(frame);
  }

  window.requestAnimationFrame(frame);
}
