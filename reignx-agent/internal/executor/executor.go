package executor

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/reignx/reignx/pkg/core"
	pkgexec "github.com/reignx/reignx/pkg/executor"
	"github.com/reignx/reignx/pkg/executor/file"
	pkgpackage "github.com/reignx/reignx/pkg/executor/package"
	"github.com/reignx/reignx/pkg/executor/patch"
	"github.com/reignx/reignx/pkg/executor/script"
	"github.com/reignx/reignx/reignx-agent/internal/cache"
	"go.uber.org/zap"
)

// Config contains executor configuration
type Config struct {
	MaxConcurrency int
	Cache          *cache.Cache
	Logger         *zap.Logger
}

// Task represents a task to be executed
type Task struct {
	ID         string
	Type       string // command, script, file, etc.
	Command    string
	Script     string
	Parameters map[string]interface{}
	Timeout    time.Duration
	Priority   int
	Signature  string // HMAC signature for validation
}

// TaskResult represents the result of task execution
type TaskResult struct {
	TaskID    string
	Status    string // completed, failed, skipped
	ExitCode  int
	Output    string
	Error     string
	StartedAt time.Time
	Duration  time.Duration
}

// Executor executes tasks on the local system
type Executor struct {
	config   *Config
	logger   *zap.Logger
	cache    *cache.Cache
	registry *pkgexec.Registry // Plugin executor registry
	sem      chan struct{}     // Semaphore for concurrency control
	mu       sync.RWMutex
	running  map[string]context.CancelFunc // Running tasks
}

// New creates a new executor
func New(config *Config) *Executor {
	// Create executor registry and register plugins
	registry := pkgexec.NewRegistry()
	registry.Register(script.NewScriptExecutor("/tmp/reignx-scripts"))
	registry.Register(patch.NewPatchExecutor())
	registry.Register(pkgpackage.NewPackageExecutor())
	registry.Register(file.NewFileExecutor())

	return &Executor{
		config:   config,
		logger:   config.Logger,
		cache:    config.Cache,
		registry: registry,
		sem:      make(chan struct{}, config.MaxConcurrency),
		running:  make(map[string]context.CancelFunc),
	}
}

// Execute executes a task
func (e *Executor) Execute(ctx context.Context, task *Task) *TaskResult {
	result := &TaskResult{
		TaskID:    task.ID,
		StartedAt: time.Now(),
	}

	// Validate task signature
	if !e.validateTask(task) {
		result.Status = "failed"
		result.Error = "invalid task signature"
		result.ExitCode = -1
		return result
	}

	// Check if task already completed (idempotency)
	fingerprint := e.taskFingerprint(task)
	if e.cache.IsTaskCompleted(fingerprint) {
		e.logger.Info("Task already completed, skipping",
			zap.String("task_id", task.ID),
		)
		result.Status = "skipped"
		result.ExitCode = 0
		result.Output = "Task already executed successfully"
		return result
	}

	// Acquire semaphore for concurrency control
	select {
	case e.sem <- struct{}{}:
		defer func() { <-e.sem }()
	case <-ctx.Done():
		result.Status = "failed"
		result.Error = "context cancelled while waiting for execution slot"
		result.ExitCode = -1
		return result
	}

	// Create task context with timeout
	taskCtx := ctx
	var cancel context.CancelFunc
	if task.Timeout > 0 {
		taskCtx, cancel = context.WithTimeout(ctx, task.Timeout)
		defer cancel()
	}

	// Register running task
	e.mu.Lock()
	e.running[task.ID] = cancel
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.running, task.ID)
		e.mu.Unlock()
	}()

	// Execute based on task type - use plugin registry if available
	taskType := core.TaskType(task.Type)
	if executor, err := e.registry.Get(taskType); err == nil {
		// Use plugin executor
		coreTask := e.convertToCoreTask(task)
		pluginResult, execErr := executor.Execute(taskCtx, coreTask)
		if execErr != nil {
			e.logger.Error("Plugin execution error",
				zap.String("task_id", task.ID),
				zap.Error(execErr),
			)
		}
		if pluginResult != nil {
			if pluginResult.Success {
				result.Status = "completed"
			} else {
				result.Status = "failed"
			}
			result.ExitCode = pluginResult.ExitCode
			result.Output = pluginResult.Stdout
			if pluginResult.Error != "" {
				result.Error = pluginResult.Error
			} else if pluginResult.Stderr != "" {
				result.Error = pluginResult.Stderr
			}
		}
	} else {
		// Fallback to legacy execution methods
		switch task.Type {
		case "command":
			e.executeCommand(taskCtx, task, result)
		case "script":
			e.executeScript(taskCtx, task, result)
		case "file":
			e.executeFileOp(taskCtx, task, result)
		default:
			result.Status = "failed"
			result.Error = fmt.Sprintf("unsupported task type: %s", task.Type)
			result.ExitCode = -1
		}
	}

	// Calculate duration
	result.Duration = time.Since(result.StartedAt)

	// Cache successful execution
	if result.Status == "completed" {
		e.cache.MarkTaskCompleted(fingerprint)
	}

	e.logger.Info("Task executed",
		zap.String("task_id", task.ID),
		zap.String("type", task.Type),
		zap.String("status", result.Status),
		zap.Int("exit_code", result.ExitCode),
		zap.Duration("duration", result.Duration),
	)

	return result
}

