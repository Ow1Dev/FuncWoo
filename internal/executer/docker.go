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

func (d *DockerContainer) exist(key string, ctx context.Context) bool {
	log.Info().Msgf("Checking if Docker container exists for key: %s", key)
	_, err := d.cli.ContainerInspect(ctx, key)
	return err == nil
}

func (d *DockerContainer) start(key string, port int, ctx context.Context) error {
	internalPort := nat.Port("8080/tcp")
	log.Info().Msgf("Starting Docker container for key: %s", key)
	
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
					HostPort: strconv.Itoa(port),
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
		return fmt.Errorf("failed to create container: %w", err)
	}
	
	log.Info().Msgf("Docker container created with ID: %s", resp.ID)
	
	if err := d.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		log.Error().Err(err).Msgf("Failed to start container %s", resp.ID)
		return fmt.Errorf("failed to start container: %w", err)
	}
	
	log.Info().Msgf("Docker container started successfully for key: %s", key)
	return nil
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
