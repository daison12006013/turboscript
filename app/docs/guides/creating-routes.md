# Creating Route Handlers

This guide explains how to create and organize route handlers in TurboScript using TypeScript.

## Basic Route Structure

TurboScript routes are TypeScript files that export an async `handle` function:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    // Your route logic here
    return {
        code: 200,
        response: {
            status: "success",
            message: "Hello, World!"
        }
    };
};
```

## Route Configuration

Routes are defined in `turboscript.yml` under the `endpoints` section:

```yaml
endpoints:
  - path: "/api/users"
    method: "GET"
    file: "routes/api/users/list.ts"

  - path: "/api/users/:id"
    method: "GET"
    file: "routes/api/users/get.ts"
```

## Route Parameters

Access path parameters through the `event.pathParameters` object:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const userId = event.pathParameters.id;

    const user = await turboQuery('SELECT * FROM users WHERE id = $1', [userId]);

    return {
        code: 200,
        response: {
            status: "success",
            data: user[0]
        }
    };
};
```

## Request Body Handling

Access request body data through the `event.body` object:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { name, email } = event.body;

    // Validate input
    if (!name || !email) {
        return {
            code: 400,
            response: {
                status: "error",
                message: "Name and email are required"
            }
        };
    }

    // Insert user
    const result = await turboQuery(
        'INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id',
        [name, email]
    );

    return {
        code: 201,
        response: {
            status: "success",
            data: { id: result[0].id, name, email }
        }
    };
};
```

## Query Parameters

Access query parameters through `event.queryStringParameters`:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const page = parseInt(event.queryStringParameters?.page || '1');
    const limit = parseInt(event.queryStringParameters?.limit || '10');
    const offset = (page - 1) * limit;

    const users = await turboQuery(
        'SELECT * FROM users LIMIT $1 OFFSET $2',
        [limit, offset]
    );

    return {
        code: 200,
        response: {
            status: "success",
            data: users,
            pagination: { page, limit, total: users.length }
        }
    };
};
```

## Headers Access

Access request headers through `event.headers`:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const userAgent = event.headers['user-agent'];
    const contentType = event.headers['content-type'];

    return {
        code: 200,
        response: {
            status: "success",
            data: { userAgent, contentType }
        }
    };
};
```

## Error Handling

Always implement proper error handling in your routes:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const users = await turboQuery('SELECT * FROM users');

        return {
            code: 200,
            response: {
                status: "success",
                data: users
            }
        };
    } catch (error) {
        console.error('Database error:', error);

        return {
            code: 500,
            response: {
                status: "error",
                message: "Internal server error"
            }
        };
    }
};
```

## File Organization

Organize your routes logically in the `app/routes/` directory:

```text
app/routes/
├── index.ts              # Homepage
├── 404.ts               # Not found handler
├── api/
│   ├── users/
│   │   ├── list.ts      # GET /api/users
│   │   ├── get.ts       # GET /api/users/:id
│   │   ├── create.ts    # POST /api/users
│   │   └── update.ts    # PUT /api/users/:id
│   └── auth/
│       ├── login.ts     # POST /api/auth/login
│       └── logout.ts    # POST /api/auth/logout
└── static/
    └── assets.ts        # Static file serving
```

## Next Steps

- Learn about [database operations](database-basics.md) for working with data
- Implement [authentication](authentication-basics.md) for secure endpoints
- Explore [advanced routing patterns](../api/wildcard-routing.md) for complex URLs

## Related Documentation

- [Route Handlers Deep Dive](../api/route-handlers.md)
- [Async Database Operations](../api/database-operations.md)
- [Authentication & Security](../api/authentication.md)
