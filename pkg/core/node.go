package core

import (
	"context"
	"time"
)

// NodeMode represents the management mode of a node
type NodeMode string

const (
	// NodeModeSSH represents SSH-based agentless management
	NodeModeSSH NodeMode = "ssh"

	// NodeModeAgent represents persistent agent-based management
	NodeModeAgent NodeMode = "agent"

	// NodeModeHybrid supports both SSH and Agent modes
	NodeModeHybrid NodeMode = "hybrid"
)

// NodeStatus represents the operational status of a node
type NodeStatus string

const (
	NodeStatusOnline      NodeStatus = "online"
	NodeStatusOffline     NodeStatus = "offline"
	NodeStatusMaintenance NodeStatus = "maintenance"
	NodeStatusInstalling  NodeStatus = "installing"
	NodeStatusUpgrading   NodeStatus = "upgrading"
	NodeStatusFailed      NodeStatus = "failed"
)

// Node represents a managed server/node
type Node struct {
	ID           string                 `json:"id"`
	Hostname     string                 `json:"hostname"`
	IPAddress    string                 `json:"ip_address"`
	MACAddress   string                 `json:"mac_address,omitempty"`
	Mode         NodeMode               `json:"mode"`
	Status       NodeStatus             `json:"status"`
	OSType       string                 `json:"os_type"`        // linux, windows
	OSVersion    string                 `json:"os_version"`     // ubuntu-22.04, rocky-9
	Architecture string                 `json:"architecture"`   // amd64, arm64
	AgentVersion string                 `json:"agent_version,omitempty"`
	LastSeen     time.Time              `json:"last_seen"`
	Tags         map[string]string      `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
	SSHConfig    *SSHConfig             `json:"ssh_config,omitempty"`
	BMCConfig    *BMCConfig             `json:"bmc_config,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// SSHConfig contains SSH connection configuration
type SSHConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	KeyPath    string `json:"key_path,omitempty"`
	Password   string `json:"password,omitempty"` // Encrypted
	JumpHost   string `json:"jump_host,omitempty"`
	Timeout    int    `json:"timeout"` // seconds
}

// BMCConfig contains BMC/IPMI configuration
type BMCConfig struct {
	Address  string `json:"address"`
	User     string `json:"user"`
	Password string `json:"password"` // Encrypted
	Type     string `json:"type"`     // ipmi, redfish
}

// NodeFilter defines criteria for filtering nodes
type NodeFilter struct {
	IDs          []string          `json:"ids,omitempty"`
	Hostnames    []string          `json:"hostnames,omitempty"`
	Mode         NodeMode          `json:"mode,omitempty"`
	Status       NodeStatus        `json:"status,omitempty"`
	OSType       string            `json:"os_type,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
	IPRange      string            `json:"ip_range,omitempty"`
	LastSeenFrom *time.Time        `json:"last_seen_from,omitempty"`
	LastSeenTo   *time.Time        `json:"last_seen_to,omitempty"`
	Limit        int               `json:"limit,omitempty"`
	Offset       int               `json:"offset,omitempty"`
}

// NodeRepository defines the interface for node persistence
type NodeRepository interface {
	// Create adds a new node
	Create(ctx context.Context, node *Node) error

	// Get retrieves a node by ID
	Get(ctx context.Context, id string) (*Node, error)

	// GetByIP retrieves a node by IP address
	GetByIP(ctx context.Context, ip string) (*Node, error)

	// GetByHostname retrieves a node by hostname
	GetByHostname(ctx context.Context, hostname string) (*Node, error)

	// List retrieves nodes matching the filter
	List(ctx context.Context, filter *NodeFilter) ([]*Node, error)

	// Update updates an existing node
	Update(ctx context.Context, node *Node) error

	// UpdateStatus updates only the status and last_seen
	UpdateStatus(ctx context.Context, id string, status NodeStatus, lastSeen time.Time) error

	// UpdateMode changes the node's management mode
	UpdateMode(ctx context.Context, id string, mode NodeMode) error

	// Delete removes a node
	Delete(ctx context.Context, id string) error

	// Count returns the total number of nodes matching the filter
	Count(ctx context.Context, filter *NodeFilter) (int64, error)

	// AddTags adds or updates tags for a node
	AddTags(ctx context.Context, id string, tags map[string]string) error

	// RemoveTags removes tags from a node
	RemoveTags(ctx context.Context, id string, tagKeys []string) error
}

// NodeService defines the business logic for node management
type NodeService interface {
	// Register registers a new node (SSH mode initial onboarding)
	Register(ctx context.Context, ip string, sshConfig *SSHConfig) (*Node, error)

	// Discover attempts to discover node information via SSH
	Discover(ctx context.Context, ip string, sshConfig *SSHConfig) (*Node, error)

	// Get retrieves a node by ID or identifier
	Get(ctx context.Context, identifier string) (*Node, error)

	// List retrieves nodes matching the filter
	List(ctx context.Context, filter *NodeFilter) ([]*Node, error)

	// Update updates node information
	Update(ctx context.Context, node *Node) error

	// Delete removes a node from management
	Delete(ctx context.Context, id string) error

	// SwitchMode switches a node's management mode
	SwitchMode(ctx context.Context, id string, fromMode, toMode NodeMode) error

	// UpdateTags updates node tags
	UpdateTags(ctx context.Context, id string, tags map[string]string) error

	// HealthCheck performs health check on nodes
	HealthCheck(ctx context.Context, filter *NodeFilter) error

	// GetStats returns statistics about managed nodes
	GetStats(ctx context.Context) (*NodeStats, error)
}

// NodeStats contains statistics about managed nodes
type NodeStats struct {
	Total          int64            `json:"total"`
	Online         int64            `json:"online"`
	Offline        int64            `json:"offline"`
	ByMode         map[NodeMode]int64   `json:"by_mode"`
	ByOS           map[string]int64 `json:"by_os"`
	ByArchitecture map[string]int64 `json:"by_architecture"`
}
