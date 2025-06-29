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

// Package tsengine provides JavaScript runtime management utilities.
package tsengine

import (
	"sync"

	"github.com/daison12006013/turboscript/internal/plugins"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/url"
)

// RuntimeUtils handles JavaScript runtime pool management and module registration.
type RuntimeUtils struct {
	pool sync.Pool
}

// NewRuntimeUtils creates a new runtime utilities instance.
func NewRuntimeUtils() *RuntimeUtils {
	ru := &RuntimeUtils{}
	ru.pool.New = func() any {
		rt := goja.New()
		registry := require.NewRegistry()

		// Register bcryptjs module for password utilities
		ru.registerBcryptModule(rt, registry)

		// Register crypto module for hashing and signing
		ru.registerCryptoModule(rt, registry)

		// Register plugins with the runtime
		ru.registerPlugins(rt, registry)

		// Register turboPlugin function
		RegisterSharedPluginModule(rt, registry)

		registry.Enable(rt) // Enable require() function
		console.Enable(rt)  // Enable console.log, console.error, etc.
		url.Enable(rt)      // Enable URL and URLSearchParams
		buffer.Enable(rt)   // Enable Buffer for binary data handling

		return &JSRuntime{
			Runtime:  rt,
			Registry: registry,
		}
	}
	return ru
}

// registerBcryptModule registers a bcryptjs-compatible module in the goja runtime.
// This implementation delegates to the shared bcrypt utilities for consistency.
func (ru *RuntimeUtils) registerBcryptModule(rt *goja.Runtime, registry *require.Registry) {
	RegisterSharedBcryptModule(rt, registry)
}

// registerCryptoModule registers a crypto module in the goja runtime.
// This implementation provides a consistent crypto interface for hashing and signing.
func (ru *RuntimeUtils) registerCryptoModule(rt *goja.Runtime, registry *require.Registry) {
	RegisterSharedCryptoModule(rt, registry)
}

// registerPlugins registers all enabled plugins with the JavaScript runtime.
func (ru *RuntimeUtils) registerPlugins(rt *goja.Runtime, registry *require.Registry) {
	// Register plugins with the runtime using the global plugin manager
	if err := plugins.GlobalManager.RegisterWithRuntime(rt, registry); err != nil {
		// Log error but don't fail - plugins are optional
		// TODO: Add proper logging here when needed
	}
}
