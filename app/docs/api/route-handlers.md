# Route Handlers in TurboScript

## Overview

Route handlers in TurboScript are TypeScript files that export an async `handle()` function to process HTTP requests.

## Basic Structure

```typescript
import { Event, TurboScriptResponse } from '../global';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Your business logic here
        return {
            code: 200,
            response: {
                status: "success",
                data: { /* your response data */ }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "An unexpected error occurred"
            }
        };
    }
};
```

## Database Operations

Use `turboQuery()` for database operations:

```typescript
// Single query
const users = await turboQuery(
    'SELECT * FROM users WHERE active = $1',
    [true]
);

// Parallel queries
const [orders, stats] = await Promise.all([
    turboQuery('SELECT * FROM orders WHERE user_id = $1', [userId]),
    turboQuery('SELECT COUNT(*) FROM orders WHERE user_id = $1', [userId])
]);
```

## Error Handling

Always use try/catch blocks and return standardized error responses:

```typescript
try {
    // Your code
} catch (error) {
    return {
        code: 500,
        response: {
            status: "error",
            message: error instanceof Error ? error.message : "An unexpected error occurred"
        }
    };
}
```

## Best Practices

1. Use TypeScript types for request/response data
2. Implement proper error handling
3. Use async/await for asynchronous operations
4. Validate input data
5. Use authorization when needed
6. Keep handlers focused and small
7. Use shared utilities for common operations

## Response Format

All responses should follow this format:

```typescript
interface SuccessResponse {
    code: number;
    response: {
        status: "success";
        data: any;
        meta?: Record<string, any>;
    };
}

interface ErrorResponse {
    code: number;
    response: {
        status: "error";
        message: string;
        details?: Record<string, any>;
    };
}
```

## Examples

### Basic GET Endpoint

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const items = await turboQuery('SELECT * FROM items LIMIT 10');

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

### Protected POST Endpoint

```typescript
import { verifyAuth, createAuthErrorResponse } from '../utils/auth';
import { meta } from '../utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authentication directly in handle function
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Authentication required", event);
        }

        // Use authenticated user's data directly
        const userId = userPayload.uid;

        const result = await turboQuery(
            'INSERT INTO items (name, user_id) VALUES ($1, $2) RETURNING *',
            [data.name, userId]
        );

        return {
            code: 201,
            response: {
                status: "success",
                data: { item: result[0] },
                ...meta(event),
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to create item",
                ...meta(event),
            }
        };
    }
};
```

## Best Practices

- Always use async/await patterns for database operations
- Validate input data before processing
- Use proper HTTP status codes
- Handle errors gracefully with try/catch blocks
- Leverage `Promise.all()` for parallel database queries
- Use `verifyAuth()` directly in `handle()` functions for authentication

## Common Patterns

- **CRUD Operations**: Create, Read, Update, Delete patterns
- **Validation**: Input validation and sanitization
- **Error Handling**: Consistent error response formats
- **Performance**: Parallel database queries with `Promise.all()`

## Advanced Route Handler Patterns

### Request Validation and Sanitization

```typescript
// app/routes/users/create.ts
interface CreateUserRequest {
    name: string;
    email: string;
    password: string;
    confirm_password?: string;
    age?: number;
}

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Type-safe request validation
        const input = event.body as CreateUserRequest;

        // Required field validation
        const requiredFields = ['name', 'email', 'password'];
        const missingFields = requiredFields.filter(field => !input[field]);

        if (missingFields.length > 0) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Missing required fields",
                    errors: missingFields.map(field => `${field} is required`)
                }
            };
        }

        // Email validation
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        if (!emailRegex.test(input.email)) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Invalid email format"
                }
            };
        }

        // Age validation (if provided)
        if (input.age !== undefined && (input.age < 13 || input.age > 120)) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Age must be between 13 and 120"
                }
            };
        }

        // Password confirmation check
        if (input.confirm_password && input.password !== input.confirm_password) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Password confirmation does not match"
                }
            };
        }

        // Check if email already exists
        const existingUsers = await turboQuery(
            'SELECT id FROM users WHERE email = $1',
            [input.email.toLowerCase()]
        );

        if (existingUsers.length > 0) {
            return {
                code: 409,
                response: {
                    status: "error",
                    message: "Email address is already registered"
                }
            };
        }

        // Create user with parallel operations
        const hashedPassword = await hashPassword(input.password);
        const userResult = await turboQuery(
            'INSERT INTO users (name, email, password, age, created_at) VALUES ($1, $2, $3, $4, NOW()) RETURNING id, name, email, created_at',
            [input.name.trim(), input.email.toLowerCase(), hashedPassword, input.age || null]
        );

        const user = userResult[0];

        // Dispatch background jobs
        await Promise.all([
            turboJob('send-welcome-email', {
                userId: user.id,
                email: user.email,
                name: user.name
            }),
            turboJob('track-user-registration', {
                userId: user.id,
                source: event.headers.referer || 'direct'
            })
        ]);

        return {
            code: 201,
            response: {
                status: "success",
                message: "User created successfully",
                data: {
                    user: {
                        id: user.id,
                        name: user.name,
                        email: user.email,
                        created_at: user.created_at
                    }
                }
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to create user"
            }
        };
    }
};
```

