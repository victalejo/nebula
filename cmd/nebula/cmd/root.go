package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	serverURL string
	token     string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "nebula",
	Short: "Nebula - Lightweight PaaS for Docker deployments",
	Long: `Nebula is a lightweight Platform as a Service (PaaS) that supports
multiple deployment modes: Git, Docker Image, and Docker Compose.

Features:
  - Zero-downtime blue-green deployments
  - Automatic SSL via Caddy
  - Managed databases (Postgres, MySQL, Redis, MongoDB)
  - Real-time log streaming`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.nebula/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "", "Nebula server URL")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "Authentication token")

	viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server"))
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		configDir := filepath.Join(home, ".nebula")
		if err := os.MkdirAll(configDir, 0700); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("NEBULA")
	viper.AutomaticEnv()

	viper.ReadInConfig()
}

// GetServerURL returns the configured server URL
func GetServerURL() string {
	url := viper.GetString("server")
	if url == "" {
		url = "http://localhost:8080"
	}
	return url
}

// GetToken returns the configured token
func GetToken() string {
	return viper.GetString("token")
}
