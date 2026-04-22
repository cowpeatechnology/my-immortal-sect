package authority

import (
	"fmt"
	"math"

	"google.golang.org/protobuf/encoding/protowire"
)

const maxProtoInt = int(^uint(0) >> 1)

func encodePersistedSessionStateProto(state persistedSessionState) ([]byte, error) {
	var out []byte
	out = appendProtoStringField(out, 1, state.SessionID)
	out = appendProtoVarintField(out, 2, uint64(state.GameTick))
	out = appendProtoVarintField(out, 3, uint64(state.NextBuildingSeq))
	out = appendProtoBytesField(out, 4, encodeStockpileProto(state.Stockpile))
	for _, node := range state.ResourceNodes {
		out = appendProtoBytesField(out, 5, encodePersistedResourceNodeProto(node))
	}
	for _, building := range state.Buildings {
		out = appendProtoBytesField(out, 6, encodePersistedBuildingProto(building))
	}
	out = appendProtoVarintField(out, 16, uint64(state.DiscipleHP))
	out = appendProtoVarintField(out, 17, uint64(state.DiscipleAttackCooldown))
	if state.Hostile != nil {
		out = appendProtoBytesField(out, 18, encodePersistedHostileProto(*state.Hostile))
	}
	out = appendProtoStringField(out, 7, string(state.Phase))
	out = appendProtoStringField(out, 8, string(state.Outcome))
	out = appendProtoStringField(out, 9, state.Objective)
	if state.GuardTowerID != nil {
		out = appendProtoStringField(out, 10, *state.GuardTowerID)
	}
	if state.RuinBuildingID != nil {
		out = appendProtoStringField(out, 11, *state.RuinBuildingID)
	}
	if state.FirstRaidTriggered {
		out = appendProtoVarintField(out, 12, protowire.EncodeBool(state.FirstRaidTriggered))
	}
	if state.FirstRaidResolved {
		out = appendProtoVarintField(out, 13, protowire.EncodeBool(state.FirstRaidResolved))
	}
	out = appendProtoVarintField(out, 14, uint64(state.RaidCountdownSeconds))
	out = appendProtoVarintField(out, 15, uint64(state.DefendRemainingSeconds))
	return out, nil
}

