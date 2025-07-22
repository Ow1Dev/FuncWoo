package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/Ow1Dev/NoctiFunc/internal/container"
	"github.com/Ow1Dev/NoctiFunc/internal/executer"
	"github.com/Ow1Dev/NoctiFunc/internal/funcinvoker"
	"github.com/Ow1Dev/NoctiFunc/internal/keyservice"
	pb "github.com/Ow1Dev/NoctiFunc/pkg/api/communication"
	"github.com/Ow1Dev/NoctiFunc/pkg/logger"
	"github.com/Ow1Dev/NoctiFunc/pkg/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

const (
	Version = "0.1.0"
	AppName = "igniterelay"
	Port    = 5001
)

type serviceServer struct {
	pb.UnimplementedCommunicationServiceServer
	Executer executer.Executer
}

// Execute implements gateway.ServerServiceServer.
func (s *serviceServer) Execute(ctx context.Context, r *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
	rsp, err := s.Executer.Execute(r.GetAction(), r.GetBody(), ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error executing command: %s\n", err)
		return &pb.ExecuteResponse{
			Status: "error",
		}, nil
	}

	return &pb.ExecuteResponse{
		Status: "success",
		Resp:   rsp,
	}, nil
}

func run(ctx context.Context, w io.Writer, args []string) error {
	_ = args

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	debug := flag.Bool("debug", false, "sets log level to debug")
	flag.Parse()

	logger := logger.InitLog(logger.Config{
		Writer:        w,
		Level:         utils.Ternary(*debug, zerolog.DebugLevel, zerolog.InfoLevel),
		AppName:       AppName,
		AppVersion:    Version,
		EnableCaller:  true,
		PrettyConsole: true,
	})
	defer logger.Close()

	dockerRunner, err := container.NewDockerContainerWithDefaults(*logger.GetLogger())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating docker runner: %s\n", err)
		os.Exit(1)
	}

	grpcFuncExecuter := funcinvoker.NewStandardGRPCClient(10 * time.Second)
	fileKeyService := keyservice.NewFileSystemKeyService("/var/lib/noctifunc/action")

	executer := executer.NewExecuter(dockerRunner, fileKeyService, grpcFuncExecuter, *logger.GetLogger())

	s := grpc.NewServer()
	pb.RegisterCommunicationServiceServer(s, &serviceServer{
		Executer: *executer,
	})

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", Port))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
		}

		log.Info().Msgf("IgniteRelay server listening on %s", lis.Addr().String())
		if err := s.Serve(lis); err != nil {
			fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		defer cancel()
		s.GracefulStop()
	}()
	wg.Wait()
	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stdout, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