### Pagination and Filtering

```typescript
// app/routes/users/list.ts
interface ListUsersQuery {
    page?: string;
    limit?: string;
    search?: string;
    status?: 'active' | 'inactive' | 'all';
    sort?: 'name' | 'email' | 'created_at';
    order?: 'asc' | 'desc';
}

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const query = event.queryParameters as ListUsersQuery;

        // Parse and validate pagination parameters
        const page = Math.max(1, parseInt(query.page || '1', 10));
        const limit = Math.min(100, Math.max(1, parseInt(query.limit || '20', 10))); // Max 100 items
        const offset = (page - 1) * limit;

        // Build dynamic query
        let whereClause = '1=1';
        const queryParams: any[] = [];
        let paramIndex = 1;

        // Search filter
        if (query.search) {
            whereClause += ` AND (name ILIKE $${paramIndex} OR email ILIKE $${paramIndex})`;
            queryParams.push(`%${query.search}%`);
            paramIndex++;
        }

        // Status filter
        if (query.status && query.status !== 'all') {
            const isActive = query.status === 'active';
            whereClause += ` AND active = $${paramIndex}`;
            queryParams.push(isActive);
            paramIndex++;
        }

        // Sorting
        const allowedSortFields = ['name', 'email', 'created_at'];
        const sortField = allowedSortFields.includes(query.sort || '') ? query.sort : 'created_at';
        const sortOrder = query.order === 'asc' ? 'ASC' : 'DESC';

        // Execute queries in parallel
        const [users, totalCount] = await Promise.all([
            turboQuery(
                `SELECT id, name, email, active, created_at, last_login
                 FROM users
                 WHERE ${whereClause}
                 ORDER BY ${sortField} ${sortOrder}
                 LIMIT $${paramIndex} OFFSET $${paramIndex + 1}`,
                [...queryParams, limit, offset]
            ),
            turboQuery(
                `SELECT COUNT(*) as total FROM users WHERE ${whereClause}`,
                queryParams
            )
        ]);

        const total = parseInt(totalCount[0].total, 10);
        const totalPages = Math.ceil(total / limit);

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    users,
                    pagination: {
                        page,
                        limit,
                        total,
                        total_pages: totalPages,
                        has_next: page < totalPages,
                        has_prev: page > 1
                    },
                    filters: {
                        search: query.search || null,
                        status: query.status || 'all'
                    }
                }
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to fetch users"
            }
        };
    }
};
```

### File Upload Handling

