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
	alertRepo *repo.AlertRepo
}

func NewAlertService(alertRepo *repo.AlertRepo) *AlertService {
	return &AlertService{alertRepo: alertRepo}
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
