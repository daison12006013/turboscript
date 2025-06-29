# Building Your First App: Todo List

In this tutorial, you'll build a complete Todo application using TurboScript. By the end, you'll understand how to:

- Create multiple API endpoints
- Work with databases
- Handle authentication
- Manage file uploads
- Build a complete application

## What We'll Build

A Todo app with these features:

- ✅ Create, read, update, delete todos
- 🔐 User authentication
- 📎 File attachments for todos
- 📝 Rich text descriptions
- 🏷️ Categories and tags

## Prerequisites

- TurboScript installed and running
- PostgreSQL database
- Basic understanding of TypeScript and SQL

## Step 1: Database Setup

First, let's create the database tables. Create `init.sql`:

```sql
-- Users table
CREATE TABLE users (
    uid VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Todo categories
CREATE TABLE todo_categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    color VARCHAR(7) DEFAULT '#007bff',
    user_id VARCHAR(50) REFERENCES users(uid) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Todos table
CREATE TABLE todos (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    completed BOOLEAN DEFAULT FALSE,
    priority INTEGER DEFAULT 1, -- 1=low, 2=medium, 3=high
    category_id INTEGER REFERENCES todo_categories(id),
    user_id VARCHAR(50) REFERENCES users(uid) ON DELETE CASCADE,
    due_date TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Todo attachments
CREATE TABLE todo_attachments (
    id SERIAL PRIMARY KEY,
    todo_id INTEGER REFERENCES todos(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_url VARCHAR(500) NOT NULL,
    file_size INTEGER NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Todo tags (many-to-many)
CREATE TABLE todo_tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    user_id VARCHAR(50) REFERENCES users(uid) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, user_id)
);

CREATE TABLE todo_tag_assignments (
    todo_id INTEGER REFERENCES todos(id) ON DELETE CASCADE,
    tag_id INTEGER REFERENCES todo_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (todo_id, tag_id)
);

-- Sample data
INSERT INTO users (uid, name, email, password_hash) VALUES
('user-123', 'John Doe', 'john@example.com', '$2b$10$hash...'), -- Replace with real hash
('user-456', 'Jane Smith', 'jane@example.com', '$2b$10$hash...');

INSERT INTO todo_categories (name, color, user_id) VALUES
('Work', '#dc3545', 'user-123'),
('Personal', '#28a745', 'user-123'),
('Shopping', '#ffc107', 'user-123');

INSERT INTO todos (title, description, category_id, user_id, priority) VALUES
('Complete project proposal', 'Write the quarterly project proposal for the new client', 1, 'user-123', 3),
('Buy groceries', 'Milk, bread, eggs, apples', 3, 'user-123', 1),
('Schedule dentist appointment', 'Annual checkup', 2, 'user-123', 2);
```

## Step 2: Authentication Routes

### Login Route

Create `app/routes/auth/login.ts`:

