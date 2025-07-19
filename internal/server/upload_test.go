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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/valyala/fasthttp"
)

// Test configuration for file uploads using /tmp directory
func getTestUploadConfig() *config.Config {
	return &config.Config{
		Port:     7891, // Use different port for testing
		Debug:    true,
		Info:     true,
		Warning:  true,
		PreferTS: true,
		Endpoints: []config.EndpointConfig{
			{
				Route:  "/test/upload",
				Method: "GET",
				Path:   "./app/routes/demo/upload",
			},
			{
				Route:  "/test/upload",
				Method: "POST",
				Path:   "./app/routes/demo/upload",
			},
		},
		Plugins: []config.PluginConfig{
			{
				Name:    "fileupload",
				Enabled: true,
				Options: map[string]any{
					"storage_type":   "local",
					"upload_dir":     "/tmp/turboscript-test-uploads", // Use /tmp for tests
					"max_file_size":  10485760,                        // 10MB
					"allowed_types": []string{
						"image/jpeg", "image/png", "image/gif", "image/webp",
						"application/pdf", "text/plain", "application/octet-stream",
					},
				},
			},
		},
	}
}

// Helper function to create test files
func createTestFile(t *testing.T, content string, filename string) string {
	tmpDir := "/tmp/turboscript-test-files"
	err := os.MkdirAll(tmpDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	filePath := filepath.Join(tmpDir, filename)
	err = os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	return filePath
}

// Helper function to create binary test image (1x1 PNG)
func createTestImage(t *testing.T) string {
	// 1x1 pixel PNG image data (base64 decoded)
	pngData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChAFPp1zHqwAAAABJRU5ErkJggg=="
	imageBytes, err := base64.StdEncoding.DecodeString(pngData)
	if err != nil {
		t.Fatalf("Failed to decode test image: %v", err)
	}

	tmpDir := "/tmp/turboscript-test-files"
	err = os.MkdirAll(tmpDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	filePath := filepath.Join(tmpDir, "test-image.png")
	err = os.WriteFile(filePath, imageBytes, 0644)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	return filePath
}

// Cleanup test files and directories
func cleanupTestFiles(t *testing.T) {
	dirs := []string{
		"/tmp/turboscript-test-files",
		"/tmp/turboscript-test-uploads",
	}

	for _, dir := range dirs {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Logf("Warning: Failed to cleanup test directory %s: %v", dir, err)
		}
	}
}

// Test server creation with upload configuration
func createTestServerWithUpload(t *testing.T) *Server {
	cfg := getTestUploadConfig()

	// Create server without full services for testing
	server := NewServerWithServices(cfg, nil, nil, nil)

	// Initialize path patterns for parameter extraction
	server.pathPatterns = make(map[string]*regexp.Regexp)
	server.pathParams = make(map[string][]string)

	return server
}

// Test parsing of multipart/form-data request body
func TestParseMultipartBody(t *testing.T) {
	t.Skip("skipped by user request")
	defer cleanupTestFiles(t)

	server := createTestServerWithUpload(t)

	// Create test file content
	testContent := "Hello, this is a test file for multipart upload!"
	testImagePath := createTestImage(t)
	imageData, err := os.ReadFile(testImagePath)
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add text file
	textWriter, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	_, err = textWriter.Write([]byte(testContent))
	if err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}

	// Add PNG image file
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="image"; filename="test.png"`)
	h.Set("Content-Type", "image/png")
	imageWriter, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("Failed to create image part: %v", err)
	}
	_, err = imageWriter.Write(imageData)
	if err != nil {
		t.Fatalf("Failed to write image content: %v", err)
	}

	// Add form fields
	err = writer.WriteField("directory", "test-multipart")
	if err != nil {
		t.Fatalf("Failed to write directory field: %v", err)
	}
	err = writer.WriteField("generateHash", "true")
	if err != nil {
		t.Fatalf("Failed to write generateHash field: %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close multipart writer: %v", err)
	}

	// Create fasthttp request context
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetBody(buf.Bytes())
	ctx.Request.Header.SetContentType(fmt.Sprintf("multipart/form-data; boundary=%s", writer.Boundary()))
	ctx.Request.Header.SetMethod("POST")

	// Test parsing
	result, err := server.parseMultipartBody(ctx)
	if err != nil {
		t.Fatalf("Failed to parse multipart body: %v", err)
	}

	// Verify results
	files, ok := result["files"].([]map[string]any)
	if !ok {
		t.Fatalf("Expected files array, got %T", result["files"])
	}

	if len(files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(files))
	}

	// Check text file
	textFile := files[0]
	if textFile["filename"] != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got %v", textFile["filename"])
	}
	if textFile["size"] != len(testContent) {
		t.Errorf("Expected size %d, got %v", len(testContent), textFile["size"])
	}

	// Check image file
	imageFile := files[1]
	if imageFile["filename"] != "test.png" {
		t.Errorf("Expected filename 'test.png', got %v", imageFile["filename"])
	}
	if imageFile["mimeType"] != "image/png" {
		t.Errorf("Expected mimeType 'image/png', got %v", imageFile["mimeType"])
	}
	if imageFile["size"] != len(imageData) {
		t.Errorf("Expected size %d, got %v", len(imageData), imageFile["size"])
	}

	// Check fields
	fields, ok := result["fields"].(map[string]string)
	if !ok {
		t.Fatalf("Expected fields map, got %T", result["fields"])
	}

	if fields["directory"] != "test-multipart" {
		t.Errorf("Expected directory 'test-multipart', got %v", fields["directory"])
	}
	if fields["generateHash"] != "true" {
		t.Errorf("Expected generateHash 'true', got %v", fields["generateHash"])
	}

	t.Logf("Successfully parsed multipart data with %d files and %d fields", len(files), len(fields))
}

// Test parsing of raw binary request body
func TestParseRawBinaryBody(t *testing.T) {
	t.Skip("skipped by user request")
	defer cleanupTestFiles(t)

	server := createTestServerWithUpload(t)

	// Create test image
	testImagePath := createTestImage(t)
	imageData, err := os.ReadFile(testImagePath)
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	// Create fasthttp request context with binary data
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetBody(imageData)
	ctx.Request.Header.SetContentType("image/png")
	ctx.Request.Header.SetMethod("POST")

	// Test parsing
	result, err := server.parseRawBinaryBody(ctx, "image/png")
	if err != nil {
		t.Fatalf("Failed to parse raw binary body: %v", err)
	}

	// Verify results
	binaryData, ok := result["binaryData"].(string)
	if !ok {
		t.Fatalf("Expected binaryData string, got %T", result["binaryData"])
	}

	contentType, ok := result["contentType"].(string)
	if !ok {
		t.Fatalf("Expected contentType string, got %T", result["contentType"])
	}

	size, ok := result["size"].(int)
	if !ok {
		t.Fatalf("Expected size int, got %T", result["size"])
	}

	// Verify values
	if contentType != "image/png" {
		t.Errorf("Expected contentType 'image/png', got %s", contentType)
	}

	if size != len(imageData) {
		t.Errorf("Expected size %d, got %d", len(imageData), size)
	}

	// Verify binary data is base64 encoded
	decodedData, err := base64.StdEncoding.DecodeString(binaryData)
	if err != nil {
		t.Fatalf("Failed to decode base64 data: %v", err)
	}

	if !bytes.Equal(decodedData, imageData) {
		t.Errorf("Decoded binary data doesn't match original image data")
	}

	t.Logf("Successfully parsed raw binary data: %d bytes, content-type: %s", size, contentType)
}

// Test parsing of JSON request body (base64 upload)
func TestParseJSONBody(t *testing.T) {
	t.Skip("skipped by user request")
	defer cleanupTestFiles(t)

	server := createTestServerWithUpload(t)

	// Create test JSON data
	testContent := "Hello, this is a test file for base64 upload!"
	base64Content := base64.StdEncoding.EncodeToString([]byte(testContent))

	jsonData := map[string]any{
		"fileData":     base64Content,
		"filename":     "test-base64.txt",
		"directory":    "base64-uploads",
		"generateHash": true,
		"maxSize":      1048576,
	}

	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Create fasthttp request context
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetBody(jsonBytes)
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.Header.SetMethod("POST")

	// Test parsing
	result, err := server.parseRequestBody(ctx)
	if err != nil {
		t.Fatalf("Failed to parse JSON body: %v", err)
	}

	// Verify results
	fileData, ok := result["fileData"].(string)
	if !ok {
		t.Fatalf("Expected fileData string, got %T", result["fileData"])
	}

	filename, ok := result["filename"].(string)
	if !ok {
		t.Fatalf("Expected filename string, got %T", result["filename"])
	}

	directory, ok := result["directory"].(string)
	if !ok {
		t.Fatalf("Expected directory string, got %T", result["directory"])
	}

	// Verify values
	if fileData != base64Content {
		t.Errorf("Expected fileData to match base64 content")
	}

	if filename != "test-base64.txt" {
		t.Errorf("Expected filename 'test-base64.txt', got %s", filename)
	}

	if directory != "base64-uploads" {
		t.Errorf("Expected directory 'base64-uploads', got %s", directory)
	}

	t.Logf("Successfully parsed JSON body with fileData, filename: %s, directory: %s", filename, directory)
}

// Test request body parsing with different content types
func TestParseRequestBodyContentTypes(t *testing.T) {
	t.Skip("skipped by user request")
	defer cleanupTestFiles(t)

	server := createTestServerWithUpload(t)

	tests := []struct {
		name        string
		method      string
		contentType string
		body        []byte
		expectError bool
	}{
		{
			name:        "GET request",
			method:      "GET",
			contentType: "application/json",
			body:        []byte(`{"test": "data"}`),
			expectError: false, // Should return empty body, not error
		},
		{
			name:        "POST with JSON",
			method:      "POST",
			contentType: "application/json",
			body:        []byte(`{"fileData": "SGVsbG8=", "filename": "test.txt"}`),
			expectError: false,
		},
		{
			name:        "POST with image/png",
			method:      "POST",
			contentType: "image/png",
			body:        []byte("fake-png-data"),
			expectError: false,
		},
		{
			name:        "POST with invalid JSON",
			method:      "POST",
			contentType: "application/json",
			body:        []byte(`{"invalid": json}`),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			ctx.Request.SetBody(tt.body)
			ctx.Request.Header.SetContentType(tt.contentType)
			ctx.Request.Header.SetMethod(tt.method)

			result, err := server.parseRequestBody(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for test %s, but got none", tt.name)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for test %s: %v", tt.name, err)
				return
			}

			if result == nil {
				t.Errorf("Expected result for test %s, got nil", tt.name)
				return
			}

			t.Logf("Test %s passed: parsed %d keys from body", tt.name, len(result))
		})
	}
}

// Test complete request parsing to event for file upload
func TestParseRequestToEventUpload(t *testing.T) {
	t.Skip("skipped by user request")
	defer cleanupTestFiles(t)

	server := createTestServerWithUpload(t)

	// Create test endpoint config
	ep := config.EndpointConfig{
		Route:  "/test/upload/{id}",
		Method: "POST",
		Path:   "./app/routes/demo/upload",
	}

	// Create test JSON data
	jsonData := map[string]any{
		"fileData": "SGVsbG8gd29ybGQ=",
		"filename": "test.txt",
	}
	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Create fasthttp request context
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/test/upload/123?param1=value1&param2=value2")
	ctx.Request.SetBody(jsonBytes)
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.Header.Set("User-Agent", "TurboScript-Test/1.0")
	ctx.Request.Header.Set("Authorization", "Bearer test-token")

	// Compile path pattern for parameter extraction
	if server.pathPatterns == nil {
		server.pathPatterns = make(map[string]*regexp.Regexp)
	}
	if server.pathParams == nil {
		server.pathParams = make(map[string][]string)
	}
	pattern, params := server.compilePathPattern(ep.Route)
	server.pathPatterns[ep.Route] = pattern
	server.pathParams[ep.Route] = params

	// Test parsing
	event, err := server.parseRequestToEvent(ctx, ep, nil)
	if err != nil {
		t.Fatalf("Failed to parse request to event: %v", err)
	}

	// Verify basic event structure
	if event["method"] != "POST" {
		t.Errorf("Expected method 'POST', got %v", event["method"])
	}

	if event["path"] != "/test/upload/123" {
		t.Errorf("Expected path '/test/upload/123', got %v", event["path"])
	}

	// Verify headers
	headers, ok := event["headers"].(map[string]string)
	if !ok {
		t.Fatalf("Expected headers map, got %T", event["headers"])
	}

	if headers["User-Agent"] != "TurboScript-Test/1.0" {
		t.Errorf("Expected User-Agent 'TurboScript-Test/1.0', got %v", headers["User-Agent"])
	}

	// Verify query parameters
	queryParams, ok := event["queryParameters"].(map[string]string)
	if !ok {
		t.Fatalf("Expected queryParameters map, got %T", event["queryParameters"])
	}

	if queryParams["param1"] != "value1" {
		t.Errorf("Expected param1 'value1', got %v", queryParams["param1"])
	}

	// Verify path parameters
	pathParams, ok := event["pathParameters"].(map[string]string)
	if !ok {
		t.Fatalf("Expected pathParameters map, got %T", event["pathParameters"])
	}

	if pathParams["id"] != "123" {
		t.Errorf("Expected id '123', got %v", pathParams["id"])
	}

	// Verify body
	body, ok := event["body"].(map[string]any)
	if !ok {
		t.Fatalf("Expected body map, got %T", event["body"])
	}

	if body["filename"] != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got %v", body["filename"])
	}

	t.Logf("Successfully parsed complete request to event with %d headers, %d query params, %d path params",
		len(headers), len(queryParams), len(pathParams))
}

// Benchmark test for multipart parsing performance
func BenchmarkParseMultipartBody(b *testing.B) {
	server := createTestServerWithUpload(&testing.T{})

	// Create test multipart data
	testContent := strings.Repeat("Hello, this is test content for benchmarking! ", 100)
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add multiple files
	for i := 0; i < 5; i++ {
		fileWriter, _ := writer.CreateFormFile("file", fmt.Sprintf("test%d.txt", i))
		fileWriter.Write([]byte(testContent))
	}

	writer.WriteField("directory", "benchmark-test")
	writer.Close()

	bodyBytes := buf.Bytes()
	contentType := fmt.Sprintf("multipart/form-data; boundary=%s", writer.Boundary())

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetBody(bodyBytes)
		ctx.Request.Header.SetContentType(contentType)
		ctx.Request.Header.SetMethod("POST")

		_, err := server.parseMultipartBody(ctx)
		if err != nil {
			b.Fatalf("Failed to parse multipart body: %v", err)
		}
	}
}

// Test error handling for malformed requests
func TestParseRequestErrorHandling(t *testing.T) {
	t.Skip("skipped by user request")
	defer cleanupTestFiles(t)

	server := createTestServerWithUpload(t)

	tests := []struct {
		name        string
		contentType string
		body        string
		expectError bool
		errorSubstr string
	}{
		{
			name:        "Invalid JSON",
			contentType: "application/json",
			body:        `{"invalid": json}`,
			expectError: true,
			errorSubstr: "invalid JSON",
		},
		{
			name:        "Malformed multipart",
			contentType: "multipart/form-data; boundary=invalid",
			body:        "not-multipart-data",
			expectError: true,
			errorSubstr: "multipart",
		},
		{
			name:        "Missing boundary",
			contentType: "multipart/form-data",
			body:        "some-data",
			expectError: true,
			errorSubstr: "boundary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			ctx.Request.SetBody([]byte(tt.body))
			ctx.Request.Header.SetContentType(tt.contentType)
			ctx.Request.Header.SetMethod("POST")

			_, err := server.parseRequestBody(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for test %s, but got none", tt.name)
					return
				}
				if !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorSubstr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for test %s: %v", tt.name, err)
				}
			}
		})
	}
}

// Integration test that simulates the full upload workflow
func TestFileUploadIntegration(t *testing.T) {
	t.Skip("skipped by user request")
	defer cleanupTestFiles(t)

	// This test simulates what happens when the TypeScript upload route
	// receives different types of requests

	tests := []struct {
		name        string
		method      string
		contentType string
		setupBody   func() []byte
		expectCode  int
	}{
		{
			name:        "GET request for demo",
			method:      "GET",
			contentType: "text/html",
			setupBody:   func() []byte { return []byte{} },
			expectCode:  200, // Should show demo info
		},
		{
			name:        "Base64 JSON upload",
			method:      "POST",
			contentType: "application/json",
			setupBody: func() []byte {
				data := map[string]any{
					"fileData": base64.StdEncoding.EncodeToString([]byte("Hello World")),
					"filename": "test.txt",
				}
				jsonBytes, _ := json.Marshal(data)
				return jsonBytes
			},
			expectCode: 200,
		},
		{
			name:        "Raw binary upload",
			method:      "POST",
			contentType: "image/png",
			setupBody: func() []byte {
				// Return fake PNG data
				return []byte("fake-png-binary-data")
			},
			expectCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createTestServerWithUpload(t)

			ctx := &fasthttp.RequestCtx{}
			ctx.Request.SetBody(tt.setupBody())
			ctx.Request.Header.SetContentType(tt.contentType)
			ctx.Request.Header.SetMethod(tt.method)

			ep := config.EndpointConfig{
				Route:  "/test/upload",
				Method: tt.method,
				Path:   "./app/routes/demo/upload",
			}

			// Test request parsing
			event, err := server.parseRequestToEvent(ctx, ep, nil)
			if err != nil {
				t.Fatalf("Failed to parse request: %v", err)
			}

			// Verify the event was parsed correctly
			if event["method"] != tt.method {
				t.Errorf("Expected method %s, got %v", tt.method, event["method"])
			}

			// Check that body was parsed according to content type
			body := event["body"].(map[string]any)

			switch tt.contentType {
			case "application/json":
				if body["fileData"] == nil {
					t.Errorf("Expected fileData in JSON body")
				}
			case "image/png":
				if body["binaryData"] == nil {
					t.Errorf("Expected binaryData in binary body")
				}
			}

			t.Logf("Integration test %s passed: method=%s, body keys=%d",
				tt.name, event["method"], len(body))
		})
	}
}

// Test setup and teardown
func TestMain(m *testing.M) {
	// Setup
	logger.Info("Starting file upload tests...")

	// Run tests
	code := m.Run()

	// Cleanup
	logger.Info("File upload tests completed")

	os.Exit(code)
}
