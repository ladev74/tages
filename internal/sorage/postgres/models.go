package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Config struct {
	Host        string        `yaml:"host" env-required:"true"`
	Port        string        `yaml:"port" env-required:"true"`
	User        string        `yaml:"user" env-required:"true"`
	Password    string        `yaml:"password" env-required:"true"`
	Database    string        `yaml:"database" env-required:"true"`
	Timeout     time.Duration `yaml:"timeout" env-required:"true"`
	MaxRetries  int           `yaml:"max_retries" env-required:"true"`
	BaseBackoff time.Duration `yaml:"base_backoff" env-required:"true"`
	MaxConns    int           `yaml:"max_connections" env-required:"true"`
	MinConns    int           `yaml:"min_connections" env-required:"true"`
}

type Storage struct {
	pool        *pgxpool.Pool
	logger      *zap.Logger
	timeout     time.Duration
	maxRetries  int
	baseBackoff time.Duration
}
