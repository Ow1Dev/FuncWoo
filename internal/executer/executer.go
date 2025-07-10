package executer

import (
	"bufio"
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fmt"
	"time"

	pb "github.com/Ow1Dev/FuncWoo/pkgs/api/server"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Container interface {
	getPort (key string, ctx context.Context) int
	waitForContainer (key string, ctx context.Context) error
	exist (key string, ctx context.Context) bool
	start (key string, port int, ctx context.Context) error
}

type Executer struct {
	container Container
}

func NewExecuter(container Container) *Executer {
	return &Executer{
		container: container,
	}
}

func (e *Executer) Execute(action string, body string, ctx context.Context) (string, error) {
	key, err := getKeyFromAction(action)
	if err != nil {
		return "", fmt.Errorf("failed to get key from action: %w", err)
	}
	
	if !e.container.exist(key, ctx) {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return "", fmt.Errorf("failed to create listener: %w", err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close() // Close the listener since we only needed the port
		
		log.Debug().Msgf("Starting container on port: %d", port)
		err = e.container.start(key, port, ctx)
		if err != nil {
			return "", fmt.Errorf("failed to start container: %w", err)
		}
		
		// Wait for container to be ready
		if err := e.container.waitForContainer(key, ctx); err != nil {
			return "", fmt.Errorf("container failed to start properly: %w", err)
		}
	}
	
	port := e.container.getPort(key, ctx)
	if port == 0 {
		return "", fmt.Errorf("failed to get port for container: %s", key)
	}
	
	log.Debug().Msgf("Making request to localhost:%d", port)
	rsp, err := handleRequest("localhost:"+strconv.Itoa(port), body, ctx)
	if err != nil {
		return "", fmt.Errorf("failed to handle request: %w", err)
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

func handleRequest(url string, body string, ctx context.Context) (string, error) {
	//TODO: get url from Container implementation
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	defer conn.Close()

	client := pb.NewFunctionRunnerServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	log.Debug().Msgf("Request URL: %s", url)
	log.Debug().Msgf("Request body: %s", body)
	r, err := client.Invoke(ctx, &pb.InvokeRequest{
		Payload: body,
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute command in Docker container: %w", err)
	}

	// Implement the logic to execute the command in the Docker container
	return r.Output, nil
}
