# Performance Optimization Guide

## Overview

This guide covers performance optimization techniques for TurboScript applications, focusing on database optimization, async operations, and system-level performance tuning.

## Database Performance

### Async Operations with turboQuery()

TurboScript's async `turboQuery()` function enables powerful performance optimizations through parallel execution:

```typescript
// ✅ Excellent: Parallel database operations
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // All queries execute simultaneously
        const [userData, userStats, recentOrders, preferences] = await Promise.all([
            turboQuery('SELECT * FROM users WHERE uid = $1', [userId]),
            turboQuery('SELECT COUNT(*) as total_orders, SUM(total) as total_spent FROM orders WHERE user_id = $1', [userId]),
            turboQuery('SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC LIMIT 5', [userId]),
            turboQuery('SELECT * FROM user_preferences WHERE user_id = $1', [userId])
        ]);

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    user: userData[0],
                    stats: userStats[0],
                    recent_orders: recentOrders,
                    preferences: preferences[0]
                }
            }
        };
    } catch (error) {
        // Error handling
    }
};

// ❌ Poor: Sequential database operations
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Each query waits for the previous one - much slower!
        const userData = await turboQuery('SELECT * FROM users WHERE uid = $1', [userId]);
        const userStats = await turboQuery('SELECT COUNT(*) as total_orders FROM orders WHERE user_id = $1', [userId]);
        const recentOrders = await turboQuery('SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC LIMIT 5', [userId]);
        const preferences = await turboQuery('SELECT * FROM user_preferences WHERE user_id = $1', [userId]);

        // This takes 4x longer than the parallel version!
    } catch (error) {
        // Error handling
    }
};
```

### Query Optimization

#### 1. Use Proper Indexes

```sql
-- Essential indexes for performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_uid ON users(uid);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_created_at ON orders(created_at DESC);
CREATE INDEX idx_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON user_sessions(expires_at);

-- Composite indexes for complex queries
CREATE INDEX idx_orders_user_status ON orders(user_id, status);
CREATE INDEX idx_products_category_active ON products(category_id, active) WHERE active = true;
```

#### 2. Efficient Query Patterns

```typescript
// ✅ Good: Specific columns with LIMIT
const users = await turboQuery(
    'SELECT id, name, email, created_at FROM users WHERE active = $1 ORDER BY created_at DESC LIMIT $2',
    [true, 50]
);

// ❌ Bad: SELECT * without LIMIT
const users = await turboQuery('SELECT * FROM users WHERE active = $1', [true]);

// ✅ Good: Use indexes effectively
const user = await turboQuery('SELECT * FROM users WHERE email = $1', [email]);

// ❌ Bad: Functions on columns prevent index usage
const users = await turboQuery('SELECT * FROM users WHERE LOWER(email) = LOWER($1)', [email]);
```

#### 3. Batch Operations

```typescript
// ✅ Good: Single query with array parameter
const products = await turboQuery(
    'SELECT * FROM products WHERE id = ANY($1)',
    [[1, 2, 3, 4, 5]]
);

// ❌ Bad: Multiple individual queries
const products = [];
for (const id of [1, 2, 3, 4, 5]) {
    const product = await turboQuery('SELECT * FROM products WHERE id = $1', [id]);
    products.push(product[0]);
}
```

#### 4. Pagination Optimization

```typescript
// ✅ Good: Cursor-based pagination (for large datasets)
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { cursor, limit = '20' } = event.queryParameters;
        const limitNum = Math.min(100, parseInt(limit, 10));

        let query = 'SELECT id, name, email, created_at FROM users WHERE active = true';
        const params: any[] = [];

        if (cursor) {
            query += ' AND created_at < $1';
            params.push(new Date(cursor));
        }

        query += ' ORDER BY created_at DESC LIMIT $' + (params.length + 1);
        params.push(limitNum);

        const users = await turboQuery(query, params);

        const nextCursor = users.length === limitNum ?
            users[users.length - 1].created_at : null;

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    users,
                    pagination: {
                        next_cursor: nextCursor,
                        has_more: users.length === limitNum
                    }
                }
            }
        };
    } catch (error) {
        // Error handling
    }
};

// ✅ Good: Offset pagination (for smaller datasets)
const [users, totalCount] = await Promise.all([
    turboQuery(
        'SELECT id, name, email FROM users WHERE active = true ORDER BY created_at DESC LIMIT $1 OFFSET $2',
        [limit, offset]
    ),
    turboQuery('SELECT COUNT(*) as total FROM users WHERE active = true')
]);
```

### Connection Pool Optimization

Configure database connections in `turboscript.yml`:

```yaml
database:
  host: "localhost"
  port: 5432
  name: "turboscript"
  user: "postgres"
  password: "postgres"
  max_connections: 20        # Tune based on your workload
  connection_timeout: "5s"   # Connection establishment timeout
  idle_timeout: "10m"        # Close idle connections
  max_lifetime: "1h"         # Maximum connection lifetime
```

