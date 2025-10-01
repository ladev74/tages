package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

const (
	statusPending = "pending"
	statusSuccess = "success"
)

type Config struct {
	Host              string        `yaml:"host" env-required:"true"`
	Port              string        `yaml:"port" env-required:"true"`
	User              string        `yaml:"user" env-required:"true"`
	Password          string        `yaml:"password" env-required:"true"`
	Database          string        `yaml:"database" env-required:"true"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout" env-required:"true"`
	OperationTimeout  time.Duration `yaml:"operation_timeout" env-required:"true"`
	MaxConns          int           `yaml:"max_connections" env-required:"true"`
	MinConns          int           `yaml:"min_connections" env-required:"true"`
}

type Storage struct {
	pool    *pgxpool.Pool
	logger  *zap.Logger
	timeout time.Duration
}

type Client interface {
	SaveFileInfo(ctx context.Context, id string, fileName string, createdAt time.Time, updatedAt time.Time) error
	SetSuccessStatus(ctx context.Context, id string) error
	DeleteFileInfo(ctx context.Context, id string) error
	ListFilesInfo(ctx context.Context, limit int64, offset int64) ([]*fileservice.FileInfo, error)
	Close()
}

type MockPostgresService struct {
	mock.Mock
}
