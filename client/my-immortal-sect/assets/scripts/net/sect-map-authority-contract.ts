export type AuthorityResourceKind = 'spirit_wood' | 'spirit_stone' | 'herb';
export type AuthorityBuildingType = 'main_hall' | 'disciple_quarters' | 'warehouse' | 'herb_garden' | 'guard_tower';
export type AuthorityBuildingState = 'planned' | 'supplied' | 'constructing' | 'active' | 'damaged';
export type AuthorityBuildingWorkKind = 'build' | 'upgrade';
export type AuthorityDiscipleAssignmentKind = 'idle' | 'gather' | 'haul' | 'build' | 'repair' | 'guard' | 'demolish';
export type AuthoritySessionPhase =
    | 'clear_ruin'
    | 'place_guard_tower'
    | 'upgrade_guard_tower'
    | 'raid_countdown'
    | 'defend'
    | 'recover'
    | 'second_cycle_ready'
    | 'victory'
    | 'defeat';
export type AuthoritySessionOutcome = 'in_progress' | 'victory' | 'defeat';

export type AuthoritySessionRecoverReason =
    | 'none'
    | 'damaged_buildings'
    | 'resource_regeneration'
    | 'damaged_buildings_and_resource_regeneration';
export type AuthorityBootstrapMode = 'restore_latest' | 'reset';

export type AuthorityTileCoord = {
    col: number;
    row: number;
};

export type AuthorityStockpile = Record<AuthorityResourceKind, number>;

export type AuthorityResourceNodeState = 'available' | 'regenerating';

export type AuthorityResourceNodeSnapshot = {
    tile: AuthorityTileCoord;
    kind: AuthorityResourceKind;
    designated: boolean;
    state: AuthorityResourceNodeState;
    remainingCharges: number;
    maxCharges: number;
    regenSeconds: number;
    regenTimerSeconds: number;
};

export type AuthorityBuildingSnapshot = {
    id: string;
    type: AuthorityBuildingType;
    origin: AuthorityTileCoord;
    state: AuthorityBuildingState;
    level: number;
    hp: number;
    maxHp: number;
    durability?: number;
    efficiency?: number;
    maintenanceDebt?: number;
    damagedReason?: string | null;
    markedForDemolition: boolean;
    pendingAction: AuthorityBuildingWorkKind | null;
    pendingLevel: number | null;
    supplied: AuthorityStockpile;
};

export type AuthorityDiscipleCarryingSnapshot = {
    kind: AuthorityResourceKind | null;
    amount: number;
};

export type AuthorityDiscipleSnapshot = {
    archetypeId: 'sect_disciple' | 'bandit_scout';
    id: string;
    name: string;
    assignmentKind: AuthorityDiscipleAssignmentKind;
    targetBuildingId: string | null;
    targetResourceKind: AuthorityResourceKind | null;
    targetTile: AuthorityTileCoord | null;
    carrying: AuthorityDiscipleCarryingSnapshot;
    hp: number;
    maxHp: number;
    visualState: 'idle' | 'moving' | 'working' | 'carrying' | 'guarding' | 'attacking' | 'injured';
    workProgressTicks: number;
    expectedNextTransition: string | null;
};

export type AuthorityHostileSnapshot = {
    id: string;
    archetypeId: 'sect_disciple' | 'bandit_scout';
    name: string;
    tile: AuthorityTileCoord;
    hp: number;
    maxHp: number;
    visualState: 'idle' | 'moving' | 'working' | 'carrying' | 'guarding' | 'attacking' | 'injured';
    active: boolean;
    targetBuildingId: string | null;
};

export type AuthoritySessionSnapshot = {
    phase: AuthoritySessionPhase;
    outcome: AuthoritySessionOutcome;
    objective: string;
    guardTowerId: string | null;
    ruinBuildingId: string | null;
    firstRaidTriggered: boolean;
    firstRaidResolved: boolean;
    raidCountdownSeconds: number;
    defendRemainingSeconds: number;
    recoverReason: AuthoritySessionRecoverReason;
    damagedBuildingCount: number;
    regeneratingNodeCount: number;
    riskIntensity: number;
    riskMitigation: number;
    threatCurve: number;
    defenseRating: number;
    guardDiscipleCount: number;
    omenStatus: string;
    omenText: string;
    defenseSummary: string;
    damageSummary: string;
    repairSuggestion: string;
    sourceSummary?: Array<{
        source: string;
        label: string;
        delta: number;
    }>;
};

export type AuthoritySnapshot = {
    sessionId: string;
    gameTick: number;
    stockpile: AuthorityStockpile;
    resourceNodes: AuthorityResourceNodeSnapshot[];
    buildings: AuthorityBuildingSnapshot[];
    disciples: AuthorityDiscipleSnapshot[];
    hostiles: AuthorityHostileSnapshot[];
    session: AuthoritySessionSnapshot;
};

export type AuthorityCommandName =
    | 'place_building'
    | 'request_upgrade'
    | 'toggle_demolition'
    | 'set_resource_designation'
    | 'collect_stockpile'
    | 'deliver_build_resource'
    | 'complete_demolition'
    | 'expire_session'
    | 'trigger_first_raid'
    | 'resolve_first_raid';

export type AuthorityCommandEnvelope<TPayload = unknown> = {
    name: AuthorityCommandName;
    payload: TPayload;
};

export type AuthorityCommandResult = {
    accepted: boolean;
    event: string;
    message: string;
};

export type AuthoritySessionResponse = {
    identity: {
        playerId: string;
        playerToken: string;
        playerSessionId: string;
    };
    snapshot: AuthoritySnapshot;
    result?: AuthorityCommandResult;
};
