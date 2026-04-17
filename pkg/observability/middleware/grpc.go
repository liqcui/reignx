package middleware

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/reignx/reignx/pkg/observability/metrics"
)

// UnaryServerInterceptor creates a gRPC interceptor for unary RPCs that records metrics
func UnaryServerInterceptor(m *metrics.Metrics) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Call the handler
		resp, err := handler(ctx, req)

		// Record metrics
		duration := time.Since(start)
		statusCode := "OK"
		if err != nil {
			statusCode = status.Code(err).String()
		}

		m.RecordGRPCRequest(info.FullMethod, statusCode, duration)

		return resp, err
	}
}

// StreamServerInterceptor creates a gRPC interceptor for streaming RPCs that records metrics
func StreamServerInterceptor(m *metrics.Metrics) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		// Call the handler
		err := handler(srv, ss)

		// Record metrics
		duration := time.Since(start)
		statusCode := "OK"
		if err != nil {
			statusCode = status.Code(err).String()
		}

		m.RecordGRPCRequest(info.FullMethod, statusCode, duration)

		return err
	}
}
