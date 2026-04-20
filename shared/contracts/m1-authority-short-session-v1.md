# M1 Authority Short Session Contract V1

**状态**: active  
**最后更新**: 2026-04-21  
**适用范围**: `M1-D` 最小 authority-backed 宗门地图短会话

## Purpose

This contract freezes the minimum client/server boundary for the `M1-D` bridge:

- authoritative stockpile settlement
- authoritative building placement / upgrade / demolition state
- authoritative short-session snapshot for build-state and key progression fields

It is intentionally narrow. Pathfinding, token motion, local combat presentation, and resource-node regeneration remain client-driven in this loop.

## Transport

- HTTP JSON over localhost for preview-time validation
- default authority base URL: `http://127.0.0.1:8787`

## Session Endpoints

- `POST /v1/authority/m1/session/bootstrap`
  - creates or resets the local preview session and returns the latest snapshot
- `GET /v1/authority/m1/session/snapshot?sessionId=<id>`
  - returns the latest snapshot for an existing session
- `POST /v1/authority/m1/session/command`
  - applies one authoritative command and returns the latest snapshot plus command result

## Canonical IDs

### ResourceKind

- `spirit_wood`
- `spirit_stone`
- `herb`

### BuildingType

- `main_hall`
- `disciple_quarters`
- `warehouse`
- `herb_garden`
- `guard_tower`

### BuildingState

- `planned`
- `supplied`
- `constructing`
- `active`
- `damaged`

### BuildingWorkKind

- `build`
- `upgrade`

### SessionPhase

- `clear_ruin`
- `place_guard_tower`
- `upgrade_guard_tower`
- `raid_countdown`
- `defend`
- `recover`
- `victory`
- `defeat`

### SessionOutcome

- `in_progress`
- `victory`
- `defeat`

## Snapshot Shape

```json
{
  "sessionId": "preview-local",
  "stockpile": {
    "spirit_wood": 0,
    "spirit_stone": 0,
    "herb": 0
  },
  "buildings": [
    {
      "id": "building-1",
      "type": "main_hall",
      "origin": { "col": 6, "row": 5 },
      "state": "active",
      "level": 1,
      "hp": 18,
      "maxHp": 18,
      "markedForDemolition": false,
      "pendingAction": null,
      "pendingLevel": null,
      "supplied": {
        "spirit_wood": 0,
        "spirit_stone": 0,
        "herb": 0
      }
    }
  ],
  "session": {
    "phase": "clear_ruin",
    "outcome": "in_progress",
    "objective": "长按废弃仓房并拆除，为护山台腾出位置",
    "guardTowerId": null,
    "ruinBuildingId": "building-2",
    "firstRaidTriggered": false,
    "firstRaidResolved": false
  }
}
```

## Command Envelope

```json
{
  "sessionId": "preview-local",
  "command": {
    "name": "place_building",
    "payload": {}
  }
}
```

## Supported Commands

### `place_building`

```json
{
  "buildingType": "guard_tower",
  "origin": { "col": 8, "row": 8 }
}
```

Effect:

- creates a `planned` building
- clears demolition mark
- updates phase if the first guard tower contract is satisfied later

### `request_upgrade`

```json
{
  "buildingId": "building-3"
}
```

Effect:

- converts an `active` building into an upgrade `planned` state
- resets `supplied`
- records `pendingAction = "upgrade"` and `pendingLevel`

### `toggle_demolition`

```json
{
  "buildingId": "building-3"
}
```

Effect:

- toggles demolition intent on a non-`main_hall` building

### `collect_stockpile`

```json
{
  "resourceKind": "spirit_wood",
  "amount": 1
}
```

Effect:

- increases stockpile after a local gather/dropoff loop completes

### `deliver_build_resource`

```json
{
  "buildingId": "building-3",
  "resourceKind": "spirit_stone"
}
```

Effect:

- decrements stockpile
- increments building `supplied`
- transitions `planned -> supplied` when all costs are met

### `start_building_work`

```json
{
  "buildingId": "building-3"
}
```

Effect:

- transitions `supplied -> constructing`

### `complete_building_work`

```json
{
  "buildingId": "building-3"
}
```

Effect:

- completes build or upgrade
- clears pending action
- restores HP to max
- transitions to `active`

### `complete_demolition`

```json
{
  "buildingId": "building-3"
}
```

Effect:

- returns salvage to stockpile
- removes the building from the authoritative snapshot

### `complete_repair`

```json
{
  "buildingId": "building-3"
}
```

Effect:

- consumes repair cost from stockpile
- restores HP to max
- transitions `damaged -> active`

### `sync_session_progress`

```json
{
  "phase": "defend",
  "outcome": "in_progress",
  "objective": "守住首波敌袭，不要让主殿瘫痪",
  "firstRaidTriggered": true,
  "firstRaidResolved": false
}
```

Effect:

- mirrors local short-session progression fields that are still produced by the current client-only combat loop

## Command Result Shape

```json
{
  "accepted": true,
  "event": "build.placed",
  "message": "已下达护山台蓝图，等待弟子备料和施工"
}
```

## Notes

- This contract does not claim that all `M1-C` simulation has moved server-side.
- It only freezes the minimum authority-backed slice required for `M1-D`.
