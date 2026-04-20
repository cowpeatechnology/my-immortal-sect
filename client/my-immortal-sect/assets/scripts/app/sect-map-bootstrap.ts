import {
    _decorator,
    Color,
    Component,
    EventTouch,
    Graphics,
    Input,
    Label,
    Node,
    resources,
    Sprite,
    SpriteFrame,
    TiledLayer,
    TiledMap,
    UITransform,
    Vec3,
    director,
    input,
    v3,
    view,
} from 'cc';
import {
    createLocalUnitModel,
    getActionDurationSeconds,
    getMitigatedDamage,
    type LocalUnitModel,
} from './local-unit-model';
import { getRuntimeScreenMetrics } from './runtime-screen-metrics';
import type { RuntimeScreenMetrics } from './runtime-screen-metrics';
import { SectMapAuthorityClient } from '../net/sect-map-authority-client';
import type {
    AuthorityBuildingSnapshot,
    AuthorityBuildingType,
    AuthorityCommandEnvelope,
    AuthorityResourceKind,
    AuthoritySessionOutcome,
    AuthoritySessionPhase,
    AuthoritySessionResponse,
    AuthoritySnapshot,
    AuthorityStockpile,
} from '../net/sect-map-authority-contract';

const { ccclass } = _decorator;

// Keep disciple and hostile balance math aligned inside the local-only map loop.

const MAP_ROOT = 'MapRoot';
const GROUND_ROOT = 'GroundMap';
const RESOURCE_ROOT = 'ResourceRoot';
const BUILDING_ROOT = 'BuildingRoot';
const TOKEN_ROOT = 'TokenRoot';
const OVERLAY_ROOT = 'OverlayRoot';
const SELECTION_CURSOR = 'SelectionCursor';
const BLUEPRINT_PREVIEW = 'BlueprintPreview';
const OBJECTIVE_MARKER = 'ObjectiveMarker';
const STATUS_LABEL = 'StatusLabel';
const HUD_ROOT = 'HudRoot';
const TOOLBAR_ROOT = 'ToolbarRoot';
const BUILD_PANEL_ROOT = 'BuildPanelRoot';
const RADIAL_MENU_ROOT = 'RadialMenuRoot';

const DRAG_THRESHOLD = 12;
const LONG_PRESS_SECONDS = 0.48;
const MOUSE_POINTER_ID = -1;
const BASE_HARVEST_SECONDS = 0.85;
const BASE_HAUL_SECONDS = 0.55;
const BASE_BUILD_SECONDS = 1.6;
const BASE_REPAIR_SECONDS = 1.2;
const DEMOLISH_SECONDS = 1.0;
const RADIAL_RADIUS = 92;
const HOSTILE_RESPAWN_SECONDS = 9;
const BUILDING_HIT_FLASH_SECONDS = 0.42;
const UNIT_HIT_FLASH_SECONDS = 0.3;
const SESSION_TARGET_SECONDS = 600;
const FIRST_RAID_PREP_SECONDS = 24;
const RUIN_WAREHOUSE_PREFERRED_TILE: TileCoord = { col: 8, row: 8 };

const SECT_MAP_VISUAL_ASSET_PATHS = {
    'building.main_hall': 'generated-buildings/sect-map-raster/main_hall/spriteFrame',
    'building.disciple_quarters': 'generated-buildings/sect-map-raster/disciple_quarters/spriteFrame',
    'building.warehouse': 'generated-buildings/sect-map-raster/warehouse/spriteFrame',
    'building.herb_garden': 'generated-buildings/sect-map-raster/herb_garden/spriteFrame',
    'building.guard_tower': 'generated-buildings/sect-map-raster/guard_tower/spriteFrame',
    'hostile.normal': 'generated-buildings/sect-map-raster/bandit_scout_normal/spriteFrame',
    'hostile.injured': 'generated-buildings/sect-map-raster/bandit_scout_injured/spriteFrame',
    'resource.spirit_wood': 'generated-buildings/sect-map-raster/spirit_wood/spriteFrame',
    'resource.spirit_stone': 'generated-buildings/sect-map-raster/spirit_stone/spriteFrame',
    'resource.herb': 'generated-buildings/sect-map-raster/herb/spriteFrame',
    'disciple.normal': 'generated-buildings/sect-map-raster/sect_disciple_normal/spriteFrame',
    'disciple.injured': 'generated-buildings/sect-map-raster/sect_disciple_injured/spriteFrame',
    'disciple.dying': 'generated-buildings/sect-map-raster/sect_disciple_dying/spriteFrame',
    'disciple.dead': 'generated-buildings/sect-map-raster/sect_disciple_dead/spriteFrame',
    'signal.planned': 'generated-buildings/sect-map-raster/building_signal_planned/spriteFrame',
    'signal.supplied': 'generated-buildings/sect-map-raster/building_signal_supplied/spriteFrame',
    'signal.constructing': 'generated-buildings/sect-map-raster/building_signal_constructing/spriteFrame',
    'signal.damaged': 'generated-buildings/sect-map-raster/building_signal_damaged/spriteFrame',
    'signal.disabled': 'generated-buildings/sect-map-raster/building_signal_disabled/spriteFrame',
} as const;

type TileCoord = {
    col: number;
    row: number;
};

type ResourceKind = 'spirit_wood' | 'spirit_stone' | 'herb';
type ResourceNodeState = 'available' | 'regenerating';
type InputMode = 'browse' | 'gather' | 'build_select' | 'build_place' | 'demolish';
type BuildingState = 'planned' | 'supplied' | 'constructing' | 'active' | 'damaged';
type BuildingWorkKind = 'build' | 'upgrade';
type UnitVisualState = 'idle' | 'moving' | 'working' | 'carrying' | 'guarding' | 'attacking' | 'injured';
type SectMapVisualAssetId = keyof typeof SECT_MAP_VISUAL_ASSET_PATHS;
type SessionPhase = 'clear_ruin' | 'place_guard_tower' | 'upgrade_guard_tower' | 'raid_countdown' | 'defend' | 'recover' | 'victory' | 'defeat';
type SessionOutcome = 'in_progress' | 'victory' | 'defeat';

type Stockpile = Record<ResourceKind, number>;

type ResourceRule = {
    maxCharges: number;
    regenSeconds: number;
};

type GuardProfile = {
    attackPower: number;
    rangeTiles: number;
    attackInterval: number;
};

type BuildingDefinition = {
    id: string;
    label: string;
    width: number;
    height: number;
    cost: Stockpile;
    activeColor: Color;
    maxHp: number;
    repairCost: Stockpile;
    structureDefense: number;
    guardProfile?: GuardProfile;
};

type BuildingEntity = {
    id: string;
    definition: BuildingDefinition;
    origin: TileCoord;
    state: BuildingState;
    supplied: Stockpile;
    level: number;
    markedForDemolition: boolean;
    currentHp: number;
    damageFlashSeconds: number;
    attackCooldownSeconds: number;
    pendingAction: BuildingWorkKind | null;
    pendingLevel: number | null;
};

type ResourceNode = {
    kind: ResourceKind;
    tile: TileCoord;
    designated: boolean;
    state: ResourceNodeState;
    remainingCharges: number;
    maxCharges: number;
    regenSeconds: number;
    regenTimerSeconds: number;
};

type BuildPlacement = {
    definition: BuildingDefinition;
    origin: TileCoord;
};

type DiscipleTask =
    | {
          kind: 'gather';
          resourceTile: TileCoord;
          resourceKind: ResourceKind;
          phase: 'move-to-resource' | 'harvest' | 'move-to-dropoff' | 'dropoff';
          timer: number;
          targetTile: TileCoord;
      }
    | {
          kind: 'haul';
          buildingId: string;
          resourceKind: ResourceKind;
          phase: 'move-to-site' | 'deliver';
          timer: number;
          targetTile: TileCoord;
      }
    | {
          kind: 'build';
          buildingId: string;
          phase: 'move-to-site' | 'construct';
          timer: number;
          targetTile: TileCoord;
      }
    | {
          kind: 'demolish';
          buildingId: string;
          phase: 'move-to-site' | 'demolish';
          timer: number;
          targetTile: TileCoord;
      }
    | {
          kind: 'repair';
          buildingId: string;
          phase: 'move-to-site' | 'repair';
          timer: number;
          targetTile: TileCoord;
      }
    | {
          kind: 'guard';
          hostileId: string;
          phase: 'move-to-site' | 'attack';
          timer: number;
          targetTile: TileCoord;
      };

type PathingUnit = {
    tile: TileCoord;
    worldPosition: Vec3;
    path: TileCoord[];
    pathIndex: number;
    model: LocalUnitModel;
};

type DiscipleEntity = {
    id: string;
    name: string;
    model: LocalUnitModel;
    tile: TileCoord;
    worldPosition: Vec3;
    visualState: UnitVisualState;
    carrying: ResourceKind | null;
    currentTask: DiscipleTask | null;
    path: TileCoord[];
    pathIndex: number;
    node: Node | null;
    currentHp: number;
    attackCooldownSeconds: number;
    hitFlashSeconds: number;
};

type HostileNpcEntity = {
    id: string;
    name: string;
    model: LocalUnitModel;
    tile: TileCoord;
    worldPosition: Vec3;
    visualState: UnitVisualState;
    path: TileCoord[];
    pathIndex: number;
    node: Node | null;
    currentHp: number;
    attackCooldownSeconds: number;
    hitFlashSeconds: number;
    active: boolean;
    respawnTimerSeconds: number;
    targetBuildingId: string | null;
};

type ToolbarButtonConfig = {
    key: 'browse' | 'gather' | 'build' | 'demolish';
    label: string;
    positionX: number;
};

type RadialAction = {
    key: string;
    label: string;
    execute: () => void;
};

type RuntimeLogLevel = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';
type RuntimeLogChannel = 'BOOT' | 'INPUT' | 'MODE' | 'MAP' | 'BUILD' | 'TASK' | 'RESOURCE' | 'HUD' | 'COMBAT';
type RuntimeLogPayload = Record<string, unknown>;

type RuntimeLogEntry = {
    seq: number;
    timestamp: string;
    elapsedMs: number;
    level: RuntimeLogLevel;
    channel: RuntimeLogChannel;
    event: string;
    message: string;
    payload?: RuntimeLogPayload;
};

type RuntimeViewportSnapshot = {
    designWidth: number;
    designHeight: number;
    visibleWidth: number;
    visibleHeight: number;
};

type RuntimeMapContractSnapshot = {
    columns: number;
    rows: number;
    tileWidth: number;
    tileHeight: number;
    roadTiles: number;
    blockedTiles: number;
    resourceTiles: number;
    buildableTiles: number;
    mainHallTile: string;
    discipleTile: string;
};

type RuntimeResourceSummary = {
    designatedCount: number;
    availableCount: number;
    regeneratingCount: number;
    totalRemainingCharges: number;
    byKind: Record<
        ResourceKind,
        {
            designatedNodes: number;
            availableNodes: number;
            regeneratingNodes: number;
            remainingCharges: number;
        }
    >;
};

type RuntimeResourceNodeSnapshot = {
    tile: string;
    kind: ResourceKind;
    designated: boolean;
    state: ResourceNodeState;
    remainingCharges: number;
    maxCharges: number;
    regenSeconds: number;
    regenTimerSeconds: number;
};

type RuntimeBuildingSnapshot = {
    id: string;
    type: string;
    origin: string;
    state: BuildingState;
    level: number;
    hp: number;
    maxHp: number;
    markedForDemolition: boolean;
    pendingAction: BuildingWorkKind | null;
};

type RuntimeHostileSnapshot = {
    id: string;
    archetypeId: string;
    tile: string;
    visualState: UnitVisualState;
    hp: number;
    maxHp: number;
    active: boolean;
    targetBuildingId: string | null;
};

type RuntimeSafeInsets = {
    top: number;
    right: number;
    bottom: number;
    left: number;
};

type RuntimePortraitBaselineSnapshot = {
    safeInsets: RuntimeSafeInsets;
    safeFrameWidth: number;
    safeFrameHeight: number;
    metricsSource: RuntimeScreenMetrics['source'];
    safeAreaFallbackApplied: boolean;
    hud: {
        centerX: number;
        horizontalPaddingLeft: number;
        horizontalPaddingRight: number;
        topAnchorY: number;
        bottomAnchorY: number;
        statusY: number;
        toolbarY: number;
        buildPanelY: number;
    };
};

type ObjectiveMarkerTone = 'goal' | 'alert' | 'success' | 'failure';

type SessionGuidance = {
    headline: string;
    detail: string;
    threat: string;
    markerText: string | null;
    markerTile: TileCoord | null;
    markerPosition: Vec3 | null;
    markerTone: ObjectiveMarkerTone;
    focusBuildingId: string | null;
};

type RuntimeSnapshot = {
    mapReady: boolean;
    inputMode: InputMode;
    lastMessage: string;
    selectedTile: string | null;
    viewport: RuntimeViewportSnapshot;
    mapContract: RuntimeMapContractSnapshot;
    screenMetrics: RuntimeScreenMetrics;
    screenMetricsFallback: RuntimeScreenMetrics;
    portraitBaseline: RuntimePortraitBaselineSnapshot | null;
    stockpile: Stockpile;
    resourceSummary: RuntimeResourceSummary;
    resourceNodes: RuntimeResourceNodeSnapshot[];
    designatedResources: number;
    buildingCounts: Record<BuildingState, number>;
    buildings: RuntimeBuildingSnapshot[];
    session: {
        phase: SessionPhase;
        outcome: SessionOutcome;
        elapsedSeconds: number;
        limitSeconds: number;
        objective: string;
        guidanceHeadline: string;
        guidanceDetail: string;
        guidanceThreat: string;
        markerTile: string | null;
        markerText: string | null;
        focusBuildingId: string | null;
        raidCountdownSeconds: number;
        guardTowerId: string | null;
        ruinBuildingId: string | null;
        firstRaidTriggered: boolean;
        firstRaidResolved: boolean;
    };
    disciple: {
        tile: string;
        visualState: UnitVisualState;
        carrying: ResourceKind | null;
        task: string | null;
        hp: number;
        maxHp: number;
        model: LocalUnitModel;
    };
    hostiles: RuntimeHostileSnapshot[];
    authority: {
        mode: 'authority' | 'local_fallback';
        connected: boolean;
        sessionId: string;
        baseUrl: string;
        pendingCommands: string[];
        lastEvent: string | null;
        lastError: string | null;
    };
};

type RuntimeDebugBridge = {
    getRecentLogs: (limit?: number) => RuntimeLogEntry[];
    clearLogs: () => void;
    getSnapshot: () => RuntimeSnapshot;
};

type RuntimeDebugGlobal = typeof globalThis & {
    __MIS_RUNTIME_DEBUG__?: RuntimeDebugBridge;
};

const EMPTY_STOCKPILE = (): Stockpile => ({
    spirit_wood: 0,
    spirit_stone: 0,
    herb: 0,
});

const INITIAL_MAIN_HALL_TILE: TileCoord = { col: 6, row: 5 };
const INITIAL_DISCIPLE_TILE: TileCoord = { col: 5, row: 6 };

const RESOURCE_DISPLAY: Record<ResourceKind, { short: string; color: Color; title: string }> = {
    spirit_wood: {
        short: '木',
        color: new Color(76, 160, 92, 255),
        title: '灵木',
    },
    spirit_stone: {
        short: '石',
        color: new Color(123, 138, 170, 255),
        title: '灵石',
    },
    herb: {
        short: '药',
        color: new Color(118, 208, 122, 255),
        title: '药草',
    },
};

const RESOURCE_RULES: Record<ResourceKind, ResourceRule> = {
    spirit_wood: {
        maxCharges: 3,
        regenSeconds: 9,
    },
    spirit_stone: {
        maxCharges: 2,
        regenSeconds: 12,
    },
    herb: {
        maxCharges: 3,
        regenSeconds: 8,
    },
};

const BUILDING_DEFINITIONS: BuildingDefinition[] = [
    {
        id: 'main_hall',
        label: '主殿',
        width: 3,
        height: 3,
        cost: {
            spirit_wood: 0,
            spirit_stone: 0,
            herb: 0,
        },
        activeColor: new Color(186, 126, 72, 220),
        maxHp: 28,
        repairCost: {
            spirit_wood: 2,
            spirit_stone: 1,
            herb: 0,
        },
        structureDefense: 1.6,
    },
    {
        id: 'disciple_quarters',
        label: '弟子居',
        width: 2,
        height: 3,
        cost: {
            spirit_wood: 2,
            spirit_stone: 1,
            herb: 0,
        },
        activeColor: new Color(92, 144, 210, 220),
        maxHp: 16,
        repairCost: {
            spirit_wood: 1,
            spirit_stone: 1,
            herb: 0,
        },
        structureDefense: 1.1,
    },
    {
        id: 'warehouse',
        label: '仓库',
        width: 2,
        height: 2,
        cost: {
            spirit_wood: 2,
            spirit_stone: 2,
            herb: 0,
        },
        activeColor: new Color(198, 156, 78, 220),
        maxHp: 20,
        repairCost: {
            spirit_wood: 1,
            spirit_stone: 1,
            herb: 0,
        },
        structureDefense: 1.3,
    },
    {
        id: 'herb_garden',
        label: '药圃',
        width: 2,
        height: 2,
        cost: {
            spirit_wood: 1,
            spirit_stone: 0,
            herb: 1,
        },
        activeColor: new Color(82, 176, 114, 220),
        maxHp: 14,
        repairCost: {
            spirit_wood: 1,
            spirit_stone: 0,
            herb: 1,
        },
        structureDefense: 0.9,
    },
    {
        id: 'guard_tower',
        label: '护山台',
        width: 1,
        height: 2,
        cost: {
            spirit_wood: 1,
            spirit_stone: 2,
            herb: 0,
        },
        activeColor: new Color(176, 92, 96, 220),
        maxHp: 18,
        repairCost: {
            spirit_wood: 1,
            spirit_stone: 1,
            herb: 0,
        },
        structureDefense: 1.4,
        guardProfile: {
            attackPower: 4.4,
            rangeTiles: 4,
            attackInterval: 1.2,
        },
    },
];

const TOOLBAR_BUTTONS: ToolbarButtonConfig[] = [
    { key: 'browse', label: '查看', positionX: -240 },
    { key: 'gather', label: '采集', positionX: -80 },
    { key: 'build', label: '建造', positionX: 80 },
    { key: 'demolish', label: '拆除', positionX: 240 },
];

@ccclass('SectMapBootstrap')
export class SectMapBootstrap extends Component {
    private readonly runtimeSessionStartedAt = Date.now();
    private readonly runtimeLogLimit = 240;
    private canvas: Node | null = null;
    private mapRoot!: Node;
    private groundRoot!: Node;
    private resourceRoot!: Node;
    private buildingRoot!: Node;
    private tokenRoot!: Node;
    private overlayRoot!: Node;
    private blueprintPreview!: Node;
    private selectionCursor!: Node;
    private objectiveMarker!: Node;
    private hudRoot: Node | null = null;
    private toolbarRoot: Node | null = null;
    private buildPanelRoot: Node | null = null;
    private radialMenuRoot: Node | null = null;
    private statusLabel: Label | null = null;

    private groundLayer: TiledLayer | null = null;
    private roadLayer: TiledLayer | null = null;
    private blockedLayer: TiledLayer | null = null;
    private resourceLayers = new Map<ResourceKind, TiledLayer>();

    private tileCenters = new Map<string, Vec3>();
    private roadTiles = new Set<string>();
    private blockedTiles = new Set<string>();
    private resourceNodes = new Map<string, ResourceNode>();
    private buildingEntities = new Map<string, BuildingEntity>();
    private visualAssetFrames: Partial<Record<SectMapVisualAssetId, SpriteFrame>> = {};
    private toolbarButtons = new Map<string, Node>();
    private buildButtons = new Map<string, Node>();

    private selectedTile: TileCoord | null = null;
    private buildPlacement: BuildPlacement | null = null;
    private preferredBuildTile: TileCoord | null = null;

    private mapColumns = 0;
    private mapRows = 0;
    private tileWidth = 0;
    private tileHeight = 0;
    private halfTileWidth = 0;
    private halfTileHeight = 0;
    private mapPixelWidth = 0;
    private mapPixelHeight = 0;

    private stockpile: Stockpile = EMPTY_STOCKPILE();
    private mapReady = false;
    private inputMode: InputMode = 'browse';
    private lastMessage = '地图加载中';
    private nextBuildingId = 1;
    private runtimeLogSeq = 0;
    private runtimeLogs: RuntimeLogEntry[] = [];
    private viewportLayoutSignature = '';
    private portraitBaselineSnapshot: RuntimePortraitBaselineSnapshot | null = null;
    private buildableTileCount = 0;
    private sessionPhase: SessionPhase = 'clear_ruin';
    private sessionOutcome: SessionOutcome = 'in_progress';
    private sessionElapsedSeconds = 0;
    private sessionObjectiveText = '';
    private sessionGuardTowerId: string | null = null;
    private sessionRuinBuildingId: string | null = null;
    private firstRaidTriggered = false;
    private firstRaidResolved = false;
    private sessionBuildPanelPrompted = false;
    private authorityClient = new SectMapAuthorityClient();
    private authoritySessionId = 'preview-local';
    private authorityConnected = false;
    private authorityMode: 'authority' | 'local_fallback' = 'local_fallback';
    private authorityHydratingSnapshot = false;
    private authorityPendingCommands = new Set<string>();
    private authorityLastEvent: string | null = null;
    private authorityLastError: string | null = null;

    private activePointerId: number | null = null;
    private pointerStart = v3();
    private pointerCurrent = v3();
    private pointerHoldSeconds = 0;
    private pointerStartTile: TileCoord | null = null;
    private pointerDownOnBlueprint = false;
    private draggingMap = false;
    private draggingBlueprint = false;
    private longPressTriggered = false;

    private disciple: DiscipleEntity = {
        id: 'disciple-01',
        name: '玄',
        model: createLocalUnitModel('sect_disciple', '玄'),
        tile: { col: 5, row: 6 },
        worldPosition: v3(),
        visualState: 'idle',
        carrying: null,
        currentTask: null,
        path: [],
        pathIndex: 0,
        node: null,
        currentHp: createLocalUnitModel('sect_disciple', '玄').stats.maxHp,
        attackCooldownSeconds: 0,
        hitFlashSeconds: 0,
    };

    private hostileNpc: HostileNpcEntity = {
        id: 'npc-bandit-01',
        name: '寇',
        model: createLocalUnitModel('bandit_scout', '寇'),
        tile: { col: 0, row: 0 },
        worldPosition: v3(),
        visualState: 'idle',
        path: [],
        pathIndex: 0,
        node: null,
        currentHp: createLocalUnitModel('bandit_scout', '寇').stats.maxHp,
        attackCooldownSeconds: 0,
        hitFlashSeconds: 0,
        active: false,
        respawnTimerSeconds: HOSTILE_RESPAWN_SECONDS,
        targetBuildingId: null,
    };

    onLoad(): void {
        const scene = director.getScene();
        this.canvas = scene?.getChildByName('Canvas') ?? null;
        this.statusLabel = this.canvas?.getChildByName(STATUS_LABEL)?.getComponent(Label) ?? null;
        this.attachRuntimeDebugBridge();

        this.mapRoot = this.ensureChild(MAP_ROOT);
        this.groundRoot = this.resolveGroundRoot();
        this.resourceRoot = this.ensureChild(RESOURCE_ROOT, this.mapRoot);
        this.buildingRoot = this.ensureChild(BUILDING_ROOT, this.mapRoot);
        this.tokenRoot = this.ensureChild(TOKEN_ROOT, this.mapRoot);
        this.overlayRoot = this.ensureChild(OVERLAY_ROOT, this.mapRoot);
        this.selectionCursor = this.ensureChild(SELECTION_CURSOR, this.overlayRoot);
        this.blueprintPreview = this.ensureChild(BLUEPRINT_PREVIEW, this.overlayRoot);
        this.objectiveMarker = this.ensureChild(OBJECTIVE_MARKER, this.overlayRoot);

        this.groundRoot.setSiblingIndex(0);
        this.resourceRoot.setSiblingIndex(1);
        this.buildingRoot.setSiblingIndex(2);
        this.tokenRoot.setSiblingIndex(3);
        this.overlayRoot.setSiblingIndex(4);

        this.selectionCursor.active = false;
        this.blueprintPreview.active = false;
        this.objectiveMarker.active = false;

        this.configureNodeSize(this.node, view.getVisibleSize().width, view.getVisibleSize().height);
        this.configureNodeSize(this.mapRoot, view.getVisibleSize().width, view.getVisibleSize().height);
        this.configureNodeSize(this.overlayRoot, view.getVisibleSize().width, view.getVisibleSize().height);

        if (this.statusLabel) {
            this.statusLabel.fontSize = 17;
            this.statusLabel.lineHeight = 22;
            this.statusLabel.enableWrapText = true;
            this.statusLabel.color = new Color(246, 248, 252, 255);
        }

        this.ensureHudRoots();
        this.refreshViewportLayout('bootstrap.on_load');
        view.on('canvas-resize', this.onCanvasResize, this);
        view.on('design-resolution-changed', this.onCanvasResize, this);
        this.logRuntime('INFO', 'BOOT', 'bootstrap.loaded', 'SectMapBootstrap 已完成节点装配', {
            sceneName: scene?.name ?? 'unknown',
            canvasReady: !!this.canvas,
            statusLabelReady: !!this.statusLabel,
        });
        this.renderStatus();
    }

