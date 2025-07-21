package sigil

import (
	"context"
	"fmt"
	"net"
	"os"

	pb "github.com/Ow1Dev/NoctiFunc/pkgs/api/server"
	"google.golang.org/grpc"
)

// Server wraps the gRPC server and provides testability
type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
	handler    *Handler
}

func NewServer(handlerFunc any) (*Server, error) {
	handler, err := NewHandler(handlerFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to create handler: %w", err)
	}

	return &Server{
		handler: handler,
	}, nil
}

type serviceServer struct {
	pb.UnimplementedFunctionRunnerServiceServer
	handler *Handler
}

// Invoke handles the gRPC invoke request - extracted for easier testing
func (s *serviceServer) Invoke(ctx context.Context, req *pb.InvokeRequest) (*pb.InvokeResult, error) {
	return s.handler.Invoke(ctx, []byte(req.GetPayload()))
}

func (s *Server) listen(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}
	s.listener = lis
	return nil
}

func (s *Server) serve() error {
	if s.listener == nil {
		return fmt.Errorf("server not listening, call Listen() first")
	}

	s.grpcServer = grpc.NewServer()
	pb.RegisterFunctionRunnerServiceServer(s.grpcServer, &serviceServer{
		handler: s.handler,
	})

	fmt.Printf("Gateway server listening on %s\n", s.listener.Addr().String())
	return s.grpcServer.Serve(s.listener)
}

func Start(handlerFunc any) {
	server, err := NewServer(handlerFunc)
		if err != nil {
		fmt.Fprintf(os.Stderr, "error creating server: %s\n", err)
		os.Exit(1)
	}

	if err := server.listen(8080); err != nil {
		fmt.Fprintf(os.Stderr, "error listening: %s\n", err)
		os.Exit(1)
	}

	if err := server.serve(); err != nil {
		fmt.Fprintf(os.Stderr, "error serving: %s\n", err)
		os.Exit(1)
	}
}
