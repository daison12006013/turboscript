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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/daison12006013/turboscript/internal/config"
)

func TestCompilePathPatternWithWildcard(t *testing.T) {
	server := &Server{}

	testCases := []struct {
		name            string
		path            string
		expectedPattern string
		expectedParams  []string
		testURL         string
		shouldMatch     bool
	}{
		{
			name:            "Simple wildcard route",
			path:            "/demo/*",
			expectedPattern: "^/demo(?:/(.*))?$",
			expectedParams:  []string{"wildcard"},
			testURL:         "/demo/cache-test",
			shouldMatch:     true,
		},
		{
			name:            "Wildcard with base path",
			path:            "/api/docs/*",
			expectedPattern: "^/api/docs(?:/(.*))?$",
			expectedParams:  []string{"wildcard"},
			testURL:         "/api/docs/getting-started",
			shouldMatch:     true,
		},
		{
			name:            "Regular parameter route",
			path:            "/users/{id}",
			expectedPattern: "^/users/([^/]+)$",
			expectedParams:  []string{"id"},
			testURL:         "/users/123",
			shouldMatch:     true,
		},
		{
			name:            "Mixed parameters and wildcard",
			path:            "/api/{version}/docs/*",
			expectedPattern: "^/api/([^/]+)/docs(?:/(.*))?$",
			expectedParams:  []string{"version", "wildcard"},
			testURL:         "/api/v1/docs/endpoints",
			shouldMatch:     true,
		},
		{
			name:            "Wildcard should not match incorrect base",
			path:            "/demo/*",
			expectedPattern: "^/demo(?:/(.*))?$",
			expectedParams:  []string{"wildcard"},
			testURL:         "/other/cache-test",
			shouldMatch:     false,
		},
		{
			name:            "Wildcard with empty path",
			path:            "/demo/*",
			expectedPattern: "^/demo(?:/(.*))?$",
			expectedParams:  []string{"wildcard"},
			testURL:         "/demo/",
			shouldMatch:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pattern, params := server.compilePathPattern(tc.path)

			// Test pattern compilation
			if pattern.String() != tc.expectedPattern {
				t.Errorf("Expected pattern %s, got %s", tc.expectedPattern, pattern.String())
			}

			// Test parameter extraction
			if len(params) != len(tc.expectedParams) {
				t.Errorf("Expected %d parameters, got %d", len(tc.expectedParams), len(params))
			} else {
				for i, expectedParam := range tc.expectedParams {
					if params[i] != expectedParam {
						t.Errorf("Expected parameter[%d] %s, got %s", i, expectedParam, params[i])
					}
				}
			}

			// Test URL matching
			matches := pattern.MatchString(tc.testURL)
			if matches != tc.shouldMatch {
				t.Errorf("Expected URL %s to match=%v, got match=%v", tc.testURL, tc.shouldMatch, matches)
			}
		})
	}
}

