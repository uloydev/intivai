package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

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

// ValidateOrgObjectPath — C5 defense-in-depth: an object path is acceptable
// for orgID only when every dot-separated leading segment belongs to the org
// prefix (paths like `contexts/{orgID}/...` and `cvs/{orgID}/...`). Rejects:
//   - foreign/empty leading segment (other tenant's object),
//   - traversal (`..`) or absolute (`/`) components,
//   - empty path.
//
// Pure function, no I/O — unit-testable without MinIO.
func ValidateOrgObjectPath(path, orgID string) bool {
	if path == "" || orgID == "" {
		return false
	}
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "../") {
		return false
	}
	segments := strings.Split(path, "/")
	for i := 0; i < len(segments); i++ {
		s := segments[i]
		if s == "" || s == "." || s == ".." {
			return false
		}
	}
	// First two segments must be `{namespace}/{orgID}`.
	if len(segments) < 2 {
		return false
	}
	return segments[1] == orgID
}

// DownloadOrgScoped — Download plus org-scope enforcement: the object path
// must live under the tenant's prefix (`{namespace}/{orgID}/...`) before any
// storage I/O. Defense-in-depth for callers that already hold RLS-scoped
// paths (workers, interview connect, evaluation); a cross-tenant path can
// never reach MinIO.
func (s *Storage) DownloadOrgScoped(ctx context.Context, path, orgID string) (io.ReadCloser, error) {
	if !ValidateOrgObjectPath(path, orgID) {
		return nil, fmt.Errorf("storage: object path %q is not org %s scoped", path, orgID)
	}
	return s.Download(ctx, path)
}

// Upload writes an object, creating the bucket on first use (fresh volumes
// / new tenants must not require an app restart after EnsureBucket).
func (s *Storage) Upload(ctx context.Context, path string, r io.Reader, size int64, contentType string) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check bucket %s: %w", s.bucket, err)
	}
	if !exists {
		if err := s.EnsureBucket(ctx); err != nil {
			return fmt.Errorf("ensure bucket %s: %w", s.bucket, err)
		}
	}
	_, err = s.client.PutObject(ctx, s.bucket, path, r, size, minio.PutObjectOptions{
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

// Exists reports whether an object exists — GetObject returns a lazy reader
// that only errors on first Read, so callers must Stat before trusting it.
func (s *Storage) Exists(ctx context.Context, path string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, path, minio.StatObjectOptions{})
	if err == nil {
		return true, nil
	}
	var resp minio.ErrorResponse
	if errors.As(err, &resp) && resp.Code == "NoSuchKey" {
		return false, nil
	}
	return false, err
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
	Exists(ctx context.Context, path string) (bool, error)
}

var _ FileStorage = (*Storage)(nil)
