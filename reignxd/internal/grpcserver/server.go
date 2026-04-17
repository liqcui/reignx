package grpcserver

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	agentpb "github.com/reignx/reignx/api/proto"
	"github.com/reignx/reignx/pkg/core"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// Config contains gRPC server configuration
type Config struct {
	Logger   *zap.Logger
	NodeRepo core.NodeRepository
	TaskRepo core.TaskRepository
}

// AgentServer implements the gRPC agent service
type AgentServer struct {
	agentpb.UnimplementedAgentServiceServer
	logger      *zap.Logger
	nodeRepo    core.NodeRepository
	taskRepo    core.TaskRepository
	taskStreams sync.Map // map[agentID]chan *agentpb.TaskMessage
	mu          sync.RWMutex
}

// Register registers the gRPC server
func Register(srv *grpc.Server, config *Config) {
	server := &AgentServer{
		logger:   config.Logger,
		nodeRepo: config.NodeRepo,
		taskRepo: config.TaskRepo,
	}

	agentpb.RegisterAgentServiceServer(srv, server)

	config.Logger.Info("gRPC server registered")
}

// Register handles agent registration
func (s *AgentServer) Register(ctx context.Context, req *agentpb.RegisterRequest) (*agentpb.RegisterResponse, error) {
	// Get peer info for logging
	p, _ := peer.FromContext(ctx)

	s.logger.Info("Agent registration request received",
		zap.String("hostname", req.Hostname),
		zap.String("ip", req.IpAddress),
		zap.String("os", req.OsType),
		zap.String("peer", p.Addr.String()),
	)

	// Create or update node
	node := &core.Node{
		Hostname:  req.Hostname,
		IPAddress: req.IpAddress,
		Mode:      core.NodeModeAgent,
		Status:    core.NodeStatusOnline,
		OSType:    req.OsType,
		LastSeen:  time.Now(),
		Tags:      req.Tags,
		Metadata:  make(map[string]interface{}),
	}

	// Add metadata from request
	if req.OsVersion != "" {
		node.Metadata["os_version"] = req.OsVersion
	}
	if req.Architecture != "" {
		node.Metadata["architecture"] = req.Architecture
	}
	if req.AgentVersion != "" {
		node.Metadata["agent_version"] = req.AgentVersion
	}
	if req.MacAddress != "" {
		node.Metadata["mac_address"] = req.MacAddress
	}

	// Merge additional metadata
	for k, v := range req.Metadata {
		node.Metadata[k] = v
	}

	var agentID string

	// Check if node exists by IP
	existingNode, err := s.nodeRepo.GetByIP(ctx, node.IPAddress)
	if err == nil {
		// Update existing node
		existingNode.Hostname = node.Hostname
		existingNode.Mode = core.NodeModeAgent
		existingNode.Status = core.NodeStatusOnline
		existingNode.OSType = node.OSType
		existingNode.LastSeen = time.Now()
		existingNode.Tags = node.Tags
		existingNode.Metadata = node.Metadata

		if err := s.nodeRepo.Update(ctx, existingNode); err != nil {
			s.logger.Error("Failed to update node", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to update node")
		}
		agentID = existingNode.ID

		s.logger.Info("Updated existing node",
			zap.String("node_id", agentID),
			zap.String("hostname", existingNode.Hostname),
		)
	} else {
		// Create new node
		if err := s.nodeRepo.Create(ctx, node); err != nil {
			s.logger.Error("Failed to create node", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to create node")
		}
		agentID = node.ID

		s.logger.Info("Created new node",
			zap.String("node_id", agentID),
			zap.String("hostname", node.Hostname),
		)
	}

	// Return registration response
	return &agentpb.RegisterResponse{
		Success:                  true,
		NodeId:                   agentID,
		Message:                  "Agent registered successfully",
		HeartbeatIntervalSeconds: 30,
		Config:                   make(map[string]string),
	}, nil
}

// Heartbeat handles agent heartbeat
func (s *AgentServer) Heartbeat(ctx context.Context, req *agentpb.HeartbeatRequest) (*agentpb.HeartbeatResponse, error) {
	s.logger.Debug("Heartbeat received",
		zap.String("node_id", req.NodeId),
		zap.String("status", req.Status),
	)

	// Update node last_seen and status
	err := s.nodeRepo.UpdateStatus(ctx, req.NodeId, core.NodeStatusOnline, time.Now())
	if err != nil {
		s.logger.Error("Failed to update node status",
			zap.String("node_id", req.NodeId),
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, "failed to update node status")
	}

	// Store metrics if provided
	if req.Metrics != nil {
		s.storeMetrics(ctx, req.NodeId, req.Metrics)
	}

	// Get pending tasks for this node
	pendingTasks, err := s.taskRepo.GetPendingTasks(ctx, req.NodeId, 10)
	if err != nil {
		s.logger.Error("Failed to get pending tasks",
			zap.String("node_id", req.NodeId),
			zap.Error(err))
		// Don't fail heartbeat if task query fails
		pendingTasks = []*core.Task{}
	}

	// Convert to proto format and mark as assigned
	protoTasks := make([]*agentpb.Task, 0, len(pendingTasks))
	for _, task := range pendingTasks {
		protoTask := &agentpb.Task{
			Id:             task.ID,
			Type:           string(task.Type),
			Command:        task.Command,
			Script:         task.Script,
			TimeoutSeconds: int32(task.Timeout.Seconds()),
		}

		protoTasks = append(protoTasks, protoTask)

		// Mark task as assigned to prevent duplicate delivery
		s.taskRepo.UpdateStatus(ctx, task.ID, core.TaskStatusAssigned)

		s.logger.Info("Task assigned to agent via heartbeat",
			zap.String("task_id", task.ID),
			zap.String("node_id", req.NodeId),
			zap.String("command", task.Command),
		)
	}

	if len(protoTasks) > 0 {
		s.logger.Info("Delivering tasks in heartbeat response",
			zap.String("node_id", req.NodeId),
			zap.Int("task_count", len(protoTasks)),
		)
	}

	return &agentpb.HeartbeatResponse{
		Acknowledged:  true,
		ConfigUpdates: make(map[string]string),
		PendingTasks:  protoTasks,
	}, nil
}

// StreamTasks handles bidirectional task streaming
func (s *AgentServer) StreamTasks(stream agentpb.AgentService_StreamTasksServer) error {
	ctx := stream.Context()

	s.logger.Info("Task stream opened")

	var nodeID string

	// Process messages from agent
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Task stream closed",
				zap.String("node_id", nodeID),
				zap.Error(ctx.Err()),
			)
			return ctx.Err()

		default:
			msg, err := stream.Recv()
			if err == io.EOF {
				s.logger.Info("Task stream closed by agent",
					zap.String("node_id", nodeID),
				)
				return nil
			}
			if err != nil {
				s.logger.Error("Error receiving from stream",
					zap.String("node_id", nodeID),
					zap.Error(err),
				)
				return err
			}

			// Process message based on oneof type
			switch m := msg.Message.(type) {
			case *agentpb.TaskMessage_Ack:
				// Task acknowledgment
				s.logger.Debug("Task acknowledged",
					zap.String("task_id", m.Ack.TaskId),
				)
				// NodeID will be tracked from first result message

			case *agentpb.TaskMessage_Result:
				// Task execution result
				status := "unknown"
				if m.Result.Success {
					status = "completed"
				} else {
					status = "failed"
				}

				s.logger.Info("Task result received",
					zap.String("task_id", m.Result.TaskId),
					zap.String("status", status),
					zap.Int32("exit_code", m.Result.ExitCode),
				)

				// Update task in database
				if err := s.updateTaskResult(ctx, m.Result); err != nil {
					s.logger.Error("Failed to update task result",
						zap.String("task_id", m.Result.TaskId),
						zap.Error(err),
					)
				}

			case *agentpb.TaskMessage_Task:
				// Should not receive tasks from agent (only send to agent)
				s.logger.Warn("Received task from agent (unexpected)")

			default:
				s.logger.Warn("Unknown message type in TaskMessage oneof")
			}
		}
	}
}

