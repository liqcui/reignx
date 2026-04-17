package file

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/reignx/reignx/pkg/core"
	"github.com/reignx/reignx/pkg/executor"
)

// FileExecutor handles file operations
type FileExecutor struct {
	*executor.BaseExecutor
}

// FileParams defines the parameters for file operations
type FileParams struct {
	Action      string `json:"action"`       // copy, move, delete, download, upload
	Source      string `json:"source"`       // Source path or URL
	Destination string `json:"destination"`  // Destination path
	Content     string `json:"content"`      // Content for create action
	Mode        string `json:"mode"`         // File permissions (e.g., "0644")
	Owner       string `json:"owner"`        // File owner (user:group)
	CreateDirs  bool   `json:"create_dirs"`  // Create parent directories
	Overwrite   bool   `json:"overwrite"`    // Overwrite existing files
}

// NewFileExecutor creates a new file executor
func NewFileExecutor() *FileExecutor {
	return &FileExecutor{
		BaseExecutor: executor.NewBaseExecutor(core.TaskTypeFile),
	}
}

// Validate checks if the file parameters are valid
func (e *FileExecutor) Validate(task *core.Task) error {
	var params FileParams
	if err := executor.ParseParameters(task.Parameters, &params); err != nil {
		return fmt.Errorf("invalid file parameters: %w", err)
	}

	if params.Action == "" {
		return fmt.Errorf("action is required (copy, move, delete, download, create)")
	}

	validActions := map[string]bool{
		"copy":     true,
		"move":     true,
		"delete":   true,
		"download": true,
		"create":   true,
	}
	if !validActions[params.Action] {
		return fmt.Errorf("invalid action: %s", params.Action)
	}

	// Validate action-specific requirements
	switch params.Action {
	case "copy", "move":
		if params.Source == "" || params.Destination == "" {
			return fmt.Errorf("%s action requires both source and destination", params.Action)
		}
	case "delete":
		if params.Destination == "" {
			return fmt.Errorf("delete action requires destination path")
		}
	case "download":
		if params.Source == "" || params.Destination == "" {
			return fmt.Errorf("download action requires both source (URL) and destination")
		}
	case "create":
		if params.Destination == "" || params.Content == "" {
			return fmt.Errorf("create action requires both destination and content")
		}
	}

	return nil
}

// Execute runs the file operation
func (e *FileExecutor) Execute(ctx context.Context, task *core.Task) (*core.TaskResult, error) {
	startedAt := time.Now()

	var params FileParams
	if err := executor.ParseParameters(task.Parameters, &params); err != nil {
		return executor.CreateFailureResult(1, "", "", err, time.Since(startedAt)), err
	}

	var output string
	var err error

	switch params.Action {
	case "copy":
		output, err = e.copyFile(params)
	case "move":
		output, err = e.moveFile(params)
	case "delete":
		output, err = e.deleteFile(params)
	case "download":
		output, err = e.downloadFile(ctx, params)
	case "create":
		output, err = e.createFile(params)
	default:
		err = fmt.Errorf("unsupported action: %s", params.Action)
	}

	duration := time.Since(startedAt)

	if err != nil {
		return executor.CreateFailureResult(1, output, "", err, duration), nil
	}

	return executor.CreateSuccessResult(output, duration), nil
}

// copyFile copies a file from source to destination
func (e *FileExecutor) copyFile(params FileParams) (string, error) {
	// Check if destination exists
	if _, err := os.Stat(params.Destination); err == nil && !params.Overwrite {
		return "", fmt.Errorf("destination file already exists: %s", params.Destination)
	}

	// Create parent directories if needed
	if params.CreateDirs {
		dir := filepath.Dir(params.Destination)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create parent directories: %w", err)
		}
	}

	// Open source file
	srcFile, err := os.Open(params.Source)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(params.Destination)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy content
	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	// Set permissions if specified
	if params.Mode != "" {
		mode, err := parseFileMode(params.Mode)
		if err == nil {
			os.Chmod(params.Destination, mode)
		}
	}

	return fmt.Sprintf("Copied %d bytes from %s to %s", written, params.Source, params.Destination), nil
}

// moveFile moves a file from source to destination
func (e *FileExecutor) moveFile(params FileParams) (string, error) {
	// Check if destination exists
	if _, err := os.Stat(params.Destination); err == nil && !params.Overwrite {
		return "", fmt.Errorf("destination file already exists: %s", params.Destination)
	}

	// Create parent directories if needed
	if params.CreateDirs {
		dir := filepath.Dir(params.Destination)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create parent directories: %w", err)
		}
	}

	// Move file
	if err := os.Rename(params.Source, params.Destination); err != nil {
		return "", fmt.Errorf("failed to move file: %w", err)
	}

	// Set permissions if specified
	if params.Mode != "" {
		mode, err := parseFileMode(params.Mode)
		if err == nil {
			os.Chmod(params.Destination, mode)
		}
	}

	return fmt.Sprintf("Moved %s to %s", params.Source, params.Destination), nil
}

// deleteFile deletes a file
func (e *FileExecutor) deleteFile(params FileParams) (string, error) {
	// Check if file exists
	if _, err := os.Stat(params.Destination); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", params.Destination)
	}

	// Delete file
	if err := os.Remove(params.Destination); err != nil {
		return "", fmt.Errorf("failed to delete file: %w", err)
	}

	return fmt.Sprintf("Deleted %s", params.Destination), nil
}

// downloadFile downloads a file from a URL
func (e *FileExecutor) downloadFile(ctx context.Context, params FileParams) (string, error) {
	// Check if destination exists
	if _, err := os.Stat(params.Destination); err == nil && !params.Overwrite {
		return "", fmt.Errorf("destination file already exists: %s", params.Destination)
	}

	// Create parent directories if needed
	if params.CreateDirs {
		dir := filepath.Dir(params.Destination)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create parent directories: %w", err)
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", params.Source, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Download file
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Create destination file
	dstFile, err := os.Create(params.Destination)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy content
	written, err := io.Copy(dstFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Set permissions if specified
	if params.Mode != "" {
		mode, err := parseFileMode(params.Mode)
		if err == nil {
			os.Chmod(params.Destination, mode)
		}
	}

	return fmt.Sprintf("Downloaded %d bytes from %s to %s", written, params.Source, params.Destination), nil
}

// createFile creates a new file with content
func (e *FileExecutor) createFile(params FileParams) (string, error) {
	// Check if destination exists
	if _, err := os.Stat(params.Destination); err == nil && !params.Overwrite {
		return "", fmt.Errorf("destination file already exists: %s", params.Destination)
	}

	// Create parent directories if needed
	if params.CreateDirs {
		dir := filepath.Dir(params.Destination)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create parent directories: %w", err)
		}
	}

	// Write content to file
	mode := os.FileMode(0644)
	if params.Mode != "" {
		if m, err := parseFileMode(params.Mode); err == nil {
			mode = m
		}
	}

	if err := os.WriteFile(params.Destination, []byte(params.Content), mode); err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	return fmt.Sprintf("Created %s with %d bytes", params.Destination, len(params.Content)), nil
}

// parseFileMode parses a file mode string (e.g., "0644") to os.FileMode
func parseFileMode(modeStr string) (os.FileMode, error) {
	var mode uint32
	_, err := fmt.Sscanf(modeStr, "%o", &mode)
	if err != nil {
		return 0, fmt.Errorf("invalid file mode: %s", modeStr)
	}
	return os.FileMode(mode), nil
}
