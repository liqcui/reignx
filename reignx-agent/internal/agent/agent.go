package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	agentpb "github.com/reignx/reignx/api/proto"
	"github.com/reignx/reignx/reignx-agent/internal/cache"
	"github.com/reignx/reignx/reignx-agent/internal/executor"
	"github.com/reignx/reignx/reignx-agent/internal/metrics"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Config contains agent configuration
type Config struct {
	ServerAddr        string
	TLSEnabled        bool
	CertFile          string
	KeyFile           string
	CAFile            string
	HeartbeatInterval time.Duration
	CacheDir          string
	MaxConcurrency    int
	Logger            *zap.Logger
}

// Agent represents the ReignX agent
type Agent struct {
	config    *Config
	logger    *zap.Logger
	conn      *grpc.ClientConn
	client    agentpb.AgentServiceClient
	executor  *executor.Executor
	metrics   *metrics.Collector
	cache     *cache.Cache
	stream    agentpb.AgentService_StreamTasksClient
	stopChan  chan struct{}
	wg        sync.WaitGroup
	mu        sync.RWMutex
	running   bool
	agentID   string
	nodeID    string
	hostname  string
}

// New creates a new agent instance
func New(config *Config) (*Agent, error) {
	// Validate configuration
	if config.ServerAddr == "" {
		return nil, fmt.Errorf("server address is required")
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	// Initialize cache
	cacheInstance, err := cache.New(config.CacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector(config.Logger)

	// Initialize executor
	exec := executor.New(&executor.Config{
		MaxConcurrency: config.MaxConcurrency,
		Cache:          cacheInstance,
		Logger:         config.Logger,
	})

	agent := &Agent{
		config:   config,
		logger:   config.Logger,
		executor: exec,
		metrics:  metricsCollector,
		cache:    cacheInstance,
		stopChan: make(chan struct{}),
		hostname: hostname,
	}

	return agent, nil
}

// Start starts the agent
func (a *Agent) Start(ctx context.Context) error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return fmt.Errorf("agent is already running")
	}
	a.running = true
	a.mu.Unlock()

	a.logger.Info("Starting agent",
		zap.String("server", a.config.ServerAddr),
		zap.String("hostname", a.hostname),
	)

	// Connect to server
	if err := a.connect(); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Register with server
	if err := a.register(ctx); err != nil {
		return fmt.Errorf("failed to register with server: %w", err)
	}

	// Start heartbeat routine
	a.wg.Add(1)
	go a.heartbeatLoop(ctx)

	// Start task processing
	a.wg.Add(1)
	go a.taskLoop(ctx)

	// Start metrics collection
	a.wg.Add(1)
	go a.metricsLoop(ctx)

	a.logger.Info("Agent started successfully",
		zap.String("agent_id", a.agentID),
		zap.String("node_id", a.nodeID),
	)

	// Wait for stop signal
	select {
	case <-ctx.Done():
		a.logger.Info("Agent stopping due to context cancellation")
	case <-a.stopChan:
		a.logger.Info("Agent stopping due to stop signal")
	}

	return nil
}

// Stop gracefully stops the agent
func (a *Agent) Stop(ctx context.Context) error {
	a.mu.Lock()
	if !a.running {
		a.mu.Unlock()
		return nil
	}
	a.running = false
	a.mu.Unlock()

	a.logger.Info("Stopping agent gracefully...")

	// Send disconnect message
	if a.client != nil {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := a.client.Disconnect(disconnectCtx, &agentpb.DisconnectRequest{
			NodeId: a.nodeID,
			Reason: "graceful shutdown",
		})
		if err != nil {
			a.logger.Warn("Failed to send disconnect message", zap.Error(err))
		}
	}

	// Signal stop
	close(a.stopChan)

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		a.logger.Info("All routines stopped")
	case <-ctx.Done():
		a.logger.Warn("Shutdown timeout reached, forcing stop")
	}

	// Close connection
	if a.conn != nil {
		a.conn.Close()
	}

	// Close cache
	if err := a.cache.Close(); err != nil {
		a.logger.Error("Failed to close cache", zap.Error(err))
	}

	a.logger.Info("Agent stopped")
	return nil
}

