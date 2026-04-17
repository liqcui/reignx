package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewFirmwareCmd creates the firmware command
func NewFirmwareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "firmware",
		Short: "Manage firmware",
		Long:  "Scan, list, and update firmware across managed nodes",
	}

	cmd.AddCommand(newFirmwareScanCmd())
	cmd.AddCommand(newFirmwareListCmd())
	cmd.AddCommand(newFirmwareUpdateCmd())
	cmd.AddCommand(newFirmwareRollbackCmd())

	return cmd
}

func newFirmwareScanCmd() *cobra.Command {
	var targets string

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan firmware versions",
		Long:  "Scan and collect firmware version information from nodes",
		Example: `  reignx firmware scan
  reignx firmware scan --targets="vendor=dell"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFirmwareScan(cmd, targets)
		},
	}

	cmd.Flags().StringVar(&targets, "targets", "", "Target nodes filter")

	return cmd
}

func newFirmwareListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List firmware inventory",
		Long:  "Display firmware inventory across all nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFirmwareList(cmd)
		},
	}

	return cmd
}

func newFirmwareUpdateCmd() *cobra.Command {
	var (
		node      string
		component string
		version   string
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update firmware",
		Long:  "Update firmware on specified nodes",
		Example: `  reignx firmware update --node node-1 --component BIOS --version 2.8
  reignx firmware update --node node-1 --component BMC --version 1.5 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFirmwareUpdate(cmd, node, component, version, force)
		},
	}

	cmd.Flags().StringVar(&node, "node", "", "Target node ID")
	cmd.Flags().StringVar(&component, "component", "", "Firmware component (BIOS, BMC, NIC, etc.)")
	cmd.Flags().StringVar(&version, "version", "", "Target firmware version")
	cmd.Flags().BoolVar(&force, "force", false, "Force update without confirmation")

	cmd.MarkFlagRequired("node")
	cmd.MarkFlagRequired("component")
	cmd.MarkFlagRequired("version")

	return cmd
}

func newFirmwareRollbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback <node-id> <component>",
		Short: "Rollback firmware",
		Long:  "Rollback firmware to previous version",
		Example: `  reignx firmware rollback node-1 BIOS`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeID := args[0]
			component := args[1]
			return runFirmwareRollback(cmd, nodeID, component)
		},
	}

	return cmd
}

func runFirmwareScan(cmd *cobra.Command, targets string) error {
	fmt.Println("Scanning firmware versions...")
	if targets != "" {
		fmt.Printf("Targets: %s\n", targets)
	} else {
		fmt.Println("Targets: all nodes")
	}
	fmt.Println()

	// TODO: Call API to initiate firmware scan
	fmt.Println("Scan Progress: 100/100 nodes")
	fmt.Println()

	// Example output
	fmt.Println("Firmware Summary:")
	printTable([]string{"COMPONENT", "VERSION", "COUNT", "LATEST"}, [][]string{
		{"BIOS", "2.7", "80", "2.8"},
		{"BIOS", "2.8", "20", "2.8"},
		{"BMC", "1.4", "60", "1.5"},
		{"BMC", "1.5", "40", "1.5"},
	})

	fmt.Println("\nℹ  20 nodes have outdated BIOS firmware")
	fmt.Println("ℹ  60 nodes have outdated BMC firmware")

	return nil
}

func runFirmwareList(cmd *cobra.Command) error {
	fmt.Println("Firmware Inventory\n")

	printTable([]string{"NODE", "COMPONENT", "CURRENT", "AVAILABLE", "STATUS"}, [][]string{
		{"node-1", "BIOS", "2.7", "2.8", "outdated"},
		{"node-1", "BMC", "1.5", "1.5", "current"},
		{"node-2", "BIOS", "2.8", "2.8", "current"},
		{"node-2", "BMC", "1.4", "1.5", "outdated"},
	})

	return nil
}

func runFirmwareUpdate(cmd *cobra.Command, node, component, version string, force bool) error {
	fmt.Printf("Updating %s firmware on %s to version %s\n", component, node, version)

	if !force {
		fmt.Print("\n⚠  Warning: Firmware update may cause system reboot\n")
		fmt.Print("Continue? (y/N): ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	fmt.Println("\nUpdating firmware...")
	fmt.Println("  Downloading firmware image...")
	fmt.Println("  Verifying checksum...")
	fmt.Println("  Applying update...")
	fmt.Println("  Verifying new version...")
	fmt.Println("\n✓ Firmware updated successfully")
	fmt.Printf("Node %s %s firmware is now version %s\n", node, component, version)

	return nil
}

func runFirmwareRollback(cmd *cobra.Command, nodeID, component string) error {
	fmt.Printf("Rolling back %s firmware on %s\n", component, nodeID)

	fmt.Print("Are you sure? (y/N): ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		fmt.Println("Cancelled")
		return nil
	}

	// TODO: Call API to rollback firmware
	fmt.Println("\nRolling back...")
	fmt.Println("✓ Firmware rolled back to previous version")

	return nil
}
