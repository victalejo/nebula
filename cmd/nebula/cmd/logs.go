package cmd

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <app-name>",
	Short: "Stream application logs",
	Long: `Stream logs from an application in real-time.

Examples:
  nebula logs myapp
  nebula logs myapp -f
  nebula logs myapp --tail=100`,
	Args: cobra.ExactArgs(1),
	RunE: runLogs,
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	logsCmd.Flags().StringP("tail", "t", "100", "Number of lines to show from the end")
	logsCmd.Flags().StringP("service", "s", "", "Service name (for compose apps)")
}

func runLogs(cmd *cobra.Command, args []string) error {
	appName := args[0]
	follow, _ := cmd.Flags().GetBool("follow")
	tail, _ := cmd.Flags().GetString("tail")
	service, _ := cmd.Flags().GetString("service")

	// Build URL with query parameters
	url := fmt.Sprintf("/api/v1/apps/%s/logs?tail=%s", appName, tail)
	if follow {
		url += "&follow=true"
	}
	if service != "" {
		url += "&service=" + service
	}

	client := NewClient()

	// Make SSE request
	req, err := http.NewRequest("GET", client.baseURL+url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+client.token)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to get logs: %s", resp.Status)
	}

	fmt.Printf("Streaming logs for %s...\n\n", appName)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// Parse SSE format
		if strings.HasPrefix(line, "data: ") {
			logLine := strings.TrimPrefix(line, "data: ")
			fmt.Println(logLine)
		} else if strings.HasPrefix(line, "event: ") {
			// Handle events if needed
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading logs: %w", err)
	}

	return nil
}
