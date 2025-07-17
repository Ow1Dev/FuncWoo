package network

import (
	"errors"
	"net"
	"time"
)

type MockNetwork struct {
	ListenFunc      func(network, address string) (net.Listener, error)
	DialTimeoutFunc func(network, address string, timeout time.Duration) (net.Conn, error)
}

func (m *MockNetwork) Listen(network, address string) (net.Listener, error) {
	if m.ListenFunc != nil {
		return m.ListenFunc(network, address)
	}
	// Create a mock listener that returns a fixed port
	return &MockListener{Port: 8080}, nil
}

func (m *MockNetwork) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	if m.DialTimeoutFunc != nil {
		return m.DialTimeoutFunc(network, address, timeout)
	}
	return &MockConn{}, nil
}

type MockListener struct {
	Port int
}

func (m *MockListener) Accept() (net.Conn, error) {
	return nil, errors.New("not implemented")
}

func (m *MockListener) Close() error {
	return nil
}

func (m *MockListener) Addr() net.Addr {
	return &net.TCPAddr{Port: m.Port}
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
