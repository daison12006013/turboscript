# Troubleshooting Guide

This guide helps you diagnose and fix common issues in TurboScript applications.

## Common Issues

### 1. Development Server Issues

#### Server Won't Start

**Symptoms:**

- Error when running `make dev`
- Port already in use
- Database connection failed

**Solutions:**

1. **Port in Use:**

   ```bash
   # Check what's using the port
   lsof -i :7890

   # Kill the process using the port
   kill -9 <PID>

   # Or use a different port in turboscript.dev.yml
   server:
     port: 7891
   ```

2. **Database Connection:**

   ```bash
   # Check if PostgreSQL container is running
   docker ps | grep postgres

   # Check database container logs
   docker logs turboscript-postgres-1

   # Restart the database container
   docker-compose -f docker-compose.dev.yml restart postgres

   # Recreate the database container if needed
   docker-compose -f docker-compose.dev.yml down
   docker-compose -f docker-compose.dev.yml up -d
   ```

3. **Environment Variables Missing:**

   ```bash
   # Check if .env file exists
   ls -la .env

   # Copy from template if missing
   cp .env.example .env

   # Verify JWT secrets are set
   grep JWT_ .env
   ```

4. **Go Module Issues:**

   ```bash
   # Clean and rebuild Go modules
   go clean -modcache
   go mod download
   go mod tidy

   # Check for Go version compatibility
   go version  # Should be 1.21+
   ```

#### TypeScript Compilation Errors

**Symptoms:**

- Route handlers not loading
- TypeScript errors during startup
- Hot reloading not working

**Solutions:**

1. **Check TypeScript Syntax:**

   ```bash
   # Check all TypeScript files for errors
   npx tsc --noEmit

   # Check specific route file
   npx tsc --noEmit app/routes/users/create.ts
   ```

2. **Common TypeScript Issues:**

   ```typescript
   // ❌ Common mistake: Missing return type
   export const handle = async (event: Event) => {
       // Should specify Promise<TurboScriptResponse>
   }

   // ✅ Correct: Proper return type
   export const handle = async (event: Event): Promise<TurboScriptResponse> => {
       // Implementation
   }

   // ❌ Common mistake: Incorrect interface usage
   const input = event.body as { email: string };  // Missing proper validation

   // ✅ Correct: Proper input validation
   const input = event.body as { email?: string };
   if (!input.email) {
       return {
           code: 400,
           response: {
               status: "error",
               message: "Email is required"
           }
       };
   }
   ```

3. **Global Types Issues:**

   ```typescript
   // Make sure global.d.ts is properly configured
   // Check that all global functions are declared

   // If turboQuery is not recognized, check global.d.ts:
   declare global {
       function turboQuery(query: string, params?: unknown[]): Promise<unknown[]>;
   }
   ```

### 2. Database Issues

#### Connection Problems

**Symptoms:**

- Database connection timeouts
- `turboQuery()` errors
- PostgreSQL connection refused

**Diagnostic Steps:**

```bash
# 1. Check if PostgreSQL container is running
docker ps | grep postgres

# 2. Check database logs
docker logs turboscript-postgres-1 --tail=50

# 3. Test database connection manually
docker exec -it turboscript-postgres-1 psql -U postgres -d turboscript

# 4. Check database configuration
cat turboscript.dev.yml | grep -A 10 database

# 5. Verify environment variables
env | grep DB_
```

**Solutions:**

1. **Container Not Running:**

   ```bash
   # Start the database container
   docker-compose -f docker-compose.dev.yml up -d postgres

   # If container keeps failing, check logs
   docker logs turboscript-postgres-1
   ```

2. **Wrong Credentials:**

   ```yaml
   # Check turboscript.dev.yml database section
   database:
     host: "localhost"
     port: 5432
     name: "turboscript"
     user: "postgres"
     password: "postgres"  # Must match docker-compose.dev.yml
   ```

3. **Connection Pool Exhaustion:**

   ```yaml
   # Increase connection pool size
   database:
     max_connections: 20  # Increase from default
     connection_timeout: "10s"
   ```

#### Query Errors

**Common Query Problems:**

1. **Table Not Allowed:**

   ```text
   Error: table "products" is not in allowed_tables list
   ```

   **Solution:**

   ```yaml
   # Add table to turboscript.dev.yml
   database:
     allowed_tables:
       - users
       - sessions
       - products  # Add your table here
   ```

