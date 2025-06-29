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

// Package tsengine provides async response utilities with event loop support.
package tsengine

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/daison12006013/turboscript/internal/plugins"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// AsyncResponseUtils handles TypeScript response execution with async support.
type AsyncResponseUtils struct {
	cacheUtils         *CacheUtils
	runtimeUtils       *RuntimeUtils
	errorUtils         *ErrorUtils
	stackTraceUtils    *StackTraceUtils
	eventLoop          *EventLoopManager
	turboMarkdownUtils *TurboMarkdownUtils
	db                 *sql.DB
	dbManager          any
}

// NewAsyncResponseUtils creates a new async response utilities instance.
func NewAsyncResponseUtils(cacheUtils *CacheUtils, runtimeUtils *RuntimeUtils, errorUtils *ErrorUtils) *AsyncResponseUtils {
	eventLoop := NewEventLoopManager()

	return &AsyncResponseUtils{
		cacheUtils:         cacheUtils,
		runtimeUtils:       runtimeUtils,
		errorUtils:         errorUtils,
		stackTraceUtils:    NewStackTraceUtils(),
		eventLoop:          eventLoop,
		turboMarkdownUtils: NewTurboMarkdownUtils(""), // Initialize with empty base path
		db:                 nil,                       // Will be set via SetDatabase
		dbManager:          nil,                       // Will be set via SetDatabaseManager
	}
}

// SetDatabase sets the database connection.
func (aru *AsyncResponseUtils) SetDatabase(db *sql.DB) {
	aru.db = db
	aru.eventLoop.SetDatabase(db)
}

// SetDatabaseManager sets the database manager for multi-connection support.
func (aru *AsyncResponseUtils) SetDatabaseManager(dbManager any) {
	aru.dbManager = dbManager
	aru.eventLoop.SetDatabaseManager(dbManager)
}

// SetServer sets the server reference for broadcasting capabilities.
func (aru *AsyncResponseUtils) SetServer(server interface{}) {
	aru.eventLoop.SetServer(server)
}

// SetMarkdownBasePath sets the base path for turboMarkdownHtml function.
func (aru *AsyncResponseUtils) SetMarkdownBasePath(basePath string) {
	aru.turboMarkdownUtils.SetBasePath(basePath)
}

// SetCacheConfig sets the cache configuration for turboCache operations.
func (aru *AsyncResponseUtils) SetCacheConfig(cacheConfig *config.CacheConfig) {
	if cacheConfig != nil {
		turboCacheUtils := NewTurboCacheUtils(cacheConfig)
		aru.eventLoop.SetTurboCacheUtils(turboCacheUtils)
	}
}

// StartEventLoop starts the event loop for async operations.
func (aru *AsyncResponseUtils) StartEventLoop() {
	aru.eventLoop.Start()
}

// TerminateEventLoop terminates the event loop and cleans up resources.
func (aru *AsyncResponseUtils) TerminateEventLoop() {
	aru.eventLoop.Terminate()
}

