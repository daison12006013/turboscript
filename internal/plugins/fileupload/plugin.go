/*
 * TurboScript - A hybrid web framework combining TypeScript and Go
 *
 * Copyright (c) 2025 TurboScript Project Contributors
 * Author: Daison Cariño <daison12006013@gmail.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Based on TurboScript: https://github.com/daison12006013/turboscript
 */

// Package fileupload provides binary file upload and management functionality for TurboScript.
//
// This plugin allows developers to handle file uploads, binary data processing, and file
// storage operations from TypeScript route handlers. It supports multiple storage backends
// including local filesystem, AWS S3, and other cloud storage providers.
package fileupload

import (
	"context"
	"crypto/md5" // #nosec G501 - MD5 is used for file integrity checks, not cryptographic security
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// FileUploadPlugin provides file upload and binary data management capabilities.
type FileUploadPlugin struct {
	uploadDir      string
	maxFileSize    int64
	allowedTypes   []string
	generatePath   func(filename string) string
	storageBackend StorageBackend
	storageType    string
	s3Config       *S3Config
}

// FileInfo represents uploaded file information.
type FileInfo struct {
	OriginalName string `json:"originalName"`
	Filename     string `json:"filename"`
	Size         int64  `json:"size"`
	MimeType     string `json:"mimeType"`
	Extension    string `json:"extension"`
	Path         string `json:"path"`
	URL          string `json:"url"`
	MD5Hash      string `json:"md5Hash"`
	SHA256Hash   string `json:"sha256Hash"`
	UploadedAt   string `json:"uploadedAt"`
}

// UploadOptions represents options for file upload operations.
type UploadOptions struct {
	Directory    string   `json:"directory"`
	Filename     string   `json:"filename"`
	AllowedTypes []string `json:"allowedTypes"`
	MaxSize      int64    `json:"maxSize"`
	GenerateHash bool     `json:"generateHash"`
}

// Name returns the plugin name.
func (p *FileUploadPlugin) Name() string {
	return "fileupload"
}

// Description returns the plugin description.
func (p *FileUploadPlugin) Description() string {
	return "File upload and binary data management plugin for TurboScript"
}

// Version returns the plugin version.
func (p *FileUploadPlugin) Version() string {
	return "1.0.0"
}

// Initialize sets up the plugin with the provided configuration.
func (p *FileUploadPlugin) Initialize(config map[string]any) error {
	// Set default values
	p.uploadDir = "./uploads"
	p.maxFileSize = 10 * 1024 * 1024 // 10MB default
	p.allowedTypes = []string{"image/jpeg", "image/png", "image/gif", "image/webp", "application/pdf"}
	p.storageType = "local" // Default to local storage

	// Apply configuration
	if uploadDir, ok := config["upload_dir"].(string); ok && uploadDir != "" {
		p.uploadDir = uploadDir
	}

	if maxSize, ok := config["max_file_size"].(int64); ok && maxSize > 0 {
		p.maxFileSize = maxSize
	} else if maxSizeInt, ok := config["max_file_size"].(int); ok && maxSizeInt > 0 {
		p.maxFileSize = int64(maxSizeInt)
	}

	if allowedTypes, ok := config["allowed_types"].([]interface{}); ok {
		p.allowedTypes = make([]string, len(allowedTypes))
		for i, t := range allowedTypes {
			if str, ok := t.(string); ok {
				p.allowedTypes[i] = str
			}
		}
	}

	// Storage configuration
	if storageType, ok := config["storage_type"].(string); ok && storageType != "" {
		p.storageType = storageType
	}

	// Initialize storage backend based on type
	switch p.storageType {
	case "s3", "minio":
		s3Config, err := p.parseS3Config(config)
		if err != nil {
			return fmt.Errorf("failed to parse S3 config: %w", err)
		}
		p.s3Config = s3Config

		backend, err := NewS3StorageBackend(*s3Config)
		if err != nil {
			return fmt.Errorf("failed to create S3 storage backend: %w", err)
		}

		// Ensure bucket exists
		ctx := context.Background()
		if err := backend.EnsureBucket(ctx); err != nil {
			return fmt.Errorf("failed to ensure bucket exists: %w", err)
		}

		p.storageBackend = backend

	case "local":
		fallthrough
	default:
		// Create upload directory if it doesn't exist
		if err := os.MkdirAll(p.uploadDir, 0750); err != nil {
			return fmt.Errorf("failed to create upload directory: %w", err)
		}

		p.storageBackend = NewLocalStorageBackend(p.uploadDir, "")
	}

	// Set up path generator
	p.generatePath = p.defaultPathGenerator

	return nil
}

// Register adds the plugin's functions to the JavaScript runtime.
func (p *FileUploadPlugin) Register(runtime *goja.Runtime, registry *require.Registry) error {
	// Create the exports object with all functions
	exports := runtime.NewObject()

	// Add test function
	if err := exports.Set("test", runtime.ToValue(func() string {
		return "test function works"
	})); err != nil {
		log.Printf("Failed to set test function: %v", err)
	}

	// Add saveBase64 function
	saveBase64Fn := runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(runtime.NewTypeError("saveBase64 requires at least 2 arguments: base64Data, filename"))
		}

		base64Data := call.Arguments[0].String()
		filename := call.Arguments[1].String()

		var options map[string]interface{}
		if len(call.Arguments) > 2 {
			if err := runtime.ExportTo(call.Arguments[2], &options); err != nil {
				log.Printf("Failed to export options: %v", err)
			}
		}

		result, err := p.saveBase64(base64Data, filename, options)
		if err != nil {
			panic(runtime.NewTypeError(err.Error()))
		}

		return runtime.ToValue(result)
	})

	// Add generatePresignedURL function for S3/MinIO
	generatePresignedURLFn := runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(runtime.NewTypeError("generatePresignedURL requires at least 2 arguments: key, operation"))
		}

		key := call.Arguments[0].String()
		operation := call.Arguments[1].String()

		var expiryMinutes int64 = 60 // Default 1 hour
		if len(call.Arguments) > 2 {
			if exp := call.Arguments[2].ToInteger(); exp > 0 {
				expiryMinutes = exp
			}
		}

		result, err := p.generatePresignedURL(key, operation, time.Duration(expiryMinutes)*time.Minute)
		if err != nil {
			panic(runtime.NewTypeError(err.Error()))
		}

		return runtime.ToValue(result)
	})

	// Add listFiles function
	listFilesFn := runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		var prefix string
		if len(call.Arguments) > 0 {
			prefix = call.Arguments[0].String()
		}

		result, err := p.listFiles(prefix)
		if err != nil {
			panic(runtime.NewTypeError(err.Error()))
		}

		return runtime.ToValue(result)
	})

	// Add deleteFile function
	deleteFileFn := runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(runtime.NewTypeError("deleteFile requires 1 argument: key"))
		}

		key := call.Arguments[0].String()

		err := p.deleteFileByKey(key)
		if err != nil {
			panic(runtime.NewTypeError(err.Error()))
		}

		return runtime.ToValue(map[string]interface{}{
			"success": true,
			"message": "File deleted successfully",
		})
	})

	// Add getFileInfo function
	getFileInfoFn := runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(runtime.NewTypeError("getFileInfo requires 1 argument: key"))
		}

		key := call.Arguments[0].String()

		result, err := p.getFileInfoByKey(key)
		if err != nil {
			panic(runtime.NewTypeError(err.Error()))
		}

		return runtime.ToValue(result)
	})

	if err := exports.Set("saveBase64", saveBase64Fn); err != nil {
		log.Printf("Failed to set saveBase64 function: %v", err)
	}
	if err := exports.Set("generatePresignedURL", generatePresignedURLFn); err != nil {
		log.Printf("Failed to set generatePresignedURL function: %v", err)
	}
	if err := exports.Set("listFiles", listFilesFn); err != nil {
		log.Printf("Failed to set listFiles function: %v", err)
	}
	if err := exports.Set("deleteFile", deleteFileFn); err != nil {
		log.Printf("Failed to set deleteFile function: %v", err)
	}
	if err := exports.Set("getFileInfo", getFileInfoFn); err != nil {
		log.Printf("Failed to set getFileInfo function: %v", err)
	}

	// Register as a require module using the standard Node.js exports pattern
	registry.RegisterNativeModule("fileupload", func(_rt *goja.Runtime, module *goja.Object) {
		if err := module.Set("exports", exports); err != nil {
			// Handle error but don't fail completely
		}
	})

	// TEMPORARY: Also register as a global function for immediate testing
	global := runtime.GlobalObject()
	if err := global.Set("fileUploadSaveBase64", saveBase64Fn); err != nil {
		log.Printf("Failed to set global fileUploadSaveBase64 function: %v", err)
	}

	return nil
}

