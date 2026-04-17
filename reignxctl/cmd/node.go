package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewNodeCmd creates the node command
func NewNodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Manage nodes",
		Long:  "Manage nodes in the ReignX infrastructure",
	}

	cmd.AddCommand(newNodeListCmd())
	cmd.AddCommand(newNodeAddCmd())
	cmd.AddCommand(newNodeShowCmd())
	cmd.AddCommand(newNodeDeleteCmd())
	cmd.AddCommand(newNodeTagCmd())
	cmd.AddCommand(newNodeStatusCmd())

	return cmd
}

// newNodeListCmd creates the 'node list' command
func newNodeListCmd() *cobra.Command {
	var (
		filter string
		limit  int
		offset int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all nodes",
		Long:  "List all nodes managed by ReignX with optional filtering",
		Example: `  reignx node list
  reignx node list --filter="env=prod"
  reignx node list --filter="os=ubuntu,region=us-east"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNodeList(cmd, filter, limit, offset)
		},
	}

	cmd.Flags().StringVarP(&filter, "filter", "f", "", "Filter nodes (key=value,key=value)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "Maximum number of nodes to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "Offset for pagination")

	return cmd
}

// newNodeAddCmd creates the 'node add' command
func newNodeAddCmd() *cobra.Command {
	var (
		user     string
		key      string
		password string
		port     int
		jumpHost string
		tags     []string
		dryRun   bool
	)

	cmd := &cobra.Command{
		Use:   "add <ip-address>",
		Short: "Add a new node",
		Long:  "Add a new node to ReignX management via SSH",
		Example: `  reignx node add 192.168.1.10 --user root --key ~/.ssh/id_rsa
  reignx node add 192.168.1.20 --user admin --password secret
  reignx node add 192.168.1.30 --user root --key ~/.ssh/id_rsa --jump-host bastion.example.com
  reignx node add 192.168.1.40 --user root --key ~/.ssh/id_rsa --tags env=prod,region=us-east`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ipAddress := args[0]
			return runNodeAdd(cmd, ipAddress, user, key, password, port, jumpHost, tags, dryRun)
		},
	}

	cmd.Flags().StringVarP(&user, "user", "u", "root", "SSH username")
	cmd.Flags().StringVarP(&key, "key", "k", "", "Path to SSH private key")
	cmd.Flags().StringVarP(&password, "password", "p", "", "SSH password (not recommended)")
	cmd.Flags().IntVar(&port, "port", 22, "SSH port")
	cmd.Flags().StringVar(&jumpHost, "jump-host", "", "Jump host address")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "Node tags (key=value)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Test connection without adding node")

	return cmd
}

// newNodeShowCmd creates the 'node show' command
func newNodeShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <node-id>",
		Short: "Show node details",
		Long:  "Display detailed information about a specific node",
		Example: `  reignx node show node-1
  reignx node show web-server-01`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeID := args[0]
			return runNodeShow(cmd, nodeID)
		},
	}

	return cmd
}

// newNodeDeleteCmd creates the 'node delete' command
func newNodeDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <node-id>",
		Short: "Delete a node",
		Long:  "Remove a node from ReignX management",
		Example: `  reignx node delete node-1
  reignx node delete node-1 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeID := args[0]
			return runNodeDelete(cmd, nodeID, force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force delete without confirmation")

	return cmd
}

// newNodeTagCmd creates the 'node tag' command
func newNodeTagCmd() *cobra.Command {
	var remove bool

	cmd := &cobra.Command{
		Use:   "tag <node-id> <key=value>...",
		Short: "Tag a node",
		Long:  "Add or remove tags from a node",
		Example: `  reignx node tag node-1 env=prod region=us-east
  reignx node tag node-1 env=staging --remove`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeID := args[0]
			tags := args[1:]
			return runNodeTag(cmd, nodeID, tags, remove)
		},
	}

	cmd.Flags().BoolVar(&remove, "remove", false, "Remove tags instead of adding")

	return cmd
}

// newNodeStatusCmd creates the 'node status' command
func newNodeStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show cluster status",
		Long:  "Display overall cluster status and statistics",
		Example: `  reignx node status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNodeStatus(cmd)
		},
	}

	return cmd
}

// Implementation functions

func runNodeList(cmd *cobra.Command, filter string, limit, offset int) error {
	fmt.Println("Listing nodes...")
	fmt.Printf("Filter: %s, Limit: %d, Offset: %d\n", filter, limit, offset)

	// TODO: Call API to get nodes
	// For now, show example output
	printTable([]string{"ID", "HOSTNAME", "IP", "MODE", "STATUS", "OS"}, [][]string{
		{"node-1", "web-server-01.example.com", "192.168.1.10", "ssh", "online", "ubuntu-22.04"},
		{"node-2", "db-server-01.example.com", "192.168.1.20", "agent", "online", "rocky-9"},
		{"node-3", "app-server-01.example.com", "192.168.1.30", "hybrid", "online", "debian-12"},
	})

	return nil
}

func runNodeAdd(cmd *cobra.Command, ipAddress, user, key, password string, port int, jumpHost string, tags []string, dryRun bool) error {
	fmt.Printf("Adding node %s...\n", ipAddress)
	fmt.Printf("User: %s, Port: %d\n", user, port)
	if key != "" {
		fmt.Printf("Authentication: SSH key (%s)\n", key)
	} else if password != "" {
		fmt.Println("Authentication: Password")
	}
	if jumpHost != "" {
		fmt.Printf("Jump Host: %s\n", jumpHost)
	}
	if len(tags) > 0 {
		fmt.Printf("Tags: %v\n", tags)
	}
	if dryRun {
		fmt.Println("\n✓ Dry run: Connection test successful (not added)")
		return nil
	}

	// TODO: Call API to add node
	fmt.Println("\n✓ Node added successfully")
	fmt.Println("Node ID: node-" + ipAddress)

	return nil
}

func runNodeShow(cmd *cobra.Command, nodeID string) error {
	fmt.Printf("Node Details: %s\n\n", nodeID)

	// TODO: Call API to get node details
	// For now, show example output
	details := map[string]string{
		"ID":           nodeID,
		"Hostname":     "web-server-01.example.com",
		"IP Address":   "192.168.1.10",
		"Mode":         "ssh",
		"Status":       "online",
		"OS Type":      "linux",
		"OS Version":   "ubuntu-22.04",
		"Architecture": "amd64",
		"Last Seen":    "2026-04-15 15:30:45",
		"Tags":         "env=prod, region=us-east, team=platform",
	}

	for key, value := range details {
		fmt.Printf("  %-15s: %s\n", key, value)
	}

	return nil
}

func runNodeDelete(cmd *cobra.Command, nodeID string, force bool) error {
	if !force {
		fmt.Printf("Are you sure you want to delete node %s? (y/N): ", nodeID)
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	fmt.Printf("Deleting node %s...\n", nodeID)
	// TODO: Call API to delete node
	fmt.Println("✓ Node deleted successfully")

	return nil
}

func runNodeTag(cmd *cobra.Command, nodeID string, tags []string, remove bool) error {
	action := "Adding"
	if remove {
		action = "Removing"
	}

	fmt.Printf("%s tags for node %s...\n", action, nodeID)
	fmt.Printf("Tags: %v\n", tags)

	// TODO: Call API to update tags
	fmt.Println("✓ Tags updated successfully")

	return nil
}

func runNodeStatus(cmd *cobra.Command) error {
	fmt.Println("Cluster Status\n")

	// TODO: Call API to get cluster statistics
	// For now, show example output
	fmt.Println("Total Nodes:     150")
	fmt.Println("Online:          145 (96.7%)")
	fmt.Println("Offline:         5 (3.3%)")
	fmt.Println()
	fmt.Println("By Mode:")
	fmt.Println("  SSH:           80 (53.3%)")
	fmt.Println("  Agent:         60 (40.0%)")
	fmt.Println("  Hybrid:        10 (6.7%)")
	fmt.Println()
	fmt.Println("By OS:")
	fmt.Println("  Ubuntu:        90 (60.0%)")
	fmt.Println("  Rocky Linux:   40 (26.7%)")
	fmt.Println("  Debian:        20 (13.3%)")

	return nil
}

// printTable prints data in table format
func printTable(headers []string, rows [][]string) {
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, h := range headers {
		fmt.Printf("%-*s  ", widths[i], h)
	}
	fmt.Println()

	// Print separator
	for _, w := range widths {
		for j := 0; j < w; j++ {
			fmt.Print("-")
		}
		fmt.Print("  ")
	}
	fmt.Println()

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			fmt.Printf("%-*s  ", widths[i], cell)
		}
		fmt.Println()
	}
}
