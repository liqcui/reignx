package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	agentpb "github.com/reignx/reignx/api/proto/gen"
	"github.com/reignx/reignx/pkg/config"
	"github.com/reignx/reignx/pkg/observability/metrics"
)

// Client wraps the gRPC client for agent-server communication
type Client struct {
	conn          *grpc.ClientConn
	client        agentpb.AgentServiceClient
	config        config.AgentConfig
	securityCfg   config.SecurityConfig
	logger        *zap.Logger
	metrics       *metrics.Metrics
	retryInterval time.Duration
	maxRetries    int
}

// NewClient creates a new agent gRPC client
func NewClient(agentCfg config.AgentConfig, securityCfg config.SecurityConfig, logger *zap.Logger, m *metrics.Metrics) *Client {
	return &Client{
		config:        agentCfg,
		securityCfg:   securityCfg,
		logger:        logger,
		metrics:       m,
		retryInterval: 5 * time.Second,
		maxRetries:    10,
	}
}

// Connect establishes a connection to the API server
func (c *Client) Connect(ctx context.Context) error {
	var opts []grpc.DialOption

	// Configure TLS credentials
	if c.securityCfg.TLSEnabled {
		creds, err := c.loadTLSCredentials()
		if err != nil {
			return fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
		c.logger.Info("TLS enabled for agent client",
			zap.Bool("mtls", c.securityCfg.EnableMTLS),
		)
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		c.logger.Warn("TLS disabled - using insecure connection")
	}

	// Configure keepalive
	opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                30 * time.Second,
		Timeout:             10 * time.Second,
		PermitWithoutStream: true,
	}))

	// Retry connection with exponential backoff
	var conn *grpc.ClientConn
	var err error

	backoff := c.retryInterval
	maxBackoff := 30 * time.Second

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		c.logger.Info("Attempting to connect to API server",
			zap.String("address", c.config.APIServerAddr),
			zap.Int("attempt", attempt+1),
			zap.Int("max_retries", c.maxRetries),
		)

		conn, err = grpc.NewClient(c.config.APIServerAddr, opts...)
		if err == nil {
			c.conn = conn
			c.client = agentpb.NewAgentServiceClient(conn)
			c.logger.Info("Successfully connected to API server",
				zap.String("address", c.config.APIServerAddr),
			)
			return nil
		}

		c.logger.Warn("Failed to connect to API server, retrying",
			zap.Error(err),
			zap.Duration("backoff", backoff),
		)

		select {
		case <-ctx.Done():
			return fmt.Errorf("connection cancelled: %w", ctx.Err())
		case <-time.After(backoff):
			// Exponential backoff
			backoff = backoff * 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	return fmt.Errorf("failed to connect after %d attempts: %w", c.maxRetries, err)
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		c.logger.Info("Closing gRPC connection")
		return c.conn.Close()
	}
	return nil
}

// RegisterAgent calls the RegisterAgent RPC
func (c *Client) RegisterAgent(ctx context.Context, info *agentpb.AgentInfo) (*agentpb.RegistrationResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}

	c.logger.Info("Registering agent with server",
		zap.String("hostname", info.Hostname),
		zap.String("ip_address", info.IpAddress),
	)

	resp, err := c.client.RegisterAgent(ctx, info)
	if err != nil {
		c.logger.Error("Failed to register agent", zap.Error(err))
		if c.metrics != nil {
			c.metrics.RecordCounter("agent_registration_errors_total", 1, nil)
		}
		return nil, fmt.Errorf("registration failed: %w", err)
	}

	if c.metrics != nil {
		c.metrics.RecordCounter("agent_registrations_total", 1, nil)
	}

	c.logger.Info("Agent registered successfully",
		zap.String("server_id", resp.ServerId),
		zap.Int32("heartbeat_interval", resp.HeartbeatIntervalSeconds),
	)

	return resp, nil
}

// Heartbeat calls the Heartbeat RPC
func (c *Client) Heartbeat(ctx context.Context, req *agentpb.HeartbeatRequest) (*agentpb.HeartbeatResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}

	resp, err := c.client.Heartbeat(ctx, req)
	if err != nil {
		c.logger.Error("Failed to send heartbeat",
			zap.String("server_id", req.ServerId),
			zap.Error(err),
		)
		if c.metrics != nil {
			c.metrics.RecordCounter("agent_heartbeat_errors_total", 1, nil)
		}
		return nil, fmt.Errorf("heartbeat failed: %w", err)
	}

	return resp, nil
}

// loadTLSCredentials loads TLS credentials for the client
func (c *Client) loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load CA certificate
	caBytes, err := os.ReadFile(c.securityCfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caBytes) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	config := &tls.Config{
		RootCAs:    certPool,
		MinVersion: tls.VersionTLS12,
	}

	// Load client certificate for mTLS if enabled
	if c.securityCfg.EnableMTLS {
		cert, err := tls.LoadX509KeyPair(c.securityCfg.CertFile, c.securityCfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		config.Certificates = []tls.Certificate{cert}
	}

	return credentials.NewTLS(config), nil
}
