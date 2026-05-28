-- 系统配置表
CREATE TABLE IF NOT EXISTS system_config (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id   INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id),
    category    TEXT NOT NULL,           -- general/notification/retention/ai
    key         TEXT NOT NULL,
    value       TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    updated_by  INTEGER REFERENCES users(id),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(tenant_id, category, key)
);

-- 默认配置
INSERT OR IGNORE INTO system_config (tenant_id, category, key, value, description) VALUES
    (1, 'general', 'platform_name', 'AIOps', '平台名称'),
    (1, 'general', 'data_retention_days', '30', '指标数据保留天数'),
    (1, 'general', 'log_retention_days', '30', '日志数据保留天数'),
    (1, 'notification', 'dingtalk_webhook', '', '钉钉 Webhook URL'),
    (1, 'notification', 'email_smtp_host', '', 'SMTP 服务器'),
    (1, 'notification', 'email_smtp_port', '587', 'SMTP 端口'),
    (1, 'notification', 'email_from', '', '发件人邮箱'),
    (1, 'notification', 'email_password', '', '邮箱密码'),
    (1, 'ai', 'llm_provider', 'rule_engine', 'LLM 提供商: rule_engine/qwen2-1.5b/qwen2-7b'),
    (1, 'ai', 'anomaly_sensitivity', 'medium', '异常检测灵敏度: low/medium/high'),
    (1, 'ai', 'summary_enabled', 'true', '是否启用 AI 健康摘要');