## Application Performance

### Async Patterns and Concurrency

#### Parallel Background Jobs

```typescript
// Execute multiple background jobs in parallel
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { orderId, userId, email } = event.body as OrderData;

        // Create order first
        const order = await turboQuery(
            'INSERT INTO orders (user_id, status, created_at) VALUES ($1, $2, NOW()) RETURNING id',
            [userId, 'pending']
        );

        // Dispatch multiple background jobs in parallel
        await Promise.all([
            turboJob('send-order-confirmation', { orderId: order[0].id, email }),
            turboJob('update-inventory', { orderId: order[0].id }),
            turboJob('process-payment', { orderId: order[0].id }),
            turboJob('send-admin-notification', { orderId: order[0].id })
        ]);

        return {
            code: 201,
            response: {
                status: "success",
                data: { order_id: order[0].id }
            }
        };
    } catch (error) {
        // Error handling
    }
};
```

#### Smart Data Fetching

```typescript
// Fetch only required data for the specific use case
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { view_type } = event.queryParameters;

        let userFields = 'id, name, email';
        let orderFields = 'id, total, status';

        // Adjust fields based on view type
        if (view_type === 'detailed') {
            userFields += ', phone, address, created_at, last_login';
            orderFields += ', created_at, updated_at, shipping_address';
        }

        const [users, orders] = await Promise.all([
            turboQuery(`SELECT ${userFields} FROM users WHERE active = true LIMIT 50`),
            turboQuery(`SELECT ${orderFields} FROM orders WHERE status = 'active' LIMIT 100`)
        ]);

        return {
            code: 200,
            response: {
                status: "success",
                data: { users, orders }
            }
        };
    } catch (error) {
        // Error handling
    }
};
```

### Caching Strategies

#### Application-Level Caching

```typescript
// Simple in-memory cache for frequently accessed data
const cache = new Map<string, { data: any; expires: number }>();

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const cacheKey = 'featured_products';
        const now = Date.now();

        // Check cache first
        const cached = cache.get(cacheKey);
        if (cached && cached.expires > now) {
            return {
                code: 200,
                response: {
                    status: "success",
                    data: cached.data,
                    cached: true
                }
            };
        }

        // Cache miss - fetch from database
        const products = await turboQuery(
            'SELECT id, name, price, image_url FROM products WHERE featured = true AND active = true ORDER BY priority DESC LIMIT 20'
        );

        // Cache for 5 minutes
        cache.set(cacheKey, {
            data: { products },
            expires: now + (5 * 60 * 1000)
        });

        return {
            code: 200,
            response: {
                status: "success",
                data: { products }
            }
        };
    } catch (error) {
        // Error handling
    }
};
```

#### Database Query Result Caching

```typescript
// Cache expensive query results
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { category_id } = event.pathParameters;
        const cacheKey = `category_stats_${category_id}`;

        // Try cache first (if you implement caching)
        let stats = getCachedData(cacheKey);

        if (!stats) {
            // Expensive aggregation query
            const [categoryStats] = await turboQuery(`
                SELECT
                    c.name as category_name,
                    COUNT(p.id) as product_count,
                    AVG(p.price) as avg_price,
                    MIN(p.price) as min_price,
                    MAX(p.price) as max_price,
                    SUM(oi.quantity) as total_sold
                FROM categories c
                LEFT JOIN products p ON p.category_id = c.id
                LEFT JOIN order_items oi ON oi.product_id = p.id
                WHERE c.id = $1 AND p.active = true
                GROUP BY c.id, c.name
            `, [category_id]);

            stats = categoryStats;
            setCachedData(cacheKey, stats, 300); // Cache for 5 minutes
        }

        return {
            code: 200,
            response: {
                status: "success",
                data: { stats }
            }
        };
    } catch (error) {
        // Error handling
    }
};
```

## System Performance

### Server Configuration

#### Optimize turboscript.yml Settings

```yaml
server:
  port: 7890
  timeout: "30s"              # Reasonable timeout
  max_body_size: "10MB"       # Limit request body size
  cors:
    enabled: true
    origins: ["https://yourapp.com"]

database:
  max_connections: 25         # Tune based on your workload
  connection_timeout: "5s"
  idle_timeout: "10m"
  max_lifetime: "1h"

jobs:
  max_workers: 10             # Scale with CPU cores
  queue_size: 2000            # Handle traffic spikes
  retry_attempts: 3
  retry_delay: "30s"

logging:
  level: "error"              # Reduce logging overhead in production
  format: "json"              # Faster than text format
```

#### Memory Management

```go
// Example Go optimizations in the runtime engine
func (s *Server) optimizeMemory() {
    // Set GC target percentage
    debug.SetGCPercent(50)  // More frequent GC for lower memory usage

    // Set memory limit (Go 1.19+)
    debug.SetMemoryLimit(1 << 30) // 1GB limit
}
```

### Monitoring and Profiling

