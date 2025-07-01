package executer

import (
	"context"

	"time"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "github.com/Ow1Dev/FuncWoo/pkgs/api/server"
)

type Container interface {
	exist (key string, ctx context.Context) bool
	start (key string, ctx context.Context) error
}

type Executer struct {
	container Container
}

func NewExecuter(container Container) *Executer {
	return &Executer{
		container: container,
	}
}

func(e *Executer) Execeute(key string, body string, ctx context.Context) (string, error) {
	// override key with a fixed value for testing purposes
	key = "7c7677eec81f1b60dc19db9dbe06113c2af58b020cca5aca6106366f38fe11ae"
	if !e.container.exist(key, ctx) {
		err := e.container.start(key, ctx)
		if err != nil {
			return "", err
		}
	}

	rsp, err := handleRequest(key, body, ctx)
	if err != nil {
		return "", err
	}

	return rsp, nil
}


func handleRequest(key string, body string, ctx context.Context) (string, error) {
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
