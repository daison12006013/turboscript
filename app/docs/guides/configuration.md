# Configuration Guide

## Overview

TurboScript uses YAML-based configuration with environment variable support. This guide covers all configuration options and best practices.

## Configuration File Structure

The main configuration file is `turboscript.yml`. You can create different configurations for different environments (e.g., `turboscript.dev.yml` for development).

```yaml
server:
  port: 7890
  timeout: "30s"
  cors:
    enabled: true
    origins: ["http://localhost:3000"]

database:
  host: "localhost"
  port: 5432
  name: "turboscript"
  user: "postgres"
  password: "postgres"
  ssl: false
  max_connections: 10
  connection_timeout: "5s"
  allowed_tables:
    - users
    - sessions
    - audit_logs
    - orders
    - products

security:
  jwt_access_secret: "${JWT_ACCESS_SECRET}"
  jwt_refresh_secret: "${JWT_REFRESH_SECRET}"
  bcrypt_cost: 12

logging:
  level: "debug"
  format: "json"

email:
  driver: "smtp"
  smtp_host: "smtp.example.com"
  smtp_port: 587
  smtp_user: "${SMTP_USER}"
  smtp_password: "${SMTP_PASSWORD}"
  from_address: "noreply@example.com"
  from_name: "TurboScript App"

jobs:
  max_workers: 5
  queue_size: 1000
  path: ./app/queue
  data_retention:
    jobs_days: 15
    history_days: 15
    auto_cleanup: true

debug: true

endpoints:
  - route: /
    method: GET
    path: ./app/routes/index

  - route: /users
    method: POST
    path: ./app/routes/users/create

  - route: /auth/login
    method: POST
    path: ./app/routes/auth/login

  - route: /auth/refresh
    method: POST
    path: ./app/routes/auth/refresh

  - route: /users
    method: GET
    path: ./app/routes/users/paginated

  - route: /users/{uid}
    method: GET
    path: ./app/routes/users/filter-by-uid

  - route: /users/change-password
    method: POST
    path: ./app/routes/users/change-password
```

## Server Configuration

### Basic Server Settings

- **`port`**: Server port number (default: 7890)
- **`timeout`**: Request timeout duration (e.g., "30s", "1m")
- **`debug`**: Enable debug mode for detailed logging (true/false)

### CORS Configuration

```yaml
server:
  cors:
    enabled: true
    origins:
      - "http://localhost:3000"
      - "https://yourapp.com"
    methods:
      - "GET"
      - "POST"
      - "PUT"
      - "DELETE"
    headers:
      - "Content-Type"
      - "Authorization"
```

## Database Configuration

### Connection Settings

```yaml
database:
  host: "localhost"
  port: 5432
  name: "turboscript"
  user: "postgres"
  password: "postgres"
  ssl: false
  max_connections: 10
  connection_timeout: "5s"
```

### Security: Allowed Tables

TurboScript restricts database access to specific tables for security:

```yaml
database:
  allowed_tables:
    - users
    - sessions
    - audit_logs
    - orders
    - order_items
    - products
    - user_preferences
    - notifications
```

**Important**: Only tables listed in `allowed_tables` can be accessed via `turboQuery()`. This prevents unauthorized access to system tables or other sensitive data.

## Security Configuration

### JWT Settings

```yaml
security:
  jwt_access_secret: "${JWT_ACCESS_SECRET}"     # Environment variable
  jwt_refresh_secret: "${JWT_REFRESH_SECRET}"   # Environment variable
  bcrypt_cost: 12                               # Password hashing cost
```

### Best Practices

- **Use Environment Variables**: Never hardcode secrets in YAML files
- **Strong Secrets**: Use cryptographically secure random strings
- **Bcrypt Cost**: Use cost 12+ for production (balance security vs performance)

## Email Configuration

### SMTP Configuration

```yaml
email:
  driver: "smtp"
  smtp_host: "smtp.gmail.com"
  smtp_port: 587
  smtp_user: "${SMTP_USER}"
  smtp_password: "${SMTP_PASSWORD}"
  from_address: "noreply@yourapp.com"
  from_name: "Your App Name"
  encryption: "tls"
```

