package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/reignx/reignx/pkg/core"
	"go.uber.org/zap"
)

// Config contains scheduler configuration
type Config struct {
	Logger        *zap.Logger
	JobRepo       core.JobRepository
	TaskRepo      core.TaskRepository
	NodeRepo      core.NodeRepository
	SSHExecutor   core.Executor
	AgentExecutor core.Executor
	PollInterval  time.Duration
}

// Scheduler orchestrates job execution
type Scheduler struct {
	config   *Config
	logger   *zap.Logger
	jobRepo  core.JobRepository
	taskRepo core.TaskRepository
	nodeRepo core.NodeRepository
	executors map[core.NodeMode]core.Executor
}

// New creates a new scheduler
func New(config *Config) *Scheduler {
	return &Scheduler{
		config:   config,
		logger:   config.Logger,
		jobRepo:  config.JobRepo,
		taskRepo: config.TaskRepo,
		nodeRepo: config.NodeRepo,
		executors: map[core.NodeMode]core.Executor{
			core.NodeModeSSH:    config.SSHExecutor,
			core.NodeModeAgent:  config.AgentExecutor,
			core.NodeModeHybrid: config.AgentExecutor, // Prefer agent for hybrid
		},
	}
}

// Start starts the scheduler loop
func (s *Scheduler) Start(ctx context.Context) error {
	s.logger.Info("Scheduler started")

	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Scheduler stopping...")
			return ctx.Err()

		case <-ticker.C:
			if err := s.processPendingJobs(ctx); err != nil {
				s.logger.Error("Failed to process pending jobs", zap.Error(err))
			}

			// NOTE: With polling approach (Option 1), tasks are delivered via heartbeat
			// The scheduler no longer executes tasks directly
			// if err := s.processPendingTasks(ctx); err != nil {
			// 	s.logger.Error("Failed to process pending tasks", zap.Error(err))
			// }
		}
	}
}

// processPendingJobs processes jobs in pending status
func (s *Scheduler) processPendingJobs(ctx context.Context) error {
	// Get pending jobs
	jobs, err := s.jobRepo.List(ctx, &core.JobFilter{
		Status: []core.TaskStatus{core.TaskStatusPending},
		Limit:  10,
	})
	if err != nil {
		return fmt.Errorf("failed to list pending jobs: %w", err)
	}

	for _, job := range jobs {
		if err := s.scheduleJob(ctx, job); err != nil {
			s.logger.Error("Failed to schedule job",
				zap.String("job_id", job.ID),
				zap.Error(err),
			)
			continue
		}
	}

	return nil
}

// scheduleJob creates tasks for a job
func (s *Scheduler) scheduleJob(ctx context.Context, job *core.Job) error {
	s.logger.Info("Scheduling job",
		zap.String("job_id", job.ID),
		zap.String("type", string(job.Type)),
	)

	// Resolve target nodes based on filter
	nodes, err := s.nodeRepo.List(ctx, job.Filter)
	if err != nil {
		return fmt.Errorf("failed to resolve target nodes: %w", err)
	}

	if len(nodes) == 0 {
		s.logger.Warn("No nodes match job filter", zap.String("job_id", job.ID))
		job.Status = core.TaskStatusCompleted
		return s.jobRepo.Update(ctx, job)
	}

	// Create tasks for each node
	tasks := make([]*core.Task, 0, len(nodes))
	for _, node := range nodes {
		task := &core.Task{
			ID:         uuid.New().String(),
			JobID:      job.ID,
			NodeID:     node.ID,
			Type:       job.Template.Type,
			Command:    job.Template.Command,
			Script:     job.Template.Script,
			Parameters: job.Template.Parameters,
			Status:     core.TaskStatusPending,
			Priority:   job.Template.Priority,
			Timeout:    job.Template.Timeout,
			MaxRetries: job.Template.MaxRetries,
			CreatedAt:  time.Now(),
		}

		if err := s.taskRepo.Create(ctx, task); err != nil {
			s.logger.Error("Failed to create task",
				zap.String("job_id", job.ID),
				zap.String("node_id", node.ID),
				zap.Error(err),
			)
			continue
		}

		tasks = append(tasks, task)
	}

	// Update job status
	now := time.Now()
	job.Status = core.TaskStatusRunning
	job.StartedAt = &now
	job.TotalTasks = len(tasks)

	if err := s.jobRepo.Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	s.logger.Info("Job scheduled",
		zap.String("job_id", job.ID),
		zap.Int("tasks", len(tasks)),
	)

	return nil
}

// processPendingTasks processes tasks in pending status
func (s *Scheduler) processPendingTasks(ctx context.Context) error {
	// Get pending tasks
	tasks, err := s.taskRepo.List(ctx, &core.TaskFilter{
		Status: []core.TaskStatus{core.TaskStatusPending},
		Limit:  50,
	})
	if err != nil {
		return fmt.Errorf("failed to list pending tasks: %w", err)
	}

	for _, task := range tasks {
		// Execute task in background
		go func(t *core.Task) {
			if err := s.executeTask(context.Background(), t); err != nil {
				s.logger.Error("Failed to execute task",
					zap.String("task_id", t.ID),
					zap.Error(err),
				)
			}
		}(task)
	}

	return nil
}

