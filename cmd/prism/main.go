package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/Ow1Dev/NoctiFunc/internal/logger"
	"github.com/Ow1Dev/NoctiFunc/pkg/communication"
	"github.com/Ow1Dev/NoctiFunc/pkg/prism"
	"github.com/Ow1Dev/NoctiFunc/pkg/utils"
	"github.com/rs/zerolog"
)

const (
	Version = "0.1.0"
	AppName = "prism"
	Port    = 5000
)

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

	// Create dependencies
	fileReader := &prism.OSFileReader{}
	grpcClient := communication.NewGRPCClient("localhost:5001", time.Second)

	// Create server
	srv := prism.NewServer(grpcClient, fileReader, "/var/lib/noctifunc/routes", *logger.GetLogger())

	httpServer := &http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", fmt.Sprintf("%d", Port)),
		Handler: srv.Handler(),
	}

	go func() {
		logger.GetLogger().Info().Msgf("prim server listening on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx := context.Background()
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "error shutting down http server: %s\n", err)
		}
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
