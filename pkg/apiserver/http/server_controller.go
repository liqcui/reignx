package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/reignx/reignx/pkg/database/models"
	"github.com/reignx/reignx/pkg/database/repository"
	"github.com/reignx/reignx/pkg/observability/metrics"
)

// ServerController handles HTTP requests for server management
type ServerController struct {
	repo    repository.ServerRepository
	logger  *zap.Logger
	metrics *metrics.Metrics
}

// NewServerController creates a new server controller
func NewServerController(repo repository.ServerRepository, logger *zap.Logger, m *metrics.Metrics) *ServerController {
	return &ServerController{
		repo:    repo,
		logger:  logger,
		metrics: m,
	}
}

// ServerResponse represents a server in API responses
type ServerResponse struct {
	ID            string            `json:"id"`
	Hostname      string            `json:"hostname"`
	IPAddress     string            `json:"ip"`
	MACAddress    string            `json:"mac_address,omitempty"`
	OSType        string            `json:"os"`
	OSVersion     string            `json:"os_version,omitempty"`
	Architecture  string            `json:"architecture,omitempty"`
	Status        string            `json:"status"`
	Mode          string            `json:"mode"`
	AgentVersion  string            `json:"agent_version,omitempty"`
	LastSeen      string            `json:"last_seen"`
	CPUUsage      *float64          `json:"cpu_usage,omitempty"`
	MemoryUsage   *float64          `json:"memory_usage,omitempty"`
	DiskUsage     *float64          `json:"disk_usage,omitempty"`
	Uptime        *int64            `json:"uptime,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// ListServersResponse represents the response for listing servers
type ListServersResponse struct {
	Servers []ServerResponse `json:"servers"`
	Total   int              `json:"total"`
}

// StatsResponse represents cluster statistics
type StatsResponse struct {
	Total     int            `json:"total"`
	Online    int            `json:"online"`
	Offline   int            `json:"offline"`
	ByStatus  map[string]int `json:"by_status"`
	ByOSType  map[string]int `json:"by_os_type"`
	ByMode    map[string]int `json:"by_mode"`
}

// ListServers handles GET /api/v1/servers
func (sc *ServerController) ListServers(c *gin.Context) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if sc.metrics != nil {
			sc.metrics.RecordHistogram("http_request_duration_seconds", duration.Seconds(), map[string]string{
				"method": "GET",
				"path":   "/api/v1/servers",
			})
		}
	}()

	// Parse query parameters
	status := c.Query("status")

	// Build filters
	filters := make(map[string]interface{})
	if status != "" {
		filters["status"] = status
	}

	// Query database
	servers, err := sc.repo.List(c.Request.Context(), filters)
	if err != nil {
		sc.logger.Error("Failed to list servers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve servers",
		})
		return
	}

	// Convert to response format
	response := ListServersResponse{
		Servers: make([]ServerResponse, 0, len(servers)),
		Total:   len(servers),
	}

	for _, server := range servers {
		response.Servers = append(response.Servers, sc.convertToResponse(server))
	}

	c.JSON(http.StatusOK, response)
}

// GetServer handles GET /api/v1/servers/:id
func (sc *ServerController) GetServer(c *gin.Context) {
	serverID := c.Param("id")

	// Query database
	server, err := sc.repo.GetByID(c.Request.Context(), serverID)
	if err != nil {
		sc.logger.Error("Failed to get server",
			zap.String("server_id", serverID),
			zap.Error(err),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Server not found",
		})
		return
	}

	c.JSON(http.StatusOK, sc.convertToResponse(server))
}

// GetServerByHostname handles GET /api/v1/servers/by-hostname/:hostname
func (sc *ServerController) GetServerByHostname(c *gin.Context) {
	hostname := c.Param("hostname")

	// Query database
	server, err := sc.repo.GetByHostname(c.Request.Context(), hostname)
	if err != nil {
		sc.logger.Error("Failed to get server by hostname",
			zap.String("hostname", hostname),
			zap.Error(err),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Server not found",
		})
		return
	}

	c.JSON(http.StatusOK, sc.convertToResponse(server))
}

// GetStats handles GET /api/v1/stats
func (sc *ServerController) GetStats(c *gin.Context) {
	// Query all servers
	servers, err := sc.repo.List(c.Request.Context(), nil)
	if err != nil {
		sc.logger.Error("Failed to get stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve statistics",
		})
		return
	}

	// Calculate statistics
	stats := StatsResponse{
		Total:    len(servers),
		ByStatus: make(map[string]int),
		ByOSType: make(map[string]int),
		ByMode:   make(map[string]int),
	}

	for _, server := range servers {
		// Count by status
		stats.ByStatus[server.Status]++
		if server.Status == string(models.ServerStatusOnline) {
			stats.Online++
		} else if server.Status == string(models.ServerStatusOffline) {
			stats.Offline++
		}

		// Count by OS type
		stats.ByOSType[server.OSType]++

		// Count by mode
		stats.ByMode[server.Mode]++
	}

	c.JSON(http.StatusOK, stats)
}

// GetHealth handles GET /api/v1/health
func (sc *ServerController) GetHealth(c *gin.Context) {
	// Simple health check
	servers, err := sc.repo.List(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "Database connection failed",
		})
		return
	}

	online := 0
	for _, server := range servers {
		if server.Status == string(models.ServerStatusOnline) {
			online++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "healthy",
		"total_servers":  len(servers),
		"online_servers": online,
		"timestamp":      time.Now().Format(time.RFC3339),
	})
}

// convertToResponse converts a database model to API response format
func (sc *ServerController) convertToResponse(server *models.Server) ServerResponse {
	resp := ServerResponse{
		ID:           server.ID,
		Hostname:     server.Hostname,
		IPAddress:    server.IPAddress,
		OSType:       server.OSType,
		Status:       server.Status,
		Mode:         server.Mode,
		CreatedAt:    server.CreatedAt,
		UpdatedAt:    server.UpdatedAt,
	}

	// Optional fields
	if server.MACAddress.Valid {
		resp.MACAddress = server.MACAddress.String
	}
	if server.OSVersion.Valid {
		resp.OSVersion = server.OSVersion.String
	}
	if server.Architecture.Valid {
		resp.Architecture = server.Architecture.String
	}
	if server.AgentVersion.Valid {
		resp.AgentVersion = server.AgentVersion.String
	}

	// Last seen timestamp
	if server.LastSeen.Valid {
		resp.LastSeen = formatTimeSince(server.LastSeen.Time)
	} else {
		resp.LastSeen = "never"
	}

	// Tags
	if server.Tags != nil && len(server.Tags) > 0 {
		resp.Tags = make(map[string]string)
		for k, v := range server.Tags {
			if str, ok := v.(string); ok {
				resp.Tags[k] = str
			}
		}
	}

	// Extract metrics from metadata if available
	if server.Metadata != nil {
		if metrics, ok := server.Metadata["metrics"].(map[string]interface{}); ok {
			if cpu, ok := metrics["cpu_percent"].(float64); ok {
				resp.CPUUsage = &cpu
			}
			if memory, ok := metrics["memory_percent"].(float64); ok {
				resp.MemoryUsage = &memory
			}
			if disk, ok := metrics["disk_percent"].(float64); ok {
				resp.DiskUsage = &disk
			}
			if uptime, ok := metrics["uptime_seconds"].(float64); ok {
				uptimeInt := int64(uptime)
				resp.Uptime = &uptimeInt
			}
		}
	}

	return resp
}

// formatTimeSince formats a time as a human-readable "time ago" string
func formatTimeSince(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	}

	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return formatInt(minutes) + " minutes ago"
	}

	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return formatInt(hours) + " hours ago"
	}

	days := int(duration.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return formatInt(days) + " days ago"
}

func formatInt(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
