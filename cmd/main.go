package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"fileservice/internal/app"
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

	fmt.Println(config)

	logger, err := llogger.New(config)
	if err != nil {
		log.Fatalf("cannot initialize logger: %v", err)
	}

	application := app.New(logger, config.GRPC.Port)

	go func() {
		err = application.GRPCServer.Start()
		if err != nil {
			log.Fatalf("cannot start grpc server: %v", err)
		}
	}()

	<-ctx.Done()

	logger.Info("received shutdown signal")

	err = application.GRPCServer.Stop()
	if err != nil {
		log.Fatalf("cannot gracefully stop grpc server: %v", err)
	}

	//postgresClient.Close()

	logger.Info("stopping http service", zap.Int("addr", 50051))

	logger.Info("application shutdown completed successfully")
}
