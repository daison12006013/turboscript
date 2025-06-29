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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUtilityCreation(t *testing.T) {
	// Test that individual utilities can be created
	compiler := NewCompilerUtils()
	if compiler == nil {
		t.Error("NewCompilerUtils() returned nil")
	}

	runtimeUtils := NewRuntimeUtils()
	if runtimeUtils == nil {
		t.Error("NewRuntimeUtils() returned nil")
	}

	cacheUtils := NewCacheUtils(compiler)
	if cacheUtils == nil {
		t.Error("NewCacheUtils() returned nil")
	}

	errorUtils := NewErrorUtils()
	if errorUtils == nil {
		t.Error("NewErrorUtils() returned nil")
	}

	responseUtils := NewResponseUtils(cacheUtils, runtimeUtils, errorUtils)
	if responseUtils == nil {
		t.Error("NewResponseUtils() returned nil")
	}
}

func TestCompilerUtils(t *testing.T) {
	compiler := NewCompilerUtils()

	// Test basic TypeScript compilation
	t.Run("CompileSimpleTypeScript", func(t *testing.T) {
		// Create a temporary TypeScript file
		tmpDir, err := ioutil.TempDir("", "tsengine_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		tsContent := `
function handle(event) {
	return {
		code: 200,
		response: {
			status: "success",
			data: { users: [] }
		}
	};
}

globalThis.handle = handle;
`
		tsFile := filepath.Join(tmpDir, "test.ts")
		err = ioutil.WriteFile(tsFile, []byte(tsContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Test compilation
		jsCode, err := compiler.ConvertTSToJS(tsFile)
		if err != nil {
			t.Errorf("Failed to compile TypeScript: %v", err)
		}

		if len(jsCode) == 0 {
			t.Error("Compiled JavaScript code is empty")
		}

		// Check that the output contains expected elements
		if !strings.Contains(jsCode, "function handle") && !strings.Contains(jsCode, "handle") {
			t.Error("Compiled code doesn't contain handle function")
		}
	})

	t.Run("CompileWithoutImports", func(t *testing.T) {
		// Create a temporary TypeScript file without imports
		tmpDir, err := ioutil.TempDir("", "tsengine_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Create main TypeScript file with inline function
		tsContent := `
function handle(event) {
	function getMeta(event) {
		return {
			timestamp: new Date().toISOString()
		};
	}

	return {
		code: 200,
		response: {
			status: "success",
			data: event.body,
			meta: getMeta(event)
		}
	};
}

globalThis.handle = handle;
`
		tsFile := filepath.Join(tmpDir, "test.ts")
		err = ioutil.WriteFile(tsFile, []byte(tsContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Test compilation
		jsCode, err := compiler.ConvertTSToJS(tsFile)
		if err != nil {
			t.Errorf("Failed to compile TypeScript: %v", err)
		}

		if len(jsCode) == 0 {
			t.Error("Compiled JavaScript code is empty")
		}

		// The compiled code should contain the handle function
		if !strings.Contains(jsCode, "handle") {
			t.Error("Compiled code doesn't contain handle function")
		}
	})
}

func TestCacheUtils(t *testing.T) {
	compiler := NewCompilerUtils()
	cacheUtils := NewCacheUtils(compiler)

	t.Run("CacheCompilation", func(t *testing.T) {
		// Create a temporary TypeScript file
		tmpDir, err := ioutil.TempDir("", "tsengine_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		tsContent := `
function handle(event) {
	return {
		code: 200,
		response: { status: "success", data: { test: true } }
	};
}

globalThis.handle = handle;
`
		tsFile := filepath.Join(tmpDir, "cache_test.ts")
		err = ioutil.WriteFile(tsFile, []byte(tsContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// First compilation - should cache the result
		start1 := time.Now()
		jsCode1, err := cacheUtils.GetCompiledJS(tsFile)
		if err != nil {
			t.Errorf("First compilation failed: %v", err)
		}
		duration1 := time.Since(start1)

		// Second compilation - should use cache
		start2 := time.Now()
		jsCode2, err := cacheUtils.GetCompiledJS(tsFile)
		if err != nil {
			t.Errorf("Second compilation failed: %v", err)
		}
		duration2 := time.Since(start2)

		// Verify results are identical
		if jsCode1 != jsCode2 {
			t.Error("Cached compilation result differs from original")
		}

		// Second compilation should be faster (cached)
		if duration2 > duration1 {
			t.Logf("Warning: Second compilation (%v) took longer than first (%v), cache might not be working", duration2, duration1)
		}
	})
}

func TestErrorUtils(t *testing.T) {
	errorUtils := NewErrorUtils()

	t.Run("CheckForTSExecutionError", func(t *testing.T) {
		// Test with no error
		noErrorResult := `{"query": "SELECT 1", "params": []}`
		err := errorUtils.CheckForTSExecutionError(noErrorResult, "test.ts")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Test with TypeScript error
		errorResult := `{"__ts_error": {"message": "Test error", "stack": "Error stack"}}`
		err = errorUtils.CheckForTSExecutionError(errorResult, "test.ts")
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Test error") {
			t.Errorf("Error message doesn't contain expected text: %v", err)
		}
	})

	t.Run("ProcessHandleError", func(t *testing.T) {
		errorData := map[string]any{
			"message": "Handle function failed",
			"name":    "HandleError",
		}

		err := errorUtils.ProcessHandleError(errorData, "test.ts")
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Handle function failed") {
			t.Errorf("Error message doesn't contain expected text: %v", err)
		}
	})
}
