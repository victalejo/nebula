package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy an application",
	Long:  `Deploy an application using Git, Docker Image, or Docker Compose.`,
}

var deployImageCmd = &cobra.Command{
	Use:   "image <app-name>",
	Short: "Deploy from a Docker image",
	Long: `Deploy an application from a Docker image.

Examples:
  nebula deploy image myapp --image=nginx:alpine --port=80
  nebula deploy image myapp --image=ghcr.io/user/app:latest --port=3000`,
	Args: cobra.ExactArgs(1),
	RunE: runDeployImage,
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.AddCommand(deployImageCmd)

	deployImageCmd.Flags().StringP("image", "i", "", "Docker image to deploy (required)")
	deployImageCmd.Flags().IntP("port", "p", 0, "Container port to expose (required)")
	deployImageCmd.Flags().StringP("registry", "r", "", "Registry URL")
	deployImageCmd.Flags().StringP("registry-user", "", "", "Registry username")
	deployImageCmd.Flags().StringP("registry-password", "", "", "Registry password")
	deployImageCmd.Flags().StringSliceP("env", "e", []string{}, "Environment variables (KEY=VALUE)")

	deployImageCmd.MarkFlagRequired("image")
	deployImageCmd.MarkFlagRequired("port")
}

// Deployment represents a deployment response
type Deployment struct {
	ID        string `json:"id"`
	AppID     string `json:"app_id"`
	Version   string `json:"version"`
	Slot      string `json:"slot"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func runDeployImage(cmd *cobra.Command, args []string) error {
	appName := args[0]

	image, _ := cmd.Flags().GetString("image")
	port, _ := cmd.Flags().GetInt("port")
	registry, _ := cmd.Flags().GetString("registry")
	registryUser, _ := cmd.Flags().GetString("registry-user")
	registryPassword, _ := cmd.Flags().GetString("registry-password")
	envVars, _ := cmd.Flags().GetStringSlice("env")

	// Parse environment variables
	env := make(map[string]string)
	for _, e := range envVars {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	// Build request body
	body := map[string]interface{}{
		"image": image,
		"port":  port,
	}

	if registry != "" {
		body["registry"] = registry
	}

	if registryUser != "" || registryPassword != "" {
		body["registry_auth"] = map[string]string{
			"username": registryUser,
			"password": registryPassword,
		}
	}

	if len(env) > 0 {
		body["environment"] = env
	}

	fmt.Printf("Deploying %s from image %s...\n", appName, image)

	client := NewClient()
	resp, err := client.Post(fmt.Sprintf("/api/v1/apps/%s/deploy/image", appName), body)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	var result struct {
		Data    Deployment `json:"data"`
		Message string     `json:"message"`
	}
	if err := ParseResponse(resp, &result); err != nil {
		return err
	}

	fmt.Printf("✓ Deployment started\n")
	fmt.Printf("  ID: %s\n", result.Data.ID)
	fmt.Printf("  Version: %s\n", result.Data.Version)
	fmt.Printf("  Slot: %s\n", result.Data.Slot)
	fmt.Printf("  Status: %s\n", result.Data.Status)
	fmt.Println("\nUse 'nebula status " + appName + "' to check deployment progress")

	return nil
}
