package logger

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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
