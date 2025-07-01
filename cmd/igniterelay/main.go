package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/Ow1Dev/FuncWoo/internal/executer"
	pb "github.com/Ow1Dev/FuncWoo/pkgs/api/communication"
	"google.golang.org/grpc"
)

type serviceServer struct {
	pb.UnimplementedCommunicationServiceServer
	Executer executer.Executer
}

// Execute implements gateway.ServerServiceServer.
func (s *serviceServer) Execute(ctx context.Context, r *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
	rsp, err := s.Executer.Execeute(r.GetAction(), r.GetBody(), ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error executing command: %s\n", err)
		return &pb.ExecuteResponse{
			Status: "error",
		}, nil
	}

	return &pb.ExecuteResponse{
		Status: "success",
		Resp: rsp,
	}, nil
}

func main() {
	dockerRunner, err := executer.NewDockerContainer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating docker runner: %s\n", err)
		os.Exit(1)
	}

	executer := executer.NewExecuter(dockerRunner)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 5001))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
	}

	s := grpc.NewServer()
	pb.RegisterCommunicationServiceServer(s, &serviceServer{
		Executer: *executer,
	})

	fmt.Printf("Gateway server listening on %s\n", lis.Addr().String())
	if err := s.Serve(lis); err != nil {
		fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
	}
}
