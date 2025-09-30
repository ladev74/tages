package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	cconfig "fileservice/internal/config"
	"fileservice/internal/grpc/grpc_app"
	llogger "fileservice/internal/logger"
	minio2 "fileservice/internal/sorage/minio"
)

// TODO: ask about:
// TODO: how to pass path to the config file (when starting by flag?),
// TODO: is it necessary grpc_app or use only service
// TODO: alias in import
// TODO: logging level

// TODO: use buffers
// TODO: metrics
// TODO: go doc comm

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

	fmt.Println(config)

	logger, err := llogger.New(config)
	if err != nil {
		log.Fatalf("cannot initialize logger: %v", err)
	}

	minio, err := minio2.New(ctx, config.Minio, logger)
	if err != nil {
		log.Fatalf("cannot initialize minio: %v", err)
	}

	application := grpcapp.New(minio, logger, &config.GRPC)

	go func() {
		err = application.Start()
		if err != nil {
			log.Fatalf("cannot start grpc server: %v", err)
		}
	}()

	<-ctx.Done()

	logger.Info("received shutdown signal")

	err = application.Stop()
	if err != nil {
		log.Fatalf("cannot gracefully stop grpc server: %v", err)
	}

	//postgresClient.Close()

	logger.Info("stopping http service", zap.Int("addr", 50051))

	logger.Info("application shutdown completed successfully")
}
