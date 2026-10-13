package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/troubleman96/ziadapos/internal/config"
)

// Storage wraps a MinIO/S3-compatible client for all object storage needs:
// product images, organisation logos, report exports, receipts.
type Storage struct {
	client  *minio.Client
	bucket  string
	public  string
	useSSL  bool
	endpoint string
}

// New connects to MinIO and ensures the bucket exists.
func New(ctx context.Context, cfg *config.Config) (*Storage, error) {
	client, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: cfg.MinIOUseSSL,
		Region: cfg.MinIORegion,
	})
	if err != nil {
		return nil, err
	}

	s := &Storage{
		client:   client,
		bucket:   cfg.MinIOBucket,
		public:   strings.TrimRight(cfg.PublicBaseURL, "/"),
		useSSL:   cfg.MinIOUseSSL,
		endpoint: cfg.MinIOEndpoint,
	}

	if err := s.ensureBucket(ctx); err != nil {
		log.Printf("storage: could not ensure bucket %q (%v) - uploads will fail until MinIO is up", cfg.MinIOBucket, err)
	} else {
		log.Printf("storage: connected to MinIO (%s) bucket %q", cfg.MinIOEndpoint, cfg.MinIOBucket)
	}
	return s, nil
}

func (s *Storage) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: s.endpointRegion()})
	}
	return nil
}

func (s *Storage) endpointRegion() string {
	if r := strings.SplitN(s.endpoint, ":", 2); len(r) == 2 {
		return r[0]
	}
	return "us-east-1"
}

// Put uploads raw bytes under a key and returns the public URL.
func (s *Storage) Put(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return s.URL(key), nil
}

// Upload streams from a reader to a key (used for large exports).
func (s *Storage) Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", err
	}
	return s.URL(key), nil
}

// Get streams an object's contents. Returns minio.ErrNoSuchObject if missing.
func (s *Storage) Get(ctx context.Context, key string) (io.Reader, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Delete removes an object by key. Missing objects are treated as success.
func (s *Storage) Delete(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

// PresignedURL returns a temporary, time-limited download URL.
func (s *Storage) PresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// URL builds a stable public URL for a key. When a public base URL is
// configured (e.g. a CDN in front of MinIO) it is preferred.
func (s *Storage) URL(key string) string {
	if key == "" {
		return ""
	}
	if s.public != "" {
		return s.public + "/" + strings.TrimLeft(key, "/")
	}
	scheme := "http"
	if s.useSSL {
		scheme = "https"
	}
	u := url.URL{Scheme: scheme, Host: s.endpoint, Path: "/" + path.Join(s.bucket, key)}
	return u.String()
}

// BucketName exposes the configured bucket for admin/debug endpoints.
func (s *Storage) BucketName() string { return s.bucket }

// IsEmpty guards against accidental empty-key operations.
func (s *Storage) IsEmpty(key string) bool { return strings.TrimSpace(key) == "" }

func (s *Storage) String() string {
	return fmt.Sprintf("storage(minio://%s/%s)", s.endpoint, s.bucket)
}
