package executer

import (
	"context"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
)

type MockContainer struct {
	IsRunningFunc func(key string, ctx context.Context) bool
	StartFunc     func(key string, ctx context.Context) error
	GetPortFunc   func(key string, ctx context.Context) int
}

func (m *MockContainer) IsRunning(key string, ctx context.Context) bool {
	if m.IsRunningFunc != nil {
		return m.IsRunningFunc(key, ctx)
	}
	return false
}

func (m *MockContainer) Start(key string, ctx context.Context) error {
	if m.StartFunc != nil {
		return m.StartFunc(key, ctx)
	}
	return nil
}

func (m *MockContainer) GetPort(key string, ctx context.Context) int {
	if m.GetPortFunc != nil {
		return m.GetPortFunc(key, ctx)
	}
	return 8080
}

type MockKeyService struct {
	GetKeyFromActionFunc func(action string) (string, error)
}

func (m *MockKeyService) GetKeyFromAction(action string) (string, error) {
	if m.GetKeyFromActionFunc != nil {
		return m.GetKeyFromActionFunc(action)
	}
	return "test-key", nil
}

type MockGRPCFuncExecuter struct {
	invokeFunc func(ctx context.Context, url, payload string) (string, error)
}

func (m *MockGRPCFuncExecuter) Invoke(ctx context.Context, url, payload string) (string, error) {
	if m.invokeFunc != nil {
		return m.invokeFunc(ctx, url, payload)
	}
	return "mocked response", nil
}

// Test cases
func TextExecuter_Execute_Success_ContainerRunning(t *testing.T) {
	ctx := context.Background()
	mockContainer := &MockContainer{
		IsRunningFunc: func(key string, ctx context.Context) bool {
			return true
		},
		GetPortFunc: func(key string, ctx context.Context) int {
			return 8080
		},
	}

	mockKeyService := &MockKeyService{
		GetKeyFromActionFunc: func(action string) (string, error) {
			return "test-key", nil
		},
	}

	mockGRPCFuncExecuter := &MockGRPCFuncExecuter{
		invokeFunc: func(ctx context.Context, url, payload string) (string, error) {
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
		IsRunningFunc: func(key string, ctx context.Context) bool {
			return false
		},
		StartFunc: func(key string, ctx context.Context) error {
			return nil
		},
		GetPortFunc: func(key string, ctx context.Context) int {
			return 8080
		},
	}

	mockKeyService := &MockKeyService{
		GetKeyFromActionFunc: func(action string) (string, error) {
			return "test-key", nil
		},
	}

	mockGRPCFuncExecuter := &MockGRPCFuncExecuter{
		invokeFunc: func(ctx context.Context, url, payload string) (string, error) {
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
		IsRunningFunc: func(key string, ctx context.Context) bool {
			return false
		},
		StartFunc: func(key string, ctx context.Context) error {
			return nil
		},
		GetPortFunc: func(key string, ctx context.Context) int {
			return 8080
		},
	}

	mockKeyService := &MockKeyService{
		GetKeyFromActionFunc: func(action string) (string, error) {
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
		IsRunningFunc: func(key string, ctx context.Context) bool {
			return false
		},
		StartFunc: func(key string, ctx context.Context) error {
			return fmt.Errorf("container start error")
		},
		GetPortFunc: func(key string, ctx context.Context) int {
			return 8080
		},
	}

	mockKeyService := &MockKeyService{
		GetKeyFromActionFunc: func(action string) (string, error) {
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
		IsRunningFunc: func(key string, ctx context.Context) bool {
			return false
		},
		StartFunc: func(key string, ctx context.Context) error {
			return nil
		},
		GetPortFunc: func(key string, ctx context.Context) int {
			return 0 // Simulating port zero error
		},
	}

	mockKeyService := &MockKeyService{
		GetKeyFromActionFunc: func(action string) (string, error) {
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
		IsRunningFunc: func(key string, ctx context.Context) bool {
			return true
		},
		GetPortFunc: func(key string, ctx context.Context) int {
			return 8080
		},
	}

	mockKeyService := &MockKeyService{
		GetKeyFromActionFunc: func(action string) (string, error) {
			return "test-key", nil
		},
	}

	mockGRPCFuncExecuter := &MockGRPCFuncExecuter{
		invokeFunc: func(ctx context.Context, url, payload string) (string, error) {
			return "", fmt.Errorf("gRPC client error")
		},
	}

	executer := NewExecuter(mockContainer, mockKeyService, mockGRPCFuncExecuter, zerolog.Nop())

	response, err := executer.Execute("test-action", "test-body", ctx)
	if err == nil || response != "" {
		t.Errorf("Expected gRPC client error, got %v", err)
	}
}