func decodePersistedSessionStateProto(data []byte) (persistedSessionState, error) {
	var state persistedSessionState
	for len(data) > 0 {
		num, typ, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return persistedSessionState{}, protowire.ParseError(tagLen)
		}
		fieldData := data[tagLen:]
		var fieldLen int
		switch num {
		case 1:
			var err error
			state.SessionID, fieldLen, err = consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
		case 2:
			gameTick, n, err := consumeProtoInt64Field(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.GameTick = gameTick
			fieldLen = n
		case 3:
			nextBuildingSeq, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.NextBuildingSeq = nextBuildingSeq
			fieldLen = n
		case 4:
			stockpileBlob, n, err := consumeProtoBytesField(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.Stockpile, err = decodeStockpileProto(stockpileBlob)
			if err != nil {
				return persistedSessionState{}, err
			}
			fieldLen = n
		case 5:
			nodeBlob, n, err := consumeProtoBytesField(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			node, err := decodePersistedResourceNodeProto(nodeBlob)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.ResourceNodes = append(state.ResourceNodes, node)
			fieldLen = n
		case 6:
			buildingBlob, n, err := consumeProtoBytesField(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			building, err := decodePersistedBuildingProto(buildingBlob)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.Buildings = append(state.Buildings, building)
			fieldLen = n
		case 16:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.DiscipleHP = value
			fieldLen = n
		case 17:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.DiscipleAttackCooldown = value
			fieldLen = n
		case 18:
			hostileBlob, n, err := consumeProtoBytesField(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			hostile, err := decodePersistedHostileProto(hostileBlob)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.Hostile = &hostile
			fieldLen = n
		case 7:
			phase, n, err := consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.Phase = SessionPhase(phase)
			fieldLen = n
		case 8:
			outcome, n, err := consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.Outcome = SessionOutcome(outcome)
			fieldLen = n
		case 9:
			objective, n, err := consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.Objective = objective
			fieldLen = n
		case 10:
			guardTowerID, n, err := consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.GuardTowerID = stringPtr(guardTowerID)
			fieldLen = n
		case 11:
			ruinBuildingID, n, err := consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.RuinBuildingID = stringPtr(ruinBuildingID)
			fieldLen = n
		case 12:
			firstRaidTriggered, n, err := consumeProtoBoolField(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.FirstRaidTriggered = firstRaidTriggered
			fieldLen = n
		case 13:
			firstRaidResolved, n, err := consumeProtoBoolField(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.FirstRaidResolved = firstRaidResolved
			fieldLen = n
		case 14:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.RaidCountdownSeconds = value
			fieldLen = n
		case 15:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return persistedSessionState{}, err
			}
			state.DefendRemainingSeconds = value
			fieldLen = n
		default:
			fieldLen = protowire.ConsumeFieldValue(num, typ, fieldData)
			if fieldLen < 0 {
				return persistedSessionState{}, protowire.ParseError(fieldLen)
			}
		}
		data = data[tagLen+fieldLen:]
	}
	return state, nil
}

func encodePersistedHostileProto(hostile persistedHostileEntity) []byte {
	var out []byte
	out = appendProtoStringField(out, 1, hostile.ID)
	out = appendProtoStringField(out, 2, string(hostile.ArchetypeID))
	out = appendProtoStringField(out, 3, hostile.Name)
	out = appendProtoBytesField(out, 4, encodeTileCoordProto(hostile.Tile))
	out = appendProtoVarintField(out, 5, uint64(hostile.HP))
	out = appendProtoStringField(out, 6, string(hostile.VisualState))
	if hostile.TargetBuildingID != nil {
		out = appendProtoStringField(out, 7, *hostile.TargetBuildingID)
	}
	out = appendProtoVarintField(out, 8, uint64(hostile.AttackCooldown))
	if hostile.Active {
		out = appendProtoVarintField(out, 9, protowire.EncodeBool(hostile.Active))
	}
	return out
}

func decodePersistedHostileProto(data []byte) (persistedHostileEntity, error) {
	var hostile persistedHostileEntity
	for len(data) > 0 {
		num, typ, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return persistedHostileEntity{}, protowire.ParseError(tagLen)
		}
		fieldData := data[tagLen:]
		var fieldLen int
		switch num {
		case 1:
			value, n, err := consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedHostileEntity{}, err
			}
			hostile.ID = value
			fieldLen = n
		case 2:
			value, n, err := consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedHostileEntity{}, err
			}
			hostile.ArchetypeID = UnitArchetypeID(value)
			fieldLen = n
		case 3:
			value, n, err := consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedHostileEntity{}, err
			}
			hostile.Name = value
			fieldLen = n
		case 4:
			value, n, err := consumeProtoBytesField(num, typ, fieldData)
			if err != nil {
				return persistedHostileEntity{}, err
			}
			tile, err := decodeTileCoordProto(value)
			if err != nil {
				return persistedHostileEntity{}, err
			}
			hostile.Tile = tile
			fieldLen = n
		case 5:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return persistedHostileEntity{}, err
			}
			hostile.HP = value
			fieldLen = n
		case 6:
			value, n, err := consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedHostileEntity{}, err
			}
			hostile.VisualState = UnitVisualState(value)
			fieldLen = n
		case 7:
			value, n, err := consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedHostileEntity{}, err
			}
			hostile.TargetBuildingID = stringPtr(value)
			fieldLen = n
		case 8:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return persistedHostileEntity{}, err
			}
			hostile.AttackCooldown = value
			fieldLen = n
		case 9:
			value, n, err := consumeProtoBoolField(num, typ, fieldData)
			if err != nil {
				return persistedHostileEntity{}, err
			}
			hostile.Active = value
			fieldLen = n
		default:
			fieldLen = protowire.ConsumeFieldValue(num, typ, fieldData)
			if fieldLen < 0 {
				return persistedHostileEntity{}, protowire.ParseError(fieldLen)
			}
		}
		data = data[tagLen+fieldLen:]
	}
	return hostile, nil
}

