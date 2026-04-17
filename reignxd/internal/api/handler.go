package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/reignx/reignx/pkg/core"
	"go.uber.org/zap"
)

// Config contains API handler configuration
type Config struct {
	Logger       *zap.Logger
	NodeRepo     core.NodeRepository
	JobRepo      core.JobRepository
	TaskRepo     core.TaskRepository
	ModeSwitcher core.ModeSwitcher
}

// Handler implements the HTTP API
type Handler struct {
	logger       *zap.Logger
	nodeRepo     core.NodeRepository
	jobRepo      core.JobRepository
	taskRepo     core.TaskRepository
	modeSwitcher core.ModeSwitcher
}

// NewHandler creates a new API handler
func NewHandler(config *Config) http.Handler {
	h := &Handler{
		logger:       config.Logger,
		nodeRepo:     config.NodeRepo,
		jobRepo:      config.JobRepo,
		taskRepo:     config.TaskRepo,
		modeSwitcher: config.ModeSwitcher,
	}

	router := gin.Default()

	// Health check
	router.GET("/health", h.handleHealth)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Nodes
		nodes := v1.Group("/nodes")
		{
			nodes.GET("", h.handleListNodes)
			nodes.POST("", h.handleCreateNode)
			nodes.GET("/:id", h.handleGetNode)
			nodes.PUT("/:id", h.handleUpdateNode)
			nodes.DELETE("/:id", h.handleDeleteNode)
			nodes.POST("/:id/tags", h.handleUpdateNodeTags)
			nodes.POST("/:id/switch-mode", h.handleSwitchMode)
			nodes.GET("/:id/switch-progress", h.handleGetSwitchProgress)
		}

		// Jobs
		jobs := v1.Group("/jobs")
		{
			jobs.GET("", h.handleListJobs)
			jobs.POST("", h.handleCreateJob)
			jobs.GET("/:id", h.handleGetJob)
			jobs.DELETE("/:id", h.handleCancelJob)
			jobs.POST("/:id/retry", h.handleRetryJob)
		}

		// Tasks
		tasks := v1.Group("/tasks")
		{
			tasks.GET("", h.handleListTasks)
			tasks.GET("/:id", h.handleGetTask)
		}

		// Stats
		v1.GET("/stats", h.handleGetStats)
	}

	return router
}

// Health check
func (h *Handler) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// Node handlers

func (h *Handler) handleListNodes(c *gin.Context) {
	filter := &core.NodeFilter{
		Limit:  100,
		Offset: 0,
	}

	// Parse query parameters
	if mode := c.Query("mode"); mode != "" {
		filter.Mode = core.NodeMode(mode)
	}
	if status := c.Query("status"); status != "" {
		filter.Status = core.NodeStatus(status)
	}

	nodes, err := h.nodeRepo.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list nodes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list nodes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes": nodes,
		"total": len(nodes),
	})
}

func (h *Handler) handleCreateNode(c *gin.Context) {
	var req struct {
		Hostname  string            `json:"hostname" binding:"required"`
		IPAddress string            `json:"ip_address" binding:"required"`
		Mode      core.NodeMode     `json:"mode"`
		OSType    string            `json:"os_type"`
		Tags      map[string]string `json:"tags"`
		SSHConfig *core.SSHConfig   `json:"ssh_config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node := &core.Node{
		ID:        uuid.New().String(),
		Hostname:  req.Hostname,
		IPAddress: req.IPAddress,
		Mode:      req.Mode,
		Status:    core.NodeStatusOffline,
		OSType:    req.OSType,
		Tags:      req.Tags,
		Metadata:  make(map[string]interface{}),
		SSHConfig: req.SSHConfig,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if node.Mode == "" {
		node.Mode = core.NodeModeSSH
	}

	if err := h.nodeRepo.Create(c.Request.Context(), node); err != nil {
		h.logger.Error("Failed to create node", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create node"})
		return
	}

	h.logger.Info("Node created", zap.String("node_id", node.ID))
	c.JSON(http.StatusCreated, node)
}

func (h *Handler) handleGetNode(c *gin.Context) {
	id := c.Param("id")

	node, err := h.nodeRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}

	c.JSON(http.StatusOK, node)
}

func (h *Handler) handleUpdateNode(c *gin.Context) {
	id := c.Param("id")

	node, err := h.nodeRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}

	if err := c.ShouldBindJSON(node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.nodeRepo.Update(c.Request.Context(), node); err != nil {
		h.logger.Error("Failed to update node", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update node"})
		return
	}

	c.JSON(http.StatusOK, node)
}

func (h *Handler) handleDeleteNode(c *gin.Context) {
	id := c.Param("id")

	if err := h.nodeRepo.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("Failed to delete node", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete node"})
		return
	}

	h.logger.Info("Node deleted", zap.String("node_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "node deleted"})
}

func (h *Handler) handleUpdateNodeTags(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Tags map[string]string `json:"tags" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.nodeRepo.AddTags(c.Request.Context(), id, req.Tags); err != nil {
		h.logger.Error("Failed to update tags", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update tags"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tags updated"})
}

// Job handlers

func (h *Handler) handleListJobs(c *gin.Context) {
	filter := &core.JobFilter{
		Limit:  50,
		Offset: 0,
	}

	jobs, err := h.jobRepo.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list jobs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list jobs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":  jobs,
		"total": len(jobs),
	})
}

