package sshmode

import (
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/reignx/reignx/pkg/core"
	"golang.org/x/crypto/ssh"
)

// ConnectionPool manages SSH connections
type ConnectionPool struct {
	connections map[string]*pooledConnection
	maxSize     int
	mu          sync.RWMutex
}

// pooledConnection wraps an SSH client with metadata
type pooledConnection struct {
	client    *ssh.Client
	lastUsed  time.Time
	inUse     bool
	config    *core.SSHConfig
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxSize int) *ConnectionPool {
	pool := &ConnectionPool{
		connections: make(map[string]*pooledConnection),
		maxSize:     maxSize,
	}

	// Start cleanup goroutine
	go pool.cleanup()

	return pool
}

// Get retrieves or creates an SSH connection
func (p *ConnectionPool) Get(nodeID string, sshConfig *core.SSHConfig) (*ssh.Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if connection exists and is valid
	if conn, exists := p.connections[nodeID]; exists {
		if !conn.inUse && p.isHealthy(conn.client) {
			conn.inUse = true
			conn.lastUsed = time.Now()
			return conn.client, nil
		}
		// Connection is unhealthy or in use, remove it
		conn.client.Close()
		delete(p.connections, nodeID)
	}

	// Create new connection
	client, err := p.createConnection(sshConfig)
	if err != nil {
		return nil, err
	}

	p.connections[nodeID] = &pooledConnection{
		client:   client,
		lastUsed: time.Now(),
		inUse:    true,
		config:   sshConfig,
	}

	return client, nil
}

// Release marks a connection as not in use
func (p *ConnectionPool) Release(nodeID string, client *ssh.Client) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, exists := p.connections[nodeID]; exists && conn.client == client {
		conn.inUse = false
		conn.lastUsed = time.Now()
	}
}

// createConnection creates a new SSH connection
func (p *ConnectionPool) createConnection(config *core.SSHConfig) (*ssh.Client, error) {
	sshClientConfig, err := p.buildSSHConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build SSH config: %w", err)
	}

	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	if config.Port == 0 {
		address = fmt.Sprintf("%s:22", config.Host)
	}

	// Handle jump host if configured
	if config.JumpHost != "" {
		return p.createConnectionViaJumpHost(address, config.JumpHost, sshClientConfig)
	}

	// Direct connection
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	client, err := ssh.Dial("tcp", address, sshClientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH: %w", err)
	}

	return client, nil
}

// buildSSHConfig builds an ssh.ClientConfig from core.SSHConfig
func (p *ConnectionPool) buildSSHConfig(config *core.SSHConfig) (*ssh.ClientConfig, error) {
	authMethods := []ssh.AuthMethod{}

	// Add key authentication if configured
	if config.KeyPath != "" {
		key, err := ioutil.ReadFile(config.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Add password authentication if configured
	if config.Password != "" {
		// TODO: Decrypt password
		authMethods = append(authMethods, ssh.Password(config.Password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method configured")
	}

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &ssh.ClientConfig{
		User:            config.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Implement proper host key verification
		Timeout:         timeout,
	}, nil
}

// createConnectionViaJumpHost creates an SSH connection through a jump host
func (p *ConnectionPool) createConnectionViaJumpHost(target, jumpHost string, config *ssh.ClientConfig) (*ssh.Client, error) {
	// Connect to jump host
	jumpClient, err := ssh.Dial("tcp", jumpHost, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to jump host: %w", err)
	}

	// Connect to target through jump host
	conn, err := jumpClient.Dial("tcp", target)
	if err != nil {
		jumpClient.Close()
		return nil, fmt.Errorf("failed to dial target through jump host: %w", err)
	}

	ncc, chans, reqs, err := ssh.NewClientConn(conn, target, config)
	if err != nil {
		conn.Close()
		jumpClient.Close()
		return nil, fmt.Errorf("failed to create client connection: %w", err)
	}

	return ssh.NewClient(ncc, chans, reqs), nil
}

// isHealthy checks if an SSH connection is still healthy
func (p *ConnectionPool) isHealthy(client *ssh.Client) bool {
	if client == nil {
		return false
	}

	// Try to create a session to check if connection is alive
	session, err := client.NewSession()
	if err != nil {
		return false
	}
	session.Close()

	return true
}

// cleanup periodically removes idle connections
func (p *ConnectionPool) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		now := time.Now()
		for nodeID, conn := range p.connections {
			// Remove connections idle for more than 5 minutes
			if !conn.inUse && now.Sub(conn.lastUsed) > 5*time.Minute {
				conn.client.Close()
				delete(p.connections, nodeID)
			}
		}
		p.mu.Unlock()
	}
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.connections {
		conn.client.Close()
	}
	p.connections = make(map[string]*pooledConnection)

	return nil
}

// Size returns the current size of the pool
func (p *ConnectionPool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.connections)
}
