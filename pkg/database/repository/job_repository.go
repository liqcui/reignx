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

// JobRepository defines the interface for job data access
type JobRepository interface {
	Create(ctx context.Context, job *models.Job) (*models.Job, error)
	Get(ctx context.Context, id string) (*models.Job, error)
	List(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*models.Job, error)
	Update(ctx context.Context, job *models.Job) (*models.Job, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateProgress(ctx context.Context, id string, completed, failed int) error
	Delete(ctx context.Context, id string) error
}

type jobRepository struct {
	db      *sqlx.DB
	logger  *zap.Logger
	metrics *metrics.Metrics
}

// NewJobRepository creates a new job repository
func NewJobRepository(db *sqlx.DB, logger *zap.Logger, metrics *metrics.Metrics) JobRepository {
	return &jobRepository{
		db:      db,
		logger:  logger,
		metrics: metrics,
	}
}

// Create creates a new job
func (r *jobRepository) Create(ctx context.Context, job *models.Job) (*models.Job, error) {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "jobs",
				"operation": "create",
			})
		}
	}()

	query := `
		INSERT INTO jobs (id, name, description, type, mode, filter, template, status, batch_size, concurrency, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, name, description, type, mode, filter, template, status, batch_size, concurrency, total_tasks, completed, failed, created_by, created_at, started_at, completed_at
	`

	var createdJob models.Job
	err := r.db.GetContext(ctx, &createdJob, query,
		job.ID, job.Name, job.Description, job.Type, job.Mode, job.Filter, job.Template,
		job.Status, job.BatchSize, job.Concurrency, job.CreatedBy, job.CreatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create job", zap.Error(err), zap.String("name", job.Name))
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	r.logger.Info("Job created", zap.String("id", createdJob.ID), zap.String("name", createdJob.Name))
	return &createdJob, nil
}

// Get retrieves a job by ID
func (r *jobRepository) Get(ctx context.Context, id string) (*models.Job, error) {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "jobs",
				"operation": "get",
			})
		}
	}()

	var job models.Job
	query := `
		SELECT id, name, description, type, mode, filter, template, status, batch_size, concurrency,
		       total_tasks, completed, failed, created_by, created_at, started_at, completed_at
		FROM jobs
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &job, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s", id)
		}
		r.logger.Error("Failed to get job", zap.Error(err), zap.String("id", id))
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &job, nil
}

// List retrieves jobs with optional filters
func (r *jobRepository) List(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*models.Job, error) {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "jobs",
				"operation": "list",
			})
		}
	}()

	query := `
		SELECT id, name, description, type, mode, filter, template, status, batch_size, concurrency,
		       total_tasks, completed, failed, created_by, created_at, started_at, completed_at
		FROM jobs
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	// Apply filters
	if status, ok := filters["status"].(string); ok && status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, status)
		argPos++
	}

	if createdBy, ok := filters["created_by"].(string); ok && createdBy != "" {
		query += fmt.Sprintf(" AND created_by = $%d", argPos)
		args = append(args, createdBy)
		argPos++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	var jobs []*models.Job
	err := r.db.SelectContext(ctx, &jobs, query, args...)
	if err != nil {
		r.logger.Error("Failed to list jobs", zap.Error(err))
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	return jobs, nil
}

// Update updates a job
func (r *jobRepository) Update(ctx context.Context, job *models.Job) (*models.Job, error) {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "jobs",
				"operation": "update",
			})
		}
	}()

	query := `
		UPDATE jobs
		SET name = $2, description = $3, type = $4, mode = $5, filter = $6, template = $7,
		    status = $8, batch_size = $9, concurrency = $10, total_tasks = $11,
		    completed = $12, failed = $13, started_at = $14, completed_at = $15
		WHERE id = $1
		RETURNING id, name, description, type, mode, filter, template, status, batch_size, concurrency,
		          total_tasks, completed, failed, created_by, created_at, started_at, completed_at
	`

	var updatedJob models.Job
	err := r.db.GetContext(ctx, &updatedJob, query,
		job.ID, job.Name, job.Description, job.Type, job.Mode, job.Filter, job.Template,
		job.Status, job.BatchSize, job.Concurrency, job.TotalTasks, job.Completed, job.Failed,
		job.StartedAt, job.CompletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s", job.ID)
		}
		r.logger.Error("Failed to update job", zap.Error(err), zap.String("id", job.ID))
		return nil, fmt.Errorf("failed to update job: %w", err)
	}

	return &updatedJob, nil
}

// UpdateStatus updates a job's status
func (r *jobRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "jobs",
				"operation": "update_status",
			})
		}
	}()

	query := `UPDATE jobs SET status = $2 WHERE id = $1`

	// Also update started_at if status is running
	if status == "running" {
		query = `UPDATE jobs SET status = $2, started_at = NOW() WHERE id = $1`
	}

	// Update completed_at if status is completed or failed
	if status == "completed" || status == "failed" || status == "completed_with_errors" {
		query = `UPDATE jobs SET status = $2, completed_at = NOW() WHERE id = $1`
	}

	result, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		r.logger.Error("Failed to update job status", zap.Error(err), zap.String("id", id))
		return fmt.Errorf("failed to update job status: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("job not found: %s", id)
	}

	return nil
}

// UpdateProgress updates a job's progress counters
func (r *jobRepository) UpdateProgress(ctx context.Context, id string, completed, failed int) error {
	query := `
		UPDATE jobs
		SET completed = completed + $2, failed = failed + $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, completed, failed)
	if err != nil {
		r.logger.Error("Failed to update job progress", zap.Error(err), zap.String("id", id))
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("job not found: %s", id)
	}

	return nil
}

// Delete deletes a job
func (r *jobRepository) Delete(ctx context.Context, id string) error {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "jobs",
				"operation": "delete",
			})
		}
	}()

	query := `DELETE FROM jobs WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete job", zap.Error(err), zap.String("id", id))
		return fmt.Errorf("failed to delete job: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("job not found: %s", id)
	}

	r.logger.Info("Job deleted", zap.String("id", id))
	return nil
}
