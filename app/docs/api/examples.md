# API Examples

This document provides examples of common API patterns in TurboScript.

## Database Operations

### Basic CRUD Operations

#### Create Record

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const result = await turboQuery(
            'INSERT INTO items (name, description) VALUES ($1, $2) RETURNING *',
            [data.name, data.description]
        );

        return {
            code: 201,
            response: {
                status: "success",
                data: { item: result[0] }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to create item"
            }
        };
    }
};
```

#### Read Records

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const items = await turboQuery('SELECT * FROM items ORDER BY created_at DESC LIMIT 10');

        return {
            code: 200,
            response: {
                status: "success",
                data: { items }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to fetch items"
            }
        };
    }
};
```

#### Update Record

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const result = await turboQuery(
            'UPDATE items SET name = $1, description = $2 WHERE id = $3 RETURNING *',
            [data.name, data.description, event.pathParameters.id]
        );

        if (!result.length) {
            return {
                code: 404,
                response: {
                    status: "error",
                    message: "Item not found"
                }
            };
        }

        return {
            code: 200,
            response: {
                status: "success",
                data: { item: result[0] }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to update item"
            }
        };
    }
};
```

#### Delete Record

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const result = await turboQuery(
            'DELETE FROM items WHERE id = $1 RETURNING id',
            [event.pathParameters.id]
        );

        if (!result.length) {
            return {
                code: 404,
                response: {
                    status: "error",
                    message: "Item not found"
                }
            };
        }

        return {
            code: 200,
            response: {
                status: "success",
                data: { message: "Item deleted successfully" }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to delete item"
            }
        };
    }
};
```

## Authentication

### Login Endpoint with JWT

```typescript
import { createJWT, hashPassword, verifyPassword } from '../utils/auth';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { email, password } = data;

        // Get user from database
        const users = await turboQuery('SELECT * FROM users WHERE email = $1', [email]);

        if (users.length === 0) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "Invalid credentials"
                }
            };
        }

        const user = users[0];

        // Verify password
        if (!verifyPassword(password, user.password_hash)) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "Invalid credentials"
                }
            };
        }

        // Create JWT token
        const token = createJWT({
            uid: user.uid,
            email: user.email,
            name: user.name
        });

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    token,
                    user: {
                        uid: user.uid,
                        email: user.email,
                        name: user.name
                    }
                }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Login failed"
            }
        };
    }
};
```

### Protected Route with Authentication

```typescript
import { verifyAuth, createAuthErrorResponse } from '../utils/auth';
import { meta } from '../utils/meta';

// Handle function with authentication
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authentication directly in handle function
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Access token is required for this endpoint", event);
        }

        // Use authenticated user's data directly
        const userUid = userPayload.uid;

        const userProfile = await turboQuery(
            'SELECT * FROM user_profiles WHERE user_id = $1',
            [userUid]
        );

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    user: userPayload,
                    profile: userProfile[0]
                },
                ...meta(event),
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to fetch profile",
                ...meta(event),
            }
        };
    }
};
```

## Background Jobs

### Email Queue Example

```typescript
import { queueEmail } from '../utils/queue';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Save contact request to database
        const contact = await turboQuery(
            'INSERT INTO contact_requests (name, email, message) VALUES ($1, $2, $3) RETURNING *',
            [data.name, data.email, data.message]
        );

        // Queue email notification (non-blocking)
        queueEmail({
            to: 'admin@example.com',
            subject: 'New Contact Request',
            template: 'contact-notification',
            data: {
                name: data.name,
                email: data.email,
                message: data.message
            }
        });

        // Queue confirmation email to user
        queueEmail({
            to: data.email,
            subject: 'Thank you for contacting us',
            template: 'contact-confirmation',
            data: {
                name: data.name
            }
        });

        return {
            code: 201,
            response: {
                status: "success",
                message: "Contact request submitted successfully",
                data: { id: contact[0].id }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to submit contact request"
            }
        };
    }
};
```

## Response Types

### Standard Response Format

All TurboScript API endpoints should return responses in this format:

```typescript
interface TurboScriptResponse {
    code: number;
    response: {
        status: "success" | "error";
        message?: string;
        data?: any;
        meta?: {
            timestamp: string;
            requestId: string;
            [key: string]: any;
        };
    };
}
```

