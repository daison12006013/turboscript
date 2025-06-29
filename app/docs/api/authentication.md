# Authentication

TurboScript provides a comprehensive JWT-based authentication system with built-in utilities for secure user authentication and authorization. This guide covers everything you need to implement authentication in your TurboScript application.

## Overview

TurboScript's authentication system includes:

- **JWT Token Management**: Access and refresh tokens
- **Password Security**: Bcrypt hashing with automatic rehashing
- **Cookie Support**: Secure JWT cookie management
- **User Context**: Automatic user data injection into request context

## Authentication Flow

1. **Login**: User provides credentials, receives JWT tokens
2. **Token Validation**: Every protected request validates the JWT token
3. **User Context**: Authenticated user data is attached to request context
4. **Token Refresh**: Refresh tokens extend session lifetime
5. **Logout**: Tokens are invalidated

## JWT Configuration

Configure JWT settings in your environment variables:

```bash
# .env file
JWT_ACCESS_SECRET=your-super-secret-access-key-here
JWT_REFRESH_SECRET=your-super-secret-refresh-key-here
```

**Important Security Notes:**

- Use strong, unique secrets for production
- Access tokens expire in 15 minutes
- Refresh tokens expire in 7 days
- Never store secrets in your code

## JWT Payload Structure

The JWT token contains user information in a structured format:

```typescript
interface JWTPayload {
    uid: string;           // User unique identifier
    email: string;         // User email address
    name: string;          // User display name
    iat: number;          // Issued at timestamp
    exp: number;          // Expiration timestamp
    type: 'access' | 'refresh';  // Token type
}
```

## Authentication Utilities

TurboScript provides built-in authentication utilities for protecting endpoints:

### Authentication Pattern

Authentication is now handled directly in the `handle()` function:

