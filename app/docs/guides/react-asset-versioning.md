# React Asset Versioning System

## Overview

TurboScript provides a powerful asset versioning system for React applications that automatically generates cache-busting URLs for your CSS, JavaScript, and other static assets. This system ensures that browsers always load the latest version of your assets while enabling aggressive caching for performance.

## How Asset Versioning Works

### 1. Hash-Based Versioning

The system generates MD5 hashes based on file content:

```go
// Example: styles.css content -> MD5 hash -> b48d5304f1ffdd2c
/frontend/assets/styles.css?v=b48d5304f1ffdd2c
```

**Key Benefits:**

- **Content-based**: Version changes only when file content changes
- **Cache-friendly**: Unchanged files keep the same version, maximizing cache hits
- **Automatic**: No manual version management required
- **Collision-resistant**: MD5 provides sufficient uniqueness for asset versioning

### 2. Automatic Asset Map Generation

The system automatically scans your assets directory and creates a mapping:

```go
assetMap := map[string]string{
    "styles.css": "/frontend/assets/styles.css?v=b48d5304f1ffdd2c",
    "app.js":     "/frontend/assets/app.js?v=68f2ea58eb4c7770",
}
```

### 3. Template Integration

Assets are injected into your React app template using Go templates:

```html
{{if and .Assets (index .Assets "styles.css")}}
    <link href='{{index .Assets "styles.css"}}' rel="stylesheet">
{{else}}
    <link href="/frontend/assets/styles.css" rel="stylesheet">
{{end}}
```

## Implementation Details

### Configuration

Enable asset versioning in your `turboscript.yml`:

```yaml
endpoints:
  - route: "/frontend/*"
    path: "app/frontend"
    type: "react"
    options:
      assets: "assets"
      app: "App.html"
      cache_control:
        asset_versioning: true
        static_assets:
          development: "no-cache, no-store, must-revalidate"
          production: "public, max-age=31536000"
        html_pages:
          development: "no-cache, no-store, must-revalidate"
          production: "public, max-age=300"
```

### Asset Discovery Process

1. **Directory Scanning**: The system scans the configured assets directory
2. **Common Assets**: Automatically includes `styles.css` and `app.js`
3. **Hash Generation**: Creates MD5 hash from file content (first 8 bytes for shorter URLs)
4. **URL Generation**: Constructs versioned URLs with base route and query parameters

```go
// Implementation in internal/server/react.go
func (s *Server) generateAssetMap(ep config.EndpointConfig) map[string]string {
    // Scans assets directory
    // Generates content-based hashes
    // Creates versioned URL mapping
}
```

### Template Data Structure

The React template receives a `ReactData` struct:

```go
type ReactData struct {
    Route  string            // Current route (e.g., "/frontend")
    Data   string            // JSON-encoded initial data
    Assets map[string]string // Asset name -> Versioned URL mapping
}
```

### Fallback Mechanism

If asset versioning fails or is disabled, the template falls back to standard URLs:

```html
<!-- With versioning -->
<link href="/frontend/assets/styles.css?v=b48d5304f1ffdd2c" rel="stylesheet">

<!-- Fallback -->
<link href="/frontend/assets/styles.css" rel="stylesheet">
```

## Adding New Assets

### 1. Standard Assets (Automatic)

For common assets like CSS and JavaScript, simply place them in your assets directory:

```text
app/frontend/
├── App.html
├── assets/
│   ├── styles.css    ← Automatically versioned
│   ├── app.js        ← Automatically versioned
│   └── custom.css    ← Manual template addition required
```

### 2. Custom Assets (Manual Template Integration)

For additional assets, update your `App.html` template:

```html
<!-- Add custom CSS -->
{{if and .Assets (index .Assets "custom.css")}}
<link href='{{index .Assets "custom.css"}}' rel="stylesheet">
{{else}}
<link href="/frontend/assets/custom.css" rel="stylesheet">
{{end}}

<!-- Add custom JavaScript -->
{{if and .Assets (index .Assets "custom.js")}}
<script src='{{index .Assets "custom.js"}}'></script>
{{else}}
<script src="/frontend/assets/custom.js"></script>
{{end}}

<!-- Add images with versioning -->
{{if and .Assets (index .Assets "logo.png")}}
<img src='{{index .Assets "logo.png"}}' alt="Logo">
{{else}}
<img src="/frontend/assets/logo.png" alt="Logo">
{{end}}
```

### 3. Extending Asset Discovery

To automatically version additional asset types, modify the asset generation logic in `internal/server/react.go`:

