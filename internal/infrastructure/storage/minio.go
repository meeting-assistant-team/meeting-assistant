package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/johnquangdev/meeting-assistant/pkg/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOClient wraps MinIO operations
type MinIOClient struct {
	client    *minio.Client
	bucket    string
	publicURL string // Public URL for generating accessible URLs (e.g., https://minio.example.com)
}

// NewMinIOClient creates a new MinIO client
func NewMinIOClient(cfg *config.StorageConfig) (*MinIOClient, error) {
	// Initialize MinIO client
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	client := &MinIOClient{
		client:    minioClient,
		bucket:    cfg.BucketName,
		publicURL: cfg.PublicURL,
	}

	// Initialize bucket with public read policy
	ctx := context.Background()
	if err := client.ensureBucketWithPolicy(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize bucket: %w", err)
	}

	return client, nil
}

// ensureBucketWithPolicy ensures bucket exists and has public read policy
func (m *MinIOClient) ensureBucketWithPolicy(ctx context.Context) error {
	// Check if bucket exists
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	// Create bucket if it doesn't exist
	if !exists {
		err = m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	// Set public read policy for the bucket
	// This allows presigned URLs to work and enables AssemblyAI to download files
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, m.bucket)

	err = m.client.SetBucketPolicy(ctx, m.bucket, policy)
	if err != nil {
		return fmt.Errorf("failed to set bucket policy: %w", err)
	}

	return nil
}

// UploadFile uploads a file to MinIO
func (m *MinIOClient) UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error {
	// Upload file
	_, err := m.client.PutObject(ctx, m.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}

// UploadText uploads text content to MinIO
func (m *MinIOClient) UploadText(ctx context.Context, objectName string, content string) error {
	reader := bytes.NewReader([]byte(content))
	return m.UploadFile(ctx, objectName, reader, int64(len(content)), "text/plain")
}

// GetFileURL gets a public URL for accessing a file
// Since the bucket has public read policy, we return direct URL without signature
func (m *MinIOClient) GetFileURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	// If public URL is configured, return direct public URL (no signature needed)
	// This is useful when MinIO is behind a reverse proxy with public bucket policy
	if m.publicURL != "" {
		// Format: https://minio.infoquang.id.vn/bucket-name/object-path
		publicURL := fmt.Sprintf("%s/%s/%s", m.publicURL, m.bucket, objectName)
		return publicURL, nil
	}

	// Fallback to standard presigned URL
	url, err := m.client.PresignedGetObject(ctx, m.bucket, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url.String(), nil
}

// ListFiles lists all files in the bucket
func (m *MinIOClient) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string

	// List objects in bucket with prefix
	objectCh := m.client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects: %w", object.Err)
		}
		files = append(files, object.Key)
	}

	return files, nil
}

// GetBucketInfo returns information about the bucket and connection
func (m *MinIOClient) GetBucketInfo(ctx context.Context) (map[string]interface{}, error) {
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}

	info := map[string]interface{}{
		"bucket":        m.bucket,
		"bucket_exists": exists,
		"endpoint":      m.client.EndpointURL().String(),
	}

	if exists {
		// Count objects
		files, err := m.ListFiles(ctx, "")
		if err != nil {
			info["error"] = err.Error()
		} else {
			info["total_files"] = len(files)
		}
	}

	return info, nil
}
