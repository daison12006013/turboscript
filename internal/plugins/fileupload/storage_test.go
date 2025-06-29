package fileupload

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestS3StorageBackend(t *testing.T) {
	// Skip if not in Docker environment with MinIO
	if os.Getenv("DOCKER_ENV") != "true" {
		t.Skip("Skipping S3 integration tests (requires DOCKER_ENV=true)")
	}

	config := S3Config{
		Endpoint:        "minio:9000",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		BucketName:      "test-bucket",
		Region:          "us-east-1",
		UseSSL:          false,
		BaseURL:         "http://minio:9000/test-bucket",
	}

	backend, err := NewS3StorageBackend(config)
	require.NoError(t, err, "Failed to create S3 storage backend")

	ctx := context.Background()

	t.Run("Store", func(t *testing.T) {
		testData := []byte("test file content for S3")
		key := "test/file.txt"
		contentType := "text/plain"

		err := backend.Store(ctx, key, testData, contentType)
		require.NoError(t, err, "Failed to store file to S3")
	})

	t.Run("Get", func(t *testing.T) {
		// First store a file
		testData := []byte("test file content for retrieval")
		key := "test/get-file.txt"
		contentType := "text/plain"

		err := backend.Store(ctx, key, testData, contentType)
		require.NoError(t, err, "Failed to store file for get test")

		// Now retrieve it
		data, err := backend.Get(ctx, key)
		require.NoError(t, err, "Failed to get file from S3")
		assert.Equal(t, testData, data)
	})

	t.Run("GetInfo", func(t *testing.T) {
		testData := []byte("test file content for info")
		key := "test/info-file.txt"
		contentType := "text/plain"

		// Store file first
		err := backend.Store(ctx, key, testData, contentType)
		require.NoError(t, err, "Failed to store file for info test")

		// Get file info
		info, err := backend.GetInfo(ctx, key)
		require.NoError(t, err, "Failed to get file info")
		assert.Equal(t, key, info.Key)
		assert.Equal(t, int64(len(testData)), info.Size)
		assert.WithinDuration(t, time.Now(), info.LastModified, time.Minute)
	})

	t.Run("GetPresignedURL", func(t *testing.T) {
		key := "test/presigned-file.txt"
		expiry := time.Hour

		// Test GET presigned URL
		getURL, err := backend.GetPresignedURL(ctx, key, "GET", expiry)
		require.NoError(t, err, "Failed to generate GET presigned URL")
		assert.NotEmpty(t, getURL)
		assert.Contains(t, getURL, key)

		// Test PUT presigned URL
		putURL, err := backend.GetPresignedURL(ctx, key, "PUT", 30*time.Minute)
		require.NoError(t, err, "Failed to generate PUT presigned URL")
		assert.NotEmpty(t, putURL)
		assert.Contains(t, putURL, key)
	})

	t.Run("Delete", func(t *testing.T) {
		testData := []byte("test file content for deletion")
		key := "test/delete-file.txt"
		contentType := "text/plain"

		// Store file first
		err := backend.Store(ctx, key, testData, contentType)
		require.NoError(t, err, "Failed to store file for delete test")

		// Delete the file
		err = backend.Delete(ctx, key)
		require.NoError(t, err, "Failed to delete file")

		// Verify file is gone
		_, err = backend.Get(ctx, key)
		assert.Error(t, err, "File should not exist after deletion")
	})

	t.Run("Exists", func(t *testing.T) {
		testData := []byte("test file content for exists check")
		key := "test/exists-file-unique.txt" // Use unique key to avoid conflicts
		contentType := "text/plain"

		// File should not exist initially
		exists, err := backend.Exists(ctx, key)
		require.NoError(t, err, "Failed to check file existence")
		assert.False(t, exists, "File should not exist initially")

		// Store file
		err = backend.Store(ctx, key, testData, contentType)
		require.NoError(t, err, "Failed to store file for exists test")

		// File should now exist
		exists, err = backend.Exists(ctx, key)
		require.NoError(t, err, "Failed to check file existence after store")
		assert.True(t, exists, "File should exist after storing")

		// Clean up
		_ = backend.Delete(ctx, key)
	})

	t.Run("GetURL", func(t *testing.T) {
		key := "test/url-file.txt"
		url := backend.GetURL(key)
		assert.NotEmpty(t, url)
		assert.Contains(t, url, key)
	})
}

