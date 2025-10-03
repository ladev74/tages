package interceptor

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func UnaryLoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		start := time.Now()

		logger.Info("New unary request",
			zap.String("method", info.FullMethod),
		)

		resp, err = handler(ctx, req)

		st, _ := status.FromError(err)
		duration := time.Since(start)

		logger.Info("Unary request completed",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("status_code", st.Code().String()),
			zap.Error(err),
		)

		return resp, err
	}
}

func StreamLoggingInterceptor(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		logger.Info("New stream request",
			zap.String("method", info.FullMethod),
		)

		err := handler(srv, ss)

		st, _ := status.FromError(err)
		duration := time.Since(start)

		logger.Info("Stream request completed",
			zap.String("method", info.FullMethod),
			zap.Bool("is_client_stream", info.IsClientStream),
			zap.Bool("is_server_stream", info.IsServerStream),
			zap.Duration("duration", duration),
			zap.String("status_code", st.Code().String()),
			zap.Error(err),
		)

		return err
	}
}
