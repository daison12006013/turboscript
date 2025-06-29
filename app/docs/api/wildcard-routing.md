# Wildcard Routing

TurboScript's wildcard routing feature allows you to create dynamic endpoints that automatically map URLs to TypeScript files in your filesystem. This eliminates the need to manually configure every individual endpoint and provides a more flexible, file-based routing system.

## Overview

Wildcard routing uses the `/*` pattern in route definitions to capture any path segment and dynamically resolve it to a corresponding TypeScript file. This approach is inspired by modern web frameworks like Next.js but tailored for API development.

## Configuration

### Basic Wildcard Route

```yaml
# turboscript.yml
endpoints:
  - route: /demo/*
    method: GET
    path: ./app/routes/demo/*
```

### Multiple Wildcard Routes

```yaml
# turboscript.yml
endpoints:
  # Demo endpoints
  - route: /demo/*
    method: GET
    path: ./app/routes/demo/*

  # API documentation
  - route: /api/docs/*
    method: GET
    path: ./app/routes/api/docs/*

  # Admin panel endpoints
  - route: /admin/*
    method: POST
    path: ./app/routes/admin/*
```

### Mixed Route Types

You can combine wildcard routes with traditional parameter routes:

```yaml
# turboscript.yml
endpoints:
  # Specific user endpoints
  - route: /users/{id}
    method: GET
    path: ./app/routes/users/get-by-id.ts

  # Wildcard for user-related utilities
  - route: /users/utils/*
    method: GET
    path: ./app/routes/users/utils/*

  # Wildcard for all demo endpoints
  - route: /demo/*
    method: GET
    path: ./app/routes/demo/*
```

## File Resolution Algorithm

When a request matches a wildcard route, TurboScript follows this resolution process:

1. **Extract Wildcard Path**: Parse the URL to extract the portion matching the `*`
2. **Clean Path**: Remove query parameters, fragments, and normalize the path
3. **Security Check**: Validate the path to prevent directory traversal attacks
4. **File Search**: Look for files with the following priority:
   - `.ts` files (TypeScript)
   - `.js` files (JavaScript)
5. **Index Resolution**: If the wildcard path is empty, look for `index.ts` or `index.js`
6. **Return Result**: If found, execute the file; otherwise, return 404

### Example Resolution

Given this configuration:

```yaml
endpoints:
  - route: /demo/*
    method: GET
    path: ./app/routes/demo/*
```

And this file structure:

```text
app/routes/demo/
├── index.ts
├── cache-test.ts
├── response-types.ts
└── api/
    ├── index.ts
    └── endpoints.ts
```

**URL Resolution:**

| Request URL | Resolved File | Status |
|-------------|---------------|--------|
| `/demo/` | `./app/routes/demo/index.ts` | ✅ Success |
| `/demo` | `./app/routes/demo/index.ts` | ✅ Success |
| `/demo/cache-test` | `./app/routes/demo/cache-test.ts` | ✅ Success |
| `/demo/response-types` | `./app/routes/demo/response-types.ts` | ✅ Success |
| `/demo/api/` | `./app/routes/demo/api/index.ts` | ✅ Success |
| `/demo/api/endpoints` | `./app/routes/demo/api/endpoints.ts` | ✅ Success |
| `/demo/non-existent` | None | ❌ 404 Not Found |

## Security Features

### Directory Traversal Prevention

Wildcard routing includes robust security measures to prevent directory traversal attacks:

```typescript
// ❌ These requests are automatically blocked:
GET /demo/../../../etc/passwd
GET /demo/%2e%2e/secret-file
GET /demo//etc/passwd
```

### Path Validation

- **Base Path Enforcement**: Resolved paths must stay within the configured base directory
- **Clean Path Resolution**: Uses `filepath.Clean()` to normalize paths
- **Prefix Checking**: Validates that resolved paths have the correct base prefix
- **Error Logging**: Security violations are logged for monitoring

### File Existence Verification

Only files that actually exist on the filesystem are served. Non-existent files return proper 404 responses.

## Route Handler Structure

Wildcard routes use the same handler structure as regular routes:

```typescript
// app/routes/demo/cache-test.ts
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Your route logic here
        const result = await turboQuery('SELECT * FROM cache_demo');

        return {
            code: 200,
            response: {
                status: "success",
                data: result,
                meta: {
                    resolved_via: "wildcard_routing",
                    file_path: "demo/cache-test.ts",
                    request_path: event.path
                }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "An error occurred"
            }
        };
    }
};
```

