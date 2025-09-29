package app

import (
	"go.uber.org/zap"

	grpcapp "fileservice/internal/app/grpc_app"
)

type App struct {
	GRPCServer *grpcapp.App
}

func New(logger *zap.Logger, port int) *App {
	grpcApp := grpcapp.New(logger, port)

	return &App{
		GRPCServer: grpcApp,
	}

}
