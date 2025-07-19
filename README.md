# TurboScript

A hybrid web framework that combines TypeScript for business logic and Go for runtime execution. TurboScript uses JavaScript VM (goja) to execute TypeScript code at runtime, providing a unique development experience where TypeScript defines the API logic and Go handles the execution engine.

## 💡 Why TurboScript?

TurboScript was born from a real-world pain point: while building APIs on AWS Lambda with Node.js, I constantly ran into high memory usage and slow cold starts. In contrast, my experience with Go showed me how fast and efficient backend services could be—yet, in my company, Go adoption was a challenge since most developers were comfortable with Node.js and TypeScript, not Go.

I wanted to bring Go's performance and efficiency to teams who prefer TypeScript, without forcing everyone to learn a new language or toolchain. TurboScript lets you write API logic in TypeScript—using familiar patterns and strict typing—while the Go runtime (powered by FastHTTP and goja) delivers maximum speed and concurrency.

With TurboScript, you get:

- The productivity and safety of TypeScript
- The raw performance and minimal resource usage of Go
- Seamless async database access and modern API patterns
- No need to retrain your team or abandon TypeScript

TurboScript is designed for TypeScript developers who want to build APIs that are both fast and enjoyable to write, while finally unlocking the performance benefits of Go.

## ✨ Features

- **Hybrid Architecture**: TypeScript for business logic, Go for performance
- **Type-Safe Development**: Full TypeScript support with global type definitions
- **Intelligent File Resolution**: Automatic .ts file resolution for seamless development-to-production workflow
- **Wildcard Routing**: Dynamic file-based routing with /* patterns for flexible endpoint organization
- **Security First**: Built-in SQL injection protection and table access restrictions
- **Hot Reloading**: Development environment with automatic TypeScript compilation
- **High Performance**: FastHTTP-based server with optimized runtime execution
- **JWT Authentication**: Built-in authentication utilities and cookie management
- **Database Integration**: Secure PostgreSQL integration with query protection
- **Dynamic Database Operations**: `turboQuery()` function for direct database updates from route handlers
- **Multi-Platform**: Support for Linux and macOS binary generation
- **Distribution Packaging**: Automated dist folder generation with runner scripts

## ⚡ Performance

TurboScript delivers exceptional performance with minimal resource usage:

### Resource Usage

- **Memory**: Only **12.1MB** RAM usage
- **Platform**: Tested on Apple MacBook Pro M3 with 36GB RAM

### Endpoint Response Times

Latest E2E benchmark results (`make test-e2e-bench`):

| Endpoint                        | Avg Response Time | Requests/sec | Memory/Allocations |
|----------------------------------|------------------|--------------|-------------------|
| Root Endpoint (JSON)             | **0.92ms**       | 1298 req/s   | 18.6KB / 140      |
| Root Endpoint (HTML)             | **1.60ms**       | 715 req/s    | 19.1KB / 145      |
| Authenticated Endpoint           | **15.35ms**      | 73 req/s     | 19.4KB / 146      |

*Benchmarks run on Apple M3 Pro (darwin/arm64) using Go 1.23.10. See `internal/tests/` for details.*

**Note**: Authenticated endpoints include JWT verification and database queries, resulting in higher response times and allocations.