    private resolveGroundRoot(): Node {
        const existingInMapRoot = this.mapRoot.getChildByName(GROUND_ROOT);
        const existingSibling = this.node.getChildByName(GROUND_ROOT);
        const candidates = [existingInMapRoot, existingSibling].filter((node): node is Node => !!node);
        const tiledGroundRoot = candidates.find((node) => !!node.getComponent(TiledMap));

        if (tiledGroundRoot) {
            if (existingInMapRoot && existingInMapRoot !== tiledGroundRoot && !existingInMapRoot.getComponent(TiledMap)) {
                existingInMapRoot.destroy();
            }
            if (tiledGroundRoot.parent !== this.mapRoot) {
                tiledGroundRoot.setParent(this.mapRoot);
            }
            tiledGroundRoot.setSiblingIndex(0);
            return tiledGroundRoot;
        }

        if (existingInMapRoot) {
            return existingInMapRoot;
        }

        if (existingSibling) {
            existingSibling.setParent(this.mapRoot);
            existingSibling.setSiblingIndex(0);
            return existingSibling;
        }

        return this.ensureChild(GROUND_ROOT, this.mapRoot);
    }

    start(): void {
        const tiledMap = this.groundRoot.getComponent(TiledMap);
        if (!tiledMap?.tmxAsset) {
            this.logRuntime('ERROR', 'BOOT', 'bootstrap.tmx_missing', 'GroundMap 缺少 Tmx Asset', {
                groundRoot: this.groundRoot.name,
            });
            this.setMessage('GroundMap 缺少 Tmx Asset，无法初始化 M1 地图底座');
            return;
        }

        this.groundLayer = tiledMap.getLayer('ground') ?? null;
        this.roadLayer = tiledMap.getLayer('road') ?? null;
        this.blockedLayer = tiledMap.getLayer('blocked') ?? null;
        this.resourceLayers.set('spirit_wood', tiledMap.getLayer('resource_wood') ?? null!);
        this.resourceLayers.set('spirit_stone', tiledMap.getLayer('resource_stone') ?? null!);
        this.resourceLayers.set('herb', tiledMap.getLayer('resource_herb') ?? null!);

        if (!this.groundLayer) {
            this.logRuntime('ERROR', 'BOOT', 'bootstrap.ground_layer_missing', '缺少 ground 图层，无法读取 TileMap');
            this.setMessage('缺少 ground 图层，无法读取 TileMap');
            return;
        }

        const mapSize = tiledMap.getMapSize();
        const tileSize = tiledMap.getTileSize();

        this.mapColumns = mapSize.width;
        this.mapRows = mapSize.height;
        this.tileWidth = tileSize.width;
        this.tileHeight = tileSize.height;
        this.halfTileWidth = this.tileWidth * 0.5;
        this.halfTileHeight = this.tileHeight * 0.5;
        this.mapPixelWidth = (this.mapColumns + this.mapRows) * this.halfTileWidth;
        this.mapPixelHeight = (this.mapColumns + this.mapRows) * this.halfTileHeight;

        this.configureNodeSize(this.mapRoot, this.mapPixelWidth, this.mapPixelHeight);
        this.configureNodeSize(this.groundRoot, this.mapPixelWidth, this.mapPixelHeight);
        this.configureNodeSize(this.resourceRoot, this.mapPixelWidth, this.mapPixelHeight);
        this.configureNodeSize(this.buildingRoot, this.mapPixelWidth, this.mapPixelHeight);
        this.configureNodeSize(this.tokenRoot, this.mapPixelWidth, this.mapPixelHeight);
        this.configureNodeSize(this.overlayRoot, this.mapPixelWidth, this.mapPixelHeight);
        this.mapRoot.setPosition(0, 0, 0);

        this.cacheTileCenters();
        this.readLogicLayers();
        this.drawSelectionCursor();
        this.loadVisualAssets();
        this.spawnInitialWorld();
        this.buildToolbar();
        this.buildBuildPanel();
        this.registerInput();

        const discipleCenter = this.getTileCenter(this.disciple.tile);
        this.disciple.worldPosition.set(discipleCenter.x, discipleCenter.y, 0);
        this.refreshResourceMarkers();
        this.refreshBuildings();
        this.refreshDiscipleToken();
        this.refreshHostileNpcToken();

        this.mapReady = true;
        this.logRuntime('INFO', 'BOOT', 'bootstrap.ready', 'M1 地图底座初始化完成', {
            columns: this.mapColumns,
            rows: this.mapRows,
            tileWidth: this.tileWidth,
            tileHeight: this.tileHeight,
            resourceCount: this.resourceNodes.size,
            roadCount: this.roadTiles.size,
            blockedCount: this.blockedTiles.size,
        });
        this.setMessage('M1 地图底座已加载：拖图、双指缩放预留、工具栏标记、长按环形快捷操作已启用');
        void this.bootstrapAuthoritySession();
    }

    private async bootstrapAuthoritySession(): Promise<void> {
        try {
            const response = await this.authorityClient.bootstrapSession(this.authoritySessionId);
            this.applyAuthorityResponse(response, 'bootstrap');
            this.authorityConnected = true;
            this.authorityMode = 'authority';
            this.authorityLastError = null;
            this.logRuntime('INFO', 'BOOT', 'authority.bootstrap_ready', 'M1 authority snapshot 已接入当前预览会话', {
                sessionId: this.authoritySessionId,
                baseUrl: this.authorityClient.getBaseUrl(),
            });
            this.setMessage('M1 authority 短会话已接管建造与资源结算路径');
        } catch (error) {
            const message = error instanceof Error ? error.message : 'authority_bootstrap_failed';
            this.authorityConnected = false;
            this.authorityMode = 'local_fallback';
            this.authorityLastError = message;
            this.logRuntime('WARN', 'BOOT', 'authority.bootstrap_failed', 'authority server 不可用，当前预览回退到本地会话', {
                sessionId: this.authoritySessionId,
                baseUrl: this.authorityClient.getBaseUrl(),
                error: message,
            });
            this.setMessage(`authority 未连接，当前回退为本地预览：${message}`);
        }
    }

    private applyAuthorityResponse(response: AuthoritySessionResponse, reason: string): void {
        this.applyAuthoritySnapshot(response.snapshot, reason);
        if (response.result) {
            this.authorityLastEvent = response.result.event;
            this.authorityLastError = null;
            if (response.result.message) {
                this.setMessage(response.result.message);
            }
        }
    }

    private applyAuthoritySnapshot(snapshot: AuthoritySnapshot, reason: string): void {
        this.authorityHydratingSnapshot = true;
        try {
            this.authoritySessionId = snapshot.sessionId;
            this.stockpile = this.fromAuthorityStockpile(snapshot.stockpile);
            this.buildingEntities.clear();

            let highestBuildingNumericID = 0;
            for (const snapshotBuilding of snapshot.buildings) {
                const entity = this.buildEntityFromAuthority(snapshotBuilding);
                this.buildingEntities.set(entity.id, entity);
                const numericID = Number(entity.id.replace('building-', ''));
                if (Number.isFinite(numericID)) {
                    highestBuildingNumericID = Math.max(highestBuildingNumericID, numericID);
                }
            }
            this.nextBuildingId = highestBuildingNumericID + 1;

            this.sessionPhase = snapshot.session.phase as SessionPhase;
            this.sessionOutcome = snapshot.session.outcome as SessionOutcome;
            this.sessionObjectiveText = snapshot.session.objective;
            this.sessionGuardTowerId = snapshot.session.guardTowerId;
            this.sessionRuinBuildingId = snapshot.session.ruinBuildingId;
            this.firstRaidTriggered = snapshot.session.firstRaidTriggered;
            this.firstRaidResolved = snapshot.session.firstRaidResolved;

            if (this.sessionPhase === 'place_guard_tower' && !this.sessionBuildPanelPrompted) {
                this.sessionBuildPanelPrompted = true;
                const focusTile =
                    this.getSessionRuinBuilding()?.origin ??
                    this.findNearestBuildableTile(RUIN_WAREHOUSE_PREFERRED_TILE) ??
                    this.findScreenCenterTile();
                if (focusTile) {
                    this.openBuildPanel(focusTile);
                }
            }

            if (this.disciple.currentTask) {
                switch (this.disciple.currentTask.kind) {
                    case 'build':
                    case 'haul':
                    case 'repair':
                    case 'demolish':
                        if (!this.buildingEntities.has(this.disciple.currentTask.buildingId)) {
                            this.disciple.currentTask = null;
                            this.disciple.path = [];
                            this.disciple.pathIndex = 0;
                            this.disciple.carrying = null;
                        }
                        break;
                    default:
                        break;
                }
            }

            this.refreshBuildings();
            this.refreshObjectiveMarker();
            this.renderStatus();
            this.logRuntime('INFO', 'BOOT', 'authority.snapshot_applied', 'authority snapshot 已写回当前 runtime', {
                reason,
                buildingCount: this.buildingEntities.size,
                stockpile: this.summarizeStockpile(),
                phase: this.sessionPhase,
                outcome: this.sessionOutcome,
            });
        } finally {
            this.authorityHydratingSnapshot = false;
        }
    }

    private buildEntityFromAuthority(snapshotBuilding: AuthorityBuildingSnapshot): BuildingEntity {
        const definition = BUILDING_DEFINITIONS.find((entry) => entry.id === snapshotBuilding.type);
        if (!definition) {
            throw new Error(`unknown_authority_building_type:${snapshotBuilding.type}`);
        }

        return {
            id: snapshotBuilding.id,
            definition,
            origin: { ...snapshotBuilding.origin },
            state: snapshotBuilding.state as BuildingState,
            supplied: this.fromAuthorityStockpile(snapshotBuilding.supplied),
            level: snapshotBuilding.level,
            markedForDemolition: snapshotBuilding.markedForDemolition,
            currentHp: snapshotBuilding.hp,
            damageFlashSeconds: 0,
            attackCooldownSeconds: 0,
            pendingAction: snapshotBuilding.pendingAction as BuildingWorkKind | null,
            pendingLevel: snapshotBuilding.pendingLevel,
        };
    }

    private fromAuthorityStockpile(stockpile: AuthorityStockpile): Stockpile {
        return {
            spirit_wood: stockpile.spirit_wood ?? 0,
            spirit_stone: stockpile.spirit_stone ?? 0,
            herb: stockpile.herb ?? 0,
        };
    }

    private executeAuthorityCommand<TPayload>(
        command: AuthorityCommandEnvelope<TPayload>,
        options?: {
            commandKey?: string;
            onAccepted?: () => void;
        },
    ): void {
        const commandKey = options?.commandKey ?? command.name;
        if (!this.authorityConnected || this.authorityPendingCommands.has(commandKey)) {
            return;
        }

        this.authorityPendingCommands.add(commandKey);
        void this.authorityClient
            .executeCommand(this.authoritySessionId, command)
            .then((response) => {
                this.applyAuthorityResponse(response, command.name);
                options?.onAccepted?.();
            })
            .catch((error) => {
                const message = error instanceof Error ? error.message : `${command.name}_failed`;
                this.authorityLastError = message;
                this.logRuntime('ERROR', 'BUILD', 'authority.command_failed', 'authority command 执行失败', {
                    command: command.name,
                    error: message,
                });
                this.setMessage(`authority 命令失败：${message}`);
            })
            .finally(() => {
                this.authorityPendingCommands.delete(commandKey);
            });
    }

    private syncSessionProgressToAuthority(trigger: string): void {
        if (!this.authorityConnected || this.authorityHydratingSnapshot) {
            return;
        }

        this.executeAuthorityCommand(
            {
                name: 'sync_session_progress',
                payload: {
                    phase: this.sessionPhase as AuthoritySessionPhase,
                    outcome: this.sessionOutcome as AuthoritySessionOutcome,
                    objective: this.sessionObjectiveText,
                    firstRaidTriggered: this.firstRaidTriggered,
                    firstRaidResolved: this.firstRaidResolved,
                },
            },
            {
                commandKey: `sync_session_progress:${trigger}`,
            },
        );
    }

    update(deltaTime: number): void {
        if (!this.mapReady) {
            return;
        }

        this.advanceLongPress(deltaTime);
        this.advanceSessionLoop(deltaTime);
        this.advanceResourceRespawns(deltaTime);
        this.advanceCombatFeedback(deltaTime);
        this.advanceHostileNpc(deltaTime);
        this.advanceGuardTowers(deltaTime);
        this.advanceDisciple(deltaTime);
        this.refreshObjectiveMarker();
        this.renderStatus();
    }

    onDestroy(): void {
        input.off(Input.EventType.TOUCH_START, this.onTouchStart, this);
        input.off(Input.EventType.TOUCH_MOVE, this.onTouchMove, this);
        input.off(Input.EventType.TOUCH_END, this.onTouchEnd, this);
        input.off(Input.EventType.TOUCH_CANCEL, this.onTouchCancel, this);
        input.off(Input.EventType.MOUSE_DOWN, this.onMouseDown, this);
        input.off(Input.EventType.MOUSE_MOVE, this.onMouseMove, this);
        input.off(Input.EventType.MOUSE_UP, this.onMouseUp, this);
        view.off('canvas-resize', this.onCanvasResize, this);
        view.off('design-resolution-changed', this.onCanvasResize, this);
        this.logRuntime('INFO', 'BOOT', 'bootstrap.destroy', 'SectMapBootstrap 已销毁');
        this.detachRuntimeDebugBridge();
    }

    private loadVisualAssets(): void {
        const entries = Object.entries(SECT_MAP_VISUAL_ASSET_PATHS) as [SectMapVisualAssetId, string][];
        if (entries.length === 0) {
            return;
        }

        let pending = entries.length;
        let loadedCount = 0;

        entries.forEach(([assetId, path]) => {
            resources.load(path, SpriteFrame, (error, spriteFrame) => {
                pending -= 1;

                if (error || !spriteFrame) {
                    this.logRuntime('WARN', 'BOOT', 'assets.visual_load_failed', '宗门地图贴图资源加载失败，回退 Graphics 占位表现', {
                        assetId,
                        path,
                        error: error?.message ?? 'missing_sprite_frame',
                    });
                } else {
                    this.visualAssetFrames[assetId] = spriteFrame;
                    loadedCount += 1;
                }

                if (pending === 0) {
                    this.logRuntime('INFO', 'BOOT', 'assets.visual_load_complete', '宗门地图贴图资源加载完成', {
                        loadedCount,
                        totalCount: entries.length,
                    });
                    this.refreshVisualLayersAfterAssetLoad();
                }
            });
        });
    }

    private refreshVisualLayersAfterAssetLoad(): void {
        if (this.mapColumns <= 0 || this.mapRows <= 0) {
            return;
        }

        this.refreshResourceMarkers();
        this.refreshBuildings();
        this.refreshDiscipleToken();
        this.refreshHostileNpcToken();
        this.refreshObjectiveMarker();
    }

    private getVisualAssetFrame(assetId: SectMapVisualAssetId): SpriteFrame | null {
        return this.visualAssetFrames[assetId] ?? null;
    }

    private getBuildingVisualAssetId(buildingId: BuildingDefinition['id']): SectMapVisualAssetId {
        switch (buildingId) {
            case 'disciple_quarters':
                return 'building.disciple_quarters';
            case 'warehouse':
                return 'building.warehouse';
            case 'herb_garden':
                return 'building.herb_garden';
            case 'guard_tower':
                return 'building.guard_tower';
            case 'main_hall':
            default:
                return 'building.main_hall';
        }
    }

    private getBuildingMaxLevel(building: BuildingEntity): number {
        return building.definition.id === 'main_hall' ? 1 : 2;
    }

    private getBuildingUpgradeCost(building: BuildingEntity): Stockpile | null {
        if (building.level >= this.getBuildingMaxLevel(building)) {
            return null;
        }

        switch (building.definition.id) {
            case 'guard_tower':
                return { spirit_wood: 1, spirit_stone: 1, herb: 0 };
            case 'disciple_quarters':
                return { spirit_wood: 1, spirit_stone: 0, herb: 1 };
            case 'herb_garden':
                return { spirit_wood: 1, spirit_stone: 0, herb: 1 };
            case 'warehouse':
                return { spirit_wood: 1, spirit_stone: 1, herb: 0 };
            case 'main_hall':
            default:
                return null;
        }
    }

    private getActiveCostForBuilding(building: BuildingEntity): Stockpile {
        return building.pendingAction === 'upgrade' ? this.getBuildingUpgradeCost(building) ?? EMPTY_STOCKPILE() : building.definition.cost;
    }

    private getBuildingMaxHp(building: BuildingEntity): number {
        return Math.round(building.definition.maxHp * (1 + (building.level - 1) * 0.45));
    }

    private getBuildingRepairCost(building: BuildingEntity): Stockpile {
        const base = building.definition.repairCost;
        if (building.level <= 1) {
            return { ...base };
        }

        return {
            spirit_wood: base.spirit_wood + (building.definition.id === 'guard_tower' ? 1 : 0),
            spirit_stone: base.spirit_stone + (building.definition.id === 'guard_tower' || building.definition.id === 'warehouse' ? 1 : 0),
            herb: base.herb + (building.definition.id === 'herb_garden' ? 1 : 0),
        };
    }

    private getBuildingStructureDefense(building: BuildingEntity): number {
        return Number((building.definition.structureDefense + (building.level - 1) * 0.8).toFixed(2));
    }

    private getBuildingGuardProfile(building: BuildingEntity): GuardProfile | null {
        if (!building.definition.guardProfile) {
            return null;
        }

        return {
            attackPower: Number((building.definition.guardProfile.attackPower + (building.level - 1) * 1.8).toFixed(2)),
            rangeTiles: building.definition.guardProfile.rangeTiles + (building.level - 1),
            attackInterval: Number(Math.max(0.75, building.definition.guardProfile.attackInterval - (building.level - 1) * 0.12).toFixed(2)),
        };
    }

    private canUpgradeBuilding(building: BuildingEntity): boolean {
        return (
            building.definition.id !== 'main_hall' &&
            building.state === 'active' &&
            building.pendingAction === null &&
            !building.markedForDemolition &&
            building.id !== this.sessionRuinBuildingId &&
            building.level < this.getBuildingMaxLevel(building)
        );
    }

    private getResourceVisualAssetId(resourceKind: ResourceKind): SectMapVisualAssetId {
        switch (resourceKind) {
            case 'spirit_stone':
                return 'resource.spirit_stone';
            case 'herb':
                return 'resource.herb';
            case 'spirit_wood':
            default:
                return 'resource.spirit_wood';
        }
    }

    private getDisciplePortraitAssetId(unit: DiscipleEntity): SectMapVisualAssetId {
        if (unit.currentHp <= 0) {
            return 'disciple.dead';
        }

        const hpRatio = unit.currentHp / Math.max(1, unit.model.stats.maxHp);
        if (hpRatio <= 0.3) {
            return 'disciple.dying';
        }
        if (hpRatio <= 0.68 || unit.visualState === 'injured') {
            return 'disciple.injured';
        }
        return 'disciple.normal';
    }

    private getDisciplePortraitFrame(unit: DiscipleEntity): SpriteFrame | null {
        return this.getVisualAssetFrame(this.getDisciplePortraitAssetId(unit));
    }

    private getHostilePortraitFrame(unit: HostileNpcEntity): SpriteFrame | null {
        const hpRatio = unit.currentHp / Math.max(1, unit.model.stats.maxHp);
        const assetId: SectMapVisualAssetId = hpRatio <= 0.68 ? 'hostile.injured' : 'hostile.normal';
        return this.getVisualAssetFrame(assetId);
    }

    private getSpriteFrameDimensions(spriteFrame: SpriteFrame): { width: number; height: number } {
        const size = spriteFrame.originalSize;
        return {
            width: size?.width ?? spriteFrame.rect.width,
            height: size?.height ?? spriteFrame.rect.height,
        };
    }

    private configureSpriteNode(
        target: Node,
        spriteFrame: SpriteFrame,
        anchorX = 0.5,
        anchorY = 0.5,
        color: Color = new Color(255, 255, 255, 255),
    ): Sprite {
        const sprite = target.getComponent(Sprite) ?? target.addComponent(Sprite);
        const { width, height } = this.getSpriteFrameDimensions(spriteFrame);
        sprite.spriteFrame = spriteFrame;
        sprite.trim = false;
        sprite.sizeMode = Sprite.SizeMode.RAW;
        sprite.color = color;
        this.configureNodeSize(target, width, height, anchorX, anchorY);
        return sprite;
    }

    private getBuildingArtColor(building: BuildingEntity): Color {
        if (building.damageFlashSeconds > 0) {
            return new Color(255, 234, 194, 255);
        }

        switch (building.state) {
            case 'planned':
                return new Color(142, 146, 156, 186);
            case 'supplied':
                return new Color(214, 244, 214, 232);
            case 'constructing':
                return new Color(242, 216, 164, 236);
            case 'damaged':
                return building.currentHp <= 0 ? new Color(128, 104, 104, 156) : new Color(224, 170, 156, 236);
            case 'active':
            default:
                return new Color(255, 255, 255, 255);
        }
    }

    private getResourceArtColor(resource: ResourceNode, isHarvestable: boolean): Color {
        if (!isHarvestable) {
            return new Color(150, 158, 172, 178);
        }
        if (resource.designated) {
            return new Color(255, 255, 255, 255);
        }
        return new Color(234, 236, 244, 232);
    }

    private ensureHudRoots(): void {
        if (!this.canvas) {
            return;
        }

        this.hudRoot = this.ensureChild(HUD_ROOT, this.canvas);
        this.toolbarRoot = this.ensureChild(TOOLBAR_ROOT, this.hudRoot);
        this.buildPanelRoot = this.ensureChild(BUILD_PANEL_ROOT, this.hudRoot);
        this.radialMenuRoot = this.ensureChild(RADIAL_MENU_ROOT, this.hudRoot);

        this.configureNodeSize(this.hudRoot, view.getVisibleSize().width, view.getVisibleSize().height);
        this.configureNodeSize(this.toolbarRoot, 760, 88);
        this.configureNodeSize(this.buildPanelRoot, 640, 128);
        this.configureNodeSize(this.radialMenuRoot, 320, 320);
        this.toolbarRoot.setPosition(0, -286, 0);
        this.buildPanelRoot.setPosition(0, -168, 0);
        this.radialMenuRoot.active = false;
        this.buildPanelRoot.active = false;
    }

    private onCanvasResize(): void {
        this.refreshViewportLayout('view.canvas_resize');
    }

    private refreshViewportLayout(reason: string): void {
        const visibleSize = view.getVisibleSize();
        const screenMetrics = getRuntimeScreenMetrics();
        this.configureNodeSize(this.node, visibleSize.width, visibleSize.height);
        this.applyHudLayout(visibleSize.width, visibleSize.height, screenMetrics);

        const designResolution = view.getDesignResolutionSize();

        const signature = [
            designResolution.width.toFixed(2),
            designResolution.height.toFixed(2),
            visibleSize.width.toFixed(2),
            visibleSize.height.toFixed(2),
            screenMetrics.windowWidth.toFixed(2),
            screenMetrics.windowHeight.toFixed(2),
            screenMetrics.safeArea.top.toFixed(2),
            screenMetrics.safeArea.bottom.toFixed(2),
            screenMetrics.safeArea.height.toFixed(2),
            screenMetrics.statusBarHeight.toFixed(2),
            screenMetrics.safeAreaFallbackApplied ? 'fallback' : 'native',
        ].join('|');

        if (signature !== this.viewportLayoutSignature) {
            this.viewportLayoutSignature = signature;
            this.logRuntime('INFO', 'BOOT', 'viewport.layout_ready', 'Cocos 视图布局已更新', {
                reason,
                designWidth: designResolution.width,
                designHeight: designResolution.height,
                visibleWidth: visibleSize.width,
                visibleHeight: visibleSize.height,
                windowWidth: screenMetrics.windowWidth,
                windowHeight: screenMetrics.windowHeight,
                safeArea: screenMetrics.safeArea,
                statusBarHeight: screenMetrics.statusBarHeight,
                safeAreaFallbackApplied: screenMetrics.safeAreaFallbackApplied,
                metricsSource: screenMetrics.source,
                portraitBaseline: this.portraitBaselineSnapshot,
            });
        }
    }

    private getSafeInsetsInViewSpace(
        visibleWidth: number,
        visibleHeight: number,
        screenMetrics: RuntimeScreenMetrics,
    ): RuntimeSafeInsets {
        const scaleX = visibleWidth / Math.max(1, screenMetrics.windowWidth);
        const scaleY = visibleHeight / Math.max(1, screenMetrics.windowHeight);
        const safeArea = screenMetrics.safeArea;

        return {
            left: Math.max(0, safeArea.left * scaleX),
            right: Math.max(0, (screenMetrics.windowWidth - safeArea.right) * scaleX),
            top: Math.max(0, safeArea.top * scaleY),
            bottom: Math.max(0, (screenMetrics.windowHeight - safeArea.bottom) * scaleY),
        };
    }

