# Architecture Overview

TurboScript is a hybrid web framework that combines TypeScript for business logic and Go for runtime execution. This document explains the key architectural components and how they work together.

## High-Level Architecture

TurboScript uses a unique architecture where:

- TypeScript code defines the API business logic
- Go runtime executes the TypeScript code using a JavaScript VM (goja)
- FastHTTP handles web server functionality
- PostgreSQL serves as the primary database

## Core Components

### TypeScript Application Layer (`app/`)

- **Routes (`app/routes/`)**: Contains API endpoint handlers
  - Async `handle()` functions with direct `turboQuery()` access
  - Clean async/await pattern for database operations

- **Utils (`app/utils/`)**: Shared utilities
  - Authentication helpers
  - Password hashing
  - JWT token management
  - Response metadata generators

- **Global Types (`app/global.d.ts`)**: Framework type definitions

### Go Runtime Engine (`internal/`)

- **Server (`internal/server/`)**:
  - FastHTTP-based web server
  - Dynamic route handling
  - Request/response management

- **TypeScript Engine (`internal/tsengine/`)**:
  - Goja VM integration
  - TypeScript execution
  - Runtime utilities
  - Error handling

- **Database Executor (`internal/dbexecutor/`)**:
  - Secure query execution
  - Table access restrictions
  - Connection pooling

- **Configuration (`internal/config/`)**:
  - YAML configuration loader
  - Environment variable handling

## Request Flow

1. HTTP request received by FastHTTP server
2. Route matched to TypeScript handler
3. TypeScript code executed in Goja VM
4. Database queries processed (if any)
5. Response formatted and returned

## Security Architecture

- JWT-based authentication
- Secure password hashing
- SQL injection prevention
- Table-level access control
- CORS and security headers

## Performance Features

- Non-blocking async operations
- Connection pooling
- Response caching
- Hot-reloading for development
- Production optimizations

## Configuration

The `turboscript.yml` file controls:

- Route definitions
- Database settings
- Security parameters
- Debug options
- Environment variables

## Development vs Production

### Development

- Hot-reloading enabled
- Detailed error messages
- Debug logging
- Source maps

### Production

- Optimized builds
- Minified code
- Error sanitization
- Performance monitoring

## Summary

TurboScript's architecture provides a unique blend of TypeScript's developer experience with Go's performance. The key benefits include:

- **Developer Productivity**: Write business logic in TypeScript with full type safety
- **Runtime Performance**: Go handles execution, networking, and system operations
- **Async Database Access**: Modern Promise-based database operations
- **Hot Reloading**: Instant feedback during development
- **Security**: Built-in protection against common vulnerabilities

## Detailed Architecture Deep Dive

### TypeScript Execution Flow

TurboScript uses a sophisticated execution model that bridges TypeScript and Go:

```text
1. HTTP Request → FastHTTP Server
2. Route Matching → Configuration Lookup
3. TypeScript File → esbuild Compilation
4. JavaScript VM → Goja Runtime Execution
5. Database Queries → Go Query Executor
6. Response → JSON/HTML/Text Output
```

### Runtime Components

#### JavaScript VM Integration

TurboScript uses Goja (pure Go JavaScript engine) for executing TypeScript code:

```go
// Runtime setup in Go
vm := goja.New()
vm.Set("turboQuery", func(call goja.FunctionCall) goja.Value {
    // Secure database query execution
    return executeQuery(call, rt)
})

// Execute TypeScript-compiled JavaScript
result, err := vm.RunString(compiledJS)
```

#### Event Loop Manager

For async operations, TurboScript implements an event loop:

```go
type EventLoopManager struct {
    registry      *require.Registry
    runtimePool   sync.Pool
    eventLoop     *eventloop.EventLoop
}

// Handles Promise resolution, setTimeout, async/await
func (elm *EventLoopManager) RunAsync(tsCode string) (any, error) {
    // Execute with full async support
}
```

#### Compilation Pipeline

TypeScript to JavaScript compilation using esbuild:

```javascript
// scripts/build-ts.js - Production compilation
const result = await esbuild.build({
    entryPoints: [tsFilePath],
    target: 'es2020',
    format: 'cjs',
    bundle: false,
    minify: true,        // Production optimization
    sourcemap: false,    // Disable for production
    write: false,
    platform: 'node'
});
```

### Security Architecture Deep Dive

#### Table Access Control

Database security is enforced at the Go level:

```go
// internal/dbexecutor/executor.go
type Executor struct {
    db            *sql.DB
    allowedTables map[string]bool
    dangerousOps  []string
}

func (e *Executor) validateQuery(query string) error {
    // Parse SQL to extract table names
    // Check against whitelist
    // Block dangerous operations
}
```

