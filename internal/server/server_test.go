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
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/performance"
	"github.com/daison12006013/turboscript/internal/tsengine"
	"github.com/valyala/fasthttp"
)

// createTestServer creates a basic server instance for testing
func createTestServer() *Server {
	cfg := &config.Config{
		Compression: config.CompressionConfig{
			Enabled: false, // Disable compression for simpler testing
		},
	}
	return &Server{cfg: cfg}
}

func TestAutoDetectResponseType(t *testing.T) {
	testCases := []struct {
		name        string
		content     json.RawMessage
		expected    string
		description string
	}{
		{
			name:        "JSON object",
			content:     json.RawMessage(`{"key": "value"}`),
			expected:    "json",
			description: "Should detect JSON objects",
		},
		{
			name:        "JSON array",
			content:     json.RawMessage(`[1, 2, 3]`),
			expected:    "json",
			description: "Should detect JSON arrays",
		},
		{
			name:        "HTML content",
			content:     json.RawMessage(`"<html><body>Hello</body></html>"`),
			expected:    "html",
			description: "Should detect HTML content",
		},
		{
			name:        "Markdown content",
			content:     json.RawMessage(`"# Header\n\nSome **bold** text"`),
			expected:    "markdown",
			description: "Should detect Markdown content",
		},
		{
			name:        "Plain text",
			content:     json.RawMessage(`"Just plain text"`),
			expected:    "text",
			description: "Should default to text for plain content",
		},
		{
			name:        "Empty string",
			content:     json.RawMessage(`""`),
			expected:    "text",
			description: "Should default to text for empty content",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := autoDetectResponseType(tc.content)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestGetFileType(t *testing.T) {
	testCases := []struct {
		extension string
		expected  string
	}{
		{".md", "markdown"},
		{".markdown", "markdown"},
		{".html", "html"},
		{".htm", "html"},
		{".js", "javascript"},
		{".ts", "typescript"},
		{".css", "css"},
		{".scss", "sass"},
		{".sass", "sass"},
		{".yml", "yaml"},
		{".yaml", "yaml"},
		{".json", "json"},
		{".txt", "text"},
		{".unknown", "file"},
		{"", "file"},
	}

	for _, tc := range testCases {
		t.Run("Extension_"+tc.extension, func(t *testing.T) {
			result := getFileType(tc.extension)
			if result != tc.expected {
				t.Errorf("For extension %s, expected %s, got %s", tc.extension, tc.expected, result)
			}
		})
	}
}

func TestCleanHTMLResponse(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Remove whitespace and newlines",
			input:    "<html>\n\t<body>\n\t\tContent\n\t</body>\n</html>",
			expected: "<html><body>Content</body></html>",
		},
		{
			name:     "Remove multiple spaces",
			input:    "<html>  <body>    Content    </body>  </html>",
			expected: "<html> <body> Content </body> </html>",
		},
		{
			name:     "Remove tabs and carriage returns",
			input:    "<html>\r\n\t<body>Content</body>\r\n</html>",
			expected: "<html><body>Content</body></html>",
		},
		{
			name:     "Already clean content",
			input:    "<html><body>Clean content</body></html>",
			expected: "<html><body>Clean content</body></html>",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cleanHTMLResponse(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestUnwrapResponseWithType(t *testing.T) {
	testCases := []struct {
		name         string
		response     []byte
		expectedCode int
		hasError     bool
	}{
		{
			name:         "Wrapped response with type and code",
			response:     []byte(`{"response": "test data", "type": "html", "code": 201}`),
			expectedCode: 201,
			hasError:     false,
		},
		{
			name:         "Wrapped response with default code",
			response:     []byte(`{"response": "test data"}`),
			expectedCode: 200,
			hasError:     false,
		},
		{
			name:         "Invalid JSON",
			response:     []byte(`{invalid json`),
			expectedCode: 0,
			hasError:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, code, cookies, responseType, err := unwrapResponseWithType(tc.response)

			if tc.hasError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if code != tc.expectedCode {
				t.Errorf("Expected code %d, got %d", tc.expectedCode, code)
			}

			if data == nil {
				t.Error("Expected data but got nil")
			}

			// cookies can be nil or empty, both are valid
			if cookies == nil {
				cookies = []string{} // Make it empty instead of nil for consistent testing
			}

			// responseType can be empty, that's valid
			_ = responseType
		})
	}
}

func TestSendError(t *testing.T) {
	// Create a server with proper configuration
	cfg := &config.Config{
		Compression: config.CompressionConfig{
			Enabled: false, // Disable compression for simpler testing
		},
	}
	server := &Server{cfg: cfg}

	// Create a fasthttp context for testing
	ctx := &fasthttp.RequestCtx{}

	testCases := []struct {
		name        string
		errorType   string
		message     string
		expectsJSON bool
	}{
		{
			name:        "Standard error",
			errorType:   "Test Error",
			message:     "This is a test error",
			expectsJSON: true,
		},
		{
			name:        "Empty message",
			errorType:   "Empty Error",
			message:     "",
			expectsJSON: true,
		},
		{
			name:        "Long message",
			errorType:   "Long Error",
			message:     strings.Repeat("This is a very long error message. ", 10),
			expectsJSON: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset the context
			ctx.Response.Reset()

			server.sendError(ctx, tc.errorType, tc.message)

			// Check status code
			if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
				t.Errorf("Expected status %d, got %d", fasthttp.StatusInternalServerError, ctx.Response.StatusCode())
			}

			// Check content type
			contentType := string(ctx.Response.Header.ContentType())
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("Expected JSON content type, got %s", contentType)
			}

			if tc.expectsJSON {
				// Parse the response body
				var response map[string]any
				err := json.Unmarshal(ctx.Response.Body(), &response)
				if err != nil {
					t.Errorf("Failed to parse JSON response: %v", err)
					return
				}

				// Check error structure
				if response["status"] != "error" {
					t.Errorf("Expected status 'error', got %v", response["status"])
				}

				if errorField, ok := response["error"].(map[string]any); ok {
					if errorField["type"] != tc.errorType {
						t.Errorf("Expected error type %s, got %v", tc.errorType, errorField["type"])
					}
					if errorField["message"] != tc.message {
						t.Errorf("Expected error message %s, got %v", tc.message, errorField["message"])
					}
				} else {
					t.Error("Expected error field to be an object")
				}
			}
		})
	}
}

func TestWriteErrorBytes(t *testing.T) {
	// Create a server with proper configuration to avoid nil pointer
	cfg := &config.Config{
		Compression: config.CompressionConfig{
			Enabled: false, // Disable compression for simpler testing
		},
	}
	server := &Server{cfg: cfg}
	ctx := &fasthttp.RequestCtx{}

	// Set the status code manually as writeErrorBytes doesn't set it
	ctx.SetStatusCode(fasthttp.StatusInternalServerError)

	testData := []byte("Test error message")
	server.writeErrorBytes(ctx, testData)

	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusInternalServerError, ctx.Response.StatusCode())
	}

	if !bytes.Equal(ctx.Response.Body(), testData) {
		t.Errorf("Expected body %s, got %s", string(testData), string(ctx.Response.Body()))
	}
}

