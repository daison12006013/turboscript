/*
 * TurboScript - A hybrid web framework combining TypeScript and Go
 *
 * Copyright (c) 2025 TurboScript Project Contributors
 * Author: Daison Cariño <daison12006013@gmail.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Based on TurboScript: https://github.com/daison12006013/turboscript
 */

// Package tsengine provides event loop management for async operations.
package tsengine

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/daison12006013/turboscript/turbo_modules/argon2"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
)

// EventLoopManager manages goja event loops for async operations.
type EventLoopManager struct {
	loop            *eventloop.EventLoop
	db              *sql.DB
	dbManager       any
	registry        *require.Registry
	turboJobUtils   *TurboJobUtils
	turboEmailUtils *TurboEmailUtils
	turboCacheUtils *TurboCacheUtils
	stackTraceUtils *StackTraceUtils
	server          interface{} // Server interface to avoid import cycles
}

// NewEventLoopManager creates a new event loop manager.
func NewEventLoopManager() *EventLoopManager {
	return NewEventLoopManagerWithServer(nil)
}

// NewEventLoopManagerWithServer creates a new event loop manager with server reference.
func NewEventLoopManagerWithServer(server interface{}) *EventLoopManager {
	// Create event loop with require registry support
	registry := require.NewRegistry()
	loop := eventloop.NewEventLoop(
		eventloop.WithRegistry(registry),
		eventloop.EnableConsole(false), // Disable default console, we'll use our custom one
	)

	elm := &EventLoopManager{
		loop:            loop,
		registry:        registry,
		turboCacheUtils: nil, // Will be set later via SetTurboCacheUtils when cache is initialized
		stackTraceUtils: NewStackTraceUtils(),
		server:          server,
	}

	return elm
}

// SetDatabase sets the database connection.
func (elm *EventLoopManager) SetDatabase(db *sql.DB) {
	elm.db = db
}

// SetDatabaseManager sets the database manager for multi-connection support.
func (elm *EventLoopManager) SetDatabaseManager(dbManager any) {
	elm.dbManager = dbManager
}

// SetServer sets the server reference for broadcasting.
func (elm *EventLoopManager) SetServer(server interface{}) {
	elm.server = server
}

// SetTurboJobUtils sets the turboJob utilities.
func (elm *EventLoopManager) SetTurboJobUtils(turboJobUtils *TurboJobUtils) {
	elm.turboJobUtils = turboJobUtils
}

// SetTurboEmailUtils sets the turboEmail utilities.
func (elm *EventLoopManager) SetTurboEmailUtils(turboEmailUtils *TurboEmailUtils) {
	elm.turboEmailUtils = turboEmailUtils
}

// SetTurboCacheUtils sets the turboCache utilities.
func (elm *EventLoopManager) SetTurboCacheUtils(turboCacheUtils *TurboCacheUtils) {
	elm.turboCacheUtils = turboCacheUtils
}

// Start starts the event loop in background.
func (elm *EventLoopManager) Start() {
	elm.loop.Start()
}

// Stop stops the event loop.
func (elm *EventLoopManager) Stop() {
	elm.loop.Stop()
}

// Terminate terminates the event loop and cleans up all resources.
func (elm *EventLoopManager) Terminate() {
	elm.loop.Terminate()
}

// RunOnLoop executes a function in the event loop context.
func (elm *EventLoopManager) RunOnLoop(fn func(*goja.Runtime)) bool {
	return elm.loop.RunOnLoop(func(rt *goja.Runtime) {
		// Always ensure the runtime has Node.js globals - this is necessary
		// because each execution may be on a fresh runtime context
		if elm.registry != nil {
			// Enable require() function and console
			elm.registry.Enable(rt)
			// Register bcryptjs module for password utilities
			elm.registerBcryptModule(rt, elm.registry)
			// Register shared crypto module for hashing and signing
			elm.registerCryptoModule(rt, elm.registry)
		}

		// Explicitly add module and exports globals that might be missing
		moduleObj := rt.NewObject()
		exportsObj := rt.NewObject()

		// Set module.exports to exports
		if err := moduleObj.Set("exports", exportsObj); err != nil {
			logger.Error("Failed to set module.exports: %v", err)
		}

		// Set the globals
		if err := rt.Set("module", moduleObj); err != nil {
			logger.Error("Failed to set module global: %v", err)
		}
		if err := rt.Set("exports", exportsObj); err != nil {
			logger.Error("Failed to set exports global: %v", err)
		}

		// Run the actual function
		fn(rt)
	})
}