```typescript
import { verifyAuth, createAuthErrorResponse } from '@app/utils/auth';
import { meta } from '@app/utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authentication first
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Access token is required for this endpoint", event);
        }

        // Use authenticated user's data directly
        const userUid = userPayload.uid;
        const userEmail = userPayload.email;
        const userName = userPayload.name;

        console.log(`Request from user: ${userName} (${userEmail})`);

        // Your protected endpoint logic here
        return {
            code: 200,
            response: {
                status: "success",
                message: `Hello, ${userName}!`,
                data: {
                    user_id: userUid
                },
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

## Complete Authentication Implementation

### Login Endpoint

Create a login endpoint to authenticate users:

```typescript
// app/routes/auth/login.ts
import { verifyPassword } from '@app/utils/password';
import { generateAccessToken, generateRefreshToken } from '@app/utils/jwt';
import { meta } from '@app/utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { email, password } = event.body as {
            email?: string;
            password?: string;
        };

        // Validate input
        if (!email || !password) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Email and password are required",
                    ...meta(event),
                }
            };
        }

        // Find user in database
        const users = await turboQuery(
            'SELECT uid, email, name, password FROM users WHERE email = $1 AND active = true',
            [email]
        );

        if (!users || users.length === 0) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "Invalid email or password",
                    ...meta(event),
                }
            };
        }

        const user = users[0];

        // Verify password
        const isValidPassword = await verifyPassword(password, user.password);
        if (!isValidPassword) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "Invalid email or password",
                    ...meta(event),
                }
            };
        }

        // Generate tokens
        const accessToken = generateAccessToken({
            uid: user.uid,
            email: user.email,
            name: user.name
        });

        const refreshToken = generateRefreshToken({
            uid: user.uid,
            email: user.email,
            name: user.name
        });

        // Update last login and save refresh token
        await Promise.all([
            turboQuery(
                'UPDATE users SET last_login = NOW() WHERE uid = $1',
                [user.uid]
            ),
            turboQuery(
                'INSERT INTO user_sessions (uid, refresh_token, created_at, expires_at) VALUES ($1, $2, NOW(), NOW() + INTERVAL \'7 days\')',
                [user.uid, refreshToken]
            )
        ]);

        return {
            code: 200,
            response: {
                status: "success",
                message: "Login successful",
                data: {
                    user: {
                        uid: user.uid,
                        email: user.email,
                        name: user.name
                    },
                    access_token: accessToken,
                    refresh_token: refreshToken
                },
                ...meta(event),
            },
            cookies: [
                `access_token=${accessToken}; HttpOnly; Secure; SameSite=Strict; Max-Age=900`, // 15 minutes
                `refresh_token=${refreshToken}; HttpOnly; Secure; SameSite=Strict; Max-Age=604800` // 7 days
            ]
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "Login failed",
                errors: [error instanceof Error ? error.message : "Unknown error"],
                ...meta(event),
            }
        };
    }
};
```

### Protected Endpoint with Authentication

```typescript
// app/routes/users/profile.ts
import { verifyAuth, createAuthErrorResponse } from '@app/utils/auth';
import { meta } from '@app/utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authentication directly in handle function
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Access token is required for this endpoint", event);
        }

        // Use authenticated user's data directly
        const userUid = userPayload.uid;

        // Fetch user profile with parallel queries
        const [userProfile, userPreferences, recentActivity] = await Promise.all([
            turboQuery('SELECT uid, name, email, created_at, last_login FROM users WHERE uid = $1', [userUid]),
            turboQuery('SELECT * FROM user_preferences WHERE user_id = $1', [userUid]),
            turboQuery('SELECT * FROM user_activities WHERE user_id = $1 ORDER BY created_at DESC LIMIT 10', [userUid])
        ]);

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    profile: userProfile[0],
                    preferences: userPreferences[0] || {},
                    recent_activity: recentActivity
                },
                ...meta(event),
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "Failed to fetch profile",
                errors: [error instanceof Error ? error.message : "Unknown error"],
                ...meta(event),
            }
        };
    }
};
```

### Token Refresh Endpoint

```typescript
// app/routes/auth/refresh.ts
import { verifyRefreshToken, generateAccessToken } from '@app/utils/jwt';
import { meta } from '@app/utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { refresh_token } = event.body as { refresh_token?: string };

        if (!refresh_token) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Refresh token is required",
                    ...meta(event),
                }
            };
        }

        // Verify refresh token
        const payload = verifyRefreshToken(refresh_token);
        if (!payload) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "Invalid or expired refresh token",
                    ...meta(event),
                }
            };
        }

        // Check if refresh token exists in database
        const sessions = await turboQuery(
            'SELECT uid FROM user_sessions WHERE refresh_token = $1 AND expires_at > NOW()',
            [refresh_token]
        );

        if (!sessions || sessions.length === 0) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "Refresh token not found or expired",
                    ...meta(event),
                }
            };
        }

        // Generate new access token
        const newAccessToken = generateAccessToken({
            uid: payload.uid,
            email: payload.email,
            name: payload.name
        });

        return {
            code: 200,
            response: {
                status: "success",
                message: "Token refreshed successfully",
                data: {
                    access_token: newAccessToken
                },
                ...meta(event),
            },
            cookies: [
                `access_token=${newAccessToken}; HttpOnly; Secure; SameSite=Strict; Max-Age=900` // 15 minutes
            ]
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "Token refresh failed",
                errors: [error instanceof Error ? error.message : "Unknown error"],
                ...meta(event),
            }
        };
    }
};
```

## Authentication Endpoints

### Login Endpoint

Create a login endpoint to authenticate users:

```typescript
// app/routes/auth/login.ts
import { generateJWTCookies } from '@app/utils/cookies';
import { generateTokenPair } from '@app/utils/jwt';
import { meta } from '@app/utils/meta';
import { verifyPassword, needsRehash } from '@app/utils/password';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const input = event.body as {
            email?: string;
            password?: string;
        };

        // Validation
        if (!input.email || !input.password) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Validation failed",
                    errors: ["Email and password are required"],
                    ...meta(event),
                },
            };
        }

        // Fetch user from database
        const userResult = await turboQuery(
            'SELECT id, uid, name, email, password FROM users WHERE email = $1 LIMIT 1',
            [input.email]
        );

        if (userResult.length === 0) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Invalid credentials",
                    errors: ["Email or password is incorrect"],
                    ...meta(event),
                },
            };
        }

        const user = userResult[0] as User;

        // Verify password
        const isValidPassword = await verifyPassword(input.password, user.password);
        if (!isValidPassword) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Invalid credentials",
                    errors: ["Email or password is incorrect"],
                    ...meta(event),
                },
            };
        }

        // Check if password needs rehashing (security improvement)
        if (needsRehash(user.password)) {
            const { hashPassword } = await import('@app/utils/password');
            const newHash = await hashPassword(input.password);
            await turboQuery(
                'UPDATE users SET password = $1 WHERE uid = $2',
                [newHash, user.uid]
            );
        }

        // Generate JWT tokens
        const tokenPair = generateTokenPair(user, event);

        // Generate secure cookies
        const cookies = generateJWTCookies(tokenPair);

        // Update last login
        await turboQuery(
            'UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE uid = $1',
            [user.uid]
        );

        return {
            code: 200,
            response: {
                status: "success",
                message: "Login successful",
                data: {
                    user: {
                        uid: user.uid,
                        name: user.name,
                        email: user.email,
                    },
                    tokens: tokenPair,
                },
                ...meta(event),
            },
            cookies,
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "Authentication failed",
                errors: [error instanceof Error ? error.message : "Unknown error"],
                ...meta(event),
            },
        };
    }
};
```

### Token Refresh Endpoint

```typescript
// app/routes/auth/refresh.ts
import { verifyRefreshToken, generateTokenPair } from '@app/utils/jwt';
import { generateJWTCookies } from '@app/utils/cookies';
import { meta } from '@app/utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const input = event.body as {
            refreshToken?: string;
        };

        if (!input.refreshToken) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "Refresh token is required",
                    ...meta(event),
                },
            };
        }

        // Verify refresh token
        const payload = verifyRefreshToken(input.refreshToken, event);
        if (!payload) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "Invalid or expired refresh token",
                    ...meta(event),
                },
            };
        }

        // Fetch current user data
        const userResult = await turboQuery(
            'SELECT id, uid, name, email FROM users WHERE uid = $1 LIMIT 1',
            [payload.uid]
        );

        if (userResult.length === 0) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "User not found",
                    ...meta(event),
                },
            };
        }

        const user = userResult[0] as User;

        // Generate new token pair
        const tokenPair = generateTokenPair(user, event);
        const cookies = generateJWTCookies(tokenPair);

        return {
            code: 200,
            response: {
                status: "success",
                message: "Token refreshed successfully",
                data: {
                    user: {
                        uid: user.uid,
                        name: user.name,
                        email: user.email,
                    },
                    tokens: tokenPair,
                },
                ...meta(event),
            },
            cookies,
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "Token refresh failed",
                errors: [error instanceof Error ? error.message : "Unknown error"],
                ...meta(event),
            },
        };
    }
};
```

### Logout Endpoint

```typescript
// app/routes/auth/logout.ts
import { meta } from '@app/utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Clear JWT cookies
        const clearCookies = [
            "accessToken=; Path=/; HttpOnly; Secure; SameSite=Strict; Max-Age=0",
            "refreshToken=; Path=/; HttpOnly; Secure; SameSite=Strict; Max-Age=0"
        ];

        return {
            code: 200,
            response: {
                status: "success",
                message: "Logged out successfully",
                ...meta(event),
            },
            cookies: clearCookies,
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "Logout failed",
                errors: [error instanceof Error ? error.message : "Unknown error"],
                ...meta(event),
            },
        };
    }
};
```

## Protected Endpoints

### Basic Protected Endpoint

```typescript
// app/routes/profile/me.ts
import { verifyAuth, createAuthErrorResponse } from '@app/utils/auth';
import { meta } from '@app/utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authentication directly in handle function
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Authentication required", event);
        }

        // Use authenticated user's data directly
        const userUid = userPayload.uid;

        // Fetch full user profile
        const profileResult = await turboQuery(
            'SELECT uid, name, email, created_at, updated_at FROM users WHERE uid = $1',
            [userUid]
        );

        if (profileResult.length === 0) {
            return {
                code: 404,
                response: {
                    status: "error",
                    message: "User profile not found",
                    ...meta(event),
                }
            };
        }

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    profile: profileResult[0]
                },
                ...meta(event),
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to fetch profile",
                ...meta(event),
            }
        };
    }
};
```

### Role-Based Authorization

```typescript
// app/routes/admin/users.ts
import { verifyAuth, createAuthErrorResponse } from '@app/utils/auth';
import { meta } from '@app/utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authentication first
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Authentication required", event);
        }

        // Check user role in database
        const userRoleResult = await turboQuery(
            'SELECT role FROM users WHERE uid = $1',
            [userPayload.uid]
        );

        if (userRoleResult.length === 0 || userRoleResult[0].role !== 'admin') {
            return {
                code: 403,
                response: {
                    status: "error",
                    message: "Insufficient permissions. Admin access required.",
                    ...meta(event),
                }
            };
        }

        // Admin-only endpoint logic here
        const allUsers = await turboQuery(
            'SELECT uid, name, email, role, created_at FROM users ORDER BY created_at DESC'
        );

        return {
            code: 200,
            response: {
                status: "success",
                data: { users: allUsers },
                requestedBy: userPayload.email,
                ...meta(event),
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to fetch users",
                ...meta(event),
            }
        };
    }
};
```

## User Registration

```typescript
// app/routes/auth/register.ts
import { hashPassword } from '@app/utils/password';
import { generateTokenPair } from '@app/utils/jwt';
import { generateJWTCookies } from '@app/utils/cookies';
import { meta } from '@app/utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const input = event.body as {
            name?: string;
            email?: string;
            password?: string;
        };

        // Validation
        if (!input.name || !input.email || !input.password) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Validation failed",
                    errors: ["Name, email, and password are required"],
                    ...meta(event),
                },
            };
        }

        // Email format validation
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        if (!emailRegex.test(input.email)) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Invalid email format",
                    ...meta(event),
                },
            };
        }

        // Password strength validation
        if (input.password.length < 8) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Password must be at least 8 characters long",
                    ...meta(event),
                },
            };
        }

        // Check if user already exists
        const existingUserResult = await turboQuery(
            'SELECT uid FROM users WHERE email = $1',
            [input.email]
        );

        if (existingUserResult.length > 0) {
            return {
                code: 409,
                response: {
                    status: "error",
                    message: "User already exists",
                    errors: ["Email is already registered"],
                    ...meta(event),
                },
            };
        }

        // Hash password
        const hashedPassword = await hashPassword(input.password);

        // Generate unique user ID
        const uid = `user_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

        // Create user
        const userResult = await turboQuery(
            'INSERT INTO users (uid, name, email, password, created_at, updated_at) VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING uid, name, email',
            [uid, input.name, input.email, hashedPassword]
        );

        const newUser = userResult[0] as User;

        // Generate JWT tokens
        const tokenPair = generateTokenPair(newUser, event);
        const cookies = generateJWTCookies(tokenPair);

        return {
            code: 201,
            response: {
                status: "success",
                message: "User registered successfully",
                data: {
                    user: {
                        uid: newUser.uid,
                        name: newUser.name,
                        email: newUser.email,
                    },
                    tokens: tokenPair,
                },
                ...meta(event),
            },
            cookies,
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "Registration failed",
                errors: [error instanceof Error ? error.message : "Unknown error"],
                ...meta(event),
            },
        };
    }
};
```