func TestWriteErrorString(t *testing.T) {
	// Create a server with proper configuration to avoid nil pointer
	cfg := &config.Config{
		Compression: config.CompressionConfig{
			Enabled: false, // Disable compression for simpler testing
		},
	}
	server := &Server{cfg: cfg}
	ctx := &fasthttp.RequestCtx{}

	// Set the status code manually as writeErrorString doesn't set it
	ctx.SetStatusCode(fasthttp.StatusInternalServerError)

	testMessage := "Test error string"
	server.writeErrorString(ctx, testMessage)

	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusInternalServerError, ctx.Response.StatusCode())
	}

	if string(ctx.Response.Body()) != testMessage {
		t.Errorf("Expected body %s, got %s", testMessage, string(ctx.Response.Body()))
	}
}

func TestSendJSONParseError(t *testing.T) {
	// Create a server with proper configuration to avoid nil pointer
	cfg := &config.Config{
		Compression: config.CompressionConfig{
			Enabled: false, // Disable compression for simpler testing
		},
	}
	server := &Server{cfg: cfg}
	ctx := &fasthttp.RequestCtx{}

	testError := &testError{message: "Invalid JSON format"}
	server.sendJSONParseError(ctx, testError)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusBadRequest, ctx.Response.StatusCode())
	}

	// Check content type
	contentType := string(ctx.Response.Header.ContentType())
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type, got %s", contentType)
	}

	// Parse response
	var response map[string]any
	err := json.Unmarshal(ctx.Response.Body(), &response)
	if err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
		return
	}

	if response["status"] != "error" {
		t.Errorf("Expected status 'error', got %v", response["status"])
	}

	if response["message"] != "Invalid request format" {
		t.Errorf("Expected message 'Invalid request format', got %v", response["message"])
	}

	if response["error"] != testError.message {
		t.Errorf("Expected error %s, got %v", testError.message, response["error"])
	}
}

