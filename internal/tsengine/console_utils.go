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

// Package tsengine provides custom console implementation for JavaScript runtime.
package tsengine

import (
	"fmt"
	"strings"

	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/dop251/goja"
)

// RegisterCustomConsole registers a custom console object that integrates with TurboScript logger.
//
// This replaces the default goja_nodejs console with our own implementation that
// prefixes all console output with [CONSOLE] tags and routes them through our
// structured logging system.
func RegisterCustomConsole(rt *goja.Runtime) {
	console := rt.NewObject()

	// console.log - General logging
	console.Set("log", func(call goja.FunctionCall) goja.Value {
		args := convertGojaArgs(call.Arguments)
		logger.ConsoleLog(args...)
		return goja.Undefined()
	})

	// console.error - Error logging
	console.Set("error", func(call goja.FunctionCall) goja.Value {
		args := convertGojaArgs(call.Arguments)
		logger.ConsoleError(args...)
		return goja.Undefined()
	})

	// console.warn - Warning logging
	console.Set("warn", func(call goja.FunctionCall) goja.Value {
		args := convertGojaArgs(call.Arguments)
		logger.ConsoleWarn(args...)
		return goja.Undefined()
	})

	// console.info - Info logging
	console.Set("info", func(call goja.FunctionCall) goja.Value {
		args := convertGojaArgs(call.Arguments)
		logger.ConsoleInfo(args...)
		return goja.Undefined()
	})

	// console.debug - Debug logging
	console.Set("debug", func(call goja.FunctionCall) goja.Value {
		args := convertGojaArgs(call.Arguments)
		logger.ConsoleDebug(args...)
		return goja.Undefined()
	})

	// console.trace - Stack trace logging (same as error but could be enhanced)
	console.Set("trace", func(call goja.FunctionCall) goja.Value {
		args := convertGojaArgs(call.Arguments)
		if len(args) == 0 {
			logger.ConsoleError("Trace")
		} else {
			logger.ConsoleError(args...)
		}
		return goja.Undefined()
	})

	// console.assert - Conditional error logging
	console.Set("assert", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			logger.ConsoleError("Assertion failed")
			return goja.Undefined()
		}

		assertion := call.Arguments[0]
		if !assertion.ToBoolean() {
			args := []any{"Assertion failed"}
			if len(call.Arguments) > 1 {
				additionalArgs := convertGojaArgs(call.Arguments[1:])
				args = append(args, additionalArgs...)
			}
			logger.ConsoleError(args...)
		}
		return goja.Undefined()
	})

	// console.clear - Clear console (just log a message)
	console.Set("clear", func(call goja.FunctionCall) goja.Value {
		logger.ConsoleInfo("Console cleared")
		return goja.Undefined()
	})

	// console.count - Simple counter (basic implementation)
	console.Set("count", func(call goja.FunctionCall) goja.Value {
		label := "default"
		if len(call.Arguments) > 0 {
			label = call.Arguments[0].String()
		}
		logger.ConsoleInfo(fmt.Sprintf("%s: count", label))
		return goja.Undefined()
	})

	// console.countReset - Reset counter (basic implementation)
	console.Set("countReset", func(call goja.FunctionCall) goja.Value {
		label := "default"
		if len(call.Arguments) > 0 {
			label = call.Arguments[0].String()
		}
		logger.ConsoleInfo(fmt.Sprintf("%s: count reset", label))
		return goja.Undefined()
	})

	// console.group - Group start
	console.Set("group", func(call goja.FunctionCall) goja.Value {
		args := convertGojaArgs(call.Arguments)
		if len(args) == 0 {
			logger.ConsoleInfo("▼ Group")
		} else {
			logger.ConsoleInfo(fmt.Sprintf("▼ %v", args...))
		}
		return goja.Undefined()
	})

	// console.groupCollapsed - Collapsed group start
	console.Set("groupCollapsed", func(call goja.FunctionCall) goja.Value {
		args := convertGojaArgs(call.Arguments)
		if len(args) == 0 {
			logger.ConsoleInfo("▶ Group (collapsed)")
		} else {
			logger.ConsoleInfo(fmt.Sprintf("▶ %v", args...))
		}
		return goja.Undefined()
	})

	// console.groupEnd - Group end
	console.Set("groupEnd", func(call goja.FunctionCall) goja.Value {
		logger.ConsoleInfo("▲ Group end")
		return goja.Undefined()
	})

	// console.table - Table logging (simplified)
	console.Set("table", func(call goja.FunctionCall) goja.Value {
		args := convertGojaArgs(call.Arguments)
		logger.ConsoleInfo(fmt.Sprintf("Table: %v", args...))
		return goja.Undefined()
	})

	// console.time - Start timer
	console.Set("time", func(call goja.FunctionCall) goja.Value {
		label := "default"
		if len(call.Arguments) > 0 {
			label = call.Arguments[0].String()
		}
		logger.ConsoleInfo(fmt.Sprintf("Timer '%s' started", label))
		return goja.Undefined()
	})

	// console.timeEnd - End timer
	console.Set("timeEnd", func(call goja.FunctionCall) goja.Value {
		label := "default"
		if len(call.Arguments) > 0 {
			label = call.Arguments[0].String()
		}
		logger.ConsoleInfo(fmt.Sprintf("Timer '%s' ended", label))
		return goja.Undefined()
	})

	// console.timeLog - Log timer
	console.Set("timeLog", func(call goja.FunctionCall) goja.Value {
		label := "default"
		if len(call.Arguments) > 0 {
			label = call.Arguments[0].String()
		}
		logger.ConsoleInfo(fmt.Sprintf("Timer '%s' log", label))
		return goja.Undefined()
	})

	// Set the console object as a global
	rt.Set("console", console)
}

// convertGojaArgs converts Goja values to Go values for logging.
//
// This function handles the conversion of JavaScript values passed to console
// methods into Go values that can be properly formatted by our logger.
func convertGojaArgs(args []goja.Value) []any {
	result := make([]any, len(args))
	for i, arg := range args {
		result[i] = convertGojaValue(arg)
	}
	return result
}

// convertGojaValue converts a single Goja value to a Go value.
//
// This handles various JavaScript types and converts them to appropriate
// Go representations for logging.
func convertGojaValue(value goja.Value) any {
	if value == nil {
		return nil
	}

	// Handle different value types
	switch {
	case goja.IsUndefined(value):
		return "undefined"
	case goja.IsNull(value):
		return "null"
	case value.ToBoolean() == true || value.ToBoolean() == false:
		// Check if it's actually a boolean or just truthy/falsy
		if value.String() == "true" || value.String() == "false" {
			return value.ToBoolean()
		}
		fallthrough
	default:
		// For objects, arrays, functions, etc., use the string representation
		str := value.String()

		// Try to detect and format objects/arrays nicely
		if strings.HasPrefix(str, "{") || strings.HasPrefix(str, "[") {
			return str
		}

		// For numbers, try to preserve the numeric value
		if num := value.ToNumber(); !goja.IsNaN(num) && !goja.IsInfinity(num) {
			// Check if it's an integer
			if float64(int64(num.ToFloat())) == num.ToFloat() {
				return int64(num.ToFloat())
			}
			return num.ToFloat()
		}

		return str
	}
}
