package main

import (
	"fmt"
	"os"

	"github.com/reignx/reignx/reignxctl/cmd"
	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "reignx",
		Short: "ReignX - Hybrid Infrastructure Management CLI",
		Long: `ReignX is a powerful infrastructure management system that supports
both SSH-based (agentless) and Agent-based (persistent) management modes.

Use reignx to manage thousands of servers with ease.`,
		Version: fmt.Sprintf("%s (built %s, commit %s)", Version, BuildTime, GitCommit),
	}

	// Add global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "Configuration file path")
	rootCmd.PersistentFlags().StringP("server", "s", "http://localhost:8080", "ReignX API server URL")
	rootCmd.PersistentFlags().StringP("token", "t", "", "Authentication token")
	rootCmd.PersistentFlags().StringP("output", "o", "table", "Output format (table, json, yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode")

	// Add command groups
	rootCmd.AddCommand(cmd.NewNodeCmd())
	rootCmd.AddCommand(cmd.NewDeployCmd())
	rootCmd.AddCommand(cmd.NewSwitchCmd())
	rootCmd.AddCommand(cmd.NewExecCmd())
	rootCmd.AddCommand(cmd.NewFirmwareCmd())
	rootCmd.AddCommand(cmd.NewOSCmd())
	rootCmd.AddCommand(cmd.NewModeCmd())
	rootCmd.AddCommand(cmd.NewVersionCmd(Version, BuildTime, GitCommit))

	// Execute root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
