package executer

import (
	"context"
	"strconv"

	"fmt"

	"github.com/rs/zerolog/log"
)

type GRPCFuncExecuter interface {
	Invoke(ctx context.Context, url string, payload string) (string, error)
}

type KeyService interface {
	getKeyFromAction(action string) (string, error)
}

type Container interface {
	getPort (key string, ctx context.Context) int
	isRunning (key string, ctx context.Context) bool
	start (key string, ctx context.Context) error
}

type Executer struct {
	container Container
  grpcFuncExecuter GRPCFuncExecuter
  keyService KeyService
}

func NewExecuter(container Container, keyService KeyService, grpcFuncExecuter GRPCFuncExecuter) *Executer {
	return &Executer{
		container: container,
		grpcFuncExecuter: grpcFuncExecuter,
		keyService: keyService,
	}
}

func (e *Executer) Execute(action string, body string, ctx context.Context) (string, error) {
	key, err := e.keyService.getKeyFromAction(action)
	if err != nil {
		return "", fmt.Errorf("failed to get key from action: %w", err)
	}

	var port int
	if !e.container.isRunning(key, ctx) {
		log.Info().Msgf("Container is not running, starting new container with key: %s", key)
		err = e.container.start(key, ctx)
		if err != nil {
			return "", fmt.Errorf("failed to start container: %w", err)
		}
	} 

	log.Debug().Msgf("Container already exists, getting port for key: %s", key)
	port = e.container.getPort(key, ctx)
	
	if port == 0 {
		return "", fmt.Errorf("failed to get port for container: %s", key)
	}
	
	// TODO: get url from configuration or environment variable
	log.Info().Msgf("Making request to localhost:%d", port)
	rsp, err := e.grpcFuncExecuter.Invoke(ctx, "localhost:"+strconv.Itoa(port), body)
	if err != nil {
		return "", fmt.Errorf("failed to handle request: %w", err)
	}
	
	return rsp, nil
}