    private applyHudLayout(visibleWidth: number, visibleHeight: number, screenMetrics: RuntimeScreenMetrics): void {
        if (this.hudRoot) {
            this.configureNodeSize(this.hudRoot, visibleWidth, visibleHeight);
        }

        const safeInsets = this.getSafeInsetsInViewSpace(visibleWidth, visibleHeight, screenMetrics);
        const sideGutter = Math.max(24, visibleWidth * 0.04);
        const horizontalPaddingLeft = safeInsets.left + sideGutter;
        const horizontalPaddingRight = safeInsets.right + sideGutter;
        const safeFrameWidth = Math.max(1, visibleWidth - safeInsets.left - safeInsets.right);
        const safeFrameHeight = Math.max(1, visibleHeight - safeInsets.top - safeInsets.bottom);
        const safeCenterX = (safeInsets.left - safeInsets.right) * 0.5;
        const topAnchorY = visibleHeight * 0.5 - safeInsets.top;
        const bottomAnchorY = -visibleHeight * 0.5 + safeInsets.bottom;
        const statusWidth = Math.max(300, visibleWidth - horizontalPaddingLeft - horizontalPaddingRight);
        const toolbarWidth = Math.max(420, Math.min(640, visibleWidth - horizontalPaddingLeft - horizontalPaddingRight));
        const buildPanelWidth = Math.max(420, Math.min(580, visibleWidth - horizontalPaddingLeft - horizontalPaddingRight));
        const statusY = topAnchorY - 56;
        const toolbarY = bottomAnchorY + 72;
        const buildPanelY = bottomAnchorY + 188;

        if (this.statusLabel) {
            const statusNode = this.statusLabel.node;
            this.configureNodeSize(statusNode, statusWidth, 140);
            statusNode.setPosition(safeCenterX, statusY, 0);
        }

        if (this.toolbarRoot) {
            this.configureNodeSize(this.toolbarRoot, toolbarWidth, 88);
            this.toolbarRoot.setPosition(safeCenterX, toolbarY, 0);
        }

        if (this.buildPanelRoot) {
            this.configureNodeSize(this.buildPanelRoot, buildPanelWidth, 128);
            this.buildPanelRoot.setPosition(safeCenterX, buildPanelY, 0);
        }

        this.portraitBaselineSnapshot = {
            safeInsets,
            safeFrameWidth,
            safeFrameHeight,
            metricsSource: screenMetrics.source,
            safeAreaFallbackApplied: screenMetrics.safeAreaFallbackApplied,
            hud: {
                centerX: safeCenterX,
                horizontalPaddingLeft,
                horizontalPaddingRight,
                topAnchorY,
                bottomAnchorY,
                statusY,
                toolbarY,
                buildPanelY,
            },
        };
    }

    private buildToolbar(): void {
        if (!this.toolbarRoot) {
            return;
        }

        for (const child of [...this.toolbarRoot.children]) {
            child.destroy();
        }
        this.toolbarButtons.clear();

        for (const button of TOOLBAR_BUTTONS) {
            const node = this.createScreenButton(button.label, 138, 56, () => {
                switch (button.key) {
                    case 'browse':
                        this.closeBuildPanel();
                        this.clearBuildPlacement();
                        this.closeRadialMenu();
                        this.setMode('browse', '已切回查看模式');
                        break;
                    case 'gather':
                        this.closeBuildPanel();
                        this.clearBuildPlacement();
                        this.closeRadialMenu();
                        this.setMode('gather', '采集模式：点击资源格添加或取消采集标记');
                        break;
                    case 'build':
                        this.closeRadialMenu();
                        this.openBuildPanel(this.selectedTile ?? this.findScreenCenterTile());
                        break;
                    case 'demolish':
                        this.closeBuildPanel();
                        this.clearBuildPlacement();
                        this.closeRadialMenu();
                        this.setMode('demolish', '拆除模式：点击建筑添加拆除标记');
                        break;
                }
            });

            node.setPosition(button.positionX, 0, 0);
            node.name = `ToolbarButton-${button.key}`;
            node.setParent(this.toolbarRoot);
            this.toolbarButtons.set(button.key, node);
        }

        this.refreshToolbarButtons();
    }

    private buildBuildPanel(): void {
        if (!this.buildPanelRoot) {
            return;
        }

        for (const child of [...this.buildPanelRoot.children]) {
            child.destroy();
        }
        this.buildButtons.clear();

        const panelBackground = new Node('BuildPanelBackground');
        panelBackground.setParent(this.buildPanelRoot);
        this.configureNodeSize(panelBackground, 620, 116);
        this.paintPanelBackground(panelBackground, new Color(28, 36, 48, 220), new Color(96, 120, 146, 255));

        const candidates = BUILDING_DEFINITIONS.filter((definition) => definition.id !== 'main_hall');
        const spacing = 148;
        const originX = -((candidates.length - 1) * spacing) * 0.5;

        candidates.forEach((definition, index) => {
            const node = this.createScreenButton(
                `${definition.label}\n${this.getCostText(definition.cost)}`,
                132,
                84,
                () => {
                    this.enterBuildPlacement(definition, this.preferredBuildTile ?? this.selectedTile ?? this.findScreenCenterTile());
                },
                20,
            );
            node.setPosition(originX + spacing * index, 0, 0);
            node.name = `BuildButton-${definition.id}`;
            node.setParent(this.buildPanelRoot!);
            this.buildButtons.set(definition.id, node);
        });
    }

    private createScreenButton(labelText: string, width: number, height: number, onClick: () => void, fontSize = 24): Node {
        const node = new Node(labelText);
        this.configureNodeSize(node, width, height);
        this.paintButtonBackground(node, false);

        const labelNode = new Node('Label');
        labelNode.setParent(node);
        labelNode.setPosition(0, 0, 0);
        const labelTransform = labelNode.addComponent(UITransform);
        labelTransform.setContentSize(width - 16, height - 10);
        const label = labelNode.addComponent(Label);
        label.string = labelText;
        label.fontSize = fontSize;
        label.lineHeight = fontSize + 4;
        label.color = new Color(236, 241, 248, 255);
        label.enableWrapText = true;

        this.bindButtonActivation(node, onClick);

        return node;
    }

    private paintButtonBackground(node: Node, active: boolean): void {
        const graphics = node.getComponent(Graphics) ?? node.addComponent(Graphics);
        const transform = node.getComponent(UITransform);
        const width = transform?.width ?? 120;
        const height = transform?.height ?? 44;

        graphics.clear();
        graphics.lineWidth = 2;
        graphics.fillColor = active ? new Color(70, 122, 184, 245) : new Color(31, 43, 57, 224);
        graphics.strokeColor = active ? new Color(180, 220, 255, 255) : new Color(102, 124, 148, 255);
        graphics.roundRect(-width * 0.5, -height * 0.5, width, height, 12);
        graphics.fill();
        graphics.stroke();
    }

    private paintPanelBackground(node: Node, fill: Color, stroke: Color): void {
        const graphics = node.getComponent(Graphics) ?? node.addComponent(Graphics);
        const transform = node.getComponent(UITransform);
        const width = transform?.width ?? 120;
        const height = transform?.height ?? 44;

        graphics.clear();
        graphics.lineWidth = 2;
        graphics.fillColor = fill;
        graphics.strokeColor = stroke;
        graphics.roundRect(-width * 0.5, -height * 0.5, width, height, 16);
        graphics.fill();
        graphics.stroke();
    }

    private refreshToolbarButtons(): void {
        const activeKey =
            this.inputMode === 'build_select' || this.inputMode === 'build_place'
                ? 'build'
                : this.inputMode;

        for (const [key, node] of this.toolbarButtons) {
            this.paintButtonBackground(node, key === activeKey);
        }
    }

    private openBuildPanel(preferredTile: TileCoord | null): void {
        if (!this.buildPanelRoot) {
            return;
        }

        this.preferredBuildTile = preferredTile;
        this.buildPanelRoot.active = true;
        this.clearBuildPlacement();
        this.logRuntime('INFO', 'BUILD', 'build.panel_open', '已打开建造面板', {
            preferredTile: this.formatTile(preferredTile),
        });
        this.setMode('build_select', '建造模式：先选建筑，再拖动蓝图，点击其他位置确认');
    }

    private closeBuildPanel(): void {
        if (this.buildPanelRoot) {
            this.buildPanelRoot.active = false;
        }
        if (this.inputMode === 'build_select') {
            this.inputMode = 'browse';
        }
        this.refreshToolbarButtons();
    }

    private enterBuildPlacement(definition: BuildingDefinition, preferredTile: TileCoord | null): void {
        const fallbackTile = this.findNearestBuildableTile(preferredTile ?? this.findScreenCenterTile());
        if (!fallbackTile) {
            this.setMessage('当前视野附近没有可放置蓝图的位置');
            return;
        }

        this.closeRadialMenu();
        this.buildPanelRoot!.active = false;
        this.buildPlacement = {
            definition,
            origin: fallbackTile,
        };
        this.logRuntime('INFO', 'BUILD', 'build.preview_enter', '已进入蓝图摆放阶段', {
            buildingId: definition.id,
            origin: this.formatTile(fallbackTile),
        });
        this.setMode('build_place', `正在放置 ${definition.label}：拖动蓝图调整位置，点击其他位置确认`);
        this.refreshBlueprintPreview();
    }

    private clearBuildPlacement(): void {
        this.buildPlacement = null;
        this.blueprintPreview.active = false;
        if (this.inputMode === 'build_place') {
            this.inputMode = 'browse';
            this.refreshToolbarButtons();
        }
    }

    private registerInput(): void {
        input.on(Input.EventType.TOUCH_START, this.onTouchStart, this);
        input.on(Input.EventType.TOUCH_MOVE, this.onTouchMove, this);
        input.on(Input.EventType.TOUCH_END, this.onTouchEnd, this);
        input.on(Input.EventType.TOUCH_CANCEL, this.onTouchCancel, this);
        input.on(Input.EventType.MOUSE_DOWN, this.onMouseDown, this);
        input.on(Input.EventType.MOUSE_MOVE, this.onMouseMove, this);
        input.on(Input.EventType.MOUSE_UP, this.onMouseUp, this);
    }

    private onTouchStart(event: EventTouch): void {
        this.beginPointer(event.getID(), event.getUILocation());
    }

    private onTouchMove(event: EventTouch): void {
        this.movePointer(event.getID(), event.getUILocation(), event.getUIDelta());
    }

    private onTouchEnd(event: EventTouch): void {
        this.endPointer(event.getID(), event.getUILocation());
    }

    private onTouchCancel(event: EventTouch): void {
        this.cancelPointer(event.getID());
    }

    private onMouseDown(event: EventTouch): void {
        if ((event as any).getButton && (event as any).getButton() !== 0) {
            return;
        }
        this.beginPointer(MOUSE_POINTER_ID, event.getUILocation());
    }

    private onMouseMove(event: EventTouch): void {
        this.movePointer(MOUSE_POINTER_ID, event.getUILocation(), event.getUIDelta());
    }

    private onMouseUp(event: EventTouch): void {
        if ((event as any).getButton && (event as any).getButton() !== 0) {
            return;
        }
        this.endPointer(MOUSE_POINTER_ID, event.getUILocation());
    }

    private beginPointer(pointerId: number, location: Readonly<Vec3>): void {
        if (!this.mapReady || this.activePointerId !== null) {
            return;
        }

        if (this.isLocationBlockedByScreenUI(location)) {
            return;
        }

        this.activePointerId = pointerId;
        this.pointerStart.set(location.x, location.y, 0);
        this.pointerCurrent.set(location.x, location.y, 0);
        this.pointerHoldSeconds = 0;
        this.pointerStartTile = this.pickTileFromUILocation(this.pointerCurrent);
        this.pointerDownOnBlueprint = this.inputMode === 'build_place' && this.isPointInsideNode(this.blueprintPreview, location);
        this.draggingMap = false;
        this.draggingBlueprint = false;
        this.longPressTriggered = false;

        this.logRuntime('DEBUG', 'INPUT', 'pointer.begin', '已捕获新的交互起点', {
            pointerId,
            locationX: Math.round(location.x),
            locationY: Math.round(location.y),
            tile: this.formatTile(this.pointerStartTile),
            mode: this.inputMode,
        });

        if (this.radialMenuRoot?.active && !this.isPointInsideNode(this.radialMenuRoot, location)) {
            this.closeRadialMenu();
        }
    }

    private movePointer(pointerId: number, location: Readonly<Vec3>, delta: Readonly<Vec3>): void {
        if (!this.mapReady || pointerId !== this.activePointerId) {
            return;
        }

        this.pointerCurrent.set(location.x, location.y, 0);

        const distance = Vec3.distance(this.pointerStart, this.pointerCurrent);

        if (this.inputMode === 'build_place' && this.pointerDownOnBlueprint) {
            if (!this.draggingBlueprint && distance >= DRAG_THRESHOLD) {
                this.draggingBlueprint = true;
            }

            if (this.draggingBlueprint) {
                const tile = this.pickTileFromUILocation(this.pointerCurrent);
                if (tile && this.buildPlacement) {
                    this.buildPlacement.origin = tile;
                    this.refreshBlueprintPreview();
                }
            }
            return;
        }

        if (!this.draggingMap && distance >= DRAG_THRESHOLD) {
            this.draggingMap = true;
        }

        if (this.draggingMap && this.inputMode !== 'build_place') {
            const position = this.mapRoot.position;
            this.mapRoot.setPosition(position.x + delta.x, position.y + delta.y, position.z);
        }
    }

    private endPointer(pointerId: number, location: Readonly<Vec3>): void {
        if (pointerId !== this.activePointerId) {
            return;
        }

        this.pointerCurrent.set(location.x, location.y, 0);

        if (this.longPressTriggered) {
            this.clearPointer();
            return;
        }

        if (this.inputMode === 'build_place') {
            if (!this.draggingBlueprint && !this.pointerDownOnBlueprint) {
                this.confirmBuildPlacement();
            }
            this.clearPointer();
            return;
        }

        if (this.draggingMap || this.draggingBlueprint) {
            this.clearPointer();
            return;
        }

        const tile = this.pickTileFromUILocation(this.pointerCurrent);
        if (!tile) {
            this.logRuntime('WARN', 'INPUT', 'pointer.tap_miss', '点击未命中有效地块', {
                pointerId,
                locationX: Math.round(location.x),
                locationY: Math.round(location.y),
            });
            this.setMessage('未命中有效地块');
            this.clearPointer();
            return;
        }

        this.selectedTile = tile;
        this.showSelection(tile);
        this.logRuntime('INFO', 'INPUT', 'pointer.tap_tile', '点击命中地块', {
            pointerId,
            tile: this.formatTile(tile),
            mode: this.inputMode,
        });
        this.handleTileTap(tile);
        this.clearPointer();
    }

    private cancelPointer(pointerId: number): void {
        if (pointerId !== this.activePointerId) {
            return;
        }
        this.clearPointer();
    }

    private clearPointer(): void {
        this.activePointerId = null;
        this.pointerHoldSeconds = 0;
        this.pointerStartTile = null;
        this.pointerDownOnBlueprint = false;
        this.draggingMap = false;
        this.draggingBlueprint = false;
        this.longPressTriggered = false;
    }

    private advanceLongPress(deltaTime: number): void {
        if (
            this.activePointerId === null ||
            this.draggingMap ||
            this.draggingBlueprint ||
            this.longPressTriggered ||
            this.inputMode === 'build_place' ||
            !this.pointerStartTile
        ) {
            return;
        }

        this.pointerHoldSeconds += deltaTime;
        if (this.pointerHoldSeconds < LONG_PRESS_SECONDS) {
            return;
        }

        this.openRadialMenu(this.pointerStartTile, this.pointerCurrent);
        this.longPressTriggered = true;
    }

    private handleTileTap(tile: TileCoord): void {
        if (this.radialMenuRoot?.active) {
            this.closeRadialMenu();
        }

        switch (this.inputMode) {
            case 'gather':
                this.toggleGatherDesignation(tile);
                return;
            case 'demolish':
                this.toggleDemolition(tile);
                return;
            default:
                this.describeTile(tile);
                return;
        }
    }

    private openRadialMenu(tile: TileCoord, uiLocation: Readonly<Vec3>): void {
        if (!this.radialMenuRoot || !this.canvas) {
            return;
        }

        const actions = this.getRadialActionsForTile(tile);
        if (actions.length === 0) {
            return;
        }

        this.logRuntime('INFO', 'INPUT', 'radial.open', '已打开长按环形菜单', {
            tile: this.formatTile(tile),
            actions: actions.map((action) => action.key).join(','),
            uiX: Math.round(uiLocation.x),
            uiY: Math.round(uiLocation.y),
        });

        this.showSelection(tile);
        this.selectedTile = tile;

        for (const child of [...this.radialMenuRoot.children]) {
            child.destroy();
        }

        const canvasTransform = this.canvas.getComponent(UITransform);
        if (!canvasTransform) {
            return;
        }

        const localPoint = canvasTransform.convertToNodeSpaceAR(v3(uiLocation.x, uiLocation.y, 0));
        this.radialMenuRoot.setPosition(localPoint.x, localPoint.y, 0);
        this.radialMenuRoot.active = true;

        const total = actions.length;
        actions.forEach((action, index) => {
            const angle = (-90 + (360 / total) * index) * (Math.PI / 180);
            const node = this.createRadialButton(action.label, () => {
                action.execute();
                this.closeRadialMenu();
            });
            node.setPosition(Math.cos(angle) * RADIAL_RADIUS, Math.sin(angle) * RADIAL_RADIUS, 0);
            node.setParent(this.radialMenuRoot!);
        });

        const center = new Node('CenterDisc');
        center.setParent(this.radialMenuRoot);
        this.configureNodeSize(center, 92, 92);
        const centerGraphics = center.addComponent(Graphics);
        centerGraphics.fillColor = new Color(26, 31, 42, 228);
        centerGraphics.strokeColor = new Color(132, 160, 190, 255);
        centerGraphics.lineWidth = 2;
        centerGraphics.circle(0, 0, 46);
        centerGraphics.fill();
        centerGraphics.stroke();

        const labelNode = new Node('CenterLabel');
        labelNode.setParent(center);
        const labelTransform = labelNode.addComponent(UITransform);
        labelTransform.setContentSize(84, 64);
        const label = labelNode.addComponent(Label);
        label.string = this.getTileContextTitle(tile);
        label.fontSize = 18;
        label.lineHeight = 22;
        label.color = new Color(240, 244, 248, 255);
        label.enableWrapText = true;
    }

    private closeRadialMenu(): void {
        if (!this.radialMenuRoot) {
            return;
        }

        this.radialMenuRoot.active = false;
        for (const child of [...this.radialMenuRoot.children]) {
            child.destroy();
        }
    }

    private createRadialButton(labelText: string, onClick: () => void): Node {
        const node = new Node(labelText);
        this.configureNodeSize(node, 84, 84);
        const graphics = node.addComponent(Graphics);
        graphics.fillColor = new Color(49, 76, 106, 238);
        graphics.strokeColor = new Color(185, 220, 248, 255);
        graphics.lineWidth = 2;
        graphics.circle(0, 0, 42);
        graphics.fill();
        graphics.stroke();

        const labelNode = new Node('Label');
        labelNode.setParent(node);
        const labelTransform = labelNode.addComponent(UITransform);
        labelTransform.setContentSize(72, 54);
        const label = labelNode.addComponent(Label);
        label.string = labelText;
        label.fontSize = 18;
        label.lineHeight = 20;
        label.color = new Color(246, 248, 252, 255);
        label.enableWrapText = true;

        this.bindButtonActivation(node, onClick);
        return node;
    }

    private bindButtonActivation(node: Node, onClick: () => void): void {
        node.on(Node.EventType.TOUCH_END, () => onClick());
        node.on(Node.EventType.MOUSE_UP, () => onClick());
    }

    private getRadialActionsForTile(tile: TileCoord): RadialAction[] {
        const building = this.getBuildingAtTile(tile);
        if (building) {
            const actions: RadialAction[] = [];
            if (this.canUpgradeBuilding(building)) {
                actions.push({
                    key: 'upgrade',
                    label: '升级',
                    execute: () => this.requestBuildingUpgrade(building),
                });
            }
            if (building.definition.id !== 'main_hall' && !building.markedForDemolition) {
                actions.push({
                    key: 'demolish',
                    label: '拆除',
                    execute: () => this.toggleDemolition(tile),
                });
            }
            return actions;
        }

        const resource = this.resourceNodes.get(this.getTileKey(tile.col, tile.row));
        if (resource) {
            return [
                {
                    key: 'gather',
                    label: resource.designated ? '取消采集' : '标记采集',
                    execute: () => this.toggleGatherDesignation(tile),
                },
            ];
        }

        if (this.isTileBuildable(tile)) {
            return [
                {
                    key: 'build',
                    label: '建造',
                    execute: () => this.openBuildPanel(tile),
                },
            ];
        }

        return [];
    }

    private requestBuildingUpgrade(building: BuildingEntity): void {
        if (!this.canUpgradeBuilding(building)) {
            this.setMessage(`${building.definition.label} 当前不可升级`);
            return;
        }

        const upgradeCost = this.getBuildingUpgradeCost(building);
        if (!upgradeCost) {
            this.setMessage(`${building.definition.label} 已达当前版本最高等级`);
            return;
        }

        if (this.authorityConnected) {
            this.executeAuthorityCommand(
                {
                    name: 'request_upgrade',
                    payload: {
                        buildingId: building.id,
                    },
                },
                {
                    commandKey: `request_upgrade:${building.id}`,
                },
            );
            return;
        }

        building.pendingAction = 'upgrade';
        building.pendingLevel = building.level + 1;
        building.supplied = EMPTY_STOCKPILE();
        building.markedForDemolition = false;
        this.setBuildingState(building, 'planned', 'upgrade_requested');
        this.refreshBuildings();
        this.logRuntime('INFO', 'BUILD', 'build.upgrade_requested', '已下达建筑升级指令', {
            buildingId: building.id,
            buildingType: building.definition.id,
            fromLevel: building.level,
            toLevel: building.pendingLevel,
            cost: this.getCostText(upgradeCost),
        });
        this.setMessage(`${building.definition.label} 已进入升级筹备，等待补足 ${this.getCostText(upgradeCost)}`);
    }

    private getTileContextTitle(tile: TileCoord): string {
        const building = this.getBuildingAtTile(tile);
        if (building) {
            return building.definition.label;
        }

        const resource = this.resourceNodes.get(this.getTileKey(tile.col, tile.row));
        if (resource) {
            return RESOURCE_DISPLAY[resource.kind].title;
        }

        if (this.isTileBuildable(tile)) {
            return `地块 ${tile.col},${tile.row}`;
        }

        return '不可操作';
    }

    private setMode(mode: InputMode, message?: string): void {
        const previousMode = this.inputMode;
        this.inputMode = mode;
        this.refreshToolbarButtons();
        if (previousMode !== mode) {
            this.logRuntime('INFO', 'MODE', 'mode.change', '输入模式已切换', {
                from: previousMode,
                to: mode,
            });
        }
        if (message) {
            this.setMessage(message);
        }
    }

    private readLogicLayers(): void {
        this.roadTiles.clear();
        this.blockedTiles.clear();
        this.resourceNodes.clear();
        this.buildableTileCount = 0;

        if (this.roadLayer) {
            for (let row = 0; row < this.mapRows; row += 1) {
                for (let col = 0; col < this.mapColumns; col += 1) {
                    if (this.roadLayer.getTileGIDAt(col, row) > 0) {
                        this.roadTiles.add(this.getTileKey(col, row));
                    }
                }
            }
        }

        if (this.blockedLayer) {
            for (let row = 0; row < this.mapRows; row += 1) {
                for (let col = 0; col < this.mapColumns; col += 1) {
                    if (this.blockedLayer.getTileGIDAt(col, row) > 0) {
                        this.blockedTiles.add(this.getTileKey(col, row));
                    }
                }
            }
        }

        (['spirit_wood', 'spirit_stone', 'herb'] as ResourceKind[]).forEach((kind) => {
            const layer = this.resourceLayers.get(kind);
            if (!layer) {
                return;
            }

            for (let row = 0; row < this.mapRows; row += 1) {
                for (let col = 0; col < this.mapColumns; col += 1) {
                    if (layer.getTileGIDAt(col, row) <= 0) {
                        continue;
                    }

                    this.resourceNodes.set(this.getTileKey(col, row), {
                        kind,
                        tile: { col, row },
                        designated: false,
                        state: 'available',
                        remainingCharges: RESOURCE_RULES[kind].maxCharges,
                        maxCharges: RESOURCE_RULES[kind].maxCharges,
                        regenSeconds: RESOURCE_RULES[kind].regenSeconds,
                        regenTimerSeconds: 0,
                    });
                }
            }
        });

        for (let row = 0; row < this.mapRows; row += 1) {
            for (let col = 0; col < this.mapColumns; col += 1) {
                const key = this.getTileKey(col, row);
                if (!this.blockedTiles.has(key) && !this.roadTiles.has(key) && !this.resourceNodes.has(key)) {
                    this.buildableTileCount += 1;
                }
            }
        }

        this.logRuntime('INFO', 'MAP', 'map.logic_layers_ready', 'TileMap 逻辑层解析完成', {
            roadCount: this.roadTiles.size,
            blockedCount: this.blockedTiles.size,
            resourceCount: this.resourceNodes.size,
            buildableCount: this.buildableTileCount,
        });
    }