// ExecuteHandleAsyncWithTimeout executes TypeScript handle function with async support and configurable timeout.
func (aru *AsyncResponseUtils) ExecuteHandleAsyncWithTimeout(tsPath string, event any, preserveResponse bool, timeoutSeconds int) (json.RawMessage, error) {
	startTime := time.Now()
	logger.Debug("Starting async handle execution for %s", tsPath)

	// Get compiled JavaScript from cache
	compiledJS, err := aru.cacheUtils.GetCompiledJS(tsPath)
	if err != nil {
		detailedErr := aru.stackTraceUtils.LogAndWrapErrorWithSource(
			err,
			ErrorTypeJSCompilation,
			"Failed to get compiled JavaScript from cache",
			tsPath,
			map[string]any{
				"typescript_file":      tsPath,
				"cache_lookup_time_ms": time.Since(startTime).Milliseconds(),
			},
		)
		return nil, detailedErr
	}

	compilationTime := time.Since(startTime)
	if compilationTime > 50*time.Millisecond {
		logger.Debug("JS compilation/cache lookup took %.2f seconds for %s", compilationTime.Seconds(), tsPath)
	}

	// Execute using the event loop's RunAsync method
	var tsCode string
	if preserveResponse {
		// When preserveResponse is true, stringify in JavaScript to maintain field order
		tsCode = fmt.Sprintf(`
%s

// Store result in a global variable that the Go code can access
var __turboscript_result = null;
var __turboscript_error = null;
var __turboscript_complete = false;

// Execute the handle function and store the result
(async function() {
	try {
		if (typeof handle === 'function') {
			const result = await handle(event);
			// Stringify in JavaScript to preserve field order
			__turboscript_result = JSON.stringify(result);
		} else {
			__turboscript_error = new Error('handle function not found in ' + %q);
		}
	} catch (error) {
		__turboscript_error = error;
	} finally {
		__turboscript_complete = true;
	}
})();

// Return completion flag for checking
__turboscript_complete;
	`, compiledJS, tsPath)
	} else {
		// Legacy mode: let Go handle the marshaling (may reorder fields)
		tsCode = fmt.Sprintf(`
%s

// Store result in a global variable that the Go code can access
var __turboscript_result = null;
var __turboscript_error = null;
var __turboscript_complete = false;

// Execute the handle function and store the result
(async function() {
	try {
		if (typeof handle === 'function') {
			const result = await handle(event);
			__turboscript_result = result;
		} else {
			__turboscript_error = new Error('handle function not found in ' + %q);
		}
	} catch (error) {
		__turboscript_error = error;
	} finally {
		__turboscript_complete = true;
	}
})();

// Return completion flag for checking
__turboscript_complete;
	`, compiledJS, tsPath)
	}

	// Setup function to inject variables and utilities
	setupFn := func(rt *goja.Runtime) {
		// Set event data
		if err := rt.Set("event", event); err != nil {
			logger.Error("Failed to set event variable: %v", err)
		}

		// Inject async turboQuery function
		if aru.db != nil {
			err := aru.eventLoop.InjectAsyncTurboQuery(rt)
			if err != nil {
				logger.Error("Failed to inject turboQuery: %v", err)
			}
		}

		// Inject async turboJob function
		err := aru.eventLoop.InjectAsyncTurboJob(rt)
		if err != nil {
			logger.Error("Failed to inject turboJob: %v", err)
		}

		// Inject async turbo broadcast functions
		err = aru.eventLoop.InjectAsyncTurboBroadcast(rt)
		if err != nil {
			logger.Error("Failed to inject turboBroadcast: %v", err)
		}

		// Inject async turboEmail function
		err = aru.eventLoop.InjectAsyncTurboEmail(rt)
		if err != nil {
			logger.Error("Failed to inject turboEmail: %v", err)
		}

		// Inject async turboCache function
		err = aru.eventLoop.InjectAsyncTurboCache(rt, aru.eventLoop.turboCacheUtils)
		if err != nil {
			logger.Error("Failed to inject turboCache: %v", err)
		}

		// Register plugins and turboPlugin function
		registry := require.NewRegistry()
		err = plugins.GlobalManager.RegisterWithRuntime(rt, registry)
		if err != nil {
			logger.Error("Failed to register plugins with runtime: %v", err)
		}
		registry.Enable(rt)                      // Enable require() function for plugins
		RegisterSharedPluginModule(rt, registry) // Register turboPlugin function

		// Inject turboMarkdownHtml function
		err = rt.Set("turboMarkdownHtml", func(call goja.FunctionCall) goja.Value {
			return aru.turboMarkdownUtils.ExecuteMarkdownHTML(call, rt)
		})
		if err != nil {
			logger.Error("Failed to set turboMarkdownHtml function: %v", err)
		}

		// Inject turboHtml function
		err = rt.Set("turboHtml", func(call goja.FunctionCall) goja.Value {
			return aru.turboMarkdownUtils.ExecuteHTML(call, rt)
		})
		if err != nil {
			logger.Error("Failed to set turboHtml function: %v", err)
		}
	}

	// Execute the TypeScript code with async support
	execStartTime := time.Now()
	result, err := aru.eventLoop.RunAsyncWithCompletionTimeout(tsCode, setupFn, timeoutSeconds)
	if err != nil {
		// Create detailed error with full stacktrace and context
		detailedErr := aru.stackTraceUtils.LogAndWrapErrorWithSource(
			err,
			ErrorTypeAsyncExecution,
			"Async handle execution failed",
			tsPath,
			map[string]any{
				"typescript_file":     tsPath,
				"timeout_seconds":     timeoutSeconds,
				"preserve_response":   preserveResponse,
				"execution_time_ms":   time.Since(execStartTime).Milliseconds(),
				"compilation_time_ms": time.Since(startTime).Milliseconds(),
			},
		)
		return nil, detailedErr
	}

	executionTime := time.Since(execStartTime)
	totalTime := time.Since(startTime)

	// Log timing information for cold start debugging
	if totalTime > 1*time.Second {
		logger.Debug("Async execution completed for %s - execution: %.2fs, total: %.2fs",
			tsPath, executionTime.Seconds(), totalTime.Seconds())
	} else if totalTime > 100*time.Millisecond {
		logger.Debug("Async execution completed for %s - total: %.2fs",
			tsPath, totalTime.Seconds())
	}

	// Convert result to JSON - preserve order if requested
	resultBytes, err := aru.processResultToJSON(result, preserveResponse)
	if err != nil {
		return nil, err
	}

	// Check for execution errors in the result
	resultStr := string(resultBytes)
	if err := aru.errorUtils.CheckForExecutionError(resultStr, tsPath); err != nil {
		return nil, err
	}

	return json.RawMessage(resultStr), nil
}