// executeCommand executes a shell command
func (e *Executor) executeCommand(ctx context.Context, task *Task, result *TaskResult) {
	e.logger.Debug("Executing command",
		zap.String("task_id", task.ID),
		zap.String("command", task.Command),
	)

	cmd := exec.CommandContext(ctx, "sh", "-c", task.Command)
	output, err := cmd.CombinedOutput()

	result.Output = string(output)

	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	} else {
		result.Status = "completed"
		result.ExitCode = 0
	}
}

// executeScript executes a script
func (e *Executor) executeScript(ctx context.Context, task *Task, result *TaskResult) {
	e.logger.Debug("Executing script",
		zap.String("task_id", task.ID),
	)

	// Write script to temporary file
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("reignx-script-%s-*.sh", task.ID))
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to create temp file: %v", err)
		result.ExitCode = -1
		return
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(task.Script); err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to write script: %v", err)
		result.ExitCode = -1
		return
	}

	if err := tmpFile.Close(); err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to close temp file: %v", err)
		result.ExitCode = -1
		return
	}

	// Make script executable
	if err := os.Chmod(tmpFile.Name(), 0700); err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to chmod script: %v", err)
		result.ExitCode = -1
		return
	}

	// Execute script
	cmd := exec.CommandContext(ctx, "sh", tmpFile.Name())
	output, err := cmd.CombinedOutput()

	result.Output = string(output)

	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	} else {
		result.Status = "completed"
		result.ExitCode = 0
	}
}

// executeFileOp executes a file operation
func (e *Executor) executeFileOp(ctx context.Context, task *Task, result *TaskResult) {
	e.logger.Debug("Executing file operation",
		zap.String("task_id", task.ID),
	)

	// Get operation type from parameters
	op, ok := task.Parameters["operation"].(string)
	if !ok {
		result.Status = "failed"
		result.Error = "operation parameter is required"
		result.ExitCode = -1
		return
	}

	switch op {
	case "copy":
		e.fileCopy(task, result)
	case "move":
		e.fileMove(task, result)
	case "delete":
		e.fileDelete(task, result)
	case "mkdir":
		e.filemkdir(task, result)
	default:
		result.Status = "failed"
		result.Error = fmt.Sprintf("unsupported file operation: %s", op)
		result.ExitCode = -1
	}
}

// fileCopy copies a file
func (e *Executor) fileCopy(task *Task, result *TaskResult) {
	src, _ := task.Parameters["source"].(string)
	dst, _ := task.Parameters["destination"].(string)

	if src == "" || dst == "" {
		result.Status = "failed"
		result.Error = "source and destination are required"
		result.ExitCode = -1
		return
	}

	data, err := os.ReadFile(src)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to read source: %v", err)
		result.ExitCode = -1
		return
	}

	if err := os.WriteFile(dst, data, 0644); err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to write destination: %v", err)
		result.ExitCode = -1
		return
	}

	result.Status = "completed"
	result.ExitCode = 0
	result.Output = fmt.Sprintf("Copied %s to %s", src, dst)
}

