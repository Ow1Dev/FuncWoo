package funcinvoker

import (
	"context"
	"fmt"
	"time"

	pb "github.com/Ow1Dev/NoctiFunc/pkgs/api/server"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type StandardGRPCClient struct {
	timeout time.Duration
}

func NewStandardGRPCClient(timeout time.Duration) *StandardGRPCClient {
	return &StandardGRPCClient{
		timeout: timeout,
	}
}

func (c *StandardGRPCClient) Invoke(ctx context.Context, url string, payload string) (string, error) {
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	defer conn.Close()

	client := pb.NewFunctionRunnerServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	log.Debug().Msgf("Request body: %s", payload)
	r, err := client.Invoke(ctx, &pb.InvokeRequest{
		Payload: payload,
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute command in Docker container: %w", err)
	}

	// Implement the logic to execute the command in the Docker container
	return r.Output, nil
}
