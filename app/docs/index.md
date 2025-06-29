# TurboScript Documentation

Welcome to the TurboScript documentation. TurboScript is a hybrid web framework that combines TypeScript for business logic and Go for runtime execution, providing a unique development experience with powerful async database operations.

## What is TurboScript?

TurboScript is a modern web framework that allows you to write API logic in TypeScript while leveraging Go's performance for runtime execution. Key features include:

- **Async Database Operations**: Direct `turboQuery()` with Promise.all support
- **Hot Reloading**: Instant TypeScript and Go code updates
- **Type Safety**: Full TypeScript support with framework-specific types
- **Security Built-in**: JWT authentication, input validation, and SQL injection protection
- **High Performance**: Go runtime with FastHTTP and efficient database connections
- **Plugin System**: Extensible architecture with built-in plugins for file uploads and more

## Quick Start

Get up and running in minutes:

```bash
# Clone and setup
git clone git@github.com:daison12006013/turboscript.git
cd turboscript
npm install && go mod download

# Start development environment
docker-compose -f docker-compose.dev.yml up -d
make dev
```

Your API will be running at `http://localhost:7890` with hot reloading enabled.

## Documentation Structure

### 🚀 Getting Started

Start here if you're new to TurboScript:

- [Installation & Setup](guides/getting-started.md) - Get TurboScript running locally
- [Configuration](guides/configuration.md) - Configure your application
- [Development Workflow](guides/development.md) - Daily development practices

### 🏗️ Architecture & Core Concepts

Understand how TurboScript works:

- [Architecture Overview](guides/architecture.md) - System design and components
- [Route Handlers](api/route-handlers.md) - Creating API endpoints
- [Database Operations](api/database-operations.md) - Async database queries

### 📋 API Reference

Detailed API documentation:

- [Route Handler API](api/route-handlers.md) - Complete handler reference
- [Wildcard Routing](api/wildcard-routing.md) - Dynamic file-based routing with /* patterns
- [Database API](api/database-operations.md) - TurboQuery usage
- [Authentication API](api/authentication.md) - JWT and auth helpers
- [Background Jobs](api/background-jobs.md) - Async job processing

### 📖 Guides & Best Practices

Level up your TurboScript skills:

- [Best Practices](guides/best-practices.md) - Recommended patterns
- [Performance Optimization](guides/performance.md) - Speed and efficiency tips
- [Security Guidelines](guides/security.md) - Keep your app secure
- [Deployment Guide](guides/deployment.md) - Production deployment
- [Troubleshooting](guides/troubleshooting.md) - Common issues and solutions

## Key Features

### Async Database Operations

```typescript
// Parallel database queries for maximum performance
const [user, orders, analytics] = await Promise.all([
    turboQuery('SELECT * FROM users WHERE id = $1', [userId]),
    turboQuery('SELECT * FROM orders WHERE user_id = $1', [userId]),
    turboQuery('SELECT COUNT(*) FROM page_views WHERE user_id = $1', [userId])
]);
```

### Built-in Authentication

```typescript
import { verifyAuth, createAuthErrorResponse } from '../utils/auth';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    // Check authentication directly in handle function
    const userPayload = verifyAuth(event);
    if (!userPayload) {
        return createAuthErrorResponse("Access token required", event);
    }

    // Use authenticated user's data directly
    const userUid = userPayload.uid;

    // Your protected endpoint logic here
    return {
        code: 200,
        response: {
            status: "success",
            message: `Hello, ${userPayload.name}!`,
            user_id: userUid
        }
    };
};
```

### Type-Safe Route Handlers

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const userId = event.pathParameters.uid;
    const userData = await turboQuery('SELECT * FROM users WHERE uid = $1', [userId]);

    return {
        code: 200,
        response: { status: "success", data: userData[0] }
    };
};
```

## Community & Support

- [GitHub Repository](https://github.com/daison12006013/turboscript)
- [Report Issues](https://github.com/daison12006013/turboscript/issues)
- [Contributing Guide](../CONTRIBUTING.md)
- [License](../LICENSE)

---

**Ready to build something amazing?** Start with the [Getting Started Guide](guides/getting-started.md) or jump into [API Examples](api/examples.md).

---

**Next:** [Getting Started →](guides/getting-started.md)
