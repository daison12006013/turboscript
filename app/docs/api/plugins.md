# Plugin System

TurboScript features a powerful plugin system that allows you to extend the framework with custom functionality. Plugins are automatically loaded and their functions become available in your TypeScript route handlers.

## Built-in Plugins

### File Upload Plugin

The file upload plugin provides binary file handling and upload management capabilities.

#### Configuration

Add the file upload plugin to your `turboscript.yml`:

```yaml
plugins:
  - name: fileupload
    enabled: true
    options:
      upload_dir: "./uploads"           # Directory to store uploaded files
      max_file_size: 10485760          # Maximum file size in bytes (10MB)
      allowed_types:                   # Allowed MIME types
        - "image/jpeg"
        - "image/png"
        - "image/gif"
        - "image/webp"
        - "application/pdf"
        - "text/plain"
```

#### Usage in TypeScript

```typescript
// Import the plugin (automatically available when enabled)
const { saveBase64, saveFile, getFileInfo, deleteFile } = require('fileupload');

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Example 1: Save base64 encoded file
        const base64Data = data.image; // "data:image/png;base64,iVBORw0KGgoAAAA..."
        const fileInfo = await saveBase64(base64Data, "photo.png", {
            directory: "photos",
            generateHash: true
        });

        // Example 2: Save binary file data
        const binaryData = new Uint8Array([...]); // Binary file data
        const fileInfo2 = await saveFile(binaryData, {
            filename: "document.pdf",
            directory: "documents",
            allowedTypes: ["application/pdf"],
            maxSize: 5242880 // 5MB
        });

        // Example 3: Get file information
        const info = await getFileInfo("./uploads/photos/photo.png");

        // Example 4: Delete a file
        await deleteFile("./uploads/old-file.png");

        return {
            code: 200,
            response: {
                status: "success",
                uploaded: fileInfo,
                info: info
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

#### File Upload API

saveBase64(base64Data, filename, options)

- Saves base64 encoded file data
- Returns: FileInfo object

saveFile(binaryData, options)

- Saves binary file data
- Returns: FileInfo object

getFileInfo(filePath)

- Gets information about an existing file
- Returns: FileInfo object

deleteFile(filePath)

- Deletes a file from the filesystem
- Returns: void

validateFile(data, contentType, options)

- Validates file against constraints
- Returns: validation result

generateFilename(originalName)

- Generates a unique filename
- Returns: string

hashFile(filePath)

- Generates MD5 and SHA256 hashes for a file
- Returns: object with hash values

#### FileInfo Object

```typescript
interface FileInfo {
    originalName: string;  // Original filename
    filename: string;      // Generated filename
    size: number;         // File size in bytes
    mimeType: string;     // MIME type
    extension: string;    // File extension
    path: string;         // Full file path
    url: string;          // URL to access the file
    md5Hash: string;      // MD5 hash (if generateHash: true)
    sha256Hash: string;   // SHA256 hash (if generateHash: true)
    uploadedAt: string;   // Upload timestamp (ISO 8601)
}
```

#### Upload Options

```typescript
interface UploadOptions {
    directory?: string;      // Subdirectory within upload_dir
    filename?: string;       // Custom filename
    allowedTypes?: string[]; // Override allowed MIME types
    maxSize?: number;        // Override max file size
    generateHash?: boolean;  // Generate file hashes (default: true)
}
```

## Creating Custom Plugins

### Plugin Interface

To create a custom plugin, implement the Plugin interface:

```go
type Plugin interface {
    Name() string                                              // Unique plugin name
    Initialize(config map[string]any) error                   // Initialize with config
    Register(runtime *goja.Runtime, registry *require.Registry) error // Register with JS runtime
    Description() string                                       // Human-readable description
    Version() string                                          // Plugin version
}
```

### Example Custom Plugin

```go
package myplugin

import (
    "github.com/dop251/goja"
    "github.com/dop251/goja_nodejs/require"
)

type MyPlugin struct {
    config map[string]any
}

func (p *MyPlugin) Name() string {
    return "myplugin"
}

func (p *MyPlugin) Description() string {
    return "My custom plugin for TurboScript"
}

func (p *MyPlugin) Version() string {
    return "1.0.0"
}

func (p *MyPlugin) Initialize(config map[string]any) error {
    p.config = config
    return nil
}

func (p *MyPlugin) Register(runtime *goja.Runtime, registry *require.Registry) error {
    // Register your plugin's functions
    module := map[string]interface{}{
        "myFunction": p.myFunction,
        "anotherFunction": p.anotherFunction,
    }

    // Register as require module
    registry.RegisterNativeModule("myplugin", func(runtime *goja.Runtime, module *goja.Object) {
        for name, fn := range module {
            module.Set(name, fn)
        }
    })

    return nil
}

func (p *MyPlugin) myFunction(param string) string {
    return "Hello from plugin: " + param
}

func (p *MyPlugin) anotherFunction(data any) (map[string]any, error) {
    return map[string]any{
        "processed": true,
        "data": data,
    }, nil
}

func NewMyPlugin() *MyPlugin {
    return &MyPlugin{}
}
```

### Register Your Plugin

In your main.go or an init file:

```go
import (
    "github.com/daison12006013/turboscript/internal/plugins"
    "yourproject/internal/plugins/myplugin"
)

func init() {
    // Register your custom plugin
    plugins.RegisterGlobalPlugin(myplugin.NewMyPlugin())
}
```

### Configure Your Plugin

Add to `turboscript.yml`:

```yaml
plugins:
  - name: myplugin
    enabled: true
    options:
      setting1: "value1"
      setting2: 42
      setting3: true
```

### Use Your Plugin

In TypeScript route handlers:

```typescript
const { myFunction, anotherFunction } = require('myplugin');

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const result = myFunction("test");
    const processed = await anotherFunction(data);

    return {
        code: 200,
        response: { result, processed }
    };
};
```

## Plugin Best Practices

### Security

1. **Validate all inputs** in your plugin functions
2. **Sanitize file paths** for file-based operations
3. **Limit resource usage** (memory, CPU, disk)
4. **Use safe defaults** in configuration

### Performance

1. **Cache expensive operations** when possible
2. **Use async operations** for I/O
3. **Implement proper timeouts**
4. **Clean up resources** in plugin lifecycle

### Error Handling

1. **Return descriptive errors** from plugin functions
2. **Handle edge cases gracefully**
3. **Log errors appropriately**
4. **Provide recovery mechanisms**

### Setup

1. **Use meaningful configuration keys**
2. **Provide sensible defaults**
3. **Validate configuration** during initialization
4. **Document all options**

## Plugin Development Workflow

1. **Design your plugin interface** - what functions will it expose?
2. **Implement the Plugin interface** in Go
3. **Register JavaScript functions** that TypeScript can call
4. **Add configuration support** for customization
5. **Write tests** for your plugin functionality
6. **Document usage** and configuration options
7. **Register with plugin manager** during startup

## Available Plugin Types

TurboScript plugins can provide various types of functionality:

- **File Processing**: Upload, resize, convert files
- **External APIs**: Integrate with third-party services
- **Data Processing**: Transform, validate, analyze data
- **Authentication**: Custom auth providers
- **Notifications**: Email, SMS, push notifications
- **Caching**: Custom cache implementations
- **Monitoring**: Metrics, logging, tracing
- **Database**: Custom database drivers or ORMs

The plugin system is designed to be flexible and extensible, allowing you to add any functionality your application needs while maintaining the security and performance of the TurboScript runtime.
