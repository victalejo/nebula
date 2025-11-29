package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/victalejo/nebula/internal/container"
	"github.com/victalejo/nebula/internal/core/deployer"
	"github.com/victalejo/nebula/internal/core/logger"
)

// ComposeFile represents a docker-compose.yml structure
type ComposeFile struct {
	Version  string                    `yaml:"version,omitempty"`
	Services map[string]ComposeService `yaml:"services"`
	Networks map[string]ComposeNetwork `yaml:"networks,omitempty"`
	Volumes  map[string]ComposeVolume  `yaml:"volumes,omitempty"`
}

type ComposeService struct {
	Image       string            `yaml:"image,omitempty"`
	Build       *ComposeBuild     `yaml:"build,omitempty"`
	Command     interface{}       `yaml:"command,omitempty"`
	Entrypoint  interface{}       `yaml:"entrypoint,omitempty"`
	Environment interface{}       `yaml:"environment,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	DependsOn   interface{}       `yaml:"depends_on,omitempty"`
	Networks    []string          `yaml:"networks,omitempty"`
	Restart     string            `yaml:"restart,omitempty"`
	HealthCheck *ComposeHealth    `yaml:"healthcheck,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
}

type ComposeBuild struct {
	Context    string `yaml:"context,omitempty"`
	Dockerfile string `yaml:"dockerfile,omitempty"`
}

type ComposeNetwork struct {
	Driver   string `yaml:"driver,omitempty"`
	External bool   `yaml:"external,omitempty"`
}

type ComposeVolume struct {
	Driver   string `yaml:"driver,omitempty"`
	External bool   `yaml:"external,omitempty"`
}

type ComposeHealth struct {
	Test        interface{} `yaml:"test,omitempty"`
	Interval    string      `yaml:"interval,omitempty"`
	Timeout     string      `yaml:"timeout,omitempty"`
	Retries     int         `yaml:"retries,omitempty"`
	StartPeriod string      `yaml:"start_period,omitempty"`
}

type Deployer struct {
	runtime container.Runtime
	log     logger.Logger
	dataDir string
}

func New(runtime container.Runtime, log logger.Logger, dataDir string) *Deployer {
	return &Deployer{
		runtime: runtime,
		log:     log,
		dataDir: dataDir,
	}
}

func (d *Deployer) Mode() deployer.DeploymentMode {
	return deployer.ModeCompose
}

func (d *Deployer) Validate(ctx context.Context, spec *deployer.DeploymentSpec) error {
	if spec.ComposeFile == "" {
		return fmt.Errorf("compose file content is required")
	}

	// Parse and validate compose file
	var compose ComposeFile
	if err := yaml.Unmarshal([]byte(spec.ComposeFile), &compose); err != nil {
		return fmt.Errorf("invalid compose file: %w", err)
	}

	if len(compose.Services) == 0 {
		return fmt.Errorf("compose file must define at least one service")
	}

	// Validate each service has an image or build context
	for name, svc := range compose.Services {
		if svc.Image == "" && svc.Build == nil {
			return fmt.Errorf("service %s must have either 'image' or 'build' defined", name)
		}
	}

	return nil
}

func (d *Deployer) Prepare(ctx context.Context, spec *deployer.DeploymentSpec) (*deployer.PrepareResult, error) {
	var compose ComposeFile
	if err := yaml.Unmarshal([]byte(spec.ComposeFile), &compose); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Create project directory
	projectDir := filepath.Join(d.dataDir, "compose", spec.AppName)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project directory: %w", err)
	}

	// Write compose file
	composeFilePath := filepath.Join(projectDir, "docker-compose.yml")
	if err := os.WriteFile(composeFilePath, []byte(spec.ComposeFile), 0644); err != nil {
		return nil, fmt.Errorf("failed to write compose file: %w", err)
	}

	// Pull images for services that don't need building
	var imagesToPull []string
	for _, svc := range compose.Services {
		if svc.Image != "" && svc.Build == nil {
			imagesToPull = append(imagesToPull, svc.Image)
		}
	}

	for _, img := range imagesToPull {
		d.log.Info("pulling image", "image", img)
		if err := d.runtime.PullImage(ctx, img, nil); err != nil {
			d.log.Warn("failed to pull image, will try to use local", "image", img, "error", err)
		}
	}

	return &deployer.PrepareResult{
		ImageID:   "compose:" + spec.AppName,
		BuildLogs: fmt.Sprintf("Prepared compose project with %d services", len(compose.Services)),
	}, nil
}

