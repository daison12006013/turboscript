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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileResolver(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := ioutil.TempDir("", "file_resolver_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	tsFile := filepath.Join(tmpDir, "test.ts")
	jsFile := filepath.Join(tmpDir, "test.js")

	tsContent := `export const handle = () => ({ code: 200, response: { from: "ts" } });`
	jsContent := `exports.handle = () => ({ code: 200, response: { from: "js" } });`

	if err := ioutil.WriteFile(tsFile, []byte(tsContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(jsFile, []byte(jsContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("PreferTS", func(t *testing.T) {
		resolver := NewFileResolver(false) // Prefer .ts files

		// Test with extension
		resolved, err := resolver.ResolveFile(tsFile)
		if err != nil {
			t.Errorf("Failed to resolve .ts file: %v", err)
		}
		if resolved != tsFile {
			t.Errorf("Expected %s, got %s", tsFile, resolved)
		}

		// Test without extension - should prefer .ts
		basePath := strings.TrimSuffix(tsFile, ".ts")
		resolved, err = resolver.ResolveFile(basePath)
		if err != nil {
			t.Errorf("Failed to resolve file without extension: %v", err)
		}
		if !strings.HasSuffix(resolved, ".ts") {
			t.Errorf("Expected .ts file, got %s", resolved)
		}
	})

	t.Run("PreferJS", func(t *testing.T) {
		resolver := NewFileResolver(true) // Prefer .js files

		// Test with extension
		resolved, err := resolver.ResolveFile(jsFile)
		if err != nil {
			t.Errorf("Failed to resolve .js file: %v", err)
		}
		if resolved != jsFile {
			t.Errorf("Expected %s, got %s", jsFile, resolved)
		}

		// Test without extension - should prefer .js
		basePath := strings.TrimSuffix(jsFile, ".js")
		resolved, err = resolver.ResolveFile(basePath)
		if err != nil {
			t.Errorf("Failed to resolve file without extension: %v", err)
		}
		if !strings.HasSuffix(resolved, ".js") {
			t.Errorf("Expected .js file, got %s", resolved)
		}
	})

	t.Run("FallbackToAlternative", func(t *testing.T) {
		resolver := NewFileResolver(false) // Prefer .ts files

		// Remove .ts file, only .js should exist
		os.Remove(tsFile)

		basePath := strings.TrimSuffix(tsFile, ".ts")
		resolved, err := resolver.ResolveFile(basePath)
		if err != nil {
			t.Errorf("Failed to resolve file with fallback: %v", err)
		}
		if !strings.HasSuffix(resolved, ".js") {
			t.Errorf("Expected .js file as fallback, got %s", resolved)
		}
	})

	t.Run("FileNotFound", func(t *testing.T) {
		resolver := NewFileResolver(false)

		nonExistentFile := filepath.Join(tmpDir, "nonexistent")
		_, err := resolver.ResolveFile(nonExistentFile)
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})
}

func TestGetRecommendedResolver(t *testing.T) {
	// Save original working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	t.Run("ProductionDetection", func(t *testing.T) {
		// Create a temporary dist directory
		tmpDir, err := ioutil.TempDir("", "dist_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		distDir := filepath.Join(tmpDir, "dist")
		if err := os.Mkdir(distDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Change to the dist directory
		if err := os.Chdir(distDir); err != nil {
			t.Fatal(err)
		}

		resolver := GetRecommendedResolver()
		if !resolver.preferJS {
			t.Error("Expected resolver to prefer JS files in dist directory")
		}
	})

	t.Run("DevelopmentDetection", func(t *testing.T) {
		// Create a temporary regular directory (not dist)
		tmpDir, err := ioutil.TempDir("", "dev_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Change to the regular directory
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		resolver := GetRecommendedResolver()
		if resolver.preferJS {
			t.Error("Expected resolver to prefer TS files in development directory")
		}
	})
}
