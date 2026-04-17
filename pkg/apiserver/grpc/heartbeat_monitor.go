package grpc

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/reignx/reignx/pkg/config"
	"github.com/reignx/reignx/pkg/database/repository"
	"github.com/reignx/reignx/pkg/observability/metrics"
)

// HeartbeatMonitor monitors agent heartbeats and marks stale servers offline
type HeartbeatMonitor struct {
	repository repository.ServerRepository
	logger     *zap.Logger
	metrics    *metrics.Metrics
	config     *config.Config
	threshold  time.Duration
	interval   time.Duration
}

// NewHeartbeatMonitor creates a new heartbeat monitor
func NewHeartbeatMonitor(repo repository.ServerRepository, logger *zap.Logger, m *metrics.Metrics, cfg *config.Config) *HeartbeatMonitor {
	// Threshold is 3x the heartbeat interval (90 seconds by default)
	threshold := cfg.Agent.HeartbeatInterval * 3

	return &HeartbeatMonitor{
		repository: repo,
		logger:     logger,
		metrics:    m,
		config:     cfg,
		threshold:  threshold,
		interval:   30 * time.Second, // Check every 30 seconds
	}
}

// Run starts the heartbeat monitoring loop
func (m *HeartbeatMonitor) Run(ctx context.Context) {
	m.logger.Info("Starting heartbeat monitor",
		zap.Duration("threshold", m.threshold),
		zap.Duration("check_interval", m.interval),
	)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Run initial check immediately
	m.checkStaleServers(ctx)

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Heartbeat monitor stopping")
			return
		case <-ticker.C:
			m.checkStaleServers(ctx)
		}
	}
}

// checkStaleServers marks servers as offline if they haven't sent heartbeats
func (m *HeartbeatMonitor) checkStaleServers(ctx context.Context) {
	start := time.Now()

	count, err := m.repository.MarkStaleServersOffline(ctx, m.threshold)
	if err != nil {
		m.logger.Error("Failed to mark stale servers offline",
			zap.Error(err),
			zap.Duration("threshold", m.threshold),
		)
		if m.metrics != nil {
			m.metrics.RecordCounter("heartbeat_monitor_errors_total", 1, nil)
		}
		return
	}

	duration := time.Since(start)

	if count > 0 {
		m.logger.Info("Marked stale servers as offline",
			zap.Int("count", count),
			zap.Duration("threshold", m.threshold),
			zap.Duration("duration", duration),
		)
	}

	// Update metrics
	if m.metrics != nil {
		m.metrics.RecordHistogram("heartbeat_monitor_check_duration_seconds", duration.Seconds(), nil)
		m.metrics.RecordCounter("heartbeat_monitor_checks_total", 1, nil)
		if count > 0 {
			m.metrics.RecordCounter("heartbeat_monitor_servers_marked_offline", float64(count), nil)
		}
	}

	// Get server counts by status and update gauge
	// This would require a new repository method, but for now we'll just update the metric
	// based on what we know
	m.updateServerStatusMetrics(ctx)
}

// updateServerStatusMetrics updates metrics for server status counts
func (m *HeartbeatMonitor) updateServerStatusMetrics(ctx context.Context) {
	// List all servers to count by status
	servers, err := m.repository.List(ctx, nil)
	if err != nil {
		m.logger.Error("Failed to list servers for metrics",
			zap.Error(err),
		)
		return
	}

	// Count servers by status
	statusCounts := make(map[string]int)
	for _, server := range servers {
		statusCounts[server.Status]++
	}

	// Update metrics
	if m.metrics != nil {
		for status, count := range statusCounts {
			m.metrics.RecordGauge("nodes_total", float64(count), map[string]string{
				"status": status,
			})
		}
	}
}
