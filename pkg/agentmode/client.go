package agentmode

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/reignx/reignx/pkg/core"
	"google.golang.org/grpc"
)

// AgentClient manages communication with a single agent
type AgentClient struct {
	nodeID     string
	address    string
	conn       *grpc.ClientConn
	config     *Config
	stream     grpc.BidiStreamingClient[TaskMessage, TaskMessage]
	streamMu   sync.Mutex
	healthy    bool
	healthMu   sync.RWMutex
	lastSeen   time.Time
	closeOnce  sync.Once
}

// TaskMessage placeholder - will be generated from proto
type TaskMessage struct {
	TaskID string
	// Will be replaced with actual proto-generated type
}

// ExecuteTask executes a task via the agent
func (c *AgentClient) ExecuteTask(ctx context.Context, task *core.Task) (*core.TaskResult, error) {
	startTime := time.Now()

	// Apply timeout
	if task.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, task.Timeout)
		defer cancel()
	}

	// Create task message
	taskMsg := c.convertTask(task)

	// Send task via stream
	err := c.sendTask(ctx, taskMsg)
	if err != nil {
		return &core.TaskResult{
			Success:  false,
			ExitCode: -1,
			Error:    fmt.Sprintf("failed to send task: %v", err),
			Duration: time.Since(startTime),
		}, err
	}

	// Wait for result
	result, err := c.waitForResult(ctx, task.ID, task.Timeout)
	if err != nil {
		return &core.TaskResult{
			Success:  false,
			ExitCode: -1,
			Error:    fmt.Sprintf("failed to receive result: %v", err),
			Duration: time.Since(startTime),
		}, err
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// sendTask sends a task to the agent
func (c *AgentClient) sendTask(ctx context.Context, task interface{}) error {
	c.streamMu.Lock()
	defer c.streamMu.Unlock()

	// Ensure stream is initialized
	if c.stream == nil {
		// TODO: Initialize bidirectional stream
		// This will use the proto-generated client
		return fmt.Errorf("stream not initialized")
	}

	// TODO: Send task via stream
	// stream.Send(task)

	return nil
}

// waitForResult waits for task execution result
func (c *AgentClient) waitForResult(ctx context.Context, taskID string, timeout time.Duration) (*core.TaskResult, error) {
	resultChan := make(chan *core.TaskResult, 1)
	errChan := make(chan error, 1)

	// Start goroutine to receive result
	go func() {
		// TODO: Receive result from stream
		// This is a placeholder implementation
		result := &core.TaskResult{
			Success:  true,
			ExitCode: 0,
			Stdout:   "Task executed successfully",
		}
		resultChan <- result
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// convertTask converts core.Task to proto Task
func (c *AgentClient) convertTask(task *core.Task) interface{} {
	// TODO: Convert to proto-generated Task type
	return task
}

// IsHealthy checks if the agent client is healthy
func (c *AgentClient) IsHealthy() bool {
	c.healthMu.RLock()
	defer c.healthMu.RUnlock()

	// Consider unhealthy if no recent activity
	if time.Since(c.lastSeen) > 2*time.Minute {
		return false
	}

	return c.healthy && c.conn != nil
}

// UpdateHealth updates the health status
func (c *AgentClient) UpdateHealth(healthy bool) {
	c.healthMu.Lock()
	defer c.healthMu.Unlock()

	c.healthy = healthy
	if healthy {
		c.lastSeen = time.Now()
	}
}

// Close closes the agent client
func (c *AgentClient) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.streamMu.Lock()
		defer c.streamMu.Unlock()

		if c.stream != nil {
			// TODO: Close stream properly
			c.stream = nil
		}

		if c.conn != nil {
			err = c.conn.Close()
			c.conn = nil
		}

		c.UpdateHealth(false)
	})

	return err
}

// GetNodeID returns the node ID
func (c *AgentClient) GetNodeID() string {
	return c.nodeID
}

// GetAddress returns the agent address
func (c *AgentClient) GetAddress() string {
	return c.address
}

// Ping sends a ping to check agent connectivity
func (c *AgentClient) Ping(ctx context.Context) error {
	// TODO: Implement ping using heartbeat RPC
	c.UpdateHealth(true)
	return nil
}
