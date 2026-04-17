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

func installOSCmd() *cobra.Command {
	var (
		servers    string
		filter     string
		osType     string
		osVersion  string
		rootPass   string
		sshKey     string
		partition  string
		packages   string
		skipConfirm bool
	)

	cmd := &cobra.Command{
		Use:   "install-os [server-id]",
		Short: "Install operating system on server(s)",
		Long: `Install or reinstall operating system on one or more servers using PXE boot.

This command will:
1. Configure PXE boot with specified OS
2. Set server to boot from network (via IPMI)
3. Power cycle the server to begin installation
4. Monitor installation progress

Examples:
  # Install Ubuntu 22.04 on single server
  bm-cli install-os server-001 --os ubuntu --version 22.04

  # Batch install CentOS on multiple servers
  bm-cli install-os --servers server-001,server-002,server-003 \
    --os centos --version 8 \
    --root-password 'SecurePass123!'

  # Install with custom partitioning
  bm-cli install-os server-001 --os ubuntu --version 22.04 \
    --partition '/=20G,/home=50G,swap=8G'

  # Install with SSH key and packages
  bm-cli install-os server-001 --os ubuntu --version 22.04 \
    --ssh-key ~/.ssh/id_rsa.pub \
    --packages 'nginx,mysql-server,docker.io'`,
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

			// Validate required parameters
			if osType == "" || osVersion == "" {
				return fmt.Errorf("--os and --version are required")
			}

			// Confirm before proceeding
			if !skipConfirm {
				fmt.Printf("⚠️  WARNING: This will REINSTALL the operating system on %d server(s):\n", len(serverIDs))
				for _, id := range serverIDs {
					fmt.Printf("   - %s\n", id)
				}
				fmt.Printf("\nOS: %s %s\n", osType, osVersion)
				fmt.Printf("\nThis will ERASE ALL DATA on these servers!\n")
				fmt.Printf("\nType 'yes' to continue: ")

				var response string
				fmt.Scanln(&response)
				if response != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			// Parse packages
			var pkgList []string
			if packages != "" {
				pkgList = strings.Split(packages, ",")
			}

			// Execute OS installation for each server
			return executeOSInstallation(serverIDs, osType, osVersion, rootPass, sshKey, partition, pkgList)
		},
	}

	cmd.Flags().StringVar(&servers, "servers", "", "Comma-separated list of server IDs")
	cmd.Flags().StringVar(&filter, "filter", "", "Filter expression to select servers")
	cmd.Flags().StringVar(&osType, "os", "", "Operating system (ubuntu, debian, centos, rhel) [required]")
	cmd.Flags().StringVar(&osVersion, "version", "", "OS version (e.g., 22.04, 8, 9) [required]")
	cmd.Flags().StringVar(&rootPass, "root-password", "", "Root password (will prompt if not provided)")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH public key file")
	cmd.Flags().StringVar(&partition, "partition", "", "Partition layout (e.g., '/=20G,/home=50G,swap=8G')")
	cmd.Flags().StringVar(&packages, "packages", "", "Comma-separated list of packages to install")
	cmd.Flags().BoolVar(&skipConfirm, "yes", false, "Skip confirmation prompt")

	return cmd
}

type OSInstallRequest struct {
	OSType     string   `json:"os_type"`
	OSVersion  string   `json:"os_version"`
	RootPass   string   `json:"root_password,omitempty"`
	SSHKey     string   `json:"ssh_key,omitempty"`
	Partition  string   `json:"partition,omitempty"`
	Packages   []string `json:"packages,omitempty"`
}

func executeOSInstallation(serverIDs []string, osType, osVersion, rootPass, sshKey, partition string, packages []string) error {
	client := &http.Client{}
	successCount := 0
	failCount := 0

	// Read SSH key if provided
	var sshKeyContent string
	if sshKey != "" {
		// TODO: Read SSH key file
		sshKeyContent = sshKey
	}

	for _, serverID := range serverIDs {
		fmt.Printf("📦 Installing %s %s on %s...\n", osType, osVersion, serverID)

		url := fmt.Sprintf("%s/api/v1/servers/%s/install-os", apiServer, serverID)

		reqBody := OSInstallRequest{
			OSType:    osType,
			OSVersion: osVersion,
			RootPass:  rootPass,
			SSHKey:    sshKeyContent,
			Partition: partition,
			Packages:  packages,
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
			fmt.Printf("   Monitor progress: bm-cli job show %s\n", jobID)
		}
		successCount++
	}

	fmt.Printf("\n========================================\n")
	fmt.Printf("Results: %d succeeded, %d failed\n", successCount, failCount)
	fmt.Printf("========================================\n")

	if successCount > 0 {
		fmt.Printf("\n📋 Next Steps:\n")
		fmt.Printf("1. Servers will now PXE boot and begin OS installation\n")
		fmt.Printf("2. Installation typically takes 10-30 minutes\n")
		fmt.Printf("3. Monitor progress: bm-cli server show <server-id>\n")
		fmt.Printf("4. Check installation logs if needed\n")
	}

	if failCount > 0 {
		return fmt.Errorf("%d installation(s) failed to start", failCount)
	}

	return nil
}
