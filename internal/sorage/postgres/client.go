package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	statusPending = "pending"
	statusSuccess = "success"
)

func New(ctx context.Context, config *Config, logger *zap.Logger) (*Storage, error) {
	ctx, cancel := context.WithTimeout(ctx, config.Timeout)
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
		pool:        pool,
		logger:      logger,
		timeout:     config.Timeout,
		maxRetries:  config.MaxRetries,
		baseBackoff: config.BaseBackoff,
	}, nil
}

func (s *Storage) SaveFileInfo(ctx context.Context, id string, fileName string, createdAt time.Time, updatedAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	tag, err := withRetry(ctx, s.maxRetries, s.baseBackoff, s.logger, func() (pgconn.CommandTag, error) {
		tag, err := s.pool.Exec(ctx, querySaveFileInfo, id, fileName, createdAt, updatedAt, statusPending)
		return tag, err
	})
	if err != nil {
		s.logger.Error("Save: failed to insert file", zap.Error(err))
		return err
	}
	if tag.RowsAffected() == 0 {
		s.logger.Error("Save: no rows affected")
		return fmt.Errorf("Save: no rows affected")
	}

	s.logger.Info("Save: successfully inserted file", zap.String("file_id", id))
	return nil
}

func (s *Storage) SetSuccessStatus(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	tag, err := withRetry(ctx, s.maxRetries, s.baseBackoff, s.logger, func() (pgconn.CommandTag, error) {
		tag, err := s.pool.Exec(ctx, querySetSuccessStatus, statusSuccess, id)
		return tag, err
	})
	if err != nil {
		s.logger.Error("SetSuccessStatus: failed to set success status", zap.Error(err))
		return fmt.Errorf("SetSuccessStatus: failed to set success status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		s.logger.Error("SetSuccessStatus: no rows affected")
		return fmt.Errorf("SetSuccessStatus: no rows affected")
	}

	s.logger.Info("SetSuccessStatus: successfully set success status", zap.String("id", id))
	return nil
}

func (s *Storage) ListFilesInfo(ctx context.Context, limit int64, offset int64) ([]*fileservice.FileInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	rows, err := withRetry(ctx, s.maxRetries, s.baseBackoff, s.logger, func() (pgx.Rows, error) {
		rows, err := s.pool.Query(ctx, queryListFilesInfo, limit, offset)
		return rows, err
	})
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

	tag, err := withRetry(ctx, s.maxRetries, s.baseBackoff, s.logger, func() (pgconn.CommandTag, error) {
		tag, err := s.pool.Exec(ctx, queryDeleteFileInfo, id)
		return tag, err
	})
	if err != nil {
		s.logger.Error("Delete: failed to delete file info", zap.String("id", id), zap.Error(err))
		return fmt.Errorf("Delete: failed to delete file info")
	}

	if tag.RowsAffected() == 0 {
		s.logger.Error("SetSuccessStatus: no rows affected")
		return fmt.Errorf("SetSuccessStatus: no rows affected")
	}

	s.logger.Info("Delete: successfully deleted file info", zap.String("id", id))
	return nil
}

func (s *Storage) GetFileName(ctx context.Context, id string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	var fileName string

	_, err := withRetry(ctx, s.maxRetries, s.baseBackoff, s.logger, func() (struct{}, error) {
		err := s.pool.QueryRow(ctx, queryGetFIleName, id).Scan(&fileName)
		return struct{}{}, err
	})
	if err != nil {
		s.logger.Error("GetFileName: failed to get file info", zap.String("id", id), zap.Error(err))
		return "", fmt.Errorf("GetFileName: failed to get file info: %w", err)
	}

	s.logger.Info("GetFileName: successfully retrieved file info", zap.String("id", id))
	return fileName, nil
}

func (s *Storage) Close() {
	s.pool.Close()
}

func withRetry[T any](ctx context.Context, maxRetries int, baseBackoff time.Duration, logger *zap.Logger, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		res, err := fn()
		if err == nil {
			return res, nil
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "42P01" {
				logger.Error("withRetry: non-retryable Postgres error", zap.Error(err))
				return zero, err
			}
		}

		lastErr = err

		if i == maxRetries-1 {
			break
		}

		backoff := baseBackoff * time.Duration(math.Pow(2, float64(i)))
		jitter := time.Duration(rand.Float64() * float64(baseBackoff))
		pause := backoff + jitter

		select {
		case <-time.After(pause):
		case <-ctx.Done():
			logger.Error("withRetry: context canceled", zap.Int("attempts", i+1), zap.Duration("backoff", baseBackoff))
			return zero, ctx.Err()
		}

		logger.Warn("withRetry: retrying", zap.Int("attempt", i+1), zap.Duration("backoff", pause))
	}

	return zero, fmt.Errorf("withRetry: all retries failed, lastErr: %w", lastErr)
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
