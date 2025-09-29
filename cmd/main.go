package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	cconfig "fileservice/internal/config"
	llogger "fileservice/internal/logger"
)

// TODO: ask how pass path to the config file (when starting by flag?),
// TODO: alias in import

// TODO: use buffers
// TODO: metrics

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGQUIT,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	config, err := cconfig.New()
	if err != nil {
		log.Fatalf("cannot initialize config: %v", err)
	}

	logger, err := llogger.New(config)
	if err != nil {
		log.Fatalf("cannot initialize logger: %v", err)
	}

	//service := sservice.New(logger, &sservice.Config{Timeout: 3 * time.Second})

	grpcLis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", 50051))
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(llogger.Interceptor(logger)))
	//api.RegisterFileServiceServer(server, service)
	reflection.Register(server)

	go func() {
		logger.Info("GRPC server started", zap.Int("addr", 50051))
		if err = server.Serve(grpcLis); err != nil {
			logger.Fatal("failed to serve", zap.Error(err))
		}
	}()

	<-ctx.Done()

	logger.Info("received shutdown signal")

	server.GracefulStop()

	//postgresClient.Close()

	logger.Info("stopping http server", zap.Int("addr", 50051))

	logger.Info("application shutdown completed successfully")
}
