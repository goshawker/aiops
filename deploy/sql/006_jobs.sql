-- 作业定义表
CREATE TABLE IF NOT EXISTS jobs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id   INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    -- 作业类型: shell | python | ansible
    job_type    TEXT NOT NULL DEFAULT 'shell' CHECK (job_type IN ('shell', 'python', 'ansible')),
    -- 脚本内容
    script      TEXT NOT NULL DEFAULT '',
    -- 参数定义 (JSON): [{"name":"host","type":"string","required":true}]
    params      TEXT NOT NULL DEFAULT '[]',
    -- 超时（秒）
    timeout     INTEGER NOT NULL DEFAULT 300,
    -- 审批: 是否需要审批后执行
    need_approval INTEGER NOT NULL DEFAULT 0,
    enabled     INTEGER NOT NULL DEFAULT 1,
    created_by  INTEGER REFERENCES users(id),
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- 作业执行记录表
CREATE TABLE IF NOT EXISTS job_executions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id   INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id),
    job_id      INTEGER NOT NULL REFERENCES jobs(id),
    -- 执行参数 (JSON)
    params      TEXT NOT NULL DEFAULT '{}',
    -- 执行目标
    target_host TEXT NOT NULL DEFAULT '',
    -- 状态
    status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'success', 'failed', 'timeout', 'cancelled')),
    exit_code   INTEGER,
    output      TEXT NOT NULL DEFAULT '',      -- 执行输出（截断到 1MB）
    error       TEXT NOT NULL DEFAULT '',
    -- 时间
    started_at  DATETIME,
    finished_at DATETIME,
    created_by  INTEGER REFERENCES users(id),
    created_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_job_executions_job_id ON job_executions(job_id);
CREATE INDEX idx_job_executions_status ON job_executions(status);
