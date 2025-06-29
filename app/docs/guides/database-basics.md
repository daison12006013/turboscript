# Database Basics

This guide covers the fundamentals of working with databases in TurboScript using the `turboQuery()` function.

## Database Configuration

Configure your database connection in `turboscript.yml`:

```yaml
database:
  host: ${env:DB_HOST, "localhost"}
  port: ${env:DB_PORT, 5432}
  user: ${env:DB_USER, "postgres"}
  password: ${env:DB_PASSWORD, ""}
  name: ${env:DB_NAME, "turboscript"}
  maxConnections: ${env:DB_MAX_CONNECTIONS, 10}
```

## Basic Query Operations

### Simple SELECT Query

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const users = await turboQuery('SELECT * FROM users');

    return {
        code: 200,
        response: {
            status: "success",
            data: users
        }
    };
};
```

### Parameterized Queries

Always use parameterized queries to prevent SQL injection:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const userId = event.pathParameters.id;

    // ✅ Secure - uses parameterized query
    const user = await turboQuery('SELECT * FROM users WHERE id = $1', [userId]);

    // ❌ Never do this - vulnerable to SQL injection
    // const user = await turboQuery(`SELECT * FROM users WHERE id = ${userId}`);

    if (user.length === 0) {
        return {
            code: 404,
            response: {
                status: "error",
                message: "User not found"
            }
        };
    }

    return {
        code: 200,
        response: {
            status: "success",
            data: user[0]
        }
    };
};
```

## CRUD Operations

### Creating Records (INSERT)

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { name, email, age } = event.body;

    // Validate input
    if (!name || !email) {
        return {
            code: 400,
            response: {
                status: "error",
                message: "Name and email are required"
            }
        };
    }

    try {
        const result = await turboQuery(
            'INSERT INTO users (name, email, age) VALUES ($1, $2, $3) RETURNING id, created_at',
            [name, email, age]
        );

        return {
            code: 201,
            response: {
                status: "success",
                message: "User created successfully",
                data: {
                    id: result[0].id,
                    name,
                    email,
                    age,
                    created_at: result[0].created_at
                }
            }
        };
    } catch (error) {
        console.error('Database error:', error);

        return {
            code: 500,
            response: {
                status: "error",
                message: "Failed to create user"
            }
        };
    }
};
```

### Updating Records (UPDATE)

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const userId = event.pathParameters.id;
    const { name, email, age } = event.body;

    try {
        const result = await turboQuery(
            'UPDATE users SET name = $1, email = $2, age = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4 RETURNING *',
            [name, email, age, userId]
        );

        if (result.length === 0) {
            return {
                code: 404,
                response: {
                    status: "error",
                    message: "User not found"
                }
            };
        }

        return {
            code: 200,
            response: {
                status: "success",
                message: "User updated successfully",
                data: result[0]
            }
        };
    } catch (error) {
        console.error('Database error:', error);

        return {
            code: 500,
            response: {
                status: "error",
                message: "Failed to update user"
            }
        };
    }
};
```

### Deleting Records (DELETE)

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const userId = event.pathParameters.id;

    try {
        const result = await turboQuery(
            'DELETE FROM users WHERE id = $1 RETURNING id',
            [userId]
        );

        if (result.length === 0) {
            return {
                code: 404,
                response: {
                    status: "error",
                    message: "User not found"
                }
            };
        }

        return {
            code: 200,
            response: {
                status: "success",
                message: "User deleted successfully"
            }
        };
    } catch (error) {
        console.error('Database error:', error);

        return {
            code: 500,
            response: {
                status: "error",
                message: "Failed to delete user"
            }
        };
    }
};
```

## Advanced Queries

### Complex WHERE Clauses

```typescript
// Multiple conditions
const users = await turboQuery(
    'SELECT * FROM users WHERE age >= $1 AND status = $2 ORDER BY created_at DESC',
    [18, 'active']
);

// IN clause
const userIds = [1, 2, 3, 4, 5];
const users = await turboQuery(
    'SELECT * FROM users WHERE id = ANY($1)',
    [userIds]
);

