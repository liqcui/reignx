package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestSize     *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec

	// gRPC metrics
	GRPCRequestsTotal   *prometheus.CounterVec
	GRPCRequestDuration *prometheus.HistogramVec

	// Node metrics
	NodesTotal          *prometheus.GaugeVec
	NodeHeartbeatAge    *prometheus.GaugeVec
	NodeLastSeen        *prometheus.GaugeVec

	// Task metrics
	TasksTotal          *prometheus.CounterVec
	TaskDuration        *prometheus.HistogramVec
	TasksInProgress     *prometheus.GaugeVec
	TaskQueueDepth      *prometheus.GaugeVec

	// Job metrics
	JobsTotal           *prometheus.CounterVec
	JobDuration         *prometheus.HistogramVec
	JobTasksTotal       *prometheus.GaugeVec

	// Agent metrics
	AgentConnections    *prometheus.GaugeVec
	AgentHeartbeats     *prometheus.CounterVec
	AgentErrors         *prometheus.CounterVec

	// Database metrics
	DBQueriesTotal      *prometheus.CounterVec
	DBQueryDuration     *prometheus.HistogramVec
	DBConnectionsActive prometheus.Gauge
	DBConnectionsIdle   prometheus.Gauge

	// Scheduler metrics
	SchedulerRunsTotal  prometheus.Counter
	SchedulerDuration   prometheus.Histogram
	SchedulerJobsQueued prometheus.Gauge
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		// HTTP metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path"},
		),
		HTTPRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_size_bytes",
				Help:      "HTTP request size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path"},
		),
		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path"},
		),

		// gRPC metrics
		GRPCRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "grpc_requests_total",
				Help:      "Total number of gRPC requests",
			},
			[]string{"method", "status"},
		),
		GRPCRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "grpc_request_duration_seconds",
				Help:      "gRPC request duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method"},
		),

		// Node metrics
		NodesTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "nodes_total",
				Help:      "Total number of nodes by status and mode",
			},
			[]string{"status", "mode"},
		),
		NodeHeartbeatAge: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "node_heartbeat_age_seconds",
				Help:      "Age of last heartbeat in seconds",
			},
			[]string{"node_id"},
		),
		NodeLastSeen: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "node_last_seen_timestamp",
				Help:      "Timestamp of last node heartbeat",
			},
			[]string{"node_id"},
		),

		// Task metrics
		TasksTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "tasks_total",
				Help:      "Total number of tasks by type and status",
			},
			[]string{"type", "status"},
		),
		TaskDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "task_duration_seconds",
				Help:      "Task execution duration in seconds",
				Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600},
			},
			[]string{"type", "status"},
		),
		TasksInProgress: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "tasks_in_progress",
				Help:      "Number of tasks currently in progress",
			},
			[]string{"type", "node_id"},
		),
		TaskQueueDepth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "task_queue_depth",
				Help:      "Number of tasks waiting in queue",
			},
			[]string{"node_id"},
		),

		// Job metrics
		JobsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "jobs_total",
				Help:      "Total number of jobs by type and status",
			},
			[]string{"type", "status"},
		),
		JobDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "job_duration_seconds",
				Help:      "Job execution duration in seconds",
				Buckets:   []float64{60, 300, 600, 1800, 3600, 7200, 14400, 28800},
			},
			[]string{"type", "status"},
		),
		JobTasksTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "job_tasks_total",
				Help:      "Total number of tasks in a job",
			},
			[]string{"job_id", "status"},
		),

		// Agent metrics
		AgentConnections: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "agent_connections",
				Help:      "Number of active agent connections",
			},
			[]string{"node_id"},
		),
		AgentHeartbeats: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "agent_heartbeats_total",
				Help:      "Total number of agent heartbeats",
			},
			[]string{"node_id"},
		),
		AgentErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "agent_errors_total",
				Help:      "Total number of agent errors",
			},
			[]string{"node_id", "error_type"},
		),

		// Database metrics
		DBQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_queries_total",
				Help:      "Total number of database queries",
			},
			[]string{"operation", "table", "status"},
		),
		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "db_query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"operation", "table"},
		),
		DBConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_active",
				Help:      "Number of active database connections",
			},
		),
		DBConnectionsIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_idle",
				Help:      "Number of idle database connections",
			},
		),

		// Scheduler metrics
		SchedulerRunsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "scheduler_runs_total",
				Help:      "Total number of scheduler runs",
			},
		),
		SchedulerDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "scheduler_duration_seconds",
				Help:      "Scheduler run duration in seconds",
				Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 30},
			},
		),
		SchedulerJobsQueued: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "scheduler_jobs_queued",
				Help:      "Number of jobs waiting to be scheduled",
			},
		),
	}

	return m
}

// RecordHTTPRequest records HTTP request metrics
func (m *Metrics) RecordHTTPRequest(method, path string, status int, duration time.Duration, requestSize, responseSize int64) {
	m.HTTPRequestsTotal.WithLabelValues(method, path, statusCode(status)).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
	m.HTTPRequestSize.WithLabelValues(method, path).Observe(float64(requestSize))
	m.HTTPResponseSize.WithLabelValues(method, path).Observe(float64(responseSize))
}

// RecordGRPCRequest records gRPC request metrics
func (m *Metrics) RecordGRPCRequest(method string, status string, duration time.Duration) {
	m.GRPCRequestsTotal.WithLabelValues(method, status).Inc()
	m.GRPCRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// RecordTaskExecution records task execution metrics
func (m *Metrics) RecordTaskExecution(taskType, status string, duration time.Duration) {
	m.TasksTotal.WithLabelValues(taskType, status).Inc()
	m.TaskDuration.WithLabelValues(taskType, status).Observe(duration.Seconds())
}

// RecordJobExecution records job execution metrics
func (m *Metrics) RecordJobExecution(jobType, status string, duration time.Duration) {
	m.JobsTotal.WithLabelValues(jobType, status).Inc()
	m.JobDuration.WithLabelValues(jobType, status).Observe(duration.Seconds())
}

// RecordDBQuery records database query metrics
func (m *Metrics) RecordDBQuery(operation, table, status string, duration time.Duration) {
	m.DBQueriesTotal.WithLabelValues(operation, table, status).Inc()
	m.DBQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordGauge records a gauge metric value
func (m *Metrics) RecordGauge(name string, value float64, labels map[string]string) {
	// This is a helper method for recording arbitrary gauge metrics
	// For now, we map to specific metrics based on name
	switch name {
	case "db_open_connections":
		m.DBConnectionsActive.Set(value)
	case "db_in_use_connections", "db_idle_connections":
		m.DBConnectionsIdle.Set(value)
	}
}

// RecordCounter records a counter metric increment
func (m *Metrics) RecordCounter(name string, value float64, labels map[string]string) {
	// This is a helper method for recording arbitrary counter metrics
	// For specific counters, use the dedicated methods like RecordHTTPRequest
}

// RecordHistogram records a histogram observation
func (m *Metrics) RecordHistogram(name string, value float64, labels map[string]string) {
	// This is a helper method for recording arbitrary histogram metrics
	// For specific histograms, use the dedicated methods like RecordDBQuery
}

// statusCode converts HTTP status code to string
func statusCode(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}
