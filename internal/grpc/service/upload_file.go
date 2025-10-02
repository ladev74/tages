package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *service) UploadFile(stream grpc.ClientStreamingServer[fileservice.UploadFileRequest, fileservice.UploadFileResponse]) error {
	ctx, cancel := context.WithTimeout(stream.Context(), s.timeout)
	defer cancel()

	firstReq, err := stream.Recv()
	if err != nil {
		s.logger.Error("UploadFile: failed to receive first chunk", zap.Error(err))
		return status.Errorf(codes.InvalidArgument, "failed to receive first chunk: %v", err)
	}

	fileName := firstReq.GetFileName()
	if fileName == "" {
		s.logger.Error("UploadFile: empty fileName")
		return status.Errorf(codes.InvalidArgument, "filename is required")
	}

	var buf bytes.Buffer

	if len(firstReq.GetChunk()) > 0 {
		_, err := buf.Write(firstReq.GetChunk())
		if err != nil {
			s.logger.Error("UploadFile: failed to write first chunk to buffer", zap.Error(err))
			return status.Errorf(codes.Internal, "failed to write first chunk: %v", err)
		}
	}

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			s.logger.Error("UploadFile: failed to receive chunk", zap.Error(err))
			return status.Errorf(codes.Internal, "failed to receive chunk: %v", err)
		}

		_, err = buf.Write(req.GetChunk())
		if err != nil {
			s.logger.Error("UploadFile: failed to write chunk to buffer", zap.Error(err))
			return status.Errorf(codes.Internal, "failed to write chunk: %v", err)
		}
	}

	id := uuid.New().String()
	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()

	err = s.metaStorage.SaveFileInfo(ctx, id, fileName, createdAt, updatedAt)
	if err != nil {
		s.logger.Error("UploadFile: failed to save file info", zap.Error(err))
		return status.Errorf(codes.Internal, "failed to save file info: %v", err)
	}

	err = s.objectStorage.PutObject(ctx, id, &buf, int64(buf.Len()))
	if err != nil {
		s.logger.Error("UploadFile: failed to put object", zap.Error(err))
		err = s.metaStorage.DeleteFileInfo(ctx, id)
		if err != nil {
			s.logger.Error("UploadFile: failed to delete file info", zap.Error(err))
			return status.Errorf(codes.Internal, "failed to delete file info: %v", err)
		}

		return status.Errorf(codes.Internal, "failed to put object: %v", err)
	}

	err = s.metaStorage.SetSuccessStatus(ctx, id)
	if err != nil {
		s.logger.Error("UploadFile: failed to set success status", zap.Error(err))
		return status.Errorf(codes.Internal, "failed to set success status: %v", err)
	}

	s.logger.Info("UploadFile: successfully uploaded file", zap.String("id", id))
	return stream.SendAndClose(
		&fileservice.UploadFileResponse{
			FileId: id,
		},
	)
}
