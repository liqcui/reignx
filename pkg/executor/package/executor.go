package pkg

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

// PackageExecutor handles package installation
type PackageExecutor struct {
	*executor.BaseExecutor
}

// PackageParams defines the parameters for package operations
type PackageParams struct {
	Action      string   `json:"action"`       // install, remove, upgrade
	Packages    []string `json:"packages"`     // Package names
	Version     string   `json:"version"`      // Specific version (optional)
	Repository  string   `json:"repository"`   // Repository URL (optional)
	Force       bool     `json:"force"`        // Force installation
	SkipVerify  bool     `json:"skip_verify"`  // Skip signature verification
}

// NewPackageExecutor creates a new package executor
func NewPackageExecutor() *PackageExecutor {
	return &PackageExecutor{
		BaseExecutor: executor.NewBaseExecutor(core.TaskTypePackage),
	}
}

// Validate checks if the package parameters are valid
func (e *PackageExecutor) Validate(task *core.Task) error {
	var params PackageParams
	if err := executor.ParseParameters(task.Parameters, &params); err != nil {
		return fmt.Errorf("invalid package parameters: %w", err)
	}

	if params.Action == "" {
		return fmt.Errorf("action is required (install, remove, upgrade)")
	}

	if len(params.Packages) == 0 {
		return fmt.Errorf("at least one package must be specified")
	}

	validActions := map[string]bool{"install": true, "remove": true, "upgrade": true}
	if !validActions[params.Action] {
		return fmt.Errorf("invalid action: %s (must be install, remove, or upgrade)", params.Action)
	}

	return nil
}

// Execute runs the package operation
func (e *PackageExecutor) Execute(ctx context.Context, task *core.Task) (*core.TaskResult, error) {
	startedAt := time.Now()

	var params PackageParams
	if err := executor.ParseParameters(task.Parameters, &params); err != nil {
		return executor.CreateFailureResult(1, "", "", err, time.Since(startedAt)), err
	}

	// Detect OS and run appropriate package command
	osType := runtime.GOOS
	var cmd *exec.Cmd
	var output []byte
	var err error

	switch osType {
	case "linux":
		cmd, output, err = e.manageLinuxPackage(ctx, params)
	case "windows":
		return executor.CreateFailureResult(1, "", "Windows package management not yet implemented", nil, time.Since(startedAt)), fmt.Errorf("Windows package management not yet implemented")
	case "darwin":
		cmd, output, err = e.manageDarwinPackage(ctx, params)
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

// manageLinuxPackage handles package management for Linux
func (e *PackageExecutor) manageLinuxPackage(ctx context.Context, params PackageParams) (*exec.Cmd, []byte, error) {
	distro, err := e.detectLinuxDistro()
	if err != nil {
		return nil, nil, err
	}

	switch distro {
	case "ubuntu", "debian":
		return e.manageDebianPackage(ctx, params)
	case "rhel", "centos", "rocky", "alma":
		return e.manageRedHatPackage(ctx, params)
	default:
		return nil, nil, fmt.Errorf("unsupported Linux distribution: %s", distro)
	}
}

// manageDebianPackage handles Debian-based package management
func (e *PackageExecutor) manageDebianPackage(ctx context.Context, params PackageParams) (*exec.Cmd, []byte, error) {
	args := []string{"apt-get", "-y"}

	switch params.Action {
	case "install":
		args = append(args, "install")
		for _, pkg := range params.Packages {
			if params.Version != "" {
				args = append(args, fmt.Sprintf("%s=%s", pkg, params.Version))
			} else {
				args = append(args, pkg)
			}
		}
	case "remove":
		args = append(args, "remove")
		args = append(args, params.Packages...)
	case "upgrade":
		args = append(args, "upgrade")
		args = append(args, params.Packages...)
	}

	if params.Force {
		args = append(args, "--force-yes")
	}

	if params.SkipVerify {
		args = append(args, "--allow-unauthenticated")
	}

	cmd := exec.CommandContext(ctx, "sudo", args...)
	output, err := cmd.CombinedOutput()
	return cmd, output, err
}

// manageRedHatPackage handles Red Hat-based package management
func (e *PackageExecutor) manageRedHatPackage(ctx context.Context, params PackageParams) (*exec.Cmd, []byte, error) {
	args := []string{"yum", "-y"}

	switch params.Action {
	case "install":
		args = append(args, "install")
		for _, pkg := range params.Packages {
			if params.Version != "" {
				args = append(args, fmt.Sprintf("%s-%s", pkg, params.Version))
			} else {
				args = append(args, pkg)
			}
		}
	case "remove":
		args = append(args, "remove")
		args = append(args, params.Packages...)
	case "upgrade":
		args = append(args, "update")
		args = append(args, params.Packages...)
	}

	if params.SkipVerify {
		args = append(args, "--nogpgcheck")
	}

	cmd := exec.CommandContext(ctx, "sudo", args...)
	output, err := cmd.CombinedOutput()
	return cmd, output, err
}

// manageDarwinPackage handles macOS package management using Homebrew
func (e *PackageExecutor) manageDarwinPackage(ctx context.Context, params PackageParams) (*exec.Cmd, []byte, error) {
	var args []string

	switch params.Action {
	case "install":
		args = append(args, "install")
		args = append(args, params.Packages...)
	case "remove":
		args = append(args, "uninstall")
		args = append(args, params.Packages...)
	case "upgrade":
		args = append(args, "upgrade")
		args = append(args, params.Packages...)
	}

	cmd := exec.CommandContext(ctx, "brew", args...)
	output, err := cmd.CombinedOutput()
	return cmd, output, err
}

// detectLinuxDistro detects the Linux distribution
func (e *PackageExecutor) detectLinuxDistro() (string, error) {
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
