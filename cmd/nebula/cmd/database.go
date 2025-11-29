package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:     "db",
	Aliases: []string{"database", "databases"},
	Short:   "Manage databases",
	Long:    `Create and manage Postgres, MySQL, Redis, and MongoDB databases.`,
}

var dbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all databases",
	RunE:  runDBList,
}

var dbCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new database",
	Long: `Create a new managed database.

Examples:
  nebula db create mydb --type postgres
  nebula db create cache --type redis
  nebula db create users --type mongodb`,
	Args: cobra.ExactArgs(1),
	RunE: runDBCreate,
}

var dbInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show database information",
	Args:  cobra.ExactArgs(1),
	RunE:  runDBInfo,
}

var dbDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a database",
	Args:  cobra.ExactArgs(1),
	RunE:  runDBDelete,
}

var dbCredentialsCmd = &cobra.Command{
	Use:     "credentials <name>",
	Aliases: []string{"creds"},
	Short:   "Show database credentials",
	Args:    cobra.ExactArgs(1),
	RunE:    runDBCredentials,
}

var dbRestartCmd = &cobra.Command{
	Use:   "restart <name>",
	Short: "Restart a database",
	Args:  cobra.ExactArgs(1),
	RunE:  runDBRestart,
}

func init() {
	rootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(dbListCmd)
	dbCmd.AddCommand(dbCreateCmd)
	dbCmd.AddCommand(dbInfoCmd)
	dbCmd.AddCommand(dbDeleteCmd)
	dbCmd.AddCommand(dbCredentialsCmd)
	dbCmd.AddCommand(dbRestartCmd)

	dbCreateCmd.Flags().StringP("type", "t", "postgres", "Database type (postgres, mysql, redis, mongodb)")
	dbCreateCmd.Flags().StringP("version", "v", "", "Database version (e.g., 16 for postgres:16)")
}

type DatabaseResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Username      string `json:"username"`
	Password      string `json:"password,omitempty"`
	Database      string `json:"database"`
	ConnectionURL string `json:"connection_url,omitempty"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

func runDBList(cmd *cobra.Command, args []string) error {
	client := NewClient()

	resp, err := client.Get("/api/v1/databases")
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	var result struct {
		Data []DatabaseResponse `json:"data"`
	}
	if err := ParseResponse(resp, &result); err != nil {
		return err
	}

	if len(result.Data) == 0 {
		fmt.Println("No databases found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tSTATUS\tHOST\tPORT")
	for _, db := range result.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n", db.Name, db.Type, db.Status, db.Host, db.Port)
	}
	w.Flush()

	return nil
}

func runDBCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	dbType, _ := cmd.Flags().GetString("type")
	version, _ := cmd.Flags().GetString("version")

	client := NewClient()

	body := map[string]interface{}{
		"name": name,
		"type": dbType,
	}
	if version != "" {
		body["version"] = version
	}

	fmt.Printf("Creating %s database '%s'...\n", dbType, name)

	resp, err := client.Post("/api/v1/databases", body)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	var result struct {
		Data DatabaseResponse `json:"data"`
	}
	if err := ParseResponse(resp, &result); err != nil {
		return err
	}

	db := result.Data
	fmt.Println("\nDatabase created successfully!")
	fmt.Println()
	fmt.Println("Connection Details:")
	fmt.Printf("  Host:     %s\n", db.Host)
	fmt.Printf("  Port:     %d\n", db.Port)
	fmt.Printf("  Username: %s\n", db.Username)
	fmt.Printf("  Password: %s\n", db.Password)
	fmt.Printf("  Database: %s\n", db.Database)
	fmt.Println()
	fmt.Printf("Connection URL:\n  %s\n", db.ConnectionURL)

	return nil
}

func runDBInfo(cmd *cobra.Command, args []string) error {
	name := args[0]
	client := NewClient()

	resp, err := client.Get("/api/v1/databases/" + name)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	var result struct {
		Data DatabaseResponse `json:"data"`
	}
	if err := ParseResponse(resp, &result); err != nil {
		return err
	}

	db := result.Data
	fmt.Printf("Name:     %s\n", db.Name)
	fmt.Printf("Type:     %s\n", db.Type)
	fmt.Printf("Status:   %s\n", db.Status)
	fmt.Printf("Host:     %s\n", db.Host)
	fmt.Printf("Port:     %d\n", db.Port)
	fmt.Printf("Username: %s\n", db.Username)
	fmt.Printf("Database: %s\n", db.Database)
	fmt.Printf("Created:  %s\n", db.CreatedAt)

	return nil
}

func runDBDelete(cmd *cobra.Command, args []string) error {
	name := args[0]
	client := NewClient()

	fmt.Printf("Deleting database '%s'...\n", name)

	resp, err := client.Delete("/api/v1/databases/" + name)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	var result struct {
		Message string `json:"message"`
	}
	if err := ParseResponse(resp, &result); err != nil {
		return err
	}

	fmt.Println("Database deleted successfully")
	return nil
}

func runDBCredentials(cmd *cobra.Command, args []string) error {
	name := args[0]
	client := NewClient()

	resp, err := client.Get("/api/v1/databases/" + name + "/credentials")
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	var result struct {
		Data struct {
			Host          string `json:"host"`
			Port          int    `json:"port"`
			Username      string `json:"username"`
			Password      string `json:"password"`
			Database      string `json:"database"`
			ConnectionURL string `json:"connection_url"`
		} `json:"data"`
	}
	if err := ParseResponse(resp, &result); err != nil {
		return err
	}

	creds := result.Data
	fmt.Printf("Host:     %s\n", creds.Host)
	fmt.Printf("Port:     %d\n", creds.Port)
	fmt.Printf("Username: %s\n", creds.Username)
	fmt.Printf("Password: %s\n", creds.Password)
	fmt.Printf("Database: %s\n", creds.Database)
	fmt.Println()
	fmt.Printf("Connection URL:\n%s\n", creds.ConnectionURL)

	return nil
}

func runDBRestart(cmd *cobra.Command, args []string) error {
	name := args[0]
	client := NewClient()

	fmt.Printf("Restarting database '%s'...\n", name)

	resp, err := client.Post("/api/v1/databases/"+name+"/restart", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	var result struct {
		Message string `json:"message"`
	}
	if err := ParseResponse(resp, &result); err != nil {
		return err
	}

	fmt.Println("Database restarted successfully")
	return nil
}