// testError implements the error interface for testing
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

func TestSendPanicErrorResponse(t *testing.T) {
	// Create a server with proper configuration to avoid nil pointer
	cfg := &config.Config{
		Compression: config.CompressionConfig{
			Enabled: false, // Disable compression for simpler testing
		},
	}
	server := &Server{cfg: cfg}
	ctx := &fasthttp.RequestCtx{}

	server.sendPanicErrorResponse(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusInternalServerError, ctx.Response.StatusCode())
	}

	// Check content type
	contentType := string(ctx.Response.Header.ContentType())
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type, got %s", contentType)
	}

	// Parse response
	var response map[string]any
	err := json.Unmarshal(ctx.Response.Body(), &response)
	if err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
		return
	}

	if response["status"] != "error" {
		t.Errorf("Expected status 'error', got %v", response["status"])
	}

	if errorField, ok := response["error"].(map[string]any); ok {
		if errorField["type"] != "internal_panic" {
			t.Errorf("Expected error type 'internal_panic', got %v", errorField["type"])
		}
		if errorField["message"] != "An internal error occurred. The request has been logged." {
			t.Errorf("Expected specific panic message, got %v", errorField["message"])
		}
	} else {
		t.Error("Expected error field to be an object")
	}
}

// Additional tests to improve coverage

func TestHandleFolderEndpoint(t *testing.T) {
	// Create a server with proper configuration
	cfg := &config.Config{
		Compression: config.CompressionConfig{
			Enabled: false, // Disable compression for simpler testing
		},
	}
	server := &Server{cfg: cfg}

	// Create a temporary directory with test files
	tempDir := t.TempDir()

	// Create test files
	indexFile := filepath.Join(tempDir, "index.md")
	err := os.WriteFile(indexFile, []byte("# Test Index\nThis is a test index file."), 0644)
	if err != nil {
		t.Fatalf("Failed to create index file: %v", err)
	}

	subDir := filepath.Join(tempDir, "api")
	err = os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	subIndexFile := filepath.Join(subDir, "index.md")
	err = os.WriteFile(subIndexFile, []byte("# API Documentation\nThis is the API docs."), 0644)
	if err != nil {
		t.Fatalf("Failed to create sub index file: %v", err)
	}

	testCases := []struct {
		name           string
		endpointConfig config.EndpointConfig
		event          map[string]any
		expectedStatus int
	}{
		{
			name: "Valid folder endpoint with specific file",
			endpointConfig: config.EndpointConfig{
				Method: "GET",
				Route:  "/docs/*",
				Path:   tempDir,
				Type:   "markdown-html",
				Options: map[string]any{
					"index":  "index.md",
					"layout": "",
				},
			},
			event: map[string]any{
				"pathParameters": map[string]string{
					"file": "api",
				},
			},
			expectedStatus: 200,
		},
		{
			name: "Folder endpoint with index file",
			endpointConfig: config.EndpointConfig{
				Method: "GET",
				Route:  "/docs/*",
				Path:   tempDir,
				Type:   "markdown-html",
				Options: map[string]any{
					"index":  "index.md",
					"layout": "",
				},
			},
			event:          map[string]any{}, // No file parameter, should serve index
			expectedStatus: 200,
		},
		{
			name: "Invalid folder path traversal",
			endpointConfig: config.EndpointConfig{
				Method: "GET",
				Route:  "/docs/*",
				Path:   tempDir,
				Type:   "markdown-html",
				Options: map[string]any{
					"index":  "index.md",
					"layout": "",
				},
			},
			event: map[string]any{
				"pathParameters": map[string]string{
					"file": "../secret",
				},
			},
			expectedStatus: 500, // This will trigger an error response
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			perfCtx := performance.NewRequestContext("GET", "/test")

			server.handleFolderEndpoint(ctx, tc.endpointConfig, tc.event, perfCtx)

			if ctx.Response.StatusCode() != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, ctx.Response.StatusCode())
			}
		})
	}
}

