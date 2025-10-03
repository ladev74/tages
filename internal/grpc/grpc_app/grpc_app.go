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
	Host             string        `yaml:"host" env-required:"true"`
	Port             int           `yaml:"port" env-required:"true"`
	OperationTimeout time.Duration `yaml:"operation_timeout" env-required:"true"`
	ShutdownTimeout  time.Duration `yaml:"shutdown_timeout" env-required:"true"`
	LoadConcurrent   int           `yaml:"load_concurrent" env-required:"true"`
	ReadConcurrent   int           `yaml:"read_concurrent" env-required:"true"`
	IdleTTL          time.Duration `yaml:"idle_ttl: 10m" env-default:"10m"`
}

type App struct {
	gRPCServer       *grpc.Server
	host             string
	port             int
	operationTimeout time.Duration
	logger           *zap.Logger
}

func New(objectStorage service.ObjectStorage, metaStorage service.MetaStorage, log *zap.Logger, config *Config) *App {
	lim := limiter.NewRegistry(config.LoadConcurrent, config.ReadConcurrent, config.IdleTTL)

	concurrencyInterceptor := interceptor.NewConcurrencyInterceptor(lim, log)
	loggingInterceptor := interceptor.NewLoggingInterceptor(log)

	gRPCServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			concurrencyInterceptor.Unary(),
			loggingInterceptor.Unary(),
		),
		grpc.ChainStreamInterceptor(
			concurrencyInterceptor.Stream(),
			loggingInterceptor.StreamLoggingInterceptor(),
		),
	)

	service.Register(gRPCServer, objectStorage, metaStorage, config.OperationTimeout, log)

	reflection.Register(gRPCServer)

	return &App{
		gRPCServer:       gRPCServer,
		host:             config.Host,
		port:             config.Port,
		operationTimeout: config.OperationTimeout,
		logger:           log,
	}
}

func (a *App) Start() error {
	addr := fmt.Sprintf("%s:%d", a.host, a.port)
	fmt.Println(addr)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		a.logger.Error("Start: failed to create listen", zap.Error(err))
		return fmt.Errorf("Start: failed to create listen: %w", err)
	}

	a.logger.Info("Start: gRPC server is starting", zap.String("addr", addr))

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
