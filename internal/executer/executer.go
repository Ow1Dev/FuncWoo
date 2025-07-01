package executer

import "context"

type Container interface {
	execute (key string, body string, ctx context.Context) (string, error)
	exist (key string, ctx context.Context) bool
	start (key string, ctx context.Context) error
}

type Executer struct {
	container Container
}

func NewExecuter(container Container) *Executer {
	return &Executer{
		container: container,
	}
}

func(e *Executer) Execeute(key string, body string, ctx context.Context) (string, error) {
	// override key with a fixed value for testing purposes
	key = "7c7677eec81f1b60dc19db9dbe06113c2af58b020cca5aca6106366f38fe11ae"
	if !e.container.exist(key, ctx) {
		err := e.container.start(key, ctx)
		if err != nil {
			return "", err
		}
	}

	rsp, err := e.container.execute(key, body, ctx)
	if err != nil {
		return "", err
	}

	return rsp, nil
}
