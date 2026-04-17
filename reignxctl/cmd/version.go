package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewVersionCmd creates the version command
func NewVersionCmd(version, buildTime, gitCommit string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Display detailed version information about reignxctl",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ReignX CLI (reignxctl)\n\n")
			fmt.Printf("Version:    %s\n", version)
			fmt.Printf("Build Time: %s\n", buildTime)
			fmt.Printf("Git Commit: %s\n", gitCommit)
			fmt.Printf("Go Version: %s\n", "go1.25.0")
			fmt.Printf("\n")
			fmt.Printf("Project:    https://github.com/reignx/reignx\n")
			fmt.Printf("License:    MIT\n")
		},
	}

	return cmd
}
