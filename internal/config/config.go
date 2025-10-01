package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"

	"fileservice/internal/grpc/grpc_app"
	"fileservice/internal/sorage/minio"
	"fileservice/internal/sorage/postgres"
)

type Config struct {
	Env      string          `yaml:"env" env-required:"true"`
	GRPC     grpcapp.Config  `yaml:"grpc" env-required:"true"`
	Postgres postgres.Config `yaml:"postgres" env-required:"true"`
	Minio    minio.Config    `yaml:"minio" env-required:"true"`
}

func New(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("path to the config is not specified")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file: %s, does not exist", path)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return &cfg, nil
}
