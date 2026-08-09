package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage struct {
	client *minio.Client
	bucket string
}

func New(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*Storage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	return &Storage{client: client, bucket: bucket}, nil
}

func (s *Storage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("bucket exists: %w", err)
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			// Concurrent first uploads race MakeBucket — the loser sees
			// BucketAlreadyOwned; the bucket exists, that's success.
			if existsNow, e2 := s.client.BucketExists(ctx, s.bucket); e2 == nil && existsNow {
				return nil
			}
			return fmt.Errorf("make bucket: %w", err)
		}
	}
	return nil
}

// Upload writes an object, creating the bucket on first use (fresh volumes
// / new tenants must not require an app restart after EnsureBucket).
func (s *Storage) Upload(ctx context.Context, path string, r io.Reader, size int64, contentType string) error {
	if exists, err := s.client.BucketExists(ctx, s.bucket); err == nil && !exists {
		if err := s.EnsureBucket(ctx); err != nil {
			return fmt.Errorf("ensure bucket %s: %w", s.bucket, err)
		}
	}
	_, err := s.client.PutObject(ctx, s.bucket, path, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *Storage) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (s *Storage) Delete(ctx context.Context, path string) error {
	return s.client.RemoveObject(ctx, s.bucket, path, minio.RemoveObjectOptions{})
}

func (s *Storage) Ping(ctx context.Context) error {
	_, err := s.client.ListBuckets(ctx)
	return err
}

// FileStorage port from the architecture docs — Storage implements it.
type FileStorage interface {
	Upload(ctx context.Context, path string, r io.Reader, size int64, contentType string) error
	Download(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
}

var _ FileStorage = (*Storage)(nil)
