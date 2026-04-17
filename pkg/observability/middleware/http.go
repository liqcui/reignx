package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/reignx/reignx/pkg/observability/metrics"
)

// MetricsMiddleware creates a Gin middleware that automatically records HTTP metrics
func MetricsMiddleware(m *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Record request size
		requestSize := computeRequestSize(c.Request)

		// Process request
		c.Next()

		// Record metrics after request is processed
		duration := time.Since(start)
		status := c.Writer.Status()
		responseSize := int64(c.Writer.Size())

		m.RecordHTTPRequest(
			c.Request.Method,
			c.FullPath(),
			status,
			duration,
			requestSize,
			responseSize,
		)
	}
}

// computeRequestSize calculates the approximate size of an HTTP request
func computeRequestSize(r *http.Request) int64 {
	size := int64(0)

	// Request line
	size += int64(len(r.Method))
	size += int64(len(r.URL.String()))
	size += int64(len(r.Proto))

	// Headers
	for name, values := range r.Header {
		size += int64(len(name))
		for _, value := range values {
			size += int64(len(value))
		}
	}

	// Body
	if r.ContentLength > 0 {
		size += r.ContentLength
	}

	return size
}

// RecoveryMiddleware provides panic recovery with logging and metrics
func RecoveryMiddleware(m *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Record panic as 500 error
				m.HTTPRequestsTotal.WithLabelValues(
					c.Request.Method,
					c.FullPath(),
					"5xx",
				).Inc()

				c.AbortWithStatus(500)
			}
		}()

		c.Next()
	}
}
