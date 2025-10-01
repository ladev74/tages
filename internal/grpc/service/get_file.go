package service

import (
	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"google.golang.org/grpc"
)

func (s *service) GetFile(*fileservice.GetFileRequest, grpc.ServerStreamingServer[fileservice.GetFileResponse]) error {

	return nil
}
