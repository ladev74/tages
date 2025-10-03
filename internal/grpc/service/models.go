package service

import (
	"context"
	"io"
	"time"

	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Config struct {
	BufSize       int
	MaxLimit      int64
	DefaultLimit  int64
	MaxOffset     int64
	DefaultOffset int64
	Timeout       time.Duration
}

type service struct {
	fileservice.UnimplementedFileServiceServer
	objectStorage ObjectStorage
	metaStorage   MetaStorage
	logger        *zap.Logger
	config        *Config
}

func Register(grpc *grpc.Server, objectStorage ObjectStorage, metaStorage MetaStorage, config *Config, logger *zap.Logger) {
	fileservice.RegisterFileServiceServer(grpc,
		&service{
			metaStorage:   metaStorage,
			objectStorage: objectStorage,
			logger:        logger,
			config:        config,
		},
	)
}

type ObjectStorage interface {
	PutObject(ctx context.Context, fileId string, reader io.Reader, size int64) error
	GetObject(ctx context.Context, id string) (io.ReadCloser, error)
}

type MetaStorage interface {
	SaveFileInfo(ctx context.Context, id string, fileName string, createdAt time.Time, updatedAt time.Time) error
	SetSuccessStatus(ctx context.Context, id string) error
	ListFilesInfo(ctx context.Context, limit int64, offset int64) ([]*fileservice.FileInfo, error)
	DeleteFileInfo(ctx context.Context, id string) error
	GetFileName(ctx context.Context, id string) (string, error)
}
