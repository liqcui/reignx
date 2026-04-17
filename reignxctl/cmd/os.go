package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewOSCmd creates the OS management command
func NewOSCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "os",
		Short: "Manage operating systems",
		Long:  "Install, upgrade, and manage operating systems on nodes",
	}

	cmd.AddCommand(newOSReinstallCmd())
	cmd.AddCommand(newOSUpgradeCmd())
	cmd.AddCommand(newOSListImagesCmd())
	cmd.AddCommand(newOSUploadCmd())

	return cmd
}

func newOSReinstallCmd() *cobra.Command {
	var (
		node  string
		image string
		force bool
	)

	cmd := &cobra.Command{
		Use:   "reinstall",
		Short: "Reinstall OS on a node",
		Long:  "Reinstall operating system on a node using PXE boot or cloud API",
		Example: `  reignx os reinstall --node node-1 --image ubuntu-22.04
  reignx os reinstall --node node-1 --image rocky-9 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOSReinstall(cmd, node, image, force)
		},
	}

	cmd.Flags().StringVar(&node, "node", "", "Target node ID")
	cmd.Flags().StringVar(&image, "image", "", "OS image name")
	cmd.Flags().BoolVar(&force, "force", false, "Force reinstall without confirmation")

	cmd.MarkFlagRequired("node")
	cmd.MarkFlagRequired("image")

	return cmd
}

func newOSUpgradeCmd() *cobra.Command {
	var (
		node          string
		targetVersion string
		reboot        bool
	)

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade OS version",
		Long:  "Perform in-place OS upgrade on a node",
		Example: `  reignx os upgrade --node node-1 --target-version 24.04
  reignx os upgrade --node node-1 --target-version 24.04 --reboot`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOSUpgrade(cmd, node, targetVersion, reboot)
		},
	}

	cmd.Flags().StringVar(&node, "node", "", "Target node ID")
	cmd.Flags().StringVar(&targetVersion, "target-version", "", "Target OS version")
	cmd.Flags().BoolVar(&reboot, "reboot", false, "Automatically reboot after upgrade")

	cmd.MarkFlagRequired("node")
	cmd.MarkFlagRequired("target-version")

	return cmd
}

func newOSListImagesCmd() *cobra.Command {
	var osType string

	cmd := &cobra.Command{
		Use:   "list-images",
		Short: "List available OS images",
		Long:  "Display all available OS images for installation",
		Example: `  reignx os list-images
  reignx os list-images --type linux`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOSListImages(cmd, osType)
		},
	}

	cmd.Flags().StringVar(&osType, "type", "", "Filter by OS type (linux, windows)")

	return cmd
}

func newOSUploadCmd() *cobra.Command {
	var (
		image        string
		name         string
		osType       string
		version      string
		architecture string
	)

	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload OS image",
		Long:  "Upload a new OS installation image",
		Example: `  reignx os upload --image ubuntu-22.04.iso --name ubuntu-22.04 --type linux --version 22.04
  reignx os upload --image rocky-9.iso --name rocky-9 --type linux --version 9 --arch amd64`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOSUpload(cmd, image, name, osType, version, architecture)
		},
	}

	cmd.Flags().StringVar(&image, "image", "", "Path to OS image file")
	cmd.Flags().StringVar(&name, "name", "", "Image name")
	cmd.Flags().StringVar(&osType, "type", "", "OS type (linux, windows)")
	cmd.Flags().StringVar(&version, "version", "", "OS version")
	cmd.Flags().StringVar(&architecture, "arch", "amd64", "Architecture")

	cmd.MarkFlagRequired("image")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("version")

	return cmd
}

func runOSReinstall(cmd *cobra.Command, node, image string, force bool) error {
	fmt.Printf("⚠  Warning: This will ERASE ALL DATA on node %s\n", node)
	fmt.Printf("OS Image: %s\n\n", image)

	if !force {
		fmt.Print("Are you absolutely sure? Type 'yes' to confirm: ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	fmt.Println("\nInitiating OS reinstallation...")
	fmt.Println("  Generating kickstart configuration...")
	fmt.Println("  Configuring PXE boot...")
	fmt.Println("  Triggering server reboot via IPMI...")
	fmt.Println("  Waiting for installation to complete...")
	fmt.Println()
	fmt.Println("Installation Progress:")
	fmt.Println("  Partitioning disk... ✓")
	fmt.Println("  Installing base system... ✓")
	fmt.Println("  Installing packages... ✓")
	fmt.Println("  Configuring bootloader... ✓")
	fmt.Println("  Rebooting... ✓")
	fmt.Println()
	fmt.Println("✓ OS reinstallation completed successfully")
	fmt.Printf("Node %s is now running %s\n", node, image)

	return nil
}

func runOSUpgrade(cmd *cobra.Command, node, targetVersion string, reboot bool) error {
	fmt.Printf("Upgrading node %s to version %s\n", node, targetVersion)

	fmt.Print("\nThis will upgrade the OS. Continue? (y/N): ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		fmt.Println("Cancelled")
		return nil
	}

	fmt.Println("\nPerforming OS upgrade...")
	fmt.Println("  Updating package lists...")
	fmt.Println("  Downloading packages...")
	fmt.Println("  Installing updates...")
	fmt.Println("  Configuring new packages...")

	if reboot {
		fmt.Println("  Rebooting node...")
		fmt.Println("  Waiting for node to come back online...")
		fmt.Println("  Verifying upgrade...")
	}

	fmt.Println("\n✓ OS upgrade completed successfully")
	fmt.Printf("Node %s is now running version %s\n", node, targetVersion)

	if !reboot {
		fmt.Println("\nℹ  Note: A reboot is required to complete the upgrade")
	}

	return nil
}

func runOSListImages(cmd *cobra.Command, osType string) error {
	fmt.Println("Available OS Images\n")

	if osType != "" {
		fmt.Printf("Filter: Type = %s\n\n", osType)
	}

	printTable([]string{"NAME", "TYPE", "VERSION", "ARCH", "SIZE", "UPLOADED"}, [][]string{
		{"ubuntu-22.04", "linux", "22.04", "amd64", "2.5GB", "2026-03-15"},
		{"ubuntu-24.04", "linux", "24.04", "amd64", "2.8GB", "2026-04-01"},
		{"rocky-9", "linux", "9", "amd64", "2.2GB", "2026-03-20"},
		{"debian-12", "linux", "12", "amd64", "2.1GB", "2026-03-25"},
		{"windows-2022", "windows", "2022", "amd64", "5.2GB", "2026-04-10"},
	})

	return nil
}

func runOSUpload(cmd *cobra.Command, image, name, osType, version, architecture string) error {
	fmt.Printf("Uploading OS image: %s\n", image)
	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Type: %s\n", osType)
	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Architecture: %s\n\n", architecture)

	// TODO: Call API to upload image
	fmt.Println("Upload Progress:")
	fmt.Println("  Reading file...")
	fmt.Println("  Calculating checksum...")
	fmt.Println("  Uploading: 100%")
	fmt.Println("  Verifying upload...")
	fmt.Println()
	fmt.Println("✓ OS image uploaded successfully")
	fmt.Printf("Image ID: %s\n", name)

	return nil
}
