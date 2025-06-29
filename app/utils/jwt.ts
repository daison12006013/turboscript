/**
 * Simple JWT implementation for TurboScript that works with goja VM
 * This is a basic implementation that doesn't use external Node.js dependencies
 * Updated to fix btoa/atob and process.env issues
 */

/**
 * JWT configuration constants - loaded from environment variables
 * Fallback to default values only for development/testing
 */
const getJWTSecret = (event: Event, type: 'access' | 'refresh'): string => {
    const environmentKey = type === 'access' ? 'JWT_ACCESS_SECRET' : 'JWT_REFRESH_SECRET';
    return event.env[environmentKey];
}

const ACCESS_TOKEN_EXPIRES_IN = 15 * 60 * 1000; // 15 minutes in milliseconds
const REFRESH_TOKEN_EXPIRES_IN = 7 * 24 * 60 * 60 * 1000; // 7 days in milliseconds

/**
 * Interface for JWT token pair
 */
export interface TokenPair {
    accessToken: string;
    refreshToken: string;
    accessTokenExpiresAt: Date;
    refreshTokenExpiresAt: Date;
}

/**
 * Interface for JWT payload
 */
export interface JWTPayload {
    uid: string;
    email: string;
    name: string;
    type: 'access' | 'refresh';
    iat: number;
    exp: number;
}

/**
 * Simple base64 encoding for goja_nodejs environment
 */
const base64Encode = (strr: string): string => {
    // Simple character-based base64 encoding
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
    let result = '';
    let index = 0;

    while (index < strr.length) {
        const a = strr.charCodeAt(index++);
        const b = index < strr.length ? strr.charCodeAt(index++) : 0;
        const c = index < strr.length ? strr.charCodeAt(index++) : 0;

        const bitmap = (a << 16) | (b << 8) | c;

        result += chars.charAt((bitmap >> 18) & 63);
        result += chars.charAt((bitmap >> 12) & 63);
        result += index - 2 < strr.length ? chars.charAt((bitmap >> 6) & 63) : '=';
        result += index - 1 < strr.length ? chars.charAt(bitmap & 63) : '=';
    }

    return result;
}

/**
 * Simple base64 decoding for goja_nodejs environment
 */
const base64Decode = (strr: string): string => {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
    let result = '';
    let index = 0;

    // Remove padding
    strr = strr.replace(/=/g, '');

    while (index < strr.length) {
        const encoded1 = chars.indexOf(strr.charAt(index++));
        const encoded2 = chars.indexOf(strr.charAt(index++));
        const encoded3 = chars.indexOf(strr.charAt(index++));
        const encoded4 = chars.indexOf(strr.charAt(index++));

        const bitmap = (encoded1 << 18) | (encoded2 << 12) | (encoded3 << 6) | encoded4;

        result += String.fromCharCode((bitmap >> 16) & 255);
        if (encoded3 !== -1) result += String.fromCharCode((bitmap >> 8) & 255);
        if (encoded4 !== -1) result += String.fromCharCode(bitmap & 255);
    }

    return result;
}

/**
 * Simple base64url encoding
 */
const base64urlEscape = (strr: string): string => strr.replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')

/**
 * Simple base64url encoding from string
 */
const base64urlEncode = (strr: string): string => base64urlEscape(base64Encode(strr))

/**
 * Simple base64url decoding to string
 */
const base64urlDecode = (strr: string): string => {
    strr += '='.repeat((4 - strr.length % 4) % 4);
    strr = strr.replace(/-/g, '+').replace(/_/g, '/');
    return base64Decode(strr);
}

/**
 * Simple hash function (for demonstration - in production use proper crypto)
 * This is a basic implementation for the goja environment
 */
const simpleHash = (data: string, secret: string): string => {
    let hash = 0;
    const input = data + secret;
    for (let index = 0; index < input.length; index++) {
        const char = input.charCodeAt(index);
        hash = ((hash << 5) - hash) + char;
        hash = hash & hash; // Convert to 32bit integer
    }
    return Math.abs(hash).toString(36);
}

/**
 * Create a simple JWT token
 */
const createToken = (payload: unknown, secret: string): string => {
    const header = {
        typ: 'JWT',
        alg: 'HS256'
    };

    const encodedHeader = base64urlEncode(JSON.stringify(header));
    const encodedPayload = base64urlEncode(JSON.stringify(payload));
    const data = `${encodedHeader}.${encodedPayload}`;
    const signature = base64urlEncode(simpleHash(data, secret));

    return `${data}.${signature}`;
}

/**
 * Verify a simple JWT token
 */
