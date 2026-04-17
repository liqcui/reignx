package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/reignx/reignx/pkg/core"
)

// JobRepository implements core.JobRepository
type JobRepository struct {
	db *DB
}

// NewJobRepository creates a new job repository
func NewJobRepository(db *DB) *JobRepository {
	return &JobRepository{db: db}
}

// Create creates a new job
func (r *JobRepository) Create(ctx context.Context, job *core.Job) error {
	filterJSON, err := json.Marshal(job.Filter)
	if err != nil {
		return fmt.Errorf("failed to marshal filter: %w", err)
	}

	templateJSON, err := json.Marshal(job.Template)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	parametersJSON, err := json.Marshal(job.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		INSERT INTO jobs (
			id, name, description, type, mode, filter, template,
			status, batch_size, concurrency, created_by, parameters
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)`

	_, err = r.db.ExecContext(ctx, query,
		job.ID, job.Name, job.Description, job.Type, job.Mode,
		filterJSON, templateJSON, job.Status, job.BatchSize,
		job.Concurrency, job.CreatedBy, parametersJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

// Get retrieves a job by ID
func (r *JobRepository) Get(ctx context.Context, id string) (*core.Job, error) {
	query := `
		SELECT
			id, name, description, type, mode, filter, template,
			status, batch_size, concurrency, total_tasks, completed, failed,
			created_by, created_at, started_at, completed_at, parameters
		FROM jobs
		WHERE id = $1`

	var job core.Job
	var filterJSON, templateJSON, parametersJSON []byte
	var startedAt, completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&job.ID, &job.Name, &job.Description, &job.Type, &job.Mode,
		&filterJSON, &templateJSON, &job.Status, &job.BatchSize,
		&job.Concurrency, &job.TotalTasks, &job.Completed, &job.Failed,
		&job.CreatedBy, &job.CreatedAt, &startedAt, &completedAt,
		&parametersJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if err := json.Unmarshal(filterJSON, &job.Filter); err != nil {
		return nil, fmt.Errorf("failed to unmarshal filter: %w", err)
	}

	if err := json.Unmarshal(templateJSON, &job.Template); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}

	if err := json.Unmarshal(parametersJSON, &job.Parameters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}

	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	return &job, nil
}

// List retrieves jobs matching criteria
func (r *JobRepository) List(ctx context.Context, filter *core.JobFilter) ([]*core.Job, error) {
	query := `
		SELECT
			id, name, description, type, mode, filter, template,
			status, batch_size, concurrency, total_tasks, completed, failed,
			created_by, created_at, started_at, completed_at, parameters
		FROM jobs
		WHERE 1=1`

	args := []interface{}{}
	argCount := 1

	if len(filter.Status) > 0 {
		query += fmt.Sprintf(" AND status = ANY($%d)", argCount)
		args = append(args, filter.Status)
		argCount++
	}

	if filter.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, filter.Type)
		argCount++
	}

	if filter.CreatedBy != "" {
		query += fmt.Sprintf(" AND created_by = $%d", argCount)
		args = append(args, filter.CreatedBy)
		argCount++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
		argCount++
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*core.Job
	for rows.Next() {
		var job core.Job
		var filterJSON, templateJSON, parametersJSON []byte
		var startedAt, completedAt sql.NullTime

		err := rows.Scan(
			&job.ID, &job.Name, &job.Description, &job.Type, &job.Mode,
			&filterJSON, &templateJSON, &job.Status, &job.BatchSize,
			&job.Concurrency, &job.TotalTasks, &job.Completed, &job.Failed,
			&job.CreatedBy, &job.CreatedAt, &startedAt, &completedAt,
			&parametersJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}

		json.Unmarshal(filterJSON, &job.Filter)
		json.Unmarshal(templateJSON, &job.Template)
		json.Unmarshal(parametersJSON, &job.Parameters)

		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}

		jobs = append(jobs, &job)
	}

	return jobs, nil
}

// Update updates a job
func (r *JobRepository) Update(ctx context.Context, job *core.Job) error {
	filterJSON, _ := json.Marshal(job.Filter)
	templateJSON, _ := json.Marshal(job.Template)
	parametersJSON, _ := json.Marshal(job.Parameters)

	query := `
		UPDATE jobs SET
			name = $2,
			description = $3,
			status = $4,
			batch_size = $5,
			concurrency = $6,
			total_tasks = $7,
			completed = $8,
			failed = $9,
			started_at = $10,
			completed_at = $11,
			filter = $12,
			template = $13,
			parameters = $14
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		job.ID, job.Name, job.Description, job.Status,
		job.BatchSize, job.Concurrency, job.TotalTasks,
		job.Completed, job.Failed, job.StartedAt, job.CompletedAt,
		filterJSON, templateJSON, parametersJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

// UpdateStatus updates job status
func (r *JobRepository) UpdateStatus(ctx context.Context, id string, status core.TaskStatus) error {
	query := `UPDATE jobs SET status = $2 WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

// Delete deletes a job
func (r *JobRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM jobs WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	return nil
}

// UpdateProgress updates job progress counters
func (r *JobRepository) UpdateProgress(ctx context.Context, id string, completed, failed int) error {
	query := `
		UPDATE jobs SET
			completed = $2,
			failed = $3
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, completed, failed)
	if err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	return nil
}
