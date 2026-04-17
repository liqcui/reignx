package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

func powerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "power",
		Short: "Manage server power state",
		Long:  "Control server power: on, off, reboot, status",
	}

	cmd.AddCommand(powerOnCmd())
	cmd.AddCommand(powerOffCmd())
	cmd.AddCommand(rebootCmd())
	cmd.AddCommand(powerStatusCmd())

	return cmd
}

func powerOnCmd() *cobra.Command {
	var (
		servers string
		filter  string
	)

	cmd := &cobra.Command{
		Use:   "on [server-id]",
		Short: "Power on server(s)",
		Long: `Power on one or more servers using IPMI/BMC.

Examples:
  # Power on single server
  bm-cli power on server-001

  # Power on multiple servers
  bm-cli power on --servers server-001,server-002,server-003

  # Power on servers matching filter
  bm-cli power on --filter 'datacenter=dc1'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var serverIDs []string

			if len(args) > 0 {
				serverIDs = []string{args[0]}
			} else if servers != "" {
				serverIDs = strings.Split(servers, ",")
			} else if filter != "" {
				// TODO: Query API with filter to get server IDs
				return fmt.Errorf("filter support not yet implemented")
			} else {
				return fmt.Errorf("must specify server-id, --servers, or --filter")
			}

			return executePowerAction(serverIDs, "poweron")
		},
	}

	cmd.Flags().StringVar(&servers, "servers", "", "Comma-separated list of server IDs")
	cmd.Flags().StringVar(&filter, "filter", "", "Filter expression (e.g., 'datacenter=dc1')")

	return cmd
}

func powerOffCmd() *cobra.Command {
	var (
		servers string
		filter  string
		force   bool
	)

	cmd := &cobra.Command{
		Use:   "off [server-id]",
		Short: "Power off server(s)",
		Long: `Power off one or more servers using IPMI/BMC.

Examples:
  # Power off single server
  bm-cli power off server-001

  # Power off multiple servers
  bm-cli power off --servers server-001,server-002

  # Force power off (immediate, no graceful shutdown)
  bm-cli power off server-001 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var serverIDs []string

			if len(args) > 0 {
				serverIDs = []string{args[0]}
			} else if servers != "" {
				serverIDs = strings.Split(servers, ",")
			} else if filter != "" {
				return fmt.Errorf("filter support not yet implemented")
			} else {
				return fmt.Errorf("must specify server-id, --servers, or --filter")
			}

			action := "poweroff"
			if force {
				fmt.Println("⚠️  Warning: Force power off will immediately cut power without graceful shutdown")
			}

			return executePowerAction(serverIDs, action)
		},
	}

	cmd.Flags().StringVar(&servers, "servers", "", "Comma-separated list of server IDs")
	cmd.Flags().StringVar(&filter, "filter", "", "Filter expression")
	cmd.Flags().BoolVar(&force, "force", false, "Force immediate power off")

	return cmd
}

func rebootCmd() *cobra.Command {
	var (
		servers string
		filter  string
	)

	cmd := &cobra.Command{
		Use:   "reboot [server-id]",
		Short: "Reboot server(s)",
		Long: `Reboot one or more servers using IPMI/BMC.

Examples:
  # Reboot single server
  bm-cli reboot server-001

  # Reboot multiple servers
  bm-cli reboot --servers server-001,server-002,server-003

  # Reboot all servers in a rack
  bm-cli reboot --filter 'rack=rack-5'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var serverIDs []string

			if len(args) > 0 {
				serverIDs = []string{args[0]}
			} else if servers != "" {
				serverIDs = strings.Split(servers, ",")
			} else if filter != "" {
				return fmt.Errorf("filter support not yet implemented")
			} else {
				return fmt.Errorf("must specify server-id, --servers, or --filter")
			}

			return executePowerAction(serverIDs, "reboot")
		},
	}

	cmd.Flags().StringVar(&servers, "servers", "", "Comma-separated list of server IDs")
	cmd.Flags().StringVar(&filter, "filter", "", "Filter expression")

	return cmd
}

func powerStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <server-id>",
		Short: "Get power status of a server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverID := args[0]

			client := &http.Client{}
			url := fmt.Sprintf("%s/api/v1/servers/%s/power/status", apiServer, serverID)

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			fmt.Printf("Server %s power status: %s\n", serverID, result["power_state"])

			return nil
		},
	}
}

func executePowerAction(serverIDs []string, action string) error {
	client := &http.Client{}
	successCount := 0
	failCount := 0

	for _, serverID := range serverIDs {
		url := fmt.Sprintf("%s/api/v1/servers/%s/power/%s", apiServer, serverID, action)

		req, err := http.NewRequest("POST", url, nil)
		if err != nil {
			fmt.Printf("❌ %s: failed to create request: %v\n", serverID, err)
			failCount++
			continue
		}

		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("❌ %s: failed to send request: %v\n", serverID, err)
			failCount++
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("❌ %s: API error (%d): %s\n", serverID, resp.StatusCode, string(body))
			failCount++
			continue
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("❌ %s: failed to decode response: %v\n", serverID, err)
			failCount++
			continue
		}

		fmt.Printf("✅ %s: %s\n", serverID, result["message"])
		successCount++
	}

	fmt.Printf("\nResults: %d succeeded, %d failed\n", successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("%d power action(s) failed", failCount)
	}

	return nil
}