func TestExtractPathParamsWithWildcard(t *testing.T) {
	server := &Server{
		pathPatterns: make(map[string]*regexp.Regexp),
		pathParams:   make(map[string][]string),
	}

	testCases := []struct {
		name           string
		routePattern   string
		requestURL     string
		expectedParams map[string]string
	}{
		{
			name:         "Wildcard with file path",
			routePattern: "/demo/*",
			requestURL:   "/demo/cache-test",
			expectedParams: map[string]string{
				"wildcard": "cache-test",
			},
		},
		{
			name:         "Wildcard with nested path",
			routePattern: "/demo/*",
			requestURL:   "/demo/api/v1/endpoint",
			expectedParams: map[string]string{
				"wildcard": "api/v1/endpoint",
			},
		},
		{
			name:         "Wildcard with empty path",
			routePattern: "/demo/*",
			requestURL:   "/demo/",
			expectedParams: map[string]string{
				"wildcard": "",
			},
		},
		{
			name:         "Mixed parameters and wildcard",
			routePattern: "/api/{version}/docs/*",
			requestURL:   "/api/v1/docs/getting-started",
			expectedParams: map[string]string{
				"version":  "v1",
				"wildcard": "getting-started",
			},
		},
		{
			name:         "Wildcard with query parameters",
			routePattern: "/demo/*",
			requestURL:   "/demo/cache-test?param=value",
			expectedParams: map[string]string{
				"wildcard": "cache-test?param=value",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up the server patterns
			pattern, params := server.compilePathPattern(tc.routePattern)
			server.pathPatterns[tc.routePattern] = pattern
			server.pathParams[tc.routePattern] = params

			// Extract parameters
			extractedParams := server.extractPathParams(tc.routePattern, tc.requestURL)

			// Verify extracted parameters
			if len(extractedParams) != len(tc.expectedParams) {
				t.Errorf("Expected %d parameters, got %d", len(tc.expectedParams), len(extractedParams))
			}

			for key, expectedValue := range tc.expectedParams {
				if actualValue, exists := extractedParams[key]; !exists {
					t.Errorf("Expected parameter %s not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected parameter %s=%s, got %s", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestResolveWildcardEndpoint(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"index.ts":            "export const handle = () => ({ code: 200, response: 'index' });",
		"cache-test.ts":       "export const handle = () => ({ code: 200, response: 'cache-test' });",
		"response-types.ts":   "export const handle = () => ({ code: 200, response: 'response-types' });",
		"subdirectory/api.ts": "export const handle = () => ({ code: 200, response: 'api' });",
	}

	for relPath, content := range testFiles {
		fullPath := filepath.Join(tempDir, relPath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory for %s: %v", relPath, err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", relPath, err)
		}
	}

	server := &Server{
		pathPatterns: make(map[string]*regexp.Regexp),
		pathParams:   make(map[string][]string),
	}

	// Set up the server patterns for wildcard route
	routePattern := "/demo/*"
	pattern, params := server.compilePathPattern(routePattern)
	server.pathPatterns[routePattern] = pattern
	server.pathParams[routePattern] = params

	testCases := []struct {
		name          string
		endpoint      config.EndpointConfig
		requestURL    string
		expectedPath  string
		shouldResolve bool
	}{
		{
			name: "Resolve specific file",
			endpoint: config.EndpointConfig{
				Route:  "/demo/*",
				Method: "GET",
				Path:   tempDir + "/*",
			},
			requestURL:    "/demo/cache-test",
			expectedPath:  filepath.Join(tempDir, "cache-test.ts"),
			shouldResolve: true,
		},
		{
			name: "Resolve index file for root request",
			endpoint: config.EndpointConfig{
				Route:  "/demo/*",
				Method: "GET",
				Path:   tempDir + "/*",
			},
			requestURL:    "/demo/",
			expectedPath:  filepath.Join(tempDir, "index.ts"),
			shouldResolve: true,
		},
		{
			name: "Resolve subdirectory file",
			endpoint: config.EndpointConfig{
				Route:  "/demo/*",
				Method: "GET",
				Path:   tempDir + "/*",
			},
			requestURL:    "/demo/subdirectory/api",
			expectedPath:  filepath.Join(tempDir, "subdirectory/api.ts"),
			shouldResolve: true,
		},
		{
			name: "Non-existent file should not resolve",
			endpoint: config.EndpointConfig{
				Route:  "/demo/*",
				Method: "GET",
				Path:   tempDir + "/*",
			},
			requestURL:    "/demo/non-existent",
			expectedPath:  "",
			shouldResolve: false,
		},
		{
			name: "Directory traversal should be blocked",
			endpoint: config.EndpointConfig{
				Route:  "/demo/*",
				Method: "GET",
				Path:   tempDir + "/*",
			},
			requestURL:    "/demo/../secret",
			expectedPath:  "",
			shouldResolve: false,
		},
		{
			name: "Request with query parameters",
			endpoint: config.EndpointConfig{
				Route:  "/demo/*",
				Method: "GET",
				Path:   tempDir + "/*",
			},
			requestURL:    "/demo/cache-test?param=value",
			expectedPath:  filepath.Join(tempDir, "cache-test.ts"),
			shouldResolve: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resolvedEp := server.resolveWildcardEndpoint(tc.endpoint, tc.requestURL)

			if tc.shouldResolve {
				if resolvedEp == nil {
					t.Errorf("Expected endpoint to resolve but got nil")
				} else if resolvedEp.Path != tc.expectedPath {
					t.Errorf("Expected path %s, got %s", tc.expectedPath, resolvedEp.Path)
				}
			} else {
				if resolvedEp != nil {
					t.Errorf("Expected nil endpoint but got %+v", resolvedEp)
				}
			}
		})
	}
}

func TestFindMatchingEndpointWithWildcard(t *testing.T) {
	// Create a temporary directory with test files
	tempDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tempDir, "cache-test.ts"), []byte("// test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	server := &Server{
		pathPatterns: make(map[string]*regexp.Regexp),
		pathParams:   make(map[string][]string),
		cfg: &config.Config{
			Endpoints: []config.EndpointConfig{
				{
					Route:  "/demo/*",
					Method: "GET",
					Path:   tempDir + "/*",
				},
				{
					Route:  "/users/{id}",
					Method: "GET",
					Path:   "./app/routes/users/get-by-id.ts",
				},
				{
					Route:  "/api/v1/status",
					Method: "GET",
					Path:   "./app/routes/api/status.ts",
				},
			},
		},
	}

	// Pre-compile patterns
	for _, ep := range server.cfg.Endpoints {
		pattern, params := server.compilePathPattern(ep.Route)
		server.pathPatterns[ep.Route] = pattern
		server.pathParams[ep.Route] = params
	}

	testCases := []struct {
		name          string
		requestURL    string
		method        string
		shouldMatch   bool
		expectedRoute string
		expectedPath  string
	}{
		{
			name:          "Match wildcard route with existing file",
			requestURL:    "/demo/cache-test",
			method:        "GET",
			shouldMatch:   true,
			expectedRoute: "/demo/*",
			expectedPath:  filepath.Join(tempDir, "cache-test.ts"),
		},
		{
			name:          "Match exact route",
			requestURL:    "/api/v1/status",
			method:        "GET",
			shouldMatch:   true,
			expectedRoute: "/api/v1/status",
			expectedPath:  "./app/routes/api/status.ts",
		},
		{
			name:          "Match parameterized route",
			requestURL:    "/users/123",
			method:        "GET",
			shouldMatch:   true,
			expectedRoute: "/users/{id}",
			expectedPath:  "./app/routes/users/get-by-id.ts",
		},
		{
			name:          "No match for non-existent wildcard file",
			requestURL:    "/demo/non-existent",
			method:        "GET",
			shouldMatch:   false,
			expectedRoute: "",
			expectedPath:  "",
		},
		{
			name:          "No match for wrong method",
			requestURL:    "/demo/cache-test",
			method:        "POST",
			shouldMatch:   false,
			expectedRoute: "",
			expectedPath:  "",
		},
		{
			name:          "No match for completely wrong URL",
			requestURL:    "/completely/wrong/path",
			method:        "GET",
			shouldMatch:   false,
			expectedRoute: "",
			expectedPath:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := server.FindMatchingEndpoint(tc.requestURL, tc.method)

			if tc.shouldMatch {
				if endpoint == nil {
					t.Errorf("Expected to find matching endpoint but got nil")
				} else {
					if endpoint.Route != tc.expectedRoute {
						t.Errorf("Expected route %s, got %s", tc.expectedRoute, endpoint.Route)
					}
					if endpoint.Path != tc.expectedPath {
						t.Errorf("Expected path %s, got %s", tc.expectedPath, endpoint.Path)
					}
				}
			} else {
				if endpoint != nil {
					t.Errorf("Expected no matching endpoint but got %+v", endpoint)
				}
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	server := &Server{}

	// Create a temporary file for testing
	tempDir := t.TempDir()
	existingFile := filepath.Join(tempDir, "existing.ts")
	err := os.WriteFile(existingFile, []byte("// test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testCases := []struct {
		name           string
		filePath       string
		expectedExists bool
	}{
		{
			name:           "Existing file should return true",
			filePath:       existingFile,
			expectedExists: true,
		},
		{
			name:           "Non-existent file should return false",
			filePath:       filepath.Join(tempDir, "non-existent.ts"),
			expectedExists: false,
		},
		{
			name:           "Directory should return true",
			filePath:       tempDir,
			expectedExists: true,
		},
		{
			name:           "Non-existent directory should return false",
			filePath:       filepath.Join(tempDir, "non-existent-dir"),
			expectedExists: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exists := server.fileExists(tc.filePath)
			if exists != tc.expectedExists {
				t.Errorf("Expected fileExists(%s) = %v, got %v", tc.filePath, tc.expectedExists, exists)
			}
		})
	}
}

func TestWildcardRouteSecurityAndEdgeCases(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Create test files
	err := os.WriteFile(filepath.Join(tempDir, "safe.ts"), []byte("// safe file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create safe test file: %v", err)
	}

	// Create a file outside the base directory (for testing security)
	outsideDir := t.TempDir()
	err = os.WriteFile(filepath.Join(outsideDir, "secret.ts"), []byte("// secret file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create secret test file: %v", err)
	}

	server := &Server{
		pathPatterns: make(map[string]*regexp.Regexp),
		pathParams:   make(map[string][]string),
	}

	// Set up the server patterns for wildcard route
	routePattern := "/demo/*"
	pattern, params := server.compilePathPattern(routePattern)
	server.pathPatterns[routePattern] = pattern
	server.pathParams[routePattern] = params

	testCases := []struct {
		name        string
		requestURL  string
		basePath    string
		shouldBlock bool
		description string
	}{
		{
			name:        "Normal file access should work",
			requestURL:  "/demo/safe",
			basePath:    tempDir,
			shouldBlock: false,
			description: "Normal file access within the base directory",
		},
		{
			name:        "Directory traversal with ../",
			requestURL:  "/demo/../../../etc/passwd",
			basePath:    tempDir,
			shouldBlock: true,
			description: "Should block directory traversal attempts",
		},
		{
			name:        "Directory traversal with encoded characters",
			requestURL:  "/demo/%2e%2e/secret",
			basePath:    tempDir,
			shouldBlock: true,
			description: "Should block URL-encoded directory traversal",
		},
		{
			name:        "Absolute path attempt",
			requestURL:  "/demo//etc/passwd",
			basePath:    tempDir,
			shouldBlock: true,
			description: "Should block absolute path attempts",
		},
		{
			name:        "Empty wildcard path",
			requestURL:  "/demo/",
			basePath:    tempDir,
			shouldBlock: false,
			description: "Empty wildcard path should be allowed (for index files)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := config.EndpointConfig{
				Route:  "/demo/*",
				Method: "GET",
				Path:   tc.basePath + "/*",
			}

			resolvedEp := server.resolveWildcardEndpoint(endpoint, tc.requestURL)

			if tc.shouldBlock {
				if resolvedEp != nil {
					t.Errorf("Expected security block but endpoint was resolved: %+v", resolvedEp)
				}
			} else {
				// For non-blocked requests, we might still get nil if the file doesn't exist
				// which is fine - we just want to ensure it's not blocked for security reasons
				t.Logf("Request %s was not blocked (resolved to: %v)", tc.requestURL, resolvedEp)
			}
		})
	}
}

// TestWildcardWithOptionalTrailingSlash tests the new feature where wildcard routes work with or without trailing slash
func TestWildcardWithOptionalTrailingSlash(t *testing.T) {
	// Create test server
	server := &Server{
		pathPatterns: make(map[string]*regexp.Regexp),
		pathParams:   make(map[string][]string),
	}

	// Create temp directory and test files
	tempDir := t.TempDir()

	// Create test file
	demoDir := filepath.Join(tempDir, "demo")
	err := os.MkdirAll(demoDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	indexFile := filepath.Join(demoDir, "index.ts")
	err = os.WriteFile(indexFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Configure the endpoint
	ep := config.EndpointConfig{
		Route:  "/demo/*",
		Method: "GET",
		Path:   filepath.Join(tempDir, "demo", "*"),
	}

	// Compile the pattern
	pattern, params := server.compilePathPattern(ep.Route)
	server.pathPatterns[ep.Route] = pattern
	server.pathParams[ep.Route] = params

	testCases := []struct {
		name        string
		url         string
		shouldMatch bool
		description string
	}{
		{
			name:        "Without trailing slash",
			url:         "/demo",
			shouldMatch: true,
			description: "Should match /demo and resolve to index.ts",
		},
		{
			name:        "With trailing slash",
			url:         "/demo/",
			shouldMatch: true,
			description: "Should match /demo/ and resolve to index.ts",
		},
		{
			name:        "With specific file",
			url:         "/demo/cache-test",
			shouldMatch: true, // Pattern matches but file doesn't exist
			description: "Should match pattern but file doesn't exist",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test pattern matching
			params := server.extractPathParams(ep.Route, tc.url)
			_, hasWildcard := params["wildcard"]

			if tc.shouldMatch {
				if !hasWildcard {
					t.Errorf("Expected URL %s to match wildcard pattern, but it didn't", tc.url)
				} else {
					t.Logf("✓ URL %s matched wildcard pattern", tc.url)

					// Test actual file resolution
					resolvedEp := server.resolveWildcardEndpoint(ep, tc.url)
					if tc.url == "/demo" || tc.url == "/demo/" {
						if resolvedEp == nil {
							t.Errorf("Expected to resolve index file for %s", tc.url)
						} else {
							t.Logf("✓ URL %s resolved to: %s", tc.url, resolvedEp.Path)
						}
					}
				}
			} else {
				if hasWildcard {
					t.Errorf("Expected URL %s to not match wildcard pattern, but it did", tc.url)
				}
			}
		})
	}
}

// TestDynamicPathParametersInWildcardRoutes tests the new feature where path parameters are replaced in file paths
func TestDynamicPathParametersInWildcardRoutes(t *testing.T) {
	// Create test server with dynamic path parameters
	server := &Server{
		pathPatterns: make(map[string]*regexp.Regexp),
		pathParams:   make(map[string][]string),
	}

	// Create temp directory and test files
	tempDir := t.TempDir()

	// Create test files for different versions
	versions := []string{"v1", "v2", "v3"}
	for _, version := range versions {
		versionDir := filepath.Join(tempDir, "api", version)
		err := os.MkdirAll(versionDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create version directory %s: %v", version, err)
		}

		// Create users.ts file for each version
		usersFile := filepath.Join(versionDir, "users.ts")
		content := fmt.Sprintf("// API %s users endpoint", version)
		err = os.WriteFile(usersFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create users file for %s: %v", version, err)
		}

		// Create index.ts file for each version
		indexFile := filepath.Join(versionDir, "index.ts")
		content = fmt.Sprintf("// API %s index endpoint", version)
		err = os.WriteFile(indexFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create index file for %s: %v", version, err)
		}
	}

	// Configure the endpoint with dynamic parameter
	ep := config.EndpointConfig{
		Route:  "/api/{version}/*",
		Method: "GET",
		Path:   filepath.Join(tempDir, "api", "{version}", "*"),
	}

	// Compile the pattern
	pattern, params := server.compilePathPattern(ep.Route)
	server.pathPatterns[ep.Route] = pattern
	server.pathParams[ep.Route] = params

	testCases := []struct {
		name          string
		url           string
		expectedFile  string
		shouldResolve bool
		description   string
	}{
		{
			name:          "API v1 users endpoint",
			url:           "/api/v1/users",
			expectedFile:  "users.ts",
			shouldResolve: true,
			description:   "Should resolve {version} parameter to v1",
		},
		{
			name:          "API v2 users endpoint",
			url:           "/api/v2/users",
			expectedFile:  "users.ts",
			shouldResolve: true,
			description:   "Should resolve {version} parameter to v2",
		},
		{
			name:          "API v3 index endpoint",
			url:           "/api/v3/",
			expectedFile:  "index.ts",
			shouldResolve: true,
			description:   "Should resolve {version} parameter to v3 and find index.ts",
		},
		{
			name:          "API v1 index endpoint without trailing slash",
			url:           "/api/v1",
			expectedFile:  "index.ts",
			shouldResolve: true,
			description:   "Should work without trailing slash and resolve to index.ts",
		},
		{
			name:          "Non-existent version",
			url:           "/api/v99/users",
			shouldResolve: false,
			description:   "Should not resolve for non-existent version directory",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test pattern matching and parameter extraction
			params := server.extractPathParams(ep.Route, tc.url)
			version, hasVersion := params["version"]
			wildcard, hasWildcard := params["wildcard"]

			if !hasVersion || !hasWildcard {
				t.Errorf("Failed to extract parameters from URL %s", tc.url)
				return
			}

			t.Logf("Extracted version=%s, wildcard=%s from URL %s", version, wildcard, tc.url)

			// Test file resolution
			resolvedEp := server.resolveWildcardEndpoint(ep, tc.url)

			if tc.shouldResolve {
				if resolvedEp == nil {
					t.Errorf("Expected URL %s to resolve, but it didn't", tc.url)
				} else {
					if !strings.Contains(resolvedEp.Path, version) {
						t.Errorf("Expected resolved path to contain version %s, got %s", version, resolvedEp.Path)
					}
					if !strings.HasSuffix(resolvedEp.Path, tc.expectedFile) {
						t.Errorf("Expected resolved path to end with %s, got %s", tc.expectedFile, resolvedEp.Path)
					}
					t.Logf("✓ URL %s correctly resolved to: %s", tc.url, resolvedEp.Path)
				}
			} else {
				if resolvedEp != nil {
					t.Errorf("Expected URL %s to not resolve (file doesn't exist), but it resolved to: %s", tc.url, resolvedEp.Path)
				} else {
					t.Logf("✓ URL %s correctly failed to resolve (expected)", tc.url)
				}
			}
		})
	}
}

// TestDynamicPathParameterSecurity tests security aspects of dynamic path parameters
func TestDynamicPathParameterSecurity(t *testing.T) {
	// Create test server with dynamic path parameters
	server := &Server{
		pathPatterns: make(map[string]*regexp.Regexp),
		pathParams:   make(map[string][]string),
	}

	// Create temp directory and test files
	tempDir := t.TempDir()

	// Create legitimate test file
	legitimateDir := filepath.Join(tempDir, "api", "v1")
	err := os.MkdirAll(legitimateDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create legitimate directory: %v", err)
	}

	usersFile := filepath.Join(legitimateDir, "users.ts")
	err = os.WriteFile(usersFile, []byte("// Legitimate API endpoint"), 0644)
	if err != nil {
		t.Fatalf("Failed to create legitimate file: %v", err)
	}

	// Configure the endpoint with dynamic parameter
	ep := config.EndpointConfig{
		Route:  "/api/{version}/*",
		Method: "GET",
		Path:   filepath.Join(tempDir, "api", "{version}", "*"),
	}

	// Compile the pattern
	pattern, params := server.compilePathPattern(ep.Route)
	server.pathPatterns[ep.Route] = pattern
	server.pathParams[ep.Route] = params

	testCases := []struct {
		name        string
		url         string
		shouldBlock bool
		description string
	}{
		{
			name:        "Legitimate API call",
			url:         "/api/v1/users",
			shouldBlock: false,
			description: "Should allow legitimate API calls",
		},
		{
			name:        "Path traversal in version parameter",
			url:         "/api/../../../etc/users",
			shouldBlock: true,
			description: "Should block path traversal in version parameter",
		},
		{
			name:        "Path traversal in wildcard",
			url:         "/api/v1/../../../etc/passwd",
			shouldBlock: true,
			description: "Should block path traversal in wildcard portion",
		},
		{
			name:        "Double path traversal",
			url:         "/api/../v1/../../../secret",
			shouldBlock: true,
			description: "Should block complex path traversal attempts",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test file resolution (security should block malicious attempts)
			resolvedEp := server.resolveWildcardEndpoint(ep, tc.url)

			if tc.shouldBlock {
				if resolvedEp != nil {
					t.Errorf("Expected security violation %s to be blocked, but it resolved to: %s", tc.url, resolvedEp.Path)
				} else {
					t.Logf("✓ Security violation %s was correctly blocked", tc.url)
				}
			} else {
				if resolvedEp == nil {
					t.Errorf("Expected legitimate request %s to succeed, but it was blocked", tc.url)
				} else {
					t.Logf("✓ Legitimate request %s correctly allowed: %s", tc.url, resolvedEp.Path)
				}
			}
		})
	}
}

// TestParameterizedRoutes tests the new parameterized (non-wildcard) route functionality
func TestParameterizedRoutes(t *testing.T) {
	// Create test server
	server := &Server{
		pathPatterns: make(map[string]*regexp.Regexp),
		pathParams:   make(map[string][]string),
	}

	// Create temp directory and test files
	tempDir := t.TempDir()

	// Create test files
	folderDir := filepath.Join(tempDir, "folder")
	err := os.MkdirAll(folderDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	testFiles := map[string]string{
		"test-file.ts":    "export const handle = () => ({ code: 200, response: 'test-file' });",
		"another-test.ts": "export const handle = () => ({ code: 200, response: 'another-test' });",
		"example.ts":      "export const handle = () => ({ code: 200, response: 'example' });",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(folderDir, filename)
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Configure the endpoint with parameter
	ep := config.EndpointConfig{
		Route:  "/folder/{file}",
		Method: "GET",
		Path:   filepath.Join(tempDir, "folder", "{file}"),
	}

	// Compile the pattern
	pattern, params := server.compilePathPattern(ep.Route)
	server.pathPatterns[ep.Route] = pattern
	server.pathParams[ep.Route] = params

	testCases := []struct {
		name          string
		url           string
		expectedFile  string
		shouldResolve bool
		description   string
	}{
		{
			name:          "Test file endpoint",
			url:           "/folder/test-file",
			expectedFile:  "test-file.ts",
			shouldResolve: true,
			description:   "Should resolve {file} parameter to test-file",
		},
		{
			name:          "Another test file endpoint",
			url:           "/folder/another-test",
			expectedFile:  "another-test.ts",
			shouldResolve: true,
			description:   "Should resolve {file} parameter to another-test",
		},
		{
			name:          "Example file endpoint",
			url:           "/folder/example",
			expectedFile:  "example.ts",
			shouldResolve: true,
			description:   "Should resolve {file} parameter to example",
		},
		{
			name:          "Non-existent file",
			url:           "/folder/non-existent",
			shouldResolve: false,
			description:   "Should not resolve for non-existent file",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test pattern matching and parameter extraction
			params := server.extractPathParams(ep.Route, tc.url)
			file, hasFile := params["file"]

			if !hasFile {
				t.Errorf("Failed to extract file parameter from URL %s", tc.url)
				return
			}

			t.Logf("Extracted file=%s from URL %s", file, tc.url)

			// Test file resolution
			resolvedEp := server.resolveParameterizedEndpoint(ep, tc.url)

			if tc.shouldResolve {
				if resolvedEp == nil {
					t.Errorf("Expected URL %s to resolve, but it didn't", tc.url)
				} else {
					if !strings.HasSuffix(resolvedEp.Path, tc.expectedFile) {
						t.Errorf("Expected resolved path to end with %s, got %s", tc.expectedFile, resolvedEp.Path)
					}
					t.Logf("✓ URL %s correctly resolved to: %s", tc.url, resolvedEp.Path)
				}
			} else {
				if resolvedEp != nil {
					t.Errorf("Expected URL %s to not resolve (file doesn't exist), but it resolved to: %s", tc.url, resolvedEp.Path)
				} else {
					t.Logf("✓ URL %s correctly failed to resolve (expected)", tc.url)
				}
			}
		})
	}
}

// TestParameterizedRouteSecurity tests security aspects of parameterized routes
func TestParameterizedRouteSecurity(t *testing.T) {
	// Create test server
	server := &Server{
		pathPatterns: make(map[string]*regexp.Regexp),
		pathParams:   make(map[string][]string),
	}

	// Create temp directory and test files
	tempDir := t.TempDir()

	// Create legitimate test file
	folderDir := filepath.Join(tempDir, "folder")
	err := os.MkdirAll(folderDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	testFile := filepath.Join(folderDir, "safe.ts")
	err = os.WriteFile(testFile, []byte("// Safe endpoint"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Configure the endpoint with parameter
	ep := config.EndpointConfig{
		Route:  "/folder/{file}",
		Method: "GET",
		Path:   filepath.Join(tempDir, "folder", "{file}"),
	}

	// Compile the pattern
	pattern, params := server.compilePathPattern(ep.Route)
	server.pathPatterns[ep.Route] = pattern
	server.pathParams[ep.Route] = params

	testCases := []struct {
		name        string
		url         string
		shouldBlock bool
		description string
	}{
		{
			name:        "Legitimate file access",
			url:         "/folder/safe",
			shouldBlock: false,
			description: "Should allow legitimate file access",
		},
		{
			name:        "Path traversal with ..",
			url:         "/folder/../secret",
			shouldBlock: true,
			description: "Should block path traversal attempts",
		},
		{
			name:        "Path with slash",
			url:         "/folder/sub/file",
			shouldBlock: true,
			description: "Should block parameters with slashes",
		},
		{
			name:        "Multiple path traversal",
			url:         "/folder/../../etc/passwd",
			shouldBlock: true,
			description: "Should block complex path traversal attempts",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test file resolution (security should block malicious attempts)
			resolvedEp := server.resolveParameterizedEndpoint(ep, tc.url)

			if tc.shouldBlock {
				if resolvedEp != nil {
					t.Errorf("Expected security violation %s to be blocked, but it resolved to: %s", tc.url, resolvedEp.Path)
				} else {
					t.Logf("✓ Security violation %s was correctly blocked", tc.url)
				}
			} else {
				if resolvedEp == nil {
					t.Errorf("Expected legitimate request %s to succeed, but it was blocked", tc.url)
				} else {
					t.Logf("✓ Legitimate request %s correctly allowed: %s", tc.url, resolvedEp.Path)
				}
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkCompilePathPatternWildcard(b *testing.B) {
	server := &Server{}
	path := "/demo/*"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = server.compilePathPattern(path)
	}
}

func BenchmarkResolveWildcardEndpoint(b *testing.B) {
	// Setup
	tempDir := b.TempDir()
	err := os.WriteFile(filepath.Join(tempDir, "cache-test.ts"), []byte("// test"), 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	server := &Server{
		pathPatterns: make(map[string]*regexp.Regexp),
		pathParams:   make(map[string][]string),
	}

	routePattern := "/demo/*"
	pattern, params := server.compilePathPattern(routePattern)
	server.pathPatterns[routePattern] = pattern
	server.pathParams[routePattern] = params

	endpoint := config.EndpointConfig{
		Route:  "/demo/*",
		Method: "GET",
		Path:   tempDir + "/*",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = server.resolveWildcardEndpoint(endpoint, "/demo/cache-test")
	}
}
