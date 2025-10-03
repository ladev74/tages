package minio

import (
	"time"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

type Config struct {
	Host        string        `yaml:"host" env-required:"true"`
	Port        int           `yaml:"port" env-required:"true"`
	BucketName  string        `yaml:"bucket_name" env-required:"true"`
	User        string        `yaml:"user" env-required:"true"`
	Password    string        `yaml:"password" env-required:"true"`
	UseSSL      bool          `yaml:"use_ssl"`
	Timeout     time.Duration `yaml:"timeout" env-required:"true"`
	MaxRetries  int           `yaml:"max_retries" env-required:"true"`
	BaseBackoff time.Duration `yaml:"base_backoff" env-required:"true"`
}

type Storage struct {
	mc          *minio.Client
	bucketName  string
	timeout     time.Duration
	logger      *zap.Logger
	maxRetries  int
	baseBackoff time.Duration
}
