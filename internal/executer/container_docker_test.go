package executer

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"

	cerrdefs "github.com/containerd/errdefs"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type MockDockerClient struct {
	containerInspectFunc func(ctx context.Context, containerID string) (container.InspectResponse, error)
	containerStartFunc   func(ctx context.Context, containerID string, options container.StartOptions) error
	containerCreateFunc  func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, 
		networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error)
}

func (m *MockDockerClient) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	if m.containerInspectFunc != nil {
		return m.containerInspectFunc(ctx, containerID)
	}
	return container.InspectResponse{}, nil
}

func (m *MockDockerClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	if m.containerStartFunc != nil {
		return m.containerStartFunc(ctx, containerID, options)
	}
	return nil
}

func (m *MockDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, 
	networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error) {
	if m.containerCreateFunc != nil {
		return m.containerCreateFunc(ctx, config, hostConfig, networkingConfig, platform, containerName)
	}
	return container.CreateResponse{ID: "mock-container-id"}, nil
}

type MockNetwork struct {
	listenFunc      func(network, address string) (net.Listener, error)
	dialTimeoutFunc func(network, address string, timeout time.Duration) (net.Conn, error)
}

func (m *MockNetwork) Listen(network, address string) (net.Listener, error) {
	if m.listenFunc != nil {
		return m.listenFunc(network, address)
	}
	// Create a mock listener that returns a fixed port
	return &MockListener{port: 8080}, nil
}

func (m *MockNetwork) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	if m.dialTimeoutFunc != nil {
		return m.dialTimeoutFunc(network, address, timeout)
	}
	return &MockConn{}, nil
}

type MockListener struct {
	port int
}

func (m *MockListener) Accept() (net.Conn, error) {
	return nil, errors.New("not implemented")
}

func (m *MockListener) Close() error {
	return nil
}

func (m *MockListener) Addr() net.Addr {
	return &net.TCPAddr{Port: m.port}
}

type MockConn struct{}

