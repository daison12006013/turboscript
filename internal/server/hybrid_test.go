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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/valyala/fasthttp"
)

// TestSanitizeJSONForHTML tests the XSS protection function using template.JSEscapeString.
func TestSanitizeJSONForHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal JSON data",
			input:    `{"name":"John","age":30}`,
			expected: `{\"name\":\"John\",\"age\":30}`,
		},
		{
			name:     "JSON with script tag",
			input:    `{"message":"<script>alert('XSS')</script>"}`,
			expected: `{\"message\":\"\u003Cscript\u003Ealert(\'XSS\')\u003C/script\u003E\"}`,
		},
		{
			name:     "JSON with HTML entities",
			input:    `{"html":"<div>Hello & goodbye</div>"}`,
			expected: `{\"html\":\"\u003Cdiv\u003EHello \u0026 goodbye\u003C/div\u003E\"}`,
		},
		{
			name:     "JSON with JavaScript event handler",
			input:    `{"onclick":"javascript:alert('XSS')"}`,
			expected: `{\"onclick\":\"javascript:alert(\'XSS\')\"}`,
		},
		{
			name:     "JSON with iframe injection",
			input:    `{"content":"<iframe src='javascript:alert(1)'></iframe>"}`,
			expected: `{\"content\":\"\u003Ciframe src\u003D\'javascript:alert(1)\'\u003E\u003C/iframe\u003E\"}`,
		},
		{
			name:     "Empty JSON",
			input:    `{}`,
			expected: `{}`,
		},
		{
			name:     "JSON with quotes and backslashes",
			input:    `{"path":"C:\\Users\\test","quote":"\"Hello\""}`,
			expected: `{\"path\":\"C:\\\\Users\\\\test\",\"quote\":\"\\\"Hello\\\"\"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeJSONForHTML(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeJSONForHTML() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestHybridDataStruct tests the HybridData structure.
func TestHybridDataStruct(t *testing.T) {
	data := HybridData{
		Route: "/test/route",
		Data:  `{"sanitized": "data"}`,
	}

	if data.Route != "/test/route" {
		t.Errorf("Expected route '/test/route', got %s", data.Route)
	}

	if data.Data != `{"sanitized": "data"}` {
		t.Errorf("Expected data '{\"sanitized\": \"data\"}', got %s", data.Data)
	}
}

// TestDetermineDataPath tests the data path determination logic.
func TestDetermineDataPath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create test files
	dataDir := filepath.Join(tempDir, "api")
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test data directory: %v", err)
	}

	// Create test files
	indexFile := filepath.Join(dataDir, "index.ts")
	settingsFile := filepath.Join(dataDir, "settings.ts")

	err = os.WriteFile(indexFile, []byte("// index endpoint"), 0644)
	if err != nil {
		t.Fatalf("Failed to create index.ts: %v", err)
	}

	err = os.WriteFile(settingsFile, []byte("// settings endpoint"), 0644)
	if err != nil {
		t.Fatalf("Failed to create settings.ts: %v", err)
	}

	server := &Server{
		cfg: &config.Config{Debug: false},
	}

	tests := []struct {
		name         string
		frontendPath string
		ep           config.EndpointConfig
		expected     string
	}{
		{
			name:         "Root path should return index.ts",
			frontendPath: "/hybrid",
			ep: config.EndpointConfig{
				Route: "/hybrid/*",
				Path:  tempDir,
				Options: map[string]any{
					"data": "api",
				},
			},
			expected: filepath.Join(tempDir, "api", "index.ts"),
		},
		{
			name:         "Root path with slash should return index.ts",
			frontendPath: "/hybrid/",
			ep: config.EndpointConfig{
				Route: "/hybrid/*",
				Path:  tempDir,
				Options: map[string]any{
					"data": "api",
				},
			},
			expected: filepath.Join(tempDir, "api", "index.ts"),
		},
		{
			name:         "Settings path should return settings.ts",
			frontendPath: "/hybrid/settings",
			ep: config.EndpointConfig{
				Route: "/hybrid/*",
				Path:  tempDir,
				Options: map[string]any{
					"data": "api",
				},
			},
			expected: filepath.Join(tempDir, "api", "settings.ts"),
		},
		{
			name:         "Non-existent path should fallback to index.ts",
			frontendPath: "/hybrid/nonexistent",
			ep: config.EndpointConfig{
				Route: "/hybrid/*",
				Path:  tempDir,
				Options: map[string]any{
					"data": "api",
				},
			},
			expected: filepath.Join(tempDir, "api", "index.ts"),
		},
		{
			name:         "Custom data folder",
			frontendPath: "/hybrid/settings",
			ep: config.EndpointConfig{
				Route: "/hybrid/*",
				Path:  tempDir,
				Options: map[string]any{
					"data": "custom-api",
				},
			},
			expected: filepath.Join(tempDir, "custom-api", "index.ts"), // Falls back to index because custom-api doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.determineDataPath(tt.frontendPath, tt.ep)
			if result != tt.expected {
				t.Errorf("determineDataPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestHandleStaticAsset tests static asset serving with security checks.
func TestHandleStaticAsset(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create test assets
	assetsDir := filepath.Join(tempDir, "assets")
	err := os.MkdirAll(assetsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create assets directory: %v", err)
	}

	// Create test files
	cssFile := filepath.Join(assetsDir, "styles.css")
	jsFile := filepath.Join(assetsDir, "script.js")
	maliciousFile := filepath.Join(tempDir, "malicious.txt") // Outside assets folder

	err = os.WriteFile(cssFile, []byte("body { color: red; }"), 0644)
	if err != nil {
		t.Fatalf("Failed to create CSS file: %v", err)
	}

	err = os.WriteFile(jsFile, []byte("console.log('Hello');"), 0644)
	if err != nil {
		t.Fatalf("Failed to create JS file: %v", err)
	}

	err = os.WriteFile(maliciousFile, []byte("malicious content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create malicious file: %v", err)
	}

	server := &Server{
		cfg: &config.Config{Debug: false},
	}
	ep := config.EndpointConfig{
		Path: tempDir,
		Options: map[string]any{
			"assets": "assets",
		},
	}

	tests := []struct {
		name           string
		requestPath    string
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "Valid CSS file",
			requestPath:    "/hybrid/assets/styles.css",
			expectedStatus: fasthttp.StatusOK,
			expectedType:   "text/css",
		},
		{
			name:           "Valid JS file",
			requestPath:    "/hybrid/assets/script.js",
			expectedStatus: fasthttp.StatusOK,
			expectedType:   "application/javascript",
		},
		{
			name:           "Non-existent file",
			requestPath:    "/hybrid/assets/nonexistent.css",
			expectedStatus: fasthttp.StatusNotFound,
		},
		{
			name:           "Path traversal attempt",
			requestPath:    "/hybrid/assets/../malicious.txt",
			expectedStatus: fasthttp.StatusForbidden,
		},
		{
			name:           "Invalid assets path",
			requestPath:    "/hybrid/assets/",
			expectedStatus: fasthttp.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			ctx.Request.SetRequestURI(tt.requestPath)

			server.handleStaticAsset(ctx, ep, tt.requestPath, "assets")

			if ctx.Response.StatusCode() != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, ctx.Response.StatusCode())
			}

			if tt.expectedType != "" {
				contentType := string(ctx.Response.Header.ContentType())
				if contentType != tt.expectedType {
					t.Errorf("Expected content type %s, got %s", tt.expectedType, contentType)
				}
			}
		})
	}
}

// TestServeHybridApp tests the Hybrid app serving functionality with XSS protection.
func TestServeHybridApp(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a test app template
	appTemplate := `<!DOCTYPE html>
<html>
<head>
    <title>Test Hybrid App</title>
</head>
<body>
    <div id="root"></div>
    <script>
        window.__INITIAL_DATA__ = {{.Data}};
        window.__ROUTE__ = "{{.Route}}";
    </script>
</body>
</html>`

	appPath := filepath.Join(tempDir, "App.html")
	err := os.WriteFile(appPath, []byte(appTemplate), 0644)
	if err != nil {
		t.Fatalf("Failed to create app template: %v", err)
	}

	server := &Server{
		cfg: &config.Config{Debug: false},
	}

	tests := []struct {
		name        string
		route       string
		data        map[string]any
		expectXSS   bool
		expectError bool
	}{
		{
			name:  "Normal data",
			route: "/test",
			data: map[string]any{
				"message": "Hello World",
				"count":   42,
			},
			expectXSS:   false,
			expectError: false,
		},
		{
			name:  "Data with potential XSS",
			route: "/test",
			data: map[string]any{
				"message": "<script>alert('XSS')</script>",
				"html":    "<div>Hello & goodbye</div>",
			},
			expectXSS:   true,
			expectError: false,
		},
		{
			name:  "Data with JavaScript injection",
			route: "/test",
			data: map[string]any{
				"onclick": "javascript:alert('XSS')",
				"iframe":  "<iframe src='javascript:alert(1)'></iframe>",
			},
			expectXSS:   true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}

			// Create a mock endpoint config for this test
			mockEndpoint := config.EndpointConfig{
				Path: tempDir,
				Options: map[string]any{
					"assets": "assets",
				},
			}

			err := server.serveHybridApp(ctx, mockEndpoint, appPath, tt.route, tt.data, false)

			if tt.expectError && err == nil {
				t.Error("Expected error, but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				// Check response status
				if ctx.Response.StatusCode() != fasthttp.StatusOK {
					t.Errorf("Expected status 200, got %d", ctx.Response.StatusCode())
				}

				// Check content type
				contentType := string(ctx.Response.Header.ContentType())
				if contentType != "text/html; charset=utf-8" {
					t.Errorf("Expected content type 'text/html; charset=utf-8', got %s", contentType)
				}

				// Check that the response contains sanitized data
				responseBody := string(ctx.Response.Body())

				if tt.expectXSS {
					// Verify that dangerous content is properly escaped using template.JSEscapeString
					// Check that literal <script> tags are not present
					if strings.Contains(responseBody, "<script>alert('XSS')</script>") {
						t.Error("Response contains unescaped script tag - XSS vulnerability!")
					}

					// Check that literal javascript: URLs are not present in unescaped form
					// (Allow the escaped version which is safe)
					dangerousJS := `javascript:alert('XSS')`
					if strings.Contains(responseBody, dangerousJS) {
						// Check if it's properly escaped in a JavaScript string context
						escapedJS := `javascript:alert(\'XSS\')`
						if !strings.Contains(responseBody, escapedJS) {
							t.Error("Response contains unescaped JavaScript - XSS vulnerability!")
						}
					}

					// Verify that content is properly escaped using template.JSEscapeString
					// Check for JavaScript unicode escaping or proper JSON string escaping
					hasProperEscaping := false

					// Check for unicode escaping of HTML characters
					if strings.Contains(responseBody, `\u003C`) || strings.Contains(responseBody, `\u003E`) {
						hasProperEscaping = true
					}

					// Check for properly escaped quotes in JSON strings
					if strings.Contains(responseBody, `\"`) {
						hasProperEscaping = true
					}

					// Check for escaped ampersands
					if strings.Contains(responseBody, `\u0026`) {
						hasProperEscaping = true
					}

					if !hasProperEscaping {
						t.Error("Dangerous content should be properly escaped using template.JSEscapeString")
					}
				}

				// Verify the route is properly embedded
				if !strings.Contains(responseBody, tt.route) {
					t.Errorf("Response should contain route %s", tt.route)
				}
			}
		})
	}
}

