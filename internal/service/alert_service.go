package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"aiops/internal/model"
	"aiops/internal/repo"
)

// AlertService processes alerts and manages incidents
type AlertService struct {
	alertRepo    *repo.AlertRepo
	alertRuleRepo *repo.AlertRuleRepo
	vmClient     *repo.VMClient
}

func NewAlertService(alertRepo *repo.AlertRepo, alertRuleRepo *repo.AlertRuleRepo, vmClient *repo.VMClient) *AlertService {
	return &AlertService{alertRepo: alertRepo, alertRuleRepo: alertRuleRepo, vmClient: vmClient}
}

// ProcessAlertEvent handles an incoming alert event
func (s *AlertService) ProcessAlertEvent(ctx context.Context, msg map[string]interface{}) error {
	input := parseAlertInput(msg)

	event := &model.AlertEvent{
		TenantID:   1,
		SourceType: input.SourceType,
		Source:     input.Source,
		Host:       input.Host,
		Service:    input.Service,
		Severity:   input.Severity,
		Title:      input.Title,
		Message:    input.Message,
		Value:      input.Value,
		Threshold:  input.Threshold,
		Status:     "firing",
		FiredAt:    time.Now(),
	}

	labelsJSON, _ := json.Marshal(input.Labels)
	event.Labels = string(labelsJSON)

	if err := s.alertRepo.SaveEvent(event); err != nil {
		return fmt.Errorf("save alert event: %w", err)
	}

	log.Printf("alert: saved event %d - %s [%s]", event.ID, event.Title, event.Severity)
	return nil
}

// ProcessIncident handles an incoming incident
func (s *AlertService) ProcessIncident(ctx context.Context, msg map[string]interface{}) error {
	input := parseIncidentInput(msg)

	servicesJSON, _ := json.Marshal(input.AffectedServices)
	hostsJSON, _ := json.Marshal(input.AffectedHosts)

	inc := &model.Incident{
		TenantID:         1,
		Title:            input.Title,
		Description:      input.Description,
		Severity:         input.Severity,
		AffectedServices: string(servicesJSON),
		AffectedHosts:    string(hostsJSON),
		EventCount:       input.EventCount,
		Status:           "open",
	}

	if err := s.alertRepo.SaveIncident(inc); err != nil {
		return fmt.Errorf("save incident: %w", err)
	}

	log.Printf("alert: saved incident %d - %s [%s]", inc.ID, inc.Title, inc.Severity)
	return nil
}

// ListEvents returns alert events
func (s *AlertService) ListEvents(tenantID int64, status string, limit, offset int) ([]model.AlertEvent, int64, error) {
	return s.alertRepo.ListEvents(tenantID, status, limit, offset)
}

// ListIncidents returns incidents
func (s *AlertService) ListIncidents(tenantID int64, status string, limit, offset int) ([]model.Incident, int64, error) {
	return s.alertRepo.ListIncidents(tenantID, status, limit, offset)
}

// AcknowledgeIncident acknowledges an incident
func (s *AlertService) AcknowledgeIncident(id int64) error {
	return s.alertRepo.UpdateIncidentStatus(id, "acknowledged")
}

// ResolveIncident resolves an incident
func (s *AlertService) ResolveIncident(id int64) error {
	return s.alertRepo.UpdateIncidentStatus(id, "resolved")
}

func parseAlertInput(msg map[string]interface{}) model.AlertEventInput {
	return model.AlertEventInput{
		SourceType: getString(msg, "source_type"),
		Source:     getString(msg, "source"),
		Host:       getString(msg, "host"),
		Service:    getString(msg, "service"),
		Severity:   getString(msg, "severity"),
		Title:      getString(msg, "title"),
		Message:    getString(msg, "message"),
		Value:      getString(msg, "value"),
		Threshold:  getString(msg, "threshold"),
	}
}

