package web

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	httpcontroller "github.com/reignx/reignx/pkg/apiserver/http"
	"github.com/reignx/reignx/pkg/auth"
	"github.com/reignx/reignx/pkg/database/repository"
)

//go:embed static
var frontendFS embed.FS

// Server represents the web server
type Server struct {
	router           *gin.Engine
	logger           *zap.Logger
	terminalHandler  *TerminalHandler
	sshExecutor      *SSHExecutor
	jobScheduler     *JobScheduler
	httpServer       *http.Server
	serverController *httpcontroller.ServerController
	jwtManager       *auth.JWTManager
	userRepo         repository.UserRepository
	sshConfigRepo    repository.SSHConfigRepository
	jobRepo          repository.JobRepository
	taskRepo         repository.TaskRepository
}

// Config holds web server configuration
type Config struct {
	Host         string
	Port         int
	SSHUsername  string
	SSHPassword  string
	SSHKeyPath   string
	SSHTimeout   time.Duration
	JWTSecret    string
	AllowOrigins []string
}

// NewServer creates a new web server instance
func NewServer(logger *zap.Logger, config *Config, serverController *httpcontroller.ServerController, serverRepo repository.ServerRepository, userRepo repository.UserRepository, sshConfigRepo repository.SSHConfigRepository, jobRepo repository.JobRepository, taskRepo repository.TaskRepository) *Server {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(LoggerMiddleware(logger))
	router.Use(CORSMiddleware(config.AllowOrigins))

	// Create JWT manager
	jwtManager := auth.NewJWTManager(config.JWTSecret)

	// Create SSH configuration
	sshConfig := &SSHConfig{
		Username: config.SSHUsername,
		Password: config.SSHPassword,
		KeyPath:  config.SSHKeyPath,
		Timeout:  config.SSHTimeout,
	}

	// Create terminal handler
	terminalHandler := NewTerminalHandler(logger, sshConfig, serverRepo, sshConfigRepo)

	// Create SSH executor for command execution
	sshExecutor := NewSSHExecutor(logger, sshConfig, serverRepo, sshConfigRepo)

	// Create job scheduler
	var jobScheduler *JobScheduler
	if jobRepo != nil && taskRepo != nil {
		jobScheduler = NewJobScheduler(jobRepo, taskRepo, serverRepo, sshExecutor, logger)
	}

	server := &Server{
		router:           router,
		logger:           logger,
		terminalHandler:  terminalHandler,
		sshExecutor:      sshExecutor,
		jobScheduler:     jobScheduler,
		serverController: serverController,
		jwtManager:       jwtManager,
		userRepo:         userRepo,
		sshConfigRepo:    sshConfigRepo,
		jobRepo:          jobRepo,
		taskRepo:         taskRepo,
	}

	// Setup routes
	server.setupRoutes()

	// Create HTTP server
	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// WebSocket routes
	s.router.GET("/ws/terminal/:serverID", func(c *gin.Context) {
		s.terminalHandler.HandleWebSocket(c.Writer, c.Request)
	})

	// API routes
	api := s.router.Group("/api/v1")
	{
		// Authentication
		api.POST("/auth/login", s.handleLogin)
		api.POST("/auth/refresh", s.handleRefreshToken)

		// Protected auth routes
		authProtected := api.Group("/auth")
		authProtected.Use(s.AuthMiddleware())
		authProtected.POST("/logout", s.handleLogout)

		// Protected routes
		protected := api.Group("")
		protected.Use(s.AuthMiddleware())
		{
			// Servers
			protected.GET("/servers", s.handleListServers)
			protected.GET("/servers/:id", s.handleGetServer)
			protected.POST("/servers", s.handleCreateServer)
			protected.PUT("/servers/:id", s.handleUpdateServer)
			protected.DELETE("/servers/:id", s.handleDeleteServer)

			// Server Actions
			protected.POST("/servers/:id/power/:action", s.handleServerPowerAction)
			protected.POST("/servers/:id/execute", s.handleExecuteCommand)
			protected.POST("/servers/:id/upload", s.handleUploadFile)
			protected.POST("/servers/:id/deploy", s.handleDeployPackage) // Deprecated, use /upload
			protected.POST("/servers/:id/install-os", s.handleInstallOS)

			// Jobs
			protected.GET("/jobs", s.handleListJobs)
			protected.GET("/jobs/:id", s.handleGetJob)
			protected.POST("/jobs", s.handleCreateJob)
			protected.DELETE("/jobs/:id", s.handleCancelJob)

			// Tasks
			protected.GET("/tasks", s.handleListTasks)
			protected.GET("/tasks/:id", s.handleGetTask)
			protected.POST("/tasks/:id/retry", s.handleRetryTask)

			// Metrics
			protected.GET("/metrics/dashboard", s.handleGetDashboardMetrics)
		}
	}

	// Serve frontend static files
	s.setupFrontend()

	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
}

// setupFrontend configures serving of frontend static files
func (s *Server) setupFrontend() {
	// Get embedded frontend files
	distFS, err := fs.Sub(frontendFS, "static")
	if err != nil {
		s.logger.Warn("Frontend files not found, skipping static file serving", zap.Error(err))
		return
	}

	// Serve static files
	assetsFS, err := fs.Sub(distFS, "assets")
	if err != nil {
		s.logger.Warn("Assets directory not found", zap.Error(err))
	} else {
		s.router.StaticFS("/assets", http.FS(assetsFS))
	}

	// Serve index.html for all non-API routes (SPA routing)
	s.router.NoRoute(func(c *gin.Context) {
		// Don't serve index.html for API routes
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}

		// Don't serve index.html for WebSocket routes
		if len(c.Request.URL.Path) >= 3 && c.Request.URL.Path[:3] == "/ws" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}

		// Serve index.html
		data, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			s.logger.Error("Failed to read index.html", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
}

// Start starts the web server
func (s *Server) Start() error {
	s.logger.Info("Starting web server", zap.String("addr", s.httpServer.Addr))
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the web server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down web server")
	return s.httpServer.Shutdown(ctx)
}

// LoggerMiddleware is a Gin middleware for logging
func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)

		logger.Info("HTTP request",
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
			zap.String("user-agent", c.Request.UserAgent()),
		)
	}
}

// CORSMiddleware is a Gin middleware for CORS
func CORSMiddleware(allowOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

