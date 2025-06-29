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

func TestFolderTraversalSecurity(t *testing.T) {
	// Create a temporary test directory structure
	tempDir, err := os.MkdirTemp("", "folder_security_test")
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
	//   secret/
	//     sensitive.txt
	docsDir := filepath.Join(tempDir, "docs")
	apiDir := filepath.Join(docsDir, "api")
	secretDir := filepath.Join(tempDir, "secret")

	err = os.MkdirAll(apiDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}
	err = os.MkdirAll(secretDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create secret dir: %v", err)
	}

	// Create test files
	err = os.WriteFile(filepath.Join(docsDir, "index.md"), []byte("# Docs Home"), 0644)
	if err != nil {
		t.Fatalf("Failed to create docs index.md: %v", err)
	}

	err = os.WriteFile(filepath.Join(apiDir, "index.md"), []byte("# API Documentation"), 0644)
	if err != nil {
		t.Fatalf("Failed to create api index.md: %v", err)
	}

	err = os.WriteFile(filepath.Join(secretDir, "sensitive.txt"), []byte("SECRET DATA"), 0644)
	if err != nil {
		t.Fatalf("Failed to create sensitive.txt: %v", err)
	}

	server := &Server{}

	// Test cases for path traversal attacks
	testCases := []struct {
		name          string
		fileName      string
		indexFile     string
		shouldSucceed bool
		description   string
	}{
		{
			name:          "Normal file access",
			fileName:      "index.md",
			indexFile:     "index.md",
			shouldSucceed: true,
			description:   "Should allow access to normal files within the folder",
		},
		{
			name:          "Normal subdirectory access",
			fileName:      "api",
			indexFile:     "index.md",
			shouldSucceed: true,
			description:   "Should allow access to subdirectories with index files",
		},
		{
			name:          "Path traversal with ../ attack",
			fileName:      "../secret/sensitive.txt",
			indexFile:     "index.md",
			shouldSucceed: false,
			description:   "Should block path traversal attempts using ../",
		},
		{
			name:          "Path traversal with ../../ attack",
			fileName:      "../../secret/sensitive.txt",
			indexFile:     "index.md",
			shouldSucceed: false,
			description:   "Should block deep path traversal attempts",
		},
		{
			name:          "Path traversal with encoded ../ attack",
			fileName:      "%2e%2e/secret/sensitive.txt",
			indexFile:     "index.md",
			shouldSucceed: false,
			description:   "Should block URL-encoded path traversal attempts",
		},
		{
			name:          "Mixed path traversal",
			fileName:      "api/../../secret/sensitive.txt",
			indexFile:     "index.md",
			shouldSucceed: false,
			description:   "Should block mixed legitimate and traversal paths",
		},
		{
			name:          "Absolute path attack",
			fileName:      "/etc/passwd",
			indexFile:     "index.md",
			shouldSucceed: false,
			description:   "Should block absolute path attempts",
		},
		{
			name:          "Windows drive traversal",
			fileName:      "C:\\Windows\\System32\\config\\sam",
			indexFile:     "index.md",
			shouldSucceed: false,
			description:   "Should block Windows absolute path attempts",
		},
		{
			name:          "Null byte injection",
			fileName:      "index.md\x00../../secret/sensitive.txt",
			indexFile:     "index.md",
			shouldSucceed: false,
			description:   "Should block null byte injection attacks",
		},
		{
			name:          "Directory traversal via symlink name",
			fileName:      "../secret",
			indexFile:     "index.md",
			shouldSucceed: false,
			description:   "Should block traversal to directories outside allowed folder",
		},
		{
			name:          "Index file traversal attack",
			fileName:      "api",
			indexFile:     "../../../secret/sensitive.txt",
			shouldSucceed: false,
			description:   "Should block path traversal in index file parameter",
		},
		{
			name:          "Complex traversal with valid subdirectory",
			fileName:      "api/../../../secret/sensitive.txt",
			indexFile:     "index.md",
			shouldSucceed: false,
			description:   "Should block complex traversal even when starting with valid path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := server.serveFolderFile(
				docsDir,         // folderPath (docs directory)
				tc.fileName,     // fileName (potentially malicious)
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

				// Parse the response to check if it's successful
				var response map[string]any
				if err := json.Unmarshal(result, &response); err != nil {
					t.Errorf("Failed to parse response JSON: %v", err)
					return
				}

				if code, ok := response["code"].(float64); !ok || code != 200 {
					t.Errorf("Expected successful response (code 200), got: %v", response)
				}
			} else {
				if err == nil {
					// If no error returned, check if it's a 404 response
					var response map[string]any
					if err := json.Unmarshal(result, &response); err != nil {
						t.Errorf("Failed to parse response JSON: %v", err)
						return
					}

					if code, ok := response["code"].(float64); ok && code == 404 {
						// 404 is acceptable for security (file not found due to security restrictions)
						return
					}

					t.Errorf("Expected security error but got successful response: %v", response)
				} else {
					// Check if the error message indicates security violation
					errMsg := err.Error()
					securityKeywords := []string{
						"access denied",
						"outside allowed folder",
						"failed to resolve",
					}

					foundSecurityKeyword := false
					for _, keyword := range securityKeywords {
						if strings.Contains(strings.ToLower(errMsg), keyword) {
							foundSecurityKeyword = true
							break
						}
					}

					if !foundSecurityKeyword {
						t.Errorf("Expected security-related error but got: %v", err)
					}
				}
			}
		})
	}
}

