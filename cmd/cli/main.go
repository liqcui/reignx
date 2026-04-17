package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

var (
	// Global flags
	apiServer string
	token     string
	verbose   bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "bm-cli",
		Short: "ReignX CLI - Command line interface for bare metal server management",
		Long: `ReignX CLI provides command-line access to manage servers, deploy packages,
install operating systems, and control server power remotely.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&apiServer, "server", "http://localhost:8090", "API server URL")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "Authentication token")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Add commands
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(serverCmd())
	rootCmd.AddCommand(powerCmd())
	rootCmd.AddCommand(installOSCmd())
	rootCmd.AddCommand(jobCmd())
	rootCmd.AddCommand(patchCmd())
	rootCmd.AddCommand(packageCmd())
	rootCmd.AddCommand(loginCmd())
	rootCmd.AddCommand(encryptCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ReignX CLI\n")
			fmt.Printf("Version:    %s\n", Version)
			fmt.Printf("Build Time: %s\n", BuildTime)
			fmt.Printf("Git Commit: %s\n", GitCommit)
		},
	}
}
