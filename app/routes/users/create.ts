import { meta } from '@app/utils/meta';
import { hashPassword, validatePassword } from '@app/utils/password';
import { generateConfirmationToken } from '@app/utils/crypto';
import type { User } from '@app/routes/types';

// New synchronous handler pattern using turboQuery
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const input = event.body as {
            name?: string;
            email?: string;
            password?: string;
            confirm_password?: string;
        };

        if (!input.name) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Validation failed",
                    errors: ["Name is required"],
                    ...meta(event),
                }
            };
        }

        if (!input.email) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Validation failed",
                    errors: ["Email is required"],
                    ...meta(event),
                }
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
                }
            };
        }

        if (input.password !== input.confirm_password) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Validation failed",
                    errors: ["Password mismatch"],
                    ...meta(event),
                }
            };
        }

        const passwordValidation = validatePassword(input.password, {
            minLength: 12,               // Increased for better security
            requireUppercase: true,
            requireLowercase: true,
            requireNumbers: true,
            requireSymbols: true,
            forbidPersonalInfo: [        // Prevent using personal info in password
                input.name,
                input.email
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

        // Hash password using async Argon2
        const hashedPassword = await hashPassword(input.password);

        // Check if user already exists
        const existingUserResult = await turboQuery(
            "SELECT uid FROM users WHERE email = $1 LIMIT 1",
            [input.email]
        ) as User[];

        if (existingUserResult.length > 0) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "User already exists",
                    errors: ["Email is already registered"],
                    ...meta(event),
                }
            };
        }

        // Create user and dispatch job sequentially for better error handling
        await turboQuery(
            "INSERT INTO users (name, email, password) VALUES ($1, $2, $3)",
            [input.name, input.email, hashedPassword]
        );

        // Fetch the user that was just created
        const fetchUserResult = await turboQuery(
            "SELECT * FROM users WHERE email = $1",
            [input.email]
        ) as User[];

        if (!Array.isArray(fetchUserResult) || fetchUserResult.length === 0) {
            return {
                code: 500,
                response: {
                    status: "error",
                    message: "Failed to create user",
                    errors: ["User creation failed unexpectedly after insert"],
                    ...meta(event),
                }
            };
        }

        const createdUser = fetchUserResult[0];
        const appSecret = event.env.APP_SECRET || 'fallback-secret';
        const confirmationToken = generateConfirmationToken(createdUser.uid, appSecret);

        // Dispatch email job in background after user creation
        await turboJob("send-confirmation-email", {
            email: input.email,
            name: input.name,
            confirmationToken,
        });

        return {
            code: 201,
            response: {
                status: "success",
                message: "User created successfully",
                data: {
                    uid: createdUser.uid,
                    name: createdUser.name,
                    email: createdUser.email,
                    created_at: createdUser.created_at,
                },
                ...meta(event),
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "User creation failed",
                errors: [error instanceof Error ? error.message : "An unexpected error occurred"],
                ...meta(event),
            }
        };
    }
};