func encodePersistedResourceNodeProto(node persistedResourceNodeEntity) []byte {
	var out []byte
	out = appendProtoStringField(out, 1, string(node.Kind))
	out = appendProtoBytesField(out, 2, encodeTileCoordProto(node.Tile))
	out = appendProtoStringField(out, 3, string(node.State))
	out = appendProtoVarintField(out, 4, uint64(node.RemainingCharges))
	out = appendProtoVarintField(out, 5, uint64(node.MaxCharges))
	out = appendProtoVarintField(out, 6, uint64(node.RegenSeconds))
	out = appendProtoVarintField(out, 7, uint64(node.RegenTimerSeconds))
	return out
}

func decodePersistedResourceNodeProto(data []byte) (persistedResourceNodeEntity, error) {
	var node persistedResourceNodeEntity
	for len(data) > 0 {
		num, typ, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return persistedResourceNodeEntity{}, protowire.ParseError(tagLen)
		}
		fieldData := data[tagLen:]
		var fieldLen int
		switch num {
		case 1:
			kind, n, err := consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedResourceNodeEntity{}, err
			}
			node.Kind = ResourceKind(kind)
			fieldLen = n
		case 2:
			tileBlob, n, err := consumeProtoBytesField(num, typ, fieldData)
			if err != nil {
				return persistedResourceNodeEntity{}, err
			}
			node.Tile, err = decodeTileCoordProto(tileBlob)
			if err != nil {
				return persistedResourceNodeEntity{}, err
			}
			fieldLen = n
		case 3:
			state, n, err := consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedResourceNodeEntity{}, err
			}
			node.State = ResourceNodeState(state)
			fieldLen = n
		case 4:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return persistedResourceNodeEntity{}, err
			}
			node.RemainingCharges = value
			fieldLen = n
		case 5:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return persistedResourceNodeEntity{}, err
			}
			node.MaxCharges = value
			fieldLen = n
		case 6:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return persistedResourceNodeEntity{}, err
			}
			node.RegenSeconds = value
			fieldLen = n
		case 7:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return persistedResourceNodeEntity{}, err
			}
			node.RegenTimerSeconds = value
			fieldLen = n
		default:
			fieldLen = protowire.ConsumeFieldValue(num, typ, fieldData)
			if fieldLen < 0 {
				return persistedResourceNodeEntity{}, protowire.ParseError(fieldLen)
			}
		}
		data = data[tagLen+fieldLen:]
	}
	return node, nil
}

func encodePersistedBuildingProto(building persistedBuildingEntity) []byte {
	var out []byte
	out = appendProtoStringField(out, 1, building.ID)
	out = appendProtoStringField(out, 2, string(building.Type))
	out = appendProtoBytesField(out, 3, encodeTileCoordProto(building.Origin))
	out = appendProtoStringField(out, 4, string(building.State))
	out = appendProtoVarintField(out, 5, uint64(building.Level))
	out = appendProtoVarintField(out, 6, uint64(building.HP))
	if building.MarkedForDemolition {
		out = appendProtoVarintField(out, 7, protowire.EncodeBool(building.MarkedForDemolition))
	}
	if building.PendingAction != nil {
		out = appendProtoStringField(out, 8, string(*building.PendingAction))
	}
	if building.PendingLevel != nil {
		out = appendProtoVarintField(out, 9, uint64(*building.PendingLevel))
	}
	out = appendProtoBytesField(out, 10, encodeStockpileProto(building.Supplied))
	if building.WorkProgressTicks > 0 {
		out = appendProtoVarintField(out, 11, uint64(building.WorkProgressTicks))
	}
	if building.AttackCooldown > 0 {
		out = appendProtoVarintField(out, 12, uint64(building.AttackCooldown))
	}
	return out
}