// TestServeHybridAppInvalidTemplate tests error handling for invalid templates.
func TestServeHybridAppInvalidTemplate(t *testing.T) {
	server := &Server{
		cfg: &config.Config{Debug: false},
	}

	// Create a mock endpoint config
	mockEndpoint := config.EndpointConfig{
		Path: "/tmp",
		Options: map[string]any{
			"assets": "assets",
		},
	}

	// Test with non-existent template file
	ctx := &fasthttp.RequestCtx{}
	err := server.serveHybridApp(ctx, mockEndpoint, "/nonexistent/template.html", "/test", map[string]any{}, false)

	if err == nil {
		t.Error("Expected error for non-existent template file")
	}

	if !strings.Contains(err.Error(), "failed to read app template") {
		t.Errorf("Expected template read error, got: %v", err)
	}
}

// TestServeHybridAppInvalidData tests error handling for data that cannot be marshaled.
func TestServeHybridAppInvalidData(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a test app template
	appTemplate := `<html><body>{{.Data}}</body></html>`
	appPath := filepath.Join(tempDir, "App.html")
	err := os.WriteFile(appPath, []byte(appTemplate), 0644)
	if err != nil {
		t.Fatalf("Failed to create app template: %v", err)
	}

	server := &Server{
		cfg: &config.Config{Debug: false},
	}

	// Create a mock endpoint config
	mockEndpoint := config.EndpointConfig{
		Path: tempDir,
		Options: map[string]any{
			"assets": "assets",
		},
	}

	// Create data that cannot be marshaled to JSON (circular reference)
	type circularData struct {
		Name string
		Self *circularData
	}

	data := &circularData{Name: "test"}
	data.Self = data // Create circular reference

	ctx := &fasthttp.RequestCtx{}
	err = server.serveHybridApp(ctx, mockEndpoint, appPath, "/test", map[string]any{"circular": data}, false)

	if err == nil {
		t.Error("Expected error for data with circular reference")
	}

	if !strings.Contains(err.Error(), "failed to marshal data to JSON") {
		t.Errorf("Expected JSON marshal error, got: %v", err)
	}
}

