-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id       INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id),
    username        TEXT NOT NULL,
    password_hash   TEXT NOT NULL,  -- SHA-256 hash
    display_name    TEXT NOT NULL DEFAULT '',
    email           TEXT NOT NULL DEFAULT '',
    role            TEXT NOT NULL DEFAULT 'viewer' CHECK (role IN ('admin', 'operator', 'viewer')),
    status          TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'locked')),
    last_login_at   DATETIME,
    created_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(tenant_id, username)
);

-- Casbin RBAC 策略表
CREATE TABLE IF NOT EXISTS casbin_rule (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    ptype TEXT NOT NULL DEFAULT '',
    v0    TEXT NOT NULL DEFAULT '',
    v1    TEXT NOT NULL DEFAULT '',
    v2    TEXT NOT NULL DEFAULT '',
    v3    TEXT NOT NULL DEFAULT '',
    v4    TEXT NOT NULL DEFAULT '',
    v5    TEXT NOT NULL DEFAULT '',
    UNIQUE(ptype, v0, v1, v2, v3, v4, v5)
);

-- 默认 admin 用户（密码: admin123，SHA-256 hash，首次登录后强制修改）
INSERT OR IGNORE INTO users (id, tenant_id, username, password_hash, display_name, role)
VALUES (1, 1, 'admin', '240be518fabd2724ddb6f04eeb1da5967448d7e831c08c8fa822809f74c720a9', '系统管理员', 'admin');

-- 默认 RBAC 策略
INSERT OR IGNORE INTO casbin_rule (ptype, v0, v1, v2) VALUES ('p', 'admin', '/*', '(GET|POST|PUT|DELETE)');
INSERT OR IGNORE INTO casbin_rule (ptype, v0, v1, v2) VALUES ('p', 'operator', '/api/v1/alerts/*', '(GET|POST|PUT)');
INSERT OR IGNORE INTO casbin_rule (ptype, v0, v1, v2) VALUES ('p', 'operator', '/api/v1/metrics/*', 'GET');
INSERT OR IGNORE INTO casbin_rule (ptype, v0, v1, v2) VALUES ('p', 'operator', '/api/v1/logs/*', 'GET');
INSERT OR IGNORE INTO casbin_rule (ptype, v0, v1, v2) VALUES ('p', 'operator', '/api/v1/jobs/*', '(GET|POST)');
INSERT OR IGNORE INTO casbin_rule (ptype, v0, v1, v2) VALUES ('p', 'viewer', '/api/v1/metrics/*', 'GET');
INSERT OR IGNORE INTO casbin_rule (ptype, v0, v1, v2) VALUES ('p', 'viewer', '/api/v1/logs/*', 'GET');
INSERT OR IGNORE INTO casbin_rule (ptype, v0, v1, v2) VALUES ('p', 'viewer', '/api/v1/alerts', 'GET');
