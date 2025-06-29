# Best Practices

This guide outlines the best practices for developing with TurboScript, ensuring consistent, maintainable, and high-quality code.

## TypeScript Development

### Route Structure

- Use async pattern in route handlers
- Use structured error handling with try/catch blocks
- Validate all inputs before database operations

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const userId = event.body.__user?.uid;
        if (!userId) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "User ID is required"
                }
            };
        }

        // Your route logic here

    } catch (error) {
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

### Database Operations

- Use parallel queries with `Promise.all()` for better performance
- Always use parameterized queries to prevent SQL injection
- Handle database errors explicitly
- Use transactions for multiple related operations

```typescript
// Good: Parallel queries
const [user, preferences] = await Promise.all([
    turboQuery('SELECT * FROM users WHERE uid = $1', [userId]),
    turboQuery('SELECT * FROM preferences WHERE user_id = $1', [userId])
]);

// Bad: Sequential queries
const user = await turboQuery('SELECT * FROM users WHERE uid = $1', [userId]);
const preferences = await turboQuery('SELECT * FROM preferences WHERE user_id = $1', [userId]);
```

### Async/Await Patterns

Always use async functions with `turboQuery()` for optimal performance:

```typescript
// ✅ Recommended: Async handle with parallel database operations
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Execute multiple queries in parallel
        const [userData, userStats, recentActivity] = await Promise.all([
            turboQuery('SELECT * FROM users WHERE uid = $1', [event.pathParameters.uid]),
            turboQuery('SELECT COUNT(*) FROM user_sessions WHERE uid = $1', [event.pathParameters.uid]),
            turboQuery('SELECT * FROM activity_log WHERE uid = $1 ORDER BY created_at DESC LIMIT 5', [event.pathParameters.uid])
        ]);

        return {
            code: 200,
            response: {
                status: "success",
                data: { userData, userStats, recentActivity }
            }
        };
    } catch (error) {
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

### Error Handling

- Use structured error responses
- Include appropriate HTTP status codes
- Provide meaningful error messages
- Log errors for debugging
- Don't expose sensitive information in error messages

```typescript
// Good error handling pattern
try {
    const result = await turboQuery('SELECT * FROM users WHERE id = $1', [userId]);
    if (!result || result.length === 0) {
        return {
            code: 404,
            response: {
                status: "error",
                message: "User not found"
            }
        };
    }
    // Handle success case...
} catch (error) {
    return {
        code: 500,
        response: {
            status: "error",
            message: "Failed to fetch user data",
            // Don't expose internal error details to client
            ...(process.env.NODE_ENV === 'development' && {
                debug: error instanceof Error ? error.message : "Unknown error"
            })
        }
    };
}
```

### Input Validation

Always validate and sanitize inputs before processing:

```typescript
// Comprehensive input validation pattern
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const input = event.body as {
            name?: string;
            email?: string;
            age?: number;
        };

        // Required field validation
        const requiredFields = ['name', 'email'];
        const missingFields = requiredFields.filter(field => !input[field]);

        if (missingFields.length > 0) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Missing required fields",
                    errors: missingFields.map(field => `${field} is required`)
                }
            };
        }

        // Email format validation
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        if (!emailRegex.test(input.email)) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Invalid email format"
                }
            };
        }

        // Age validation (if provided)
        if (input.age !== undefined && (input.age < 13 || input.age > 120)) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Age must be between 13 and 120"
                }
            };
        }

        // Sanitize inputs
        const sanitizedData = {
            name: input.name.trim(),
            email: input.email.toLowerCase().trim(),
            age: input.age
        };

        // Proceed with database operations...
    } catch (error) {
        // Error handling...
    }
};
```

### Code Organization

- Group related routes in subdirectories
- Use shared utilities for common operations
- Keep route handlers focused and single-purpose
- Document complex business logic
- Use TypeScript types effectively

```typescript
// Example directory structure for routes
app/routes/
├── auth/
│   ├── login.ts
│   ├── logout.ts
│   ├── refresh.ts
│   └── register.ts
├── users/
│   ├── create.ts
│   ├── update.ts
│   ├── delete.ts
│   └── list.ts
├── orders/
│   ├── create.ts
│   ├── list.ts
│   └── status.ts
└── index.ts
```

### Response Standards

Use consistent response structures across your API:

```typescript
// Success response format
interface SuccessResponse {
    code: number;
    response: {
        status: "success";
        data: any;
        meta?: {
            timestamp: string;
            request_id?: string;
            pagination?: PaginationMeta;
        };
    };
}

