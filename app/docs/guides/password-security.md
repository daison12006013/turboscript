# Password Security with Argon2

TurboScript provides enterprise-grade password security using the Argon2 hashing algorithm, which is the winner of the Password Hashing Competition and recommended by security experts worldwide.

## Why Argon2?

Argon2 offers superior security compared to older algorithms:

- **Memory-hard**: Requires significant memory, making GPU/ASIC attacks expensive
- **Time-tunable**: Configurable time cost for future-proofing
- **Side-channel resistant**: Argon2id variant protects against timing and cache attacks
- **Industry standard**: Recommended by OWASP and used by major platforms

## Basic Usage

### Hashing Passwords

```typescript
import { hashPassword } from '@app/utils/password';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { password } = event.body as { password: string };

    try {
        // Hash the password with secure defaults
        const hashedPassword = await hashPassword(password);

        // Store in database
        await turboQuery(
            'INSERT INTO users (email, password_hash) VALUES ($1, $2)',
            [email, hashedPassword]
        );

        return {
            code: 201,
            response: {
                status: "success",
                message: "User created successfully"
            }
        };
    } catch (error) {
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

### Verifying Passwords

```typescript
import { verifyPassword } from '@app/utils/password';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { email, password } = event.body as { email: string; password: string };

    try {
        // Get user from database
        const users = await turboQuery(
            'SELECT password_hash FROM users WHERE email = $1',
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

        // Verify password
        const isValid = await verifyPassword(password, users[0].password_hash);

        if (!isValid) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "Invalid credentials"
                }
            };
        }

        // Generate JWT and return success
        return {
            code: 200,
            response: {
                status: "success",
                message: "Login successful"
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "Authentication failed"
            }
        };
    }
};
```

## Advanced Configuration

### Custom Argon2 Parameters

You can customize the security parameters based on your needs:

```typescript
import { hashPassword } from '@app/utils/password';

// High-security configuration for sensitive applications
const highSecurityHash = await hashPassword(password, {
    memoryCost: 131072,    // 128 MB memory
    timeCost: 6,          // 6 iterations
    parallelism: 8,       // 8 parallel threads
    hashLength: 64,       // 64-byte output
});

// Fast configuration for high-throughput applications
const fastHash = await hashPassword(password, {
    memoryCost: 32768,    // 32 MB memory
    timeCost: 2,          // 2 iterations
    parallelism: 2,       // 2 parallel threads
});
```

### Password Rehashing

Automatically upgrade password security when parameters improve:

```typescript
import { verifyPassword, needsRehash, hashPassword } from '@app/utils/password';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { email, password } = event.body as { email: string; password: string };

    const users = await turboQuery(
        'SELECT id, password_hash FROM users WHERE email = $1',
        [email]
    );

    if (users.length === 0 || !await verifyPassword(password, users[0].password_hash)) {
        return {
            code: 401,
            response: {
                status: "error",
                message: "Invalid credentials"
            }
        };
    }

    // Check if password needs rehashing with improved parameters
    if (needsRehash(users[0].password_hash)) {
        const newHash = await hashPassword(password);
        await turboQuery(
            'UPDATE users SET password_hash = $1 WHERE id = $2',
            [newHash, users[0].id]
        );
    }

    return {
        code: 200,
        response: {
            status: "success",
            message: "Login successful"
        }
    };
};
```

## Password Validation

### Basic Validation

```typescript
import { validatePassword } from '@app/utils/password';

const validation = validatePassword('MySecureP@ssw0rd!');

if (!validation.valid) {
    return {
        code: 422,
        response: {
            status: "error",
            message: "Password validation failed",
            errors: validation.errors
        }
    };
}

console.log(`Password strength: ${validation.strength}`); // weak, fair, good, or strong
```

### Advanced Validation

```typescript
import { validatePassword } from '@app/utils/password';

const validation = validatePassword(password, {
    minLength: 16,                    // Require 16+ characters
    maxLength: 128,                   // Limit to prevent DoS
    requireUppercase: true,           // Require A-Z
    requireLowercase: true,           // Require a-z
    requireNumbers: true,             // Require 0-9
    requireSymbols: true,             // Require special chars
    forbidCommonPasswords: true,      // Block common patterns
    forbidPersonalInfo: [             // Block personal information
        userEmail,
        userName,
        userBirthdate
    ]
});

if (!validation.valid) {
    return {
        code: 422,
        response: {
            status: "error",
            message: "Password does not meet security requirements",
            errors: validation.errors,
            strength: validation.strength
        }
    };
}
```

## Secure Password Generation

Generate cryptographically secure passwords for temporary access or password resets:

```typescript
import { generateSecurePassword } from '@app/utils/password';

// Generate a 16-character password with default settings
const tempPassword = generateSecurePassword();

// Generate a 20-character password without similar-looking characters
const userFriendlyPassword = generateSecurePassword(20, {
    excludeSimilar: true,    // Exclude 0, O, l, 1, etc.
    includeSymbols: false    // Easier to type
});

