package core

import (
	"context"
	"time"
)

// TaskType represents the type of task to execute
type TaskType string

const (
	TaskTypeCommand    TaskType = "command"      // Execute shell command
	TaskTypeScript     TaskType = "script"       // Execute script
	TaskTypeFile       TaskType = "file"         // File operations
	TaskTypeFirmware   TaskType = "firmware"     // Firmware update
	TaskTypeOSInstall  TaskType = "os_install"   // OS installation
	TaskTypeOSUpgrade  TaskType = "os_upgrade"   // OS upgrade
	TaskTypePackage    TaskType = "package"      // Package management
	TaskTypePatch      TaskType = "patch"        // System patching
	TaskTypeReboot     TaskType = "reboot"       // System reboot
	TaskTypeHealthCheck TaskType = "health_check" // Health check
)

// TaskStatus represents the execution status of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusAssigned  TaskStatus = "assigned"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
	TaskStatusTimeout   TaskStatus = "timeout"
)

// Task represents a unit of work to be executed on a node
type Task struct {
	ID          string                 `json:"id"`
	JobID       string                 `json:"job_id,omitempty"`
	NodeID      string                 `json:"node_id"`
	Type        TaskType               `json:"type"`
	Command     string                 `json:"command,omitempty"`
	Script      string                 `json:"script,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
	Status      TaskStatus             `json:"status"`
	Priority    int                    `json:"priority"` // 0=low, 1=normal, 2=high, 3=critical
	Timeout     time.Duration          `json:"timeout"`
	Retries     int                    `json:"retries"`
	MaxRetries  int                    `json:"max_retries"`
	Result      *TaskResult            `json:"result,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// TaskResult contains the result of task execution
type TaskResult struct {
	Success      bool              `json:"success"`
	ExitCode     int               `json:"exit_code"`
	Stdout       string            `json:"stdout"`
	Stderr       string            `json:"stderr"`
	Error        string            `json:"error,omitempty"`
	Duration     time.Duration     `json:"duration"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Job represents a batch of tasks across multiple nodes
type Job struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        TaskType               `json:"type"`
	Mode        NodeMode               `json:"mode"` // ssh, agent, or auto
	Filter      *NodeFilter            `json:"filter"`
	Template    *Task                  `json:"template"`
	Status      TaskStatus             `json:"status"`
	BatchSize   int                    `json:"batch_size"`   // Parallel execution batch size
	Concurrency int                    `json:"concurrency"`  // Max concurrent tasks
	TotalTasks  int                    `json:"total_tasks"`
	Completed   int                    `json:"completed"`
	Failed      int                    `json:"failed"`
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Executor defines the interface for executing tasks on nodes
type Executor interface {
	// Execute runs a single task on a node
	Execute(ctx context.Context, node *Node, task *Task) (*TaskResult, error)

	// ExecuteBatch runs tasks in batch across multiple nodes
	ExecuteBatch(ctx context.Context, nodes []*Node, task *Task, concurrency int) (map[string]*TaskResult, error)

	// SupportsMode indicates if this executor supports the given mode
	SupportsMode(mode NodeMode) bool

	// Name returns the executor name
	Name() string
}

// TaskRepository defines the interface for task persistence
type TaskRepository interface {
	// Create creates a new task
	Create(ctx context.Context, task *Task) error

	// Get retrieves a task by ID
	Get(ctx context.Context, id string) (*Task, error)

	// List retrieves tasks matching criteria
	List(ctx context.Context, filter *TaskFilter) ([]*Task, error)

	// Update updates a task
	Update(ctx context.Context, task *Task) error

	// UpdateStatus updates task status
	UpdateStatus(ctx context.Context, id string, status TaskStatus) error

	// UpdateResult updates task result
	UpdateResult(ctx context.Context, id string, result *TaskResult) error

	// Delete deletes a task
	Delete(ctx context.Context, id string) error

	// GetByJob retrieves all tasks for a job
	GetByJob(ctx context.Context, jobID string) ([]*Task, error)

	// GetByNode retrieves all tasks for a node
	GetByNode(ctx context.Context, nodeID string, limit int) ([]*Task, error)

	// CountByStatus counts tasks by status
	CountByStatus(ctx context.Context, jobID string) (map[TaskStatus]int, error)

	// GetPendingTasks retrieves pending tasks for a specific node
	GetPendingTasks(ctx context.Context, nodeID string, limit int) ([]*Task, error)
}

// TaskFilter defines criteria for filtering tasks
type TaskFilter struct {
	JobID     string       `json:"job_id,omitempty"`
	NodeID    string       `json:"node_id,omitempty"`
	Status    []TaskStatus `json:"status,omitempty"`
	Type      TaskType     `json:"type,omitempty"`
	CreatedFrom *time.Time `json:"created_from,omitempty"`
	CreatedTo   *time.Time `json:"created_to,omitempty"`
	Limit     int          `json:"limit,omitempty"`
	Offset    int          `json:"offset,omitempty"`
}

// JobRepository defines the interface for job persistence
type JobRepository interface {
	// Create creates a new job
	Create(ctx context.Context, job *Job) error

	// Get retrieves a job by ID
	Get(ctx context.Context, id string) (*Job, error)

	// List retrieves jobs matching criteria
	List(ctx context.Context, filter *JobFilter) ([]*Job, error)

	// Update updates a job
	Update(ctx context.Context, job *Job) error

	// UpdateStatus updates job status
	UpdateStatus(ctx context.Context, id string, status TaskStatus) error

	// Delete deletes a job
	Delete(ctx context.Context, id string) error

	// UpdateProgress updates job progress counters
	UpdateProgress(ctx context.Context, id string, completed, failed int) error
}

// JobFilter defines criteria for filtering jobs
type JobFilter struct {
	Status      []TaskStatus `json:"status,omitempty"`
	Type        TaskType     `json:"type,omitempty"`
	CreatedBy   string       `json:"created_by,omitempty"`
	CreatedFrom *time.Time   `json:"created_from,omitempty"`
	CreatedTo   *time.Time   `json:"created_to,omitempty"`
	Limit       int          `json:"limit,omitempty"`
	Offset      int          `json:"offset,omitempty"`
}

// Scheduler defines the interface for task scheduling
type Scheduler interface {
	// Schedule schedules a job for execution
	Schedule(ctx context.Context, job *Job) error

	// Cancel cancels a running job
	Cancel(ctx context.Context, jobID string) error

	// Retry retries failed tasks in a job
	Retry(ctx context.Context, jobID string) error

	// GetStatus gets the current status of a job
	GetStatus(ctx context.Context, jobID string) (*JobStatus, error)
}

// JobStatus contains the current status of a job
type JobStatus struct {
	Job       *Job              `json:"job"`
	Tasks     []*Task           `json:"tasks"`
	Progress  float64           `json:"progress"` // Percentage
	ETA       *time.Time        `json:"eta,omitempty"`
	ByStatus  map[TaskStatus]int `json:"by_status"`
}
