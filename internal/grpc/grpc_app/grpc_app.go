package grpcapp

import (
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"fileservice/internal/grpc/interceptor"
	"fileservice/internal/grpc/service"
	"fileservice/internal/limiter"
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

// TODO: вынести в main файл

func New(objectStorage service.ObjectStorage, metaStorage service.MetaStorage, log *zap.Logger, config *Config) *App {

	limCfg := &limiter.Config{
		LoadConcurrent: 3,
		ReadConcurrent: 3,
		TTL:            5 * time.Minute,
	}

	lim := limiter.NewRegistry(limCfg)

	concurrencyInterceptor := interceptor.New(lim)

	gRPCServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			concurrencyInterceptor.Unary(),
			interceptor.UnaryLoggingInterceptor(log),
		),
		grpc.ChainStreamInterceptor(
			concurrencyInterceptor.Stream(),
			interceptor.StreamLoggingInterceptor(log),
		),
	)

	service.Register(gRPCServer, objectStorage, metaStorage, config.OperationTimeout, log)

	reflection.Register(gRPCServer)

	return &App{
		gRPCServer:       gRPCServer,
		port:             config.Port,
		operationTimeout: config.OperationTimeout,
		logger:           log,
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

func (a *App) Stop() {
	a.gRPCServer.GracefulStop()
}