// LIKE search
const searchTerm = '%john%';
const users = await turboQuery(
    'SELECT * FROM users WHERE name ILIKE $1',
    [searchTerm]
);
```

### JOINs and Relationships

```typescript
// Inner join
const usersWithProfiles = await turboQuery(`
    SELECT u.id, u.name, u.email, p.bio, p.avatar_url
    FROM users u
    INNER JOIN profiles p ON u.id = p.user_id
    WHERE u.status = $1
`, ['active']);

// Left join with aggregation
const usersWithPostCounts = await turboQuery(`
    SELECT u.id, u.name, u.email, COUNT(p.id) as post_count
    FROM users u
    LEFT JOIN posts p ON u.id = p.user_id
    GROUP BY u.id, u.name, u.email
    ORDER BY post_count DESC
`);
```

### Transactions

For operations that need to be atomic, you can use multiple queries with proper error handling:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Begin transaction by ensuring both operations succeed or fail together
        const { userId, amount, recipientId } = event.body;

        // Check sender balance
        const sender = await turboQuery(
            'SELECT balance FROM accounts WHERE user_id = $1',
            [userId]
        );

        if (sender[0].balance < amount) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Insufficient balance"
                }
            };
        }

        // Update sender balance
        await turboQuery(
            'UPDATE accounts SET balance = balance - $1 WHERE user_id = $2',
            [amount, userId]
        );

        // Update recipient balance
        await turboQuery(
            'UPDATE accounts SET balance = balance + $1 WHERE user_id = $2',
            [amount, recipientId]
        );

        // Record transaction
        const transaction = await turboQuery(
            'INSERT INTO transactions (from_user_id, to_user_id, amount, created_at) VALUES ($1, $2, $3, CURRENT_TIMESTAMP) RETURNING id',
            [userId, recipientId, amount]
        );

        return {
            code: 200,
            response: {
                status: "success",
                message: "Transfer completed successfully",
                data: { transactionId: transaction[0].id }
            }
        };
    } catch (error) {
        console.error('Transaction error:', error);

        return {
            code: 500,
            response: {
                status: "error",
                message: "Transfer failed"
            }
        };
    }
};
```

## Parallel Database Operations

Use `Promise.all()` for concurrent database operations:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const userId = event.pathParameters.id;

    // Execute multiple queries in parallel
    const [user, posts, followers, following] = await Promise.all([
        turboQuery('SELECT * FROM users WHERE id = $1', [userId]),
        turboQuery('SELECT * FROM posts WHERE user_id = $1 ORDER BY created_at DESC LIMIT 10', [userId]),
        turboQuery('SELECT COUNT(*) as count FROM followers WHERE following_id = $1', [userId]),
        turboQuery('SELECT COUNT(*) as count FROM followers WHERE follower_id = $1', [userId])
    ]);

    if (user.length === 0) {
        return {
            code: 404,
            response: {
                status: "error",
                message: "User not found"
            }
        };
    }

    return {
        code: 200,
        response: {
            status: "success",
            data: {
                user: user[0],
                recent_posts: posts,
                followers_count: followers[0].count,
                following_count: following[0].count
            }
        }
    };
};
```

## Error Handling Best Practices

1. **Always use try-catch blocks** for database operations
2. **Log errors** for debugging purposes
3. **Return appropriate HTTP status codes**
4. **Don't expose internal error details** to clients

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const result = await turboQuery('SELECT * FROM users');

        return {
            code: 200,
            response: {
                status: "success",
                data: result
            }
        };
    } catch (error) {
        // Log the full error for debugging
        console.error('Database operation failed:', error);

        // Return generic error to client
        return {
            code: 500,
            response: {
                status: "error",
                message: "An internal error occurred"
            }
        };
    }
};
```

## Next Steps

- Learn about [authentication](authentication-basics.md) to secure your database operations
- Explore [advanced database operations](../api/database-operations.md) for complex scenarios
- Check out [performance optimization](performance.md) for database queries

## Related Documentation

- [Database Operations API](../api/database-operations.md)
- [Configuration Guide](configuration.md)
- [Security Guidelines](security.md)
