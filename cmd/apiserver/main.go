package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"time"

	"github.com/gin-gonic/gin"

	grpcpkg "github.com/reignx/reignx/pkg/apiserver/grpc"
	httpcontroller "github.com/reignx/reignx/pkg/apiserver/http"
	"github.com/reignx/reignx/pkg/config"
	"github.com/reignx/reignx/pkg/database"
	"github.com/reignx/reignx/pkg/database/repository"
	metricsPackage "github.com/reignx/reignx/pkg/observability/metrics"
	"go.uber.org/zap"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "config/apiserver.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("BM Solution API Server\n")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := initLogger(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting BM Solution API Server",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
		zap.String("git_commit", GitCommit),
	)

	// Create context that listens for termination signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("Received termination signal", zap.String("signal", sig.String()))
		cancel()
	}()

	// Initialize metrics
	metrics := metricsPackage.NewMetrics("reignx")

	// Initialize database
	logger.Info("Database configuration",
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("user", cfg.Database.User),
		zap.String("database", cfg.Database.Database),
		zap.String("dsn", cfg.Database.GetDSN()))
	db, err := database.NewDatabase(cfg.Database, logger, metrics)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	logger.Info("Database connection established")

	// Initialize repository
	serverRepo := repository.NewServerRepository(db.DB, logger, metrics)

	// Initialize gRPC service
	agentSvc := grpcpkg.NewAgentService(serverRepo, logger, metrics)

	// Create gRPC server
	grpcServer, err := grpcpkg.NewServer(cfg, agentSvc, logger, metrics)
	if err != nil {
		logger.Fatal("Failed to create gRPC server", zap.Error(err))
	}

	// Start gRPC server in goroutine
	go func() {
		if err := grpcServer.Start(); err != nil {
			logger.Error("gRPC server error", zap.Error(err))
			cancel()
		}
	}()

	// Start heartbeat monitor in goroutine
	monitor := grpcpkg.NewHeartbeatMonitor(serverRepo, logger, metrics, cfg)
	go monitor.Run(ctx)

	// Create HTTP server controller
	serverController := httpcontroller.NewServerController(serverRepo, logger, metrics)

	// Start HTTP server for REST API
	gin.SetMode(gin.ReleaseMode)
	httpRouter := gin.New()
	httpRouter.Use(gin.Recovery())

	// API routes
	api := httpRouter.Group("/api/v1")
	{
		api.GET("/servers", serverController.ListServers)
		api.GET("/servers/:id", serverController.GetServer)
		api.GET("/servers/by-hostname/:hostname", serverController.GetServerByHostname)
		api.GET("/stats", serverController.GetStats)
		api.GET("/health", serverController.GetHealth)
	}

	// Start HTTP server in goroutine
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
		logger.Info("Starting HTTP server", zap.String("address", addr))
		if err := httpRouter.Run(addr); err != nil {
			logger.Error("HTTP server error", zap.Error(err))
			cancel()
		}
	}()

	logger.Info("API Server started successfully",
		zap.Int("grpc_port", cfg.Server.GRPCPort),
		zap.Int("http_port", cfg.Server.HTTPPort),
		zap.Int("metrics_port", cfg.Server.MetricsPort),
	)

	// Wait for context cancellation
	<-ctx.Done()

	logger.Info("Shutting down API Server gracefully...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown gRPC server
	if err := grpcServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown gRPC server gracefully", zap.Error(err))
	}


	logger.Info("API Server stopped")
}

func initLogger(cfg *config.Config) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	if cfg.Observability.Logging.Format == "json" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}

	if err != nil {
		return nil, err
	}

	return logger, nil
}
