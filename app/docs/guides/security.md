# Security Guide

This guide covers security best practices for TurboScript applications, including authentication, database security, and production hardening.

## Authentication and Authorization

### JWT Token Security

TurboScript provides built-in JWT authentication with security best practices:

#### 1. Strong JWT Configuration

```bash
# Generate cryptographically secure secrets
openssl rand -base64 64  # For JWT_ACCESS_SECRET
openssl rand -base64 64  # For JWT_REFRESH_SECRET

# Add to environment variables
JWT_ACCESS_SECRET=your_super_secure_access_secret_minimum_64_characters_long
JWT_REFRESH_SECRET=your_super_secure_refresh_secret_also_minimum_64_characters_long
```

#### 2. Token Lifecycle Management

```typescript
// app/utils/jwt.ts - Built-in security features
const ACCESS_TOKEN_EXPIRES_IN = 15 * 60 * 1000; // 15 minutes (short for security)
const REFRESH_TOKEN_EXPIRES_IN = 7 * 24 * 60 * 60 * 1000; // 7 days

// Token pair generation with automatic expiration
export const generateTokenPair = (payload: UserPayload, event: Event): TokenPair => {
    const now = Date.now();

    const accessPayload = {
        ...payload,
        type: 'access' as const,
        iat: Math.floor(now / 1000),
        exp: Math.floor((now + ACCESS_TOKEN_EXPIRES_IN) / 1000)
    };

    const refreshPayload = {
        ...payload,
        type: 'refresh' as const,
        iat: Math.floor(now / 1000),
        exp: Math.floor((now + REFRESH_TOKEN_EXPIRES_IN) / 1000)
    };

    return {
        accessToken: createJWT(accessPayload, event, 'access'),
        refreshToken: createJWT(refreshPayload, event, 'refresh'),
        accessTokenExpiresAt: new Date(now + ACCESS_TOKEN_EXPIRES_IN),
        refreshTokenExpiresAt: new Date(now + REFRESH_TOKEN_EXPIRES_IN)
    };
};
```

#### 3. Secure Token Validation

```typescript
// app/utils/auth.ts - Multi-layer token validation
export const verifyAuth = (event: Event): JWTPayload | null => {
    // 1. Extract token from Authorization header
    const authHeader = event.headers.authorization || event.headers.Authorization;

    if (!authHeader) {
        return null;
    }

    const token = extractToken(authHeader);
    if (!token) {
        return null;
    }

    // 2. Verify token signature and expiration
    return verifyAccessToken(token, event);
};

// Built-in token verification with comprehensive checks
export const verifyAccessToken = (token: string, event: Event): JWTPayload | null => {
    try {
        const payload = verifyJWT(token, event, 'access');

        // 3. Validate token type
        if (payload.type !== 'access') {
            return null;
        }

        // 4. Check expiration
        const now = Math.floor(Date.now() / 1000);
        if (payload.exp <= now) {
            return null;
        }

        return payload;
    } catch (error) {
        return null;  // Never expose token validation errors
    }
};
```

#### 4. Secure Authentication Pattern

```typescript
// Standard authentication pattern for protected endpoints
import { verifyAuth, createAuthErrorResponse } from '@app/utils/auth';
import { meta } from '@app/utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authentication directly in handle function
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Access token is required for this endpoint", event);
        }

        // Additional role-based checks (optional)
        if (requiresAdminRole && userPayload.role !== 'admin') {
            return createAuthErrorResponse("Admin access required", event);
        }

        // Use authenticated user's data directly
        const userUid = userPayload.uid;

        // Your protected endpoint logic here
        return {
            code: 200,
            response: {
                status: "success",
                data: { user_id: userUid },
                ...meta(event),
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "An unexpected error occurred",
                ...meta(event),
            }
        };
    }
};
```

### Password Security

#### 1. Strong Password Hashing

```typescript
// app/utils/password.ts
import { hashSync, compareSync, getRounds } from 'bcryptjs';

const BCRYPT_COST = 12;  // High cost for security (minimum 12 in production)

export const hashPassword = (password: string): string => {
    return hashSync(password, BCRYPT_COST);
};

export const verifyPassword = (password: string, hash: string): boolean => {
    try {
        return compareSync(password, hash);
    } catch (error) {
        return false;  // Never expose hashing errors
    }
};

// Check if password needs rehashing (security maintenance)
export const needsRehash = (hash: string): boolean => {
    try {
        return getRounds(hash) < BCRYPT_COST;
    } catch (error) {
        return true;  // Rehash on error
    }
};
```

#### 2. Password Validation Rules

