package service

import (
	"context"

	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultLimit = 100

// TODO: separate message for list files response without id?

func (s *service) ListFiles(ctx context.Context, req *fileservice.ListFilesRequest) (*fileservice.ListFilesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	limit := req.GetLimit()
	if limit <= 0 {
		limit = defaultLimit
	}

	offset := req.GetOffset()

	files, err := s.postgres.ListFilesInfo(ctx, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get files")
	}

	s.logger.Info("ListFiles: successfully get files", zap.Int("count", len(files)))
	return &fileservice.ListFilesResponse{Files: files}, nil
}