// Generate a password for API keys (symbols allowed)
const apiKeyPassword = generateSecurePassword(32, {
    includeSymbols: true,
    excludeSimilar: false
});
```

## Security Best Practices

### 1. Always Use Async Functions

Password operations are computationally expensive and should never block:

```typescript
// ✅ Good - Non-blocking
const hash = await hashPassword(password);
const isValid = await verifyPassword(password, hash);

// ❌ Bad - Would block if synchronous version existed
// const hash = hashPasswordSync(password); // Don't do this
```

### 2. Handle Errors Gracefully

```typescript
try {
    const hash = await hashPassword(password);
    // Store hash...
} catch (error) {
    console.error('Password hashing failed:', error.message);
    return {
        code: 500,
        response: {
            status: "error",
            message: "Unable to process password"
        }
    };
}
```

### 3. Use Timing-Safe Comparisons

The library provides timing-safe verification to prevent timing attacks:

```typescript
import { timingSafeVerify } from '@app/utils/password';

// This is equivalent to verifyPassword but explicitly timing-safe
const isValid = await timingSafeVerify(password, storedHash);
```

### 4. Validate Before Hashing

Always validate password strength before expensive hashing:

```typescript
// Validate first (fast)
const validation = validatePassword(password);
if (!validation.valid) {
    return { code: 422, response: { errors: validation.errors } };
}

// Then hash (expensive)
const hash = await hashPassword(password);
```

### 5. Never Log Passwords

```typescript
// ❌ Never do this
console.log('User password:', password);
console.log('Password hash:', hash);

// ✅ Log validation results only
console.log('Password validation:', validation.valid);
console.log('Password strength:', validation.strength);
```

## Configuration Parameters

### Memory Cost

- **Default**: 65536 (64 MB)
- **Range**: 8-2^32
- **Effect**: Higher values increase memory usage and security

### Time Cost

- **Default**: 3 iterations
- **Range**: 1-2^32
- **Effect**: Higher values increase computation time and security

### Parallelism

- **Default**: 4 threads
- **Range**: 1-224
- **Effect**: Number of parallel threads, should match CPU cores

### Hash Length

- **Default**: 32 bytes
- **Range**: 4-2^32
- **Effect**: Length of the output hash

## Performance Considerations

### Benchmarking

Test different parameters for your hardware:

```typescript
const start = Date.now();
const hash = await hashPassword(password, {
    memoryCost: 65536,
    timeCost: 3,
    parallelism: 4
});
const duration = Date.now() - start;
console.log(`Hashing took ${duration}ms`);
```

### Recommended Settings

| Use Case | Memory (KB) | Time | Parallelism | Target Time |
|----------|-------------|------|-------------|-------------|
| High Security | 131072 | 6 | 8 | 2-5 seconds |
| Standard | 65536 | 3 | 4 | 0.5-2 seconds |
| High Throughput | 32768 | 2 | 2 | 0.1-0.5 seconds |

## Migration from Other Algorithms

### From bcrypt

```typescript
import { verifyPassword as verifyArgon2, hashPassword } from '@app/utils/password';
import bcrypt from 'bcryptjs';

const migratePassword = async (plainPassword: string, oldBcryptHash: string) => {
    // Verify with old algorithm
    const isValid = bcrypt.compareSync(plainPassword, oldBcryptHash);

    if (isValid) {
        // Generate new Argon2 hash
        const newHash = await hashPassword(plainPassword);

        // Update in database
        await turboQuery(
            'UPDATE users SET password_hash = $1, hash_algorithm = $2 WHERE id = $3',
            [newHash, 'argon2id', userId]
        );

        return newHash;
    }

    throw new Error('Password verification failed');
};
```

## Troubleshooting

### Common Issues

**Memory allocation errors**

```typescript
// Reduce memory cost if getting allocation errors
const hash = await hashPassword(password, {
    memoryCost: 32768  // Reduce from default 65536
});
```

**Performance too slow**

```typescript
// Reduce time cost for faster hashing
const hash = await hashPassword(password, {
    timeCost: 2  // Reduce from default 3
});
```

**Verification failures**

```typescript
// Ensure you're using the exact same algorithm
const isValid = await verifyPassword(password, hash);
if (!isValid) {
    console.log('Hash format:', hash.substring(0, 20));
    console.log('Password length:', password.length);
}
```

### Error Handling

```typescript
try {
    const hash = await hashPassword(password);
} catch (error) {
    if (error.message.includes('memory')) {
        // Reduce memory cost and retry
        return await hashPassword(password, { memoryCost: 32768 });
    }
    throw error;
}
```

## Security Considerations

1. **Never store plain text passwords**
2. **Always validate input before hashing**
3. **Use HTTPS for password transmission**
4. **Implement rate limiting for authentication**
5. **Consider two-factor authentication**
6. **Regularly update password parameters**
7. **Monitor for password breaches**

The Argon2 implementation in TurboScript provides enterprise-grade security with excellent performance characteristics, making it suitable for applications ranging from high-traffic web services to security-critical enterprise systems.
