package executer

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type DockerContainer struct {
	cli *client.Client
}

func (d *DockerContainer) execute(key string, body string, ctx context.Context) (string, error) {
	// Implement the logic to execute the command in the Docker container
	return "sailor execute", nil
}

func (d *DockerContainer) exist(key string, ctx context.Context) bool {
	// Implement the logic to check if the Docker container exists
	return false
}

func (d *DockerContainer) start(key string, ctx context.Context) error {
	// TODO: Use own docker client to create and start a container
	fmt.Println("creating Docker container...")
	resp, err := d.cli.ContainerCreate(ctx, &container.Config{
		Image: "alpine",
		Cmd:   []string{"echo", "Hello from Docker SDK"},
	}, nil, nil, nil, "")

	if err != nil {
		return err
	}

	fmt.Println("Docker container created with ID:", resp.ID)
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
