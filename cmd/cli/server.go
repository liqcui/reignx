package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func serverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Manage servers",
		Long:  "List, view, and manage server inventory",
	}

	cmd.AddCommand(serverListCmd())
	cmd.AddCommand(serverShowCmd())

	return cmd
}

func serverListCmd() *cobra.Command {
	var (
		status string
		limit  int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := &http.Client{}
			url := fmt.Sprintf("%s/api/v1/servers", apiServer)

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}

			// Add query parameters
			q := req.URL.Query()
			if status != "" {
				q.Add("status", status)
			}
			if limit > 0 {
				q.Add("limit", fmt.Sprintf("%d", limit))
			}
			req.URL.RawQuery = q.Encode()

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
			}

			var result struct {
				Servers []map[string]interface{} `json:"servers"`
				Total   int                      `json:"total"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			// Print table
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tHOSTNAME\tIP ADDRESS\tOS\tSTATUS\tMODE\tLAST SEEN")
			fmt.Fprintln(w, "--\t--------\t----------\t--\t------\t----\t---------")

			for _, server := range result.Servers {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					server["id"],
					server["hostname"],
					server["ip"],
					server["os"],
					server["status"],
					server["mode"],
					server["last_seen"],
				)
			}

			w.Flush()
			fmt.Printf("\nTotal: %d servers\n", result.Total)

			return nil
		},
	}

	cmd.Flags().StringVar(&status, "status", "", "Filter by status (active, inactive, failed)")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of servers to return")

	return cmd
}

func serverShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <server-id>",
		Short: "Show server details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverID := args[0]

			client := &http.Client{}
			url := fmt.Sprintf("%s/api/v1/servers/%s", apiServer, serverID)

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

			var server map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&server); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			// Print server details
			fmt.Printf("Server Details\n")
			fmt.Printf("==============\n\n")
			fmt.Printf("ID:             %s\n", server["id"])
			fmt.Printf("Hostname:       %s\n", server["hostname"])
			fmt.Printf("IP Address:     %s\n", server["ip"])
			fmt.Printf("Operating System: %s\n", server["os"])
			fmt.Printf("Status:         %s\n", server["status"])
			fmt.Printf("Mode:           %s\n", server["mode"])
			fmt.Printf("Last Seen:      %s\n", server["last_seen"])

			if cpu, ok := server["cpu_usage"]; ok {
				fmt.Printf("\nResource Usage\n")
				fmt.Printf("--------------\n")
				fmt.Printf("CPU:            %.0f%%\n", cpu)
				fmt.Printf("Memory:         %.0f%%\n", server["memory_usage"])
				fmt.Printf("Disk:           %.0f%%\n", server["disk_usage"])
			}

			if uptime, ok := server["uptime"]; ok {
				fmt.Printf("\nSystem Info\n")
				fmt.Printf("-----------\n")
				fmt.Printf("Uptime:         %s\n", uptime)
				fmt.Printf("Packages:       %v\n", server["packages"])
				fmt.Printf("Pending Patches: %v\n", server["pending_patches"])
			}

			return nil
		},
	}
}