func (m *MockConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (m *MockConn) Close() error {
	return nil
}

func (m *MockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{Port: 8080}
}

func (m *MockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{Port: 8080}
}

func (m *MockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type MockTimeProvider struct {
	sleepFunc func(duration time.Duration)
	nowFunc   func() time.Time
	currentTime time.Time
}

func (m *MockTimeProvider) Sleep(duration time.Duration) {
	if m.sleepFunc != nil {
		m.sleepFunc(duration)
	}
	m.currentTime = m.currentTime.Add(duration)
}

func (m *MockTimeProvider) Now() time.Time {
	if m.nowFunc != nil {
		return m.nowFunc()
	}
	return m.currentTime
}

// Test cases
func TestDockerContainer_isRunning_True(t *testing.T) {
	mockClient := &MockDockerClient{
		containerInspectFunc: func(ctx context.Context, containerID string) (container.InspectResponse, error) {
			return container.InspectResponse{
				ContainerJSONBase: &container.ContainerJSONBase{
					State: &container.State{
						Running: true,
					},
				},
			}, nil
		},
	}
	dockerContainer := NewDockerContainer(mockClient, &MockNetwork{}, &MockTimeProvider{}, DefaultDockerConfig())

	result := dockerContainer.isRunning("test-key", context.Background())

	if !result {
		t.Error("Expected container to be running")
	}
}

func TestDockerContainer_isRunning_False(t *testing.T) {
	mockClient := &MockDockerClient{
		containerInspectFunc: func(ctx context.Context, containerID string) (container.InspectResponse, error) {
			return container.InspectResponse{}, errors.New("container not found")
		},
	}

	dockerContainer := NewDockerContainer(mockClient, &MockNetwork{}, &MockTimeProvider{}, DefaultDockerConfig())

	result := dockerContainer.isRunning("test-key", context.Background())

	if result {
		t.Error("Expected container to not be running")
	}
}

func TestDockerContainer_getPort_Success(t *testing.T) {
	mockClient := &MockDockerClient{
		containerInspectFunc: func(ctx context.Context, containerID string) (container.InspectResponse, error) {
			return container.InspectResponse{
				NetworkSettings: &container.NetworkSettings{
					NetworkSettingsBase: container.NetworkSettingsBase{
						Ports: nat.PortMap{
							"8080/tcp": []nat.PortBinding{
								{HostPort: "9090"},
							},
						},
					},
				},
			}, nil
		},
	}

	dockerContainer := NewDockerContainer(mockClient, &MockNetwork{}, &MockTimeProvider{}, DefaultDockerConfig())

	port := dockerContainer.getPort("test-key", context.Background())

	if port != 9090 {
		t.Errorf("Expected port 9090, got %d", port)
	}
}
func TestDockerContainer_getPort_NoMapping(t *testing.T) {
	mockClient := &MockDockerClient{
		containerInspectFunc: func(ctx context.Context, containerID string) (container.InspectResponse, error) {
			return container.InspectResponse{
				NetworkSettings: &container.NetworkSettings{
					NetworkSettingsBase: container.NetworkSettingsBase{
						Ports: nat.PortMap{},
					},
				},
			}, nil
		},
	}

	dockerContainer := NewDockerContainer(mockClient, &MockNetwork{}, &MockTimeProvider{}, DefaultDockerConfig())

	port := dockerContainer.getPort("test-key", context.Background())

	if port != 0 {
		t.Errorf("Expected port 0, got %d", port)
	}
}

func TestDockerContainer_start_Success(t *testing.T) {
	startCalled := false
	inspectCallCount := 0
	
	mockClient := &MockDockerClient{
		containerInspectFunc: func(ctx context.Context, containerID string) (container.InspectResponse, error) {
			inspectCallCount++
			if inspectCallCount == 1 {
				// First call - container doesn't exist
				return container.InspectResponse{}, cerrdefs.ErrNotFound
			}
			// Subsequent calls - container is running
			return container.InspectResponse{
				ContainerJSONBase: &container.ContainerJSONBase{
					State: &container.State{
						Running: true,
					},
				},
				NetworkSettings: &container.NetworkSettings{
					NetworkSettingsBase: container.NetworkSettingsBase{
						Ports: nat.PortMap{
							"8080/tcp": []nat.PortBinding{
								{HostPort: "9090"},
							},
						},
					},
				},
			}, nil
		},
		containerStartFunc: func(ctx context.Context, containerID string, options container.StartOptions) error {
			startCalled = true
			return nil
		},
		containerCreateFunc: func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, 
			networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error) {
			return container.CreateResponse{ID: "new-container-id"}, nil
		},
	}

	mockNetwork := &MockNetwork{
		listenFunc: func(network, address string) (net.Listener, error) {
			return &MockListener{port: 9090}, nil
		},
		dialTimeoutFunc: func(network, address string, timeout time.Duration) (net.Conn, error) {
			return &MockConn{}, nil
		},
	}

	mockTime := &MockTimeProvider{
		currentTime: time.Now(),
	}

	dockerContainer := NewDockerContainer(mockClient, mockNetwork, mockTime, DefaultDockerConfig())

	err := dockerContainer.start("test-key", context.Background())

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !startCalled {
		t.Error("Expected container start to be called")
	}
}

func TestDockerContainer_start_StartError(t *testing.T) {
	mockClient := &MockDockerClient{
		containerInspectFunc: func(ctx context.Context, containerID string) (container.InspectResponse, error) {
			return container.InspectResponse{}, cerrdefs.ErrNotFound
		},
		containerStartFunc: func(ctx context.Context, containerID string, options container.StartOptions) error {
			return errors.New("start failed")
		},
		containerCreateFunc: func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, 
			networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error) {
			return container.CreateResponse{ID: "new-container-id"}, nil
		},
	}

	dockerContainer := NewDockerContainer(mockClient, &MockNetwork{}, &MockTimeProvider{}, DefaultDockerConfig())

	err := dockerContainer.start("test-key", context.Background())

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !errors.Is(err, errors.New("start failed")) {
		expectedMsg := "failed to start container: start failed"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	}
}

