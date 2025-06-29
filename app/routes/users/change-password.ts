import { createAuthErrorResponse, verifyAuth } from '@app/utils/auth';
import { meta } from '@app/utils/meta';
import { hashPassword, validatePassword, verifyPassword } from '@app/utils/password';
import type { User } from '@app/routes/types';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authorization first
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Access token is required for this endpoint", event);
        }

        const input = event.body as {
            current_password?: string;
            new_password?: string;
            confirm_new_password?: string;
        };

        // Use authenticated user's UID instead of requiring it in the request body
        const userUid = userPayload.uid;

        if (!input.current_password) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Validation failed",
                    errors: ["Current password is required"],
                    ...meta(event),
                }
            };
        }

        if (!input.new_password) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Validation failed",
                    errors: ["New password is required"],
                    ...meta(event),
                }
            };
        }

        if (input.new_password !== input.confirm_new_password) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Validation failed",
                    errors: ["Password confirmation does not match"],
                    ...meta(event),
                }
            };
        }

        // Validate new password strength
        const passwordValidation = validatePassword(input.new_password, {
            minLength: 12,               // Increased for better security
            requireUppercase: true,
            requireLowercase: true,
            requireNumbers: true,
            requireSymbols: true,        // Enable symbols for better security
            forbidPersonalInfo: [        // Prevent using user's email in password
                userPayload.email,
                userPayload.name
            ]
        });

        if (!passwordValidation.valid) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Password validation failed",
                    errors: passwordValidation.errors,
                    strength: passwordValidation.strength,
                    ...meta(event),
                }
            };
        }

        // Fetch user to verify current password using turboQuery
        const userResult = await turboQuery(
            `SELECT uid, password FROM users WHERE uid = $1 LIMIT 1`,
            [userUid]
        );

        if (!Array.isArray(userResult) || userResult.length === 0) {
            return {
                code: 404,
                response: {
                    status: "error",
                    message: "User not found",
                    errors: ["User with provided UID does not exist"],
                    ...meta(event),
                }
            };
        }

        const user = userResult[0] as User;

        // Verify current password using async Argon2 verification
        const isCurrentPasswordValid = await verifyPassword(input.current_password, user.password);

        if (!isCurrentPasswordValid) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Current password is incorrect",
                    errors: ["Please provide your current password"],
                    ...meta(event),
                }
            };
        }

        // Hash the new password using async Argon2
        const newHashedPassword = await hashPassword(input.new_password);

        // Update the password using turboQuery
        await turboQuery(
            'UPDATE users SET password = $1, updated_at = CURRENT_TIMESTAMP WHERE uid = $2',
            [newHashedPassword, userUid]
        );

        return {
            code: 200,
            response: {
                status: "success",
                message: "Password updated successfully",
                data: {
                    uid: userUid,
                    message: "Password has been changed successfully"
                },
                ...meta(event),
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "Password change failed",
                errors: [error instanceof Error ? error.message : "An unexpected error occurred"],
                ...meta(event),
            }
        };
    }
};
