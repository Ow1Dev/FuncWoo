package communication

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/Ow1Dev/NoctiFunc/pkgs/api/communication"
)

type GRPCClient struct {
	address string
	timeout time.Duration
}

func NewGRPCClient(address string, timeout time.Duration) *GRPCClient {
	return &GRPCClient{
		address: address,
		timeout: timeout,
	}
}

func (c *GRPCClient) SendAction(ctx context.Context, action string, body string) (string, error) {
	conn, err := grpc.NewClient(c.address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("failed to connect to gRPC server: %w", err)
	}
	defer conn.Close()

	client := pb.NewCommunicationServiceClient(conn)

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.Execute(ctx, &pb.ExecuteRequest{
		Action: action,
		Body:   body,
	})

	if err != nil {
		return "", fmt.Errorf("failed to send action to remote service: %w", err)
	}

	if resp.Status != "success" {
		return "", fmt.Errorf("remote service returned error status: %s", resp.Status)
	}

	return resp.Resp, nil
}
