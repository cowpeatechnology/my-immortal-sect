# Server Workspace

This directory is the reserved home for the real Go server runtime of `My Immortal Sect`.

Current status:
- `Go + Hollywood` 最小 authority runtime 已初始化
- `cmd/gameserver/` 可启动本地 `M1-D` authority HTTP 入口
- 当前 authority slice 仅覆盖 `M1-C` 宗门地图中的建造状态、资源结算与短会话关键快照

Expected ownership:
- `cmd/gameserver/`: runtime entrypoint
- `internal/slggame/`: business code only
- `internal/proto/`: project protobuf definitions
- `configs/`: server-facing generated or loaded config data
- `migrations/`: database and config migrations
- `tools/`: server-local developer tooling such as config helpers and sim linting
- `tests/`: integration or package-level acceptance tests that do not belong inside one package

Hard constraints:
- do not modify Hollywood upstream for business logic
- keep simulation authority on the server
- keep protobuf definitions isolated from Hollywood internals

## Current Preview Runtime

Current local authority entry:

- command: `go run ./cmd/gameserver`
- default address: `127.0.0.1:8787`

Current HTTP surface:

- `POST /v1/authority/m1/session/bootstrap`
- `GET /v1/authority/m1/session/snapshot?sessionId=<id>`
- `POST /v1/authority/m1/session/command`

The current actor-backed session is intentionally narrow:

- authoritative:
  - `place_building`
  - `request_upgrade`
  - `toggle_demolition`
  - `collect_stockpile`
  - `deliver_build_resource`
  - `start_building_work`
  - `complete_building_work`
  - `complete_demolition`
  - `complete_repair`
  - `sync_session_progress`
- out of scope for this loop:
  - full pathfinding authority
  - hostile AI authority
  - platform-container networking
  - protobuf and persistence rollout
