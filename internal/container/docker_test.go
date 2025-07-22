package container

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	dockernet "github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/rs/zerolog"

	cerrdefs "github.com/containerd/errdefs"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	netpkg "github.com/Ow1Dev/NoctiFunc/pkg/network"
)

type MockDockerClient struct {
	containerInspectFunc func(ctx context.Context, containerID string) (container.InspectResponse, error)
	containerStartFunc   func(ctx context.Context, containerID string, options container.StartOptions) error
	containerCreateFunc  func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig,
		networkingConfig *dockernet.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error)
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
	networkingConfig *dockernet.NetworkingConfig, platform *ocispec.Platform, containerName string,
) (container.CreateResponse, error) {
	if m.containerCreateFunc != nil {
		return m.containerCreateFunc(ctx, config, hostConfig, networkingConfig, platform, containerName)
	}
	return container.CreateResponse{ID: "mock-container-id"}, nil
}

type MockTimeProvider struct {
	sleepFunc   func(duration time.Duration)
	nowFunc     func() time.Time
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
	dockerContainer := NewDockerContainer(mockClient, &netpkg.MockPortAllocator{}, &netpkg.MockNetwork{}, &MockTimeProvider{}, DefaultDockerConfig(), zerolog.Nop())

	result := dockerContainer.IsRunning("test-key", context.Background())

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

	dockerContainer := NewDockerContainer(mockClient, &netpkg.MockPortAllocator{}, &netpkg.MockNetwork{}, &MockTimeProvider{}, DefaultDockerConfig(), zerolog.Nop())

	result := dockerContainer.IsRunning("test-key", context.Background())

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

	dockerContainer := NewDockerContainer(mockClient, netpkg.MockPortAllocator{}, &netpkg.MockNetwork{}, &MockTimeProvider{}, DefaultDockerConfig(), zerolog.Nop())

	port := dockerContainer.GetPort("test-key", context.Background())

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

	dockerContainer := NewDockerContainer(mockClient, &netpkg.MockPortAllocator{}, &netpkg.MockNetwork{}, &MockTimeProvider{}, DefaultDockerConfig(), zerolog.Nop())

	port := dockerContainer.GetPort("test-key", context.Background())

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
			networkingConfig *dockernet.NetworkingConfig, platform *ocispec.Platform, containerName string,
		) (container.CreateResponse, error) {
			return container.CreateResponse{ID: "new-container-id"}, nil
		},
	}

	mockNetwork := &netpkg.MockNetwork{
		ListenFunc: func(network, address string) (net.Listener, error) {
			return &netpkg.MockListener{Port: 9090}, nil
		},
		DialTimeoutFunc: func(network, address string, timeout time.Duration) (net.Conn, error) {
			return &netpkg.MockConn{}, nil
		},
	}

	mockPortAllocator := &netpkg.MockPortAllocator{
		GetRandomPortFunc: func() (int, error) {
			return 9090, nil
		},
	}

	mockTime := &MockTimeProvider{
		currentTime: time.Now(),
	}

	dockerContainer := NewDockerContainer(mockClient, mockPortAllocator, mockNetwork, mockTime, DefaultDockerConfig(), zerolog.Nop())

	err := dockerContainer.Start("test-key", context.Background())
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
			networkingConfig *dockernet.NetworkingConfig, platform *ocispec.Platform, containerName string,
		) (container.CreateResponse, error) {
			return container.CreateResponse{ID: "new-container-id"}, nil
		},
	}

	dockerContainer := NewDockerContainer(mockClient, &netpkg.MockPortAllocator{}, &netpkg.MockNetwork{}, &MockTimeProvider{}, DefaultDockerConfig(), zerolog.Nop())

	err := dockerContainer.Start("test-key", context.Background())

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

	mockNetwork := &netpkg.MockNetwork{
		DialTimeoutFunc: func(network, address string, timeout time.Duration) (net.Conn, error) {
			return &netpkg.MockConn{}, nil
		},
	}

	mockPortAllocator := &netpkg.MockPortAllocator{
		GetRandomPortFunc: func() (int, error) {
			return 9090, nil
		},
	}

	mockTime := &MockTimeProvider{
		currentTime: time.Now(),
	}

	dockerContainer := NewDockerContainer(mockClient, mockPortAllocator, mockNetwork, mockTime, DefaultDockerConfig(), zerolog.Nop())

	err := dockerContainer.WaitForContainer("test-key", context.Background())
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

	dockerContainer := NewDockerContainer(mockClient, &netpkg.MockPortAllocator{}, &netpkg.MockNetwork{}, mockTime, config, zerolog.Nop())

	err := dockerContainer.WaitForContainer("test-key", context.Background())

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

func TestDockerContainer_create_Success(t *testing.T) {
	createCalled := false

	mockClient := &MockDockerClient{
		containerCreateFunc: func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig,
			networkingConfig *dockernet.NetworkingConfig, platform *ocispec.Platform, containerName string,
		) (container.CreateResponse, error) {
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

	mockNetwork := &netpkg.MockNetwork{
		ListenFunc: func(network, address string) (net.Listener, error) {
			return &netpkg.MockListener{Port: 9090}, nil
		},
	}

	mockPortAllocator := &netpkg.MockPortAllocator{
		GetRandomPortFunc: func() (int, error) {
			return 9090, nil
		},
	}

	dockerContainer := NewDockerContainer(mockClient, mockPortAllocator, mockNetwork, &MockTimeProvider{}, DefaultDockerConfig(), zerolog.Nop())

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
			networkingConfig *dockernet.NetworkingConfig, platform *ocispec.Platform, containerName string,
		) (container.CreateResponse, error) {
			return container.CreateResponse{}, errors.New("create failed")
		},
	}

	mockNetwork := &netpkg.MockNetwork{
		ListenFunc: func(network, address string) (net.Listener, error) {
			return &netpkg.MockListener{Port: 9090}, nil
		},
	}

	mockPortAllocator := &netpkg.MockPortAllocator{
		GetRandomPortFunc: func() (int, error) {
			return 9090, nil
		},
	}

	dockerContainer := NewDockerContainer(mockClient, mockPortAllocator, mockNetwork, &MockTimeProvider{}, DefaultDockerConfig(), zerolog.Nop())

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
