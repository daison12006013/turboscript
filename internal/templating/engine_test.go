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

package templating

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractTitleFromMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "Simple H1 title",
			content:  "# Hello World\n\nThis is content.",
			expected: "Hello World",
		},
		{
			name:     "H1 with bold formatting",
			content:  "# **Bold Title**\n\nContent here.",
			expected: "Bold Title",
		},
		{
			name:     "H1 with italic formatting",
			content:  "# *Italic Title*\n\nContent here.",
			expected: "Italic Title",
		},
		{
			name:     "No H1 header",
			content:  "## H2 Header\n\nContent here.",
			expected: "Document",
		},
		{
			name:     "Empty content",
			content:  "",
			expected: "Document",
		},
		{
			name:     "H1 with extra spaces",
			content:  "#   Spaced Title   \n\nContent here.",
			expected: "Spaced Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTitleFromMarkdown(tt.content)
			if result != tt.expected {
				t.Errorf("ExtractTitleFromMarkdown() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractTitleFromHTML(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "Title tag",
			content:  "<html><head><title>Page Title</title></head><body>Content</body></html>",
			expected: "Page Title",
		},
		{
			name:     "H1 tag fallback",
			content:  "<html><body><h1>Header Title</h1><p>Content</p></body></html>",
			expected: "Header Title",
		},
		{
			name:     "Title tag takes precedence over H1",
			content:  "<html><head><title>Page Title</title></head><body><h1>Header Title</h1></body></html>",
			expected: "Page Title",
		},
		{
			name:     "H1 with nested tags",
			content:  "<html><body><h1>Header <span>with</span> tags</h1></body></html>",
			expected: "Header with tags",
		},
		{
			name:     "No title or H1",
			content:  "<html><body><h2>H2 Header</h2><p>Content</p></body></html>",
			expected: "Document",
		},
		{
			name:     "Empty content",
			content:  "",
			expected: "Document",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTitleFromHTML(tt.content)
			if result != tt.expected {
				t.Errorf("ExtractTitleFromHTML() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTemplateProcessing(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "templating_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	markdownFile := filepath.Join(tempDir, "test.md")
	err = os.WriteFile(markdownFile, []byte("# Test Title\n\nThis is **markdown** content."), 0644)
	if err != nil {
		t.Fatalf("Failed to create markdown file: %v", err)
	}

	htmlFile := filepath.Join(tempDir, "test.html")
	err = os.WriteFile(htmlFile, []byte("<div>This is <strong>HTML</strong> content.</div>"), 0644)
	if err != nil {
		t.Fatalf("Failed to create HTML file: %v", err)
	}

	layoutFile := filepath.Join(tempDir, "layout.html")
	layoutContent := `<!DOCTYPE html>
<html>
<head>
    <title>{{title}}</title>
</head>
<body>
    @turboMarkdownHtml("test.md")
    <main>{{content}}</main>
    @turboHtml("test.html")
</body>
</html>`
	err = os.WriteFile(layoutFile, []byte(layoutContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create layout file: %v", err)
	}

	// Test the templating engine
	engine := NewEngine(tempDir)
	result, err := engine.ProcessLayout("layout.html", "Main content here", "Test Page")
	if err != nil {
		t.Fatalf("ProcessLayout failed: %v", err)
	}

	// Verify the result contains expected content
	if !strings.Contains(result, "<title>Test Page</title>") {
		t.Error("Title not properly substituted")
	}
	if !strings.Contains(result, "Main content here") {
		t.Error("Content not properly substituted")
	}
	if !strings.Contains(result, "<h1 id=\"test-title\">Test Title</h1>") {
		t.Error("Markdown not properly converted and included")
	}
	if !strings.Contains(result, "<div>This is <strong>HTML</strong> content.</div>") {
		t.Error("HTML not properly included")
	}
}

func TestTemplateProcessingWithBasePath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "templating_basepath_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test markdown file with relative links
	markdownFile := filepath.Join(tempDir, "nav.md")
	err = os.WriteFile(markdownFile, []byte(`# Navigation
- [Home](./index.md)
- [About](./about.md)
- [Contact](contact.md)
- [External](https://example.com)
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create markdown file: %v", err)
	}

	layoutFile := filepath.Join(tempDir, "layout.html")
	layoutContent := `<!DOCTYPE html>
<html>
<head>
    <title>{{title}}</title>
</head>
<body>
    <nav>@turboMarkdownHtml("nav.md")</nav>
    <main>{{content}}</main>
</body>
</html>`
	err = os.WriteFile(layoutFile, []byte(layoutContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create layout file: %v", err)
	}

	// Test the templating engine with base path
	engine := NewEngineWithBasePath(tempDir, "/docs/")
	result, err := engine.ProcessLayout("layout.html", "Main content", "Test Page")
	if err != nil {
		t.Fatalf("ProcessLayout failed: %v", err)
	}

	// Verify that relative links are adjusted
	if !strings.Contains(result, `href="/docs/index.md"`) {
		t.Error("Relative link ./index.md not properly adjusted to /docs/index.md")
	}
	if !strings.Contains(result, `href="/docs/about.md"`) {
		t.Error("Relative link ./about.md not properly adjusted to /docs/about.md")
	}
	if !strings.Contains(result, `href="/docs/contact.md"`) {
		t.Error("Relative link contact.md not properly adjusted to /docs/contact.md")
	}
	if !strings.Contains(result, `href="https://example.com"`) {
		t.Error("External link should not be modified")
	}
}

func TestTemplateProcessingWithTypo(t *testing.T) {
	// Test support for the typo in current layout.html: @mardownHtml instead of @turboMarkdownHtml
	tempDir, err := os.MkdirTemp("", "templating_typo_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test markdown file
	markdownFile := filepath.Join(tempDir, "sidebar.md")
	err = os.WriteFile(markdownFile, []byte("# Sidebar\n\nSidebar content here."), 0644)
	if err != nil {
		t.Fatalf("Failed to create markdown file: %v", err)
	}

	layoutFile := filepath.Join(tempDir, "layout.html")
	layoutContent := `<!DOCTYPE html>
<html>
<head>
    <title>{{title}}</title>
</head>
<body>
    @mardownHtml("sidebar.md")
    <main>{{content}}</main>
</body>
</html>`
	err = os.WriteFile(layoutFile, []byte(layoutContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create layout file: %v", err)
	}

	// Test the templating engine with typo
	engine := NewEngine(tempDir)
	result, err := engine.ProcessLayout("layout.html", "Main content", "Test Page")
	if err != nil {
		t.Fatalf("ProcessLayout failed: %v", err)
	}

	// Verify the typo version works
	if !strings.Contains(result, "<h1 id=\"sidebar\">Sidebar</h1>") {
		t.Error("Typo version @mardownHtml not properly processed")
	}
	if !strings.Contains(result, "Sidebar content here.") {
		t.Error("Markdown content not properly included via typo directive")
	}
}

func TestSecurityChecks(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "templating_security_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file outside the allowed directory
	outsideDir, err := os.MkdirTemp("", "outside_dir")
	if err != nil {
		t.Fatalf("Failed to create outside dir: %v", err)
	}
	defer os.RemoveAll(outsideDir)

	outsideFile := filepath.Join(outsideDir, "outside.md")
	err = os.WriteFile(outsideFile, []byte("# Outside File\n\nThis should not be accessible."), 0644)
	if err != nil {
		t.Fatalf("Failed to create outside file: %v", err)
	}

	engine := NewEngine(tempDir)

	// Test attempting to access file outside allowed directory
	_, err = engine.processMarkdownInclude("../../../" + outsideFile)
	if err == nil {
		t.Error("Expected security error when accessing file outside allowed directory")
	}
	if !strings.Contains(err.Error(), "access denied") {
		t.Errorf("Expected 'access denied' error, got: %v", err)
	}

	// Test attempting to access non-existent file
	_, err = engine.processMarkdownInclude("nonexistent.md")
	if err == nil {
		t.Error("Expected error when accessing non-existent file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestExtractBasePath(t *testing.T) {
	tests := []struct {
		name     string
		route    string
		expected string
	}{
		{
			name:     "Simple docs route",
			route:    "/docs{file:(?:/.*)?}",
			expected: "/docs/",
		},
		{
			name:     "API route with version",
			route:    "/api/v1{path:.*}",
			expected: "/api/v1/",
		},
		{
			name:     "Root route",
			route:    "/{file:.*}",
			expected: "/",
		},
		{
			name:     "Static route without parameters",
			route:    "/static",
			expected: "/static/",
		},
		{
			name:     "Complex route with multiple parameters",
			route:    "/users/{id}/posts{slug:(?:/.*)?}",
			expected: "/users//posts/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractBasePath(tt.route)
			if result != tt.expected {
				t.Errorf("ExtractBasePath(%s) = %v, want %v", tt.route, result, tt.expected)
			}
		})
	}
}

func TestProcessMarkdownWithLayout(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "templating_markdown_layout_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create layout file
	layoutFile := filepath.Join(tempDir, "layout.html")
	layoutContent := `<!DOCTYPE html>
<html>
<head><title>{{title}}</title></head>
<body>
<nav>Navigation</nav>
<main>{{content}}</main>
</body>
</html>`
	err = os.WriteFile(layoutFile, []byte(layoutContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create layout file: %v", err)
	}

	// Test markdown processing with layout
	engine := NewEngineWithBasePath(tempDir, "/docs/")
	markdownContent := "# Test Page\n\nThis is [a link](./other.md) to another page."
	result, err := engine.ProcessMarkdownWithLayout("layout.html", markdownContent, "Test Page")
	if err != nil {
		t.Fatalf("ProcessMarkdownWithLayout failed: %v", err)
	}

	// Verify the result
	if !strings.Contains(result, "<title>Test Page</title>") {
		t.Error("Title not properly substituted in layout")
	}
	if !strings.Contains(result, "<h1 id=\"test-page\">Test Page</h1>") {
		t.Error("Markdown not properly converted in layout")
	}
	if !strings.Contains(result, `href="/docs/other.md"`) {
		t.Error("Link not properly adjusted with base path")
	}
}

func TestProcessMarkdownToHTML(t *testing.T) {
	// Test with base path
	engine := NewEngineWithBasePath("", "/docs/")
	markdownContent := "# Test\n\nLink to [page](./test.md) and [external](https://example.com)."
	result := engine.ProcessMarkdownToHTML(markdownContent)

	// Verify markdown conversion and link adjustment
	if !strings.Contains(result, "<h1 id=\"test\">Test</h1>") {
		t.Error("Markdown not properly converted")
	}
	if !strings.Contains(result, `href="/docs/test.md"`) {
		t.Error("Relative link not properly adjusted")
	}
	if !strings.Contains(result, `href="https://example.com"`) {
		t.Error("External link should not be modified")
	}

	// Test without base path
	engineNoBase := NewEngine("")
	resultNoBase := engineNoBase.ProcessMarkdownToHTML(markdownContent)
	if !strings.Contains(resultNoBase, `href="./test.md"`) {
		t.Error("Link should not be adjusted without base path")
	}
}

func TestConvertMarkdownToHTML(t *testing.T) {
	markdownContent := "# Header\n\nThis is **bold** text with `code`."
	result := ConvertMarkdownToHTML(markdownContent)

	// Verify basic markdown conversion
	if !strings.Contains(result, "<h1 id=\"header\">Header</h1>") {
		t.Error("Header not properly converted")
	}
	if !strings.Contains(result, "<strong>bold</strong>") {
		t.Error("Bold text not properly converted")
	}
	if !strings.Contains(result, "<code>code</code>") {
		t.Error("Code not properly converted")
	}
}
