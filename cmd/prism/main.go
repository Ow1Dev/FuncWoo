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
	"github.com/Ow1Dev/NoctiFunc/pkgs/communication"
	"github.com/Ow1Dev/NoctiFunc/pkgs/prism"
)

func run(ctx context.Context, w io.Writer, args []string) error {
	_ = args

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	debug := flag.Bool("debug", false, "sets log level to debug")
	flag.Parse()

	logger := logger.InitLog(w, *debug)

	// Create dependencies
	fileReader := &prism.OSFileReader{}
	grpcClient := communication.NewGRPCClient("localhost:5001", time.Second)
	
	// Create server
	srv := prism.NewServer(grpcClient, fileReader, "/var/lib/noctifunc/routes", logger)
	
	httpServer := &http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", "5000"),
		Handler: srv.Handler(),
	}

	go func() {
		logger.Info().Msgf("prim server listening on 5000")
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