2. **SQL Syntax Errors:**

   ```typescript
   // ❌ Common SQL mistakes
   const users = await turboQuery('SELECT * FROM users WHERE id = $1', [userId]);
   // Check: Is the column name correct? Is the table name correct?

   // ✅ Debug by logging the query
   console.log('Executing query:', 'SELECT * FROM users WHERE uid = $1', [userId]);
   const users = await turboQuery('SELECT * FROM users WHERE uid = $1', [userId]);
   ```

3. **Parameter Mismatch:**

   ```typescript
   // ❌ Wrong number of parameters
   const users = await turboQuery('SELECT * FROM users WHERE id = $1 AND status = $2', [userId]);
   // Missing second parameter for $2

   // ✅ Correct parameter count
   const users = await turboQuery('SELECT * FROM users WHERE id = $1 AND status = $2', [userId, 'active']);
   ```

4. **Type Conversion Issues:**

   ```typescript
   // ❌ Common type issues
   const id = event.pathParameters.id;  // String from URL
   const users = await turboQuery('SELECT * FROM users WHERE id = $1', [id]);
   // If id column is integer, this might fail

   // ✅ Proper type conversion
   const id = parseInt(event.pathParameters.id, 10);
   if (isNaN(id)) {
       return { code: 400, response: { status: "error", message: "Invalid ID" } };
   }
   const users = await turboQuery('SELECT * FROM users WHERE id = $1', [id]);
   ```

### 3. Authentication Issues

#### JWT Token Problems

**Symptoms:**

- "Invalid token" errors
- Authentication always failing
- Token refresh not working

**Diagnostic Steps:**

```bash
# 1. Check JWT secrets are set
grep JWT_ .env

# 2. Test token generation manually
curl -X POST http://localhost:7890/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password"}'

# 3. Decode JWT token to check content
# Use online JWT decoder or:
node -e "console.log(JSON.stringify(JSON.parse(Buffer.from('PAYLOAD_PART'.split('.')[1], 'base64')), null, 2))"
```

**Solutions:**

1. **Missing JWT Secrets:**

   ```bash
   # Generate secure JWT secrets
   openssl rand -base64 64  # For JWT_ACCESS_SECRET
   openssl rand -base64 64  # For JWT_REFRESH_SECRET

   # Add to .env file
   echo "JWT_ACCESS_SECRET=your_generated_access_secret" >> .env
   echo "JWT_REFRESH_SECRET=your_generated_refresh_secret" >> .env
   ```

2. **Token Expiration Issues:**

   ```typescript
   // Check token expiration in utils/jwt.ts
   const ACCESS_TOKEN_EXPIRES_IN = 15 * 60 * 1000; // 15 minutes
   const REFRESH_TOKEN_EXPIRES_IN = 7 * 24 * 60 * 60 * 1000; // 7 days

   // For debugging, temporarily increase expiration
   const ACCESS_TOKEN_EXPIRES_IN = 60 * 60 * 1000; // 1 hour
   ```

3. **Header Format Issues:**

   ```bash
   # ❌ Wrong authorization header format
   curl -H "Authorization: your_token" http://localhost:7890/protected

   # ✅ Correct format
   curl -H "Authorization: Bearer your_token" http://localhost:7890/protected
   ```

4. **Cookie vs Header Authentication:**

   ```typescript
   // If using cookies, check the auth utility handles both
   export const verifyAuth = (event: Event): JWTPayload | null => {
       // Check Authorization header first
       const authHeader = event.headers.authorization || event.headers.Authorization;

       if (authHeader) {
           const token = extractToken(authHeader);
           return verifyAccessToken(token, event);
       }

       // Fall back to cookie if header not present
       const cookieToken = extractTokenFromCookie(event.headers.cookie, 'access_token');
       if (cookieToken) {
           return verifyAccessToken(cookieToken, event);
       }

       return null;
   };
   ```

### 4. Route Handler Issues

#### Route Not Found (404 Errors)

**Symptoms:**

- 404 errors for routes that should exist
- Routes work sometimes but not others
- New routes not being recognized

**Diagnostic Steps:**

```bash
# 1. Check turboscript.yml configuration
cat turboscript.dev.yml | grep -A 20 endpoints

# 2. Verify TypeScript file exists
ls -la app/routes/users/create.ts

# 3. Check for TypeScript compilation errors
npx tsc --noEmit app/routes/users/create.ts

# 4. Test route directly
curl -X POST http://localhost:7890/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Test", "email": "test@example.com"}'
```

**Solutions:**

1. **Route Configuration Mismatch:**

   ```yaml
   # ❌ Common configuration mistakes
   endpoints:
     - route: /users      # Missing HTTP method
       path: ./app/routes/users/create.ts

   # ✅ Correct configuration
   endpoints:
     - route: /users
       method: POST
       path: ./app/routes/users/create.ts
   ```