#### SQL Injection Prevention

Multi-layer protection:

```go
func (e *Executor) ExecuteQuery(query string, params []any) ([]map[string]any, error) {
    // 1. Static analysis of query structure
    if err := e.analyzeQuery(query); err != nil {
        return nil, fmt.Errorf("dangerous query detected: %w", err)
    }

    // 2. Parameter binding validation
    stmt, err := e.db.Prepare(query)
    if err != nil {
        return nil, err
    }
    defer stmt.Close()

    // 3. Execute with bound parameters
    rows, err := stmt.Query(params...)
    // ...
}
```

#### JWT Security Implementation

Token verification with multiple layers:

```typescript
// utils/auth.ts
export function verifyAuth(event: Event): JWTPayload | null {
    // 1. Extract from Authorization header (Bearer token)
    let token = extractBearerToken(event.headers.authorization);

    // 2. Fallback to case-insensitive header
    if (!token) {
        token = extractBearerToken(event.headers.Authorization);
    }

    // 3. Fallback to cookie
    if (!token) {
        token = extractFromCookie(event.headers.cookie);
    }

    if (!token) return null;

    // 4. Verify signature and expiration
    return verifyJWTToken(token);
}
```

### Performance Architecture

#### Connection Pooling

Database connections are managed efficiently:

```go
// Database configuration
db, err := sql.Open("postgres", connectionString)
db.SetMaxOpenConns(25)      // Maximum connections
db.SetMaxIdleConns(10)      // Idle connections
db.SetConnMaxLifetime(5 * time.Minute)
```

#### Runtime Pooling

JavaScript runtimes are pooled for performance:

```go
type RuntimeUtils struct {
    pool sync.Pool
}

func (ru *RuntimeUtils) GetRuntime() *JSRuntime {
    // Get from pool or create new
    if rt := ru.pool.Get(); rt != nil {
        return rt.(*JSRuntime)
    }
    return ru.createNewRuntime()
}

func (ru *RuntimeUtils) ReturnRuntime(rt *JSRuntime) {
    // Clean runtime state and return to pool
    rt.Reset()
    ru.pool.Put(rt)
}
```

#### Query Compilation Caching

Compiled JavaScript is cached for performance:

```go
type CacheUtils struct {
    compiledJS map[string]string  // File path -> compiled JS
    mutex      sync.RWMutex
    lastMod    map[string]time.Time
}

func (cu *CacheUtils) GetCompiledJS(tsPath string) (string, error) {
    // Check cache first
    if cached, exists := cu.getFromCache(tsPath); exists {
        return cached, nil
    }

    // Compile and cache
    compiled, err := cu.compileTypeScript(tsPath)
    if err != nil {
        return "", err
    }

    cu.setCache(tsPath, compiled)
    return compiled, nil
}
```

### Request/Response Flow

#### Detailed Request Processing

```go
// internal/server/routing.go
func (s *Server) routeHandler(ctx *fasthttp.RequestCtx) {
    // 1. Performance monitoring start
    perfCtx := s.perfManager.StartRequest()
    defer s.perfManager.EndRequest(perfCtx)

    // 2. Extract request data
    requestData := s.extractRequestData(ctx)

    // 3. Find matching endpoint
    endpoint := s.findMatchingEndpoint(string(ctx.Path()), string(ctx.Method()))
    if endpoint == nil {
        s.handle404(ctx)
        return
    }

    // 4. Execute TypeScript handler
    s.handleEndpoint(ctx, *endpoint, perfCtx)
}

func (s *Server) handleEndpoint(ctx *fasthttp.RequestCtx, ep config.EndpointConfig, perfCtx *performance.RequestContext) {
    // Authorization check
    if authResult := s.executeAuth(ep.Path, event); authResult != nil {
        s.sendResponse(ctx, authResult)
        return
    }

    // Execute main handler
    result := s.responseUtils.ExecuteHandle(ep.Path, data, event, false)
    s.sendResponse(ctx, result)
}
```

#### Response Type Detection

```go
// internal/server/server.go
func autoDetectResponseType(response any) string {
    switch v := response.(type) {
    case string:
        content := strings.TrimSpace(v)
        if strings.HasPrefix(content, "<!DOCTYPE") || strings.HasPrefix(content, "<html") {
            return "html"
        }
        if strings.HasPrefix(content, "#") || strings.Contains(content, "##") {
            return "markdown"
        }
        return "text"
    default:
        return "json"
    }
}
```

### Development vs Production Architecture

#### Development Mode Features

```yaml
# turboscript.dev.yml
debug: true
monitoring: true
hot_reload: true

database:
  debug: true                # Log all queries
  log_slow_queries: true     # Performance monitoring

jobs:
  max_workers: 2             # Fewer workers for development
  debug: true                # Detailed job logging
```

