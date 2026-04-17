package ipmi

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Manager manages IPMI connections to multiple servers
type Manager struct {
	clients map[string]*Client
	mu      sync.RWMutex
}

// NewManager creates a new IPMI manager
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]*Client),
	}
}

// AddServer adds a server to the IPMI manager
func (m *Manager) AddServer(serverID string, config *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clients[serverID]; exists {
		return fmt.Errorf("server %s already exists", serverID)
	}

	client := NewClient(config)
	m.clients[serverID] = client

	return nil
}

// RemoveServer removes a server from the IPMI manager
func (m *Manager) RemoveServer(serverID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[serverID]
	if !exists {
		return fmt.Errorf("server %s not found", serverID)
	}

	client.Close()
	delete(m.clients, serverID)

	return nil
}

// GetClient retrieves the IPMI client for a server
func (m *Manager) GetClient(serverID string) (*Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[serverID]
	if !exists {
		return nil, fmt.Errorf("server %s not found", serverID)
	}

	return client, nil
}

// PowerOn powers on a server
func (m *Manager) PowerOn(ctx context.Context, serverID string) error {
	client, err := m.GetClient(serverID)
	if err != nil {
		return err
	}

	if !client.connected {
		if err := client.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	return client.PowerOn(ctx)
}

// PowerOff powers off a server
func (m *Manager) PowerOff(ctx context.Context, serverID string) error {
	client, err := m.GetClient(serverID)
	if err != nil {
		return err
	}

	if !client.connected {
		if err := client.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	return client.PowerOff(ctx)
}

// PowerCycle power cycles a server
func (m *Manager) PowerCycle(ctx context.Context, serverID string) error {
	client, err := m.GetClient(serverID)
	if err != nil {
		return err
	}

	if !client.connected {
		if err := client.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	return client.PowerCycle(ctx)
}

// SetBootDevice sets the boot device for next boot
func (m *Manager) SetBootDevice(ctx context.Context, serverID string, device BootDevice, persistent bool) error {
	client, err := m.GetClient(serverID)
	if err != nil {
		return err
	}

	if !client.connected {
		if err := client.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	return client.SetBootDevice(ctx, device, persistent)
}

// GetPowerStatus retrieves the power status of a server
func (m *Manager) GetPowerStatus(ctx context.Context, serverID string) (PowerStatus, error) {
	client, err := m.GetClient(serverID)
	if err != nil {
		return "", err
	}

	if !client.connected {
		if err := client.Connect(ctx); err != nil {
			return "", fmt.Errorf("failed to connect: %w", err)
		}
	}

	return client.GetPowerStatus(ctx)
}

// ServerStatus represents the complete status of a server
type ServerStatus struct {
	PowerState PowerStatus
	BootDevice BootDevice
	Sensors    []SensorReading
	Online     bool
	LastCheck  time.Time
}

// GetServerStatus retrieves complete server status
func (m *Manager) GetServerStatus(ctx context.Context, serverID string) (*ServerStatus, error) {
	client, err := m.GetClient(serverID)
	if err != nil {
		return nil, err
	}

	if !client.connected {
		if err := client.Connect(ctx); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	status := &ServerStatus{
		Online:    true,
		LastCheck: time.Now(),
	}

	// Get power status
	powerStatus, err := client.GetPowerStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get power status: %w", err)
	}
	status.PowerState = powerStatus

	// Get boot device
	bootDevice, err := client.GetBootDevice(ctx)
	if err != nil {
		// Don't fail if boot device retrieval fails
		bootDevice = BootDeviceNone
	}
	status.BootDevice = bootDevice

	// Get sensor readings
	sensors, err := client.GetSensorReadings(ctx)
	if err != nil {
		// Don't fail if sensor reading fails
		sensors = []SensorReading{}
	}
	status.Sensors = sensors

	return status, nil
}

// InstallOS triggers OS installation via PXE boot
func (m *Manager) InstallOS(ctx context.Context, serverID string) error {
	// Set boot device to PXE (one-time boot)
	if err := m.SetBootDevice(ctx, serverID, BootDevicePXE, false); err != nil {
		return fmt.Errorf("failed to set PXE boot: %w", err)
	}

	// Power cycle to boot from PXE
	if err := m.PowerCycle(ctx, serverID); err != nil {
		return fmt.Errorf("failed to power cycle: %w", err)
	}

	return nil
}

// Close closes all IPMI connections
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, client := range m.clients {
		client.Close()
	}

	m.clients = make(map[string]*Client)
	return nil
}
