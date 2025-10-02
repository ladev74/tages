package minio

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

// TODO: retries!
// TODO: circuit breaker?
// TODO: fatal on restart (bucket exists)
// TODO: connection timeout?
// TODO: reties, metrics, ssl

var (
	ErrNotFound = errors.New("object not found")
)

func New(ctx context.Context, config Config, logger *zap.Logger) (*Storage, error) {
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
		//return nil, fmt.Errorf("cannot create bucket %s: %w", config.BucketName, err)
	}

	exists, err := mc.BucketExists(ctx, config.BucketName)
	if err != nil {
		return nil, fmt.Errorf("cannot check if bucket exists: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("bucket %q does not exist", config.BucketName)
	}

	return &Storage{
		mc:         mc,
		bucketName: config.BucketName,
		timeout:    config.OperationTimeout,
		logger:     logger,
	}, nil
}

func (s *Storage) PutObject(ctx context.Context, id string, reader io.Reader, size int64) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	_, err := s.mc.PutObject(ctx, s.bucketName, id, reader, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		s.logger.Error("PutObject: cannot upload file", zap.Error(err))
		return fmt.Errorf("PutObject: cannot upload file: %w", err)
	}

	s.logger.Info("PutObject: successfully put object", zap.String("id", id))
	return nil
}

func (s *Storage) GetObject(ctx context.Context, id string) (io.ReadCloser, error) {
	//ctx, cancel := context.WithTimeout(ctx, s.timeout)
	//defer cancel()

	object, err := s.mc.GetObject(ctx, s.bucketName, id, minio.GetObjectOptions{})
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
