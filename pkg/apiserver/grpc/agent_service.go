package grpc

import (
	"context"
	"database/sql"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	agentpb "github.com/reignx/reignx/api/proto/gen"
	"github.com/reignx/reignx/pkg/database/models"
	"github.com/reignx/reignx/pkg/database/repository"
	"github.com/reignx/reignx/pkg/observability/metrics"
)

// agentService implements the AgentServiceServer interface
type agentService struct {
	agentpb.UnimplementedAgentServiceServer
	repository repository.ServerRepository
	logger     *zap.Logger
	metrics    *metrics.Metrics
}

// NewAgentService creates a new agent service
func NewAgentService(repo repository.ServerRepository, logger *zap.Logger, metrics *metrics.Metrics) agentpb.AgentServiceServer {
	return &agentService{
		repository: repo,
		logger:     logger,
		metrics:    metrics,
	}
}

// RegisterAgent handles agent registration
func (s *agentService) RegisterAgent(ctx context.Context, req *agentpb.AgentInfo) (*agentpb.RegistrationResponse, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if s.metrics != nil {
			s.metrics.RecordHistogram("grpc_request_duration_seconds", duration, map[string]string{
				"method": "RegisterAgent",
			})
			s.metrics.RecordCounter("grpc_requests_total", 1, map[string]string{
				"method": "RegisterAgent",
			})
		}
	}()

	s.logger.Info("Agent registration request received",
		zap.String("agent_id", req.AgentId),
		zap.String("hostname", req.Hostname),
		zap.String("ip_address", req.IpAddress),
		zap.String("os_type", req.OsType),
		zap.String("agent_version", req.AgentVersion),
	)

	// Validate required fields
	if req.Hostname == "" {
		return nil, status.Error(codes.InvalidArgument, "hostname is required")
	}
	if req.IpAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "ip_address is required")
	}
	if req.OsType == "" {
		return nil, status.Error(codes.InvalidArgument, "os_type is required")
	}

	// Convert tags map to JSONB
	tags := make(models.JSONB)
	for k, v := range req.Tags {
		tags[k] = v
	}

	// Convert capabilities to metadata
	metadata := make(models.JSONB)
	if req.Capabilities != nil {
		capabilities := make(map[string]interface{})
		for k, v := range req.Capabilities {
			capabilities[k] = v
		}
		metadata["capabilities"] = capabilities
	}

	// Create server model
	server := &models.Server{
		Hostname:     req.Hostname,
		IPAddress:    req.IpAddress,
		OSType:       req.OsType,
		Mode:    string(models.AgentModePersistent),
		Status:       string(models.ServerStatusOnline),
		LastSeen: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
		Tags:     tags,
		Metadata: metadata,
	}

	// Set optional fields
	if req.OsVersion != "" {
		server.OSVersion = sql.NullString{String: req.OsVersion, Valid: true}
	}
	if req.Architecture != "" {
		server.Architecture = sql.NullString{String: req.Architecture, Valid: true}
	}
	if req.AgentVersion != "" {
		server.AgentVersion = sql.NullString{String: req.AgentVersion, Valid: true}
	}

	// Upsert server record
	result, err := s.repository.Upsert(ctx, server)
	if err != nil {
		s.logger.Error("Failed to upsert server",
			zap.String("hostname", req.Hostname),
			zap.Error(err),
		)
		if s.metrics != nil {
			s.metrics.RecordCounter("agent_registration_errors_total", 1, nil)
		}
		return nil, status.Errorf(codes.Internal, "failed to register server: %v", err)
	}

	// Record successful registration
	if s.metrics != nil {
		s.metrics.RecordCounter("agent_registrations_total", 1, map[string]string{
			"os_type": req.OsType,
		})
		s.metrics.RecordGauge("agent_connections", 1, map[string]string{
			"status": "online",
		})
	}

	s.logger.Info("Agent registered successfully",
		zap.String("server_id", result.ID),
		zap.String("hostname", result.Hostname),
		zap.String("ip_address", result.IPAddress),
	)

	// Return registration response
	return &agentpb.RegistrationResponse{
		ServerId:                 result.ID,
		Message:                  "Registration successful",
		HeartbeatIntervalSeconds: 30, // 30 seconds default
		Configuration:            make(map[string]string),
	}, nil
}

