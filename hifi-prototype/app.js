const screens = [
  {
    id: "home",
    module: "主城系统",
    title: "主城首页",
    subtitle: "首页只承载 2.5D 地图观察、蓝图摆放、弟子优先级与昼夜排班，不再出现挂机式收取。",
    nav: "主城",
    render: () => {
      const priorities = [
        ["1", "搬运", "建筑落蓝图后先把材料送到位"],
        ["1", "建造", "工务弟子会立刻接手开工"],
        ["2", "种植", "保证药田与灵田不断档"],
        ["2", "炼丹", "夜间补足突破资源"],
        ["3", "修炼", "工作完成后自动转入修炼"],
      ];

      const shifts = [
        ["白昼", "06:00 - 18:00", "建造 / 搬运 / 药田照料", "82%"],
        ["黄昏", "18:00 - 22:00", "炼丹 / 修炼 / 回库", "56%"],
        ["夜间", "22:00 - 06:00", "休息 / 巡山 / 待命", "28%"],
      ];

      const structures = [
        ["hall", "宗门大殿", "命令中枢"],
        ["dorm", "弟子居", "床位 4 / 6"],
        ["warehouse", "仓苑", "木石入库"],
        ["field", "药田", "灵草成熟"],
      ];

      const pawns = [
        ["one", "顾青遥"],
        ["two", "林采微"],
        ["three", "沈见山"],
      ];

      const jobs = [
        ["顾青遥", "仓苑 → 炼丹房蓝图", "搬运木材 · 优先级 1"],
        ["林采微", "主路东侧地块", "放样与搭设地基"],
        ["沈见山", "药田 → 仓苑", "收拢灵草并入库"],
      ];

      return `
        <div class="screen home-screen home-overview screen-enter">
          <div class="home-overview__sky"></div>
          <div class="home-overview__mountains"></div>
          <div class="resource-ribbon home-resource-ribbon">
            ${resourceChip("木", "268", "木材")}
            ${resourceChip("石", "154", "石料")}
            ${resourceChip("草", "82", "灵草")}
            ${resourceChip("香", "48", "香火")}
          </div>
          <section class="home-cycle-ribbon">
            <div class="home-cycle-ribbon__meta">
              <strong>第 12 日 · 辰时</strong>
              <span>1 小时 = 1 天游戏时间 · 当前处于白昼作业段</span>
            </div>
            <div class="home-day-track">
              <span class="home-day-track__segment is-work">劳作</span>
              <span class="home-day-track__segment is-cultivate">修炼</span>
              <span class="home-day-track__segment is-rest">休息</span>
              <i class="home-day-track__cursor" style="--position: 34%"></i>
            </div>
            <p class="home-cycle-ribbon__note">白昼主抓建造与搬运，夜间转入修炼和休息。</p>
          </section>
          <div class="sect-plaque home-main-plaque">青岚宗 · 可操作主城</div>
          <section class="home-map-shell">
            <header class="home-map-shell__head">
              <div>
                <div class="design-kicker">2.5D Colony View</div>
                <strong>斜向俯视主地图 · 放置蓝图与观察弟子执行同屏完成</strong>
              </div>
              <div class="tag-row">
                <span class="pill good">施工队列 1</span>
                <span class="pill warn">待分配 1 人</span>
              </div>
            </header>
            <div class="home-map">
              <aside class="home-overlay-card home-overlay-card--left">
                <div class="design-kicker">Build Mode</div>
                <h3>建造模式</h3>
                <p>玩家只负责落位，搬料与施工交给弟子执行。</p>
                <div class="tag-row">
                  <span class="pill good">占地 2 × 3</span>
                  <span class="pill">邻接药田更优</span>
                </div>
                <div class="cost-row">
                  <span class="pill warn">木材 260</span>
                  <span class="pill warn">石料 120</span>
                  <span class="pill warn">香火 35</span>
                </div>
                <div class="home-mini-list">
                  <div class="home-mini-row"><strong>可放区域</strong><span>东侧空坪</span></div>
                  <div class="home-mini-row"><strong>施工条件</strong><span>需 2 名工务弟子</span></div>
                </div>
              </aside>
              <aside class="home-overlay-card home-overlay-card--right">
                <div class="design-kicker">Work Order</div>
                <h3>弟子规则</h3>
                <div class="home-priority-list">
                  ${priorities
                    .map(
                      ([rank, title, detail]) => `
                        <div class="home-priority-row">
                          <span class="home-priority-rank">${rank}</span>
                          <div>
                            <strong>${title}</strong>
                            <span>${detail}</span>
                          </div>
                        </div>
                      `,
                    )
                    .join("")}
                </div>
                <div class="home-shift-panel">
                  ${shifts
                    .map(
                      ([title, time, detail, fill]) => `
                        <div class="home-shift-row">
                          <div class="home-shift-row__head">
                            <strong>${title}</strong>
                            <span>${time}</span>
                          </div>
                          <div class="home-shift-row__bar"><span style="--fill: ${fill}"></span></div>
                          <p>${detail}</p>
                        </div>
                      `,
                    )
                    .join("")}
                </div>
              </aside>
              <div class="home-map__water home-map__water--upper"></div>
              <div class="home-map__water home-map__water--lower"></div>
              <div class="home-map__path"></div>
              <div class="home-map__board"></div>
              <div class="home-map__route home-map__route--one"></div>
              <div class="home-map__route home-map__route--two"></div>
              ${structures
                .map(
                  ([type, title, meta]) => `
                    <div class="home-structure home-structure--${type}">
                      <div class="home-structure__footprint"></div>
                      <div class="home-structure__body"></div>
                      <div class="home-structure__roof"></div>
                      <div class="home-structure__label">
                        <strong>${title}</strong>
                        <span>${meta}</span>
                      </div>
                    </div>
                  `,
                )
                .join("")}
              <div class="home-blueprint">
                <div class="home-blueprint__footprint"></div>
                <div class="home-blueprint__ghost"></div>
                <div class="home-blueprint__label">
                  <strong>炼丹房蓝图</strong>
                  <span>待搬运木材 260 / 石料 120</span>
                </div>
              </div>
              <div class="home-open-lot home-open-lot--north"><span>可放置 2 × 2</span></div>
              <div class="home-open-lot home-open-lot--south"><span>备用空地</span></div>
              ${pawns
                .map(
                  ([slot, name]) => `
                    <div class="home-pawn home-pawn--${slot}">
                      <span class="home-pawn__token"></span>
                      <div class="home-pawn__label">
                        <strong>${name}</strong>
                      </div>
                    </div>
                  `,
                )
                .join("")}
            </div>
            <footer class="home-map-shell__foot">
              <div class="home-status-chip"><strong>当前指令</strong><span>放置炼丹房蓝图</span></div>
              <div class="home-status-chip"><strong>物流规则</strong><span>材料由弟子搬运，不设手动收取</span></div>
              <div class="home-status-chip"><strong>执行规则</strong><span>先搬料，再施工，再入库</span></div>
            </footer>
          </section>
          <section class="home-job-ribbon">
            ${jobs
              .map(
                ([name, route, detail]) => `
                  <article class="home-job-card">
                    <strong>${name}</strong>
                    <span>${route}</span>
                    <em>${detail}</em>
                  </article>
                `,
              )
              .join("")}
          </section>
          ${bottomDock("主城")}
        </div>
      `;
    },
  },
  {
    id: "build",
    module: "建造系统",
    title: "建筑建造界面",
    subtitle: "以绢纸卷轴承载可建造建筑卡组，卡面沿用同一套青瓦屋脊和木构描边语言。",
    nav: "建造",
    render: () => sheetShell(
      "建造图谱",
      "仙居、生产、修炼、功能建筑在统一卷轴中切换。",
      `
      <div class="build-masterpiece">
        <div class="task-tabs">
          <span class="tab-chip is-active">营造图谱</span>
          <span class="tab-chip">生产建筑</span>
          <span class="tab-chip">修炼建筑</span>
          <span class="tab-chip">风水邻接</span>
        </div>
        <section class="build-feature-card">
          <div class="build-feature-art">
            <div class="build-feature-badge">今日推荐</div>
            <div class="build-feature-grid">
              ${new Array(18).fill("<span></span>").join("")}
            </div>
            <div class="roof-silhouette"></div>
          </div>
          <div class="build-feature-copy">
            <div class="design-kicker">丹火产线核心</div>
            <h4>二阶炼丹房</h4>
            <p>把灵草转化为突破资源，是 V1 经营闭环最关键的工坊节点。</p>
            <div class="build-stat-rack">
              <div class="build-stat-card"><strong>占地</strong><span>2 × 3</span></div>
              <div class="build-stat-card"><strong>工位</strong><span>2 名弟子</span></div>
              <div class="build-stat-card"><strong>邻接</strong><span>靠药田更优</span></div>
            </div>
            <div class="cost-row">
              <span class="pill warn">木材 260</span>
              <span class="pill warn">石料 120</span>
              <span class="pill warn">香火 35</span>
            </div>
          </div>
        </section>
        <section class="screen-card build-fengshui-card">
          <h3 class="screen-panel-title">邻接与风水</h3>
          <div class="build-note-list">
            <div class="build-note">
              <strong>贴近药田</strong>
              <p>周转提高 8%</p>
            </div>
            <div class="build-note">
              <strong>靠近仓苑</strong>
              <p>补料更稳定</p>
            </div>
            <div class="build-note">
              <strong>远离主路</strong>
              <p>保留主景视野</p>
            </div>
          </div>
        </section>
        <section class="screen-card">
          <div class="building-title">
            <strong>相关建筑</strong>
            <span class="pill good">当前主推 2 项</span>
          </div>
          <div class="build-library-grid">
            ${buildLibraryCard("药田", "灵草基础供给", "立即建造")}
            ${buildLibraryCard("仓苑", "扩大缓存容量", "建议尽早")}
          </div>
        </section>
        <div class="action-row">
          <button class="stage-btn" type="button">查看灵脉地块</button>
          <button class="secondary-cta" type="button">进入摆放</button>
        </div>
      </div>
      ${bottomDock("建造")}
    `,
      "build-screen",
    ),
  },
  {
    id: "building-detail",
    module: "建造系统",
    title: "建筑详情与升级",
    subtitle: "建筑详情不再只是表单，而是一张带预览、产线和成长收益的工坊图谱。",
    nav: "建造",
    render: () => sheetShell(
      "炼丹房 · 二阶",
      "产线状态、输入输出、升级收益在同一屏完成阅读。",
      `
      <div class="sheet-hero">
        <div class="hero-art-card">
          <div class="roof-silhouette"></div>
        </div>
        <div class="sheet-hero__copy">
          <div class="sheet-hero__title">山门炼丹坊</div>
          <p class="small-copy">云火温养，药鼎常明。当前由两名弟子轮值，承担全宗 78% 的丹药产出。</p>
          <div class="jade-metrics">
            <div class="metric-chip"><strong>状态</strong> 工作中</div>
            <div class="metric-chip"><strong>效率</strong> 126%</div>
            <div class="metric-chip"><strong>队列</strong> 3 批</div>
          </div>
        </div>
      </div>
      <div class="sheet-grid equal-col">
        <div class="screen-card">
          <h3 class="screen-panel-title">输入与产出</h3>
          <div class="input-output-grid">
            <div class="input-card">
              <strong>消耗</strong>
              <div class="cost-row">
                <span class="pill">灵草 × 12</span>
                <span class="pill">山泉 × 3</span>
                <span class="pill">香灰 × 1</span>
              </div>
            </div>
            <div class="output-card">
              <strong>产出</strong>
              <div class="reward-row">
                <span class="pill good">凝气丹 × 4</span>
                <span class="pill good">丹香值 +6</span>
              </div>
            </div>
          </div>
        </div>
        <div class="screen-card">
          <h3 class="screen-panel-title">升级预览</h3>
          <div class="attribute-list">
            ${attributeRow("产速", 72, "+18%")}
            ${attributeRow("暴丹率", 48, "+6%")}
            ${attributeRow("缓存", 62, "+2 格")}
          </div>
          <div class="cost-row">
            <span class="pill warn">木材 260</span>
            <span class="pill warn">石料 120</span>
            <span class="pill warn">香火 35</span>
          </div>
        </div>
      </div>
      <div class="action-row">
        <button class="stage-btn" type="button">查看岗位</button>
        <button class="secondary-cta" type="button">立即升级</button>
      </div>
      ${bottomDock("建造")}
    `),
  },
  {
    id: "resources",
    module: "仓储系统",
    title: "资源与仓库",
    subtitle: "仓库页强化“容量、流向、紧缺度”三个关键信息，让资源管理更像修仙工坊总账。",
    nav: "背包",
    render: () => sheetShell(
      "宗门仓储总览",
      "所有资源统一以库藏、占用和待搬运状态展示。",
      `
      <div class="sheet-grid two-col">
        <div class="profile-column">
          <div class="screen-card">
            <h3 class="screen-panel-title">仓库容量</h3>
            <div class="storage-meter">
              <div class="meter-head"><strong>总占用</strong><span>824 / 1100</span></div>
              <div class="progress-bar"><span style="--fill: 74%"></span></div>
              <div class="meter-stack">
                <div class="stock-chip"><span>基础材料</span><strong>362</strong></div>
                <div class="stock-chip"><span>灵草丹药</span><strong>228</strong></div>
                <div class="stock-chip"><span>法器材料</span><strong>146</strong></div>
                <div class="stock-chip"><span>香火奇物</span><strong>88</strong></div>
              </div>
            </div>
          </div>
          <div class="screen-card">
            <h3 class="screen-panel-title">库存清单</h3>
            <div class="supply-list">
              ${supplyItem("木材", "2680", "稳定", 84)}
              ${supplyItem("石料", "1540", "稳定", 68)}
              ${supplyItem("灵草", "126", "偏紧", 38)}
              ${supplyItem("矿石", "72", "短缺", 22)}
              ${supplyItem("香火", "45", "珍稀", 18)}
            </div>
          </div>
        </div>
        <div class="details-column">
          <div class="screen-card">
            <h3 class="screen-panel-title">仓储分区</h3>
            <div class="storage-grid">
              <div class="storage-card"><strong>主仓</strong><p>靠近大殿，搬运优先级最高。</p></div>
              <div class="storage-card"><strong>丹房副仓</strong><p>灵草与药材缓存，降低来回搬运。</p></div>
              <div class="storage-card"><strong>矿料棚</strong><p>靠近炼器房，减少堵塞。</p></div>
              <div class="storage-card"><strong>奇物柜</strong><p>只放香火、遗物和事件奖励。</p></div>
            </div>
          </div>
          <div class="screen-card">
            <h3 class="screen-panel-title">当前流向</h3>
            <div class="flow-diagram">
              ${flowLink("伐木坊", "主仓")}
              ${flowLink("药田", "炼丹房")}
              ${flowLink("矿坑", "炼器房")}
            </div>
          </div>
        </div>
      </div>
      ${bottomDock("背包")}
    `),
  },
  {
    id: "disciples",
    module: "弟子系统",
    title: "弟子总览",
    subtitle: "弟子列表改为“名册 + 侧边情报页”的结构，先看到分工，再进入培养。",
    nav: "宗门",
    render: () => sheetShell(
      "弟子总览",
      "外门、内门、长老与岗位分布统一在一页完成查看。",
      `
      <div class="disciple-masterpiece">
        <div class="jade-metrics">
          <div class="metric-chip"><strong>弟子总数</strong> 12</div>
          <div class="metric-chip"><strong>外门</strong> 7</div>
          <div class="metric-chip"><strong>内门</strong> 4</div>
          <div class="metric-chip"><strong>可重点培养</strong> 2</div>
        </div>
        <section class="disciple-hero-card">
          <div class="disciple-hero-portrait">
            <div class="portrait large heroine"></div>
            <div class="disciple-realm-seal">
              <strong>炼气七层</strong>
              <span>可冲筑基</span>
            </div>
          </div>
          <div class="disciple-hero-copy">
            <div class="design-kicker">丹房主修弟子</div>
            <h4>顾青遥</h4>
            <p>水灵根，主修《碧海凝气诀》，当前宗门最值得优先突破的弟子。</p>
            <div class="tag-row">
              <span class="pill good">丹道悟性 91</span>
              <span class="pill">灵根纯度 82%</span>
              <span class="pill warn">突破待准备</span>
            </div>
            <div class="disciple-vitals">
              <div class="disciple-vital"><strong>岗位</strong><span>炼丹房值守</span></div>
              <div class="disciple-vital"><strong>心境</strong><span>稳定</span></div>
              <div class="disciple-vital"><strong>忠诚</strong><span>极高</span></div>
            </div>
          </div>
        </section>
        <section class="screen-card">
          <h3 class="screen-panel-title">修行命盘</h3>
          <div class="attribute-list">
            ${attributeRow("悟性", 86, "86")}
            ${attributeRow("丹道", 91, "91")}
            ${attributeRow("神识", 73, "73")}
            ${attributeRow("体魄", 55, "55")}
            ${attributeRow("心境", 69, "69")}
          </div>
        </section>
        <section class="screen-card">
          <div class="building-title">
            <strong>功法 / 装备 / 建议</strong>
            <span class="pill good">突破前准备</span>
          </div>
          <div class="reward-row">
            <span class="pill good">碧海凝气诀</span>
            <span class="pill">青玉丹炉</span>
            <span class="pill">护身玉简</span>
            <span class="pill warn">凝神露 ×1</span>
            <span class="pill warn">香火 ×20</span>
          </div>
        </section>
        <section class="screen-card">
          <div class="building-title">
            <strong>弟子名册</strong>
            <span class="pill good">重点可调度 2 人</span>
          </div>
          <div class="disciple-roster-grid">
            ${rosterTile("林清禾", "药田驻守", "木灵根 · 灵植 +20%", "稳定")}
            ${rosterTile("叶微澜", "候补修行", "土灵根 · 可调往练功", "待命")}
          </div>
        </section>
        <div class="action-row">
          <button class="stage-btn" type="button">调整岗位</button>
          <button class="secondary-cta" type="button">准备突破</button>
        </div>
      </div>
      ${bottomDock("宗门")}
    `,
      "disciple-screen",
    ),
  },
  {
    id: "training",
    module: "修炼系统",
    title: "弟子详情与培养",
    subtitle: "培养页突出境界、灵根和突破材料，视觉上更接近一张弟子命盘。",
    nav: "修行",
    render: () => sheetShell(
      "弟子培养",
      "以命盘式排布承载属性、功法、突破材料与装备位。",
      `
      <div class="sheet-grid two-col">
        <div class="profile-column">
          <div class="screen-card">
            <div class="profile-head">
              <div class="portrait large heroine"></div>
              <div class="hero-copy">
                <h4>顾青遥</h4>
                <p>水灵根 · 炼气七层 · 炼丹悟性极高，适合作为丹房主修弟子。</p>
              </div>
            </div>
            <div class="tag-row">
              <span class="pill good">灵根纯度 82%</span>
              <span class="pill">主修《碧海凝气诀》</span>
              <span class="pill warn">可突破筑基</span>
            </div>
          </div>
          <div class="screen-card">
            <h3 class="screen-panel-title">属性命盘</h3>
            <div class="attribute-list">
              ${attributeRow("悟性", 86, "86")}
              ${attributeRow("丹道", 91, "91")}
              ${attributeRow("神识", 73, "73")}
              ${attributeRow("体魄", 55, "55")}
              ${attributeRow("心境", 69, "69")}
            </div>
          </div>
        </div>
        <div class="details-column">
          <div class="screen-card">
            <h3 class="screen-panel-title">功法与装备</h3>
            <div class="equipment-row">
              <span class="pill good">青玉丹炉</span>
              <span class="pill">护身玉简</span>
              <span class="pill">本命符袋</span>
            </div>
            <div class="task-list">
              <div class="task-card">
                <div class="task-step">功</div>
                <div>
                  <h4>碧海凝气诀</h4>
                  <p>提升灵气回复与丹火控制，当前熟练度 72%。</p>
                  <div class="progress-bar"><span style="--fill: 72%"></span></div>
                </div>
              </div>
              <div class="task-card">
                <div class="task-step">突</div>
                <div>
                  <h4>筑基准备</h4>
                  <p>缺少凝神露 1 瓶、香火 20 点，满足后可尝试突破。</p>
                  <div class="cost-row">
                    <span class="pill warn">凝神露 ×1</span>
                    <span class="pill warn">香火 ×20</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div class="action-row">
            <button class="stage-btn" type="button">调整功法</button>
            <button class="secondary-cta" type="button">开始突破</button>
          </div>
        </div>
      </div>
      ${bottomDock("修行")}
    `),
  },
  {
    id: "crafting",
    module: "生产系统",
    title: "炼丹炼器生产",
    subtitle: "生产页拆成工坊队列 + 当前材料 + 产出货架，让经营链条读起来更直观。",
    nav: "修行",
    render: () => sheetShell(
      "炼丹炼器",
      "两个工坊共用一套材质语言，通过卷签切换。",
      `
      <div class="task-tabs">
        <span class="tab-chip is-active">炼丹房</span>
        <span class="tab-chip">炼器房</span>
        <span class="tab-chip">丹方典籍</span>
      </div>
      <div class="sheet-grid equal-col">
        <div class="screen-card">
          <h3 class="screen-panel-title">生产队列</h3>
          <div class="queue-list">
            ${queueCard("凝气丹", "2m 40s / 批", "灵草×12 · 山泉×3", "稳定批量产出")}
            ${queueCard("回灵丹", "4m 10s / 批", "灵花×8 · 香灰×2", "用于突破前补给")}
          </div>
        </div>
        <div class="screen-card">
          <h3 class="screen-panel-title">工坊侧记</h3>
          <div class="craft-art">
            <div class="roof-silhouette small"></div>
          </div>
          <div class="reward-row">
            <span class="pill good">丹香充盈</span>
            <span class="pill warn">灵草库存偏紧</span>
          </div>
          <p class="caption">建议优先补充药田驻守弟子，以保证灵草供应不断档。</p>
        </div>
      </div>
      <div class="screen-card">
        <h3 class="screen-panel-title">产出货架</h3>
        <div class="inventory-grid">
          ${inventoryCard("凝气丹", "12 瓶", "常规突破与修炼消耗")}
          ${inventoryCard("醒神散", "4 包", "事件前提升神降成功率")}
          ${inventoryCard("火纹胚", "2 件", "炼器房半成品，待继续锻造")}
        </div>
      </div>
      ${bottomDock("修行")}
    `),
  },
  {
    id: "descent",
    module: "神降系统",
    title: "神降事件",
    subtitle: "神降界面强化异界卷宗感，以深色背景托起剧情和风险选择，而不是普通弹窗。",
    nav: "历练",
    render: () => sheetShell(
      "神降事件",
      "故事、抉择、奖励与代价在一张卷轴中递进展示。",
      `
      <div class="descent-masterpiece">
        <div class="descent-rune-bar">
          <span>当前锚点：末法小城 · 香火薄弱</span>
          <span>剩余神降次数：2</span>
        </div>
        <section class="descent-hero-card">
          <div class="event-art descent-event-art">
            <div class="roof-silhouette"></div>
          </div>
          <div class="descent-hero-copy">
            <div class="design-kicker">异界异闻</div>
            <h4>香火薄城</h4>
            <p>疫病与饥荒并起，守灯老者仍在残庙中祈祷，凡民盼望神明回应。</p>
            <div class="tag-row">
              <span class="pill good">民心可救</span>
              <span class="pill">医者可结缘</span>
              <span class="pill warn">豪强暗中窥视</span>
            </div>
          </div>
        </section>
        <section class="screen-card descent-prophecy-card">
          <h3 class="screen-panel-title">天机批注</h3>
          <div class="reward-row">
            <span class="pill">守灯老者可续香火</span>
            <span class="pill">凡城医者可结缘</span>
            <span class="pill warn">豪强反噬需提防</span>
          </div>
        </section>
        <div class="descent-choice-stack">
          ${oracleChoice("壹", "显化神迹，救治重症", "直接稳住局势，快速收拢信仰。", "香火 +12~20", "耗香火 10 · 低风险", "is-favored")}
          ${oracleChoice("贰", "借医者身传药布道", "以人间手段行神意，换取长线结缘。", "后续收益高", "中风险", "")}
          ${oracleChoice("叁", "震慑豪强，强夺药库", "资源最多，但凡民会又敬又惧。", "资源爆发", "高风险", "is-risky")}
        </div>
        <div class="action-row">
          <button class="stage-btn" type="button">暂不降临</button>
          <button class="secondary-cta" type="button">确认神降</button>
        </div>
      </div>
      ${bottomDock("历练")}
    `,
      "descent-screen",
    ),
  },
  {
    id: "expansion",
    module: "扩张系统",
    title: "宗门扩张与地块解锁",
    subtitle: "扩张页用斜置灵脉地块表达外圈山门扩展，突出地块关系而不是纯表格。",
    nav: "建造",
    render: () => sheetShell(
      "宗门扩张",
      "灵脉地块以等角菱形呈现，强化“向外开山辟地”的体验。",
      `
      <div class="sheet-grid equal-col">
        <div class="screen-card">
          <h3 class="screen-panel-title">地块总览</h3>
          <div class="land-grid">
            <div class="land-tile" data-name="大殿"></div>
            <div class="land-tile is-highlight" data-name="丹坊"></div>
            <div class="land-tile is-locked" data-name="药谷"></div>
            <div class="land-tile is-locked" data-name="矿坡"></div>
            <div class="land-tile" data-name="仓苑"></div>
            <div class="land-tile" data-name="讲经"></div>
            <div class="land-tile is-locked" data-name="灵池"></div>
            <div class="land-tile is-locked" data-name="祭坛"></div>
          </div>
        </div>
        <div class="screen-card">
          <h3 class="screen-panel-title">待解锁地块</h3>
          <div class="profile-column">
            <div class="building-card">
              <div class="building-title"><strong>药谷地块</strong><span class="pill good">推荐优先</span></div>
              <p class="building-meta">可额外容纳 2 块药田，并获得山泉加成。</p>
              <div class="cost-row">
                <span class="pill warn">木材 320</span>
                <span class="pill warn">香火 40</span>
                <span class="pill warn">名望 2 级</span>
              </div>
            </div>
            <div class="building-card">
              <div class="building-title"><strong>矿坡地块</strong><span class="pill">后续</span></div>
              <p class="building-meta">解锁采石与矿脉建筑，支撑炼器链。</p>
              <div class="cost-row">
                <span class="pill warn">石料 460</span>
                <span class="pill warn">弟子 2 人</span>
              </div>
            </div>
          </div>
          <div class="action-row">
            <button class="stage-btn" type="button">查看灵脉</button>
            <button class="secondary-cta" type="button">解锁药谷</button>
          </div>
        </div>
      </div>
      ${bottomDock("建造")}
    `),
  },
  {
    id: "tasks",
    module: "目标系统",
    title: "任务阶段目标",
    subtitle: "任务页改成章节卷轴，强调当前阶段、关键目标和下一步引导，避免密集清单感。",
    nav: "宗门",
    render: () => sheetShell(
      "宗门阶段目标",
      "阶段奖励与关键任务并排展示，方便新玩家理解当前经营主线。",
      `
      <div class="task-tabs">
        <span class="tab-chip is-active">第一章 · 开山立派</span>
        <span class="tab-chip">第二章 · 丹坊起火</span>
        <span class="tab-chip">第三章 · 初试神降</span>
      </div>
      <div class="sheet-grid two-col">
        <div class="stage-rail">
          ${taskStage("壹", "建造第一块药田", "完成度 100%", 100)}
          ${taskStage("贰", "让炼丹房开始运转", "完成度 75%", 75)}
          ${taskStage("叁", "完成一次神降", "完成度 20%", 20)}
          ${taskStage("肆", "解锁药谷地块", "完成度 0%", 0)}
        </div>
        <div class="details-column">
          <div class="screen-card">
            <h3 class="screen-panel-title">阶段奖励</h3>
            <div class="reward-row">
              <span class="pill good">香火 +30</span>
              <span class="pill good">弟子招募帖 ×1</span>
              <span class="pill good">青玉丹炉图纸</span>
            </div>
            <p class="caption">完成本章后将开放第一轮宗门扩张，并解锁神降的第二套事件模板。</p>
          </div>
          <div class="screen-card">
            <h3 class="screen-panel-title">建议路径</h3>
            <div class="diagnostic-list">
              <div class="diagnostic-item"><strong>1</strong><p>补足药田驻守弟子，先保证灵草稳定产出。</p></div>
              <div class="diagnostic-item"><strong>2</strong><p>让炼丹房维持两批基础丹药队列，建立正向循环。</p></div>
              <div class="diagnostic-item"><strong>3</strong><p>等丹药和香火到位后，再尝试神降节点。</p></div>
            </div>
          </div>
        </div>
      </div>
      ${bottomDock("宗门")}
    `),
  },
  {
    id: "placement",
    module: "建造系统",
    title: "建设摆放界面",
    subtitle: "摆放界面保留场景感，用网格幽灵体和底部决策条提示当前占地是否合法。",
    nav: "建造",
    render: () => sheetShell(
      "建筑摆放",
      "半透地图叠加逻辑网格，让玩家能在不破坏氛围的前提下完成摆放判断。",
      `
      <div class="sheet-grid two-col">
        <div class="screen-card">
          <div class="ghost-map">
            <div class="grid-overlay">${new Array(30).fill("<span></span>").join("")}</div>
            <div class="ghost-building"></div>
          </div>
        </div>
        <div class="details-column">
          <div class="screen-card">
            <h3 class="screen-panel-title">当前建筑</h3>
            <div class="building-card">
              <div class="building-title"><strong>炼丹房</strong><span class="pill good">占地 2 × 3</span></div>
              <p class="building-meta">靠近药田会获得额外 8% 搬运效率，避免与道路和主建筑重叠。</p>
            </div>
          </div>
          <div class="screen-card">
            <h3 class="screen-panel-title">摆放判断</h3>
            <div class="reward-list">
              <span class="pill good">地块合法</span>
              <span class="pill good">邻近药田</span>
              <span class="pill warn">离仓库偏远</span>
            </div>
          </div>
          <div class="action-row">
            <button class="stage-btn" type="button">旋转朝向</button>
            <button class="secondary-cta" type="button">确认建造</button>
          </div>
        </div>
      </div>
      ${bottomDock("建造")}
    `),
  },
  {
    id: "jobs",
    module: "岗位系统",
    title: "弟子岗位安排",
    subtitle: "岗位页采用工坊排班表语义，让弟子数量、空位和推荐操作一眼可见。",
    nav: "修行",
    render: () => sheetShell(
      "岗位安排",
      "岗位需求、现有人手与推荐调度集中在一张工坊排班表中。",
      `
      <div class="sheet-grid two-col">
        <div class="screen-card">
          <h3 class="screen-panel-title">岗位需求</h3>
          <div class="job-table">
            ${assignmentRow("药田", 3, 2)}
            ${assignmentRow("炼丹房", 2, 2)}
            ${assignmentRow("炼器房", 2, 1)}
            ${assignmentRow("仓库搬运", 2, 1)}
            ${assignmentRow("练功场", 1, 0)}
          </div>
        </div>
        <div class="details-column">
          <div class="screen-card">
            <h3 class="screen-panel-title">可调度弟子</h3>
            <div class="split-list">
              ${miniDisciple("叶微澜", "土灵根 · 可转入炼功")}
              ${miniDisciple("沈知白", "杂灵根 · 擅长搬运")}
              ${miniDisciple("周映丹", "火灵根 · 炼器效率 +12%")}
            </div>
          </div>
          <div class="screen-card">
            <h3 class="screen-panel-title">系统建议</h3>
            <div class="recommend-row">
              <span class="recommend-chip">优先补足药田 1 人</span>
              <span class="recommend-chip">安排采矿，恢复炼器</span>
              <span class="recommend-chip">练功位可临时撤人</span>
            </div>
          </div>
        </div>
      </div>
      ${bottomDock("修行")}
    `),
  },
  {
    id: "bag",
    module: "背包系统",
    title: "背包与法宝道具",
    subtitle: "背包页以匣盒陈列的方式组织材料、丹药、奇物和法宝，避免普通 RPG 列表感。",
    nav: "背包",
    render: () => sheetShell(
      "百宝匣",
      "法宝、丹药、材料与奇物统一收纳在宗门匣盒之中。",
      `
      <div class="task-tabs">
        <span class="tab-chip is-active">法宝</span>
        <span class="tab-chip">丹药</span>
        <span class="tab-chip">材料</span>
        <span class="tab-chip">奇物</span>
      </div>
      <div class="sheet-grid two-col">
        <div class="screen-card">
          <h3 class="screen-panel-title">法宝匣盒</h3>
          <div class="artifact-grid">
            ${artifactCard("青玉炉心", "丹火稳定 +10%")}
            ${artifactCard("木灵护符", "药田产量 +8%")}
            ${artifactCard("山海罗盘", "神降成功率 +5%")}
            ${artifactCard("镇魂玉简", "突破失败损耗降低")}
            ${artifactCard("玄铁胚子", "可继续锻造成器")}
            ${artifactCard("香火牌位", "被动积累香火")}
          </div>
        </div>
        <div class="details-column">
          <div class="screen-card">
            <h3 class="screen-panel-title">当前选中</h3>
            <div class="artifact-preview">
              <div class="artifact-shape"></div>
            </div>
            <div class="inventory-title"><strong>青玉炉心</strong><span class="pill good">传承法宝</span></div>
            <p class="inventory-meta">炉心灵韵充盈，可安放于炼丹房中央，使基础丹药成丹率提升 10%。</p>
            <div class="reward-row">
              <span class="pill good">成丹率 +10%</span>
              <span class="pill">丹火稳定 +12</span>
            </div>
          </div>
          <div class="action-row">
            <button class="stage-btn" type="button">收纳入匣</button>
            <button class="secondary-cta" type="button">装备到丹房</button>
          </div>
        </div>
      </div>
      ${bottomDock("背包")}
    `),
  },
  {
    id: "diagnostics",
    module: "经营总览",
    title: "宗门经营总览 / 生产诊断",
    subtitle: "诊断页做成‘经营总管面板’，核心是瓶颈、建议和一键跳转，而不是纯报表。",
    nav: "宗门",
    render: () => sheetShell(
      "经营总览 / 生产诊断",
      "把运行状态、产线瓶颈与建议操作统一到一页，减少玩家迷失。",
      `
      <div class="stage-summary">
        <span class="metric-chip"><strong>总产出效率</strong> 良</span>
        <span class="metric-chip"><strong>空缺岗位</strong> 2</span>
        <span class="metric-chip"><strong>缺料建筑</strong> 3</span>
        <span class="metric-chip"><strong>推荐操作</strong> 补仓 / 调岗</span>
      </div>
      <div class="sheet-grid two-col">
        <div class="profile-column">
          <div class="screen-card">
            <h3 class="screen-panel-title">宗门运行状态</h3>
            <div class="state-list">
              ${stateItem("药田", "工作中", "产量偏低，建议增加 1 名木灵根弟子")}
              ${stateItem("炼丹房", "工作中", "灵草不足，建议优先补给")}
              ${stateItem("炼器房", "待机", "矿石不足，需安排采矿")}
              ${stateItem("仓库", "繁忙", "搬运压力中，考虑加开副仓")}
            </div>
          </div>
          <div class="screen-card">
            <h3 class="screen-panel-title">当前瓶颈</h3>
            <div class="diagnostic-list">
              <div class="diagnostic-item"><strong>1</strong><p>灵草产量不足，限制炼丹循环。</p></div>
              <div class="diagnostic-item"><strong>2</strong><p>炼器房缺少矿石，产线中断。</p></div>
              <div class="diagnostic-item"><strong>3</strong><p>仓库搬运略慢，影响部分建造补货。</p></div>
              <div class="diagnostic-item"><strong>4</strong><p>练功位空置，可考虑调派低阶弟子。</p></div>
            </div>
          </div>
        </div>
        <div class="details-column">
          <div class="screen-card">
            <h3 class="screen-panel-title">生产链总览</h3>
            <div class="flow-diagram">
              ${flowLink("伐木坊", "主仓")}
              ${flowLink("药田", "炼丹房")}
              ${flowLink("矿坑", "炼器房")}
            </div>
          </div>
          <div class="screen-card">
            <h3 class="screen-panel-title">建议优先处理</h3>
            <div class="reward-list">
              <span class="pill bad">高 · 补充药田驻守弟子</span>
              <span class="pill bad">高 · 安排采矿恢复炼器</span>
              <span class="pill warn">中 · 调整仓库并开启优先搬运</span>
              <span class="pill good">低 · 安排一名弟子进入练功</span>
            </div>
          </div>
          <div class="action-row">
            <button class="stage-btn" type="button">前往调岗</button>
            <button class="stage-btn" type="button">前往仓库</button>
            <button class="secondary-cta" type="button">前往建造</button>
          </div>
        </div>
      </div>
      ${bottomDock("宗门")}
    `),
  },
];

