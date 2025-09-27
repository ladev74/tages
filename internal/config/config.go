package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"

	"fileservice/internal/logger"
)

type Config struct {
	Logger logger.Config `env-required:"true"`
}

func New(path string) (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return &cfg, nil
}
