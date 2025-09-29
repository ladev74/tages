package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type Config struct {
	Host              string        `yaml:"postgres_host" env-required:"true"`
	Port              string        `yaml:"postgres_port" env-required:"true"`
	User              string        `yaml:"postgres_user" env-required:"true"`
	Password          string        `yaml:"postgres_password" env-required:"true"`
	Database          string        `yaml:"postgres_database" env-required:"true"`
	ConnectionTimeout time.Duration `yaml:"postgres_connection_timeout" env-required:"true"`
	OperationTimeout  time.Duration `yaml:"postgres_operation_timeout" env-required:"true"`
	MigrationsPath    string        `yaml:"postgres_migration_path" env-required:"true"`
	MaxConns          int           `yaml:"postgres_max_connections" env-required:"true"`
	MinConns          int           `yaml:"postgres_min_connections" env-required:"true"`
}

type Storage struct {
	pool    *pgxpool.Pool
	logger  *zap.Logger
	timeout time.Duration
}

type Client interface {
	Close()
}

type MockPostgresService struct {
	mock.Mock
}