### Success Response Examples

```typescript
// Simple success
return {
    code: 200,
    response: {
        status: "success",
        data: { users }
    }
};

// Success with metadata
return {
    code: 200,
    response: {
        status: "success",
        data: { users },
        meta: {
            total: 150,
            page: 1,
            limit: 10,
            timestamp: new Date().toISOString()
        }
    }
};
```

### Error Response Examples

```typescript
// Validation error
return {
    code: 400,
    response: {
        status: "error",
        message: "Email is required"
    }
};

// Authorization error
return {
    code: 401,
    response: {
        status: "error",
        message: "Invalid or expired token"
    }
};

// Server error
return {
    code: 500,
    response: {
        status: "error",
        message: "Internal server error"
    }
};
```

## Summary

These examples demonstrate the core patterns for building APIs with TurboScript:

- **Database Operations**: Use `await turboQuery()` for database access with parallel execution via `Promise.all()`
- **Background Jobs**: Queue non-blocking operations like email sending
- **Response Consistency**: Use standard response formats across all endpoints
- **Error Handling**: Always wrap operations in try/catch blocks with proper HTTP status codes

For more advanced patterns and performance optimization, see the [Best Practices Guide](../guides/best-practices.md).

## Advanced Examples

### Real-World E-commerce API

#### Product Catalog with Filtering

```typescript
// app/routes/products/search.ts
interface ProductSearchQuery {
    q?: string;           // Search query
    category?: string;    // Category filter
    min_price?: string;   // Price range
    max_price?: string;
    sort?: 'price' | 'name' | 'rating' | 'created_at';
    order?: 'asc' | 'desc';
    page?: string;
    limit?: string;
}

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const query = event.queryParameters as ProductSearchQuery;

        // Build dynamic query
        let whereClause = 'p.active = true';
        const queryParams: any[] = [];
        let paramIndex = 1;

        // Search filter
        if (query.q) {
            whereClause += ` AND (p.name ILIKE $${paramIndex} OR p.description ILIKE $${paramIndex})`;
            queryParams.push(`%${query.q}%`);
            paramIndex++;
        }

        // Category filter
        if (query.category) {
            whereClause += ` AND c.slug = $${paramIndex}`;
            queryParams.push(query.category);
            paramIndex++;
        }

        // Price range filter
        if (query.min_price) {
            whereClause += ` AND p.price >= $${paramIndex}`;
            queryParams.push(parseFloat(query.min_price));
            paramIndex++;
        }

        if (query.max_price) {
            whereClause += ` AND p.price <= $${paramIndex}`;
            queryParams.push(parseFloat(query.max_price));
            paramIndex++;
        }

        // Pagination
        const page = Math.max(1, parseInt(query.page || '1', 10));
        const limit = Math.min(50, Math.max(1, parseInt(query.limit || '20', 10)));
        const offset = (page - 1) * limit;

        // Sorting
        const allowedSortFields = ['price', 'name', 'rating', 'created_at'];
        const sortField = allowedSortFields.includes(query.sort || '') ? query.sort : 'created_at';
        const sortOrder = query.order === 'asc' ? 'ASC' : 'DESC';

        // Execute queries in parallel
        const [products, totalCount] = await Promise.all([
            turboQuery(`
                SELECT
                    p.id, p.name, p.description, p.price, p.rating, p.image_url,
                    c.name as category_name, c.slug as category_slug,
                    p.created_at
                FROM products p
                LEFT JOIN categories c ON p.category_id = c.id
                WHERE ${whereClause}
                ORDER BY p.${sortField} ${sortOrder}
                LIMIT $${paramIndex} OFFSET $${paramIndex + 1}
            `, [...queryParams, limit, offset]),

            turboQuery(`
                SELECT COUNT(*) as total
                FROM products p
                LEFT JOIN categories c ON p.category_id = c.id
                WHERE ${whereClause}
            `, queryParams)
        ]);

        const total = parseInt(totalCount[0].total, 10);
        const totalPages = Math.ceil(total / limit);

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    products,
                    pagination: {
                        page,
                        limit,
                        total,
                        total_pages: totalPages,
                        has_next: page < totalPages,
                        has_prev: page > 1
                    },
                    filters: {
                        search: query.q || null,
                        category: query.category || null,
                        price_range: {
                            min: query.min_price ? parseFloat(query.min_price) : null,
                            max: query.max_price ? parseFloat(query.max_price) : null
                        }
                    }
                }
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to search products"
            }
        };
    }
};
```

