# Testing Cache Drivers

This guide explains how to test TurboScript's cache drivers and troubleshoot common issues.

## Overview

TurboScript supports multiple cache drivers with comprehensive test coverage:

- **Memory**: In-memory caching for development
- **Redis**: Distributed caching with persistence
- **Memcached**: High-performance distributed caching
- **File**: File-system based caching for simple setups

## Running Cache Tests

### Unit Tests

Run all cache-related unit tests:

```bash
go test -v ./internal/tsengine -run "TestTurboCacheAllDrivers"
```

### Docker Environment Tests

For Redis and Memcached tests requiring Docker services:

```bash
# Start Docker services
make up

# Run tests with proper environment detection
go test -v ./internal/tsengine -run "TestTurboCacheAllDrivers"
```

### Integration Tests

Run integration tests in Docker environment:

```bash
docker-compose -f docker-compose.dev.yml exec app-dev sh -c "DOCKER_ENV=true go test -v ./internal/config/..."
```

## Environment Configuration

The cache tests automatically detect the environment and configure connection hosts:

### Local Development

- **Redis**: `localhost:6379`
- **Memcached**: `localhost:11211`

### Docker Environment

- **Redis**: `redis:6379` (container name)
- **Memcached**: `memcached:11211` (container name)

### Environment Variables

Override default hosts using environment variables:

```bash
export REDIS_HOST="custom-redis-host"
export MEMCACHED_HOST="custom-memcached-host"
export DOCKER_ENV="true"  # Force Docker environment detection
```

## Test Configuration

### Cache Configuration

Tests use isolated configurations to avoid conflicts:

```yaml
cache:
  default: "memory-test"
  drivers:
    memory-test:
      driver: "memory"
    redis-test:
      driver: "redis"
      host: ${env:REDIS_HOST, "localhost"}
      port: 6379
      password: "turboscript_redis_pass"
      db: 1  # Use separate DB for tests
    memcached-test:
      driver: "memcached"
      host: ${env:MEMCACHED_HOST, "localhost"}
      port: 11211
```

### Connection Testing

Tests automatically verify service availability:

1. **Connection Test**: Attempts a simple set/get operation
2. **Graceful Skipping**: Skips tests if services are unavailable
3. **Error Handling**: Provides clear error messages for debugging

## Test Coverage

### Basic Operations

- **Set**: Store values with optional TTL
- **Get**: Retrieve stored values
- **Delete**: Remove stored values
- **Has**: Check key existence
- **Flush**: Clear all cached data

### Data Types

- **Primitives**: String, int, float, boolean
- **Collections**: Arrays, objects
- **Complex**: Nested structures with mixed types

### TTL (Time To Live)

- **Memory Driver**: In-memory expiration tracking
- **Redis Driver**: Native Redis TTL support
- **File/Memcached**: Basic TTL support

## Troubleshooting

### Common Issues

#### Redis Connection Failures

```sh
Error: dial tcp: lookup redis: no such host
```

**Solution**: Ensure Redis service is running and accessible:

```bash
# Check Docker services
docker-compose -f docker-compose.dev.yml ps

# Test Redis connection
redis-cli -h localhost -p 6379 ping
```

#### Memcached Connection Failures

```sh
Error: memcache: no servers configured or available
```

**Solution**: Verify Memcached service status:

```bash
# Check service status
telnet localhost 11211

# Test connection
echo "stats" | nc localhost 11211
```

#### Environment Detection Issues

**Problem**: Tests failing with wrong host configuration

**Solution**: Set environment variables explicitly:

```bash
export DOCKER_ENV="true"
export REDIS_HOST="localhost"
export MEMCACHED_HOST="localhost"
```

### Test Isolation

Each test run uses:

- **Unique Keys**: Prevents conflicts between parallel tests
- **Separate Databases**: Redis tests use DB 1, production uses DB 0
- **Cleanup**: Automatic cleanup after each test
- **Temporary Directories**: File driver uses temporary paths

## Best Practices

### Development

1. **Start Services**: Always start Docker services before testing
2. **Clean Environment**: Use fresh Redis/Memcached instances for testing
3. **Monitor Resources**: Cache tests can be resource-intensive

### CI/CD

1. **Service Dependencies**: Ensure cache services are available in CI
2. **Environment Variables**: Set appropriate host configurations
3. **Timeout Handling**: Allow sufficient time for service startup

### Production

1. **Separate Configuration**: Use different cache configuration for tests
2. **Resource Isolation**: Don't run tests against production cache
3. **Monitoring**: Monitor cache performance during test execution

## Performance Considerations

### Test Performance

- **Memory Driver**: Fastest, no network overhead
- **Redis Driver**: Network latency, serialization overhead
- **Memcached Driver**: Network latency, minimal serialization
- **File Driver**: Disk I/O limitations

### Optimization Tips

1. **Parallel Testing**: Run cache tests in parallel when possible
2. **Service Warmup**: Allow cache services to warm up before testing
3. **Connection Pooling**: Reuse connections across test runs
4. **Cleanup Strategy**: Efficient cleanup to avoid test pollution

## Related Documentation

- [Cache Configuration Guide](cache-configuration.md)
- [Performance Optimization](performance-optimization.md)
- [Docker Development Environment](docker-development.md)
