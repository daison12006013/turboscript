/**
 * Cookie configuration for secure JWT storage
 * Note: For goja environment compatibility
 */

export interface CookieOptions {
    httpOnly?: boolean;
    secure?: boolean;
    sameSite?: 'strict' | 'lax' | 'none';
    maxAge?: number; // in seconds
    domain?: string;
    path?: string;
}

/**
 * Determine if we're in production environment
 * Fallback for goja environment where process.env is not available
 */
const isProduction = (): boolean => {
    try {
        return typeof process !== 'undefined' && process.env.NODE_ENV === 'production';
    } catch {
        // Default to false for development in goja environment
        return false;
    }
};

/**
 * Generate a secure cookie string for setting in HTTP headers
 * @param name - Cookie name
 * @param value - Cookie value
 * @param options - Cookie options
 * @returns Cookie string for Set-Cookie header
 */
export const generateSecureCookie = (name: string, value: string, options: CookieOptions = {}): string => {
    const {
        httpOnly = true,
        secure = isProduction(),
        sameSite = 'lax',
        maxAge,
        domain,
        path = '/'
    } = options;

    let cookie = `${name}=${value}; Path=${path}`;

    if (httpOnly) {
        cookie += '; HttpOnly';
    }

    if (secure) {
        cookie += '; Secure';
    }

    cookie += `; SameSite=${sameSite}`;

    if (maxAge !== undefined) {
        cookie += `; Max-Age=${maxAge}`;
    }

    if (domain) {
        cookie += `; Domain=${domain}`;
    }

    return cookie;
};

/**
 * Generate cookies for JWT token pair
 * @param accessToken - Access token value
 * @param refreshToken - Refresh token value
 * @param accessTokenExpiresAt - Access token expiration date
 * @param refreshTokenExpiresAt - Refresh token expiration date
 * @returns Array of cookie strings for Set-Cookie headers
 */
export const generateJWTCookies = (
    accessToken: string,
    refreshToken: string,
    accessTokenExpiresAt: Date,
    refreshTokenExpiresAt: Date
): string[] => {
    const now = new Date();
    const MILLISECONDS_TO_SECONDS = 1000;

    const accessTokenMaxAge = Math.floor((accessTokenExpiresAt.getTime() - now.getTime()) / MILLISECONDS_TO_SECONDS);
    const refreshTokenMaxAge = Math.floor((refreshTokenExpiresAt.getTime() - now.getTime()) / MILLISECONDS_TO_SECONDS);

    const accessCookie = generateSecureCookie('turboscript_access_token', accessToken, {
        httpOnly: true,
        secure: isProduction(),
        sameSite: 'lax',
        maxAge: accessTokenMaxAge,
        path: '/'
    });

    const refreshCookie = generateSecureCookie('turboscript_refresh_token', refreshToken, {
        httpOnly: true,
        secure: isProduction(),
        sameSite: 'lax',
        maxAge: refreshTokenMaxAge,
        path: '/'
    });

    return [accessCookie, refreshCookie];
};

/**
 * Generate cookie string to clear/logout user tokens
 * @returns Array of cookie strings to clear both tokens
 */
export const generateLogoutCookies = (): string[] => {
    const clearAccessCookie = generateSecureCookie('turboscript_access_token', '', {
        httpOnly: true,
        secure: isProduction(),
        sameSite: 'lax',
        maxAge: 0,
        path: '/'
    });

    const clearRefreshCookie = generateSecureCookie('turboscript_refresh_token', '', {
        httpOnly: true,
        secure: isProduction(),
        sameSite: 'lax',
        maxAge: 0,
        path: '/'
    });

    return [clearAccessCookie, clearRefreshCookie];
};