func parseIncidentInput(msg map[string]interface{}) model.IncidentInput {
	input := model.IncidentInput{
		ID:          getString(msg, "id"),
		Title:       getString(msg, "title"),
		Description: getString(msg, "description"),
		Severity:    getString(msg, "severity"),
		Status:      getString(msg, "status"),
	}

	if v, ok := msg["event_count"].(float64); ok {
		input.EventCount = int(v)
	}

	if v, ok := msg["affected_services"].([]interface{}); ok {
		for _, s := range v {
			if str, ok := s.(string); ok {
				input.AffectedServices = append(input.AffectedServices, str)
			}
		}
	}

	if v, ok := msg["affected_hosts"].([]interface{}); ok {
		for _, h := range v {
			if str, ok := h.(string); ok {
				input.AffectedHosts = append(input.AffectedHosts, str)
			}
		}
	}

	return input
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// --- Alert Rule CRUD ---

func (s *AlertService) CreateRule(rule *model.AlertRule) error {
	return s.alertRuleRepo.CreateRule(rule)
}

func (s *AlertService) GetRule(id int64) (*model.AlertRule, error) {
	return s.alertRuleRepo.GetRule(id)
}

func (s *AlertService) ListRules(tenantID int64, limit, offset int) ([]model.AlertRule, int, error) {
	return s.alertRuleRepo.ListRules(tenantID, limit, offset)
}

func (s *AlertService) UpdateRule(rule *model.AlertRule) error {
	return s.alertRuleRepo.UpdateRule(rule)
}

func (s *AlertService) DeleteRule(id int64) error {
	return s.alertRuleRepo.DeleteRule(id)
}

// --- Rule Evaluation Engine ---

// StartRuleEvaluator runs a periodic loop that evaluates enabled alert rules.
func (s *AlertService) StartRuleEvaluator(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("alert: rule evaluator started (interval=%s)", interval)
	for {
		select {
		case <-ctx.Done():
			log.Println("alert: rule evaluator stopped")
			return
		case <-ticker.C:
			s.evaluateRules(ctx)
		}
	}
}

// evaluateRules fetches all enabled rules, evaluates threshold rules against
// VictoriaMetrics, and fires alert events for triggered rules.
func (s *AlertService) evaluateRules(ctx context.Context) {
	rules, err := s.alertRuleRepo.GetEnabledRules()
	if err != nil {
		log.Printf("alert: failed to fetch enabled rules: %v", err)
		return
	}

	for _, rule := range rules {
		switch rule.RuleType {
		case "threshold":
			s.evaluateThresholdRule(ctx, rule)
		case "anomaly":
			// Anomaly rules are evaluated by the anomaly detection service (Python).
			// The alert engine does not re-evaluate them.
		case "log_pattern":
			// Log pattern rules require ClickHouse queries, not yet implemented.
		}
	}
}

// evaluateThresholdRule evaluates a single threshold rule against VictoriaMetrics.
// Rule config JSON: {"metric":"cpu_usage","op":">","value":80,"duration":"5m"}
func (s *AlertService) evaluateThresholdRule(ctx context.Context, rule model.AlertRule) {
	if s.vmClient == nil {
		return
	}

	cfg := repo.ParseRuleConfig(rule.RuleConfig)
	metric, _ := cfg["metric"].(string)
	op, _ := cfg["op"].(string)
	thresholdValue, _ := cfg["value"].(float64)

	if metric == "" || op == "" || thresholdValue == 0 {
		return
	}

	// Build PromQL: the metric name as-is; user can put a full PromQL expression in metric field.
	promql := metric

	results, err := s.vmClient.QueryInstant(promql, time.Now())
	if err != nil {
		log.Printf("alert: rule %d query failed: %v", rule.ID, err)
		return
	}

	for _, r := range results {
		if len(r.Values) == 0 {
			continue
		}
		val := r.Values[0].Value
		triggered := false

		switch op {
		case ">":
			triggered = val > thresholdValue
		case ">=":
			triggered = val >= thresholdValue
		case "<":
			triggered = val < thresholdValue
		case "<=":
			triggered = val <= thresholdValue
		case "==":
			triggered = val == thresholdValue
		case "!=":
			triggered = val != thresholdValue
		}

		if !triggered {
			continue
		}

		// Build and save alert event
		instance, _ := r.Metric["instance"]
		job, _ := r.Metric["job"]

		event := &model.AlertEvent{
			TenantID:   rule.TenantID,
			RuleID:     rule.ID,
			RuleName:   rule.Name,
			SourceType: "metric",
			Source:     metric,
			Host:       instance,
			Service:    job,
			Severity:   rule.Severity,
			Title:      fmt.Sprintf("%s %s %.2f", rule.Name, op, thresholdValue),
			Message:    fmt.Sprintf("指标 %s 当前值 %.2f %s 阈值 %.2f", metric, val, op, thresholdValue),
			Value:      fmt.Sprintf("%.2f", val),
			Threshold:  fmt.Sprintf("%.2f", thresholdValue),
			Status:     "firing",
			FiredAt:    time.Now(),
		}
		labelsJSON, _ := json.Marshal(r.Metric)
		event.Labels = string(labelsJSON)

		if err := s.alertRepo.SaveEvent(event); err != nil {
			log.Printf("alert: failed to save event for rule %d: %v", rule.ID, err)
		} else {
			log.Printf("alert: rule %d triggered - %s = %.2f %s %.2f", rule.ID, metric, val, op, thresholdValue)
		}
	}
}
