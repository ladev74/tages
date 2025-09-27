package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	cconfig "fileservice/internal/config"
	llogger "fileservice/internal/logger"
)

const (
	pathToConfigFile = "./config/config.env"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGQUIT,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	config, err := cconfig.New(pathToConfigFile)
	if err != nil {
		log.Fatalf("cannot initialize config:%v", err)
	}

	logger, err := llogger.New(&config.Logger)
	if err != nil {
		log.Fatalf("cannot initialize logger:%v", err)
	}

}
