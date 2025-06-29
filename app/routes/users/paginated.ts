import { verifyAuth, createAuthErrorResponse } from "../../utils/auth";
import { meta } from "../../utils/meta";
import type { User } from '@app/routes/types';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authorization first
        const authUser = verifyAuth(event);
        if (!authUser) {
            return createAuthErrorResponse("Invalid or expired token", event);
        }

        const input = event.queryParameters as {
            page?: number;
            limit?: number;
            name_filter?: string;
            email_filter?: string;
            status_filter?: string;
            created_after?: string;
            created_before?: string;
            sort_by?: 'name' | 'email' | 'created_at' | 'id';
            sort_order?: 'ASC' | 'DESC';
        };

        // Set default pagination values
        const page = Math.max(1, input.page ?? 1);
        const limit = Math.min(100, Math.max(1, input.limit ?? 20));
        const offset = (page - 1) * limit;

        // Set default sorting
        const sortBy = input.sort_by ?? 'created_at';
        const sortOrder = input.sort_order ?? 'DESC';

        // Build WHERE conditions
        const conditions: string[] = [];
        const parameters: unknown[] = [];
        let parameterIndex = 1;

        if (input.name_filter) {
            conditions.push(`name ILIKE $${parameterIndex}`);
            parameters.push(`%${input.name_filter}%`);
            parameterIndex++;
        }

        if (input.email_filter) {
            conditions.push(`email ILIKE $${parameterIndex}`);
            parameters.push(`%${input.email_filter}%`);
            parameterIndex++;
        }

        if (input.created_after) {
            conditions.push(`created_at >= $${parameterIndex}`);
            parameters.push(input.created_after);
            parameterIndex++;
        }

        if (input.created_before) {
            conditions.push(`created_at <= $${parameterIndex}`);
            parameters.push(input.created_before);
            parameterIndex++;
        }

        const whereClause = conditions.length > 0 ? `WHERE ${conditions.join(' AND ')}` : '';

        // Build query with total count
        const usersQuery = `
            SELECT
                uid,
                name,
                email,
                created_at,
                updated_at,
                COUNT(*) OVER() as total_count
            FROM users
            ${whereClause}
            ORDER BY ${sortBy} ${sortOrder}
            LIMIT $${parameterIndex} OFFSET $${parameterIndex + 1}
        `;

        // Add pagination parameters
        parameters.push(limit, offset);

        // Execute query using async turboQuery
        const usersResult = await turboQuery(usersQuery, parameters);

        if (!Array.isArray(usersResult)) {
            throw new TypeError('Database query failed');
        }

        const users = usersResult as Array<User & { total_count: number }>;
        const totalCount = users.length > 0 ? users[0].total_count : 0;

        // Calculate pagination metadata
        const totalPages = Math.ceil(totalCount / limit);
        const hasNextPage = page < totalPages;
        const hasPreviousPage = page > 1;

        // Create pagination metadata for meta object (Postman contract expects this structure)
        const paginationMeta = {
            page: page,
            limit: limit,
            total_count: totalCount,
            total_pages: totalPages,
            has_next_page: hasNextPage,
            has_previous_page: hasPreviousPage,
            filters_applied: {
                name_filter: input.name_filter ?? null,
                email_filter: input.email_filter ?? null,
                created_after: input.created_after ?? null,
                created_before: input.created_before ?? null,
            },
            sorting: {
                sort_by: sortBy,
                sort_order: sortOrder,
            }
        };

        // Transform users data for response
        const responseUsers = users.map(user => ({
            uid: user.uid,
            name: user.name,
            email: user.email,
            created_at: user.created_at,
        }));

        const { meta: extractedMeta } = meta(event);

        return {
            code: 200,
            response: {
                status: "success",
                data: responseUsers, // Postman expects data to be the users array directly
                meta: {
                    ...(extractedMeta ?? {}),
                    pagination: paginationMeta // Postman expects pagination in meta object
                }
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "Failed to fetch paginated users",
                errors: [error instanceof Error ? error.message : "Unknown error"],
                meta: meta(event),
            }
        };
    }
};