const navEl = document.getElementById("screenNav");
const phoneScreenEl = document.getElementById("phoneScreen");
const moduleEl = document.getElementById("screenModule");
const titleEl = document.getElementById("screenTitle");
const subtitleEl = document.getElementById("screenSubtitle");
const prevButton = document.getElementById("prevScreen");
const nextButton = document.getElementById("nextScreen");

let currentIndex = 0;

function init() {
  currentIndex = getInitialIndex();

  navEl.innerHTML = screens
    .map(
      (screen, index) => `
        <button type="button" data-index="${index}">
          <div>
            <strong>${String(index + 1).padStart(2, "0")} · ${screen.title}</strong>
            <span>${screen.module}</span>
          </div>
          <span>${screen.nav}</span>
        </button>
      `,
    )
    .join("");

  navEl.addEventListener("click", (event) => {
    const button = event.target.closest("button[data-index]");

    if (!button) {
      return;
    }

    currentIndex = Number(button.dataset.index);
    render();
  });

  prevButton.addEventListener("click", () => {
    currentIndex = (currentIndex - 1 + screens.length) % screens.length;
    render();
  });

  nextButton.addEventListener("click", () => {
    currentIndex = (currentIndex + 1) % screens.length;
    render();
  });

  window.addEventListener("keydown", (event) => {
    if (event.key === "ArrowLeft") {
      currentIndex = (currentIndex - 1 + screens.length) % screens.length;
      render();
    }

    if (event.key === "ArrowRight") {
      currentIndex = (currentIndex + 1) % screens.length;
      render();
    }
  });

  render();
}

