package observability

import (
	"context"
	"fmt"

	"github.com/reignx/reignx/pkg/observability/logging"
	"github.com/reignx/reignx/pkg/observability/metrics"
	"github.com/reignx/reignx/pkg/observability/tracing"
)

// Config contains all observability configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Metrics        MetricsConfig
	Logging        LoggingConfig
	Tracing        TracingConfig
}

// MetricsConfig contains metrics-specific configuration
type MetricsConfig struct {
	Namespace string
	Enabled   bool
}

// LoggingConfig contains logging-specific configuration
type LoggingConfig struct {
	Level       string
	OutputPaths []string
}

// TracingConfig contains tracing-specific configuration
type TracingConfig struct {
	JaegerEndpoint string
	Enabled        bool
}

// Stack contains all observability components
type Stack struct {
	Metrics *metrics.Metrics
	Logger  *logging.Logger
	Tracer  *tracing.TracerProvider
}

// NewStack creates a new observability stack with all components
func NewStack(config *Config) (*Stack, error) {
	stack := &Stack{}

	// Initialize metrics
	if config.Metrics.Enabled {
		namespace := config.Metrics.Namespace
		if namespace == "" {
			namespace = config.ServiceName
		}
		stack.Metrics = metrics.NewMetrics(namespace)
	}

	// Initialize logging
	logger, err := logging.NewLogger(&logging.Config{
		Level:       config.Logging.Level,
		Environment: config.Environment,
		OutputPaths: config.Logging.OutputPaths,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	stack.Logger = logger

	// Initialize tracing
	tracerProvider, err := tracing.NewTracerProvider(&tracing.Config{
		ServiceName:    config.ServiceName,
		ServiceVersion: config.ServiceVersion,
		Environment:    config.Environment,
		JaegerEndpoint: config.Tracing.JaegerEndpoint,
		Enabled:        config.Tracing.Enabled,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}
	stack.Tracer = tracerProvider

	return stack, nil
}

// Shutdown gracefully shuts down all observability components
func (s *Stack) Shutdown(ctx context.Context) error {
	var errs []error

	// Shutdown tracer
	if s.Tracer != nil {
		if err := s.Tracer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown tracer: %w", err))
		}
	}

	// Sync logger
	if s.Logger != nil {
		if err := s.Logger.Sync(); err != nil {
			errs = append(errs, fmt.Errorf("failed to sync logger: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	return nil
}

// DefaultConfig returns a default observability configuration
func DefaultConfig(serviceName string) *Config {
	return &Config{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		Environment:    "development",
		Metrics: MetricsConfig{
			Namespace: serviceName,
			Enabled:   true,
		},
		Logging: LoggingConfig{
			Level:       "info",
			OutputPaths: []string{"stdout"},
		},
		Tracing: TracingConfig{
			JaegerEndpoint: "http://localhost:14268/api/traces",
			Enabled:        false, // Disabled by default
		},
	}
}
