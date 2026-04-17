package agentmode

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// GRPCPool manages a pool of gRPC connections
type GRPCPool struct {
	connections map[string]*pooledConnection
	maxSize     int
	mu          sync.RWMutex
}

// pooledConnection wraps a gRPC connection with metadata
type pooledConnection struct {
	conn      *grpc.ClientConn
	address   string
	lastUsed  time.Time
	refCount  int
	healthy   bool
}

// NewGRPCPool creates a new gRPC connection pool
func NewGRPCPool(maxSize int) *GRPCPool {
	pool := &GRPCPool{
		connections: make(map[string]*pooledConnection),
		maxSize:     maxSize,
	}

	// Start cleanup goroutine
	go pool.cleanup()

	return pool
}

// Get retrieves or creates a gRPC connection
func (p *GRPCPool) Get(address string, config *Config) (*grpc.ClientConn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if connection exists and is healthy
	if pc, exists := p.connections[address]; exists {
		if p.isHealthy(pc.conn) {
			pc.refCount++
			pc.lastUsed = time.Now()
			return pc.conn, nil
		}
		// Connection is unhealthy, remove it
		pc.conn.Close()
		delete(p.connections, address)
	}

	// Create new connection
	conn, err := p.createConnection(address, config)
	if err != nil {
		return nil, err
	}

	p.connections[address] = &pooledConnection{
		conn:     conn,
		address:  address,
		lastUsed: time.Now(),
		refCount: 1,
		healthy:  true,
	}

	return conn, nil
}

// Release decrements the reference count for a connection
func (p *GRPCPool) Release(address string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if pc, exists := p.connections[address]; exists {
		pc.refCount--
		pc.lastUsed = time.Now()
	}
}

// createConnection creates a new gRPC connection with proper configuration
func (p *GRPCPool) createConnection(address string, config *Config) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	// Configure TLS if enabled
	if config.TLSEnabled {
		tlsConfig, err := p.buildTLSConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS config: %w", err)
		}
		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Configure keepalive
	kaParams := keepalive.ClientParameters{
		Time:                10 * time.Second, // Send keepalive ping every 10s
		Timeout:             5 * time.Second,  // Wait 5s for ping ack
		PermitWithoutStream: true,             // Send pings even without active streams
	}
	opts = append(opts, grpc.WithKeepaliveParams(kaParams))

	// Configure default call options
	opts = append(opts,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(16*1024*1024), // 16MB
			grpc.MaxCallSendMsgSize(16*1024*1024), // 16MB
		),
	)

	// Create connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectionTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC: %w", err)
	}

	return conn, nil
}

// buildTLSConfig builds TLS configuration for secure connections
func (p *GRPCPool) buildTLSConfig(config *Config) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Load CA certificate if provided
	if config.CAFile != "" {
		caCert, err := ioutil.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append CA certificate")
		}

		tlsConfig.RootCAs = caCertPool
	}

	// Load client certificate if provided (for mTLS)
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// isHealthy checks if a gRPC connection is healthy
func (p *GRPCPool) isHealthy(conn *grpc.ClientConn) bool {
	if conn == nil {
		return false
	}

	state := conn.GetState()
	// Connection is healthy if ready or connecting
	return state == 0 || state == 1 || state == 2 // IDLE, CONNECTING, READY
}

// cleanup periodically removes idle connections
func (p *GRPCPool) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		now := time.Now()
		for address, pc := range p.connections {
			// Remove connections that are:
			// 1. Not in use (refCount == 0)
			// 2. Idle for more than 5 minutes
			// 3. Unhealthy
			if pc.refCount == 0 &&
			   (now.Sub(pc.lastUsed) > 5*time.Minute || !p.isHealthy(pc.conn)) {
				pc.conn.Close()
				delete(p.connections, address)
			}
		}
		p.mu.Unlock()
	}
}

// Close closes all connections in the pool
func (p *GRPCPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, pc := range p.connections {
		pc.conn.Close()
	}
	p.connections = make(map[string]*pooledConnection)

	return nil
}

// Size returns the current number of connections in the pool
func (p *GRPCPool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.connections)
}

// GetStats returns pool statistics
func (p *GRPCPool) GetStats() *PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := &PoolStats{
		TotalConnections: len(p.connections),
		ByState:          make(map[string]int),
	}

	for _, pc := range p.connections {
		stats.ActiveConnections += pc.refCount
		if pc.refCount > 0 {
			stats.InUse++
		} else {
			stats.Idle++
		}

		// Track connection state
		if pc.healthy {
			stats.Healthy++
		}
	}

	return stats
}

// PoolStats contains connection pool statistics
type PoolStats struct {
	TotalConnections  int
	ActiveConnections int
	InUse             int
	Idle              int
	Healthy           int
	ByState           map[string]int
}
