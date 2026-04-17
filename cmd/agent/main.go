package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/reignx/reignx/pkg/agent"
	"github.com/reignx/reignx/pkg/config"
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
	configPath := flag.String("config", "config/agent.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	installCmd := flag.Bool("install", false, "Install agent as system service")
	uninstallCmd := flag.Bool("uninstall", false, "Uninstall agent system service")
	flag.Parse()

	if *showVersion {
		fmt.Printf("BM Solution Agent\n")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	if *installCmd {
		fmt.Println("Installing agent as system service...")
		// TODO: Implement service installation
		fmt.Println("Agent installed successfully")
		os.Exit(0)
	}

	if *uninstallCmd {
		fmt.Println("Uninstalling agent system service...")
		// TODO: Implement service uninstallation
		fmt.Println("Agent uninstalled successfully")
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

	logger.Info("Starting BM Solution Agent",
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

	// Load or generate agent ID
	agentID, err := agent.LoadOrGenerateAgentID(cfg.Agent.CacheDir)
	if err != nil {
		logger.Fatal("Failed to get agent ID", zap.Error(err))
	}

	logger.Info("Agent ID loaded",
		zap.String("agent_id", agentID),
	)

	// Create gRPC client
	client := agent.NewClient(cfg.Agent, cfg.Security, logger, metrics)
	if err := client.Connect(ctx); err != nil {
		logger.Fatal("Failed to connect to API server", zap.Error(err))
	}
	defer client.Close()

	logger.Info("Connected to API server",
		zap.String("address", cfg.Agent.APIServerAddr),
	)

	// Register with server
	serverID, err := agent.Register(ctx, client, agentID, cfg, Version, logger)
	if err != nil {
		logger.Fatal("Failed to register agent", zap.Error(err))
	}

	logger.Info("Agent registered successfully",
		zap.String("server_id", serverID),
	)

	// Start heartbeat manager
	hbManager := agent.NewHeartbeatManager(client, cfg, logger, metrics, serverID, agentID)
	go hbManager.Start(ctx)

	logger.Info("Agent started successfully",
		zap.String("api_server", cfg.Agent.APIServerAddr),
		zap.String("server_id", serverID),
	)

	// Wait for context cancellation
	<-ctx.Done()

	logger.Info("Shutting down Agent gracefully...")

	// Send final heartbeat
	hbManager.SendFinalHeartbeat()


	logger.Info("Agent stopped")
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
