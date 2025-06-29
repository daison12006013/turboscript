# File & Binary Uploads

TurboScript provides built-in support for file uploads and binary data handling through the File Upload Plugin. This allows you to easily handle file uploads, process binary data, and manage file storage in your applications.

## Quick Start

### 1. Enable the Plugin

Add the file upload plugin to your `turboscript.yml`:

```yaml
plugins:
  - name: fileupload
    enabled: true
    options:
      upload_dir: "./uploads"
      max_file_size: 10485760  # 10MB
      allowed_types:
        - "image/jpeg"
        - "image/png"
        - "image/gif"
        - "image/webp"
        - "application/pdf"
```

### 2. Create an Upload Endpoint

Create `app/routes/upload.ts`:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { saveBase64 } = require('fileupload');

    try {
        // Save the uploaded file
        const fileInfo = await saveBase64(data.file, data.filename, {
            directory: "user-uploads",
            generateHash: true
        });

        return {
            code: 200,
            response: {
                status: "success",
                file: fileInfo
            }
        };
    } catch (error) {
        return {
            code: 400,
            response: {
                status: "error",
                message: error.message
            }
        };
    }
};
```

### 3. Add the Route

In your `turboscript.yml`:

```yaml
endpoints:
  - route: /upload
    method: POST
    path: ./app/routes/upload
```

### 4. Test the Upload

```bash
# Upload a base64 encoded file
curl -X POST http://localhost:7890/upload \
  -H "Content-Type: application/json" \
  -d '{
    "file": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
    "filename": "test.png"
  }'
```

## API Reference

### saveBase64(base64Data, filename, options)

Saves a base64 encoded file to the filesystem.

**Parameters:**

- `base64Data` (string): Base64 encoded file data (with or without data URL prefix)
- `filename` (string): Original filename
- `options` (object, optional): Upload options

**Returns:** FileInfo object

**Example:**

```typescript
const fileInfo = await saveBase64(
    "data:image/png;base64,iVBORw0KGgo...",
    "photo.png",
    {
        directory: "photos",
        maxSize: 5242880, // 5MB
        generateHash: true
    }
);
```

### saveFile(binaryData, options)

Saves binary file data to the filesystem.

**Parameters:**

- `binaryData` (Uint8Array | Buffer): Binary file data
- `options` (object): Upload options including filename

**Returns:** FileInfo object

**Example:**

```typescript
const fileInfo = await saveFile(binaryData, {
    filename: "document.pdf",
    directory: "documents",
    allowedTypes: ["application/pdf"]
});
```

### getFileInfo(filePath)

Gets information about an existing file.

**Parameters:**

- `filePath` (string): Path to the file

**Returns:** FileInfo object

**Example:**

```typescript
const info = await getFileInfo("./uploads/photos/photo.png");
console.log(`File size: ${info.size} bytes`);
```

### deleteFile(filePath)

Deletes a file from the filesystem.

**Parameters:**

- `filePath` (string): Path to the file to delete

**Returns:** void

**Example:**

```typescript
await deleteFile("./uploads/old-file.png");
```

### validateFile(data, contentType, options)

Validates file data against constraints without saving it.

**Parameters:**

- `data` (string | Uint8Array): File data
- `contentType` (string): MIME type of the file
- `options` (object): Validation options

**Returns:** Validation result

**Example:**

```typescript
try {
    await validateFile(fileData, "image/jpeg", {
        maxSize: 2097152, // 2MB
        allowedTypes: ["image/jpeg", "image/png"]
    });
    console.log("File is valid");
} catch (error) {
    console.log(`Invalid file: ${error.message}`);
}
```

## Data Types

### FileInfo Object

```typescript
interface FileInfo {
    originalName: string;   // Original filename
    filename: string;       // Generated unique filename
    size: number;          // File size in bytes
    mimeType: string;      // MIME type (e.g., "image/jpeg")
    extension: string;     // File extension (e.g., ".jpg")
    path: string;          // Full filesystem path
    url: string;           // URL to access the file
    md5Hash?: string;      // MD5 hash (if generateHash: true)
    sha256Hash?: string;   // SHA256 hash (if generateHash: true)
    uploadedAt: string;    // Upload timestamp (ISO 8601)
}
```

### Upload Options

```typescript
interface UploadOptions {
    directory?: string;        // Subdirectory within upload_dir
    filename?: string;         // Custom filename (for saveFile)
    allowedTypes?: string[];   // Allowed MIME types
    maxSize?: number;          // Maximum file size in bytes
    generateHash?: boolean;    // Generate MD5/SHA256 hashes
}
```

## Configuration Options

### Plugin Configuration

```yaml
plugins:
  - name: fileupload
    enabled: true
    options:
      upload_dir: "./uploads"           # Base upload directory
      max_file_size: 10485760          # Default max size (10MB)
      allowed_types:                   # Default allowed MIME types
        - "image/jpeg"
        - "image/png"
        - "image/gif"
        - "image/webp"
        - "application/pdf"
        - "text/plain"
        - "application/msword"
        - "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
