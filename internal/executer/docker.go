package executer

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/Ow1Dev/FuncWoo/pkgs/api/server"
)

type DockerContainer struct {
	cli *client.Client
}

func (d *DockerContainer) execute(key string, body string, ctx context.Context) (string, error) {
	conn, err := grpc.NewClient("localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	defer conn.Close()

	client := pb.NewFunctionRunnerServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	fmt.Println("executing command in Docker container for key:", key)
	fmt.Println("Request body:", body)
	r, err := client.Invoke(ctx, &pb.InvokeRequest{
		Payload: body,
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute command in Docker container: %w", err)
	}

	// Implement the logic to execute the command in the Docker container
	return r.Output, nil
}

func (d *DockerContainer) exist(key string, ctx context.Context) bool {
	fmt.Println("checking if Docker container exists for key:", key)
		_, err := d.cli.ContainerInspect(ctx, key)
	return err == nil
}

func (d *DockerContainer) start(key string, ctx context.Context) error {
	// TODO: Use own docker client to create and start a container
	fmt.Println("starting Docker container for key:", key)
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
				Source: "/var/lib/funcwoo/funcs/" + key,
				Target: "/func/",
				ReadOnly: true,
			},
		},
	}, nil, nil, key)
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
