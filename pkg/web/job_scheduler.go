package web

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/reignx/reignx/pkg/database/models"
	"github.com/reignx/reignx/pkg/database/repository"
	"go.uber.org/zap"
)

// JobScheduler handles job orchestration and task execution
type JobScheduler struct {
	jobRepo    repository.JobRepository
	taskRepo   repository.TaskRepository
	serverRepo repository.ServerRepository
	executor   *SSHExecutor
	logger     *zap.Logger
}

// NewJobScheduler creates a new job scheduler
func NewJobScheduler(
	jobRepo repository.JobRepository,
	taskRepo repository.TaskRepository,
	serverRepo repository.ServerRepository,
	executor *SSHExecutor,
	logger *zap.Logger,
) *JobScheduler {
	return &JobScheduler{
		jobRepo:    jobRepo,
		taskRepo:   taskRepo,
		serverRepo: serverRepo,
		executor:   executor,
		logger:     logger,
	}
}

// ScheduleJob schedules a job for execution
func (s *JobScheduler) ScheduleJob(jobID string) {
	ctx := context.Background()

	s.logger.Info("Scheduling job", zap.String("job_id", jobID))

	// Get job from database
	job, err := s.jobRepo.Get(ctx, jobID)
	if err != nil {
		s.logger.Error("Failed to get job", zap.String("job_id", jobID), zap.Error(err))
		return
	}

	// Update job status to running
	if err := s.jobRepo.UpdateStatus(ctx, jobID, "running"); err != nil {
		s.logger.Error("Failed to update job status", zap.String("job_id", jobID), zap.Error(err))
		return
	}

	// Parse filter to query servers
	filter := map[string]interface{}(job.Filter)

	// Query servers matching filter
	servers, err := s.serverRepo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list servers", zap.String("job_id", jobID), zap.Error(err))
		s.jobRepo.UpdateStatus(ctx, jobID, "failed")
		return
	}

	if len(servers) == 0 {
		s.logger.Warn("No servers match filter", zap.String("job_id", jobID))
		s.jobRepo.UpdateStatus(ctx, jobID, "completed")
		return
	}

	s.logger.Info("Creating tasks for job",
		zap.String("job_id", jobID),
		zap.Int("server_count", len(servers)))

	// Parse template
	template := map[string]interface{}(job.Template)

	// Create tasks for each server
	for _, server := range servers {
		task := &models.Task{
			ID:        uuid.New().String(),
			JobID:     jobID,
			NodeID:    server.ID,
			Type:      job.Type,
			Status:    "pending",
			CreatedAt: time.Now(),
		}

		// Extract command from template
		if cmd, ok := template["command"].(string); ok {
			task.Command = cmd
		}

		// Extract timeout from template (default 300 seconds)
		if timeout, ok := template["timeout"].(float64); ok {
			task.Timeout = int(timeout)
		} else {
			task.Timeout = 300
		}

		// Extract priority from template (default 1)
		if priority, ok := template["priority"].(float64); ok {
			task.Priority = int(priority)
		} else {
			task.Priority = 1
		}

		if _, err := s.taskRepo.Create(ctx, task); err != nil {
			s.logger.Error("Failed to create task",
				zap.String("job_id", jobID),
				zap.String("server_id", server.ID),
				zap.Error(err))
		}
	}

	// Execute tasks with concurrency control
	concurrency := job.Concurrency
	if concurrency <= 0 {
		concurrency = 5 // Default concurrency
	}

	go s.executeTasks(jobID, concurrency)
}

// executeTasks executes all pending tasks for a job with concurrency control
func (s *JobScheduler) executeTasks(jobID string, concurrency int) {
	ctx := context.Background()

	s.logger.Info("Executing tasks for job",
		zap.String("job_id", jobID),
		zap.Int("concurrency", concurrency))

	// Get all tasks for the job
	tasks, err := s.taskRepo.GetByJob(ctx, jobID)
	if err != nil {
		s.logger.Error("Failed to get tasks", zap.String("job_id", jobID), zap.Error(err))
		s.jobRepo.UpdateStatus(ctx, jobID, "failed")
		return
	}

	// Use semaphore for concurrency control
	sem := make(chan struct{}, concurrency)
	done := make(chan bool, len(tasks))

	completedCount := 0
	failedCount := 0

	for _, task := range tasks {
		if task.Status != "pending" {
			continue
		}

		sem <- struct{}{} // Acquire semaphore

		go func(t *models.Task) {
			defer func() {
				<-sem // Release semaphore
				done <- true
			}()

			// Execute task
			success := s.executeTask(ctx, t)

			if success {
				completedCount++
			} else {
				failedCount++
			}

			// Update job progress
			s.jobRepo.UpdateProgress(ctx, jobID, completedCount, failedCount)
		}(task)
	}

	// Wait for all tasks to complete
	totalTasks := len(tasks)
	for i := 0; i < totalTasks; i++ {
		<-done
	}

	// Update job status based on results
	finalStatus := "completed"
	if failedCount > 0 {
		if completedCount == 0 {
			finalStatus = "failed"
		} else {
			finalStatus = "completed_with_errors"
		}
	}

	s.jobRepo.UpdateStatus(ctx, jobID, finalStatus)

	s.logger.Info("Job execution completed",
		zap.String("job_id", jobID),
		zap.String("status", finalStatus),
		zap.Int("completed", completedCount),
		zap.Int("failed", failedCount))
}

// executeTask executes a single task
func (s *JobScheduler) executeTask(ctx context.Context, task *models.Task) bool {
	s.logger.Info("Executing task",
		zap.String("task_id", task.ID),
		zap.String("node_id", task.NodeID),
		zap.String("command", task.Command))

	// Update task status to running
	if err := s.taskRepo.UpdateStatus(ctx, task.ID, "running"); err != nil {
		s.logger.Error("Failed to update task status", zap.String("task_id", task.ID), zap.Error(err))
		return false
	}

	task.StartedAt = sql.NullTime{Time: time.Now(), Valid: true}

	// Execute command via SSH
	timeout := time.Duration(task.Timeout) * time.Second
	result, err := s.executor.ExecuteCommand(ctx, task.NodeID, task.Command, timeout)

	task.CompletedAt = sql.NullTime{Time: time.Now(), Valid: true}

	// Prepare result
	var taskResult map[string]interface{}
	if result != nil {
		taskResult = map[string]interface{}{
			"exit_code": result.ExitCode,
			"stdout":    result.Stdout,
			"stderr":    result.Stderr,
			"duration":  result.Duration.Seconds(),
		}

		if result.Error != "" {
			taskResult["error"] = result.Error
		}
	} else if err != nil {
		taskResult = map[string]interface{}{
			"error": err.Error(),
		}
	}

	// Determine task status
	status := "completed"
	if err != nil || (result != nil && result.ExitCode != 0) {
		status = "failed"
	}

	// Update task with result
	task.Status = status
	task.Result = models.JSONB(taskResult)

	if err := s.taskRepo.Update(ctx, task); err != nil {
		s.logger.Error("Failed to update task", zap.String("task_id", task.ID), zap.Error(err))
		return false
	}

	success := status == "completed"

	s.logger.Info("Task execution completed",
		zap.String("task_id", task.ID),
		zap.String("status", status),
		zap.Bool("success", success))

	return success
}
