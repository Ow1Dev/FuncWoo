package container

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/rs/zerolog"

	cerrdefs "github.com/containerd/errdefs"
	dockernet "github.com/docker/docker/api/types/network"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/Ow1Dev/NoctiFunc/pkg/network"
)

type DockerClientInterface interface {
	ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error)
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig,
		networkingConfig *dockernet.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error)
}

type DockerContainer struct {
	cli           DockerClientInterface
	portAllocator network.PortAllocator
	network       network.NetTransport
	timeProvider  TimeProvider
	config        DockerConfig
	logger        zerolog.Logger
}

type DockerConfig struct {
	Image                 string
	InternalPort          string
	MountSourcePrefix     string
	MountTarget           string
	ContainerReadyTimeout time.Duration
	ConnectionTimeout     time.Duration
	RetryInterval         time.Duration
}

func DefaultDockerConfig() DockerConfig {
	return DockerConfig{
		Image:                 "noctifunc/base",
		InternalPort:          "8080/tcp",
		MountSourcePrefix:     "/var/lib/noctifunc/funcs/",
		MountTarget:           "/func/",
		ContainerReadyTimeout: 30 * time.Second,
		ConnectionTimeout:     time.Second,
		RetryInterval:         time.Second,
	}
}

func NewDockerContainer(
	cli DockerClientInterface,
	portAllocator network.PortAllocator,
	network network.NetTransport,
	timeProvider TimeProvider,
	config DockerConfig,
	logger zerolog.Logger) *DockerContainer {
	return &DockerContainer{
		cli:           cli,
		config:        config,
		logger:        logger.With().Str("component", "docker_container").Logger(),
		network:       network,
		portAllocator: portAllocator,
		timeProvider:  timeProvider,
	}
}

func NewDockerContainerWithDefaults(logger zerolog.Logger) (*DockerContainer, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost("unix:///var/run/docker.sock"),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	nettransport := &network.RealNetwork{}
	return NewDockerContainer(
		&DockerClientAdapter{cli: cli},
		network.NewNetworkPortAllocator(nettransport),
		nettransport,
		&RealTimeProvider{},
		DefaultDockerConfig(),
		logger,
	), nil
}

func (d *DockerClientAdapter) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	return d.cli.ContainerInspect(ctx, containerID)
}

func (d *DockerClientAdapter) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	return d.cli.ContainerStart(ctx, containerID, options)
}

func (d *DockerClientAdapter) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig,
	networkingConfig *dockernet.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error) {
	return d.cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, containerName)
}

type DockerClientAdapter struct {
	cli *client.Client
}

func (d *DockerContainer) WaitForContainer(key string, ctx context.Context) error {
	d.logger.Info().Msgf("Waiting for container to be ready: %s", key)

	timeout := d.timeProvider.Now().Add(d.config.ContainerReadyTimeout)

	// Wait for container to be in running state
	for d.timeProvider.Now().Before(timeout) {
		containerJSON, err := d.cli.ContainerInspect(ctx, key)
		if err != nil {
			return fmt.Errorf("failed to inspect container: %w", err)
		}

		if containerJSON.State.Running {
			d.logger.Info().Msgf("Container %s is running", key)

			// Additional check: try to connect to the port
			port := d.GetPort(key, ctx)
			if port > 0 {
				conn, err := d.network.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), d.config.ConnectionTimeout)
				if err == nil {
					conn.Close()
					d.logger.Info().Msgf("Container %s is ready and accepting connections", key)
					return nil
				}
			}
		}

		d.timeProvider.Sleep(d.config.RetryInterval)
	}

	return fmt.Errorf("container %s did not become ready within timeout", key)
}

func (d *DockerContainer) GetPort(key string, ctx context.Context) int {
	d.logger.Info().Msgf("Getting port for Docker container with key: %s", key)

	containerJSON, err := d.cli.ContainerInspect(ctx, key)
	if err != nil {
		d.logger.Error().Err(err).Msgf("Failed to inspect container: %s", key)
		return 0
	}

	// Look for the first TCP port mapping
	for containerPort, bindings := range containerJSON.NetworkSettings.Ports {
		if containerPort.Proto() == "tcp" && len(bindings) > 0 {
			// Return the first host port found
			hostPort := bindings[0].HostPort
			if port, err := strconv.Atoi(hostPort); err == nil {
				d.logger.Info().Msgf("Found port %d for container %s", port, key)
				return port
			}
		}
	}

	d.logger.Warn().Msgf("No port mapping found for container: %s", key)
	return 0
}

func (d *DockerContainer) IsRunning(key string, ctx context.Context) bool {
	d.logger.Info().Msgf("Checking if Docker container exists for key: %s", key)
	v, err := d.cli.ContainerInspect(ctx, key)
	if err != nil {
		return false
	}

	return v.State.Running
}

func (d *DockerContainer) Start(key string, ctx context.Context) error {
	containerId, err := d.getOrCreateContainer(key, ctx)
	if err != nil {
		return fmt.Errorf("failed to get or create container")
	}

	d.logger.Info().Msgf("Starting Docker container with ID: %s for key: %s", containerId, key)
	if err := d.cli.ContainerStart(ctx, containerId, container.StartOptions{}); err != nil {
		d.logger.Error().Err(err).Msgf("Failed to start container %s", containerId)
		return fmt.Errorf("failed to start container: %w", err)
	}

	d.logger.Info().Msgf("Docker container started successfully for key: %s", key)

	// Wait for the container to be ready
	if err := d.WaitForContainer(key, ctx); err != nil {
		return fmt.Errorf("container failed to start properly: %w", err)
	}

	return nil
}

func (d *DockerContainer) getIdByKey(key string, ctx context.Context) (string, error) {
	d.logger.Info().Msgf("Getting Docker container ID for key: %s", key)
	containerJSON, err := d.cli.ContainerInspect(ctx, key)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			d.logger.Debug().Msgf("Container %s not found", key)
			return "", err
		}
		d.logger.Error().Err(err).Msgf("Failed to inspect container: %s", key)
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}
	d.logger.Debug().Msgf("Container ID for key %s is %s", key, containerJSON.ID)
	return containerJSON.ID, nil
}

func (d *DockerContainer) getOrCreateContainer(key string, ctx context.Context) (string, error) {
	// First try to get existing container
	containerID, err := d.getIdByKey(key, ctx)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			d.logger.Info().Msgf("Container %s not found, creating new one", key)
			return d.create(key, ctx)
		}
		return "", err
	}

	return containerID, nil
}

func (d *DockerContainer) create(key string, ctx context.Context) (string, error) {
	d.logger.Info().Msgf("Creating Docker container for key: %s", key)

	port, err := d.portAllocator.GetRandomPort()
	if err != nil {
		d.logger.Error().Err(err).Msg("Failed to get random port")
		return "", fmt.Errorf("failed to get random port: %w", err)
	}

	internalPort := nat.Port(d.config.InternalPort)

	resp, err := d.cli.ContainerCreate(ctx, &container.Config{
		Image: d.config.Image,
		Cmd:   []string{"/func/main"},
		ExposedPorts: nat.PortSet{
			internalPort: struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			internalPort: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: strconv.Itoa(port),
				},
			},
		},
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   d.config.MountSourcePrefix + key,
				Target:   d.config.MountTarget,
				ReadOnly: true,
			},
		},
	}, nil, nil, key)

	if err != nil {
		d.logger.Error().Err(err).Msgf("Failed to create container for key: %s", key)
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	d.logger.Info().Msgf("Docker container created with ID: %s", resp.ID)

	return resp.ID, nil
}