## Password Management

### Password Hashing

TurboScript uses bcrypt for secure password hashing:

```typescript
import { hashPassword, verifyPassword, needsRehash } from '@app/utils/password';

// Hash a password
const hashedPassword = await hashPassword('user-password');

// Verify a password
const isValid = await verifyPassword('user-password', hashedPassword);

// Check if password needs rehashing (for security updates)
if (needsRehash(hashedPassword)) {
    const newHash = await hashPassword('user-password');
    // Update in database
}
```

### Password Reset

```typescript
// app/routes/auth/reset-password.ts
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const input = event.body as {
            email?: string;
            resetToken?: string;
            newPassword?: string;
        };

        if (input.resetToken && input.newPassword) {
            // Verify reset token and update password
            const tokenResult = await turboQuery(
                'SELECT user_id FROM password_reset_tokens WHERE token = $1 AND expires_at > CURRENT_TIMESTAMP',
                [input.resetToken]
            );

            if (tokenResult.length === 0) {
                return {
                    code: 400,
                    response: {
                        status: "error",
                        message: "Invalid or expired reset token"
                    }
                };
            }

            const hashedPassword = await hashPassword(input.newPassword);

            await turboQuery(
                'UPDATE users SET password = $1 WHERE id = $2',
                [hashedPassword, tokenResult[0].user_id]
            );

            // Delete used token
            await turboQuery(
                'DELETE FROM password_reset_tokens WHERE token = $1',
                [input.resetToken]
            );

            return {
                code: 200,
                response: {
                    status: "success",
                    message: "Password updated successfully"
                }
            };
        } else if (input.email) {
            // Send password reset email
            const userResult = await turboQuery(
                'SELECT id, email FROM users WHERE email = $1',
                [input.email]
            );

            if (userResult.length === 0) {
                // Don't reveal if email exists
                return {
                    code: 200,
                    response: {
                        status: "success",
                        message: "If the email exists, a reset link has been sent"
                    }
                };
            }

            // Generate reset token
            const resetToken = Math.random().toString(36).substr(2, 32);
            const expiresAt = new Date(Date.now() + 60 * 60 * 1000); // 1 hour

            await turboQuery(
                'INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)',
                [userResult[0].id, resetToken, expiresAt]
            );

            // Send reset email (using turboEmail)
            await turboEmail({
                to: input.email,
                subject: 'Password Reset Request',
                content: `Click here to reset your password: ${event.env.APP_URL}/reset-password?token=${resetToken}`,
                driver: 'smtp'
            });

            return {
                code: 200,
                response: {
                    status: "success",
                    message: "If the email exists, a reset link has been sent"
                }
            };
        }

        return {
            code: 400,
            response: {
                status: "error",
                message: "Invalid request parameters"
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Password reset failed"
            }
        };
    }
};
```

