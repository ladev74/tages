package service

import (
	"context"

	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxLimit     = 1_000_000
	defaultLimit = 100

	maxOffset     = 1_000_000
	defaultOffset = 0
)

func (s *service) ListFiles(ctx context.Context, req *fileservice.ListFilesRequest) (*fileservice.ListFilesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	limit := req.GetLimit()
	switch {
	case limit < 0:
		s.logger.Warn("ListFiles: invalid limit value", zap.Int64("limit", limit))
		return nil, status.Error(codes.InvalidArgument, "invalid limit")

	case limit > maxLimit:
		s.logger.Warn("ListFiles: limit too large", zap.Int64("limit", limit))
		return nil, status.Error(codes.InvalidArgument, "too large limit")

	case limit == 0:
		limit = defaultLimit
	}

	offset := req.GetOffset()
	switch {
	case offset < 0:
		s.logger.Warn("ListFiles: invalid offset value", zap.Int64("offset", offset))
		return nil, status.Error(codes.InvalidArgument, "invalid offset")

	case offset > maxOffset:
		s.logger.Warn("ListFiles: offset too large", zap.Int64("offset", offset))
		return nil, status.Error(codes.InvalidArgument, "too large offset")

	case offset == 0:
		offset = defaultOffset
	}

	files, err := s.metaStorage.ListFilesInfo(ctx, limit, offset)
	if err != nil {
		s.logger.Error("ListFiles: cannot get for files", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "cannot get files")
	}

	s.logger.Info("ListFiles: successfully get files", zap.Int("count", len(files)))
	return &fileservice.ListFilesResponse{Files: files}, nil
}
