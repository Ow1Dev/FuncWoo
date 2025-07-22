package sigil

import (
	"context"
	"fmt"
	"net"
	"os"

	pb "github.com/Ow1Dev/NoctiFunc/pkg/api/server"
	"google.golang.org/grpc"
)

const (
	defaultGRPCPort        = 8080
)

type serviceServer struct {
	pb.UnimplementedFunctionRunnerServiceServer
	handler Handler
}

func (s *serviceServer) Invoke(ctx context.Context, req *pb.InvokeRequest) (*pb.InvokeResult, error) {
	fmt.Printf("[Invoke] Received request: %s\n", req.GetPayload())

	resp, err := s.handler.Invoke(ctx, []byte(req.GetPayload()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "[Invoke] Error invoking handler: %v\n", err)
		return nil, fmt.Errorf("failed to invoke handler: %w", err)
	}

	fmt.Printf("[Invoke] Response: %s\n", resp)

	return &pb.InvokeResult{
		Output: string(resp),
	}, nil
}

// StartGRPCServer launches a gRPC server with the given handler on the specified port.
func StartGRPCServer(handler Handler, port int) error {
	addr := fmt.Sprintf(":%d", port)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	server := grpc.NewServer()
	pb.RegisterFunctionRunnerServiceServer(server, &serviceServer{handler: handler})

	fmt.Printf("[gRPC] Server listening on %s\n", addr)

	if err := server.Serve(lis); err != nil {
		return fmt.Errorf("gRPC server failed: %w", err)
	}

	return nil
}

// startRuntimeGRPCLoop is a backward-compatible entry point that uses the default port.
func startRuntimeGRPCLoop(handler Handler) error {
	return StartGRPCServer(handler, defaultGRPCPort)
}