// connect establishes connection to the server
func (a *Agent) connect() error {
	var opts []grpc.DialOption

	if a.config.TLSEnabled {
		// Load client certificate
		cert, err := tls.LoadX509KeyPair(a.config.CertFile, a.config.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load client certificate: %w", err)
		}

		// Load CA certificate
		caCert, err := os.ReadFile(a.config.CAFile)
		if err != nil {
			return fmt.Errorf("failed to read CA certificate: %w", err)
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("failed to add CA certificate to pool")
		}

		// Create TLS configuration
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      certPool,
			MinVersion:   tls.VersionTLS12,
		}

		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	a.logger.Info("Connecting to server", zap.String("address", a.config.ServerAddr))

	conn, err := grpc.Dial(a.config.ServerAddr, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial server: %w", err)
	}

	a.conn = conn
	a.client = agentpb.NewAgentServiceClient(conn)
	a.logger.Info("Connected to server successfully")

	return nil
}

// register registers the agent with the server
func (a *Agent) register(ctx context.Context) error {
	a.logger.Info("Registering with server")

	// Get system info
	sysInfo := a.metrics.GetSystemInfo()

	// Get local IP
	localIP := a.getLocalIP()

	// Create registration request
	req := &agentpb.RegisterRequest{
		Hostname:     a.hostname,
		IpAddress:    localIP,
		OsType:       getStringFromMap(sysInfo, "os"),
		OsVersion:    getStringFromMap(sysInfo, "platform_version"),
		Architecture: getStringFromMap(sysInfo, "kernel_arch"),
		AgentVersion: "0.1.0",
		Tags:         make(map[string]string),
		Metadata:     make(map[string]string),
	}

	// Add additional system info to metadata
	if platform, ok := sysInfo["platform"].(string); ok {
		req.Metadata["platform"] = platform
	}
	if kernelVersion, ok := sysInfo["kernel_version"].(string); ok {
		req.Metadata["kernel_version"] = kernelVersion
	}

	resp, err := a.client.Register(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("registration failed: %s", resp.Message)
	}

	a.agentID = resp.NodeId
	a.nodeID = resp.NodeId

	a.logger.Info("Registered with server",
		zap.String("agent_id", a.agentID),
		zap.String("node_id", a.nodeID),
		zap.Int32("heartbeat_interval", resp.HeartbeatIntervalSeconds),
	)

	// Update heartbeat interval if provided
	if resp.HeartbeatIntervalSeconds > 0 {
		a.config.HeartbeatInterval = time.Duration(resp.HeartbeatIntervalSeconds) * time.Second
	}

	return nil
}

// heartbeatLoop sends periodic heartbeats to the server
func (a *Agent) heartbeatLoop(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(a.config.HeartbeatInterval)
	defer ticker.Stop()

	a.logger.Info("Heartbeat loop started",
		zap.Duration("interval", a.config.HeartbeatInterval),
	)

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("Heartbeat loop stopped due to context cancellation")
			return
		case <-a.stopChan:
			a.logger.Info("Heartbeat loop stopped")
			return
		case <-ticker.C:
			if err := a.sendHeartbeat(ctx); err != nil {
				a.logger.Error("Failed to send heartbeat", zap.Error(err))
				// Attempt reconnection on heartbeat failure
				if err := a.reconnect(ctx); err != nil {
					a.logger.Error("Failed to reconnect", zap.Error(err))
				}
			}
		}
	}
}

// sendHeartbeat sends a heartbeat to the server
func (a *Agent) sendHeartbeat(ctx context.Context) error {
	// Collect current metrics
	currentMetrics := a.metrics.Collect()

	// Create heartbeat request
	req := &agentpb.HeartbeatRequest{
		NodeId:       a.nodeID,
		AgentVersion: "0.1.0",
		Status:       "online",
		Timestamp:    timestamppb.New(currentMetrics.Timestamp),
		Metrics: &agentpb.SystemMetrics{
			CpuPercent:    currentMetrics.CPUPercent,
			MemoryPercent: currentMetrics.MemoryPercent,
			DiskPercent:   currentMetrics.DiskPercent,
			UptimeSeconds: uint64(currentMetrics.Uptime),
		},
		ActiveTasks: int32(a.executor.GetRunningTasks()),
	}

	a.logger.Debug("Sending heartbeat",
		zap.Float64("cpu_percent", currentMetrics.CPUPercent),
		zap.Float64("memory_percent", currentMetrics.MemoryPercent),
	)

	resp, err := a.client.Heartbeat(ctx, req)
	if err != nil {
		return fmt.Errorf("heartbeat failed: %w", err)
	}

	// Process tasks received in heartbeat response
	if len(resp.PendingTasks) > 0 {
		a.logger.Info("Received tasks in heartbeat",
			zap.Int("task_count", len(resp.PendingTasks)),
		)

		for _, protoTask := range resp.PendingTasks {
			// Execute task asynchronously
			go a.executeTaskFromHeartbeat(ctx, protoTask)
		}
	}

	return nil
}

