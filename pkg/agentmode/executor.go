package agentmode

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/reignx/reignx/pkg/core"
)

// Executor implements task execution via persistent agents
type Executor struct {
	config       *Config
	clients      map[string]*AgentClient
	clientsMu    sync.RWMutex
	grpcPool     *GRPCPool
}

// Config contains agent executor configuration
type Config struct {
	MaxConnections    int
	ConnectionTimeout time.Duration
	TaskTimeout       time.Duration
	RetryAttempts     int
	RetryDelay        time.Duration
	TLSEnabled        bool
	CertFile          string
	KeyFile           string
	CAFile            string
}

// NewExecutor creates a new agent mode executor
func NewExecutor(config *Config) *Executor {
	if config == nil {
		config = DefaultConfig()
	}

	return &Executor{
		config:   config,
		clients:  make(map[string]*AgentClient),
		grpcPool: NewGRPCPool(config.MaxConnections),
	}
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxConnections:    1000,
		ConnectionTimeout: 10 * time.Second,
		TaskTimeout:       300 * time.Second,
		RetryAttempts:     3,
		RetryDelay:        2 * time.Second,
		TLSEnabled:        true,
	}
}

// Execute executes a single task on a node via its agent
func (e *Executor) Execute(ctx context.Context, node *core.Node, task *core.Task) (*core.TaskResult, error) {
	if node.Mode != core.NodeModeAgent && node.Mode != core.NodeModeHybrid {
		return nil, fmt.Errorf("node %s is not in agent mode", node.ID)
	}

	if node.Status != core.NodeStatusOnline {
		return nil, fmt.Errorf("node %s is not online (status: %s)", node.ID, node.Status)
	}

	// Get or create agent client
	client, err := e.getClient(node)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent client: %w", err)
	}

	// Execute task with retries
	var result *core.TaskResult
	var lastErr error

	for attempt := 0; attempt <= e.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(e.config.RetryDelay):
			}
		}

		result, lastErr = client.ExecuteTask(ctx, task)
		if lastErr == nil {
			return result, nil
		}

		// Don't retry if context is cancelled
		if ctx.Err() != nil {
			break
		}
	}

	return result, fmt.Errorf("task execution failed after %d attempts: %w", e.config.RetryAttempts+1, lastErr)
}

// ExecuteBatch executes tasks in batch across multiple nodes
func (e *Executor) ExecuteBatch(ctx context.Context, nodes []*core.Node, task *core.Task, concurrency int) (map[string]*core.TaskResult, error) {
	if concurrency <= 0 {
		concurrency = 10
	}

	results := make(map[string]*core.TaskResult)
	resultsMu := sync.Mutex{}

	// Filter nodes to only include those in agent mode and online
	var agentNodes []*core.Node
	for _, node := range nodes {
		if (node.Mode == core.NodeModeAgent || node.Mode == core.NodeModeHybrid) &&
		   node.Status == core.NodeStatusOnline {
			agentNodes = append(agentNodes, node)
		}
	}

	if len(agentNodes) == 0 {
		return results, fmt.Errorf("no online nodes in agent mode found")
	}

	// Semaphore for concurrency control
	sem := make(chan struct{}, concurrency)
	wg := sync.WaitGroup{}
	errChan := make(chan error, len(agentNodes))

	for _, node := range agentNodes {
		wg.Add(1)
		go func(n *core.Node) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}

			// Execute task
			result, err := e.Execute(ctx, n, task)
			if err != nil {
				errChan <- fmt.Errorf("node %s: %w", n.ID, err)
				// Store failed result
				result = &core.TaskResult{
					Success:  false,
					ExitCode: -1,
					Error:    err.Error(),
				}
			}

			// Store result
			resultsMu.Lock()
			results[n.ID] = result
			resultsMu.Unlock()
		}(node)
	}

	// Wait for all tasks to complete
	wg.Wait()
	close(errChan)

	// Collect errors
	var lastErr error
	for err := range errChan {
		lastErr = err
	}

	return results, lastErr
}

// getClient retrieves or creates an agent client for a node
func (e *Executor) getClient(node *core.Node) (*AgentClient, error) {
	e.clientsMu.RLock()
	client, exists := e.clients[node.ID]
	e.clientsMu.RUnlock()

	if exists && client.IsHealthy() {
		return client, nil
	}

	// Need to create new client
	e.clientsMu.Lock()
	defer e.clientsMu.Unlock()

	// Double-check after acquiring write lock
	client, exists = e.clients[node.ID]
	if exists && client.IsHealthy() {
		return client, nil
	}

	// Close old client if exists
	if exists {
		client.Close()
	}

	// Create new client
	newClient, err := e.createClient(node)
	if err != nil {
		return nil, err
	}

	e.clients[node.ID] = newClient
	return newClient, nil
}

// createClient creates a new agent client
func (e *Executor) createClient(node *core.Node) (*AgentClient, error) {
	// Build gRPC connection address
	address := fmt.Sprintf("%s:50051", node.IPAddress) // Default agent port

	// Get gRPC connection from pool
	conn, err := e.grpcPool.Get(address, e.config)
	if err != nil {
		return nil, fmt.Errorf("failed to get gRPC connection: %w", err)
	}

	client := &AgentClient{
		nodeID:  node.ID,
		address: address,
		conn:    conn,
		config:  e.config,
	}

	return client, nil
}

// SupportsMode checks if this executor supports the given mode
func (e *Executor) SupportsMode(mode core.NodeMode) bool {
	return mode == core.NodeModeAgent || mode == core.NodeModeHybrid
}

// Name returns the executor name
func (e *Executor) Name() string {
	return "agent-executor"
}

// Close closes the executor and cleans up resources
func (e *Executor) Close() error {
	e.clientsMu.Lock()
	defer e.clientsMu.Unlock()

	for _, client := range e.clients {
		client.Close()
	}
	e.clients = make(map[string]*AgentClient)

	return e.grpcPool.Close()
}

// GetStats returns executor statistics
func (e *Executor) GetStats() *ExecutorStats {
	e.clientsMu.RLock()
	defer e.clientsMu.RUnlock()

	healthy := 0
	for _, client := range e.clients {
		if client.IsHealthy() {
			healthy++
		}
	}

	return &ExecutorStats{
		TotalClients:   len(e.clients),
		HealthyClients: healthy,
		PoolSize:       e.grpcPool.Size(),
	}
}

// ExecutorStats contains executor statistics
type ExecutorStats struct {
	TotalClients   int
	HealthyClients int
	PoolSize       int
}

// RemoveClient removes a client from the executor
func (e *Executor) RemoveClient(nodeID string) {
	e.clientsMu.Lock()
	defer e.clientsMu.Unlock()

	if client, exists := e.clients[nodeID]; exists {
		client.Close()
		delete(e.clients, nodeID)
	}
}