func TestIndexFileTraversalSecurity(t *testing.T) {
	// Create a temporary test directory structure
	tempDir, err := os.MkdirTemp("", "index_security_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test structure:
	// tempDir/
	//   docs/
	//     api/
	//       (empty directory)
	//   secret/
	//     index.md (malicious content)
	docsDir := filepath.Join(tempDir, "docs")
	apiDir := filepath.Join(docsDir, "api")
	secretDir := filepath.Join(tempDir, "secret")

	err = os.MkdirAll(apiDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}
	err = os.MkdirAll(secretDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create secret dir: %v", err)
	}

	// Create a malicious index.md file outside the allowed directory
	err = os.WriteFile(filepath.Join(secretDir, "index.md"), []byte("# SECRET CONTENT"), 0644)
	if err != nil {
		t.Fatalf("Failed to create secret index.md: %v", err)
	}

	server := &Server{}

	// Test accessing a directory with traversal in the index file path
	maliciousIndexPaths := []string{
		"../../secret/index.md",
		"../../../secret/index.md",
		"/secret/index.md",
		"../../secret/../secret/index.md",
		"..\\..\\secret\\index.md", // Windows style
	}

	for _, indexPath := range maliciousIndexPaths {
		t.Run("IndexTraversal_"+indexPath, func(t *testing.T) {
			result, err := server.serveFolderFile(
				docsDir,         // folderPath (docs directory)
				"api",           // fileName (valid subdirectory)
				false,           // isMarkdown
				"markdown-html", // responseType
				indexPath,       // indexFile (malicious path)
				"",              // layoutFile
				"/docs/",        // basePath
			)

			// This should either return an error or a 404 response
			if err == nil {
				var response map[string]any
				if err := json.Unmarshal(result, &response); err != nil {
					t.Errorf("Failed to parse response JSON: %v", err)
					return
				}

				if code, ok := response["code"].(float64); !ok || code == 200 {
					t.Errorf("Expected security protection but got successful response: %v", response)
				}
			} else {
				// Check if the error indicates security protection
				errMsg := strings.ToLower(err.Error())
				if !strings.Contains(errMsg, "access denied") &&
					!strings.Contains(errMsg, "outside allowed folder") &&
					!strings.Contains(errMsg, "failed to resolve") {
					t.Errorf("Expected security error but got: %v", err)
				}
			}
		})
	}
}

func TestSymlinkSecurity(t *testing.T) {
	// Create a temporary test directory structure
	tempDir, err := os.MkdirTemp("", "symlink_security_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test structure:
	// tempDir/
	//   docs/
	//     index.md
	//   secret/
	//     sensitive.txt
	//   link_to_secret -> ../secret/sensitive.txt (symlink)
	docsDir := filepath.Join(tempDir, "docs")
	secretDir := filepath.Join(tempDir, "secret")

	err = os.MkdirAll(docsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create docs dir: %v", err)
	}
	err = os.MkdirAll(secretDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create secret dir: %v", err)
	}

	// Create files
	err = os.WriteFile(filepath.Join(docsDir, "index.md"), []byte("# Docs"), 0644)
	if err != nil {
		t.Fatalf("Failed to create docs index.md: %v", err)
	}

	err = os.WriteFile(filepath.Join(secretDir, "sensitive.txt"), []byte("SECRET"), 0644)
	if err != nil {
		t.Fatalf("Failed to create sensitive.txt: %v", err)
	}

	// Create symlink pointing outside the allowed directory
	symlinkPath := filepath.Join(docsDir, "link_to_secret")
	targetPath := filepath.Join("..", "secret", "sensitive.txt")
	err = os.Symlink(targetPath, symlinkPath)
	if err != nil {
		// Skip this test if symlinks are not supported (e.g., on Windows without admin)
		t.Skipf("Symlinks not supported on this system: %v", err)
	}

	server := &Server{}

	// Test accessing the symlink
	result, err := server.serveFolderFile(
		docsDir,          // folderPath
		"link_to_secret", // fileName (symlink)
		false,            // isMarkdown
		"text",           // responseType
		"index.md",       // indexFile
		"",               // layoutFile
		"/docs/",         // basePath
	)

	// This should be blocked by the security check
	if err == nil {
		var response map[string]any
		if err := json.Unmarshal(result, &response); err != nil {
			t.Errorf("Failed to parse response JSON: %v", err)
			return
		}

		if code, ok := response["code"].(float64); ok && code == 200 {
			if respData, ok := response["response"].(string); ok && strings.Contains(respData, "SECRET") {
				t.Errorf("Symlink traversal attack succeeded! Got secret content: %v", response)
			}
		}
	} else {
		// Verify it's a security-related error
		errMsg := strings.ToLower(err.Error())
		if !strings.Contains(errMsg, "access denied") &&
			!strings.Contains(errMsg, "outside allowed folder") &&
			!strings.Contains(errMsg, "failed to resolve") {
			t.Errorf("Expected security error for symlink traversal but got: %v", err)
		}
	}
}

func TestEdgeCaseSecurityScenarios(t *testing.T) {
	// Create a temporary test directory structure
	tempDir, err := os.MkdirTemp("", "edge_case_security_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test structure
	docsDir := filepath.Join(tempDir, "docs")
	err = os.MkdirAll(docsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create docs dir: %v", err)
	}

	err = os.WriteFile(filepath.Join(docsDir, "index.md"), []byte("# Docs"), 0644)
	if err != nil {
		t.Fatalf("Failed to create index.md: %v", err)
	}

	server := &Server{}

	// Test edge cases
	edgeCases := []struct {
		name      string
		fileName  string
		indexFile string
	}{
		{"Empty filename", "", "index.md"},
		{"Filename with spaces", "file with spaces.md", "index.md"},
		{"Unicode filename", "файл.md", "index.md"},
		{"Very long filename", strings.Repeat("a", 255), "index.md"},
		{"Filename with special chars", "file!@#$%^&*().md", "index.md"},
		{"Index file with spaces", "normal.md", "index file.md"},
		{"Unicode index file", "normal.md", "индекс.md"},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := server.serveFolderFile(
				docsDir,         // folderPath
				tc.fileName,     // fileName
				false,           // isMarkdown
				"markdown-html", // responseType
				tc.indexFile,    // indexFile
				"",              // layoutFile
				"/docs/",        // basePath
			)

			// These should either succeed (if file exists) or fail gracefully with 404
			// but should never cause security issues
			if err != nil {
				errMsg := strings.ToLower(err.Error())
				// Should not contain panic or unexpected errors
				unexpectedErrors := []string{"panic", "runtime error", "segmentation fault"}
				for _, unexpected := range unexpectedErrors {
					if strings.Contains(errMsg, unexpected) {
						t.Errorf("Unexpected error for edge case %s: %v", tc.name, err)
					}
				}
			}
		})
	}
}
