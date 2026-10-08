package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type ObjectStorage interface {
	Put(ctx context.Context, key, contentType string, body []byte) (string, error)
}

type Config struct {
	Driver        string
	LocalDir      string
	PublicBaseURL string
	S3Bucket      string
	S3Region      string
	S3Endpoint    string
	S3AccessKey   string
	S3SecretKey   string
}

func ConfigFromEnv() Config {
	return Config{
		Driver:        envOrDefault("STORAGE_DRIVER", "local"),
		LocalDir:      envOrDefault("MEDIA_LOCAL_DIR", "./data/media"),
		PublicBaseURL: envOrDefault("MEDIA_PUBLIC_BASE_URL", "http://localhost:8080/media"),
		S3Bucket:      os.Getenv("S3_BUCKET"),
		S3Region:      envOrDefault("S3_REGION", "auto"),
		S3Endpoint:    os.Getenv("S3_ENDPOINT"),
		S3AccessKey:   os.Getenv("S3_ACCESS_KEY_ID"),
		S3SecretKey:   os.Getenv("S3_SECRET_ACCESS_KEY"),
	}
}

func New(ctx context.Context, cfg Config) (ObjectStorage, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Driver)) {
	case "", "local":
		return NewLocal(cfg.LocalDir, cfg.PublicBaseURL)
	case "s3", "r2":
		return NewS3(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported STORAGE_DRIVER %q", cfg.Driver)
	}
}

type Local struct {
	dir           string
	publicBaseURL string
}

func NewLocal(dir, publicBaseURL string) (*Local, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("MEDIA_LOCAL_DIR is required for local storage")
	}
	if strings.TrimSpace(publicBaseURL) == "" {
		return nil, fmt.Errorf("MEDIA_PUBLIC_BASE_URL is required for local storage")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create media directory: %w", err)
	}
	return &Local{dir: dir, publicBaseURL: strings.TrimRight(publicBaseURL, "/")}, nil
}

func (s *Local) Put(_ context.Context, key, _ string, body []byte) (string, error) {
	cleanKey, err := safeKey(key)
	if err != nil {
		return "", err
	}
	filename := filepath.Join(s.dir, filepath.FromSlash(cleanKey))
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return "", fmt.Errorf("create media path: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(filename), ".upload-*")
	if err != nil {
		return "", fmt.Errorf("create temporary media object: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := io.Copy(tmp, bytes.NewReader(body)); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("write media object: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close media object: %w", err)
	}
	if err := os.Rename(tmpName, filename); err != nil {
		return "", fmt.Errorf("publish media object: %w", err)
	}

	return s.publicBaseURL + "/" + cleanKey, nil
}

func (s *Local) Directory() string {
	return s.dir
}

type S3 struct {
	client        *s3.Client
	bucket        string
	publicBaseURL string
}

func NewS3(ctx context.Context, cfg Config) (*S3, error) {
	if strings.TrimSpace(cfg.S3Bucket) == "" {
		return nil, fmt.Errorf("S3_BUCKET is required for S3-compatible storage")
	}
	if strings.TrimSpace(cfg.PublicBaseURL) == "" {
		return nil, fmt.Errorf("MEDIA_PUBLIC_BASE_URL is required for S3-compatible storage")
	}
	if strings.TrimSpace(cfg.S3AccessKey) == "" || strings.TrimSpace(cfg.S3SecretKey) == "" {
		return nil, fmt.Errorf("S3_ACCESS_KEY_ID and S3_SECRET_ACCESS_KEY are required for S3-compatible storage")
	}

	awsConfig, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(cfg.S3Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load S3 configuration: %w", err)
	}

	client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		if strings.TrimSpace(cfg.S3Endpoint) != "" {
			options.BaseEndpoint = aws.String(cfg.S3Endpoint)
			options.UsePathStyle = true
		}
	})
	return &S3{client: client, bucket: cfg.S3Bucket, publicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/")}, nil
}

func (s *S3) Put(ctx context.Context, key, contentType string, body []byte) (string, error) {
	cleanKey, err := safeKey(key)
	if err != nil {
		return "", err
	}
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(s.bucket),
		Key:          aws.String(cleanKey),
		Body:         bytes.NewReader(body),
		ContentType:  aws.String(contentType),
		CacheControl: aws.String("public, max-age=31536000, immutable"),
	})
	if err != nil {
		return "", fmt.Errorf("put S3 object: %w", err)
	}
	return s.publicBaseURL + "/" + cleanKey, nil
}

func safeKey(key string) (string, error) {
	key = strings.TrimPrefix(strings.TrimSpace(key), "/")
	if key == "" || strings.Contains(key, "\\") || path.Clean(key) != key || strings.HasPrefix(key, "../") {
		return "", fmt.Errorf("invalid object key")
	}
	return key, nil
}

func PublicURL(baseURL, key string) (string, error) {
	cleanKey, err := safeKey(key)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/") + "/" + cleanKey)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid public media URL")
	}
	return parsed.String(), nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
