package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Docker   DockerConfig   `mapstructure:"docker"`
	Caddy    CaddyConfig    `mapstructure:"caddy"`
	Log      LogConfig      `mapstructure:"log"`
	Auth     AuthConfig     `mapstructure:"auth"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

// DockerConfig holds Docker client configuration
type DockerConfig struct {
	Host       string `mapstructure:"host"`
	APIVersion string `mapstructure:"api_version"`
	Network    string `mapstructure:"network"`
}

// CaddyConfig holds Caddy proxy configuration
type CaddyConfig struct {
	AdminAPI string `mapstructure:"admin_api"`
	Network  string `mapstructure:"network"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret     string `mapstructure:"jwt_secret"`
	TokenDuration int    `mapstructure:"token_duration"` // in hours
	AdminUsername string `mapstructure:"admin_username"`
	AdminPassword string `mapstructure:"admin_password"`
}

// Load reads configuration from file and environment
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("database.path", "./data/nebula.db")
	v.SetDefault("docker.host", "unix:///var/run/docker.sock")
	v.SetDefault("docker.network", "nebula-network")
	v.SetDefault("caddy.admin_api", "http://localhost:2019")
	v.SetDefault("caddy.network", "web")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("auth.token_duration", 24)
	v.SetDefault("auth.admin_username", "admin")
	v.SetDefault("auth.admin_password", "admin")

	// Config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("/etc/nebula")
	}

	// Environment variables
	v.SetEnvPrefix("NEBULA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// LoadDefault loads configuration with defaults
func LoadDefault() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Path: "./data/nebula.db",
		},
		Docker: DockerConfig{
			Host:    "unix:///var/run/docker.sock",
			Network: "nebula-network",
		},
		Caddy: CaddyConfig{
			AdminAPI: "http://localhost:2019",
			Network:  "web",
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
		Auth: AuthConfig{
			JWTSecret:     "change-me-in-production",
			TokenDuration: 24,
			AdminUsername: "admin",
			AdminPassword: "admin",
		},
	}
}
