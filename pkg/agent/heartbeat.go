package agent

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	agentpb "github.com/reignx/reignx/api/proto/gen"
	"github.com/reignx/reignx/pkg/config"
	"github.com/reignx/reignx/pkg/observability/metrics"
)

// HeartbeatManager manages periodic heartbeat sending
type HeartbeatManager struct {
	client   *Client
	config   *config.Config
	logger   *zap.Logger
	metrics  *metrics.Metrics
	serverID string
	agentID  string
	interval time.Duration
}

// NewHeartbeatManager creates a new heartbeat manager
func NewHeartbeatManager(client *Client, cfg *config.Config, logger *zap.Logger, m *metrics.Metrics, serverID, agentID string) *HeartbeatManager {
	return &HeartbeatManager{
		client:   client,
		config:   cfg,
		logger:   logger,
		metrics:  m,
		serverID: serverID,
		agentID:  agentID,
		interval: cfg.Agent.HeartbeatInterval,
	}
}

// Start begins the heartbeat loop
func (h *HeartbeatManager) Start(ctx context.Context) {
	h.logger.Info("Starting heartbeat manager",
		zap.String("server_id", h.serverID),
		zap.Duration("interval", h.interval),
	)

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	// Send initial heartbeat immediately
	h.sendHeartbeat(ctx, "healthy")

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("Heartbeat manager stopping")
			return
		case <-ticker.C:
			h.sendHeartbeat(ctx, "healthy")
		}
	}
}

// sendHeartbeat sends a heartbeat to the API server
func (h *HeartbeatManager) sendHeartbeat(ctx context.Context, status string) {
	start := time.Now()

	// Collect current metrics
	metricsSnapshot, err := GetMetricsSnapshot()
	if err != nil {
		h.logger.Warn("Failed to collect metrics snapshot",
			zap.Error(err),
		)
		// Continue with empty metrics
		metricsSnapshot = &MetricsSnapshot{}
	}

	// Create heartbeat request
	req := &agentpb.HeartbeatRequest{
		AgentId:   h.agentID,
		ServerId:  h.serverID,
		Status:    status,
		Timestamp: timestamppb.Now(),
		Metrics:   metricsSnapshot.ToProto(),
	}

	// Send heartbeat with timeout
	hbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := h.client.Heartbeat(hbCtx, req)
	if err != nil {
		h.logger.Error("Heartbeat failed",
			zap.String("server_id", h.serverID),
			zap.Error(err),
		)
		if h.metrics != nil {
			h.metrics.RecordCounter("agent_heartbeat_attempts_total", 1, map[string]string{
				"status": "error",
			})
		}
		return
	}

	duration := time.Since(start)

	// Record successful heartbeat
	if h.metrics != nil {
		h.metrics.RecordCounter("agent_heartbeat_attempts_total", 1, map[string]string{
			"status": "success",
		})
		h.metrics.RecordHistogram("agent_heartbeat_duration_seconds", duration.Seconds(), nil)
	}

	h.logger.Debug("Heartbeat sent successfully",
		zap.String("server_id", h.serverID),
		zap.Bool("acknowledged", resp.Acknowledged),
		zap.Duration("duration", duration),
	)

	// Apply any configuration updates from the response
	if len(resp.ConfigurationUpdates) > 0 {
		h.logger.Info("Received configuration updates",
			zap.Any("updates", resp.ConfigurationUpdates),
		)
		// Configuration updates would be applied here
		// For now, just log them
	}
}

// SendFinalHeartbeat sends a final heartbeat with shutting_down status
func (h *HeartbeatManager) SendFinalHeartbeat() {
	h.logger.Info("Sending final heartbeat",
		zap.String("server_id", h.serverID),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	h.sendHeartbeat(ctx, "shutting_down")
}
