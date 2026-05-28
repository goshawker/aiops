package repo

import (
	"encoding/json"
	"fmt"
	"time"

	"aiops/internal/model"
)

// AlertRepo handles alert event persistence
type AlertRepo struct {
	db DBExecutor
}

func NewAlertRepo(db DBExecutor) *AlertRepo {
	return &AlertRepo{db: db}
}

// SaveEvent saves an alert event
func (r *AlertRepo) SaveEvent(e *model.AlertEvent) error {
	labelsJSON, _ := json.Marshal(e.Labels)
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	if e.FiredAt.IsZero() {
		e.FiredAt = time.Now()
	}

	query := `INSERT INTO alert_events
		(tenant_id, rule_id, rule_name, source_type, source, host, service,
		 severity, title, message, value, threshold, labels, status, incident_id, fired_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.Exec(query,
		e.TenantID, e.RuleID, e.RuleName, e.SourceType, e.Source, e.Host, e.Service,
		e.Severity, e.Title, e.Message, e.Value, e.Threshold, string(labelsJSON),
		e.Status, e.IncidentID, e.FiredAt, e.CreatedAt)
	if err != nil {
		return fmt.Errorf("save alert event: %w", err)
	}

	id, _ := result.LastInsertId()
	e.ID = id
	return nil
}

// ListEvents returns alert events with filters
func (r *AlertRepo) ListEvents(tenantID int64, status string, limit, offset int) ([]model.AlertEvent, int64, error) {
	where := "tenant_id = ?"
	args := []interface{}{tenantID}

	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}

	// Count
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM alert_events WHERE %s", where)
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count events: %w", err)
	}

	// Query
	if limit <= 0 {
		limit = 50
	}
	query := fmt.Sprintf(`SELECT id, tenant_id, rule_id, rule_name, source_type, source,
		host, service, severity, title, message, value, threshold, labels, status,
		incident_id, fired_at, resolved_at, created_at
		FROM alert_events WHERE %s ORDER BY fired_at DESC LIMIT ? OFFSET ?`, where)
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var events []model.AlertEvent
	for rows.Next() {
		var e model.AlertEvent
		var labelsStr string
		if err := rows.Scan(&e.ID, &e.TenantID, &e.RuleID, &e.RuleName, &e.SourceType,
			&e.Source, &e.Host, &e.Service, &e.Severity, &e.Title, &e.Message,
			&e.Value, &e.Threshold, &labelsStr, &e.Status, &e.IncidentID,
			&e.FiredAt, &e.ResolvedAt, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan event: %w", err)
		}
		json.Unmarshal([]byte(labelsStr), &e.Labels)
		events = append(events, e)
	}

	return events, total, rows.Err()
}

// SaveIncident saves an incident
func (r *AlertRepo) SaveIncident(inc *model.Incident) error {
	servicesJSON, _ := json.Marshal(inc.AffectedServices)
	hostsJSON, _ := json.Marshal(inc.AffectedHosts)
	if inc.CreatedAt.IsZero() {
		inc.CreatedAt = time.Now()
	}
	inc.UpdatedAt = time.Now()

	query := `INSERT INTO incidents
		(tenant_id, title, description, severity, affected_services, affected_hosts,
		 event_count, ai_summary, root_cause, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.Exec(query,
		inc.TenantID, inc.Title, inc.Description, inc.Severity,
		string(servicesJSON), string(hostsJSON), inc.EventCount,
		inc.AISummary, inc.RootCause, inc.Status, inc.CreatedAt, inc.UpdatedAt)
	if err != nil {
		return fmt.Errorf("save incident: %w", err)
	}

	id, _ := result.LastInsertId()
	inc.ID = id
	return nil
}

// ListIncidents returns incidents with filters
func (r *AlertRepo) ListIncidents(tenantID int64, status string, limit, offset int) ([]model.Incident, int64, error) {
	where := "tenant_id = ?"
	args := []interface{}{tenantID}

	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM incidents WHERE %s", where)
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count incidents: %w", err)
	}

	if limit <= 0 {
		limit = 50
	}
	query := fmt.Sprintf(`SELECT id, tenant_id, title, description, severity,
		affected_services, affected_hosts, event_count, ai_summary, root_cause,
		status, assigned_to, acknowledged_at, resolved_at, closed_at, created_at, updated_at
		FROM incidents WHERE %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, where)
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query incidents: %w", err)
	}
	defer rows.Close()

	var incidents []model.Incident
	for rows.Next() {
		var inc model.Incident
		var servicesStr, hostsStr string
		if err := rows.Scan(&inc.ID, &inc.TenantID, &inc.Title, &inc.Description,
			&inc.Severity, &servicesStr, &hostsStr, &inc.EventCount,
			&inc.AISummary, &inc.RootCause, &inc.Status, &inc.AssignedTo,
			&inc.AcknowledgedAt, &inc.ResolvedAt, &inc.ClosedAt,
			&inc.CreatedAt, &inc.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan incident: %w", err)
		}
		json.Unmarshal([]byte(servicesStr), &inc.AffectedServices)
		json.Unmarshal([]byte(hostsStr), &inc.AffectedHosts)
		incidents = append(incidents, inc)
	}

	return incidents, total, rows.Err()
}

// UpdateIncidentStatus updates incident status
func (r *AlertRepo) UpdateIncidentStatus(id int64, status string) error {
	now := time.Now()
	query := `UPDATE incidents SET status = ?, updated_at = ?`
	args := []interface{}{status, now}

	switch status {
	case "acknowledged":
		query += ", acknowledged_at = ?"
		args = append(args, now)
	case "resolved":
		query += ", resolved_at = ?"
		args = append(args, now)
	case "closed":
		query += ", closed_at = ?"
		args = append(args, now)
	}

	query += " WHERE id = ?"
	args = append(args, id)

	_, err := r.db.Exec(query, args...)
	return err
}