2. **File Path Issues:**

   ```yaml
   # ❌ Wrong file path
   endpoints:
     - route: /users
       method: POST
       path: ./routes/users/create.ts  # Missing 'app/' prefix

   # ✅ Correct file path
   endpoints:
     - route: /users
       method: POST
       path: ./app/routes/users/create.ts
   ```

3. **Missing Export:**

   ```typescript
   // ❌ Missing or wrong export
   function handleevent: Event): TurboScriptResponse {
       // Implementation
   }
   // Missing: export const handle = ...

   // ✅ Correct export
   export const handle = async (event: Event): Promise<TurboScriptResponse> => {
       // Implementation
   };
   ```

#### Runtime Errors in Handlers

**Common Runtime Issues:**

1. **Undefined Variables:**

   ```typescript
   // ❌ Common undefined access
   const userId = event.body.user.uid;  // Will crash if user is undefined

   // ✅ Safe access
   const userId = event.body.__user?.uid;
   if (!userId) {
       return { code: 401, response: { status: "error", message: "Unauthorized" } };
   }
   ```

2. **Async/Await Issues:**

   ```typescript
   // ❌ Missing await
   export const handle = async (event: Event): Promise<TurboScriptResponse> => {
       const users = turboQuery('SELECT * FROM users');  // Missing await!

       return { code: 200, response: { data: users } };
   }

   // ✅ Proper await
   export const handle = async (event: Event): Promise<TurboScriptResponse> => {
       const users = await turboQuery('SELECT * FROM users');

       return { code: 200, response: { data: users } };
   }
   ```

3. **Error Handling:**

   ```typescript
   // ❌ No error handling
   export const handle = async (event: Event): Promise<TurboScriptResponse> => {
       const users = await turboQuery('SELECT * FROM users WHERE id = $1', [userId]);
       return { code: 200, response: { data: users[0] } };  // Will crash if empty array
   }

   // ✅ Proper error handling
   export const handle = async (event: Event): Promise<TurboScriptResponse> => {
       try {
           const users = await turboQuery('SELECT * FROM users WHERE id = $1', [userId]);

           if (!users || users.length === 0) {
               return { code: 404, response: { status: "error", message: "User not found" } };
           }

           return { code: 200, response: { status: "success", data: users[0] } };
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

### 5. Background Jobs Issues

#### Jobs Not Processing

**Symptoms:**

- Jobs queued but never executed
- Job workers not starting
- Email jobs failing

**Diagnostic Steps:**

```bash
# 1. Check if job workers are configured
cat turboscript.dev.yml | grep -A 10 jobs

# 2. Check job queue directory exists
ls -la app/queue/

