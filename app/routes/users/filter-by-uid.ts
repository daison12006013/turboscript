import { createAuthErrorResponse, verifyAuth } from "../../utils/auth";
import { meta } from "../../utils/meta";
import type { User } from '@app/routes/types';

/**
 * Validate UUID v4 format
 * @param uuid - The UUID string to validate
 * @returns boolean indicating if the UUID is valid
 */
const isValidUUIDv4 = (uuid: string): boolean => {
    const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
    return uuidRegex.test(uuid);
}

/**
 * Fetches a single user record by UID using async turboQuery
 * This endpoint returns exactly one user or 404 if not found
 * Public endpoint - no authentication required
 */
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authorization first
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Access token is required for this endpoint", event);
        }

        if (!event.pathParameters.uid) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Validation failed",
                    errors: ["User UID is required"],
                    ...meta(event),
                }
            };
        }

        // Validate UUID format to prevent PostgreSQL errors
        if (!isValidUUIDv4(event.pathParameters.uid)) {
            return {
                code: 404,
                response: {
                    status: "not_found",
                    message: "User not found",
                    ...meta(event),
                }
            };
        }

        // Fetch user using async turboQuery
        const userResult = await turboQuery(
            `select * from users where uid = $1 limit 1`,
            [event.pathParameters.uid]
        );

        if (!Array.isArray(userResult) || userResult.length === 0) {
            return {
                code: 404,
                response: {
                    status: "not_found",
                    message: "User not found",
                    ...meta(event),
                }
            };
        }

        const user = userResult[0] as User;

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    uid: user.uid,
                    name: user.name,
                    email: user.email,
                    created_at: user.created_at,
                },
                ...meta(event),
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "Internal server error occurred while fetching user",
                errors: [error instanceof Error ? error.message : "Unknown error"],
                ...meta(event),
            }
        };
    }
};
