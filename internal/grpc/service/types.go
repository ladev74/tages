package service

import (
	"time"

	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"fileservice/internal/sorage/minio"
)

type Service struct {
	fileservice.UnimplementedFileServiceServer
	//postgres postgres.Client
	minio  minio.Client
	timout time.Duration
	logger *zap.Logger
}

func Register(grpc *grpc.Server, minio minio.Client, timeout time.Duration, logger *zap.Logger) {
	fileservice.RegisterFileServiceServer(grpc,
		&Service{
			//postgresClient: postgresClient,
			timout: timeout,
			minio:  minio,
			logger: logger,
		},
	)
}