// ReportMetrics handles metrics reporting
func (s *AgentServer) ReportMetrics(ctx context.Context, req *agentpb.MetricsRequest) (*agentpb.MetricsResponse, error) {
	s.logger.Debug("Metrics report received",
		zap.String("node_id", req.NodeId),
	)

	// Store metrics
	if req.System != nil {
		s.storeMetrics(ctx, req.NodeId, req.System)
	}

	return &agentpb.MetricsResponse{
		Acknowledged: true,
	}, nil
}

// Disconnect handles graceful agent disconnect
func (s *AgentServer) Disconnect(ctx context.Context, req *agentpb.DisconnectRequest) (*agentpb.DisconnectResponse, error) {
	s.logger.Info("Agent disconnect request",
		zap.String("node_id", req.NodeId),
		zap.String("reason", req.Reason),
	)

	// Update node status to offline
	err := s.nodeRepo.UpdateStatus(ctx, req.NodeId, core.NodeStatusOffline, time.Now())
	if err != nil {
		s.logger.Error("Failed to update node status", zap.Error(err))
		// Don't return error, allow disconnect to proceed
	}

	return &agentpb.DisconnectResponse{
		Acknowledged: true,
	}, nil
}

// storeMetrics stores system metrics (placeholder - could write to time-series DB)
func (s *AgentServer) storeMetrics(ctx context.Context, nodeID string, metrics *agentpb.SystemMetrics) {
	s.logger.Debug("Storing metrics",
		zap.String("node_id", nodeID),
		zap.Float64("cpu_percent", metrics.CpuPercent),
		zap.Float64("memory_percent", metrics.MemoryPercent),
		zap.Float64("disk_percent", metrics.DiskPercent),
	)

	// TODO: Store in time-series database (Prometheus, InfluxDB, etc.)
	// For now, we could update node metadata
	node, err := s.nodeRepo.Get(ctx, nodeID)
	if err != nil {
		s.logger.Warn("Failed to get node for metrics storage", zap.Error(err))
		return
	}

	// Update metadata with latest metrics
	if node.Metadata == nil {
		node.Metadata = make(map[string]interface{})
	}

	node.Metadata["last_cpu_percent"] = metrics.CpuPercent
	node.Metadata["last_memory_percent"] = metrics.MemoryPercent
	node.Metadata["last_disk_percent"] = metrics.DiskPercent
	node.Metadata["last_metrics_update"] = time.Now().Format(time.RFC3339)

	if err := s.nodeRepo.Update(ctx, node); err != nil {
		s.logger.Warn("Failed to update node with metrics", zap.Error(err))
	}
}

