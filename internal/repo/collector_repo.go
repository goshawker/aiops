package repo

import (
	"database/sql"
	"time"

	"aiops/internal/model"
)

// CollectorRepo handles collector persistence.
type CollectorRepo struct {
	db DBExecutor
}

func NewCollectorRepo(db DBExecutor) *CollectorRepo {
	return &CollectorRepo{db: db}
}

// --- Collectors ---

func (r *CollectorRepo) RegisterCollector(c *model.Collector) error {
	now := time.Now()
	result, err := r.db.Exec(
		`INSERT INTO collectors (name, hostname, ip, version, status, tags, tenant_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.Name, c.Hostname, c.IP, c.Version, "offline", c.Tags, c.TenantID, now, now,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	c.ID = id
	c.Status = "offline"
	c.CreatedAt = now
	c.UpdatedAt = now
	return nil
}

func (r *CollectorRepo) GetCollector(id int64) (*model.Collector, error) {
	c := &model.Collector{}
	err := r.db.QueryRow(
		`SELECT id, name, hostname, ip, version, status, last_heartbeat, tags, tenant_id, created_at, updated_at
		 FROM collectors WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &c.Hostname, &c.IP, &c.Version, &c.Status, &c.LastHeartbeat, &c.Tags, &c.TenantID, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (r *CollectorRepo) ListCollectors(limit, offset int, status string) ([]model.Collector, int, error) {
	where := "1=1"
	args := []interface{}{}
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}

	var total int
	r.db.QueryRow(`SELECT COUNT(*) FROM collectors WHERE `+where, args...).Scan(&total)

	args = append(args, limit, offset)
	rows, err := r.db.Query(
		`SELECT id, name, hostname, ip, version, status, last_heartbeat, tags, tenant_id, created_at, updated_at
		 FROM collectors WHERE `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var collectors []model.Collector
	for rows.Next() {
		var c model.Collector
		if err := rows.Scan(&c.ID, &c.Name, &c.Hostname, &c.IP, &c.Version, &c.Status, &c.LastHeartbeat, &c.Tags, &c.TenantID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		collectors = append(collectors, c)
	}
	return collectors, total, nil
}

func (r *CollectorRepo) UpdateHeartbeat(collectorID int64, hb *model.CollectorHeartbeat) error {
	now := time.Now()
	_, err := r.db.Exec(
		`UPDATE collectors SET status='online', last_heartbeat=?, updated_at=? WHERE id=?`,
		now, now, collectorID,
	)
	return err
}

func (r *CollectorRepo) UpdateStatus(id int64, status string) error {
	now := time.Now()
	_, err := r.db.Exec(`UPDATE collectors SET status=?, updated_at=? WHERE id=?`, status, now, id)
	return err
}

func (r *CollectorRepo) DeleteCollector(id int64) error {
	_, err := r.db.Exec(`DELETE FROM collectors WHERE id=?`, id)
	return err
}

// Mark offline collectors (no heartbeat in 5 minutes)
func (r *CollectorRepo) MarkStaleCollectors() error {
	_, err := r.db.Exec(
		`UPDATE collectors SET status='offline', updated_at=? WHERE status='online' AND last_heartbeat < ?`,
		time.Now(), time.Now().Add(-5*time.Minute),
	)
	return err
}

// --- Collector Configs ---

func (r *CollectorRepo) SaveConfig(cfg *model.CollectorConfig) error {
	// Get current version
	var maxVersion int
	r.db.QueryRow(`SELECT COALESCE(MAX(version),0) FROM collector_configs WHERE collector_id=?`, cfg.CollectorID).Scan(&maxVersion)

	now := time.Now()
	result, err := r.db.Exec(
		`INSERT INTO collector_configs (collector_id, config_type, content, version, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		cfg.CollectorID, cfg.ConfigType, cfg.Content, maxVersion+1, now,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	cfg.ID = id
	cfg.Version = maxVersion + 1
	cfg.CreatedAt = now
	return nil
}

func (r *CollectorRepo) GetLatestConfig(collectorID int64, configType string) (*model.CollectorConfig, error) {
	cfg := &model.CollectorConfig{}
	err := r.db.QueryRow(
		`SELECT id, collector_id, config_type, content, version, applied_at, created_at
		 FROM collector_configs WHERE collector_id=? AND config_type=? ORDER BY version DESC LIMIT 1`,
		collectorID, configType,
	).Scan(&cfg.ID, &cfg.CollectorID, &cfg.ConfigType, &cfg.Content, &cfg.Version, &cfg.AppliedAt, &cfg.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return cfg, err
}
