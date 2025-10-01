package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	cconfig "fileservice/internal/config"
	"fileservice/internal/grpc/grpc_app"
	llogger "fileservice/internal/logger"
	mminio "fileservice/internal/sorage/minio"
	ppostgres "fileservice/internal/sorage/postgres"
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

	configPath := fetchPath()

	config, err := cconfig.New(configPath)
	if err != nil {
		log.Fatalf("cannot initialize config: %v", err)
	}

	fmt.Println(config)

	logger, err := llogger.New(config.Env)
	if err != nil {
		log.Fatalf("cannot initialize logger: %v", err)
	}

	postgres, err := ppostgres.New(ctx, &config.Postgres, logger)
	if err != nil {
		logger.Fatal("cannot initialize postgres", zap.Error(err))
	}

	minio, err := mminio.New(ctx, config.Minio, logger)
	if err != nil {
		logger.Fatal("cannot initialize minio", zap.Error(err))
	}

	application := grpcapp.New(postgres, minio, logger, &config.GRPC)

	go func() {
		err = application.Start()
		if err != nil {
			logger.Fatal("cannot start grpc server", zap.Error(err))
		}
	}()

	<-ctx.Done()

	logger.Info("received shutdown signal")

	err = application.Stop()
	if err != nil {
		logger.Fatal("cannot gracefully stop grpc server", zap.Error(err))
	}

	postgres.Close()

	logger.Info("stopping http service", zap.Int("addr", 50051))

	logger.Info("application shutdown completed successfully")
}

func fetchPath() string {
	var path string

	flag.StringVar(&path, "config_path", "", "path to config file")
	flag.Parse()

	if path == "" {
		os.Getenv("CONFIG_PATH")
	}

	return path
}