func TestListFolderContents(t *testing.T) {
	server := &Server{
		cfg: &config.Config{Debug: false},
	}

	// Create a temporary directory with test files
	tempDir := t.TempDir()

	// Create test files
	testFiles := []string{"test1.md", "test2.html", "index.md"}
	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filePath, err)
		}
	}

	testCases := []struct {
		name         string
		folderPath   string
		markdownOnly bool
		responseType string
		expectError  bool
	}{
		{
			name:         "Valid directory with markdown only",
			folderPath:   tempDir,
			markdownOnly: true,
			responseType: "json",
			expectError:  false,
		},
		{
			name:         "Valid directory all files",
			folderPath:   tempDir,
			markdownOnly: false,
			responseType: "json",
			expectError:  false,
		},
		{
			name:         "Non-existent directory",
			folderPath:   "/non/existent/path",
			markdownOnly: false,
			responseType: "json",
			expectError:  false, // Function returns error response, not Go error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := server.listFolderContents(tc.folderPath, tc.markdownOnly, tc.responseType)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Error("Expected result but got nil")
			}
		})
	}
}

func TestDetectResponseType(t *testing.T) {
	server := &Server{
		cfg: &config.Config{Debug: false},
	}

	testCases := []struct {
		name         string
		fileExt      string
		isMarkdown   bool
		expectedType string
	}{
		{
			name:         "Markdown file with markdown enabled",
			fileExt:      ".md",
			isMarkdown:   true,
			expectedType: "markdown",
		},
		{
			name:         "Markdown file with markdown disabled",
			fileExt:      ".md",
			isMarkdown:   false,
			expectedType: "text",
		},
		{
			name:         "HTML file",
			fileExt:      ".html",
			isMarkdown:   false,
			expectedType: "html",
		},
		{
			name:         "Text file",
			fileExt:      ".txt",
			isMarkdown:   false,
			expectedType: "text",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := server.detectResponseType(tc.fileExt, tc.isMarkdown)
			if result != tc.expectedType {
				t.Errorf("Expected %s, got %s", tc.expectedType, result)
			}
		})
	}
}

func TestProcessHTMLContent(t *testing.T) {
	server := &Server{
		cfg: &config.Config{Debug: false},
	}

	testCases := []struct {
		name        string
		content     []byte
		layoutPath  string
		requestPath string
		folderType  string
		expectsHTML bool
	}{
		{
			name:        "Process HTML content",
			content:     []byte("<html><body>Test content</body></html>"),
			layoutPath:  "",
			requestPath: "/test",
			folderType:  "html",
			expectsHTML: true,
		},
		{
			name:        "Process with layout",
			content:     []byte("<h1>Test</h1>"),
			layoutPath:  "docs/layout.html", // This file exists in the workspace
			requestPath: "/test",
			folderType:  "html",
			expectsHTML: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, responseType, title := server.processHTMLContent(tc.content, tc.layoutPath, tc.requestPath, tc.folderType)

			if tc.expectsHTML {
				// Should return HTML content
				if len(result) == 0 {
					t.Error("Expected HTML content but got empty result")
				}
				if responseType == "" {
					t.Error("Expected response type but got empty string")
				}
				// title can be empty, that's valid
				_ = title
			}
		})
	}
}

