package patch

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/reignx/reignx/pkg/core"
	"github.com/reignx/reignx/pkg/executor"
)

// PatchExecutor handles OS patching
type PatchExecutor struct {
	*executor.BaseExecutor
}

// PatchParams defines the parameters for patch execution
type PatchParams struct {
	PatchIDs     []string `json:"patch_ids"`      // Specific patches to install (optional)
	UpdateAll    bool     `json:"update_all"`     // Update all available patches
	SecurityOnly bool     `json:"security_only"`  // Only security patches
	Reboot       bool     `json:"reboot"`         // Reboot after patching if required
	Force        bool     `json:"force"`          // Force installation
}

// NewPatchExecutor creates a new patch executor
func NewPatchExecutor() *PatchExecutor {
	return &PatchExecutor{
		BaseExecutor: executor.NewBaseExecutor(core.TaskTypePatch),
	}
}

// Validate checks if the patch parameters are valid
func (e *PatchExecutor) Validate(task *core.Task) error {
	var params PatchParams
	if err := executor.ParseParameters(task.Parameters, &params); err != nil {
		return fmt.Errorf("invalid patch parameters: %w", err)
	}

	if !params.UpdateAll && len(params.PatchIDs) == 0 {
		return fmt.Errorf("either update_all must be true or patch_ids must be provided")
	}

	return nil
}

// Execute runs the patching operation
func (e *PatchExecutor) Execute(ctx context.Context, task *core.Task) (*core.TaskResult, error) {
	startedAt := time.Now()

	var params PatchParams
	if err := executor.ParseParameters(task.Parameters, &params); err != nil {
		return executor.CreateFailureResult(1, "", "", err, time.Since(startedAt)), err
	}

	// Detect OS and run appropriate patch command
	osType := runtime.GOOS
	var cmd *exec.Cmd
	var output []byte
	var err error

	switch osType {
	case "linux":
		cmd, output, err = e.patchLinux(ctx, params)
	case "windows":
		return executor.CreateFailureResult(1, "", "Windows patching not yet implemented", nil, time.Since(startedAt)), fmt.Errorf("Windows patching not yet implemented")
	case "darwin":
		return executor.CreateFailureResult(1, "", "macOS patching not yet implemented", nil, time.Since(startedAt)), fmt.Errorf("macOS patching not yet implemented")
	default:
		msg := fmt.Sprintf("unsupported OS: %s", osType)
		return executor.CreateFailureResult(1, "", msg, nil, time.Since(startedAt)), fmt.Errorf(msg)
	}

	duration := time.Since(startedAt)

	if err != nil {
		exitCode := 1
		if cmd != nil && cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		return executor.CreateFailureResult(exitCode, string(output), "", err, duration), nil
	}

	return executor.CreateSuccessResult(string(output), duration), nil
}

// patchLinux handles patching for Linux distributions
func (e *PatchExecutor) patchLinux(ctx context.Context, params PatchParams) (*exec.Cmd, []byte, error) {
	distro, err := e.detectLinuxDistro()
	if err != nil {
		return nil, nil, err
	}

	switch distro {
	case "ubuntu", "debian":
		return e.patchDebian(ctx, params)
	case "rhel", "centos", "rocky", "alma":
		return e.patchRedHat(ctx, params)
	default:
		return nil, nil, fmt.Errorf("unsupported Linux distribution: %s", distro)
	}
}

// patchDebian handles Debian-based distributions (Ubuntu, Debian)
func (e *PatchExecutor) patchDebian(ctx context.Context, params PatchParams) (*exec.Cmd, []byte, error) {
	// Update package list
	updateCmd := exec.CommandContext(ctx, "apt-get", "update")
	if output, err := updateCmd.CombinedOutput(); err != nil {
		return updateCmd, output, fmt.Errorf("failed to update package list: %w", err)
	}

	// Build upgrade command
	args := []string{"apt-get", "-y"}

	if params.SecurityOnly {
		args = append(args, "upgrade", "-s")
		// Filter security updates
		args = append(args, "--security")
	} else if params.UpdateAll {
		args = append(args, "dist-upgrade")
	} else {
		// Install specific packages
		args = append(args, "install")
		args = append(args, params.PatchIDs...)
	}

	if params.Force {
		args = append(args, "--force-yes")
	}

	cmd := exec.CommandContext(ctx, "sudo", args...)
	output, err := cmd.CombinedOutput()
	return cmd, output, err
}

// patchRedHat handles Red Hat-based distributions (RHEL, CentOS, Rocky, AlmaLinux)
func (e *PatchExecutor) patchRedHat(ctx context.Context, params PatchParams) (*exec.Cmd, []byte, error) {
	args := []string{"yum", "-y"}

	if params.SecurityOnly {
		args = append(args, "update", "--security")
	} else if params.UpdateAll {
		args = append(args, "update")
	} else {
		// Install specific packages
		args = append(args, "install")
		args = append(args, params.PatchIDs...)
	}

	cmd := exec.CommandContext(ctx, "sudo", args...)
	output, err := cmd.CombinedOutput()
	return cmd, output, err
}

// detectLinuxDistro detects the Linux distribution
func (e *PatchExecutor) detectLinuxDistro() (string, error) {
	// Try to read /etc/os-release
	cmd := exec.Command("cat", "/etc/os-release")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to detect Linux distribution: %w", err)
	}

	osRelease := string(output)

	if strings.Contains(osRelease, "Ubuntu") {
		return "ubuntu", nil
	} else if strings.Contains(osRelease, "Debian") {
		return "debian", nil
	} else if strings.Contains(osRelease, "Red Hat") || strings.Contains(osRelease, "RHEL") {
		return "rhel", nil
	} else if strings.Contains(osRelease, "CentOS") {
		return "centos", nil
	} else if strings.Contains(osRelease, "Rocky") {
		return "rocky", nil
	} else if strings.Contains(osRelease, "AlmaLinux") {
		return "alma", nil
	}

	return "", fmt.Errorf("unknown Linux distribution")
}
