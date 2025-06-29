# Caching Guide

This comprehensive guide covers TurboScript's caching system, including configuration, drivers, and best practices.

## Overview

TurboScript provides a powerful caching system with support for multiple drivers including Memory, Redis, Memcached, and File-based storage. The system ensures consistent behavior across all drivers with proper JSON serialization and async support.

## Supported Cache Drivers

### Memory Driver (`driver: "memory"`)

- **Use Case**: Development, testing, single-instance applications
- **Performance**: Fastest access, no network overhead
- **Persistence**: Data lost on application restart
- **Features**: In-memory storage with TTL support, native object storage

### Redis Driver (`driver: "redis"`)

- **Use Case**: Production applications, distributed caching
- **Performance**: Fast network-based access
- **Persistence**: Data survives application restarts
- **Features**: Persistent storage, TTL support, clustering, JSON serialization

### Memcached Driver (`driver: "memcached"`)

- **Use Case**: High-performance distributed caching
- **Performance**: Very fast, optimized for high throughput
- **Persistence**: Data lost on service restart
- **Features**: Memory-based distributed storage, TTL support, JSON serialization

### File Driver (`driver: "file"`)

- **Use Case**: Development, simple persistent storage
- **Performance**: Slower than memory-based solutions
- **Persistence**: Data survives application restarts
- **Features**: File-system based storage, JSON serialization

## Configuration

### Basic Configuration

```yaml
cache:
  default: "redis-server"  # Default cache driver to use
  drivers:
    redis-server:
      driver: "redis"
      host: "localhost"
      port: 6379
      password: ""
      db: 0
```

### Environment Variable Configuration

```yaml
cache:
  default: "redis-server"
  drivers:
    redis-server:
      driver: "redis"
      host: ${env:REDIS_HOST, "localhost"}
      port: ${env:REDIS_PORT, 6379}
      password: ${env:REDIS_PASSWORD, ""}
      db: ${env:REDIS_DB, 0}
      max_idle_connections: ${env:REDIS_MAX_IDLE, 10}
      max_active_connections: ${env:REDIS_MAX_ACTIVE, 50}
      idle_timeout: ${env:REDIS_IDLE_TIMEOUT, 300}
      read_timeout: ${env:REDIS_READ_TIMEOUT, 30}
      write_timeout: ${env:REDIS_WRITE_TIMEOUT, 30}
```

### Multiple Cache Drivers

```yaml
cache:
  default: "redis-server"
  drivers:
    # Memory cache for fast local storage
    memory-local:
      driver: "memory"
      max_size: ${env:MEMORY_MAX_SIZE, 100}
      expiration: ${env:MEMORY_EXPIRATION, 3600}

    # Redis for distributed caching
    redis-server:
      driver: "redis"
      host: ${env:REDIS_HOST, "localhost"}
      port: ${env:REDIS_PORT, 6379}
      password: ${env:REDIS_PASSWORD, ""}
      db: ${env:REDIS_DB, 0}

    # Memcached for high-performance caching
    memcached-server:
      driver: "memcached"
      host: ${env:MEMCACHED_HOST, "localhost"}
      port: ${env:MEMCACHED_PORT, 11211}
      max_idle_connections: ${env:MEMCACHED_MAX_IDLE, 10}
      max_active_connections: ${env:MEMCACHED_MAX_ACTIVE, 50}

    # File cache for simple persistence
    file-system:
      driver: "file"
      root: ${env:FILE_CACHE_ROOT, "./cache"}
      max_size: ${env:FILE_CACHE_MAX_SIZE, 10}
```

## TypeScript Usage

### Basic Operations

```typescript
import { turboCache } from "../../utils/turbo-cache";

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    // Get cache instance for specific driver
    const cache = turboCache("redis-server");

    // Store data with TTL (60 seconds)
    await cache.set("user:123", { name: "John", email: "john@example.com" }, 60);

    // Retrieve data
    const userData = await cache.get("user:123");

    // Check if key exists
    const exists = await cache.has("user:123");

    // Delete data
    await cache.del("user:123");

    // Clear all cache data
    await cache.flush();

    return {
        code: 200,
        response: { userData, exists }
    };
};
```

### Complex Data Types

```typescript
// Store complex objects
const complexData = {
    user: { id: 123, name: "John" },
    settings: { theme: "dark", notifications: true },
    array: [1, 2, 3, { nested: "value" }],
    timestamp: Date.now()
};

await cache.set("complex:data", complexData, 300); // 5 minutes TTL
const retrieved = await cache.get("complex:data");
```

### Driver-Specific Usage

```typescript
// Use memory cache for temporary data
const memoryCache = turboCache("memory-local");
await memoryCache.set("temp:data", tempValue, 30);

// Use Redis for distributed caching
const redisCache = turboCache("redis-server");
await redisCache.set("shared:data", sharedValue, 3600);

// Use default driver (as configured)
const defaultCache = turboCache();
await defaultCache.set("default:data", defaultValue, 600);
```

## Environment Variables

### Environment Variable Syntax

TurboScript supports two environment variable formats:

1. **With Default Value**: `${env:VAR_NAME, default_value}`
   - Uses environment variable if set
   - Falls back to default value if not set
   - Supports quoted strings, unquoted numbers, and booleans

2. **Required Variable**: `${env:VAR_NAME}`
   - Uses environment variable
   - Application fails to start if variable is not set

### Environment Variable Examples

