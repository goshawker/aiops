package model

import "time"

// Tenant represents a multi-tenant organization.
type Tenant struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Code      string    `json:"code" db:"code"`     // unique short code
	Status    string    `json:"status" db:"status"` // "active", "disabled"
	Plan      string    `json:"plan" db:"plan"`     // "free", "basic", "pro", "enterprise"
	MaxHosts  int       `json:"max_hosts" db:"max_hosts"`
	MaxUsers  int       `json:"max_users" db:"max_users"`
	Settings  string    `json:"settings" db:"settings"` // JSON
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// LDAPConfig holds LDAP connection settings.
type LDAPConfig struct {
	ID       int64  `json:"id" db:"id"`
	TenantID int64  `json:"tenant_id" db:"tenant_id"`
	Enabled  bool   `json:"enabled" db:"enabled"`
	Host     string `json:"host" db:"host"`
	Port     int    `json:"port" db:"port"`
	BaseDN   string `json:"base_dn" db:"base_dn"`
	BindDN   string `json:"bind_dn" db:"bind_dn"`
	BindPass string `json:"bind_pass" db:"bind_pass"`
	UserAttr string `json:"user_attr" db:"user_attr"` // "uid", "sAMAccountName", "mail"
	GroupAttr string `json:"group_attr" db:"group_attr"`
	UseTLS   bool   `json:"use_tls" db:"use_tls"`
}

// PlanLimits defines resource limits per plan.
var PlanLimits = map[string]Tenant{
	"free":       {MaxHosts: 5, MaxUsers: 3},
	"basic":      {MaxHosts: 20, MaxUsers: 10},
	"pro":        {MaxHosts: 100, MaxUsers: 50},
	"enterprise": {MaxHosts: 9999, MaxUsers: 9999},
}