```typescript
// Comprehensive password validation
export interface PasswordRules {
    minLength: number;
    requireUppercase: boolean;
    requireLowercase: boolean;
    requireNumbers: boolean;
    requireSymbols: boolean;
}

export const validatePassword = (password: string, rules: PasswordRules): PasswordValidationResult => {
    const errors: string[] = [];

    if (password.length < rules.minLength) {
        errors.push(`Password must be at least ${rules.minLength} characters long`);
    }

    if (rules.requireUppercase && !/[A-Z]/.test(password)) {
        errors.push('Password must contain at least one uppercase letter');
    }

    if (rules.requireLowercase && !/[a-z]/.test(password)) {
        errors.push('Password must contain at least one lowercase letter');
    }

    if (rules.requireNumbers && !/\d/.test(password)) {
        errors.push('Password must contain at least one number');
    }

    if (rules.requireSymbols && !/[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\?]/.test(password)) {
        errors.push('Password must contain at least one special character');
    }

    return {
        valid: errors.length === 0,
        errors
    };
};
```

#### 3. Secure Password Change Implementation

```typescript
// app/routes/users/change-password.ts - Secure password change
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // 1. Verify current authentication
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Access token is required for this endpoint", event);
        }

        const input = event.body as {
            current_password?: string;
            new_password?: string;
            confirm_new_password?: string;
        };

        // 2. Validate input
        if (!input.current_password || !input.new_password) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Current password and new password are required"
                }
            };
        }

        if (input.new_password !== input.confirm_new_password) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Password confirmation does not match"
                }
            };
        }

        // 3. Validate new password strength
        const passwordValidation = validatePassword(input.new_password, {
            minLength: 8,
            requireUppercase: true,
            requireLowercase: true,
            requireNumbers: true,
            requireSymbols: false
        });

        if (!passwordValidation.valid) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Password validation failed",
                    errors: passwordValidation.errors
                }
            };
        }

        // 4. Verify current password
        const userResult = await turboQuery(
            'SELECT uid, password FROM users WHERE uid = $1 LIMIT 1',
            [userPayload.uid]
        );

        if (!userResult || userResult.length === 0) {
            return {
                code: 404,
                response: {
                    status: "error",
                    message: "User not found"
                }
            };
        }

        const user = userResult[0] as User;

        if (!verifyPassword(input.current_password, user.password)) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Current password is incorrect"
                }
            };
        }

        // 5. Hash new password and update
        const newHashedPassword = hashPassword(input.new_password);

        await turboQuery(
            'UPDATE users SET password = $1, updated_at = NOW() WHERE uid = $2',
            [newHashedPassword, userPayload.uid]
        );

        // 6. Optional: Revoke all sessions for security
        await turboQuery(
            'DELETE FROM user_sessions WHERE user_id = $1',
            [userPayload.uid]
        );

        return {
            code: 200,
            response: {
                status: "success",
                message: "Password changed successfully"
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "Password change failed"
            }
        };
    }
};
```

## Database Security

### Query Security and SQL Injection Prevention

TurboScript provides multiple layers of SQL injection protection:

#### 1. Parameterized Queries (Always Use)

```typescript
// ✅ SECURE: Always use parameterized queries
const users = await turboQuery(
    'SELECT * FROM users WHERE email = $1 AND status = $2',
    [userEmail, 'active']
);

const orders = await turboQuery(
    'SELECT * FROM orders WHERE user_id = $1 AND created_at > $2',
    [userId, new Date('2024-01-01')]
);

// ❌ DANGEROUS: Never use string concatenation
const users = await turboQuery(
    `SELECT * FROM users WHERE email = '${userEmail}'`  // SQL INJECTION RISK!
);
```

#### 2. Table Access Restrictions

```yaml
# turboscript.yml - Restrict database access to specific tables
database:
  allowed_tables:
    - users
    - orders
    - products
    - user_sessions
    - audit_logs
    # System tables are automatically blocked
```

#### 3. Dangerous Operation Prevention

```typescript
// TurboScript automatically blocks dangerous SQL operations:

// ❌ Blocked: DROP statements
await turboQuery('DROP TABLE users');  // Will be rejected

// ❌ Blocked: TRUNCATE statements
await turboQuery('TRUNCATE TABLE users');  // Will be rejected

// ❌ Blocked: DELETE without WHERE (mass deletion)
await turboQuery('DELETE FROM users');  // Will be rejected

// ✅ Allowed: Safe DELETE with WHERE clause
await turboQuery('DELETE FROM users WHERE uid = $1', [userId]);
```

#### 4. Input Validation and Sanitization

