// Example: Using the new argon2 module in TurboScript

// Hash a password with default options
const hash = await argon2.hash('my-secure-password');
console.log('Hashed password:', hash);

// Verify a password
const isValid = await argon2.verify(hash, 'my-secure-password');
console.log('Password is valid:', isValid);

// Hash with custom options
const customHash = await argon2.hash('another-password', {
    memoryCost: 131072,  // 128MB
    timeCost: 4,         // 4 iterations
    parallelism: 8,      // 8 threads
    hashLength: 64,      // 64 bytes
    saltLength: 32,      // 32 bytes salt
    variant: 'argon2id'  // Argon2id variant
});

// Use synchronous functions (blocking)
const syncHash = argon2.hashSync('sync-password');
const syncValid = argon2.verifySync(syncHash, 'sync-password');

// Access default options
console.log('Default options:', argon2.defaults);