    private spawnInitialWorld(): void {
        this.stockpile = EMPTY_STOCKPILE();
        this.buildingEntities.clear();
        this.resetResourceNodesForNewRun();
        this.resetSessionProgress();

        const mainHall = this.addBuildingEntity(
            BUILDING_DEFINITIONS.find((definition) => definition.id === 'main_hall')!,
            INITIAL_MAIN_HALL_TILE,
            'active',
        );
        mainHall.level = 1;

        const ruinOrigin = this.findNearestBuildableTile(RUIN_WAREHOUSE_PREFERRED_TILE) ?? { col: 8, row: 8 };
        const ruinWarehouse = this.addBuildingEntity(
            BUILDING_DEFINITIONS.find((definition) => definition.id === 'warehouse')!,
            ruinOrigin,
            'damaged',
        );
        ruinWarehouse.currentHp = 0;
        ruinWarehouse.level = 1;
        ruinWarehouse.pendingAction = null;
        ruinWarehouse.pendingLevel = null;
        this.sessionRuinBuildingId = ruinWarehouse.id;

        this.disciple.tile = { ...INITIAL_DISCIPLE_TILE };
        this.disciple.currentTask = null;
        this.disciple.carrying = null;
        this.disciple.path = [];
        this.disciple.pathIndex = 0;
        this.disciple.visualState = 'idle';
        this.disciple.currentHp = this.disciple.model.stats.maxHp;
        this.disciple.attackCooldownSeconds = 0;
        this.disciple.hitFlashSeconds = 0;

        this.hostileNpc.active = false;
        this.hostileNpc.tile = { col: 0, row: 0 };
        this.hostileNpc.worldPosition.set(0, 0, 0);
        this.hostileNpc.path = [];
        this.hostileNpc.pathIndex = 0;
        this.hostileNpc.visualState = 'idle';
        this.hostileNpc.currentHp = this.hostileNpc.model.stats.maxHp;
        this.hostileNpc.attackCooldownSeconds = 0;
        this.hostileNpc.hitFlashSeconds = 0;
        this.hostileNpc.respawnTimerSeconds = HOSTILE_RESPAWN_SECONDS;
        this.hostileNpc.targetBuildingId = null;

        this.designateSessionStarterResources();
        this.setSessionPhase(
            'clear_ruin',
            '长按废弃仓房并拆除，为护山台腾出位置',
            '新档目标：先拆除废弃仓房，再建护山台',
        );

        this.logRuntime('INFO', 'BOOT', 'world.spawn_initial', '初始世界状态已生成', {
            mainHallTile: this.formatTile(mainHall.origin),
            discipleTile: this.formatTile(this.disciple.tile),
            ruinBuildingTile: this.formatTile(ruinWarehouse.origin),
        });
    }

    private resetSessionProgress(): void {
        this.sessionPhase = 'clear_ruin';
        this.sessionOutcome = 'in_progress';
        this.sessionElapsedSeconds = 0;
        this.sessionObjectiveText = '';
        this.sessionGuardTowerId = null;
        this.sessionRuinBuildingId = null;
        this.firstRaidTriggered = false;
        this.firstRaidResolved = false;
        this.sessionBuildPanelPrompted = false;
        this.closeBuildPanel();
        this.closeRadialMenu();
        this.clearBuildPlacement();
    }

    private designateSessionStarterResources(): void {
        for (const resource of this.resourceNodes.values()) {
            resource.designated = true;
        }
    }

    private advanceSessionLoop(deltaTime: number): void {
        if (this.sessionOutcome !== 'in_progress') {
            return;
        }

        this.sessionElapsedSeconds += deltaTime;
        if (this.sessionElapsedSeconds >= SESSION_TARGET_SECONDS) {
            this.finishSession('defeat', '超过 10 分钟仍未完成本轮收口，短会话失败');
            return;
        }

        const mainHall = [...this.buildingEntities.values()].find((building) => building.definition.id === 'main_hall') ?? null;
        if (mainHall && mainHall.currentHp <= 0) {
            this.finishSession('defeat', '主殿已失守，本轮短会话失败');
            return;
        }

        if (this.disciple.currentHp <= 0) {
            this.finishSession('defeat', '弟子已倒下，本轮短会话失败');
            return;
        }

        const guardTower = this.getSessionGuardTower();
        const ruinBuilding = this.getSessionRuinBuilding();

        switch (this.sessionPhase) {
            case 'clear_ruin':
                if (!ruinBuilding) {
                    this.setSessionPhase(
                        'place_guard_tower',
                        '在清出的地块放置并建成护山台',
                        '废弃仓房已清出，请在空地建造护山台',
                    );
                }
                return;
            case 'place_guard_tower':
                if (!guardTower) {
                    return;
                }

                if (guardTower.state === 'active' && guardTower.level >= 1) {
                    this.setSessionPhase(
                        'upgrade_guard_tower',
                        '长按护山台执行升级，升到 Lv.2 后准备敌袭',
                        '护山台已落成，下一步请长按它并升级到 Lv.2',
                    );
                }
                return;
            case 'upgrade_guard_tower':
                if (!guardTower) {
                    this.setSessionPhase(
                        'place_guard_tower',
                        '重新放置并建成护山台',
                        '护山台不存在，请重新放置一座护山台',
                    );
                    return;
                }

                if (guardTower.level >= 2 && guardTower.state === 'active') {
                    this.hostileNpc.respawnTimerSeconds = FIRST_RAID_PREP_SECONDS;
                    this.setSessionPhase(
                        'raid_countdown',
                        '首波敌袭即将到达，保持库存并准备守住护山台',
                        `护山台已升至 Lv.${guardTower.level}，首波敌袭将在 ${FIRST_RAID_PREP_SECONDS}s 后到达`,
                    );
                }
                return;
            case 'raid_countdown':
                this.hostileNpc.respawnTimerSeconds = Math.max(0, this.hostileNpc.respawnTimerSeconds - deltaTime);
                if (this.hostileNpc.respawnTimerSeconds <= 0 && !this.hostileNpc.active && !this.firstRaidTriggered) {
                    this.firstRaidTriggered = true;
                    this.spawnHostileNpc();
                    this.setSessionPhase('defend', '守住首波敌袭，不要让主殿瘫痪', '首波敌袭已到达，守住主殿与护山台');
                }
                return;
            case 'defend':
                if (this.firstRaidTriggered && !this.hostileNpc.active) {
                    this.firstRaidResolved = true;
                    this.setSessionPhase('recover', '修复受损建筑并恢复宗门运转', '首波敌袭已被击退，尽快修复受损建筑');
                }
                return;
            case 'recover': {
                if (!guardTower) {
                    this.finishSession('defeat', '护山台已不复存在，本轮短会话失败');
                    return;
                }

                const damagedBuildings = [...this.buildingEntities.values()].filter(
                    (building) => building.state === 'damaged' && building.currentHp < this.getBuildingMaxHp(building),
                );

                if (
                    this.firstRaidResolved &&
                    damagedBuildings.length === 0 &&
                    guardTower.state === 'active' &&
                    guardTower.level >= 2 &&
                    guardTower.currentHp >= this.getBuildingMaxHp(guardTower)
                ) {
                    this.finishSession('victory', '首波敌袭已守住且宗门修复完毕，本轮短会话达成');
                }
                return;
            }
            case 'victory':
            case 'defeat':
            default:
                return;
        }
    }

    private finishSession(outcome: SessionOutcome, message: string): void {
        if (this.sessionOutcome === outcome && this.lastMessage === message) {
            return;
        }

        this.sessionOutcome = outcome;
        this.sessionPhase = outcome === 'victory' ? 'victory' : 'defeat';
        this.sessionObjectiveText = outcome === 'victory' ? '本轮短会话已完成，可继续自由观察' : '短会话失败，请重新开局再试';
        this.hostileNpc.active = false;
        this.hostileNpc.path = [];
        this.hostileNpc.pathIndex = 0;
        this.hostileNpc.targetBuildingId = null;
        this.hostileNpc.attackCooldownSeconds = 0;
        this.disciple.currentTask = null;
        this.disciple.path = [];
        this.disciple.pathIndex = 0;
        this.closeBuildPanel();
        this.clearBuildPlacement();
        this.logRuntime('INFO', 'BOOT', 'session.finished', '本地短会话结果已确定', {
            outcome,
            elapsedSeconds: Number(this.sessionElapsedSeconds.toFixed(2)),
            message,
        });
        this.setMessage(message);
        this.refreshHostileNpcToken();
        this.refreshDiscipleToken();
        this.syncSessionProgressToAuthority('finish_session');
    }

    private setSessionPhase(phase: SessionPhase, objective: string, message?: string): void {
        const previousPhase = this.sessionPhase;
        this.sessionPhase = phase;
        this.sessionObjectiveText = objective;
        this.logRuntime('INFO', 'BOOT', 'session.phase_change', '本地短会话阶段已更新', {
            from: previousPhase,
            to: phase,
            objective,
        });

        if (phase === 'place_guard_tower' && !this.sessionBuildPanelPrompted) {
            this.sessionBuildPanelPrompted = true;
            const focusTile =
                this.getSessionRuinBuilding()?.origin ??
                this.findNearestBuildableTile(RUIN_WAREHOUSE_PREFERRED_TILE) ??
                this.findScreenCenterTile();
            if (focusTile) {
                this.openBuildPanel(focusTile);
            }
        }

        if (message) {
            this.setMessage(message);
        }
        this.syncSessionProgressToAuthority('phase_change');
    }

    private getSessionRuinBuilding(): BuildingEntity | null {
        return this.sessionRuinBuildingId ? this.buildingEntities.get(this.sessionRuinBuildingId) ?? null : null;
    }

    private getSessionGuardTower(): BuildingEntity | null {
        return this.sessionGuardTowerId ? this.buildingEntities.get(this.sessionGuardTowerId) ?? null : null;
    }

    private isSessionRuinBuilding(building: BuildingEntity): boolean {
        return building.id === this.sessionRuinBuildingId;
    }

    private getSessionPriorityBuilding(): BuildingEntity | null {
        if (this.sessionPhase === 'place_guard_tower' || this.sessionPhase === 'upgrade_guard_tower') {
            return this.getSessionGuardTower();
        }

        return null;
    }

    private addStockpile(amount: Stockpile): void {
        (Object.keys(amount) as ResourceKind[]).forEach((key) => {
            this.stockpile[key] += amount[key];
        });
    }

    private getDemolitionYield(building: BuildingEntity): Stockpile {
        if (this.isSessionRuinBuilding(building)) {
            return { spirit_wood: 1, spirit_stone: 1, herb: 0 };
        }

        if (building.pendingAction === 'upgrade') {
            return { spirit_wood: 0, spirit_stone: 0, herb: 0 };
        }

        return {
            spirit_wood: building.level >= 2 && building.definition.id !== 'herb_garden' ? 1 : 0,
            spirit_stone: building.level >= 2 && building.definition.id !== 'disciple_quarters' ? 1 : 0,
            herb: building.definition.id === 'herb_garden' ? 1 : 0,
        };
    }

    private resetResourceNodesForNewRun(): void {
        for (const resource of this.resourceNodes.values()) {
            resource.designated = false;
            resource.state = 'available';
            resource.remainingCharges = resource.maxCharges;
            resource.regenTimerSeconds = 0;
        }
    }

    private isResourceHarvestable(resource: ResourceNode): boolean {
        return resource.state === 'available' && resource.remainingCharges > 0;
    }

    private advanceResourceRespawns(deltaTime: number): void {
        let needsRefresh = false;

        for (const resource of this.resourceNodes.values()) {
            if (resource.state !== 'regenerating') {
                continue;
            }

            const previousRoundedSeconds = Math.ceil(resource.regenTimerSeconds);
            resource.regenTimerSeconds = Math.max(0, resource.regenTimerSeconds - deltaTime);
            const nextRoundedSeconds = Math.ceil(resource.regenTimerSeconds);

            if (nextRoundedSeconds !== previousRoundedSeconds) {
                needsRefresh = true;
            }

            if (resource.regenTimerSeconds > 0) {
                continue;
            }

            resource.state = 'available';
            resource.remainingCharges = resource.maxCharges;
            resource.regenTimerSeconds = 0;
            needsRefresh = true;
            this.logRuntime('INFO', 'RESOURCE', 'resource.regenerated', '资源节点已刷新，可重新采集', {
                tile: this.formatTile(resource.tile),
                resourceKind: resource.kind,
                remainingCharges: resource.remainingCharges,
                maxCharges: resource.maxCharges,
                designated: resource.designated,
            });
            this.setMessage(
                resource.designated
                    ? `${RESOURCE_DISPLAY[resource.kind].title} 已刷新，弟子可继续采集`
                    : `${RESOURCE_DISPLAY[resource.kind].title} 已刷新`,
            );
        }

        if (needsRefresh) {
            this.refreshResourceMarkers();
        }
    }

    private advanceCombatFeedback(deltaTime: number): void {
        let needsBuildingRefresh = false;

        for (const building of this.buildingEntities.values()) {
            if (building.damageFlashSeconds > 0) {
                building.damageFlashSeconds = Math.max(0, building.damageFlashSeconds - deltaTime);
                needsBuildingRefresh ||= building.damageFlashSeconds <= 0;
            }

            if (building.attackCooldownSeconds > 0) {
                building.attackCooldownSeconds = Math.max(0, building.attackCooldownSeconds - deltaTime);
            }
        }

        if (this.disciple.hitFlashSeconds > 0) {
            this.disciple.hitFlashSeconds = Math.max(0, this.disciple.hitFlashSeconds - deltaTime);
        }
        if (this.disciple.attackCooldownSeconds > 0) {
            this.disciple.attackCooldownSeconds = Math.max(0, this.disciple.attackCooldownSeconds - deltaTime);
        }

        if (this.hostileNpc.hitFlashSeconds > 0) {
            this.hostileNpc.hitFlashSeconds = Math.max(0, this.hostileNpc.hitFlashSeconds - deltaTime);
        }
        if (this.hostileNpc.attackCooldownSeconds > 0) {
            this.hostileNpc.attackCooldownSeconds = Math.max(0, this.hostileNpc.attackCooldownSeconds - deltaTime);
        }

        if (needsBuildingRefresh) {
            this.refreshBuildings();
        }
    }

    private advanceHostileNpc(deltaTime: number): void {
        if (!this.hostileNpc.active) {
            this.refreshHostileNpcToken();
            return;
        }

        const targetBuilding = this.getRaidTargetBuilding();
        if (!targetBuilding) {
            this.retireHostileNpc('没有可攻击目标，当前威胁已撤离');
            return;
        }

        this.hostileNpc.targetBuildingId = targetBuilding.id;
        const attackTile = this.findBuildingStandTileFromTile(targetBuilding, this.hostileNpc.tile);
        if (!attackTile) {
            this.retireHostileNpc('外敌失去可到达目标，当前威胁已撤离');
            return;
        }

        if (this.hostileNpc.tile.col !== attackTile.col || this.hostileNpc.tile.row !== attackTile.row) {
            const needsPath =
                this.hostileNpc.path.length === 0 ||
                this.hostileNpc.pathIndex >= this.hostileNpc.path.length ||
                !this.isPathStillLeadingToTile(this.hostileNpc.path, attackTile);

            if (needsPath) {
                const path = this.findPath(this.hostileNpc.tile, attackTile);
                if (!path) {
                    this.retireHostileNpc('外敌路径中断，当前威胁已撤离');
                    return;
                }
                this.hostileNpc.path = path;
                this.hostileNpc.pathIndex = 0;
            }

            this.hostileNpc.visualState = 'moving';
            this.advancePathingUnit(this.hostileNpc, deltaTime, attackTile);
            this.refreshHostileNpcToken();
            return;
        }

        this.hostileNpc.visualState = 'attacking';
        if (this.hostileNpc.attackCooldownSeconds > 0) {
            this.refreshHostileNpcToken();
            return;
        }

        this.hostileNpc.attackCooldownSeconds = this.hostileNpc.model.stats.attackInterval;
        this.applyDamageToBuilding(targetBuilding, this.hostileNpc.model.stats.attackPower, `${this.hostileNpc.name}袭扰`);
        this.logRuntime('INFO', 'COMBAT', 'combat.hostile_attack', '外敌命中建筑', {
            hostileId: this.hostileNpc.id,
            buildingId: targetBuilding.id,
            buildingType: targetBuilding.definition.id,
            hp: `${Math.ceil(targetBuilding.currentHp)}/${this.getBuildingMaxHp(targetBuilding)}`,
        });
        this.refreshHostileNpcToken();
    }

    private shouldSpawnHostileNpc(): boolean {
        return this.sessionPhase === 'raid_countdown' && !this.firstRaidTriggered && this.sessionOutcome === 'in_progress';
    }

    private spawnHostileNpc(): void {
        const targetBuilding = this.getRaidTargetBuilding();
        if (!targetBuilding) {
            this.hostileNpc.respawnTimerSeconds = HOSTILE_RESPAWN_SECONDS;
            return;
        }

        const attackTile = this.findBuildingStandTileFromTile(targetBuilding, targetBuilding.origin);
        const spawnTile = this.findEdgeSpawnTile(attackTile ?? targetBuilding.origin);
        const path = this.findPath(spawnTile, attackTile ?? targetBuilding.origin);

        this.hostileNpc.active = true;
        this.hostileNpc.tile = { ...spawnTile };
        this.hostileNpc.worldPosition = this.getTileCenter(spawnTile);
        this.hostileNpc.path = path ?? [];
        this.hostileNpc.pathIndex = 0;
        this.hostileNpc.visualState = 'moving';
        this.hostileNpc.currentHp = this.hostileNpc.model.stats.maxHp;
        this.hostileNpc.attackCooldownSeconds = 0;
        this.hostileNpc.hitFlashSeconds = 0;
        this.hostileNpc.targetBuildingId = targetBuilding.id;
        this.logRuntime('INFO', 'COMBAT', 'combat.hostile_spawned', '外敌已从地图边缘进入', {
            hostileId: this.hostileNpc.id,
            spawnTile: this.formatTile(spawnTile),
            targetBuildingId: targetBuilding.id,
            targetBuildingType: targetBuilding.definition.id,
        });
        this.setMessage(`${this.hostileNpc.name} 已逼近 ${targetBuilding.definition.label}`);
        this.refreshHostileNpcToken();
    }

    private retireHostileNpc(message: string): void {
        this.hostileNpc.active = false;
        this.hostileNpc.path = [];
        this.hostileNpc.pathIndex = 0;
        this.hostileNpc.targetBuildingId = null;
        this.hostileNpc.attackCooldownSeconds = 0;
        this.hostileNpc.visualState = 'idle';
        this.hostileNpc.respawnTimerSeconds = 0;
        this.setMessage(message);
        this.refreshHostileNpcToken();
    }

    private getRaidTargetBuilding(): BuildingEntity | null {
        const preferred = [...this.buildingEntities.values()].filter(
            (building) =>
                building.definition.id !== 'main_hall' &&
                (building.state === 'active' || building.state === 'damaged') &&
                building.currentHp > 0,
        );
        if (preferred.length > 0) {
            preferred.sort((left, right) => left.currentHp - right.currentHp);
            return preferred[0];
        }

        return (
            [...this.buildingEntities.values()].find(
                (building) => building.definition.id === 'main_hall' && building.currentHp > 0,
            ) ?? null
        );
    }

    private findEdgeSpawnTile(targetTile: TileCoord): TileCoord {
        const edgeTiles: TileCoord[] = [];

        for (let col = 0; col < this.mapColumns; col += 1) {
            edgeTiles.push({ col, row: 0 });
            edgeTiles.push({ col, row: this.mapRows - 1 });
        }
        for (let row = 1; row < this.mapRows - 1; row += 1) {
            edgeTiles.push({ col: 0, row });
            edgeTiles.push({ col: this.mapColumns - 1, row });
        }

        const walkableEdges = edgeTiles.filter((tile) => this.isWalkable(tile));
        if (walkableEdges.length === 0) {
            return { ...INITIAL_DISCIPLE_TILE };
        }

        walkableEdges.sort((left, right) => manhattanDistance(left, targetTile) - manhattanDistance(right, targetTile));
        return walkableEdges[0];
    }

    private advanceGuardTowers(deltaTime: number): void {
        if (!this.hostileNpc.active) {
            return;
        }

        for (const building of this.buildingEntities.values()) {
            if (
                !this.getBuildingGuardProfile(building) ||
                building.currentHp <= 0 ||
                (building.state !== 'active' && building.state !== 'damaged')
            ) {
                continue;
            }

            const profile = this.getBuildingGuardProfile(building)!;
            if (building.attackCooldownSeconds > 0) {
                continue;
            }

            const distance = this.getDistanceFromBuildingToTile(building, this.hostileNpc.tile);
            if (distance > profile.rangeTiles) {
                continue;
            }

            building.attackCooldownSeconds = profile.attackInterval;
            this.applyDamageToHostile(this.hostileNpc, profile.attackPower, `${building.definition.label}守御`);
            this.logRuntime('INFO', 'COMBAT', 'combat.guard_tower_attack', '护山台命中外敌', {
                buildingId: building.id,
                hostileId: this.hostileNpc.id,
                hostileHp: `${Math.ceil(this.hostileNpc.currentHp)}/${this.hostileNpc.model.stats.maxHp}`,
            });
        }
    }

    private applyDamageToBuilding(building: BuildingEntity, rawDamage: number, source: string): number {
        const maxHp = this.getBuildingMaxHp(building);
        const damage = getMitigatedDamage(rawDamage, this.getBuildingStructureDefense(building));
        building.currentHp = Math.max(0, building.currentHp - damage);
        building.damageFlashSeconds = BUILDING_HIT_FLASH_SECONDS;
        if (building.state === 'active' && building.currentHp < maxHp) {
            this.setBuildingState(building, 'damaged', 'combat_damage_taken');
        }
        this.logRuntime('INFO', 'COMBAT', 'combat.building_damaged', '建筑承受袭击', {
            buildingId: building.id,
            buildingType: building.definition.id,
            source,
            damage,
            hp: `${Math.ceil(building.currentHp)}/${maxHp}`,
        });
        this.refreshBuildings();
        this.setMessage(
            building.currentHp <= 0
                ? `${building.definition.label} 被${source}打至瘫痪，待修复`
                : `${building.definition.label} 受击 -${damage}，HP ${Math.ceil(building.currentHp)}/${maxHp}`,
        );
        return damage;
    }

    private applyDamageToHostile(hostile: HostileNpcEntity, rawDamage: number, source: string): number {
        const damage = getMitigatedDamage(rawDamage, hostile.model.stats.defense);
        hostile.currentHp = Math.max(0, hostile.currentHp - damage);
        hostile.hitFlashSeconds = UNIT_HIT_FLASH_SECONDS;

        if (hostile.currentHp <= 0) {
            this.logRuntime('INFO', 'COMBAT', 'combat.hostile_defeated', '外敌已被击退', {
                hostileId: hostile.id,
                source,
            });
            this.retireHostileNpc(`${source} 击退了 ${hostile.name}`);
        }

        return damage;
    }

    private isPathStillLeadingToTile(path: TileCoord[], targetTile: TileCoord): boolean {
        if (path.length === 0) {
            return false;
        }

        const tail = path[path.length - 1];
        return tail.col === targetTile.col && tail.row === targetTile.row;
    }

    private getDistanceFromBuildingToTile(building: BuildingEntity, tile: TileCoord): number {
        const footprint = this.getFootprintTiles(building.origin, building.definition.width, building.definition.height);
        let best = Number.POSITIVE_INFINITY;

        for (const cell of footprint) {
            best = Math.min(best, manhattanDistance(cell, tile));
        }

        return best;
    }

    private addBuildingEntity(definition: BuildingDefinition, origin: TileCoord, state: BuildingState): BuildingEntity {
        const entity: BuildingEntity = {
            id: `${definition.id}-${this.nextBuildingId}`,
            definition,
            origin: { ...origin },
            state,
            supplied: EMPTY_STOCKPILE(),
            level: 1,
            markedForDemolition: false,
            currentHp: definition.maxHp,
            damageFlashSeconds: 0,
            attackCooldownSeconds: 0,
            pendingAction: state === 'active' ? null : 'build',
            pendingLevel: state === 'active' ? 1 : null,
        };

        this.nextBuildingId += 1;
        entity.currentHp = this.getBuildingMaxHp(entity);
        this.buildingEntities.set(entity.id, entity);
        return entity;
    }

    private toggleGatherDesignation(tile: TileCoord): void {
        const resource = this.resourceNodes.get(this.getTileKey(tile.col, tile.row));
        if (!resource) {
            this.logRuntime('WARN', 'RESOURCE', 'resource.designation_missing', '尝试标记不存在的资源格', {
                tile: this.formatTile(tile),
            });
            this.setMessage('该地块没有可采集资源');
            return;
        }

        resource.designated = !resource.designated;
        this.refreshResourceMarkers();
        this.logRuntime('INFO', 'RESOURCE', 'resource.designation_toggle', '资源采集标记已切换', {
            tile: this.formatTile(tile),
            resourceKind: resource.kind,
            designated: resource.designated,
            resourceState: resource.state,
            remainingCharges: resource.remainingCharges,
        });
        this.setMessage(
            resource.designated
                ? resource.state === 'available'
                    ? `已标记 ${RESOURCE_DISPLAY[resource.kind].title} 采集点，剩余 ${resource.remainingCharges}/${resource.maxCharges}`
                    : `已标记 ${RESOURCE_DISPLAY[resource.kind].title} 采集点，当前待刷新 ${Math.ceil(resource.regenTimerSeconds)}s`
                : `已取消 ${RESOURCE_DISPLAY[resource.kind].title} 采集点`,
        );
    }

    private toggleDemolition(tile: TileCoord): void {
        const building = this.getBuildingAtTile(tile);
        if (!building || building.definition.id === 'main_hall') {
            this.logRuntime('WARN', 'BUILD', 'build.demolition_invalid', '当前点击位置没有可拆除建筑', {
                tile: this.formatTile(tile),
            });
            this.setMessage('该位置没有可拆除建筑');
            return;
        }

        if (this.authorityConnected) {
            this.executeAuthorityCommand(
                {
                    name: 'toggle_demolition',
                    payload: {
                        buildingId: building.id,
                    },
                },
                {
                    commandKey: `toggle_demolition:${building.id}`,
                },
            );
            return;
        }

        building.markedForDemolition = !building.markedForDemolition;
        this.refreshBuildings();
        this.logRuntime('INFO', 'BUILD', 'build.demolition_toggle', '建筑拆除标记已切换', {
            tile: this.formatTile(tile),
            buildingId: building.id,
            buildingType: building.definition.id,
            markedForDemolition: building.markedForDemolition,
        });
        this.setMessage(building.markedForDemolition ? `已标记拆除 ${building.definition.label}` : `已取消拆除 ${building.definition.label}`);
    }

