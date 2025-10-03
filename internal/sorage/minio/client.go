package minio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

// TODO: circuit breaker?
// TODO: ssl

var (
	ErrNotFound = errors.New("object not found")
)

func New(ctx context.Context, config Config, logger *zap.Logger) (*Storage, error) {
	ctx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	mc, err := minio.New(addr, &minio.Options{
		Creds:  credentials.NewStaticV4(config.User, config.Password, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot initialize minio client: %w", err)
	}

	err = mc.MakeBucket(ctx, config.BucketName, minio.MakeBucketOptions{})
	if err != nil {
		resp := minio.ToErrorResponse(err)
		if resp.Code != "BucketAlreadyOwnedByYou" {
			return nil, fmt.Errorf("cannot create bucket %s: %w", config.BucketName, err)
		}
	}

	exists, err := mc.BucketExists(ctx, config.BucketName)
	if err != nil {
		return nil, fmt.Errorf("cannot check if bucket exists: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("bucket %q does not exist", config.BucketName)
	}

	return &Storage{
		mc:          mc,
		bucketName:  config.BucketName,
		timeout:     config.Timeout,
		logger:      logger,
		maxRetries:  config.MaxRetries,
		baseBackoff: config.BaseBackoff,
	}, nil
}

func (s *Storage) PutObject(ctx context.Context, id string, reader io.Reader, size int64) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	_, err := withRetry(ctx, s.maxRetries, s.baseBackoff, s.logger, func() (struct{}, error) {
		_, err := s.mc.PutObject(ctx, s.bucketName, id, reader, size, minio.PutObjectOptions{
			ContentType: "application/octet-stream",
		})
		return struct{}{}, err
	})
	if err != nil {
		s.logger.Error("PutObject: cannot upload file", zap.Error(err))
		return fmt.Errorf("PutObject: cannot upload file: %w", err)
	}

	s.logger.Info("PutObject: successfully put object", zap.String("id", id))
	return nil
}

func (s *Storage) GetObject(ctx context.Context, id string) (io.ReadCloser, error) {
	object, err := withRetry(ctx, s.maxRetries, s.baseBackoff, s.logger, func() (*minio.Object, error) {
		return s.mc.GetObject(ctx, s.bucketName, id, minio.GetObjectOptions{})
	})
	if err != nil {
		resp := minio.ToErrorResponse(err)
		if resp.Code == minio.NoSuchKey {
			s.logger.Warn("GetObject: object not found", zap.String("id", id))
			return nil, fmt.Errorf("GetObject: %w: %s", ErrNotFound, id)
		}

		s.logger.Error("GetObject: failed to get object", zap.String("id", id), zap.Error(err))
		return nil, fmt.Errorf("GetObject: failed to get object: %w", err)
	}

	s.logger.Info("GetObject: successfully get object", zap.String("id", id))
	return object, nil
}

func withRetry[T any](ctx context.Context, maxRetries int, baseBackoff time.Duration, logger *zap.Logger, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		res, err := fn()
		switch {
		case err == nil:
			return res, nil
		case errors.Is(err, ErrNotFound):
			return zero, ErrNotFound
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