// parseMultipart parses multipart form data and returns file information.
func (p *FileUploadPlugin) parseMultipart(_formData string, boundary string) (map[string]interface{}, error) {
	// This would typically receive binary data from the HTTP request
	// For now, return a structured response for the file upload
	result := map[string]interface{}{
		"files":  []FileInfo{},
		"fields": map[string]string{},
	}

	return result, nil
}

// saveFile saves uploaded file data using the configured storage backend.
func (p *FileUploadPlugin) saveFile(data interface{}, options map[string]interface{}) (FileInfo, error) {
	opts := p.parseUploadOptions(options)
	ctx := context.Background()

	// Handle different data types
	var fileData []byte
	var originalName string
	var contentType string

	switch v := data.(type) {
	case string:
		// Base64 encoded data
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return FileInfo{}, fmt.Errorf("invalid base64 data: %w", err)
		}
		fileData = decoded
		originalName = opts.Filename
		contentType = getContentType(originalName)
	case []byte:
		fileData = v
		originalName = opts.Filename
		contentType = getContentType(originalName)
	default:
		return FileInfo{}, fmt.Errorf("unsupported data type: %T", data)
	}

	// Validate file
	if err := p.validateFileData(fileData, contentType, opts); err != nil {
		return FileInfo{}, err
	}

	// Generate storage key
	storageKey := p.generateStorageKey(originalName, opts)

	// Store file using the storage backend
	if err := p.storageBackend.Store(ctx, storageKey, fileData, contentType); err != nil {
		return FileInfo{}, fmt.Errorf("failed to store file: %w", err)
	}

	// Generate file info
	fileInfo := FileInfo{
		OriginalName: originalName,
		Filename:     filepath.Base(storageKey),
		Size:         int64(len(fileData)),
		MimeType:     contentType,
		Extension:    filepath.Ext(originalName),
		Path:         storageKey,
		URL:          p.storageBackend.GetURL(storageKey),
		UploadedAt:   time.Now().Format(time.RFC3339),
	}

	// Generate hashes if requested
	if opts.GenerateHash {
		fileInfo.MD5Hash = p.generateMD5Hash(fileData)
		fileInfo.SHA256Hash = p.generateSHA256Hash(fileData)
	}

	return fileInfo, nil
}