// turboQueryAsync creates an async version of turboQuery that returns a Promise.
func (elm *EventLoopManager) turboQueryAsync(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	if elm.db == nil {
		panic(rt.NewGoError(fmt.Errorf("database connection not available")))
	}

	// Get query and params from the call
	if len(call.Arguments) == 0 {
		panic(rt.NewGoError(fmt.Errorf("turboQuery requires at least 1 argument (query)")))
	}

	query := call.Argument(0).String()
	var params []any

	if len(call.Arguments) > 1 && !goja.IsUndefined(call.Argument(1)) && !goja.IsNull(call.Argument(1)) {
		paramsValue := call.Argument(1).Export()
		if paramsSlice, ok := paramsValue.([]any); ok {
			params = paramsSlice
		} else if paramsValue != nil {
			params = []any{paramsValue}
		}
	}

	logger.Debug("turboQueryAsync called with query: %s, params: %+v", query, params)

	// Create a new Promise and get resolve/reject functions
	promise, resolve, reject := rt.NewPromise()

	// Execute the database query in a goroutine
	go func() {
		rows, err := elm.db.Query(query, params...)
		if err != nil {
			// Schedule error resolution on the event loop
			elm.loop.RunOnLoop(func(*goja.Runtime) {
				detailedErr := elm.stackTraceUtils.LogAndWrapError(
					err,
					ErrorTypeDatabaseQuery,
					"Database query execution failed",
					map[string]any{
						"query":        query,
						"params":       params,
						"params_count": len(params),
					},
				)
				_ = reject(detailedErr)
			})
			return
		}

		// Process results in background
		results := make([]map[string]any, 0)
		columns, err := rows.Columns()
		if err != nil {
			_ = rows.Close()
			elm.loop.RunOnLoop(func(*goja.Runtime) {
				detailedErr := elm.stackTraceUtils.LogAndWrapError(
					err,
					ErrorTypeDatabaseQuery,
					"Failed to get database column information",
					map[string]any{
						"query":  query,
						"params": params,
					},
				)
				_ = reject(detailedErr)
			})
			return
		}

		for rows.Next() {
			values := make([]any, len(columns))
			valuePtrs := make([]any, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				_ = rows.Close()
				elm.loop.RunOnLoop(func(*goja.Runtime) {
					detailedErr := elm.stackTraceUtils.LogAndWrapError(
						err,
						ErrorTypeDatabaseQuery,
						"Failed to scan database row",
						map[string]any{
							"query":        query,
							"params":       params,
							"columns":      columns,
							"column_count": len(columns),
						},
					)
					_ = reject(detailedErr)
				})
				return
			}

			rowMap := make(map[string]any)
			for i, col := range columns {
				val := values[i]
				if b, ok := val.([]byte); ok {
					val = string(b)
				}
				rowMap[col] = val
			}
			results = append(results, rowMap)
		}

		// Close rows after processing
		if closeErr := rows.Close(); closeErr != nil {
			logger.Error("Failed to close database rows: %v", closeErr)
		}

		if err := rows.Err(); err != nil {
			elm.loop.RunOnLoop(func(*goja.Runtime) {
				detailedErr := elm.stackTraceUtils.LogAndWrapError(
					err,
					ErrorTypeDatabaseQuery,
					"Error occurred during database rows iteration",
					map[string]any{
						"query":          query,
						"params":         params,
						"rows_processed": len(results),
					},
				)
				_ = reject(detailedErr)
			})
			return
		}

		// Schedule successful resolution on the event loop
		elm.loop.RunOnLoop(func(*goja.Runtime) {
			_ = resolve(results)
		})
	}()

	return rt.ToValue(promise)
}

// InjectAsyncTurboQuery injects the async turboQuery function into a runtime.
func (elm *EventLoopManager) InjectAsyncTurboQuery(rt *goja.Runtime) error {
	return rt.Set("turboQuery", func(call goja.FunctionCall) goja.Value {
		return elm.turboQueryAsync(call, rt)
	})
}

