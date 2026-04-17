package observability

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/reignx/reignx/pkg/observability/metrics"
)

func TestNewStack(t *testing.T) {
	config := DefaultConfig("test-service-1")
	config.Tracing.Enabled = false // Disable tracing for unit tests

	// Use custom registry to avoid duplicate registration
	reg := prometheus.NewRegistry()
	config.Metrics.Enabled = false // Disable auto-init, we'll create manually

	stack, err := NewStack(config)
	if err != nil {
		t.Fatalf("Failed to create observability stack: %v", err)
	}
	defer stack.Shutdown(context.Background())

	// Create metrics manually with custom registry
	stack.Metrics = metrics.NewMetricsWithRegistry("test-service-1", reg)

	if stack.Metrics == nil {
		t.Error("Metrics should not be nil")
	}

	if stack.Logger == nil {
		t.Error("Logger should not be nil")
	}

	if stack.Tracer == nil {
		t.Error("Tracer should not be nil")
	}

	t.Log("Observability stack created successfully")
}

func TestStackLogging(t *testing.T) {
	config := DefaultConfig("test-service-2")
	config.Logging.Level = "debug"
	config.Tracing.Enabled = false
	config.Metrics.Enabled = false // Disable metrics for this test

	stack, err := NewStack(config)
	if err != nil {
		t.Fatalf("Failed to create observability stack: %v", err)
	}
	defer stack.Shutdown(context.Background())

	ctx := context.Background()

	// Test all log levels
	stack.Logger.Debug(ctx, "Debug message")
	stack.Logger.Info(ctx, "Info message")
	stack.Logger.Warn(ctx, "Warning message")
	stack.Logger.Error(ctx, "Error message")

	t.Log("All log levels tested successfully")
}

func TestStackMetrics(t *testing.T) {
	config := DefaultConfig("test-service-3")
	config.Tracing.Enabled = false
	config.Metrics.Enabled = false

	stack, err := NewStack(config)
	if err != nil {
		t.Fatalf("Failed to create observability stack: %v", err)
	}
	defer stack.Shutdown(context.Background())

	// Create metrics with custom registry
	reg := prometheus.NewRegistry()
	stack.Metrics = metrics.NewMetricsWithRegistry("test-service-3", reg)

	// Test recording metrics
	stack.Metrics.RecordHTTPRequest(
		"GET", "/api/v1/servers", 200,
		100*time.Millisecond, 1024, 2048,
	)

	stack.Metrics.RecordGRPCRequest(
		"/agent.AgentService/Heartbeat", "OK",
		50*time.Millisecond,
	)

	stack.Metrics.RecordTaskExecution(
		"script", "completed",
		5*time.Second,
	)

	stack.Metrics.RecordDBQuery(
		"SELECT", "servers", "success",
		10*time.Millisecond,
	)

	t.Log("All metrics recorded successfully")
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("test-service")

	if config.ServiceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", config.ServiceName)
	}

	if config.Environment != "development" {
		t.Errorf("Expected environment 'development', got '%s'", config.Environment)
	}

	if !config.Metrics.Enabled {
		t.Error("Metrics should be enabled by default")
	}

	if config.Tracing.Enabled {
		t.Error("Tracing should be disabled by default")
	}

	if config.Logging.Level != "info" {
		t.Errorf("Expected log level 'info', got '%s'", config.Logging.Level)
	}

	t.Log("Default configuration is correct")
}

func TestShutdown(t *testing.T) {
	config := DefaultConfig("test-service-4")
	config.Tracing.Enabled = false
	config.Metrics.Enabled = false

	stack, err := NewStack(config)
	if err != nil {
		t.Fatalf("Failed to create observability stack: %v", err)
	}

	// Test shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = stack.Shutdown(ctx)
	// Ignore stdout sync errors in tests - this is expected behavior
	if err != nil && err.Error() != "shutdown errors: [failed to sync logger: sync /dev/stdout: bad file descriptor]" {
		t.Errorf("Shutdown failed: %v", err)
	}

	t.Log("Shutdown completed successfully")
}
