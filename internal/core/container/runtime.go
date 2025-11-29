package container

import (
	"context"
	"io"
	"time"
)

// ContainerRuntime abstracts Docker operations
type ContainerRuntime interface {
	// Image operations
	PullImage(ctx context.Context, ref string, auth *RegistryAuth) error
	BuildImage(ctx context.Context, opts BuildOptions) (string, error)
	ListImages(ctx context.Context) ([]Image, error)
	RemoveImage(ctx context.Context, id string) error

	// Container operations
	CreateContainer(ctx context.Context, config ContainerConfig) (string, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string, timeout time.Duration) error
	RestartContainer(ctx context.Context, id string, timeout time.Duration) error
	RemoveContainer(ctx context.Context, id string, force bool) error
	InspectContainer(ctx context.Context, id string) (*ContainerInfo, error)
	ListContainers(ctx context.Context, filter ContainerFilter) ([]ContainerInfo, error)
	ContainerLogs(ctx context.Context, id string, opts LogOptions) (io.ReadCloser, error)
	WaitContainer(ctx context.Context, id string) (<-chan WaitResult, <-chan error)

	// Network operations
	CreateNetwork(ctx context.Context, name string, opts NetworkOptions) (string, error)
	RemoveNetwork(ctx context.Context, id string) error
	ConnectToNetwork(ctx context.Context, containerID, networkID string) error
	DisconnectFromNetwork(ctx context.Context, containerID, networkID string) error

	// Volume operations
	CreateVolume(ctx context.Context, name string, opts VolumeOptions) error
	RemoveVolume(ctx context.Context, name string) error
	ListVolumes(ctx context.Context) ([]Volume, error)

	// Health
	Ping(ctx context.Context) error
}

// RegistryAuth holds registry authentication
type RegistryAuth struct {
	Username string
	Password string
	Token    string
}

// BuildOptions for building images
type BuildOptions struct {
	ContextPath    string
	DockerfilePath string
	Tags           []string
	BuildArgs      map[string]string
	NoCache        bool
}

// Image represents a Docker image
type Image struct {
	ID      string
	Tags    []string
	Size    int64
	Created time.Time
}

// ContainerConfig for creating containers
type ContainerConfig struct {
	Name         string
	Image        string
	Env          map[string]string
	Labels       map[string]string
	Ports        []PortBinding
	Volumes      []VolumeMount
	Networks     []string
	Command      []string
	HealthCheck  *HealthCheckConfig
	RestartPolicy string
	Resources    *ResourceConfig
}

// PortBinding represents a port mapping
type PortBinding struct {
	ContainerPort int
	HostPort      int
	Protocol      string // tcp, udp
}

// VolumeMount represents a volume mount
type VolumeMount struct {
	Source   string
	Target   string
	ReadOnly bool
}

// HealthCheckConfig for container health checks
type HealthCheckConfig struct {
	Test        []string
	Interval    time.Duration
	Timeout     time.Duration
	Retries     int
	StartPeriod time.Duration
}

// ResourceConfig for container resources
type ResourceConfig struct {
	CPULimit    int64 // in nanocores
	MemoryLimit int64 // in bytes
}

// ContainerInfo holds container information
type ContainerInfo struct {
	ID      string
	Name    string
	Image   string
	Status  string
	State   string
	Created time.Time
	Ports   []PortBinding
	Labels  map[string]string
	Health  string
}

// ContainerFilter for listing containers
type ContainerFilter struct {
	All    bool
	Labels map[string]string
	Names  []string
}

// LogOptions for getting container logs
type LogOptions struct {
	Follow     bool
	Tail       string
	Since      time.Time
	Until      time.Time
	Timestamps bool
	Stdout     bool
	Stderr     bool
}

// WaitResult from waiting for container
type WaitResult struct {
	StatusCode int64
	Error      string
}

// NetworkOptions for creating networks
type NetworkOptions struct {
	Driver   string
	Internal bool
	Labels   map[string]string
}

// VolumeOptions for creating volumes
type VolumeOptions struct {
	Driver string
	Labels map[string]string
}

// Volume represents a Docker volume
type Volume struct {
	Name       string
	Driver     string
	Mountpoint string
	Labels     map[string]string
	CreatedAt  time.Time
}
