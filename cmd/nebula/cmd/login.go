package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Nebula server",
	Long:  `Login to a Nebula server using username and password.`,
	RunE:  runLogin,
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from Nebula server",
	RunE:  runLogout,
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current logged in user",
	RunE:  runWhoami,
}

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(whoamiCmd)

	loginCmd.Flags().StringP("server", "s", "", "Server URL")
	loginCmd.Flags().StringP("username", "u", "", "Username")
	loginCmd.Flags().StringP("password", "p", "", "Password")
}

func runLogin(cmd *cobra.Command, args []string) error {
	server, _ := cmd.Flags().GetString("server")
	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")

	reader := bufio.NewReader(os.Stdin)

	// Get server URL
	if server == "" {
		server = viper.GetString("server")
		if server == "" {
			fmt.Print("Server URL: ")
			server, _ = reader.ReadString('\n')
			server = strings.TrimSpace(server)
		}
	}

	// Get username
	if username == "" {
		fmt.Print("Username: ")
		username, _ = reader.ReadString('\n')
		username = strings.TrimSpace(username)
	}

	// Get password
	if password == "" {
		fmt.Print("Password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		password = string(bytePassword)
		fmt.Println()
	}

	// Create client with server URL
	viper.Set("server", server)
	client := NewClient()

	// Make login request
	resp, err := client.Post("/api/v1/auth/login", map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	var result struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := ParseResponse(resp, &result); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Save configuration
	viper.Set("server", server)
	viper.Set("token", result.Token)
	viper.Set("username", username)
	viper.Set("expires_at", result.ExpiresAt)

	// Write config file
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".nebula", "config.yaml")
	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Successfully logged in as %s\n", username)
	fmt.Printf("  Server: %s\n", server)
	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	viper.Set("token", "")
	viper.Set("username", "")

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".nebula", "config.yaml")
	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("✓ Successfully logged out")
	return nil
}

func runWhoami(cmd *cobra.Command, args []string) error {
	username := viper.GetString("username")
	server := viper.GetString("server")
	token := viper.GetString("token")

	if token == "" {
		fmt.Println("Not logged in")
		return nil
	}

	fmt.Printf("Logged in as: %s\n", username)
	fmt.Printf("Server: %s\n", server)
	return nil
}