func TestDockerContainer_waitForContainer_Success(t *testing.T) {
	mockClient := &MockDockerClient{
		containerInspectFunc: func(ctx context.Context, containerID string) (container.InspectResponse, error) {
			return container.InspectResponse{
				ContainerJSONBase: &container.ContainerJSONBase{
					State: &container.State{
						Running: true,
					},
				},
				NetworkSettings: &container.NetworkSettings{
					NetworkSettingsBase: container.NetworkSettingsBase{
						Ports: nat.PortMap{
							"8080/tcp": []nat.PortBinding{
								{HostPort: "9090"},
							},
						},
					},
				},
			}, nil
		},
	}

	mockNetwork := &MockNetwork{
		dialTimeoutFunc: func(network, address string, timeout time.Duration) (net.Conn, error) {
			return &MockConn{}, nil
		},
	}

	mockTime := &MockTimeProvider{
		currentTime: time.Now(),
	}

	dockerContainer := NewDockerContainer(mockClient, mockNetwork, mockTime, DefaultDockerConfig())

	err := dockerContainer.waitForContainer("test-key", context.Background())

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestDockerContainer_waitForContainer_Timeout(t *testing.T) {
	mockClient := &MockDockerClient{
		containerInspectFunc: func(ctx context.Context, containerID string) (container.InspectResponse, error) {
			return container.InspectResponse{
				ContainerJSONBase: &container.ContainerJSONBase{
					State: &container.State{
						Running: false,
					},
				},
			}, nil
		},
	}

	mockTime := &MockTimeProvider{
		currentTime: time.Now(),
	}

	// Configure a very short timeout
	config := DefaultDockerConfig()
	config.ContainerReadyTimeout = time.Millisecond

	dockerContainer := NewDockerContainer(mockClient, &MockNetwork{}, mockTime, config)

	err := dockerContainer.waitForContainer("test-key", context.Background())

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	if !errors.Is(err, errors.New("container test-key did not become ready within timeout")) {
		expectedMsg := "container test-key did not become ready within timeout"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	}
}

func TestDockerContainer_getRandomPort(t *testing.T) {
	mockNetwork := &MockNetwork{
		listenFunc: func(network, address string) (net.Listener, error) {
			return &MockListener{port: 12345}, nil
		},
	}

	dockerContainer := NewDockerContainer(&MockDockerClient{}, mockNetwork, &MockTimeProvider{}, DefaultDockerConfig())

	port, err := dockerContainer.getRandomPort()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if port != 12345 {
		t.Errorf("Expected port 12345, got %d", port)
	}
}

func TestDockerContainer_getRandomPort_Error(t *testing.T) {
	mockNetwork := &MockNetwork{
		listenFunc: func(network, address string) (net.Listener, error) {
			return nil, errors.New("network error")
		},
	}

	dockerContainer := NewDockerContainer(&MockDockerClient{}, mockNetwork, &MockTimeProvider{}, DefaultDockerConfig())

	port, err := dockerContainer.getRandomPort()

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if port != 0 {
		t.Errorf("Expected port 0, got %d", port)
	}
}

func TestDockerContainer_create_Success(t *testing.T) {
	createCalled := false
	
	mockClient := &MockDockerClient{
		containerCreateFunc: func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, 
			networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error) {
			createCalled = true
			
			// Verify configuration
			if config.Image != "noctifunc/base" {
				t.Errorf("Expected image 'noctifunc/base', got '%s'", config.Image)
			}
			if containerName != "test-key" {
				t.Errorf("Expected container name 'test-key', got '%s'", containerName)
			}
			
			return container.CreateResponse{ID: "created-container-id"}, nil
		},
	}

	mockNetwork := &MockNetwork{
		listenFunc: func(network, address string) (net.Listener, error) {
			return &MockListener{port: 9090}, nil
		},
	}

	dockerContainer := NewDockerContainer(mockClient, mockNetwork, &MockTimeProvider{}, DefaultDockerConfig())

	containerID, err := dockerContainer.create("test-key", context.Background())

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if containerID != "created-container-id" {
		t.Errorf("Expected container ID 'created-container-id', got '%s'", containerID)
	}
	if !createCalled {
		t.Error("Expected container create to be called")
	}
}

func TestDockerContainer_create_Error(t *testing.T) {
	mockClient := &MockDockerClient{
		containerCreateFunc: func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, 
			networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error) {
			return container.CreateResponse{}, errors.New("create failed")
		},
	}

	mockNetwork := &MockNetwork{
		listenFunc: func(network, address string) (net.Listener, error) {
			return &MockListener{port: 9090}, nil
		},
	}

	dockerContainer := NewDockerContainer(mockClient, mockNetwork, &MockTimeProvider{}, DefaultDockerConfig())

	containerID, err := dockerContainer.create("test-key", context.Background())

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if containerID != "" {
		t.Errorf("Expected empty container ID, got '%s'", containerID)
	}
}

func TestDefaultDockerConfig(t *testing.T) {
	config := DefaultDockerConfig()
	
	if config.Image != "noctifunc/base" {
		t.Errorf("Expected image 'noctifunc/base', got '%s'", config.Image)
	}
	if config.InternalPort != "8080/tcp" {
		t.Errorf("Expected internal port '8080/tcp', got '%s'", config.InternalPort)
	}
	if config.MountSourcePrefix != "/var/lib/noctifunc/funcs/" {
		t.Errorf("Expected mount source prefix '/var/lib/noctifunc/funcs/', got '%s'", config.MountSourcePrefix)
	}
	if config.ContainerReadyTimeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.ContainerReadyTimeout)
	}
}
