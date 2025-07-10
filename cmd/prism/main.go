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
	"strings"
	"sync"
	"time"

	"github.com/Ow1Dev/FuncWoo/internal/logger"
	"github.com/Ow1Dev/FuncWoo/internal/routes"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"

	pb "github.com/Ow1Dev/FuncWoo/pkgs/api/communication"
)

func newServer() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Path

    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read body", http.StatusBadRequest)
        return
    }
    defer r.Body.Close()

		

		// if the URL is just "/", return a message
		if url == "/" {
			http.Error(w, "No action specified", http.StatusBadRequest)
			return
		}

		// get action from route folder
		action := r.URL.Path[1:]
		action = strings.ReplaceAll(action, "/", ".") 

		log.Debug().Msgf("Received action: %s", action)

		// find file that matches the action
		filePath := fmt.Sprintf("/var/lib/funcwoo/routes/%s.yml", action)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf("Action %s not found", action), http.StatusNotFound)
			return
		}
		// read the file and parse the action
		// for now, just return a success message
		log.Printf("Action %s found, processing...\n", action)

		// get the content of the file
		data, err := os.ReadFile(filePath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading action file: %s", err), http.StatusInternalServerError)
			return
		}

		cfg := routes.RouteConfig{}
		yerr := yaml.Unmarshal([]byte(data), &cfg)
		if yerr != nil {
			http.Error(w, fmt.Sprintf("Error parsing action file: %s", yerr), http.StatusInternalServerError)
			return
		}

		if cfg.Method != r.Method {
			http.Error(w, fmt.Sprintf("Method %s not allowed for action %s", r.Method, action), http.StatusMethodNotAllowed)
			return
		}

		log.Printf("Parsed action: %+v\n", cfg)

		// send the action to the executer
		sendAction, err := sendAction(cfg.Action, string(body), r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("Error sending action: %s", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sendAction))
	})
	return mux
}


func sendAction(action string, body string, ctx context.Context) (string, error) {
	//TODO: get url from Container implementation
	conn, err := grpc.NewClient("localhost:5001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	defer conn.Close()

	client := pb.NewCommunicationServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := client.Execute(ctx, &pb.ExecuteRequest{
		Action: action,
		Body: body,
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute command in Docker container: %w", err)
	}

	if r.Status != "success" {
		return "", fmt.Errorf("There was a error executing the command")
	}

	// Implement the logic to execute the command in the Docker container
	return r.Resp, nil
}

func run(ctx context.Context, w io.Writer, args []string) error {
	_ = args

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	debug := flag.Bool("debug", false, "sets log level to debug")
	flag.Parse()

	logger.InitLog(w, *debug)

	srv := newServer()
	httpServer := &http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", "5000"),
		Handler: srv,
	}
	go func() {
		log.Printf("listening on %s\n", httpServer.Addr)
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
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 10 * time.Second)
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

