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

// Package tsengine provides shared bcrypt utilities for JavaScript runtime.
package tsengine

import (
	"crypto/rand"
	"fmt"

	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"golang.org/x/crypto/bcrypt"
)

// RegisterSharedBcryptModule registers a bcryptjs-compatible module in the goja runtime.
// This is a shared implementation to avoid code duplication across runtime utilities.
func RegisterSharedBcryptModule(rt *goja.Runtime, registry *require.Registry) {
	bcryptObj := rt.NewObject()

	// genSaltSync function - compatible with bcryptjs
	if err := bcryptObj.Set("genSaltSync", func(rounds int) string {
		if rounds < 4 {
			rounds = 4
		}
		if rounds > 31 {
			rounds = 31
		}
		// Generate a salt with the specified cost
		salt := make([]byte, 22)
		_, err := rand.Read(salt)
		if err != nil {
			// Fallback to a default salt if random generation fails
			copy(salt, []byte("defaultsaltdefaultsa12"))
		}

		// Create bcrypt-compatible salt string
		return fmt.Sprintf("$2a$%02d$%s", rounds, string(salt)[:22])
	}); err != nil {
		logger.Error("Failed to set genSaltSync function: %v", err)
	}

	// hashSync function - compatible with bcryptjs
	if err := bcryptObj.Set("hashSync", func(password string, saltOrRounds any) string {
		var cost int
		switch v := saltOrRounds.(type) {
		case int, int64:
			if intVal, ok := v.(int); ok {
				cost = intVal
			} else if int64Val, ok := v.(int64); ok {
				cost = int(int64Val)
			}
			if cost < 4 {
				cost = 4
			}
			if cost > 31 {
				cost = 31
			}
		case string:
			// If salt string is provided, use default cost
			cost = 10
		default:
			cost = 10
		}

		hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("bcrypt hash failed: %w", err)))
		}
		return string(hashedBytes)
	}); err != nil {
		logger.Error("Failed to set hashSync function: %v", err)
	}

	// compareSync function - compatible with bcryptjs
	if err := bcryptObj.Set("compareSync", func(password, hash string) bool {
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		return err == nil
	}); err != nil {
		logger.Error("Failed to set compareSync function: %v", err)
	}

	// Register the bcryptjs module
	registry.RegisterNativeModule("bcryptjs", func(_ *goja.Runtime, module *goja.Object) {
		if err := module.Set("exports", bcryptObj); err != nil {
			logger.Error("Failed to set bcryptjs exports: %v", err)
		}
	})
}