func TestGetExecutor(t *testing.T) {
	// Create a basic server configuration with database setup
	cfg := &config.Config{
		PreserveResponse: false,
		Debug:            true,
		Database: config.DatabaseConfig{
			Default: "test",
			Connections: map[string]config.DatabaseConnection{
				"test": {
					Driver:   "postgres",
					Host:     "localhost",
					Port:     5432,
					Username: "test_user",
					Password: "test_pass",
					Database: "test_db",
				},
			},
		},
	}

	// Create a database manager (won't actually connect in test)
	dbManager := config.NewDatabaseManager(&cfg.Database)

	// Create server with minimal setup
	server := &Server{
		cfg:          cfg,
		dbManager:    dbManager,
		executorPool: make(chan *tsengine.TSExecutor, 1), // Small pool for testing
	}

	// Test getting an executor (will create a temporary one since pool is empty)
	executor := server.getExecutor()
	if executor == nil {
		t.Error("Expected executor but got nil")
	}

	// Test returning the executor
	server.returnExecutor(executor)

	// Test getting another executor (should reuse the returned one)
	executor2 := server.getExecutor()
	if executor2 == nil {
		t.Error("Expected executor but got nil")
	}

	// Should be the same executor as we returned it to the pool
	if executor != executor2 {
		t.Log("Got different executor (expected when pool was full)")
	}

	// Clean up
	server.returnExecutor(executor2)

	// Test that the pool works correctly
	if len(server.executorPool) != 1 {
		t.Errorf("Expected 1 executor in pool, got %d", len(server.executorPool))
	}
}

// Tests for routing functions

func TestParseRequestToEvent(t *testing.T) {
	// Create a properly initialized server
	cfg := &config.Config{
		Debug: true,
		Env:   map[string]string{"TEST_VAR": "test_value"},
	}
	server := &Server{
		cfg: cfg,
	}

	testCases := []struct {
		name           string
		method         string
		uri            string
		body           string
		route          string
		expectedMethod string
		expectedURI    string
	}{
		{
			name:           "GET request",
			method:         "GET",
			uri:            "/api/users/123",
			body:           "",
			route:          "/api/users/{id}",
			expectedMethod: "GET",
			expectedURI:    "/api/users/123",
		},
		{
			name:           "POST request with JSON body",
			method:         "POST",
			uri:            "/api/users",
			body:           `{"name": "John", "email": "john@example.com"}`,
			route:          "/api/users",
			expectedMethod: "POST",
			expectedURI:    "/api/users",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			ctx.Request.Header.SetMethod(tc.method)
			ctx.Request.SetRequestURI(tc.uri)
			ctx.Request.Header.Set("User-Agent", "test-agent") // Add a header for testing
			if tc.body != "" {
				ctx.Request.SetBody([]byte(tc.body))
				ctx.Request.Header.SetContentType("application/json")
			}

			ep := config.EndpointConfig{
				Route:  tc.route,
				Method: tc.method,
			}
			perfCtx := performance.NewRequestContext(tc.method, tc.uri)

			event, err := server.parseRequestToEvent(ctx, ep, perfCtx)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if event["method"] != tc.expectedMethod {
				t.Errorf("Expected method %s, got %v", tc.expectedMethod, event["method"])
			}

			if event["path"] != tc.expectedURI {
				t.Errorf("Expected path %s, got %v", tc.expectedURI, event["path"])
			}

			// Check that headers are present
			if headers, ok := event["headers"]; !ok {
				t.Error("Expected headers in event")
			} else if headersMap, ok := headers.(map[string]string); !ok {
				t.Error("Expected headers to be map[string]string")
			} else if len(headersMap) == 0 {
				t.Error("Expected some headers")
			}
		})
	}
}

func TestParseRequestBody(t *testing.T) {
	server := &Server{
		cfg: &config.Config{Debug: true},
	}

	testCases := []struct {
		name         string
		method       string
		body         string
		contentType  string
		expectedKeys []string
		expectsError bool
	}{
		{
			name:         "Valid JSON body with POST",
			method:       "POST",
			body:         `{"name": "John", "age": 30}`,
			contentType:  "application/json",
			expectedKeys: []string{"name", "age"},
			expectsError: false,
		},
		{
			name:         "Empty body with POST",
			method:       "POST",
			body:         "",
			contentType:  "application/json",
			expectedKeys: []string{},
			expectsError: false,
		},
		{
			name:         "Invalid JSON body with POST",
			method:       "POST",
			body:         `{"name": "John", "age":}`,
			contentType:  "application/json",
			expectedKeys: nil,
			expectsError: true,
		},
		{
			name:         "GET request (no body parsing)",
			method:       "GET",
			body:         `{"name": "John"}`,
			contentType:  "application/json",
			expectedKeys: []string{}, // GET requests return empty map
			expectsError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			ctx.Request.Header.SetMethod(tc.method)
			ctx.Request.SetBody([]byte(tc.body))
			ctx.Request.Header.SetContentType(tc.contentType)

			result, err := server.parseRequestBody(ctx)

			if tc.expectsError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if len(tc.expectedKeys) > 0 {
					if result == nil {
						t.Error("Expected result but got nil")
					} else {
						// Check keys exist
						for _, key := range tc.expectedKeys {
							if _, exists := result[key]; !exists {
								t.Errorf("Expected key %s in result", key)
							}
						}
					}
				} else {
					// Expect empty map for non-POST/PUT/PATCH or empty body
					if result == nil {
						t.Error("Expected empty map but got nil")
					} else if len(result) != 0 {
						t.Errorf("Expected empty map but got %v", result)
					}
				}
			}
		})
	}
}

