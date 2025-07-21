# Getting Started with TurboScript

Get TurboScript running in under 5 minutes! This guide provides the fastest way to set up a development environment and create your first API endpoint.

## Prerequisites

Before you begin, ensure you have the following installed:

- **Node.js** (v18 or higher)
- **Go** (1.23 or higher)
- **Docker and Docker Compose**
- **Git**

## Quick Installation

### 1. Clone and Setup

```bash
# Clone the repository
git clone https://github.com/daison12006013/turboscript.git
cd turboscript

# Install dependencies
npm install && go mod download
```

### 2. Start Development Environment

```bash
# Start PostgreSQL and other services
docker-compose -f docker-compose.dev.yml up -d

# Start the development server with hot reloading
make up
```

Your API will be running at `http://localhost:7890` with hot reloading enabled!

## Test Your Installation

Open your browser or use curl to test the default endpoint:

```bash
curl http://localhost:7890/
```

You should see a welcome response:

```json
{
  "status": "success",
  "message": "Welcome to TurboScript!",
  "meta": {
    "timestamp": "2025-07-04T10:00:00.000Z",
    "queryParameters": {},
    "pathParameters": {}
  }
}
```

## Create Your First API Endpoint

Let's create a simple "Hello World" endpoint:

### 1. Add Route Configuration

Edit `turboscript.yml` and add a new endpoint:

```yaml
endpoints:
  # ... existing endpoints ...

  - route: /hello
    method: GET
    path: app/routes/hello
```

### 2. Create the Handler

Create `app/routes/hello.ts`:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const name = event.queryParameters.name || 'World';

    return {
        code: 200,
        response: {
            status: "success",
            message: `Hello, ${name}!`,
            timestamp: new Date().toISOString()
        }
    };
};
```

### 3. Test Your Endpoint

The server will automatically reload. Test your new endpoint:

```bash
curl "http://localhost:7890/hello?name=Developer"
```

Expected response:

```json
{
  "status": "success",
  "message": "Hello, Developer!",
  "timestamp": "2025-07-04T10:00:00.000Z"
}
```

## Add Database Operations

Let's create an endpoint that uses the database:

### 1. Create a User Endpoint

Create `app/routes/users/list.ts`:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Get users from database with async turboQuery
        const users = await turboQuery(
            'SELECT id, name, email, created_at FROM users ORDER BY created_at DESC LIMIT 10'
        );

        return {
            code: 200,
            response: {
                status: "success",
                data: { users },
                count: users.length
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

### 2. Add Route Configuration

Add to `turboscript.yml`:

```yaml
endpoints:
  # ... existing endpoints ...

  - route: /users/list
    method: GET
    path: app/routes/users/list
```

### 3. Test Database Endpoint

```bash
curl http://localhost:7890/users/list
```

## Development Commands

Essential commands for daily development:

```bash
# Start development server with hot reloading
make up

# View application logs
docker logs turboscript-app-dev-1 --tail=20

# Run tests
make test

# Run linting (for Go code changes)
./golangci-lint run

# Find failing tests
make find-fail

# Build production distribution
make build-dist
```

## Project Structure Overview

```text
app/                    # TypeScript Application Layer
├── routes/             # API endpoint handlers
│   ├── index.ts        # Main route handler
│   ├── auth/           # Authentication endpoints
│   └── users/          # User management endpoints
├── utils/              # Shared utilities
│   ├── auth.ts         # Authentication helpers
│   ├── password.ts     # Password utilities
│   └── meta.ts         # Response metadata
├── queue/              # Background job handlers
└── global.d.ts         # Global type definitions

internal/               # Go Runtime Engine
├── server/             # FastHTTP web server
├── tsengine/          # TypeScript execution engine
├── dbexecutor/        # Database query executor
└── config/            # Configuration management

turboscript.yml         # Application configuration
```

## Next Steps

Now that you have TurboScript running, explore these topics:

1. **[Architecture Overview](guides/architecture.md)** - Understand how TurboScript works
2. **[Route Handlers](api/route-handlers.md)** - Learn about creating API endpoints
3. **[Database Operations](api/database-operations.md)** - Master async database queries
4. **[Authentication](api/authentication.md)** - Secure your endpoints
5. **[Background Jobs](api/background-jobs.md)** - Handle async tasks
6. **[Best Practices](guides/best-practices.md)** - Follow recommended patterns

## Troubleshooting

### Common Issues

**Port 7890 already in use:**

```bash
# Check what's using the port
lsof -i :7890

# Kill the process or change the port in turboscript.yml
```

**Database connection errors:**

```bash
# Ensure PostgreSQL is running
docker-compose -f docker-compose.dev.yml ps

# Restart the database service
docker-compose -f docker-compose.dev.yml restart postgres
```

**TypeScript compilation errors:**

- Check your TypeScript syntax in the route handlers
- Ensure all imports are correct
- Review the console output for detailed error messages

### Getting Help

- **[Troubleshooting Guide](guides/troubleshooting.md)** - Detailed problem solutions
- **[GitHub Issues](https://github.com/daison12006013/turboscript/issues)** - Report bugs or ask questions
- **[Contributing Guidelines](https://github.com/daison12006013/turboscript/blob/main/CONTRIBUTING.md)** - Contribute to the project
