# @turboscript/pg - PostgreSQL Client for Goja

A comprehensive PostgreSQL client library for the [goja JavaScript VM](https://github.com/dop251/goja), fully compatible with the Node.js [pg](https://github.com/brianc/node-postgres) library API.

## 🚀 Features

- ✅ **Full Node.js pg compatibility** - Drop-in replacement for existing code
- ✅ **Connection pooling** - Efficient connection management with configurable pool settings
- ✅ **Async/Promise support** - Modern async/await patterns with full Promise support
- ✅ **Type safety** - Complete TypeScript definitions included
- ✅ **Secure by default** - Parameterized queries prevent SQL injection
- ✅ **Transaction support** - Full transaction handling with commit/rollback
- ✅ **Event handling** - Complete event system for connection lifecycle
- ✅ **SSL support** - Configurable SSL/TLS connections
- ✅ **Error handling** - Comprehensive error reporting and handling
- ✅ **Timeouts & retries** - Configurable timeouts for all operations
- ✅ **Zero dependencies** - Pure Go implementation using lib/pq driver

## 📦 Installation

Copy the pg module to your project:

```bash
cp -r turbo_modules/pg your_project/modules/
```

## 🏁 Quick Start

### Basic Client Usage

```javascript
const pg = require('pg');

// Create a client
const client = new pg.Client({
    host: 'localhost',
    port: 5432,
    database: 'mydb',
    user: 'postgres',
    password: 'password'
});

// Connect and query
async function example() {
    await client.connect();

    const result = await client.query('SELECT NOW()');
    console.log(result.rows[0]);

    await client.end();
}
```

### Connection Pool Usage

```javascript
const pg = require('pg');

// Create a pool (recommended for applications)
const pool = new pg.Pool({
    host: 'localhost',
    port: 5432,
    database: 'mydb',
    user: 'postgres',
    password: 'password',
    max: 10,  // Maximum pool size
    idleCount: 2  // Minimum idle connections
});

// Direct pool querying (recommended)
async function poolExample() {
    const result = await pool.query('SELECT * FROM users WHERE id = $1', [123]);
    console.log(result.rows);

    await pool.end();
}
```

## 🔧 Configuration Options

### Connection Configuration

```javascript
const config = {
    // Basic connection
    host: 'localhost',                    // Database host
    port: 5432,                          // Database port
    database: 'mydb',                    // Database name
    user: 'postgres',                    // Username
    password: 'secret',                  // Password

    // Alternative: connection string
    connectionString: 'postgresql://user:pass@host:port/db',

    // SSL Configuration
    ssl: false,                          // Enable SSL
    sslmode: 'prefer',                   // SSL mode: disable, allow, prefer, require, verify-ca, verify-full

    // Timeout settings (milliseconds)
    connectionTimeoutMillis: 30000,      // Connection timeout
    queryTimeoutMillis: 30000,           // Query timeout
    idleTimeoutMillis: 30000,            // Idle connection timeout

    // Pool settings (for Pool only)
    max: 10,                            // Maximum pool size
    idleCount: 0,                       // Minimum idle connections

    // PostgreSQL specific
    application_name: 'MyApp',           // Application name in pg_stat_activity
    statement_timeout: 30000,            // Statement timeout
    lock_timeout: 10000,                // Lock timeout
    idle_in_transaction_session_timeout: 60000  // Idle in transaction timeout
};
```

### Environment Variables

The module supports standard PostgreSQL environment variables:

```bash
PGHOST=localhost
PGPORT=5432
PGDATABASE=mydb
PGUSER=postgres
PGPASSWORD=secret
PGSSLMODE=prefer
```

## 📚 API Reference

### Client Class

#### Constructor

```javascript
const client = new pg.Client(config)
```

- `config` - Connection configuration object or connection string

#### Methods

##### connect()

```javascript
await client.connect()
```

Establishes connection to the database.

##### end()

```javascript
await client.end()
```

Closes the database connection.

##### query()

```javascript
// Simple query
const result = await client.query('SELECT NOW()')

// Parameterized query
const result = await client.query('SELECT * FROM users WHERE id = $1', [123])

// Query config object
const result = await client.query({
    text: 'SELECT * FROM users WHERE age > $1',
    values: [25],
    rowMode: 'array'  // 'array' or 'object'
})
```

##### Event Methods

```javascript
client.on('connect', () => console.log('Connected'))
client.on('end', () => console.log('Disconnected'))
client.on('error', (err) => console.error('Error:', err))
client.on('notice', (notice) => console.log('Notice:', notice))
client.on('notification', (msg) => console.log('Notification:', msg))

client.off('event', listener)
client.removeListener('event', listener)
```

##### Utility Methods

```javascript
const safeId = client.escapeIdentifier('column_name')    // "column_name"
const safeLiteral = client.escapeLiteral("John's data")  // 'John''s data'
```

### Pool Class

#### Constructor

```javascript
const pool = new pg.Pool(config)
```

#### Methods

##### connect()

```javascript
const client = await pool.connect()
// Use client...
client.release()  // Return to pool
```

##### query()

```javascript
// Direct pool querying (recommended)
const result = await pool.query('SELECT * FROM users')
```

##### end()

```javascript
await pool.end()  // Close all pool connections
```

##### Pool Statistics

```javascript
console.log(pool.totalCount)    // Total connections
console.log(pool.idleCount)     // Idle connections
console.log(pool.waitingCount)  // Waiting connections
```

### Query Result

```javascript
const result = await client.query('SELECT * FROM users')

console.log(result.rows)           // Array of row objects
console.log(result.rowCount)       // Number of rows
console.log(result.command)        // SQL command (SELECT, INSERT, etc.)
console.log(result.fields)         // Field metadata
console.log(result.processingTimeMs) // Query execution time
```

### Result Fields

```javascript
result.fields.forEach(field => {
    console.log(field.name)         // Column name
    console.log(field.dataTypeID)   // PostgreSQL data type ID
    console.log(field.format)       // 'text' or 'binary'
})
```

## 🔄 Transaction Handling

```javascript
const client = new pg.Client(config)
await client.connect()

try {
    await client.query('BEGIN')

    await client.query('INSERT INTO users (name) VALUES ($1)', ['John'])
    await client.query('INSERT INTO logs (action) VALUES ($1)', ['user_created'])

    await client.query('COMMIT')
} catch (error) {
    await client.query('ROLLBACK')
    throw error
} finally {
    await client.end()
}
```

## 🔒 Security Best Practices

### Parameterized Queries

Always use parameterized queries to prevent SQL injection:

```javascript
// ✅ Good - Parameterized query
const result = await client.query(
    'SELECT * FROM users WHERE email = $1 AND active = $2',
    [email, true]
)

// ❌ Bad - String concatenation
const result = await client.query(
    `SELECT * FROM users WHERE email = '${email}'`
)
```

### Connection Security

```javascript
const pool = new pg.Pool({
    host: 'localhost',
    port: 5432,
    database: 'mydb',
    user: 'app_user',        // Use limited privilege user
    password: process.env.DB_PASSWORD,  // Use environment variables
    sslmode: 'require',      // Require SSL in production
    max: 10,                 // Limit connection pool size
    idleTimeoutMillis: 30000 // Close idle connections
})
```

## ⚡ Performance Tips

### Use Connection Pooling

```javascript
// ✅ Recommended - Use pool for applications
const pool = new pg.Pool(config)
const result = await pool.query('SELECT * FROM users')

// ❌ Avoid - Creating new client for each query
const client = new pg.Client(config)
await client.connect()
const result = await client.query('SELECT * FROM users')
await client.end()
```

### Optimize Pool Settings

```javascript
const pool = new pg.Pool({
    max: 20,                    // Match your application's concurrency
    idleCount: 5,               // Keep some connections warm
    idleTimeoutMillis: 30000,   // Close idle connections
    connectionTimeoutMillis: 5000,  // Fail fast on connection issues
    queryTimeoutMillis: 10000   // Prevent long-running queries
})
```

### Batch Operations

```javascript
// Batch inserts with transactions
await client.query('BEGIN')
for (const user of users) {
    await client.query('INSERT INTO users (name, email) VALUES ($1, $2)',
                      [user.name, user.email])
}
await client.query('COMMIT')
```

## 🐛 Error Handling

### Connection Errors

```javascript
const client = new pg.Client(config)

try {
    await client.connect()
} catch (error) {
    if (error.code === 'ECONNREFUSED') {
        console.error('Database is not running')
    } else if (error.code === '28P01') {
        console.error('Authentication failed')
    } else {
        console.error('Connection error:', error.message)
    }
}
```

### Query Errors

```javascript
try {
    const result = await client.query('SELECT * FROM nonexistent_table')
} catch (error) {
    if (error.code === '42P01') {
        console.error('Table does not exist')
    } else if (error.code === '42703') {
        console.error('Column does not exist')
    } else {
        console.error('Query error:', error.message)
    }
}
```

### Pool Error Handling

```javascript
const pool = new pg.Pool(config)

pool.on('error', (err, client) => {
    console.error('Unexpected error on idle client', err)
    process.exit(-1)
})

pool.on('connect', (client) => {
    console.log('New client connected')
})
```

## 🧪 Testing

### Unit Tests

```javascript
// Basic connection test
async function testConnection() {
    const client = new pg.Client(testConfig)
    await client.connect()
    const result = await client.query('SELECT 1 as test')
    assert.equal(result.rows[0].test, 1)
    await client.end()
}

// Pool test
async function testPool() {
    const pool = new pg.Pool(testConfig)
    const result = await pool.query('SELECT COUNT(*) FROM users')
    assert(typeof result.rows[0].count === 'number')
    await pool.end()
}
```

## 🔗 Integration with TurboScript

In your TurboScript Go application:

```go
import (
    "github.com/dop251/goja"
    "path/to/your/turbo_modules/pg"
)

func setupDatabase(runtime *goja.Runtime, loop EventLoopRunner) error {
    // Create and register the pg module
    pgModule := pg.New(runtime, loop)
    return pgModule.Register()
}
```

In your TypeScript routes:

```typescript
import { Client, Pool } from 'pg';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const pool = new Pool({
        host: 'localhost',
        database: 'myapp',
        user: 'postgres',
        password: 'password'
    });

    try {
        const result = await pool.query(
            'SELECT * FROM users WHERE id = $1',
            [event.pathParameters.id]
        );

        return {
            code: 200,
            response: {
                status: 'success',
                data: result.rows[0]
            }
        };
    } finally {
        await pool.end();
    }
};
```

## 📝 Migration from Node.js

This module is designed to be a drop-in replacement for the Node.js pg library:

```javascript
// Node.js code works as-is
const { Client, Pool } = require('pg')
// or
import pg from 'pg'

// All standard APIs are supported
const client = new Client(config)
const pool = new Pool(config)
```

## 🆚 Differences from Node.js pg

- **Native bindings**: Uses Go's lib/pq driver instead of libpq
- **Event system**: Simplified event handling (some events may not be emitted)
- **COPY commands**: copyFrom/copyTo methods are placeholder implementations
- **Cursor support**: Not implemented in this version
- **Custom types**: Limited custom type support compared to Node.js version

## 🤝 Contributing

Contributions are welcome! Please:

1. Follow the existing code style
2. Add tests for new features
3. Update documentation
4. Ensure compatibility with Node.js pg API

## 📄 License

This module is licensed under the MIT License. See the LICENSE file for details.

## 🔗 Related Links

- [Node.js pg documentation](https://node-postgres.com/)
- [PostgreSQL documentation](https://www.postgresql.org/docs/)
- [Goja JavaScript VM](https://github.com/dop251/goja)
- [TurboScript Framework](https://github.com/daison12006013/turboscript)