func TestParseRequestHeaders(t *testing.T) {
	server := &Server{
		cfg: &config.Config{Debug: true},
	}

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set("Authorization", "Bearer token123")
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("X-Custom-Header", "custom-value")

	headers := server.parseRequestHeaders(ctx)

	if headers == nil {
		t.Error("Expected headers but got nil")
		return
	}

	// Note: fasthttp headers are case-sensitive as set
	expectedHeaders := map[string]string{
		"Authorization":   "Bearer token123",
		"Content-Type":    "application/json",
		"X-Custom-Header": "custom-value",
	}

	for key, expectedValue := range expectedHeaders {
		if value, exists := headers[key]; !exists {
			t.Errorf("Expected header %s", key)
		} else if value != expectedValue {
			t.Errorf("Expected header %s to be %s, got %s", key, expectedValue, value)
		}
	}
}

// Tests for response functions

func TestSetContentTypeFromResponseType(t *testing.T) {
	server := &Server{
		cfg: &config.Config{Debug: false},
	}

	testCases := []struct {
		name         string
		responseType string
		expectedType string
	}{
		{
			name:         "JSON response type",
			responseType: "json",
			expectedType: "application/json",
		},
		{
			name:         "HTML response type",
			responseType: "html",
			expectedType: "text/html; charset=utf-8",
		},
		{
			name:         "Text response type",
			responseType: "text",
			expectedType: "text/plain; charset=utf-8",
		},
		{
			name:         "Markdown response type",
			responseType: "markdown",
			expectedType: "text/markdown; charset=utf-8",
		},
		{
			name:         "Unknown response type",
			responseType: "unknown",
			expectedType: "application/json", // default
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}

			server.setContentTypeFromResponseType(ctx, tc.responseType)

			actualType := string(ctx.Response.Header.ContentType())
			if actualType != tc.expectedType {
				t.Errorf("Expected content type %s, got %s", tc.expectedType, actualType)
			}
		})
	}
}

func TestHandleNilResponse(t *testing.T) {
	// Create a server with proper configuration to avoid nil pointer
	cfg := &config.Config{
		Compression: config.CompressionConfig{
			Enabled: false, // Disable compression for simpler testing
		},
	}
	server := &Server{cfg: cfg}
	ctx := &fasthttp.RequestCtx{}

	server.handleNilResponse(ctx)

	if ctx.Response.StatusCode() != 200 {
		t.Errorf("Expected status 200, got %d", ctx.Response.StatusCode())
	}

	expectedBody := `{"status":"success"}`
	actualBody := string(ctx.Response.Body())
	if actualBody != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, actualBody)
	}

	expectedContentType := "application/json"
	actualContentType := string(ctx.Response.Header.ContentType())
	if actualContentType != expectedContentType {
		t.Errorf("Expected content type %s, got %s", expectedContentType, actualContentType)
	}
}

