package repo

import (
	"database/sql"
	"time"

	"aiops/internal/model"
)

// TenantRepo handles tenant persistence.
type TenantRepo struct {
	db DBExecutor
}

func NewTenantRepo(db DBExecutor) *TenantRepo {
	return &TenantRepo{db: db}
}

func (r *TenantRepo) CreateTenant(t *model.Tenant) error {
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.Status == "" {
		t.Status = "active"
	}
	if t.Plan == "" {
		t.Plan = "free"
	}
	if t.Settings == "" {
		t.Settings = "{}"
	}

	result, err := r.db.Exec(
		`INSERT INTO tenants (name, code, status, plan, max_hosts, max_users, settings, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.Name, t.Code, t.Status, t.Plan, t.MaxHosts, t.MaxUsers, t.Settings, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	t.ID = id
	return nil
}

func (r *TenantRepo) GetTenant(id int64) (*model.Tenant, error) {
	t := &model.Tenant{}
	err := r.db.QueryRow(
		`SELECT id, name, code, status, plan, max_hosts, max_users, settings, created_at, updated_at
		 FROM tenants WHERE id = ?`, id,
	).Scan(&t.ID, &t.Name, &t.Code, &t.Status, &t.Plan, &t.MaxHosts, &t.MaxUsers, &t.Settings, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (r *TenantRepo) ListTenants(limit, offset int) ([]model.Tenant, int, error) {
	var total int
	r.db.QueryRow(`SELECT COUNT(*) FROM tenants`).Scan(&total)

	rows, err := r.db.Query(
		`SELECT id, name, code, status, plan, max_hosts, max_users, settings, created_at, updated_at
		 FROM tenants ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tenants []model.Tenant
	for rows.Next() {
		var t model.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Code, &t.Status, &t.Plan, &t.MaxHosts, &t.MaxUsers, &t.Settings, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		tenants = append(tenants, t)
	}
	return tenants, total, nil
}

func (r *TenantRepo) UpdateTenant(t *model.Tenant) error {
	t.UpdatedAt = time.Now()
	_, err := r.db.Exec(
		`UPDATE tenants SET name=?, code=?, status=?, plan=?, max_hosts=?, max_users=?, settings=?, updated_at=? WHERE id=?`,
		t.Name, t.Code, t.Status, t.Plan, t.MaxHosts, t.MaxUsers, t.Settings, t.UpdatedAt, t.ID,
	)
	return err
}

func (r *TenantRepo) DeleteTenant(id int64) error {
	_, err := r.db.Exec(`DELETE FROM tenants WHERE id=?`, id)
	return err
}
