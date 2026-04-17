package grpc

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	agentpb "github.com/reignx/reignx/api/proto/gen"
	"github.com/reignx/reignx/pkg/database/models"
	"github.com/reignx/reignx/pkg/observability/metrics"
)

// mockServerRepository is a mock implementation of ServerRepository for testing
type mockServerRepository struct {
	servers          map[string]*models.Server
	upsertError      error
	heartbeatError   error
	lastHeartbeatID  string
	lastHeartbeatTime time.Time
}

func newMockServerRepository() *mockServerRepository {
	return &mockServerRepository{
		servers: make(map[string]*models.Server),
	}
}

func (m *mockServerRepository) Upsert(ctx context.Context, server *models.Server) (*models.Server, error) {
	if m.upsertError != nil {
		return nil, m.upsertError
	}

	// Generate ID if not set
	if server.ID == "" {
		server.ID = "test-server-id-" + server.Hostname
	}

	server.CreatedAt = time.Now()
	server.UpdatedAt = time.Now()

	m.servers[server.Hostname] = server
	return server, nil
}

func (m *mockServerRepository) UpdateHeartbeat(ctx context.Context, serverID string, timestamp time.Time) error {
	if m.heartbeatError != nil {
		return m.heartbeatError
	}

	m.lastHeartbeatID = serverID
	m.lastHeartbeatTime = timestamp

	// Update server if exists
	for _, server := range m.servers {
		if server.ID == serverID {
			server.LastHeartbeat = sql.NullTime{Time: timestamp, Valid: true}
			server.Status = string(models.ServerStatusOnline)
			break
		}
	}

	return nil
}

func (m *mockServerRepository) UpdateStatus(ctx context.Context, serverID string, status string) error {
	return nil
}

func (m *mockServerRepository) GetByID(ctx context.Context, id string) (*models.Server, error) {
	for _, server := range m.servers {
		if server.ID == id {
			return server, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockServerRepository) GetByHostname(ctx context.Context, hostname string) (*models.Server, error) {
	if server, ok := m.servers[hostname]; ok {
		return server, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockServerRepository) MarkStaleServersOffline(ctx context.Context, threshold time.Duration) (int, error) {
	return 0, nil
}

func (m *mockServerRepository) List(ctx context.Context, filters map[string]interface{}) ([]*models.Server, error) {
	servers := make([]*models.Server, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, server)
	}
	return servers, nil
}

func TestAgentService_RegisterAgent(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	m := metrics.NewMetricsWithRegistry("test", nil)
	repo := newMockServerRepository()

	service := NewAgentService(repo, logger, m)

	t.Run("Successful registration", func(t *testing.T) {
		req := &agentpb.AgentInfo{
			AgentId:      "test-agent-1",
			Hostname:     "test-host-1",
			IpAddress:    "192.168.1.100",
			OsType:       "linux",
			OsVersion:    "Ubuntu 22.04",
			Architecture: "amd64",
			AgentVersion: "1.0.0",
			Capabilities: map[string]string{"install_os": "true"},
			Tags:         map[string]string{"env": "test"},
		}

		resp, err := service.RegisterAgent(context.Background(), req)
		if err != nil {
			t.Fatalf("RegisterAgent failed: %v", err)
		}

		if resp.ServerId == "" {
			t.Error("Expected server_id to be set")
		}
		if resp.HeartbeatIntervalSeconds != 30 {
			t.Errorf("Expected heartbeat interval 30, got %d", resp.HeartbeatIntervalSeconds)
		}
		if resp.Message == "" {
			t.Error("Expected registration message")
		}

		// Verify server was created
		server, err := repo.GetByHostname(context.Background(), "test-host-1")
		if err != nil {
			t.Fatalf("Server not found in repository: %v", err)
		}
		if server.IPAddress != "192.168.1.100" {
			t.Errorf("Expected IP 192.168.1.100, got %s", server.IPAddress)
		}
		if server.Status != string(models.ServerStatusOnline) {
			t.Errorf("Expected status online, got %s", server.Status)
		}
	})

	t.Run("Missing hostname", func(t *testing.T) {
		req := &agentpb.AgentInfo{
			AgentId:   "test-agent-2",
			IpAddress: "192.168.1.101",
			OsType:    "linux",
		}

		_, err := service.RegisterAgent(context.Background(), req)
		if err == nil {
			t.Fatal("Expected error for missing hostname")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("Expected gRPC status error")
		}
		if st.Code() != codes.InvalidArgument {
			t.Errorf("Expected InvalidArgument code, got %v", st.Code())
		}
	})

	t.Run("Missing IP address", func(t *testing.T) {
		req := &agentpb.AgentInfo{
			AgentId:  "test-agent-3",
			Hostname: "test-host-3",
			OsType:   "linux",
		}

		_, err := service.RegisterAgent(context.Background(), req)
		if err == nil {
			t.Fatal("Expected error for missing IP address")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("Expected gRPC status error")
		}
		if st.Code() != codes.InvalidArgument {
			t.Errorf("Expected InvalidArgument code, got %v", st.Code())
		}
	})
}

func TestAgentService_Heartbeat(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	m := metrics.NewMetricsWithRegistry("test", nil)
	repo := newMockServerRepository()

	service := NewAgentService(repo, logger, m)

	// First register a server
	regReq := &agentpb.AgentInfo{
		AgentId:   "test-agent-hb",
		Hostname:  "test-host-hb",
		IpAddress: "192.168.1.200",
		OsType:    "linux",
	}
	regResp, _ := service.RegisterAgent(context.Background(), regReq)

	t.Run("Successful heartbeat", func(t *testing.T) {
		req := &agentpb.HeartbeatRequest{
			AgentId:  "test-agent-hb",
			ServerId: regResp.ServerId,
			Status:   "healthy",
			Metrics: &agentpb.MetricsSnapshot{
				CpuPercent:    45.5,
				MemoryPercent: 60.2,
				DiskPercent:   70.0,
			},
		}

		resp, err := service.Heartbeat(context.Background(), req)
		if err != nil {
			t.Fatalf("Heartbeat failed: %v", err)
		}

		if !resp.Acknowledged {
			t.Error("Expected heartbeat to be acknowledged")
		}

		// Verify heartbeat was updated
		if repo.lastHeartbeatID != regResp.ServerId {
			t.Errorf("Expected heartbeat for server %s, got %s", regResp.ServerId, repo.lastHeartbeatID)
		}
	})

	t.Run("Missing server_id", func(t *testing.T) {
		req := &agentpb.HeartbeatRequest{
			AgentId: "test-agent-hb",
			Status:  "healthy",
		}

		_, err := service.Heartbeat(context.Background(), req)
		if err == nil {
			t.Fatal("Expected error for missing server_id")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("Expected gRPC status error")
		}
		if st.Code() != codes.InvalidArgument {
			t.Errorf("Expected InvalidArgument code, got %v", st.Code())
		}
	})
}