// executeTaskFromHeartbeat executes a task received in heartbeat response
func (a *Agent) executeTaskFromHeartbeat(ctx context.Context, protoTask *agentpb.Task) {
	a.logger.Info("Executing task from heartbeat",
		zap.String("task_id", protoTask.Id),
		zap.String("type", protoTask.Type),
		zap.String("command", protoTask.Command),
	)

	// Convert proto task to internal format
	task := &executor.Task{
		ID:      protoTask.Id,
		Type:    protoTask.Type,
		Command: protoTask.Command,
		Script:  protoTask.Script,
		Timeout: time.Duration(protoTask.TimeoutSeconds) * time.Second,
	}

	// Execute task
	result := a.executor.Execute(ctx, task)

	// Report result back to server via stream
	success := result.Status == "completed" && result.ExitCode == 0
	taskResult := &agentpb.TaskResult{
		TaskId:      result.TaskID,
		Success:     success,
		ExitCode:    int32(result.ExitCode),
		Stdout:      result.Output,
		Stderr:      "",
		Error:       result.Error,
		StartedAt:   timestamppb.New(result.StartedAt),
		CompletedAt: timestamppb.New(result.StartedAt.Add(result.Duration)),
		DurationMs:  result.Duration.Milliseconds(),
		Metadata:    make(map[string]string),
	}

	// Send result via ReportTaskResult RPC (polling approach)
	reportCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := a.client.ReportTaskResult(reportCtx, taskResult)
	if err != nil {
		a.logger.Error("Failed to report task result",
			zap.String("task_id", result.TaskID),
			zap.Error(err),
		)
	} else if !resp.Acknowledged {
		a.logger.Warn("Task result not acknowledged",
			zap.String("task_id", result.TaskID),
			zap.String("message", resp.Message),
		)
	} else {
		a.logger.Info("Task result reported successfully",
			zap.String("task_id", result.TaskID),
			zap.Bool("success", success),
			zap.Int("exit_code", result.ExitCode),
		)
	}
}

// taskLoop processes tasks from the server
func (a *Agent) taskLoop(ctx context.Context) {
	defer a.wg.Done()

	a.logger.Info("Task loop started")

	// Create task stream
	stream, err := a.client.StreamTasks(ctx)
	if err != nil {
		a.logger.Error("Failed to create task stream", zap.Error(err))
		return
	}

	// Store stream for use in executeTaskFromHeartbeat
	a.mu.Lock()
	a.stream = stream
	a.mu.Unlock()

	// Process messages from server
	for {
		select {
		case <-ctx.Done():
			a.logger.Info("Task loop stopped due to context cancellation")
			return
		case <-a.stopChan:
			a.logger.Info("Task loop stopped")
			return
		default:
			msg, err := stream.Recv()
			if err == io.EOF {
				a.logger.Info("Task stream closed by server")
				return
			}
			if err != nil {
				a.logger.Error("Error receiving from stream", zap.Error(err))
				time.Sleep(5 * time.Second) // Brief pause before reconnect
				return
			}

			// Process message
			go a.handleTaskMessage(ctx, stream, msg)
		}
	}
}

// handleTaskMessage handles a task message from the server
func (a *Agent) handleTaskMessage(ctx context.Context, stream agentpb.AgentService_StreamTasksClient, msg *agentpb.TaskMessage) {
	// Use oneof pattern to get task
	if task := msg.GetTask(); task != nil {
		a.logger.Info("Task received",
			zap.String("task_id", task.Id),
			zap.String("type", task.Type),
		)

		// Send acknowledgment
		ack := &agentpb.TaskMessage{
			Message: &agentpb.TaskMessage_Ack{
				Ack: &agentpb.TaskAck{
					TaskId:    task.Id,
					Received:  true,
					Timestamp: timestamppb.Now(),
				},
			},
		}
		stream.Send(ack)

		// Execute task
		result := a.executeTask(ctx, task)

		// Send result
		resultMsg := &agentpb.TaskMessage{
			Message: &agentpb.TaskMessage_Result{
				Result: result,
			},
		}
		if err := stream.Send(resultMsg); err != nil {
			a.logger.Error("Failed to send task result", zap.Error(err))
		}
	}
}

