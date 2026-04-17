package test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"go.uber.org/zap"

	agentpb "github.com/reignx/reignx"
	agentpkg "github.com/reignx/reignx/pkg/agent"
	grpcpkg "github.com/reignx/reignx/pkg/apiserver/grpc"
	"github.com/reignx/reignx/pkg/config"
	"github.com/reignx/reignx/pkg/database/models"
	"github.com/reignx/reignx/pkg/observability/metrics"
)

// TestAgentAutoDiscoveryFlow tests the complete agent auto-discovery flow
// without requiring a real database or network
func TestAgentAutoDiscoveryFlow(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Create mock repository
	repo := newMockRepository()

	// Create gRPC service
	m := metrics.NewMetricsWithRegistry("test", nil)
	agentSvc := grpcpkg.NewAgentService(repo, logger, m)

	t.Run("Complete flow: Registration -> Heartbeat -> Stale detection", func(t *testing.T) {
		// Step 1: Register agent
		regReq := &agentpb.AgentInfo{
			AgentId:      "integration-test-agent",
			Hostname:     "integration-test-host",
			IpAddress:    "10.0.0.1",
			OsType:       "linux",
			OsVersion:    "Ubuntu 22.04",
			Architecture: "amd64",
			AgentVersion: "1.0.0",
			Capabilities: map[string]string{
				"install_os": "true",
				"patch":      "true",
			},
			Tags: map[string]string{
				"env": "test",
			},
		}

		regResp, err := agentSvc.RegisterAgent(context.Background(), regReq)
		if err != nil {
			t.Fatalf("Registration failed: %v", err)
		}

		if regResp.ServerId == "" {
			t.Fatal("Expected server ID to be set")
		}

		t.Logf("✓ Agent registered successfully with server_id: %s", regResp.ServerId)

		// Verify server was created in repository
		server, err := repo.GetByHostname(context.Background(), "integration-test-host")
		if err != nil {
			t.Fatalf("Server not found after registration: %v", err)
		}

		if server.Status != string(models.ServerStatusOnline) {
			t.Errorf("Expected server status online, got %s", server.Status)
		}

		t.Logf("✓ Server status is online")

		// Step 2: Send heartbeats
		for i := 0; i < 3; i++ {
			hbReq := &agentpb.HeartbeatRequest{
				AgentId:  "integration-test-agent",
				ServerId: regResp.ServerId,
				Status:   "healthy",
				Metrics: &agentpb.MetricsSnapshot{
					CpuPercent:    float64(10 + i*5),
					MemoryPercent: float64(50 + i*2),
					DiskPercent:   70.0,
					UptimeSeconds: int64(3600 + i*60),
				},
			}

			hbResp, err := agentSvc.Heartbeat(context.Background(), hbReq)
			if err != nil {
				t.Fatalf("Heartbeat %d failed: %v", i+1, err)
			}

			if !hbResp.Acknowledged {
				t.Error("Heartbeat not acknowledged")
			}

			time.Sleep(100 * time.Millisecond)
		}

		t.Logf("✓ Successfully sent 3 heartbeats")

		// Step 3: Verify heartbeat was updated
		serverUpdated, _ := repo.GetByID(context.Background(), regResp.ServerId)
		if !serverUpdated.LastHeartbeat.Valid {
			t.Error("Expected last_heartbeat to be set")
		}

		heartbeatAge := time.Since(serverUpdated.LastHeartbeat.Time)
		if heartbeatAge > 5*time.Second {
			t.Errorf("Last heartbeat is too old: %v", heartbeatAge)
		}

		t.Logf("✓ Last heartbeat is recent (age: %v)", heartbeatAge)

		// Step 4: Test stale server detection
		// Simulate stale server by setting old heartbeat
		oldTime := time.Now().Add(-2 * time.Minute)
		repo.UpdateHeartbeat(context.Background(), regResp.ServerId, oldTime)

		// Mark stale servers offline
		count, err := repo.MarkStaleServersOffline(context.Background(), 90*time.Second)
		if err != nil {
			t.Fatalf("Failed to mark stale servers offline: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 server to be marked offline, got %d", count)
		}

		// Verify server is now offline
		serverFinal, _ := repo.GetByID(context.Background(), regResp.ServerId)
		if serverFinal.Status != string(models.ServerStatusOffline) {
			t.Errorf("Expected server to be offline, got %s", serverFinal.Status)
		}

		t.Logf("✓ Stale server correctly marked as offline")
	})
}