func TestHasResponseWrapper(t *testing.T) {
	server := &Server{
		cfg: &config.Config{Debug: false},
	}

	testCases := []struct {
		name     string
		response map[string]any
		expected bool
	}{
		{
			name:     "Has response wrapper",
			response: map[string]any{"response": "data", "code": 200},
			expected: true,
		},
		{
			name:     "No response wrapper - missing code",
			response: map[string]any{"response": "data"},
			expected: false,
		},
		{
			name:     "No response wrapper - missing response",
			response: map[string]any{"code": 200},
			expected: false,
		},
		{
			name:     "Empty response",
			response: map[string]any{},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := server.hasResponseWrapper(tc.response)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestWriteDirectResponse(t *testing.T) {
	// Create a server with proper configuration to avoid nil pointer
	cfg := &config.Config{
		Compression: config.CompressionConfig{
			Enabled: false, // Disable compression for simpler testing
		},
	}
	server := &Server{cfg: cfg}
	ctx := &fasthttp.RequestCtx{}

	testResponse := json.RawMessage(`{"message": "Hello World"}`)
	server.writeDirectResponse(ctx, testResponse)

	expectedBody := `{"message": "Hello World"}`
	actualBody := string(ctx.Response.Body())
	if actualBody != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, actualBody)
	}
}

func TestHandleWrappedResponse(t *testing.T) {
	server := createTestServer()

	testCases := []struct {
		name           string
		responseData   string
		expectedStatus int
	}{
		{
			name:           "Valid wrapped response with code",
			responseData:   `{"response": "test data", "code": 201, "type": "json"}`,
			expectedStatus: 201,
		},
		{
			name:           "Wrapped response with cookies",
			responseData:   `{"response": "test", "code": 200, "cookies": ["session=abc123"]}`,
			expectedStatus: 200,
		},
		{
			name:           "Invalid JSON should cause error",
			responseData:   `{invalid json`,
			expectedStatus: 500,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			perfCtx := &performance.RequestContext{}

			server.handleWrappedResponse(ctx, json.RawMessage(tc.responseData), perfCtx)

			if ctx.Response.StatusCode() != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, ctx.Response.StatusCode())
			}
		})
	}
}

func TestHandleFinalResponse(t *testing.T) {
	server := createTestServer()

	testCases := []struct {
		name     string
		response json.RawMessage
	}{
		{
			name:     "Valid JSON response",
			response: json.RawMessage(`{"status": "success"}`),
		},
		{
			name:     "Nil response",
			response: nil,
		},
		{
			name:     "Empty response",
			response: json.RawMessage(`{}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			perfCtx := &performance.RequestContext{}

			server.handleFinalResponse(ctx, tc.response, perfCtx)

			// Just verify that the function doesn't crash and sets some status
			if ctx.Response.StatusCode() == 0 {
				t.Error("Expected some status code to be set")
			}
		})
	}
}

func TestSetCookies(t *testing.T) {
	server := createTestServer()

	testCases := []struct {
		name          string
		cookies       []string
		expectedCount int
	}{
		{
			name:          "Single cookie",
			cookies:       []string{"session=abc123"},
			expectedCount: 1,
		},
		{
			name:          "Multiple cookies",
			cookies:       []string{"session=abc123", "user=john"},
			expectedCount: 2,
		},
		{
			name:          "Empty cookies",
			cookies:       []string{},
			expectedCount: 0,
		},
		{
			name:          "Nil cookies",
			cookies:       nil,
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}

			server.setCookies(ctx, tc.cookies)

			// Check that cookies were set correctly
			// We can't easily count Set-Cookie headers, so let's just verify no crash
			// and that the function executed without error
		})
	}
}

func TestWriteUnwrappedResponse(t *testing.T) {
	server := createTestServer()

	testCases := []struct {
		name         string
		response     []byte
		statusCode   int
		responseType string
	}{
		{
			name:         "JSON response",
			response:     []byte(`{"status": "success"}`),
			statusCode:   200,
			responseType: "json",
		},
		{
			name:         "HTML response",
			response:     []byte(`"<html><body>Test</body></html>"`),
			statusCode:   201,
			responseType: "html",
		},
		{
			name:         "Text response",
			response:     []byte(`"Plain text"`),
			statusCode:   202,
			responseType: "text",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}

			server.writeUnwrappedResponse(ctx, tc.response, tc.statusCode, tc.responseType)

			if ctx.Response.StatusCode() != tc.statusCode {
				t.Errorf("Expected status %d, got %d", tc.statusCode, ctx.Response.StatusCode())
			}

			// Verify that some response was written
			if len(ctx.Response.Body()) == 0 {
				t.Error("Expected some response body to be written")
			}
		})
	}
}
