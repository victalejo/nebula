package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	nebulacontainer "github.com/victalejo/nebula/internal/core/container"
)

// Client wraps the Docker SDK client
type Client struct {
	cli *client.Client
}

// NewClient creates a new Docker client
func NewClient(host string) (*Client, error) {
	opts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}

	if host != "" {
		opts = append(opts, client.WithHost(host))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &Client{cli: cli}, nil
}

// Ping checks Docker daemon connectivity
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.cli.Ping(ctx)
	return err
}

// PullImage pulls an image from a registry
func (c *Client) PullImage(ctx context.Context, ref string, auth *nebulacontainer.RegistryAuth) error {
	opts := image.PullOptions{}

	if auth != nil {
		authConfig := registry.AuthConfig{
			Username: auth.Username,
			Password: auth.Password,
		}
		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			return err
		}
		opts.RegistryAuth = base64.URLEncoding.EncodeToString(encodedJSON)
	}

	reader, err := c.cli.ImagePull(ctx, ref, opts)
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", ref, err)
	}
	defer reader.Close()

	// Consume the output to complete the pull
	_, err = io.Copy(io.Discard, reader)
	return err
}

// BuildImage builds an image from a Dockerfile
func (c *Client) BuildImage(ctx context.Context, opts nebulacontainer.BuildOptions) (string, error) {
	// TODO: Implement build functionality
	return "", fmt.Errorf("build not implemented yet")
}

// ListImages lists all images
func (c *Client) ListImages(ctx context.Context) ([]nebulacontainer.Image, error) {
	images, err := c.cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]nebulacontainer.Image, len(images))
	for i, img := range images {
		result[i] = nebulacontainer.Image{
			ID:      img.ID,
			Tags:    img.RepoTags,
			Size:    img.Size,
			Created: time.Unix(img.Created, 0),
		}
	}
	return result, nil
}

// RemoveImage removes an image
func (c *Client) RemoveImage(ctx context.Context, id string) error {
	_, err := c.cli.ImageRemove(ctx, id, image.RemoveOptions{})
	return err
}

// CreateContainer creates a new container
func (c *Client) CreateContainer(ctx context.Context, config nebulacontainer.ContainerConfig) (string, error) {
	// Prepare environment variables
	env := make([]string, 0, len(config.Env))
	for k, v := range config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Prepare port bindings
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	for _, p := range config.Ports {
		port := nat.Port(fmt.Sprintf("%d/%s", p.ContainerPort, p.Protocol))
		exposedPorts[port] = struct{}{}
		portBindings[port] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: fmt.Sprintf("%d", p.HostPort),
			},
		}
	}

	// Prepare mounts
	mounts := make([]mount.Mount, len(config.Volumes))
	for i, v := range config.Volumes {
		mounts[i] = mount.Mount{
			Type:     mount.TypeVolume,
			Source:   v.Source,
			Target:   v.Target,
			ReadOnly: v.ReadOnly,
		}
	}

	// Container config
	containerConfig := &container.Config{
		Image:        config.Image,
		Env:          env,
		Labels:       config.Labels,
		ExposedPorts: exposedPorts,
		Cmd:          config.Command,
	}

	// Host config
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Mounts:       mounts,
	}

	// Set restart policy
	if config.RestartPolicy != "" {
		hostConfig.RestartPolicy = container.RestartPolicy{
			Name: container.RestartPolicyMode(config.RestartPolicy),
		}
	}

	// Set resource limits
	if config.Resources != nil {
		hostConfig.Resources = container.Resources{
			NanoCPUs: config.Resources.CPULimit,
			Memory:   config.Resources.MemoryLimit,
		}
	}

	// Health check
	if config.HealthCheck != nil {
		containerConfig.Healthcheck = &container.HealthConfig{
			Test:        config.HealthCheck.Test,
			Interval:    config.HealthCheck.Interval,
			Timeout:     config.HealthCheck.Timeout,
			Retries:     config.HealthCheck.Retries,
			StartPeriod: config.HealthCheck.StartPeriod,
		}
	}

	// Network config
	networkConfig := &network.NetworkingConfig{}
	if len(config.Networks) > 0 {
		networkConfig.EndpointsConfig = map[string]*network.EndpointSettings{
			config.Networks[0]: {},
		}
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, config.Name)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Connect to additional networks
	for i := 1; i < len(config.Networks); i++ {
		if err := c.ConnectToNetwork(ctx, resp.ID, config.Networks[i]); err != nil {
			// Cleanup on error
			c.RemoveContainer(ctx, resp.ID, true)
			return "", err
		}
	}

	return resp.ID, nil
}

