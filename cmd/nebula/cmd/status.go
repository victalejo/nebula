package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <app-name>",
	Short: "Show application status",
	Long:  `Show the current status of an application and its deployments.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	appName := args[0]

	client := NewClient()

	// Get app info
	appResp, err := client.Get("/api/v1/apps/" + appName)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	var appResult struct {
		Data App `json:"data"`
	}
	if err := ParseResponse(appResp, &appResult); err != nil {
		return err
	}

	app := appResult.Data

	// Get deployments
	deploymentsResp, err := client.Get(fmt.Sprintf("/api/v1/apps/%s/deployments", appName))
	if err != nil {
		return fmt.Errorf("failed to get deployments: %w", err)
	}

	var deploymentsResult struct {
		Data []Deployment `json:"data"`
	}
	if err := ParseResponse(deploymentsResp, &deploymentsResult); err != nil {
		return err
	}

	// Display status
	fmt.Printf("App: %s\n", app.Name)
	fmt.Printf("Mode: %s\n", app.DeploymentMode)
	if app.Domain != "" {
		fmt.Printf("URL: https://%s\n", app.Domain)
	}
	fmt.Println()

	// Find current running deployment
	var currentDeployment *Deployment
	for i := range deploymentsResult.Data {
		if deploymentsResult.Data[i].Status == "running" {
			currentDeployment = &deploymentsResult.Data[i]
			break
		}
	}

	if currentDeployment != nil {
		fmt.Println("Current Deployment:")
		fmt.Printf("  Version: %s\n", currentDeployment.Version)
		fmt.Printf("  Slot: %s\n", currentDeployment.Slot)
		fmt.Printf("  Status: %s\n", currentDeployment.Status)
		fmt.Printf("  Deployed: %s\n", currentDeployment.CreatedAt)
	} else {
		fmt.Println("No running deployment")
	}

	// Show recent deployments
	if len(deploymentsResult.Data) > 0 {
		fmt.Println("\nRecent Deployments:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  VERSION\tSLOT\tSTATUS\tCREATED")

		limit := 5
		if len(deploymentsResult.Data) < limit {
			limit = len(deploymentsResult.Data)
		}

		for i := 0; i < limit; i++ {
			d := deploymentsResult.Data[i]
			statusIcon := getStatusIcon(d.Status)
			fmt.Fprintf(w, "  %s\t%s\t%s %s\t%s\n", d.Version, d.Slot, statusIcon, d.Status, d.CreatedAt)
		}
		w.Flush()
	}

	return nil
}

func getStatusIcon(status string) string {
	switch status {
	case "running":
		return "●"
	case "stopped":
		return "○"
	case "failed":
		return "✗"
	case "pending", "preparing", "deploying":
		return "◐"
	default:
		return "?"
	}
}