## Security Best Practices

### Token Security

1. **Use HTTPS**: Always use HTTPS in production
2. **Secure Cookies**: Set `HttpOnly`, `Secure`, and `SameSite` flags
3. **Short Expiration**: Keep access tokens short-lived (15 minutes)
4. **Rotate Secrets**: Regularly rotate JWT secrets
5. **Token Blacklisting**: Implement token blacklisting for logout

### Input Validation

```typescript
const validateEmail = (email: string): boolean => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
};

const validatePassword = (password: string): string[] => {
    const errors = [];

    if (password.length < 8) {
        errors.push("Password must be at least 8 characters long");
    }

    if (!/[A-Z]/.test(password)) {
        errors.push("Password must contain at least one uppercase letter");
    }

    if (!/[a-z]/.test(password)) {
        errors.push("Password must contain at least one lowercase letter");
    }

    if (!/\d/.test(password)) {
        errors.push("Password must contain at least one number");
    }

    return errors;
};
```

### Rate Limiting

Implement rate limiting for authentication endpoints:

```typescript
// app/routes/auth/login.ts
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const clientIP = event.headers['x-forwarded-for'] || event.headers['x-real-ip'] || 'unknown';

        // Check rate limit
        const recentAttempts = await turboQuery(
            'SELECT COUNT(*) as count FROM login_attempts WHERE ip_address = $1 AND created_at > NOW() - INTERVAL \'15 minutes\'',
            [clientIP]
        );

        if (recentAttempts[0].count >= 5) {
            return {
                code: 429,
                response: {
                    status: "error",
                    message: "Too many login attempts. Please try again in 15 minutes."
                }
            };
        }

        // Continue with login logic...

        // Record login attempt
        await turboQuery(
            'INSERT INTO login_attempts (ip_address, email, success, created_at) VALUES ($1, $2, $3, CURRENT_TIMESTAMP)',
            [clientIP, input.email, loginSuccessful]
        );

    } catch (error) {
        // Handle error
    }
};
```

