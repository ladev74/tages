package service

import (
	"context"

	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type service struct {
	fileservice.UnimplementedFileServiceServer
}

func Register(grpc *grpc.Server) {
	fileservice.RegisterFileServiceServer(grpc, &service{})
}

func (s *service) UploadFile(ctx context.Context, req *fileservice.UploadFileRequest) (*fileservice.UploadFileResponse, error) {

	status.Error(codes.Unimplemented, "not implemented")
	status.Errorf()
	return &fileservice.UploadFileResponse{Id: "123"}, nil
}
