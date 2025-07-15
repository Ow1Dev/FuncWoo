package executer

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/rs/zerolog/log"

	cerrdefs "github.com/containerd/errdefs"
)

type DockerContainer struct {
	cli *client.Client
}

func (d *DockerContainer) waitForContainer(key string, ctx context.Context) error {
	log.Info().Msgf("Waiting for container to be ready: %s", key)
	
	// Wait for container to be in running state
	for range 30 {
		containerJSON, err := d.cli.ContainerInspect(ctx, key)
		if err != nil {
			return fmt.Errorf("failed to inspect container: %w", err)
		}
		
		if containerJSON.State.Running {
			log.Info().Msgf("Container %s is running", key)
			
			// Additional check: try to connect to the port
			port := d.getPort(key, ctx)
			if port > 0 {
				conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), time.Second)
				if err == nil {
					conn.Close()
					log.Info().Msgf("Container %s is ready and accepting connections", key)
					return nil
				}
			}
		}
		
		time.Sleep(time.Second)
	}
	
	return fmt.Errorf("container %s did not become ready within timeout", key)
}


func (d *DockerContainer) getPort(key string, ctx context.Context) int {
	log.Info().Msgf("Getting port for Docker container with key: %s", key)
	
	containerJSON, err := d.cli.ContainerInspect(ctx, key)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to inspect container: %s", key)
		return 0
	}

	// Look for the first TCP port mapping
	for containerPort, bindings := range containerJSON.NetworkSettings.Ports {
		if containerPort.Proto() == "tcp" && len(bindings) > 0 {
			// Return the first host port found
			hostPort := bindings[0].HostPort
			if port, err := strconv.Atoi(hostPort); err == nil {
				log.Info().Msgf("Found port %d for container %s", port, key)
				return port
			}
		}
	}
	
	log.Warn().Msgf("No port mapping found for container: %s", key)
	return 0
}

func (d *DockerContainer) isRunning(key string, ctx context.Context) bool {
	log.Info().Msgf("Checking if Docker container exists for key: %s", key)
	v, err := d.cli.ContainerInspect(ctx, key)
	if err != nil {
		return false
	}

	return v.State.Running
}

func (d *DockerContainer) start(key string, ctx context.Context) error {
	containerId, err := d.getOrCreateContainer(key, ctx)
	if err != nil {
		return fmt.Errorf("Failed to get or create container") 
	}

	log.Info().Msgf("Starting Docker container with ID: %s for key: %s", containerId, key)
	if err := d.cli.ContainerStart(ctx, containerId, container.StartOptions{}); err != nil {
		log.Error().Err(err).Msgf("Failed to start container %s", containerId)
		return fmt.Errorf("failed to start container: %w", err)
	}
	
	log.Info().Msgf("Docker container started successfully for key: %s", key)

	// Wait for the container to be ready
	if err := d.waitForContainer(key, ctx); err != nil {
		return fmt.Errorf("container failed to start properly: %w", err)
	}

	return nil
}

func getRandomPort() (*int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close() // Close the listener since we only needed the port

	return &port, nil
}

func (d *DockerContainer) getIdByKey(key string, ctx context.Context) (string, error) {
	log.Info().Msgf("Getting Docker container ID for key: %s", key)
	containerJSON, err := d.cli.ContainerInspect(ctx, key)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			log.Debug().Msgf("Container %s not found", key)
			return "", err 
		}
		log.Error().Err(err).Msgf("Failed to inspect container: %s", key)
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}
	log.Debug().Msgf("Container ID for key %s is %s", key, containerJSON.ID)
	return containerJSON.ID, nil
}

func (d *DockerContainer) getOrCreateContainer(key string, ctx context.Context) (string, error) {
	// First try to get existing container
	containerID, err := d.getIdByKey(key, ctx)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			log.Info().Msgf("Container %s not found, creating new one", key)
			return d.create(key, ctx)
		}
		return "", err
	}
	
	return containerID, nil
}

func (d *DockerContainer) create(key string, ctx context.Context) (string, error) {
	log.Info().Msgf("Creating Docker container for key: %s", key)

	port, err := getRandomPort()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get random port")
		return "", fmt.Errorf("failed to get random port: %w", err)
	}

	internalPort := nat.Port("8080/tcp")

	resp, err := d.cli.ContainerCreate(ctx, &container.Config{
		Image: "funcwoo/base",
		Cmd:   []string{"/func/main"},
		ExposedPorts: nat.PortSet{
			internalPort: struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			internalPort: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: strconv.Itoa(*port),
				},
			},
		},
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   "/var/lib/funcwoo/funcs/" + key,
				Target:   "/func/",
				ReadOnly: true,
			},
		},
	}, nil, nil, key)
	
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create container for key: %s", key)
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	
	log.Info().Msgf("Docker container created with ID: %s", resp.ID)

	return resp.ID, nil
}

func NewDockerContainer() (*DockerContainer, error) {
	// TODO: Use own docker client to create and start a container
	// TODO: Use environment variables or configuration files to set the Docker host
	cli, err := client.NewClientWithOpts(
		client.WithHost("unix:///var/run/docker.sock"),
		client.WithAPIVersionNegotiation(),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DockerContainer{
		cli: cli,
	}, nil
}