```typescript
export const handle = async (_event: Event): Promise<TurboScriptResponse> => {
    const { verifyPassword } = require("bcryptjs");

    try {
        const requestData = data as { email: string; password: string };

        if (!requestData.email || !requestData.password) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Email and password are required"
                }
            };
        }

        // Find user
        const users = await turboQuery(
            'SELECT uid, name, email, password_hash FROM users WHERE email = $1',
            [requestData.email]
        );

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
        const isValid = await verifyPassword(requestData.password, user.password_hash);
        if (!isValid) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "Invalid credentials"
                }
            };
        }

        // Generate JWT token
        const accessToken = generateAccessToken({
            uid: user.uid,
            email: user.email,
            name: user.name
        });

        const refreshToken = generateRefreshToken({
            uid: user.uid
        });

        return {
            code: 200,
            response: {
                status: "success",
                message: "Login successful",
                data: {
                    user: {
                        uid: user.uid,
                        name: user.name,
                        email: user.email
                    },
                    tokens: {
                        access_token: accessToken,
                        refresh_token: refreshToken
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

## Step 3: Todo CRUD Operations

### List Todos

Create `app/routes/todos/list.ts`:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const user = event.body.__user;
        const query = event.queryParameters || {};

        // Build dynamic query
        let sql = `
            SELECT
                t.id,
                t.title,
                t.description,
                t.completed,
                t.priority,
                t.due_date,
                t.created_at,
                t.updated_at,
                c.name as category_name,
                c.color as category_color,
                COALESCE(
                    JSON_AGG(
                        JSON_BUILD_OBJECT('id', tag.id, 'name', tag.name)
                    ) FILTER (WHERE tag.id IS NOT NULL),
                    '[]'
                ) as tags
            FROM todos t
            LEFT JOIN todo_categories c ON t.category_id = c.id
            LEFT JOIN todo_tag_assignments tta ON t.id = tta.todo_id
            LEFT JOIN todo_tags tag ON tta.tag_id = tag.id
            WHERE t.user_id = $1
        `;

        const params = [user.uid];
        let paramIndex = 2;

        // Filter by completion status
        if (query.completed !== undefined) {
            sql += ` AND t.completed = $${paramIndex}`;
            params.push(query.completed === 'true');
            paramIndex++;
        }

        // Filter by category
        if (query.category) {
            sql += ` AND c.name = $${paramIndex}`;
            params.push(query.category);
            paramIndex++;
        }

        // Filter by priority
        if (query.priority) {
            sql += ` AND t.priority = $${paramIndex}`;
            params.push(parseInt(query.priority));
            paramIndex++;
        }

        sql += `
            GROUP BY t.id, c.name, c.color
            ORDER BY
                CASE WHEN t.completed THEN 1 ELSE 0 END,
                t.priority DESC,
                t.created_at DESC
        `;

        // Add pagination
        const page = parseInt(query.page || '1');
        const limit = parseInt(query.limit || '20');
        const offset = (page - 1) * limit;

        sql += ` LIMIT $${paramIndex} OFFSET $${paramIndex + 1}`;
        params.push(limit, offset);

        const todos = await turboQuery(sql, params);

        // Get total count
        const countResult = await turboQuery(
            'SELECT COUNT(*) as total FROM todos WHERE user_id = $1',
            [user.uid]
        );

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    todos,
                    pagination: {
                        page,
                        limit,
                        total: parseInt(countResult[0].total),
                        totalPages: Math.ceil(countResult[0].total / limit)
                    }
                }
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to fetch todos"
            }
        };
    }
};
```

### Create Todo

Create `app/routes/todos/create.ts`:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const user = event.body.__user;
        const requestData = data as {
            title: string;
            description?: string;
            category_id?: number;
            priority?: number;
            due_date?: string;
            tags?: string[];
            attachment?: { file: string; filename: string };
        };

        // Validate required fields
        if (!requestData.title || requestData.title.trim().length === 0) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Title is required"
                }
            };
        }

        // Create the todo
        const todoResult = await turboQuery(
            `INSERT INTO todos (title, description, category_id, priority, due_date, user_id)
             VALUES ($1, $2, $3, $4, $5, $6)
             RETURNING id, title, description, priority, due_date, created_at`,
            [
                requestData.title.trim(),
                requestData.description || null,
                requestData.category_id || null,
                requestData.priority || 1,
                requestData.due_date || null,
                user.uid
            ]
        );

        const todo = todoResult[0];

        // Handle file attachment if provided
        let attachment = null;
        if (requestData.attachment) {
            const fileUpload = require('fileupload');
            const { saveBase64 } = fileUpload;

            try {
                const fileInfo = await saveBase64(
                    requestData.attachment.file,
                    requestData.attachment.filename,
                    {
                        directory: `todos/${user.uid}`,
                        generateHash: true
                    }
                );

                // Save attachment info to database
                const attachmentResult = await turboQuery(
                    `INSERT INTO todo_attachments
                     (todo_id, filename, original_name, file_path, file_url, file_size, mime_type)
                     VALUES ($1, $2, $3, $4, $5, $6, $7)
                     RETURNING *`,
                    [
                        todo.id,
                        fileInfo.filename,
                        fileInfo.originalName,
                        fileInfo.path,
                        fileInfo.url,
                        fileInfo.size,
                        fileInfo.mimeType
                    ]
                );

                attachment = attachmentResult[0];
            } catch (uploadError) {
                // Log error but don't fail the todo creation
                console.error("File upload failed:", uploadError);
            }
        }

        // Handle tags if provided
        const tags = [];
        if (requestData.tags && requestData.tags.length > 0) {
            for (const tagName of requestData.tags) {
                // Get or create tag
                let tagResult = await turboQuery(
                    'SELECT id FROM todo_tags WHERE name = $1 AND user_id = $2',
                    [tagName, user.uid]
                );

                if (tagResult.length === 0) {
                    tagResult = await turboQuery(
                        'INSERT INTO todo_tags (name, user_id) VALUES ($1, $2) RETURNING id',
                        [tagName, user.uid]
                    );
                }

                const tagId = tagResult[0].id;

                // Assign tag to todo
                await turboQuery(
                    'INSERT INTO todo_tag_assignments (todo_id, tag_id) VALUES ($1, $2)',
                    [todo.id, tagId]
                );

                tags.push({ id: tagId, name: tagName });
            }
        }

        return {
            code: 201,
            response: {
                status: "success",
                message: "Todo created successfully",
                data: {
                    ...todo,
                    tags,
                    attachment
                }
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to create todo"
            }
        };
    }
};
```

## Step 4: Category Management

Create `app/routes/categories/list.ts`:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const user = event.body.__user;

        const categories = await turboQuery(
            `SELECT
                c.id,
                c.name,
                c.color,
                c.created_at,
                COUNT(t.id) as todo_count
             FROM todo_categories c
             LEFT JOIN todos t ON c.id = t.category_id AND t.user_id = c.user_id
             WHERE c.user_id = $1
             GROUP BY c.id, c.name, c.color, c.created_at
             ORDER BY c.name`,
            [user.uid]
        );

        return {
            code: 200,
            response: {
                status: "success",
                data: { categories }
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to fetch categories"
            }
        };
    }
};
```