## Advanced Usage

### Nested Wildcard Routes

You can create deeply nested wildcard structures:

```yaml
endpoints:
  # API versioning with wildcards
  - route: /api/{version}/docs/*
    method: GET
    path: ./app/routes/api/{version}/docs/*

  # Multi-level admin routes
  - route: /admin/{section}/*
    method: POST
    path: ./app/routes/admin/{section}/*
```

### Method-Specific Wildcards

Different HTTP methods can have different wildcard handlers:

```yaml
endpoints:
  # GET requests for reading
  - route: /api/data/*
    method: GET
    path: ./app/routes/api/data/read/*

  # POST requests for writing
  - route: /api/data/*
    method: POST
    path: ./app/routes/api/data/write/*

  # DELETE requests for removal
  - route: /api/data/*
    method: DELETE
    path: ./app/routes/api/data/delete/*
```

### Query Parameters and Fragments

Wildcard routing automatically handles query parameters and URL fragments:

```bash
# These all resolve to the same file:
GET /demo/cache-test
GET /demo/cache-test?param=value
GET /demo/cache-test?param=value&other=123
GET /demo/cache-test#section

# Resolved file: ./app/routes/demo/cache-test.ts
```

## Performance Considerations

### File System Caching

- File existence checks are performed using `os.Stat()`
- Consider implementing file existence caching for high-traffic applications
- The filesystem resolution is optimized for development convenience

### Route Ordering

Place more specific routes before wildcard routes in your configuration:

```yaml
endpoints:
  # Specific routes first
  - route: /demo/special-endpoint
    method: GET
    path: ./app/routes/demo/special.ts

  # Wildcard routes last
  - route: /demo/*
    method: GET
    path: ./app/routes/demo/*
```

### Development vs Production

- **Development**: TypeScript files are resolved directly
- **Production**: Use the same TypeScript files (no compilation to JavaScript needed)
- **Hot Reloading**: Changes to TypeScript files are automatically detected in development

## Debugging

Enable debug logging to see wildcard route resolution:

```yaml
# turboscript.yml
debug: true
```

This will log:

- Route matching decisions
- File resolution attempts
- Security blocking events
- Successful file resolutions

## Migration from Manual Routes

If you have existing manual route definitions, you can gradually migrate to wildcard routing:

### Before (Manual Routes)

```yaml
endpoints:
  - route: /demo/cache-test
    method: GET
    path: ./app/routes/demo/cache-test.ts
  - route: /demo/response-types
    method: GET
    path: ./app/routes/demo/response-types.ts
  - route: /demo/api-demo
    method: GET
    path: ./app/routes/demo/api-demo.ts
```

### After (Wildcard Route)

```yaml
endpoints:
  - route: /demo/*
    method: GET
    path: ./app/routes/demo/*
```

**Benefits of Migration:**

- **Reduced Configuration**: Single route definition replaces multiple entries
- **Automatic Discovery**: New files are automatically available as endpoints
- **Easier Maintenance**: No need to update configuration when adding/removing endpoints
- **Better Organization**: File structure directly reflects URL structure

## Best Practices

1. **Organize by Feature**: Group related endpoints in directories
2. **Use Index Files**: Provide `index.ts` files for directory root endpoints
3. **Consistent Naming**: Use kebab-case for file names to match URL conventions
4. **Security First**: Never disable security checks; they're essential for production
5. **Document Structure**: Keep your file structure well-documented for team development
6. **Test Coverage**: Write tests for wildcard routes to ensure proper resolution

## Troubleshooting

### Common Issues

#### File Not Found (404)

- Check file name spelling and case sensitivity
- Ensure the file has the correct extension (`.ts` or `.js`)
- Verify the file exists in the expected directory

#### Security Blocking

- Check logs for security violation messages
- Ensure request URLs don't contain `../` or similar patterns
- Verify the request stays within the configured base path

#### Wrong File Resolved

- Check route ordering in `turboscript.yml`
- Ensure more specific routes come before wildcard routes
- Verify the wildcard pattern matches your intention

### Debug Commands

```bash
# Check file structure
find app/routes -name "*.ts" -type f

# Test endpoint
curl -v http://localhost:7890/demo/cache-test

# Check logs
docker logs turboscript-app-dev-1 --tail=20
```
