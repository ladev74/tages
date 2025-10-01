package postgres

import (
	"context"
	"fmt"
	"time"

	_ "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TODO: retries!
// TODO: circuit breaker?

func New(ctx context.Context, config *Config, logger *zap.Logger) (*Storage, error) {
	ctx, cancel := context.WithTimeout(ctx, config.ConnectionTimeout)
	defer cancel()

	dsn := buildDSN(config)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &Storage{
		pool:    pool,
		logger:  logger,
		timeout: config.OperationTimeout,
	}, nil
}

func (s *Storage) SaveFileInfo(ctx context.Context, id string, fileName string, createdAt time.Time, updatedAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	tag, err := s.pool.Exec(ctx, querySaveFileInfo, id, fileName, createdAt, updatedAt, statusPending)
	if err != nil {
		s.logger.Error("Save: failed to insert file", zap.Error(err))
		return err
	}
	if tag.RowsAffected() == 0 {
		s.logger.Error("Save: no rows affected", zap.Error(err))
		return fmt.Errorf("Save: no rows affected")
	}

	s.logger.Info("Save: successfully inserted file", zap.String("file_id", id))
	return nil
}

func (s *Storage) SetSuccessStatus(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	tag, err := s.pool.Exec(ctx, querySetSuccessStatus, statusSuccess, id)
	if err != nil {
		s.logger.Error("SetSuccessStatus: failed to set success status", zap.Error(err))
		return fmt.Errorf("SetSuccessStatus: failed to set success status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		s.logger.Error("SetSuccessStatus: no rows affected", zap.Error(err))
		return fmt.Errorf("SetSuccessStatus: no rows affected")
	}

	s.logger.Info("SetSuccessStatus: successfully set success status", zap.String("id", id))
	return nil
}

func (s *Storage) ListFilesInfo(ctx context.Context, limit int64, offset int64) ([]*fileservice.FileInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	rows, err := s.pool.Query(ctx, queryListFilesInfo, limit, offset)
	if err != nil {
		s.logger.Error("ListFilesInfo: failed to get files", zap.Error(err))
		return nil, fmt.Errorf("ListFilesInfo: failed to get files: %w", err)
	}

	defer rows.Close()
	files := make([]*fileservice.FileInfo, 0, limit)

	for rows.Next() {
		file := &fileservice.FileInfo{}
		var tempCreatedAt time.Time
		var tempUpdatedAt time.Time

		err = rows.Scan(&file.Name, &tempCreatedAt, &tempUpdatedAt)
		if err != nil {
			s.logger.Error("ListFilesInfo: failed to scan files", zap.Error(err))
			return nil, fmt.Errorf("ListFilesInfo: failed to scan files: %w", err)
		}

		file.CreatedAt = timestamppb.New(tempCreatedAt)
		file.UpdatedAt = timestamppb.New(tempUpdatedAt)

		files = append(files, file)
	}

	err = rows.Err()
	if err != nil {
		s.logger.Error("ListFilesInfo: failed to scan files", zap.Error(err))
		return nil, fmt.Errorf("ListFilesInfo: failed to scan files: %w", err)
	}

	s.logger.Info("ListFilesInfo: successfully retrieved files", zap.Int("count", len(files)))
	return files, nil
}

func (s *Storage) DeleteFileInfo(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	_, err := s.pool.Exec(ctx, queryDeleteFileInfo, id)
	if err != nil {
		s.logger.Error("Delete: failed to delete file info", zap.String("id", id), zap.Error(err))
		return fmt.Errorf("Delete: failed to delete file info")
	}

	s.logger.Info("Delete: successfully deleted file info", zap.String("id", id))
	return nil
}

func (s *Storage) Close() {
	s.pool.Close()
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
