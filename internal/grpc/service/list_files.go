package service

import (
	"context"
	"fmt"

	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *service) ListFiles(ctx context.Context, req *fileservice.ListFilesRequest) (*fileservice.ListFilesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	limit, reason := validateLimit(req.GetLimit(), s.config)
	if reason != "" {
		s.logger.Warn(fmt.Sprintf("ListFiles: %s", reason), zap.Int64("limit", limit))
		return nil, status.Error(codes.InvalidArgument, reason)
	}

	offset, reason := validateOffset(req.GetOffset(), s.config)
	if reason != "" {
		s.logger.Warn(fmt.Sprintf("ListFiles: %s", reason), zap.Int64("offset", offset))
		return nil, status.Error(codes.InvalidArgument, reason)
	}

	filesInfo, err := s.metaStorage.ListFilesInfo(ctx, limit, offset)
	if err != nil {
		s.logger.Error("ListFiles: cannot get for files", zap.Error(err))
		return nil, status.Error(codes.Internal, "cannot get files")
	}

	s.logger.Info("ListFiles: successfully get files", zap.Int("count", len(filesInfo)))
	return &fileservice.ListFilesResponse{Files: filesInfo}, nil
}

func validateLimit(limit int64, cfg *Config) (int64, string) {
	switch {
	case limit < 0:
		return 0, "limit must not be negative"

	case limit > cfg.MaxLimit:
		return 0, "too large limit"

	case limit == 0:
		limit = cfg.DefaultLimit
	}

	return limit, ""
}

func validateOffset(offset int64, cfg *Config) (int64, string) {
	switch {
	case offset < 0:
		return 0, "offset must not be negative"

	case offset > cfg.MaxOffset:
		return 0, "too large offset"

	case offset == 0:
		offset = cfg.DefaultOffset
	}

	return offset, ""
}