## Testing Authentication

### Unit Testing Examples

```typescript
// Test password utilities
import { hashPassword, verifyPassword, validatePassword } from '../utils/password';

describe('Password Utilities', () => {
    test('should hash and verify password correctly', async () => {
        const password = 'TestPassword123!';
        const hash = await hashPassword(password);

        expect(await verifyPassword(password, hash)).toBe(true);
        expect(await verifyPassword('wrongpassword', hash)).toBe(false);
    });

    test('should validate password requirements', () => {
        const validation = validatePassword('weak');
        expect(validation.isValid).toBe(false);
        expect(validation.errors).toContain('Password must be at least 8 characters long');
    });
});
```

### Integration Testing

```bash
# Test login endpoint
curl -X POST http://localhost:7890/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password123"}'

# Test protected endpoint
curl -X GET http://localhost:7890/users/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# Test token refresh
curl -X POST http://localhost:7890/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "YOUR_REFRESH_TOKEN"}'
```

## Authentication Utilities Reference

### verifyAuth() Function

The `verifyAuth()` function handles token validation:

```typescript
function verifyAuth(event: Event): JWTPayload | null {
    // Checks Authorization header and cookies
    // Returns decoded JWT payload or null if invalid
}
```

**Token Sources (in order of preference):**

1. `Authorization: Bearer <token>` header
2. `authorization: Bearer <token>` header (case-insensitive)
3. `access_token` cookie

