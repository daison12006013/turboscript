import { meta } from '@app/utils/meta';
import { validateConfirmationToken } from '@app/utils/crypto';
import type { User } from '@app/routes/types';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const confirmationToken = event.pathParameters.confirmationToken;

        if (!confirmationToken) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Validation failed",
                    errors: ["Confirmation token is required"],
                    ...meta(event),
                }
            };
        }

        // Validate and decrypt the confirmation token
        const appSecret = event.env.APP_SECRET || 'fallback-secret';
        const tokenData = validateConfirmationToken(confirmationToken, appSecret);

        if (!tokenData) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Invalid or expired confirmation token",
                    errors: ["The confirmation token is invalid or has expired. Please request a new verification email."],
                    ...meta(event),
                }
            };
        }

        const { uid: userUid } = tokenData;

        // Find user by UID
        const userResult = await turboQuery(
            "SELECT uid, email, email_verified_at FROM users WHERE uid = $1 LIMIT 1",
            [userUid]
        ) as User[];

        if (userResult.length === 0) {
            return {
                code: 404,
                response: {
                    status: "error",
                    message: "User not found",
                    errors: ["No user found with the provided confirmation token"],
                    ...meta(event),
                }
            };
        }

        const user = userResult[0];

        // Check if email is already verified
        if (user.email_verified_at) {
            return {
                code: 200,
                response: {
                    status: "success",
                    message: "Email already verified",
                    data: {
                        email: user.email,
                        verified_at: user.email_verified_at,
                        message: "Your email address has already been verified."
                    },
                    ...meta(event),
                }
            };
        }

        // Update email_verified_at timestamp
        await turboQuery(
            "UPDATE users SET email_verified_at = CURRENT_TIMESTAMP WHERE uid = $1",
            [userUid]
        );

        // Fetch updated user data
        const updatedUserResult = await turboQuery(
            "SELECT uid, email, email_verified_at FROM users WHERE uid = $1 LIMIT 1",
            [userUid]
        ) as User[];

        return {
            code: 200,
            response: {
                status: "success",
                message: "Email verified successfully",
                data: {
                    email: updatedUserResult[0].email,
                    verified_at: updatedUserResult[0].email_verified_at,
                    message: "Your email address has been successfully verified!"
                },
                ...meta(event),
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "Email verification failed",
                errors: [error instanceof Error ? error.message : "An unexpected error occurred"],
                ...meta(event),
            }
        };
    }
};