```typescript
// Comprehensive input validation pattern
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const input = event.body as {
            email?: string;
            name?: string;
            age?: number;
        };

        // 1. Required field validation
        if (!input.email || !input.name) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Email and name are required"
                }
            };
        }

        // 2. Format validation
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

        // 3. Length validation
        if (input.name.length > 255) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Name must be less than 255 characters"
                }
            };
        }

        // 4. Data sanitization
        const sanitizedEmail = input.email.toLowerCase().trim();
        const sanitizedName = input.name.trim();

        // 5. Range validation
        if (input.age !== undefined && (input.age < 0 || input.age > 150)) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Age must be between 0 and 150"
                }
            };
        }

        // 6. Safe database operation
        const result = await turboQuery(
            'INSERT INTO users (email, name, age) VALUES ($1, $2, $3) RETURNING uid',
            [sanitizedEmail, sanitizedName, input.age || null]
        );

        return {
            code: 201,
            response: {
                status: "success",
                data: { user_id: result[0].uid }
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "User creation failed"
            }
        };
    }
};
```

### UUID Security for User IDs

```typescript
// Use UUIDs for user identification security
const isValidUUIDv4 = (uuid: string): boolean => {
    const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
    return uuidRegex.test(uuid);
};

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { uid } = event.pathParameters;

        // Validate UUID format to prevent injection attempts
        if (!isValidUUIDv4(uid)) {
            return {
                code: 404,
                response: {
                    status: "not_found",
                    message: "User not found"
                }
            };
        }

        const users = await turboQuery('SELECT * FROM users WHERE uid = $1', [uid]);
        // Safe to proceed...
    } catch (error) {
        // Error handling
    }
};
```

## CORS and Request Security

### Production CORS Configuration

```yaml
# turboscript.yml - Production CORS settings
server:
  cors:
    enabled: true
    origins:
      - "https://yourapp.com"
      - "https://admin.yourapp.com"
    methods:
      - "GET"
      - "POST"
      - "PUT"
      - "DELETE"
    headers:
      - "Content-Type"
      - "Authorization"
    credentials: true
    max_age: 86400  # 24 hours
```

### Request Size Limits

```yaml
server:
  max_body_size: "10MB"    # Prevent large payload attacks
  timeout: "30s"           # Prevent slowloris attacks
```

## Secure Cookie Management

```typescript
// app/utils/cookies.ts - Secure cookie implementation
export const generateJWTCookies = (
    accessToken: string,
    refreshToken: string,
    accessExpires: Date,
    refreshExpires: Date
): string[] => {
    const cookies: string[] = [];

    // Access token cookie
    cookies.push(
        `turboscript_access_token=${accessToken}; ` +
        `HttpOnly; ` +                           // Prevent XSS access
        `Secure; ` +                            // HTTPS only
        `SameSite=Strict; ` +                   // CSRF protection
        `Expires=${accessExpires.toUTCString()}; ` +
        `Path=/`
    );

    // Refresh token cookie
    cookies.push(
        `turboscript_refresh_token=${refreshToken}; ` +
        `HttpOnly; ` +
        `Secure; ` +
        `SameSite=Strict; ` +
        `Expires=${refreshExpires.toUTCString()}; ` +
        `Path=/auth`                            // Limit path scope
    );

    return cookies;
};

// Secure logout - clear cookies
export const generateLogoutCookies = (): string[] => {
    return [
        `turboscript_access_token=; HttpOnly; Secure; SameSite=Strict; Expires=${new Date(0).toUTCString()}; Path=/`,
        `turboscript_refresh_token=; HttpOnly; Secure; SameSite=Strict; Expires=${new Date(0).toUTCString()}; Path=/auth`
    ];
};
```

## Production Security Hardening

### Environment Security

```bash
# Production environment variables
# Use strong, unique secrets
JWT_ACCESS_SECRET=$(openssl rand -base64 64)
JWT_REFRESH_SECRET=$(openssl rand -base64 64)

# Database security
DB_HOST=secure-db-host.com
DB_SSL=true
DB_SSL_MODE=require

# HTTPS enforcement
FORCE_HTTPS=true
HSTS_MAX_AGE=31536000

# Rate limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_WINDOW=900  # 15 minutes
RATE_LIMIT_MAX=100     # 100 requests per window
```

### Security Headers

```typescript
// Add security headers to all responses
export const addSecurityHeaders = (response: TurboScriptResponse): TurboScriptResponse => {
    return {
        ...response,
        headers: {
            ...response.headers,
            'X-Content-Type-Options': 'nosniff',
            'X-Frame-Options': 'DENY',
            'X-XSS-Protection': '1; mode=block',
            'Strict-Transport-Security': 'max-age=31536000; includeSubDomains',
            'Content-Security-Policy': "default-src 'self'",
            'Referrer-Policy': 'strict-origin-when-cross-origin'
        }
    };
};
```

### Rate Limiting Implementation

