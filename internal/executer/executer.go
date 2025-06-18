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
	// if !e.container.exist(key, ctx) {
	err := e.container.start(key, ctx)
	if err != nil {
		return "", err
	}
	// }

	rsp, err := e.container.execute(key, body, ctx)
	if err != nil {
		return "", err
	}

	return rsp, nil
}