func (d *Deployer) Deploy(ctx context.Context, spec *deployer.DeploymentSpec) (*deployer.DeploymentResult, error) {
	var compose ComposeFile
	if err := yaml.Unmarshal([]byte(spec.ComposeFile), &compose); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	projectName := fmt.Sprintf("%s-%s", spec.AppName, spec.Slot)
	networkName := fmt.Sprintf("%s_default", projectName)

	// Create network for the project
	if err := d.runtime.CreateNetwork(ctx, networkName); err != nil {
		d.log.Warn("network may already exist", "network", networkName, "error", err)
	}

	// Sort services by dependencies
	serviceOrder := d.sortServicesByDependency(compose.Services)

	var containerIDs []string
	var primaryPort int

	for _, serviceName := range serviceOrder {
		svc := compose.Services[serviceName]
		containerName := fmt.Sprintf("%s_%s", projectName, serviceName)

		// Build container config
		config := &container.ContainerConfig{
			Name:    containerName,
			Image:   svc.Image,
			Network: networkName,
			Labels: map[string]string{
				"nebula.app":     spec.AppName,
				"nebula.slot":    string(spec.Slot),
				"nebula.service": serviceName,
				"nebula.project": projectName,
			},
		}

		// Parse environment variables
		config.Env = d.parseEnvironment(svc.Environment, spec.EnvVars)

		// Parse command
		if svc.Command != nil {
			config.Cmd = d.parseStringOrSlice(svc.Command)
		}

		// Parse entrypoint
		if svc.Entrypoint != nil {
			config.Entrypoint = d.parseStringOrSlice(svc.Entrypoint)
		}

		// Parse volumes
		for _, vol := range svc.Volumes {
			parts := strings.Split(vol, ":")
			if len(parts) >= 2 {
				// Convert relative paths to absolute within project dir
				source := parts[0]
				if !filepath.IsAbs(source) && !strings.HasPrefix(source, ".") {
					// Named volume
					source = fmt.Sprintf("%s_%s", projectName, source)
				}
				config.Volumes = append(config.Volumes, container.VolumeMount{
					Source: source,
					Target: parts[1],
				})
			}
		}

		// Parse ports - only expose for the first service or service named "web" or "app"
		if serviceName == "web" || serviceName == "app" || serviceName == serviceOrder[0] {
			for _, port := range svc.Ports {
				parts := strings.Split(port, ":")
				if len(parts) == 2 {
					var hostPort, containerPort int
					fmt.Sscanf(parts[0], "%d", &hostPort)
					fmt.Sscanf(parts[1], "%d", &containerPort)
					if containerPort > 0 {
						config.Ports = append(config.Ports, container.PortMapping{
							HostPort:      0, // Let Docker assign
							ContainerPort: containerPort,
						})
						if primaryPort == 0 {
							primaryPort = containerPort
						}
					}
				}
			}
		}

		// Parse health check
		if svc.HealthCheck != nil {
			config.HealthCheck = d.parseHealthCheck(svc.HealthCheck)
		}

		// Parse restart policy
		switch svc.Restart {
		case "always":
			config.RestartPolicy = "always"
		case "unless-stopped":
			config.RestartPolicy = "unless-stopped"
		case "on-failure":
			config.RestartPolicy = "on-failure"
		default:
			config.RestartPolicy = "unless-stopped"
		}

		// Create and start container
		d.log.Info("creating container", "service", serviceName, "container", containerName)
		containerID, err := d.runtime.CreateContainer(ctx, config)
		if err != nil {
			// Cleanup already created containers
			for _, id := range containerIDs {
				_ = d.runtime.StopContainer(ctx, id, 10*time.Second)
				_ = d.runtime.RemoveContainer(ctx, id)
			}
			return nil, fmt.Errorf("failed to create container %s: %w", serviceName, err)
		}

		if err := d.runtime.StartContainer(ctx, containerID); err != nil {
			_ = d.runtime.RemoveContainer(ctx, containerID)
			for _, id := range containerIDs {
				_ = d.runtime.StopContainer(ctx, id, 10*time.Second)
				_ = d.runtime.RemoveContainer(ctx, id)
			}
			return nil, fmt.Errorf("failed to start container %s: %w", serviceName, err)
		}

		containerIDs = append(containerIDs, containerID)
	}

	// Get the actual assigned port from the first container
	if len(containerIDs) > 0 && primaryPort > 0 {
		info, err := d.runtime.InspectContainer(ctx, containerIDs[0])
		if err == nil {
			for _, pm := range info.Ports {
				if pm.ContainerPort == primaryPort && pm.HostPort > 0 {
					primaryPort = pm.HostPort
					break
				}
			}
		}
	}

	return &deployer.DeploymentResult{
		ContainerIDs: containerIDs,
		Port:         primaryPort,
		Version:      uuid.New().String()[:8],
	}, nil
}

