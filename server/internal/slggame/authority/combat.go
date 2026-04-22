package authority

type UnitFaction string

const (
	UnitFactionSect    UnitFaction = "sect"
	UnitFactionHostile UnitFaction = "hostile"
)

type UnitRole string

const (
	UnitRoleWorkerGuard UnitRole = "worker_guard"
	UnitRoleRaider      UnitRole = "raider"
)

type UnitAttributes struct {
	Physique int
	Agility  int
	Mind     int
	Spirit   int
	Craft    int
	Resolve  int
}

type UnitStats struct {
	MaxHP               int
	AttackPower         int
	Defense             int
	AttackIntervalTicks int
	AttackRangeTiles    int
}

type UnitArchetypeDefinition struct {
	ID          UnitArchetypeID
	DisplayName string
	Faction     UnitFaction
	Role        UnitRole
	Attributes  UnitAttributes
	Stats       UnitStats
}

type GuardProfile struct {
	AttackPower         int
	RangeTiles          int
	AttackIntervalTicks int
}

var authorityUnitArchetypes = map[UnitArchetypeID]UnitArchetypeDefinition{
	UnitArchetypeSectDisciple: {
		ID:          UnitArchetypeSectDisciple,
		DisplayName: "玄",
		Faction:     UnitFactionSect,
		Role:        UnitRoleWorkerGuard,
		Attributes: UnitAttributes{
			Physique: 3,
			Agility:  3,
			Mind:     2,
			Spirit:   2,
			Craft:    3,
			Resolve:  3,
		},
		Stats: UnitStats{
			MaxHP:               25,
			AttackPower:         4,
			Defense:             3,
			AttackIntervalTicks: 1,
			AttackRangeTiles:    1,
		},
	},
	UnitArchetypeBanditScout: {
		ID:          UnitArchetypeBanditScout,
		DisplayName: "寇",
		Faction:     UnitFactionHostile,
		Role:        UnitRoleRaider,
		Attributes: UnitAttributes{
			Physique: 3,
			Agility:  4,
			Mind:     1,
			Spirit:   1,
			Craft:    1,
			Resolve:  2,
		},
		Stats: UnitStats{
			MaxHP:               23,
			AttackPower:         4,
			Defense:             2,
			AttackIntervalTicks: 1,
			AttackRangeTiles:    1,
		},
	},
}

var authorityBuildingDefense = map[BuildingType]int{
	BuildingMainHall:         2,
	BuildingDiscipleQuarters: 1,
	BuildingWarehouse:        1,
	BuildingHerbGarden:       1,
	BuildingGuardTower:       1,
}

var authorityGuardProfiles = map[BuildingType]GuardProfile{
	BuildingGuardTower: {
		AttackPower:         4,
		RangeTiles:          2,
		AttackIntervalTicks: 1,
	},
}

const hostileSpawnOffsetCols = 4

func mustUnitArchetype(id UnitArchetypeID) UnitArchetypeDefinition {
	archetype, ok := authorityUnitArchetypes[id]
	if !ok {
		panic("missing unit archetype")
	}
	return archetype
}

func getMitigatedDamage(rawDamage, defense int) int {
	damage := rawDamage - ((defense*45 + 50) / 100)
	if damage < 1 {
		return 1
	}
	return damage
}

func getBuildingStructureDefense(building *buildingEntity) int {
	base := authorityBuildingDefense[building.Type]
	if building.Level <= 1 {
		return base
	}
	return base + (building.Level - 1)
}

func getBuildingGuardProfile(building *buildingEntity) *GuardProfile {
	profile, ok := authorityGuardProfiles[building.Type]
	if !ok {
		return nil
	}
	scaled := profile
	if building.Level > 1 {
		scaled.AttackPower += (building.Level - 1) * 2
		if scaled.AttackIntervalTicks > 1 {
			scaled.AttackIntervalTicks--
		}
	}
	return &scaled
}
