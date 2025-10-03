package service

import (
	"context"
	"errors"
	"io"

	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"fileservice/internal/sorage/minio"
)

func (s *service) GetFile(req *fileservice.GetFileRequest, stream grpc.ServerStreamingServer[fileservice.GetFileResponse]) error {
	ctx, cancel := context.WithTimeout(stream.Context(), s.config.Timeout)
	defer cancel()

	id := req.GetFileId()
	if id == "" {
		s.logger.Warn("GetFile: file id is empty")
		return status.Errorf(codes.InvalidArgument, "file id is required")
	}

	object, err := s.objectStorage.GetObject(ctx, id)
	if err != nil {
		if errors.Is(err, minio.ErrNotFound) {
			s.logger.Warn("GetFile: file not found")
			return status.Errorf(codes.NotFound, "file not found")
		}

		s.logger.Error("GetFile: failed to get file", zap.String("id", id), zap.Error(err))
		return status.Errorf(codes.Internal, "failed to get file: %s", id)
	}

	defer func() {
		err = object.Close()
		if err != nil {
			s.logger.Warn("GetFile: failed to close object", zap.String("id", id), zap.Error(err))
		}
	}()

	fileName, err := s.metaStorage.GetFileName(ctx, id)
	if err != nil {
		if errors.Is(err, minio.ErrNotFound) {
			s.logger.Warn("GetFile: file not found")
			return status.Errorf(codes.NotFound, "file not found")
		}

		s.logger.Error("GetFile: failed to get file name", zap.String("id", id), zap.Error(err))
		return status.Errorf(codes.Internal, "failed to get file name: %s", id)
	}

	firstResp := &fileservice.GetFileResponse{
		FileName: fileName,
	}

	err = stream.Send(firstResp)
	if err != nil {
		s.logger.Error("GetFile: failed to send file name", zap.String("id", id), zap.Error(err))
		return status.Errorf(codes.Internal, "failed to send file name: %s", id)
	}

	buf := make([]byte, s.config.BufSize)

	for {
		n, err := object.Read(buf)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			s.logger.Error("GetFile: failed to read object", zap.String("id", id), zap.Error(err))
			return status.Errorf(codes.Internal, "failed to get file: %s", id)
		}

		resp := &fileservice.GetFileResponse{
			Chunk: buf[:n],
		}

		err = stream.Send(resp)
		if err != nil {
			s.logger.Error("GetFile: failed to send response", zap.String("id", id), zap.Error(err))
			return status.Errorf(codes.Internal, "failed to send response: %s", id)
		}
	}

	s.logger.Info("GetFile: successfully get file", zap.String("id", id))
	return nil
}
