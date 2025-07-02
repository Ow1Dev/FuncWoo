package executer

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"

	"fmt"
	"time"

	pb "github.com/Ow1Dev/FuncWoo/pkgs/api/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

func(e *Executer) Execeute(action string, body string, ctx context.Context) (string, error) {
	key, err := getKeyFromAction(action)
	if err != nil {
		return "", fmt.Errorf("failed to get key from action: %w", err)
	}

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

func getKeyFromAction(action string) (string, error) {
	path := filepath.Join("/var/lib/funcwoo/action", action)
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open action file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading action file: %w", err)
	}
	return "", fmt.Errorf("action file %s is empty", path)
}

func handleRequest(key string, body string, ctx context.Context) (string, error) {
	//TODO: get url from Container implementation
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
