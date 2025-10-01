package main

import (
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	"go.uber.org/zap"

	cconfig "fileservice/internal/config"
	llogger "fileservice/internal/logger"
)

func main() {

	var configPath, migrationPath string

	flag.StringVar(&configPath, "config_path", "", "Path to the config file")
	flag.StringVar(&migrationPath, "migration_path", "", "Path to the migration file")
	flag.Parse()

	config, err := cconfig.New(configPath)
	if err != nil {
		log.Fatal(err)
	}

	logger, err := llogger.New(config.Env)
	if err != nil {
		log.Fatal(err)
	}

	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		config.Postgres.User,
		config.Postgres.Password,
		config.Postgres.Host,
		config.Postgres.Port,
		config.Postgres.Database,
	)

	migration, err := migrate.New("file://"+migrationPath, url)
	if err != nil {
		logger.Fatal("failed to create migration", zap.Error(err))
	}

	err = migration.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Fatal("failed to run migration", zap.Error(err))
	}

	logger.Info("successfully migrated")
}
