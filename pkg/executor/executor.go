package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/reignx/reignx/pkg/core"
)

// Executor defines the interface for task execution plugins
type Executor interface {
	// Type returns the task type this executor handles
	Type() core.TaskType

	// Execute runs the task and returns the result
	Execute(ctx context.Context, task *core.Task) (*core.TaskResult, error)

	// Validate checks if the task parameters are valid
	Validate(task *core.Task) error
}

// Registry manages available task executors
type Registry struct {
	executors map[core.TaskType]Executor
}

// NewRegistry creates a new executor registry
func NewRegistry() *Registry {
	return &Registry{
		executors: make(map[core.TaskType]Executor),
	}
}

// Register adds an executor to the registry
func (r *Registry) Register(executor Executor) {
	r.executors[executor.Type()] = executor
}

// Get retrieves an executor for the given task type
func (r *Registry) Get(taskType core.TaskType) (Executor, error) {
	executor, ok := r.executors[taskType]
	if !ok {
		return nil, fmt.Errorf("no executor registered for task type: %s", taskType)
	}
	return executor, nil
}

// Execute runs a task using the appropriate executor
func (r *Registry) Execute(ctx context.Context, task *core.Task) (*core.TaskResult, error) {
	executor, err := r.Get(task.Type)
	if err != nil {
		return &core.TaskResult{
			Success:  false,
			ExitCode: 1,
			Stderr:   "",
			Error:    err.Error(),
		}, err
	}

	// Validate task before execution
	if err := executor.Validate(task); err != nil {
		return &core.TaskResult{
			Success:  false,
			ExitCode: 1,
			Stderr:   "",
			Error:    fmt.Sprintf("task validation failed: %v", err),
		}, err
	}

	// Execute the task
	return executor.Execute(ctx, task)
}

// BaseExecutor provides common functionality for executors
type BaseExecutor struct {
	taskType core.TaskType
}

// NewBaseExecutor creates a new base executor
func NewBaseExecutor(taskType core.TaskType) *BaseExecutor {
	return &BaseExecutor{
		taskType: taskType,
	}
}

// Type returns the task type
func (e *BaseExecutor) Type() core.TaskType {
	return e.taskType
}

// ParseParameters converts task.Parameters map to a typed struct
func ParseParameters(params map[string]interface{}, target interface{}) error {
	// Marshal to JSON and then unmarshal to target struct
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal parameters: %w", err)
	}
	return nil
}

// CreateSuccessResult creates a successful task result
func CreateSuccessResult(stdout string, duration time.Duration) *core.TaskResult {
	return &core.TaskResult{
		Success:  true,
		ExitCode: 0,
		Stdout:   stdout,
		Stderr:   "",
		Error:    "",
		Duration: duration,
	}
}

// CreateFailureResult creates a failed task result
func CreateFailureResult(exitCode int, stdout string, stderr string, err error, duration time.Duration) *core.TaskResult {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	return &core.TaskResult{
		Success:  false,
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
		Error:    errorMsg,
		Duration: duration,
	}
}
