package executor_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/reignx/reignx/pkg/core"
	"github.com/reignx/reignx/pkg/executor"
	"github.com/reignx/reignx/pkg/executor/file"
	pkgpackage "github.com/reignx/reignx/pkg/executor/package"
	"github.com/reignx/reignx/pkg/executor/patch"
	"github.com/reignx/reignx/pkg/executor/script"
)

func TestScriptExecutor(t *testing.T) {
	registry := executor.NewRegistry()
	registry.Register(script.NewScriptExecutor("/tmp/reignx-test-scripts"))

	task := &core.Task{
		ID:   "test-script-1",
		Type: core.TaskTypeScript,
		Parameters: map[string]interface{}{
			"script":      "#!/bin/bash\necho 'Hello from script executor'\necho 'Task ID: test-script-1'",
			"interpreter": "bash",
			"arguments":   []string{},
		},
		Timeout: 10 * time.Second,
	}

	ctx := context.Background()
	result, err := registry.Execute(ctx, task)

	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if result.Stdout == "" {
		t.Error("Expected output, got empty string")
	}

	t.Logf("Script output: %s", result.Stdout)
	t.Logf("Duration: %v", result.Duration)
}

func TestScriptExecutorWithEnvironment(t *testing.T) {
	registry := executor.NewRegistry()
	registry.Register(script.NewScriptExecutor("/tmp/reignx-test-scripts"))

	task := &core.Task{
		ID:   "test-script-env",
		Type: core.TaskTypeScript,
		Parameters: map[string]interface{}{
			"script":      "#!/bin/bash\necho \"TEST_VAR=$TEST_VAR\"\necho \"CUSTOM_PATH=$CUSTOM_PATH\"",
			"interpreter": "bash",
			"environment": map[string]interface{}{
				"TEST_VAR":    "test_value_123",
				"CUSTOM_PATH": "/custom/path/to/bin",
			},
		},
		Timeout: 10 * time.Second,
	}

	ctx := context.Background()
	result, err := registry.Execute(ctx, task)

	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	t.Logf("Environment test output: %s", result.Stdout)
}

