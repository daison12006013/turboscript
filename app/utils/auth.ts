/**
 * Authentication utilities for TurboScript
 */

import type { JWTPayload } from './jwt';
import { verifyAccessToken } from './jwt';

/**
 * Extract JWT token from Authorization header
 * @param authHeader - Authorization header value
 * @returns JWT token or null if invalid format
 */
export const extractToken = (authHeader: string): string | null => {
    if (!authHeader.startsWith('Bearer ')) {
        return null;
    }
    const BEARER_PREFIX_LENGTH = 7;
    return authHeader.substring(BEARER_PREFIX_LENGTH); // Remove 'Bearer ' prefix
};

/**
 * Verify user authentication from event headers
 * @param event - Event object containing headers and environment variables
 * @returns User payload if authenticated, null otherwise
 */
export const verifyAuth = (event: Event): JWTPayload | null => {
    // HTTP headers can be case-insensitive - check both lowercase and capitalized
    const authHeader = event.headers.authorization || event.headers.Authorization;

    if (!authHeader) {
        return null;
    }

    const token = extractToken(authHeader);
    if (!token) {
        return null;
    }

    return verifyAccessToken(token, event);
};

/**
 * Create authentication error response
 * @param message - Error message
 * @param event - Event object for metadata
 * @returns TurboScript error response
 */
export const createAuthErrorResponse = (message: string, event: Event): TurboScriptResponse => ({
    code: 401,
    response: {
        status: "error",
        message: "Authentication failed",
        errors: [message],
        meta: {
            timestamp: new Date().toISOString(),
            path: event.headers['x-original-url'] || 'unknown',
            method: event.headers['x-original-method'] || 'unknown'
        }
    }
});
