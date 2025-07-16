package executer

import (
	"context"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
)

type MockContainer struct {
	isRunningFunc func(key string, ctx context.Context) bool
	startFunc func(key string, ctx context.Context) error
	getPortFunc func(key string, ctx context.Context) int
}

func (m *MockContainer) isRunning(key string, ctx context.Context) bool {
	if m.isRunningFunc != nil {
		return m.isRunningFunc(key, ctx)
	}
	return false
}

func (m *MockContainer) start(key string, ctx context.Context) error {
	if m.startFunc != nil {
		return m.startFunc(key, ctx)
	}
	return nil
} 

func (m *MockContainer) getPort(key string, ctx context.Context) int {
	if m.getPortFunc != nil {
		return m.getPortFunc(key, ctx)
	}
	return 8080
}

type MockKeyService struct {
	getKeyFromActionFunc func(action string) (string, error)
}

func (m *MockKeyService) getKeyFromAction(action string) (string, error) {
	if m.getKeyFromActionFunc != nil {
		return m.getKeyFromActionFunc(action)
	}
	return "test-key", nil
}

type MockGRPCFuncExecuter struct {
	invokeFunc func(ctx context.Context, url string, payload string) (string, error)
}

func (m *MockGRPCFuncExecuter) Invoke(ctx context.Context, url string, payload string) (string, error) {
	if m.invokeFunc != nil {
		return m.invokeFunc(ctx, url, payload)
	}
	return "mocked response", nil
}

// Test cases
func TextExecuter_Execute_Success_ContainerRunning(t *testing.T) {
	ctx := context.Background()
	mockContainer := &MockContainer{
		isRunningFunc: func(key string, ctx context.Context) bool {
			return true
		},
		getPortFunc: func(key string, ctx context.Context) int {
			return 8080
		},
	}

	mockKeyService := &MockKeyService{
		getKeyFromActionFunc: func(action string) (string, error) {
			return "test-key", nil
		},
	}

	mockGRPCFuncExecuter := &MockGRPCFuncExecuter{
		invokeFunc: func(ctx context.Context, url string, payload string) (string, error) {
			return "mocked response", nil
		},
	}

	executer := NewExecuter(mockContainer, mockKeyService, mockGRPCFuncExecuter, zerolog.Nop())

	result, err := executer.Execute("test-action", "test-body", ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result != "mocked response" {
		t.Errorf("Expected 'success response', got %s", result)
	}
}

func TestExecuter_Execute_Success_ContainerNotRunning(t *testing.T) {
	ctx := context.Background()
	mockContainer := &MockContainer{
		isRunningFunc: func(key string, ctx context.Context) bool {
			return false
		},
		startFunc: func(key string, ctx context.Context) error {
			return nil
		},
		getPortFunc: func(key string, ctx context.Context) int {
			return 8080
		},
	}

	mockKeyService := &MockKeyService{
		getKeyFromActionFunc: func(action string) (string, error) {
			return "test-key", nil
		},
	}

	mockGRPCFuncExecuter := &MockGRPCFuncExecuter{
		invokeFunc: func(ctx context.Context, url string, payload string) (string, error) {
			return "mocked response", nil
		},
	}

	executer := NewExecuter(mockContainer, mockKeyService, mockGRPCFuncExecuter, zerolog.Nop())

	response, err := executer.Execute("test-action", "test-body", ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if response != "mocked response" {
		t.Errorf("Expected 'mocked response', got %s", response)
	}
}

func TestExecuter_Execute_FileReaderError(t *testing.T) {
	ctx := context.Background()
	mockContainer := &MockContainer{
		isRunningFunc: func(key string, ctx context.Context) bool {
			return false
		},
		startFunc: func(key string, ctx context.Context) error {
			return nil
		},
		getPortFunc: func(key string, ctx context.Context) int {
			return 8080
		},
	}

	mockKeyService := &MockKeyService{
		getKeyFromActionFunc: func(action string) (string, error) {
			return "", fmt.Errorf("file read error")
		},
	}

	mockGRPCFuncExecuter := &MockGRPCFuncExecuter{}

	executer := NewExecuter(mockContainer, mockKeyService, mockGRPCFuncExecuter, zerolog.Nop())

	response, err := executer.Execute("test-action", "test-body", ctx)
	if err == nil || response != "" {
		t.Errorf("Expected file read error, got %v", err)
	}
}

func TestExecuter_Execute_ContainerStartError(t *testing.T) {
	ctx := context.Background()
	mockContainer := &MockContainer{
		isRunningFunc: func(key string, ctx context.Context) bool {
			return false
		},
		startFunc: func(key string, ctx context.Context) error {
			return fmt.Errorf("container start error")
		},
		getPortFunc: func(key string, ctx context.Context) int {
			return 8080
		},
	}

	mockKeyService := &MockKeyService{
		getKeyFromActionFunc: func(action string) (string, error) {
			return "test-key", nil
		},
	}

	mockGRPCFuncExecuter := &MockGRPCFuncExecuter{}

	executer := NewExecuter(mockContainer, mockKeyService, mockGRPCFuncExecuter, zerolog.Nop())

	response, err := executer.Execute("test-action", "test-body", ctx)
	if err == nil || response != "" {
		t.Errorf("Expected container start error, got %v", err)
	}
}

func TestExecuter_Execute_PortZeroError(t *testing.T) {
	ctx := context.Background()
	mockContainer := &MockContainer{
		isRunningFunc: func(key string, ctx context.Context) bool {
			return false
		},
		startFunc: func(key string, ctx context.Context) error {
			return nil
		},
		getPortFunc: func(key string, ctx context.Context) int {
			return 0 // Simulating port zero error
		},
	}

	mockKeyService := &MockKeyService{
		getKeyFromActionFunc: func(action string) (string, error) {
			return "test-key", nil
		},
	}

	mockGRPCFuncExecuter := &MockGRPCFuncExecuter{}

	executer := NewExecuter(mockContainer, mockKeyService, mockGRPCFuncExecuter, zerolog.Nop())

	response, err := executer.Execute("test-action", "test-body", ctx)
	if err == nil || response != "" {
		t.Errorf("Expected port zero error, got %v", err)
	}
}

func TestExecuter_Execute_GRPCFuncExecuter(t *testing.T) {
	ctx := context.Background()
	mockContainer := &MockContainer{
		isRunningFunc: func(key string, ctx context.Context) bool {
			return true
		},
		getPortFunc: func(key string, ctx context.Context) int {
			return 8080
		},
	}

	mockKeyService := &MockKeyService{
		getKeyFromActionFunc: func(action string) (string, error) {
			return "test-key", nil
		},
	}

	mockGRPCFuncExecuter := &MockGRPCFuncExecuter{
		invokeFunc: func(ctx context.Context, url string, payload string) (string, error) {
			return "", fmt.Errorf("gRPC client error")
		},
	}

	executer := NewExecuter(mockContainer, mockKeyService, mockGRPCFuncExecuter, zerolog.Nop())

	response, err := executer.Execute("test-action", "test-body", ctx)
	if err == nil || response != "" {
		t.Errorf("Expected gRPC client error, got %v", err)
	}
}


