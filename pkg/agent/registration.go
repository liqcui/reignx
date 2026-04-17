package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/google/uuid"
	"go.uber.org/zap"

	agentpb "github.com/reignx/reignx/api/proto/gen"
	"github.com/reignx/reignx/pkg/config"
)

const (
	agentIDFile = "agent_id"
)

// LoadOrGenerateAgentID loads the agent ID from cache or generates a new one
func LoadOrGenerateAgentID(cacheDir string) (string, error) {
	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	agentIDPath := filepath.Join(cacheDir, agentIDFile)

	// Try to load existing agent ID
	if data, err := os.ReadFile(agentIDPath); err == nil {
		agentID := string(data)
		if agentID != "" {
			return agentID, nil
		}
	}

	// Generate new agent ID
	agentID := uuid.New().String()

	// Save to file
	if err := os.WriteFile(agentIDPath, []byte(agentID), 0644); err != nil {
		return "", fmt.Errorf("failed to save agent ID: %w", err)
	}

	return agentID, nil
}

// CollectSystemInfo gathers system information for agent registration
func CollectSystemInfo(agentID string, cfg *config.Config, agentVersion string, logger *zap.Logger) (*agentpb.AgentInfo, error) {
	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	// Get primary IP address
	ipAddress, err := GetPrimaryIP()
	if err != nil {
		logger.Warn("Failed to get primary IP, using localhost",
			zap.Error(err),
		)
		ipAddress = "127.0.0.1"
	}

	// Get OS information
	osType, osVersion, err := GetOSInfo()
	if err != nil {
		logger.Warn("Failed to get OS info",
			zap.Error(err),
		)
		osType = runtime.GOOS
		osVersion = "unknown"
	}

	// Get architecture
	architecture := runtime.GOARCH

	// Build capabilities map (features this agent supports)
	capabilities := make(map[string]string)
	capabilities["install_os"] = "true"
	capabilities["patch"] = "true"
	capabilities["deploy_package"] = "true"
	capabilities["upgrade_os"] = "true"
	capabilities["script"] = "true"
	capabilities["file"] = "true"

	// Build tags from config
	tags := make(map[string]string)
	// Tags can be added from config or environment in the future

	return &agentpb.AgentInfo{
		AgentId:      agentID,
		Hostname:     hostname,
		IpAddress:    ipAddress,
		OsType:       osType,
		OsVersion:    osVersion,
		Architecture: architecture,
		AgentVersion: agentVersion,
		Capabilities: capabilities,
		Tags:         tags,
	}, nil
}

// Register performs the registration flow with the API server
func Register(ctx context.Context, client *Client, agentID string, cfg *config.Config, agentVersion string, logger *zap.Logger) (string, error) {
	// Collect system info
	sysInfo, err := CollectSystemInfo(agentID, cfg, agentVersion, logger)
	if err != nil {
		return "", fmt.Errorf("failed to collect system info: %w", err)
	}

	logger.Info("Collected system information",
		zap.String("agent_id", agentID),
		zap.String("hostname", sysInfo.Hostname),
		zap.String("ip_address", sysInfo.IpAddress),
		zap.String("os_type", sysInfo.OsType),
		zap.String("os_version", sysInfo.OsVersion),
		zap.String("architecture", sysInfo.Architecture),
		zap.String("agent_version", sysInfo.AgentVersion),
	)

	// Register with server
	resp, err := client.RegisterAgent(ctx, sysInfo)
	if err != nil {
		return "", fmt.Errorf("failed to register with server: %w", err)
	}

	logger.Info("Registration successful",
		zap.String("server_id", resp.ServerId),
		zap.String("message", resp.Message),
		zap.Int32("heartbeat_interval", resp.HeartbeatIntervalSeconds),
	)

	return resp.ServerId, nil
}
