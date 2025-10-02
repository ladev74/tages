package service

import (
	"context"
	"io"
	"time"

	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type service struct {
	fileservice.UnimplementedFileServiceServer
	//storage Storage
	objectStorage ObjectStorage
	metaStorage   MetaStorage
	timeout       time.Duration
	logger        *zap.Logger
}

func Register(grpc *grpc.Server, objectStorage ObjectStorage, metaStorage MetaStorage, timeout time.Duration, logger *zap.Logger) {
	fileservice.RegisterFileServiceServer(grpc,
		&service{
			//storage: storage,
			metaStorage:   metaStorage,
			objectStorage: objectStorage,
			timeout:       timeout,
			logger:        logger,
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
}

// TODO: generate mock
