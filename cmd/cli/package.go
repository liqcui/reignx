package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func packageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "package",
		Short: "Manage packages",
		Long:  "Deploy packages to servers",
	}

	cmd.AddCommand(packageDeployCmd())

	return cmd
}

func packageDeployCmd() *cobra.Command {
	var (
		servers        string
		packageType    string
		packageFile    string
		installPath    string
		installCommand string
		postInstall    string
	)

	cmd := &cobra.Command{
		Use:   "deploy [server-id]",
		Short: "Deploy package to server(s)",
		Long: `Deploy RPM, DEB, TAR, or ZIP packages to servers.

Examples:
  # Deploy RPM package
  bm-cli package deploy server-001 \
    --file nginx-1.20.1.rpm \
    --type rpm

  # Deploy TAR archive with custom install path
  bm-cli package deploy server-001 \
    --file myapp-1.0.0.tar.gz \
    --type tar \
    --install-path /opt/myapp \
    --post-install "systemctl start myapp"

  # Deploy to multiple servers
  bm-cli package deploy --servers server-001,server-002 \
    --file app.rpm --type rpm`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var serverIDs []string

			if len(args) > 0 {
				serverIDs = []string{args[0]}
			} else if servers != "" {
				serverIDs = strings.Split(servers, ",")
			} else {
				return fmt.Errorf("must specify server-id or --servers")
			}

			if packageFile == "" {
				return fmt.Errorf("--file is required")
			}

			if packageType == "" {
				// Auto-detect package type from extension
				ext := strings.ToLower(filepath.Ext(packageFile))
				switch ext {
				case ".rpm":
					packageType = "rpm"
				case ".deb":
					packageType = "deb"
				case ".tar", ".gz", ".tgz":
					packageType = "tar"
				case ".zip":
					packageType = "zip"
				default:
					return fmt.Errorf("cannot auto-detect package type, use --type")
				}
			}

			return executePackageDeploy(serverIDs, packageFile, packageType, installPath, installCommand, postInstall)
		},
	}

	cmd.Flags().StringVar(&servers, "servers", "", "Comma-separated list of server IDs")
	cmd.Flags().StringVar(&packageFile, "file", "", "Package file path [required]")
	cmd.Flags().StringVar(&packageType, "type", "", "Package type (rpm, deb, tar, zip) [auto-detected]")
	cmd.Flags().StringVar(&installPath, "install-path", "", "Installation path (for tar/zip)")
	cmd.Flags().StringVar(&installCommand, "install-command", "", "Custom install command")
	cmd.Flags().StringVar(&postInstall, "post-install", "", "Post-install commands")

	return cmd
}

func executePackageDeploy(serverIDs []string, packageFile, packageType, installPath, installCommand, postInstall string) error {
	// Check if file exists
	if _, err := os.Stat(packageFile); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", packageFile)
	}

	client := &http.Client{}
	successCount := 0
	failCount := 0

	for _, serverID := range serverIDs {
		fmt.Printf("📦 Deploying %s to %s...\n", filepath.Base(packageFile), serverID)

		url := fmt.Sprintf("%s/api/v1/servers/%s/deploy", apiServer, serverID)

		// Create multipart form
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add file
		file, err := os.Open(packageFile)
		if err != nil {
			fmt.Printf("❌ %s: failed to open file: %v\n", serverID, err)
			failCount++
			continue
		}

		part, err := writer.CreateFormFile("file", filepath.Base(packageFile))
		if err != nil {
			file.Close()
			fmt.Printf("❌ %s: failed to create form file: %v\n", serverID, err)
			failCount++
			continue
		}

		if _, err := io.Copy(part, file); err != nil {
			file.Close()
			fmt.Printf("❌ %s: failed to copy file: %v\n", serverID, err)
			failCount++
			continue
		}
		file.Close()

		// Add form fields
		writer.WriteField("packageType", packageType)
		if installPath != "" {
			writer.WriteField("installPath", installPath)
		}
		if installCommand != "" {
			writer.WriteField("installCommand", installCommand)
		}
		if postInstall != "" {
			writer.WriteField("postInstall", postInstall)
		}

		writer.Close()

		// Create request
		req, err := http.NewRequest("POST", url, body)
		if err != nil {
			fmt.Printf("❌ %s: failed to create request: %v\n", serverID, err)
			failCount++
			continue
		}

		req.Header.Set("Content-Type", writer.FormDataContentType())
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		// Send request
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("❌ %s: failed to send request: %v\n", serverID, err)
			failCount++
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("❌ %s: API error (%d): %s\n", serverID, resp.StatusCode, string(respBody))
			failCount++
			continue
		}

		var result map[string]interface{}
		if err := json.Unmarshal(respBody, &result); err != nil {
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
		return fmt.Errorf("%d package deployment(s) failed", failCount)
	}

	return nil
}
