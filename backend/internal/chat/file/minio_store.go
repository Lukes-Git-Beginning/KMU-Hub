package file

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStore implements FileStore using MinIO/S3.
// publicClient, when set, is used exclusively for presigned URL generation so
// that the resulting URLs carry the browser-reachable host (e.g.
// s3.zentria.tech) instead of the internal docker-network address.
type MinIOStore struct {
	client       *minio.Client
	publicClient *minio.Client // nil → fall back to client for presign
	bucket       string
}

// NewMinIOStore creates a new MinIO file store and ensures the bucket exists.
func NewMinIOStore(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinIOStore, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	// Ensure bucket exists
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
		slog.Info("created minio bucket", "bucket", bucket)
	}

	return &MinIOStore{
		client: client,
		bucket: bucket,
	}, nil
}

// NewMinIOStoreWithPublicEndpoint creates a MinIO file store like NewMinIOStore
// but additionally builds a second minio-go client bound to publicEndpoint.
// The public client is used only for presigning (GetPresignedURL /
// GetPresignedUploadURL) so the resulting URLs are browser-reachable.
// All other operations (Upload, Download, Delete, …) use the internal client.
// If publicEndpoint is empty the function behaves identically to NewMinIOStore.
func NewMinIOStoreWithPublicEndpoint(
	endpoint, accessKey, secretKey, bucket string, useSSL bool,
	publicEndpoint string, publicUseSSL bool,
) (*MinIOStore, error) {
	store, err := NewMinIOStore(endpoint, accessKey, secretKey, bucket, useSSL)
	if err != nil {
		return nil, err
	}
	if publicEndpoint == "" {
		return store, nil
	}
	pubClient, err := minio.New(publicEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: publicUseSSL,
	})
	if err != nil {
		return nil, err
	}
	slog.Info("minio public presign client configured", "public_endpoint", publicEndpoint)
	store.publicClient = pubClient
	return store, nil
}

// presignClient returns the client to use for generating presigned URLs.
// If a public-endpoint client has been configured it is preferred so the
// resulting URL carries the browser-reachable hostname.
func (s *MinIOStore) presignClient() *minio.Client {
	if s.publicClient != nil {
		return s.publicClient
	}
	return s.client
}

func (s *MinIOStore) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *MinIOStore) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
}

func (s *MinIOStore) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *MinIOStore) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	url, err := s.presignClient().PresignedGetObject(ctx, s.bucket, key, expiry, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func (s *MinIOStore) GetPresignedUploadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	url, err := s.presignClient().PresignedPutObject(ctx, s.bucket, key, expiry)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}