#### Performance Monitoring

```typescript
// Add performance timing to critical endpoints
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const startTime = Date.now();

    try {
        // Your endpoint logic
        const result = await turboQuery('SELECT * FROM complex_view WHERE id = $1', [id]);

        const duration = Date.now() - startTime;

        // Log slow queries
        if (duration > 1000) {
            console.warn(`Slow query detected: ${duration}ms for endpoint ${event.path}`);
        }

        return {
            code: 200,
            response: {
                status: "success",
                data: result,
                meta: {
                    query_time_ms: duration
                }
            }
        };
    } catch (error) {
        // Error handling
    }
};
```

#### Database Query Analysis

```sql
-- Enable slow query logging in PostgreSQL
ALTER SYSTEM SET log_min_duration_statement = 1000; -- Log queries > 1s
SELECT pg_reload_conf();

-- Analyze query performance
EXPLAIN ANALYZE SELECT * FROM users WHERE email = 'user@example.com';

-- Find missing indexes
SELECT schemaname, tablename, attname, n_distinct, correlation
FROM pg_stats
WHERE schemaname = 'public'
AND n_distinct > 100
AND correlation < 0.1;
```

### Load Testing and Benchmarking

#### Basic Load Testing

```bash
# Install load testing tools
npm install -g autocannon

# Test basic endpoint
autocannon -c 50 -d 60 http://localhost:7890/

# Test specific endpoint with JSON payload
autocannon -c 50 -d 60 -m POST \
  -H "Content-Type: application/json" \
  -b '{"name":"test","email":"test@example.com"}' \
  http://localhost:7890/users

# Test authenticated endpoints
autocannon -c 50 -d 60 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:7890/api/protected
```

#### Database Load Testing

```sql
-- Generate test data for performance testing
INSERT INTO users (name, email, password, created_at)
SELECT
    'User ' || generate_series,
    'user' || generate_series || '@example.com',
    '$2b$12$hash_here',
    NOW() - (random() * interval '365 days')
FROM generate_series(1, 100000);

-- Create realistic order data
INSERT INTO orders (user_id, total, status, created_at)
SELECT
    (random() * 100000)::int + 1,
    (random() * 1000)::decimal(10,2),
    CASE (random() * 3)::int
        WHEN 0 THEN 'pending'
        WHEN 1 THEN 'completed'
        ELSE 'cancelled'
    END,
    NOW() - (random() * interval '365 days')
FROM generate_series(1, 500000);
```

## Production Optimization

### Deployment Performance

#### Docker Optimization

```dockerfile
# Multi-stage build for smaller image
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o turboscript .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/turboscript .
COPY --from=builder /app/dist ./dist
CMD ["./turboscript"]
```

#### Production Configuration

```yaml
# turboscript.prod.yml
server:
  port: 8080
  timeout: "30s"
  cors:
    enabled: true
    origins: ["https://yourapp.com"]

database:
  max_connections: 50
  connection_timeout: "10s"
  idle_timeout: "30m"
  max_lifetime: "2h"

jobs:
  max_workers: 20
  queue_size: 5000

logging:
  level: "warn"
  format: "json"
```

### CDN and Static Asset Optimization

```typescript
// Serve optimized assets
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { file_id } = event.pathParameters;

        // Add caching headers for static content
        return {
            code: 200,
            response: fileContent,
            type: 'binary',
            headers: {
                'Cache-Control': 'public, max-age=31536000', // 1 year
                'ETag': generateETag(fileContent),
                'Content-Type': detectContentType(file_id)
            }
        };
    } catch (error) {
        // Error handling
    }
};
```

## Performance Best Practices Summary

### Database

1. **Use async turboQuery() with Promise.all()** for parallel operations
2. **Create proper indexes** on frequently queried columns
3. **Use LIMIT clauses** to prevent large result sets
4. **Implement cursor-based pagination** for large datasets
5. **Use batch operations** instead of loops
6. **Monitor slow queries** and optimize them

### Application

1. **Parallel background jobs** with Promise.all()
2. **Smart data fetching** - only request needed fields
3. **Implement caching** for frequently accessed data
4. **Use appropriate HTTP status codes**
5. **Add performance monitoring** to critical endpoints

### System

1. **Tune database connection pools**
2. **Configure appropriate timeouts**
3. **Use production-optimized logging levels**
4. **Implement proper error handling**
5. **Monitor memory usage and GC performance**

### Monitoring

1. **Track query execution times**
2. **Monitor database connection usage**
3. **Set up alerts for slow endpoints**
4. **Use load testing to validate performance**
5. **Profile memory usage under load**

---

## Navigation

**Previous:** [← Best Practices](guides/best-practices.md)
**Next:** [Security Guide →](guides/security.md)

## Related Topics

- [Best Practices Guide](guides/best-practices.md)
- [Architecture Overview](guides/architecture.md)
- [Database Operations](api/database-operations.md)
- [Deployment Guide](guides/deployment.md)