// InjectAsyncTurboJob injects the async turboJob function into a runtime.
func (elm *EventLoopManager) InjectAsyncTurboJob(rt *goja.Runtime) error {
	if elm.turboJobUtils == nil {
		return nil // Skip if job utils not configured
	}
	return rt.Set("turboJob", func(call goja.FunctionCall) goja.Value {
		return elm.turboJobAsync(call, rt)
	})
}

// InjectAsyncTurboEmail injects the async turboEmail function into a runtime.
func (elm *EventLoopManager) InjectAsyncTurboEmail(rt *goja.Runtime) error {
	if elm.turboEmailUtils == nil {
		return nil // Skip if email utils not configured
	}
	return rt.Set("turboEmail", func(call goja.FunctionCall) goja.Value {
		return elm.turboEmailAsync(call, rt)
	})
}

// InjectAsyncTurboCache injects the async turboCache function into a runtime.
func (elm *EventLoopManager) InjectAsyncTurboCache(rt *goja.Runtime, cacheUtils *TurboCacheUtils) error {
	cacheObj := rt.NewObject()

	mergeOptions := func(base map[string]any, opts ...any) map[string]any {
		out := make(map[string]any)
		for k, v := range base {
			out[k] = v
		}
		if len(opts) > 0 {
			if userOpts, ok := opts[0].(map[string]any); ok {
				for k, v := range userOpts {
					out[k] = v
				}
			}
		}
		return out
	}

	cacheCall := func(args ...goja.Value) goja.FunctionCall {
		return goja.FunctionCall{Arguments: args}
	}

	_ = cacheObj.Set("get", func(key string, opts ...any) goja.Value {
		options := mergeOptions(map[string]any{"op": "get"}, opts...)
		call := cacheCall(rt.ToValue(key), rt.ToValue(options))
		return cacheUtils.turboCacheAsync(call, rt, elm)
	})
	_ = cacheObj.Set("set", func(key string, value any, ttlSeconds int64, opts ...any) goja.Value {
		options := mergeOptions(map[string]any{"op": "set"}, opts...)
		call := cacheCall(rt.ToValue(key), rt.ToValue(value), rt.ToValue(ttlSeconds), rt.ToValue(options))
		return cacheUtils.turboCacheAsync(call, rt, elm)
	})
	_ = cacheObj.Set("del", func(key string, opts ...any) goja.Value {
		options := mergeOptions(map[string]any{"op": "del"}, opts...)
		options["op"] = "del" // Ensure op is always del
		call := cacheCall(rt.ToValue(key), rt.ToValue(options))
		return cacheUtils.turboCacheAsync(call, rt, elm)
	})
	_ = cacheObj.Set("has", func(key string, opts ...any) goja.Value {
		options := mergeOptions(map[string]any{"op": "has"}, opts...)
		call := cacheCall(rt.ToValue(key), rt.ToValue(options))
		return cacheUtils.turboCacheAsync(call, rt, elm)
	})
	_ = cacheObj.Set("flush", func(opts ...any) goja.Value {
		options := mergeOptions(map[string]any{"op": "flush"}, opts...)
		call := cacheCall(rt.ToValue(options))
		return cacheUtils.turboCacheAsync(call, rt, elm)
	})
	if err := rt.Set("turboCache", cacheObj); err != nil {
		return err
	}
	return rt.Set("turboCacheImpl", cacheObj)
}

// InjectAsyncTurboBroadcast injects the turboBroadcastWebSocket and turboBroadcastSSE functions.
func (elm *EventLoopManager) InjectAsyncTurboBroadcast(rt *goja.Runtime) error {
	// turboBroadcastWebSocket function
	if err := rt.Set("turboBroadcastWebSocket", func(call goja.FunctionCall) goja.Value {
		return elm.turboBroadcastWebSocketAsync(call, rt)
	}); err != nil {
		return err
	}

	// turboBroadcastSSE function
	if err := rt.Set("turboBroadcastSSE", func(call goja.FunctionCall) goja.Value {
		return elm.turboBroadcastSSEAsync(call, rt)
	}); err != nil {
		return err
	}

	// turboGetConnections function
	return rt.Set("turboGetConnections", func(call goja.FunctionCall) goja.Value {
		return elm.turboGetConnectionsAsync(call, rt)
	})
}