func decodePersistedBuildingProto(data []byte) (persistedBuildingEntity, error) {
	var building persistedBuildingEntity
	for len(data) > 0 {
		num, typ, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return persistedBuildingEntity{}, protowire.ParseError(tagLen)
		}
		fieldData := data[tagLen:]
		var fieldLen int
		switch num {
		case 1:
			id, n, err := consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedBuildingEntity{}, err
			}
			building.ID = id
			fieldLen = n
		case 2:
			buildingType, n, err := consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedBuildingEntity{}, err
			}
			building.Type = BuildingType(buildingType)
			fieldLen = n
		case 3:
			originBlob, n, err := consumeProtoBytesField(num, typ, fieldData)
			if err != nil {
				return persistedBuildingEntity{}, err
			}
			building.Origin, err = decodeTileCoordProto(originBlob)
			if err != nil {
				return persistedBuildingEntity{}, err
			}
			fieldLen = n
		case 4:
			state, n, err := consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedBuildingEntity{}, err
			}
			building.State = BuildingState(state)
			fieldLen = n
		case 5:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return persistedBuildingEntity{}, err
			}
			building.Level = value
			fieldLen = n
		case 6:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return persistedBuildingEntity{}, err
			}
			building.HP = value
			fieldLen = n
		case 7:
			value, n, err := consumeProtoBoolField(num, typ, fieldData)
			if err != nil {
				return persistedBuildingEntity{}, err
			}
			building.MarkedForDemolition = value
			fieldLen = n
		case 8:
			pendingAction, n, err := consumeProtoStringField(num, typ, fieldData)
			if err != nil {
				return persistedBuildingEntity{}, err
			}
			action := BuildingWorkKind(pendingAction)
			building.PendingAction = &action
			fieldLen = n
		case 9:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return persistedBuildingEntity{}, err
			}
			building.PendingLevel = &value
			fieldLen = n
		case 10:
			suppliedBlob, n, err := consumeProtoBytesField(num, typ, fieldData)
			if err != nil {
				return persistedBuildingEntity{}, err
			}
			building.Supplied, err = decodeStockpileProto(suppliedBlob)
			if err != nil {
				return persistedBuildingEntity{}, err
			}
			fieldLen = n
		case 11:
			value, n, err := consumeProtoInt64Field(num, typ, fieldData)
			if err != nil {
				return persistedBuildingEntity{}, err
			}
			building.WorkProgressTicks = value
			fieldLen = n
		case 12:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return persistedBuildingEntity{}, err
			}
			building.AttackCooldown = value
			fieldLen = n
		default:
			fieldLen = protowire.ConsumeFieldValue(num, typ, fieldData)
			if fieldLen < 0 {
				return persistedBuildingEntity{}, protowire.ParseError(fieldLen)
			}
		}
		data = data[tagLen+fieldLen:]
	}
	return building, nil
}

func encodeStockpileProto(stockpile Stockpile) []byte {
	var out []byte
	out = appendProtoVarintField(out, 1, uint64(stockpile.SpiritWood))
	out = appendProtoVarintField(out, 2, uint64(stockpile.SpiritStone))
	out = appendProtoVarintField(out, 3, uint64(stockpile.Herb))
	return out
}

func decodeStockpileProto(data []byte) (Stockpile, error) {
	var stockpile Stockpile
	for len(data) > 0 {
		num, typ, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return Stockpile{}, protowire.ParseError(tagLen)
		}
		fieldData := data[tagLen:]
		var fieldLen int
		switch num {
		case 1:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return Stockpile{}, err
			}
			stockpile.SpiritWood = value
			fieldLen = n
		case 2:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return Stockpile{}, err
			}
			stockpile.SpiritStone = value
			fieldLen = n
		case 3:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return Stockpile{}, err
			}
			stockpile.Herb = value
			fieldLen = n
		default:
			fieldLen = protowire.ConsumeFieldValue(num, typ, fieldData)
			if fieldLen < 0 {
				return Stockpile{}, protowire.ParseError(fieldLen)
			}
		}
		data = data[tagLen+fieldLen:]
	}
	return stockpile, nil
}

func encodeTileCoordProto(tile TileCoord) []byte {
	var out []byte
	out = appendProtoVarintField(out, 1, uint64(tile.Col))
	out = appendProtoVarintField(out, 2, uint64(tile.Row))
	return out
}

