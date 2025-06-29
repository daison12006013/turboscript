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

// Package tsengine provides Node.js compatibility modules for the goja runtime.
package tsengine

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// RegisterNodeJSModules registers essential Node.js modules that argon2 and other packages need.
func RegisterNodeJSModules(rt *goja.Runtime, registry *require.Registry) {
	// Register fs module (minimal implementation)
	registry.RegisterNativeModule("fs", func(_ *goja.Runtime, module *goja.Object) {
		fsObj := rt.NewObject()

		// existsSync function
		if err := fsObj.Set("existsSync", func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		}); err != nil {
			logger.Error("Failed to set fs.existsSync: %v", err)
		}

		// readFileSync function (minimal implementation)
		if err := fsObj.Set("readFileSync", func(path string) string {
			data, err := os.ReadFile(path)
			if err != nil {
				panic(rt.NewGoError(err))
			}
			return string(data)
		}); err != nil {
			logger.Error("Failed to set fs.readFileSync: %v", err)
		}

		if err := module.Set("exports", fsObj); err != nil {
			logger.Error("Failed to set fs exports: %v", err)
		}
	})

	// Register path module
	registry.RegisterNativeModule("path", func(_ *goja.Runtime, module *goja.Object) {
		pathObj := rt.NewObject()

		// join function
		if err := pathObj.Set("join", func(call goja.FunctionCall) goja.Value {
			parts := make([]string, len(call.Arguments))
			for i, arg := range call.Arguments {
				parts[i] = arg.String()
			}
			return rt.ToValue(filepath.Join(parts...))
		}); err != nil {
			logger.Error("Failed to set path.join: %v", err)
		}

		// dirname function
		if err := pathObj.Set("dirname", func(path string) string {
			return filepath.Dir(path)
		}); err != nil {
			logger.Error("Failed to set path.dirname: %v", err)
		}

		// basename function
		if err := pathObj.Set("basename", func(path string) string {
			return filepath.Base(path)
		}); err != nil {
			logger.Error("Failed to set path.basename: %v", err)
		}

		// resolve function
		if err := pathObj.Set("resolve", func(call goja.FunctionCall) goja.Value {
			parts := make([]string, len(call.Arguments))
			for i, arg := range call.Arguments {
				parts[i] = arg.String()
			}
			result, _ := filepath.Abs(filepath.Join(parts...))
			return rt.ToValue(result)
		}); err != nil {
			logger.Error("Failed to set path.resolve: %v", err)
		}

		if err := module.Set("exports", pathObj); err != nil {
			logger.Error("Failed to set path exports: %v", err)
		}
	})

	// Register os module
	registry.RegisterNativeModule("os", func(_ *goja.Runtime, module *goja.Object) {
		osObj := rt.NewObject()

		// platform function
		if err := osObj.Set("platform", func() string {
			return runtime.GOOS
		}); err != nil {
			logger.Error("Failed to set os.platform: %v", err)
		}

		// arch function
		if err := osObj.Set("arch", func() string {
			return runtime.GOARCH
		}); err != nil {
			logger.Error("Failed to set os.arch: %v", err)
		}

		// tmpdir function
		if err := osObj.Set("tmpdir", func() string {
			return os.TempDir()
		}); err != nil {
			logger.Error("Failed to set os.tmpdir: %v", err)
		}

		if err := module.Set("exports", osObj); err != nil {
			logger.Error("Failed to set os exports: %v", err)
		}
	})

	// Register node:assert module
	registry.RegisterNativeModule("node:assert", func(_ *goja.Runtime, module *goja.Object) {
		assertObj := rt.NewObject()

		// assert function
		if err := assertObj.Set("assert", func(condition bool, message string) {
			if !condition {
				if message == "" {
					message = "Assertion failed"
				}
				panic(rt.NewTypeError(message))
			}
		}); err != nil {
			logger.Error("Failed to set assert function: %v", err)
		}

		// ok function (alias for assert)
		if err := assertObj.Set("ok", func(condition bool, message string) {
			if !condition {
				if message == "" {
					message = "Assertion failed"
				}
				panic(rt.NewTypeError(message))
			}
		}); err != nil {
			logger.Error("Failed to set assert.ok: %v", err)
		}

		if err := module.Set("exports", assertObj); err != nil {
			logger.Error("Failed to set node:assert exports: %v", err)
		}
	})

	// Register node:util module
	registry.RegisterNativeModule("node:util", func(_ *goja.Runtime, module *goja.Object) {
		utilObj := rt.NewObject()

		// promisify function (basic implementation)
		if err := utilObj.Set("promisify", func(fn goja.Value) goja.Value {
			// Return a function that creates a Promise
			return rt.ToValue(func(call goja.FunctionCall) goja.Value {
				promise, resolve, reject := rt.NewPromise()

				// Create a callback function
				callback := func(call goja.FunctionCall) goja.Value {
					if len(call.Arguments) > 0 && !goja.IsNull(call.Arguments[0]) && !goja.IsUndefined(call.Arguments[0]) {
						// First argument is error
						reject(call.Arguments[0])
					} else if len(call.Arguments) > 1 {
						// Second argument is result
						resolve(call.Arguments[1])
					} else {
						resolve(goja.Undefined())
					}
					return goja.Undefined()
				}

				// Add callback to arguments
				args := make([]goja.Value, len(call.Arguments)+1)
				copy(args, call.Arguments)
				args[len(args)-1] = rt.ToValue(callback)

				// Call original function with callback
				if fnCallable, ok := goja.AssertFunction(fn); ok {
					fnCallable(goja.Undefined(), args...)
				}

				return rt.ToValue(promise)
			})
		}); err != nil {
			logger.Error("Failed to set util.promisify: %v", err)
		}

		if err := module.Set("exports", utilObj); err != nil {
			logger.Error("Failed to set node:util exports: %v", err)
		}
	})
}