function render() {
  const screen = screens[currentIndex];

  moduleEl.textContent = screen.module;
  titleEl.textContent = screen.title;
  subtitleEl.textContent = screen.subtitle;
  phoneScreenEl.innerHTML = screen.render();
  updateLocation(screen.id);

  [...navEl.querySelectorAll("button")].forEach((button, index) => {
    button.classList.toggle("is-active", index === currentIndex);
  });
}

function getInitialIndex() {
  const params = new URLSearchParams(window.location.search);
  const queryScreen = params.get("screen");
  const hashScreen = window.location.hash.replace("#", "");
  const target = queryScreen || hashScreen;

  if (!target) {
    return 0;
  }

  const foundIndex = screens.findIndex((screen) => screen.id === target);
  return foundIndex >= 0 ? foundIndex : 0;
}

function updateLocation(screenId) {
  const url = new URL(window.location.href);
  url.searchParams.set("screen", screenId);
  window.history.replaceState({}, "", url);
}

function sheetShell(title, subtitle, body, screenClass = "") {
  return `
    <div class="screen panel-screen ${screenClass} screen-enter">
      <section class="sheet-shell">
        <header class="sheet-header">
          <div>
            <div class="design-kicker">QINGLAN SECT UI</div>
            <h3 class="sheet-title">${title}</h3>
            <p class="sheet-subtitle">${subtitle}</p>
          </div>
          <div class="tag-row">
            <span class="pill">云岚卷</span>
            <span class="pill">竖屏原型</span>
          </div>
        </header>
        ${body}
      </section>
    </div>
  `;
}