```bash
# Redis Configuration
export REDIS_HOST="redis.example.com"
export REDIS_PORT="6380"
export REDIS_PASSWORD="your_secure_password"
export REDIS_DB="1"
export REDIS_MAX_IDLE="20"
export REDIS_MAX_ACTIVE="100"

# Memcached Configuration
export MEMCACHED_HOST="memcached.example.com"
export MEMCACHED_PORT="11211"
export MEMCACHED_MAX_IDLE="15"

# Memory Cache Configuration
export MEMORY_MAX_SIZE="200"
export MEMORY_EXPIRATION="7200"

# File Cache Configuration
export FILE_CACHE_ROOT="/var/cache/turboscript"
export FILE_CACHE_MAX_SIZE="100"
```

## Development Environment

### Docker Services

TurboScript includes Docker services for development:

```yaml
# docker-compose.dev.yml
services:
  redis:
    image: redis:7.4.2-alpine
    environment:
      REDIS_PASSWORD: turboscript_redis_pass
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "turboscript_redis_pass", "ping"]

  memcached:
    image: memcached:1.6.34-alpine
    command: memcached -m 64 -p 11211 -u memcache
    healthcheck:
      test: ["CMD", "nc", "-z", "localhost", "11211"]
```

### Starting Development Environment

```bash
# Start all services (app, database, Redis, Memcached)
make up

# View logs
make logs

# Stop all services
make down
```

## Data Serialization

### Memory Driver

- Stores objects natively without serialization
- Preserves exact object references and types
- Best performance for complex objects

### Network-Based Drivers (Redis, Memcached, File)

- Automatic JSON serialization for complex objects
- Primitive types (string, number, boolean) stored directly
- Objects and arrays serialized as JSON strings
- Automatic deserialization on retrieval

## TTL (Time To Live) Support

### Supported Drivers

- **Memory**: Full TTL support with automatic cleanup
- **Redis**: Full TTL support with server-side expiration
- **Memcached**: Full TTL support with server-side expiration
- **File**: Basic TTL support (implementation-dependent)

### Usage Examples

```typescript
// Set with 60-second TTL
await cache.set("short:lived", data, 60);

// Set with 1-hour TTL
await cache.set("long:lived", data, 3600);

// Set without TTL (permanent until manually deleted)
await cache.set("permanent", data);
```

## Best Practices

### 1. Choose the Right Driver

- **Development**: Memory driver for simplicity
- **Production**: Redis for reliability and persistence
- **High throughput**: Memcached for performance
- **Simple persistence**: File driver for small applications

### 2. Key Naming Conventions

```typescript
// Use descriptive, hierarchical keys
await cache.set("user:profile:123", userProfile);
await cache.set("session:token:abc123", sessionData);
await cache.set("api:response:v1:users", apiResponse);
```

### 3. TTL Management

```typescript
// Short TTL for frequently changing data
await cache.set("live:feed", feedData, 60); // 1 minute

// Medium TTL for semi-static data
await cache.set("user:settings", userSettings, 3600); // 1 hour

// Long TTL for rarely changing data
await cache.set("config:app", appConfig, 86400); // 24 hours
```

### 4. Error Resilience

```typescript
// Always provide fallbacks for cache failures
const getCachedUserData = async (userId: string) => {
    try {
        const cached = await cache.get(`user:${userId}`);
        if (cached) return cached;
    } catch (error) {
        console.warn("Cache get failed, falling back to database");
    }

    // Fallback to database
    const userData = await getUserFromDatabase(userId);

    // Try to cache for next time (but don't fail if caching fails)
    try {
        await cache.set(`user:${userId}`, userData, 600);
    } catch (error) {
        console.warn("Cache set failed, continuing without caching");
    }

    return userData;
};
```

## Testing

### Unit Tests

```bash
go test -v ./internal/config/cache_test.go ./internal/config/loader.go
```

### Integration Tests

```bash
# Start services
docker-compose -f docker-compose.dev.yml up -d

# Run tests
DOCKER_ENV=true go test -v ./internal/tsengine/
```

### Live API Testing

```bash
# Test all drivers via HTTP endpoint
curl http://localhost:7890/demo/cache-drivers-test | jq
```

## Production Deployment

### Environment Setup

```bash
# Production Redis (managed service)
export REDIS_HOST="prod-redis-cluster.example.com"
export REDIS_PORT="6380"
export REDIS_PASSWORD="production_password"
export REDIS_DB="0"

# Production Memcached
export MEMCACHED_HOST="prod-memcached.example.com"
export MEMCACHED_PORT="11211"

# Performance tuning
export REDIS_MAX_ACTIVE="200"
export REDIS_MAX_IDLE="50"
export REDIS_IDLE_TIMEOUT="600"
```

### Configuration Best Practices

1. **Use Environment Variables**: Always use env vars for deployment flexibility
2. **Set Reasonable Defaults**: Provide sensible defaults for development
3. **Connection Pooling**: Configure appropriate pool sizes for your load
4. **Timeouts**: Set realistic timeouts based on your network latency
5. **Monitoring**: Enable monitoring for cache hit rates and performance

## Troubleshooting

### Common Issues

1. **Environment Variable Not Resolved**
   - Check variable name spelling
   - Ensure variable is set in environment
   - Verify syntax: `${env:VAR_NAME, default}`

2. **YAML Parsing Errors**
   - Environment variables are resolved before YAML parsing
   - Use unquoted numeric defaults: `${env:PORT, 6379}`
   - Use quoted string defaults: `${env:HOST, "localhost"}`

3. **Cache Connection Failures**
   - Verify service is running and accessible
   - Check firewall and network configuration
   - Validate credentials and connection parameters

4. **Docker Service Issues**
   - Ensure Docker services are healthy: `docker-compose ps`
   - Check service logs: `docker-compose logs redis`
   - Verify port mapping and network connectivity

### Debug Mode

Enable debug logging to see cache operations:

```yaml
debug: true
```

This will log:

- Environment variable resolution
- Cache driver initialization
- Connection establishment
- Cache operations and performance