func (h *Handler) handleCreateJob(c *gin.Context) {
	var req struct {
		Name        string                   `json:"name" binding:"required"`
		Description string                   `json:"description"`
		Type        core.TaskType            `json:"type" binding:"required"`
		Mode        core.NodeMode            `json:"mode"`
		Filter      json.RawMessage          `json:"filter" binding:"required"`
		Template    *core.Task               `json:"template" binding:"required"`
		BatchSize   int                      `json:"batch_size"`
		Concurrency int                      `json:"concurrency"`
		Parameters  map[string]interface{}   `json:"parameters"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse filter
	var nodeFilter core.NodeFilter
	if err := json.Unmarshal(req.Filter, &nodeFilter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filter"})
		return
	}

	job := &core.Job{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Mode:        req.Mode,
		Filter:      &nodeFilter,
		Template:    req.Template,
		Status:      core.TaskStatusPending,
		BatchSize:   req.BatchSize,
		Concurrency: req.Concurrency,
		CreatedBy:   "api", // TODO: Get from auth context
		CreatedAt:   time.Now(),
		Parameters:  req.Parameters,
	}

	if job.Mode == "" {
		job.Mode = core.NodeModeSSH
	}
	if job.BatchSize == 0 {
		job.BatchSize = 10
	}
	if job.Concurrency == 0 {
		job.Concurrency = 10
	}

	if err := h.jobRepo.Create(c.Request.Context(), job); err != nil {
		h.logger.Error("Failed to create job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create job"})
		return
	}

	h.logger.Info("Job created", zap.String("job_id", job.ID))
	c.JSON(http.StatusCreated, job)
}

func (h *Handler) handleGetJob(c *gin.Context) {
	id := c.Param("id")

	job, err := h.jobRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Get tasks for the job
	tasks, _ := h.taskRepo.GetByJob(c.Request.Context(), id)

	c.JSON(http.StatusOK, gin.H{
		"job":   job,
		"tasks": tasks,
	})
}

func (h *Handler) handleCancelJob(c *gin.Context) {
	id := c.Param("id")

	err := h.jobRepo.UpdateStatus(c.Request.Context(), id, core.TaskStatusCancelled)
	if err != nil {
		h.logger.Error("Failed to cancel job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel job"})
		return
	}

	h.logger.Info("Job cancelled", zap.String("job_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "job cancelled"})
}

func (h *Handler) handleRetryJob(c *gin.Context) {
	id := c.Param("id")

	// Get failed tasks
	tasks, err := h.taskRepo.List(c.Request.Context(), &core.TaskFilter{
		JobID:  id,
		Status: []core.TaskStatus{core.TaskStatusFailed},
	})
	if err != nil {
		h.logger.Error("Failed to get tasks", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retry job"})
		return
	}

	// Reset tasks to pending
	for _, task := range tasks {
		task.Status = core.TaskStatusPending
		task.Retries = 0
		h.taskRepo.Update(c.Request.Context(), task)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "job retried",
		"tasks":   len(tasks),
	})
}

// Task handlers

func (h *Handler) handleListTasks(c *gin.Context) {
	filter := &core.TaskFilter{
		Limit: 100,
	}

	if jobID := c.Query("job_id"); jobID != "" {
		filter.JobID = jobID
	}
	if nodeID := c.Query("node_id"); nodeID != "" {
		filter.NodeID = nodeID
	}

	tasks, err := h.taskRepo.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list tasks", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tasks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"total": len(tasks),
	})
}

func (h *Handler) handleGetTask(c *gin.Context) {
	id := c.Param("id")

	task, err := h.taskRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// Stats handler

func (h *Handler) handleGetStats(c *gin.Context) {
	// Get node stats
	totalNodes, _ := h.nodeRepo.Count(c.Request.Context(), &core.NodeFilter{})
	onlineNodes, _ := h.nodeRepo.Count(c.Request.Context(), &core.NodeFilter{
		Status: core.NodeStatusOnline,
	})

	c.JSON(http.StatusOK, gin.H{
		"nodes": gin.H{
			"total":  totalNodes,
			"online": onlineNodes,
		},
	})
}

// Mode switching handlers

type SwitchModeRequest struct {
	ToMode string `json:"to_mode" binding:"required,oneof=ssh agent"`
}

func (h *Handler) handleSwitchMode(c *gin.Context) {
	nodeID := c.Param("id")

	var req SwitchModeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get node
	node, err := h.nodeRepo.Get(c.Request.Context(), nodeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}

	// Check if mode switcher is available
	if h.modeSwitcher == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "mode switching not configured"})
		return
	}

	toMode := core.NodeMode(req.ToMode)

	// Check if node is already in target mode
	if node.Mode == toMode {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node is already in " + req.ToMode + " mode"})
		return
	}

	// Perform mode switch asynchronously
	go func() {
		ctx := context.Background()

		h.logger.Info("Starting mode switch",
			zap.String("node_id", nodeID),
			zap.String("from_mode", string(node.Mode)),
			zap.String("to_mode", req.ToMode),
		)

		var err error
		if toMode == core.NodeModeAgent {
			err = h.modeSwitcher.SwitchToAgent(ctx, node)
		} else {
			err = h.modeSwitcher.SwitchToSSH(ctx, node)
		}

		if err != nil {
			h.logger.Error("Mode switch failed",
				zap.String("node_id", nodeID),
				zap.String("to_mode", req.ToMode),
				zap.Error(err),
			)
		} else {
			h.logger.Info("Mode switch completed successfully",
				zap.String("node_id", nodeID),
				zap.String("to_mode", req.ToMode),
			)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "mode switch initiated",
		"node_id": nodeID,
		"to_mode": req.ToMode,
	})
}

func (h *Handler) handleGetSwitchProgress(c *gin.Context) {
	nodeID := c.Param("id")

	// Check if mode switcher is available
	if h.modeSwitcher == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "mode switching not configured"})
		return
	}

	progress, err := h.modeSwitcher.GetProgress(c.Request.Context(), nodeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, progress)
}
