package repo

import (
	"testing"

	"aiops/internal/model"
)

func TestAlertRuleRepo_CreateRule(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAlertRuleRepo(db)

	rule := &model.AlertRule{
		TenantID:   1,
		Name:       "CPU High",
		RuleType:   "threshold",
		RuleConfig: `{"metric":"cpu_usage","threshold":80}`,
		Severity:   "warning",
		Enabled:    true,
	}

	if err := repo.CreateRule(rule); err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	if rule.ID == 0 {
		t.Fatal("CreateRule should set ID")
	}
}

func TestAlertRuleRepo_GetRule(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAlertRuleRepo(db)

	rule := &model.AlertRule{TenantID: 1, Name: "Mem High", RuleType: "threshold", Severity: "critical", Enabled: true}
	repo.CreateRule(rule)

	found, err := repo.GetRule(rule.ID)
	if err != nil {
		t.Fatalf("GetRule: %v", err)
	}
	if found == nil {
		t.Fatal("rule not found")
	}
	if found.Name != "Mem High" {
		t.Errorf("name = %s, want Mem High", found.Name)
	}
	if found.Severity != "critical" {
		t.Errorf("severity = %s, want critical", found.Severity)
	}
}

func TestAlertRuleRepo_GetRule_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAlertRuleRepo(db)

	found, err := repo.GetRule(999)
	if err != nil {
		t.Fatalf("GetRule: %v", err)
	}
	if found != nil {
		t.Fatal("should return nil for nonexistent rule")
	}
}

func TestAlertRuleRepo_ListRules(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAlertRuleRepo(db)

	for i := 0; i < 5; i++ {
		repo.CreateRule(&model.AlertRule{TenantID: 1, Name: "rule", RuleType: "threshold", Severity: "warning", Enabled: true})
	}

	rules, total, err := repo.ListRules(1, 3, 0)
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(rules) != 3 {
		t.Errorf("len = %d, want 3", len(rules))
	}
}

func TestAlertRuleRepo_UpdateRule(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAlertRuleRepo(db)

	rule := &model.AlertRule{TenantID: 1, Name: "old", RuleType: "threshold", Severity: "warning", Enabled: true}
	repo.CreateRule(rule)

	rule.Name = "new"
	rule.Severity = "critical"
	rule.Enabled = false
	if err := repo.UpdateRule(rule); err != nil {
		t.Fatalf("UpdateRule: %v", err)
	}

	found, _ := repo.GetRule(rule.ID)
	if found.Name != "new" {
		t.Errorf("name = %s, want new", found.Name)
	}
	if found.Severity != "critical" {
		t.Errorf("severity = %s, want critical", found.Severity)
	}
	if found.Enabled {
		t.Error("should be disabled")
	}
}

func TestAlertRuleRepo_DeleteRule(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAlertRuleRepo(db)

	rule := &model.AlertRule{TenantID: 1, Name: "del", RuleType: "threshold", Severity: "warning", Enabled: true}
	repo.CreateRule(rule)

	if err := repo.DeleteRule(rule.ID); err != nil {
		t.Fatalf("DeleteRule: %v", err)
	}

	found, _ := repo.GetRule(rule.ID)
	if found != nil {
		t.Fatal("rule should be deleted")
	}
}

func TestAlertRuleRepo_GetEnabledRules(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAlertRuleRepo(db)

	repo.CreateRule(&model.AlertRule{TenantID: 1, Name: "enabled", RuleType: "threshold", Severity: "warning", Enabled: true})
	repo.CreateRule(&model.AlertRule{TenantID: 1, Name: "disabled", RuleType: "threshold", Severity: "warning", Enabled: false})
	repo.CreateRule(&model.AlertRule{TenantID: 1, Name: "enabled2", RuleType: "threshold", Severity: "warning", Enabled: true})

	rules, err := repo.GetEnabledRules()
	if err != nil {
		t.Fatalf("GetEnabledRules: %v", err)
	}
	if len(rules) != 2 {
		t.Errorf("len = %d, want 2", len(rules))
	}
}

func TestParseRuleConfig(t *testing.T) {
	cfg := ParseRuleConfig(`{"metric":"cpu","threshold":80}`)
	if cfg["metric"] != "cpu" {
		t.Errorf("metric = %v, want cpu", cfg["metric"])
	}
	if cfg["threshold"] != float64(80) {
		t.Errorf("threshold = %v, want 80", cfg["threshold"])
	}
}

func TestParseRuleConfig_Invalid(t *testing.T) {
	cfg := ParseRuleConfig(`{invalid json`)
	if cfg == nil {
		t.Fatal("should return empty map, not nil")
	}
	if len(cfg) != 0 {
		t.Errorf("len = %d, want 0", len(cfg))
	}
}

func TestParseRuleConfig_Empty(t *testing.T) {
	cfg := ParseRuleConfig("")
	if cfg == nil {
		t.Fatal("should return empty map, not nil")
	}
}
