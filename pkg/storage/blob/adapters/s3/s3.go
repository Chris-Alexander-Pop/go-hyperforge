package s3

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob"
)

// Store implements Store using AWS S3
type Store struct {
	client     *s3.Client
	bucket     string
	uploader   *manager.Uploader
	downloader *manager.Downloader
}

// New creates a new S3Store
func New(ctx context.Context, cfg blob.Config) (*Store, error) {
	if cfg.Bucket == "" {
		return nil, errors.New(errors.CodeInvalidArgument, "bucket name is required", nil)
	}

	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load aws config")
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true // Needed for MinIO often
		}
	})

	return &Store{
		client:     client,
		bucket:     cfg.Bucket,
		uploader:   manager.NewUploader(client),
		downloader: manager.NewDownloader(client),
	}, nil
}

func (s *Store) Upload(ctx context.Context, key string, data io.Reader) error {
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   data,
	})
	if err != nil {
		return errors.Internal("failed to upload to s3", err)
	}
	return nil
}

func (s *Store) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	// For streaming large files, GetObject is fine directly, but Downloader is efficient for chunks.
	// However, Downloader requires a WriterAt, which isn't io.ReadCloser friendly for streaming back to user.
	// So we use standard GetObject here for streaming interface.

	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check for NoSuchKey
		// In v2 we check error types
		// Simple approach for now:
		return nil, errors.Internal("failed to download from s3", err)
	}

	return out.Body, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return errors.Internal("failed to delete from s3", err)
	}
	return nil
}

func (s *Store) URL(key string) string {
	// naive public URL
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, key)
}