function resourceChip(icon, value, label = `${icon}材`) {
  return `
    <div class="resource-chip">
      <span class="resource-dot">${icon}</span>
      <span>${label} <b>${value}</b></span>
    </div>
  `;
}

function actionTile(icon, label, badge = "") {
  return `
    <div class="action-tile">
      ${badge ? `<span class="action-badge">${badge}</span>` : ""}
      <span class="action-icon">${icon}</span>
      <span>${label}</span>
    </div>
  `;
}

function homeScroll(icon, title, subtitle, badge = "") {
  return `
    <div class="home-scroll">
      ${badge ? `<span class="action-badge">${badge}</span>` : ""}
      <span class="home-scroll__icon">${icon}</span>
      <strong>${title}</strong>
      <span>${subtitle}</span>
    </div>
  `;
}

function bottomDock(active) {
  const normalizedActive =
    {
      主城: "主城",
      宗门: "主城",
      建造: "建造",
      修行: "弟子",
      背包: "弟子",
      历练: "神降",
      神降: "神降",
    }[active] || active;

  const items = [
    ["主城", "home"],
    ["建造", "build"],
    ["弟子", "disciples"],
    ["神降", "descent"],
  ];

  return `
    <div class="bottom-dock">
      ${items
        .map(
          ([label, icon]) => `
            <div class="dock-item ${label === normalizedActive ? "is-active" : ""}">
              <span class="dock-item__icon dock-item__icon--${icon}">
                <span class="dock-glyph"></span>
              </span>
              <span class="dock-item__label">${label}</span>
            </div>
          `,
        )
        .join("")}
    </div>
  `;
}

