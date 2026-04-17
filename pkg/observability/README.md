# Observability Package

This package provides a comprehensive observability stack for the ReignX distributed system, including:

- **Metrics**: Prometheus-based metrics collection
- **Logging**: Structured logging with Zap and trace correlation
- **Tracing**: Distributed tracing with OpenTelemetry and Jaeger

## Quick Start

### Initialize Observability Stack

```go
import "github.com/liqcui/bm-distributed-solution/pkg/observability"

// Create observability stack
config := observability.DefaultConfig("reignx-apiserver")
config.Tracing.Enabled = true // Enable tracing if needed

stack, err := observability.NewStack(config)
if err != nil {
    log.Fatal(err)
}
defer stack.Shutdown(context.Background())
```

### Add HTTP Middleware

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/liqcui/bm-distributed-solution/pkg/observability/middleware"
    "github.com/liqcui/bm-distributed-solution/pkg/observability/handler"
)

router := gin.New()

// Add metrics middleware
router.Use(middleware.MetricsMiddleware(stack.Metrics))
router.Use(middleware.RecoveryMiddleware(stack.Metrics))

// Add metrics endpoint
router.GET("/metrics", handler.MetricsHandler())
router.GET("/health", handler.HealthHandler())
router.GET("/ready", handler.ReadinessHandler())
```

### Add gRPC Interceptors

```go
import (
    "google.golang.org/grpc"
    "github.com/liqcui/bm-distributed-solution/pkg/observability/middleware"
)

server := grpc.NewServer(
    grpc.UnaryInterceptor(middleware.UnaryServerInterceptor(stack.Metrics)),
    grpc.StreamInterceptor(middleware.StreamServerInterceptor(stack.Metrics)),
)
```

### Logging with Trace Correlation

```go
import (
    "context"
    "go.uber.org/zap"
    "github.com/liqcui/bm-distributed-solution/pkg/observability/tracing"
)

// Start a span
ctx, span := tracing.StartSpan(ctx, "reignx-apiserver", "ProcessRequest")
defer span.End()

// Log with trace correlation
stack.Logger.Info(ctx, "Processing request",
    zap.String("request_id", requestID),
    zap.String("user", userID),
)
```

## Metrics

### Available Metrics

#### HTTP Metrics
- `http_requests_total` - Total HTTP requests (labels: method, path, status)
- `http_request_duration_seconds` - Request duration histogram (labels: method, path)
- `http_request_size_bytes` - Request size histogram (labels: method, path)
- `http_response_size_bytes` - Response size histogram (labels: method, path)

#### gRPC Metrics
- `grpc_requests_total` - Total gRPC requests (labels: method, status)
- `grpc_request_duration_seconds` - Request duration histogram (labels: method)

#### Node Metrics
- `nodes_total` - Total nodes by status and mode (labels: status, mode)
- `node_heartbeat_age_seconds` - Age of last heartbeat (labels: node_id)
- `node_last_seen_timestamp` - Timestamp of last heartbeat (labels: node_id)

#### Task Metrics
- `tasks_total` - Total tasks (labels: type, status)
- `task_duration_seconds` - Task execution duration (labels: type, status)
- `tasks_in_progress` - Currently executing tasks (labels: type, node_id)
- `task_queue_depth` - Tasks waiting in queue (labels: node_id)

#### Job Metrics
- `jobs_total` - Total jobs (labels: type, status)
- `job_duration_seconds` - Job execution duration (labels: type, status)
- `job_tasks_total` - Tasks per job (labels: job_id, status)

#### Database Metrics
- `db_queries_total` - Total database queries (labels: operation, table, status)
- `db_query_duration_seconds` - Query duration (labels: operation, table)
- `db_connections_active` - Active database connections
- `db_connections_idle` - Idle database connections

### Recording Custom Metrics

```go
// Record HTTP request
stack.Metrics.RecordHTTPRequest(
    method, path, statusCode, duration,
    requestSize, responseSize,
)

// Record task execution
stack.Metrics.RecordTaskExecution(taskType, status, duration)

// Record database query
stack.Metrics.RecordDBQuery(operation, table, status, duration)
```

## Logging

### Log Levels
- `debug` - Detailed debugging information
- `info` - Informational messages
- `warn` - Warning messages
- `error` - Error messages
- `fatal` - Fatal errors (exits application)

### Configuration

```go
config := &observability.Config{
    Logging: observability.LoggingConfig{
        Level: "info",
        OutputPaths: []string{"stdout", "/var/log/reignx/app.log"},
    },
}
```

## Tracing

### Configuration

```go
config := &observability.Config{
    Tracing: observability.TracingConfig{
        JaegerEndpoint: "http://jaeger:14268/api/traces",
        Enabled: true,
    },
}
```

### Creating Spans

```go
import "github.com/liqcui/bm-distributed-solution/pkg/observability/tracing"

// Start a span
ctx, span := tracing.StartSpan(ctx, "service-name", "operation-name")
defer span.End()

// Add attributes to span
span.SetAttributes(
    attribute.String("user.id", userID),
    attribute.Int("task.count", taskCount),
)
```

## Deployment

### Prometheus

Create a `prometheus.yml` configuration:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'reignx-apiserver'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

### Jaeger

Run Jaeger using Docker:

```bash
docker run -d --name jaeger \
  -p 6831:6831/udp \
  -p 14268:14268 \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest
```

Access Jaeger UI at `http://localhost:16686`

### Grafana Dashboards

Import pre-built dashboards for:
- System Overview (server count, job status, task throughput)
- Job Performance (completion time, failure rate)
- Server Health (heartbeat status, resource usage)
- API Performance (request rate, latency, error rate)

## Best Practices

1. **Always use context**: Pass context through all layers for trace correlation
2. **Log at appropriate levels**: Use debug for verbose, info for important events, error for failures
3. **Add meaningful labels**: Use labels to enable filtering and aggregation
4. **Monitor metric cardinality**: Avoid high-cardinality labels (e.g., UUIDs)
5. **Set resource limits**: Configure Prometheus retention and Jaeger sampling
6. **Use structured logging**: Always use zap fields instead of string formatting
7. **Close spans**: Always defer `span.End()` after starting a span
8. **Handle shutdown gracefully**: Call `stack.Shutdown()` on application exit
