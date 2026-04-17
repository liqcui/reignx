package models

import (
	"database/sql"
	"time"
)

// Task represents a task in the system
type Task struct {
	ID          string       `db:"id" json:"id"`
	JobID       string       `db:"job_id" json:"job_id"`
	NodeID      string       `db:"node_id" json:"node_id"`
	Type        string       `db:"type" json:"type"`
	Command     string       `db:"command" json:"command,omitempty"`
	Status      string       `db:"status" json:"status"`
	Priority    int          `db:"priority" json:"priority"`
	Timeout     int          `db:"timeout" json:"timeout"`
	Result      JSONB        `db:"result" json:"result,omitempty"`
	CreatedAt   time.Time    `db:"created_at" json:"created_at"`
	StartedAt   sql.NullTime `db:"started_at" json:"started_at,omitempty"`
	CompletedAt sql.NullTime `db:"completed_at" json:"completed_at,omitempty"`
}
