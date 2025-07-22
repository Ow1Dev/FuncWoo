package network

import (
	"net"
	"time"
)

type NetTransport interface {
	Listen(network, address string) (net.Listener, error)
	DialTimeout(network, address string, timeout time.Duration) (net.Conn, error)
}

type RealNetwork struct{}

func (r *RealNetwork) Listen(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

func (r *RealNetwork) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(network, address, timeout)
}
