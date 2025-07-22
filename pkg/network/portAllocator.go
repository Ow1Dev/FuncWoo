package network

import (
	"fmt"
	"net"
)

type PortAllocator interface {
	GetRandomPort() (int, error)
}

type NetworkPortAllocator struct {
	network NetTransport
}

func NewNetworkPortAllocator(network NetTransport) *NetworkPortAllocator {
	return &NetworkPortAllocator{
		network: network,
	}
}

func (a *NetworkPortAllocator) GetRandomPort() (int, error) {
	listener, err := a.network.Listen("tcp", ":0")
	if err != nil {
		return 0, fmt.Errorf("failed to create listener: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	err = listener.Close()
	if err != nil {
		return 0, fmt.Errorf("failed to close listener: %w", err)
	}

	return port, nil
}
