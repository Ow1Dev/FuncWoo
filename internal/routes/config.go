package routes

import "fmt"

type RouteConfig struct {
	Action string `yaml:"action"`
	Method string `yaml:"method"`
}

func (rc *RouteConfig) Validate() error {
	if rc.Action == "" {
		return fmt.Errorf("action is required")
	}
	switch rc.Method {
	case "GET", "POST", "PUT", "DELETE", "PATCH":
		return nil
	default:
		return fmt.Errorf("unsupported method: %s", rc.Method)
	}
}

