package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/reignx/reignx/pkg/core"
)

// TaskRepository implements core.TaskRepository
type TaskRepository struct {
	db *DB
}

// NewTaskRepository creates a new task repository
func NewTaskRepository(db *DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create creates a new task
func (r *TaskRepository) Create(ctx context.Context, task *core.Task) error {
	parametersJSON, err := json.Marshal(task.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		INSERT INTO tasks (
			id, job_id, node_id, type, command, script, parameters,
			status, priority, timeout, max_retries
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`

	_, err = r.db.ExecContext(ctx, query,
		task.ID, task.JobID, task.NodeID, task.Type,
		task.Command, task.Script, parametersJSON,
		task.Status, task.Priority, int(task.Timeout.Seconds()), task.MaxRetries,
	)

	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

// Get retrieves a task by ID
func (r *TaskRepository) Get(ctx context.Context, id string) (*core.Task, error) {
	query := `
		SELECT
			id, job_id, node_id, type, command, script, parameters,
			status, priority, timeout, retries, max_retries, result,
			created_at, started_at, completed_at
		FROM tasks
		WHERE id = $1`

	var task core.Task
	var parametersJSON, resultJSON []byte
	var jobID, command, script sql.NullString
	var startedAt, completedAt sql.NullTime
	var timeout int

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID, &jobID, &task.NodeID, &task.Type,
		&command, &script, &parametersJSON,
		&task.Status, &task.Priority, &timeout, &task.Retries,
		&task.MaxRetries, &resultJSON, &task.CreatedAt,
		&startedAt, &completedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if jobID.Valid {
		task.JobID = jobID.String
	}
	if command.Valid {
		task.Command = command.String
	}
	if script.Valid {
		task.Script = script.String
	}

	if err := json.Unmarshal(parametersJSON, &task.Parameters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	if len(resultJSON) > 0 {
		var result core.TaskResult
		if err := json.Unmarshal(resultJSON, &result); err == nil {
			task.Result = &result
		}
	}

	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}

	return &task, nil
}

// List retrieves tasks matching criteria
func (r *TaskRepository) List(ctx context.Context, filter *core.TaskFilter) ([]*core.Task, error) {
	query := `
		SELECT
			id, job_id, node_id, type, command, script, parameters,
			status, priority, timeout, retries, max_retries,
			created_at, started_at, completed_at
		FROM tasks
		WHERE 1=1`

	args := []interface{}{}
	argCount := 1

	if filter.JobID != "" {
		query += fmt.Sprintf(" AND job_id = $%d", argCount)
		args = append(args, filter.JobID)
		argCount++
	}

	if filter.NodeID != "" {
		query += fmt.Sprintf(" AND node_id = $%d", argCount)
		args = append(args, filter.NodeID)
		argCount++
	}

	if len(filter.Status) > 0 {
		query += fmt.Sprintf(" AND status = ANY($%d)", argCount)
		args = append(args, filter.Status)
		argCount++
	}

	query += " ORDER BY priority DESC, created_at ASC"

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
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*core.Task
	for rows.Next() {
		var task core.Task
		var parametersJSON []byte
		var jobID, command, script sql.NullString
		var startedAt, completedAt sql.NullTime
		var timeout int

		err := rows.Scan(
			&task.ID, &jobID, &task.NodeID, &task.Type,
			&command, &script, &parametersJSON,
			&task.Status, &task.Priority, &timeout, &task.Retries,
			&task.MaxRetries, &task.CreatedAt, &startedAt, &completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		if jobID.Valid {
			task.JobID = jobID.String
		}
		if command.Valid {
			task.Command = command.String
		}
		if script.Valid {
			task.Script = script.String
		}

		json.Unmarshal(parametersJSON, &task.Parameters)

		if startedAt.Valid {
			task.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// Update updates a task
func (r *TaskRepository) Update(ctx context.Context, task *core.Task) error {
	parametersJSON, _ := json.Marshal(task.Parameters)
	resultJSON, _ := json.Marshal(task.Result)

	query := `
		UPDATE tasks SET
			status = $2,
			priority = $3,
			retries = $4,
			result = $5,
			started_at = $6,
			completed_at = $7,
			parameters = $8
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		task.ID, task.Status, task.Priority, task.Retries,
		resultJSON, task.StartedAt, task.CompletedAt, parametersJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// UpdateStatus updates task status
func (r *TaskRepository) UpdateStatus(ctx context.Context, id string, status core.TaskStatus) error {
	query := `UPDATE tasks SET status = $2 WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	return nil
}

// UpdateResult updates task result
func (r *TaskRepository) UpdateResult(ctx context.Context, id string, result *core.TaskResult) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	query := `UPDATE tasks SET result = $2, status = $3 WHERE id = $1`

	status := core.TaskStatusCompleted
	if !result.Success {
		status = core.TaskStatusFailed
	}

	_, err = r.db.ExecContext(ctx, query, id, resultJSON, status)
	if err != nil {
		return fmt.Errorf("failed to update task result: %w", err)
	}

	return nil
}

// Delete deletes a task
func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM tasks WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

// GetByJob retrieves all tasks for a job
func (r *TaskRepository) GetByJob(ctx context.Context, jobID string) ([]*core.Task, error) {
	return r.List(ctx, &core.TaskFilter{JobID: jobID})
}

// GetByNode retrieves all tasks for a node
func (r *TaskRepository) GetByNode(ctx context.Context, nodeID string, limit int) ([]*core.Task, error) {
	return r.List(ctx, &core.TaskFilter{NodeID: nodeID, Limit: limit})
}

// CountByStatus counts tasks by status
func (r *TaskRepository) CountByStatus(ctx context.Context, jobID string) (map[core.TaskStatus]int, error) {
	query := `
		SELECT status, COUNT(*)
		FROM tasks
		WHERE job_id = $1
		GROUP BY status`

	rows, err := r.db.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to count tasks by status: %w", err)
	}
	defer rows.Close()

	counts := make(map[core.TaskStatus]int)
	for rows.Next() {
		var status core.TaskStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}

	return counts, nil
}

// GetPendingTasks retrieves pending tasks for a specific node
func (r *TaskRepository) GetPendingTasks(ctx context.Context, nodeID string, limit int) ([]*core.Task, error) {
	query := `
		SELECT
			id, job_id, node_id, type, command, script, parameters,
			status, priority, timeout, retries, max_retries, created_at
		FROM tasks
		WHERE node_id = $1
		  AND status = 'pending'
		ORDER BY priority DESC, created_at ASC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, nodeID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*core.Task
	for rows.Next() {
		task := &core.Task{}
		var parametersJSON []byte
		var script, jobID sql.NullString
		var timeout int

		err := rows.Scan(
			&task.ID, &jobID, &task.NodeID, &task.Type,
			&task.Command, &script, &parametersJSON,
			&task.Status, &task.Priority, &timeout,
			&task.Retries, &task.MaxRetries, &task.CreatedAt,
		)
		if err != nil {
			continue
		}

		if jobID.Valid {
			task.JobID = jobID.String
		}
		if script.Valid {
			task.Script = script.String
		}
		task.Timeout = time.Duration(timeout) * time.Second

		if len(parametersJSON) > 0 {
			json.Unmarshal(parametersJSON, &task.Parameters)
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}