func (d *Deployer) HealthCheck(ctx context.Context, result *deployer.DeploymentResult) (*deployer.HealthCheckResult, error) {
	healthy := true
	var details []string

	for _, containerID := range result.ContainerIDs {
		info, err := d.runtime.InspectContainer(ctx, containerID)
		if err != nil {
			healthy = false
			details = append(details, fmt.Sprintf("container %s: failed to inspect", containerID[:12]))
			continue
		}

		if info.State != "running" {
			healthy = false
			details = append(details, fmt.Sprintf("container %s: not running (state: %s)", containerID[:12], info.State))
			continue
		}

		// Check health status if available
		if info.Health != "" && info.Health != "healthy" && info.Health != "none" {
			if info.Health == "starting" {
				details = append(details, fmt.Sprintf("container %s: health check starting", containerID[:12]))
			} else {
				healthy = false
				details = append(details, fmt.Sprintf("container %s: unhealthy", containerID[:12]))
			}
		} else {
			details = append(details, fmt.Sprintf("container %s: healthy", containerID[:12]))
		}
	}

	return &deployer.HealthCheckResult{
		Healthy: healthy,
		Message: strings.Join(details, "; "),
	}, nil
}

func (d *Deployer) Stop(ctx context.Context, containerIDs []string) error {
	var errs []error
	for _, id := range containerIDs {
		if err := d.runtime.StopContainer(ctx, id, 30*time.Second); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop %s: %w", id[:12], err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors stopping containers: %v", errs)
	}
	return nil
}

func (d *Deployer) Destroy(ctx context.Context, containerIDs []string) error {
	var errs []error
	for _, id := range containerIDs {
		if err := d.runtime.StopContainer(ctx, id, 10*time.Second); err != nil {
			d.log.Warn("failed to stop container", "id", id[:12], "error", err)
		}
		if err := d.runtime.RemoveContainer(ctx, id); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove %s: %w", id[:12], err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors removing containers: %v", errs)
	}
	return nil
}

// sortServicesByDependency returns services in order of their dependencies
func (d *Deployer) sortServicesByDependency(services map[string]ComposeService) []string {
	visited := make(map[string]bool)
	var result []string

	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true

		svc, ok := services[name]
		if !ok {
			return
		}

		// Visit dependencies first
		deps := d.parseDependsOn(svc.DependsOn)
		for _, dep := range deps {
			visit(dep)
		}

		result = append(result, name)
	}

	for name := range services {
		visit(name)
	}

	return result
}

func (d *Deployer) parseDependsOn(dependsOn interface{}) []string {
	if dependsOn == nil {
		return nil
	}

	switch v := dependsOn.(type) {
	case []interface{}:
		var deps []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				deps = append(deps, s)
			}
		}
		return deps
	case map[string]interface{}:
		var deps []string
		for name := range v {
			deps = append(deps, name)
		}
		return deps
	}
	return nil
}

func (d *Deployer) parseEnvironment(env interface{}, extra map[string]string) []string {
	var result []string

	switch v := env.(type) {
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
	case map[string]interface{}:
		for key, val := range v {
			if s, ok := val.(string); ok {
				result = append(result, fmt.Sprintf("%s=%s", key, s))
			} else {
				result = append(result, fmt.Sprintf("%s=%v", key, val))
			}
		}
	}

	// Add extra env vars from deployment spec
	for key, val := range extra {
		result = append(result, fmt.Sprintf("%s=%s", key, val))
	}

	return result
}

func (d *Deployer) parseStringOrSlice(val interface{}) []string {
	switch v := val.(type) {
	case string:
		return strings.Fields(v)
	case []interface{}:
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

func (d *Deployer) parseHealthCheck(hc *ComposeHealth) *container.HealthCheck {
	if hc == nil {
		return nil
	}

	result := &container.HealthCheck{
		Retries: hc.Retries,
	}

	// Parse test
	switch v := hc.Test.(type) {
	case string:
		result.Test = []string{"CMD-SHELL", v}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				result.Test = append(result.Test, s)
			}
		}
	}

	// Parse durations
	if hc.Interval != "" {
		if d, err := time.ParseDuration(hc.Interval); err == nil {
			result.Interval = d
		}
	}
	if hc.Timeout != "" {
		if d, err := time.ParseDuration(hc.Timeout); err == nil {
			result.Timeout = d
		}
	}
	if hc.StartPeriod != "" {
		if d, err := time.ParseDuration(hc.StartPeriod); err == nil {
			result.StartPeriod = d
		}
	}

	return result
}