// turboBroadcastWebSocketAsync broadcasts a WebSocket message asynchronously.
func (elm *EventLoopManager) turboBroadcastWebSocketAsync(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	if len(call.Arguments) < 1 {
		panic(rt.NewTypeError("turboBroadcastWebSocket requires 1 argument"))
	}

	// Create a promise for async execution
	promise, resolve, reject := rt.NewPromise()

	elm.RunOnLoop(func(vm *goja.Runtime) {
		defer func() {
			if r := recover(); r != nil {
				if err := reject(vm.NewGoError(fmt.Errorf("turboBroadcastWebSocket error: %v", r))); err != nil {
					logger.Error("Failed to reject turboBroadcastWebSocket promise: %v", err)
				}
			}
		}()

		// Parse the message argument
		msgArg := call.Arguments[0]
		msgObj := msgArg.ToObject(vm)

		msgType := ""
		if typeVal := msgObj.Get("type"); typeVal != nil && !goja.IsUndefined(typeVal) {
			msgType = typeVal.String()
		}
		if msgType == "" {
			msgType = "broadcast"
		}

		room := ""
		if roomVal := msgObj.Get("room"); roomVal != nil && !goja.IsUndefined(roomVal) {
			room = roomVal.String()
		}

		target := ""
		if targetVal := msgObj.Get("target"); targetVal != nil && !goja.IsUndefined(targetVal) {
			target = targetVal.String()
		}

		// Parse data object
		data := make(map[string]interface{})
		if dataVal := msgObj.Get("data"); dataVal != nil && !goja.IsUndefined(dataVal) {
			dataObj := dataVal.ToObject(vm)
			data = dataObj.Export().(map[string]interface{})
		}

		// Add timestamp and server_sent if not present
		if _, exists := data["timestamp"]; !exists {
			data["timestamp"] = time.Now().UTC().Format(time.RFC3339)
		}
		if _, exists := data["server_sent"]; !exists {
			data["server_sent"] = true
		}

		// Call server broadcasting method using reflection
		connections := 0
		if elm.server != nil {
			if serverWithMethod, ok := elm.server.(interface {
				BroadcastToRoom(string, string, map[string]interface{}) int
				BroadcastToConnection(string, string, map[string]interface{}) int
				BroadcastToAll(string, map[string]interface{}) int
			}); ok {
				if target != "" {
					connections = serverWithMethod.BroadcastToConnection(target, msgType, data)
				} else if room != "" {
					connections = serverWithMethod.BroadcastToRoom(room, msgType, data)
				} else {
					connections = serverWithMethod.BroadcastToAll(msgType, data)
				}
			}
		}

		// Create result object
		result := vm.NewObject()
		_ = result.Set("success", true)
		_ = result.Set("connections_notified", connections)
		_ = result.Set("message_type", msgType)
		_ = result.Set("room", room)
		_ = result.Set("target", target)

		if err := resolve(result); err != nil {
			logger.Error("Failed to resolve turboBroadcastWebSocket: %v", err)
		}
	})

	return rt.ToValue(promise)
}

