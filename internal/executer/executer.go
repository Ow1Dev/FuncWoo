package executer

import (
	"context"
	"strconv"

	"fmt"

	"github.com/rs/zerolog"
)

type GRPCFuncExecuter interface {
	Invoke(ctx context.Context, url string, payload string) (string, error)
}

type KeyService interface {
	GetKeyFromAction(action string) (string, error)
}

type Container interface {
	GetPort(key string, ctx context.Context) int
	IsRunning(key string, ctx context.Context) bool
	Start(key string, ctx context.Context) error
}

type Executer struct {
	container        Container
	grpcFuncExecuter GRPCFuncExecuter
	keyService       KeyService
	logger           zerolog.Logger
}

func NewExecuter(container Container, keyService KeyService, grpcFuncExecuter GRPCFuncExecuter, logger zerolog.Logger) *Executer {
	return &Executer{
		container:        container,
		grpcFuncExecuter: grpcFuncExecuter,
		keyService:       keyService,
		logger:           logger.With().Str("component", "executer").Logger(),
	}
}

func (e *Executer) Execute(action string, body string, ctx context.Context) (string, error) {
	key, err := e.keyService.GetKeyFromAction(action)
	if err != nil {
		return "", fmt.Errorf("failed to get key from action: %w", err)
	}

	var port int
	if !e.container.IsRunning(key, ctx) {
		e.logger.Info().Msgf("Container is not running, starting new container with key: %s", key)
		err = e.container.Start(key, ctx)
		if err != nil {
			return "", fmt.Errorf("failed to start container: %w", err)
		}
	}

	e.logger.Debug().Msgf("Container already exists, getting port for key: %s", key)
	port = e.container.GetPort(key, ctx)

	if port == 0 {
		return "", fmt.Errorf("failed to get port for container: %s", key)
	}

	// TODO: get url from configuration or environment variable
	e.logger.Info().Msgf("Making request to localhost:%d", port)
	rsp, err := e.grpcFuncExecuter.Invoke(ctx, "localhost:"+strconv.Itoa(port), body)
	if err != nil {
		return "", fmt.Errorf("failed to handle request: %w", err)
	}

	return rsp, nil
}
