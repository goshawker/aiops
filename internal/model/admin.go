package model

import "time"

// User represents a system user.
type User struct {
	ID           int64     `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"`
	DisplayName  string    `json:"display_name" db:"display_name"`
	Email        string    `json:"email" db:"email"`
	Role         string    `json:"role" db:"role"` // "admin", "operator", "viewer"
	Status       string    `json:"status" db:"status"` // "active", "disabled"
	TenantID     int64     `json:"tenant_id" db:"tenant_id"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// AuditLog represents an audit trail entry.
type AuditLog struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Username  string    `json:"username" db:"username"`
	Action    string    `json:"action" db:"action"`     // "create", "update", "delete", "login", "logout"
	Resource  string    `json:"resource" db:"resource"` // "user", "alert_rule", "collector", etc.
	ResourceID string   `json:"resource_id" db:"resource_id"`
	Detail    string    `json:"detail" db:"detail"`
	IP        string    `json:"ip" db:"ip"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Role defines permission levels.
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

// RolePermissions maps roles to allowed actions.
var RolePermissions = map[string][]string{
	RoleAdmin:    {"read", "write", "delete", "manage_users", "manage_system", "acknowledge", "resolve"},
	RoleOperator: {"read", "write", "acknowledge", "resolve"},
	RoleViewer:   {"read"},
}

// HasPermission checks if a role has a specific permission.
func HasPermission(role, action string) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == action {
			return true
		}
	}
	return false
}
