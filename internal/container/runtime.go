package container

import (
	"context"
	"time"

	core "github.com/victalejo/nebula/internal/core/container"
)

// Runtime is a simplified interface for services that need container operations
type Runtime interface {
	PullImage(ctx context.Context, ref string, auth *core.RegistryAuth) error
	BuildImage(ctx context.Context, contextDir string, imageName string) (imageID string, buildLogs string, err error)
	CreateContainer(ctx context.Context, config *ContainerConfig) (string, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string, timeout time.Duration) error
	RestartContainer(ctx context.Context, id string, timeout time.Duration) error
	RemoveContainer(ctx context.Context, id string) error
	InspectContainer(ctx context.Context, id string) (*ContainerInfo, error)
	CreateNetwork(ctx context.Context, name string) error
}

// ContainerConfig simplified config for services
type ContainerConfig struct {
	Name          string
	Image         string
	Env           []string
	Cmd           []string
	Entrypoint    []string
	Labels        map[string]string
	Ports         []PortMapping
	Volumes       []VolumeMount
	Network       string
	HealthCheck   *HealthCheck
	RestartPolicy string
}

type PortMapping struct {
	HostPort      int
	ContainerPort int
}

type VolumeMount struct {
	Source string
	Target string
}

type HealthCheck struct {
	Test        []string
	Interval    time.Duration
	Timeout     time.Duration
	Retries     int
	StartPeriod time.Duration
}

type ContainerInfo struct {
	ID      string
	Name    string
	Image   string
	State   string
	Health  string
	Ports   []PortMapping
	Created time.Time
}

// RuntimeAdapter adapts core.ContainerRuntime to the simplified Runtime interface
type RuntimeAdapter struct {
	runtime core.ContainerRuntime
}

func NewRuntimeAdapter(runtime core.ContainerRuntime) *RuntimeAdapter {
	return &RuntimeAdapter{runtime: runtime}
}

func (a *RuntimeAdapter) PullImage(ctx context.Context, ref string, auth *core.RegistryAuth) error {
	return a.runtime.PullImage(ctx, ref, auth)
}

func (a *RuntimeAdapter) BuildImage(ctx context.Context, contextDir string, imageName string) (string, string, error) {
	imageID, err := a.runtime.BuildImage(ctx, core.BuildOptions{
		ContextPath: contextDir,
		Tags:        []string{imageName},
	})
	return imageID, "", err
}

func (a *RuntimeAdapter) CreateContainer(ctx context.Context, config *ContainerConfig) (string, error) {
	// Convert []string env to map[string]string
	envMap := make(map[string]string)
	for _, e := range config.Env {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				envMap[e[:i]] = e[i+1:]
				break
			}
		}
	}

	// Convert ports
	ports := make([]core.PortBinding, len(config.Ports))
	for i, p := range config.Ports {
		ports[i] = core.PortBinding{
			HostPort:      p.HostPort,
			ContainerPort: p.ContainerPort,
			Protocol:      "tcp",
		}
	}

	// Convert volumes
	volumes := make([]core.VolumeMount, len(config.Volumes))
	for i, v := range config.Volumes {
		volumes[i] = core.VolumeMount{
			Source: v.Source,
			Target: v.Target,
		}
	}

	// Convert health check
	var healthCheck *core.HealthCheckConfig
	if config.HealthCheck != nil {
		healthCheck = &core.HealthCheckConfig{
			Test:        config.HealthCheck.Test,
			Interval:    config.HealthCheck.Interval,
			Timeout:     config.HealthCheck.Timeout,
			Retries:     config.HealthCheck.Retries,
			StartPeriod: config.HealthCheck.StartPeriod,
		}
	}

	// Build networks slice
	var networks []string
	if config.Network != "" {
		networks = []string{config.Network}
	}

	// Build command
	cmd := config.Cmd
	if len(config.Entrypoint) > 0 {
		cmd = append(config.Entrypoint, config.Cmd...)
	}

	return a.runtime.CreateContainer(ctx, core.ContainerConfig{
		Name:          config.Name,
		Image:         config.Image,
		Env:           envMap,
		Labels:        config.Labels,
		Ports:         ports,
		Volumes:       volumes,
		Networks:      networks,
		Command:       cmd,
		HealthCheck:   healthCheck,
		RestartPolicy: config.RestartPolicy,
	})
}

func (a *RuntimeAdapter) StartContainer(ctx context.Context, id string) error {
	return a.runtime.StartContainer(ctx, id)
}

func (a *RuntimeAdapter) StopContainer(ctx context.Context, id string, timeout time.Duration) error {
	return a.runtime.StopContainer(ctx, id, timeout)
}

func (a *RuntimeAdapter) RestartContainer(ctx context.Context, id string, timeout time.Duration) error {
	return a.runtime.RestartContainer(ctx, id, timeout)
}

func (a *RuntimeAdapter) RemoveContainer(ctx context.Context, id string) error {
	return a.runtime.RemoveContainer(ctx, id, false)
}

func (a *RuntimeAdapter) InspectContainer(ctx context.Context, id string) (*ContainerInfo, error) {
	info, err := a.runtime.InspectContainer(ctx, id)
	if err != nil {
		return nil, err
	}

	ports := make([]PortMapping, len(info.Ports))
	for i, p := range info.Ports {
		ports[i] = PortMapping{
			HostPort:      p.HostPort,
			ContainerPort: p.ContainerPort,
		}
	}

	return &ContainerInfo{
		ID:      info.ID,
		Name:    info.Name,
		Image:   info.Image,
		State:   info.State,
		Health:  info.Health,
		Ports:   ports,
		Created: info.Created,
	}, nil
}

func (a *RuntimeAdapter) CreateNetwork(ctx context.Context, name string) error {
	_, err := a.runtime.CreateNetwork(ctx, name, core.NetworkOptions{})
	return err
}
