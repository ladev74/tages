package logger

import (
	"fmt"
	"io"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Env    string `env:"LOGGER" env-required:"true"`
	output io.Writer
}

func New(cfg *Config) (*zap.Logger, error) {
	switch cfg.Env {
	case "dev":
		config := zap.NewDevelopmentConfig()

		config.DisableCaller = true
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.LineEnding = "\n\n"
		config.EncoderConfig.ConsoleSeparator = " | "
		config.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("\033[36m" + t.Format("15:04:05") + "\033[0m")
		}

		if cfg.output != nil {
			config.OutputPaths = []string{"stdout"}
			core := zapcore.NewCore(
				zapcore.NewConsoleEncoder(config.EncoderConfig),
				zapcore.AddSync(cfg.output),
				config.Level,
			)

			logger := zap.New(core)

			return logger, nil

		} else {
			logger, err := config.Build()
			if err != nil {
				return nil, err
			}

			return logger, nil

		}

	case "prod":
		if cfg.output != nil {
			config := zap.NewProductionConfig()
			core := zapcore.NewCore(
				zapcore.NewJSONEncoder(config.EncoderConfig),
				zapcore.AddSync(cfg.output),
				config.Level,
			)

			logger := zap.New(core)

			return logger, nil

		} else {
			logger, err := zap.NewProduction()
			if err != nil {
				return nil, err
			}

			return logger, nil
		}

	default:
		return nil, fmt.Errorf("unknown environment: %s", cfg.Env)
	}
}