```

### Configuration Explanation

- **upload_dir**: Directory where files will be stored (relative to project root)
- **max_file_size**: Maximum allowed file size in bytes
- **allowed_types**: Array of allowed MIME types

## Real-World Examples

### Image Upload with Validation

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { saveBase64, validateFile } = require('fileupload');

    try {
        // Validate image before saving
        await validateFile(data.image, "image/jpeg", {
            maxSize: 2097152, // 2MB
            allowedTypes: ["image/jpeg", "image/png", "image/gif"]
        });

        // Save with custom directory structure
        const fileInfo = await saveBase64(data.image, data.filename, {
            directory: `users/${event.body.__user.uid}/photos`,
            generateHash: true
        });

        // Store file info in database
        await turboQuery(
            'INSERT INTO user_photos (user_id, filename, path, url, size) VALUES ($1, $2, $3, $4, $5)',
            [event.body.__user.uid, fileInfo.filename, fileInfo.path, fileInfo.url, fileInfo.size]
        );

        return {
            code: 200,
            response: {
                status: "success",
                message: "Photo uploaded successfully",
                file: fileInfo
            }
        };
    } catch (error) {
        return {
            code: 400,
            response: {
                status: "error",
                message: error.message
            }
        };
    }
};
```

### Document Upload with Authentication

```typescript
import { verifyAuth, createAuthErrorResponse } from '@app/utils/auth';
import { meta } from '@app/utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authentication directly in handle function
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Authentication required", event);
        }

        // Use authenticated user's data directly
        const userUid = userPayload.uid;

        const { saveBase64 } = require('fileupload');

        const fileInfo = await saveBase64(data.document, data.filename, {
            directory: `documents/${userUid}`,
            allowedTypes: [
                "application/pdf",
                "application/msword",
                "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
            ],
            maxSize: 5242880 // 5MB for documents
        });

        // Log the upload
        await turboQuery(
            'INSERT INTO document_uploads (user_id, filename, size, uploaded_at) VALUES ($1, $2, $3, NOW())',
            [userUid, fileInfo.filename, fileInfo.size]
        );

        return {
            code: 200,
            response: {
                status: "success",
                document: fileInfo,
                ...meta(event),
            }
        };
    } catch (error) {
        return {
            code: 400,
            response: {
                status: "error",
                message: error.message,
                ...meta(event),
            }
        };
    }
};
```

### File Management Endpoint

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { getFileInfo, deleteFile } = require('fileupload');
    const action = event.pathParameters.action;

    try {
        switch (action) {
            case 'info':
                const info = await getFileInfo(data.filePath);
                return {
                    code: 200,
                    response: { status: "success", file: info }
                };

            case 'delete':
                await deleteFile(data.filePath);

                // Remove from database
                await turboQuery(
                    'DELETE FROM user_files WHERE path = $1 AND user_id = $2',
                    [data.filePath, event.body.__user.uid]
                );

                return {
                    code: 200,
                    response: { status: "success", message: "File deleted" }
                };

            default:
                return {
                    code: 400,
                    response: { status: "error", message: "Invalid action" }
                };
        }
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error.message
            }
        };
    }
};
```

## Security Considerations

### File Type Validation

Always validate file types on both the client and server:

```typescript
const allowedTypes = ["image/jpeg", "image/png", "image/gif"];
if (!allowedTypes.includes(fileInfo.mimeType)) {
    throw new Error("File type not allowed");
}
```

### File Size Limits

Set appropriate file size limits:

```typescript
const maxSize = 5242880; // 5MB
if (fileData.length > maxSize) {
    throw new Error("File too large");
}
```

### Path Security

The plugin automatically prevents directory traversal attacks by:

- Sanitizing filenames
- Restricting file operations to the upload directory
- Generating safe filenames with timestamps

### Storage Location

- Store uploads outside the web-accessible directory
- Use a separate file server or CDN for serving files
- Implement proper access controls

## Error Handling

Common errors and how to handle them:

```typescript
try {
    const fileInfo = await saveBase64(data.file, data.filename);
} catch (error) {
    if (error.message.includes("file size exceeds")) {
        return { code: 413, response: { error: "File too large" } };
    } else if (error.message.includes("not allowed")) {
        return { code: 415, response: { error: "Unsupported file type" } };
    } else if (error.message.includes("invalid base64")) {
        return { code: 400, response: { error: "Invalid file data" } };
    } else {
        return { code: 500, response: { error: "Upload failed" } };
    }
}
```

## Performance Tips

1. **Validate early**: Check file size and type before processing
2. **Use streaming**: For large files, consider streaming uploads
3. **Implement cleanup**: Regular cleanup of old/unused files
4. **Monitor storage**: Track disk usage and implement alerts
5. **Use CDN**: Serve files through a CDN for better performance

## Next Steps

- [Plugin System](plugins.md) - Learn about creating custom plugins
- [Authentication](authentication.md) - Secure your file uploads
- [Database Operations](database-operations.md) - Store file metadata
- [Background Jobs](background-jobs.md) - Process files asynchronously
