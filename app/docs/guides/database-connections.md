# Database Connections and turboQuery

TurboScript supports multiple database connections that can be configured in your `turboscript.yml` file and accessed via the `turboQuery` function.

## Configuration

Configure multiple database connections in your `turboscript.yml`:

```yaml
database:
  default: "main"
  connections:
    main:
      driver: "postgres"
      host: "${DB_HOST}"
      port: 5432
      username: "${DB_USER}"
      password: "${DB_PASSWORD}"
      database: "${DB_NAME}"
      ssl_mode: "disable"
    reader:
      driver: "postgres"
      host: "${DB_READER_HOST}"
      port: 5432
      username: "${DB_READER_USER}"
      password: "${DB_READER_PASSWORD}"
      database: "${DB_READER_NAME}"
      ssl_mode: "disable"
    analytics:
      driver: "postgres"
      host: "${ANALYTICS_DB_HOST}"
      port: 5432
      username: "${ANALYTICS_DB_USER}"
      password: "${ANALYTICS_DB_PASSWORD}"
      database: "${ANALYTICS_DB_NAME}"
      ssl_mode: "disable"
```

## Environment Variables

Set the corresponding environment variables for your connections:

```bash
# Main database (read/write)
DB_HOST=localhost
DB_USER=turboscript
DB_PASSWORD=secret
DB_NAME=turboscript_app

# Reader database (read-only replica)
DB_READER_HOST=replica.example.com
DB_READER_USER=readonly
DB_READER_PASSWORD=readonly_secret
DB_READER_NAME=turboscript_app

# Analytics database
ANALYTICS_DB_HOST=analytics.example.com
ANALYTICS_DB_USER=analytics
ANALYTICS_DB_PASSWORD=analytics_secret
ANALYTICS_DB_NAME=analytics_db
```

## Using turboQuery with Multiple Connections

TurboScript's `turboQuery` function supports two formats:

### Legacy Format (Default Connection)

The traditional format uses the default connection specified in your configuration:

```typescript
// Uses the default connection ("main" in the example above)
const users = await turboQuery('SELECT * FROM users WHERE active = $1', [true]);

// With parameters
const user = await turboQuery('SELECT * FROM users WHERE id = $1', [userId]);
```

### Object Format (Connection Switching)

The new object format allows you to specify which connection to use:

```typescript
// Basic object format with connection switching
const users = await turboQuery({
    query: 'SELECT * FROM users WHERE active = $1',
    bindings: [true],
    connection: 'reader'  // Use the "reader" connection
});

// Minimal format (uses default connection)
const count = await turboQuery({
    query: 'SELECT COUNT(*) FROM orders'
});

// Analytics queries on separate database
const stats = await turboQuery({
    query: 'SELECT date, count FROM daily_stats WHERE date >= $1',
    bindings: [startDate],
    connection: 'analytics'
});
```

## Real-World Examples

### Read/Write Splitting

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const userId = event.pathParameters.uid;

        // Read operations use reader connection for better performance
        const [userProfile, userPreferences, recentOrders] = await Promise.all([
            turboQuery({
                query: 'SELECT * FROM users WHERE id = $1',
                bindings: [userId],
                connection: 'reader'
            }),
            turboQuery({
                query: 'SELECT * FROM user_preferences WHERE user_id = $1',
                bindings: [userId],
                connection: 'reader'
            }),
            turboQuery({
                query: 'SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC LIMIT 5',
                bindings: [userId],
                connection: 'reader'
            })
        ]);

        // Write operations use main connection
        await turboQuery({
            query: 'UPDATE users SET last_seen = CURRENT_TIMESTAMP WHERE id = $1',
            bindings: [userId],
            connection: 'main'  // or omit connection to use default
        });

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    profile: userProfile[0],
                    preferences: userPreferences[0],
                    recent_orders: recentOrders
                }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Database operation failed"
            }
        };
    }
};
```

### Analytics and Reporting

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Get user data from main database
        const user = await turboQuery('SELECT * FROM users WHERE id = $1', [event.pathParameters.uid]);

        // Get analytics data from separate analytics database
        const [userAnalytics, performanceMetrics] = await Promise.all([
            turboQuery({
                query: 'SELECT * FROM user_analytics WHERE user_id = $1 AND date >= $2',
                bindings: [event.pathParameters.uid, '2025-01-01'],
                connection: 'analytics'
            }),
            turboQuery({
                query: 'SELECT avg_response_time, total_requests FROM performance_metrics WHERE date = CURRENT_DATE',
                connection: 'analytics'
            })
        ]);

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    user: user[0],
                    analytics: userAnalytics,
                    performance: performanceMetrics[0]
                }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to fetch analytics data"
            }
        };
    }
};
```

## Connection Options

### Object Format Parameters

- **`query`** (required): The SQL query string
- **`bindings`** (optional): Array of parameters for the query (equivalent to the second parameter in legacy format)
- **`connection`** (optional): Name of the database connection to use (defaults to the configured default connection)

### Connection Names

Connection names must match those defined in your `turboscript.yml` configuration. If a connection name is not found, turboQuery will throw an error.

## Performance Benefits

Using multiple connections can provide several performance benefits:

1. **Read Replicas**: Route read queries to read-only replicas to reduce load on your primary database
2. **Specialized Databases**: Use separate databases for analytics, logging, or other specialized workloads
3. **Geographic Distribution**: Connect to databases closer to your users or services
4. **Load Distribution**: Spread database load across multiple instances

## Error Handling

If a specified connection is not found or fails to connect, turboQuery will throw an error:

```typescript
try {
    const result = await turboQuery({
        query: 'SELECT * FROM users',
        connection: 'nonexistent'
    });
} catch (error) {
    // Handle connection error
    console.error('Database connection failed:', error.message);
}
```

## Backward Compatibility

The legacy `turboQuery(query, params)` format is fully supported and will continue to work with existing code. The object format is an enhancement that provides additional flexibility without breaking existing functionality.
