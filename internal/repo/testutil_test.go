package repo

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database with the full schema.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:?_journal_mode=WAL")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	// Create tables
	schema := `
	CREATE TABLE IF NOT EXISTS tenants (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'active',
		plan TEXT NOT NULL DEFAULT 'free',
		max_hosts INTEGER NOT NULL DEFAULT 10,
		max_users INTEGER NOT NULL DEFAULT 5,
		settings TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME DEFAULT (datetime('now')),
		updated_at DATETIME DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_id INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id),
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		display_name TEXT NOT NULL DEFAULT '',
		email TEXT NOT NULL DEFAULT '',
		role TEXT NOT NULL DEFAULT 'viewer',
		status TEXT NOT NULL DEFAULT 'active',
		last_login_at DATETIME,
		created_at DATETIME DEFAULT (datetime('now')),
		updated_at DATETIME DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS alert_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_id INTEGER NOT NULL DEFAULT 1,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		rule_type TEXT NOT NULL DEFAULT 'threshold',
		rule_config TEXT NOT NULL DEFAULT '{}',
		severity TEXT NOT NULL DEFAULT 'warning',
		enabled INTEGER NOT NULL DEFAULT 1,
		notify_config TEXT NOT NULL DEFAULT '{}',
		silence_config TEXT NOT NULL DEFAULT '{}',
		created_by INTEGER REFERENCES users(id),
		created_at DATETIME DEFAULT (datetime('now')),
		updated_at DATETIME DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS alert_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_id INTEGER NOT NULL DEFAULT 1,
		rule_id INTEGER,
		rule_name TEXT NOT NULL DEFAULT '',
		source_type TEXT NOT NULL DEFAULT '',
		source TEXT NOT NULL DEFAULT '',
		host TEXT NOT NULL DEFAULT '',
		service TEXT NOT NULL DEFAULT '',
		severity TEXT NOT NULL DEFAULT 'info',
		title TEXT NOT NULL DEFAULT '',
		message TEXT NOT NULL DEFAULT '',
		value TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'firing',
		fired_at DATETIME DEFAULT (datetime('now')),
		resolved_at DATETIME,
		created_at DATETIME DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS incidents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_id INTEGER NOT NULL DEFAULT 1,
		title TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		severity TEXT NOT NULL DEFAULT 'info',
		affected_services TEXT NOT NULL DEFAULT '[]',
		affected_hosts TEXT NOT NULL DEFAULT '[]',
		event_count INTEGER NOT NULL DEFAULT 0,
		ai_summary TEXT NOT NULL DEFAULT '',
		root_cause TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'open',
		created_at DATETIME DEFAULT (datetime('now')),
		updated_at DATETIME DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS collectors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_id INTEGER NOT NULL DEFAULT 1,
		name TEXT NOT NULL DEFAULT '',
		hostname TEXT NOT NULL DEFAULT '',
		ip TEXT NOT NULL DEFAULT '',
		os TEXT NOT NULL DEFAULT '',
		arch TEXT NOT NULL DEFAULT '',
		version TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'offline',
		tags TEXT NOT NULL DEFAULT '{}',
		last_heartbeat DATETIME,
		created_at DATETIME DEFAULT (datetime('now')),
		updated_at DATETIME DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS collector_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		collector_id INTEGER NOT NULL,
		config_type TEXT NOT NULL DEFAULT 'scrape',
		content TEXT NOT NULL DEFAULT '{}',
		version INTEGER NOT NULL DEFAULT 1,
		applied_at DATETIME,
		created_at DATETIME DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_id INTEGER NOT NULL DEFAULT 1,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		job_type TEXT NOT NULL DEFAULT 'shell',
		content TEXT NOT NULL DEFAULT '',
		schedule TEXT NOT NULL DEFAULT 'once',
		enabled INTEGER NOT NULL DEFAULT 1,
		status TEXT NOT NULL DEFAULT 'idle',
		timeout INTEGER NOT NULL DEFAULT 300,
		retry_count INTEGER NOT NULL DEFAULT 0,
		next_run_at DATETIME,
		last_run_at DATETIME,
		created_at DATETIME DEFAULT (datetime('now')),
		updated_at DATETIME DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS job_executions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'running',
		output TEXT NOT NULL DEFAULT '',
		error TEXT NOT NULL DEFAULT '',
		duration INTEGER NOT NULL DEFAULT 0,
		started_at DATETIME DEFAULT (datetime('now')),
		ended_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_id INTEGER NOT NULL DEFAULT 1,
		user_id INTEGER,
		username TEXT NOT NULL DEFAULT '',
		action TEXT NOT NULL,
		resource TEXT NOT NULL,
		resource_id TEXT NOT NULL DEFAULT '',
		detail TEXT NOT NULL DEFAULT '',
		ip TEXT NOT NULL DEFAULT '',
		prev_hash TEXT NOT NULL DEFAULT '',
		record_hash TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT (datetime('now'))
	);

	INSERT OR IGNORE INTO tenants (id, code, name) VALUES (1, 'default', '默认租户');
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}
