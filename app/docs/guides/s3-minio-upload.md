# S3/MinIO File Upload Integration

TurboScript provides comprehensive S3 and MinIO compatible file upload capabilities through an enhanced file upload plugin system. This guide covers setup, configuration, and usage of cloud storage features.

## Overview

The file upload plugin supports multiple storage backends:

- **Local Storage**: Traditional filesystem-based uploads
- **S3 Compatible**: AWS S3, MinIO, DigitalOcean Spaces, and other S3-compatible services
- **Hybrid Mode**: Seamless switching between storage backends

## Key Features

### 🚀 Advanced Capabilities

- **Presigned URLs**: Secure direct-to-cloud uploads without exposing credentials
- **Bucket Management**: Automatic bucket creation and configuration
- **Storage Abstraction**: Unified API across local and cloud storage
- **File Operations**: Upload, download, list, delete, and file info retrieval
- **Type Safety**: Comprehensive error handling and validation

### 🔒 Security Features

- **Access Control**: Configurable file type restrictions
- **Size Limits**: Customizable maximum file size enforcement
- **Secure URLs**: Time-limited presigned URLs with configurable expiration
- **SSL Support**: Full TLS/SSL encryption for production deployments

## Configuration

### Basic Setup

Add the file upload plugin to your `turboscript.yml`:

```yaml
plugins:
  - name: fileupload
    enabled: true
    options:
      storage_type: "s3"  # or "local", "minio"
      max_file_size: 10485760  # 10MB
      allowed_types:
        - "image/jpeg"
        - "image/png"
        - "image/gif"
        - "image/webp"
        - "application/pdf"
        - "text/plain"
```

### S3 Configuration

For AWS S3 or S3-compatible services:

```yaml
plugins:
  - name: fileupload
    enabled: true
    options:
      storage_type: "s3"
      s3:
        endpoint: ""  # Leave empty for AWS S3
        access_key_id: "${env:AWS_ACCESS_KEY_ID}"
        secret_access_key: "${env:AWS_SECRET_ACCESS_KEY}"
        bucket_name: "your-bucket-name"
        region: "us-east-1"
        use_ssl: true
```

### MinIO Configuration

For MinIO object storage:

```yaml
plugins:
  - name: fileupload
    enabled: true
    options:
      storage_type: "s3"  # MinIO is S3-compatible
      s3:
        endpoint: "localhost:9000"
        access_key_id: "minioadmin"
        secret_access_key: "minioadmin"
        bucket_name: "turboscript-uploads"
        region: "us-east-1"
        use_ssl: false  # Set to true for production
        base_url: "http://localhost:9000/turboscript-uploads"
```

## Usage in TypeScript Routes

### Basic File Upload

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const fileUpload = turboPlugin('fileupload');

    // Upload base64 encoded file
    const result = fileUpload.saveBase64(
        base64Data,
        'example.jpg',
        { directory: 'uploads', generateHash: true }
    );

    return {
        code: 200,
        response: { status: "success", data: result }
    };
};
```

### Generate Presigned URLs

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const fileUpload = turboPlugin('fileupload');

    // Generate presigned URL for direct upload
    const uploadURL = fileUpload.generatePresignedURL(
        'path/to/file.jpg',
        'PUT',
        60  // Expires in 60 minutes
    );

    // Generate presigned URL for download
    const downloadURL = fileUpload.generatePresignedURL(
        'path/to/file.jpg',
        'GET',
        30  // Expires in 30 minutes
    );

    return {
        code: 200,
        response: {
            upload_url: uploadURL,
            download_url: downloadURL
        }
    };
};
```