// saveBase64 saves base64 encoded file data.
func (p *FileUploadPlugin) saveBase64(base64Data string, filename string, options map[string]interface{}) (FileInfo, error) {
	// Remove data URL prefix if present
	if strings.Contains(base64Data, ",") {
		parts := strings.Split(base64Data, ",")
		if len(parts) > 1 {
			base64Data = parts[1]
		}
	}

	// Set filename in options
	if options == nil {
		options = make(map[string]interface{})
	}
	options["filename"] = filename

	return p.saveFile(base64Data, options)
}

// deleteFile removes a file from the filesystem.
func (p *FileUploadPlugin) deleteFile(filePath string) error {
	// Ensure file is within upload directory for security
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}

	absUploadDir, err := filepath.Abs(p.uploadDir)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(absPath, absUploadDir) {
		return fmt.Errorf("file path is outside upload directory")
	}

	return os.Remove(filePath)
}

// getFileInfo returns information about an existing file.
func (p *FileUploadPlugin) getFileInfo(filePath string) (FileInfo, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return FileInfo{}, err
	}

	ext := filepath.Ext(filePath)
	filename := filepath.Base(filePath)

	return FileInfo{
		Filename:   filename,
		Size:       stat.Size(),
		MimeType:   mime.TypeByExtension(ext),
		Extension:  ext,
		Path:       filePath,
		URL:        p.generateURL(filePath),
		UploadedAt: stat.ModTime().Format(time.RFC3339),
	}, nil
}

