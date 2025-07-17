package ignite

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// Mock implementations for testing
type MockCommunicationClient struct {
	SendActionFunc func(ctx context.Context, action string, body string) (string, error)
}

func (m *MockCommunicationClient) SendAction(ctx context.Context, action string, body string) (string, error) {
	if m.SendActionFunc != nil {
		return m.SendActionFunc(ctx, action, body)
	}
	return `{"result": "success"}`, nil
}

type MockFileReader struct {
	ReadFileFunc   func(filename string) ([]byte, error)
	FileExistsFunc func(filename string) bool
}

func (m *MockFileReader) ReadFile(filename string) ([]byte, error) {
	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(filename)
	}
	return []byte(`method: POST
action: test.action`), nil
}

func (m *MockFileReader) FileExists(filename string) bool {
	if m.FileExistsFunc != nil {
		return m.FileExistsFunc(filename)
	}
	return true
}

func TestServer_HandleAction_Success(t *testing.T) {
	commClient := &MockCommunicationClient{
		SendActionFunc: func(ctx context.Context, action string, body string) (string, error) {
			if action != "test.action" {
				t.Errorf("Expected action 'test.action', got '%s'", action)
			}
			if body != "test body" {
				t.Errorf("Expected body 'test body', got '%s'", body)
			}
			return `{"result": "success"}`, nil
		},
	}

	fileReader := &MockFileReader{}
	server := NewServer(commClient, fileReader, "/test/routes", zerolog.Nop())

	req := httptest.NewRequest("POST", "/test/action", strings.NewReader("test body"))
	w := httptest.NewRecorder()

	server.handleAction(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	expected := `{"result": "success"}`
	if w.Body.String() != expected {
		t.Errorf("Expected response body '%s', got '%s'", expected, w.Body.String())
	}
}

func TestServer_HandleAction_NoAction(t *testing.T) {
	commClient := &MockCommunicationClient{}
	fileReader := &MockFileReader{}
	server := NewServer(commClient, fileReader, "/test/routes", zerolog.Nop())

	req := httptest.NewRequest("POST", "/", nil)
	w := httptest.NewRecorder()

	server.handleAction(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestServer_HandleAction_ActionNotFound(t *testing.T) {
	commClient := &MockCommunicationClient{}
	fileReader := &MockFileReader{
		FileExistsFunc: func(filename string) bool {
			return false
		},
	}
	server := NewServer(commClient, fileReader, "/test/routes", zerolog.Nop())

	req := httptest.NewRequest("POST", "/nonexistent/action", nil)
	w := httptest.NewRecorder()

	server.handleAction(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestServer_HandleAction_CommunicationError(t *testing.T) {
	commClient := &MockCommunicationClient{
		SendActionFunc: func(ctx context.Context, action string, body string) (string, error) {
			return "", errors.New("communication failed")
		},
	}
	fileReader := &MockFileReader{}
	server := NewServer(commClient, fileReader, "/test/routes", zerolog.Nop())

	req := httptest.NewRequest("POST", "/test/action", nil)
	w := httptest.NewRecorder()

	server.handleAction(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestServer_ExtractAction(t *testing.T) {
	server := NewServer(nil, nil, "", zerolog.Nop())

	tests := []struct {
		path     string
		expected string
	}{
		{"/test/action", "test.action"},
		{"/user/create", "user.create"},
		{"/api/v1/users", "api.v1.users"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := server.extractAction(tt.path)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestServer_LoadRouteConfig_Success(t *testing.T) {
	fileReader := &MockFileReader{
		ReadFileFunc: func(filename string) ([]byte, error) {
			return []byte(`method: POST action: test.action`), nil
		},
	}
	server := NewServer(nil, fileReader, "/test/routes", zerolog.Nop())

	// Mock the routes.RouteConfig.Validate method behavior
	// This assumes your RouteConfig has a working Validate method
	cfg, err := server.loadRouteConfig("test.action")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if cfg.Method != "POST" {
		t.Errorf("Expected method 'POST', got '%s'", cfg.Method)
	}
}

func TestServer_LoadRouteConfig_FileNotFound(t *testing.T) {
	fileReader := &MockFileReader{
		FileExistsFunc: func(filename string) bool {
			return false
		},
	}
	server := NewServer(nil, fileReader, "/test/routes", zerolog.Nop())

	_, err := server.loadRouteConfig("nonexistent")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if httpErr, ok := err.(*HTTPError); ok {
		if httpErr.Code != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, httpErr.Code)
		}
	} else {
		t.Errorf("Expected HTTPError, got %T", err)
	}
}