// turboBroadcastSSEAsync broadcasts an SSE message asynchronously.
func (elm *EventLoopManager) turboBroadcastSSEAsync(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	if len(call.Arguments) < 1 {
		panic(rt.NewTypeError("turboBroadcastSSE requires 1 argument"))
	}

	// Create a promise for async execution
	promise, resolve, reject := rt.NewPromise()

	elm.RunOnLoop(func(vm *goja.Runtime) {
		defer func() {
			if r := recover(); r != nil {
				if err := reject(vm.NewGoError(fmt.Errorf("turboBroadcastSSE error: %v", r))); err != nil {
					logger.Error("Failed to reject turboBroadcastSSE promise: %v", err)
				}
			}
		}()

		// Parse the message argument
		msgArg := call.Arguments[0]
		msgObj := msgArg.ToObject(vm)

		event := ""
		if eventVal := msgObj.Get("event"); eventVal != nil && !goja.IsUndefined(eventVal) {
			event = eventVal.String()
		}
		if event == "" {
			event = "message"
		}

		target := ""
		if targetVal := msgObj.Get("target"); targetVal != nil && !goja.IsUndefined(targetVal) {
			target = targetVal.String()
		}

		userID := ""
		if userIDVal := msgObj.Get("user_id"); userIDVal != nil && !goja.IsUndefined(userIDVal) {
			userID = userIDVal.String()
		}

		messageID := ""
		if idVal := msgObj.Get("id"); idVal != nil && !goja.IsUndefined(idVal) {
			messageID = idVal.String()
		}
		if messageID == "" {
			messageID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
		}

		// Parse data object
		data := make(map[string]interface{})
		if dataVal := msgObj.Get("data"); dataVal != nil && !goja.IsUndefined(dataVal) {
			dataObj := dataVal.ToObject(vm)
			data = dataObj.Export().(map[string]interface{})
		}

		// Add timestamp and server_sent if not present
		if _, exists := data["timestamp"]; !exists {
			data["timestamp"] = time.Now().UTC().Format(time.RFC3339)
		}
		if _, exists := data["server_sent"]; !exists {
			data["server_sent"] = true
		}

		// Call server broadcasting method using reflection
		connections := 0
		if elm.server != nil {
			if serverWithMethod, ok := elm.server.(interface {
				BroadcastSSEToConnection(string, string, map[string]interface{}, string) int
				BroadcastSSEToUser(string, string, map[string]interface{}, string) int
				BroadcastSSEToAll(string, map[string]interface{}, string) int
			}); ok {
				if target != "" {
					connections = serverWithMethod.BroadcastSSEToConnection(target, event, data, messageID)
				} else if userID != "" {
					connections = serverWithMethod.BroadcastSSEToUser(userID, event, data, messageID)
				} else {
					connections = serverWithMethod.BroadcastSSEToAll(event, data, messageID)
				}
			}
		}

		// Create result object
		result := vm.NewObject()
		_ = result.Set("success", true)
		_ = result.Set("connections_notified", connections)
		_ = result.Set("event", event)
		_ = result.Set("message_id", messageID)
		_ = result.Set("user_id", userID)
		_ = result.Set("target", target)

		if err := resolve(result); err != nil {
			logger.Error("Failed to resolve turboBroadcastSSE: %v", err)
		}
	})

	return rt.ToValue(promise)
}

// turboGetConnectionsAsync gets connection statistics asynchronously.
func (elm *EventLoopManager) turboGetConnectionsAsync(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	// Create a promise for async execution
	promise, resolve, reject := rt.NewPromise()

	elm.RunOnLoop(func(vm *goja.Runtime) {
		defer func() {
			if r := recover(); r != nil {
				if err := reject(vm.NewGoError(fmt.Errorf("turboGetConnections error: %v", r))); err != nil {
					logger.Error("Failed to reject turboGetConnections promise: %v", err)
				}
			}
		}()

		filter := ""
		if len(call.Arguments) > 0 {
			filter = call.Arguments[0].String()
		}

		// Call server method using reflection
		stats := make(map[string]interface{})
		if elm.server != nil {
			if serverWithMethod, ok := elm.server.(interface {
				GetConnectionStats(string) map[string]interface{}
			}); ok {
				stats = serverWithMethod.GetConnectionStats(filter)
			}
		}

		// Convert to goja value
		result := vm.ToValue(stats)
		if err := resolve(result); err != nil {
			logger.Error("Failed to resolve turboGetConnections: %v", err)
		}
	})

	return rt.ToValue(promise)
}

// RunAsyncWithCompletionTimeout executes TypeScript code and waits for async completion with specified timeout.
func (elm *EventLoopManager) RunAsyncWithCompletionTimeout(tsCode string, setupFn func(*goja.Runtime), timeoutSeconds int) (any, error) {
	var err error
	started := make(chan struct{})

	// Start the async execution
	elm.RunOnLoop(func(vm *goja.Runtime) {
		defer func() {
			if r := recover(); r != nil {
				if gojaErr, ok := r.(*goja.Exception); ok {
					err = fmt.Errorf("JavaScript error: %s", gojaErr.String())
				} else {
					err = fmt.Errorf("panic: %v", r)
				}
			}
			close(started)
		}()

		// Setup the async execution environment
		if setupErr := elm.setupAsyncExecution(vm, setupFn); setupErr != nil {
			err = setupErr
			return
		}

		// Run the TypeScript code
		_, runErr := vm.RunString(tsCode)
		if runErr != nil {
			err = runErr
		}
	})

	// Wait for execution to start
	<-started
	if err != nil {
		return nil, err
	}

	// Poll for completion outside of the event loop with configurable timeout
	return elm.pollForCompletionWithTimeout(timeoutSeconds)
}

