package s3

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	client *minio.Client
	bucket string
)

// Init connects to S3 and verifies the bucket exists.
// Reads config from environment: S3_ENDPOINT, S3_ACCESS_KEY, S3_SECRET_KEY, S3_BUCKET.
// Call this at startup — fails fast if S3 is unreachable or misconfigured.
func Init(ctx context.Context) error {
	endpoint := os.Getenv("S3_ENDPOINT")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	bucket = os.Getenv("S3_BUCKET")

	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return fmt.Errorf("S3 config incomplete: S3_ENDPOINT, S3_ACCESS_KEY, S3_SECRET_KEY, S3_BUCKET are all required")
	}

	// Parse endpoint: strip scheme for minio client, detect TLS
	useSSL := strings.HasPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")

	var err error
	client, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return fmt.Errorf("S3 client init failed: %w", err)
	}

	// Connect to S3 — retry for up to 30s (compose services may still be starting)
	for attempts := 0; attempts < 15; attempts++ {
		exists, err := client.BucketExists(ctx, bucket)
		if err == nil {
			if !exists {
				// Auto-create bucket (dev mode, first run)
				if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
					return fmt.Errorf("S3 bucket creation failed: %w", err)
				}
				slog.Info("S3 bucket created", "bucket", bucket)
			}
			slog.Info("S3 connected", "endpoint", os.Getenv("S3_ENDPOINT"), "bucket", bucket)
			return nil
		}
		slog.Info("S3 not ready, retrying...", "attempt", attempts+1, "error", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("S3 not reachable after 30s")
}

// HealthCheck verifies S3 is still reachable.
func HealthCheck(ctx context.Context) error {
	if client == nil {
		return fmt.Errorf("S3 not initialized")
	}
	_, err := client.BucketExists(ctx, bucket)
	return err
}

// Put stores content by SHA-256 hash. Idempotent — overwrites are identical by definition.
func Put(ctx context.Context, sha256 string, reader io.Reader, size int64, contentType string) error {
	_, err := client.PutObject(ctx, bucket, sha256, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

// Get retrieves content by SHA-256 hash.
func Get(ctx context.Context, sha256 string) (io.ReadCloser, int64, string, error) {
	obj, err := client.GetObject(ctx, bucket, sha256, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, "", err
	}
	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, 0, "", err
	}
	return obj, info.Size, info.ContentType, nil
}

// Head checks if a blob exists and returns its size.
func Head(ctx context.Context, sha256 string) (int64, bool, error) {
	info, err := client.StatObject(ctx, bucket, sha256, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return 0, false, nil
		}
		return 0, false, err
	}
	return info.Size, true, nil
}
