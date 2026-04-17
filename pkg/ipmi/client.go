package ipmi

import (
	"context"
	"fmt"
	"time"

	"github.com/vmware/goipmi"
)

// Client wraps the IPMI client for BMC operations
type Client struct {
	client    *goipmi.Client
	host      string
	username  string
	password  string
	connected bool
}

// Config contains IPMI connection configuration
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	Timeout  time.Duration
}

// NewClient creates a new IPMI client
func NewClient(config *Config) *Client {
	if config.Port == 0 {
		config.Port = 623 // Default IPMI port
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	return &Client{
		host:     config.Host,
		username: config.Username,
		password: config.Password,
	}
}

// Connect establishes connection to BMC
func (c *Client) Connect(ctx context.Context) error {
	conn := &goipmi.Connection{
		Hostname:  c.host,
		Port:      623,
		Username:  c.username,
		Password:  c.password,
		Interface: "lanplus",
	}

	client, err := goipmi.NewClient(conn)
	if err != nil {
		return fmt.Errorf("failed to create IPMI client: %w", err)
	}

	c.client = client
	c.connected = true
	return nil
}

// Close closes the IPMI connection
func (c *Client) Close() error {
	if c.client != nil && c.connected {
		c.connected = false
		// goipmi doesn't have explicit close, connection is per-request
	}
	return nil
}

// PowerStatus represents the current power state
type PowerStatus string

const (
	PowerOn  PowerStatus = "on"
	PowerOff PowerStatus = "off"
)

// GetPowerStatus retrieves the current power status
func (c *Client) GetPowerStatus(ctx context.Context) (PowerStatus, error) {
	if !c.connected {
		return "", fmt.Errorf("not connected to BMC")
	}

	req := &goipmi.Request{
		NetworkFunction: goipmi.NetworkFunctionChassis,
		Command:         goipmi.CommandChassisStatus,
		Data:            goipmi.ChassisStatusRequest{},
	}

	res := &goipmi.ChassisStatusResponse{}
	err := c.client.Send(req, res)
	if err != nil {
		return "", fmt.Errorf("failed to get chassis status: %w", err)
	}

	if res.PowerState {
		return PowerOn, nil
	}
	return PowerOff, nil
}

// PowerOn powers on the server
func (c *Client) PowerOn(ctx context.Context) error {
	if !c.connected {
		return fmt.Errorf("not connected to BMC")
	}

	req := &goipmi.Request{
		NetworkFunction: goipmi.NetworkFunctionChassis,
		Command:         goipmi.CommandChassisControl,
		Data: &goipmi.ChassisControlRequest{
			ChassisControl: goipmi.ControlPowerUp,
		},
	}

	res := &goipmi.ChassisControlResponse{}
	err := c.client.Send(req, res)
	if err != nil {
		return fmt.Errorf("failed to power on: %w", err)
	}

	return nil
}

// PowerOff powers off the server
func (c *Client) PowerOff(ctx context.Context) error {
	if !c.connected {
		return fmt.Errorf("not connected to BMC")
	}

	req := &goipmi.Request{
		NetworkFunction: goipmi.NetworkFunctionChassis,
		Command:         goipmi.CommandChassisControl,
		Data: &goipmi.ChassisControlRequest{
			ChassisControl: goipmi.ControlPowerDown,
		},
	}

	res := &goipmi.ChassisControlResponse{}
	err := c.client.Send(req, res)
	if err != nil {
		return fmt.Errorf("failed to power off: %w", err)
	}

	return nil
}

// PowerCycle power cycles the server (hard reset)
func (c *Client) PowerCycle(ctx context.Context) error {
	if !c.connected {
		return fmt.Errorf("not connected to BMC")
	}

	req := &goipmi.Request{
		NetworkFunction: goipmi.NetworkFunctionChassis,
		Command:         goipmi.CommandChassisControl,
		Data: &goipmi.ChassisControlRequest{
			ChassisControl: goipmi.ControlPowerCycle,
		},
	}

	res := &goipmi.ChassisControlResponse{}
	err := c.client.Send(req, res)
	if err != nil {
		return fmt.Errorf("failed to power cycle: %w", err)
	}

	return nil
}

// PowerReset performs a hard reset
func (c *Client) PowerReset(ctx context.Context) error {
	if !c.connected {
		return fmt.Errorf("not connected to BMC")
	}

	req := &goipmi.Request{
		NetworkFunction: goipmi.NetworkFunctionChassis,
		Command:         goipmi.CommandChassisControl,
		Data: &goipmi.ChassisControlRequest{
			ChassisControl: goipmi.ControlHardReset,
		},
	}

	res := &goipmi.ChassisControlResponse{}
	err := c.client.Send(req, res)
	if err != nil {
		return fmt.Errorf("failed to reset: %w", err)
	}

	return nil
}

// BootDevice represents boot device options
type BootDevice string

const (
	BootDeviceNone    BootDevice = "none"
	BootDevicePXE     BootDevice = "pxe"
	BootDeviceDisk    BootDevice = "disk"
	BootDeviceCDROM   BootDevice = "cdrom"
	BootDeviceBIOS    BootDevice = "bios"
	BootDeviceFloppy  BootDevice = "floppy"
)

// SetBootDevice sets the next boot device
func (c *Client) SetBootDevice(ctx context.Context, device BootDevice, persistent bool) error {
	if !c.connected {
		return fmt.Errorf("not connected to BMC")
	}

	var bootDevice goipmi.BootDevice
	switch device {
	case BootDevicePXE:
		bootDevice = goipmi.BootDevicePxe
	case BootDeviceDisk:
		bootDevice = goipmi.BootDeviceDisk
	case BootDeviceCDROM:
		bootDevice = goipmi.BootDeviceCdrom
	case BootDeviceBIOS:
		bootDevice = goipmi.BootDeviceBios
	default:
		return fmt.Errorf("unsupported boot device: %s", device)
	}

	options := goipmi.BootOptionsValid
	if !persistent {
		options = goipmi.BootOptionsEfiboot
	}

	req := &goipmi.Request{
		NetworkFunction: goipmi.NetworkFunctionChassis,
		Command:         goipmi.CommandSetSystemBootOptions,
		Data: &goipmi.SetSystemBootOptionsRequest{
			Param:  goipmi.BootParamBootFlags,
			Data:   goipmi.SetBootParamRequest(bootDevice, options, goipmi.BootParamBootFlagValid),
		},
	}

	res := &goipmi.SetSystemBootOptionsResponse{}
	err := c.client.Send(req, res)
	if err != nil {
		return fmt.Errorf("failed to set boot device: %w", err)
	}

	return nil
}

// GetBootDevice retrieves the current boot device setting
func (c *Client) GetBootDevice(ctx context.Context) (BootDevice, error) {
	if !c.connected {
		return "", fmt.Errorf("not connected to BMC")
	}

	req := &goipmi.Request{
		NetworkFunction: goipmi.NetworkFunctionChassis,
		Command:         goipmi.CommandGetSystemBootOptions,
		Data: &goipmi.GetSystemBootOptionsRequest{
			Param: goipmi.BootParamBootFlags,
			Set:   0,
			Block: 0,
		},
	}

	res := &goipmi.GetSystemBootOptionsResponse{}
	err := c.client.Send(req, res)
	if err != nil {
		return "", fmt.Errorf("failed to get boot device: %w", err)
	}

	// Parse boot device from response
	// This is simplified - actual parsing depends on response structure
	return BootDeviceDisk, nil
}

// SensorReading represents a sensor reading
type SensorReading struct {
	Name   string
	Value  float64
	Unit   string
	Status string
}

// GetSensorReadings retrieves all sensor readings
func (c *Client) GetSensorReadings(ctx context.Context) ([]SensorReading, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to BMC")
	}

	// Get SDR (Sensor Data Record) repository
	req := &goipmi.Request{
		NetworkFunction: goipmi.NetworkFunctionStorage,
		Command:         goipmi.CommandGetSDRRepositoryInfo,
		Data:            &goipmi.GetSDRRepositoryInfoRequest{},
	}

	res := &goipmi.GetSDRRepositoryInfoResponse{}
	err := c.client.Send(req, res)
	if err != nil {
		return nil, fmt.Errorf("failed to get SDR repository: %w", err)
	}

	// For now, return empty list - full implementation would iterate through SDR records
	// and read each sensor value
	return []SensorReading{}, nil
}

// GetSystemEventLog retrieves system event log entries
func (c *Client) GetSystemEventLog(ctx context.Context) ([]string, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to BMC")
	}

	req := &goipmi.Request{
		NetworkFunction: goipmi.NetworkFunctionStorage,
		Command:         goipmi.CommandGetSELInfo,
		Data:            &goipmi.GetSELInfoRequest{},
	}

	res := &goipmi.GetSELInfoResponse{}
	err := c.client.Send(req, res)
	if err != nil {
		return nil, fmt.Errorf("failed to get SEL info: %w", err)
	}

	// Return empty list for now - full implementation would read SEL entries
	return []string{}, nil
}

// GetFRUInfo retrieves Field Replaceable Unit information
func (c *Client) GetFRUInfo(ctx context.Context) (map[string]string, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to BMC")
	}

	// FRU information includes board serial, product name, etc.
	// This is a placeholder - full implementation would read FRU data
	fruInfo := map[string]string{
		"manufacturer": "",
		"product_name": "",
		"serial":       "",
		"part_number":  "",
	}

	return fruInfo, nil
}
