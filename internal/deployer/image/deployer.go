package image

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"

	"github.com/victalejo/nebula/internal/core/container"
	"github.com/victalejo/nebula/internal/core/deployer"
	"github.com/victalejo/nebula/internal/core/logger"
)

// Deployer implements the Deployer interface for Docker images
type Deployer struct {
	runtime container.ContainerRuntime
	network string
	log     logger.Logger
}

// New creates a new image deployer
func New(runtime container.ContainerRuntime, network string, log logger.Logger) *Deployer {
	return &Deployer{
		runtime: runtime,
		network: network,
		log:     log,
	}
}

// Mode returns the deployment mode
func (d *Deployer) Mode() deployer.DeploymentMode {
	return deployer.ModeImage
}

// Validate validates the deployment spec
func (d *Deployer) Validate(ctx context.Context, spec *deployer.DeploymentSpec) error {
	if spec.Source.Image == "" {
		return fmt.Errorf("image is required")
	}
	if spec.Source.Port <= 0 {
		return fmt.Errorf("port is required and must be positive")
	}
	return nil
}

// Prepare pulls the Docker image
func (d *Deployer) Prepare(ctx context.Context, spec *deployer.DeploymentSpec) (*deployer.PrepareResult, error) {
	d.log.Info("pulling image",
		"image", spec.Source.Image,
	)

	var auth *container.RegistryAuth
	if spec.Source.RegistryAuth != nil {
		auth = &container.RegistryAuth{
			Username: spec.Source.RegistryAuth.Username,
			Password: spec.Source.RegistryAuth.Password,
		}
	}

	if err := d.runtime.PullImage(ctx, spec.Source.Image, auth); err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}

	return &deployer.PrepareResult{
		ImageTag: spec.Source.Image,
	}, nil
}

// Deploy creates and starts the container
func (d *Deployer) Deploy(ctx context.Context, spec *deployer.DeploymentSpec) (*deployer.DeploymentResult, error) {
	d.log.Info("deploying container",
		"image", spec.Source.Image,
		"slot", spec.TargetSlot,
	)

	// Find available port
	hostPort, err := findAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}

	// Prepare environment variables
	env := make(map[string]string)
	for k, v := range spec.Environment {
		env[k] = v
	}

	// Generate container name
	containerName := fmt.Sprintf("nebula-%s-%s-%s", spec.AppID[:8], spec.TargetSlot, uuid.New().String()[:8])

	// Prepare labels
	labels := map[string]string{
		"nebula.app_id": spec.AppID,
		"nebula.slot":   string(spec.TargetSlot),
		"nebula.managed": "true",
	}

	// Create container configuration
	config := container.ContainerConfig{
		Name:  containerName,
		Image: spec.Source.Image,
		Env:   env,
		Labels: labels,
		Ports: []container.PortBinding{
			{
				ContainerPort: spec.Source.Port,
				HostPort:      hostPort,
				Protocol:      "tcp",
			},
		},
		Networks: []string{d.network},
		RestartPolicy: "unless-stopped",
		HealthCheck: &container.HealthCheckConfig{
			Test:        []string{"CMD-SHELL", fmt.Sprintf("curl -f http://localhost:%d/ || exit 1", spec.Source.Port)},
			Interval:    10 * time.Second,
			Timeout:     5 * time.Second,
			Retries:     3,
			StartPeriod: 30 * time.Second,
		},
	}

	// Ensure network exists
	_, err = d.runtime.CreateNetwork(ctx, d.network, container.NetworkOptions{})
	if err != nil {
		d.log.Warn("failed to create network", "error", err)
	}

	// Create container
	containerID, err := d.runtime.CreateContainer(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := d.runtime.StartContainer(ctx, containerID); err != nil {
		// Cleanup on failure
		d.runtime.RemoveContainer(ctx, containerID, true)
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	d.log.Info("container started",
		"container_id", containerID[:12],
		"host_port", hostPort,
	)

	return &deployer.DeploymentResult{
		ContainerIDs: []string{containerID},
		Ports: map[string]int{
			"main": hostPort,
		},
	}, nil
}

// HealthCheck performs health checks on the deployment
func (d *Deployer) HealthCheck(ctx context.Context, result *deployer.DeploymentResult) (*deployer.HealthCheckResult, error) {
	if len(result.ContainerIDs) == 0 {
		return &deployer.HealthCheckResult{
			Healthy: false,
			Message: "no containers to check",
		}, nil
	}

	// Wait for container to be healthy
	maxAttempts := 30
	interval := 2 * time.Second

	for attempt := 0; attempt < maxAttempts; attempt++ {
		allHealthy := true

		for _, containerID := range result.ContainerIDs {
			info, err := d.runtime.InspectContainer(ctx, containerID)
			if err != nil {
				return nil, fmt.Errorf("failed to inspect container: %w", err)
			}

			// Check if container is running
			if info.State != "running" {
				allHealthy = false
				break
			}

			// If health check is configured, wait for healthy status
			if info.Health != "" && info.Health != "healthy" {
				allHealthy = false
				break
			}
		}

		if allHealthy {
			return &deployer.HealthCheckResult{
				Healthy: true,
				Message: "all containers healthy",
			}, nil
		}

		select {
		case <-ctx.Done():
			return &deployer.HealthCheckResult{
				Healthy: false,
				Message: "health check timeout",
			}, nil
		case <-time.After(interval):
			continue
		}
	}

	return &deployer.HealthCheckResult{
		Healthy: false,
		Message: "health check failed after max attempts",
	}, nil
}

// Stop stops the containers
func (d *Deployer) Stop(ctx context.Context, containerIDs []string) error {
	for _, id := range containerIDs {
		if err := d.runtime.StopContainer(ctx, id, 30*time.Second); err != nil {
			d.log.Warn("failed to stop container", "container_id", id, "error", err)
		}
	}
	return nil
}

// Destroy removes the containers
func (d *Deployer) Destroy(ctx context.Context, containerIDs []string) error {
	for _, id := range containerIDs {
		if err := d.runtime.StopContainer(ctx, id, 10*time.Second); err != nil {
			d.log.Warn("failed to stop container", "container_id", id, "error", err)
		}
		if err := d.runtime.RemoveContainer(ctx, id, true); err != nil {
			d.log.Warn("failed to remove container", "container_id", id, "error", err)
		}
	}
	return nil
}

// findAvailablePort finds an available port on the host
func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}
