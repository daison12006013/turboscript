# @turboscript/mysql2 - MySQL Client for Goja

A comprehensive MySQL client library for the [goja JavaScript VM](https://github.com/dop251/goja), fully compatible with the Node.js [mysql2](https://github.com/sidorares/node-mysql2) library API.

## 🚀 Features

- ✅ **Full Node.js mysql2 compatibility** - Drop-in replacement for existing code
- ✅ **Dual API support** - Both callback-based and Promise-based APIs
- ✅ **Connection pooling** - Efficient connection management with configurable pool settings
- ✅ **Prepared statements** - Support for execute() method with prepared statements
- ✅ **Transaction support** - Full transaction handling with commit/rollback
- ✅ **Type safety** - Complete TypeScript definitions included
- ✅ **Secure by default** - Parameterized queries prevent SQL injection
- ✅ **Event handling** - Complete event system for connection lifecycle
- ✅ **SSL support** - Configurable SSL/TLS connections
- ✅ **Multiple result sets** - Support for multiple statements
- ✅ **Error handling** - Comprehensive error reporting and handling
- ✅ **Timeouts & retries** - Configurable timeouts for all operations
- ✅ **Zero dependencies** - Pure Go implementation using go-sql-driver/mysql

## 📦 Installation

Copy the mysql2 module to your project:

```bash
cp -r turbo_modules/mysql2 your_project/modules/
```

## 🏁 Quick Start

### Callback-based API

```javascript
const mysql2 = require('mysql2');

// Create a connection
const connection = mysql2.createConnection({
    host: 'localhost',
    port: 3306,
    user: 'root',
    password: 'password',
    database: 'mydb'
});

// Connect and query
connection.connect((err) => {
    if (err) {
        console.error('Connection failed:', err.message);
        return;
    }

    connection.query('SELECT * FROM users WHERE id = ?', [1], (err, results, fields) => {
        if (err) {
            console.error('Query failed:', err.message);
            return;
        }
        console.log(results[0]);
        connection.end();
    });
});
```

### Promise-based API

```javascript
const mysql2 = require('mysql2');

// Create a promise connection
const connection = mysql2.promise.createConnection({
    host: 'localhost',
    port: 3306,
    user: 'root',
    password: 'password',
    database: 'mydb'
});

// Connect and query with async/await
async function example() {
    try {
        await connection.connect();

        const [rows, fields] = await connection.query('SELECT * FROM users WHERE id = ?', [1]);
        console.log(rows[0]);

        await connection.end();
    } catch (error) {
        console.error('Error:', error.message);
    }
}
```

### Connection Pool

```javascript
const mysql2 = require('mysql2');

// Create a pool (recommended for applications)
const pool = mysql2.promise.createPool({
    host: 'localhost',
    port: 3306,
    user: 'root',
    password: 'password',
    database: 'mydb',
    connectionLimit: 10
});

// Direct pool querying
async function poolExample() {
    try {
        const [rows] = await pool.query('SELECT * FROM users');
        console.log(rows);
    } catch (error) {
        console.error('Pool error:', error.message);
    } finally {
        await pool.end();
    }
}
```

## 🔧 Configuration Options

### Connection Configuration

```javascript
const config = {
    // Basic connection
    host: 'localhost',                    // Database host
    port: 3306,                          // Database port
    user: 'root',                        // Username
    password: 'secret',                  // Password
    database: 'mydb',                    // Database name

    // Character settings
    charset: 'utf8mb4',                  // Character set
    timezone: '+00:00',                  // Timezone

    // Timeout settings (milliseconds)
    timeout: 60000,                      // Connection timeout
    readTimeout: 30000,                  // Read timeout
    writeTimeout: 30000,                 // Write timeout
    acquireTimeout: 60000,               // Pool acquire timeout

    // Pool settings (for Pool only)
    connectionLimit: 10,                 // Maximum pool size
    queueLimit: 0,                       // Queue limit (0 = unlimited)

    // SSL Configuration
    ssl: false,                          // SSL options
    insecureAuth: false,                 // Allow insecure auth

    // Query settings
    multipleStatements: false,           // Allow multiple statements
    dateStrings: false,                  // Return dates as strings
    debug: false,                        // Enable debug mode
    trace: false,                        // Enable trace mode

    // Number handling
    supportBigNumbers: false,            // Support big numbers
    bigNumberStrings: false,             // Return big numbers as strings

    // Data processing
    typeCast: true,                      // Enable type casting
    stringifyObjects: false,             // Stringify objects

    // Keep alive
    enableKeepAlive: true,               // Enable keep alive
    keepAliveInitialDelay: 0             // Keep alive delay
};
```

## 📚 API Reference

### Connection Methods

#### Callback-based Connection

```javascript
const connection = mysql2.createConnection(config)
```

##### connect()

```javascript
connection.connect((err) => {
    if (err) console.error('Connection failed:', err);
    else console.log('Connected to MySQL');
})
```

##### query()

```javascript
// Simple query
connection.query('SELECT NOW()', (err, results, fields) => {
    console.log(results[0]);
});

// Parameterized query
connection.query('SELECT * FROM users WHERE id = ?', [123], (err, results, fields) => {
    console.log(results);
});

// Query options
connection.query({
    sql: 'SELECT * FROM users WHERE age > ?',
    values: [25],
    timeout: 40000
}, (err, results, fields) => {
    console.log(results);
});
```

##### execute() (Prepared Statements)

```javascript
connection.execute('SELECT * FROM users WHERE id = ? AND status = ?', [1, 'active'], (err, results, fields) => {
    console.log(results);
});
```

##### Transaction Methods

```javascript
connection.beginTransaction((err) => {
    if (err) throw err;

    connection.query('INSERT INTO ...', (err, result) => {
        if (err) {
            return connection.rollback(() => {
                throw err;
            });
        }

        connection.commit((err) => {
            if (err) {
                return connection.rollback(() => {
                    throw err;
                });
            }
            console.log('Transaction completed');
        });
    });
});
```

##### Utility Methods

```javascript
connection.ping((err) => {
    if (err) console.error('Ping failed');
    else console.log('Server is alive');
});

connection.statistics((err, stats) => {
    console.log('Server stats:', stats);
});

const escaped = connection.escape("John's data");      // 'John\'s data'
const escapedId = connection.escapeId('table_name');   // `table_name`
const formatted = connection.format('SELECT * FROM users WHERE id = ?', [123]);
```

##### end()

```javascript
connection.end((err) => {
    if (err) console.error('Error closing connection');
    else console.log('Connection closed');
});
```

#### Promise-based Connection

```javascript
const connection = mysql2.promise.createConnection(config)
```

##### Promise Methods

```javascript
// Connect
await connection.connect();

// Query (returns [rows, fields])
const [rows, fields] = await connection.query('SELECT * FROM users');

// Execute prepared statement
const [results] = await connection.execute('SELECT * FROM users WHERE id = ?', [1]);

// Transactions
await connection.beginTransaction();
await connection.query('INSERT INTO ...');
await connection.commit(); // or connection.rollback()

// Utilities
await connection.ping();
const stats = await connection.statistics();

// Close
await connection.end();
```

### Pool Methods

#### Callback-based Pool

```javascript
const pool = mysql2.createPool(config)
```

##### getConnection()

```javascript
pool.getConnection((err, connection) => {
    if (err) throw err;

    connection.query('SELECT 1', (err, results) => {
        // Always release the connection
        connection.release();

        if (err) throw err;
        console.log(results);
    });
});
```

##### Direct Pool Querying

```javascript
pool.query('SELECT * FROM users', (err, results, fields) => {
    console.log(results);
});

pool.execute('SELECT * FROM users WHERE id = ?', [1], (err, results, fields) => {
    console.log(results);
});
```

##### end()

```javascript
pool.end((err) => {
    if (err) console.error('Error closing pool');
    else console.log('Pool closed');
});
```

#### Promise-based Pool

```javascript
const pool = mysql2.promise.createPool(config)
```

##### Promise Pool Methods

```javascript
// Get connection
const connection = await pool.getConnection();
try {
    const [rows] = await connection.query('SELECT * FROM users');
    console.log(rows);
} finally {
    connection.release(); // Always release
}

// Direct pool querying (recommended)
const [rows] = await pool.query('SELECT * FROM users');
const [results] = await pool.execute('SELECT * FROM users WHERE id = ?', [1]);

// Close pool
await pool.end();
```

### Query Results

```javascript
// Callback format: (err, results, fields)
connection.query('SELECT * FROM users', (err, results, fields) => {
    console.log(results);        // Array of row objects
    console.log(fields);         // Array of field metadata
});

// Promise format: [results, fields]
const [results, fields] = await connection.query('SELECT * FROM users');
```

### Field Metadata

```javascript
fields.forEach(field => {
    console.log(field.name);         // Column name
    console.log(field.type);         // Data type
    console.log(field.length);       // Field length
    console.log(field.database);     // Database name
    console.log(field.table);        // Table name
    console.log(field.orgTable);     // Original table name
    console.log(field.orgName);      // Original column name
});
```

## 🔒 Security Best Practices

### Parameterized Queries

Always use parameterized queries to prevent SQL injection:

```javascript
// ✅ Good - Parameterized query
const [rows] = await connection.query(
    'SELECT * FROM users WHERE email = ? AND active = ?',
    [email, true]
);

// ❌ Bad - String concatenation
const [rows] = await connection.query(
    `SELECT * FROM users WHERE email = '${email}'`
);
```

### Connection Security

```javascript
const pool = mysql2.promise.createPool({
    host: 'localhost',
    port: 3306,
    user: 'app_user',        // Use limited privilege user
    password: process.env.DB_PASSWORD,  // Use environment variables
    database: 'mydb',
    ssl: {                   // Enable SSL in production
        rejectUnauthorized: true
    },
    connectionLimit: 10,     // Limit connection pool size
    acquireTimeout: 60000,   // Prevent hanging connections
    timeout: 60000           // Connection timeout
});
```

## ⚡ Performance Tips

### Use Connection Pooling

```javascript
// ✅ Recommended - Use pool for applications
const pool = mysql2.promise.createPool(config);
const [rows] = await pool.query('SELECT * FROM users');

// ❌ Avoid - Creating new connection for each query
const connection = mysql2.promise.createConnection(config);
await connection.connect();
const [rows] = await connection.query('SELECT * FROM users');
await connection.end();
```

### Optimize Pool Settings

```javascript
const pool = mysql2.promise.createPool({
    connectionLimit: 20,        // Match your application's concurrency
    queueLimit: 50,            // Prevent excessive queueing
    acquireTimeout: 60000,     // Fail fast on pool exhaustion
    timeout: 30000,            // Prevent long-running connections
    readTimeout: 30000,        // Read operation timeout
    writeTimeout: 30000        // Write operation timeout
});
```

### Use Prepared Statements

```javascript
// Use execute() for repeated queries
const [rows] = await connection.execute(
    'SELECT * FROM users WHERE department = ? AND active = ?',
    ['Engineering', true]
);
```

### Batch Operations

```javascript
// Batch inserts with transactions
await connection.beginTransaction();
try {
    for (const user of users) {
        await connection.execute(
            'INSERT INTO users (name, email) VALUES (?, ?)',
            [user.name, user.email]
        );
    }
    await connection.commit();
} catch (error) {
    await connection.rollback();
    throw error;
}
```

## 🐛 Error Handling

### Connection Errors

```javascript
try {
    await connection.connect();
} catch (error) {
    if (error.code === 'ECONNREFUSED') {
        console.error('MySQL server is not running');
    } else if (error.code === 'ER_ACCESS_DENIED_ERROR') {
        console.error('Invalid credentials');
    } else {
        console.error('Connection error:', error.message);
    }
}
```

### Query Errors

```javascript
try {
    const [rows] = await connection.query('SELECT * FROM nonexistent_table');
} catch (error) {
    if (error.code === 'ER_NO_SUCH_TABLE') {
        console.error('Table does not exist');
    } else if (error.code === 'ER_BAD_FIELD_ERROR') {
        console.error('Invalid column name');
    } else {
        console.error('Query error:', error.message);
    }
}
```

### Pool Error Handling

```javascript
const pool = mysql2.createPool(config);

pool.on('error', (err) => {
    console.error('Pool error:', err);
    if (err.code === 'PROTOCOL_CONNECTION_LOST') {
        // Handle connection lost
    }
});

pool.on('connection', (connection) => {
    console.log('New connection as id ' + connection.threadId);
});
```

## 🧪 Testing

### Connection Test

```javascript
async function testConnection() {
    const connection = mysql2.promise.createConnection(testConfig);
    try {
        await connection.connect();
        const [rows] = await connection.query('SELECT 1 as test');
        assert.equal(rows[0].test, 1);
        console.log('Connection test passed');
    } finally {
        await connection.end();
    }
}
```

### Pool Test

```javascript
async function testPool() {
    const pool = mysql2.promise.createPool(testConfig);
    try {
        const [rows] = await pool.query('SELECT COUNT(*) as count FROM users');
        assert(typeof rows[0].count === 'number');
        console.log('Pool test passed');
    } finally {
        await pool.end();
    }
}
```

## 🔗 Integration with TurboScript

In your TurboScript Go application:

```go
import (
    "github.com/dop251/goja"
    "path/to/your/turbo_modules/mysql2"
)

func setupDatabase(runtime *goja.Runtime, loop EventLoopRunner) error {
    // Create and register the mysql2 module
    mysql2Module := mysql2.New(runtime, loop)
    return mysql2Module.Register()
}
```

In your TypeScript routes:

```typescript
import mysql2 from 'mysql2';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const pool = mysql2.promise.createPool({
        host: 'localhost',
        database: 'myapp',
        user: 'root',
        password: 'password'
    });

    try {
        const [rows] = await pool.query(
            'SELECT * FROM users WHERE id = ?',
            [event.pathParameters.id]
        );

        return {
            code: 200,
            response: {
                status: 'success',
                data: rows[0]
            }
        };
    } finally {
        await pool.end();
    }
};
```

## 📝 Migration from Node.js

This module is designed to be a drop-in replacement for the Node.js mysql2 library:

```javascript
// Node.js code works as-is
const mysql2 = require('mysql2');
// or
import mysql2 from 'mysql2';

// Both callback and promise APIs are supported
const connection = mysql2.createConnection(config);
const promiseConnection = mysql2.promise.createConnection(config);
```

## 🆚 Differences from Node.js mysql2

- **Native bindings**: Uses Go's go-sql-driver/mysql instead of native MySQL client
- **Event system**: Simplified event handling (some events may not be emitted)
- **Stream support**: Streaming queries are not implemented in this version
- **Custom types**: Limited custom type support compared to Node.js version
- **SSL options**: Simplified SSL configuration options
- **Character sets**: Limited character set support

## 🤝 Contributing

Contributions are welcome! Please:

1. Follow the existing code style
2. Add tests for new features
3. Update documentation
4. Ensure compatibility with Node.js mysql2 API

## 📄 License

This module is licensed under the MIT License. See the LICENSE file for details.

## 🔗 Related Links

- [Node.js mysql2 documentation](https://github.com/sidorares/node-mysql2)
- [MySQL documentation](https://dev.mysql.com/doc/)
- [Goja JavaScript VM](https://github.com/dop251/goja)
- [TurboScript Framework](https://github.com/daison12006013/turboscript)
