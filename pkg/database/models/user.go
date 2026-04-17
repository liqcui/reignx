package models

import (
	"database/sql"
	"time"
)

// User represents a user in the system
type User struct {
	ID             string         `db:"id" json:"id"`
	Username       string         `db:"username" json:"username"`
	Email          string         `db:"email" json:"email"`
	PasswordHash   string         `db:"password_hash" json:"-"` // Never expose password hash in JSON
	Role           string         `db:"role" json:"role"`
	Enabled        bool           `db:"enabled" json:"enabled"`
	LastLoginAt    sql.NullTime   `db:"last_login" json:"last_login,omitempty"`
	CreatedAt      time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time      `db:"updated_at" json:"updated_at"`
}

// UserRole constants
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

// IsValidRole checks if a role is valid
func IsValidRole(role string) bool {
	switch role {
	case RoleAdmin, RoleOperator, RoleViewer:
		return true
	default:
		return false
	}
}