// fileMove moves a file
func (e *Executor) fileMove(task *Task, result *TaskResult) {
	src, _ := task.Parameters["source"].(string)
	dst, _ := task.Parameters["destination"].(string)

	if src == "" || dst == "" {
		result.Status = "failed"
		result.Error = "source and destination are required"
		result.ExitCode = -1
		return
	}

	if err := os.Rename(src, dst); err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to move file: %v", err)
		result.ExitCode = -1
		return
	}

	result.Status = "completed"
	result.ExitCode = 0
	result.Output = fmt.Sprintf("Moved %s to %s", src, dst)
}

// fileDelete deletes a file
func (e *Executor) fileDelete(task *Task, result *TaskResult) {
	path, _ := task.Parameters["path"].(string)

	if path == "" {
		result.Status = "failed"
		result.Error = "path is required"
		result.ExitCode = -1
		return
	}

	if err := os.Remove(path); err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to delete file: %v", err)
		result.ExitCode = -1
		return
	}

	result.Status = "completed"
	result.ExitCode = 0
	result.Output = fmt.Sprintf("Deleted %s", path)
}

// filemkdir creates a directory
func (e *Executor) filemkdir(task *Task, result *TaskResult) {
	path, _ := task.Parameters["path"].(string)
	mode, _ := task.Parameters["mode"].(int)

	if path == "" {
		result.Status = "failed"
		result.Error = "path is required"
		result.ExitCode = -1
		return
	}

	if mode == 0 {
		mode = 0755
	}

	if err := os.MkdirAll(path, os.FileMode(mode)); err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to create directory: %v", err)
		result.ExitCode = -1
		return
	}

	result.Status = "completed"
	result.ExitCode = 0
	result.Output = fmt.Sprintf("Created directory %s", path)
}

// Cancel cancels a running task
func (e *Executor) Cancel(taskID string) error {
	e.mu.RLock()
	cancel, exists := e.running[taskID]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task not found or not running")
	}

	cancel()
	e.logger.Info("Task cancelled", zap.String("task_id", taskID))

	return nil
}

// validateTask validates the task signature
func (e *Executor) validateTask(task *Task) bool {
	// TODO: Implement HMAC signature validation
	// For now, accept all tasks
	return true
}

// taskFingerprint generates a unique fingerprint for task idempotency
func (e *Executor) taskFingerprint(task *Task) string {
	// Create fingerprint based on task type and content
	data := fmt.Sprintf("%s:%s:%s", task.Type, task.Command, task.Script)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// GetRunningTasks returns the number of currently running tasks
func (e *Executor) GetRunningTasks() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.running)
}

// GetAvailableSlots returns the number of available execution slots
func (e *Executor) GetAvailableSlots() int {
	return e.config.MaxConcurrency - e.GetRunningTasks()
}

// IsTaskRunning checks if a task is currently running
func (e *Executor) IsTaskRunning(taskID string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, exists := e.running[taskID]
	return exists
}

// GetTaskInfo returns information about a running task
func (e *Executor) GetTaskInfo(taskID string) map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if _, exists := e.running[taskID]; exists {
		return map[string]interface{}{
			"task_id": taskID,
			"status":  "running",
		}
	}

	return map[string]interface{}{
		"task_id": taskID,
		"status":  "not_found",
	}
}

// convertToCoreTask converts the local Task type to core.Task for plugin execution
func (e *Executor) convertToCoreTask(task *Task) *core.Task {
	return &core.Task{
		ID:         task.ID,
		Type:       core.TaskType(task.Type),
		Command:    task.Command,
		Script:     task.Script,
		Parameters: task.Parameters, // Already map[string]interface{}
		Timeout:    task.Timeout,
		Priority:   task.Priority,
	}
}