// Heartbeat handles periodic heartbeat updates
func (s *agentService) Heartbeat(ctx context.Context, req *agentpb.HeartbeatRequest) (*agentpb.HeartbeatResponse, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if s.metrics != nil {
			s.metrics.RecordHistogram("grpc_request_duration_seconds", duration, map[string]string{
				"method": "Heartbeat",
			})
			s.metrics.RecordCounter("grpc_requests_total", 1, map[string]string{
				"method": "Heartbeat",
			})
		}
	}()

	// Validate required fields
	if req.ServerId == "" {
		return nil, status.Error(codes.InvalidArgument, "server_id is required")
	}

	// Convert timestamp
	timestamp := time.Now()
	if req.Timestamp != nil {
		timestamp = req.Timestamp.AsTime()
	}

	// Update heartbeat
	err := s.repository.UpdateHeartbeat(ctx, req.ServerId, timestamp)
	if err != nil {
		s.logger.Error("Failed to update heartbeat",
			zap.String("server_id", req.ServerId),
			zap.Error(err),
		)
		if s.metrics != nil {
			s.metrics.RecordCounter("agent_heartbeat_errors_total", 1, nil)
		}
		return nil, status.Errorf(codes.Internal, "failed to update heartbeat: %v", err)
	}

	// Persist metrics to database
	if req.Metrics != nil {
		// Get server from database to update metadata
		server, err := s.repository.GetByID(ctx, req.ServerId)
		if err == nil {
			// Create metrics data structure
			metricsData := map[string]interface{}{
				"cpu_percent":    req.Metrics.CpuPercent,
				"memory_percent": req.Metrics.MemoryPercent,
				"disk_percent":   req.Metrics.DiskPercent,
				"uptime_seconds": req.Metrics.UptimeSeconds,
				"network_rx_mb":  req.Metrics.NetworkRxMb,
				"network_tx_mb":  req.Metrics.NetworkTxMb,
				"running_tasks":  req.Metrics.RunningTasks,
				"timestamp":      timestamp.Unix(),
			}

			// Initialize metadata if nil
			if server.Metadata == nil {
				server.Metadata = make(models.JSONB)
			}

			// Store metrics in server metadata
			server.Metadata["metrics"] = metricsData

			// Update server record
			if _, err := s.repository.Upsert(ctx, server); err != nil {
				s.logger.Warn("Failed to persist metrics to database",
					zap.String("server_id", req.ServerId),
					zap.Error(err))
			}
		}
	}

	// Record Prometheus metrics from heartbeat
	if s.metrics != nil && req.Metrics != nil {
		s.metrics.RecordCounter("agent_heartbeats_total", 1, map[string]string{
			"server_id": req.ServerId,
			"status":    req.Status,
		})
		s.metrics.RecordGauge("node_cpu_percent", req.Metrics.CpuPercent, map[string]string{
			"node_id": req.ServerId,
		})
		s.metrics.RecordGauge("node_memory_percent", req.Metrics.MemoryPercent, map[string]string{
			"node_id": req.ServerId,
		})
		s.metrics.RecordGauge("node_disk_percent", req.Metrics.DiskPercent, map[string]string{
			"node_id": req.ServerId,
		})
		s.metrics.RecordGauge("node_uptime_seconds", float64(req.Metrics.UptimeSeconds), map[string]string{
			"node_id": req.ServerId,
		})
	}

	s.logger.Debug("Heartbeat received",
		zap.String("server_id", req.ServerId),
		zap.String("status", req.Status),
	)

	return &agentpb.HeartbeatResponse{
		Acknowledged:          true,
		ConfigurationUpdates:  make(map[string]string),
	}, nil
}

// StreamTasks handles bidirectional task streaming (not implemented yet)
func (s *agentService) StreamTasks(stream agentpb.AgentService_StreamTasksServer) error {
	s.logger.Warn("StreamTasks called but not implemented yet")
	return status.Error(codes.Unimplemented, "StreamTasks not implemented yet")
}

// ReportMetrics handles detailed metrics reporting (not implemented yet)
func (s *agentService) ReportMetrics(ctx context.Context, req *agentpb.MetricsReport) (*agentpb.MetricsResponse, error) {
	s.logger.Debug("ReportMetrics called",
		zap.String("server_id", req.ServerId),
	)

	// For now, just acknowledge receipt
	// Full implementation will store metrics in a time-series database
	return &agentpb.MetricsResponse{
		Acknowledged: true,
		Message:      "Metrics received (stored in logs only for now)",
	}, nil
}
