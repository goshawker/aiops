package model

import "time"

// Job represents a scheduled or one-time job.
type Job struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	JobType     string    `json:"job_type" db:"job_type"`       // "shell", "http", "script"
	Content     string    `json:"content" db:"content"`         // command/URL/script
	Schedule    string    `json:"schedule" db:"schedule"`       // cron expression or "once"
	Enabled     bool      `json:"enabled" db:"enabled"`
	Status      string    `json:"status" db:"status"`           // "idle", "running", "success", "failed"
	Timeout     int       `json:"timeout" db:"timeout"`         // seconds
	RetryCount  int       `json:"retry_count" db:"retry_count"`
	LastRunAt   *time.Time `json:"last_run_at,omitempty" db:"last_run_at"`
	NextRunAt   *time.Time `json:"next_run_at,omitempty" db:"next_run_at"`
	TenantID    int64     `json:"tenant_id" db:"tenant_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// JobExecution represents a single execution of a job.
type JobExecution struct {
	ID        int64     `json:"id" db:"id"`
	JobID     int64     `json:"job_id" db:"job_id"`
	Status    string    `json:"status" db:"status"` // "running", "success", "failed", "timeout"
	Output    string    `json:"output" db:"output"`
	Error     string    `json:"error" db:"error"`
	Duration  int       `json:"duration" db:"duration"` // milliseconds
	StartedAt time.Time `json:"started_at" db:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty" db:"ended_at"`
}
