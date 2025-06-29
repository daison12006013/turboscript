# Goja Modules Architecture in TurboScript

## Overview

TurboScript uses a sophisticated module loading system that bridges Go and JavaScript through the **goja JavaScript engine**. This system allows TypeScript code to access Go-implemented functionality seamlessly through two primary mechanisms:

1. **Global Objects** - Direct runtime injection (e.g., `argon2`, `turboQuery`)
2. **Require Modules** - Node.js-style `require()` imports (e.g., `bcryptjs`, `crypto`)

## Architecture Flow

```text
TypeScript Code (app/routes/*.ts)
        ↓
Type Definitions (turbo_modules/*/index.d.ts + app/global.d.ts)
        ↓
Runtime Execution (internal/tsengine/eventloop_manager.go)
        ↓
Module Registration (internal/turbo_modules/* + internal/tsengine/*_utils.go)
        ↓
Go Implementation (Native Go functions)
```

## Module Loading Mechanisms

### 1. Global Object Injection

Global objects are directly injected into the JavaScript runtime and available without imports.

#### Example: Argon2 Module

**TypeScript Usage:**

```typescript
// No import needed - globally available
const hash = await argon2.hash('password');
const isValid = await argon2.verify(hash, 'password');
```

**Go Implementation Flow:**

```go
// 1. Module definition in internal/turbo_modules/argon2/index.go
type Argon2Module struct {
    runtime *goja.Runtime
    loop    EventLoopRunner
}

func (a *Argon2Module) Register() error {
    // Create the module object
    module := a.runtime.NewObject()

    // Add functions to the module
    module.Set("hash", a.hash)
    module.Set("verify", a.verify)

    // Inject as global object
    return a.runtime.Set("argon2", module)
}

// 2. Registration in internal/tsengine/eventloop_manager.go
func (elm *EventLoopManager) prepareVM(vm *goja.Runtime) error {
    // Initialize Argon2 module for password hashing
    argon2Module := argon2.New(vm, elm)
    if err := argon2Module.Register(); err != nil {
        return fmt.Errorf("failed to register argon2 module: %w", err)
    }
    return nil
}
```

**Type Definitions:**

```typescript
// turbo_modules/argon2/index.d.ts
declare global {
    const argon2: {
        hash(password: string, options?: Argon2Options): Promise<string>;
        verify(hash: string, password: string): Promise<boolean>;
        // ... more functions
    };
}

// app/global.d.ts
/// <reference path="../turbo_modules/argon2/index.d.ts" />
```

### 2. Require Module System

Modules registered with the require registry can be imported using Node.js-style `require()`.

#### Example: bcryptjs Module

**TypeScript Usage:**

```typescript
const bcrypt = require('bcryptjs');
const hash = bcrypt.hashSync('password', 10);
const isValid = bcrypt.compareSync('password', hash);
```

**Go Implementation Flow:**

```go
// 1. Module definition in internal/tsengine/bcrypt_utils.go
func RegisterSharedBcryptModule(rt *goja.Runtime, registry *require.Registry) {
    bcryptObj := rt.NewObject()

    // Add functions
    bcryptObj.Set("hashSync", func(password string, rounds int) string {
        // Implementation
    })

    // Register with require system
    registry.RegisterNativeModule("bcryptjs", func(_ *goja.Runtime, module *goja.Object) {
        module.Set("exports", bcryptObj)
    })
}

// 2. Registration in eventloop_manager.go
func (elm *EventLoopManager) prepareVM(vm *goja.Runtime) error {
    if elm.registry != nil {
        elm.registry.Enable(rt) // Enable require() function
        elm.registerBcryptModule(rt, elm.registry)
    }
    return nil
}
```

## Module Categories

### Core Runtime Modules (Global Objects)

These modules are automatically available in all TypeScript code without imports:

| Module | Global Object | Purpose | Implementation |
|--------|---------------|---------|----------------|
| **argon2** | `argon2` | Password hashing | `internal/turbo_modules/argon2/` |
| **turboQuery** | `turboQuery()` | Database queries | `internal/tsengine/turboquery_utils.go` |
| **turboEmail** | `turboEmail()` | Email sending | `internal/tsengine/turboemail_utils.go` |
| **turboJob** | `turboJob()` | Background jobs | `internal/tsengine/turbojob_utils.go` |
| **turboCache** | `turboCache()` | Caching operations | `internal/tsengine/turbocache_utils.go` |
| **turboBroadcast** | `turboBroadcastWebSocket()` | Real-time messaging | `internal/tsengine/` |