// processResultToJSON converts the execution result to JSON bytes, preserving order if requested.
func (aru *AsyncResponseUtils) processResultToJSON(result any, preserveResponse bool) ([]byte, error) {
	if preserveResponse {
		logger.Debug("Using preserveResponse mode (async) - maintaining field order")
		return aru.processPreserveResponseResult(result)
	}

	logger.Debug("Using legacy mode (async) - may reorder fields")
	return aru.processLegacyResult(result)
}

// processPreserveResponseResult handles result processing when preserveResponse is true.
func (aru *AsyncResponseUtils) processPreserveResponseResult(result any) ([]byte, error) {
	// Result should already be a JSON string from JavaScript
	if resultStr, ok := result.(string); ok {
		return aru.validateAndReturnJSONString(resultStr)
	}

	// Fallback: this shouldn't happen with preserveResponse=true, but handle it
	logger.Debug("Warning: result not a string in preserveResponse mode, falling back to marshaling")
	return aru.processLegacyResult(result)
}

// validateAndReturnJSONString validates a JSON string and returns it as bytes.
func (aru *AsyncResponseUtils) validateAndReturnJSONString(resultStr string) ([]byte, error) {
	// Validate that it's valid JSON
	var temp any
	if err := json.Unmarshal([]byte(resultStr), &temp); err != nil {
		preview := resultStr
		if len(resultStr) > 200 {
			preview = resultStr[:200] + "..."
		}
		detailedErr := aru.stackTraceUtils.LogAndWrapError(
			err,
			ErrorTypeValidation,
			"Invalid JSON returned from handle function",
			map[string]any{
				"json_string_preview": preview,
				"json_string_length":  len(resultStr),
			},
		)
		return nil, detailedErr
	}
	return []byte(resultStr), nil
}

// processLegacyResult handles result processing in legacy mode (may reorder fields).
func (aru *AsyncResponseUtils) processLegacyResult(result any) ([]byte, error) {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		resultStr := fmt.Sprintf("%+v", result)
		if len(resultStr) > 200 {
			resultStr = resultStr[:200] + "..."
		}
		detailedErr := aru.stackTraceUtils.LogAndWrapError(
			err,
			ErrorTypeValidation,
			"Failed to marshal result to JSON",
			map[string]any{
				"result_type":    fmt.Sprintf("%T", result),
				"result_preview": resultStr,
			},
		)
		return nil, detailedErr
	}
	return resultBytes, nil
}
