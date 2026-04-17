package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/reignx/reignx/pkg/database/models"
	"github.com/reignx/reignx/pkg/observability/metrics"
	"go.uber.org/zap"
)

// TaskRepository defines the interface for task data access
type TaskRepository interface {
	Create(ctx context.Context, task *models.Task) (*models.Task, error)
	Get(ctx context.Context, id string) (*models.Task, error)
	GetByJob(ctx context.Context, jobID string) ([]*models.Task, error)
	Update(ctx context.Context, task *models.Task) error
	UpdateStatus(ctx context.Context, id string, status string) error
	Delete(ctx context.Context, id string) error
}

type taskRepository struct {
	db      *sqlx.DB
	logger  *zap.Logger
	metrics *metrics.Metrics
}

// NewTaskRepository creates a new task repository
func NewTaskRepository(db *sqlx.DB, logger *zap.Logger, metrics *metrics.Metrics) TaskRepository {
	return &taskRepository{
		db:      db,
		logger:  logger,
		metrics: metrics,
	}
}

// Create creates a new task
func (r *taskRepository) Create(ctx context.Context, task *models.Task) (*models.Task, error) {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "tasks",
				"operation": "create",
			})
		}
	}()

	query := `
		INSERT INTO tasks (id, job_id, node_id, type, command, status, priority, timeout, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, job_id, node_id, type, command, status, priority, timeout, result, created_at, started_at, completed_at
	`

	var createdTask models.Task
	err := r.db.GetContext(ctx, &createdTask, query,
		task.ID, task.JobID, task.NodeID, task.Type, task.Command,
		task.Status, task.Priority, task.Timeout, task.CreatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create task", zap.Error(err), zap.String("job_id", task.JobID))
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return &createdTask, nil
}

// Get retrieves a task by ID
func (r *taskRepository) Get(ctx context.Context, id string) (*models.Task, error) {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "tasks",
				"operation": "get",
			})
		}
	}()

	var task models.Task
	query := `
		SELECT id, job_id, node_id, type, command, status, priority, timeout, result, created_at, started_at, completed_at
		FROM tasks
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &task, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found: %s", id)
		}
		r.logger.Error("Failed to get task", zap.Error(err), zap.String("id", id))
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return &task, nil
}

// GetByJob retrieves all tasks for a job
func (r *taskRepository) GetByJob(ctx context.Context, jobID string) ([]*models.Task, error) {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "tasks",
				"operation": "get_by_job",
			})
		}
	}()

	var tasks []*models.Task
	query := `
		SELECT id, job_id, node_id, type, command, status, priority, timeout, result, created_at, started_at, completed_at
		FROM tasks
		WHERE job_id = $1
		ORDER BY priority DESC, created_at ASC
	`

	err := r.db.SelectContext(ctx, &tasks, query, jobID)
	if err != nil {
		r.logger.Error("Failed to get tasks by job", zap.Error(err), zap.String("job_id", jobID))
		return nil, fmt.Errorf("failed to get tasks by job: %w", err)
	}

	return tasks, nil
}

// Update updates a task
func (r *taskRepository) Update(ctx context.Context, task *models.Task) error {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "tasks",
				"operation": "update",
			})
		}
	}()

	query := `
		UPDATE tasks
		SET status = $2, result = $3, started_at = $4, completed_at = $5
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		task.ID, task.Status, task.Result, task.StartedAt, task.CompletedAt,
	)

	if err != nil {
		r.logger.Error("Failed to update task", zap.Error(err), zap.String("id", task.ID))
		return fmt.Errorf("failed to update task: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %s", task.ID)
	}

	return nil
}

// UpdateStatus updates a task's status
func (r *taskRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "tasks",
				"operation": "update_status",
			})
		}
	}()

	query := `UPDATE tasks SET status = $2 WHERE id = $1`

	// Also update started_at if status is running
	if status == "running" {
		query = `UPDATE tasks SET status = $2, started_at = NOW() WHERE id = $1`
	}

	// Update completed_at if status is completed or failed
	if status == "completed" || status == "failed" {
		query = `UPDATE tasks SET status = $2, completed_at = NOW() WHERE id = $1`
	}

	result, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		r.logger.Error("Failed to update task status", zap.Error(err), zap.String("id", id))
		return fmt.Errorf("failed to update task status: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %s", id)
	}

	return nil
}

// Delete deletes a task
func (r *taskRepository) Delete(ctx context.Context, id string) error {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "tasks",
				"operation": "delete",
			})
		}
	}()

	query := `DELETE FROM tasks WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete task", zap.Error(err), zap.String("id", id))
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %s", id)
	}

	return nil
}
