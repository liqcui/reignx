package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/reignx/reignx/pkg/agentmode"
	"github.com/reignx/reignx/pkg/database"
	"github.com/reignx/reignx/pkg/sshmode"
	"github.com/reignx/reignx/reignxd/internal/api"
	"github.com/reignx/reignx/reignxd/internal/grpcserver"
	"github.com/reignx/reignx/reignxd/internal/scheduler"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

type Server struct {
	logger        *zap.Logger
	db            *database.DB
	grpcServer    *grpc.Server
	httpServer    *http.Server
	scheduler     *scheduler.Scheduler
	sshExecutor   *sshmode.Executor
	agentExecutor *agentmode.Executor
}

func main() {
	// Parse flags
	_ = flag.String("config", "config/reignxd.yaml", "Configuration file path")
	httpPort := flag.Int("http-port", 8080, "HTTP API port")
	grpcPort := flag.Int("grpc-port", 50051, "gRPC port")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("ReignX Daemon (reignxd)\n")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting ReignX Daemon",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
		zap.String("git_commit", GitCommit),
	)

	// Initialize database
	logger.Info("Connecting to database...")
	db, err := database.New(database.DefaultConfig())
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Verify database health
	if err := db.Health(context.Background()); err != nil {
		logger.Fatal("Database health check failed", zap.Error(err))
	}
	logger.Info("Database connected successfully")

	// Initialize executors
	sshExecutor := sshmode.NewExecutor(sshmode.DefaultConfig())
	agentExecutor := agentmode.NewExecutor(agentmode.DefaultConfig())
	defer sshExecutor.Close()
	defer agentExecutor.Close()

	// Initialize repositories
	nodeRepo := database.NewNodeRepository(db)
	jobRepo := database.NewJobRepository(db)
	taskRepo := database.NewTaskRepository(db)

	// Initialize scheduler
	logger.Info("Initializing scheduler...")
	schedulerInstance := scheduler.New(&scheduler.Config{
		Logger:        logger,
		JobRepo:       jobRepo,
		TaskRepo:      taskRepo,
		NodeRepo:      nodeRepo,
		SSHExecutor:   sshExecutor,
		AgentExecutor: agentExecutor,
		PollInterval:  5 * time.Second,
	})

	// Initialize gRPC server
	logger.Info("Starting gRPC server", zap.Int("port", *grpcPort))
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
	if err != nil {
		logger.Fatal("Failed to listen on gRPC port", zap.Error(err))
	}

	grpcSrv := grpc.NewServer(
		grpc.MaxRecvMsgSize(16*1024*1024),
		grpc.MaxSendMsgSize(16*1024*1024),
	)

	grpcserver.Register(grpcSrv, &grpcserver.Config{
		Logger:   logger,
		NodeRepo: nodeRepo,
		TaskRepo: taskRepo,
	})

	// Start gRPC server
	go func() {
		if err := grpcSrv.Serve(grpcListener); err != nil {
			logger.Error("gRPC server error", zap.Error(err))
		}
	}()
	logger.Info("gRPC server started", zap.Int("port", *grpcPort))

	// Initialize HTTP API server
	logger.Info("Starting HTTP API server", zap.Int("port", *httpPort))
	apiHandler := api.NewHandler(&api.Config{
		Logger:   logger,
		NodeRepo: nodeRepo,
		JobRepo:  jobRepo,
		TaskRepo: taskRepo,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", *httpPort),
		Handler:      apiHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", zap.Error(err))
		}
	}()
	logger.Info("HTTP API server started", zap.Int("port", *httpPort))

	// Start scheduler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := schedulerInstance.Start(ctx); err != nil {
			logger.Error("Scheduler error", zap.Error(err))
		}
	}()
	logger.Info("Scheduler started")

	// Wait for shutdown signal
	logger.Info("ReignX Daemon is running")
	logger.Info("Press Ctrl+C to shutdown")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutdown signal received, stopping gracefully...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop scheduler
	cancel()

	// Stop HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	// Stop gRPC server
	grpcSrv.GracefulStop()

	logger.Info("ReignX Daemon stopped")
}