function buildingCard(title, description, costs, status) {
  return `
    <div class="building-card">
      <div class="building-thumb">
        <div class="roof-silhouette small"></div>
      </div>
      <div class="building-title">
        <strong>${title}</strong>
        <span class="pill good">${status}</span>
      </div>
      <p class="building-meta">${description}</p>
      <div class="cost-row">
        ${costs.map((cost) => `<span class="pill warn">${cost}</span>`).join("")}
      </div>
    </div>
  `;
}

function buildLibraryCard(title, subtitle, status) {
  return `
    <div class="build-library-card">
      <div class="build-library-card__art">
        <div class="roof-silhouette small"></div>
      </div>
      <div class="build-library-card__copy">
        <strong>${title}</strong>
        <p>${subtitle}</p>
        <span class="pill good">${status}</span>
      </div>
    </div>
  `;
}

function attributeRow(label, value, tail) {
  return `
    <div class="attribute-row">
      <span>${label}</span>
      <div class="bar"><span style="--fill: ${value}%"></span></div>
      <strong>${tail}</strong>
    </div>
  `;
}

function supplyItem(label, value, status, percent) {
  return `
    <div class="supply-item">
      <span>${label}</span>
      <div class="bar"><span style="--fill: ${percent}%"></span></div>
      <strong>${value}</strong>
    </div>
    <p class="small-copy">${status}</p>
  `;
}

