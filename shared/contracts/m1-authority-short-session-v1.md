# M1 Authority Short Session Contract V1

**状态**: active  
**最后更新**: 2026-04-23  
**适用范围**: `M1` authority-only 宗门地图短会话

## Purpose

This contract freezes the authority-only runtime boundary for the current `M1` sect-map short session:

- authority owns all gameplay state and progression
- client is limited to rendering, player input, bounded local facts, and debug UI
- the transport snapshot is the single source of truth for buildings, disciples, hostiles, resources, and session progression

Pathfinding visualization, token motion interpolation, and local presentation effects may remain client-rendered in this loop, but they are no longer allowed to advance business state on their own.

## Transport

- HTTP JSON over localhost for preview-time validation
- default authority base URL: `http://127.0.0.1:8787`

## Session Endpoints

- `POST /v1/authority/m1/session/bootstrap`
  - binds or validates `playerId` + `playerToken`, restores the latest saved preview session by default, or resets to a clean test session when `mode = "reset"`, then returns the latest snapshot
- `GET /v1/authority/m1/session/snapshot?sessionId=<id>&playerId=<player>&playerToken=<token>`
  - returns the latest snapshot for an existing session
- `POST /v1/authority/m1/session/command`
  - applies one authoritative command for the bound player session and returns the latest snapshot plus command result

## Ownership Boundary

### Authority owns

Authority is the only owner of:

- disciple assignment, work target, carrying state, work progress, and next transition
- stockpile settlement and resource-node depletion / refresh timing
- building placement, supply progress, construction progress, upgrade progress, damage, repair, and demolition completion
- hostile spawn, target selection, movement, HP, attack cadence, and defeat resolution
- session phase transitions, raid trigger / outcome, and short-session closure
- deterministic save / load / restore / reset for the live short-session state

### Client owns

The client is limited to:

- rendering sect-map state, HUD, and presentation effects from the latest authority snapshot
- collecting player input and sending allowed high-level intent commands
- reporting explicitly allowed bounded local facts that authority can verify and reject
- exposing bounded debug surfaces for acceptance and desync diagnosis

### Client must not own

The client must not directly decide or announce:

- which disciple takes which haul / build / repair job
- when a building transitions `planned -> supplied -> constructing -> active`
- when a damaged building finishes repair
- when a hostile loses HP, switches target, lands a hit, or is defeated
- when a session phase closes into `recover`, `second_cycle_ready`, `victory`, or `defeat`
- when a rejected progression command should be retried as a client-side recovery loop

## Canonical IDs

The transport contract below reuses gameplay identifiers and values frozen in:

- `shared/configs/m1/sect_map_short_session.v1.json`

For M1, that shared config is the canonical source for:

- building template keys and bootstrap seeds
- resource kinds, node recharge values, and bootstrap seed tiles
- session phase order plus the timing / readiness values that drive `recover` and `second_cycle_ready`

This contract should describe payload shape and transport fields, not reintroduce a second rules table.

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

### DiscipleAssignmentKind

- `idle`
- `gather`
- `haul`
- `build`
- `repair`
- `guard`
- `demolish`

### SessionPhase

- `clear_ruin`
- `place_guard_tower`
- `upgrade_guard_tower`
- `raid_countdown`
- `defend`
- `recover`
- `second_cycle_ready`
- `victory`
- `defeat`

### SessionOutcome

- `in_progress`
- `victory`
- `defeat`

## Session Response Shape

Every session endpoint returns an envelope:

```json
{
  "identity": {
    "playerId": "preview-player",
    "playerToken": "preview-token",
    "playerSessionId": "preview-player/preview-local"
  },
  "snapshot": {},
  "result": null
}
```

`identity.playerSessionId` is the canonical server-owned session key for this bound preview player. The client may persist and reuse it, but it must continue sending the matching `playerId` and `playerToken` alongside requests.

## Snapshot Shape

