package service

import (
	"time"

	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"fileservice/internal/sorage/minio"
	"fileservice/internal/sorage/postgres"
)

type service struct {
	fileservice.UnimplementedFileServiceServer
	postgres postgres.Client
	minio    minio.Client
	timeout  time.Duration
	logger   *zap.Logger
}

func Register(grpc *grpc.Server, postgres postgres.Client, minio minio.Client, timeout time.Duration, logger *zap.Logger) {
	fileservice.RegisterFileServiceServer(grpc,
		&service{
			postgres: postgres,
			minio:    minio,
			timeout:  timeout,
			logger:   logger,
		},
	)
}
