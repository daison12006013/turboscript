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

package tsengine

import (
	"testing"

	"github.com/dop251/goja"
)

func TestTurboQueryOverloading(t *testing.T) {
	// Test parsing legacy format
	t.Run("ParseLegacyFormat", func(t *testing.T) {
		turboQueryUtils := NewTurboQueryUtils()
		rt := goja.New()

		// Create a mock function call with legacy format
		mockCall := createMockCall(rt, "SELECT * FROM users", []any{"123"})

		query, params, connection, err := turboQueryUtils.ParseTurboQueryArgs(mockCall, rt)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if query != "SELECT * FROM users" {
			t.Errorf("Expected query 'SELECT * FROM users', got: %s", query)
		}

		if len(params) != 1 || params[0] != "123" {
			t.Errorf("Expected params ['123'], got: %v", params)
		}

		if connection != "" {
			t.Errorf("Expected empty connection for legacy format, got: %s", connection)
		}
	})

	// Test parsing object format
	t.Run("ParseObjectFormat", func(t *testing.T) {
		turboQueryUtils := NewTurboQueryUtils()
		rt := goja.New()

		// Create a mock function call with object format
		optionsObj := map[string]any{
			"query":      "SELECT * FROM products",
			"bindings":   []any{"active"},
			"connection": "reader",
		}
		mockCall := createMockCallWithObject(rt, optionsObj)

		query, params, connection, err := turboQueryUtils.ParseTurboQueryArgs(mockCall, rt)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if query != "SELECT * FROM products" {
			t.Errorf("Expected query 'SELECT * FROM products', got: %s", query)
		}

		if len(params) != 1 || params[0] != "active" {
			t.Errorf("Expected params ['active'], got: %v", params)
		}

		if connection != "reader" {
			t.Errorf("Expected connection 'reader', got: %s", connection)
		}
	})

	// Test parsing object format with minimal options
	t.Run("ParseObjectFormatMinimal", func(t *testing.T) {
		turboQueryUtils := NewTurboQueryUtils()
		rt := goja.New()

		// Create a mock function call with minimal object format
		optionsObj := map[string]any{
			"query": "SELECT COUNT(*) FROM orders",
		}
		mockCall := createMockCallWithObject(rt, optionsObj)

		query, params, connection, err := turboQueryUtils.ParseTurboQueryArgs(mockCall, rt)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if query != "SELECT COUNT(*) FROM orders" {
			t.Errorf("Expected query 'SELECT COUNT(*) FROM orders', got: %s", query)
		}

		if len(params) != 0 {
			t.Errorf("Expected empty params, got: %v", params)
		}

		if connection != "" {
			t.Errorf("Expected empty connection, got: %s", connection)
		}
	})

	// Test error handling for missing query field
	t.Run("ErrorMissingQuery", func(t *testing.T) {
		turboQueryUtils := NewTurboQueryUtils()
		rt := goja.New()

		// Create a mock function call with object missing query field
		optionsObj := map[string]any{
			"bindings":   []any{"test"},
			"connection": "reader",
		}
		mockCall := createMockCallWithObject(rt, optionsObj)

		_, _, _, err := turboQueryUtils.ParseTurboQueryArgs(mockCall, rt)
		if err == nil {
			t.Fatal("Expected error for missing query field, got nil")
		}

		expectedError := "query field is required in options object"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
		}
	})

	// Test error handling for no arguments
	t.Run("ErrorNoArguments", func(t *testing.T) {
		turboQueryUtils := NewTurboQueryUtils()
		rt := goja.New()

		// Create a mock function call with no arguments
		mockCall := createMockCallEmpty(rt)

		_, _, _, err := turboQueryUtils.ParseTurboQueryArgs(mockCall, rt)
		if err == nil {
			t.Fatal("Expected error for no arguments, got nil")
		}

		expectedError := "turboQuery requires at least 1 argument"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
		}
	})
}

// Helper function to create a mock function call with legacy format
func createMockCall(rt *goja.Runtime, query string, params []any) goja.FunctionCall {
	args := make([]goja.Value, 0)
	args = append(args, rt.ToValue(query))
	if params != nil {
		args = append(args, rt.ToValue(params))
	}
	return goja.FunctionCall{Arguments: args}
}

// Helper function to create a mock function call with object format
func createMockCallWithObject(rt *goja.Runtime, options map[string]any) goja.FunctionCall {
	args := make([]goja.Value, 0)
	args = append(args, rt.ToValue(options))
	return goja.FunctionCall{Arguments: args}
}

// Helper function to create a mock function call with no arguments
func createMockCallEmpty(_ *goja.Runtime) goja.FunctionCall {
	return goja.FunctionCall{Arguments: []goja.Value{}}
}
