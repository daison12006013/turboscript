# Development Guide

## Development Environment Setup

### Prerequisites

- Node.js v18+
- Go 1.21+
- Docker and Docker Compose
- IDE with TypeScript and Go support (VS Code recommended)

### Initial Setup

1. Clone and setup the development environment:

   ```bash
   git clone https://github.com/daison12006013/turboscript.git
   cd turboscript
   npm install
   go mod download
   ```

2. Start the development containers:

   ```bash
   docker-compose -f docker-compose.dev.yml up -d
   ```

3. Copy environment variables:

   ```bash
   cp .env.example .env
   # Edit .env with your local configuration
   ```

## Development Workflow

### Running the Application

The development server runs on port 7890 with hot-reloading enabled:

```bash
make up
```

This command starts the development server with:

- **Hot reloading**: Automatic TypeScript compilation and Go restart
- **Debug logging**: Detailed logging for development
- **CORS enabled**: Allows local frontend development
- **PostgreSQL**: Database container with sample data

### Working on the React Frontend

When developing the React frontend (located in `app/hybrid/`), you must run the following command in a separate terminal to enable live rebuilding of React components:

```bash
npm run watch
```

This will watch for changes in the React source files and automatically rebuild the frontend assets. Make sure this is running alongside your backend development server for a smooth development experience.

### Viewing Logs

Monitor application logs in real-time:

```bash
# View last 20 lines and follow
docker logs turboscript-app-dev-1 --tail=20 -f

# View specific number of lines
docker logs turboscript-app-dev-1 --tail=50
```

### Database Operations

#### Connecting to PostgreSQL

```bash
# Connect to development database
docker exec -it turboscript-postgres-1 psql -U postgres -d turboscript

# Run SQL commands
\dt                           # List tables
SELECT * FROM users;          # Query users
\q                           # Exit
```

#### Migrations and Schema

```bash
# Run initial migrations (if available)
docker exec -it turboscript-postgres-1 psql -U postgres -d turboscript -f /docker-entrypoint-initdb.d/init.sql

# Create a new table
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Creating New Features

#### 1. Define the Endpoint

Add your endpoint to `turboscript.dev.yml`:

```yaml
endpoints:
  - route: /api/products
    method: GET
    path: ./app/routes/products/list.ts

  - route: /api/products/{id}
    method: GET
    path: ./app/routes/products/get-by-id.ts

  - route: /api/products
    method: POST
    path: ./app/routes/products/create.ts
```

#### 2. Create Route Handler

Create the TypeScript route file:

```typescript
// app/routes/products/list.ts
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { page = '1', limit = '20' } = event.queryParameters;

        const products = await turboQuery(
            'SELECT id, name, price, created_at FROM products ORDER BY created_at DESC LIMIT $1 OFFSET $2',
            [parseInt(limit), (parseInt(page) - 1) * parseInt(limit)]
        );

        return {
            code: 200,
            response: {
                status: "success",
                data: { products }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to fetch products"
            }
        };
    }
};
```

#### 3. Add Table to Configuration

Update `turboscript.dev.yml` to allow access to the new table:

```yaml
database:
  allowed_tables:
    - users
    - sessions
    - products  # Add your new table
```

#### 4. Test the Endpoint

```bash
# Test with curl
curl http://localhost:7890/api/products

# Test with query parameters
curl "http://localhost:7890/api/products?page=1&limit=10"
```

### Authentication Development

#### Testing Authentication

```bash
# 1. Login to get access token
curl -X POST http://localhost:7890/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password"}'

# 2. Use token in protected endpoint
curl -X POST http://localhost:7890/api/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{"name": "New Product", "price": 29.99}'
```

### Background Jobs Development

#### Creating a Job Handler

```typescript
// app/queue/process-order.ts
export const handle = async (data: unknown): Promise<void> => {
    try {
        const { orderId, userId } = data as { orderId: string; userId: string };

        // Process the order
        await turboQuery(
            'UPDATE orders SET status = $1, processed_at = NOW() WHERE id = $2',
            ['processing', orderId]
        );

        // Send notification email
        await turboEmail({
            to: 'user@example.com',
            subject: 'Order Processing Started',
            text: `Your order #${orderId} is now being processed.`
        });

        console.log(`Order ${orderId} processed successfully`);
    } catch (error) {
        console.error('Failed to process order:', error);
        throw error; // Re-throw to mark job as failed
    }
};
```

#### Dispatching Jobs

```typescript
// In your route handler
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Create order
        const order = await turboQuery(
            'INSERT INTO orders (user_id, total) VALUES ($1, $2) RETURNING id',
            [userId, total]
        );

        // Dispatch background job
        await turboJob('process-order', {
            orderId: order[0].id,
            userId: userId
        });

        return {
            code: 201,
            response: {
                status: "success",
                data: { order_id: order[0].id }
            }
        };
    } catch (error) {
        // Error handling...
    }
};
```

### Email Development

#### Setting up Email Configuration

```yaml
# turboscript.dev.yml
email:
  driver: "log"  # Use "log" driver for development
  from_address: "dev@localhost"
  from_name: "TurboScript Dev"
