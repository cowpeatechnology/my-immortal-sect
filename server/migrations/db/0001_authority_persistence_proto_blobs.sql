-- Authoritative persistence schema.
-- Protobuf blobs are the source of truth.
-- Query index tables are rebuildable projections only.

CREATE TABLE sect_snapshots (
    sect_id TEXT PRIMARY KEY,
    owner_player_id TEXT NOT NULL,
    schema_version INTEGER NOT NULL,
    simulation_version INTEGER NOT NULL,
    config_version INTEGER NOT NULL,
    state_version BIGINT NOT NULL,
    game_tick BIGINT NOT NULL,
    last_simulated_wall_ms BIGINT NOT NULL,
    state_blob BYTEA NOT NULL,
    state_size_bytes INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sect_events (
    sect_id TEXT NOT NULL,
    state_version BIGINT NOT NULL,
    event_seq INTEGER NOT NULL,
    event_type INTEGER NOT NULL,
    cmd_id TEXT,
    game_tick BIGINT NOT NULL,
    event_blob BYTEA NOT NULL,
    event_size_bytes INTEGER NOT NULL,
    acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (sect_id, state_version, event_seq)
);

CREATE INDEX idx_sect_events_sect_version
ON sect_events (sect_id, state_version);

CREATE INDEX idx_sect_events_cmd_id
ON sect_events (sect_id, cmd_id)
WHERE cmd_id IS NOT NULL;

CREATE TABLE command_log (
    sect_id TEXT NOT NULL,
    cmd_id TEXT NOT NULL,
    player_id TEXT NOT NULL,
    command_type INTEGER NOT NULL,
    command_blob BYTEA NOT NULL,
    command_size_bytes INTEGER NOT NULL,
    result_blob BYTEA,
    result_size_bytes INTEGER,
    result_status INTEGER NOT NULL,
    base_version BIGINT NOT NULL,
    state_version_after BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    PRIMARY KEY (sect_id, cmd_id)
);

CREATE INDEX idx_command_log_player_created
ON command_log (player_id, created_at DESC);

CREATE TABLE sect_runtime_index (
    sect_id TEXT PRIMARY KEY REFERENCES sect_snapshots(sect_id) ON DELETE CASCADE,
    owner_player_id TEXT NOT NULL,
    sect_name TEXT NOT NULL,
    state_version BIGINT NOT NULL,
    game_tick BIGINT NOT NULL,
    active_storylet_count INTEGER NOT NULL,
    disciple_count INTEGER NOT NULL,
    building_count INTEGER NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sect_runtime_index_owner_player
ON sect_runtime_index (owner_player_id);

CREATE TABLE sect_building_index (
    sect_id TEXT NOT NULL REFERENCES sect_snapshots(sect_id) ON DELETE CASCADE,
    building_id TEXT NOT NULL,
    building_type INTEGER NOT NULL,
    building_state INTEGER NOT NULL,
    level INTEGER NOT NULL,
    assigned_task_id TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (sect_id, building_id)
);

CREATE INDEX idx_sect_building_index_state
ON sect_building_index (sect_id, building_state);

CREATE TABLE sect_disciple_index (
    sect_id TEXT NOT NULL REFERENCES sect_snapshots(sect_id) ON DELETE CASCADE,
    disciple_id TEXT NOT NULL,
    status INTEGER NOT NULL,
    current_task_id TEXT,
    current_building_id TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (sect_id, disciple_id)
);

CREATE INDEX idx_sect_disciple_index_status
ON sect_disciple_index (sect_id, status);

CREATE TABLE sect_task_index (
    sect_id TEXT NOT NULL REFERENCES sect_snapshots(sect_id) ON DELETE CASCADE,
    task_id TEXT NOT NULL,
    task_type INTEGER NOT NULL,
    task_status INTEGER NOT NULL,
    assigned_disciple_id TEXT,
    target_building_id TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (sect_id, task_id)
);

CREATE INDEX idx_sect_task_index_status
ON sect_task_index (sect_id, task_status);
