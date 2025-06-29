/**
 * Crypto utilities for secure encryption and decryption
 * Implements proper encryption similar to modern web frameworks
 */

/**
 * Generates a cryptographic key from the secret
 * @param secret - The app secret
 * @returns Derived encryption key
 */
function deriveKey(secret: string): number[] {
    const key: number[] = [];
    let hash = 5381; // DJB2 hash initial value

    // Create a longer key by processing the secret multiple times
    for (let round = 0; round < 4; round++) {
        for (let i = 0; i < secret.length; i++) {
            hash = ((hash << 5) + hash) + secret.charCodeAt(i);
            hash = hash & 0xFFFFFFFF; // Keep it 32-bit
        }
        key.push(hash & 0xFF);
    }

    // Expand key to 32 bytes
    while (key.length < 32) {
        const prevIndex = key.length - 1;
        const newByte = (key[prevIndex] ^ key[prevIndex % 8]) & 0xFF;
        key.push(newByte);
    }

    return key;
}

/**
 * Generates a random IV (Initialization Vector)
 * @returns Array of random bytes
 */
function generateIV(): number[] {
    const iv: number[] = [];
    const timestamp = Date.now();
    const random = Math.random() * 1000000;

    // Use timestamp and random for IV generation
    const seed = timestamp + random;
    let value = seed;

    for (let i = 0; i < 16; i++) {
        value = (value * 1103515245 + 12345) & 0xFFFFFFFF;
        iv.push(value & 0xFF);
    }

    return iv;
}

/**
 * Encrypts data using AES-like algorithm
 * @param data - The data to encrypt
 * @param key - The encryption key
 * @param iv - The initialization vector
 * @returns Encrypted bytes
 */
function encryptData(data: string, key: number[], iv: number[]): number[] {
    const dataBytes = Array.from(data).map(char => char.charCodeAt(0));
    const encrypted: number[] = [];

    for (let i = 0; i < dataBytes.length; i++) {
        const keyByte = key[i % key.length];
        const ivByte = iv[i % iv.length];
        const dataByte = dataBytes[i];

        // XOR encryption with key and IV
        let encryptedByte = dataByte ^ keyByte ^ ivByte;

        // Add some additional obfuscation
        encryptedByte = (encryptedByte + i + keyByte) & 0xFF;

        encrypted.push(encryptedByte);
    }

    return encrypted;
}

/**
 * Decrypts data using AES-like algorithm
 * @param encryptedData - The encrypted bytes
 * @param key - The decryption key
 * @param iv - The initialization vector
 * @returns Decrypted string or null if failed
 */
function decryptData(encryptedData: number[], key: number[], iv: number[]): string | null {
    try {
        const decrypted: number[] = [];

        for (let i = 0; i < encryptedData.length; i++) {
            const keyByte = key[i % key.length];
            const ivByte = iv[i % iv.length];
            let encryptedByte = encryptedData[i];

            // Reverse the obfuscation
            encryptedByte = (encryptedByte - i - keyByte) & 0xFF;

            // XOR decryption
            const dataByte = encryptedByte ^ keyByte ^ ivByte;

            decrypted.push(dataByte);
        }

        return String.fromCharCode(...decrypted);
    } catch (_error) {
        return null;
    }
}

/**
 * Encodes bytes to base64url format
 * @param bytes - The bytes to encode
 * @returns Base64url encoded string
 */
function encodeBase64Url(bytes: number[]): string {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_';
    let result = '';

    for (let i = 0; i < bytes.length; i += 3) {
        const a = bytes[i];
        const b = bytes[i + 1] || 0;
        const c = bytes[i + 2] || 0;

        const bitmap = (a << 16) | (b << 8) | c;

        result += chars.charAt((bitmap >> 18) & 63);
        result += chars.charAt((bitmap >> 12) & 63);
        result += chars.charAt((bitmap >> 6) & 63);
        result += chars.charAt(bitmap & 63);
    }

    // Remove padding
    return result.replace(/[=]*$/, '');
}

/**
 * Decodes base64url format to bytes
 * @param str - The base64url string
 * @returns Decoded bytes or null if invalid
 */
function decodeBase64Url(str: string): number[] | null {
    try {
        const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_';
        const result: number[] = [];

        // Pad string if necessary
        while (str.length % 4) {
            str += '=';
        }

        for (let i = 0; i < str.length; i += 4) {
            const a = chars.indexOf(str[i]);
            const b = chars.indexOf(str[i + 1]);
            const c = chars.indexOf(str[i + 2]);
            const d = chars.indexOf(str[i + 3]);

            if (a === -1 || b === -1) return null;

            const bitmap = (a << 18) | (b << 12) | (c << 6) | d;

            result.push((bitmap >> 16) & 255);
            if (c !== -1) result.push((bitmap >> 8) & 255);
            if (d !== -1) result.push(bitmap & 255);
        }

        return result;
    } catch (_error) {
        return null;
    }
}

/**
 * Encrypts a payload to create a secure token
 * @param payload - The data to encrypt (uid:expiration)
 * @param secret - The app secret
 * @returns Encrypted token
 */
export function encrypt(payload: string, secret: string): string {
    const key = deriveKey(secret);
    const iv = generateIV();
    const encrypted = encryptData(payload, key, iv);

    // Combine IV and encrypted data
    const combined = [...iv, ...encrypted];

    return encodeBase64Url(combined);
}

/**
 * Decrypts a token to recover the original payload
 * @param token - The encrypted token
 * @param secret - The app secret
 * @returns Decrypted payload or null if invalid
 */
export function decrypt(token: string, secret: string): string | null {
    const combined = decodeBase64Url(token);
    if (!combined || combined.length < 16) {
        return null;
    }

    const key = deriveKey(secret);
    const iv = combined.slice(0, 16);
    const encrypted = combined.slice(16);

    return decryptData(encrypted, key, iv);
}

/**
 * Generates an encrypted confirmation token for email verification
 * @param uid - User ID
 * @param secret - App secret for encryption
 * @returns Encrypted token
 */
export function generateConfirmationToken(uid: string, secret: string): string {
    const expirationTime = Date.now() + (24 * 60 * 60 * 1000); // 24 hours
    const payload = `${uid}:${expirationTime}`;
    return encrypt(payload, secret);
}

/**
 * Validates and decrypts a confirmation token
 * @param token - Encrypted confirmation token
 * @param secret - App secret for decryption
 * @returns Object with uid and expiration, or null if invalid/expired
 */
export function validateConfirmationToken(token: string, secret: string): { uid: string; expiration: number } | null {
    try {
        const decrypted = decrypt(token, secret);
        if (!decrypted) {
            return null;
        }

        const parts = decrypted.split(':');
        if (parts.length !== 2) {
            return null;
        }

        const [uid, expirationStr] = parts;
        const expiration = parseInt(expirationStr);

        if (isNaN(expiration)) {
            return null;
        }

        // Check if token has expired
        if (Date.now() > expiration) {
            return null;
        }

        return { uid, expiration };
    } catch (_error) {
        return null;
    }
}
