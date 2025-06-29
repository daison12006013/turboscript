/**
 * Example route demonstrating turboQuery function overloading with multiple database connections
 *
 * This route shows how to use both the legacy format and the new object format
 * for turboQuery with different database connections.
 */

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const userId = event.pathParameters.uid;

        if (!userId) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "User ID is required"
                }
            };
        }

        // Example 1: Legacy format (uses default connection)
        const userBasic = await turboQuery('SELECT id, name, email FROM users WHERE id = $1', [userId]);

        // Example 2: Object format with default connection (equivalent to legacy)
        const userProfile = await turboQuery({
            query: 'SELECT * FROM user_profiles WHERE user_id = $1',
            bindings: [userId]
            // connection not specified, uses default
        });

        // Example 3: Object format with specific connection (reader database)
        const userActivity = await turboQuery({
            query: 'SELECT * FROM user_activity WHERE user_id = $1 ORDER BY created_at DESC LIMIT 10',
            bindings: [userId],
            connection: 'reader'  // Use read replica for read-only queries
        });

        // Example 4: Multiple parallel queries with different connections
        const [orderHistory, analyticsData, preferences] = await Promise.all([
            // Orders from main database
            turboQuery({
                query: 'SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC LIMIT 5',
                bindings: [userId],
                connection: 'main'
            }),
            // Analytics from analytics database
            turboQuery({
                query: 'SELECT page_views, session_count FROM user_analytics WHERE user_id = $1 AND date >= $2',
                bindings: [userId, '2025-01-01'],
                connection: 'analytics'
            }),
            // Preferences from reader
            turboQuery({
                query: 'SELECT * FROM user_preferences WHERE user_id = $1',
                bindings: [userId],
                connection: 'reader'
            })
        ]);

        // Example 5: Write operation using main connection
        await turboQuery({
            query: 'UPDATE users SET last_seen = CURRENT_TIMESTAMP WHERE id = $1',
            bindings: [userId],
            connection: 'main'
        });

        // Example 6: Insert into analytics database
        await turboQuery({
            query: 'INSERT INTO api_access_log (user_id, endpoint, timestamp) VALUES ($1, $2, CURRENT_TIMESTAMP)',
            bindings: [userId, `/api/users/${userId}`],
            connection: 'analytics'
        });

        return {
            code: 200,
            response: {
                status: "success",
                message: "Successfully demonstrated turboQuery overloading",
                data: {
                    user: userBasic[0],
                    profile: userProfile[0],
                    activity: userActivity,
                    orders: orderHistory,
                    analytics: analyticsData[0],
                    preferences: preferences[0]
                },
                examples: {
                    legacy_format: "turboQuery('SELECT * FROM users WHERE id = $1', [userId])",
                    object_format_default: "turboQuery({ query: 'SELECT * FROM users WHERE id = $1', bindings: [userId] })",
                    object_format_connection: "turboQuery({ query: 'SELECT * FROM users WHERE id = $1', bindings: [userId], connection: 'reader' })"
                }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "An unexpected error occurred",
                details: {
                    error_type: "database_operation",
                    suggestion: "Check your database connections in turboscript.yml and ensure all required environment variables are set"
                }
            }
        };
    }
};
