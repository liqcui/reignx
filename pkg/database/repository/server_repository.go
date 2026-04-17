package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/reignx/reignx/pkg/database/models"
	"github.com/reignx/reignx/pkg/observability/metrics"
)

// ServerRepository defines the interface for server data access
type ServerRepository interface {
	// Upsert creates or updates a server record
	Upsert(ctx context.Context, server *models.Server) (*models.Server, error)

	// UpdateHeartbeat updates the last heartbeat timestamp
	UpdateHeartbeat(ctx context.Context, serverID string, timestamp time.Time) error

	// UpdateStatus updates the server status
	UpdateStatus(ctx context.Context, serverID string, status string) error

	// GetByID retrieves a server by ID
	GetByID(ctx context.Context, id string) (*models.Server, error)

	// GetByHostname retrieves a server by hostname
	GetByHostname(ctx context.Context, hostname string) (*models.Server, error)

	// MarkStaleServersOffline marks servers as offline if they haven't sent heartbeats
	MarkStaleServersOffline(ctx context.Context, threshold time.Duration) (int, error)

	// List retrieves all servers with optional filters
	List(ctx context.Context, filters map[string]interface{}) ([]*models.Server, error)
}

// serverRepository implements ServerRepository
type serverRepository struct {
	db      *sqlx.DB
	logger  *zap.Logger
	metrics *metrics.Metrics
}

// NewServerRepository creates a new ServerRepository
func NewServerRepository(db *sqlx.DB, logger *zap.Logger, metrics *metrics.Metrics) ServerRepository {
	return &serverRepository{
		db:      db,
		logger:  logger,
		metrics: metrics,
	}
}

// Upsert creates or updates a server record
func (r *serverRepository) Upsert(ctx context.Context, server *models.Server) (*models.Server, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if r.metrics != nil {
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"operation": "upsert_server",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"operation": "upsert_server",
			})
		}
	}()

	query := `
		INSERT INTO nodes (
			hostname, ip_address, mac_address, os_type, os_version,
			architecture, mode, agent_version, status, last_seen,
			tags, metadata
		) VALUES (
			:hostname, :ip_address, :mac_address, :os_type, :os_version,
			:architecture, :mode, :agent_version, :status, :last_seen,
			:tags, :metadata
		)
		ON CONFLICT (ip_address)
		DO UPDATE SET
			ip_address = EXCLUDED.ip_address,
			mac_address = EXCLUDED.mac_address,
			os_type = EXCLUDED.os_type,
			os_version = EXCLUDED.os_version,
			architecture = EXCLUDED.architecture,
			mode = EXCLUDED.mode,
			agent_version = EXCLUDED.agent_version,
			status = 'online',
			last_seen = EXCLUDED.last_seen,
			tags = EXCLUDED.tags,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING *
	`

	rows, err := r.db.NamedQueryContext(ctx, query, server)
	if err != nil {
		r.logger.Error("Failed to upsert server",
			zap.String("hostname", server.Hostname),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to upsert server: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("no rows returned from upsert")
	}

	var result models.Server
	if err := rows.StructScan(&result); err != nil {
		return nil, fmt.Errorf("failed to scan result: %w", err)
	}

	r.logger.Info("Server upserted successfully",
		zap.String("server_id", result.ID),
		zap.String("hostname", result.Hostname),
		zap.String("status", result.Status),
	)

	return &result, nil
}

// UpdateHeartbeat updates the last heartbeat timestamp
func (r *serverRepository) UpdateHeartbeat(ctx context.Context, serverID string, timestamp time.Time) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if r.metrics != nil {
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"operation": "update_heartbeat",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"operation": "update_heartbeat",
			})
		}
	}()

	query := `
		UPDATE nodes
		SET last_seen = $1, status = 'online', updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, timestamp, serverID)
	if err != nil {
		r.logger.Error("Failed to update heartbeat",
			zap.String("server_id", serverID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("server not found: %s", serverID)
	}

	return nil
}

// UpdateStatus updates the server status
func (r *serverRepository) UpdateStatus(ctx context.Context, serverID string, status string) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if r.metrics != nil {
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"operation": "update_status",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"operation": "update_status",
			})
		}
	}()

	query := `
		UPDATE nodes
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, status, serverID)
	if err != nil {
		r.logger.Error("Failed to update status",
			zap.String("server_id", serverID),
			zap.String("status", status),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("server not found: %s", serverID)
	}

	r.logger.Info("Server status updated",
		zap.String("server_id", serverID),
		zap.String("status", status),
	)

	return nil
}

// GetByID retrieves a server by ID
func (r *serverRepository) GetByID(ctx context.Context, id string) (*models.Server, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if r.metrics != nil {
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"operation": "get_server_by_id",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"operation": "get_server_by_id",
			})
		}
	}()

	var server models.Server
	query := `SELECT * FROM nodes WHERE id = $1`

	err := r.db.GetContext(ctx, &server, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("server not found: %s", id)
	}
	if err != nil {
		r.logger.Error("Failed to get server by ID",
			zap.String("server_id", id),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	return &server, nil
}

// GetByHostname retrieves a server by hostname
func (r *serverRepository) GetByHostname(ctx context.Context, hostname string) (*models.Server, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if r.metrics != nil {
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"operation": "get_server_by_hostname",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"operation": "get_server_by_hostname",
			})
		}
	}()

	var server models.Server
	query := `SELECT * FROM nodes WHERE hostname = $1`

	err := r.db.GetContext(ctx, &server, query, hostname)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("server not found: %s", hostname)
	}
	if err != nil {
		r.logger.Error("Failed to get server by hostname",
			zap.String("hostname", hostname),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	return &server, nil
}

// MarkStaleServersOffline marks servers as offline if they haven't sent heartbeats
func (r *serverRepository) MarkStaleServersOffline(ctx context.Context, threshold time.Duration) (int, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if r.metrics != nil {
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"operation": "mark_stale_servers_offline",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"operation": "mark_stale_servers_offline",
			})
		}
	}()

	query := `
		UPDATE nodes
		SET status = 'offline', updated_at = NOW()
		WHERE status = 'online'
			AND last_seen < NOW() - $1 * INTERVAL '1 second'
		RETURNING id
	`

	rows, err := r.db.QueryContext(ctx, query, int(threshold.Seconds()))
	if err != nil {
		r.logger.Error("Failed to mark stale servers offline",
			zap.Duration("threshold", threshold),
			zap.Error(err),
		)
		return 0, fmt.Errorf("failed to mark stale servers offline: %w", err)
	}
	defer rows.Close()

	count := 0
	var serverIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			r.logger.Error("Failed to scan server ID", zap.Error(err))
			continue
		}
		serverIDs = append(serverIDs, id)
		count++
	}

	if count > 0 {
		r.logger.Info("Marked stale servers as offline",
			zap.Int("count", count),
			zap.Strings("server_ids", serverIDs),
			zap.Duration("threshold", threshold),
		)
	}

	return count, nil
}

// List retrieves all servers with optional filters
func (r *serverRepository) List(ctx context.Context, filters map[string]interface{}) ([]*models.Server, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if r.metrics != nil {
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"operation": "list_servers",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"operation": "list_servers",
			})
		}
	}()

	query := `SELECT * FROM nodes ORDER BY created_at DESC`
	var servers []*models.Server

	err := r.db.SelectContext(ctx, &servers, query)
	if err != nil {
		r.logger.Error("Failed to list servers", zap.Error(err))
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	return servers, nil
}
