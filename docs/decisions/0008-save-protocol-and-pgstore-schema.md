# ADR 0008: Protobuf Snapshot / Event Log / Command Log 存档协议

**状态**: 已确认 (Accepted)
**日期**: 2026-04-15，2026-04-23 按 GDD v3.1 收敛重写
**相关文档**:
- [docs/vision/gdd_v3_backend_design.md](/Users/mawei/MyWork/SlgGame/docs/vision/gdd_v3_backend_design.md)
- [docs/decisions/0010-offline-deterministic-simulation.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0010-offline-deterministic-simulation.md)
- [docs/decisions/0012-adopt-gdd-v3-authoritative-backend-gdd.md](/Users/mawei/MyWork/SlgGame/docs/decisions/0012-adopt-gdd-v3-authoritative-backend-gdd.md)

## Context

GDD v3.1 已裁决：

- 权威持久化主源是 snapshot，不是纯事件流
- 事件应用流水线成立，但不是 pure event sourcing
- V1 权威协议与权威存档都走 protobuf-first，而不是 JSON/JSONB

因此需要把存档协议和数据库表结构钉死在同一条线上。

## Decision

采用：

- `snapshot blob` 作为权威恢复主源
- `event_log` 作为审计、回放 snapshot 后增量、弟子日记、debug 与恢复辅助
- `command_log` 作为幂等、审计与命令结果追踪
- 所有权威载荷统一为 protobuf blob

明确拒绝：

- `state_json JSONB`
- `payload JSONB`
- JSON authority protocol
- pure event sourcing

JSON 仅允许保留在：

- debug 输出
- GM/运营工具
- 本地日志
- 配置编辑中间态

## Canonical Tables

### `sect_snapshots`

```sql
CREATE TABLE sect_snapshots (
  sect_id                TEXT PRIMARY KEY,
  owner_player_id        TEXT NOT NULL,
  schema_version         INT NOT NULL,
  simulation_version     INT NOT NULL,
  config_version         INT NOT NULL,
  state_version          BIGINT NOT NULL,
  game_tick              BIGINT NOT NULL,
  last_simulated_wall_ms BIGINT NOT NULL,
  state_blob             BYTEA NOT NULL,
  state_size_bytes       INT NOT NULL,
  created_at             TIMESTAMPTZ NOT NULL,
  updated_at             TIMESTAMPTZ NOT NULL
);
```

### `sect_events`

```sql
CREATE TABLE sect_events (
  sect_id        TEXT NOT NULL,
  state_version  BIGINT NOT NULL,
  event_seq      INT NOT NULL,
  event_type     INT NOT NULL,
  cmd_id         TEXT,
  game_tick      BIGINT NOT NULL,
  event_blob     BYTEA NOT NULL,
  event_size_bytes INT NOT NULL,
  acknowledged   BOOLEAN NOT NULL DEFAULT FALSE,
  created_at     TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (sect_id, state_version, event_seq)
);
```

### `command_log`

```sql
CREATE TABLE command_log (
  sect_id              TEXT NOT NULL,
  cmd_id               TEXT NOT NULL,
  player_id            TEXT NOT NULL,
  command_type         INT NOT NULL,
  command_blob         BYTEA NOT NULL,
  command_size_bytes   INT NOT NULL,
  result_blob          BYTEA,
  result_size_bytes    INT,
  result_status        INT NOT NULL,
  base_version         BIGINT NOT NULL,
  state_version_after  BIGINT,
  created_at           TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (sect_id, cmd_id)
);
```

### Query Index Tables

这些表是从 `state_blob` / `event_blob` 派生出来的 bounded projections，只承担查询和排障用途，不是权威真相。

```sql
CREATE TABLE sect_runtime_index (
  sect_id                TEXT PRIMARY KEY REFERENCES sect_snapshots(sect_id) ON DELETE CASCADE,
  owner_player_id        TEXT NOT NULL,
  sect_name              TEXT NOT NULL,
  state_version          BIGINT NOT NULL,
  game_tick              BIGINT NOT NULL,
  active_storylet_count  INT NOT NULL,
  disciple_count         INT NOT NULL,
  building_count         INT NOT NULL,
  updated_at             TIMESTAMPTZ NOT NULL
);

CREATE TABLE sect_building_index (
  sect_id          TEXT NOT NULL REFERENCES sect_snapshots(sect_id) ON DELETE CASCADE,
  building_id      TEXT NOT NULL,
  building_type    INT NOT NULL,
  building_state   INT NOT NULL,
  level            INT NOT NULL,
  assigned_task_id TEXT,
  updated_at       TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (sect_id, building_id)
);

CREATE TABLE sect_disciple_index (
  sect_id             TEXT NOT NULL REFERENCES sect_snapshots(sect_id) ON DELETE CASCADE,
  disciple_id         TEXT NOT NULL,
  status              INT NOT NULL,
  current_task_id     TEXT,
  current_building_id TEXT,
  updated_at          TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (sect_id, disciple_id)
);

CREATE TABLE sect_task_index (
  sect_id              TEXT NOT NULL REFERENCES sect_snapshots(sect_id) ON DELETE CASCADE,
  task_id              TEXT NOT NULL,
  task_type            INT NOT NULL,
  task_status          INT NOT NULL,
  assigned_disciple_id TEXT,
  target_building_id   TEXT,
  updated_at           TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (sect_id, task_id)
);
```

### `sect_versions`

用于记录 `schema_version / simulation_version / config_version` 迁移历史。

### `sect_deleted`

用于软删除与恢复，不做权威运行时主读表。

## Operational Rules

1. `snapshot blob` 是快速恢复与权威加载主源。
2. 加载时先读最新 snapshot，再回放 bounded event / offline catch-up。
3. 每个命令都必须带 `cmd_id`，重复提交返回已有结果。
4. `event_log` 可以辅助恢复，但不是唯一真相。
5. `sect_runtime_index / sect_building_index / sect_disciple_index / sect_task_index` 都是可重建 projection，不得作为权威恢复主源。
6. 不再同时维护 protobuf 与 JSON 两套权威持久化格式。
