package routes

import "testing"

func TestRouteConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  RouteConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: RouteConfig{
				Action: "test-action",
				Method: "GET",
			},
			wantErr: false,
		},
		{
			name: "empty action",
			config: RouteConfig{
				Action: "",
				Method: "POST",
			},
			wantErr: true,
		},
		{
			name: "unsupported method",
			config: RouteConfig{
				Action: "test-action",
				Method: "ERROR",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RouteConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
