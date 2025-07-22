package prism

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

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

func loadFromYaml(data []byte) (*RouteConfig, error) {
	var cfg RouteConfig
	err := yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}
	return &cfg, nil
}
