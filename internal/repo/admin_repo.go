package repo

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"aiops/internal/model"
)

// AdminRepo handles user and audit log persistence.
type AdminRepo struct {
	db        DBExecutor
	auditLock sync.Mutex // protects InsertAuditLog hash chain
}

func NewAdminRepo(db DBExecutor) *AdminRepo {
	return &AdminRepo{db: db}
}

// --- Users ---

func (r *AdminRepo) GetUserByUsername(username string) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(
		`SELECT id, username, password_hash, display_name, email, role, status, tenant_id, last_login_at, created_at, updated_at
		 FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Email, &u.Role, &u.Status, &u.TenantID, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *AdminRepo) GetUserByID(id int64) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(
		`SELECT id, username, password_hash, display_name, email, role, status, tenant_id, last_login_at, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Email, &u.Role, &u.Status, &u.TenantID, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (r *AdminRepo) ListUsers(limit, offset int) ([]model.User, int, error) {
	var total int
	r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&total)

	rows, err := r.db.Query(
		`SELECT id, username, password_hash, display_name, email, role, status, tenant_id, last_login_at, created_at, updated_at
		 FROM users ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Email, &u.Role, &u.Status, &u.TenantID, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		u.PasswordHash = "" // Don't expose hash
		users = append(users, u)
	}
	return users, total, nil
}

func (r *AdminRepo) CreateUser(u *model.User) error {
	now := time.Now()
	result, err := r.db.Exec(
		`INSERT INTO users (username, password_hash, display_name, email, role, status, tenant_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.Username, u.PasswordHash, u.DisplayName, u.Email, u.Role, u.Status, u.TenantID, now, now,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	u.ID = id
	u.CreatedAt = now
	u.UpdatedAt = now
	return nil
}

func (r *AdminRepo) UpdateUser(u *model.User) error {
	now := time.Now()
	_, err := r.db.Exec(
		`UPDATE users SET display_name=?, email=?, role=?, status=?, updated_at=? WHERE id=?`,
		u.DisplayName, u.Email, u.Role, u.Status, now, u.ID,
	)
	u.UpdatedAt = now
	return err
}

func (r *AdminRepo) UpdateLastLogin(userID int64) error {
	now := time.Now()
	_, err := r.db.Exec(`UPDATE users SET last_login_at=?, updated_at=? WHERE id=?`, now, now, userID)
	return err
}

func (r *AdminRepo) DeleteUser(id int64) error {
	_, err := r.db.Exec(`DELETE FROM users WHERE id=?`, id)
	return err
}

// --- Audit Logs ---

func (r *AdminRepo) InsertAuditLog(log *model.AuditLog) error {
	r.auditLock.Lock()
	defer r.auditLock.Unlock()

	// Get the previous record's hash for chain integrity
	var prevHash string
	err := r.db.QueryRow("SELECT record_hash FROM audit_logs ORDER BY id DESC LIMIT 1").Scan(&prevHash)
	if err != nil {
		prevHash = "genesis" // First record
	}

	// Compute record hash: SHA-256(prev_hash + record_data)
	now := time.Now()
	recordData := fmt.Sprintf("%d:%s:%s:%s:%s:%s:%s:%s",
		log.UserID, log.Username, log.Action, log.Resource,
		log.ResourceID, log.Detail, log.IP, now.Format(time.RFC3339))
	recordHash := sha256Hash(prevHash + recordData)

	result, err := r.db.Exec(
		`INSERT INTO audit_logs (user_id, username, action, resource, resource_id, detail, ip, prev_hash, record_hash, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.UserID, log.Username, log.Action, log.Resource, log.ResourceID, log.Detail, log.IP, prevHash, recordHash, now,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	log.ID = id
	return nil
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func (r *AdminRepo) ListAuditLogs(limit, offset int, userID int64, action string) ([]model.AuditLog, int, error) {
	where := "1=1"
	args := []interface{}{}
	if userID > 0 {
		where += " AND user_id = ?"
		args = append(args, userID)
	}
	if action != "" {
		where += " AND action = ?"
		args = append(args, action)
	}

	var total int
	r.db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE `+where, args...).Scan(&total)

	args = append(args, limit, offset)
	rows, err := r.db.Query(
		`SELECT id, user_id, username, action, resource, resource_id, detail, ip, created_at
		 FROM audit_logs WHERE `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []model.AuditLog
	for rows.Next() {
		var l model.AuditLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.Username, &l.Action, &l.Resource, &l.ResourceID, &l.Detail, &l.IP, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	return logs, total, nil
}