### createAuthErrorResponse() Function

Creates standardized authentication error responses:

```typescript
function createAuthErrorResponse(message: string, event: Event): TurboScriptResponse {
    return {
        code: 401,
        response: {
            status: "error",
            message: message,
            ...meta(event)
        }
    };
}
```

### JWT Utility Functions

```typescript
// Generate tokens
function generateAccessToken(payload: UserPayload): string;
function generateRefreshToken(payload: UserPayload): string;

// Verify tokens
function verifyAccessToken(token: string): JWTPayload | null;
function verifyRefreshToken(token: string): JWTPayload | null;
```

## Cookie Management

### Secure Cookie Settings

TurboScript sets secure cookies by default:

```typescript
// Secure cookie configuration
const secureCookieOptions = [
    'HttpOnly',           // Prevent XSS access
    'Secure',            // HTTPS only
    'SameSite=Strict',   // CSRF protection
    'Max-Age=900'        // 15 minutes for access token
];
```

## Security Best Practices

### Token Expiration

- **Access Tokens**: 15 minutes (short-lived for security)
- **Refresh Tokens**: 7 days (longer-lived for user experience)

### Session Management

```typescript
// Revoke all user sessions (logout everywhere)
await turboQuery(
    'DELETE FROM user_sessions WHERE uid = $1',
    [userUid]
);

// Revoke specific session
await turboQuery(
    'DELETE FROM user_sessions WHERE refresh_token = $1',
    [refreshToken]
);

// Clean up expired sessions
await turboQuery(
    'DELETE FROM user_sessions WHERE expires_at < NOW()'
);
```

### Rate Limiting (Implementation Example)

```typescript
// app/routes/auth/login.ts - Add rate limiting
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const clientIP = event.headers['x-forwarded-for'] || 'unknown';

    try {
        // Check recent failed attempts
        const recentAttempts = await turboQuery(
            'SELECT COUNT(*) as attempts FROM login_attempts WHERE ip_address = $1 AND created_at > NOW() - INTERVAL \'15 minutes\' AND success = false',
            [clientIP]
        );

        if (recentAttempts[0].attempts >= 5) {
            return {
                code: 429,
                response: {
                    status: "error",
                    message: "Too many failed login attempts. Please try again in 15 minutes.",
                    ...meta(event),
                }
            };
        }

        // ... login logic ...

        // Log successful attempt
        await turboQuery(
            'INSERT INTO login_attempts (ip_address, email, success, created_at) VALUES ($1, $2, $3, NOW())',
            [clientIP, email, true]
        );

    } catch (error) {
        // Log failed attempt
        await turboQuery(
            'INSERT INTO login_attempts (ip_address, email, success, created_at) VALUES ($1, $2, $3, NOW())',
            [clientIP, email || 'unknown', false]
        );

        // ... error handling ...
    }
};
```

## Troubleshooting

### Common Issues

**"Invalid or expired token" errors:**

- Check token expiration (access tokens expire in 15 minutes)
- Verify JWT secrets are set correctly
- Ensure token is being sent in correct header format

**Authentication not working:**

```typescript
// Debug token verification
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    console.log('Headers:', event.headers);
    console.log('Cookies:', event.headers.cookie);

    const userPayload = verifyAuth(event);
    console.log('User payload:', userPayload);

    // ... rest of authentication logic
};
```

**Password verification fails:**

- Ensure bcrypt is properly installed
- Check password hashing during registration
- Verify database stores full hash (not truncated)

### Debugging Tips

1. **Enable debug logging** in `turboscript.yml`:

```yaml
debug: true
```

2. **Check request headers**:

```bash
curl -v -H "Authorization: Bearer YOUR_TOKEN" http://localhost:7890/protected-endpoint
```

3. **Verify JWT payload**:

```typescript
// Add temporary logging
const userPayload = verifyAuth(event);
console.log('Decoded payload:', userPayload);
```

---

## Navigation

**Previous:** [← Database Operations](api/database-operations.md)
**Next:** [Background Jobs →](api/background-jobs.md)

## Related Topics

- [Route Handler API](api/route-handlers.md)
- [Database Operations](api/database-operations.md)
- [Security Guidelines](guides/security.md)
- [Best Practices](guides/best-practices.md)
