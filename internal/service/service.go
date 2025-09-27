package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"fileservice/internal/api"
)

type Config struct {
	timeout time.Duration
}

type Service struct {
	api.FileServiceServer
	timeout time.Duration
	logger  *zap.Logger
}

func New(logger *zap.Logger, config Config) *Service {
	return &Service{
		timeout: config.timeout,
		logger:  logger,
	}
}

func (s *Service) GetFile(ctx context.Context, req *api.GetFileRequest) (*api.GetFileResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
}