function flowLink(from, to) {
  return `
    <div class="flow-link">
      <div class="flow-node"><span class="chip-icon">源</span><strong>${from}</strong></div>
      <div class="flow-arrow"></div>
      <div class="flow-node"><span class="chip-icon">流</span><strong>${to}</strong></div>
    </div>
  `;
}

function queueCard(title, time, ingredients, caption) {
  return `
    <div class="queue-card">
      <div class="craft-art">
        <div class="roof-silhouette small"></div>
      </div>
      <h4>${title}</h4>
      <p>${time}</p>
      <div class="cost-row">
        ${ingredients.split("·").map((item) => `<span class="pill">${item.trim()}</span>`).join("")}
      </div>
      <p>${caption}</p>
    </div>
  `;
}

function inventoryCard(title, amount, caption) {
  return `
    <div class="inventory-card">
      <div class="inventory-title"><strong>${title}</strong><span class="pill good">${amount}</span></div>
      <p class="inventory-meta">${caption}</p>
    </div>
  `;
}

function choiceCard(index, title, description, gain, risk) {
  return `
    <div class="choice-card">
      <span class="choice-index">${index}</span>
      <div>
        <h4>${title}</h4>
        <p>${description}</p>
        <div class="choice-foot">
          <span class="pill good">${gain}</span>
          <span class="pill warn">${risk}</span>
        </div>
      </div>
    </div>
  `;
}