    private confirmBuildPlacement(): void {
        if (!this.buildPlacement) {
            return;
        }

        const { definition, origin } = this.buildPlacement;
        if (!this.isPlacementValid(origin, definition)) {
            this.logRuntime('WARN', 'BUILD', 'build.confirm_rejected', '蓝图确认失败，当前位置不可放置', {
                buildingId: definition.id,
                origin: this.formatTile(origin),
            });
            this.setMessage(`${definition.label} 当前位置不可放置`);
            return;
        }

        if (this.authorityConnected) {
            this.executeAuthorityCommand(
                {
                    name: 'place_building',
                    payload: {
                        buildingType: definition.id as AuthorityBuildingType,
                        origin,
                    },
                },
                {
                    commandKey: `place_building:${definition.id}:${origin.col},${origin.row}`,
                    onAccepted: () => {
                        this.clearBuildPlacement();
                        this.setMode('browse');
                    },
                },
            );
            return;
        }

        const building = this.addBuildingEntity(definition, origin, 'planned');
        if (definition.id === 'guard_tower' && !this.sessionGuardTowerId) {
            this.sessionGuardTowerId = building.id;
        }
        this.refreshBuildings();
        this.logRuntime('INFO', 'BUILD', 'build.confirmed', '已下达建筑蓝图', {
            buildingId: building.id,
            buildingType: definition.id,
            origin: this.formatTile(origin),
            cost: this.getCostText(definition.cost),
        });
        this.setMessage(`已下达 ${definition.label} 蓝图，等待弟子备料和施工`);
        this.clearBuildPlacement();
        this.setMode('browse');
    }

    private refreshResourceMarkers(): void {
        for (const child of [...this.resourceRoot.children]) {
            child.destroy();
        }

        for (const resource of this.resourceNodes.values()) {
            const isHarvestable = this.isResourceHarvestable(resource);
            const center = this.getTileCenter(resource.tile);
            const spriteFrame = this.getVisualAssetFrame(this.getResourceVisualAssetId(resource.kind));
            const root = new Node(`Resource-${resource.kind}-${resource.tile.col}-${resource.tile.row}`);
            root.setParent(this.resourceRoot);
            root.setPosition(center.x, center.y, 0);

            const graphics = root.addComponent(Graphics);
            let markerWidth = 72;
            let markerHeight = 86;

            if (spriteFrame) {
                const { width, height } = this.getSpriteFrameDimensions(spriteFrame);
                markerWidth = Math.max(96, width + 18);
                markerHeight = Math.max(96, height + 30);
                this.configureNodeSize(root, markerWidth, markerHeight);

                const artNode = new Node('Art');
                artNode.setParent(root);
                artNode.setPosition(0, -this.halfTileHeight, 0);
                this.configureSpriteNode(artNode, spriteFrame, 0.5, 0, this.getResourceArtColor(resource, isHarvestable));

                graphics.fillColor = new Color(18, 24, 30, 64);
                graphics.circle(0, -this.halfTileHeight + 12, 18);
                graphics.fill();

                graphics.lineWidth = resource.designated ? 3 : 2;
                graphics.strokeColor = resource.designated
                    ? new Color(255, 229, 158, 255)
                    : isHarvestable
                      ? new Color(64, 74, 84, 188)
                      : new Color(118, 128, 142, 188);
                graphics.fillColor = resource.designated
                    ? new Color(255, 224, 126, 38)
                    : isHarvestable
                      ? new Color(22, 28, 36, 18)
                      : new Color(68, 74, 82, 36);
                graphics.roundRect(-(width * 0.5 + 6), -this.halfTileHeight - 4, width + 12, height + 8, 18);
                graphics.fill();
                graphics.stroke();
            } else {
                this.configureNodeSize(root, 72, 72);
                markerHeight = 72;
                graphics.lineWidth = 2;
                graphics.fillColor = isHarvestable
                    ? resource.designated
                        ? withAlpha(RESOURCE_DISPLAY[resource.kind].color, 240)
                        : withAlpha(RESOURCE_DISPLAY[resource.kind].color, 184)
                    : resource.designated
                      ? new Color(92, 104, 120, 214)
                      : new Color(70, 78, 90, 188);
                graphics.strokeColor = isHarvestable
                    ? resource.designated
                        ? new Color(255, 247, 198, 255)
                        : new Color(34, 44, 54, 255)
                    : resource.designated
                      ? new Color(255, 231, 176, 255)
                      : new Color(94, 104, 116, 255);
                graphics.circle(0, 0, isHarvestable ? (resource.designated ? 24 : 20) : 16);
                graphics.fill();
                graphics.stroke();

                if (resource.designated) {
                    graphics.strokeColor = new Color(255, 235, 168, 255);
                    graphics.lineWidth = 2;
                    graphics.circle(0, 0, 28);
                    graphics.stroke();
                }

                if (!isHarvestable) {
                    graphics.strokeColor = new Color(210, 218, 228, 220);
                    graphics.lineWidth = 2;
                    graphics.moveTo(-10, 10);
                    graphics.lineTo(10, -10);
                    graphics.moveTo(-10, -10);
                    graphics.lineTo(10, 10);
                    graphics.stroke();
                }

                const labelNode = new Node('Label');
                labelNode.setParent(root);
                const labelTransform = labelNode.addComponent(UITransform);
                labelTransform.setContentSize(40, 28);
                const label = labelNode.addComponent(Label);
                label.string = RESOURCE_DISPLAY[resource.kind].short;
                label.fontSize = 22;
                label.lineHeight = 24;
                label.color = isHarvestable ? new Color(245, 248, 252, 255) : new Color(214, 220, 228, 255);
            }

            const statusNode = new Node('Status');
            statusNode.setParent(root);
            statusNode.setPosition(0, -(markerHeight * 0.5) + 12, 0);
            const statusTransform = statusNode.addComponent(UITransform);
            statusTransform.setContentSize(68, 22);
            const statusLabel = statusNode.addComponent(Label);
            statusLabel.string = isHarvestable
                ? `${resource.remainingCharges}/${resource.maxCharges}`
                : `待${Math.ceil(resource.regenTimerSeconds)}s`;
            statusLabel.fontSize = 14;
            statusLabel.lineHeight = 16;
            statusLabel.color = isHarvestable ? new Color(255, 247, 208, 255) : new Color(214, 220, 228, 255);
        }
    }

    private refreshBuildings(): void {
        for (const child of [...this.buildingRoot.children]) {
            child.destroy();
        }

        const objectiveBuildingId = this.getSessionGuidance().focusBuildingId;
        const buildings = [...this.buildingEntities.values()].sort((left, right) => {
            const leftDepth = left.origin.col + left.origin.row + left.definition.height;
            const rightDepth = right.origin.col + right.origin.row + right.definition.height;
            return leftDepth - rightDepth;
        });

        for (const building of buildings) {
            this.createBuildingNode(building, objectiveBuildingId === building.id);
        }
    }

    private createBuildingNode(building: BuildingEntity, isObjectiveBuilding: boolean): void {
        const tiles = this.getFootprintTiles(building.origin, building.definition.width, building.definition.height);
        const frame = this.getFootprintFrame(tiles);
        const spriteFrame = this.getVisualAssetFrame(this.getBuildingVisualAssetId(building.definition.id));
        const spriteSize = spriteFrame ? this.getSpriteFrameDimensions(spriteFrame) : null;
        const root = new Node(`Building-${building.id}`);
        root.setParent(this.buildingRoot);
        root.setPosition(frame.center.x, frame.center.y, 0);
        this.configureNodeSize(
            root,
            Math.max(frame.width + 36, spriteSize ? spriteSize.width + 28 : frame.width + 24),
            Math.max(frame.height + 112, spriteSize ? spriteSize.height + 56 : frame.height + 108),
        );

        const graphics = root.addComponent(Graphics);
        const palette = this.getBuildingPalette(building);
        graphics.lineWidth = 2;

        if (spriteFrame) {
            for (const tile of tiles) {
                const center = this.getTileCenter(tile);
                this.drawDiamond(
                    graphics,
                    center.x - frame.center.x,
                    center.y - frame.center.y,
                    withAlpha(palette.fill, building.state === 'active' ? 72 : 118),
                    withAlpha(palette.stroke, building.state === 'active' ? 132 : 224),
                    10,
                );
            }

            if (isObjectiveBuilding) {
                for (const tile of tiles) {
                    const center = this.getTileCenter(tile);
                    this.drawDiamond(
                        graphics,
                        center.x - frame.center.x,
                        center.y - frame.center.y,
                        new Color(255, 215, 64, 16),
                        new Color(255, 231, 154, 255),
                        2,
                    );
                }
            }

            graphics.fillColor = new Color(20, 26, 34, 60);
            graphics.roundRect(-(frame.width * 0.5), -(frame.height * 0.5) + 10, frame.width, 18, 9);
            graphics.fill();

            const artNode = new Node('Art');
            artNode.setParent(root);
            artNode.setPosition(0, -frame.height * 0.5, 0);
            this.configureSpriteNode(artNode, spriteFrame, 0.5, 0, this.getBuildingArtColor(building));
        } else {
            for (const tile of tiles) {
                const center = this.getTileCenter(tile);
                this.drawDiamond(
                    graphics,
                    center.x - frame.center.x,
                    center.y - frame.center.y,
                    palette.fill,
                    palette.stroke,
                    4,
                );
            }
        }

        const signalFrame = this.getBuildingSignalFrame(building);
        const signalText = !signalFrame ? this.getBuildingSignalFallbackText(building) : null;
        if (signalFrame || signalText) {
            const signalNode = new Node('Signal');
            signalNode.setParent(root);
            signalNode.setPosition(frame.width * 0.5 - 12, frame.height * 0.5 + 18, 0);
            this.configureNodeSize(signalNode, 54, 54);

            if (signalFrame) {
                this.configureSpriteNode(signalNode, signalFrame, 0.5, 0.5);
                signalNode.setScale(0.68, 0.68, 1);
            } else if (signalText) {
                const signalBackdrop = signalNode.addComponent(Graphics);
                signalBackdrop.fillColor = new Color(34, 42, 54, 236);
                signalBackdrop.strokeColor = new Color(228, 236, 246, 255);
                signalBackdrop.lineWidth = 2;
                signalBackdrop.circle(0, 0, 18);
                signalBackdrop.fill();
                signalBackdrop.stroke();

                const signalLabelNode = new Node('SignalLabel');
                signalLabelNode.setParent(signalNode);
                this.configureNodeSize(signalLabelNode, 28, 24);
                const signalLabel = signalLabelNode.addComponent(Label);
                signalLabel.string = signalText;
                signalLabel.fontSize = 18;
                signalLabel.lineHeight = 20;
                signalLabel.color = new Color(248, 250, 253, 255);
            }
        }

        if (building.markedForDemolition) {
            graphics.strokeColor = new Color(255, 102, 102, 255);
            graphics.lineWidth = 4;
            graphics.moveTo(-frame.width * 0.5, frame.height * 0.5);
            graphics.lineTo(frame.width * 0.5, -frame.height * 0.5);
            graphics.moveTo(-frame.width * 0.5, -frame.height * 0.5);
            graphics.lineTo(frame.width * 0.5, frame.height * 0.5);
            graphics.stroke();
        }

        const labelNode = new Node('Label');
        labelNode.setParent(root);
        const isCompactState = building.state === 'active' || building.state === 'constructing' || building.state === 'supplied';
        labelNode.setPosition(0, -frame.height * 0.5 + 18, 0);
        const labelWidth = isCompactState ? 124 : 156;
        const labelHeight = isCompactState ? 34 : 58;
        const labelTransform = labelNode.addComponent(UITransform);
        labelTransform.setContentSize(labelWidth, labelHeight);
        const labelBackdropNode = new Node('LabelBackdrop');
        labelBackdropNode.setParent(labelNode);
        this.configureNodeSize(labelBackdropNode, labelWidth, labelHeight);
        const labelBackdrop = labelBackdropNode.addComponent(Graphics);
        labelBackdrop.fillColor = new Color(22, 28, 38, isCompactState ? 154 : 186);
        labelBackdrop.roundRect(-labelWidth * 0.5, -labelHeight * 0.5, labelWidth, labelHeight, 16);
        labelBackdrop.fill();
        const label = labelNode.addComponent(Label);
        label.string = this.getBuildingLabel(building);
        label.fontSize = isCompactState ? 14 : 13;
        label.lineHeight = isCompactState ? 16 : 18;
        label.color = new Color(248, 250, 253, 255);
        label.enableWrapText = true;
    }

    private getBuildingSignalFrame(building: BuildingEntity): SpriteFrame | null {
        const assetId = this.getBuildingSignalAssetId(building);
        return assetId ? this.getVisualAssetFrame(assetId) : null;
    }

    private getBuildingSignalAssetId(building: BuildingEntity): SectMapVisualAssetId | null {
        switch (building.state) {
            case 'planned':
                return 'signal.planned';
            case 'supplied':
                return 'signal.supplied';
            case 'constructing':
                return 'signal.constructing';
            case 'damaged':
                return building.currentHp <= 0 ? 'signal.disabled' : 'signal.damaged';
            case 'active':
            default:
                return null;
        }
    }

    private getBuildingSignalFallbackText(building: BuildingEntity): string | null {
        switch (building.state) {
            case 'planned':
                return '筹';
            case 'supplied':
                return '料';
            case 'constructing':
                return '建';
            case 'damaged':
                return building.currentHp <= 0 ? '停' : '修';
            case 'active':
            default:
                return null;
        }
    }

    private getBuildingLabel(building: BuildingEntity): string {
        const maxHp = this.getBuildingMaxHp(building);
        const repairCost = this.getBuildingRepairCost(building);
        const targetLevel = building.pendingAction === 'upgrade' ? building.pendingLevel ?? building.level + 1 : null;
        const hpText = `HP ${Math.ceil(building.currentHp)}/${maxHp}`;
        if (building.state === 'planned') {
            return building.pendingAction === 'upgrade'
                ? `${building.definition.label} 升级 Lv.${targetLevel}\n${this.getSupplyProgressText(building)}`
                : `${building.definition.label} 蓝图\n${this.getSupplyProgressText(building)}`;
        }

        if (building.state === 'damaged') {
            return building.currentHp <= 0
                ? `${building.definition.label} 瘫痪\n修 ${this.getCostText(repairCost)}`
                : `${building.definition.label} 受损\n${hpText} 修 ${this.getCostText(repairCost)}`;
        }

        if (building.state === 'constructing') {
            return building.pendingAction === 'upgrade'
                ? `${building.definition.label} 升级中 Lv.${targetLevel}`
                : `${building.definition.label} 施工中`;
        }

        if (building.state === 'supplied') {
            return building.pendingAction === 'upgrade'
                ? `${building.definition.label} 待升级 Lv.${targetLevel}`
                : `${building.definition.label} 待施工`;
        }

        return `${building.definition.label} Lv.${building.level}`;
    }

    private getSupplyProgressText(building: BuildingEntity): string {
        const missing = this.getMissingResourcesForBuilding(building);
        if (!missing) {
            return '资源齐备';
        }

        const parts: string[] = [];
        const activeCost = this.getActiveCostForBuilding(building);
        (Object.keys(missing) as ResourceKind[]).forEach((key) => {
            if (missing[key] > 0) {
                parts.push(`${RESOURCE_DISPLAY[key].short}${building.supplied[key]}/${activeCost[key]}`);
            }
        });
        return parts.join(' ');
    }

    private getBuildingPalette(building: BuildingEntity): { fill: Color; stroke: Color } {
        if (building.damageFlashSeconds > 0) {
            return {
                fill: new Color(228, 128, 92, 228),
                stroke: new Color(255, 242, 214, 255),
            };
        }

        switch (building.state) {
            case 'planned':
                return {
                    fill: new Color(136, 136, 136, 170),
                    stroke: new Color(204, 204, 204, 255),
                };
            case 'supplied':
                return {
                    fill: new Color(90, 180, 96, 188),
                    stroke: new Color(208, 255, 192, 255),
                };
            case 'constructing':
                return {
                    fill: new Color(214, 163, 72, 194),
                    stroke: new Color(255, 226, 166, 255),
                };
            case 'damaged':
                return {
                    fill: new Color(173, 86, 86, 200),
                    stroke: new Color(255, 184, 184, 255),
                };
            case 'active':
            default:
                return {
                    fill: building.definition.activeColor,
                    stroke: new Color(243, 243, 243, 255),
                };
        }
    }

    private refreshBlueprintPreview(): void {
        if (!this.buildPlacement) {
            this.blueprintPreview.active = false;
            return;
        }

        const definition = this.buildPlacement.definition;
        const origin = this.buildPlacement.origin;

        const tiles = this.getFootprintTiles(origin, definition.width, definition.height);
        const frame = this.getFootprintFrame(tiles);
        this.blueprintPreview.active = true;
        this.blueprintPreview.setPosition(frame.center.x, frame.center.y, 0);
        this.configureNodeSize(this.blueprintPreview, frame.width + 24, frame.height + 64);

        const graphics = this.blueprintPreview.getComponent(Graphics) ?? this.blueprintPreview.addComponent(Graphics);
        graphics.clear();

        const valid = this.isPlacementValid(origin, definition);
        const fill = valid ? new Color(90, 184, 110, 142) : new Color(210, 76, 76, 142);
        const stroke = valid ? new Color(198, 255, 204, 255) : new Color(255, 186, 186, 255);

        graphics.lineWidth = 2;
        for (const tile of tiles) {
            const center = this.getTileCenter(tile);
            this.drawDiamond(graphics, center.x - frame.center.x, center.y - frame.center.y, fill, stroke, 4);
        }

        let labelNode = this.blueprintPreview.getChildByName('Label');
        if (!labelNode) {
            labelNode = new Node('Label');
            labelNode.setParent(this.blueprintPreview);
            labelNode.addComponent(UITransform);
            labelNode.addComponent(Label);
        }
        labelNode.setPosition(0, frame.height * 0.5 + 18, 0);
        const labelTransform = labelNode.getComponent(UITransform)!;
        labelTransform.setContentSize(Math.max(120, frame.width), 44);
        const label = labelNode.getComponent(Label)!;
        label.fontSize = 20;
        label.lineHeight = 24;
        label.color = new Color(250, 252, 255, 255);
        label.enableWrapText = true;
        label.string = `${definition.label} 蓝图\n${valid ? '点击空白确认' : '当前位置不可放置'}`;
    }

    private refreshDiscipleToken(): void {
        this.disciple.node = this.ensureUnitTokenNode(this.disciple.node, 'DiscipleToken');
        this.refreshUnitToken(this.disciple.node, this.disciple, this.getDiscipleBadgeText(), false);
    }

    private refreshHostileNpcToken(): void {
        if (!this.hostileNpc.active) {
            if (this.hostileNpc.node) {
                this.hostileNpc.node.active = false;
            }
            return;
        }

        this.hostileNpc.node = this.ensureUnitTokenNode(this.hostileNpc.node, 'HostileToken');
        this.hostileNpc.node.active = true;
        this.refreshUnitToken(this.hostileNpc.node, this.hostileNpc, this.getHostileBadgeText(), true);
    }

    private ensureUnitTokenNode(existing: Node | null, nodeName: string): Node {
        if (existing) {
            return existing;
        }

        const root = new Node(nodeName);
        root.setParent(this.tokenRoot);
        this.configureNodeSize(root, 118, 132);
        root.addComponent(Graphics);

        const portraitNode = new Node('Portrait');
        portraitNode.setParent(root);
        portraitNode.setPosition(0, 12, 0);
        this.configureNodeSize(portraitNode, 96, 96);
        portraitNode.addComponent(Sprite);

        const nameNode = new Node('Name');
        nameNode.setParent(root);
        nameNode.setPosition(0, 52, 0);
        const nameTransform = nameNode.addComponent(UITransform);
        nameTransform.setContentSize(88, 24);
        const nameLabel = nameNode.addComponent(Label);
        nameLabel.fontSize = 18;
        nameLabel.lineHeight = 20;
        nameLabel.color = new Color(246, 248, 253, 255);

        const hpNode = new Node('Hp');
        hpNode.setParent(root);
        hpNode.setPosition(0, -48, 0);
        const hpTransform = hpNode.addComponent(UITransform);
        hpTransform.setContentSize(92, 20);
        const hpLabel = hpNode.addComponent(Label);
        hpLabel.fontSize = 13;
        hpLabel.lineHeight = 16;
        hpLabel.color = new Color(255, 244, 216, 255);

        const badgeNode = new Node('Badge');
        badgeNode.setParent(root);
        badgeNode.setPosition(36, -24, 0);
        this.configureNodeSize(badgeNode, 34, 24);
        badgeNode.addComponent(Graphics);

        const badgeLabelNode = new Node('BadgeLabel');
        badgeLabelNode.setParent(badgeNode);
        const badgeLabelTransform = badgeLabelNode.addComponent(UITransform);
        badgeLabelTransform.setContentSize(28, 20);
        const badgeLabel = badgeLabelNode.addComponent(Label);
        badgeLabel.fontSize = 16;
        badgeLabel.lineHeight = 18;
        badgeLabel.color = new Color(247, 249, 252, 255);

        return root;
    }

    private refreshUnitToken(
        root: Node,
        unit: DiscipleEntity | HostileNpcEntity,
        badgeText: string,
        isHostile: boolean,
    ): void {
        const center = unit.worldPosition;
        root.setPosition(center.x, center.y + 18, 0);
        const portraitFrame = isHostile
            ? this.getHostilePortraitFrame(unit as HostileNpcEntity)
            : this.getDisciplePortraitFrame(unit as DiscipleEntity);

        const body = root.getComponent(Graphics);
        if (body) {
            body.clear();
            if (portraitFrame) {
                body.fillColor = new Color(22, 30, 42, 128);
                body.strokeColor = unit.hitFlashSeconds > 0
                    ? new Color(255, 230, 162, 255)
                    : isHostile
                      ? new Color(232, 166, 150, 255)
                      : new Color(132, 178, 226, 255);
                body.lineWidth = 3;
                body.circle(0, 12, 38);
                body.fill();
                body.stroke();
            } else {
                body.fillColor = this.getUnitTokenBodyColor(unit, isHostile);
                body.strokeColor = isHostile ? new Color(255, 226, 212, 255) : new Color(248, 248, 250, 255);
                body.lineWidth = 2;
                body.circle(0, 12, 28);
                body.fill();
                body.stroke();
            }
        }

        const portraitNode = root.getChildByName('Portrait');
        if (portraitNode) {
            if (portraitFrame) {
                portraitNode.active = true;
                const sprite = this.configureSpriteNode(
                    portraitNode,
                    portraitFrame,
                    0.5,
                    0.5,
                    unit.hitFlashSeconds > 0
                        ? new Color(255, 236, 184, 255)
                        : isHostile
                          ? new Color(255, 246, 246, 255)
                          : new Color(255, 255, 255, 255),
                );
                sprite.trim = false;
            } else {
                portraitNode.active = false;
            }
        }

        const nameLabel = root.getChildByName('Name')?.getComponent(Label);
        if (nameLabel) {
            nameLabel.string = unit.name;
        }

        const hpLabel = root.getChildByName('Hp')?.getComponent(Label);
        if (hpLabel) {
            hpLabel.string = `HP ${Math.ceil(unit.currentHp)}/${unit.model.stats.maxHp}`;
        }

        const badgeNode = root.getChildByName('Badge');
        const badgeGraphics = badgeNode?.getComponent(Graphics);
        if (badgeGraphics) {
            badgeGraphics.clear();
            badgeGraphics.fillColor = isHostile ? new Color(92, 38, 34, 236) : new Color(44, 54, 74, 236);
            badgeGraphics.strokeColor = isHostile ? new Color(255, 205, 180, 255) : new Color(204, 216, 232, 255);
            badgeGraphics.lineWidth = 2;
            badgeGraphics.roundRect(-17, -12, 34, 24, 8);
            badgeGraphics.fill();
            badgeGraphics.stroke();
        }

        const badgeLabel = badgeNode?.getChildByName('BadgeLabel')?.getComponent(Label);
        if (badgeLabel) {
            badgeLabel.string = badgeText;
        }
    }

    private getUnitTokenBodyColor(unit: DiscipleEntity | HostileNpcEntity, isHostile: boolean): Color {
        if (unit.hitFlashSeconds > 0) {
            return new Color(244, 210, 132, 255);
        }

        switch (unit.visualState) {
            case 'attacking':
                return isHostile ? new Color(204, 86, 68, 255) : new Color(100, 138, 208, 255);
            case 'guarding':
                return new Color(82, 124, 196, 255);
            case 'working':
                return isHostile ? new Color(176, 88, 74, 255) : new Color(78, 126, 178, 255);
            case 'carrying':
                return isHostile ? new Color(168, 86, 72, 255) : new Color(72, 148, 138, 255);
            case 'moving':
                return isHostile ? new Color(186, 78, 74, 255) : new Color(72, 114, 188, 255);
            case 'injured':
                return new Color(178, 96, 96, 255);
            case 'idle':
            default:
                return isHostile ? new Color(156, 70, 68, 255) : new Color(64, 102, 170, 255);
        }
    }

    private getDiscipleBadgeText(): string {
        switch (this.disciple.visualState) {
            case 'moving':
                return '行';
            case 'working':
                return '工';
            case 'carrying':
                return this.disciple.carrying ? RESOURCE_DISPLAY[this.disciple.carrying].short : '运';
            case 'guarding':
                return '守';
            case 'attacking':
                return '战';
            case 'injured':
                return '伤';
            case 'idle':
            default:
                return '闲';
        }
    }

    private getHostileBadgeText(): string {
        switch (this.hostileNpc.visualState) {
            case 'moving':
                return '袭';
            case 'attacking':
                return '攻';
            case 'injured':
                return '伤';
            case 'idle':
            default:
                return '伏';
        }
    }

