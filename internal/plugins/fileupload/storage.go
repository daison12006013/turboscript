/*
 * TurboScript - A hybrid web framework combining TypeScript and Go
 *
 * Storage abstraction layer for file upload plugin
 */

package fileupload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// StorageBackend defines the interface for different storage implementations.
type StorageBackend interface {
	// Store uploads a file and returns the storage key/path
	Store(ctx context.Context, key string, data []byte, contentType string) error

	// Get retrieves a file by its key
	Get(ctx context.Context, key string) ([]byte, error)

	// Delete removes a file by its key
	Delete(ctx context.Context, key string) error

	// GetURL returns a public URL for the file (if supported)
	GetURL(key string) string

	// GetPresignedURL returns a presigned URL for upload/download
	GetPresignedURL(ctx context.Context, key string, operation string, expiry time.Duration) (string, error)

	// Exists checks if a file exists
	Exists(ctx context.Context, key string) (bool, error)

	// GetInfo returns file information
	GetInfo(ctx context.Context, key string) (StorageFileInfo, error)
}

// StorageFileInfo represents file information from storage.
type StorageFileInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"contentType"`
	LastModified time.Time `json:"lastModified"`
	ETag         string    `json:"etag"`
}

// LocalStorageBackend implements local filesystem storage.
type LocalStorageBackend struct {
	basePath string
	baseURL  string
}

// NewLocalStorageBackend creates a new local storage backend.
func NewLocalStorageBackend(basePath, baseURL string) *LocalStorageBackend {
	return &LocalStorageBackend{
		basePath: basePath,
		baseURL:  baseURL,
	}
}

func (l *LocalStorageBackend) Store(_ctx context.Context, key string, data []byte, contentType string) error {
	fullPath := filepath.Join(l.basePath, key)

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := ensureDir(dir); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return writeFile(fullPath, data)
}

func (l *LocalStorageBackend) Get(_ctx context.Context, key string) ([]byte, error) {
	fullPath := filepath.Join(l.basePath, key)
	return readFile(fullPath)
}

func (l *LocalStorageBackend) Delete(_ctx context.Context, key string) error {
	fullPath := filepath.Join(l.basePath, key)
	return removeFile(fullPath)
}

// GetURL returns the public URL for accessing a file in local storage.
func (l *LocalStorageBackend) GetURL(key string) string {
	if l.baseURL == "" {
		return "/uploads/" + key
	}
	return l.baseURL + "/" + key
}

func (l *LocalStorageBackend) GetPresignedURL(_ctx context.Context, key string, operation string, expiry time.Duration) (string, error) {
	// Local storage doesn't support presigned URLs in the traditional sense
	// Return the regular URL
	return l.GetURL(key), nil
}

func (l *LocalStorageBackend) Exists(_ctx context.Context, key string) (bool, error) {
	fullPath := filepath.Join(l.basePath, key)
	return fileExists(fullPath), nil
}

func (l *LocalStorageBackend) GetInfo(_ctx context.Context, key string) (StorageFileInfo, error) {
	fullPath := filepath.Join(l.basePath, key)
	stat, err := getFileStat(fullPath)
	if err != nil {
		return StorageFileInfo{}, err
	}

	return StorageFileInfo{
		Key:          key,
		Size:         stat.Size(),
		LastModified: stat.ModTime(),
		ContentType:  getContentType(key),
	}, nil
}

// S3StorageBackend implements S3-compatible storage (AWS S3, MinIO, etc.)
type S3StorageBackend struct {
	client     *minio.Client
	bucketName string
	region     string
	baseURL    string
	useSSL     bool
}

// S3Config represents S3 storage configuration.
type S3Config struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	BucketName      string `json:"bucket_name"`
	Region          string `json:"region"`
	UseSSL          bool   `json:"use_ssl"`
	BaseURL         string `json:"base_url"`
}

// NewS3StorageBackend creates a new S3-compatible storage backend.
func NewS3StorageBackend(config S3Config) (*S3StorageBackend, error) {
	// Initialize MinIO client
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
		Region: config.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	return &S3StorageBackend{
		client:     client,
		bucketName: config.BucketName,
		region:     config.Region,
		baseURL:    config.BaseURL,
		useSSL:     config.UseSSL,
	}, nil
}

// Store uploads a file to S3-compatible storage.
func (s *S3StorageBackend) Store(ctx context.Context, key string, data []byte, contentType string) error {
	// Ensure bucket exists
	if err := s.EnsureBucket(ctx); err != nil {
		return err
	}

	reader := bytes.NewReader(data)

	_, err := s.client.PutObject(ctx, s.bucketName, key, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})

	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// Get retrieves a file from S3-compatible storage.
func (s *S3StorageBackend) Get(ctx context.Context, key string) ([]byte, error) {
	object, err := s.client.GetObject(ctx, s.bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer func() {
		if closeErr := object.Close(); closeErr != nil {
			log.Printf("Failed to close S3 object: %v", closeErr)
		}
	}()

	data, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}

	return data, nil
}

// Delete removes a file from S3-compatible storage.
func (s *S3StorageBackend) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucketName, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object from S3: %w", err)
	}

	return nil
}

// GetURL returns the public URL for accessing a file in S3-compatible storage.
func (s *S3StorageBackend) GetURL(key string) string {
	if s.baseURL != "" {
		return s.baseURL + "/" + key
	}

	// Generate standard S3 URL
	protocol := "https"
	if !s.useSSL {
		protocol = "http"
	}

	return fmt.Sprintf("%s://%s/%s/%s", protocol, s.client.EndpointURL().Host, s.bucketName, key)
}

// GetPresignedURL generates a presigned URL for S3-compatible storage operations.
func (s *S3StorageBackend) GetPresignedURL(ctx context.Context, key string, operation string, expiry time.Duration) (string, error) {
	switch operation {
	case "GET", "download":
		url, err := s.client.PresignedGetObject(ctx, s.bucketName, key, expiry, nil)
		if err != nil {
			return "", fmt.Errorf("failed to generate presigned GET URL: %w", err)
		}
		return url.String(), nil

	case "PUT", "upload":
		url, err := s.client.PresignedPutObject(ctx, s.bucketName, key, expiry)
		if err != nil {
			return "", fmt.Errorf("failed to generate presigned PUT URL: %w", err)
		}
		return url.String(), nil

	default:
		return "", fmt.Errorf("unsupported operation: %s", operation)
	}
}

// Exists checks if a file exists in S3-compatible storage.
func (s *S3StorageBackend) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucketName, key, minio.StatObjectOptions{})
	if err != nil {
		// Check if error is "object not found"
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// GetInfo retrieves file information from S3-compatible storage.
func (s *S3StorageBackend) GetInfo(ctx context.Context, key string) (StorageFileInfo, error) {
	stat, err := s.client.StatObject(ctx, s.bucketName, key, minio.StatObjectOptions{})
	if err != nil {
		return StorageFileInfo{}, fmt.Errorf("failed to get object info: %w", err)
	}

	return StorageFileInfo{
		Key:          key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		LastModified: stat.LastModified,
		ETag:         stat.ETag,
	}, nil
}

// EnsureBucket ensures bucket exists and creates if necessary.
func (s *S3StorageBackend) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = s.client.MakeBucket(ctx, s.bucketName, minio.MakeBucketOptions{
			Region: s.region,
		})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}