// executeTask executes a task
func (a *Agent) executeTask(ctx context.Context, protoTask *agentpb.Task) *agentpb.TaskResult {
	// Convert proto task to executor task
	task := &executor.Task{
		ID:         protoTask.Id,
		Type:       protoTask.Type,
		Command:    protoTask.Command,
		Script:     protoTask.Script,
		Parameters: make(map[string]interface{}),
		Timeout:    time.Duration(protoTask.TimeoutSeconds) * time.Second,
		Priority:   int(protoTask.Priority),
	}

	// Convert parameters
	for k, v := range protoTask.Parameters {
		task.Parameters[k] = v
	}

	// Execute
	result := a.executor.Execute(ctx, task)

	// Convert result to proto
	success := result.Status == "completed" && result.ExitCode == 0
	protoResult := &agentpb.TaskResult{
		TaskId:      result.TaskID,
		Success:     success,
		ExitCode:    int32(result.ExitCode),
		Stdout:      result.Output,
		Stderr:      "",
		Error:       result.Error,
		StartedAt:   timestamppb.New(result.StartedAt),
		CompletedAt: timestamppb.New(result.StartedAt.Add(result.Duration)),
		DurationMs:  result.Duration.Milliseconds(),
		Metadata:    make(map[string]string),
	}

	return protoResult
}

// metricsLoop collects and reports metrics periodically
func (a *Agent) metricsLoop(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	a.logger.Info("Metrics loop started")

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("Metrics loop stopped due to context cancellation")
			return
		case <-a.stopChan:
			a.logger.Info("Metrics loop stopped")
			return
		case <-ticker.C:
			if err := a.reportMetrics(ctx); err != nil {
				a.logger.Error("Failed to report metrics", zap.Error(err))
			}
		}
	}
}

// reportMetrics reports collected metrics to the server
func (a *Agent) reportMetrics(ctx context.Context) error {
	currentMetrics := a.metrics.Collect()

	req := &agentpb.MetricsRequest{
		NodeId:    a.nodeID,
		Timestamp: timestamppb.New(currentMetrics.Timestamp),
		System: &agentpb.SystemMetrics{
			CpuPercent:    currentMetrics.CPUPercent,
			MemoryPercent: currentMetrics.MemoryPercent,
			DiskPercent:   currentMetrics.DiskPercent,
			UptimeSeconds: uint64(currentMetrics.Uptime),
		},
	}

	a.logger.Debug("Reporting metrics",
		zap.Float64("cpu_percent", currentMetrics.CPUPercent),
		zap.Float64("memory_percent", currentMetrics.MemoryPercent),
	)

	_, err := a.client.ReportMetrics(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to report metrics: %w", err)
	}

	return nil
}

// reconnect attempts to reconnect to the server
func (a *Agent) reconnect(ctx context.Context) error {
	a.logger.Warn("Attempting to reconnect to server")

	// Close existing connection
	if a.conn != nil {
		a.conn.Close()
	}

	// Exponential backoff
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for attempt := 1; attempt <= 10; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-a.stopChan:
			return fmt.Errorf("agent stopped")
		default:
		}

		a.logger.Info("Reconnection attempt",
			zap.Int("attempt", attempt),
			zap.Duration("backoff", backoff),
		)

		if err := a.connect(); err != nil {
			a.logger.Error("Reconnection failed", zap.Error(err))
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Re-register
		if err := a.register(ctx); err != nil {
			a.logger.Error("Re-registration failed", zap.Error(err))
			continue
		}

		a.logger.Info("Reconnected successfully")
		return nil
	}

	return fmt.Errorf("failed to reconnect after 10 attempts")
}

// getLocalIP gets the local IP address
func (a *Agent) getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return "unknown"
}

// getStringFromMap safely gets a string from a map[string]interface{}
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
