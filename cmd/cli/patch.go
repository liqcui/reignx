package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

func patchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patch",
		Short: "Manage patches",
		Long:  "Deploy patches to servers",
	}

	cmd.AddCommand(patchDeployCmd())

	return cmd
}

func patchDeployCmd() *cobra.Command {
	var (
		servers string
		filter  string
		patchIDs string
		reboot  bool
	)

	cmd := &cobra.Command{
		Use:   "deploy [server-id]",
		Short: "Deploy patches to server(s)",
		Long: `Deploy patches to one or more servers.

Examples:
  # Deploy all available patches to a server
  bm-cli patch deploy server-001

  # Deploy specific patches
  bm-cli patch deploy server-001 --patches CVE-2026-1234,CVE-2026-5678

  # Deploy patches with automatic reboot
  bm-cli patch deploy server-001 --reboot

  # Batch deploy to multiple servers
  bm-cli patch deploy --servers server-001,server-002,server-003`,
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

			var patches []string
			if patchIDs != "" {
				patches = strings.Split(patchIDs, ",")
			}

			return executePatchDeploy(serverIDs, patches, reboot)
		},
	}

	cmd.Flags().StringVar(&servers, "servers", "", "Comma-separated list of server IDs")
	cmd.Flags().StringVar(&filter, "filter", "", "Filter expression")
	cmd.Flags().StringVar(&patchIDs, "patches", "", "Comma-separated patch IDs (empty = all available)")
	cmd.Flags().BoolVar(&reboot, "reboot", false, "Reboot after patching if required")

	return cmd
}

type PatchDeployRequest struct {
	PatchIDs []string `json:"patch_ids,omitempty"`
	Reboot   bool     `json:"reboot_if_required"`
}

func executePatchDeploy(serverIDs []string, patches []string, reboot bool) error {
	client := &http.Client{}
	successCount := 0
	failCount := 0

	for _, serverID := range serverIDs {
		fmt.Printf("🔧 Deploying patches to %s...\n", serverID)

		url := fmt.Sprintf("%s/api/v1/servers/%s/patch", apiServer, serverID)

		reqBody := PatchDeployRequest{
			PatchIDs: patches,
			Reboot:   reboot,
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			fmt.Printf("❌ %s: failed to marshal request: %v\n", serverID, err)
			failCount++
			continue
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("❌ %s: failed to create request: %v\n", serverID, err)
			failCount++
			continue
		}

		req.Header.Set("Content-Type", "application/json")
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
		if jobID, ok := result["job_id"]; ok {
			fmt.Printf("   Job ID: %s\n", jobID)
		}
		successCount++
	}

	fmt.Printf("\nResults: %d succeeded, %d failed\n", successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("%d patch deployment(s) failed", failCount)
	}

	return nil
}