// updateTaskResult updates the task result in the database
func (s *AgentServer) updateTaskResult(ctx context.Context, result *agentpb.TaskResult) error {
	task, err := s.taskRepo.Get(ctx, result.TaskId)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Update task status based on success field
	if result.Success {
		task.Status = core.TaskStatusCompleted
	} else {
		task.Status = core.TaskStatusFailed
	}

	// Create task result
	taskResult := &core.TaskResult{
		Success:  result.Success,
		ExitCode: int(result.ExitCode),
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		Error:    result.Error,
	}

	if result.StartedAt != nil {
		startTime := result.StartedAt.AsTime()
		task.StartedAt = &startTime
	}

	if result.CompletedAt != nil {
		completedTime := result.CompletedAt.AsTime()
		task.CompletedAt = &completedTime

		if task.StartedAt != nil {
			taskResult.Duration = completedTime.Sub(*task.StartedAt)
		}
	}

	task.Result = taskResult

	// Update task in database
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	s.logger.Info("Task result stored",
		zap.String("task_id", task.ID),
		zap.String("status", string(task.Status)),
		zap.Int("exit_code", taskResult.ExitCode),
	)

	return nil
}

// ReportTaskResult handles task result reporting from agents
func (s *AgentServer) ReportTaskResult(ctx context.Context, result *agentpb.TaskResult) (*agentpb.TaskResultResponse, error) {
	s.logger.Info("Task result reported",
		zap.String("task_id", result.TaskId),
		zap.Bool("success", result.Success),
		zap.Int32("exit_code", result.ExitCode),
	)

	// Update task in database
	if err := s.updateTaskResult(ctx, result); err != nil {
		s.logger.Error("Failed to store task result",
			zap.String("task_id", result.TaskId),
			zap.Error(err),
		)
		return &agentpb.TaskResultResponse{
			Acknowledged: false,
			Message:      fmt.Sprintf("failed to store result: %v", err),
		}, nil
	}

	return &agentpb.TaskResultResponse{
		Acknowledged: true,
		Message:      "result received",
	}, nil
}

// SendTaskToAgent sends a task to a specific agent (helper method for scheduler)
func (s *AgentServer) SendTaskToAgent(agentID string, task *agentpb.Task) error {
	s.mu.RLock()
	stream, ok := s.taskStreams.Load(agentID)
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("agent %s not connected", agentID)
	}

	taskChan, ok := stream.(chan *agentpb.TaskMessage)
	if !ok {
		return fmt.Errorf("invalid stream type for agent %s", agentID)
	}

	// Send task via channel using oneof pattern
	select {
	case taskChan <- &agentpb.TaskMessage{
		Message: &agentpb.TaskMessage_Task{
			Task: task,
		},
	}:
		s.logger.Info("Task sent to agent",
			zap.String("agent_id", agentID),
			zap.String("task_id", task.Id),
		)
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout sending task to agent %s", agentID)
	}
}
