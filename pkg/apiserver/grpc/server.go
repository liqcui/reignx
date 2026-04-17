package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	agentpb "github.com/reignx/reignx/api/proto/gen"
	"github.com/reignx/reignx/pkg/config"
	"github.com/reignx/reignx/pkg/observability/metrics"
	"github.com/reignx/reignx/pkg/observability/middleware"
)

// Server wraps the gRPC server with configuration
type Server struct {
	config       *config.Config
	grpcServer   *grpc.Server
	agentService agentpb.AgentServiceServer
	logger       *zap.Logger
	metrics      *metrics.Metrics
	listener     net.Listener
}

// NewServer creates a new gRPC server
func NewServer(cfg *config.Config, agentSvc agentpb.AgentServiceServer, logger *zap.Logger, m *metrics.Metrics) (*Server, error) {
	var opts []grpc.ServerOption

	// Configure TLS if enabled
	if cfg.Security.TLSEnabled {
		tlsConfig, err := loadTLSConfig(&cfg.Security)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS config: %w", err)
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))
		logger.Info("TLS enabled for gRPC server",
			zap.Bool("mtls", cfg.Security.EnableMTLS),
		)
	}

	// Configure keepalive
	opts = append(opts, grpc.KeepaliveParams(keepalive.ServerParameters{
		Time:    30 * time.Second,
		Timeout: 10 * time.Second,
	}))

	// Configure message size limits
	opts = append(opts,
		grpc.MaxRecvMsgSize(cfg.Server.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.Server.MaxSendMsgSize),
	)

	// Add interceptors
	opts = append(opts,
		grpc.ChainUnaryInterceptor(
			middleware.UnaryServerInterceptor(m),
		),
		grpc.ChainStreamInterceptor(
			middleware.StreamServerInterceptor(m),
		),
	)

	// Create gRPC server
	grpcServer := grpc.NewServer(opts...)

	// Register services
	agentpb.RegisterAgentServiceServer(grpcServer, agentSvc)

	return &Server{
		config:       cfg,
		grpcServer:   grpcServer,
		agentService: agentSvc,
		logger:       logger,
		metrics:      m,
	}, nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.GRPCPort)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.listener = listener

	s.logger.Info("Starting gRPC server",
		zap.String("address", addr),
		zap.Int("port", s.config.Server.GRPCPort),
	)

	if err := s.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the gRPC server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down gRPC server")

	// Create a channel to signal when graceful stop is complete
	done := make(chan struct{})

	go func() {
		s.grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-ctx.Done():
		s.logger.Warn("Graceful shutdown timeout, forcing stop")
		s.grpcServer.Stop()
		return ctx.Err()
	case <-done:
		s.logger.Info("gRPC server shut down gracefully")
		return nil
	}
}

// loadTLSConfig loads TLS configuration from files
func loadTLSConfig(cfg *config.SecurityConfig) (*tls.Config, error) {
	// Load server certificate and key
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// Configure mTLS if enabled
	if cfg.EnableMTLS && cfg.CAFile != "" {
		caBytes, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caBytes) {
			return nil, fmt.Errorf("failed to append CA certificates")
		}

		tlsConfig.ClientCAs = certPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig, nil
}
