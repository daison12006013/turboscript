# Database Operations

TurboScript provides powerful async database operations through the `turboQuery()` function. This guide covers everything you need to know about working with databases in TurboScript, including advanced patterns, performance optimization, and security best practices.

## Overview

TurboScript uses PostgreSQL as its primary database and provides:

- **Async/Await Support**: Use `await turboQuery()` for non-blocking operations
- **Parallel Execution**: Use `Promise.all()` for concurrent database queries
- **Security Built-in**: SQL injection protection and table access control
- **Type Safety**: Fully typed with TypeScript definitions
- **Performance Optimized**: Connection pooling and query optimization
- **Error Handling**: Comprehensive error handling with detailed messages

## Core Features

### 🚀 Async Database Operations

TurboScript's `turboQuery()` function returns a Promise, enabling powerful async patterns:

```typescript
// Single async query
const users = await turboQuery('SELECT * FROM users WHERE active = $1', [true]);

// Parallel queries for maximum performance
const [orders, analytics, notifications] = await Promise.all([
    turboQuery('SELECT * FROM orders WHERE user_id = $1', [userId]),
    turboQuery('SELECT COUNT(*) as views FROM page_views WHERE user_id = $1', [userId]),
    turboQuery('SELECT * FROM notifications WHERE user_id = $1 AND read = false', [userId])
]);
```

### 🔒 Security Features

TurboScript includes built-in security measures:

- **Table Access Control**: Only whitelisted tables from `turboscript.yml` can be accessed
- **SQL Injection Protection**: Automatic query analysis prevents dangerous statements
- **Parameter Binding**: Parameterized queries for safe data insertion
- **Dangerous Operations**: DROP, DELETE, TRUNCATE operations are blocked by default

## Basic Usage

### Simple Query Example

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const users = await turboQuery('SELECT id, name, email FROM users WHERE active = $1', [true]);

        return {
            code: 200,
            response: {
                status: "success",
                data: { users },
                count: users.length
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

### Key Features

- **Security**: Built-in SQL injection protection and table restrictions
- **Type Safety**: TypeScript support for query results
- **Connection Pooling**: Efficient database connection management
- **Error Handling**: Comprehensive error reporting and handling

## Advanced Usage Patterns

### Parallel Execution with Promise.all()

Execute multiple queries simultaneously for maximum performance:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const userId = event.pathParameters.userId;

        // Execute multiple queries in parallel - MUCH faster than sequential
        const [user, orders, preferences, activities] = await Promise.all([
            turboQuery('SELECT * FROM users WHERE id = $1', [userId]),
            turboQuery('SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC LIMIT 10', [userId]),
            turboQuery('SELECT * FROM user_preferences WHERE user_id = $1', [userId]),
            turboQuery('SELECT * FROM user_activities WHERE user_id = $1 ORDER BY created_at DESC LIMIT 5', [userId])
        ]);

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    user: user[0],
                    recent_orders: orders,
                    preferences: preferences[0] || {},
                    recent_activities: activities
                }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to fetch user data"
            }
        };
    }
};
```

### Real-World Example: User Profile Update

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { userUid, name, email, preferences } = event.body;

    try {
        // Execute multiple operations in parallel for maximum performance
        const [
            userUpdate,
            preferencesUpdate,
            auditLog,
            cacheInvalidation
        ] = await Promise.all([
            // Update user information
            turboQuery(
                'UPDATE users SET name = $1, email = $2, updated_at = NOW() WHERE uid = $3 RETURNING *',
                [name, email, userUid]
            ),

            // Update user preferences
            turboQuery(
                'INSERT INTO user_preferences (user_id, preferences, updated_at) VALUES ($1, $2, NOW()) ON CONFLICT (user_id) DO UPDATE SET preferences = $2, updated_at = NOW()',
                [userUid, JSON.stringify(preferences)]
            ),

            // Log the action for audit
            turboQuery(
                'INSERT INTO audit_logs (user_id, action, details, created_at) VALUES ($1, $2, $3, NOW())',
                [userUid, 'profile_update', JSON.stringify({ name, email })]
            ),

            // Clear cached user data
            turboQuery(
                'DELETE FROM user_cache WHERE user_id = $1',
                [userUid]
            )
        ]);

        return {
            code: 200,
            response: {
                status: "success",
                message: "Profile updated successfully",
                data: {
                    user: userUpdate[0],
                    rowsAffected: {
                        user: userUpdate.length,
                        preferences: preferencesUpdate.length,
                        audit: auditLog.length
                    }
                }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to update profile"
            }
        };
    }
};
```

