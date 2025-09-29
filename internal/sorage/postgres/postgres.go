package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// TODO: retries!
// TODO: circuit breaker?

func New(ctx context.Context, config *Config, logger *zap.Logger) (*Storage, error) {
	ctx, cancel := context.WithTimeout(ctx, config.ConnectionTimeout)
	defer cancel()

	url := buildURL(config)
	dsn := buildDSN(config)

	pool, err := pgxpool.New(ctx, dsn)

	if err != nil {
		return nil, err
	}

	err = upMigration(url, config.MigrationsPath)
	if err != nil {
		return nil, err
	}

	return &Storage{
		pool:    pool,
		logger:  logger,
		timeout: config.OperationTimeout,
	}, nil
}

func buildURL(config *Config) string {
	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	return url
}

func buildDSN(config *Config) string {
	dsn := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s pool_max_conns=%d pool_min_conns=%d",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.MaxConns,
		config.MinConns,
	)

	return dsn
}

func upMigration(url string, path string) error {
	migration, err := migrate.New(path, url)
	if err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	err = migration.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migration: %w", err)
	}

	return nil
}
