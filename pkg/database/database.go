package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/reignx/reignx/pkg/config"
	"github.com/reignx/reignx/pkg/observability/metrics"
)

// Database wraps the database connection and provides management methods
type Database struct {
	*sqlx.DB
	config  *config.DatabaseConfig
	logger  *zap.Logger
	metrics *metrics.Metrics
}

// NewDatabase creates a new database connection
func NewDatabase(cfg config.DatabaseConfig, logger *zap.Logger, metrics *metrics.Metrics) (*Database, error) {
	dsn := cfg.GetDSN()
	logger.Info("Connecting to database with DSN", zap.String("dsn", dsn))

	logger.Info("Connecting to database", zap.String("full_dsn", dsn))

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		logger.Error("Database connection failed", zap.String("dsn", dsn), zap.Error(err))
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}


	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Database),
		zap.Int("max_open_conns", cfg.MaxOpenConns),
		zap.Int("max_idle_conns", cfg.MaxIdleConns),
	)

	database := &Database{
		DB:      db,
		config:  &cfg,
		logger:  logger,
		metrics: metrics,
	}

	// Start metrics collector
	go database.collectMetrics(context.Background())

	return database, nil
}

// Ping checks the database connection
func (d *Database) Ping(ctx context.Context) error {
	return d.PingContext(ctx)
}

// Close closes the database connection
func (d *Database) Close() error {
	d.logger.Info("Closing database connection")
	return d.DB.Close()
}

// collectMetrics periodically collects database metrics
func (d *Database) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := d.Stats()

			if d.metrics != nil {
				d.metrics.RecordGauge("db_open_connections", float64(stats.OpenConnections), nil)
				d.metrics.RecordGauge("db_in_use_connections", float64(stats.InUse), nil)
				d.metrics.RecordGauge("db_idle_connections", float64(stats.Idle), nil)
				d.metrics.RecordCounter("db_wait_count", float64(stats.WaitCount), nil)
				d.metrics.RecordHistogram("db_wait_duration_seconds", stats.WaitDuration.Seconds(), nil)
				d.metrics.RecordCounter("db_max_idle_closed", float64(stats.MaxIdleClosed), nil)
				d.metrics.RecordCounter("db_max_lifetime_closed", float64(stats.MaxLifetimeClosed), nil)
			}
		}
	}
}

// Health returns the health status of the database
func (d *Database) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := d.Ping(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}