function taskStage(step, title, status, progress) {
  return `
    <div class="task-card">
      <div class="task-step">${step}</div>
      <div>
        <h4>${title}</h4>
        <p>${status}</p>
        <div class="progress-bar"><span style="--fill: ${progress}%"></span></div>
      </div>
    </div>
  `;
}

function assignmentRow(label, need, assigned) {
  return `
    <div class="assignment-row">
      <strong>${label}</strong>
      <div class="assignment-slots">
        ${new Array(assigned).fill("<span></span>").join("")}
        ${new Array(Math.max(need - assigned, 0)).fill("<span class='is-empty'></span>").join("")}
      </div>
      <span class="pill ${assigned < need ? "warn" : "good"}">${assigned}/${need}</span>
    </div>
  `;
}

function miniDisciple(name, meta) {
  return `
    <div class="disciple-card">
      <div class="portrait"></div>
      <div>
        <div class="disciple-title"><strong>${name}</strong></div>
        <p class="disciple-meta">${meta}</p>
      </div>
      <span class="status-chip">可调度</span>
    </div>
  `;
}

function rosterTile(name, role, meta, status) {
  return `
    <div class="roster-tile">
      <div class="roster-tile__head">
        <div class="portrait"></div>
        <div>
          <strong>${name}</strong>
          <span>${role}</span>
        </div>
      </div>
      <p>${meta}</p>
      <span class="pill">${status}</span>
    </div>
  `;
}

function artifactCard(title, meta) {
  return `
    <div class="artifact-card">
      <div class="artifact-title"><strong>${title}</strong></div>
      <p class="artifact-meta">${meta}</p>
    </div>
  `;
}

function oracleChoice(index, title, description, gain, cost, tone) {
  return `
    <div class="oracle-choice ${tone}">
      <div class="oracle-choice__head">
        <span class="oracle-choice__index">${index}</span>
        <div>
          <strong>${title}</strong>
          <p>${description}</p>
        </div>
      </div>
      <div class="oracle-choice__foot">
        <span class="pill good">${gain}</span>
        <span class="pill warn">${cost}</span>
      </div>
    </div>
  `;
}

function stateItem(title, state, description) {
  return `
    <div class="state-item">
      <strong>${title}</strong>
      <span class="pill good">${state}</span>
      <span></span>
    </div>
    <p class="small-copy">${description}</p>
  `;
}

init();
