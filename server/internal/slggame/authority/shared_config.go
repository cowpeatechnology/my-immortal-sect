package authority

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type sharedTileCoord struct {
	Col int `json:"col"`
	Row int `json:"row"`
}

type sharedStockpile struct {
	SpiritWood  int `json:"spirit_wood"`
	SpiritStone int `json:"spirit_stone"`
	Herb        int `json:"herb"`
}

type sharedResourceKindConfig struct {
	ConfigID      string       `json:"config_id"`
	RuntimeKey    ResourceKind `json:"runtime_key"`
	Title         string       `json:"title"`
	VisualAssetID string       `json:"visual_asset_id"`
	NodeRule      struct {
		MaxCharges   int `json:"max_charges"`
		RegenSeconds int `json:"regen_seconds"`
	} `json:"node_rule"`
	SeedTiles []sharedTileCoord `json:"seed_tiles"`
}

type sharedBuildingBootstrapSeed struct {
	Origin sharedTileCoord `json:"origin"`
	Level  int             `json:"level"`
	State  BuildingState   `json:"state"`
	HP     *int            `json:"hp,omitempty"`
	Role   string          `json:"role,omitempty"`
}

type sharedBuildingTypeConfig struct {
	ConfigID      string       `json:"config_id"`
	RuntimeKey    BuildingType `json:"runtime_key"`
	Title         string       `json:"title"`
	VisualAssetID string       `json:"visual_asset_id"`
	Footprint     struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"footprint"`
	BaseCost      sharedStockpile              `json:"base_cost"`
	BaseMaxHP     int                          `json:"base_max_hp"`
	RepairCost    sharedStockpile              `json:"repair_cost"`
	BootstrapSeed *sharedBuildingBootstrapSeed `json:"bootstrap_seed,omitempty"`
}

type sharedM1ShortSessionConfig struct {
	ResourceKinds []sharedResourceKindConfig `json:"resource_kinds"`
	BuildingTypes []sharedBuildingTypeConfig `json:"building_types"`
	Session       struct {
		TargetSeconds        int `json:"target_seconds"`
		FirstRaidPrepSeconds int `json:"first_raid_prep_seconds"`
	} `json:"session"`
}

type authoritySharedConfig struct {
	resourceNodeRules    map[ResourceKind]resourceNodeRule
	resourceLabels       map[ResourceKind]string
	initialResourceSeeds []resourceNodeSeed
	buildingDefinitions  map[BuildingType]buildingDefinition
	buildingLabels       map[BuildingType]string
	initialMainHallTile  TileCoord
	initialRuinTile      TileCoord
	targetSeconds        int
	firstRaidPrepSeconds int
}

var sharedAuthorityConfig = mustLoadAuthoritySharedConfig()

func mustLoadAuthoritySharedConfig() authoritySharedConfig {
	sourcePath := filepath.Join(repoRootFromCurrentFile(), "shared", "configs", "m1", "sect_map_short_session.v1.json")
	payload, err := os.ReadFile(sourcePath)
	if err != nil {
		panic(fmt.Sprintf("read shared M1 config: %v", err))
	}

	var parsed sharedM1ShortSessionConfig
	if err := json.Unmarshal(payload, &parsed); err != nil {
		panic(fmt.Sprintf("decode shared M1 config: %v", err))
	}

	config := authoritySharedConfig{
		resourceNodeRules:    make(map[ResourceKind]resourceNodeRule, len(parsed.ResourceKinds)),
		resourceLabels:       make(map[ResourceKind]string, len(parsed.ResourceKinds)),
		initialResourceSeeds: make([]resourceNodeSeed, 0),
		buildingDefinitions:  make(map[BuildingType]buildingDefinition, len(parsed.BuildingTypes)),
		buildingLabels:       make(map[BuildingType]string, len(parsed.BuildingTypes)),
		targetSeconds:        parsed.Session.TargetSeconds,
		firstRaidPrepSeconds: parsed.Session.FirstRaidPrepSeconds,
	}

	for _, resource := range parsed.ResourceKinds {
		config.resourceNodeRules[resource.RuntimeKey] = resourceNodeRule{
			MaxCharges:   resource.NodeRule.MaxCharges,
			RegenSeconds: resource.NodeRule.RegenSeconds,
		}
		config.resourceLabels[resource.RuntimeKey] = resource.Title
		for _, seed := range resource.SeedTiles {
			config.initialResourceSeeds = append(config.initialResourceSeeds, resourceNodeSeed{
				Kind: resource.RuntimeKey,
				Tile: TileCoord{Col: seed.Col, Row: seed.Row},
			})
		}
	}

	for _, building := range parsed.BuildingTypes {
		config.buildingDefinitions[building.RuntimeKey] = buildingDefinition{
			ID:         building.RuntimeKey,
			Width:      building.Footprint.Width,
			Height:     building.Footprint.Height,
			BaseCost:   toStockpile(building.BaseCost),
			BaseMaxHP:  building.BaseMaxHP,
			RepairCost: toStockpile(building.RepairCost),
		}
		config.buildingLabels[building.RuntimeKey] = building.Title

		if building.BootstrapSeed == nil {
			continue
		}

		switch building.BootstrapSeed.Role {
		case "main_hall":
			config.initialMainHallTile = TileCoord{
				Col: building.BootstrapSeed.Origin.Col,
				Row: building.BootstrapSeed.Origin.Row,
			}
		case "initial_ruin":
			config.initialRuinTile = TileCoord{
				Col: building.BootstrapSeed.Origin.Col,
				Row: building.BootstrapSeed.Origin.Row,
			}
		}
	}

	return config
}

func repoRootFromCurrentFile() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve authority shared config path: missing caller")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", ".."))
}

func toStockpile(source sharedStockpile) Stockpile {
	return Stockpile{
		SpiritWood:  source.SpiritWood,
		SpiritStone: source.SpiritStone,
		Herb:        source.Herb,
	}
}