```typescript
// Simple rate limiting (extend as needed)
const rateLimitStore = new Map<string, { count: number; resetTime: number }>();

export const rateLimit = (event: Event, maxRequests: number = 100, windowMs: number = 900000): boolean => {
    const clientIP = event.headers['x-forwarded-for'] || event.headers['x-real-ip'] || 'unknown';
    const now = Date.now();
    const resetTime = now + windowMs;

    const current = rateLimitStore.get(clientIP);

    if (!current || current.resetTime < now) {
        // Reset window
        rateLimitStore.set(clientIP, { count: 1, resetTime });
        return true;
    }

    if (current.count >= maxRequests) {
        return false; // Rate limit exceeded
    }

    current.count++;
    return true;
};

// Use in route handlers
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    // Check rate limit
    if (!rateLimit(event, 50, 900000)) { // 50 requests per 15 minutes
        return {
            code: 429,
            response: {
                status: "error",
                message: "Too many requests"
            }
        };
    }

    // Continue with normal handler logic...
};
```

## Security Auditing and Monitoring

### Audit Logging

```typescript
// app/utils/audit.ts - Security event logging
export const logSecurityEvent = async (event: {
    action: string;
    userId?: string;
    ipAddress?: string;
    userAgent?: string;
    success: boolean;
    details?: Record<string, unknown>;
}): Promise<void> => {
    try {
        await turboQuery(
            `INSERT INTO security_audit_log
             (action, user_id, ip_address, user_agent, success, details, created_at)
             VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
            [
                event.action,
                event.userId || null,
                event.ipAddress || null,
                event.userAgent || null,
                event.success,
                JSON.stringify(event.details || {})
            ]
        );
    } catch (error) {
        console.error('Failed to log security event:', error);
    }
};

// Use in authentication endpoints
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { email, password } = event.body as LoginData;

        // Authentication logic...
        const success = await authenticateUser(email, password);

        // Log authentication attempt
        await logSecurityEvent({
            action: 'login_attempt',
            userId: success ? user.uid : undefined,
            ipAddress: event.headers['x-forwarded-for'],
            userAgent: event.headers['user-agent'],
            success: success,
            details: { email }
        });

        if (success) {
            return { /* success response */ };
        } else {
            return { /* failure response */ };
        }
    } catch (error) {
        // Error handling
    }
};
```

### Security Monitoring Queries

```sql
-- Monitor failed login attempts
SELECT
    ip_address,
    COUNT(*) as failed_attempts,
    MAX(created_at) as last_attempt
FROM security_audit_log
WHERE action = 'login_attempt'
    AND success = false
    AND created_at > NOW() - INTERVAL '1 hour'
GROUP BY ip_address
HAVING COUNT(*) > 5
ORDER BY failed_attempts DESC;

-- Monitor unusual access patterns
SELECT
    user_id,
    COUNT(DISTINCT ip_address) as unique_ips,
    COUNT(*) as total_requests
FROM security_audit_log
WHERE created_at > NOW() - INTERVAL '24 hours'
GROUP BY user_id
HAVING COUNT(DISTINCT ip_address) > 5
ORDER BY unique_ips DESC;
```

## Security Checklist

### Development

- [ ] Use parameterized queries for all database operations
- [ ] Implement proper input validation and sanitization
- [ ] Use strong JWT secrets (minimum 64 characters)
- [ ] Hash passwords with bcrypt cost ≥ 12
- [ ] Validate all user inputs before processing
- [ ] Implement proper error handling (don't expose internals)

### Authentication

- [ ] Use secure JWT token generation and validation
- [ ] Implement token refresh mechanism
- [ ] Use httpOnly, Secure, SameSite cookies
- [ ] Implement proper authorization patterns
- [ ] Log all authentication events
- [ ] Use UUIDs for user identification

### Database

- [ ] Configure allowed_tables restriction
- [ ] Use prepared statements (turboQuery handles this)
- [ ] Implement proper database access controls
- [ ] Regular security audits of database permissions
- [ ] Encrypt sensitive data at rest
- [ ] Use SSL for database connections in production

### Production

- [ ] Use HTTPS everywhere
- [ ] Configure secure CORS policies
- [ ] Implement rate limiting
- [ ] Add security headers to responses
- [ ] Monitor and log security events
- [ ] Regular security updates and patches
- [ ] Implement intrusion detection
- [ ] Regular penetration testing

---

## Navigation

**Previous:** [← Performance Guide](guides/performance.md)
**Next:** [Deployment Guide →](guides/deployment.md)

## Related Topics

- [Authentication Examples](api/authentication.md)
- [Best Practices Guide](guides/best-practices.md)
- [Development Workflow](guides/development.md)
- [Troubleshooting Guide](guides/troubleshooting.md)