// TestAgentSystemInfoCollection tests actual system info collection
func TestAgentSystemInfoCollection(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	cfg := &config.Config{
		Agent: config.AgentConfig{
			CacheDir: t.TempDir(),
		},
	}

	// Generate agent ID
	agentID, err := agentpkg.LoadOrGenerateAgentID(cfg.Agent.CacheDir)
	if err != nil {
		t.Fatalf("Failed to generate agent ID: %v", err)
	}

	t.Logf("Agent ID: %s", agentID)

	// Collect system info
	sysInfo, err := agentpkg.CollectSystemInfo(agentID, cfg, "1.0.0-test", logger)
	if err != nil {
		t.Fatalf("Failed to collect system info: %v", err)
	}

	// Validate collected info
	if sysInfo.Hostname == "" {
		t.Error("Hostname not collected")
	}
	if sysInfo.IpAddress == "" {
		t.Error("IP address not collected")
	}
	if sysInfo.OsType == "" {
		t.Error("OS type not collected")
	}

	t.Logf("✓ System Info: hostname=%s, ip=%s, os=%s %s, arch=%s",
		sysInfo.Hostname, sysInfo.IpAddress, sysInfo.OsType, sysInfo.OsVersion, sysInfo.Architecture)

	// Collect metrics
	metrics, err := agentpkg.GetMetricsSnapshot()
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	if metrics.CPUPercent < 0 || metrics.CPUPercent > 100 {
		t.Errorf("Invalid CPU percent: %f", metrics.CPUPercent)
	}

	t.Logf("✓ Metrics: CPU=%.2f%%, Memory=%.2f%%, Disk=%.2f%%",
		metrics.CPUPercent, metrics.MemoryPercent, metrics.DiskPercent)
}

// mockRepository is a simple in-memory repository for testing
type mockRepository struct {
	servers map[string]*models.Server
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		servers: make(map[string]*models.Server),
	}
}

func (m *mockRepository) Upsert(ctx context.Context, server *models.Server) (*models.Server, error) {
	if server.ID == "" {
		server.ID = "mock-id-" + server.Hostname
	}
	server.CreatedAt = time.Now()
	server.UpdatedAt = time.Now()
	m.servers[server.ID] = server
	return server, nil
}

func (m *mockRepository) UpdateHeartbeat(ctx context.Context, serverID string, timestamp time.Time) error {
	if server, ok := m.servers[serverID]; ok {
		server.LastHeartbeat = sql.NullTime{Time: timestamp, Valid: true}
		server.Status = string(models.ServerStatusOnline)
	}
	return nil
}

func (m *mockRepository) UpdateStatus(ctx context.Context, serverID string, status string) error {
	if server, ok := m.servers[serverID]; ok {
		server.Status = status
	}
	return nil
}

func (m *mockRepository) GetByID(ctx context.Context, id string) (*models.Server, error) {
	if server, ok := m.servers[id]; ok {
		return server, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockRepository) GetByHostname(ctx context.Context, hostname string) (*models.Server, error) {
	for _, server := range m.servers {
		if server.Hostname == hostname {
			return server, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockRepository) MarkStaleServersOffline(ctx context.Context, threshold time.Duration) (int, error) {
	count := 0
	now := time.Now()
	for _, server := range m.servers {
		if server.Status == string(models.ServerStatusOnline) &&
			server.LastHeartbeat.Valid &&
			now.Sub(server.LastHeartbeat.Time) > threshold {
			server.Status = string(models.ServerStatusOffline)
			count++
		}
	}
	return count, nil
}

func (m *mockRepository) List(ctx context.Context, filters map[string]interface{}) ([]*models.Server, error) {
	servers := make([]*models.Server, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, server)
	}
	return servers, nil
}
