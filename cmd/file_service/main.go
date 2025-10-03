package main

import (
	"context"
	"flag"
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"fileservice/internal/config"
	"fileservice/internal/grpc/grpc_app"
	"fileservice/internal/logger"
	"fileservice/internal/sorage/minio"
	"fileservice/internal/sorage/postgres"
)

// TODO: go doc comm
// TODO: tests

// TODO: написать README в protos

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGQUIT,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	configPath := fetchPath()

	cfg, err := config.New(configPath)
	if err != nil {
		stdlog.Fatalf("cannot initialize cfg: %v", err)
	}

	log, err := logger.New(cfg.Env)
	if err != nil {
		stdlog.Fatalf("cannot initialize logger: %v", err)
	}

	minioStorage, err := minio.New(ctx, cfg.Minio, log)
	if err != nil {
		log.Fatal("cannot initialize minio", zap.Error(err))
	}

	postgresStorage, err := postgres.New(ctx, &cfg.Postgres, log)
	if err != nil {
		log.Fatal("cannot initialize postgres", zap.Error(err))
	}

	application := grpcapp.New(minioStorage, postgresStorage, log, &cfg.GRPC)

	go func() {
		err = application.Start()
		if err != nil {
			log.Fatal("cannot start grpc server", zap.Error(err))
		}
	}()

	<-ctx.Done()

	log.Info("received shutdown signal")

	application.Stop()
	// TODO: timeout
	if err != nil {
		log.Fatal("cannot gracefully stop grpc server", zap.Error(err))
	}

	//postgresStorage.Close()

	log.Info("stopping http service", zap.String("addr", fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port)))

	log.Info("application shutdown completed successfully")
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
