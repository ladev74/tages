package grpcapp

import (
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"fileservice/internal/grpc/service"
	"fileservice/internal/sorage/minio"
)

type Config struct {
	Port             int           `yaml:"port" env-required:"true"`
	OperationTimeout time.Duration `yaml:"operation_timeout" env-required:"true"`
	ShutdownTimeout  time.Duration `yaml:"shutdown_timeout" env-required:"true"`
}

type App struct {
	gRPCServer       *grpc.Server
	port             int
	operationTimeout time.Duration
	logger           *zap.Logger
}

func New(minio minio.Client, logger *zap.Logger, config *Config) *App {
	gRPCServer := grpc.NewServer()

	service.Register(gRPCServer, minio, config.OperationTimeout, logger)

	return &App{
		gRPCServer:       gRPCServer,
		port:             config.Port,
		operationTimeout: config.OperationTimeout,
		logger:           logger,
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