## Parameter Binding and Security

### Safe Parameter Usage

Always use parameterized queries to prevent SQL injection:

```typescript
// ✅ Good - Uses parameters
const users = await turboQuery(
    'SELECT * FROM users WHERE email = $1 AND active = $2 AND created_at > $3',
    [email, true, new Date('2025-01-01')]
);

// ❌ Bad - SQL injection risk
const users = await turboQuery(
    `SELECT * FROM users WHERE email = '${email}' AND active = true`
);
```

### Complex Parameter Examples

```typescript
// Multiple data types
const result = await turboQuery(
    'INSERT INTO orders (user_id, total, items, metadata, created_at) VALUES ($1, $2, $3, $4, $5) RETURNING *',
    [
        userId,                          // string/number
        total,                          // number
        JSON.stringify(items),          // JSON array as string
        JSON.stringify(metadata),       // JSON object as string
        new Date()                      // timestamp
    ]
);

// Array of values for IN clause
const userIds = [1, 2, 3, 4, 5];
const users = await turboQuery(
    'SELECT * FROM users WHERE id = ANY($1)',
    [userIds]
);
```

## Return Types and Error Handling

### Understanding Return Values

```typescript
interface TurboQueryResult {
    success: boolean;
    rowsAffected: number;
}

// SELECT queries return array of results
const users = await turboQuery('SELECT * FROM users');
console.log('Found users:', users.length);

// INSERT/UPDATE/DELETE return metadata
const result = await turboQuery('UPDATE users SET active = $1 WHERE id = $2', [false, userId]);
console.log('Rows affected:', result.rowsAffected);
```