```json
{
  "sessionId": "preview-player/preview-local",
  "gameTick": 0,
  "stockpile": {
    "spirit_wood": 0,
    "spirit_stone": 0,
    "herb": 0
  },
  "resourceNodes": [
    {
      "tile": { "col": 2, "row": 4 },
      "kind": "spirit_wood",
      "state": "available",
      "remainingCharges": 3,
      "maxCharges": 3,
      "regenSeconds": 9,
      "regenTimerSeconds": 0
    }
  ],
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
  "disciples": [
    {
      "archetypeId": "sect_disciple",
      "id": "disciple-01",
      "name": "玄",
      "assignmentKind": "haul",
      "targetBuildingId": "building-3",
      "targetResourceKind": "spirit_wood",
      "targetTile": { "col": 8, "row": 8 },
      "carrying": {
        "kind": "spirit_wood",
        "amount": 1
      },
      "hp": 25,
      "maxHp": 25,
      "visualState": "carrying",
      "workProgressTicks": 0,
      "expectedNextTransition": "deliver_build_resource"
    }
  ],
  "hostiles": [
    {
      "id": "hostile-01",
      "archetypeId": "bandit_scout",
      "name": "寇",
      "tile": { "col": 9, "row": 6 },
      "hp": 19,
      "maxHp": 23,
      "visualState": "attacking",
      "active": true,
      "targetBuildingId": "building-3"
    }
  ],
  "session": {
    "phase": "clear_ruin",
    "outcome": "in_progress",
    "objective": "长按废弃仓房并拆除，为护山台腾出位置",
    "guardTowerId": null,
    "ruinBuildingId": "building-2",
    "firstRaidTriggered": false,
    "firstRaidResolved": false,
    "raidCountdownSeconds": 24,
    "defendRemainingSeconds": 0,
    "recoverReason": "none",
    "damagedBuildingCount": 0,
    "regeneratingNodeCount": 0
  }
}
```

`disciples` and `hostiles` are part of the authoritative gameplay snapshot. The client may animate or interpolate from this data, but it must not synthesize a different assignment, combat target, HP delta, or progress state locally.

`session.raidCountdownSeconds` and `session.defendRemainingSeconds` are the authority-owned combat timing fields for preview presentation. The client may animate countdown or defend feedback from them, but it must not locally trigger the raid, resolve the raid, or announce combat closure from its own timers.

`session.recoverReason`, `session.damagedBuildingCount`, and `session.regeneratingNodeCount` are the authority-owned explanation fields for post-raid continuity. During `recover`, the client must use them instead of guessing from local timers or client-owned readiness heuristics. When all counts reach zero, authority is free to move the slice into `second_cycle_ready` on its own clock.

## Bootstrap Envelope

```json
{
  "sessionId": "preview-local",
  "playerId": "preview-player",
  "playerToken": "preview-token",
  "mode": "restore_latest"
}
```

Bootstrap modes:

- `restore_latest`
  - default preview bootstrap path
  - imports the latest saved authority session when one exists, otherwise seeds a clean preview session
- `reset`
  - discards the current preview session and seeds a fresh clean test session for the same `sessionId`

## Command Envelope

```json
{
  "sessionId": "preview-local",
  "playerId": "preview-player",
  "playerToken": "preview-token",
  "command": {
    "name": "place_building",
    "payload": {}
  }
}
```

## Supported Commands

Commands are split into two categories: player intent and bounded facts. Any rejected command leaves authority state unchanged; the client must clear stale local presentation state and resync from the returned or refreshed snapshot.

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
- lets authority scheduling decide later supply / construction assignment and completion

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
- lets authority scheduling decide later supply / construction assignment and completion

### `toggle_demolition`

```json
{
  "buildingId": "building-3"
}
```

Effect:

- toggles demolition intent on a non-`main_hall` building
- does not let the client announce demolition completion

## Allowed Bounded Fact Command

### `collect_stockpile`

This is a bounded fact command, not a client-owned progression command.

- allowed only while the local gather/dropoff presentation loop still exists in preview
- authority must validate the reported `resourceTile`, `resourceKind`, and collected amount
- on rejection, the client must clear the stale gather presentation state and resync from authority
- acceptance updates only the stockpile / node facts; it does not grant the client ownership over downstream build, repair, or phase progression

### `collect_stockpile`

```json
{
  "resourceKind": "spirit_wood",
  "amount": 1,
  "resourceTile": { "col": 2, "row": 4 }
}
```

Effect:

- validates the authoritative resource node at `resourceTile`
- decrements authoritative node charges and starts regen timing when depleted
- increases stockpile after a local gather/dropoff loop completes

## No Longer Allowed As Client-Owned Progression

The following preview-era commands are removed from the accepted client-owned progression surface. If an implementation still emits them, that implementation is outside this contract and must be treated as legacy / transitional behavior rather than the frozen `M1` boundary.

- `deliver_build_resource`
- `start_building_work`
- `complete_building_work`
- `complete_demolition`
- `complete_repair`
- `report_building_damage`
- `trigger_first_raid`
- `resolve_first_raid`
- `expire_session`

These transitions must now be settled inside authority runtime state from assignments, validated facts, authoritative timers, combat results, and snapshot progression.

## Command Result Shape

```json
{
  "accepted": true,
  "event": "build.placed",
  "message": "已下达护山台蓝图，等待弟子备料和施工"
}
```

## Notes

- This contract supersedes the earlier hybrid preview boundary for accepted `M1` work.
- Bounded fact commands are an exception surface, not a second progression pipeline.
- Client debug surfaces may display derived labels or warnings, but they must not mutate authority gameplay state.
