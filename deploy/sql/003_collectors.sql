-- 采集器（Agent）注册表
CREATE TABLE IF NOT EXISTS collectors (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id       INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id),
    hostname        TEXT NOT NULL,
    ip              TEXT NOT NULL DEFAULT '',
    os              TEXT NOT NULL DEFAULT '',        -- linux/windows
    arch            TEXT NOT NULL DEFAULT '',        -- amd64/arm64
    agent_version   TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'offline' CHECK (status IN ('online', 'offline', 'error')),
    last_heartbeat  DATETIME,
    config_hash     TEXT NOT NULL DEFAULT '',        -- 配置版本 hash
    labels          TEXT NOT NULL DEFAULT '{}',      -- JSON: {"env":"prod","region":"cn-east"}
    created_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(tenant_id, hostname)
);

-- 采集器配置模板
CREATE TABLE IF NOT EXISTS collector_configs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id   INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    config      TEXT NOT NULL DEFAULT '{}',  -- JSON 配置内容
    is_default  INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);
