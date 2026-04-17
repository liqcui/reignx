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

func jobCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job",
		Short: "Manage jobs",
		Long:  "Create, list, and monitor jobs",
	}

	cmd.AddCommand(jobListCmd())
	cmd.AddCommand(jobShowCmd())
	cmd.AddCommand(jobCancelCmd())

	return cmd
}

func jobListCmd() *cobra.Command {
	var (
		status string
		limit  int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := &http.Client{}
			url := fmt.Sprintf("%s/api/v1/jobs", apiServer)

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}

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
				Jobs  []map[string]interface{} `json:"jobs"`
				Total int                      `json:"total"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tTYPE\tSTATUS\tPROGRESS\tTARGETS\tCREATED")
			fmt.Fprintln(w, "--\t----\t----\t------\t--------\t-------\t-------")

			for _, job := range result.Jobs {
				progress := 0
				if p, ok := job["progress"].(float64); ok {
					progress = int(p)
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d%%\t%v\t%s\n",
					job["id"],
					job["name"],
					job["type"],
					job["status"],
					progress,
					job["target_servers"],
					job["created_at"],
				)
			}

			w.Flush()
			fmt.Printf("\nTotal: %d jobs\n", result.Total)

			return nil
		},
	}

	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of jobs to return")

	return cmd
}

func jobShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <job-id>",
		Short: "Show job details and progress",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobID := args[0]

			client := &http.Client{}
			url := fmt.Sprintf("%s/api/v1/jobs/%s", apiServer, jobID)

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

			var job map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			fmt.Printf("Job Details\n")
			fmt.Printf("===========\n\n")
			fmt.Printf("ID:          %s\n", job["id"])
			fmt.Printf("Name:        %s\n", job["name"])
			fmt.Printf("Type:        %s\n", job["type"])
			fmt.Printf("Status:      %s\n", job["status"])
			fmt.Printf("Priority:    %s\n", job["priority"])
			fmt.Printf("Progress:    %v%%\n", job["progress"])
			fmt.Printf("\nTarget Servers: %v\n", job["target_servers"])
			fmt.Printf("Completed:      %v\n", job["completed_tasks"])
			fmt.Printf("Failed:         %v\n", job["failed_tasks"])
			fmt.Printf("Pending:        %v\n", job["pending_tasks"])
			fmt.Printf("\nCreated:     %s\n", job["created_at"])
			fmt.Printf("Started:     %s\n", job["started_at"])
			if completed, ok := job["completed_at"]; ok && completed != nil {
				fmt.Printf("Completed:   %s\n", completed)
			}
			fmt.Printf("Created By:  %s\n", job["created_by"])

			if params, ok := job["parameters"].(map[string]interface{}); ok && len(params) > 0 {
				fmt.Printf("\nParameters:\n")
				for k, v := range params {
					fmt.Printf("  %s: %v\n", k, v)
				}
			}

			return nil
		},
	}
}

func jobCancelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <job-id>",
		Short: "Cancel a running job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobID := args[0]

			client := &http.Client{}
			url := fmt.Sprintf("%s/api/v1/jobs/%s/cancel", apiServer, jobID)

			req, err := http.NewRequest("DELETE", url, nil)
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

			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
			}

			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			fmt.Printf("✅ %s\n", result["message"])

			return nil
		},
	}
}