// pollForCompletionWithTimeout polls for async completion with configurable timeout.
func (elm *EventLoopManager) pollForCompletionWithTimeout(timeoutSeconds int) (any, error) {
	// First check immediately - most operations complete instantly
	result, complete, checkErr := elm.checkAsyncCompletion()
	if checkErr != nil {
		return nil, checkErr
	}
	if complete {
		return result, nil
	}

	// Calculate adaptive polling strategy based on timeout
	// For short timeouts (≤10s): aggressive polling with 10ms intervals
	// For longer timeouts (>10s): progressive backoff to conserve CPU

	var sleepDuration time.Duration
	var maxSleep time.Duration
	var adaptiveThreshold int

	if timeoutSeconds <= 10 {
		// Short timeout: use consistent 10ms polling for responsiveness
		sleepDuration = 10 * time.Millisecond
		maxSleep = 10 * time.Millisecond
		adaptiveThreshold = timeoutSeconds * 100 // Never adapt for short timeouts
	} else {
		// Longer timeout: start with 10ms, increase to 100ms after 1 second
		sleepDuration = 10 * time.Millisecond
		maxSleep = 100 * time.Millisecond
		adaptiveThreshold = 100 // Adapt after 1 second (100 iterations)
	}

	maxTime := time.Duration(timeoutSeconds) * time.Second
	startTime := time.Now()
	iteration := 0

	for time.Since(startTime) < maxTime {
		time.Sleep(sleepDuration)
		iteration++

		// Check completion status
		result, complete, checkErr := elm.checkAsyncCompletion()
		if checkErr != nil {
			return nil, checkErr
		}

		if complete {
			// Log completion for debugging long operations
			elapsed := time.Since(startTime)
			if elapsed > time.Second {
				logger.Debug("Async operation completed after %d iterations (%.1f seconds)", iteration, elapsed.Seconds())
			}
			return result, nil
		}

		// Progressive backoff for longer timeouts to reduce CPU usage
		if iteration > adaptiveThreshold && sleepDuration < maxSleep {
			sleepDuration *= 2
			if sleepDuration > maxSleep {
				sleepDuration = maxSleep
			}
		}
	}

	elapsed := time.Since(startTime)
	return nil, fmt.Errorf("async execution timeout - operation did not complete after %d iterations (%.1f seconds)", iteration, elapsed.Seconds())
}

// SetTimeout provides setTimeout functionality.
func (elm *EventLoopManager) SetTimeout(fn func(*goja.Runtime), timeout time.Duration) *eventloop.Timer {
	return elm.loop.SetTimeout(fn, timeout)
}

// SetInterval provides setInterval functionality.
func (elm *EventLoopManager) SetInterval(fn func(*goja.Runtime), interval time.Duration) *eventloop.Interval {
	return elm.loop.SetInterval(fn, interval)
}

// ClearTimeout clears a timeout.
func (elm *EventLoopManager) ClearTimeout(timer *eventloop.Timer) {
	elm.loop.ClearTimeout(timer)
}

// ClearInterval clears an interval.
func (elm *EventLoopManager) ClearInterval(interval *eventloop.Interval) {
	elm.loop.ClearInterval(interval)
}

// registerBcryptModule registers a bcryptjs-compatible module in the goja runtime.
// This implementation delegates to the shared bcrypt utilities for consistency.
func (elm *EventLoopManager) registerBcryptModule(rt *goja.Runtime, registry *require.Registry) {
	RegisterSharedBcryptModule(rt, registry)
}

// registerCryptoModule registers a crypto module in the goja runtime.
// This implementation provides a consistent crypto interface for hashing and signing.
func (elm *EventLoopManager) registerCryptoModule(rt *goja.Runtime, registry *require.Registry) {
	RegisterSharedCryptoModule(rt, registry)
}