// executeTask executes a single task
func (s *Scheduler) executeTask(ctx context.Context, task *core.Task) error {
	s.logger.Info("Executing task",
		zap.String("task_id", task.ID),
		zap.String("node_id", task.NodeID),
		zap.String("type", string(task.Type)),
	)

	// Update task status to running
	now := time.Now()
	task.Status = core.TaskStatusRunning
	task.StartedAt = &now
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Get node
	node, err := s.nodeRepo.Get(ctx, task.NodeID)
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	// Select executor based on node mode
	executor := s.executors[node.Mode]
	if executor == nil {
		// Fallback to SSH executor
		executor = s.executors[core.NodeModeSSH]
	}

	// Execute task
	result, err := executor.Execute(ctx, node, task)
	if err != nil {
		s.logger.Error("Task execution failed",
			zap.String("task_id", task.ID),
			zap.Error(err),
		)
		result = &core.TaskResult{
			Success:  false,
			ExitCode: -1,
			Error:    err.Error(),
		}
	}

	// Update task with result
	completed := time.Now()
	task.CompletedAt = &completed
	task.Result = result

	if result.Success {
		task.Status = core.TaskStatusCompleted
	} else {
		if task.Retries < task.MaxRetries {
			task.Retries++
			task.Status = core.TaskStatusPending // Retry
			s.logger.Info("Task will be retried",
				zap.String("task_id", task.ID),
				zap.Int("retry", task.Retries),
			)
		} else {
			task.Status = core.TaskStatusFailed
		}
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task result: %w", err)
	}

	// Update job progress
	if task.JobID != "" {
		s.updateJobProgress(ctx, task.JobID)
	}

	s.logger.Info("Task completed",
		zap.String("task_id", task.ID),
		zap.Bool("success", result.Success),
		zap.Duration("duration", result.Duration),
	)

	return nil
}

// updateJobProgress updates job completion statistics
func (s *Scheduler) updateJobProgress(ctx context.Context, jobID string) {
	counts, err := s.taskRepo.CountByStatus(ctx, jobID)
	if err != nil {
		s.logger.Error("Failed to count tasks", zap.String("job_id", jobID), zap.Error(err))
		return
	}

	completed := counts[core.TaskStatusCompleted]
	failed := counts[core.TaskStatusFailed]

	if err := s.jobRepo.UpdateProgress(ctx, jobID, completed, failed); err != nil {
		s.logger.Error("Failed to update job progress", zap.String("job_id", jobID), zap.Error(err))
		return
	}

	// Check if job is complete
	job, err := s.jobRepo.Get(ctx, jobID)
	if err != nil {
		return
	}

	pending := counts[core.TaskStatusPending]
	running := counts[core.TaskStatusRunning]

	if pending == 0 && running == 0 {
		// All tasks are done
		now := time.Now()
		if failed > 0 {
			job.Status = core.TaskStatusFailed
		} else {
			job.Status = core.TaskStatusCompleted
		}
		job.CompletedAt = &now
		s.jobRepo.Update(ctx, job)

		s.logger.Info("Job completed",
			zap.String("job_id", jobID),
			zap.Int("completed", completed),
			zap.Int("failed", failed),
		)
	}
}

// Schedule schedules a job for execution
func (s *Scheduler) Schedule(ctx context.Context, job *core.Job) error {
	// Create job in database
	if err := s.jobRepo.Create(ctx, job); err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	s.logger.Info("Job created", zap.String("job_id", job.ID))
	return nil
}

// Cancel cancels a running job
func (s *Scheduler) Cancel(ctx context.Context, jobID string) error {
	// Update job status
	if err := s.jobRepo.UpdateStatus(ctx, jobID, core.TaskStatusCancelled); err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	// Cancel pending tasks
	tasks, err := s.taskRepo.GetByJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job tasks: %w", err)
	}

	for _, task := range tasks {
		if task.Status == core.TaskStatusPending || task.Status == core.TaskStatusRunning {
			task.Status = core.TaskStatusCancelled
			s.taskRepo.Update(ctx, task)
		}
	}

	s.logger.Info("Job cancelled", zap.String("job_id", jobID))
	return nil
}

// Retry retries failed tasks in a job
func (s *Scheduler) Retry(ctx context.Context, jobID string) error {
	tasks, err := s.taskRepo.List(ctx, &core.TaskFilter{
		JobID:  jobID,
		Status: []core.TaskStatus{core.TaskStatusFailed},
	})
	if err != nil {
		return fmt.Errorf("failed to get failed tasks: %w", err)
	}

	for _, task := range tasks {
		task.Status = core.TaskStatusPending
		task.Retries = 0
		if err := s.taskRepo.Update(ctx, task); err != nil {
			s.logger.Error("Failed to retry task", zap.String("task_id", task.ID), zap.Error(err))
		}
	}

	s.logger.Info("Job tasks retried", zap.String("job_id", jobID), zap.Int("count", len(tasks)))
	return nil
}

// GetStatus gets the current status of a job
func (s *Scheduler) GetStatus(ctx context.Context, jobID string) (*core.JobStatus, error) {
	job, err := s.jobRepo.Get(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	tasks, err := s.taskRepo.GetByJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	counts, _ := s.taskRepo.CountByStatus(ctx, jobID)

	progress := float64(0)
	if job.TotalTasks > 0 {
		progress = float64(job.Completed+job.Failed) / float64(job.TotalTasks) * 100
	}

	return &core.JobStatus{
		Job:      job,
		Tasks:    tasks,
		Progress: progress,
		ByStatus: counts,
	}, nil
}
