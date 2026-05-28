-- 告警事件表（单条原始告警）
CREATE TABLE IF NOT EXISTS alert_events (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id   INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id),
    rule_id     INTEGER REFERENCES alert_rules(id),
    rule_name   TEXT NOT NULL DEFAULT '',
    -- 事件来源
    source_type TEXT NOT NULL DEFAULT 'metric' CHECK (source_type IN ('metric', 'log', 'agent')),
    source      TEXT NOT NULL DEFAULT '',       -- 指标名/日志服务名
    host        TEXT NOT NULL DEFAULT '',
    service     TEXT NOT NULL DEFAULT '',
    -- 告警内容
    severity    TEXT NOT NULL DEFAULT 'warning' CHECK (severity IN ('critical', 'warning', 'info')),
    title       TEXT NOT NULL DEFAULT '',
    message     TEXT NOT NULL DEFAULT '',
    value       TEXT NOT NULL DEFAULT '',       -- 触发值
    threshold   TEXT NOT NULL DEFAULT '',       -- 阈值
    labels      TEXT NOT NULL DEFAULT '{}',    -- JSON 标签
    -- 状态
    status      TEXT NOT NULL DEFAULT 'firing' CHECK (status IN ('firing', 'resolved', 'suppressed')),
    incident_id INTEGER REFERENCES incidents(id),
    fired_at    DATETIME NOT NULL DEFAULT (datetime('now')),
    resolved_at DATETIME,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_alert_events_tenant_status ON alert_events(tenant_id, status);
CREATE INDEX idx_alert_events_fired_at ON alert_events(fired_at);
CREATE INDEX idx_alert_events_service ON alert_events(service, host);

-- Incident 表（聚合后的告警事件）
CREATE TABLE IF NOT EXISTS incidents (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id   INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id),
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    severity    TEXT NOT NULL DEFAULT 'warning' CHECK (severity IN ('critical', 'warning', 'info')),
    -- 影响范围
    affected_services TEXT NOT NULL DEFAULT '[]',  -- JSON array
    affected_hosts    TEXT NOT NULL DEFAULT '[]',  -- JSON array
    event_count INTEGER NOT NULL DEFAULT 0,
    -- AI 分析
    ai_summary  TEXT NOT NULL DEFAULT '',          -- LLM 生成的健康摘要
    root_cause  TEXT NOT NULL DEFAULT '',          -- RCA 分析结果
    -- 状态
    status      TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'acknowledged', 'resolved', 'closed')),
    assigned_to INTEGER REFERENCES users(id),
    acknowledged_at DATETIME,
    resolved_at DATETIME,
    closed_at   DATETIME,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_incidents_tenant_status ON incidents(tenant_id, status);
CREATE INDEX idx_incidents_severity ON incidents(severity);
