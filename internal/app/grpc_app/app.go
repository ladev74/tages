package grpcapp

import (
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"fileservice/internal/grpc/service"
)

type Config struct {
	Port int `yaml:"port" env-required:"true"`
}

type App struct {
	logger     *zap.Logger
	gRPCServer *grpc.Server
	port       int
}

func New(logger *zap.Logger, port int) *App {
	gRPCServer := grpc.NewServer()
	service.Register(gRPCServer)

	return &App{
		logger:     logger,
		gRPCServer: gRPCServer,
		port:       port,
	}
}

func (a *App) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		a.logger.Error("Start: failed to create listen", zap.Error(err))
		return fmt.Errorf("Start: failed to create listen: %w", err)
	}

	a.logger.Info("Start: gRPC server is starting", zap.Int("port", a.port))

	err = a.gRPCServer.Serve(lis)
	if err != nil {
		a.logger.Error("Start: failed to serve", zap.Error(err))
		return fmt.Errorf("Start: failed to serve: %w", err)
	}

	return nil
}

func (a *App) Stop() error {
	a.gRPCServer.GracefulStop()

	return nil
}
