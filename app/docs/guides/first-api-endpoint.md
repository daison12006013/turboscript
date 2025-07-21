# Your First API Endpoint

## What You'll Build

In this guide, you'll create a simple API endpoint that returns a list of users. By the end, you'll understand:

- How TurboScript file-based routing works
- How to write async database queries
- How to return JSON responses

## Prerequisites

Make sure you have TurboScript installed and running. If not, follow the [Installation Guide](getting-started.md) first.

## Step 1: Create Your First Route File

Create a new file at `app/routes/api/users.ts`:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Query the database for all users
        const users = await turboQuery('SELECT uid, name, email, created_at FROM users ORDER BY created_at DESC');

        return {
            code: 200,
            response: {
                status: "success",
                data: users
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Database error"
            }
        };
    }
};
```

## Step 2: Add the Route to Configuration

Open `turboscript.yml` (or `turboscript.dev.yml` for development) and add your endpoint:

```yaml
endpoints:
  # ...existing endpoints...

  - route: /api/users
    method: GET
    path: ./app/routes/api/users
```

## Step 3: Test Your Endpoint

With TurboScript running (`make up`), test your endpoint:

```bash
curl http://localhost:7890/api/users
```

You should see a response like:

```json
{
  "status": "success",
  "data": [
    {
      "uid": "user-123",
      "name": "John Doe",
      "email": "john@example.com",
      "created_at": "2025-01-15T10:30:00Z"
    }
  ]
}
```

## Understanding What Happened

### 🗂️ **File-Based Routing**

- File: `app/routes/api/users.ts`
- Route: `/api/users`
- Method: `GET` (configured in YAML)

TurboScript maps your file structure to URL paths automatically.

### 🔄 **Async Database Query**

```typescript
const users = await turboQuery('SELECT uid, name, email, created_at FROM users ORDER BY created_at DESC');
```

- `turboQuery()` is TurboScript's built-in database function
- It's fully async - no blocking
- SQL injection protection is automatic
- Returns an array of objects

### 📄 **Response Format**

```typescript
return {
    code: 200,           // HTTP status code
    response: {          // JSON response body
        status: "success",
        data: users
    }
};
```

## Step 4: Add a Single User Endpoint

Let's create an endpoint to get a specific user. Create `app/routes/api/users/{uid}.ts`:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Get user ID from URL path parameter
        const userUid = event.pathParameters.uid;

        if (!userUid) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "User ID is required"
                }
            };
        }

        // Query for specific user
        const users = await turboQuery('SELECT uid, name, email, created_at FROM users WHERE uid = $1', [userUid]);

        if (users.length === 0) {
            return {
                code: 404,
                response: {
                    status: "error",
                    message: "User not found"
                }
            };
        }

        return {
            code: 200,
            response: {
                status: "success",
                data: users[0]
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Database error"
            }
        };
    }
};
```

Add the route configuration:

```yaml
endpoints:
  # ...existing endpoints...

  - route: /api/users/{uid}
    method: GET
    path: ./app/routes/api/users/{uid}
```

Test it:

```bash
curl http://localhost:7890/api/users/user-123
```

## Key Concepts You Learned

### 💡 **Path Parameters**

- Use `{uid}` in your route configuration
- Access via `event.pathParameters.uid` in your code
- File name must match: `{uid}.ts`

### 🛡️ **Input Validation**

```typescript
if (!userUid) {
    return { code: 400, response: { status: "error", message: "User ID is required" } };
}
```

Always validate inputs before database queries.

### 🔍 **Database Parameters**

```typescript
await turboQuery('SELECT * FROM users WHERE uid = $1', [userUid]);
```

- Use `$1, $2, $3...` for parameters
- Pass values in array: `[userUid, email, name]`
- **Never** concatenate strings - this prevents SQL injection

### 📊 **Error Handling**

```typescript
try {
    // Database operations
} catch (error) {
    return {
        code: 500,
        response: {
            status: "error",
            message: error instanceof Error ? error.message : "Database error"
        }
    };
}
```

Always wrap database operations in try/catch blocks.

## Next Steps

Now that you understand the basics:

1. **Add More Methods**: Learn to create [POST, PUT, DELETE endpoints](creating-routes.md)
2. **Add Authentication**: Secure your endpoints with [JWT auth](authentication-basics.md)
3. **Work with Complex Data**: Learn [advanced database operations](database-basics.md)
4. **Build a Complete App**: Follow the [Todo App Tutorial](tutorial-todo-app.md)

## Common Issues

### ❌ **Route Not Found**

- Check your YAML configuration
- Ensure file path matches the `path` in config
- Restart TurboScript after YAML changes

### ❌ **Database Connection Error**

- Check your database configuration in `turboscript.yml`
- Ensure PostgreSQL is running
- Verify connection credentials

### ❌ **TypeScript Errors**

- Import types: `import { Event, TurboScriptResponse } from "../../global"`
- Use proper return types
- Check for typos in function names

---

**Congratulations! 🎉** You've created your first TurboScript API endpoint. Ready to build something more complex? Try the [Todo App Tutorial](tutorial-todo-app.md).
