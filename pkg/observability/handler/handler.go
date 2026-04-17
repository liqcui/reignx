package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler creates an HTTP handler for Prometheus metrics endpoint
func MetricsHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// HealthHandler creates a basic health check endpoint
func HealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	}
}

// ReadinessHandler creates a readiness check endpoint
func ReadinessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// In a real implementation, check dependencies (database, message queue, etc.)
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	}
}