```

#### Sending Emails

```typescript
// app/queue/send-welcome-email.ts
export const handle = async (data: unknown): Promise<void> => {
    try {
        const { email, name } = data as { email: string; name: string };

        await turboEmail({
            to: email,
            subject: 'Welcome to TurboScript!',
            html: `
                <h1>Welcome ${name}!</h1>
                <p>Thank you for joining our platform.</p>
                <p>Best regards,<br>The TurboScript Team</p>
            `
        });

        console.log(`Welcome email sent to ${email}`);
    } catch (error) {
        console.error('Failed to send welcome email:', error);
        throw error;
    }
};
```

### TypeScript Configuration

#### Global Types

The `app/global.d.ts` file defines global types available in all route handlers:

```typescript
// app/global.d.ts
declare global {
    interface Event {
        headers: Record<string, string>;
        queryParameters: Record<string, string>;
        pathParameters: Record<string, string>;
        body: Record<string, unknown>;
        env: Record<string, string>;
    }

    interface User {
        id: string;
        uid: string;
        name: string;
        email: string;
        created_at: string;
        updated_at: string;
    }

    function turboQuery(query: string, params?: unknown[]): Promise<unknown[]>;
    function turboJob(jobName: string, data: unknown): Promise<void>;
    function turboEmail(config: EmailConfig): Promise<void>;
}
```

#### Adding Custom Types

```typescript
// app/types/products.d.ts
declare global {
    interface Product {
        id: string;
        name: string;
        price: number;
        category_id: string;
        created_at: string;
    }

    interface CreateProductRequest {
        name: string;
        price: number;
        category_id: string;
    }
}

export {}; // Make this a module
```

### Debugging and Troubleshooting

#### Common Development Issues

**Port Already in Use**:

```bash
# Find process using port 7890
lsof -i :7890

# Kill the process
kill -9 PID
```

**Database Connection Issues**:

```bash
# Check if PostgreSQL container is running
docker ps | grep postgres

# Restart database container
docker-compose -f docker-compose.dev.yml restart postgres

# Check container logs
docker logs turboscript-postgres-1
```

**TypeScript Compilation Errors**:

```bash
# Check TypeScript compilation
npx tsc --noEmit

# Check for syntax errors in routes
find app/routes -name "*.ts" -exec npx tsc --noEmit {} \;
```

#### Debugging Tools

**Database Queries**:

```bash
# Enable query logging in PostgreSQL
docker exec -it turboscript-postgres-1 bash
echo "log_statement = 'all'" >> /var/lib/postgresql/data/postgresql.conf
# Restart container
```

**Request Debugging**:

```typescript
// Add debugging to route handlers
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    console.log('Debug - Event:', JSON.stringify(event, null, 2));
    console.log('Debug - Data:', JSON.stringify(data, null, 2));

    // Your handler logic...
};
```

### Code Quality and Testing

#### Linting

```bash
# Run Go linter
./golangci-lint run

# Fix Go formatting
go fmt ./...

# Run TypeScript checks
npx tsc --noEmit
```

#### Testing Endpoints

```bash
# Install testing tools
npm install -g httpie

# Test endpoints with HTTPie
http GET localhost:7890/api/products
http POST localhost:7890/api/products name="Test Product" price:=29.99
```

#### Performance Testing

```bash
# Install Apache Bench
brew install httpd  # macOS

# Basic load test
ab -n 1000 -c 10 http://localhost:7890/

# Test specific endpoint
ab -n 100 -c 5 http://localhost:7890/api/products
```

### Development Best Practices

#### File Organization

```text
app/
├── routes/
│   ├── auth/
│   │   ├── login.ts
│   │   ├── logout.ts
│   │   └── refresh.ts
│   ├── users/
│   │   ├── create.ts
│   │   ├── list.ts
│   │   └── update.ts
│   └── index.ts
├── utils/
│   ├── auth.ts
│   ├── password.ts
│   └── validation.ts
├── queue/
│   ├── send-email.ts
│   └── process-payment.ts
└── global.d.ts
```

#### Environment Variables

```bash
# .env for development
JWT_ACCESS_SECRET=dev_access_secret_very_long_and_secure
JWT_REFRESH_SECRET=dev_refresh_secret_also_very_long_and_secure
DB_HOST=localhost
DB_PORT=5432
DB_NAME=turboscript
DB_USER=postgres
DB_PASSWORD=postgres
SMTP_HOST=localhost
SMTP_PORT=1025
```

#### Git Workflow

```bash
# Create feature branch
git checkout -b feature/add-products-api

# Make changes and commit
git add .
git commit -m "Add products API endpoints"

# Run quality checks before push
./golangci-lint run
make test

# Push and create PR
git push origin feature/add-products-api
```

---

## Navigation

**Previous:** [← Configuration](guides/configuration.md)
**Next:** [Architecture Overview →](guides/architecture.md)

## Related Topics

- [Getting Started Guide](guides/getting-started.md)
- [Best Practices](guides/best-practices.md)
- [API Examples](api/examples.md)
- [Troubleshooting](guides/troubleshooting.md)
