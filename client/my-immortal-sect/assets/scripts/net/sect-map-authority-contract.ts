export type AuthorityResourceKind = 'spirit_wood' | 'spirit_stone' | 'herb';
export type AuthorityBuildingType = 'main_hall' | 'disciple_quarters' | 'warehouse' | 'herb_garden' | 'guard_tower';
export type AuthorityBuildingState = 'planned' | 'supplied' | 'constructing' | 'active' | 'damaged';
export type AuthorityBuildingWorkKind = 'build' | 'upgrade';
export type AuthoritySessionPhase =
    | 'clear_ruin'
    | 'place_guard_tower'
    | 'upgrade_guard_tower'
    | 'raid_countdown'
    | 'defend'
    | 'recover'
    | 'victory'
    | 'defeat';
export type AuthoritySessionOutcome = 'in_progress' | 'victory' | 'defeat';

export type AuthorityTileCoord = {
    col: number;
    row: number;
};

export type AuthorityStockpile = Record<AuthorityResourceKind, number>;

export type AuthorityBuildingSnapshot = {
    id: string;
    type: AuthorityBuildingType;
    origin: AuthorityTileCoord;
    state: AuthorityBuildingState;
    level: number;
    hp: number;
    maxHp: number;
    markedForDemolition: boolean;
    pendingAction: AuthorityBuildingWorkKind | null;
    pendingLevel: number | null;
    supplied: AuthorityStockpile;
};

export type AuthoritySessionSnapshot = {
    phase: AuthoritySessionPhase;
    outcome: AuthoritySessionOutcome;
    objective: string;
    guardTowerId: string | null;
    ruinBuildingId: string | null;
    firstRaidTriggered: boolean;
    firstRaidResolved: boolean;
};

export type AuthoritySnapshot = {
    sessionId: string;
    stockpile: AuthorityStockpile;
    buildings: AuthorityBuildingSnapshot[];
    session: AuthoritySessionSnapshot;
};

export type AuthorityCommandName =
    | 'place_building'
    | 'request_upgrade'
    | 'toggle_demolition'
    | 'collect_stockpile'
    | 'deliver_build_resource'
    | 'start_building_work'
    | 'complete_building_work'
    | 'complete_demolition'
    | 'complete_repair'
    | 'sync_session_progress';

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
    snapshot: AuthoritySnapshot;
    result?: AuthorityCommandResult;
};