    private getSessionGuardTower(): BuildingEntity | null {
        if (this.sessionGuardTowerId) {
            return this.buildingEntities.get(this.sessionGuardTowerId) ?? null;
        }

        return [...this.buildingEntities.values()].find((building) => building.definition.id === 'guard_tower') ?? null;
    }

    private getPrimaryDamagedBuilding(): BuildingEntity | null {
        const damagedBuildings = [...this.buildingEntities.values()].filter((building) => building.state === 'damaged');
        if (damagedBuildings.length === 0) {
            return null;
        }

        damagedBuildings.sort((left, right) => {
            const leftRatio = left.currentHp / Math.max(1, this.getBuildingMaxHp(left));
            const rightRatio = right.currentHp / Math.max(1, this.getBuildingMaxHp(right));
            if (leftRatio !== rightRatio) {
                return leftRatio - rightRatio;
            }
            return manhattanDistance(left.origin, INITIAL_MAIN_HALL_TILE) - manhattanDistance(right.origin, INITIAL_MAIN_HALL_TILE);
        });
        return damagedBuildings[0];
    }

    private getMarkerPositionForTile(tile: TileCoord): Vec3 {
        const center = this.getTileCenter(tile);
        return this.clampMarkerPosition(v3(center.x, center.y + 74, 0));
    }

    private getMarkerPositionForBuilding(building: BuildingEntity): Vec3 {
        const tiles = this.getFootprintTiles(building.origin, building.definition.width, building.definition.height);
        const frame = this.getFootprintFrame(tiles);
        return this.clampMarkerPosition(v3(frame.center.x, frame.center.y + Math.max(70, frame.height * 0.5 + 58), 0));
    }

    private clampMarkerPosition(position: Vec3): Vec3 {
        const halfWidth = Math.max(112, this.mapPixelWidth * 0.5 - 112);
        const halfHeight = Math.max(104, this.mapPixelHeight * 0.5 - 104);
        return v3(
            Math.max(-halfWidth, Math.min(halfWidth, position.x)),
            Math.max(-halfHeight, Math.min(halfHeight, position.y)),
            position.z,
        );
    }

    private getSessionGuidance(): SessionGuidance {
        const guardTower = this.getSessionGuardTower();
        const damagedBuilding = this.getPrimaryDamagedBuilding();
        const ruinBuilding = this.sessionRuinBuildingId ? this.buildingEntities.get(this.sessionRuinBuildingId) ?? null : null;
        const threatText = this.hostileNpc.active
            ? `${this.hostileNpc.name} ${Math.ceil(this.hostileNpc.currentHp)}/${this.hostileNpc.model.stats.maxHp} 正在袭扰`
            : this.sessionPhase === 'raid_countdown'
              ? `首袭 ${Math.ceil(this.hostileNpc.respawnTimerSeconds)}s 后抵达`
              : this.firstRaidResolved
                ? '首袭已退，等待修复收口'
                : '暂无外敌贴脸';

        switch (this.sessionPhase) {
            case 'clear_ruin':
                return {
                    headline: '先清旧仓',
                    detail: '切到拆除并点旧仓，回收木石后再开第一座护山台',
                    threat: threatText,
                    markerText: ruinBuilding ? '拆旧仓' : null,
                    markerTile: ruinBuilding ? { ...ruinBuilding.origin } : null,
                    markerPosition: ruinBuilding ? this.getMarkerPositionForBuilding(ruinBuilding) : null,
                    markerTone: 'goal',
                    focusBuildingId: ruinBuilding?.id ?? null,
                };
            case 'place_guard_tower': {
                const placementTile = this.buildPlacement?.origin ?? this.preferredBuildTile;
                return {
                    headline: '摆下护山台',
                    detail: '在高亮空地确认蓝图，弟子会自动备料和施工',
                    threat: threatText,
                    markerText: placementTile ? '建护山台' : null,
                    markerTile: placementTile ? { ...placementTile } : null,
                    markerPosition: placementTile ? this.getMarkerPositionForTile(placementTile) : null,
                    markerTone: 'goal',
                    focusBuildingId: null,
                };
            }
            case 'upgrade_guard_tower':
                return {
                    headline: '升到 Lv.2',
                    detail: '长按护山台或点选后升级，补料后会自动施工',
                    threat: threatText,
                    markerText: guardTower ? '升 Lv.2' : null,
                    markerTile: guardTower ? { ...guardTower.origin } : null,
                    markerPosition: guardTower ? this.getMarkerPositionForBuilding(guardTower) : null,
                    markerTone: 'goal',
                    focusBuildingId: guardTower?.id ?? null,
                };
            case 'raid_countdown':
                return {
                    headline: '准备首袭',
                    detail: '倒计时结束后外敌会直扑建筑，先确保护山台可守御',
                    threat: threatText,
                    markerText: guardTower ? '守塔' : null,
                    markerTile: guardTower ? { ...guardTower.origin } : null,
                    markerPosition: guardTower ? this.getMarkerPositionForBuilding(guardTower) : null,
                    markerTone: 'alert',
                    focusBuildingId: guardTower?.id ?? null,
                };
            case 'defend':
                return {
                    headline: this.hostileNpc.active ? '击退外敌' : '守住首袭',
                    detail: this.hostileNpc.active
                        ? '护山台与弟子会自动守御，优先保住正在受击的建筑'
                        : '等待守御收尾，不要让关键建筑继续掉到瘫痪',
                    threat: threatText,
                    markerText: this.hostileNpc.active ? '迎敌' : damagedBuilding ? '稳住' : null,
                    markerTile: this.hostileNpc.active
                        ? { ...this.hostileNpc.tile }
                        : damagedBuilding
                          ? { ...damagedBuilding.origin }
                          : null,
                    markerPosition: this.hostileNpc.active
                        ? this.clampMarkerPosition(v3(this.hostileNpc.worldPosition.x, this.hostileNpc.worldPosition.y + 82, 0))
                        : damagedBuilding
                          ? this.getMarkerPositionForBuilding(damagedBuilding)
                          : null,
                    markerTone: 'alert',
                    focusBuildingId: damagedBuilding?.id ?? null,
                };
            case 'recover':
                return {
                    headline: damagedBuilding ? '先修受损建筑' : '补平战损',
                    detail: damagedBuilding
                        ? '修好全部受损或瘫痪建筑后，本轮短会话即可成功收口'
                        : '若地图已无受损建筑，本轮短会话会自动达成',
                    threat: threatText,
                    markerText: damagedBuilding ? '先修复' : null,
                    markerTile: damagedBuilding ? { ...damagedBuilding.origin } : null,
                    markerPosition: damagedBuilding ? this.getMarkerPositionForBuilding(damagedBuilding) : null,
                    markerTone: 'goal',
                    focusBuildingId: damagedBuilding?.id ?? null,
                };
            case 'victory':
                return {
                    headline: '短会话已跑通',
                    detail: '从新档完成了一轮 建 -> 运 -> 守 -> 修，本地闭环可交互验证',
                    threat: '当前结论：浏览器预览已完成收口，可继续做容器比对',
                    markerText: null,
                    markerTile: null,
                    markerPosition: null,
                    markerTone: 'success',
                    focusBuildingId: null,
                };
            case 'defeat':
                return {
                    headline: '首轮守御失败',
                    detail: '继续掉到瘫痪的建筑会推翻短会话收口，需要回到修复/守御节奏',
                    threat: threatText,
                    markerText: damagedBuilding ? '止损' : null,
                    markerTile: damagedBuilding ? { ...damagedBuilding.origin } : null,
                    markerPosition: damagedBuilding ? this.getMarkerPositionForBuilding(damagedBuilding) : null,
                    markerTone: 'failure',
                    focusBuildingId: damagedBuilding?.id ?? null,
                };
            default:
                return {
                    headline: this.getSessionPhaseLabel(),
                    detail: this.sessionObjectiveText || '等待下一步目标同步',
                    threat: threatText,
                    markerText: null,
                    markerTile: null,
                    markerPosition: null,
                    markerTone: 'goal',
                    focusBuildingId: null,
                };
        }
    }

    private refreshObjectiveMarker(): void {
        const guidance = this.getSessionGuidance();
        if (!guidance.markerPosition || !guidance.markerText) {
            this.objectiveMarker.active = false;
            return;
        }

        this.objectiveMarker.active = true;
        this.objectiveMarker.setPosition(guidance.markerPosition.x, guidance.markerPosition.y, 0);
        this.configureNodeSize(this.objectiveMarker, 196, 132);

        const graphics = this.objectiveMarker.getComponent(Graphics) ?? this.objectiveMarker.addComponent(Graphics);
        graphics.clear();

        const palette =
            guidance.markerTone === 'alert'
                ? {
                      fill: new Color(108, 44, 38, 220),
                      stroke: new Color(255, 202, 182, 255),
                      halo: new Color(255, 116, 96, 34),
                  }
                : guidance.markerTone === 'success'
                  ? {
                        fill: new Color(46, 88, 58, 220),
                        stroke: new Color(210, 255, 214, 255),
                        halo: new Color(122, 220, 152, 28),
                    }
                  : guidance.markerTone === 'failure'
                    ? {
                          fill: new Color(92, 42, 54, 220),
                          stroke: new Color(255, 194, 210, 255),
                          halo: new Color(228, 96, 126, 34),
                      }
                    : {
                          fill: new Color(60, 72, 100, 220),
                          stroke: new Color(255, 232, 166, 255),
                          halo: new Color(255, 215, 72, 26),
                      };

        graphics.fillColor = palette.halo;
        graphics.circle(0, 4, 26);
        graphics.fill();

        graphics.lineWidth = 3;
        graphics.strokeColor = palette.stroke;
        graphics.fillColor = palette.fill;
        graphics.roundRect(-48, 16, 96, 30, 12);
        graphics.fill();
        graphics.stroke();

        graphics.moveTo(0, 16);
        graphics.lineTo(0, -24);
        graphics.stroke();
        graphics.fillColor = palette.fill;
        graphics.moveTo(0, -38);
        graphics.lineTo(-10, -20);
        graphics.lineTo(10, -20);
        graphics.close();
        graphics.fill();
        graphics.stroke();

        let labelNode = this.objectiveMarker.getChildByName('Label');
        if (!labelNode) {
            labelNode = new Node('Label');
            labelNode.setParent(this.objectiveMarker);
            labelNode.addComponent(UITransform);
            labelNode.addComponent(Label);
        }
        labelNode.setPosition(0, 31, 0);
        this.configureNodeSize(labelNode, 110, 28);
        const label = labelNode.getComponent(Label)!;
        label.string = guidance.markerText;
        label.fontSize = 16;
        label.lineHeight = 18;
        label.color = new Color(250, 252, 255, 255);
    }

    private advanceDisciple(deltaTime: number): void {
        this.promoteReadyBlueprints();

        if (!this.disciple.currentTask) {
            this.assignNextTask();
        }

        if (!this.disciple.currentTask) {
            this.disciple.visualState = 'idle';
            this.refreshDiscipleToken();
            return;
        }

        const task = this.disciple.currentTask;
        if (task.kind === 'guard') {
            this.advanceGuardTask(task, deltaTime);
            this.refreshDiscipleToken();
            return;
        }

        if (task.phase === 'move-to-resource' || task.phase === 'move-to-dropoff' || task.phase === 'move-to-site') {
            this.disciple.visualState =
                task.kind === 'gather' && task.phase === 'move-to-dropoff'
                    ? 'carrying'
                    : task.kind === 'haul'
                      ? 'carrying'
                      : 'moving';

            const arrived = this.advancePathingUnit(this.disciple, deltaTime, task.targetTile);
            if (!arrived) {
                this.refreshDiscipleToken();
                return;
            }

            switch (task.kind) {
                case 'gather':
                    if (task.phase === 'move-to-resource') {
                        task.phase = 'harvest';
                        task.timer = getActionDurationSeconds(BASE_HARVEST_SECONDS, this.disciple.model.stats.harvestSpeed);
                        this.logRuntime('INFO', 'TASK', 'task.phase_change', '弟子已到达资源格，进入采集阶段', {
                            task: this.describeTask(task),
                            resourceTile: this.formatTile(task.resourceTile),
                            durationMs: Math.round(task.timer * 1000),
                        });
                    } else {
                        task.phase = 'dropoff';
                        task.timer = getActionDurationSeconds(BASE_HAUL_SECONDS, this.disciple.model.stats.haulSpeed);
                        this.logRuntime('INFO', 'TASK', 'task.phase_change', '弟子已到达交付点，进入卸货阶段', {
                            task: this.describeTask(task),
                            targetTile: this.formatTile(task.targetTile),
                            durationMs: Math.round(task.timer * 1000),
                        });
                    }
                    break;
                case 'haul':
                    task.phase = 'deliver';
                    task.timer = getActionDurationSeconds(BASE_HAUL_SECONDS, this.disciple.model.stats.haulSpeed);
                    this.logRuntime('INFO', 'TASK', 'task.phase_change', '弟子已到达工地，进入交付阶段', {
                        task: this.describeTask(task),
                        buildingId: task.buildingId,
                        resourceKind: task.resourceKind,
                        durationMs: Math.round(task.timer * 1000),
                    });
                    break;
                case 'build':
                    task.phase = 'construct';
                    task.timer = getActionDurationSeconds(BASE_BUILD_SECONDS, this.disciple.model.stats.buildSpeed);
                    this.logRuntime('INFO', 'TASK', 'task.phase_change', '弟子已到达工地，进入施工阶段', {
                        task: this.describeTask(task),
                        buildingId: task.buildingId,
                        durationMs: Math.round(task.timer * 1000),
                    });
                    break;
                case 'demolish':
                    task.phase = 'demolish';
                    task.timer = getActionDurationSeconds(DEMOLISH_SECONDS, this.disciple.model.stats.buildSpeed);
                    this.logRuntime('INFO', 'TASK', 'task.phase_change', '弟子已到达拆除点，进入拆除阶段', {
                        task: this.describeTask(task),
                        buildingId: task.buildingId,
                        durationMs: Math.round(task.timer * 1000),
                    });
                    break;
                case 'repair':
                    task.phase = 'repair';
                    task.timer = getActionDurationSeconds(BASE_REPAIR_SECONDS, this.disciple.model.stats.repairSpeed);
                    this.logRuntime('INFO', 'TASK', 'task.phase_change', '弟子已到达受损建筑，进入修复阶段', {
                        task: this.describeTask(task),
                        buildingId: task.buildingId,
                        durationMs: Math.round(task.timer * 1000),
                    });
                    break;
            }
        } else {
            this.disciple.visualState = task.kind === 'gather' || task.kind === 'haul' ? 'carrying' : 'working';
            task.timer -= deltaTime;
            if (task.timer <= 0) {
                this.completeCurrentTaskPhase();
            }
        }

        this.refreshDiscipleToken();
    }

    private advanceGuardTask(
        task: Extract<DiscipleTask, { kind: 'guard' }>,
        deltaTime: number,
    ): void {
        if (!this.hostileNpc.active || this.hostileNpc.id !== task.hostileId) {
            this.disciple.currentTask = null;
            this.disciple.path = [];
            this.disciple.pathIndex = 0;
            return;
        }

        const targetTile = this.findStandTileAroundTile(this.hostileNpc.tile, this.disciple.tile) ?? { ...this.hostileNpc.tile };
        const inRange =
            manhattanDistance(this.disciple.tile, this.hostileNpc.tile) <= Math.max(1, Math.round(this.disciple.model.stats.attackRange));

        if (!inRange) {
            task.phase = 'move-to-site';
            task.targetTile = targetTile;

            const needsPath =
                this.disciple.path.length === 0 ||
                this.disciple.pathIndex >= this.disciple.path.length ||
                !this.isPathStillLeadingToTile(this.disciple.path, targetTile);

            if (needsPath) {
                const path = this.findPath(this.disciple.tile, targetTile);
                if (!path) {
                    this.disciple.currentTask = null;
                    this.disciple.path = [];
                    this.disciple.pathIndex = 0;
                    this.logRuntime('WARN', 'TASK', 'task.assign_failed', '守御目标当前不可达', {
                        hostileId: this.hostileNpc.id,
                        targetTile: this.formatTile(targetTile),
                    });
                    return;
                }
                this.disciple.path = path;
                this.disciple.pathIndex = 0;
            }

            this.disciple.visualState = 'guarding';
            this.advancePathingUnit(this.disciple, deltaTime, targetTile);
            return;
        }

        task.phase = 'attack';
        this.disciple.path = [];
        this.disciple.pathIndex = 0;
        this.disciple.visualState = 'attacking';

        if (this.disciple.attackCooldownSeconds > 0) {
            return;
        }

        this.disciple.attackCooldownSeconds = this.disciple.model.stats.attackInterval;
        const damage = this.applyDamageToHostile(this.hostileNpc, this.disciple.model.stats.attackPower, this.disciple.name);
        this.logRuntime('INFO', 'COMBAT', 'combat.disciple_attack', '弟子命中外敌', {
            discipleId: this.disciple.id,
            hostileId: this.hostileNpc.id,
            damage,
            hostileHp: `${Math.ceil(this.hostileNpc.currentHp)}/${this.hostileNpc.model.stats.maxHp}`,
        });
    }

    private advancePathingUnit(unit: PathingUnit, deltaTime: number, fallbackTarget: TileCoord): boolean {
        if (unit.path.length === 0 || unit.pathIndex >= unit.path.length) {
            const target = this.getTileCenter(fallbackTarget);
            unit.worldPosition.set(target.x, target.y, 0);
            unit.tile = { ...fallbackTarget };
            return true;
        }

        const nextTile = unit.path[unit.pathIndex];
        const target = this.getTileCenter(nextTile);
        const toTarget = v3(target.x - unit.worldPosition.x, target.y - unit.worldPosition.y, 0);
        const distance = Math.sqrt(toTarget.x * toTarget.x + toTarget.y * toTarget.y);
        const moveSpeed = unit.model.stats.moveSpeed;

        if (distance <= moveSpeed * deltaTime) {
            unit.worldPosition.set(target.x, target.y, 0);
            unit.tile = { ...nextTile };
            unit.pathIndex += 1;
            return unit.pathIndex >= unit.path.length;
        }

        const scale = (moveSpeed * deltaTime) / Math.max(distance, 0.0001);
        unit.worldPosition.x += toTarget.x * scale;
        unit.worldPosition.y += toTarget.y * scale;
        return false;
    }

    private completeCurrentTaskPhase(): void {
        const task = this.disciple.currentTask;
        if (!task) {
            return;
        }

        switch (task.kind) {
            case 'gather': {
                if (task.phase === 'harvest') {
                    const resource = this.resourceNodes.get(this.getTileKey(task.resourceTile.col, task.resourceTile.row));
                    if (!resource || !this.isResourceHarvestable(resource)) {
                        this.disciple.currentTask = null;
                        this.disciple.path = [];
                        this.disciple.pathIndex = 0;
                        this.logRuntime('WARN', 'TASK', 'task.cancelled', '采集节点在采集阶段变为不可用，任务已取消', {
                            taskKind: task.kind,
                            resourceKind: task.resourceKind,
                            resourceTile: this.formatTile(task.resourceTile),
                            resourceState: resource?.state ?? 'missing',
                        });
                        this.setMessage(`${RESOURCE_DISPLAY[task.resourceKind].title} 当前不可采集，弟子已停止本次采集`);
                        return;
                    }

                    resource.remainingCharges = Math.max(0, resource.remainingCharges - 1);
                    const depleted = resource.remainingCharges <= 0;
                    if (depleted) {
                        resource.state = 'regenerating';
                        resource.regenTimerSeconds = resource.regenSeconds;
                        this.logRuntime('INFO', 'RESOURCE', 'resource.depleted', '资源节点已被采空，进入刷新计时', {
                            tile: this.formatTile(resource.tile),
                            resourceKind: resource.kind,
                            regenSeconds: resource.regenSeconds,
                            designated: resource.designated,
                        });
                    } else {
                        this.logRuntime('INFO', 'RESOURCE', 'resource.harvested', '资源节点已产出一次，剩余储量减少', {
                            tile: this.formatTile(resource.tile),
                            resourceKind: resource.kind,
                            remainingCharges: resource.remainingCharges,
                            maxCharges: resource.maxCharges,
                            designated: resource.designated,
                        });
                    }

                    this.refreshResourceMarkers();
                    this.disciple.carrying = task.resourceKind;
                    task.phase = 'move-to-dropoff';
                    task.targetTile = this.getDropoffTile();
                    task.timer = 0;
                    this.setMessage(
                        depleted
                            ? `${RESOURCE_DISPLAY[task.resourceKind].title} 已枯竭，等待 ${Math.ceil(resource.regenSeconds)}s 刷新`
                            : `${RESOURCE_DISPLAY[task.resourceKind].title} 已采集，剩余 ${resource.remainingCharges}/${resource.maxCharges}`,
                    );
                    this.disciple.path = this.findPath(this.disciple.tile, task.targetTile) ?? [];
                    this.disciple.pathIndex = 0;
                    this.logRuntime('INFO', 'TASK', 'task.phase_change', '弟子采集完成，开始返程交付', {
                        task: this.describeTask(task),
                        carrying: task.resourceKind,
                        targetTile: this.formatTile(task.targetTile),
                        pathLength: this.disciple.path.length,
                        depleted,
                    });
                    return;
                }

                if (task.phase === 'dropoff') {
                    if (this.authorityConnected && this.disciple.carrying) {
                        const resourceKind = this.disciple.carrying;
                        this.executeAuthorityCommand(
                            {
                                name: 'collect_stockpile',
                                payload: {
                                    resourceKind: resourceKind as AuthorityResourceKind,
                                    amount: 1,
                                },
                            },
                            {
                                commandKey: `collect_stockpile:${resourceKind}:${task.resourceTile.col},${task.resourceTile.row}`,
                                onAccepted: () => {
                                    this.disciple.carrying = null;
                                    this.disciple.currentTask = null;
                                    this.disciple.path = [];
                                    this.disciple.pathIndex = 0;
                                    this.logRuntime('INFO', 'TASK', 'task.completed', '采集任务已完成', {
                                        taskKind: task.kind,
                                        resourceKind: task.resourceKind,
                                        authority: true,
                                    });
                                },
                            },
                        );
                        return;
                    }
                    if (this.disciple.carrying) {
                        this.stockpile[this.disciple.carrying] += 1;
                        this.logRuntime('INFO', 'RESOURCE', 'resource.stockpile_gain', '资源已入库', {
                            resourceKind: this.disciple.carrying,
                            stockpile: this.summarizeStockpile(),
                            resourceSummary: this.getResourceSummary(),
                        });
                    }
                    this.disciple.carrying = null;
                    this.disciple.currentTask = null;
                    this.disciple.path = [];
                    this.disciple.pathIndex = 0;
                    this.logRuntime('INFO', 'TASK', 'task.completed', '采集任务已完成', {
                        taskKind: task.kind,
                        resourceKind: task.resourceKind,
                    });
                    this.setMessage('弟子已把采集到的资源带回宗门');
                    return;
                }
                break;
            }
            case 'haul': {
                if (task.phase === 'deliver') {
                    if (this.authorityConnected) {
                        this.executeAuthorityCommand(
                            {
                                name: 'deliver_build_resource',
                                payload: {
                                    buildingId: task.buildingId,
                                    resourceKind: task.resourceKind as AuthorityResourceKind,
                                },
                            },
                            {
                                commandKey: `deliver_build_resource:${task.buildingId}:${task.resourceKind}`,
                                onAccepted: () => {
                                    this.disciple.carrying = null;
                                    this.disciple.currentTask = null;
                                    this.disciple.path = [];
                                    this.disciple.pathIndex = 0;
                                    this.logRuntime('INFO', 'TASK', 'task.completed', '搬运任务已完成', {
                                        taskKind: task.kind,
                                        buildingId: task.buildingId,
                                        resourceKind: task.resourceKind,
                                        authority: true,
                                    });
                                },
                            },
                        );
                        return;
                    }
                    const building = this.buildingEntities.get(task.buildingId);
                    if (building && this.stockpile[task.resourceKind] > 0) {
                        this.stockpile[task.resourceKind] -= 1;
                        building.supplied[task.resourceKind] += 1;
                        if (!this.getMissingResourcesForBuilding(building)) {
                            this.setBuildingState(building, 'supplied', 'all_resources_delivered');
                            this.setMessage(
                                building.pendingAction === 'upgrade'
                                    ? `${building.definition.label} 升级物资齐备，等待弟子动工`
                                    : `${building.definition.label} 资源填充完毕，蓝图已变绿`,
                            );
                        } else {
                            this.setMessage(
                                `${building.definition.label}${building.pendingAction === 'upgrade' ? ' 升级' : ''} 已收到 ${RESOURCE_DISPLAY[task.resourceKind].title}`,
                            );
                        }
                        this.logRuntime('INFO', 'RESOURCE', 'resource.stockpile_deliver', '资源已送达建筑工地', {
                            buildingId: building.id,
                            buildingType: building.definition.id,
                            resourceKind: task.resourceKind,
                            supplied: this.getSupplyProgressText(building),
                            stockpile: this.summarizeStockpile(),
                        });
                        this.refreshBuildings();
                    }
                    this.disciple.carrying = null;
                    this.disciple.currentTask = null;
                    this.disciple.path = [];
                    this.disciple.pathIndex = 0;
                    this.logRuntime('INFO', 'TASK', 'task.completed', '搬运任务已完成', {
                        taskKind: task.kind,
                        buildingId: task.buildingId,
                        resourceKind: task.resourceKind,
                    });
                    return;
                }
                break;
            }
            case 'build': {
                if (task.phase === 'construct') {
                    if (this.authorityConnected) {
                        this.executeAuthorityCommand(
                            {
                                name: 'complete_building_work',
                                payload: {
                                    buildingId: task.buildingId,
                                },
                            },
                            {
                                commandKey: `complete_building_work:${task.buildingId}`,
                                onAccepted: () => {
                                    this.disciple.currentTask = null;
                                    this.disciple.path = [];
                                    this.disciple.pathIndex = 0;
                                    this.logRuntime('INFO', 'TASK', 'task.completed', '施工任务已完成', {
                                        taskKind: task.kind,
                                        buildingId: task.buildingId,
                                        authority: true,
                                    });
                                },
                            },
                        );
                        return;
                    }
                    const building = this.buildingEntities.get(task.buildingId);
                    if (building) {
                        const isUpgrade = building.pendingAction === 'upgrade';
                        if (isUpgrade) {
                            building.level = building.pendingLevel ?? building.level + 1;
                        }
                        building.pendingAction = null;
                        building.pendingLevel = building.level;
                        building.currentHp = this.getBuildingMaxHp(building);
                        this.setBuildingState(building, 'active', 'construction_completed');
                        this.refreshBuildings();
                        this.setMessage(
                            isUpgrade ? `${building.definition.label} 已升至 Lv.${building.level}` : `${building.definition.label} 已完工`,
                        );
                    }
                    this.disciple.currentTask = null;
                    this.disciple.path = [];
                    this.disciple.pathIndex = 0;
                    this.logRuntime('INFO', 'TASK', 'task.completed', '施工任务已完成', {
                        taskKind: task.kind,
                        buildingId: task.buildingId,
                    });
                    return;
                }
                break;
            }
            case 'repair': {
                if (task.phase === 'repair') {
                    if (this.authorityConnected) {
                        this.executeAuthorityCommand(
                            {
                                name: 'complete_repair',
                                payload: {
                                    buildingId: task.buildingId,
                                },
                            },
                            {
                                commandKey: `complete_repair:${task.buildingId}`,
                                onAccepted: () => {
                                    this.disciple.currentTask = null;
                                    this.disciple.path = [];
                                    this.disciple.pathIndex = 0;
                                    this.logRuntime('INFO', 'TASK', 'task.completed', '修复任务已完成', {
                                        taskKind: task.kind,
                                        buildingId: task.buildingId,
                                        authority: true,
                                    });
                                },
                            },
                        );
                        return;
                    }
                    const building = this.buildingEntities.get(task.buildingId);
                    if (building && this.canRepairBuilding(building)) {
                        const repairCost = this.getBuildingRepairCost(building);
                        this.spendStockpile(repairCost);
                        building.currentHp = this.getBuildingMaxHp(building);
                        building.damageFlashSeconds = 0;
                        this.setBuildingState(building, 'active', 'repair_completed');
                        this.refreshBuildings();
                        this.logRuntime('INFO', 'BUILD', 'build.repaired', '建筑已完成修复', {
                            buildingId: building.id,
                            buildingType: building.definition.id,
                            hp: `${Math.ceil(building.currentHp)}/${this.getBuildingMaxHp(building)}`,
                            stockpile: this.summarizeStockpile(),
                        });
                        this.setMessage(`${building.definition.label} 已修复完毕`);
                    }
                    this.disciple.currentTask = null;
                    this.disciple.path = [];
                    this.disciple.pathIndex = 0;
                    this.logRuntime('INFO', 'TASK', 'task.completed', '修复任务已完成', {
                        taskKind: task.kind,
                        buildingId: task.buildingId,
                    });
                    return;
                }
                break;
            }
            case 'demolish': {
                if (task.phase === 'demolish') {
                    if (this.authorityConnected) {
                        this.executeAuthorityCommand(
                            {
                                name: 'complete_demolition',
                                payload: {
                                    buildingId: task.buildingId,
                                },
                            },
                            {
                                commandKey: `complete_demolition:${task.buildingId}`,
                                onAccepted: () => {
                                    this.disciple.currentTask = null;
                                    this.disciple.path = [];
                                    this.disciple.pathIndex = 0;
                                    this.logRuntime('INFO', 'TASK', 'task.completed', '拆除任务已完成', {
                                        taskKind: task.kind,
                                        buildingId: task.buildingId,
                                        authority: true,
                                    });
                                },
                            },
                        );
                        return;
                    }
                    const building = this.buildingEntities.get(task.buildingId);
                    if (building) {
                        const salvage = this.getDemolitionYield(building);
                        if (salvage.spirit_wood > 0 || salvage.spirit_stone > 0 || salvage.herb > 0) {
                            this.addStockpile(salvage);
                        }
                        if (this.sessionGuardTowerId === building.id) {
                            this.sessionGuardTowerId = null;
                        }
                        if (this.sessionRuinBuildingId === building.id) {
                            this.sessionRuinBuildingId = null;
                        }
                        this.buildingEntities.delete(task.buildingId);
                        this.refreshBuildings();
                        this.logRuntime('INFO', 'BUILD', 'build.demolished', '建筑已拆除', {
                            buildingId: building.id,
                            buildingType: building.definition.id,
                            origin: this.formatTile(building.origin),
                            salvage,
                        });
                        this.setMessage(
                            salvage.spirit_wood > 0 || salvage.spirit_stone > 0 || salvage.herb > 0
                                ? `${building.definition.label} 已拆除，返还 ${this.getCostText(salvage)}`
                                : `${building.definition.label} 已拆除`,
                        );
                    }
                    this.disciple.currentTask = null;
                    this.disciple.path = [];
                    this.disciple.pathIndex = 0;
                    this.logRuntime('INFO', 'TASK', 'task.completed', '拆除任务已完成', {
                        taskKind: task.kind,
                        buildingId: task.buildingId,
                    });
                    return;
                }
                break;
            }
        }

        this.disciple.currentTask = null;
        this.disciple.path = [];
        this.disciple.pathIndex = 0;
    }