### Supported Drivers

- **`smtp`**: Standard SMTP email delivery
- **`sendmail`**: Local sendmail binary
- **`log`**: Log emails to console (development only)

## Background Jobs Configuration

### Queue Settings

```yaml
jobs:
  max_workers: 5        # Number of concurrent workers
  queue_size: 1000      # Maximum queued jobs
  path: ./app/queue     # Directory containing job handlers
  retry_attempts: 3     # Number of retry attempts for failed jobs
  retry_delay: "30s"    # Delay between retry attempts
```

### Data Retention

```yaml
jobs:
  data_retention:
    jobs_days: 15         # Keep completed/failed jobs for 15 days
    history_days: 15      # Keep job history for 15 days
    auto_cleanup: true    # Automatically clean up old jobs
```

## Endpoints Configuration

### Route Definition

```yaml
endpoints:
  - route: /api/users/{id}      # URL pattern with parameters
    method: GET                 # HTTP method
    path: ./app/routes/users/get-by-id.ts  # TypeScript file
    timeout: "10s"              # Optional: per-endpoint timeout

  - route: /api/upload
    method: POST
    path: ./app/routes/upload.ts
    max_body_size: "10MB"       # Optional: request body size limit
```

### Route Parameters

- **Path Parameters**: Use `{param}` syntax (e.g., `/users/{uid}`)
- **Query Parameters**: Automatically available in `event.queryParameters`
- **Request Body**: Available in `event.body`

## Environment Variables

### Required Variables

```bash
# JWT Secrets (generate secure random strings)
JWT_ACCESS_SECRET=your_super_secure_access_secret_here
JWT_REFRESH_SECRET=your_super_secure_refresh_secret_here

# Database (if not using defaults)
DB_HOST=localhost
DB_PORT=5432
DB_NAME=turboscript
DB_USER=postgres
DB_PASSWORD=postgres

# Email Configuration
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
```

### Development vs Production

Create separate configuration files:

- **Development**: `turboscript.dev.yml`
- **Production**: `turboscript.yml`

```bash
# Development
make dev  # Uses turboscript.dev.yml

# Production
./turboscript  # Uses turboscript.yml
```

## Configuration Validation

TurboScript validates configuration on startup:

- **Required Fields**: Missing required fields cause startup failure
- **Type Validation**: Ensures correct data types
- **Environment Variables**: Validates that referenced env vars exist
- **File Paths**: Checks that route files exist

## Configuration Best Practices

### Security

1. **Never commit secrets**: Use environment variables for sensitive data
2. **Principle of least privilege**: Only list necessary tables in `allowed_tables`
3. **Strong JWT secrets**: Use cryptographically secure random strings
4. **HTTPS in production**: Enable SSL/TLS for production deployments

### Performance

1. **Connection pooling**: Configure appropriate `max_connections`
2. **Timeouts**: Set reasonable timeout values
3. **Job workers**: Scale `max_workers` based on workload
4. **Logging**: Use "error" level in production for performance

### Development Workflow

1. **Environment files**: Use `.env` files for local development
2. **Hot reloading**: Enable debug mode for development
3. **Separate configs**: Maintain different configs for different environments
4. **Version control**: Commit config templates, not actual secrets

## Troubleshooting

### Common Issues

**Configuration not loading**:

- Check YAML syntax with a validator
- Ensure file permissions are correct
- Verify environment variables are set

**Database connection fails**:

- Check database credentials
- Verify database is running
- Check network connectivity

**Route not found**:

- Verify file path exists
- Check YAML syntax in endpoints section
- Ensure TypeScript file exports `handle` function

**JWT errors**:

- Verify JWT secrets are set
- Check secret length (should be sufficiently long)
- Ensure secrets are consistent across restarts

---

## Navigation

**Previous:** [← Getting Started](guides/getting-started.md)
**Next:** [Development Workflow →](guides/development.md)

## Related Topics

- [Security Guidelines](guides/security.md)
- [Deployment Guide](guides/deployment.md)
- [Getting Started](guides/getting-started.md)
- [Troubleshooting](guides/troubleshooting.md)
