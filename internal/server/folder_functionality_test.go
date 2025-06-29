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

package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFolderIndexFileHandling(t *testing.T) {
	// Create a temporary test directory structure that mimics the real docs structure
	tempDir, err := os.MkdirTemp("", "folder_index_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test structure:
	// tempDir/
	//   docs/
	//     index.md
	//     api/
	//       index.md
	//     guides/
	//       index.md
	//       development.md
	docsDir := filepath.Join(tempDir, "docs")
	apiDir := filepath.Join(docsDir, "api")
	guidesDir := filepath.Join(docsDir, "guides")

	err = os.MkdirAll(apiDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}
	err = os.MkdirAll(guidesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create guides dir: %v", err)
	}

	// Create test files
	err = os.WriteFile(filepath.Join(docsDir, "index.md"), []byte("# Documentation Home"), 0644)
	if err != nil {
		t.Fatalf("Failed to create docs index.md: %v", err)
	}

	err = os.WriteFile(filepath.Join(apiDir, "index.md"), []byte("# API Reference"), 0644)
	if err != nil {
		t.Fatalf("Failed to create api index.md: %v", err)
	}

	err = os.WriteFile(filepath.Join(guidesDir, "index.md"), []byte("# Guides Overview"), 0644)
	if err != nil {
		t.Fatalf("Failed to create guides index.md: %v", err)
	}

	err = os.WriteFile(filepath.Join(guidesDir, "development.md"), []byte("# Development Guide"), 0644)
	if err != nil {
		t.Fatalf("Failed to create development.md: %v", err)
	}

	server := &Server{}

	testCases := []struct {
		name            string
		fileName        string
		indexFile       string
		expectedContent string
		expectedCode    float64
		shouldSucceed   bool
		description     string
	}{
		{
			name:            "Root docs index",
			fileName:        "index.md",
			indexFile:       "index.md",
			expectedContent: "Documentation Home",
			expectedCode:    200,
			shouldSucceed:   true,
			description:     "Should serve the root index.md file",
		},
		{
			name:            "API subdirectory index",
			fileName:        "api",
			indexFile:       "index.md",
			expectedContent: "API Reference",
			expectedCode:    200,
			shouldSucceed:   true,
			description:     "Should serve index.md from api subdirectory",
		},
		{
			name:            "Guides subdirectory index",
			fileName:        "guides",
			indexFile:       "index.md",
			expectedContent: "Guides Overview",
			expectedCode:    200,
			shouldSucceed:   true,
			description:     "Should serve index.md from guides subdirectory",
		},
		{
			name:            "Specific file in subdirectory",
			fileName:        "guides/development.md",
			indexFile:       "index.md",
			expectedContent: "Development Guide",
			expectedCode:    200,
			shouldSucceed:   true,
			description:     "Should serve specific file from subdirectory",
		},
		{
			name:          "Non-existent subdirectory",
			fileName:      "nonexistent",
			indexFile:     "index.md",
			expectedCode:  404,
			shouldSucceed: false,
			description:   "Should return 404 for non-existent directory",
		},
		{
			name:          "Subdirectory without index file",
			fileName:      "api",
			indexFile:     "missing.md",
			expectedCode:  404,
			shouldSucceed: false,
			description:   "Should return 404 when index file doesn't exist in directory",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := server.serveFolderFile(
				docsDir,         // folderPath
				tc.fileName,     // fileName
				false,           // isMarkdown
				"markdown-html", // responseType
				tc.indexFile,    // indexFile
				"",              // layoutFile
				"/docs/",        // basePath
			)

			if tc.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
					return
				}

				var response map[string]any
				if err := json.Unmarshal(result, &response); err != nil {
					t.Errorf("Failed to parse response JSON: %v", err)
					return
				}

				if code, ok := response["code"].(float64); !ok || code != tc.expectedCode {
					t.Errorf("Expected code %v, got: %v", tc.expectedCode, code)
					return
				}

				if tc.expectedContent != "" {
					responseData, ok := response["response"].(string)
					if !ok {
						t.Errorf("Expected string response, got: %T", response["response"])
						return
					}

					if !strings.Contains(responseData, tc.expectedContent) {
						t.Errorf("Expected response to contain '%s', got: %s", tc.expectedContent, responseData)
					}
				}
			} else {
				// For cases that should fail, we expect either an error or a 404 response
				if err == nil {
					var response map[string]any
					if err := json.Unmarshal(result, &response); err != nil {
						t.Errorf("Failed to parse response JSON: %v", err)
						return
					}

					if code, ok := response["code"].(float64); !ok || code != tc.expectedCode {
						t.Errorf("Expected error code %v, got: %v", tc.expectedCode, code)
					}
				}
				// If there's an error, that's also acceptable for failure cases
			}
		})
	}
}

func TestEmptyIndexFileHandling(t *testing.T) {
	// Test what happens when index file is not configured
	tempDir, err := os.MkdirTemp("", "empty_index_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	docsDir := filepath.Join(tempDir, "docs")
	apiDir := filepath.Join(docsDir, "api")

	err = os.MkdirAll(apiDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}

	server := &Server{}

	// Test accessing a directory when no index file is configured
	result, err := server.serveFolderFile(
		docsDir,         // folderPath
		"api",           // fileName (directory)
		false,           // isMarkdown
		"markdown-html", // responseType
		"",              // indexFile (empty - no index configured)
		"",              // layoutFile
		"/docs/",        // basePath
	)

	// This should return an error because it's trying to read a directory without an index file
	if err == nil {
		t.Errorf("Expected error when accessing directory without index file, but got result: %v", string(result))
	} else {
		// Verify the error message indicates it's a directory
		if !strings.Contains(err.Error(), "is a directory") {
			t.Errorf("Expected 'is a directory' error, got: %v", err)
		}
	}
}