### File Management Operations

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const fileUpload = turboPlugin('fileupload');

    // List files with optional prefix
    const files = fileUpload.listFiles('uploads/images/');

    // Get file information
    const fileInfo = fileUpload.getFileInfo('uploads/image.jpg');

    // Delete file
    const deleteResult = fileUpload.deleteFile('uploads/old-file.jpg');

    return {
        code: 200,
        response: {
            files: files,
            file_info: fileInfo,
            deleted: deleteResult
        }
    };
};
```

## API Functions

### Core Upload Functions

#### `saveBase64(fileData, filename, options)`

Uploads a base64 encoded file to the configured storage backend.

**Parameters:**

- `fileData` (string): Base64 encoded file content
- `filename` (string): Original filename
- `options` (object): Upload options
  - `directory` (string): Target directory path
  - `generateHash` (boolean): Generate unique hash-based filename

**Returns:** Upload result with file path, size, and metadata

#### `generatePresignedURL(key, operation, expiryMinutes)`

Generates a presigned URL for direct cloud storage access.

**Parameters:**

- `key` (string): File path/key in storage
- `operation` (string): HTTP method ('GET', 'PUT', 'DELETE')
- `expiryMinutes` (number): URL expiration time in minutes

**Returns:** Presigned URL string

### File Management Functions

#### `listFiles(prefix)`

Lists files in storage with optional prefix filtering.

**Parameters:**

- `prefix` (string): Optional path prefix to filter results

**Returns:** Array of file objects with metadata

#### `getFileInfo(key)`

Retrieves detailed information about a specific file.

**Parameters:**

- `key` (string): File path/key in storage

**Returns:** File metadata object (size, modified date, etc.)

#### `deleteFile(key)`

Deletes a file from storage.

**Parameters:**

- `key` (string): File path/key to delete

**Returns:** Deletion result object

## Storage Backends

### Local Storage Backend

The local storage backend saves files to the filesystem:

```yaml
plugins:
  - name: fileupload
    options:
      storage_type: "local"
      upload_dir: "./uploads"
```

**Features:**

- Direct filesystem operations
- Fast for development and single-server deployments
- Automatic directory creation
- File permissions management

### S3 Storage Backend

The S3 backend provides cloud storage capabilities:

```yaml
plugins:
  - name: fileupload
    options:
      storage_type: "s3"
      s3:
        bucket_name: "my-bucket"
        # ... other S3 config
```

**Features:**

- Presigned URL generation
- Automatic bucket creation
- Multi-region support
- SSL/TLS encryption
- Compatible with AWS S3, MinIO, DigitalOcean Spaces

## Environment Variables

Use environment variables for sensitive configuration:

```bash
# AWS S3
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export S3_BUCKET_NAME="your-bucket"
export S3_REGION="us-east-1"

# MinIO
export MINIO_ENDPOINT="localhost:9000"
export MINIO_ACCESS_KEY="minioadmin"
export MINIO_SECRET_KEY="minioadmin"
export MINIO_BUCKET="uploads"
```

## Demo and Testing

TurboScript includes a comprehensive demo at `/demo/s3-upload` that showcases:

- File upload with drag & drop
- Presigned URL generation
- File listing and management
- Storage backend switching
- Error handling and validation

Visit the demo to see all features in action and use it as a reference for your own implementations.

## Best Practices

### Security

- Always use environment variables for credentials
- Enable SSL/TLS in production
- Set appropriate presigned URL expiration times
- Validate file types and sizes on both client and server

### Performance

- Use presigned URLs for large file uploads
- Implement client-side validation before upload
- Consider file compression for images
- Use appropriate cache headers for static assets

### Error Handling

- Implement comprehensive error catching
- Provide meaningful error messages to users
- Log upload failures for debugging
- Handle network timeouts gracefully

## Troubleshooting

### Common Issues

**Bucket Access Errors:**

- Verify credentials are correct
- Check bucket exists and permissions
- Ensure region is properly configured

**Presigned URL Failures:**

- Verify URL hasn't expired
- Check CORS configuration for browser uploads
- Ensure proper HTTP method is used

**Connection Issues:**

- Verify endpoint URLs are accessible
- Check SSL/TLS settings match server configuration
- Test network connectivity to storage service

### Debug Mode

Enable debug logging to troubleshoot issues:

```yaml
debug: true
```

This provides detailed logs of storage operations, credential validation, and error details.

## Migration Guide

### From Local to S3

1. Update configuration to use S3 backend
2. Copy existing files to S3 bucket
3. Update file URLs in database if needed
4. Test all upload/download functionality

### Between S3 Providers

1. Create new bucket on target provider
2. Update configuration with new credentials
3. Migrate files using S3 sync tools
4. Update base URLs if needed

## Advanced Configuration

### Custom Base URLs

For custom domains or CDN integration:

```yaml
s3:
  base_url: "https://cdn.yourdomain.com/uploads"
```

### Multiple Storage Backends

Configure different backends for different file types:

```typescript
// Route-specific storage selection
const fileUpload = turboPlugin('fileupload');

// Override storage type per operation
const result = fileUpload.saveBase64(data, filename, {
    storage_backend: 's3',  // Force S3 for this upload
    directory: 'critical-files'
});
```

This comprehensive S3/MinIO integration makes TurboScript a powerful choice for applications requiring robust file storage capabilities with cloud-native features.
