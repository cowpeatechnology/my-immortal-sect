export type LocalUnitArchetypeId = 'sect_disciple' | 'bandit_scout';
export type LocalUnitFaction = 'sect' | 'hostile';
export type LocalUnitRole = 'worker_guard' | 'raider';

export type LocalUnitAttributes = {
    physique: number;
    agility: number;
    mind: number;
    spirit: number;
    craft: number;
    resolve: number;
};

export type LocalUnitStats = {
    maxHp: number;
    moveSpeed: number;
    harvestSpeed: number;
    haulSpeed: number;
    buildSpeed: number;
    repairSpeed: number;
    attackPower: number;
    defense: number;
    attackInterval: number;
    attackRange: number;
    alertRange: number;
};

export type LocalUnitModel = {
    archetypeId: LocalUnitArchetypeId;
    faction: LocalUnitFaction;
    role: LocalUnitRole;
    displayName: string;
    attributes: LocalUnitAttributes;
    stats: LocalUnitStats;
};

type LocalUnitPreset = {
    faction: LocalUnitFaction;
    role: LocalUnitRole;
    displayName: string;
    attributes: LocalUnitAttributes;
};

const UNIT_PRESETS: Record<LocalUnitArchetypeId, LocalUnitPreset> = {
    sect_disciple: {
        faction: 'sect',
        role: 'worker_guard',
        displayName: '玄',
        attributes: {
            physique: 3,
            agility: 3,
            mind: 2,
            spirit: 2,
            craft: 3,
            resolve: 3,
        },
    },
    bandit_scout: {
        faction: 'hostile',
        role: 'raider',
        displayName: '寇',
        attributes: {
            physique: 3,
            agility: 4,
            mind: 1,
            spirit: 1,
            craft: 1,
            resolve: 2,
        },
    },
};

function round(value: number): number {
    return Number(value.toFixed(2));
}

function deriveUnitStats(attributes: LocalUnitAttributes, role: LocalUnitRole): LocalUnitStats {
    const maxHp = 10 + attributes.physique * 3 + attributes.resolve * 2;
    const moveSpeed = 190 + attributes.agility * 18 + attributes.resolve * 4;
    const harvestSpeed = 0.78 + attributes.craft * 0.08 + attributes.physique * 0.03;
    const haulSpeed = 0.82 + attributes.agility * 0.07 + attributes.resolve * 0.05;
    const buildSpeed = 0.78 + attributes.craft * 0.09 + attributes.mind * 0.05;
    const repairSpeed = 0.76 + attributes.craft * 0.07 + attributes.spirit * 0.08;
    const baseAttack = role === 'raider' ? 2.8 : 2.2;
    const attackPower = baseAttack + attributes.physique * 0.45 + attributes.spirit * 0.25;
    const defense = 0.8 + attributes.resolve * 0.45 + attributes.physique * 0.15;
    const attackInterval = Math.max(0.7, 1.55 - attributes.agility * 0.08);
    const attackRange = role === 'raider' ? 1 : 1;
    const alertRange = 4 + attributes.mind * 0.5 + attributes.resolve * 0.5;

    return {
        maxHp,
        moveSpeed,
        harvestSpeed: round(harvestSpeed),
        haulSpeed: round(haulSpeed),
        buildSpeed: round(buildSpeed),
        repairSpeed: round(repairSpeed),
        attackPower: round(attackPower),
        defense: round(defense),
        attackInterval: round(attackInterval),
        attackRange: round(attackRange),
        alertRange: round(alertRange),
    };
}

export function createLocalUnitModel(archetypeId: LocalUnitArchetypeId, displayName?: string): LocalUnitModel {
    const preset = UNIT_PRESETS[archetypeId];

    return {
        archetypeId,
        faction: preset.faction,
        role: preset.role,
        displayName: displayName ?? preset.displayName,
        attributes: {
            ...preset.attributes,
        },
        stats: deriveUnitStats(preset.attributes, preset.role),
    };
}

export function getActionDurationSeconds(baseSeconds: number, speedMultiplier: number): number {
    return round(baseSeconds / Math.max(0.35, speedMultiplier));
}

export function getMitigatedDamage(rawDamage: number, defense: number): number {
    return Math.max(1, Math.round(rawDamage - defense * 0.45));
}
