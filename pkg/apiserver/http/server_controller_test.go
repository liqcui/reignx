package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/reignx/reignx/pkg/database/models"
	"github.com/reignx/reignx/pkg/observability/metrics"
)

// mockServerRepository is a mock implementation for testing
type mockServerRepository struct {
	servers map[string]*models.Server
}

func newMockServerRepository() *mockServerRepository {
	return &mockServerRepository{
		servers: make(map[string]*models.Server),
	}
}

func (m *mockServerRepository) Upsert(ctx context.Context, server *models.Server) (*models.Server, error) {
	if server.ID == "" {
		server.ID = "test-id-" + server.Hostname
	}
	m.servers[server.ID] = server
	return server, nil
}

func (m *mockServerRepository) UpdateHeartbeat(ctx context.Context, serverID string, timestamp time.Time) error {
	return nil
}

func (m *mockServerRepository) UpdateStatus(ctx context.Context, serverID string, status string) error {
	return nil
}

func (m *mockServerRepository) GetByID(ctx context.Context, id string) (*models.Server, error) {
	if server, ok := m.servers[id]; ok {
		return server, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockServerRepository) GetByHostname(ctx context.Context, hostname string) (*models.Server, error) {
	for _, server := range m.servers {
		if server.Hostname == hostname {
			return server, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockServerRepository) MarkStaleServersOffline(ctx context.Context, threshold time.Duration) (int, error) {
	return 0, nil
}

func (m *mockServerRepository) List(ctx context.Context, filters map[string]interface{}) ([]*models.Server, error) {
	result := make([]*models.Server, 0, len(m.servers))
	for _, server := range m.servers {
		result = append(result, server)
	}
	return result, nil
}

func TestServerController_ListServers(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	m := metrics.NewMetricsWithRegistry("test", nil)
	repo := newMockServerRepository()

	// Add test servers
	server1 := &models.Server{
		ID:        "server-1",
		Hostname:  "test-host-1",
		IPAddress: "192.168.1.100",
		OSType:    "linux",
		Status:    string(models.ServerStatusOnline),
		AgentMode: string(models.AgentModePersistent),
		LastHeartbeat: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
		Tags:      make(models.JSONB),
		Metadata:  make(models.JSONB),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.servers[server1.ID] = server1

	controller := NewServerController(repo, logger, m)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/servers", controller.ListServers)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/servers", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response ListServersResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Total != 1 {
		t.Errorf("Expected total 1, got %d", response.Total)
	}

	if len(response.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(response.Servers))
	}

	if response.Servers[0].Hostname != "test-host-1" {
		t.Errorf("Expected hostname test-host-1, got %s", response.Servers[0].Hostname)
	}
}

func TestServerController_GetServer(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	m := metrics.NewMetricsWithRegistry("test", nil)
	repo := newMockServerRepository()

	// Add test server
	server1 := &models.Server{
		ID:        "server-123",
		Hostname:  "test-host-1",
		IPAddress: "192.168.1.100",
		OSType:    "linux",
		OSVersion: sql.NullString{String: "Ubuntu 22.04", Valid: true},
		Status:    string(models.ServerStatusOnline),
		AgentMode: string(models.AgentModePersistent),
		LastHeartbeat: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
		Tags:      make(models.JSONB),
		Metadata:  make(models.JSONB),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.servers[server1.ID] = server1

	controller := NewServerController(repo, logger, m)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/servers/:id", controller.GetServer)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/servers/server-123", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response ServerResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.ID != "server-123" {
		t.Errorf("Expected ID server-123, got %s", response.ID)
	}

	if response.Hostname != "test-host-1" {
		t.Errorf("Expected hostname test-host-1, got %s", response.Hostname)
	}

	if response.IPAddress != "192.168.1.100" {
		t.Errorf("Expected IP 192.168.1.100, got %s", response.IPAddress)
	}
}

func TestServerController_GetStats(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	m := metrics.NewMetricsWithRegistry("test", nil)
	repo := newMockServerRepository()

	// Add test servers with different statuses
	server1 := &models.Server{
		ID:        "server-1",
		Hostname:  "test-host-1",
		IPAddress: "192.168.1.100",
		OSType:    "linux",
		Status:    string(models.ServerStatusOnline),
		AgentMode: string(models.AgentModePersistent),
		Tags:      make(models.JSONB),
		Metadata:  make(models.JSONB),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	server2 := &models.Server{
		ID:        "server-2",
		Hostname:  "test-host-2",
		IPAddress: "192.168.1.101",
		OSType:    "windows",
		Status:    string(models.ServerStatusOffline),
		AgentMode: string(models.AgentModePersistent),
		Tags:      make(models.JSONB),
		Metadata:  make(models.JSONB),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	repo.servers[server1.ID] = server1
	repo.servers[server2.ID] = server2

	controller := NewServerController(repo, logger, m)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/stats", controller.GetStats)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/stats", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response StatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Total != 2 {
		t.Errorf("Expected total 2, got %d", response.Total)
	}

	if response.Online != 1 {
		t.Errorf("Expected 1 online server, got %d", response.Online)
	}

	if response.Offline != 1 {
		t.Errorf("Expected 1 offline server, got %d", response.Offline)
	}

	if response.ByOSType["linux"] != 1 {
		t.Errorf("Expected 1 linux server, got %d", response.ByOSType["linux"])
	}

	if response.ByOSType["windows"] != 1 {
		t.Errorf("Expected 1 windows server, got %d", response.ByOSType["windows"])
	}
}

func TestServerController_GetHealth(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	m := metrics.NewMetricsWithRegistry("test", nil)
	repo := newMockServerRepository()

	controller := NewServerController(repo, logger, m)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/health", controller.GetHealth)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status healthy, got %v", response["status"])
	}
}
