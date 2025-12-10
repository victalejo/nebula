package deployer

import (
	"time"
)

// DeploymentMode represents the type of deployment
type DeploymentMode string

const (
	ModeGit     DeploymentMode = "git"
	ModeImage   DeploymentMode = "docker_image"
	ModeCompose DeploymentMode = "docker_compose"
)

// Slot represents blue or green deployment slot
type Slot string

const (
	SlotBlue  Slot = "blue"
	SlotGreen Slot = "green"
)

// Opposite returns the opposite slot
func (s Slot) Opposite() Slot {
	if s == SlotBlue {
		return SlotGreen
	}
	return SlotBlue
}

// DeploymentStatus represents the status of a deployment
type DeploymentStatus string

const (
	StatusPending   DeploymentStatus = "pending"
	StatusPreparing DeploymentStatus = "preparing"
	StatusDeploying DeploymentStatus = "deploying"
	StatusRunning   DeploymentStatus = "running"
	StatusFailed    DeploymentStatus = "failed"
	StatusStopped   DeploymentStatus = "stopped"
)

// Application represents an application in Nebula
type Application struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	DeploymentMode DeploymentMode    `json:"deployment_mode"`
	Domain         string            `json:"domain"`
	Environment    map[string]string `json:"environment"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// Deployment represents a deployment of an application
type Deployment struct {
	ID           string           `json:"id"`
	AppID        string           `json:"app_id"`
	Version      string           `json:"version"`
	Slot         Slot             `json:"slot"`
	Status       DeploymentStatus `json:"status"`
	SourceConfig SourceConfig     `json:"source_config"`
	Environment  map[string]string `json:"environment"`
	ContainerIDs []string         `json:"container_ids"`
	ErrorMessage string           `json:"error_message,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	StartedAt    *time.Time       `json:"started_at,omitempty"`
	FinishedAt   *time.Time       `json:"finished_at,omitempty"`
}

// SourceConfig holds mode-specific deployment configuration
type SourceConfig struct {
	// Git mode
	GitURL         string            `json:"git_url,omitempty"`
	GitBranch      string            `json:"git_branch,omitempty"`
	GitCommit      string            `json:"git_commit,omitempty"`
	DockerfilePath string            `json:"dockerfile_path,omitempty"`
	BuildArgs      map[string]string `json:"build_args,omitempty"`

	// Docker Image mode
	Image        string        `json:"image,omitempty"`
	RegistryAuth *RegistryAuth `json:"registry_auth,omitempty"`
	PullPolicy   string        `json:"pull_policy,omitempty"`
	Port         int           `json:"port,omitempty"`

	// Docker Compose mode
	ComposeContent string   `json:"compose_content,omitempty"`
	Services       []string `json:"services,omitempty"`
}

// RegistryAuth holds registry authentication
type RegistryAuth struct {
	Registry string `json:"registry,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
}

// Route represents a routing configuration
type Route struct {
	ID         string `json:"id"`
	AppID      string `json:"app_id"`
	Domain     string `json:"domain"`
	ActiveSlot Slot   `json:"active_slot"`
	SSLEnabled bool   `json:"ssl_enabled"`
	BluePort   int    `json:"blue_port,omitempty"`
	GreenPort  int    `json:"green_port,omitempty"`
}

// HealthCheckConfig holds custom health check configuration
type HealthCheckConfig struct {
	// Test command for Docker health check (e.g., ["CMD", "mysqladmin", "ping"])
	// If nil, no Docker health check is configured (for databases)
	Test []string
	// Whether to skip HTTP-based health check in deployer
	SkipHTTPCheck bool
	// Extended timeout for services that take longer to start (e.g., databases)
	MaxAttempts int
	Interval    time.Duration
}

// DeploymentSpec contains all information needed for deployment
type DeploymentSpec struct {
	AppID       string
	AppName     string
	ServiceID   string
	App         *Application
	Source      SourceConfig
	Environment map[string]string
	EnvVars     map[string]string // Alias for Environment
	TargetSlot  Slot
	Slot        Slot // Alias for TargetSlot

	// HealthCheck configuration (optional, uses defaults if nil)
	HealthCheck *HealthCheckConfig

	// Convenience accessors (populated from Source)
	GitRepo     string
	GitBranch   string
	ComposeFile string
	Image       string
	ImageTag    string
}

// DeploymentResult contains the result of a deployment
type DeploymentResult struct {
	DeploymentID string
	ContainerIDs []string
	Ports        map[string]int // container name -> exposed port
	Port         int            // Primary port for single container deployments
	Version      string
}

// PrepareResult contains the result of preparation phase
type PrepareResult struct {
	ImageID   string
	ImageTag  string
	BuildLogs string
}

// HealthCheckResult contains health check results
type HealthCheckResult struct {
	Healthy bool
	Message string
	Checks  []HealthCheck
}

// HealthCheck represents a single health check
type HealthCheck struct {
	Name    string
	Passed  bool
	Message string
}

// ResourceSpec defines resource limits
type ResourceSpec struct {
	CPULimit    string `json:"cpu_limit,omitempty"`
	MemoryLimit string `json:"memory_limit,omitempty"`
}
