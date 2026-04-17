package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/reignx/reignx/reignx-agent/internal/agent"
	"go.uber.org/zap"
)

var (
	version   = "0.1.0"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Command line flags
	var (
		_              = flag.String("config", "/etc/reignx/agent.yaml", "Path to configuration file") // reserved for future use
		serverAddr     = flag.String("server", "localhost:50051", "ReignX server address")
		tlsEnabled     = flag.Bool("tls", false, "Enable TLS")
		certFile       = flag.String("cert", "/etc/reignx/agent.crt", "Client certificate file")
		keyFile        = flag.String("key", "/etc/reignx/agent.key", "Client key file")
		caFile         = flag.String("ca", "/etc/reignx/ca.crt", "CA certificate file")
		heartbeatSec   = flag.Int("heartbeat", 30, "Heartbeat interval in seconds")
		logLevel       = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
		cacheDir       = flag.String("cache-dir", "/var/lib/reignx-agent", "Cache directory")
		maxConcurrency = flag.Int("concurrency", 5, "Maximum concurrent tasks")
		showVersion    = flag.Bool("version", false, "Show version information")
		install        = flag.Bool("install", false, "Install agent as systemd service")
		uninstall      = flag.Bool("uninstall", false, "Uninstall agent systemd service")
	)

	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("ReignX Agent\n")
		fmt.Printf("Version:    %s\n", version)
		fmt.Printf("Build Time: %s\n", buildTime)
		fmt.Printf("Git Commit: %s\n", gitCommit)
		os.Exit(0)
	}

	// Handle install/uninstall
	if *install {
		if err := installService(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to install service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Agent service installed successfully")
		os.Exit(0)
	}

	if *uninstall {
		if err := uninstallService(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to uninstall service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Agent service uninstalled successfully")
		os.Exit(0)
	}

	// Initialize logger
	logger, err := initLogger(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting ReignX Agent",
		zap.String("version", version),
		zap.String("server", *serverAddr),
		zap.Bool("tls_enabled", *tlsEnabled),
	)

	// Create agent configuration
	config := &agent.Config{
		ServerAddr:        *serverAddr,
		TLSEnabled:        *tlsEnabled,
		CertFile:          *certFile,
		KeyFile:           *keyFile,
		CAFile:            *caFile,
		HeartbeatInterval: time.Duration(*heartbeatSec) * time.Second,
		CacheDir:          *cacheDir,
		MaxConcurrency:    *maxConcurrency,
		Logger:            logger,
	}

	// Create and start agent
	ag, err := agent.New(config)
	if err != nil {
		logger.Fatal("Failed to create agent", zap.Error(err))
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start agent in background
	errChan := make(chan error, 1)
	go func() {
		if err := ag.Start(ctx); err != nil {
			errChan <- err
		}
	}()

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
		cancel()

		// Wait for agent to stop gracefully (max 30 seconds)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := ag.Stop(shutdownCtx); err != nil {
			logger.Error("Error during shutdown", zap.Error(err))
		}

		logger.Info("Agent stopped gracefully")

	case err := <-errChan:
		logger.Fatal("Agent error", zap.Error(err))
	}
}

// initLogger initializes the logger based on log level
func initLogger(level string) (*zap.Logger, error) {
	var config zap.Config

	if level == "debug" {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	// Set log level
	switch level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	return config.Build()
}

// installService installs the agent as a systemd service
func installService() error {
	serviceContent := `[Unit]
Description=ReignX Agent
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/reignx-agent
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
`

	// Write service file
	if err := os.WriteFile("/etc/systemd/system/reignx-agent.service", []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Reload systemd
	return nil
}

// uninstallService removes the agent systemd service
func uninstallService() error {
	// Remove service file
	if err := os.Remove("/etc/systemd/system/reignx-agent.service"); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove service file: %w", err)
		}
	}

	return nil
}
