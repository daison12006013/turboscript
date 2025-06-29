# TurboScript Goja Modules

This directory contains reusable goja modules that can be used by the goja community. These modules are designed to work with the [goja JavaScript VM](https://github.com/dop251/goja) in Go applications.

## Available Modules

### @turboscript/argon2

A comprehensive Argon2 password hashing library for goja JavaScript VM. Provides secure password hashing using the Argon2 algorithm, compatible with Node.js argon2 library standards.

**Features:**

- ✅ **Argon2id, Argon2i, Argon2d** support
- ✅ **Async/Promise-based** and synchronous APIs
- ✅ **Node.js argon2 compatible** hash format
- ✅ **Configurable parameters** (memory, time, parallelism)
- ✅ **Secure defaults** following OWASP recommendations
- ✅ **Type-safe** with TypeScript definitions
- ✅ **Zero dependencies** (uses Go's crypto/rand and golang.org/x/crypto/argon2)

**Installation:**

```bash
# Copy the argon2 module to your project
cp -r turbo_modules/argon2 your_project/modules/
```

**Usage:**

```javascript
// Hash a password
const hash = await argon2.hash('password123');

// Verify a password
const isValid = await argon2.verify(hash, 'password123');
```

See the [argon2 README](./argon2/README.md) for detailed documentation.

### @turboscript/pg

A comprehensive PostgreSQL client library for goja JavaScript VM, fully compatible with the Node.js [pg](https://github.com/brianc/node-postgres) library API.

**Features:**

- ✅ **Full Node.js pg compatibility** - Drop-in replacement for existing code
- ✅ **Connection pooling** - Efficient connection management with configurable pool settings
- ✅ **Async/Promise support** - Modern async/await patterns with full Promise support
- ✅ **Type safety** - Complete TypeScript definitions included
- ✅ **Secure by default** - Parameterized queries prevent SQL injection
- ✅ **Transaction support** - Full transaction handling with commit/rollback
- ✅ **Event handling** - Complete event system for connection lifecycle
- ✅ **SSL support** - Configurable SSL/TLS connections
- ✅ **Zero dependencies** - Pure Go implementation using lib/pq driver

**Installation:**

```bash
# Copy the pg module to your project
cp -r turbo_modules/pg your_project/modules/
```

**Usage:**

```javascript
const pg = require('pg');

// Create a client
const client = new pg.Client({
    host: 'localhost',
    database: 'mydb',
    user: 'postgres',
    password: 'password'
});

await client.connect();
const result = await client.query('SELECT NOW()');
console.log(result.rows[0]);
await client.end();
```

See the [pg README](./pg/README.md) for detailed documentation.

### @turboscript/mysql2

A comprehensive MySQL client library for goja JavaScript VM, fully compatible with the Node.js [mysql2](https://github.com/sidorares/node-mysql2) library API.

**Features:**

- ✅ **Full Node.js mysql2 compatibility** - Drop-in replacement for existing code
- ✅ **Dual API support** - Both callback-based and Promise-based APIs
- ✅ **Connection pooling** - Efficient connection management with configurable pool settings
- ✅ **Prepared statements** - Support for execute() method with prepared statements
- ✅ **Transaction support** - Full transaction handling with commit/rollback
- ✅ **Type safety** - Complete TypeScript definitions included
- ✅ **Secure by default** - Parameterized queries prevent SQL injection
- ✅ **Event handling** - Complete event system for connection lifecycle
- ✅ **SSL support** - Configurable SSL/TLS connections
- ✅ **Zero dependencies** - Pure Go implementation using go-sql-driver/mysql

**Installation:**

```bash
# Copy the mysql2 module to your project
cp -r turbo_modules/mysql2 your_project/modules/
```

**Usage:**

```javascript
const mysql2 = require('mysql2');

// Promise-based API
const connection = mysql2.promise.createConnection({
    host: 'localhost',
    user: 'root',
    password: 'password',
    database: 'mydb'
});

await connection.connect();
const [rows, fields] = await connection.query('SELECT * FROM users');
console.log(rows);
await connection.end();
```

See the [mysql2 README](./mysql2/README.md) for detailed documentation.

## React SSR via Goja - Implementation Complete

### Overview

The React and ReactDOM modules provide complete React Server-Side Rendering (SSR) capability through the Goja JavaScript engine. The solution includes:

- **Full React 18+ implementation** with comprehensive hook system
- **ReactDOM with streaming SSR** capabilities
- **TypeScript definitions** for complete IDE support
- **Production-ready examples** demonstrating real-world usage

### Architecture

#### React Module (`react/`)

- **Core**: Complete React implementation with all modern hooks
- **Hooks**: useState, useEffect, useMemo, useCallback, useContext, useReducer, useRef
- **Context**: createContext with Provider/Consumer pattern
- **Utilities**: Children utilities, createElement, Fragment support
- **State Management**: Full component lifecycle and state management

#### ReactDOM Module (`react-dom/`)

- **SSR**: renderToString, renderToStaticMarkup for server-side rendering
- **Streaming**: renderToPipeableStream, renderToReadableStream for streaming SSR
- **Client**: createRoot, hydrateRoot for client-side hydration
- **Portals**: createPortal for advanced rendering patterns

### Key Features Implemented

#### React Core

- ✅ Complete hook system (useState, useEffect, useMemo, useCallback, useContext, useReducer, useRef)
- ✅ Context API with Provider/Consumer
- ✅ Fragment support
- ✅ Children utilities (count, map, forEach, toArray, only)
- ✅ Element creation and component lifecycle
- ✅ Deep equality checks for dependency arrays
- ✅ Error boundaries and error handling

#### ReactDOM SSR

- ✅ renderToString for basic SSR
- ✅ renderToStaticMarkup for static HTML
- ✅ renderToPipeableStream for streaming SSR
- ✅ renderToReadableStream for stream processing
- ✅ renderToNodeStream for Node.js compatibility
- ✅ createPortal for advanced rendering
- ✅ Client-side APIs (createRoot, hydrateRoot)
- ✅ Proper HTML attribute handling
- ✅ Self-closing tag support

#### Developer Experience

- ✅ Complete TypeScript definitions
- ✅ Comprehensive examples and documentation
- ✅ Error handling and debugging support
- ✅ Production-ready implementation
- ✅ Performance optimizations

### Integration with TurboScript

The React modules integrate seamlessly with TurboScript's server architecture for full SSR capability in your Go-based applications.

## Module Development Guidelines

### Structure

Each module should follow this structure:

```
module_name/
├── index.go          # Main Go implementation
├── index.d.ts        # TypeScript definitions
├── package.json      # Node.js package metadata
├── go.mod           # Go module definition
├── README.md        # Module documentation
├── example.js       # Usage examples
└── index_test.go    # Go tests
```

### Implementation Guidelines

1. **Compatibility**: Modules should maintain API compatibility with their Node.js counterparts
2. **Type Safety**: Always provide TypeScript definitions
3. **Documentation**: Include comprehensive README with examples
4. **Testing**: Provide test coverage for all functionality
5. **Error Handling**: Use appropriate error types and messages
6. **Performance**: Optimize for the goja runtime environment

### Contributing

When adding new modules:

1. Follow the established structure and naming conventions
2. Ensure full Node.js API compatibility where applicable
3. Provide comprehensive TypeScript definitions
4. Include detailed documentation and examples
5. Add thorough test coverage
6. Update this main README to include your module

## License

These modules are part of the TurboScript project and follow the same licensing terms.

````
