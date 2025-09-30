package minio

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

// TODO: connection timeout?
// TODO: reties, metrics, ssl

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
		return nil, fmt.Errorf("cannot create bucket %s: %w", config.BucketName, err)
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

func (s *Storage) PutObject(ctx context.Context, fileId string, reader io.Reader, size int64) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	_, err := s.mc.PutObject(ctx, s.bucketName, fileId, reader, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		s.logger.Error("UploadFile: cannot upload file", zap.Error(err))
		return fmt.Errorf("UploadFile: cannot upload file: %w", err)
	}

	s.logger.Info("UploadFile: uploaded successfully", zap.String("file_id", fileId))
	return nil
}