// validateFile validates file constraints.
func (p *FileUploadPlugin) validateFile(data interface{}, contentType string, options map[string]interface{}) error {
	opts := p.parseUploadOptions(options)

	var fileData []byte
	switch v := data.(type) {
	case string:
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return fmt.Errorf("invalid base64 data: %w", err)
		}
		fileData = decoded
	case []byte:
		fileData = v
	default:
		return fmt.Errorf("unsupported data type: %T", data)
	}

	return p.validateFileData(fileData, contentType, opts)
}

// generateFilename generates a unique filename.
func (p *FileUploadPlugin) generateFilename(originalName string) string {
	return p.defaultPathGenerator(originalName)
}

// hashFile generates hash for file data.
func (p *FileUploadPlugin) hashFile(filePath string) (map[string]string, error) {
	data, err := os.ReadFile(filePath) // #nosec G304 - This is a controlled file read operation within the file upload plugin
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"md5":    p.generateMD5Hash(data),
		"sha256": p.generateSHA256Hash(data),
	}, nil
}

// resizeImage resizes an image (placeholder for image processing).
func (p *FileUploadPlugin) resizeImage(_filePath string, width, height int) (FileInfo, error) {
	// This is a placeholder - would integrate with image processing library
	return FileInfo{}, fmt.Errorf("image resizing not implemented yet")
}

// Helper methods

func (p *FileUploadPlugin) parseUploadOptions(options map[string]interface{}) UploadOptions {
	opts := UploadOptions{
		AllowedTypes: p.allowedTypes,
		MaxSize:      p.maxFileSize,
		GenerateHash: true,
	}

	if directory, ok := options["directory"].(string); ok {
		opts.Directory = directory
	}

	if filename, ok := options["filename"].(string); ok {
		opts.Filename = filename
	}

	if allowedTypes, ok := options["allowedTypes"].([]interface{}); ok {
		opts.AllowedTypes = make([]string, len(allowedTypes))
		for i, t := range allowedTypes {
			if str, ok := t.(string); ok {
				opts.AllowedTypes[i] = str
			}
		}
	}

	if maxSize, ok := options["maxSize"].(int64); ok {
		opts.MaxSize = maxSize
	} else if maxSizeInt, ok := options["maxSize"].(int); ok {
		opts.MaxSize = int64(maxSizeInt)
	}

	if generateHash, ok := options["generateHash"].(bool); ok {
		opts.GenerateHash = generateHash
	}

	return opts
}

func (p *FileUploadPlugin) validateFileData(data []byte, contentType string, opts UploadOptions) error {
	// Check file size
	if data != nil && int64(len(data)) > opts.MaxSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", opts.MaxSize)
	}

	// Check content type
	if len(opts.AllowedTypes) > 0 {
		allowed := false
		for _, allowedType := range opts.AllowedTypes {
			if contentType == allowedType || strings.HasPrefix(contentType, allowedType) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("file type '%s' is not allowed", contentType)
		}
	}

	return nil
}

func (p *FileUploadPlugin) defaultPathGenerator(originalName string) string {
	// Generate unique filename: timestamp_random_originalname
	timestamp := time.Now().Unix()
	ext := filepath.Ext(originalName)
	baseName := strings.TrimSuffix(originalName, ext)

	// Sanitize filename
	baseName = p.sanitizeFilename(baseName)

	return fmt.Sprintf("%d_%s%s", timestamp, baseName, ext)
}

func (p *FileUploadPlugin) sanitizeFilename(filename string) string {
	// Remove or replace unsafe characters
	result := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, filename)

	// Limit length
	if len(result) > 100 {
		result = result[:100]
	}

	return result
}

func (p *FileUploadPlugin) generateURL(filePath string) string {
	// Generate URL relative to upload directory
	relPath, err := filepath.Rel(p.uploadDir, filePath)
	if err != nil {
		return ""
	}

	return "/uploads/" + strings.ReplaceAll(relPath, "\\", "/")
}

func (p *FileUploadPlugin) generateMD5Hash(data []byte) string {
	hash := md5.Sum(data) // #nosec G401 - MD5 is used for file integrity checks, not cryptographic security
	return hex.EncodeToString(hash[:])
}

