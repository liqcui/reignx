package agent

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"github.com/reignx/reignx/pkg/config"
)

func TestLoadOrGenerateAgentID(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	t.Run("Generate new agent ID", func(t *testing.T) {
		agentID, err := LoadOrGenerateAgentID(tempDir)
		if err != nil {
			t.Fatalf("Failed to generate agent ID: %v", err)
		}

		if agentID == "" {
			t.Error("Expected non-empty agent ID")
		}

		// Verify file was created
		idFile := filepath.Join(tempDir, agentIDFile)
		if _, err := os.Stat(idFile); os.IsNotExist(err) {
			t.Error("Agent ID file was not created")
		}

		// Read file content
		content, err := os.ReadFile(idFile)
		if err != nil {
			t.Fatalf("Failed to read agent ID file: %v", err)
		}

		if string(content) != agentID {
			t.Errorf("File content %s doesn't match agent ID %s", string(content), agentID)
		}

		t.Logf("Generated agent ID: %s", agentID)
	})

	t.Run("Load existing agent ID", func(t *testing.T) {
		// First call generates ID
		agentID1, _ := LoadOrGenerateAgentID(tempDir)

		// Second call should load the same ID
		agentID2, err := LoadOrGenerateAgentID(tempDir)
		if err != nil {
			t.Fatalf("Failed to load agent ID: %v", err)
		}

		if agentID1 != agentID2 {
			t.Errorf("Expected same agent ID, got %s and %s", agentID1, agentID2)
		}

		t.Logf("Loaded agent ID: %s", agentID2)
	})
}

func TestCollectSystemInfo(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{
		Agent: config.AgentConfig{
			CacheDir: "/tmp/test-agent",
		},
	}

	agentID := "test-agent-123"
	agentVersion := "1.0.0-test"

	info, err := CollectSystemInfo(agentID, cfg, agentVersion, logger)
	if err != nil {
		t.Fatalf("Failed to collect system info: %v", err)
	}

	if info.AgentId != agentID {
		t.Errorf("Expected agent ID %s, got %s", agentID, info.AgentId)
	}

	if info.Hostname == "" {
		t.Error("Expected non-empty hostname")
	}

	if info.IpAddress == "" {
		t.Error("Expected non-empty IP address")
	}

	if info.OsType == "" {
		t.Error("Expected non-empty OS type")
	}

	if info.Architecture == "" {
		t.Error("Expected non-empty architecture")
	}

	if info.AgentVersion != agentVersion {
		t.Errorf("Expected agent version %s, got %s", agentVersion, info.AgentVersion)
	}

	if info.Capabilities == nil {
		t.Error("Expected capabilities map")
	}

	if info.Tags == nil {
		t.Error("Expected tags map")
	}

	// Verify capabilities
	expectedCapabilities := []string{"install_os", "patch", "deploy_package", "upgrade_os", "script", "file"}
	for _, cap := range expectedCapabilities {
		if info.Capabilities[cap] != "true" {
			t.Errorf("Expected capability %s to be true", cap)
		}
	}

	t.Logf("System Info - Hostname: %s, IP: %s, OS: %s %s, Arch: %s",
		info.Hostname, info.IpAddress, info.OsType, info.OsVersion, info.Architecture)
}
