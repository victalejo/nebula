package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "Manage applications",
	Long:  `List, create, and manage applications on Nebula.`,
}

var appsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all applications",
	RunE:  runAppsList,
}

var appsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new application",
	Args:  cobra.ExactArgs(1),
	RunE:  runAppsCreate,
}

var appsInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show application details",
	Args:  cobra.ExactArgs(1),
	RunE:  runAppsInfo,
}

var appsDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an application",
	Args:  cobra.ExactArgs(1),
	RunE:  runAppsDelete,
}

func init() {
	rootCmd.AddCommand(appsCmd)
	appsCmd.AddCommand(appsListCmd)
	appsCmd.AddCommand(appsCreateCmd)
	appsCmd.AddCommand(appsInfoCmd)
	appsCmd.AddCommand(appsDeleteCmd)

	appsCreateCmd.Flags().StringP("mode", "m", "docker_image", "Deployment mode (git, docker_image, docker_compose)")
	appsCreateCmd.Flags().StringP("domain", "d", "", "Custom domain for the application")
}

// App represents an application
type App struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	DeploymentMode string            `json:"deployment_mode"`
	Domain         string            `json:"domain"`
	Environment    map[string]string `json:"environment"`
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
}

func runAppsList(cmd *cobra.Command, args []string) error {
	client := NewClient()
	resp, err := client.Get("/api/v1/apps")
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	var result struct {
		Data []App `json:"data"`
	}
	if err := ParseResponse(resp, &result); err != nil {
		return err
	}

	if len(result.Data) == 0 {
		fmt.Println("No applications found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tMODE\tDOMAIN\tCREATED")
	for _, app := range result.Data {
		domain := app.Domain
		if domain == "" {
			domain = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", app.Name, app.DeploymentMode, domain, app.CreatedAt)
	}
	w.Flush()

	return nil
}

func runAppsCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	mode, _ := cmd.Flags().GetString("mode")
	domain, _ := cmd.Flags().GetString("domain")

	client := NewClient()
	resp, err := client.Post("/api/v1/apps", map[string]string{
		"name":            name,
		"deployment_mode": mode,
		"domain":          domain,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	var result struct {
		Data App `json:"data"`
	}
	if err := ParseResponse(resp, &result); err != nil {
		return err
	}

	fmt.Printf("✓ Application '%s' created\n", result.Data.Name)
	fmt.Printf("  ID: %s\n", result.Data.ID)
	fmt.Printf("  Mode: %s\n", result.Data.DeploymentMode)
	if result.Data.Domain != "" {
		fmt.Printf("  Domain: %s\n", result.Data.Domain)
	}

	return nil
}

func runAppsInfo(cmd *cobra.Command, args []string) error {
	name := args[0]

	client := NewClient()
	resp, err := client.Get("/api/v1/apps/" + name)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	var result struct {
		Data App `json:"data"`
	}
	if err := ParseResponse(resp, &result); err != nil {
		return err
	}

	app := result.Data
	fmt.Printf("Name: %s\n", app.Name)
	fmt.Printf("ID: %s\n", app.ID)
	fmt.Printf("Mode: %s\n", app.DeploymentMode)
	fmt.Printf("Domain: %s\n", app.Domain)
	fmt.Printf("Created: %s\n", app.CreatedAt)
	fmt.Printf("Updated: %s\n", app.UpdatedAt)

	if len(app.Environment) > 0 {
		fmt.Println("\nEnvironment Variables:")
		envJSON, _ := json.MarshalIndent(app.Environment, "  ", "  ")
		fmt.Printf("  %s\n", envJSON)
	}

	return nil
}

func runAppsDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	client := NewClient()
	resp, err := client.Delete("/api/v1/apps/" + name)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := ParseResponse(resp, nil); err != nil {
		return err
	}

	fmt.Printf("✓ Application '%s' deleted\n", name)
	return nil
}
