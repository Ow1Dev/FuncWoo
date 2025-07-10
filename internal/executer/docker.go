package executer

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/rs/zerolog/log"
)

type DockerContainer struct {
	cli *client.Client
}

func (d *DockerContainer) exist(key string, ctx context.Context) bool {
	log.Info().Msgf("Checking if Docker container exists for key: %s", key)
	_, err := d.cli.ContainerInspect(ctx, key)
	return err == nil
}

func (d *DockerContainer) start(key string, ctx context.Context) error {
	// TODO: Use own docker client to create and start a container
	log.Info().Msgf("Starting Docker container for key: %s", key)
	resp, err := d.cli.ContainerCreate(ctx, &container.Config{
		Image: "funcwoo/base",
		Cmd:   []string{"/func/main"},
		ExposedPorts: nat.PortSet{
			// TODO: use a dynamic port
			"8080/tcp": struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			"8080/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",     // or "127.0.0.1"
				  // TODO: use a dynamic port
					HostPort: "8080",
				},
			},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/var/lib/funcwoo/funcs/" + key,
				Target: "/func/",
				ReadOnly: true,
			},
		},
	}, nil, nil, key)
	if err != nil {
		return err
	}

	log.Info().Msgf("Docker container created with ID: %s", resp.ID)
	if err := d.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}

	return nil
}

func NewDockerContainer() (*DockerContainer, error) {
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