#### Shopping Cart Management

```typescript
// app/routes/cart/add-item.ts
import { verifyAuth, createAuthErrorResponse } from '@app/utils/auth';
import { meta } from '@app/utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authentication directly in handle function
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Authentication required", event);
        }

        // Use authenticated user's data directly
        const userUid = userPayload.uid;

        const { product_id, quantity = 1 } = event.body as {
            product_id: string;
            quantity?: number;
        };

        if (!product_id) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Product ID is required",
                    ...meta(event),
                }
            };
        }

        if (quantity < 1 || quantity > 10) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Quantity must be between 1 and 10",
                    ...meta(event),
                }
            };
        }

        // Validate product and check stock
        const products = await turboQuery(
            'SELECT id, name, price, stock, active FROM products WHERE id = $1',
            [product_id]
        );

        if (!products.length) {
            return {
                code: 404,
                response: {
                    status: "error",
                    message: "Product not found",
                    ...meta(event),
                }
            };
        }

        const product = products[0];

        if (!product.active) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Product is not available",
                    ...meta(event),
                }
            };
        }

        if (product.stock < quantity) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: `Only ${product.stock} items available in stock`,
                    ...meta(event),
                }
            };
        }

        // Get user's cart and check if item already exists
        const existingCartItems = await turboQuery(
            'SELECT id, quantity FROM cart_items WHERE user_id = $1 AND product_id = $2',
            [userUid, product_id]
        );

        let cartResult;

        if (existingCartItems.length > 0) {
            // Update existing cart item
            const newQuantity = existingCartItems[0].quantity + quantity;

            if (newQuantity > product.stock) {
                return {
                    code: 400,
                    response: {
                        status: "error",
                        message: `Cannot add ${quantity} more items. Stock limit: ${product.stock}, current in cart: ${existingCartItems[0].quantity}`,
                        ...meta(event),
                    }
                };
            }

            cartResult = await turboQuery(
                'UPDATE cart_items SET quantity = $1, updated_at = NOW() WHERE id = $2 RETURNING *',
                [newQuantity, existingCartItems[0].id]
            );
        } else {
            // Add new cart item
            cartResult = await turboQuery(
                'INSERT INTO cart_items (user_id, product_id, quantity, created_at) VALUES ($1, $2, $3, NOW()) RETURNING *',
                [userUid, product_id, quantity]
            );
        }

        // Get updated cart totals
        const cartSummary = await turboQuery(`
            SELECT
                COUNT(*) as item_count,
                SUM(ci.quantity * p.price) as total_amount
            FROM cart_items ci
            JOIN products p ON ci.product_id = p.id
            WHERE ci.user_id = $1
        `, [userUid]);

        return {
            code: 200,
            response: {
                status: "success",
                message: "Item added to cart",
                data: {
                    cart_item: cartResult[0],
                    cart_summary: cartSummary[0]
                },
                ...meta(event),
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to add item to cart",
                ...meta(event),
            }
        };
    }
};
```

### Analytics and Reporting

#### Dashboard Analytics

