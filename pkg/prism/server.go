package prism

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type CommunicationClient interface {
	SendAction(ctx context.Context, action, body string) (string, error)
}

type FileReader interface {
	ReadFile(filePath string) ([]byte, error)
	FileExists(filePath string) bool
}

type OSFileReader struct{}

func (r *OSFileReader) ReadFile(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func (r *OSFileReader) FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

type Server struct {
	commClient CommunicationClient
	fileReader FileReader
	routesPath string
	logger     zerolog.Logger
}

func NewServer(commClient CommunicationClient, fileReader FileReader, routesPath string, logger zerolog.Logger) *Server {
	return &Server{
		commClient: commClient,
		fileReader: fileReader,
		routesPath: routesPath,
		logger:     logger.With().Str("component", "prism_server").Logger(),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleAction)
	return mux
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	defer func() {
		if err := r.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing request body: %v\n", err)
		}
	}()

	if r.URL.Path == "/" {
		http.Error(w, "No action specified", http.StatusBadRequest)
		return
	}

	action := s.extractAction(r.URL.Path)
	log.Debug().Msgf("Received action: %s", action)

	result, err := s.processAction(r.Context(), action, string(body), r.Method)
	if err != nil {
		s.handleError(w, err)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write([]byte(result))
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to write response")
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

// extractAction converts URL path to action name
func (s *Server) extractAction(path string) string {
	action := path[1:] // Remove leading "/"
	return strings.ReplaceAll(action, "/", ".")
}

func (s *Server) processAction(ctx context.Context, action, body, method string) (string, error) {
	cfg, err := s.loadRouteConfig(action)
	if err != nil {
		return "", err
	}

	if cfg.Method != method {
		return "", &HTTPError{
			Code:    http.StatusMethodNotAllowed,
			Message: "Method not allowed for action " + action,
		}
	}

	s.logger.Debug().Msgf("Processing action: %s with method: %s", action, method)

	resutl, err := s.commClient.SendAction(ctx, action, body)
	if err != nil {
		return "", &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Error processing action: " + err.Error(),
		}
	}

	return resutl, nil
}

// loadRouteConfig loads and parses route configuration from file
func (s *Server) loadRouteConfig(action string) (*RouteConfig, error) {
	filePath := fmt.Sprintf("%s/%s.yml", s.routesPath, action)

	if !s.fileReader.FileExists(filePath) {
		return nil, &HTTPError{
			Code:    http.StatusNotFound,
			Message: "Action " + action + " not found",
		}
	}

	data, err := s.fileReader.ReadFile(filePath)
	if err != nil {
		return nil, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Error reading action file: " + err.Error(),
		}
	}

	cfg, err := loadFromYaml(data)
	if err != nil {
		return nil, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Error parsing action file: " + err.Error(),
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Error validating action config: " + err.Error(),
		}
	}

	return cfg, nil
}

type HTTPError struct {
	Code    int
	Message string
}

func (e *HTTPError) Error() string {
	return e.Message
}

func (s *Server) handleError(w http.ResponseWriter, err error) {
	if httpErr, ok := err.(*HTTPError); ok {
		http.Error(w, httpErr.Message, httpErr.Code)
		return
	}

	s.logger.Error().Err(err).Msg("Internal server error")
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
