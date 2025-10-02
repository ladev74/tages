package logger

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// TODO: add a logging level to the production logger

func New(env string) (*zap.Logger, error) {
	switch env {
	case "local":
		loggerConfig := zap.NewDevelopmentConfig()

		loggerConfig.DisableCaller = true
		loggerConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		loggerConfig.EncoderConfig.LineEnding = "\n\n"
		loggerConfig.EncoderConfig.ConsoleSeparator = " | "
		loggerConfig.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("\033[36m" + t.Format("15:04:05") + "\033[0m")
		}
		loggerConfig.Level.SetLevel(zap.DebugLevel)

		logger, err := loggerConfig.Build()
		if err != nil {
			return nil, err
		}

		return logger, nil

	case "prod":
		logger, err := zap.NewProduction()
		if err != nil {
			return nil, err
		}

		return logger, nil

	default:
		return nil, fmt.Errorf("unknown environment: %s", env)
	}
}

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
