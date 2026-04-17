package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// NewExecCmd creates the exec command
func NewExecCmd() *cobra.Command {
	var (
		all        bool
		targets    string
		nodes      []string
		script     string
		timeout    int
		parallel   bool
		concurrent int
		sudo       bool
	)

	cmd := &cobra.Command{
		Use:   "exec [flags] <command>",
		Short: "Execute commands on remote nodes",
		Long:  "Execute shell commands or scripts on one or more nodes",
		Example: `  reignx exec --all "uptime"
  reignx exec --targets="env=prod" "df -h"
  reignx exec --node node-1 "systemctl status nginx"
  reignx exec --all --script="/path/to/script.sh"
  reignx exec --all --parallel --concurrent=50 "apt-get update"
  reignx exec --all --sudo "systemctl restart nginx"`,
		Args: func(cmd *cobra.Command, args []string) error {
			if script == "" && len(args) == 0 {
				return fmt.Errorf("command or --script is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var command string
			if len(args) > 0 {
				command = strings.Join(args, " ")
			}
			return runExec(cmd, command, all, targets, nodes, script, timeout, parallel, concurrent, sudo)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Execute on all nodes")
	cmd.Flags().StringVar(&targets, "targets", "", "Target nodes by filter (key=value,key=value)")
	cmd.Flags().StringSliceVar(&nodes, "node", nil, "Target specific node(s)")
	cmd.Flags().StringVar(&script, "script", "", "Path to script file to execute")
	cmd.Flags().IntVar(&timeout, "timeout", 300, "Execution timeout in seconds")
	cmd.Flags().BoolVar(&parallel, "parallel", false, "Execute in parallel mode")
	cmd.Flags().IntVar(&concurrent, "concurrent", 10, "Number of concurrent executions")
	cmd.Flags().BoolVar(&sudo, "sudo", false, "Execute with sudo")

	return cmd
}

func runExec(cmd *cobra.Command, command string, all bool, targets string, nodes []string, script string, timeout int, parallel bool, concurrent int, sudo bool) error {
	// Determine what to execute
	execCommand := command
	if script != "" {
		fmt.Printf("Loading script: %s\n", script)
		// TODO: Read script file
		execCommand = "bash /tmp/script.sh" // Placeholder
	}

	if sudo {
		execCommand = "sudo " + execCommand
	}

	// Determine target nodes
	var targetDesc string
	if all {
		targetDesc = "all nodes"
	} else if targets != "" {
		targetDesc = fmt.Sprintf("nodes matching filter: %s", targets)
	} else if len(nodes) > 0 {
		targetDesc = fmt.Sprintf("nodes: %s", strings.Join(nodes, ", "))
	} else {
		return fmt.Errorf("must specify --all, --targets, or --node")
	}

	fmt.Printf("Executing on %s\n", targetDesc)
	fmt.Printf("Command: %s\n", execCommand)
	fmt.Printf("Timeout: %ds\n", timeout)
	if parallel {
		fmt.Printf("Mode: Parallel (concurrent: %d)\n", concurrent)
	} else {
		fmt.Printf("Mode: Sequential\n")
	}
	fmt.Println()

	// TODO: Call API to execute command
	// For now, show example output
	results := []execResult{
		{NodeID: "node-1", Hostname: "web-01", Success: true, ExitCode: 0, Output: "15:30:42 up 42 days, load average: 0.15, 0.20, 0.18", Duration: "1.2s"},
		{NodeID: "node-2", Hostname: "web-02", Success: true, ExitCode: 0, Output: "15:30:43 up 38 days, load average: 0.08, 0.12, 0.10", Duration: "1.3s"},
		{NodeID: "node-3", Hostname: "db-01", Success: false, ExitCode: 1, Output: "", Error: "connection timeout", Duration: "30.0s"},
	}

	return printExecResults(results)
}

type execResult struct {
	NodeID   string
	Hostname string
	Success  bool
	ExitCode int
	Output   string
	Error    string
	Duration string
}

func printExecResults(results []execResult) error {
	successCount := 0
	failCount := 0

	for _, result := range results {
		if result.Success {
			successCount++
			fmt.Printf("✓ %s (%s)\n", result.NodeID, result.Hostname)
			if result.Output != "" {
				// Indent output
				lines := strings.Split(result.Output, "\n")
				for _, line := range lines {
					fmt.Printf("  %s\n", line)
				}
			}
			fmt.Printf("  Exit Code: %d | Duration: %s\n", result.ExitCode, result.Duration)
		} else {
			failCount++
			fmt.Printf("✗ %s (%s)\n", result.NodeID, result.Hostname)
			fmt.Printf("  Error: %s\n", result.Error)
			fmt.Printf("  Duration: %s\n", result.Duration)
		}
		fmt.Println()
	}

	// Summary
	total := len(results)
	fmt.Printf("Summary: %d total, %d succeeded, %d failed\n", total, successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("%d node(s) failed", failCount)
	}

	return nil
}