// setupAsyncExecution initializes the async execution environment.
//
// This method prepares a JavaScript runtime for async execution by:
//   - Calling optional setup function to inject additional utilities
//   - Injecting the async turboQuery function for database operations
//   - Initializing completion tracking variables for result monitoring
//
// Parameters:
//   - vm: The JavaScript runtime to configure
//   - setupFn: Optional function to inject additional utilities into the runtime
//
// Returns an error if runtime setup fails.
func (elm *EventLoopManager) setupAsyncExecution(vm *goja.Runtime, setupFn func(*goja.Runtime)) error {
	// Register our custom console first
	RegisterCustomConsole(vm)

	// Setup function can inject additional utilities
	if setupFn != nil {
		setupFn(vm)
	}

	// Inject async turboQuery function
	if injectErr := elm.InjectAsyncTurboQuery(vm); injectErr != nil {
		return fmt.Errorf("failed to inject turboQuery: %w", injectErr)
	}

	// Inject async turboJob function
	if injectErr := elm.InjectAsyncTurboJob(vm); injectErr != nil {
		return fmt.Errorf("failed to inject turboJob: %w", injectErr)
	}

	// Inject async turboEmail function
	if injectErr := elm.InjectAsyncTurboEmail(vm); injectErr != nil {
		return fmt.Errorf("failed to inject turboEmail: %w", injectErr)
	}

	// Inject async turboCache function
	if injectErr := elm.InjectAsyncTurboCache(vm, elm.turboCacheUtils); injectErr != nil {
		return fmt.Errorf("failed to inject turboCache: %w", injectErr)
	}

	// Initialize Argon2 module for password hashing
	argon2Module := argon2.New(vm, elm)
	if injectErr := argon2Module.Register(); injectErr != nil {
		return fmt.Errorf("failed to register argon2 module: %w", injectErr)
	}

	// Initialize completion tracking variables
	_ = vm.Set("__turboscript_result", goja.Null())
	_ = vm.Set("__turboscript_error", goja.Null())
	_ = vm.Set("__turboscript_complete", false)

	return nil
}

// checkAsyncCompletion checks if async execution has completed and returns result/error.
//
// This method polls the JavaScript runtime to determine if async operations have finished
// by checking the completion tracking variables set during execution. It safely accesses
// the runtime through the event loop to avoid race conditions.
//
// Returns:
//   - result: The execution result if completion was successful
//   - complete: Boolean indicating whether execution has finished
//   - err: Any error that occurred during execution or completion checking
func (elm *EventLoopManager) checkAsyncCompletion() (result any, complete bool, err error) {
	done := make(chan struct{})
	var checkErr error

	elm.RunOnLoop(func(vm *goja.Runtime) {
		defer close(done)

		completeVal := vm.Get("__turboscript_complete")
		if completeVal != nil && completeVal.ToBoolean() {
			complete = true

			// Check for error
			errorVal := vm.Get("__turboscript_error")
			if errorVal != nil && !goja.IsNull(errorVal) && !goja.IsUndefined(errorVal) {
				checkErr = fmt.Errorf("handle function error: %s", errorVal.String())
				return
			}

			// Get result
			resultVal := vm.Get("__turboscript_result")
			if resultVal != nil {
				result = resultVal.Export()
			}
		}
	})

	<-done
	return result, complete, checkErr
}

// turboJobAsync creates an async version of turboJob that returns a Promise.
func (elm *EventLoopManager) turboJobAsync(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	// Create a new Promise and get resolve/reject functions
	promise, resolve, reject := rt.NewPromise()

	// Execute the job dispatch in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Schedule error resolution on the event loop
				elm.loop.RunOnLoop(func(*goja.Runtime) {
					if err, ok := r.(error); ok {
						_ = reject(err)
					} else {
						_ = reject(fmt.Errorf("job dispatch panic: %v", r))
					}
				})
			}
		}()

		// Use the turboJob utils to execute the job
		elm.turboJobUtils.ExecuteJob(call, rt)

		// Schedule successful resolution on the event loop
		elm.loop.RunOnLoop(func(*goja.Runtime) {
			_ = resolve(goja.Undefined())
		})
	}()

	return rt.ToValue(promise)
}

// turboEmailAsync creates an async version of turboEmail that returns a Promise.
func (elm *EventLoopManager) turboEmailAsync(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	// Create a new Promise and get resolve/reject functions
	promise, resolve, reject := rt.NewPromise()

	// Execute the email sending in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Schedule error resolution on the event loop
				elm.loop.RunOnLoop(func(*goja.Runtime) {
					if err, ok := r.(error); ok {
						_ = reject(err)
					} else {
						_ = reject(fmt.Errorf("email sending panic: %v", r))
					}
				})
			}
		}()

		// Use the turboEmail utils to send the email
		elm.turboEmailUtils.SendEmail(call, rt)

		// Schedule successful resolution on the event loop
		elm.loop.RunOnLoop(func(*goja.Runtime) {
			_ = resolve(goja.Undefined())
		})
	}()

	return rt.ToValue(promise)
}