func TestFileExecutorCreate(t *testing.T) {
	registry := executor.NewRegistry()
	registry.Register(file.NewFileExecutor())

	// Create temp directory for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-file.txt")

	task := &core.Task{
		ID:   "test-file-create",
		Type: core.TaskTypeFile,
		Parameters: map[string]interface{}{
			"action":      "create",
			"destination": testFile,
			"content":     "Hello from file executor\nThis is a test file\n",
			"mode":        "0644",
		},
		Timeout: 5 * time.Second,
	}

	ctx := context.Background()
	result, err := registry.Execute(ctx, task)

	if err != nil {
		t.Fatalf("File creation failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	// Verify file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	expectedContent := "Hello from file executor\nThis is a test file\n"
	if string(content) != expectedContent {
		t.Errorf("File content mismatch.\nExpected: %q\nGot: %q", expectedContent, string(content))
	}

	t.Logf("File executor output: %s", result.Stdout)
}

func TestFileExecutorCopy(t *testing.T) {
	registry := executor.NewRegistry()
	registry.Register(file.NewFileExecutor())

	// Create temp directory and source file
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "source.txt")
	destFile := filepath.Join(tmpDir, "destination.txt")

	// Create source file
	if err := os.WriteFile(sourceFile, []byte("Test content for copy"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	task := &core.Task{
		ID:   "test-file-copy",
		Type: core.TaskTypeFile,
		Parameters: map[string]interface{}{
			"action":      "copy",
			"source":      sourceFile,
			"destination": destFile,
		},
		Timeout: 5 * time.Second,
	}

	ctx := context.Background()
	result, err := registry.Execute(ctx, task)

	if err != nil {
		t.Fatalf("File copy failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	// Verify destination file exists and has same content
	destContent, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(destContent) != "Test content for copy" {
		t.Error("Copied file content mismatch")
	}

	t.Logf("File copy output: %s", result.Stdout)
}

func TestFileExecutorDelete(t *testing.T) {
	registry := executor.NewRegistry()
	registry.Register(file.NewFileExecutor())

	// Create temp file to delete
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "to-delete.txt")
	if err := os.WriteFile(testFile, []byte("Delete me"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	task := &core.Task{
		ID:   "test-file-delete",
		Type: core.TaskTypeFile,
		Parameters: map[string]interface{}{
			"action":      "delete",
			"destination": testFile,
		},
		Timeout: 5 * time.Second,
	}

	ctx := context.Background()
	result, err := registry.Execute(ctx, task)

	if err != nil {
		t.Fatalf("File delete failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	// Verify file was deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File was not deleted")
	}

	t.Logf("File delete output: %s", result.Stdout)
}

func TestExecutorRegistry(t *testing.T) {
	registry := executor.NewRegistry()

	// Register all executors
	registry.Register(script.NewScriptExecutor("/tmp/reignx-test"))
	registry.Register(patch.NewPatchExecutor())
	registry.Register(pkgpackage.NewPackageExecutor())
	registry.Register(file.NewFileExecutor())

	// Test that all executors are registered
	testCases := []core.TaskType{
		core.TaskTypeScript,
		core.TaskTypePatch,
		core.TaskTypePackage,
		core.TaskTypeFile,
	}

	for _, taskType := range testCases {
		exec, err := registry.Get(taskType)
		if err != nil {
			t.Errorf("Failed to get executor for type %s: %v", taskType, err)
		}
		if exec == nil {
			t.Errorf("Executor is nil for type %s", taskType)
		}
		if exec.Type() != taskType {
			t.Errorf("Executor type mismatch: expected %s, got %s", taskType, exec.Type())
		}
	}

	// Test unknown executor type
	_, err := registry.Get(core.TaskType("unknown"))
	if err == nil {
		t.Error("Expected error for unknown task type, got nil")
	}
}

func TestScriptExecutorTimeout(t *testing.T) {
	registry := executor.NewRegistry()
	registry.Register(script.NewScriptExecutor("/tmp/reignx-test-scripts"))

	task := &core.Task{
		ID:   "test-script-timeout",
		Type: core.TaskTypeScript,
		Parameters: map[string]interface{}{
			"script":      "#!/bin/bash\nsleep 10\necho 'This should not print'",
			"interpreter": "bash",
		},
		Timeout: 1 * time.Second, // 1 second timeout for 10 second sleep
	}

	ctx := context.Background()
	result, err := registry.Execute(ctx, task)

	// Should complete but fail due to timeout
	if err == nil && result.Success {
		t.Error("Expected timeout failure, got success")
	}

	t.Logf("Timeout test result: Success=%v, ExitCode=%d, Error=%s",
		result.Success, result.ExitCode, result.Error)
}

func TestScriptExecutorValidation(t *testing.T) {
	registry := executor.NewRegistry()
	registry.Register(script.NewScriptExecutor("/tmp/reignx-test-scripts"))

	// Test missing script content
	task := &core.Task{
		ID:   "test-validation-1",
		Type: core.TaskTypeScript,
		Parameters: map[string]interface{}{
			"interpreter": "bash",
			// Missing "script" field
		},
	}

	ctx := context.Background()
	result, err := registry.Execute(ctx, task)

	if err == nil {
		t.Error("Expected validation error for missing script, got nil")
	}

	if result.Success {
		t.Error("Expected validation to fail, got success")
	}

	t.Logf("Validation error (expected): %s", result.Error)

	// Test missing interpreter
	task2 := &core.Task{
		ID:   "test-validation-2",
		Type: core.TaskTypeScript,
		Parameters: map[string]interface{}{
			"script": "echo hello",
			// Missing "interpreter" field
		},
	}

	result2, err2 := registry.Execute(ctx, task2)

	if err2 == nil {
		t.Error("Expected validation error for missing interpreter, got nil")
	}

	if result2.Success {
		t.Error("Expected validation to fail, got success")
	}

	t.Logf("Validation error (expected): %s", result2.Error)
}
