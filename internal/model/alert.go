package model

import "time"

// AlertRule represents a monitoring rule
type AlertRule struct {
	ID            int64     `json:"id" gorm:"primaryKey"`
	TenantID      int64     `json:"tenant_id" gorm:"default:1"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	RuleType      string    `json:"rule_type"`   // threshold, anomaly, log_pattern
	RuleConfig    string    `json:"rule_config"`  // JSON
	Severity      string    `json:"severity"`     // critical, warning, info
	Enabled       bool      `json:"enabled" gorm:"default:true"`
	NotifyConfig  string    `json:"notify_config"`  // JSON
	SilenceConfig string    `json:"silence_config"` // JSON
	CreatedBy     int64     `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AlertEvent represents a single alert event
type AlertEvent struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	TenantID    int64     `json:"tenant_id" gorm:"default:1"`
	RuleID      int64     `json:"rule_id"`
	RuleName    string    `json:"rule_name"`
	SourceType  string    `json:"source_type"` // metric, log, agent
	Source      string    `json:"source"`
	Host        string    `json:"host"`
	Service     string    `json:"service"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	Value       string    `json:"value"`
	Threshold   string    `json:"threshold"`
	Labels      string    `json:"labels"` // JSON
	Status      string    `json:"status"` // firing, resolved, suppressed
	IncidentID  int64     `json:"incident_id"`
	FiredAt     time.Time `json:"fired_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// Incident represents an aggregated alert incident
type Incident struct {
	ID               int64      `json:"id" gorm:"primaryKey"`
	TenantID         int64      `json:"tenant_id" gorm:"default:1"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	Severity         string     `json:"severity"`
	AffectedServices string     `json:"affected_services"` // JSON array
	AffectedHosts    string     `json:"affected_hosts"`    // JSON array
	EventCount       int        `json:"event_count"`
	AISummary        string     `json:"ai_summary"`
	RootCause        string     `json:"root_cause"`
	Status           string     `json:"status"` // open, acknowledged, resolved, closed
	AssignedTo       *int64     `json:"assigned_to"`
	AcknowledgedAt   *time.Time `json:"acknowledged_at"`
	ResolvedAt       *time.Time `json:"resolved_at"`
	ClosedAt         *time.Time `json:"closed_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// AlertEventInput is the input from Kafka
type AlertEventInput struct {
	SourceType string            `json:"source_type"`
	Source     string            `json:"source"`
	Host       string            `json:"host"`
	Service    string            `json:"service"`
	Severity   string            `json:"severity"`
	Title      string            `json:"title"`
	Message    string            `json:"message"`
	Value      string            `json:"value"`
	Threshold  string            `json:"threshold"`
	Labels     map[string]string `json:"labels"`
}

// IncidentInput is the incident from Kafka
type IncidentInput struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Severity         string   `json:"severity"`
	AffectedServices []string `json:"affected_services"`
	AffectedHosts    []string `json:"affected_hosts"`
	EventCount       int      `json:"event_count"`
	Status           string   `json:"status"`
}
