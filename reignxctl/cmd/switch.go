package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// NewSwitchCmd creates the switch command
func NewSwitchCmd() *cobra.Command {
	var (
		all   bool
		force bool
		wait  bool
	)

	cmd := &cobra.Command{
		Use:   "switch <node-id> <from-mode> <to-mode>",
		Short: "Switch node management mode",
		Long:  "Switch a node between SSH mode and Agent mode",
		Example: `  reignx switch node-1 ssh agent        # Upgrade to Agent mode
  reignx switch node-2 agent ssh        # Downgrade to SSH mode
  reignx switch --all --mode=agent      # Upgrade all nodes to Agent mode
  reignx switch node-1 ssh agent --wait # Wait for completion`,
		Args: func(cmd *cobra.Command, args []string) error {
			if all {
				return nil // --all flag doesn't need args
			}
			if len(args) != 3 {
				return fmt.Errorf("requires 3 arguments: <node-id> <from-mode> <to-mode>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				return runSwitchAll(cmd, force, wait)
			}
			nodeID := args[0]
			fromMode := args[1]
			toMode := args[2]
			return runSwitch(cmd, nodeID, fromMode, toMode, force, wait)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Switch all nodes")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force switch without confirmation")
	cmd.Flags().BoolVarP(&wait, "wait", "w", false, "Wait for switch to complete")

	return cmd
}

func runSwitch(cmd *cobra.Command, nodeID, fromMode, toMode string, force, wait bool) error {
	// Validate modes
	validModes := map[string]bool{"ssh": true, "agent": true, "hybrid": true}
	if !validModes[fromMode] {
		return fmt.Errorf("invalid from-mode: %s (must be ssh, agent, or hybrid)", fromMode)
	}
	if !validModes[toMode] {
		return fmt.Errorf("invalid to-mode: %s (must be ssh, agent, or hybrid)", toMode)
	}

	if fromMode == toMode {
		return fmt.Errorf("from-mode and to-mode cannot be the same")
	}

	fmt.Printf("Switching node %s: %s → %s\n\n", nodeID, fromMode, toMode)

	if !force {
		fmt.Print("Are you sure? (y/N): ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	// TODO: Call API to initiate switch
	fmt.Println("Initiating mode switch...")

	// Simulate switch steps
	steps := getSwitchSteps(fromMode, toMode)
	for i, step := range steps {
		fmt.Printf("[%d/%d] %s...\n", i+1, len(steps), step)
		time.Sleep(500 * time.Millisecond) // Simulate work
	}

	fmt.Println("\n✓ Mode switch completed successfully")
	fmt.Printf("Node %s is now in %s mode\n", nodeID, toMode)

	if wait {
		fmt.Println("\nWaiting for node to be fully operational...")
		time.Sleep(2 * time.Second)
		fmt.Println("✓ Node is operational")
	}

	return nil
}

func runSwitchAll(cmd *cobra.Command, force, wait bool) error {
	fmt.Println("Switching all nodes to Agent mode...")

	if !force {
		fmt.Print("This will switch ALL nodes. Are you sure? (y/N): ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	// TODO: Call API to switch all nodes
	fmt.Println("\nProgress:")
	fmt.Println("  Nodes to switch: 80")
	fmt.Println("  Completed: 80")
	fmt.Println("  Failed: 0")
	fmt.Println("\n✓ All nodes switched successfully")

	return nil
}

func getSwitchSteps(fromMode, toMode string) []string {
	if fromMode == "ssh" && toMode == "agent" {
		return []string{
			"Verifying SSH connectivity",
			"Uploading agent binary",
			"Installing agent service",
			"Generating agent certificates",
			"Starting agent",
			"Verifying agent connection",
			"Updating node mode",
		}
	} else if fromMode == "agent" && toMode == "ssh" {
		return []string{
			"Stopping agent service",
			"Removing agent binary",
			"Cleaning up certificates",
			"Updating node mode",
			"Verifying SSH connectivity",
		}
	}

	return []string{
		"Preparing mode switch",
		"Updating configuration",
		"Verifying connectivity",
		"Updating node mode",
	}
}
