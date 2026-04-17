package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpcontroller "github.com/reignx/reignx/pkg/apiserver/http"
	"github.com/reignx/reignx/pkg/config"
	"github.com/reignx/reignx/pkg/crypto"
	"github.com/reignx/reignx/pkg/database"
	"github.com/reignx/reignx/pkg/database/repository"
	metricsPackage "github.com/reignx/reignx/pkg/observability/metrics"
	"github.com/reignx/reignx/pkg/web"
	"go.uber.org/zap"
)

func main() {
	// Parse command-line flags
	host := flag.String("host", "0.0.0.0", "Host to bind to")
	port := flag.Int("port", 8080, "Port to listen on")
	configPath := flag.String("config", "config/webserver.yaml", "Path to configuration file")
	flag.Parse()

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	logger.Info("Starting ReignX Web Server",
		zap.String("host", *host),
		zap.Int("port", *port),
	)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Warn("Failed to load config file, using defaults", zap.Error(err))
		cfg = &config.Config{
			Database: config.DatabaseConfig{
				Host:            "localhost",
				Port:            5432,
				User:            "bm_user",
				Password:        "bm_password",
				Database:        "bm_solution",
				SSLMode:         "disable",
				MaxOpenConns:    100,
				MaxIdleConns:    10,
				ConnMaxLifetime: 1 * time.Hour,
			},
		}
	}

	// Initialize metrics
	metrics := metricsPackage.NewMetrics("reignx")

	// Initialize database
	db, err := database.NewDatabase(cfg.Database, logger, metrics)
	if err != nil {
		logger.Warn("Failed to connect to database, running without persistence", zap.Error(err))
	} else {
		defer db.Close()
		logger.Info("Database connection established")
	}

	// Initialize encryption for sensitive data
	var encryptor *crypto.Encryptor
	encryptionKey := os.Getenv("REIGNX_ENCRYPTION_KEY")
	if encryptionKey == "" {
		encryptionKey = "IJBBBFbEhkpsRjPDt6V3cFG78dWEb86alrGQ8DGw+jc=" // Default for development
		logger.Warn("Using default encryption key - set REIGNX_ENCRYPTION_KEY in production")
	}
	encryptor, err = crypto.NewEncryptorFromString(encryptionKey)
	if err != nil {
		logger.Fatal("Failed to create encryptor", zap.Error(err))
	}

	// Initialize repositories
	var serverRepo repository.ServerRepository
	var userRepo repository.UserRepository
	var sshConfigRepo repository.SSHConfigRepository
	var jobRepo repository.JobRepository
	var taskRepo repository.TaskRepository
	if db != nil {
		serverRepo = repository.NewServerRepository(db.DB, logger, metrics)
		userRepo = repository.NewUserRepository(db.DB, logger, metrics)
		sshConfigRepo = repository.NewSSHConfigRepository(db.DB, logger, metrics, encryptor)
		jobRepo = repository.NewJobRepository(db.DB, logger, metrics)
		taskRepo = repository.NewTaskRepository(db.DB, logger, metrics)
	}

	// Create server controller
	var serverController *httpcontroller.ServerController
	if serverRepo != nil {
		serverController = httpcontroller.NewServerController(serverRepo, logger, metrics)
	}

	// Create web server configuration
	webConfig := &web.Config{
		Host:         *host,
		Port:         *port,
		SSHUsername:  "liqcui",
		SSHPassword:  "",  // Use SSH key authentication
		SSHKeyPath:   "",  // Use default SSH keys from ~/.ssh
		SSHTimeout:   30 * time.Second,
		JWTSecret:    "change-this-secret-in-production",
		AllowOrigins: []string{"http://localhost:3000", "http://localhost:8080", "http://localhost:9000"},
	}

	// Create web server
	server := web.NewServer(logger, webConfig, serverController, serverRepo, userRepo, sshConfigRepo, jobRepo, taskRepo)

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			logger.Fatal("Failed to start web server", zap.Error(err))
		}
	}()

	fmt.Printf("\n\u001b[32m✓\u001b[0m ReignX Web Server started\n")
	fmt.Printf("  \u001b[36m→\u001b[0m API: http://%s:%d\n", *host, *port)
	fmt.Printf("  \u001b[36m→\u001b[0m Login: admin / admin\n\n")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down web server...")

	// Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Web server shutdown error", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Web server stopped")
}