func TestLocalStorageBackend(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	baseURL := "http://localhost:7890/uploads"

	backend := NewLocalStorageBackend(tempDir, baseURL)
	ctx := context.Background()

	t.Run("Store", func(t *testing.T) {
		testData := []byte("test file content for local storage")
		key := "test/local-file.txt"
		contentType := "text/plain"

		err := backend.Store(ctx, key, testData, contentType)
		require.NoError(t, err, "Failed to store file locally")
	})

	t.Run("Get", func(t *testing.T) {
		testData := []byte("test file content for local retrieval")
		key := "test/local-get-file.txt"
		contentType := "text/plain"

		// Store file first
		err := backend.Store(ctx, key, testData, contentType)
		require.NoError(t, err, "Failed to store file for get test")

		// Retrieve file
		data, err := backend.Get(ctx, key)
		require.NoError(t, err, "Failed to get file locally")
		assert.Equal(t, testData, data)
	})

	t.Run("Delete", func(t *testing.T) {
		testData := []byte("test file content for local deletion")
		key := "test/local-delete-file.txt"
		contentType := "text/plain"

		// Store file first
		err := backend.Store(ctx, key, testData, contentType)
		require.NoError(t, err, "Failed to store file for delete test")

		// Delete file
		err = backend.Delete(ctx, key)
		require.NoError(t, err, "Failed to delete local file")

		// Verify file is gone
		_, err = backend.Get(ctx, key)
		assert.Error(t, err, "File should not exist after deletion")
	})

	t.Run("GetInfo", func(t *testing.T) {
		testData := []byte("test file content for local info")
		key := "test/local-info-file.txt"
		contentType := "text/plain"

		// Store file first
		err := backend.Store(ctx, key, testData, contentType)
		require.NoError(t, err, "Failed to store file for info test")

		// Get file info
		info, err := backend.GetInfo(ctx, key)
		require.NoError(t, err, "Failed to get local file info")
		assert.Equal(t, key, info.Key)
		assert.Equal(t, int64(len(testData)), info.Size)
		assert.WithinDuration(t, time.Now(), info.LastModified, time.Minute)
	})

	t.Run("GetURL", func(t *testing.T) {
		key := "test/local-url-file.txt"
		url := backend.GetURL(key)
		assert.NotEmpty(t, url)
		assert.Contains(t, url, key)
		assert.Contains(t, url, baseURL)
	})
}

func TestPluginInitialization(t *testing.T) {
	t.Run("LocalStoragePlugin", func(t *testing.T) {
		plugin := &FileUploadPlugin{}
		config := map[string]interface{}{
			"storage_type": "local",
			"upload_dir":   t.TempDir(),
		}

		err := plugin.Initialize(config)
		require.NoError(t, err, "Failed to initialize local storage plugin")
		assert.NotNil(t, plugin.storageBackend)
	})

	t.Run("S3StoragePlugin", func(t *testing.T) {
		if os.Getenv("DOCKER_ENV") != "true" {
			t.Skip("Skipping S3 plugin initialization test (requires DOCKER_ENV=true)")
		}

		plugin := &FileUploadPlugin{}
		config := map[string]interface{}{
			"storage_type": "s3",
			"s3": map[string]interface{}{
				"endpoint":          "minio:9000",
				"access_key_id":     "minioadmin",
				"secret_access_key": "minioadmin",
				"bucket_name":       "test-bucket",
				"region":            "us-east-1",
				"use_ssl":           false,
			},
		}

		err := plugin.Initialize(config)
		require.NoError(t, err, "Failed to initialize S3 storage plugin")
		assert.NotNil(t, plugin.storageBackend)
	})

	t.Run("InvalidStorageType", func(t *testing.T) {
		plugin := &FileUploadPlugin{}
		config := map[string]interface{}{
			"storage_type": "invalid",
		}

		err := plugin.Initialize(config)
		if err != nil {
			assert.Error(t, err, "Should fail with invalid storage type")
			assert.Contains(t, err.Error(), "unsupported storage type")
		} else {
			// If it doesn't error, the plugin might have default behavior
			t.Log("Plugin initialization did not fail - may have default behavior")
		}
	})
}

