-- 告警规则表
CREATE TABLE IF NOT EXISTS alert_rules (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id   INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    -- 规则类型: threshold(阈值) | anomaly(AI异常检测) | log_pattern(日志模式)
    rule_type   TEXT NOT NULL DEFAULT 'threshold' CHECK (rule_type IN ('threshold', 'anomaly', 'log_pattern')),
    -- 规则内容 (JSON)
    -- threshold: {"metric":"cpu_usage","op":">","value":80,"duration":"5m"}
    -- anomaly:   {"metric":"cpu_usage","sensitivity":"medium"}
    -- log_pattern: {"service":"order","level":"ERROR","pattern":"timeout","count":10,"duration":"5m"}
    rule_config TEXT NOT NULL DEFAULT '{}',
    severity    TEXT NOT NULL DEFAULT 'warning' CHECK (severity IN ('critical', 'warning', 'info')),
    enabled     INTEGER NOT NULL DEFAULT 1,
    -- 通知渠道 (JSON): {"dingtalk":true,"email":["a@b.com"],"webhook":"http://..."}
    notify_config TEXT NOT NULL DEFAULT '{}',
    -- 静默规则 (JSON): {"start":"22:00","end":"06:00","days":["sat","sun"]}
    silence_config TEXT NOT NULL DEFAULT '{}',
    created_by  INTEGER REFERENCES users(id),
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);
