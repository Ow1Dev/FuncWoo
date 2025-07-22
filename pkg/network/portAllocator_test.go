package network

import (
	"errors"
	"net"
	"testing"
)

func TestPortAllocator_getRandomPort(t *testing.T) {
	networkMock := &MockNetwork{
		ListenFunc: func(network, address string) (net.Listener, error) {
			return &MockListener{Port: 12345}, nil
		},
	}

	allocator := NewNetworkPortAllocator(networkMock)

	port, err := allocator.GetRandomPort()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if port != 12345 {
		t.Fatalf("expected port 12345, got %d", port)
	}
}

func TestPortAllocator_getRandomPort_error(t *testing.T) {
	networkMock := &MockNetwork{
		ListenFunc: func(network, address string) (net.Listener, error) {
			return nil, errors.New("network error")
		},
	}

	allocator := NewNetworkPortAllocator(networkMock)

	port, err := allocator.GetRandomPort()

	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if port != 0 {
		t.Fatalf("expected port 0, got %d", port)
	}
}
