import { SECT_MAP_SHARED_SHORT_SESSION_CONFIG } from './sect-map-shared-config.generated';

export type SectMapSharedTileCoord = {
    col: number;
    row: number;
};

export type SectMapSharedStockpile = {
    spirit_wood: number;
    spirit_stone: number;
    herb: number;
};

export type SectMapSharedResourceRule = {
    kind: string;
    label: string;
    maxCharges: number;
    regenSeconds: number;
    visualAssetId: string;
};

export type SectMapSharedBuildingBootstrapSeed = {
    origin: SectMapSharedTileCoord;
    level: number;
    state: string;
    hp?: number;
    role?: string;
};

export type SectMapSharedBuildingCoreDefinition = {
    id: string;
    label: string;
    width: number;
    height: number;
    cost: SectMapSharedStockpile;
    maxHp: number;
    repairCost: SectMapSharedStockpile;
    visualAssetId: string;
    bootstrapSeed?: SectMapSharedBuildingBootstrapSeed;
};

export const SECT_MAP_SHARED_RESOURCE_RULES = Object.fromEntries(
    SECT_MAP_SHARED_SHORT_SESSION_CONFIG.resource_kinds.map((entry) => [
        entry.runtime_key,
        {
            kind: entry.runtime_key,
            label: entry.title,
            maxCharges: entry.node_rule.max_charges,
            regenSeconds: entry.node_rule.regen_seconds,
            visualAssetId: entry.visual_asset_id,
        },
    ]),
) as Record<string, SectMapSharedResourceRule>;

export const SECT_MAP_SHARED_BUILDING_CORE_DEFINITIONS = SECT_MAP_SHARED_SHORT_SESSION_CONFIG.building_types.map((entry) => ({
    id: entry.runtime_key,
    label: entry.title,
    width: entry.footprint.width,
    height: entry.footprint.height,
    cost: {
        spirit_wood: entry.base_cost.spirit_wood,
        spirit_stone: entry.base_cost.spirit_stone,
        herb: entry.base_cost.herb,
    },
    maxHp: entry.base_max_hp,
    repairCost: {
        spirit_wood: entry.repair_cost.spirit_wood,
        spirit_stone: entry.repair_cost.spirit_stone,
        herb: entry.repair_cost.herb,
    },
    visualAssetId: entry.visual_asset_id,
    bootstrapSeed: entry.bootstrap_seed
        ? {
              origin: {
                  col: entry.bootstrap_seed.origin.col,
                  row: entry.bootstrap_seed.origin.row,
              },
              level: entry.bootstrap_seed.level,
              state: entry.bootstrap_seed.state,
              hp: entry.bootstrap_seed.hp,
              role: entry.bootstrap_seed.role,
          }
        : undefined,
})) as SectMapSharedBuildingCoreDefinition[];

export const SECT_MAP_SHARED_BUILDING_BY_ID = Object.fromEntries(
    SECT_MAP_SHARED_BUILDING_CORE_DEFINITIONS.map((entry) => [entry.id, entry]),
) as Record<string, SectMapSharedBuildingCoreDefinition>;

export const SECT_MAP_SHARED_MAIN_HALL_TILE = getBootstrapTile('main_hall');
export const SECT_MAP_SHARED_INITIAL_RUIN_TILE = getBootstrapTile('initial_ruin');

export function buildSectMapRasterSpriteFramePath(assetId: string): string | null {
    const [category, name] = assetId.split('.', 2);
    if ((category !== 'building' && category !== 'resource') || !name) {
        return null;
    }

    return `generated-buildings/sect-map-raster/${name}/spriteFrame`;
}

function getBootstrapTile(role: string): SectMapSharedTileCoord {
    const seeded = SECT_MAP_SHARED_BUILDING_CORE_DEFINITIONS.find((entry) => entry.bootstrapSeed?.role === role);
    if (!seeded?.bootstrapSeed) {
        throw new Error(`Missing shared sect-map bootstrap seed for role ${role}`);
    }

    return {
        col: seeded.bootstrapSeed.origin.col,
        row: seeded.bootstrapSeed.origin.row,
    };
}
