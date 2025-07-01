package executer

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
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
		Cmd:   []string{"/func/echo"},
		ExposedPorts: nat.PortSet{
			"8080/tcp": struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			"8080/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",     // or "127.0.0.1"
					HostPort: "8080",
				},
			},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/var/lib/funcwoo/funcs/20e763a8e50b6b799c3fd419ad270403f1b184563c5533320b69da29972c6ca8",
				Target: "/func/",
				ReadOnly: true,
			},
		},
	}, nil, nil, "")
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