```typescript
// app/routes/users/upload-avatar.ts
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

        const { file_data, file_name, content_type } = event.body as {
            file_data?: string;    // Base64 encoded file data
            file_name?: string;
            content_type?: string;
        };

        // Validate file upload
        if (!file_data || !file_name) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "File data and filename are required",
                    ...meta(event),
                }
            };
        }

        // Validate file type
        const allowedTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
        if (!content_type || !allowedTypes.includes(content_type)) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Only image files (JPEG, PNG, GIF, WebP) are allowed",
                    ...meta(event),
                }
            };
        }

        // Validate file size (base64 decoded size)
        const fileSizeBytes = (file_data.length * 3) / 4; // Approximate decoded size
        const maxSizeBytes = 5 * 1024 * 1024; // 5MB

        if (fileSizeBytes > maxSizeBytes) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "File size must be less than 5MB",
                    ...meta(event),
                }
            };
        }

        // Generate unique filename
        const fileExtension = file_name.split('.').pop()?.toLowerCase();
        const uniqueFileName = `avatar_${userUid}_${Date.now()}.${fileExtension}`;

        // Save file and update user record in parallel
        const [fileResult, userUpdate] = await Promise.all([
            // Save to file storage (example using database storage)
            turboQuery(
                'INSERT INTO user_files (user_id, filename, original_name, content_type, file_data, size_bytes, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW()) RETURNING id, filename',
                [userUid, uniqueFileName, file_name, content_type, file_data, fileSizeBytes]
            ),

            // Update user avatar reference
            turboQuery(
                'UPDATE users SET avatar_file_id = (SELECT id FROM user_files WHERE filename = $1), updated_at = NOW() WHERE uid = $2',
                [uniqueFileName, userUid]
            )
        ]);

        // Clean up old avatar files
        await turboJob('cleanup-old-avatars', {
            userId: userUid,
            excludeFileId: fileResult[0].id
        });

        return {
            code: 200,
            response: {
                status: "success",
                message: "Avatar uploaded successfully",
                data: {
                    file: {
                        id: fileResult[0].id,
                        filename: fileResult[0].filename,
                        url: `/api/files/${fileResult[0].filename}`
                    }
                },
                ...meta(event),
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to upload avatar",
                ...meta(event),
            }
        };
    }
};
```

### Complex Business Logic Example