```typescript
// app/routes/analytics/dashboard.ts
import { verifyAuth, createAuthErrorResponse } from '@app/utils/auth';
import { meta } from '@app/utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authentication and admin role directly in handle function
        const userPayload = verifyAuth(event);
        if (!userPayload || !userPayload.roles?.includes('admin')) {
            return createAuthErrorResponse("Admin access required", event);
        }

        const { period = '30d' } = event.queryParameters;

        // Calculate date range
        const endDate = new Date();
        const startDate = new Date();

        switch (period) {
            case '7d':
                startDate.setDate(endDate.getDate() - 7);
                break;
            case '30d':
                startDate.setDate(endDate.getDate() - 30);
                break;
            case '90d':
                startDate.setDate(endDate.getDate() - 90);
                break;
            default:
                startDate.setDate(endDate.getDate() - 30);
        }

        // Execute all analytics queries in parallel
        const [
            userMetrics,
            orderMetrics,
            revenueMetrics,
            topProducts,
            geographicData,
            conversionRates
        ] = await Promise.all([
            // User metrics
            turboQuery(`
                SELECT
                    COUNT(*) as total_users,
                    COUNT(*) FILTER (WHERE created_at >= $1) as new_users,
                    COUNT(*) FILTER (WHERE last_login >= $1) as active_users
                FROM users
            `, [startDate]),

            // Order metrics
            turboQuery(`
                SELECT
                    COUNT(*) as total_orders,
                    COUNT(*) FILTER (WHERE status = 'completed') as completed_orders,
                    COUNT(*) FILTER (WHERE status = 'pending') as pending_orders,
                    AVG(total) as average_order_value
                FROM orders
                WHERE created_at >= $1
            `, [startDate]),

            // Revenue metrics
            turboQuery(`
                SELECT
                    SUM(total) as total_revenue,
                    SUM(total) FILTER (WHERE DATE(created_at) = CURRENT_DATE) as today_revenue,
                    SUM(total) FILTER (WHERE created_at >= CURRENT_DATE - INTERVAL '7 days') as week_revenue
                FROM orders
                WHERE created_at >= $1 AND status = 'completed'
            `, [startDate]),

            // Top products
            turboQuery(`
                SELECT
                    p.name,
                    SUM(oi.quantity) as units_sold,
                    SUM(oi.total) as revenue
                FROM order_items oi
                JOIN products p ON oi.product_id = p.id
                JOIN orders o ON oi.order_id = o.id
                WHERE o.created_at >= $1 AND o.status = 'completed'
                GROUP BY p.id, p.name
                ORDER BY revenue DESC
                LIMIT 10
            `, [startDate]),

            // Geographic data
            turboQuery(`
                SELECT
                    shipping_country,
                    COUNT(*) as order_count,
                    SUM(total) as revenue
                FROM orders
                WHERE created_at >= $1 AND status = 'completed'
                GROUP BY shipping_country
                ORDER BY revenue DESC
                LIMIT 10
            `, [startDate]),

            // Conversion rates
            turboQuery(`
                SELECT
                    DATE(date) as date,
                    SUM(page_views) as views,
                    SUM(conversions) as conversions,
                    CASE
                        WHEN SUM(page_views) > 0
                        THEN (SUM(conversions)::float / SUM(page_views)::float) * 100
                        ELSE 0
                    END as conversion_rate
                FROM daily_analytics
                WHERE date >= $1
                GROUP BY DATE(date)
                ORDER BY date DESC
            `, [startDate])
        ]);

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    period,
                    date_range: {
                        start: startDate.toISOString(),
                        end: endDate.toISOString()
                    },
                    metrics: {
                        users: userMetrics[0],
                        orders: orderMetrics[0],
                        revenue: revenueMetrics[0]
                    },
                    top_products: topProducts,
                    geographic_distribution: geographicData,
                    conversion_trends: conversionRates
                }
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to fetch analytics"
            }
        };
    }
};
```

### File Management System

#### File Upload with Validation

