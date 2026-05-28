-- 审计日志表
CREATE TABLE IF NOT EXISTS audit_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id   INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id),
    user_id     INTEGER REFERENCES users(id),
    username    TEXT NOT NULL DEFAULT '',
    -- 操作
    action      TEXT NOT NULL,           -- create/update/delete/login/logout/execute
    resource    TEXT NOT NULL,           -- user/alert_rule/job/collector/config
    resource_id TEXT NOT NULL DEFAULT '',
    -- 详情
    detail      TEXT NOT NULL DEFAULT '',  -- JSON: 变更前后对比
    ip          TEXT NOT NULL DEFAULT '',
    user_agent  TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_audit_logs_tenant ON audit_logs(tenant_id, created_at);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id, created_at);
CREATE INDEX idx_audit_logs_action ON audit_logs(action, resource);
