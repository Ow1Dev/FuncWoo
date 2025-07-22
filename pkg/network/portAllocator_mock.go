package network

type MockPortAllocator struct {
	GetRandomPortFunc func() (int, error)
}

func (m MockPortAllocator) GetRandomPort() (int, error) {
	if m.GetRandomPortFunc != nil {
		return m.GetRandomPortFunc()
	}
	// Create a mock listener that returns a fixed port
	return 8080, nil
}

