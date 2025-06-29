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

// Package tsengine provides plugin utilities for the TypeScript execution engine.
package tsengine

import (
	"fmt"
	"log"

	"github.com/daison12006013/turboscript/internal/plugins"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// RegisterSharedPluginModule registers the turboPlugin function with the JavaScript runtime.
// This provides a clean API for accessing TurboScript plugins from TypeScript code.
func RegisterSharedPluginModule(rt *goja.Runtime, registry *require.Registry) {
	// Create the turboPlugin function
	turboPluginFn := func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(rt.NewTypeError("turboPlugin requires a plugin name"))
		}

		pluginName := call.Arguments[0].String()
		if pluginName == "" {
			panic(rt.NewTypeError("plugin name cannot be empty"))
		}

		// Get the plugin from the global manager to validate it exists
		_, exists := plugins.GlobalManager.GetPlugin(pluginName)
		if !exists {
			panic(rt.NewTypeError(fmt.Sprintf("plugin '%s' not found", pluginName)))
		}

		// Check if plugin is enabled
		pluginStatus := plugins.GlobalManager.ListPlugins()
		status, hasStatus := pluginStatus[pluginName]
		if !hasStatus || !status.Enabled {
			panic(rt.NewTypeError(fmt.Sprintf("plugin '%s' is not enabled", pluginName)))
		}

		// Use require() to get the plugin module since it's already registered
		requireFn, ok := goja.AssertFunction(rt.Get("require"))
		if !ok {
			panic(rt.NewTypeError("require function not available"))
		}

		result, err := requireFn(goja.Undefined(), rt.ToValue(pluginName))
		if err != nil {
			panic(rt.NewTypeError("Failed to load plugin module: " + err.Error()))
		}

		return result
	}

	// Register turboPlugin as a global function
	if err := rt.Set("turboPlugin", turboPluginFn); err != nil {
		log.Printf("Failed to set turboPlugin function: %v", err)
	}
}