## Step 5: Configuration

Add all routes to your `turboscript.yml`:

```yaml
endpoints:
  # Authentication
  - route: /auth/login
    method: POST
    path: ./app/routes/auth/login

  # Todos
  - route: /todos
    method: GET
    path: ./app/routes/todos/list

  - route: /todos
    method: POST
    path: ./app/routes/todos/create

  - route: /todos/{id}
    method: GET
    path: ./app/routes/todos/get

  - route: /todos/{id}
    method: PUT
    path: ./app/routes/todos/update

  - route: /todos/{id}
    method: DELETE
    path: ./app/routes/todos/delete

  # Categories
  - route: /categories
    method: GET
    path: ./app/routes/categories/list

  - route: /categories
    method: POST
    path: ./app/routes/categories/create
```

## Step 6: Testing Your API

### 1. Login

```bash
curl -X POST http://localhost:7890/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "password123"
  }'
```

### 2. Create a Todo

```bash
curl -X POST http://localhost:7890/todos \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "title": "Learn TurboScript",
    "description": "Complete the tutorial and build a todo app",
    "priority": 3,
    "tags": ["learning", "turboscript"]
  }'
```

### 3. List Todos

```bash
curl -X GET "http://localhost:7890/todos?completed=false&page=1&limit=10" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## Step 7: Next Steps

Congratulations! You've built a complete Todo API. Here's what you can add next:

### Advanced Features

1. **Real-time updates** with WebSockets
2. **Todo sharing** between users
3. **Recurring todos** with scheduling
4. **Search functionality** with full-text search
5. **Email notifications** for due dates
6. **File preview** for attachments
7. **Bulk operations** (complete all, delete completed)

### Frontend Integration

1. **React/Vue.js frontend** consuming your API
2. **Mobile app** with React Native or Flutter
3. **Desktop app** with Electron

### DevOps & Production

1. **Docker containerization**
2. **CI/CD pipelines**
3. **Monitoring and logging**
4. **Database backups**
5. **Load balancing**

## Key Concepts You Learned

✅ **File-based routing** - Organizing API endpoints
✅ **Database operations** - Complex queries with joins
✅ **Authentication** - JWT tokens and middleware
✅ **File uploads** - Binary data handling
✅ **Error handling** - Graceful error responses
✅ **Data validation** - Input sanitization
✅ **Pagination** - Efficient data loading
✅ **Relationships** - Foreign keys and associations

## Troubleshooting

### Common Issues

**Database connection errors:**

- Check your database configuration
- Ensure PostgreSQL is running
- Verify connection credentials

**Authentication failures:**

- Check JWT secret configuration
- Verify token format and expiration
- Ensure authorization header is correct

**File upload errors:**

- Check upload directory permissions
- Verify file size and type restrictions
- Ensure plugin is enabled and configured

---

**Next:** [Performance Optimization](../guides/performance.md) | [Security Best Practices](../guides/security.md)