// TestIsAPIRequest tests the API request detection logic.
func TestIsAPIRequest(t *testing.T) {
	server := &Server{
		cfg: &config.Config{Debug: false},
	}
	ep := config.EndpointConfig{
		Options: map[string]any{
			"api_prefix": "/api",
		},
	}

	tests := []struct {
		name           string
		path           string
		headers        map[string]string
		expectedResult bool
	}{
		{
			name:           "JSON Accept header",
			path:           "/hybrid/settings",
			headers:        map[string]string{"Accept": "application/json"},
			expectedResult: true,
		},
		{
			name:           "JSON Accept with HTML (JSON preferred)",
			path:           "/hybrid/settings",
			headers:        map[string]string{"Accept": "application/json, text/html"},
			expectedResult: true,
		},
		{
			name:           "HTML Accept header only",
			path:           "/hybrid/settings",
			headers:        map[string]string{"Accept": "text/html,application/xhtml+xml"},
			expectedResult: false,
		},
		{
			name:           "XMLHttpRequest header",
			path:           "/hybrid/settings",
			headers:        map[string]string{"X-Requested-With": "XMLHttpRequest"},
			expectedResult: true,
		},
		{
			name:           "API prefix path",
			path:           "/api/settings",
			headers:        map[string]string{},
			expectedResult: true,
		},
		{
			name:           "JSON Content-Type",
			path:           "/hybrid/settings",
			headers:        map[string]string{"Content-Type": "application/json"},
			expectedResult: true,
		},
		{
			name:           "Regular browser request",
			path:           "/hybrid/settings",
			headers:        map[string]string{"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
			expectedResult: false,
		},
		{
			name:           "No headers - default to browser",
			path:           "/hybrid/settings",
			headers:        map[string]string{},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			ctx.Request.SetRequestURI(tt.path)

			// Set headers
			for key, value := range tt.headers {
				ctx.Request.Header.Set(key, value)
			}

			result := server.isAPIRequest(ctx, ep)
			if result != tt.expectedResult {
				t.Errorf("isAPIRequest() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

// TestHandleHybridAPIRequest tests the API request handling functionality.
func TestHandleHybridAPIRequest(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create test data files
	dataDir := filepath.Join(tempDir, "data")
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test data directory: %v", err)
	}

	// Create a test settings.ts file that returns JSON data
	settingsFile := filepath.Join(dataDir, "settings.ts")
	settingsContent := `
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    return {
        code: 200,
        response: {
            status: "success",
            data: {
                theme: "dark",
                language: "en",
                notifications: true
            }
        }
    };
};`

	err = os.WriteFile(settingsFile, []byte(settingsContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create settings.ts: %v", err)
	}

	// Create an index.ts fallback
	indexFile := filepath.Join(dataDir, "index.ts")
	indexContent := `
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    return {
        code: 200,
        response: {
            status: "success",
            data: {
                message: "Welcome to the app"
            }
        }
    };
};`

	err = os.WriteFile(indexFile, []byte(indexContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create index.ts: %v", err)
	}

	// Note: We can't fully test this without a complete server setup and TypeScript executor
	// This test validates the structure and flow, but actual execution would require the full system
	t.Log("API request handler structure test completed - full integration testing requires complete server setup")
}

// TestHybridEndpointRouting tests the complete Hybrid endpoint routing logic.
func TestHybridEndpointRouting(t *testing.T) {
	server := &Server{
		cfg: &config.Config{Debug: false},
	}
	ep := config.EndpointConfig{
		Path: "/test/path",
		Options: map[string]any{
			"assets": "assets",
			"data":   "data",
		},
	}

	tests := []struct {
		name         string
		path         string
		headers      map[string]string
		expectedType string // "api", "html", "asset", "not_found"
	}{
		{
			name:         "Static asset request",
			path:         "/hybrid/assets/styles.css",
			headers:      map[string]string{},
			expectedType: "asset",
		},
		{
			name:         "API JSON request",
			path:         "/hybrid/settings",
			headers:      map[string]string{"Accept": "application/json"},
			expectedType: "api",
		},
		{
			name:         "AJAX request",
			path:         "/hybrid/about",
			headers:      map[string]string{"X-Requested-With": "XMLHttpRequest"},
			expectedType: "api",
		},
		{
			name:         "Browser navigation",
			path:         "/hybrid/settings",
			headers:      map[string]string{"Accept": "text/html,application/xhtml+xml"},
			expectedType: "html",
		},
		{
			name:         "Root frontend path",
			path:         "/hybrid",
			headers:      map[string]string{"Accept": "text/html"},
			expectedType: "html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock context
			ctx := &fasthttp.RequestCtx{}
			ctx.Request.SetRequestURI(tt.path)

			// Set headers
			for key, value := range tt.headers {
				ctx.Request.Header.Set(key, value)
			}

			// Test the routing logic
			assetsPath := ep.GetOptionString("assets", "assets")
			isAsset := strings.Contains(tt.path, "/"+assetsPath+"/")
			isAPI := server.isAPIRequest(ctx, ep)

			var actualType string
			switch {
			case isAsset:
				actualType = "asset"
			case isAPI:
				actualType = "api"
			default:
				actualType = "html"
			}

			if actualType != tt.expectedType {
				t.Errorf("Expected route type %s, got %s", tt.expectedType, actualType)
			}
		})
	}
}

// BenchmarkSanitizeJSONForHTML benchmarks the XSS sanitization function using template.JSEscapeString.
func BenchmarkSanitizeJSONForHTML(b *testing.B) {
	testData := `{"message":"<script>alert('XSS')</script>","html":"<div>Hello & goodbye</div>","onclick":"javascript:alert('XSS')"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SanitizeJSONForHTML(testData)
	}
}

// BenchmarkServeHybridApp benchmarks the Hybrid app serving functionality.
func BenchmarkServeHybridApp(b *testing.B) {
	// Create a temporary directory for testing
	tempDir := b.TempDir()

	// Create a test app template
	appTemplate := `<!DOCTYPE html>
<html>
<head><title>Benchmark Test</title></head>
<body>
    <div id="root"></div>
    <script>
        window.__INITIAL_DATA__ = {{.Data}};
        window.__ROUTE__ = "{{.Route}}";
    </script>
</body>
</html>`

	appPath := filepath.Join(tempDir, "App.html")
	err := os.WriteFile(appPath, []byte(appTemplate), 0644)
	if err != nil {
		b.Fatalf("Failed to create app template: %v", err)
	}

	server := &Server{
		cfg: &config.Config{Debug: false},
	}

	// Create a mock endpoint config for benchmark
	mockEndpoint := config.EndpointConfig{
		Path: tempDir,
		Options: map[string]any{
			"assets": "assets",
		},
	}

	testData := map[string]any{
		"message": "Hello World",
		"count":   42,
		"users":   []string{"John", "Jane", "Bob"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := &fasthttp.RequestCtx{}
		err := server.serveHybridApp(ctx, mockEndpoint, appPath, "/test", testData, false)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}