Development mode includes:

- **Hot Reloading**: File watching and automatic restart
- **Detailed Logging**: Full request/response logging
- **Debug Information**: Stack traces and performance metrics
- **Source Maps**: For TypeScript debugging

#### Production Optimizations

```yaml
# turboscript.yml (production)
debug: false
monitoring: true

database:
  debug: false
  max_connections: 25        # Higher connection pool
  query_timeout: 10          # Shorter timeouts

jobs:
  max_workers: 10            # More workers for production
  queue_size: 5000          # Larger queue
  retry_attempts: 3          # Robust retry logic
```

Production optimizations:

- **Compiled Assets**: Pre-compiled JavaScript bundles
- **Minification**: Reduced bundle sizes
- **Error Sanitization**: Safe error messages for users
- **Performance Monitoring**: Comprehensive metrics collection
- **Resource Limits**: Memory and CPU constraints

### Monitoring and Observability

#### Built-in Performance Monitoring

```go
// internal/performance/monitor.go
type RequestContext struct {
    RequestID     string
    StartTime     time.Time
    Endpoint      string
    Method        string
    UserAgent     string
    IPAddress     string
}

func (pm *Manager) StartRequest() *RequestContext {
    return &RequestContext{
        RequestID: generateRequestID(),
        StartTime: time.Now(),
    }
}

func (pm *Manager) EndRequest(ctx *RequestContext) {
    duration := time.Since(ctx.StartTime)

    // Log performance metrics
    logger.Info("Request completed: %s %s - %v",
        ctx.Method, ctx.Endpoint, duration)

    // Update metrics
    pm.updateMetrics(ctx, duration)
}
```

#### Goroutine Monitoring

```go
// internal/performance/goroutine_debug.go
func (gd *GoroutineDebugger) LogGoroutineStats() {
    numGoroutines := runtime.NumGoroutine()

    if numGoroutines > gd.threshold {
        logger.Warn("High goroutine count detected: %d", numGoroutines)

        // Capture stack traces for debugging
        buf := make([]byte, 1<<20) // 1MB buffer
        stackSize := runtime.Stack(buf, true)
        logger.Debug("Goroutine stack trace:\n%s", buf[:stackSize])
    }
}
```

### Error Handling Architecture

#### Multi-Level Error Handling

```go
// 1. Go-level error handling
func (ru *ResponseUtils) ExecuteHandle(tsPath string, data any, event any) (json.RawMessage, error) {
    defer func() {
        if r := recover(); r != nil {
            logger.Error("Panic in TypeScript execution: %v", r)
        }
    }()

    // 2. JavaScript compilation errors
    jsCode, err := ru.compileTypeScript(tsPath)
    if err != nil {
        return nil, fmt.Errorf("compilation failed: %w", err)
    }

    // 3. Runtime execution errors
    result, err := ru.executeJS(jsCode, data, event)
    if err != nil {
        return nil, fmt.Errorf("execution failed: %w", err)
    }

    return result, nil
}
```

#### TypeScript Error Handling

```typescript
// Standardized error responses
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Business logic
        return { code: 200, response: { status: "success", data: result } };
    } catch (error) {
        // Log error with context
        console.error('Handler error:', {
            path: event.pathParameters,
            query: event.queryParameters,
            error: error.message
        });

        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "An unexpected error occurred"
            }
        };
    }
};
```

## Scaling Considerations

### Horizontal Scaling

TurboScript applications can be scaled horizontally:

```yaml
# Multiple instances behind load balancer
services:
  turboscript-1:
    build: .
    environment:
      - DB_HOST=database-host
      - DB_USERNAME=user
      - DB_PASSWORD=pass
      - DB_NAME=turboscript
      - INSTANCE_ID=1

  turboscript-2:
    build: .
    environment:
      - DB_HOST=database-host
      - DB_USERNAME=user
      - DB_PASSWORD=pass
      - DB_NAME=turboscript
      - INSTANCE_ID=2
```

### Database Scaling

- **Connection Pooling**: Shared pool across instances
- **Read Replicas**: Separate read/write operations
- **Query Optimization**: Indexed queries and efficient joins

### Job Queue Scaling

- **Worker Distribution**: Jobs distributed across instances
- **Queue Persistence**: Database-backed job storage
- **Retry Logic**: Robust failure handling

---

## Navigation

**Previous:** [← Development Workflow](guides/development.md)
**Next:** [Route Handlers →](api/route-handlers.md)

## Related Topics

- [Route Handler API](api/route-handlers.md)
- [Database Operations](api/database-operations.md)
- [Best Practices](guides/best-practices.md)
- [Performance Guide](guides/performance.md)
