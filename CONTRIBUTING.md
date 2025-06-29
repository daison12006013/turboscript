# Contributing to TurboScript

Thank you for your interest in contributing to TurboScript! This document provides guidelines on how to contribute to the project and an overview of the codebase architecture.

## Table of Contents

- [Getting Started](#getting-started)
- [Codebase Overview](#codebase-overview)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [License](#license)

## Getting Started

TurboScript is a unique web framework that combines TypeScript for business logic and query definition with Go for runtime execution and database operations. The project uses JavaScript VM (goja) to execute TypeScript code at runtime, providing a powerful and flexible development experience.

### Prerequisites

- **Go** 1.23.0 or later
- **Node.js** 18+ and npm/yarn
- **Docker** and Docker Compose
- **PostgreSQL** (handled via Docker)
- **Git**

### Quick Start

1. **Fork and Clone**

   ```bash
   git clone https://github.com/daison12006013/turboscript.git
   cd turboscript
   ```

2. **Install Dependencies**

   ```bash
   # Install Go dependencies
   go mod download

   # Install Node.js dependencies
   npm install
   ```

3. **Start Development Environment**

   ```bash
   # Start database and application
   make up

   # View logs
   make logs
   ```

4. **Verify Setup**

   ```bash
   # Test an endpoint
   curl -X GET http://localhost:7890/users
   ```

## Codebase Overview

TurboScript follows a unique hybrid architecture where TypeScript defines the business logic and Go handles the execution runtime.

### Project Structure

```text
turboscript/
├── app/                    # TypeScript application logic
│   ├── global.d.ts        # Global type definitions
│   ├── routes/            # API endpoint handlers
│   │   ├── auth/          # Authentication endpoints
│   │   └── users/         # User management endpoints
│   └── utils/             # Shared utilities
├── internal/              # Go runtime and execution engine
│   ├── config/            # Configuration management
│   ├── dbexecutor/        # Database query execution
│   ├── logger/            # Logging utilities
│   ├── server/            # HTTP server and routing
│   └── tsengine/          # TypeScript execution engine
├── scripts/               # Build and development scripts
│   └── build-ts.js        # TypeScript compilation using esbuild
├── dist/                  # Distribution build output (generated)
│   ├── app/               # Compiled JavaScript files
│   ├── turboscript        # Binary executables
│   ├── turboscript-linux  # Linux binary for deployment
│   ├── turboscript.yml    # Production-optimized configuration
│   └── runner.sh          # Production deployment script
├── docker-compose.dev.yml # Development environment
├── turboscript.yml        # Application configuration
└── init.sql              # Database schema
```

### Architecture Components

#### TypeScript Layer (`app/` folder)

The TypeScript layer contains the business logic and API endpoint definitions. Each endpoint is a TypeScript file that exports an async `handle` function:

- **Handle Function**: Async function that processes requests and returns responses
- **Database Access**: Direct async database operations using `turboQuery()`
- **Error Handling**: Comprehensive try/catch blocks for robust error management

**Example Endpoint Structure:**

```typescript
// app/routes/users/create.ts
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Validate input data
        const input = data as CreateUserInput;
        if (!input.name || !input.email || !input.password) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Missing required fields: name, email, password"
                }
            };
        }

        // Hash password for security
        const hashedPassword = await hashPassword(input.password);

        // Insert user into database
        const result = await turboQuery(
            'INSERT INTO users (uid, name, email, password) VALUES ($1, $2, $3, $4) RETURNING uid',
            [randomUUID(), input.name, input.email, hashedPassword]
        );

        return {
            code: 201,
            response: {
                status: "success",
                data: { uid: result[0].uid },
                message: "User created successfully"
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

#### Go Runtime Layer (`internal/` folder)

The Go layer provides the execution runtime that interprets and executes the TypeScript code:

1. **Server (`internal/server/`)**: HTTP server that routes requests to appropriate TypeScript handlers
2. **TSEngine (`internal/tsengine/`)**: Uses goja JavaScript VM and esbuild to execute TypeScript code
3. **DBExecutor (`internal/dbexecutor/`)**: Executes database queries with security validation
4. **Config (`internal/config/`)**: Loads and manages application configuration

### Key Concepts

#### Event Object

Every TypeScript handler receives an `Event` object containing:

```typescript
interface Event {
    headers: Record<string, string>;
    queryParameters: Record<string, string>;
    pathParameters: Record<string, string>;
    body: Record<string, any>;
}
```

#### Database Operations

TypeScript handlers use `turboQuery()` for direct database access with async/await pattern:

```typescript
// Single query
const users = await turboQuery('SELECT * FROM users WHERE active = $1', [true]);

// Parallel queries for better performance
const [user, preferences, activities] = await Promise.all([
    turboQuery('SELECT * FROM users WHERE uid = $1', [userId]),
    turboQuery('SELECT * FROM user_preferences WHERE user_id = $1', [userId]),
    turboQuery('SELECT * FROM activities WHERE user_id = $1 LIMIT 10', [userId])
]);

// Insert/Update with result
const result = await turboQuery(
    'INSERT INTO users (name, email) VALUES ($1, $2) RETURNING uid',
    [name, email]
);
```

#### Security Model

- **Table Whitelist**: Only tables listed in `turboscript.yml` can be accessed
- **SQL Injection Protection**: All queries are parameterized
- **Input Validation**: TypeScript handlers validate all inputs

## Development Setup

### Environment Configuration

1. **Database Configuration**
   - PostgreSQL runs in Docker container
   - Connection details in `docker-compose.dev.yml`
   - Schema defined in `init.sql`

2. **Application Configuration**
   - Main config in `turboscript.yml`
   - Environment variables in Docker Compose
   - Debug mode available for development

### Available Make Commands

```bash
# Development Commands
make up           # Start everything (app + database) in development mode
make down         # Stop everything
make restart      # Restart everything
make logs         # Show logs
make rebuild      # Rebuild and restart the app
make db           # Start only the database
make db-shell     # Connect to database

# Build Commands
make build        # Build the Go app locally
make build-dist   # Build complete distribution package (TS→JS, binaries, config)
make clean-dist   # Remove dist folder

# Utility Commands
make clean        # Remove all Docker containers and volumes
make help         # Show all available commands
```

### Hot Reloading

The development environment uses Air for hot reloading:

- Go code changes trigger automatic rebuilds
- TypeScript files are recompiled on each request
- Database schema changes require container restart

## Making Changes

### Adding New Endpoints

1. **Create TypeScript Handler**

   ```bash
   # Create new endpoint file
   touch app/routes/your-feature/endpoint.ts
   ```

2. **Implement Handler Functions**

   ```typescript
   export const handle = async (event: Event): Promise<TurboScriptResponse> => {
       try {
           // Your business logic here with async database operations
           const result = await turboQuery('SELECT * FROM your_table WHERE id = $1', [event.pathParameters.id]);

           return {
               code: 200,
               response: {
                   status: "success",
                   data: result
               }
           };
       } catch (error) {
           return {
               code: 500,
               response: {
                   status: "error",
                   message: error instanceof Error ? error.message : "Operation failed"
               }
           };
       }
   };
   ```

3. **Register Endpoint**

   ```yaml
   # Add to turboscript.yml
   endpoints:
     - path: /your-feature/endpoint
       method: GET
       file: app/routes/your-feature/endpoint.ts
   ```

### Modifying Database Schema

1. **Update `init.sql`** with your schema changes
2. **Restart containers** to apply changes:

   ```bash
   make clean && make up
   ```

### Adding Dependencies

**TypeScript Dependencies:**

```bash
npm install package-name
npm install --save-dev @types/package-name  # For type definitions
```

**Go Dependencies:**

```bash
go get github.com/package/name
go mod tidy
```

### Testing Your Changes

#### Local Development Testing

1. **Start Development Environment**

   ```bash
   make up
   ```

2. **Test Endpoints**

   ```bash
   # Test existing endpoints
   curl -X GET http://localhost:7890/users
   curl -X POST http://localhost:7890/users \
     -H "Content-Type: application/json" \
     -d '{"name":"Test User","email":"test@example.com","password":"password123","confirm_password":"password123"}'
   ```

3. **Check Logs**

   ```bash
   make logs
   ```

#### Production Build Testing

Test your changes with the production build system:

```bash
# Build distribution package
make build-dist

# Test the distribution
cd dist
chmod +x runner.sh

# Update database connection in runner.sh if needed
# Then run the production build
./runner.sh
```

#### Clean Build Testing

Ensure your changes work with a clean build:

```bash
# Clean everything and rebuild
make clean
make clean-dist
make up
make build-dist
```

## Coding Standards

### TypeScript Code

- Use strict TypeScript with all compiler checks enabled
- Follow the async `handle()` pattern in `app/routes/`
- Always validate inputs in handle functions
- Use descriptive error messages with proper error handling
- Leverage the global type definitions in `app/global.d.ts`
- Use `turboQuery()` for all database operations

**Example:**

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const input = event.body as { name?: string; email?: string };

        if (!input.name) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Name is required"
                }
            };
        }

        if (!input.email) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Email is required"
                }
            };
        }

        const result = await turboQuery(
            'INSERT INTO users (name, email) VALUES ($1, $2) RETURNING uid',
            [input.name, input.email]
        );

        return {
            code: 201,
            response: {
                status: "success",
                data: { uid: result[0].uid }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Operation failed"
            }
        };
    }
};
        operation: "insert",
        table: "users",
        data: {
            name: input.name,
            email: input.email
        }
    };
}
```

### Go Code

- Follow standard Go conventions and formatting
- Use the existing logger for debugging: `logger.Debug()`, `logger.Info()`, `logger.Error()`
- Handle errors appropriately
- Add tests for new functionality

**Example:**

```go
func (e *TSExecutor) newMethod() error {
    logger.Debug("Starting new method")

    if err := someOperation(); err != nil {
        logger.Error("Operation failed: %v", err)
        return fmt.Errorf("failed to execute operation: %w", err)
    }

    logger.Info("Method completed successfully")
    return nil
}
```

### Configuration

- Add new configuration options to the Config struct in `internal/config/`
- Update `turboscript.yml` with sensible defaults
- Document new configuration options

## Testing

### Manual Testing

1. **Start Development Environment**

   ```bash
   make up
   ```

2. **Test Endpoints**

   ```bash
   # Test existing endpoints
   curl -X GET http://localhost:7890/users
   curl -X POST http://localhost:7890/users \
     -H "Content-Type: application/json" \
     -d '{"name":"Test User","email":"test@example.com","password":"password123","confirm_password":"password123"}'
   ```

3. **Check Logs**

   ```bash
   make logs
   ```

### Database Testing

```bash
# Connect to database
make db-shell

# Run queries
SELECT * FROM users;
```

### Integration Testing

Test your changes with the full stack:

1. Verify TypeScript compilation
2. Test database operations
3. Validate HTTP responses
4. Check error handling

## Pull Request Process

1. **Fork the Repository**
   - Create a fork of the main repository
   - Clone your fork locally

2. **Create Feature Branch**

   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make Changes**
   - Follow the coding standards
   - Test your changes thoroughly
   - Update documentation if needed

4. **Commit Changes**

   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

   Use conventional commit messages:
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation
   - `refactor:` for code refactoring
   - `test:` for testing improvements

5. **Push and Create PR**

   ```bash
   git push origin feature/your-feature-name
   ```

   Create a pull request with:
   - Clear description of changes
   - Link to related issues
   - Screenshots if UI changes
   - Testing instructions

6. **Code Review**
   - Address feedback from maintainers
   - Update your branch as needed
   - Maintain clean commit history

## Development Tips

### Debugging TypeScript

Enable debug mode in `turboscript.yml`:

```yaml
debug: true
```

This provides verbose logging of:

- TypeScript compilation
- JavaScript execution
- Database queries
- Input/output data

### Database Inspection

```bash
# Connect to PostgreSQL
make db-shell

# Useful queries
\dt                    # List tables
\d users              # Describe users table
SELECT * FROM users LIMIT 5;
```

### Performance Considerations

- **TypeScript Compilation**: Files are compiled on-demand in development, cached in production builds
- **Database connections**: Connection pooling is managed by the Go runtime
- **Binary Size**: Production builds are optimized for minimal size and maximum performance
- **Memory Usage**: The build system optimizes for efficient memory utilization
- **Build Speed**: esbuild provides fast TypeScript compilation for quick iterations

### Production Deployment

#### Using the Distribution Package

1. **Build for production**:

   ```bash
   make build-dist
   ```

2. **Deploy the `dist/` folder** to your production server

3. **Configure your database** connection in `runner.sh` or environment variables

4. **Run the application**:

   ```bash
   cd dist
   ./runner.sh
   ```

#### Direct Binary Deployment

For more control over the deployment:

```bash
cd dist
# Configure database connections in turboscript.yml
./turboscript-linux
```

#### Docker Deployment

```bash
# Build Docker image
make build-docker

# Run in production
make up-prod
```

## Contributing Guidelines

### What We're Looking For

- **New Features**: Additional database operations, utility functions, middleware
- **Improvements**: Performance optimizations, better error handling, security enhancements
- **Documentation**: Code comments, examples, tutorials
- **Bug Fixes**: Issues with existing functionality
- **Testing**: Unit tests, integration tests, performance tests

### Code of Conduct

- Be respectful and inclusive
- Provide constructive feedback
- Help newcomers get started
- Follow the project's technical standards

## License

By contributing to TurboScript, you agree that your contributions will be licensed under the Apache License, Version 2.0, as specified in the [LICENSE](LICENSE) file.

### Important License Notes

- Contributions must be compatible with the Apache License, Version 2.0
- Attribution requirements apply to all derivatives (see NOTICE.md)
- See LICENSE and NOTICE.md for complete terms

## Getting Help

- **Issues**: Create GitHub issues for bugs or feature requests
- **Discussions**: Use GitHub Discussions for questions
- **Documentation**: Check README.md and code comments
- **Examples**: Review existing endpoints in `app/routes/`

## Resources

- **Goja Documentation**: JavaScript VM for Go
- **ESBuild Documentation**: TypeScript to JavaScript compilation
- **PostgreSQL Documentation**: Database features and SQL syntax
- **Go Documentation**: Standard library and best practices

---

Thank you for contributing to TurboScript! Your contributions help make this project better for everyone.

### Build System and Distribution

TurboScript includes a comprehensive build system that compiles TypeScript to optimized JavaScript and creates production-ready distributions:

#### Development Build Process

During development, TypeScript files are compiled on-demand by the Go runtime using esbuild integration. This provides:

- **Hot Reloading**: Changes are reflected immediately
- **Type Checking**: Full TypeScript type safety
- **Fast Compilation**: esbuild's performance for quick iterations

#### Production Build Process

The `make build-dist` command creates a complete distribution package:

1. **TypeScript Compilation**:
   - Uses `scripts/build-ts.js` for optimized compilation
   - Compiles all `.ts` files to `.js` with esbuild
   - Preserves directory structure in `dist/app/`
   - Copies type definitions and maintains compatibility

2. **Binary Generation**:
   - Builds Go binary for current platform
   - Cross-compiles Linux binary for deployment
   - Optimizes for production performance

3. **Configuration Optimization**:
   - Modifies `turboscript.yml` for production
   - Updates file extensions from `.ts` to `.js`
   - Disables debug mode and monitoring
   - Creates deployment-ready configuration

4. **Deployment Packaging**:
   - Generates `runner.sh` script with database configuration
   - Creates self-contained `dist/` folder
   - Includes all necessary files for production deployment

#### Build Script Features

The `scripts/build-ts.js` provides:

- **Recursive TypeScript compilation** from `app/` to `dist/app/`
- **esbuild integration** with Node.js target optimization
- **File size reporting** and build statistics
- **Error handling** with detailed failure messages
- **TypeScript configuration** detection and usage

#### Distribution Structure

```text
dist/
├── app/                    # Compiled JavaScript files
│   ├── routes/            # API handlers (compiled)
│   ├── utils/             # Utilities (compiled)
│   └── global.d.ts        # Type definitions
├── turboscript             # macOS/current platform binary
├── turboscript-linux       # Linux deployment binary
├── turboscript.yml         # Production configuration
└── runner.sh               # Deployment script
```
