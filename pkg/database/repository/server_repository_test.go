package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/reignx/reignx/pkg/database/models"
	"github.com/reignx/reignx/pkg/observability/metrics"
)

// TestServerRepository tests server repository operations
func TestServerRepository(t *testing.T) {
	// Skip if no database available
	// Use TEST_DATABASE_PASSWORD env var or skip test
	password := os.Getenv("TEST_DATABASE_PASSWORD")
	if password == "" {
		password = "test_password"  // Default for local testing only
	}
	dsn := fmt.Sprintf("host=localhost port=5432 user=bm_user password=%s dbname=bm_solution_test sslmode=disable", password)
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping database tests: %v", err)
		return
	}
	defer db.Close()

	// Clean up test data
	defer func() {
		db.Exec("DELETE FROM servers WHERE hostname LIKE 'test-%'")
	}()

	logger, _ := zap.NewDevelopment()
	m := metrics.NewMetrics("test")

	repo := NewServerRepository(db, logger, m)

	t.Run("Upsert new server", func(t *testing.T) {
		server := &models.Server{
			Hostname:  "test-server-1",
			IPAddress: "192.168.1.100",
			OSType:    "linux",
			OSVersion: sql.NullString{String: "Ubuntu 22.04", Valid: true},
			Architecture: sql.NullString{String: "amd64", Valid: true},
			AgentMode: string(models.AgentModePersistent),
			Status:    string(models.ServerStatusOnline),
			LastHeartbeat: sql.NullTime{Time: time.Now(), Valid: true},
			Tags:     make(models.JSONB),
			Metadata: make(models.JSONB),
		}

		result, err := repo.Upsert(context.Background(), server)
		if err != nil {
			t.Fatalf("Failed to upsert server: %v", err)
		}

		if result.ID == "" {
			t.Error("Expected server ID to be set")
		}
		if result.Hostname != "test-server-1" {
			t.Errorf("Expected hostname test-server-1, got %s", result.Hostname)
		}
		if result.Status != string(models.ServerStatusOnline) {
			t.Errorf("Expected status online, got %s", result.Status)
		}
	})

	t.Run("Upsert existing server", func(t *testing.T) {
		// First insert
		server := &models.Server{
			Hostname:  "test-server-2",
			IPAddress: "192.168.1.101",
			OSType:    "linux",
			AgentMode: string(models.AgentModePersistent),
			Status:    string(models.ServerStatusOnline),
			LastHeartbeat: sql.NullTime{Time: time.Now(), Valid: true},
			Tags:     make(models.JSONB),
			Metadata: make(models.JSONB),
		}

		result1, err := repo.Upsert(context.Background(), server)
		if err != nil {
			t.Fatalf("Failed to upsert server first time: %v", err)
		}

		// Update with new IP
		server.IPAddress = "192.168.1.102"
		result2, err := repo.Upsert(context.Background(), server)
		if err != nil {
			t.Fatalf("Failed to upsert server second time: %v", err)
		}

		if result1.ID != result2.ID {
			t.Error("Expected same server ID after upsert")
		}
		if result2.IPAddress != "192.168.1.102" {
			t.Errorf("Expected IP to be updated to 192.168.1.102, got %s", result2.IPAddress)
		}
	})

	t.Run("UpdateHeartbeat", func(t *testing.T) {
		// Create a server
		server := &models.Server{
			Hostname:  "test-server-3",
			IPAddress: "192.168.1.103",
			OSType:    "linux",
			AgentMode: string(models.AgentModePersistent),
			Status:    string(models.ServerStatusOnline),
			LastHeartbeat: sql.NullTime{Time: time.Now().Add(-5 * time.Minute), Valid: true},
			Tags:     make(models.JSONB),
			Metadata: make(models.JSONB),
		}

		result, err := repo.Upsert(context.Background(), server)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		// Update heartbeat
		newTime := time.Now()
		err = repo.UpdateHeartbeat(context.Background(), result.ID, newTime)
		if err != nil {
			t.Fatalf("Failed to update heartbeat: %v", err)
		}

		// Verify update
		updated, err := repo.GetByID(context.Background(), result.ID)
		if err != nil {
			t.Fatalf("Failed to get server: %v", err)
		}

		if !updated.LastHeartbeat.Valid {
			t.Error("Expected last_heartbeat to be valid")
		}

		timeDiff := newTime.Sub(updated.LastHeartbeat.Time)
		if timeDiff > time.Second || timeDiff < -time.Second {
			t.Errorf("Heartbeat time not updated correctly, diff: %v", timeDiff)
		}
	})

	t.Run("MarkStaleServersOffline", func(t *testing.T) {
		// Create two servers - one stale, one fresh
		staleServer := &models.Server{
			Hostname:  "test-server-stale",
			IPAddress: "192.168.1.104",
			OSType:    "linux",
			AgentMode: string(models.AgentModePersistent),
			Status:    string(models.ServerStatusOnline),
			LastHeartbeat: sql.NullTime{Time: time.Now().Add(-2 * time.Minute), Valid: true},
			Tags:     make(models.JSONB),
			Metadata: make(models.JSONB),
		}

		freshServer := &models.Server{
			Hostname:  "test-server-fresh",
			IPAddress: "192.168.1.105",
			OSType:    "linux",
			AgentMode: string(models.AgentModePersistent),
			Status:    string(models.ServerStatusOnline),
			LastHeartbeat: sql.NullTime{Time: time.Now(), Valid: true},
			Tags:     make(models.JSONB),
			Metadata: make(models.JSONB),
		}

		staleResult, _ := repo.Upsert(context.Background(), staleServer)
		freshResult, _ := repo.Upsert(context.Background(), freshServer)

		// Mark servers offline that haven't sent heartbeat in 90 seconds
		count, err := repo.MarkStaleServersOffline(context.Background(), 90*time.Second)
		if err != nil {
			t.Fatalf("Failed to mark stale servers offline: %v", err)
		}

		if count == 0 {
			t.Error("Expected at least one server to be marked offline")
		}

		// Verify stale server is offline
		staleUpdated, _ := repo.GetByID(context.Background(), staleResult.ID)
		if staleUpdated.Status != string(models.ServerStatusOffline) {
			t.Errorf("Expected stale server to be offline, got %s", staleUpdated.Status)
		}

		// Verify fresh server is still online
		freshUpdated, _ := repo.GetByID(context.Background(), freshResult.ID)
		if freshUpdated.Status != string(models.ServerStatusOnline) {
			t.Errorf("Expected fresh server to be online, got %s", freshUpdated.Status)
		}
	})

	t.Run("GetByHostname", func(t *testing.T) {
		server := &models.Server{
			Hostname:  "test-server-lookup",
			IPAddress: "192.168.1.106",
			OSType:    "linux",
			AgentMode: string(models.AgentModePersistent),
			Status:    string(models.ServerStatusOnline),
			LastHeartbeat: sql.NullTime{Time: time.Now(), Valid: true},
			Tags:     make(models.JSONB),
			Metadata: make(models.JSONB),
		}

		created, err := repo.Upsert(context.Background(), server)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		// Look up by hostname
		found, err := repo.GetByHostname(context.Background(), "test-server-lookup")
		if err != nil {
			t.Fatalf("Failed to get server by hostname: %v", err)
		}

		if found.ID != created.ID {
			t.Error("Expected to find the same server")
		}
		if found.Hostname != "test-server-lookup" {
			t.Errorf("Expected hostname test-server-lookup, got %s", found.Hostname)
		}
	})
}
