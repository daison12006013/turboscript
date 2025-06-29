# What is TurboScript?

## Overview

TurboScript is a **hybrid web framework** that combines the best of both worlds: **TypeScript for business logic** and **Go for runtime execution**. Think of it as a way to write APIs in TypeScript while getting the performance and reliability of Go.

## Why TurboScript?

### 🚀 **Developer Experience**

- Write your API logic in **TypeScript** (familiar to web developers)
- Get **instant hot reloading** during development
- Full **type safety** with IntelliSense support
- **Zero compilation step** - your TypeScript runs directly

### ⚡ **Performance**

- **Go runtime** handles execution (lightning fast)
- **FastHTTP** for maximum throughput
- **Async database operations** with Promise.all support
- **Built-in caching** with multiple drivers (Redis, Memory, File)

### 🔒 **Security Built-in**

- **JWT authentication** out of the box
- **SQL injection protection** automatically
- **Input validation** helpers
- **CORS and security headers** configured

## How It Works (Simple Explanation)

```typescript
// You write TypeScript like this:
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const users = await turboQuery('SELECT * FROM users WHERE active = $1', [true]);

    return {
        code: 200,
        response: { users }
    };
};
```

**Behind the scenes:**

1. TurboScript compiles your TypeScript to JavaScript
2. Go runtime executes it using a JavaScript VM (Goja)
3. Database queries run through Go's optimized connections
4. Response is returned with Go's performance

## Key Concepts

### 📁 **File-Based Routing**

Your file structure **IS** your API structure:

```text
app/routes/
  users/
    create.ts     → POST /users
    list.ts       → GET /users
    {uid}.ts      → GET /users/123
```

### 🗄️ **Async Database Operations**

Direct database access with modern async/await:

```typescript
// Multiple queries in parallel
const [users, stats, logs] = await Promise.all([
    turboQuery('SELECT * FROM users'),
    turboQuery('SELECT COUNT(*) FROM sessions'),
    turboQuery('SELECT * FROM activity_logs LIMIT 10')
]);
```

## What Makes TurboScript Different?

| Feature | Traditional Node.js | TurboScript |
|---------|-------------------|-------------|
| **Runtime** | JavaScript V8 | Go + JavaScript VM |
| **Performance** | Good | Excellent (Go speed) |
| **Memory Usage** | High | Low (Go efficiency) |
| **Deployment** | Complex (Node + deps) | Single binary |
| **Database** | ORM complexity | Direct SQL with safety |
| **Hot Reload** | Restart required | Instant (Go + TS) |

## Who Should Use TurboScript?

### ✅ **Perfect For:**

- API-first applications
- Microservices that need performance
- Teams familiar with TypeScript/JavaScript
- Projects requiring fast database operations
- Applications needing built-in auth and security

### 🤔 **Consider Alternatives If:**

- You need extensive frontend templating
- Your team is purely backend-focused (use pure Go)
- You're building simple CRUD with existing frameworks

## Ready to Start?

**Next Steps:**

1. [Install TurboScript](getting-started.md) - Get up and running in 5 minutes
2. [Create Your First API](first-api-endpoint.md) - Build a simple endpoint
3. [Follow the Learning Path](tutorial-todo-app.md) - Build a complete app

---

**Questions?** Check our [FAQ](../guides/troubleshooting.md) or [join the community](https://github.com/daison12006013/turboscript/discussions).