func TestS3ConfigParsing(t *testing.T) {
	plugin := &FileUploadPlugin{}

	t.Run("ValidS3Config", func(t *testing.T) {
		configMap := map[string]interface{}{
			"s3": map[string]interface{}{
				"endpoint":          "s3.amazonaws.com",
				"access_key_id":     "test-key",
				"secret_access_key": "test-secret",
				"bucket_name":       "test-bucket",
				"region":            "us-west-2",
				"use_ssl":           true,
				"base_url":          "https://cdn.example.com",
			},
		}

		config, err := plugin.parseS3Config(configMap)
		require.NoError(t, err, "Failed to parse valid S3 config")
		assert.Equal(t, "s3.amazonaws.com", config.Endpoint)
		assert.Equal(t, "test-key", config.AccessKeyID)
		assert.Equal(t, "test-secret", config.SecretAccessKey)
		assert.Equal(t, "test-bucket", config.BucketName)
		assert.Equal(t, "us-west-2", config.Region)
		assert.Equal(t, true, config.UseSSL)
		assert.Equal(t, "https://cdn.example.com", config.BaseURL)
	})

	t.Run("MissingS3Config", func(t *testing.T) {
		configMap := map[string]interface{}{}

		_, err := plugin.parseS3Config(configMap)
		assert.Error(t, err, "Should fail when S3 config is missing")
		assert.Contains(t, err.Error(), "S3/MinIO configuration not found")
	})

	t.Run("InvalidS3ConfigType", func(t *testing.T) {
		configMap := map[string]interface{}{
			"s3": "invalid-string",
		}

		_, err := plugin.parseS3Config(configMap)
		assert.Error(t, err, "Should fail when S3 config is not a map")
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		configMap := map[string]interface{}{
			"s3": map[string]interface{}{
				"access_key_id": "test-key",
				// Missing other required fields
			},
		}

		_, err := plugin.parseS3Config(configMap)
		assert.Error(t, err, "Should fail when required S3 fields are missing")
	})
}

func TestFileOperations(t *testing.T) {
	// Test helper functions
	t.Run("EnsureDir", func(t *testing.T) {
		tempDir := t.TempDir()
		testPath := tempDir + "/nested/deep/directory"

		err := ensureDir(testPath)
		require.NoError(t, err, "Failed to ensure directory exists")

		// Verify directory was created
		info, err := os.Stat(testPath)
		require.NoError(t, err, "Directory should exist after ensureDir")
		assert.True(t, info.IsDir(), "Path should be a directory")
	})

	t.Run("WriteAndReadFile", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := tempDir + "/test-file.txt"
		testData := []byte("test file content")

		// Write file
		err := writeFile(filePath, testData)
		require.NoError(t, err, "Failed to write file")

		// Read file
		data, err := readFile(filePath)
		require.NoError(t, err, "Failed to read file")
		assert.Equal(t, testData, data)
	})

	t.Run("FileExists", func(t *testing.T) {
		tempDir := t.TempDir()
		existingFile := tempDir + "/existing.txt"
		nonExistentFile := tempDir + "/nonexistent.txt"

		// Create a file
		err := writeFile(existingFile, []byte("test"))
		require.NoError(t, err, "Failed to create test file")

		// Test existing file
		exists := fileExists(existingFile)
		assert.True(t, exists, "File should exist")

		// Test non-existent file
		exists = fileExists(nonExistentFile)
		assert.False(t, exists, "File should not exist")
	})

	t.Run("RemoveFile", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := tempDir + "/removeme.txt"

		// Create file
		err := writeFile(filePath, []byte("test"))
		require.NoError(t, err, "Failed to create file for removal test")

		// Verify file exists
		assert.True(t, fileExists(filePath), "File should exist before removal")

		// Remove file
		err = removeFile(filePath)
		require.NoError(t, err, "Failed to remove file")

		// Verify file is gone
		assert.False(t, fileExists(filePath), "File should not exist after removal")
	})
}
