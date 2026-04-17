package script

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/reignx/reignx/pkg/core"
	"github.com/reignx/reignx/pkg/executor"
)

// ScriptExecutor executes shell scripts
type ScriptExecutor struct {
	*executor.BaseExecutor
	tempDir string
}

// ScriptParams defines the parameters for script execution
type ScriptParams struct {
	Script      string            `json:"script"`       // Script content
	Interpreter string            `json:"interpreter"`  // bash, sh, python, etc.
	Arguments   []string          `json:"arguments"`    // Script arguments
	Environment map[string]string `json:"environment"`  // Environment variables
	WorkingDir  string            `json:"working_dir"`  // Working directory
}

// NewScriptExecutor creates a new script executor
func NewScriptExecutor(tempDir string) *ScriptExecutor {
	if tempDir == "" {
		tempDir = "/tmp/reignx-scripts"
	}
	return &ScriptExecutor{
		BaseExecutor: executor.NewBaseExecutor(core.TaskTypeScript),
		tempDir:      tempDir,
	}
}

// Validate checks if the script parameters are valid
func (e *ScriptExecutor) Validate(task *core.Task) error {
	var params ScriptParams
	if err := executor.ParseParameters(task.Parameters, &params); err != nil {
		return fmt.Errorf("invalid script parameters: %w", err)
	}

	if params.Script == "" {
		return fmt.Errorf("script content is required")
	}

	if params.Interpreter == "" {
		return fmt.Errorf("interpreter is required")
	}

	return nil
}

// Execute runs the script
func (e *ScriptExecutor) Execute(ctx context.Context, task *core.Task) (*core.TaskResult, error) {
	startedAt := time.Now()

	var params ScriptParams
	if err := executor.ParseParameters(task.Parameters, &params); err != nil {
		return executor.CreateFailureResult(1, "", "", err, time.Since(startedAt)), err
	}

	// Create temp directory if it doesn't exist
	if err := os.MkdirAll(e.tempDir, 0755); err != nil {
		return executor.CreateFailureResult(1, "", "", err, time.Since(startedAt)), err
	}

	// Write script to temporary file
	scriptPath := filepath.Join(e.tempDir, fmt.Sprintf("script-%s.sh", task.ID))
	if err := os.WriteFile(scriptPath, []byte(params.Script), 0755); err != nil {
		return executor.CreateFailureResult(1, "", "", err, time.Since(startedAt)), err
	}
	defer os.Remove(scriptPath)

	// Create command with timeout context
	timeoutCtx := ctx
	if task.Timeout > 0 {
		var cancel context.CancelFunc
		timeoutCtx, cancel = context.WithTimeout(ctx, task.Timeout)
		defer cancel()
	}

	// Build command
	args := append([]string{scriptPath}, params.Arguments...)
	cmd := exec.CommandContext(timeoutCtx, params.Interpreter, args...)

	// Set working directory
	if params.WorkingDir != "" {
		cmd.Dir = params.WorkingDir
	}

	// Set environment variables
	if len(params.Environment) > 0 {
		cmd.Env = os.Environ()
		for key, value := range params.Environment {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	duration := time.Since(startedAt)

	if err != nil {
		exitCode := 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return executor.CreateFailureResult(exitCode, string(output), "", err, duration), nil
	}

	return executor.CreateSuccessResult(string(output), duration), nil
}
