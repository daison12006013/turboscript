import { createAuthErrorResponse, verifyAuth } from '@app/utils/auth';
import { generateLogoutCookies } from '@app/utils/cookies';
import { meta } from '@app/utils/meta';

// New async handler pattern - no query() function needed
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authorization first
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Access token is required for this endpoint", event);
        }

        // Generate cookies that clear the tokens
        const clearCookies = generateLogoutCookies();

        return {
            code: 200,
            response: {
                status: "success",
                message: "Logged out successfully",
                ...meta(event),
            },
            cookies: clearCookies
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "Logout failed",
                errors: [error instanceof Error ? error.message : "An unexpected error occurred"],
                ...meta(event),
            }
        };
    }
};