# 3. Check job handler files
ls -la app/queue/*.ts

# 4. Test job dispatch manually
curl -X POST http://localhost:7890/test-job
```

**Solutions:**

1. **Job Configuration Issues:**

   ```yaml
   # Ensure jobs are properly configured
   jobs:
     max_workers: 5
     queue_size: 1000
     path: ./app/queue  # Must point to correct directory
   ```

2. **Job Handler Issues:**

   ```typescript
   // ❌ Common job handler mistakes
   export function handle(data: unknown): void {  // Missing async, wrong return type
       // Implementation
   }

   // ✅ Correct job handler
   export const handle = async (data: unknown): Promise<void> => {
       try {
           // Your job logic here
           console.log('Processing job with data:', data);
       } catch (error) {
           console.error('Job failed:', error);
           throw error;  // Re-throw to mark job as failed
       }
   };
   ```

3. **Email Job Failures:**

   ```yaml
   # Check email configuration in turboscript.dev.yml
   email:
     driver: "log"  # Use "log" for development testing
     from_address: "dev@localhost"
     from_name: "Dev Server"
   ```

### 6. Performance Issues

#### Slow Database Queries

**Symptoms:**

- Requests taking too long
- Database timeouts
- High CPU usage

**Diagnostic Steps:**

```bash
# 1. Enable query logging in PostgreSQL
docker exec -it turboscript-postgres-1 psql -U postgres -c "ALTER SYSTEM SET log_min_duration_statement = 1000;"
docker exec -it turboscript-postgres-1 psql -U postgres -c "SELECT pg_reload_conf();"

# 2. Monitor query performance
docker logs turboscript-postgres-1 | grep "duration:"

# 3. Check for missing indexes
docker exec -it turboscript-postgres-1 psql -U postgres -d turboscript -c "\d+ users"
```

**Solutions:**

1. **Add Database Indexes:**

   ```sql
   -- Common indexes for performance
   CREATE INDEX idx_users_email ON users(email);
   CREATE INDEX idx_users_uid ON users(uid);
   CREATE INDEX idx_orders_user_id ON orders(user_id);
   CREATE INDEX idx_sessions_expires_at ON user_sessions(expires_at);
   ```

2. **Optimize Queries:**

   ```typescript
   // ❌ Inefficient: Multiple sequential queries
   const user = await turboQuery('SELECT * FROM users WHERE id = $1', [userId]);
   const orders = await turboQuery('SELECT * FROM orders WHERE user_id = $1', [userId]);

   // ✅ Efficient: Parallel queries
   const [user, orders] = await Promise.all([
       turboQuery('SELECT * FROM users WHERE id = $1', [userId]),
       turboQuery('SELECT * FROM orders WHERE user_id = $1', [userId])
   ]);
   ```

#### Memory Issues

**Symptoms:**

- High memory usage
- Out of memory errors
- Slow garbage collection

**Solutions:**

1. **Optimize Data Fetching:**

   ```typescript
   // ❌ Loading too much data
   const users = await turboQuery('SELECT * FROM users');  // No limit!

   // ✅ Limit data size
   const users = await turboQuery('SELECT id, name, email FROM users LIMIT 100');
   ```

2. **Implement Pagination:**

   ```typescript
   // Implement proper pagination
   const page = parseInt(event.queryParameters.page || '1');
   const limit = Math.min(100, parseInt(event.queryParameters.limit || '20'));
   const offset = (page - 1) * limit;

   const users = await turboQuery(
       'SELECT id, name, email FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2',
       [limit, offset]
   );
   ```

## Debugging Tools and Techniques

### Development Debugging

1. **Enable Debug Logging:**

   ```yaml
   # turboscript.dev.yml
   debug: true
   logging:
     level: "debug"
     format: "text"
   ```

2. **Add Console Logging:**

   ```typescript
   export const handle = async (event: Event): Promise<TurboScriptResponse> => {
       console.log('Handler called with:', { data, event });

       try {
           const result = await turboQuery('SELECT * FROM users WHERE id = $1', [userId]);
           console.log('Query result:', result);

           return { code: 200, response: { data: result } };
       } catch (error) {
           console.error('Handler error:', error);
           throw error;
       }
   };
   ```

3. **Test Individual Components:**

   ```bash
   # Test database connection
   docker exec -it turboscript-postgres-1 psql -U postgres -d turboscript -c "SELECT NOW();"

   # Test specific endpoints
   curl -v http://localhost:7890/users

   # Test authentication
   curl -H "Authorization: Bearer TOKEN" http://localhost:7890/protected
   ```

### Production Debugging

1. **Structured Logging:**

   ```yaml
   # turboscript.yml (production)
   logging:
     level: "error"
     format: "json"
   ```

2. **Health Check Endpoints:**

   ```typescript
   // app/routes/health.ts
   export const handle = async (_event: Event): Promise<TurboScriptResponse> => {
       try {
           // Test database connection
           await turboQuery('SELECT 1');

           return {
               code: 200,
               response: {
                   status: "healthy",
                   timestamp: new Date().toISOString(),
                   database: "connected"
               }
           };
       } catch (error) {
           return {
               code: 503,
               response: {
                   status: "unhealthy",
                   error: "Database connection failed"
               }
           };
       }
   };
   ```

## Getting Help

### Before Asking for Help

1. **Check the logs:**

   ```bash
   # Application logs
   docker logs turboscript-app-dev-1 --tail=50

   # Database logs
   docker logs turboscript-postgres-1 --tail=50
   ```

2. **Verify configuration:**

   ```bash
   # Check YAML syntax
   yamllint turboscript.dev.yml

   # Verify environment variables
   env | grep -E "(JWT|DB)_"
   ```

3. **Test minimal reproduction:**

   Create a simple test case that reproduces the issue.

### Community Resources

- **GitHub Issues**: [Report bugs](https://github.com/daison12006013/turboscript/issues)
- **Discussions**: [Ask questions](https://github.com/daison12006013/turboscript/discussions)
- **Documentation**: [Read the docs](../index.md)

---

## Navigation

**Previous:** [← Deployment Guide](guides/deployment.md)
**Next:** [API Reference →](api/route-handlers.md)

## Related Topics

- [Development Workflow](guides/development.md)
- [Configuration Guide](guides/configuration.md)
- [Performance Guide](guides/performance.md)
- [Security Guidelines](guides/security.md)
