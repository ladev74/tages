package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"fileservice/internal/api"
)

// TODO: вынести в логику grpc сервера

type Config struct {
	Timeout time.Duration
}

type Service struct {
	api.FileServiceServer
	//metric  metric.Monitoring
	timeout time.Duration
	logger  *zap.Logger
}

func New(logger *zap.Logger, config *Config) *Service {
	return &Service{
		timeout: config.Timeout,
		logger:  logger,
	}
}

func (s *Service) UploadFile(ctx context.Context, req *api.UploadFileRequest) (*api.UploadFileResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return &api.UploadFileResponse{
		Id: req.Id,
	}, nil
}

//func (s *Service) ListFiles(ctx context.Context, req *api.ListFilesRequest) (*api.ListFilesResponse, error) {
//	ctx, cancel := context.WithTimeout(ctx, s.timeout)
//	defer cancel()
//}
//
//func (s *Service) GetFile(ctx context.Context, req *api.GetFileRequest) (*api.GetFileResponse, error) {
//	ctx, cancel := context.WithTimeout(ctx, s.timeout)
//	defer cancel()
//}