```typescript
// app/routes/orders/create.ts
import { verifyAuth, createAuthErrorResponse } from '@app/utils/auth';
import { meta } from '@app/utils/meta';

interface CreateOrderRequest {
    items: OrderItem[];
    shipping_address: Address;
    payment_method_id: string;
    promo_code?: string;
}

interface OrderItem {
    product_id: string;
    quantity: number;
}

interface Address {
    street: string;
    city: string;
    state: string;
    zip_code: string;
    country: string;
}

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Check authentication directly in handle function
        const userPayload = verifyAuth(event);
        if (!userPayload) {
            return createAuthErrorResponse("Authentication required", event);
        }

        // Use authenticated user's data directly
        const userUid = userPayload.uid;
        const orderData = event.body as CreateOrderRequest;

        // Validate order data
        if (!orderData.items || orderData.items.length === 0) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Order must contain at least one item",
                    ...meta(event),
                }
            };
        }

        // Validate items and get product details
        const productIds = orderData.items.map(item => item.product_id);
        const [products, user_info] = await Promise.all([
            turboQuery(
                'SELECT id, name, price, stock, active FROM products WHERE id = ANY($1)',
                [productIds]
            ),
            turboQuery(
                'SELECT id, email, name FROM users WHERE uid = $1',
                [userUid]
            )
        ]);

        if (user_info.length === 0) {
            return {
                code: 404,
                response: {
                    status: "error",
                    message: "User not found",
                    ...meta(event),
                }
            };
        }

        const userInfo = user_info[0];

        // Validate product availability
        const productMap = new Map(products.map(p => [p.id, p]));
        const orderItems = [];
        let subtotal = 0;

        for (const item of orderData.items) {
            const product = productMap.get(item.product_id);

            if (!product) {
                return {
                    code: 400,
                    response: {
                        status: "error",
                        message: `Product ${item.product_id} not found`,
                        ...meta(event),
                    }
                };
            }

            if (!product.active) {
                return {
                    code: 400,
                    response: {
                        status: "error",
                        message: `Product ${product.name} is not available`,
                        ...meta(event),
                    }
                };
            }

            if (product.stock < item.quantity) {
                return {
                    code: 400,
                    response: {
                        status: "error",
                        message: `Insufficient stock for ${product.name}. Available: ${product.stock}, Requested: ${item.quantity}`
                    }
                };
            }

            const itemTotal = product.price * item.quantity;
            subtotal += itemTotal;

            orderItems.push({
                product_id: product.id,
                name: product.name,
                price: product.price,
                quantity: item.quantity,
                total: itemTotal
            });
        }

        // Apply promo code if provided
        let discount = 0;
        let promoCodeId = null;

        if (orderData.promo_code) {
            const promoCodes = await turboQuery(
                'SELECT id, discount_percent, discount_amount, minimum_order, expires_at, usage_limit, times_used FROM promo_codes WHERE code = $1 AND active = true',
                [orderData.promo_code]
            );

            if (promoCodes.length > 0) {
                const promo = promoCodes[0];

                // Validate promo code
                if (new Date(promo.expires_at) < new Date()) {
                    return {
                        code: 400,
                        response: {
                            status: "error",
                            message: "Promo code has expired"
                        }
                    };
                }

                if (promo.usage_limit && promo.times_used >= promo.usage_limit) {
                    return {
                        code: 400,
                        response: {
                            status: "error",
                            message: "Promo code usage limit reached"
                        }
                    };
                }

                if (promo.minimum_order && subtotal < promo.minimum_order) {
                    return {
                        code: 400,
                        response: {
                            status: "error",
                            message: `Minimum order amount of $${promo.minimum_order} required for this promo code`
                        }
                    };
                }

                // Calculate discount
                if (promo.discount_percent) {
                    discount = subtotal * (promo.discount_percent / 100);
                } else if (promo.discount_amount) {
                    discount = Math.min(promo.discount_amount, subtotal);
                }

                promoCodeId = promo.id;
            }
        }

        // Calculate totals
        const tax = subtotal * 0.08; // 8% tax
        const shipping = subtotal > 100 ? 0 : 15; // Free shipping over $100
        const total = subtotal - discount + tax + shipping;

        // Create order with transaction-like operations
        const orderResult = await turboQuery(
            'INSERT INTO orders (user_id, subtotal, discount, tax, shipping, total, status, shipping_address, promo_code_id, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW()) RETURNING id, created_at',
            [
                userInfo.id,
                subtotal,
                discount,
                tax,
                shipping,
                total,
                'pending',
                JSON.stringify(orderData.shipping_address),
                promoCodeId
            ]
        );

        const order = orderResult[0];

        // Create order items and update inventory in parallel
        const operations = [
            // Insert order items
            ...orderItems.map(item =>
                turboQuery(
                    'INSERT INTO order_items (order_id, product_id, name, price, quantity, total) VALUES ($1, $2, $3, $4, $5, $6)',
                    [order.id, item.product_id, item.name, item.price, item.quantity, item.total]
                )
            ),

            // Update product stock
            ...orderItems.map(item =>
                turboQuery(
                    'UPDATE products SET stock = stock - $1 WHERE id = $2',
                    [item.quantity, item.product_id]
                )
            )
        ];

        // Update promo code usage if applicable
        if (promoCodeId) {
            operations.push(
                turboQuery(
                    'UPDATE promo_codes SET times_used = times_used + 1 WHERE id = $1',
                    [promoCodeId]
                )
            );
        }

        await Promise.all(operations);

        // Dispatch background jobs
        await Promise.all([
            turboJob('process-payment', {
                orderId: order.id,
                paymentMethodId: orderData.payment_method_id,
                amount: total
            }),
            turboJob('send-order-confirmation', {
                userId: userInfo.id,
                orderId: order.id,
                email: userInfo.email
            }),
            turboJob('notify-inventory-update', {
                items: orderItems.map(item => ({
                    product_id: item.product_id,
                    quantity_sold: item.quantity
                }))
            })
        ]);

        return {
            code: 201,
            response: {
                status: "success",
                message: "Order created successfully",
                data: {
                    order: {
                        id: order.id,
                        subtotal,
                        discount,
                        tax,
                        shipping,
                        total,
                        status: 'pending',
                        created_at: order.created_at,
                        items: orderItems
                    }
                }
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to create order"
            }
        };
    }
};
```

### Response Type Variations

TurboScript supports multiple response types beyond JSON:

```typescript
// app/routes/reports/export.ts
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { format } = event.queryParameters;

        const reportData = await turboQuery(
            'SELECT * FROM sales_report WHERE date >= $1',
            [new Date('2025-01-01')]
        );

        switch (format) {
            case 'csv':
                const csvContent = generateCSV(reportData);
                return {
                    code: 200,
                    response: csvContent,
                    type: 'text',
                    headers: {
                        'Content-Type': 'text/csv',
                        'Content-Disposition': 'attachment; filename="sales_report.csv"'
                    }
                };

            case 'html':
                const htmlReport = generateHTMLReport(reportData);
                return {
                    code: 200,
                    response: htmlReport,
                    type: 'html'
                };

            case 'markdown':
                const markdownReport = generateMarkdownReport(reportData);
                return {
                    code: 200,
                    response: markdownReport,
                    type: 'markdown'
                };

            default:
                return {
                    code: 200,
                    response: {
                        status: "success",
                        data: { report: reportData }
                    },
                    type: 'json'
                };
        }
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to generate report"
            }
        };
    }
};

function generateCSV(data: any[]): string {
    if (data.length === 0) return '';

    const headers = Object.keys(data[0]);
    const csvRows = [headers.join(',')];

    for (const row of data) {
        const values = headers.map(header => {
            const value = row[header];
            return typeof value === 'string' ? `"${value.replace(/"/g, '""')}"` : value;
        });
        csvRows.push(values.join(','));
    }

    return csvRows.join('\n');
}
```

### Webhook Handler Example

```typescript
// app/routes/webhooks/payment-status.ts
import { createHash } from 'crypto';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const signature = event.headers['x-webhook-signature'];
        const payload = JSON.stringify(event.body);

        // Verify webhook signature
        const expectedSignature = createHash('sha256')
            .update(payload + process.env.WEBHOOK_SECRET)
            .digest('hex');

        if (signature !== expectedSignature) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "Invalid webhook signature"
                }
            };
        }

        const { order_id, status, payment_id, amount } = event.body as {
            order_id: string;
            status: 'success' | 'failed' | 'pending';
            payment_id: string;
            amount: number;
        };

        // Update order status based on payment result
        const updateResult = await turboQuery(
            'UPDATE orders SET payment_status = $1, payment_id = $2, payment_updated_at = NOW() WHERE id = $3',
            [status, payment_id, order_id]
        );

        if (updateResult.rowsAffected === 0) {
            return {
                code: 404,
                response: {
                    status: "error",
                    message: "Order not found"
                }
            };
        }

        // Handle different payment statuses
        if (status === 'success') {
            // Dispatch fulfillment job
            await turboJob('process-order-fulfillment', {
                orderId: order_id
            });

            // Send confirmation email
            await turboJob('send-payment-confirmation', {
                orderId: order_id,
                paymentId: payment_id
            });
        } else if (status === 'failed') {
            // Handle payment failure
            await Promise.all([
                // Restore inventory
                turboJob('restore-order-inventory', {
                    orderId: order_id
                }),

                // Notify customer
                turboJob('send-payment-failed-notification', {
                    orderId: order_id
                })
            ]);
        }

        return {
            code: 200,
            response: {
                status: "success",
                message: "Webhook processed successfully"
            }
        };

    } catch (error) {
        // Log webhook errors for debugging
        console.error('Webhook processing error:', {
            headers: event.headers,
            body: event.body,
            error: error.message
        });

        return {
            code: 500,
            response: {
                status: "error",
                message: "Webhook processing failed"
            }
        };
    }
};
```

## Performance Optimization Techniques

### Database Query Optimization

```typescript
// Efficient pagination with cursor-based approach
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { cursor, limit = '20' } = event.queryParameters;
        const limitNum = Math.min(100, parseInt(limit, 10));

        let query = 'SELECT id, name, email, created_at FROM users WHERE active = true';
        const params: any[] = [];

        if (cursor) {
            query += ' AND created_at < $1';
            params.push(new Date(cursor));
        }

        query += ' ORDER BY created_at DESC LIMIT $' + (params.length + 1);
        params.push(limitNum);

        const users = await turboQuery(query, params);

        const nextCursor = users.length === limitNum ?
            users[users.length - 1].created_at : null;

        return {
            code: 200,
            response: {
                status: "success",
                data: { users },
                pagination: {
                    next_cursor: nextCursor,
                    has_more: users.length === limitNum
                }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to fetch users"
            }
        };
    }
};
```

---

## Navigation

**Previous:** [← Architecture Overview](guides/architecture.md)
**Next:** [Database Operations →](api/database-operations.md)

## Related Topics

- [Database Operations](api/database-operations.md)
- [Authentication Examples](api/authentication.md)
- [Best Practices Guide](guides/best-practices.md)
- [Performance Tips](guides/performance.md)
