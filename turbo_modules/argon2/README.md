# @turboscript/argon2

A comprehensive Argon2 password hashing library for the goja JavaScript VM in Go. This module provides secure password hashing using the Argon2 algorithm, compatible with Node.js argon2 library standards.

## Features

- ✅ **Argon2id, Argon2i, Argon2d** support
- ✅ **Async/Promise-based** and synchronous APIs
- ✅ **Node.js argon2 compatible** hash format
- ✅ **Configurable parameters** (memory, time, parallelism)
- ✅ **Secure defaults** following OWASP recommendations
- ✅ **Type-safe** with TypeScript definitions
- ✅ **Zero dependencies** (uses Go's crypto/rand and golang.org/x/crypto/argon2)

## Installation

For goja-based projects:

```bash
# Copy the turbo_modules/argon2 directory to your project
cp -r turbo_modules/argon2 your_project/turbo_modules/
```

## Usage

### Async API (Recommended)

```javascript
// Hash a password
const hash = await argon2.hash('password123', {
    memoryCost: 65536,  // 64MB
    timeCost: 3,        // 3 iterations
    parallelism: 4,     // 4 threads
    hashLength: 32,     // 32 bytes output
    saltLength: 16      // 16 bytes salt
});

// Verify a password
const isValid = await argon2.verify(hash, 'password123');
console.log(isValid); // true
```

### Synchronous API

```javascript
// Hash a password (blocks execution)
const hash = argon2.hashSync('password123');

// Verify a password (blocks execution)
const isValid = argon2.verifySync(hash, 'password123');
```

### Advanced Configuration

```javascript
const options = {
    memoryCost: 131072,    // 128MB memory usage
    timeCost: 4,           // 4 iterations
    parallelism: 8,        // 8 parallel threads
    hashLength: 64,        // 64 bytes hash output
    saltLength: 32,        // 32 bytes salt
    variant: 'argon2id'    // argon2id (default), argon2i, or argon2d
};

const hash = await argon2.hash('secret-password', options);
```

## API Reference

### `argon2.hash(password, options?)`

Asynchronously hash a password using Argon2.

**Parameters:**

- `password` (string): The password to hash
- `options` (object, optional): Hashing options

**Returns:** `Promise<string>` - The encoded hash string

### `argon2.verify(hash, password)`

Asynchronously verify a password against a hash.

**Parameters:**

- `hash` (string): The encoded hash string
- `password` (string): The password to verify

**Returns:** `Promise<boolean>` - True if password matches

### `argon2.hashSync(password, options?)`

Synchronously hash a password using Argon2.

**Parameters:**

- `password` (string): The password to hash
- `options` (object, optional): Hashing options

**Returns:** `string` - The encoded hash string

### `argon2.verifySync(hash, password)`

Synchronously verify a password against a hash.

**Parameters:**

- `hash` (string): The encoded hash string
- `password` (string): The password to verify

**Returns:** `boolean` - True if password matches

## Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `memoryCost` | number | 65536 | Memory usage in KB |
| `timeCost` | number | 3 | Number of iterations |
| `parallelism` | number | 4 | Degree of parallelism |
| `hashLength` | number | 32 | Length of hash output in bytes |
| `saltLength` | number | 16 | Length of salt in bytes |
| `variant` | string | 'argon2id' | Argon2 variant (argon2id, argon2i, argon2d) |

## Security Recommendations

The default parameters follow OWASP recommendations for Argon2:

- **Memory Cost:** 64MB (65536 KB) - Balance between security and performance
- **Time Cost:** 3 iterations - Sufficient for most applications
- **Parallelism:** 4 threads - Good for multi-core systems
- **Hash Length:** 32 bytes - Provides 256-bit security
- **Variant:** Argon2id - Best overall choice (hybrid of Argon2i and Argon2d)

For high-security applications, consider increasing `memoryCost` to 128MB+ and `timeCost` to 4+.

## Hash Format

The module produces hashes compatible with the Node.js argon2 library:

```
$argon2id$v=19$m=65536,t=3,p=4$base64Salt$base64Hash
```

## Error Handling

All functions throw descriptive errors for invalid inputs:

```javascript
try {
    const hash = await argon2.hash(''); // Empty password
} catch (error) {
    console.error(error.message); // "Password cannot be empty"
}

try {
    await argon2.verify('invalid-hash', 'password');
} catch (error) {
    console.error(error.message); // "Invalid hash format"
}
```

## Performance

- **Async functions** are non-blocking and suitable for web servers
- **Sync functions** block execution but may be faster for batch operations
- Hashing time scales with `memoryCost` and `timeCost` parameters
- Typical hash time: 50-200ms with default parameters

## Compatibility

- ✅ Compatible with Node.js `argon2` library hash format
- ✅ Works with goja JavaScript VM
- ✅ Cross-platform (Windows, macOS, Linux)
- ✅ Go 1.19+

## Contributing

This module is part of the TurboScript project. Contributions are welcome!

## License

MIT License - see LICENSE file for details.
