package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func loginCmd() *cobra.Command {
	var username string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to ReignX API server",
		Long: `Login to the ReignX API server and save authentication token.

Example:
  bm-cli login --username admin`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" {
				fmt.Print("Username: ")
				fmt.Scanln(&username)
			}

			fmt.Print("Password: ")
			passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			password := string(passwordBytes)

			// Send login request
			client := &http.Client{}
			url := fmt.Sprintf("%s/api/v1/auth/login", apiServer)

			loginReq := map[string]string{
				"username": username,
				"password": password,
			}

			jsonData, err := json.Marshal(loginReq)
			if err != nil {
				return fmt.Errorf("failed to marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("login failed (%d): %s", resp.StatusCode, string(body))
			}

			var result struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
				User         struct {
					ID       string `json:"id"`
					Username string `json:"username"`
					Role     string `json:"role"`
				} `json:"user"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			// Save token to config file
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}

			configDir := filepath.Join(homeDir, ".reignx")
			if err := os.MkdirAll(configDir, 0700); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}

			configFile := filepath.Join(configDir, "config.json")
			config := map[string]interface{}{
				"api_server":    apiServer,
				"access_token":  result.AccessToken,
				"refresh_token": result.RefreshToken,
				"username":      result.User.Username,
				"role":          result.User.Role,
			}

			configData, err := json.MarshalIndent(config, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}

			if err := os.WriteFile(configFile, configData, 0600); err != nil {
				return fmt.Errorf("failed to write config file: %w", err)
			}

			fmt.Printf("✅ Login successful!\n")
			fmt.Printf("   User: %s (%s)\n", result.User.Username, result.User.Role)
			fmt.Printf("   Token saved to: %s\n", configFile)
			fmt.Printf("\nYou can now use bm-cli commands without --token flag\n")

			return nil
		},
	}

	cmd.Flags().StringVar(&username, "username", "", "Username")

	return cmd
}

// loadToken loads saved authentication token from config file
func loadToken() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	configFile := filepath.Join(homeDir, ".reignx", "config.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return ""
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return ""
	}

	if accessToken, ok := config["access_token"].(string); ok {
		return accessToken
	}

	return ""
}
