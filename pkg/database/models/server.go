package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Server represents a managed bare metal server in the database
type Server struct {
	ID            string         `db:"id" json:"id"`
	Hostname      string         `db:"hostname" json:"hostname"`
	IPAddress     string         `db:"ip_address" json:"ip_address"`
	MACAddress    sql.NullString `db:"mac_address" json:"mac_address,omitempty"`
	OSType        string         `db:"os_type" json:"os_type"`
	OSVersion     sql.NullString `db:"os_version" json:"os_version,omitempty"`
	Architecture  sql.NullString `db:"architecture" json:"architecture,omitempty"`
	Mode          string         `db:"mode" json:"mode"`
	AgentVersion  sql.NullString `db:"agent_version" json:"agent_version,omitempty"`
	Status        string         `db:"status" json:"status"`
	LastSeen      sql.NullTime   `db:"last_seen" json:"last_seen,omitempty"`
	Tags          JSONB          `db:"tags" json:"tags"`
	Metadata      JSONB          `db:"metadata" json:"metadata"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at" json:"updated_at"`
}

// JSONB represents a JSONB field for PostgreSQL
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface for JSONB
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for JSONB
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	result := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}

// ServerStatus represents the possible server statuses
type ServerStatus string

const (
	ServerStatusOnline      ServerStatus = "online"
	ServerStatusOffline     ServerStatus = "offline"
	ServerStatusInstalling  ServerStatus = "installing"
	ServerStatusUpgrading   ServerStatus = "upgrading"
	ServerStatusMaintenance ServerStatus = "maintenance"
)

// AgentMode represents the agent deployment mode
type AgentMode string

const (
	AgentModePersistent AgentMode = "persistent"
	AgentModeAgentless  AgentMode = "agentless"
)

// IsOnline returns true if the server is online
func (s *Server) IsOnline() bool {
	return s.Status == string(ServerStatusOnline)
}

// IsStale returns true if the server hasn't sent a heartbeat within the threshold
func (s *Server) IsStale(threshold time.Duration) bool {
	if !s.LastSeen.Valid {
		return true
	}
	return time.Since(s.LastSeen.Time) > threshold
}
