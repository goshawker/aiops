package model

import "time"

// Collector represents a registered data collector agent.
type Collector struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Hostname    string    `json:"hostname" db:"hostname"`
	IP          string    `json:"ip" db:"ip"`
	Version     string    `json:"version" db:"version"`
	Status      string    `json:"status" db:"status"` // "online", "offline", "error"
	LastHeartbeat *time.Time `json:"last_heartbeat,omitempty" db:"last_heartbeat"`
	Tags        string    `json:"tags" db:"tags"` // JSON: {"env":"prod","region":"cn-east"}
	TenantID    int64     `json:"tenant_id" db:"tenant_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CollectorConfig represents a configuration pushed to a collector.
type CollectorConfig struct {
	ID          int64     `json:"id" db:"id"`
	CollectorID int64     `json:"collector_id" db:"collector_id"`
	ConfigType  string    `json:"config_type" db:"config_type"` // "scrape", "log", "custom"
	Content     string    `json:"content" db:"content"`         // YAML content
	Version     int       `json:"version" db:"version"`
	AppliedAt   *time.Time `json:"applied_at,omitempty" db:"applied_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// CollectorHeartbeat tracks collector liveness.
type CollectorHeartbeat struct {
	CollectorID int64     `json:"collector_id" db:"collector_id"`
	CPU         float64   `json:"cpu" db:"cpu"`
	Memory      float64   `json:"memory" db:"memory"`
	Uptime      int64     `json:"uptime" db:"uptime"` // seconds
	Collected   int64     `json:"collected" db:"collected"` // metrics collected
	Errors      int64     `json:"errors" db:"errors"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
}