// StartContainer starts a container
func (c *Client) StartContainer(ctx context.Context, id string) error {
	return c.cli.ContainerStart(ctx, id, container.StartOptions{})
}

// StopContainer stops a container
func (c *Client) StopContainer(ctx context.Context, id string, timeout time.Duration) error {
	timeoutSec := int(timeout.Seconds())
	return c.cli.ContainerStop(ctx, id, container.StopOptions{
		Timeout: &timeoutSec,
	})
}

// RestartContainer restarts a container
func (c *Client) RestartContainer(ctx context.Context, id string, timeout time.Duration) error {
	timeoutSec := int(timeout.Seconds())
	return c.cli.ContainerRestart(ctx, id, container.StopOptions{
		Timeout: &timeoutSec,
	})
}

// RemoveContainer removes a container
func (c *Client) RemoveContainer(ctx context.Context, id string, force bool) error {
	return c.cli.ContainerRemove(ctx, id, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: false,
	})
}

// InspectContainer returns container information
func (c *Client) InspectContainer(ctx context.Context, id string) (*nebulacontainer.ContainerInfo, error) {
	info, err := c.cli.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}

	ports := []nebulacontainer.PortBinding{}
	for port, bindings := range info.NetworkSettings.Ports {
		for _, b := range bindings {
			ports = append(ports, nebulacontainer.PortBinding{
				ContainerPort: port.Int(),
				HostPort:      parsePort(b.HostPort),
				Protocol:      port.Proto(),
			})
		}
	}

	health := ""
	if info.State.Health != nil {
		health = info.State.Health.Status
	}

	createdTime, _ := time.Parse(time.RFC3339Nano, info.Created)

	return &nebulacontainer.ContainerInfo{
		ID:      info.ID,
		Name:    info.Name,
		Image:   info.Config.Image,
		Status:  info.State.Status,
		State:   info.State.Status,
		Created: createdTime,
		Ports:   ports,
		Labels:  info.Config.Labels,
		Health:  health,
	}, nil
}

