package models

import (
	"database/sql"
	"time"
)

// Job represents a job in the system
type Job struct {
	ID          string         `db:"id" json:"id"`
	Name        string         `db:"name" json:"name"`
	Description string         `db:"description" json:"description,omitempty"`
	Type        string         `db:"type" json:"type"`
	Mode        string         `db:"mode" json:"mode"`
	Filter      JSONB          `db:"filter" json:"filter"`
	Template    JSONB          `db:"template" json:"template"`
	Status      string         `db:"status" json:"status"`
	BatchSize   int            `db:"batch_size" json:"batch_size"`
	Concurrency int            `db:"concurrency" json:"concurrency"`
	TotalTasks  int            `db:"total_tasks" json:"total_tasks"`
	Completed   int            `db:"completed" json:"completed"`
	Failed      int            `db:"failed" json:"failed"`
	CreatedBy   string         `db:"created_by" json:"created_by"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
	StartedAt   sql.NullTime   `db:"started_at" json:"started_at,omitempty"`
	CompletedAt sql.NullTime   `db:"completed_at" json:"completed_at,omitempty"`
}
