-- 作业定义表
CREATE TABLE IF NOT EXISTS jobs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id   INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    -- 作业类型: shell | http
    job_type    TEXT NOT NULL DEFAULT 'shell' CHECK (job_type IN ('shell', 'http')),
    -- 命令/URL 内容
    content     TEXT NOT NULL DEFAULT '',
    -- 调度: cron 表达式或 "once"
    schedule    TEXT NOT NULL DEFAULT 'once',
    enabled     INTEGER NOT NULL DEFAULT 1,
    status      TEXT NOT NULL DEFAULT 'idle' CHECK (status IN ('idle', 'running', 'success', 'failed')),
    timeout     INTEGER NOT NULL DEFAULT 300,
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_run_at DATETIME,
    next_run_at DATETIME,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- 作业执行记录表
CREATE TABLE IF NOT EXISTS job_executions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id      INTEGER NOT NULL REFERENCES jobs(id),
    status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'success', 'failed', 'timeout')),
    output      TEXT NOT NULL DEFAULT '',
    error       TEXT NOT NULL DEFAULT '',
    duration    INTEGER NOT NULL DEFAULT 0,   -- milliseconds
    started_at  DATETIME,
    ended_at    DATETIME
);

CREATE INDEX idx_job_executions_job_id ON job_executions(job_id);
CREATE INDEX idx_job_executions_status ON job_executions(status);