package service

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/google/uuid"
	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const emptyFileName = ""

func (s *Service) UploadFile(stream grpc.ClientStreamingServer[fileservice.UploadFileRequest, fileservice.UploadFileResponse]) error {
	ctx, cancel := context.WithTimeout(stream.Context(), s.timout)
	defer cancel()

	firstReq, err := stream.Recv()
	if err != nil {
		s.logger.Error("UploadFile: failedl to get first request", zap.Error(err))
		return status.Errorf(codes.InvalidArgument, "failed to get first request: %v", err)
	}

	fileName := firstReq.GetFileName()
	if fileName == emptyFileName {
		s.logger.Error("UploadFile: empty fileName")
		return status.Errorf(codes.InvalidArgument, "filename is required")
	}

	var buf bytes.Buffer

	if len(firstReq.GetChunk()) > 0 {
		buf.Write(firstReq.GetChunk())
	}

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			s.logger.Error("UploadFile: stream error", zap.Error(err))
			return status.Errorf(codes.Internal, "stream error: %v", err)
		}

		buf.Write(req.GetChunk())
	}

	id := uuid.New().String()

	err = s.minio.PutObject(ctx, id, bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		s.logger.Error("UploadFile: upload error", zap.Error(err))
		return status.Errorf(codes.Internal, "upload error: %v", err)
	}

	s.logger.Info("UploadFile: upload success", zap.String("fileName", fileName))
	return stream.SendAndClose(
		&fileservice.UploadFileResponse{
			FileId: id,
		},
	)
}