### Node.js Compatibility Modules (Require-able)

These modules follow Node.js patterns and must be imported with `require()`:

| Module | Import | Purpose | Implementation |
|--------|--------|---------|----------------|
| **bcryptjs** | `require('bcryptjs')` | Legacy password hashing | `internal/tsengine/bcrypt_utils.go` |
| **crypto** | `require('crypto')` | Cryptographic functions | `internal/tsengine/crypto_utils.go` |
| **fs** | `require('fs')` | File system operations | `internal/tsengine/nodejs_compat.go` |
| **path** | `require('path')` | Path utilities | `internal/tsengine/nodejs_compat.go` |
| **os** | `require('os')` | Operating system info | `internal/tsengine/nodejs_compat.go` |

## TypeScript Integration

### 1. Global Type Definitions

The `app/global.d.ts` file provides types for all global objects:

```typescript
// app/global.d.ts
/// <reference path="../turbo_modules/argon2/index.d.ts" />

declare global {
    // Global functions
    function turboQuery(query: string, params?: unknown[]): Promise<unknown[]>;
    function turboEmail(config: EmailConfig): Promise<void>;

    // Global objects are defined in their respective .d.ts files
}
```

### 2. Module-Specific Type Definitions

Each goja module has its own TypeScript definitions:

```typescript
// turbo_modules/argon2/index.d.ts
declare global {
    const argon2: {
        hash(password: string, options?: Argon2Options): Promise<string>;
        verify(hash: string, password: string): Promise<boolean>;
        hashSync(password: string, options?: Argon2Options): string;
        verifySync(hash: string, password: string): boolean;
    };
}
```

### 3. TypeScript Configuration

The `tsconfig.json` includes goja modules in the type resolution:

```jsonc
{
  "compilerOptions": {
    "typeRoots": ["./node_modules/@types", "./turbo_modules/", "./app"],
    "baseUrl": ".",
    "paths": {
      "@app/*": ["app/*"]
    }
  },
  "include": [
    "app/**/*",
    "turbo_modules/**/*.d.ts"
  ],
  "files": [
    "app/global.d.ts",
    "turbo_modules/argon2/index.d.ts",
    "turbo_modules/mysql2/index.d.ts",
    "turbo_modules/pg/index.d.ts"
  ]
}
```

## Runtime Initialization Process

### 1. Event Loop Creation

```go
// internal/tsengine/eventloop_manager.go
func NewEventLoopManagerWithServer(server interface{}) *EventLoopManager {
    // Create require registry
    registry := require.NewRegistry()

    // Create event loop with registry support
    loop := eventloop.NewEventLoop(
        eventloop.WithRegistry(registry),
        eventloop.EnableConsole(true),
    )

    return &EventLoopManager{
        loop:     loop,
        registry: registry,
        // ...
    }
}
```

### 2. Module Registration

```go
func (elm *EventLoopManager) prepareVM(vm *goja.Runtime) error {
    // Enable require() system
    if elm.registry != nil {
        elm.registry.Enable(rt)

        // Register require-able modules
        elm.registerBcryptModule(rt, elm.registry)
        elm.registerCryptoModule(rt, elm.registry)
    }

    // Register global objects
    argon2Module := argon2.New(vm, elm)
    if err := argon2Module.Register(); err != nil {
        return fmt.Errorf("failed to register argon2 module: %w", err)
    }

    // Register global functions
    elm.InjectAsyncTurboQuery(vm)
    elm.InjectAsyncTurboEmail(vm, elm.turboEmailUtils)
    elm.InjectAsyncTurboCache(vm, elm.turboCacheUtils)

    return nil
}
```

### 3. TypeScript Execution

```go
func (elm *EventLoopManager) RunTSCode(code string) (interface{}, error) {
    var result interface{}
    var execError error

    // Execute code in the prepared VM
    elm.loop.RunOnLoop(func(vm *goja.Runtime) {
        // VM already has all modules registered
        val, err := vm.RunString(code)
        if err != nil {
            execError = err
            return
        }
        result = val.Export()
    })

    return result, execError
}
```

## Creating New Modules

### Option 1: Global Object Module

**1. Create Go Implementation:**

```go
// internal/turbo_modules/mymodule/index.go
package mymodule

type MyModule struct {
    runtime *goja.Runtime
}

func New(runtime *goja.Runtime) *MyModule {
    return &MyModule{runtime: runtime}
}

func (m *MyModule) Register() error {
    module := m.runtime.NewObject()

    module.Set("myFunction", func(call goja.FunctionCall) goja.Value {
        // Implementation
        return m.runtime.ToValue("result")
    })

    return m.runtime.Set("myModule", module)
}
```

