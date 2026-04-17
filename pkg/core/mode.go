package core

import (
	"context"
)

// ModeSwitcher handles switching nodes between different management modes
type ModeSwitcher interface {
	// SwitchToAgent upgrades a node from SSH mode to Agent mode
	// This involves deploying the agent binary and starting it
	SwitchToAgent(ctx context.Context, node *Node) error

	// SwitchToSSH downgrades a node from Agent mode to SSH mode
	// This involves stopping and removing the agent
	SwitchToSSH(ctx context.Context, node *Node) error

	// CanSwitch checks if a node can switch modes
	CanSwitch(ctx context.Context, node *Node, fromMode, toMode NodeMode) (bool, error)

	// GetProgress returns the progress of an ongoing mode switch
	GetProgress(ctx context.Context, nodeID string) (*SwitchProgress, error)
}

// SwitchProgress tracks the progress of a mode switch operation
type SwitchProgress struct {
	NodeID      string     `json:"node_id"`
	FromMode    NodeMode   `json:"from_mode"`
	ToMode      NodeMode   `json:"to_mode"`
	Status      TaskStatus `json:"status"`
	CurrentStep string     `json:"current_step"`
	TotalSteps  int        `json:"total_steps"`
	CompletedSteps int     `json:"completed_steps"`
	Error       string     `json:"error,omitempty"`
}

// Firmware represents firmware information
type Firmware struct {
	ID          string                 `json:"id"`
	Vendor      string                 `json:"vendor"`
	Model       string                 `json:"model"`
	Component   string                 `json:"component"` // BIOS, BMC, NIC, etc.
	Version     string                 `json:"version"`
	ReleaseDate string                 `json:"release_date"`
	FilePath    string                 `json:"file_path"`
	Checksum    string                 `json:"checksum"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NodeFirmware represents firmware installed on a node
type NodeFirmware struct {
	NodeID      string `json:"node_id"`
	Component   string `json:"component"`
	Version     string `json:"version"`
	LastUpdated string `json:"last_updated"`
}

// FirmwareRepository defines firmware persistence operations
type FirmwareRepository interface {
	// Create adds a new firmware entry
	Create(ctx context.Context, firmware *Firmware) error

	// Get retrieves firmware by ID
	Get(ctx context.Context, id string) (*Firmware, error)

	// List retrieves firmware matching criteria
	List(ctx context.Context, filter *FirmwareFilter) ([]*Firmware, error)

	// Update updates firmware information
	Update(ctx context.Context, firmware *Firmware) error

	// Delete removes firmware
	Delete(ctx context.Context, id string) error

	// GetNodeFirmware retrieves firmware installed on a node
	GetNodeFirmware(ctx context.Context, nodeID string) ([]*NodeFirmware, error)

	// UpdateNodeFirmware updates firmware version for a node
	UpdateNodeFirmware(ctx context.Context, nf *NodeFirmware) error
}

// FirmwareFilter defines criteria for filtering firmware
type FirmwareFilter struct {
	Vendor    string `json:"vendor,omitempty"`
	Model     string `json:"model,omitempty"`
	Component string `json:"component,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

// FirmwareManager handles firmware operations
type FirmwareManager interface {
	// Scan scans firmware versions across nodes
	Scan(ctx context.Context, filter *NodeFilter) ([]*NodeFirmware, error)

	// Update updates firmware on nodes
	Update(ctx context.Context, nodeIDs []string, firmware *Firmware) error

	// Rollback rolls back firmware to previous version
	Rollback(ctx context.Context, nodeID string, component string) error

	// GetAvailableUpdates checks for available firmware updates
	GetAvailableUpdates(ctx context.Context, nodeID string) ([]*Firmware, error)
}

// OSImage represents an OS installation image
type OSImage struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	OSType      string                 `json:"os_type"`    // linux, windows
	OSVersion   string                 `json:"os_version"` // ubuntu-22.04, rocky-9
	Architecture string                `json:"architecture"`
	ImageType   string                 `json:"image_type"` // iso, qcow2, vmdk
	FilePath    string                 `json:"file_path"`
	Size        int64                  `json:"size"`
	Checksum    string                 `json:"checksum"`
	KickstartTemplate string            `json:"kickstart_template,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// OSImageRepository defines OS image persistence operations
type OSImageRepository interface {
	// Create adds a new OS image
	Create(ctx context.Context, image *OSImage) error

	// Get retrieves an OS image by ID
	Get(ctx context.Context, id string) (*OSImage, error)

	// GetByName retrieves an OS image by name
	GetByName(ctx context.Context, name string) (*OSImage, error)

	// List retrieves OS images matching criteria
	List(ctx context.Context, filter *OSImageFilter) ([]*OSImage, error)

	// Update updates OS image information
	Update(ctx context.Context, image *OSImage) error

	// Delete removes an OS image
	Delete(ctx context.Context, id string) error
}

// OSImageFilter defines criteria for filtering OS images
type OSImageFilter struct {
	OSType       string `json:"os_type,omitempty"`
	Architecture string `json:"architecture,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Offset       int    `json:"offset,omitempty"`
}

// OSManager handles OS installation and upgrade operations
type OSManager interface {
	// Reinstall reinstalls the OS on a node
	Reinstall(ctx context.Context, nodeID string, image *OSImage) error

	// Upgrade upgrades the OS on a node
	Upgrade(ctx context.Context, nodeID string, targetVersion string) error

	// ListAvailableImages lists available OS images
	ListAvailableImages(ctx context.Context, filter *OSImageFilter) ([]*OSImage, error)

	// UploadImage uploads a new OS image
	UploadImage(ctx context.Context, image *OSImage, reader interface{}) error
}
