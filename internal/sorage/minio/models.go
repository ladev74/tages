package minio

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

type Config struct {
	Host             string        `yaml:"host" env-required:"true"`
	Port             int           `yaml:"port" env-required:"true"`
	BucketName       string        `yaml:"bucket_name" env-required:"true"`
	User             string        `yaml:"user" env-required:"true"`
	Password         string        `yaml:"password" env-required:"true"`
	UseSSL           bool          `yaml:"use_ssl"`
	OperationTimeout time.Duration `yaml:"operation_timeout" env-required:"true"`
}

type Client interface {
	PutObject(ctx context.Context, fileId string, reader io.Reader, size int64) error
	GetObject(ctx context.Context, id string) (io.ReadCloser, error)
	//List(offset int) ([]string, error)
}

type Storage struct {
	mc         *minio.Client
	bucketName string
	timeout    time.Duration
	logger     *zap.Logger
}
