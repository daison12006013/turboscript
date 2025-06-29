# Authentication Basics

This guide covers implementing authentication in TurboScript using JWT tokens and the built-in authentication utilities.

## Overview

TurboScript provides built-in authentication utilities for:

- JWT token verification
- User context management
- Protected endpoint authorization
- Standardized error responses

## Authentication Utilities

Import the authentication utilities in your route files:

```typescript
import { verifyAuth, createAuthErrorResponse } from "../../utils/auth";
```

### JWT Token Verification

The `verifyAuth()` function validates JWT tokens from request headers:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    // Verify JWT token from Authorization header
    const userPayload = verifyAuth(event);

    if (!userPayload) {
        return createAuthErrorResponse("Access token required", event);
    }

    // User is authenticated, access user data
    const userId = userPayload.uid;
    const userEmail = userPayload.email;

    return {
        code: 200,
        response: {
            status: "success",
            data: { userId, userEmail }
        }
    };
};
```

### Authorization Headers

TurboScript supports standard Bearer token format:

```text
Authorization: Bearer <jwt-token>
```

The authentication is case-insensitive and handles both `authorization` and `Authorization` headers.

## JWT Token Structure

TurboScript expects JWT tokens with this payload structure:

```typescript
interface JWTPayload {
    uid: string;        // User unique identifier
    email: string;      // User email address
    name?: string;      // User display name (optional)
    role?: string;      // User role (optional)
    exp: number;        // Token expiration timestamp
    iat: number;        // Token issued at timestamp
}
```

## User Context Access

Access authenticated user context in your route handlers:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const user = event.body.__user;

    // Get user's own data
    const [userData, userPosts] = await Promise.all([
        turboQuery('SELECT * FROM users WHERE uid = $1', [user.uid]),
        turboQuery('SELECT * FROM posts WHERE user_id = $1 ORDER BY created_at DESC', [user.uid])
    ]);

    return {
        code: 200,
        response: {
            status: "success",
            data: {
                user: userData[0],
                posts: userPosts
            }
        }
    };
};
```

## Login Implementation

Create a login endpoint that generates JWT tokens:

```typescript
import { hashPassword, verifyPassword } from "../../utils/password";
import { generateJWT } from "../../utils/jwt";

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { email, password } = event.body;

    if (!email || !password) {
        return {
            code: 400,
            response: {
                status: "error",
                message: "Email and password are required"
            }
        };
    }

    try {
        // Get user from database
        const users = await turboQuery(
            'SELECT uid, email, name, password_hash, role FROM users WHERE email = $1',
            [email]
        );

        if (users.length === 0) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "Invalid credentials"
                }
            };
        }

        const user = users[0];

        // Verify password
        const isValidPassword = await verifyPassword(password, user.password_hash);

        if (!isValidPassword) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "Invalid credentials"
                }
            };
        }

        // Generate JWT token
        const token = generateJWT({
            uid: user.uid,
            email: user.email,
            name: user.name,
            role: user.role
        });

        return {
            code: 200,
            response: {
                status: "success",
                message: "Login successful",
                data: {
                    token: token,
                    user: {
                        uid: user.uid,
                        email: user.email,
                        name: user.name,
                        role: user.role
                    }
                }
            }
        };
    } catch (error) {
        console.error('Login error:', error);

        return {
            code: 500,
            response: {
                status: "error",
                message: "Login failed"
            }
        };
    }
};
```

## Registration Implementation

Create a registration endpoint with password hashing:

```typescript
import { hashPassword } from "../../utils/password";
import { generateJWT } from "../../utils/jwt";

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { email, password, name } = event.body;

    // Validate input
    if (!email || !password || !name) {
        return {
            code: 400,
            response: {
                status: "error",
                message: "Name, email, and password are required"
            }
        };
    }

    if (password.length < 8) {
        return {
            code: 400,
            response: {
                status: "error",
                message: "Password must be at least 8 characters long"
            }
        };
    }

    try {
        // Check if user already exists
        const existingUsers = await turboQuery(
            'SELECT uid FROM users WHERE email = $1',
            [email]
        );

        if (existingUsers.length > 0) {
            return {
                code: 409,
                response: {
                    status: "error",
                    message: "User with this email already exists"
                }
            };
        }

        // Hash password
        const passwordHash = await hashPassword(password);

        // Create user
        const newUsers = await turboQuery(
            'INSERT INTO users (email, name, password_hash, role, created_at) VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP) RETURNING uid, email, name, role',
            [email, name, passwordHash, 'user']
        );

        const newUser = newUsers[0];

        // Generate JWT token
        const token = generateJWT({
            uid: newUser.uid,
            email: newUser.email,
            name: newUser.name,
            role: newUser.role
        });

        return {
            code: 201,
            response: {
                status: "success",
                message: "Registration successful",
                data: {
                    token: token,
                    user: {
                        uid: newUser.uid,
                        email: newUser.email,
                        name: newUser.name,
                        role: newUser.role
                    }
                }
            }
        };
    } catch (error) {
        console.error('Registration error:', error);

        return {
            code: 500,
            response: {
                status: "error",
                message: "Registration failed"
            }
        };
    }
};
```

## Testing Authentication

Test your authentication with curl or HTTP client:

```bash
# Login to get token
curl -X POST http://localhost:7890/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'

# Use token for protected endpoint
curl -X GET http://localhost:7890/api/users/profile \
  -H "Authorization: Bearer <jwt-token>"
```

## Security Best Practices

1. **Always hash passwords** using the provided utilities
2. **Validate JWT tokens** on every protected request
3. **Use HTTPS** in production to protect tokens in transit
4. **Set appropriate token expiration times**
5. **Implement token refresh mechanisms** for long-lived sessions
6. **Log authentication failures** for security monitoring

## Error Response Format

Authentication errors use a standardized format:

```typescript
{
    code: 401,
    response: {
        status: "error",
        message: "Access token is required for this endpoint",
        timestamp: "2024-01-15T10:30:00Z",
        path: "/api/users/profile"
    }
}
```

## Next Steps

- Learn about [database operations](database-basics.md) for user management
- Explore [security guidelines](security.md) for advanced protection
- Check out [JWT utilities](../api/authentication.md) for token management

## Related Documentation

- [Authentication & Security API](../api/authentication.md)
- [Security Guidelines](security.md)
- [Route Handlers](../api/route-handlers.md)