func (p *FileUploadPlugin) generateSHA256Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// parseS3Config parses S3/MinIO configuration from the plugin config.
func (p *FileUploadPlugin) parseS3Config(config map[string]any) (*S3Config, error) {
	s3Config := &S3Config{
		Region: "us-east-1", // Default region
		UseSSL: true,        // Default to secure connections
	}

	// Look for s3 or minio specific config
	var storageConfig map[string]interface{}
	if s3Cfg, ok := config["s3"].(map[string]interface{}); ok {
		storageConfig = s3Cfg
	} else if minioCfg, ok := config["minio"].(map[string]interface{}); ok {
		storageConfig = minioCfg
	} else if storageCfg, ok := config["storage"].(map[string]interface{}); ok {
		storageConfig = storageCfg
	}

	if storageConfig == nil {
		return nil, fmt.Errorf("S3/MinIO configuration not found")
	}

	// Parse configuration
	if endpoint, ok := storageConfig["endpoint"].(string); ok {
		s3Config.Endpoint = endpoint
	} else {
		return nil, fmt.Errorf("endpoint is required for S3/MinIO storage")
	}

	if accessKey, ok := storageConfig["access_key_id"].(string); ok {
		s3Config.AccessKeyID = accessKey
	} else {
		return nil, fmt.Errorf("access_key_id is required for S3/MinIO storage")
	}

	if secretKey, ok := storageConfig["secret_access_key"].(string); ok {
		s3Config.SecretAccessKey = secretKey
	} else {
		return nil, fmt.Errorf("secret_access_key is required for S3/MinIO storage")
	}

	if bucket, ok := storageConfig["bucket_name"].(string); ok {
		s3Config.BucketName = bucket
	} else {
		return nil, fmt.Errorf("bucket_name is required for S3/MinIO storage")
	}

	if region, ok := storageConfig["region"].(string); ok {
		s3Config.Region = region
	}

	if useSSL, ok := storageConfig["use_ssl"].(bool); ok {
		s3Config.UseSSL = useSSL
	}

	if baseURL, ok := storageConfig["base_url"].(string); ok {
		s3Config.BaseURL = baseURL
	}

	return s3Config, nil
}

// generateStorageKey generates a storage key for the file.
func (p *FileUploadPlugin) generateStorageKey(originalName string, opts UploadOptions) string {
	filename := p.generateFilename(originalName)
	if opts.Filename != "" {
		filename = opts.Filename
	}

	if opts.Directory != "" {
		return opts.Directory + "/" + filename
	}

	return filename
}

// generatePresignedURL generates a presigned URL for S3/MinIO operations.
func (p *FileUploadPlugin) generatePresignedURL(key string, operation string, expiry time.Duration) (string, error) {
	ctx := context.Background()
	return p.storageBackend.GetPresignedURL(ctx, key, operation, expiry)
}

// listFiles lists files with optional prefix.
func (p *FileUploadPlugin) listFiles(_prefix string) ([]FileInfo, error) {
	// This would need to be implemented by each storage backend
	// For now, return a placeholder
	return []FileInfo{}, fmt.Errorf("listFiles not implemented for this storage backend")
}

// deleteFileByKey deletes a file by its storage key.
func (p *FileUploadPlugin) deleteFileByKey(key string) error {
	ctx := context.Background()
	return p.storageBackend.Delete(ctx, key)
}

// getFileInfoByKey gets file information by storage key.
func (p *FileUploadPlugin) getFileInfoByKey(key string) (FileInfo, error) {
	ctx := context.Background()

	// Check if file exists
	exists, err := p.storageBackend.Exists(ctx, key)
	if err != nil {
		return FileInfo{}, err
	}

	if !exists {
		return FileInfo{}, fmt.Errorf("file not found: %s", key)
	}

	// Get storage file info
	storageInfo, err := p.storageBackend.GetInfo(ctx, key)
	if err != nil {
		return FileInfo{}, err
	}

	// Convert to FileInfo
	return FileInfo{
		Filename:   filepath.Base(key),
		Size:       storageInfo.Size,
		MimeType:   storageInfo.ContentType,
		Extension:  filepath.Ext(key),
		Path:       key,
		URL:        p.storageBackend.GetURL(key),
		UploadedAt: storageInfo.LastModified.Format(time.RFC3339),
	}, nil
}

// NewFileUploadPlugin creates a new file upload plugin instance.
func NewFileUploadPlugin() *FileUploadPlugin {
	return &FileUploadPlugin{}
}