### Comprehensive Error Handling

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { email, password } = event.body;

        // Validate input
        if (!email || !password) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Email and password are required"
                }
            };
        }

        // Query database
        const users = await turboQuery(
            'SELECT id, email, password_hash FROM users WHERE email = $1 AND active = true',
            [email]
        );

        if (!users || users.length === 0) {
            return {
                code: 404,
                response: {
                    status: "error",
                    message: "User not found"
                }
            };
        }

        // Process results...
        const user = users[0];

        return {
            code: 200,
            response: {
                status: "success",
                data: { user: { id: user.id, email: user.email } }
            }
        };

    } catch (error) {
        // Log error for debugging
        console.error('Database operation failed:', error);

        // Return user-friendly error
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

## Performance Optimization

### Indexing and Query Optimization

```sql
-- Add indexes for frequently queried columns
CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
CREATE INDEX CONCURRENTLY idx_orders_user_id_created ON orders(user_id, created_at);
CREATE INDEX CONCURRENTLY idx_activities_user_id_type ON user_activities(user_id, activity_type);
```

### Query Performance Best Practices

```typescript
// ✅ Good - Use LIMIT for large datasets
const recentOrders = await turboQuery(
    'SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50',
    [userId]
);

// ✅ Good - Use specific columns instead of SELECT *
const userEmails = await turboQuery(
    'SELECT id, email FROM users WHERE active = true',
    []
);

// ✅ Good - Use EXISTS for checking existence
const hasOrders = await turboQuery(
    'SELECT EXISTS(SELECT 1 FROM orders WHERE user_id = $1)',
    [userId]
);
```

### Connection Pooling

TurboScript automatically handles connection pooling for optimal performance:

```yaml
# turboscript.yml
database:
  max_connections: 25      # Maximum connections in pool
  max_idle: 10            # Maximum idle connections
  connection_lifetime: 5   # Connection lifetime in minutes
```

## Configuration and Security

### Table Access Control

Configure allowed tables in `turboscript.yml`:

```yaml
database:
  allowed_tables:
    - users
    - orders
    - user_preferences
    - audit_logs
    - user_activities
    - notifications

  # Block dangerous operations
  dangerous_operations:
    - DROP
    - TRUNCATE
    - DELETE   # Can be enabled for specific use cases
    - ALTER
```

### Environment Configuration

```yaml
# Development
database:
  debug: true              # Log all queries
  log_slow_queries: true   # Log queries > 100ms
  query_timeout: 30        # 30 second timeout

# Production
database:
  debug: false
  log_slow_queries: true
  query_timeout: 10        # Shorter timeout for production
```

## Advanced Patterns

### Transaction-like Operations

While TurboScript doesn't expose transactions directly, you can achieve transaction-like behavior with proper error handling:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { fromUserId, toUserId, amount } = event.body;

    try {
        // Step 1: Validate balances
        const [fromUser, toUser] = await Promise.all([
            turboQuery('SELECT balance FROM users WHERE id = $1', [fromUserId]),
            turboQuery('SELECT id FROM users WHERE id = $1', [toUserId])
        ]);

        if (!fromUser.length || !toUser.length) {
            return {
                code: 404,
                response: {
                    status: "error",
                    message: "One or both users not found"
                }
            };
        }

        if (fromUser[0].balance < amount) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Insufficient balance"
                }
            };
        }

        // Step 2: Execute transfer operations
        const [debitResult, creditResult, transactionLog] = await Promise.all([
            turboQuery(
                'UPDATE users SET balance = balance - $1 WHERE id = $2',
                [amount, fromUserId]
            ),
            turboQuery(
                'UPDATE users SET balance = balance + $1 WHERE id = $2',
                [amount, toUserId]
            ),
            turboQuery(
                'INSERT INTO transactions (from_user_id, to_user_id, amount, created_at) VALUES ($1, $2, $3, NOW()) RETURNING *',
                [fromUserId, toUserId, amount]
            )
        ]);

        return {
            code: 200,
            response: {
                status: "success",
                message: "Transfer completed successfully",
                data: {
                    transaction: transactionLog[0],
                    rowsAffected: {
                        debit: debitResult.rowsAffected,
                        credit: creditResult.rowsAffected
                    }
                }
            }
        };

    } catch (error) {
        // Log error for investigation
        console.error('Transfer failed:', {
            fromUserId,
            toUserId,
            amount,
            error: error.message
        });

        return {
            code: 500,
            response: {
                status: "error",
                message: "Transfer failed. Please try again."
            }
        };
    }
};
```

### Batch Operations

```typescript
// Batch insert with parallel execution
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { users } = event.body; // Array of user objects

    try {
        // Create all users in parallel
        const userInserts = users.map(user =>
            turboQuery(
                'INSERT INTO users (name, email, created_at) VALUES ($1, $2, NOW()) RETURNING *',
                [user.name, user.email]
            )
        );

        const results = await Promise.all(userInserts);

        return {
            code: 201,
            response: {
                status: "success",
                message: `Successfully created ${results.length} users`,
                data: {
                    users: results.map(result => result[0]),
                    created_count: results.length
                }
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Batch insert failed"
            }
        };
    }
};
```

## Troubleshooting

### Common Issues

**Query Timeout Errors:**

```typescript
// Increase timeout for long-running queries
// Configure in turboscript.yml:
database:
  query_timeout: 60  # 60 seconds for complex reports
```

**Connection Pool Exhaustion:**

```typescript
// Monitor connection usage and adjust pool size
database:
  max_connections: 50   # Increase if needed
  max_idle: 15         # Adjust based on load
```

**Table Access Denied:**

```yaml
# Add table to allowed list in turboscript.yml
database:
  allowed_tables:
    - your_new_table
```

### Debugging Queries

Enable query logging for development:

```yaml
# turboscript.yml
debug: true
database:
  debug: true
  log_slow_queries: true
```

View logs:

```bash
docker logs turboscript-app-dev-1 --tail=20
```

## Performance Metrics

TurboScript provides built-in performance monitoring:

### Response Time Benchmarks

Real-world performance from E2E tests:

| Operation Type | Response Time | Operations/sec |
|----------------|---------------|----------------|
| Simple SELECT | **1.55ms** | 783 ops/sec |
| JOIN queries | **3.2ms** | 312 ops/sec |
| Authenticated endpoints | **4.78ms** | 241 ops/sec |
| Parallel operations | **5.1ms** | 196 ops/sec |

*Metrics measured on Apple MacBook Pro M3*

### Memory Usage

- **Base Memory**: 12.1MB RAM
- **Per Connection**: ~0.5MB additional
- **Query Cache**: ~2MB for compiled queries

---

## Navigation

**Previous:** [← Route Handlers](api/route-handlers.md)
**Next:** [Authentication →](api/authentication.md)

## Related Topics

- [Route Handler API](api/route-handlers.md)
- [Authentication Guide](api/authentication.md)
- [Performance Optimization](guides/performance.md)
- [Security Guidelines](guides/security.md)
- [Best Practices](guides/best-practices.md)