```typescript
// app/routes/files/upload.ts
import { verifyAuth, createAuthErrorResponse } from '@app/utils/auth';
import { meta } from '@app/utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authentication directly in handle function
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Authentication required", event);
        }

        // Use authenticated user's data directly
        const userUid = userPayload.uid;

        const {
            fileData,
            file_name,
            content_type,
            folder = 'general',
            is_public = false
        } = event.body as {
            fileData: string;        // Base64 encoded
            file_name: string;
            content_type: string;
            folder?: string;
            is_public?: boolean;
        };

        // Validation
        if (!fileData || !file_name || !content_type) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "File data, name, and content type are required",
                    ...meta(event),
                }
            };
        }

        // File type validation
        const allowedTypes = [
            'image/jpeg', 'image/png', 'image/gif', 'image/webp',
            'application/pdf', 'text/plain', 'text/csv',
            'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
        ];

        if (!allowedTypes.includes(content_type)) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "File type not allowed",
                    allowed_types: allowedTypes,
                    ...meta(event),
                }
            };
        }

        // File size validation
        const fileSizeBytes = (fileData.length * 3) / 4;
        const maxSizeBytes = 10 * 1024 * 1024; // 10MB

        if (fileSizeBytes > maxSizeBytes) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "File size exceeds 10MB limit",
                    ...meta(event),
                }
            };
        }

        // Generate unique filename
        const timestamp = Date.now();
        const fileExtension = file_name.split('.').pop()?.toLowerCase();
        const uniqueFileName = `${folder}/${timestamp}_${userUid}.${fileExtension}`;

        // Check user storage quota
        const storageUsage = await turboQuery(
            'SELECT COALESCE(SUM(file_size), 0) as total_size FROM user_files WHERE user_id = $1',
            [userUid]
        );

        const currentUsage = parseInt(storageUsage[0].total_size, 10);
        const storageLimit = 100 * 1024 * 1024; // 100MB per user

        if (currentUsage + fileSizeBytes > storageLimit) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Storage quota exceeded",
                    ...meta(event),
                    quota: {
                        used: currentUsage,
                        limit: storageLimit,
                        remaining: storageLimit - currentUsage
                    }
                }
            };
        }

        // Save file to database and update user quota
        const [fileResult, quotaUpdate] = await Promise.all([
            turboQuery(`
                INSERT INTO user_files (
                    user_id, filename, original_name, content_type,
                    fileData, file_size, folder, is_public, created_at
                ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
                RETURNING id, filename, file_size, created_at
            `, [
                userUid, uniqueFileName, file_name, content_type,
                fileData, fileSizeBytes, folder, is_public
            ]),

            turboQuery(
                'UPDATE users SET storage_used = storage_used + $1 WHERE uid = $2',
                [fileSizeBytes, userUid]
            )
        ]);

        // Generate access URL
        const file = fileResult[0];
        const accessUrl = is_public
            ? `/api/files/public/${file.filename}`
            : `/api/files/private/${file.filename}?token=${generateFileToken(file.id, userUid)}`;

        return {
            code: 201,
            response: {
                status: "success",
                message: "File uploaded successfully",
                data: {
                    file: {
                        id: file.id,
                        filename: file.filename,
                        original_name: file_name,
                        size: file.file_size,
                        url: accessUrl,
                        is_public,
                        created_at: file.created_at
                    },
                    quota: {
                        used: currentUsage + fileSizeBytes,
                        limit: storageLimit
                    }
                }
            };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to upload file"
            }
        };
    }
};

// Helper function for generating file access tokens
function generateFileToken(fileId: string, userId: string): string {
    // Implementation would generate a secure token for file access
    return `${fileId}_${userId}_${Date.now()}`;
}
```

### Real-time Notifications

#### Notification System

```typescript
// app/routes/notifications/send.ts
import { verifyAuth, createAuthErrorResponse } from '@app/utils/auth';
import { meta } from '@app/utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authentication directly in handle function
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Authentication required", event);
        }

        // Use authenticated user's data directly
        const senderUid = userPayload.uid;

        const {
            recipient_ids,
            title,
            message,
            type = 'info',
            action_url,
            priority = 'normal'
        } = event.body as {
            recipient_ids: string[];
            title: string;
            message: string;
            type?: 'info' | 'warning' | 'success' | 'error';
            action_url?: string;
            priority?: 'low' | 'normal' | 'high' | 'urgent';
        };

        // Validation
        if (!recipient_ids || !Array.isArray(recipient_ids) || recipient_ids.length === 0) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "At least one recipient is required",
                    ...meta(event),
                }
            };
        }

        if (!title || !message) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Title and message are required",
                    ...meta(event),
                }
            };
        }

        // Validate recipients exist
        const recipients = await turboQuery(
            'SELECT uid, email, name, notification_preferences FROM users WHERE uid = ANY($1) AND active = true',
            [recipient_ids]
        );

        if (recipients.length === 0) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "No valid recipients found",
                    ...meta(event),
                }
            };
        }

        // Create notifications for all recipients
        const notificationInserts = recipients.map(recipient =>
            turboQuery(`
                INSERT INTO notifications (
                    user_id, sender_id, title, message, type,
                    action_url, priority, created_at
                ) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
                RETURNING id
            `, [
                recipient.uid, senderUid, title, message,
                type, action_url, priority
            ])
        );

        const notificationResults = await Promise.all(notificationInserts);
        const notificationIds = notificationResults.map(result => result[0].id);

        // Process different notification types in parallel
        const backgroundJobs = [];

        // Email notifications (for users who have email enabled)
        const emailRecipients = recipients.filter(r =>
            r.notification_preferences?.email !== false
        );

        if (emailRecipients.length > 0) {
            backgroundJobs.push(
                turboJob('send-notification-emails', {
                    recipients: emailRecipients.map(r => ({
                        email: r.email,
                        name: r.name,
                        notification_id: notificationIds[recipients.indexOf(r)]
                    })),
                    title,
                    message,
                    action_url
                })
            );
        }

        // Push notifications (for users with push enabled)
        const pushRecipients = recipients.filter(r =>
            r.notification_preferences?.push !== false
        );

        if (pushRecipients.length > 0) {
            backgroundJobs.push(
                turboJob('send-push-notifications', {
                    recipients: pushRecipients.map(r => r.uid),
                    title,
                    message,
                    action_url,
                    priority
                })
            );
        }

        // SMS notifications (for urgent priority)
        if (priority === 'urgent') {
            const smsRecipients = recipients.filter(r =>
                r.notification_preferences?.sms === true && r.phone_number
            );

            if (smsRecipients.length > 0) {
                backgroundJobs.push(
                    turboJob('send-sms-notifications', {
                        recipients: smsRecipients.map(r => ({
                            phone: r.phone_number,
                            name: r.name
                        })),
                        message: `${title}: ${message}`
                    })
                );
            }
        }

        // Dispatch all background jobs
        await Promise.all(backgroundJobs);

        // Update notification stats
        await turboQuery(
            'INSERT INTO notification_stats (date, sent_count, type, priority) VALUES (CURRENT_DATE, $1, $2, $3) ON CONFLICT (date, type, priority) DO UPDATE SET sent_count = notification_stats.sent_count + $1',
            [recipients.length, type, priority]
        );

        return {
            code: 201,
            response: {
                status: "success",
                message: "Notifications sent successfully",
                data: {
                    sent_to: recipients.length,
                    notification_ids: notificationIds,
                    delivery_methods: {
                        in_app: recipients.length,
                        email: emailRecipients.length,
                        push: pushRecipients.length,
                        sms: priority === 'urgent' ? recipients.filter(r => r.phone_number).length : 0
                    }
                }
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to send notifications"
            }
        };
    }
};
```

### Content Management

#### Blog Post Management

```typescript
// app/routes/blog/create.ts
import { verifyAuth, createAuthErrorResponse } from '@app/utils/auth';
import { meta } from '@app/utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authentication and editor role directly in handle function
        const userPayload = verifyAuth(event);
        if (!userPayload || !userPayload.roles?.includes('editor')) {
            return createAuthErrorResponse("Editor access required", event);
        }

        // Use authenticated user's data directly
        const authorUid = userPayload.uid;

        const {
            title,
            content,
            excerpt,
            tags = [],
            category_id,
            featured_image_url,
            meta_description,
            publish_date,
            status = 'draft'
        } = event.body as {
            title: string;
            content: string;
            excerpt?: string;
            tags?: string[];
            category_id?: string;
            featured_image_url?: string;
            meta_description?: string;
            publish_date?: string;
            status?: 'draft' | 'published' | 'scheduled';
        };

        // Validation
        if (!title || !content) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Title and content are required",
                    ...meta(event),
                }
            };
        }

        // Generate slug from title
        const slug = title
            .toLowerCase()
            .replace(/[^a-z0-9\s-]/g, '')
            .replace(/\s+/g, '-')
            .replace(/-+/g, '-')
            .trim('-');

        // Check if slug is unique
        const existingSlugs = await turboQuery(
            'SELECT id FROM blog_posts WHERE slug = $1',
            [slug]
        );

        let uniqueSlug = slug;
        if (existingSlugs.length > 0) {
            uniqueSlug = `${slug}-${Date.now()}`;
        }

        // Validate category if provided
        if (category_id) {
            const categories = await turboQuery(
                'SELECT id FROM blog_categories WHERE id = $1 AND active = true',
                [category_id]
            );

            if (categories.length === 0) {
                return {
                    code: 400,
                    response: {
                        status: "error",
                        message: "Invalid category ID",
                        ...meta(event),
                    }
                };
            }
        }

        // Generate excerpt if not provided
        const autoExcerpt = excerpt || content.substring(0, 150).replace(/<[^>]*>/g, '') + '...';

        // Auto-generate meta description if not provided
        const autoMetaDescription = meta_description || autoExcerpt;

        // Handle publish date
        let publishAt = null;
        if (status === 'published') {
            publishAt = new Date();
        } else if (status === 'scheduled' && publish_date) {
            publishAt = new Date(publish_date);
            if (publishAt <= new Date()) {
                return {
                    code: 400,
                    response: {
                        status: "error",
                        message: "Scheduled publish date must be in the future",
                        ...meta(event),
                    }
                };
            }
        }

        // Create blog post and handle tags in parallel
        const [postResult, tagProcessing] = await Promise.all([
            // Create blog post
            turboQuery(`
                INSERT INTO blog_posts (
                    title, slug, content, excerpt, meta_description,
                    author_id, category_id, featured_image_url,
                    status, published_at, created_at
                ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
                RETURNING id, slug, created_at
            `, [
                title, uniqueSlug, content, autoExcerpt, autoMetaDescription,
                authorUid, category_id, featured_image_url,
                status, publishAt
            ]),

            // Process tags
            tags.length > 0 ? Promise.all(
                tags.map(async (tagName) => {
                    // Create tag if it doesn't exist
                    const tagSlug = tagName.toLowerCase().replace(/\s+/g, '-');
                    const existingTags = await turboQuery(
                        'SELECT id FROM blog_tags WHERE slug = $1',
                        [tagSlug]
                    );

                    if (existingTags.length > 0) {
                        return existingTags[0];
                    } else {
                        const newTag = await turboQuery(
                            'INSERT INTO blog_tags (name, slug, created_at) VALUES ($1, $2, NOW()) RETURNING id',
                            [tagName, tagSlug]
                        );
                        return newTag[0];
                    }
                })
            ) : Promise.resolve([])
        ]);

        const post = postResult[0];
        const processedTags = await tagProcessing;

        // Associate tags with post
        if (processedTags.length > 0) {
            const tagAssociations = processedTags.map(tag =>
                turboQuery(
                    'INSERT INTO blog_post_tags (post_id, tag_id) VALUES ($1, $2)',
                    [post.id, tag.id]
                )
            );
            await Promise.all(tagAssociations);
        }

        // Dispatch background jobs
        const backgroundJobs = [];

        if (status === 'published') {
            // Send notifications to subscribers
            backgroundJobs.push(
                turboJob('notify-blog-subscribers', {
                    postId: post.id,
                    title,
                    excerpt: autoExcerpt,
                    url: `/blog/${uniqueSlug}`
                })
            );

            // Update search index
            backgroundJobs.push(
                turboJob('update-search-index', {
                    type: 'blog_post',
                    id: post.id,
                    action: 'create'
                })
            );

            // Generate social media posts
            backgroundJobs.push(
                turboJob('generate-social-posts', {
                    postId: post.id,
                    title,
                    excerpt: autoExcerpt,
                    featuredImage: featured_image_url
                })
            );
        } else if (status === 'scheduled') {
            // Schedule publication job
            backgroundJobs.push(
                turboJob('schedule-blog-publication', {
                    postId: post.id,
                    publishAt: publishAt?.toISOString()
                })
            );
        }

        await Promise.all(backgroundJobs);

        return {
            code: 201,
            response: {
                status: "success",
                message: "Blog post created successfully",
                data: {
                    post: {
                        id: post.id,
                        title,
                        slug: uniqueSlug,
                        status,
                        published_at: publishAt?.toISOString(),
                        created_at: post.created_at,
                        url: `/blog/${uniqueSlug}`,
                        tags: tags,
                        author: {
                            id: author.uid,
                            name: author.name
                        }
                    }
                }
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to create blog post"
            }
        };
    }
};
```

---

## Navigation

**Previous:** [← Background Jobs](api/background-jobs.md)
**Next:** [Best Practices →](guides/best-practices.md)

## Related Topics

- [Best Practices Guide](guides/best-practices.md)
- [Security Guidelines](guides/security.md)
- [Performance Optimization](guides/performance.md)
- [Development Workflow](guides/development.md)
