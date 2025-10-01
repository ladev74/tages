package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TODO: saga
// TODO: create file struct?

const emptyFileName = ""

func (s *service) UploadFile(stream grpc.ClientStreamingServer[fileservice.UploadFileRequest, fileservice.UploadFileResponse]) error {
	ctx, cancel := context.WithTimeout(stream.Context(), s.timeout)
	defer cancel()

	firstReq, err := stream.Recv()
	if err != nil {
		s.logger.Error("UploadFile: failedl to get first request", zap.Error(err))
		return status.Errorf(codes.InvalidArgument, "failed to receive first chunk: %v", err)
	}

	fileName := firstReq.GetFileName()
	if fileName == emptyFileName {
		s.logger.Error("UploadFile: empty fileName")
		return status.Errorf(codes.InvalidArgument, "filename is required")
	}

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		if len(firstReq.GetChunk()) > 0 {
			_, err = pw.Write(firstReq.GetChunk())
			if err != nil {
				s.logger.Error("UploadFile: failed to write chunk", zap.Error(err))
				pw.CloseWithError(fmt.Errorf("failed to write chunk: %w", err))
				return
			}
		}

		for {
			req, err := stream.Recv()
			if err == io.EOF {
				break
			}

			if err != nil {
				s.logger.Error("UploadFile: failed to receive chunk", zap.Error(err))
				pw.CloseWithError(fmt.Errorf("failed to receive chunk: %w", err))
				return
			}

			_, err = pw.Write(req.GetChunk())
			if err != nil {
				s.logger.Error("UploadFile: failed to write chunk", zap.Error(err))
				pw.CloseWithError(fmt.Errorf("failed to write chunk: %w", err))
				return
			}
		}
	}()

	id := uuid.New().String()
	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()

	err = s.postgres.SaveFileInfo(ctx, id, fileName, createdAt, updatedAt)
	if err != nil {
		s.logger.Error("UploadFile: failed to save file info", zap.Error(err))
		return status.Errorf(codes.Internal, "failed to save file info: %v", err)
	}

	err = s.minio.PutObject(ctx, id, pr, -1)
	if err != nil {
		s.logger.Error("UploadFile: failed to put object", zap.Error(err))
		err = s.postgres.DeleteFileInfo(ctx, id)
		if err != nil {
			s.logger.Error("UploadFile: failed to delete file info", zap.Error(err))
			return status.Errorf(codes.Internal, "failed to delete file info: %v", err)
		}

		return status.Errorf(codes.Internal, "failed to put object: %v", err)
	}

	err = s.postgres.SetSuccessStatus(ctx, id)
	if err != nil {
		s.logger.Error("UploadFile: failed to set success status", zap.Error(err))
		return status.Errorf(codes.Internal, "failed to set success status: %v", err)
	}

	return stream.SendAndClose(
		&fileservice.UploadFileResponse{
			FileId: id,
		},
	)
}