func decodeTileCoordProto(data []byte) (TileCoord, error) {
	var tile TileCoord
	for len(data) > 0 {
		num, typ, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return TileCoord{}, protowire.ParseError(tagLen)
		}
		fieldData := data[tagLen:]
		var fieldLen int
		switch num {
		case 1:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return TileCoord{}, err
			}
			tile.Col = value
			fieldLen = n
		case 2:
			value, n, err := consumeProtoIntFieldWithLen(num, typ, fieldData)
			if err != nil {
				return TileCoord{}, err
			}
			tile.Row = value
			fieldLen = n
		default:
			fieldLen = protowire.ConsumeFieldValue(num, typ, fieldData)
			if fieldLen < 0 {
				return TileCoord{}, protowire.ParseError(fieldLen)
			}
		}
		data = data[tagLen+fieldLen:]
	}
	return tile, nil
}

func appendProtoStringField(out []byte, num protowire.Number, value string) []byte {
	out = protowire.AppendTag(out, num, protowire.BytesType)
	return protowire.AppendString(out, value)
}

func appendProtoVarintField(out []byte, num protowire.Number, value uint64) []byte {
	out = protowire.AppendTag(out, num, protowire.VarintType)
	return protowire.AppendVarint(out, value)
}

func appendProtoBytesField(out []byte, num protowire.Number, value []byte) []byte {
	out = protowire.AppendTag(out, num, protowire.BytesType)
	return protowire.AppendBytes(out, value)
}

func consumeProtoStringField(num protowire.Number, typ protowire.Type, data []byte) (string, int, error) {
	if typ != protowire.BytesType {
		return "", 0, protoFieldTypeError(num, protowire.BytesType, typ)
	}
	value, n := protowire.ConsumeString(data)
	if n < 0 {
		return "", 0, protowire.ParseError(n)
	}
	return value, n, nil
}

func consumeProtoBytesField(num protowire.Number, typ protowire.Type, data []byte) ([]byte, int, error) {
	if typ != protowire.BytesType {
		return nil, 0, protoFieldTypeError(num, protowire.BytesType, typ)
	}
	value, n := protowire.ConsumeBytes(data)
	if n < 0 {
		return nil, 0, protowire.ParseError(n)
	}
	return value, n, nil
}

func consumeProtoBoolField(num protowire.Number, typ protowire.Type, data []byte) (bool, int, error) {
	if typ != protowire.VarintType {
		return false, 0, protoFieldTypeError(num, protowire.VarintType, typ)
	}
	value, n := protowire.ConsumeVarint(data)
	if n < 0 {
		return false, 0, protowire.ParseError(n)
	}
	return protowire.DecodeBool(value), n, nil
}

func consumeProtoInt64Field(num protowire.Number, typ protowire.Type, data []byte) (int64, int, error) {
	if typ != protowire.VarintType {
		return 0, 0, protoFieldTypeError(num, protowire.VarintType, typ)
	}
	value, n := protowire.ConsumeVarint(data)
	if n < 0 {
		return 0, 0, protowire.ParseError(n)
	}
	if value > math.MaxInt64 {
		return 0, 0, fmt.Errorf("protobuf field %d exceeds int64 range", num)
	}
	return int64(value), n, nil
}

func consumeProtoIntFieldWithLen(num protowire.Number, typ protowire.Type, data []byte) (int, int, error) {
	if typ != protowire.VarintType {
		return 0, 0, protoFieldTypeError(num, protowire.VarintType, typ)
	}
	value, n := protowire.ConsumeVarint(data)
	if n < 0 {
		return 0, 0, protowire.ParseError(n)
	}
	if value > uint64(maxProtoInt) {
		return 0, 0, fmt.Errorf("protobuf field %d exceeds int range", num)
	}
	return int(value), n, nil
}

func protoFieldTypeError(num protowire.Number, want, got protowire.Type) error {
	return fmt.Errorf("protobuf field %d has wire type %d, want %d", num, got, want)
}