    private promoteReadyBlueprints(): void {
        for (const building of this.buildingEntities.values()) {
            if (building.state !== 'planned') {
                continue;
            }

            if (!this.getMissingResourcesForBuilding(building)) {
                this.setBuildingState(building, 'supplied', 'resource_requirements_satisfied');
                this.refreshBuildings();
            }
        }
    }

    private assignNextTask(): void {
        if (this.sessionOutcome !== 'in_progress') {
            this.disciple.visualState = 'idle';
            return;
        }

        const guardTarget = this.findGuardTarget();
        if (guardTarget) {
            this.startGuardTask(guardTarget);
            return;
        }

        const repairTarget = this.findRepairTarget();
        if (repairTarget) {
            this.startRepairTask(repairTarget);
            return;
        }

        const demolitionTarget = this.findDemolitionTarget();
        if (demolitionTarget) {
            this.startDemolitionTask(demolitionTarget);
            return;
        }

        const buildTarget = this.findConstructibleBuilding();
        if (buildTarget) {
            this.startBuildTask(buildTarget);
            return;
        }

        const haulingTarget = this.findHaulingTarget();
        if (haulingTarget) {
            this.startHaulTask(haulingTarget.building, haulingTarget.resourceKind);
            return;
        }

        const gatherTarget = this.findGatherTarget();
        if (gatherTarget) {
            this.startGatherTask(gatherTarget);
            return;
        }

        this.disciple.visualState = 'idle';
    }

    private findDemolitionTarget(): BuildingEntity | null {
        const candidates = [...this.buildingEntities.values()].filter((building) => building.markedForDemolition);
        return this.pickNearestBuildingByStandTile(candidates);
    }

    private findConstructibleBuilding(): BuildingEntity | null {
        const priority = this.getSessionPriorityBuilding();
        if (priority && priority.state === 'supplied') {
            return priority;
        }

        const candidates = [...this.buildingEntities.values()].filter((building) => building.state === 'supplied');
        return this.pickNearestBuildingByStandTile(candidates);
    }

    private findGuardTarget(): HostileNpcEntity | null {
        return this.hostileNpc.active ? this.hostileNpc : null;
    }

    private findRepairTarget(): BuildingEntity | null {
        const candidates = [...this.buildingEntities.values()].filter((building) => this.canRepairBuilding(building));
        return this.pickNearestBuildingByStandTile(candidates);
    }

    private canRepairBuilding(building: BuildingEntity): boolean {
        return (
            building.state === 'damaged' &&
            !this.isSessionRuinBuilding(building) &&
            building.currentHp < this.getBuildingMaxHp(building) &&
            this.hasStockpile(this.getBuildingRepairCost(building))
        );
    }

    private hasStockpile(cost: Stockpile): boolean {
        return (Object.keys(cost) as ResourceKind[]).every((key) => this.stockpile[key] >= cost[key]);
    }

    private spendStockpile(cost: Stockpile): void {
        (Object.keys(cost) as ResourceKind[]).forEach((key) => {
            this.stockpile[key] = Math.max(0, this.stockpile[key] - cost[key]);
        });
    }

    private findHaulingTarget(): { building: BuildingEntity; resourceKind: ResourceKind } | null {
        const priority = this.getSessionPriorityBuilding();
        if (priority && priority.state === 'planned') {
            const missing = this.getMissingResourcesForBuilding(priority);
            if (missing) {
                const resourceKind = (Object.keys(missing) as ResourceKind[]).find((key) => missing[key] > 0 && this.stockpile[key] > 0);
                if (resourceKind) {
                    return { building: priority, resourceKind };
                }
            }
        }

        const candidates = [...this.buildingEntities.values()].filter((building) => building.state === 'planned');
        for (const building of candidates) {
            const missing = this.getMissingResourcesForBuilding(building);
            if (!missing) {
                continue;
            }

            const resourceKind = (Object.keys(missing) as ResourceKind[]).find((key) => missing[key] > 0 && this.stockpile[key] > 0);
            if (resourceKind) {
                return { building, resourceKind };
            }
        }
        return null;
    }

    private findGatherTarget(): ResourceNode | null {
        const designated = [...this.resourceNodes.values()].filter(
            (resource) => resource.designated && this.isResourceHarvestable(resource),
        );
        if (designated.length === 0) {
            return null;
        }

        const repairDeficit = this.getRepairResourceDeficit();
        if (repairDeficit) {
            const repairKinds = (Object.keys(repairDeficit) as ResourceKind[]).filter((key) => repairDeficit[key] > 0);
            const target = this.pickNearestResource(
                designated.filter((resource) => repairKinds.indexOf(resource.kind) >= 0),
            );
            if (target) {
                return target;
            }
        }

        const priority = this.getSessionPriorityBuilding();
        if (priority && priority.state === 'planned') {
            const missing = this.getMissingResourcesForBuilding(priority);
            if (missing) {
                const neededKinds = (Object.keys(missing) as ResourceKind[]).filter((key) => missing[key] > 0);
                const target = this.pickNearestResource(
                    designated.filter((resource) => neededKinds.indexOf(resource.kind) >= 0),
                );
                if (target) {
                    return target;
                }
            }
        }

        const plannedBuildings = [...this.buildingEntities.values()].filter((building) => building.state === 'planned');
        if (plannedBuildings.length > 0) {
            for (const building of plannedBuildings) {
                const missing = this.getMissingResourcesForBuilding(building);
                if (!missing) {
                    continue;
                }

                const neededKinds = (Object.keys(missing) as ResourceKind[]).filter((key) => missing[key] > 0);
                const target = this.pickNearestResource(
                    designated.filter((resource) => neededKinds.indexOf(resource.kind) >= 0),
                );
                if (target) {
                    return target;
                }
            }
        }

        return this.pickNearestResource(designated);
    }

    private getRepairResourceDeficit(): Stockpile | null {
        const deficit = EMPTY_STOCKPILE();
        let hasDeficit = false;

        for (const building of this.buildingEntities.values()) {
            const repairCost = this.getBuildingRepairCost(building);
            if (building.state !== 'damaged' || this.isSessionRuinBuilding(building) || building.currentHp >= this.getBuildingMaxHp(building)) {
                continue;
            }

            (Object.keys(repairCost) as ResourceKind[]).forEach((key) => {
                const missing = Math.max(0, repairCost[key] - this.stockpile[key]);
                if (missing > 0) {
                    deficit[key] = Math.max(deficit[key], missing);
                    hasDeficit = true;
                }
            });
        }

        return hasDeficit ? deficit : null;
    }

    private pickNearestBuildingByStandTile(buildings: BuildingEntity[]): BuildingEntity | null {
        let best: BuildingEntity | null = null;
        let bestDistance = Number.POSITIVE_INFINITY;

        for (const building of buildings) {
            const standTile = this.findBuildingStandTileFromTile(building, this.disciple.tile);
            if (!standTile) {
                continue;
            }

            const distance = manhattanDistance(this.disciple.tile, standTile);
            if (distance < bestDistance) {
                bestDistance = distance;
                best = building;
            }
        }

        return best;
    }

    private pickNearestResource(resources: ResourceNode[]): ResourceNode | null {
        let best: ResourceNode | null = null;
        let bestDistance = Number.POSITIVE_INFINITY;

        for (const resource of resources) {
            const distance = manhattanDistance(this.disciple.tile, resource.tile);
            if (distance < bestDistance) {
                bestDistance = distance;
                best = resource;
            }
        }

        return best;
    }

    private startGatherTask(resource: ResourceNode): void {
        if (!this.isResourceHarvestable(resource)) {
            this.logRuntime('WARN', 'TASK', 'task.assign_failed', '资源节点当前不可采集', {
                resourceKind: resource.kind,
                resourceTile: this.formatTile(resource.tile),
                resourceState: resource.state,
                remainingCharges: resource.remainingCharges,
            });
            this.setMessage(
                resource.state === 'regenerating'
                    ? `${RESOURCE_DISPLAY[resource.kind].title} 正在刷新中`
                    : `${RESOURCE_DISPLAY[resource.kind].title} 当前不可采集`,
            );
            return;
        }

        const path = this.findPath(this.disciple.tile, resource.tile);
        if (!path) {
            this.logRuntime('WARN', 'TASK', 'task.assign_failed', '资源采集任务不可达', {
                resourceKind: resource.kind,
                resourceTile: this.formatTile(resource.tile),
            });
            this.setMessage(`${RESOURCE_DISPLAY[resource.kind].title} 当前没有可达路径`);
            return;
        }

        this.disciple.currentTask = {
            kind: 'gather',
            resourceTile: { ...resource.tile },
            resourceKind: resource.kind,
            phase: 'move-to-resource',
            timer: 0,
            targetTile: { ...resource.tile },
        };
        this.disciple.path = path;
        this.disciple.pathIndex = 0;
        this.logRuntime('INFO', 'TASK', 'task.assigned', '已分配采集任务', {
            task: this.describeTask(this.disciple.currentTask),
            resourceKind: resource.kind,
            targetTile: this.formatTile(resource.tile),
            pathLength: path.length,
            remainingCharges: resource.remainingCharges,
        });
        this.setMessage(
            `弟子已前往 ${RESOURCE_DISPLAY[resource.kind].title} 采集点（剩余 ${resource.remainingCharges}/${resource.maxCharges}）`,
        );
    }

    private startHaulTask(building: BuildingEntity, resourceKind: ResourceKind): void {
        const standTile = this.findBuildingStandTile(building);
        if (!standTile) {
            this.logRuntime('WARN', 'TASK', 'task.assign_failed', '搬运任务缺少可施工站位', {
                buildingId: building.id,
                buildingType: building.definition.id,
            });
            this.setMessage(`${building.definition.label} 周边没有可施工站位`);
            return;
        }

        const path = this.findPath(this.disciple.tile, standTile);
        if (!path) {
            this.logRuntime('WARN', 'TASK', 'task.assign_failed', '搬运任务不可达', {
                buildingId: building.id,
                buildingType: building.definition.id,
                standTile: this.formatTile(standTile),
            });
            this.setMessage(`${building.definition.label} 当前无法送达资源`);
            return;
        }

        this.disciple.carrying = resourceKind;
        this.disciple.currentTask = {
            kind: 'haul',
            buildingId: building.id,
            resourceKind,
            phase: 'move-to-site',
            timer: 0,
            targetTile: standTile,
        };
        this.disciple.path = path;
        this.disciple.pathIndex = 0;
        this.logRuntime('INFO', 'TASK', 'task.assigned', '已分配搬运任务', {
            task: this.describeTask(this.disciple.currentTask),
            buildingId: building.id,
            buildingType: building.definition.id,
            resourceKind,
            targetTile: this.formatTile(standTile),
            pathLength: path.length,
        });
        this.setMessage(`弟子正在为 ${building.definition.label} 运送 ${RESOURCE_DISPLAY[resourceKind].title}`);
    }

    private startBuildTask(building: BuildingEntity): void {
        const standTile = this.findBuildingStandTile(building);
        if (!standTile) {
            this.logRuntime('WARN', 'TASK', 'task.assign_failed', '施工任务缺少可施工站位', {
                buildingId: building.id,
                buildingType: building.definition.id,
            });
            this.setMessage(`${building.definition.label} 周边没有可施工站位`);
            return;
        }

        const path = this.findPath(this.disciple.tile, standTile);
        if (!path) {
            this.logRuntime('WARN', 'TASK', 'task.assign_failed', '施工任务不可达', {
                buildingId: building.id,
                buildingType: building.definition.id,
                standTile: this.formatTile(standTile),
            });
            this.setMessage(`${building.definition.label} 当前无法施工`);
            return;
        }

        this.setBuildingState(building, 'constructing', 'disciple_started_construction');
        this.refreshBuildings();
        if (this.authorityConnected) {
            this.executeAuthorityCommand(
                {
                    name: 'start_building_work',
                    payload: {
                        buildingId: building.id,
                    },
                },
                {
                    commandKey: `start_building_work:${building.id}`,
                },
            );
        }
        this.disciple.currentTask = {
            kind: 'build',
            buildingId: building.id,
            phase: 'move-to-site',
            timer: 0,
            targetTile: standTile,
        };
        this.disciple.path = path;
        this.disciple.pathIndex = 0;
        this.logRuntime('INFO', 'TASK', 'task.assigned', '已分配施工任务', {
            task: this.describeTask(this.disciple.currentTask),
            buildingId: building.id,
            buildingType: building.definition.id,
            targetTile: this.formatTile(standTile),
            pathLength: path.length,
        });
        this.setMessage(`弟子已前往 ${building.definition.label} 工地`);
    }

    private startRepairTask(building: BuildingEntity): void {
        const standTile = this.findBuildingStandTile(building);
        if (!standTile) {
            this.logRuntime('WARN', 'TASK', 'task.assign_failed', '修复任务缺少可施工站位', {
                buildingId: building.id,
                buildingType: building.definition.id,
            });
            this.setMessage(`${building.definition.label} 周边没有可修复站位`);
            return;
        }

        const path = this.findPath(this.disciple.tile, standTile);
        if (!path) {
            this.logRuntime('WARN', 'TASK', 'task.assign_failed', '修复任务不可达', {
                buildingId: building.id,
                buildingType: building.definition.id,
                standTile: this.formatTile(standTile),
            });
            this.setMessage(`${building.definition.label} 当前无法修复`);
            return;
        }

        this.disciple.currentTask = {
            kind: 'repair',
            buildingId: building.id,
            phase: 'move-to-site',
            timer: 0,
            targetTile: standTile,
        };
        this.disciple.path = path;
        this.disciple.pathIndex = 0;
        this.logRuntime('INFO', 'TASK', 'task.assigned', '已分配修复任务', {
            task: this.describeTask(this.disciple.currentTask),
            buildingId: building.id,
            buildingType: building.definition.id,
            targetTile: this.formatTile(standTile),
            pathLength: path.length,
        });
        this.setMessage(`弟子已前往修复 ${building.definition.label}`);
    }

    private startGuardTask(hostile: HostileNpcEntity): void {
        const standTile = this.findStandTileAroundTile(hostile.tile, this.disciple.tile) ?? { ...hostile.tile };
        const inRange =
            manhattanDistance(this.disciple.tile, hostile.tile) <= Math.max(1, Math.round(this.disciple.model.stats.attackRange));

        let path: TileCoord[] = [];
        if (!inRange) {
            const resolvedPath = this.findPath(this.disciple.tile, standTile);
            if (!resolvedPath) {
                this.logRuntime('WARN', 'TASK', 'task.assign_failed', '守御任务不可达', {
                    hostileId: hostile.id,
                    hostileTile: this.formatTile(hostile.tile),
                    targetTile: this.formatTile(standTile),
                });
                this.setMessage(`${hostile.name} 当前不可达，暂无法迎击`);
                return;
            }
            path = resolvedPath;
        }

        this.disciple.currentTask = {
            kind: 'guard',
            hostileId: hostile.id,
            phase: inRange ? 'attack' : 'move-to-site',
            timer: 0,
            targetTile: standTile,
        };
        this.disciple.path = path;
        this.disciple.pathIndex = 0;
        this.logRuntime('INFO', 'TASK', 'task.assigned', '已分配守御任务', {
            task: this.describeTask(this.disciple.currentTask),
            hostileId: hostile.id,
            hostileTile: this.formatTile(hostile.tile),
            targetTile: this.formatTile(standTile),
            pathLength: path.length,
        });
        this.setMessage(`弟子已前往拦截 ${hostile.name}`);
    }

    private startDemolitionTask(building: BuildingEntity): void {
        const standTile = this.findBuildingStandTile(building);
        if (!standTile) {
            this.logRuntime('WARN', 'TASK', 'task.assign_failed', '拆除任务缺少可施工站位', {
                buildingId: building.id,
                buildingType: building.definition.id,
            });
            this.setMessage(`${building.definition.label} 周边没有可拆除站位`);
            return;
        }

        const path = this.findPath(this.disciple.tile, standTile);
        if (!path) {
            this.logRuntime('WARN', 'TASK', 'task.assign_failed', '拆除任务不可达', {
                buildingId: building.id,
                buildingType: building.definition.id,
                standTile: this.formatTile(standTile),
            });
            this.setMessage(`${building.definition.label} 当前无法拆除`);
            return;
        }

        this.disciple.currentTask = {
            kind: 'demolish',
            buildingId: building.id,
            phase: 'move-to-site',
            timer: 0,
            targetTile: standTile,
        };
        this.disciple.path = path;
        this.disciple.pathIndex = 0;
        this.logRuntime('INFO', 'TASK', 'task.assigned', '已分配拆除任务', {
            task: this.describeTask(this.disciple.currentTask),
            buildingId: building.id,
            buildingType: building.definition.id,
            targetTile: this.formatTile(standTile),
            pathLength: path.length,
        });
        this.setMessage(`弟子已前往拆除 ${building.definition.label}`);
    }

    private getDropoffTile(): TileCoord {
        return { col: 5, row: 6 };
    }

    private findBuildingStandTile(building: BuildingEntity): TileCoord | null {
        return this.findBuildingStandTileFromTile(building, this.disciple.tile);
    }

    private findBuildingStandTileFromTile(building: BuildingEntity, fromTile: TileCoord): TileCoord | null {
        const footprint = this.getFootprintTiles(building.origin, building.definition.width, building.definition.height);
        const candidateKeys = new Set<string>();
        const candidates: TileCoord[] = [];

        for (const tile of footprint) {
            const neighbors = this.getNeighbors(tile);
            for (const neighbor of neighbors) {
                const key = this.getTileKey(neighbor.col, neighbor.row);
                if (candidateKeys.has(key)) {
                    continue;
                }
                candidateKeys.add(key);
                if (this.isWalkable(neighbor)) {
                    candidates.push(neighbor);
                }
            }
        }

        if (candidates.length === 0) {
            return null;
        }

        candidates.sort((left, right) => manhattanDistance(fromTile, left) - manhattanDistance(fromTile, right));
        return candidates[0];
    }

    private findStandTileAroundTile(targetTile: TileCoord, fromTile: TileCoord): TileCoord | null {
        const candidates = this.getNeighbors(targetTile).filter((neighbor) => this.isWalkable(neighbor));
        if (candidates.length === 0) {
            return this.isWalkable(targetTile) ? { ...targetTile } : null;
        }

        candidates.sort((left, right) => manhattanDistance(fromTile, left) - manhattanDistance(fromTile, right));
        return candidates[0];
    }

    private getMissingResourcesForBuilding(building: BuildingEntity): Stockpile | null {
        const missing = EMPTY_STOCKPILE();
        let hasMissing = false;
        const cost = this.getActiveCostForBuilding(building);

        (Object.keys(cost) as ResourceKind[]).forEach((key) => {
            missing[key] = Math.max(0, cost[key] - building.supplied[key]);
            hasMissing ||= missing[key] > 0;
        });

        return hasMissing ? missing : null;
    }

    private showSelection(tile: TileCoord): void {
        const center = this.getTileCenter(tile);
        this.selectionCursor.active = true;
        this.selectionCursor.setPosition(center.x, center.y, 0);
    }

    private describeTile(tile: TileCoord): void {
        const building = this.getBuildingAtTile(tile);
        if (building) {
            const upgradeHint = this.canUpgradeBuilding(building)
                ? `，可长按升级到 Lv.${building.level + 1}`
                : '';
            this.setMessage(
                `已选中 ${building.definition.label} Lv.${building.level}，状态：${building.state}，HP ${Math.ceil(building.currentHp)}/${this.getBuildingMaxHp(building)}${upgradeHint}`,
            );
            return;
        }

        const resource = this.resourceNodes.get(this.getTileKey(tile.col, tile.row));
        if (resource) {
            this.setMessage(
                this.isResourceHarvestable(resource)
                    ? `已选中 ${RESOURCE_DISPLAY[resource.kind].title} 资源格，剩余 ${resource.remainingCharges}/${resource.maxCharges}`
                    : `已选中 ${RESOURCE_DISPLAY[resource.kind].title} 资源格，待刷新 ${Math.ceil(resource.regenTimerSeconds)}s`,
            );
            return;
        }

        if (this.isTileBuildable(tile)) {
            this.setMessage(`已选中可建地块 col=${tile.col}, row=${tile.row}`);
            return;
        }

        if (this.roadTiles.has(this.getTileKey(tile.col, tile.row))) {
            this.setMessage(`已选中道路地块 col=${tile.col}, row=${tile.row}`);
            return;
        }

        this.setMessage(`已选中阻挡地块 col=${tile.col}, row=${tile.row}`);
    }