// ListContainers lists containers
func (c *Client) ListContainers(ctx context.Context, filter nebulacontainer.ContainerFilter) ([]nebulacontainer.ContainerInfo, error) {
	opts := container.ListOptions{
		All: filter.All,
	}

	if len(filter.Labels) > 0 || len(filter.Names) > 0 {
		f := filters.NewArgs()
		for k, v := range filter.Labels {
			f.Add("label", fmt.Sprintf("%s=%s", k, v))
		}
		for _, name := range filter.Names {
			f.Add("name", name)
		}
		opts.Filters = f
	}

	containers, err := c.cli.ContainerList(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make([]nebulacontainer.ContainerInfo, len(containers))
	for i, cont := range containers {
		ports := make([]nebulacontainer.PortBinding, len(cont.Ports))
		for j, p := range cont.Ports {
			ports[j] = nebulacontainer.PortBinding{
				ContainerPort: int(p.PrivatePort),
				HostPort:      int(p.PublicPort),
				Protocol:      p.Type,
			}
		}

		result[i] = nebulacontainer.ContainerInfo{
			ID:      cont.ID,
			Name:    cont.Names[0],
			Image:   cont.Image,
			Status:  cont.Status,
			State:   cont.State,
			Created: time.Unix(cont.Created, 0),
			Ports:   ports,
			Labels:  cont.Labels,
		}
	}
	return result, nil
}

// ContainerLogs returns container logs
func (c *Client) ContainerLogs(ctx context.Context, id string, opts nebulacontainer.LogOptions) (io.ReadCloser, error) {
	options := container.LogsOptions{
		ShowStdout: opts.Stdout,
		ShowStderr: opts.Stderr,
		Follow:     opts.Follow,
		Timestamps: opts.Timestamps,
		Tail:       opts.Tail,
	}

	if !opts.Since.IsZero() {
		options.Since = opts.Since.Format(time.RFC3339)
	}
	if !opts.Until.IsZero() {
		options.Until = opts.Until.Format(time.RFC3339)
	}

	return c.cli.ContainerLogs(ctx, id, options)
}

// WaitContainer waits for a container to stop
func (c *Client) WaitContainer(ctx context.Context, id string) (<-chan nebulacontainer.WaitResult, <-chan error) {
	resultCh := make(chan nebulacontainer.WaitResult, 1)
	errCh := make(chan error, 1)

	waitCh, waitErrCh := c.cli.ContainerWait(ctx, id, container.WaitConditionNotRunning)

	go func() {
		select {
		case result := <-waitCh:
			errMsg := ""
			if result.Error != nil {
				errMsg = result.Error.Message
			}
			resultCh <- nebulacontainer.WaitResult{
				StatusCode: result.StatusCode,
				Error:      errMsg,
			}
		case err := <-waitErrCh:
			errCh <- err
		}
	}()

	return resultCh, errCh
}

// CreateNetwork creates a Docker network
func (c *Client) CreateNetwork(ctx context.Context, name string, opts nebulacontainer.NetworkOptions) (string, error) {
	// Check if network already exists
	networks, err := c.cli.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.Arg("name", name)),
	})
	if err != nil {
		return "", err
	}
	if len(networks) > 0 {
		return networks[0].ID, nil
	}

	driver := opts.Driver
	if driver == "" {
		driver = "bridge"
	}

	resp, err := c.cli.NetworkCreate(ctx, name, network.CreateOptions{
		Driver:   driver,
		Internal: opts.Internal,
		Labels:   opts.Labels,
	})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// RemoveNetwork removes a network
func (c *Client) RemoveNetwork(ctx context.Context, id string) error {
	return c.cli.NetworkRemove(ctx, id)
}

// ConnectToNetwork connects a container to a network
func (c *Client) ConnectToNetwork(ctx context.Context, containerID, networkID string) error {
	return c.cli.NetworkConnect(ctx, networkID, containerID, nil)
}

// DisconnectFromNetwork disconnects a container from a network
func (c *Client) DisconnectFromNetwork(ctx context.Context, containerID, networkID string) error {
	return c.cli.NetworkDisconnect(ctx, networkID, containerID, false)
}

// CreateVolume creates a Docker volume
func (c *Client) CreateVolume(ctx context.Context, name string, opts nebulacontainer.VolumeOptions) error {
	driver := opts.Driver
	if driver == "" {
		driver = "local"
	}

	_, err := c.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   name,
		Driver: driver,
		Labels: opts.Labels,
	})
	return err
}

// RemoveVolume removes a volume
func (c *Client) RemoveVolume(ctx context.Context, name string) error {
	return c.cli.VolumeRemove(ctx, name, false)
}

// ListVolumes lists all volumes
func (c *Client) ListVolumes(ctx context.Context) ([]nebulacontainer.Volume, error) {
	resp, err := c.cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]nebulacontainer.Volume, len(resp.Volumes))
	for i, v := range resp.Volumes {
		result[i] = nebulacontainer.Volume{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
			Labels:     v.Labels,
		}
	}
	return result, nil
}

// Close closes the Docker client
func (c *Client) Close() error {
	return c.cli.Close()
}

func parsePort(s string) int {
	var port int
	fmt.Sscanf(s, "%d", &port)
	return port
}
