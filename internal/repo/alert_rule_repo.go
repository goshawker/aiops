package repo

import (
	"database/sql"
	"encoding/json"
	"time"

	"aiops/internal/model"
)

// AlertRuleRepo handles alert rule persistence.
type AlertRuleRepo struct {
	db DBExecutor
}

func NewAlertRuleRepo(db DBExecutor) *AlertRuleRepo {
	return &AlertRuleRepo{db: db}
}

func (r *AlertRuleRepo) CreateRule(rule *model.AlertRule) error {
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	result, err := r.db.Exec(
		`INSERT INTO alert_rules (tenant_id, name, description, rule_type, rule_config, severity, enabled, notify_config, silence_config, created_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.TenantID, rule.Name, rule.Description, rule.RuleType, rule.RuleConfig,
		rule.Severity, boolToInt(rule.Enabled), rule.NotifyConfig, rule.SilenceConfig,
		rule.CreatedBy, rule.CreatedAt, rule.UpdatedAt,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	rule.ID = id
	return nil
}

func (r *AlertRuleRepo) GetRule(id int64) (*model.AlertRule, error) {
	rule := &model.AlertRule{}
	var enabledInt int
	err := r.db.QueryRow(
		`SELECT id, tenant_id, name, description, rule_type, rule_config, severity, enabled, notify_config, silence_config, created_by, created_at, updated_at
		 FROM alert_rules WHERE id = ?`, id,
	).Scan(&rule.ID, &rule.TenantID, &rule.Name, &rule.Description, &rule.RuleType,
		&rule.RuleConfig, &rule.Severity, &enabledInt, &rule.NotifyConfig,
		&rule.SilenceConfig, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	rule.Enabled = enabledInt != 0
	return rule, err
}

func (r *AlertRuleRepo) ListRules(tenantID int64, limit, offset int) ([]model.AlertRule, int, error) {
	var total int
	r.db.QueryRow(`SELECT COUNT(*) FROM alert_rules WHERE tenant_id = ?`, tenantID).Scan(&total)

	rows, err := r.db.Query(
		`SELECT id, tenant_id, name, description, rule_type, rule_config, severity, enabled, notify_config, silence_config, created_by, created_at, updated_at
		 FROM alert_rules WHERE tenant_id = ? ORDER BY id DESC LIMIT ? OFFSET ?`, tenantID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var rules []model.AlertRule
	for rows.Next() {
		var rule model.AlertRule
		var enabledInt int
		if err := rows.Scan(&rule.ID, &rule.TenantID, &rule.Name, &rule.Description,
			&rule.RuleType, &rule.RuleConfig, &rule.Severity, &enabledInt,
			&rule.NotifyConfig, &rule.SilenceConfig, &rule.CreatedBy,
			&rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, 0, err
		}
		rule.Enabled = enabledInt != 0
		rules = append(rules, rule)
	}
	return rules, total, nil
}

func (r *AlertRuleRepo) UpdateRule(rule *model.AlertRule) error {
	rule.UpdatedAt = time.Now()
	_, err := r.db.Exec(
		`UPDATE alert_rules SET name=?, description=?, rule_type=?, rule_config=?, severity=?, enabled=?, notify_config=?, silence_config=?, updated_at=? WHERE id=?`,
		rule.Name, rule.Description, rule.RuleType, rule.RuleConfig,
		rule.Severity, boolToInt(rule.Enabled), rule.NotifyConfig,
		rule.SilenceConfig, rule.UpdatedAt, rule.ID,
	)
	return err
}

func (r *AlertRuleRepo) DeleteRule(id int64) error {
	_, err := r.db.Exec(`DELETE FROM alert_rules WHERE id=?`, id)
	return err
}

func (r *AlertRuleRepo) GetEnabledRules() ([]model.AlertRule, error) {
	rows, err := r.db.Query(
		`SELECT id, tenant_id, name, description, rule_type, rule_config, severity, enabled, notify_config, silence_config, created_by, created_at, updated_at
		 FROM alert_rules WHERE enabled = 1 ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []model.AlertRule
	for rows.Next() {
		var rule model.AlertRule
		var enabledInt int
		if err := rows.Scan(&rule.ID, &rule.TenantID, &rule.Name, &rule.Description,
			&rule.RuleType, &rule.RuleConfig, &rule.Severity, &enabledInt,
			&rule.NotifyConfig, &rule.SilenceConfig, &rule.CreatedBy,
			&rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rule.Enabled = enabledInt != 0
		rules = append(rules, rule)
	}
	return rules, nil
}

// RuleConfig parses the JSON rule_config field into a structured map.
func ParseRuleConfig(ruleConfigJSON string) map[string]interface{} {
	var cfg map[string]interface{}
	json.Unmarshal([]byte(ruleConfigJSON), &cfg)
	return cfg
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