const verifyToken = (token: string, secret: string): JWTPayload | null => {
    if (!secret) {
        return null;
    }

    try {
        const parts = token.split('.');
        if (parts.length !== 3) {
            return null;
        }

        const [encodedHeader, encodedPayload, signature] = parts;
        const data = `${encodedHeader}.${encodedPayload}`;
        const expectedSignature = base64urlEncode(simpleHash(data, secret));

        if (signature !== expectedSignature) {
            return null;
        }

        // Fix for null bytes in base64 decoding - remove trailing null characters
        const decodedPayloadString = base64urlDecode(encodedPayload).replace(/\0+$/, '');
        const payload = JSON.parse(decodedPayloadString) as JWTPayload;

        // Check expiration
        const MILLISECONDS_TO_SECONDS = 1000;
        if (payload.exp && Date.now() > payload.exp * MILLISECONDS_TO_SECONDS) {
            return null;
        }

        return payload;
    } catch (_) {
        return null;
    }
}

/**
 * Generate both access and refresh tokens for a user
 * @param user - User data containing uid, email, and name
 * @returns TokenPair with both tokens and their expiration dates
 */
export const generateTokenPair = (user: { uid: string; email: string; name: string }, event: Event): TokenPair => {
    const now = new Date();
    const nowTimestamp = Math.floor(now.getTime() / 1000);

    // Calculate expiration times
    const accessTokenExpiresAt = new Date(now.getTime() + ACCESS_TOKEN_EXPIRES_IN);
    const refreshTokenExpiresAt = new Date(now.getTime() + REFRESH_TOKEN_EXPIRES_IN);

    // Access token payload
    const accessPayload: JWTPayload = {
        uid: user.uid,
        email: user.email,
        name: user.name,
        type: 'access',
        iat: nowTimestamp,
        exp: Math.floor(accessTokenExpiresAt.getTime() / 1000)
    };

    // Refresh token payload
    const refreshPayload: JWTPayload = {
        uid: user.uid,
        email: user.email,
        name: user.name,
        type: 'refresh',
        iat: nowTimestamp,
        exp: Math.floor(refreshTokenExpiresAt.getTime() / 1000)
    };

    // Generate tokens
    const accessToken = createToken(accessPayload, getJWTSecret(event, 'access'));
    const refreshToken = createToken(refreshPayload, getJWTSecret(event, 'refresh'));

    return {
        accessToken,
        refreshToken,
        accessTokenExpiresAt,
        refreshTokenExpiresAt
    };
}

/**
 * Verify an access token
 * @param token - The JWT access token to verify
 * @param event - Event object containing environment variables
 * @returns Decoded payload if valid, null if invalid
 */
export const verifyAccessToken = (token: string, event: Event): JWTPayload | null => {
    const secret = getJWTSecret(event, 'access');

    if (!secret) {
        return null;
    }

    const decoded = verifyToken(token, secret);

    if (!decoded) {
        return null;
    }

    if (decoded.type !== 'access') {
        return null;
    }

    return decoded;
}

/**
 * Verify a refresh token
 * @param token - The JWT refresh token to verify
 * @param event - Event object containing environment variables
 * @returns Decoded payload if valid, null if invalid
 */
export const verifyRefreshToken = (token: string, event: Event): JWTPayload | null => {
    const decoded = verifyToken(token, getJWTSecret(event, 'refresh'));

    if (!decoded || decoded.type !== 'refresh') {
        return null;
    }

    return decoded;
}

/**
 * Refresh tokens using a refresh token (rotating refresh token pattern)
 * @param refreshToken - The refresh token
 * @param event - Event object containing environment variables
 * @returns New token pair if refresh token is valid, null otherwise
 */
export const refreshTokenPair = (refreshToken: string, event: Event): TokenPair | null => {
    const decoded = verifyRefreshToken(refreshToken, event);
    if (!decoded) {
        return null;
    }

    // Generate a new token pair (both access and refresh tokens)
    return generateTokenPair({
        uid: decoded.uid,
        email: decoded.email,
        name: decoded.name
    }, event);
}

/**
 * Refresh an access token using a refresh token (legacy method - use refreshTokenPair for better security)
 * @param refreshToken - The refresh token
 * @param event - Event object containing environment variables
 * @returns New access token if refresh token is valid, null otherwise
 */
export const refreshAccessToken = (refreshToken: string, event: Event): { accessToken: string; accessTokenExpiresAt: Date } | null => {
    const decoded = verifyRefreshToken(refreshToken, event);
    if (!decoded) {
        return null;
    }

    const now = new Date();
    const nowTimestamp = Math.floor(now.getTime() / 1000);
    const accessTokenExpiresAt = new Date(now.getTime() + ACCESS_TOKEN_EXPIRES_IN);

    const accessPayload: JWTPayload = {
        uid: decoded.uid,
        email: decoded.email,
        name: decoded.name,
        type: 'access',
        iat: nowTimestamp,
        exp: Math.floor(accessTokenExpiresAt.getTime() / 1000)
    };

    const accessToken = createToken(accessPayload, getJWTSecret(event, 'access'));

    return {
        accessToken,
        accessTokenExpiresAt
    };
}