    private drawSelectionCursor(): void {
        const graphics = this.selectionCursor.getComponent(Graphics) ?? this.selectionCursor.addComponent(Graphics);
        this.configureNodeSize(this.selectionCursor, this.tileWidth, this.tileHeight);
        graphics.clear();
        this.drawDiamond(graphics, 0, 0, new Color(255, 215, 64, 48), new Color(255, 215, 64, 255), 3);
    }

    private drawDiamond(graphics: Graphics, x: number, y: number, fillColor: Color, strokeColor: Color, inset = 4): void {
        const insetX = Math.max(this.halfTileWidth - inset, 8);
        const insetY = Math.max(this.halfTileHeight - inset * 0.5, 6);

        graphics.fillColor = fillColor;
        graphics.strokeColor = strokeColor;
        graphics.moveTo(x, y + insetY);
        graphics.lineTo(x + insetX, y);
        graphics.lineTo(x, y - insetY);
        graphics.lineTo(x - insetX, y);
        graphics.close();
        graphics.fill();
        graphics.stroke();
    }

    private pickTileFromUILocation(uiLocation: Readonly<Vec3>): TileCoord | null {
        const transform = this.mapRoot.getComponent(UITransform);
        if (!transform) {
            return null;
        }

        const local = transform.convertToNodeSpaceAR(v3(uiLocation.x, uiLocation.y, 0));
        let bestTile: TileCoord | null = null;
        let bestScore = Number.POSITIVE_INFINITY;

        for (let row = 0; row < this.mapRows; row += 1) {
            for (let col = 0; col < this.mapColumns; col += 1) {
                const center = this.tileCenters.get(this.getTileKey(col, row));
                if (!center) {
                    continue;
                }

                const dx = Math.abs(local.x - center.x);
                const dy = Math.abs(local.y - center.y);
                const normalized = dx / this.halfTileWidth + dy / this.halfTileHeight;

                if (normalized > 1 || normalized >= bestScore) {
                    continue;
                }

                bestScore = normalized;
                bestTile = { col, row };
            }
        }

        return bestTile;
    }

    private cacheTileCenters(): void {
        this.tileCenters.clear();
        if (!this.groundLayer) {
            return;
        }

        const originOffsetX = this.mapPixelWidth * 0.5;
        const originOffsetY = this.mapPixelHeight * 0.5;

        for (let row = 0; row < this.mapRows; row += 1) {
            for (let col = 0; col < this.mapColumns; col += 1) {
                const tilePosition = this.groundLayer.getPositionAt(col, row);
                const x = tilePosition.x + this.halfTileWidth - originOffsetX;
                const y = tilePosition.y + this.halfTileHeight - originOffsetY;
                this.tileCenters.set(this.getTileKey(col, row), v3(x, y, 0));
            }
        }
    }

    private getTileCenter(tile: TileCoord): Vec3 {
        return this.tileCenters.get(this.getTileKey(tile.col, tile.row))?.clone() ?? v3();
    }

    private isTileInBounds(tile: TileCoord): boolean {
        return tile.col >= 0 && tile.row >= 0 && tile.col < this.mapColumns && tile.row < this.mapRows;
    }

    private isTileBuildable(tile: TileCoord): boolean {
        const key = this.getTileKey(tile.col, tile.row);
        return (
            this.isTileInBounds(tile) &&
            !this.blockedTiles.has(key) &&
            !this.roadTiles.has(key) &&
            !this.resourceNodes.has(key) &&
            !this.isTileOccupied(tile)
        );
    }

    private isPlacementValid(origin: TileCoord, definition: BuildingDefinition): boolean {
        const footprint = this.getFootprintTiles(origin, definition.width, definition.height);
        return footprint.every((tile) => this.isTileBuildable(tile));
    }

    private isTileOccupied(tile: TileCoord): boolean {
        const key = this.getTileKey(tile.col, tile.row);
        for (const building of this.buildingEntities.values()) {
            const footprint = this.getFootprintTiles(building.origin, building.definition.width, building.definition.height);
            if (footprint.some((entry) => this.getTileKey(entry.col, entry.row) === key)) {
                return true;
            }
        }
        return false;
    }

    private getBuildingAtTile(tile: TileCoord): BuildingEntity | null {
        const key = this.getTileKey(tile.col, tile.row);
        for (const building of this.buildingEntities.values()) {
            const footprint = this.getFootprintTiles(building.origin, building.definition.width, building.definition.height);
            if (footprint.some((entry) => this.getTileKey(entry.col, entry.row) === key)) {
                return building;
            }
        }
        return null;
    }

    private getFootprintTiles(origin: TileCoord, width: number, height: number): TileCoord[] {
        const tiles: TileCoord[] = [];
        for (let rowOffset = 0; rowOffset < height; rowOffset += 1) {
            for (let colOffset = 0; colOffset < width; colOffset += 1) {
                tiles.push({
                    col: origin.col + colOffset,
                    row: origin.row + rowOffset,
                });
            }
        }
        return tiles;
    }

    private getFootprintFrame(tiles: TileCoord[]): { center: Vec3; width: number; height: number } {
        const centers = tiles.map((tile) => this.getTileCenter(tile));
        const xs = centers.map((center) => center.x);
        const ys = centers.map((center) => center.y);
        const minX = Math.min(...xs);
        const maxX = Math.max(...xs);
        const minY = Math.min(...ys);
        const maxY = Math.max(...ys);

        return {
            center: v3((minX + maxX) * 0.5, (minY + maxY) * 0.5, 0),
            width: maxX - minX + this.tileWidth,
            height: maxY - minY + this.tileHeight,
        };
    }

    private findNearestBuildableTile(start: TileCoord | null): TileCoord | null {
        const initial = start && this.isTileInBounds(start) ? start : { col: Math.floor(this.mapColumns * 0.5), row: Math.floor(this.mapRows * 0.5) };
        if (this.isTileBuildable(initial)) {
            return initial;
        }

        const queue: TileCoord[] = [initial];
        const visited = new Set<string>([this.getTileKey(initial.col, initial.row)]);

        while (queue.length > 0) {
            const current = queue.shift()!;
            for (const neighbor of this.getNeighbors(current)) {
                const key = this.getTileKey(neighbor.col, neighbor.row);
                if (visited.has(key)) {
                    continue;
                }
                visited.add(key);
                if (this.isTileBuildable(neighbor)) {
                    return neighbor;
                }
                queue.push(neighbor);
            }
        }

        return null;
    }

    private findScreenCenterTile(): TileCoord | null {
        return this.pickTileFromUILocation(v3(view.getVisibleSize().width * 0.5, view.getVisibleSize().height * 0.5, 0));
    }

    private getNeighbors(tile: TileCoord): TileCoord[] {
        return [
            { col: tile.col + 1, row: tile.row },
            { col: tile.col - 1, row: tile.row },
            { col: tile.col, row: tile.row + 1 },
            { col: tile.col, row: tile.row - 1 },
        ].filter((neighbor) => this.isTileInBounds(neighbor));
    }

    private isWalkable(tile: TileCoord): boolean {
        if (!this.isTileInBounds(tile)) {
            return false;
        }

        const key = this.getTileKey(tile.col, tile.row);
        return !this.blockedTiles.has(key) && !this.isTileOccupied(tile);
    }

    private findPath(start: TileCoord, goal: TileCoord): TileCoord[] | null {
        if (!this.isTileInBounds(start) || !this.isTileInBounds(goal)) {
            return null;
        }

        if (start.col === goal.col && start.row === goal.row) {
            return [];
        }

        const openSet = new Set<string>([this.getTileKey(start.col, start.row)]);
        const cameFrom = new Map<string, string>();
        const gScore = new Map<string, number>([[this.getTileKey(start.col, start.row), 0]]);
        const fScore = new Map<string, number>([
            [this.getTileKey(start.col, start.row), manhattanDistance(start, goal)],
        ]);

        while (openSet.size > 0) {
            const currentKey = this.pickLowestScoreKey(openSet, fScore);
            if (!currentKey) {
                break;
            }

            const current = this.parseTileKey(currentKey);
            if (current.col === goal.col && current.row === goal.row) {
                return this.reconstructPath(cameFrom, currentKey);
            }

            openSet.delete(currentKey);

            for (const neighbor of this.getNeighbors(current)) {
                const neighborKey = this.getTileKey(neighbor.col, neighbor.row);
                if (!this.isWalkable(neighbor) && neighborKey !== this.getTileKey(goal.col, goal.row)) {
                    continue;
                }

                const tentativeG =
                    (gScore.get(currentKey) ?? Number.POSITIVE_INFINITY) + this.getTileTraversalCost(neighbor);
                if (tentativeG >= (gScore.get(neighborKey) ?? Number.POSITIVE_INFINITY)) {
                    continue;
                }

                cameFrom.set(neighborKey, currentKey);
                gScore.set(neighborKey, tentativeG);
                fScore.set(neighborKey, tentativeG + manhattanDistance(neighbor, goal));
                openSet.add(neighborKey);
            }
        }

        return null;
    }

    private getTileTraversalCost(tile: TileCoord): number {
        return this.roadTiles.has(this.getTileKey(tile.col, tile.row)) ? 0.7 : 1;
    }

    private pickLowestScoreKey(keys: Set<string>, scores: Map<string, number>): string | null {
        let bestKey: string | null = null;
        let bestScore = Number.POSITIVE_INFINITY;

        for (const key of keys) {
            const score = scores.get(key) ?? Number.POSITIVE_INFINITY;
            if (score < bestScore) {
                bestScore = score;
                bestKey = key;
            }
        }

        return bestKey;
    }

    private reconstructPath(cameFrom: Map<string, string>, currentKey: string): TileCoord[] {
        const pathKeys: string[] = [currentKey];
        let walker = currentKey;
        while (cameFrom.has(walker)) {
            walker = cameFrom.get(walker)!;
            pathKeys.unshift(walker);
        }

        pathKeys.shift();
        return pathKeys.map((key) => this.parseTileKey(key));
    }

    private parseTileKey(key: string): TileCoord {
        const [col, row] = key.split(',').map((entry) => Number.parseInt(entry, 10));
        return { col, row };
    }

    private renderStatus(): void {
        const guidance = this.getSessionGuidance();
        const modeText =
            this.inputMode === 'build_select'
                ? '建造-选型'
                : this.inputMode === 'build_place'
                  ? '建造-摆放'
                  : this.inputMode === 'gather'
                    ? '采集'
                    : this.inputMode === 'demolish'
                      ? '拆除'
                      : '查看';

        const stockText = `木 ${this.stockpile.spirit_wood} / 石 ${this.stockpile.spirit_stone} / 药 ${this.stockpile.herb}`;
        const buildingCounts = this.getBuildingStateCounts();
        const discipleTask = this.describeTask(this.disciple.currentTask) ?? '待命';
        const elapsedMinutes = Math.floor(this.sessionElapsedSeconds / 60);
        const elapsedSeconds = Math.floor(this.sessionElapsedSeconds % 60);
        const limitMinutes = Math.floor(SESSION_TARGET_SECONDS / 60);
        const limitSeconds = Math.floor(SESSION_TARGET_SECONDS % 60);
        const authorityText = this.authorityConnected ? `权威 ${this.authoritySessionId}` : `本地回退 ${this.authorityLastError ?? 'offline'}`;

        const lines = [
            `阶段 ${this.getSessionPhaseLabel()} ${elapsedMinutes.toString().padStart(2, '0')}:${elapsedSeconds.toString().padStart(2, '0')}/${limitMinutes.toString().padStart(2, '0')}:${limitSeconds.toString().padStart(2, '0')} | ${guidance.headline}`,
            `目标 ${guidance.detail}`,
            `局势 ${guidance.threat} | 库存 ${stockText}`,
            `模式 ${modeText} | ${authorityText} | 弟子 ${discipleTask} HP ${Math.ceil(this.disciple.currentHp)}/${this.disciple.model.stats.maxHp} | 建筑 启${buildingCounts.active} 损${buildingCounts.damaged} 蓝${buildingCounts.planned}`,
            `提示 ${this.lastMessage}`,
        ];

        if (this.statusLabel) {
            this.statusLabel.string = lines.join('\n');
        } else {
            console.log(`[SectMapBootstrap] ${lines.join(' | ')}`);
        }
    }

    private setMessage(message: string): void {
        if (this.lastMessage === message) {
            return;
        }
        this.lastMessage = message;
        this.logRuntime('INFO', 'HUD', 'status.message', '状态栏消息已更新', {
            mode: this.inputMode,
            message,
        });
    }

    private attachRuntimeDebugBridge(): void {
        const runtimeGlobal = globalThis as RuntimeDebugGlobal;
        runtimeGlobal.__MIS_RUNTIME_DEBUG__ = {
            getRecentLogs: (limit = 40) => this.runtimeLogs.slice(-Math.max(1, Math.floor(limit))),
            clearLogs: () => {
                this.runtimeLogs = [];
                this.runtimeLogSeq = 0;
                this.logRuntime('INFO', 'HUD', 'logs.cleared', '预览页运行时日志缓存已清空');
            },
            getSnapshot: () => this.getRuntimeSnapshot(),
        };
    }

    private detachRuntimeDebugBridge(): void {
        const runtimeGlobal = globalThis as RuntimeDebugGlobal;
        delete runtimeGlobal.__MIS_RUNTIME_DEBUG__;
    }

    private getRuntimeSnapshot(): RuntimeSnapshot {
        const visibleSize = view.getVisibleSize();
        const designResolution = view.getDesignResolutionSize();
        const screenMetrics = getRuntimeScreenMetrics();
        const screenMetricsFallback = getRuntimeScreenMetrics({ forceSafeAreaFallback: true });
        const guidance = this.getSessionGuidance();

        return {
            mapReady: this.mapReady,
            inputMode: this.inputMode,
            lastMessage: this.lastMessage,
            selectedTile: this.formatTile(this.selectedTile),
            viewport: {
                designWidth: designResolution.width,
                designHeight: designResolution.height,
                visibleWidth: visibleSize.width,
                visibleHeight: visibleSize.height,
            },
            mapContract: this.getMapContractSnapshot(),
            screenMetrics,
            screenMetricsFallback,
            portraitBaseline: this.portraitBaselineSnapshot,
            stockpile: { ...this.stockpile },
            resourceSummary: this.getResourceSummary(),
            resourceNodes: this.getRuntimeResourceNodes(),
            designatedResources: [...this.resourceNodes.values()].filter((resource) => resource.designated).length,
            buildingCounts: this.getBuildingStateCounts(),
            buildings: this.getRuntimeBuildingSnapshots(),
            session: {
                phase: this.sessionPhase,
                outcome: this.sessionOutcome,
                elapsedSeconds: Number(this.sessionElapsedSeconds.toFixed(2)),
                limitSeconds: SESSION_TARGET_SECONDS,
                objective: this.sessionObjectiveText,
                guidanceHeadline: guidance.headline,
                guidanceDetail: guidance.detail,
                guidanceThreat: guidance.threat,
                markerTile: this.formatTile(guidance.markerTile),
                markerText: guidance.markerText,
                focusBuildingId: guidance.focusBuildingId,
                raidCountdownSeconds: Number(this.hostileNpc.respawnTimerSeconds.toFixed(2)),
                guardTowerId: this.sessionGuardTowerId,
                ruinBuildingId: this.sessionRuinBuildingId,
                firstRaidTriggered: this.firstRaidTriggered,
                firstRaidResolved: this.firstRaidResolved,
            },
            disciple: {
                tile: this.formatTile(this.disciple.tile) ?? 'unknown',
                visualState: this.disciple.visualState,
                carrying: this.disciple.carrying,
                task: this.describeTask(this.disciple.currentTask),
                hp: Math.ceil(this.disciple.currentHp),
                maxHp: this.disciple.model.stats.maxHp,
                model: this.disciple.model,
            },
            hostiles: this.getRuntimeHostileSnapshots(),
            authority: {
                mode: this.authorityMode,
                connected: this.authorityConnected,
                sessionId: this.authoritySessionId,
                baseUrl: this.authorityClient.getBaseUrl(),
                pendingCommands: [...this.authorityPendingCommands.values()].sort(),
                lastEvent: this.authorityLastEvent,
                lastError: this.authorityLastError,
            },
        };
    }

    private getMapContractSnapshot(): RuntimeMapContractSnapshot {
        return {
            columns: this.mapColumns,
            rows: this.mapRows,
            tileWidth: this.tileWidth,
            tileHeight: this.tileHeight,
            roadTiles: this.roadTiles.size,
            blockedTiles: this.blockedTiles.size,
            resourceTiles: this.resourceNodes.size,
            buildableTiles: this.buildableTileCount,
            mainHallTile: this.formatTile(INITIAL_MAIN_HALL_TILE) ?? 'unknown',
            discipleTile: this.formatTile(INITIAL_DISCIPLE_TILE) ?? 'unknown',
        };
    }

    private getResourceSummary(): RuntimeResourceSummary {
        const summary: RuntimeResourceSummary = {
            designatedCount: 0,
            availableCount: 0,
            regeneratingCount: 0,
            totalRemainingCharges: 0,
            byKind: {
                spirit_wood: {
                    designatedNodes: 0,
                    availableNodes: 0,
                    regeneratingNodes: 0,
                    remainingCharges: 0,
                },
                spirit_stone: {
                    designatedNodes: 0,
                    availableNodes: 0,
                    regeneratingNodes: 0,
                    remainingCharges: 0,
                },
                herb: {
                    designatedNodes: 0,
                    availableNodes: 0,
                    regeneratingNodes: 0,
                    remainingCharges: 0,
                },
            },
        };

        for (const resource of this.resourceNodes.values()) {
            const kindSummary = summary.byKind[resource.kind];
            if (resource.designated) {
                summary.designatedCount += 1;
                kindSummary.designatedNodes += 1;
            }

            if (this.isResourceHarvestable(resource)) {
                summary.availableCount += 1;
                kindSummary.availableNodes += 1;
            } else {
                summary.regeneratingCount += 1;
                kindSummary.regeneratingNodes += 1;
            }

            summary.totalRemainingCharges += resource.remainingCharges;
            kindSummary.remainingCharges += resource.remainingCharges;
        }

        return summary;
    }

    private getRuntimeResourceNodes(): RuntimeResourceNodeSnapshot[] {
        return [...this.resourceNodes.values()]
            .map((resource) => ({
                tile: this.formatTile(resource.tile) ?? 'unknown',
                kind: resource.kind,
                designated: resource.designated,
                state: resource.state,
                remainingCharges: resource.remainingCharges,
                maxCharges: resource.maxCharges,
                regenSeconds: resource.regenSeconds,
                regenTimerSeconds: Math.max(0, Number(resource.regenTimerSeconds.toFixed(2))),
            }))
            .sort((left, right) => left.tile.localeCompare(right.tile));
    }

    private getRuntimeBuildingSnapshots(): RuntimeBuildingSnapshot[] {
        return [...this.buildingEntities.values()]
            .map((building) => ({
                id: building.id,
                type: building.definition.id,
                origin: this.formatTile(building.origin) ?? 'unknown',
                state: building.state,
                level: building.level,
                hp: Math.ceil(building.currentHp),
                maxHp: this.getBuildingMaxHp(building),
                markedForDemolition: building.markedForDemolition,
                pendingAction: building.pendingAction,
            }))
            .sort((left, right) => left.id.localeCompare(right.id));
    }

    private getRuntimeHostileSnapshots(): RuntimeHostileSnapshot[] {
        if (!this.hostileNpc.active) {
            return [];
        }

        return [
            {
                id: this.hostileNpc.id,
                archetypeId: this.hostileNpc.model.archetypeId,
                tile: this.formatTile(this.hostileNpc.tile) ?? 'unknown',
                visualState: this.hostileNpc.visualState,
                hp: Math.ceil(this.hostileNpc.currentHp),
                maxHp: this.hostileNpc.model.stats.maxHp,
                active: this.hostileNpc.active,
                targetBuildingId: this.hostileNpc.targetBuildingId,
            },
        ];
    }

    private getBuildingStateCounts(): Record<BuildingState, number> {
        const counts: Record<BuildingState, number> = {
            planned: 0,
            supplied: 0,
            constructing: 0,
            active: 0,
            damaged: 0,
        };

        for (const building of this.buildingEntities.values()) {
            counts[building.state] += 1;
        }

        return counts;
    }

    private getSessionPhaseLabel(): string {
        switch (this.sessionPhase) {
            case 'clear_ruin':
                return '清旧仓';
            case 'place_guard_tower':
                return '建护山台';
            case 'upgrade_guard_tower':
                return '升护山台';
            case 'raid_countdown':
                return '敌袭倒计时';
            case 'defend':
                return '守首波';
            case 'recover':
                return '修复恢复';
            case 'victory':
                return '已达成';
            case 'defeat':
                return '已失败';
            default:
                return this.sessionPhase;
        }
    }

    private setBuildingState(building: BuildingEntity, nextState: BuildingState, reason: string): void {
        const previousState = building.state;
        if (previousState === nextState) {
            return;
        }

        building.state = nextState;
        this.logRuntime('INFO', 'BUILD', 'build.state_change', '建筑状态已变更', {
            buildingId: building.id,
            buildingType: building.definition.id,
            origin: this.formatTile(building.origin),
            from: previousState,
            to: nextState,
            reason,
        });
    }

    private describeTask(task: DiscipleTask | null): string | null {
        if (!task) {
            return null;
        }

        return `${task.kind}:${task.phase}`;
    }

    private summarizeStockpile(): string {
        return `wood=${this.stockpile.spirit_wood},stone=${this.stockpile.spirit_stone},herb=${this.stockpile.herb}`;
    }

    private formatTile(tile: TileCoord | null): string | null {
        if (!tile) {
            return null;
        }

        return `${tile.col},${tile.row}`;
    }

    private logRuntime(
        level: RuntimeLogLevel,
        channel: RuntimeLogChannel,
        event: string,
        message: string,
        payload?: RuntimeLogPayload,
    ): void {
        const entry: RuntimeLogEntry = {
            seq: this.runtimeLogSeq + 1,
            timestamp: new Date().toISOString(),
            elapsedMs: Date.now() - this.runtimeSessionStartedAt,
            level,
            channel,
            event,
            message,
            payload,
        };

        this.runtimeLogSeq = entry.seq;
        this.runtimeLogs.push(entry);
        if (this.runtimeLogs.length > this.runtimeLogLimit) {
            this.runtimeLogs.shift();
        }

        const serializedPayload = this.serializeLogPayload(payload);
        const line = `[MIS-RT][${level}][${channel}][#${entry.seq}] ${event}: ${message}${serializedPayload ? ` | ${serializedPayload}` : ''}`;

        switch (level) {
            case 'ERROR':
                console.error(line);
                break;
            case 'WARN':
                console.warn(line);
                break;
            case 'DEBUG':
                console.debug(line);
                break;
            case 'INFO':
            default:
                console.info(line);
                break;
        }
    }

    private serializeLogPayload(payload?: RuntimeLogPayload): string {
        if (!payload || Object.keys(payload).length === 0) {
            return '';
        }

        try {
            return JSON.stringify(payload);
        } catch {
            return '{"error":"payload_not_serializable"}';
        }
    }

    private isLocationBlockedByScreenUI(location: Readonly<Vec3>): boolean {
        const blockers = [this.toolbarRoot, this.buildPanelRoot, this.radialMenuRoot];
        return blockers.some((node) => this.isPointInsideNode(node, location));
    }

    private isPointInsideNode(node: Node | null, location: Readonly<Vec3>): boolean {
        if (!node || !node.activeInHierarchy) {
            return false;
        }

        const transform = node.getComponent(UITransform);
        if (transform) {
            const worldBounds = transform.getBoundingBoxToWorld();
            if (worldBounds.contains(v3(location.x, location.y, 0))) {
                return true;
            }
        }

        return node.children.some((child) => this.isPointInsideNode(child, location));
    }

    private getCostText(cost: Stockpile): string {
        const parts: string[] = [];
        (Object.keys(cost) as ResourceKind[]).forEach((key) => {
            if (cost[key] > 0) {
                parts.push(`${RESOURCE_DISPLAY[key].short}${cost[key]}`);
            }
        });
        return parts.length > 0 ? parts.join(' ') : '无需材料';
    }

    private getTileKey(col: number, row: number): string {
        return `${col},${row}`;
    }

    private configureNodeSize(target: Node, width: number, height: number, anchorX = 0.5, anchorY = 0.5): void {
        const transform = target.getComponent(UITransform) ?? target.addComponent(UITransform);
        transform.setContentSize(width, height);
        transform.anchorX = anchorX;
        transform.anchorY = anchorY;
    }

    private ensureChild(name: string, parent: Node = this.node): Node {
        const existing = parent.getChildByName(name);
        if (existing) {
            return existing;
        }

        const created = new Node(name);
        created.setParent(parent);
        return created;
    }
}

function withAlpha(color: Color, alpha: number): Color {
    return new Color(color.r, color.g, color.b, alpha);
}

function manhattanDistance(left: TileCoord, right: TileCoord): number {
    return Math.abs(left.col - right.col) + Math.abs(left.row - right.row);
}
