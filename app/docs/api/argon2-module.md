# Argon2 Module API Reference

The Argon2 module provides secure password hashing functionality using the Argon2id algorithm, which is the recommended standard for password hashing according to OWASP guidelines.

## Overview

The Argon2 module is available globally in all TurboScript routes and provides both asynchronous and synchronous APIs for password hashing and verification.

## Global Usage

```typescript
// Async password hashing
const hash = await argon2.hash('mypassword');
const isValid = await argon2.verify(hash, 'mypassword');

// Sync password hashing
const hash = argon2.hashSync('mypassword');
const isValid = argon2.verifySync(hash, 'mypassword');
```

## API Reference

### `argon2.hash(password, options?): Promise<string>`

Asynchronously hash a password using Argon2id.

**Parameters:**

- `password` (string): The password to hash
- `options` (object, optional): Hashing configuration

**Options:**

- `memoryCost` (number): Memory usage in KB (default: 65536 = 64MB)
- `timeCost` (number): Number of iterations (default: 3)
- `parallelism` (number): Degree of parallelism (default: 4)
- `hashLength` (number): Length of hash output in bytes (default: 32)
- `saltLength` (number): Length of salt in bytes (default: 16)
- `variant` (string): Argon2 variant - 'argon2id', 'argon2i', or 'argon2d' (default: 'argon2id')

**Returns:** Promise that resolves to the encoded hash string

**Example:**

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const password = event.body.password;

    // Use default settings (recommended)
    const hash = await argon2.hash(password);

    // Or customize settings
    const customHash = await argon2.hash(password, {
        memoryCost: 32768,  // 32MB
        timeCost: 4,        // 4 iterations
        parallelism: 2      // 2 threads
    });

    return { code: 200, response: { hash } };
};
```

### `argon2.verify(hash, password): Promise<boolean>`

Asynchronously verify a password against a hash.

**Parameters:**

- `hash` (string): The encoded hash string to verify against
- `password` (string): The password to verify

**Returns:** Promise that resolves to true if password matches

**Example:**

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { password, storedHash } = event.body;

    const isValid = await argon2.verify(storedHash, password);

    if (isValid) {
        return { code: 200, response: { message: "Login successful" } };
    } else {
        return { code: 401, response: { message: "Invalid credentials" } };
    }
};
```

### `argon2.hashSync(password, options?): string`

Synchronously hash a password using Argon2id.

**Parameters:** Same as `hash()` method

**Returns:** The encoded hash string

**Example:**

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const password = event.body.password;

    // Note: Sync operations block the event loop
    // Use async versions when possible
    const hash = argon2.hashSync(password);

    return { code: 200, response: { hash } };
};
```

### `argon2.verifySync(hash, password): boolean`

Synchronously verify a password against a hash.

**Parameters:** Same as `verify()` method

**Returns:** True if password matches

### `argon2.defaults`

Object containing the default Argon2 configuration values.

**Properties:**

```typescript
{
    memoryCost: 65536,    // 64MB
    timeCost: 3,          // 3 iterations
    parallelism: 4,       // 4 threads
    hashLength: 32,       // 32 bytes
    saltLength: 16,       // 16 bytes
    variant: "argon2id"   // Recommended variant
}
```

## Security Best Practices

### 1. Use Async Methods

Always prefer async methods (`hash()` and `verify()`) over sync methods to avoid blocking the event loop:

```typescript
// Good - Non-blocking
const hash = await argon2.hash(password);

// Avoid - Blocks event loop
const hash = argon2.hashSync(password);
```

### 2. Default Settings Are Secure

The default settings follow OWASP recommendations and provide excellent security:

```typescript
// These defaults are secure for most applications
const hash = await argon2.hash(password);
```

### 3. Adjust for Your Environment

Only customize settings if you have specific performance or security requirements:

```typescript
// High-security application (more memory/time)
const hash = await argon2.hash(password, {
    memoryCost: 131072,  // 128MB
    timeCost: 5,         // 5 iterations
    parallelism: 8       // 8 threads
});

// Low-resource environment (less memory/time)
const hash = await argon2.hash(password, {
    memoryCost: 32768,   // 32MB
    timeCost: 2,         // 2 iterations
    parallelism: 2       // 2 threads
});
```

### 4. Always Use Argon2id

The default variant (argon2id) provides the best security against both side-channel and GPU attacks:

```typescript
// Good - Uses argon2id by default
const hash = await argon2.hash(password);

// Only change if you have specific requirements
const hash = await argon2.hash(password, { variant: 'argon2i' });
```

## Hash Format

The module generates hashes in the standard Argon2 format compatible with other libraries:

```
$argon2id$v=19$m=65536,t=3,p=4$saltbase64$hashbase64
```

Where:

- `argon2id`: Variant
- `v=19`: Version
- `m=65536,t=3,p=4`: Memory cost, time cost, parallelism
- `saltbase64`: Base64-encoded salt
- `hashbase64`: Base64-encoded hash

## Error Handling

The module throws descriptive errors for common issues:

```typescript
try {
    const hash = await argon2.hash('');  // Empty password
} catch (error) {
    // Error: "password cannot be empty"
}

try {
    const isValid = await argon2.verify('invalid-hash', 'password');
} catch (error) {
    // Error: "invalid hash format"
}
```

## Performance Considerations

- **Memory Usage**: Higher `memoryCost` increases security but uses more RAM
- **Time Cost**: Higher `timeCost` increases security but takes longer
- **Parallelism**: Should match available CPU cores for optimal performance
- **Async vs Sync**: Always use async methods in production to avoid blocking

## Integration with TurboScript Utils

The argon2 module integrates seamlessly with TurboScript's password utilities:

```typescript
import { hashPassword, verifyPassword } from '../utils/password';

// These utilities use the argon2 module internally
const hash = await hashPassword('mypassword');
const isValid = await verifyPassword('mypassword', hash);
```

## Community Module

This argon2 module is designed to be reusable across the goja community. It can be installed in any goja-based project by copying the `turbo_modules/argon2/` directory.

For more information about creating and using goja modules, see the [Goja Modules Guide](../guides/goja-modules.md).
