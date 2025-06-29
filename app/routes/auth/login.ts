import { generateJWTCookies } from '@app/utils/cookies';
import { generateTokenPair } from '@app/utils/jwt';
import { meta } from '@app/utils/meta';
import { verifyPassword, needsRehash, hashPassword } from '@app/utils/password';
import type { User } from '../types';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const input = event.body as {
            email?: string;
            password?: string;
        };

        if (!input.email) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Validation failed",
                    errors: ["Email is required"],
                    ...meta(event),
                },
            };
        }

        if (!input.password) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Validation failed",
                    errors: ["Password is required"],
                    ...meta(event),
                },
            };
        }

        const invalidCredentials: TurboScriptResponse = {
            code: 422,
            response: {
                status: "not_found",
                message: "Invalid credentials",
                errors: ["Email or password is incorrect"],
                ...meta(event),
            },
        };

        // Fetch user data using turboQuery
        const userResult = await turboQuery(`SELECT id, uid, name, email, password FROM users WHERE email = $1 LIMIT 1`, [input.email]);

        // Check if user exists
        if (!Array.isArray(userResult) || userResult.length === 0) {
            return invalidCredentials;
        }

        const user = userResult[0] as User;

        // Verify password using async Argon2 verification
        const isValidPassword = await verifyPassword(input.password, user.password);

        if (!isValidPassword) {
            return invalidCredentials;
        }

        // Check if password needs rehashing with improved security parameters
        const shouldRehash = needsRehash(user.password);

        // If password needs rehashing, update it in the background
        if (shouldRehash) {
            try {
                const newHash = await hashPassword(input.password);
                // Update the password hash in the database asynchronously
                await turboQuery(
                    'UPDATE users SET password = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2',
                    [newHash, user.id]
                );
            } catch (rehashError) {
                // Log the error but don't fail the login
                console.error('Failed to rehash password during login:', rehashError);
            }
        }

        // Generate JWT token pair
        const tokenPair = generateTokenPair({
            uid: user.uid,
            name: user.name,
            email: user.email
        }, event);

        // Generate secure cookies for tokens
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
                message: "Login successful",
                data: {
                    uid: user.uid,
                    name: user.name,
                    email: user.email,
                    access_token: tokenPair.accessToken,
                    refresh_token: tokenPair.refreshToken,
                    access_token_expires_at: tokenPair.accessTokenExpiresAt.toISOString(),
                    refresh_token_expires_at: tokenPair.refreshTokenExpiresAt.toISOString(),
                    password_needs_rehash: shouldRehash
                },
                ...meta(event),
            },
            cookies,
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "Login failed",
                errors: [error instanceof Error ? error.message : "An unexpected error occurred"],
                ...meta(event),
            },
        };
    }
};