// Error response format
interface ErrorResponse {
    code: number;
    response: {
        status: "error";
        message: string;
        errors?: string[];
        meta?: {
            timestamp: string;
            request_id?: string;
        };
    };
}
```

## Go Development

### Code Quality

- Run golangci-lint before committing: `./golangci-lint run`
- Fix all linting issues
- Follow Go naming conventions
- Write comprehensive tests
- Use interfaces for testability

```bash
# Run all quality checks before committing
./golangci-lint run
make test
make find-fail
```

### Go Error Handling

```go
// Good error handling
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
    result, err := s.processRequest(r)
    if err != nil {
        s.logger.Error("Failed to process request",
            "error", err,
            "path", r.URL.Path,
            "method", r.Method)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // Handle success case...
}
```

### Performance

- Use connection pooling for database connections
- Implement proper timeouts
- Use buffered channels for async operations
- Profile performance-critical code

### Testing

- Write unit tests for all public functions
- Use table-driven tests
- Mock external dependencies
- Test error conditions

```go
func TestUserValidation(t *testing.T) {
    tests := []struct {
        name     string
        input    User
        expected error
    }{
        {
            name:     "valid user",
            input:    User{Name: "John", Email: "john@example.com"},
            expected: nil,
        },
        {
            name:     "missing name",
            input:    User{Email: "john@example.com"},
            expected: ErrMissingName,
        },
        // Add more test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateUser(tt.input)
            if err != tt.expected {
                t.Errorf("expected %v, got %v", tt.expected, err)
            }
        })
    }
}
```

## Database Best Practices

### Query Optimization

- Use indexes for frequently queried columns
- Limit result sets with LIMIT clauses
- Use appropriate data types
- Avoid SELECT * in production code

```typescript
// Good: Specific columns with limit
const users = await turboQuery(
    'SELECT id, name, email, created_at FROM users WHERE active = $1 ORDER BY created_at DESC LIMIT $2',
    [true, 50]
);

// Bad: Select all columns without limit
const users = await turboQuery('SELECT * FROM users WHERE active = $1', [true]);
```

### Security

- Always use parameterized queries
- Validate input data types
- Implement proper access controls
- Use the `allowed_tables` configuration

```typescript
// Good: Parameterized query
const user = await turboQuery(
    'SELECT * FROM users WHERE email = $1 AND active = $2',
    [userEmail, true]
);

// Bad: String concatenation (SQL injection risk)
const user = await turboQuery(
    `SELECT * FROM users WHERE email = '${userEmail}' AND active = true`
);
```

### Transactions

For multiple related operations, consider using database transactions:

```typescript
// Example: Order creation with inventory update
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Begin transaction (if your database supports it)
        const orderResult = await turboQuery(
            'INSERT INTO orders (user_id, total, status) VALUES ($1, $2, $3) RETURNING id',
            [userId, totalAmount, 'pending']
        );

        const orderId = orderResult[0].id;

        // Update inventory for each item
        const inventoryUpdates = orderItems.map(item =>
            turboQuery(
                'UPDATE products SET stock = stock - $1 WHERE id = $2 AND stock >= $1',
                [item.quantity, item.product_id]
            )
        );

        await Promise.all(inventoryUpdates);

        return {
            code: 201,
            response: {
                status: "success",
                data: { order_id: orderId }
            }
        };
    } catch (error) {
        // Handle rollback if needed
        return {
            code: 500,
            response: {
                status: "error",
                message: "Failed to create order"
            }
        };
    }
};
```

## Security Best Practices

### JWT Tokens

- Use strong, random secrets
- Implement token refresh logic
- Set appropriate expiration times
- Store refresh tokens securely

### Password Security

- Use bcrypt for password hashing
- Implement password strength requirements
- Use salt for additional security
- Never log or expose passwords

```typescript
import { hashPassword, verifyPassword, validatePassword } from '../utils/password';

// Password validation with strength requirements
const passwordValidation = validatePassword(newPassword, {
    minLength: 8,
    requireUppercase: true,
    requireLowercase: true,
    requireNumbers: true,
    requireSymbols: false
});

if (!passwordValidation.valid) {
    return {
        code: 400,
        response: {
            status: "error",
            message: "Password validation failed",
            errors: passwordValidation.errors
        }
    };
}

// Hash password before storing
const hashedPassword = hashPassword(newPassword);
```

### CORS Configuration

Configure CORS appropriately for your deployment environment:

```yaml
# Development
server:
  cors:
    enabled: true
    origins: ["http://localhost:3000"]

# Production
server:
  cors:
    enabled: true
    origins: ["https://yourapp.com", "https://admin.yourapp.com"]
    methods: ["GET", "POST", "PUT", "DELETE"]
    headers: ["Content-Type", "Authorization"]
```

## Performance Optimization

### Database Performance

- Use connection pooling
- Implement query caching where appropriate
- Use database indexes
- Monitor slow queries

### Application Performance

- Use async operations with `Promise.all()`
- Implement proper error handling
- Use appropriate HTTP status codes
- Minimize memory allocations in Go code

### Monitoring

- Log important events and errors
- Monitor database connection usage
- Track response times
- Set up alerts for failures

## Development Workflow

### Code Reviews

- Review all TypeScript route logic
- Check for proper error handling
- Verify input validation
- Ensure security best practices

### Testing Strategy

- Write unit tests for route handlers
- Test authentication/authorization flows
- Verify database operations
- Test error conditions

### Deployment

- Use environment-specific configuration files
- Implement proper logging
- Set up monitoring and alerts
- Use HTTPS in production

---

## Navigation

**Previous:** [← API Examples](api/examples.md)
**Next:** [Performance Guide →](guides/performance.md)

## Related Topics

- [Security Guidelines](guides/security.md)
- [Development Workflow](guides/development.md)
- [API Examples](api/examples.md)
- [Architecture Overview](guides/architecture.md)
