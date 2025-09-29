package logger

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"

	"fileservice/internal/config"
)

// TODO: add a logging level to the production logger

func New(cfg *config.Config) (*zap.Logger, error) {
	switch cfg.Env {
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
		return nil, fmt.Errorf("unknown environment: %s", cfg.Env)
	}
}

func Interceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		next grpc.UnaryHandler,

	) (resp any, err error) {

		logger.Info(
			"new request", zap.String("method", info.FullMethod),
			zap.Any("request", req),
			zap.Time("time", time.Now()),
		)

		return next(ctx, req)
	}
}