**2. Register in Event Loop:**

```go
// internal/tsengine/eventloop_manager.go
func (elm *EventLoopManager) prepareVM(vm *goja.Runtime) error {
    // ... existing code ...

    myModule := mymodule.New(vm)
    if err := myModule.Register(); err != nil {
        return fmt.Errorf("failed to register myModule: %w", err)
    }

    return nil
}
```

**3. Create Type Definitions:**

```typescript
// turbo_modules/mymodule/index.d.ts
declare global {
    const myModule: {
        myFunction(): string;
    };
}

export {};
```

**4. Reference in Global Types:**

```typescript
// app/global.d.ts
/// <reference path="../turbo_modules/mymodule/index.d.ts" />
```

### Option 2: Require-able Module

**1. Create Go Implementation:**

```go
// internal/tsengine/mymodule_utils.go
func RegisterMyModule(rt *goja.Runtime, registry *require.Registry) {
    moduleObj := rt.NewObject()

    moduleObj.Set("myFunction", func(arg string) string {
        // Implementation
        return "result: " + arg
    })

    registry.RegisterNativeModule("mymodule", func(_ *goja.Runtime, module *goja.Object) {
        module.Set("exports", moduleObj)
    })
}
```

**2. Register in Event Loop:**

```go
func (elm *EventLoopManager) prepareVM(vm *goja.Runtime) error {
    if elm.registry != nil {
        elm.registry.Enable(rt)
        RegisterMyModule(rt, elm.registry)
    }
    return nil
}
```

**3. Use in TypeScript:**

```typescript
const myModule = require('mymodule');
const result = myModule.myFunction('input');
```

## Best Practices

### 1. Choosing Module Type

- **Global Objects**: For core TurboScript functionality used frequently
- **Require Modules**: For Node.js compatibility or optional functionality

### 2. Error Handling

```go
// Always use proper error handling in Go implementations
func (m *MyModule) myFunction(call goja.FunctionCall) goja.Value {
    defer func() {
        if r := recover(); r != nil {
            if err, ok := r.(error); ok {
                panic(m.runtime.NewGoError(err))
            } else {
                panic(m.runtime.NewTypeError(fmt.Sprintf("panic: %v", r)))
            }
        }
    }()

    // Implementation
}
```

### 3. Async Operations

```go
// Use event loop for async operations
func (m *MyModule) asyncFunction(call goja.FunctionCall) goja.Value {
    promise, resolve, reject := m.runtime.NewPromise()

    go func() {
        defer func() {
            if r := recover(); r != nil {
                m.loop.RunOnLoop(func(*goja.Runtime) {
                    reject(fmt.Errorf("async error: %v", r))
                })
            }
        }()

        // Async work
        result := doAsyncWork()

        m.loop.RunOnLoop(func(*goja.Runtime) {
            resolve(m.runtime.ToValue(result))
        })
    }()

    return m.runtime.ToValue(promise)
}
```

### 4. Type Safety

```typescript
// Always provide comprehensive TypeScript definitions
interface MyModuleOptions {
    timeout?: number;
    retries?: number;
}

declare global {
    const myModule: {
        process(data: string, options?: MyModuleOptions): Promise<string>;
        processSync(data: string, options?: MyModuleOptions): string;
    };
}
```

## Debugging and Development

### 1. Module Loading Debug

```go
// Enable debug logging to see module registration
logger.Debug("Registering module: %s", moduleName)
```

### 2. Type Checking

```bash
# Verify TypeScript types
npx tsc --noEmit

# Check specific files
npx tsc --noEmit app/routes/my-route.ts
```

### 3. Runtime Testing

```typescript
// Test module availability in TypeScript
console.log('argon2 available:', typeof argon2);
console.log('turboQuery available:', typeof turboQuery);

// Test require modules
try {
    const bcrypt = require('bcryptjs');
    console.log('bcryptjs available:', typeof bcrypt);
} catch (e) {
    console.log('bcryptjs not available:', e.message);
}
```

## Summary

TurboScript's goja module system provides a seamless bridge between Go and TypeScript through:

1. **Direct Global Injection** - For core framework functionality
2. **Require Registry** - For Node.js compatibility
3. **TypeScript Integration** - Complete type safety
4. **Event Loop Management** - Proper async handling

This architecture allows TypeScript code to access powerful Go functionality while maintaining type safety and familiar JavaScript patterns.
