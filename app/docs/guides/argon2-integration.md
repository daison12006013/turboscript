# Argon2 Password Hashing Integration

## Overview

TurboScript provides complete Argon2 password hashing support through a dedicated goja module located in `internal/turbo_modules/argon2/`. This module implements the Node.js argon2 library interface, providing both async and synchronous password hashing functionality.

## Architecture

### Module Structure

```text
internal/turbo_modules/argon2/
├── index.go         # Main module implementation
├── index_test.go    # Comprehensive test suite
└── README.md        # Module documentation

turbo_modules/argon2/
├── index.d.ts       # TypeScript definitions
├── package.json     # Module metadata
└── example.js       # Usage examples
```

### Integration Flow

1. **Module Registration**: The argon2 module is registered in `eventloop_manager.go` during runtime initialization
2. **Event Loop Integration**: Uses proper event loop integration for async operations
3. **TypeScript Support**: Full TypeScript definitions with global `argon2` object
4. **Security Defaults**: Follows OWASP recommendations for Argon2 parameters

## Features

### Async Functions

- **`argon2.hash(password, options?)`** - Asynchronously hash passwords
- **`argon2.verify(hash, password)`** - Asynchronously verify passwords

### Sync Functions

- **`argon2.hashSync(password, options?)`** - Synchronously hash passwords
- **`argon2.verifySync(hash, password)`** - Synchronously verify passwords

### Configuration

- **`argon2.defaults`** - OWASP-recommended default parameters
- **Customizable Options**: Memory cost, time cost, parallelism, hash length, salt length
- **Variant Support**: Argon2i, Argon2id (Argon2d fallback to Argon2id)

## Usage Examples

### Basic Password Hashing

```typescript
// Async hashing (recommended for routes)
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { password } = event.body as { password: string };

    // Hash password with secure defaults
    const hash = await argon2.hash(password);

    // Store hash in database
    await turboQuery('INSERT INTO users (password_hash) VALUES ($1)', [hash]);

    return {
        code: 201,
        response: { status: "success", message: "User created" }
    };
};

// Verify password during login
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { email, password } = event.body as { email: string; password: string };

    const users = await turboQuery('SELECT password_hash FROM users WHERE email = $1', [email]);
    if (users.length === 0) {
        return { code: 401, response: { status: "error", message: "Invalid credentials" } };
    }

    const isValid = await argon2.verify(users[0].password_hash, password);
    if (!isValid) {
        return { code: 401, response: { status: "error", message: "Invalid credentials" } };
    }

    return { code: 200, response: { status: "success", message: "Login successful" } };
};
```

### Custom Configuration

```typescript
// Custom Argon2 parameters for high-security applications
const customOptions: Argon2Options = {
    memoryCost: 131072,    // 128MB (double default)
    timeCost: 5,           // 5 iterations (higher than default)
    parallelism: 8,        // 8 threads
    hashLength: 64,        // 64-byte hash
    variant: 'argon2id'    // Recommended variant
};

const hash = await argon2.hash(password, customOptions);
```

### Synchronous Operations

```typescript
// For non-route utility functions where blocking is acceptable
function hashPasswordSync(password: string): string {
    return argon2.hashSync(password, {
        memoryCost: 65536,
        timeCost: 3,
        parallelism: 4
    });
}

function verifyPasswordSync(hash: string, password: string): boolean {
    return argon2.verifySync(hash, password);
}
```

## Security Best Practices

### OWASP Recommendations

The module follows OWASP guidelines with these defaults:

```typescript
const defaults = {
    memoryCost: 65536,     // 64MB memory usage
    timeCost: 3,           // 3 iterations
    parallelism: 4,        // 4 threads
    hashLength: 32,        // 32-byte output
    saltLength: 16,        // 16-byte salt
    variant: 'argon2id'    // Recommended variant
};
```

### Production Recommendations

1. **Use Async Functions**: Always use `argon2.hash()` and `argon2.verify()` in routes to avoid blocking
2. **Monitor Performance**: Argon2 is CPU-intensive; monitor server performance
3. **Tune Parameters**: Adjust `memoryCost` and `timeCost` based on server capacity
4. **Use Argon2id**: Preferred variant for password hashing (default)

## Error Handling

```typescript
try {
    const hash = await argon2.hash(password);
    // Handle success
} catch (error) {
    // Handle hashing errors
    return {
        code: 500,
        response: {
            status: "error",
            message: "Password hashing failed"
        }
    };
}

try {
    const isValid = await argon2.verify(hash, password);
    if (!isValid) {
        // Invalid password
    }
} catch (error) {
    // Handle verification errors
    return {
        code: 500,
        response: {
            status: "error",
            message: "Password verification failed"
        }
    };
}
```

## Integration with Authentication

The argon2 module works seamlessly with TurboScript's authentication system:

```typescript
import { verifyAuth, createAuthErrorResponse } from "../../utils/auth";
import { meta } from "../../utils/meta";

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Verify authentication
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Access token required", event);
        }

        const { currentPassword, newPassword } = event.body as {
            currentPassword: string;
            newPassword: string;
        };

        // Get current password hash
        const users = await turboQuery(
            'SELECT password_hash FROM users WHERE uid = $1',
            [userPayload.uid]
        );

        // Verify current password
        const isValidCurrent = await argon2.verify(users[0].password_hash, currentPassword);
        if (!isValidCurrent) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Current password is incorrect",
                    ...meta(event)
                }
            };
        }

        // Hash new password
        const newHash = await argon2.hash(newPassword);

        // Update password
        await turboQuery(
            'UPDATE users SET password_hash = $1, updated_at = CURRENT_TIMESTAMP WHERE uid = $2',
            [newHash, userPayload.uid]
        );

        return {
            code: 200,
            response: {
                status: "success",
                message: "Password updated successfully",
                ...meta(event)
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Password update failed",
                ...meta(event)
            }
        };
    }
};
```

## Testing

The module includes comprehensive tests covering:

- Module registration and initialization
- Password hashing with various options
- Password verification (correct and incorrect passwords)
- Synchronous and asynchronous operations
- Error handling and edge cases

Run tests with:

```bash
go test -v ./internal/turbo_modules/argon2/
```

## Performance Considerations

1. **Memory Usage**: Default 64MB per hash operation
2. **CPU Impact**: Argon2 is CPU-intensive by design
3. **Concurrency**: Uses Go's crypto/argon2 package with proper goroutine handling
4. **Event Loop**: Async operations don't block the JavaScript event loop

## Migration Notes

If migrating from other password hashing libraries:

1. **From bcrypt**: Argon2 provides better security but higher resource usage
2. **Hash Format**: Argon2 hashes are incompatible with bcrypt; plan migration strategy
3. **Performance**: Benchmark on your hardware to tune parameters appropriately

## Implementation Details

### Module Registration

The argon2 module is registered in `internal/tsengine/eventloop_manager.go`:

```go
// Initialize Argon2 module for password hashing
argon2Module := argon2.New(vm, elm)
if injectErr := argon2Module.Register(); injectErr != nil {
    return fmt.Errorf("failed to register argon2 module: %w", injectErr)
}
```

### Event Loop Integration

The module implements proper event loop integration for async operations:

```go
type EventLoopRunner interface {
    RunOnLoop(fn func(*goja.Runtime)) bool
}
```

This ensures async operations complete correctly without blocking the JavaScript runtime.

### Hash Format

Generated hashes use the standard Argon2 encoded format:

```text
$argon2id$v=19$m=65536,t=3,p=4$saltBase64$hashBase64
```

This format is compatible with other Argon2 implementations and libraries.