```go
func (s *Server) generateAssetMap(ep config.EndpointConfig) map[string]string {
    // Current common assets
    commonAssets := []string{"styles.css", "app.js"}

    // Add your custom assets
    commonAssets = append(commonAssets, "custom.css", "vendor.js", "logo.png")

    // Rest of the implementation...
}
```

## Accessing Assets in JavaScript

### Via Template Data

Assets are also available through the route data injected into `window.__ROUTE_DATA__`:

```javascript
// Access asset URLs in your React components
const assetMap = window.__ROUTE_DATA__.assets;

// Use versioned URLs
const logoUrl = assetMap['logo.png'] || '/frontend/assets/logo.png';
const customCssUrl = assetMap['custom.css'] || '/frontend/assets/custom.css';
```

### Dynamic Asset Loading

For dynamically loaded assets:

```javascript
// Utility function to get versioned asset URL
function getAssetUrl(assetName, fallbackPath) {
    const assetMap = window.__ROUTE_DATA__.assets || {};
    return assetMap[assetName] || fallbackPath;
}

// Usage examples
const dynamicStylesheet = getAssetUrl('theme.css', '/frontend/assets/theme.css');
const dynamicScript = getAssetUrl('plugin.js', '/frontend/assets/plugin.js');

// Dynamically load CSS
const link = document.createElement('link');
link.rel = 'stylesheet';
link.href = dynamicStylesheet;
document.head.appendChild(link);

// Dynamically load JavaScript
const script = document.createElement('script');
script.src = dynamicScript;
document.head.appendChild(script);
```

## Best Practices

### 1. Template Syntax

**✅ Correct:**

```html
{{if and .Assets (index .Assets "styles.css")}}
<link href='{{index .Assets "styles.css"}}' rel="stylesheet">
```

**❌ Incorrect:**

```html
{{if and .Assets (index .Assets " styles.css")}}
<link href='{{index .Assets " styles.css"}}' rel="stylesheet">
```

> **Note**: Extra spaces in asset keys will cause template lookup failures.

### Asset Organization

```text
app/frontend/assets/
├── styles.css          # Main stylesheet
├── app.js             # Main application bundle
├── vendor/            # Third-party assets
│   ├── bootstrap.css
│   └── jquery.js
├── images/            # Image assets
│   ├── logo.png
│   └── icons/
└── fonts/             # Font assets
    ├── roboto.woff2
    └── icons.ttf
```

### 3. Cache Configuration

Configure different cache strategies for development and production:

```yaml
cache_control:
  asset_versioning: true
  static_assets:
    development: "no-cache, no-store, must-revalidate"
    production: "public, max-age=31536000"  # 1 year
  html_pages:
    development: "no-cache, no-store, must-revalidate"
    production: "public, max-age=300"       # 5 minutes
```

### 4. Build Pipeline Integration

Ensure your build process generates assets before starting the server:

```bash
# In your development workflow
npm run build:frontend  # Generates styles.css and app.js
go run main.go          # Starts server with asset versioning
```

## Troubleshooting

### Common Issues

1. **Empty href/src attributes**: Check for extra spaces in template asset keys
2. **Assets not versioned**: Verify `asset_versioning: true` in configuration
3. **File not found**: Ensure assets exist in the configured assets directory
4. **Cache not working**: Check cache control headers in network tab

### Debug Mode

Enable debug logging to see asset map generation:

```yaml
debug: true
```

Look for logs like:

```text
Generated asset map: map[app.js:/frontend/assets/app.js?v=68f2ea58eb4c7770 styles.css:/frontend/assets/styles.css?v=b48d5304f1ffdd2c]
```

### Manual Testing

Test asset URLs directly:

```bash
# Test versioned asset
curl -I http://localhost:7890/frontend/assets/styles.css?v=b48d5304f1ffdd2c

# Test fallback asset
curl -I http://localhost:7890/frontend/assets/styles.css
```

## Security Considerations

- **MD5 for versioning**: MD5 is used only for asset versioning, not security
- **Path validation**: All asset paths are validated to prevent directory traversal
- **Content-Type headers**: Proper MIME types are set for all asset types
- **XSS prevention**: JSON data is sanitized when embedded in HTML templates

## Performance Benefits

- **Browser caching**: Versioned assets can be cached aggressively
- **CDN optimization**: Content-based hashing works well with CDNs
- **Parallel loading**: CSS and JS can be loaded concurrently
- **Cache invalidation**: Only changed files require cache invalidation
