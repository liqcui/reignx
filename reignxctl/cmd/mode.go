package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

// NewModeCmd creates the mode command
func NewModeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mode",
		Short: "Switch node management mode",
		Long:  `Switch a node between SSH mode and Agent mode`,
	}

	cmd.AddCommand(newModeSwitchCmd())
	cmd.AddCommand(newModeProgressCmd())

	return cmd
}

func newModeSwitchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch <node-id> <mode>",
		Short: "Switch node to specified mode",
		Long: `Switch a node to the specified mode (ssh or agent).

Examples:
  # Switch node to agent mode
  reignxctl mode switch node-123 agent

  # Switch node to SSH mode
  reignxctl mode switch node-123 ssh`,
		Args: cobra.ExactArgs(2),
		Run:  runModeSwitch,
	}

	cmd.Flags().BoolP("wait", "w", false, "Wait for switch to complete")
	cmd.Flags().IntP("timeout", "t", 300, "Timeout in seconds when waiting")

	return cmd
}

func newModeProgressCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "progress <node-id>",
		Short: "Check mode switch progress",
		Long:  `Check the progress of an ongoing mode switch operation`,
		Args:  cobra.ExactArgs(1),
		Run:   runModeProgress,
	}
}

func runModeSwitch(cmd *cobra.Command, args []string) {
	nodeID := args[0]
	toMode := args[1]

	// Validate mode
	if toMode != "ssh" && toMode != "agent" {
		fmt.Printf("Error: invalid mode '%s' (must be 'ssh' or 'agent')\n", toMode)
		return
	}

	wait, _ := cmd.Flags().GetBool("wait")
	timeout, _ := cmd.Flags().GetInt("timeout")

	// Get API URL from parent command
	apiURL, _ := cmd.Root().PersistentFlags().GetString("server")

	// Create request
	reqBody := map[string]string{
		"to_mode": toMode,
	}
	jsonData, _ := json.Marshal(reqBody)

	// Send request
	url := fmt.Sprintf("%s/api/v1/nodes/%s/switch-mode", apiURL, nodeID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error: failed to initiate mode switch: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		fmt.Printf("Error: %s\n", errResp["error"])
		return
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	fmt.Printf("✓ Mode switch initiated\n")
	fmt.Printf("  Node ID: %s\n", nodeID)
	fmt.Printf("  Target Mode: %s\n", toMode)

	if wait {
		fmt.Printf("\nWaiting for mode switch to complete (timeout: %ds)...\n", timeout)

		start := time.Now()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				progress, err := getModeProgress(cmd, nodeID)
				if err != nil {
					fmt.Printf("Error checking progress: %v\n", err)
					return
				}

				elapsed := int(time.Since(start).Seconds())
				fmt.Printf("[%ds] %s (%d/%d steps)\n",
					elapsed,
					progress["current_step"],
					int(progress["completed_steps"].(float64)),
					int(progress["total_steps"].(float64)))

				status := progress["status"].(string)
				if status == "completed" {
					fmt.Printf("\n✓ Mode switch completed successfully!\n")
					return
				} else if status == "failed" {
					fmt.Printf("\n✗ Mode switch failed: %s\n", progress["error"])
					return
				}

				if elapsed >= timeout {
					fmt.Printf("\n✗ Timeout waiting for mode switch to complete\n")
					return
				}

			case <-time.After(time.Duration(timeout) * time.Second):
				fmt.Printf("\n✗ Timeout\n")
				return
			}
		}
	} else {
		fmt.Printf("\nTo check progress: reignxctl mode progress %s\n", nodeID)
	}
}

func runModeProgress(cmd *cobra.Command, args []string) {
	nodeID := args[0]

	progress, err := getModeProgress(cmd, nodeID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Mode Switch Progress\n")
	fmt.Printf("====================\n")
	fmt.Printf("Node ID:        %s\n", progress["node_id"])
	fmt.Printf("From Mode:      %s\n", progress["from_mode"])
	fmt.Printf("To Mode:        %s\n", progress["to_mode"])
	fmt.Printf("Status:         %s\n", progress["status"])
	fmt.Printf("Current Step:   %s\n", progress["current_step"])
	fmt.Printf("Progress:       %d/%d steps\n",
		int(progress["completed_steps"].(float64)),
		int(progress["total_steps"].(float64)))

	if errMsg, ok := progress["error"]; ok && errMsg != "" {
		fmt.Printf("Error:          %s\n", errMsg)
	}
}

func getModeProgress(cmd *cobra.Command, nodeID string) (map[string]interface{}, error) {
	apiURL, _ := cmd.Root().PersistentFlags().GetString("server")

	url := fmt.Sprintf("%s/api/v1/nodes/%s/switch-progress", apiURL, nodeID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf(errResp["error"])
	}

	var progress map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&progress); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return progress, nil
}
