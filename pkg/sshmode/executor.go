package sshmode

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/reignx/reignx/pkg/core"
	"golang.org/x/crypto/ssh"
)

// Executor implements task execution via SSH
type Executor struct {
	config     *Config
	pool       *ConnectionPool
	mu         sync.RWMutex
}

// Config contains SSH executor configuration
type Config struct {
	MaxConnections int
	Timeout        time.Duration
	KeepAlive      time.Duration
	RetryAttempts  int
	RetryDelay     time.Duration
}

// NewExecutor creates a new SSH executor
func NewExecutor(config *Config) *Executor {
	if config == nil {
		config = DefaultConfig()
	}

	return &Executor{
		config: config,
		pool:   NewConnectionPool(config.MaxConnections),
	}
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxConnections: 100,
		Timeout:        30 * time.Second,
		KeepAlive:      10 * time.Second,
		RetryAttempts:  3,
		RetryDelay:     2 * time.Second,
	}
}

// Execute executes a single task on a node via SSH
func (e *Executor) Execute(ctx context.Context, node *core.Node, task *core.Task) (*core.TaskResult, error) {
	if node.SSHConfig == nil {
		return nil, fmt.Errorf("node %s has no SSH configuration", node.ID)
	}

	startTime := time.Now()

	// Get SSH client from pool
	client, err := e.pool.Get(node.ID, node.SSHConfig)
	if err != nil {
		return &core.TaskResult{
			Success:  false,
			ExitCode: -1,
			Error:    fmt.Sprintf("failed to get SSH connection: %v", err),
			Duration: time.Since(startTime),
		}, err
	}
	defer e.pool.Release(node.ID, client)

	// Create session with timeout
	sessionCtx, cancel := context.WithTimeout(ctx, task.Timeout)
	defer cancel()

	result, err := e.executeWithSession(sessionCtx, client, task)
	result.Duration = time.Since(startTime)

	return result, err
}

// executeWithSession executes a task using an SSH session
func (e *Executor) executeWithSession(ctx context.Context, client *ssh.Client, task *core.Task) (*core.TaskResult, error) {
	session, err := client.NewSession()
	if err != nil {
		return &core.TaskResult{
			Success:  false,
			ExitCode: -1,
			Error:    fmt.Sprintf("failed to create SSH session: %v", err),
		}, err
	}
	defer session.Close()

	// Capture stdout and stderr
	var stdoutBuf, stderrBuf limitedBuffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	// Build command based on task type
	command := e.buildCommand(task)

	// Execute command
	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				exitCode = exitErr.ExitStatus()
			} else {
				exitCode = -1
			}
		}

		return &core.TaskResult{
			Success:  err == nil && exitCode == 0,
			ExitCode: exitCode,
			Stdout:   stdoutBuf.String(),
			Stderr:   stderrBuf.String(),
			Error:    formatError(err),
		}, nil

	case <-ctx.Done():
		// Attempt to signal the session to stop
		session.Signal(ssh.SIGTERM)
		time.Sleep(1 * time.Second)
		session.Signal(ssh.SIGKILL)

		return &core.TaskResult{
			Success:  false,
			ExitCode: -1,
			Stdout:   stdoutBuf.String(),
			Stderr:   stderrBuf.String(),
			Error:    "execution timeout",
		}, ctx.Err()
	}
}

// buildCommand builds the command string based on task type
func (e *Executor) buildCommand(task *core.Task) string {
	switch task.Type {
	case core.TaskTypeCommand:
		return task.Command

	case core.TaskTypeScript:
		// Create temporary script file and execute
		return fmt.Sprintf("cat > /tmp/reignx-script-%s.sh << 'REIGNX_EOF'\n%s\nREIGNX_EOF\nchmod +x /tmp/reignx-script-%s.sh && /tmp/reignx-script-%s.sh && rm -f /tmp/reignx-script-%s.sh",
			task.ID, task.Script, task.ID, task.ID, task.ID)

	case core.TaskTypeHealthCheck:
		return "uptime && df -h / && free -m"

	case core.TaskTypeReboot:
		return "sudo reboot"

	default:
		return task.Command
	}
}

// ExecuteBatch executes tasks in batch across multiple nodes
func (e *Executor) ExecuteBatch(ctx context.Context, nodes []*core.Node, task *core.Task, concurrency int) (map[string]*core.TaskResult, error) {
	if concurrency <= 0 {
		concurrency = 10
	}

	results := make(map[string]*core.TaskResult)
	resultsMu := sync.Mutex{}

	// Create semaphore for concurrency control
	sem := make(chan struct{}, concurrency)
	wg := sync.WaitGroup{}
	errChan := make(chan error, len(nodes))

	for _, node := range nodes {
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

	// Collect any errors
	var lastErr error
	for err := range errChan {
		lastErr = err
	}

	return results, lastErr
}

// SupportsMode checks if this executor supports the given mode
func (e *Executor) SupportsMode(mode core.NodeMode) bool {
	return mode == core.NodeModeSSH || mode == core.NodeModeHybrid
}

// Name returns the executor name
func (e *Executor) Name() string {
	return "ssh-executor"
}

// formatError formats an error for display
func formatError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// limitedBuffer is a buffer that limits the amount of data stored
type limitedBuffer struct {
	buf      []byte
	maxSize  int
	overflow bool
}

func (lb *limitedBuffer) Write(p []byte) (n int, err error) {
	if lb.maxSize == 0 {
		lb.maxSize = 1024 * 1024 // 1MB default
	}

	if len(lb.buf)+len(p) > lb.maxSize {
		lb.overflow = true
		// Only write what fits
		remaining := lb.maxSize - len(lb.buf)
		if remaining > 0 {
			lb.buf = append(lb.buf, p[:remaining]...)
		}
		return len(p), nil
	}

	lb.buf = append(lb.buf, p...)
	return len(p), nil
}

func (lb *limitedBuffer) String() string {
	s := string(lb.buf)
	if lb.overflow {
		s += "\n... (output truncated)"
	}
	return s
}

// Close closes the executor and its resources
func (e *Executor) Close() error {
	return e.pool.Close()
}
