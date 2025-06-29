import { generateJWTCookies } from '@app/utils/cookies';
import { refreshTokenPair } from '@app/utils/jwt';
import { meta } from '@app/utils/meta';

// Helper function to extract token from cookie string
const extractTokenFromCookie = (cookieHeader: string | undefined, tokenName: string): string | null => {
    if (!cookieHeader) {
        return null;
    }

    const cookies = cookieHeader.split(';').map(cookie => cookie.trim());
    for (const cookie of cookies) {
        const [name, value] = cookie.split('=');
        if (name === tokenName) {
            return value;
        }
    }

    return null;
}

// New async handler pattern - no query() function needed
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Extract refresh token from cookie or body
        const refreshToken = (event.body.refresh_token as string) || extractTokenFromCookie(event.headers.cookie, 'turboscript_refresh_token');

        if (!refreshToken) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "Refresh token required",
                    errors: ["No refresh token provided"],
                    ...meta(event),
                }
            };
        }

        // Verify and refresh the token pair (both access and refresh tokens)
        const tokenPair = refreshTokenPair(refreshToken, event);

        if (!tokenPair) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "Invalid or expired refresh token",
                    errors: ["Please log in again"],
                    ...meta(event),
                }
            };
        }

        // Generate secure cookies for both tokens
        const cookies = generateJWTCookies(
            tokenPair.accessToken,
            tokenPair.refreshToken,
            tokenPair.accessTokenExpiresAt,
            tokenPair.refreshTokenExpiresAt
        );

        return {
            code: 200,
            response: {
                status: "success",
                message: "Token refreshed successfully",
                data: {
                    access_token: tokenPair.accessToken,
                    refresh_token: tokenPair.refreshToken,
                    access_token_expires_at: tokenPair.accessTokenExpiresAt.toISOString(),
                    refresh_token_expires_at: tokenPair.refreshTokenExpiresAt.toISOString(),
                },
                ...meta(event),
            },
            cookies
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "Token refresh failed",
                errors: [error instanceof Error ? error.message : "An unexpected error occurred"],
                ...meta(event),
            }
        };
    }
};
