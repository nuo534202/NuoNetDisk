package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type FileStorage interface {
	Upload(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, objectKey string) (io.ReadCloser, error)
	Stat(ctx context.Context, objectKey string) (size int64, contentType string, err error)
	Delete(ctx context.Context, objectKey string) error
	Copy(ctx context.Context, srcKey, dstKey string) error
	HealthCheck(ctx context.Context) error
}

type ObjectInfo struct {
	Size        int64
	ContentType string
}

type MinioStorage struct {
	client *minio.Client
	bucket string
}

func NewMinioStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinioStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	ctx := context.Background()
	bucketOpts := minio.MakeBucketOptions{}

	if err := client.MakeBucket(ctx, bucket, bucketOpts); err != nil {
		exists, errExists := client.BucketExists(ctx, bucket)
		if errExists != nil || !exists {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &MinioStorage{
		client: client,
		bucket: bucket,
	}, nil
}

func (s *MinioStorage) Upload(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, objectKey, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}
	return nil
}

func (s *MinioStorage) Download(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download object: %w", err)
	}
	return obj, nil
}

func (s *MinioStorage) Stat(ctx context.Context, objectKey string) (int64, string, error) {
	info, err := s.client.StatObject(ctx, s.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return 0, "", fmt.Errorf("failed to stat object: %w", err)
	}
	return info.Size, info.ContentType, nil
}

func (s *MinioStorage) Delete(ctx context.Context, objectKey string) error {
	err := s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

func (s *MinioStorage) HealthCheck(ctx context.Context) error {
	_, err := s.client.BucketExists(ctx, s.bucket)
	return err
}

func (s *MinioStorage) Copy(ctx context.Context, srcKey, dstKey string) error {
	src := minio.CopySrcOptions{
		Bucket: s.bucket,
		Object: srcKey,
	}
	dst := minio.CopyDestOptions{
		Bucket: s.bucket,
		Object: dstKey,
	}

	_, err := s.client.CopyObject(ctx, dst, src)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}
	return nil
}
