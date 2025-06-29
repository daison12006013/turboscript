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

// Package templating provides a templating engine for processing layout files with markdown and HTML inclusion support.
package templating

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/russross/blackfriday/v2"
)

// Engine represents the templating engine for processing layout files.
type Engine struct {
	folderPath string
	basePath   string
}

// NewEngine creates a new templating engine instance.
func NewEngine(folderPath string) *Engine {
	return &Engine{
		folderPath: folderPath,
		basePath:   "",
	}
}

// NewEngineWithBasePath creates a new templating engine instance with base path for link adjustment.
func NewEngineWithBasePath(folderPath, basePath string) *Engine {
	return &Engine{
		folderPath: folderPath,
		basePath:   basePath,
	}
}

// ProcessLayout processes a layout template with content and title substitution.
func (e *Engine) ProcessLayout(layoutFile, content, title string) (string, error) {
	layoutPath := filepath.Join(e.folderPath, layoutFile)

	// Security check - ensure layout is within the allowed folder
	absFolder, err := filepath.Abs(e.folderPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve folder path: %w", err)
	}

	absLayout, err := filepath.Abs(layoutPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve layout path: %w", err)
	}

	if !strings.HasPrefix(absLayout, absFolder) {
		return "", fmt.Errorf("access denied: layout file outside allowed folder")
	}

	// Check if layout file exists
	if _, err := os.Stat(layoutPath); os.IsNotExist(err) {
		return "", fmt.Errorf("layout file not found: %s", layoutFile)
	}

	// #nosec G304: layoutPath is validated above to be within allowed folder
	layoutContent, err := os.ReadFile(layoutPath)
	if err != nil {
		return "", fmt.Errorf("failed to read layout file: %w", err)
	}

	// Process template with enhanced templating engine
	result := e.processTemplate(string(layoutContent), content, title)

	return result, nil
}

// processTemplate processes template content with support for @turboMarkdownHtml() and @html() directives.
func (e *Engine) processTemplate(template, content, title string) string {
	result := template

	// Replace basic template variables
	result = strings.ReplaceAll(result, "{{content}}", content)
	result = strings.ReplaceAll(result, "{{title}}", title)

	// Process @turboMarkdownHtml("file.md") and @mardownHtml("file.md") directives (with typo support)
	markdownPatterns := []*regexp.Regexp{
		regexp.MustCompile(`@turboMarkdownHtml\("([^"]+)"\)`),
		regexp.MustCompile(`@mardownHtml\("([^"]+)"\)`), // Support the typo in current layout.html
	}

	for _, pattern := range markdownPatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			matches := pattern.FindStringSubmatch(match)
			if len(matches) != 2 {
				logger.Warn("Invalid @turboMarkdownHtml directive: %s", match)
				return match
			}

			fileName := matches[1]
			htmlContent, err := e.processMarkdownInclude(fileName)
			if err != nil {
				logger.Warn("Failed to process @turboMarkdownHtml(%s): %v", fileName, err)
				return fmt.Sprintf("<!-- Error processing @turboMarkdownHtml(%s): %v -->", fileName, err)
			}
			return htmlContent
		})
	}

	// Process @turboHtml("file.html") directives
	htmlPatterns := []*regexp.Regexp{
		regexp.MustCompile(`@turboHtml\("([^"]+)"\)`),
	}

	for _, htmlPattern := range htmlPatterns {
		result = htmlPattern.ReplaceAllStringFunc(result, func(match string) string {
			matches := htmlPattern.FindStringSubmatch(match)
			if len(matches) != 2 {
				logger.Warn("Invalid HTML directive: %s", match)
				return match
			}

			fileName := matches[1]
			htmlContent, err := e.processHTMLInclude(fileName)
			if err != nil {
				logger.Warn("Failed to process HTML directive(%s): %v", fileName, err)
				return fmt.Sprintf("<!-- Error processing HTML directive(%s): %v -->", fileName, err)
			}
			return htmlContent
		})
	}

	return result
}

// processMarkdownInclude processes a markdown file for inclusion.
func (e *Engine) processMarkdownInclude(fileName string) (string, error) {
	filePath := filepath.Join(e.folderPath, fileName)

	// Security check - ensure file is within the allowed folder
	absFolder, err := filepath.Abs(e.folderPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve folder path: %w", err)
	}

	absFile, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve file path: %w", err)
	}

	if !strings.HasPrefix(absFile, absFolder) {
		return "", fmt.Errorf("access denied: file outside allowed folder")
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("markdown file not found: %s", fileName)
	}

	// #nosec G304: filePath is validated above to be within allowed folder
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read markdown file: %w", err)
	}

	// Convert markdown to HTML
	htmlContent := convertMarkdownToHTML(string(content))

	// Adjust relative links if base path is provided
	if e.basePath != "" {
		htmlContent = e.adjustMarkdownLinks(htmlContent)
	}

	return htmlContent, nil
}

// processHTMLInclude processes an HTML file for inclusion.
func (e *Engine) processHTMLInclude(fileName string) (string, error) {
	filePath := filepath.Join(e.folderPath, fileName)

	// Security check - ensure file is within the allowed folder
	absFolder, err := filepath.Abs(e.folderPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve folder path: %w", err)
	}

	absFile, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve file path: %w", err)
	}

	if !strings.HasPrefix(absFile, absFolder) {
		return "", fmt.Errorf("access denied: file outside allowed folder")
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("HTML file not found: %s", fileName)
	}

	// #nosec G304: filePath is validated above to be within allowed folder
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read HTML file: %w", err)
	}

	return string(content), nil
}

// convertMarkdownToHTML converts markdown content to HTML using blackfriday.
func convertMarkdownToHTML(markdown string) string {
	// Configure blackfriday with common extensions
	extensions := blackfriday.CommonExtensions | blackfriday.AutoHeadingIDs | blackfriday.Footnotes
	renderer := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
		Flags: blackfriday.CommonHTMLFlags | blackfriday.HrefTargetBlank,
	})

	// Convert markdown to HTML
	htmlBytes := blackfriday.Run([]byte(markdown), blackfriday.WithRenderer(renderer), blackfriday.WithExtensions(extensions))
	return string(htmlBytes)
}

// ExtractTitleFromMarkdown extracts the title from markdown content.
// It looks for the first H1 header (# Title) and returns it as the title.
func ExtractTitleFromMarkdown(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			title := strings.TrimPrefix(line, "# ")
			title = strings.TrimSpace(title)
			// Remove any markdown formatting like ** or *
			title = strings.ReplaceAll(title, "**", "")
			title = strings.ReplaceAll(title, "*", "")
			return title
		}
	}
	return "Document" // Default title if no H1 found
}

// ExtractTitleFromHTML extracts the title from HTML content.
// It looks for the <title> tag first, then falls back to the first <h1> tag.
func ExtractTitleFromHTML(content string) string {
	// First try to extract from <title> tag
	titlePattern := regexp.MustCompile(`<title[^>]*>([^<]*)</title>`)
	if matches := titlePattern.FindStringSubmatch(content); len(matches) > 1 {
		title := strings.TrimSpace(matches[1])
		if title != "" {
			return title
		}
	}

	// Fallback to first <h1> tag - capture everything between opening and closing tags
	h1Pattern := regexp.MustCompile(`<h1[^>]*>(.*?)</h1>`)
	if matches := h1Pattern.FindStringSubmatch(content); len(matches) > 1 {
		title := strings.TrimSpace(matches[1])
		// Remove any HTML tags that might be inside h1
		htmlTagPattern := regexp.MustCompile(`<[^>]+>`)
		title = htmlTagPattern.ReplaceAllString(title, "")
		title = strings.TrimSpace(title)
		if title != "" {
			return title
		}
	}

	return "Document" // Default title if no title found
}

// adjustMarkdownLinks adjusts relative links in HTML to include the base path prefix.
func (e *Engine) adjustMarkdownLinks(html string) string {
	// Use regex to find all href attributes
	// Pattern matches href="..." to capture all href values
	hrefPattern := regexp.MustCompile(`href="([^"]*)"`)

	return hrefPattern.ReplaceAllStringFunc(html, func(match string) string {
		// Extract the href value
		hrefMatch := hrefPattern.FindStringSubmatch(match)
		if len(hrefMatch) < 2 {
			return match // Return original if we can't extract href
		}

		originalHref := hrefMatch[1]

		// Skip if it's already absolute or an anchor
		if strings.HasPrefix(originalHref, "http://") ||
			strings.HasPrefix(originalHref, "https://") ||
			strings.HasPrefix(originalHref, "//") ||
			strings.HasPrefix(originalHref, "#") ||
			strings.HasPrefix(originalHref, "mailto:") {
			return match
		}

		// Remove leading "./" if present
		cleanHref := strings.TrimPrefix(originalHref, "./")

		// Construct the new href with base path
		var newHref string
		if strings.HasPrefix(cleanHref, "/") {
			// Already absolute path, just use it
			newHref = cleanHref
		} else {
			// Relative path, add base path
			newHref = e.basePath + cleanHref
		}

		return fmt.Sprintf(`href="%s"`, newHref)
	})
}

// ExtractBasePath extracts the base path from a route pattern for link adjustment.
// For example: "/docs{file:(?:/.*)?}" -> "/docs/".
func ExtractBasePath(route string) string {
	// Remove parameter patterns to get the base path
	// Match patterns like {param} or {param:pattern}
	paramPattern := regexp.MustCompile(`\{[^}]*\}`)
	basePath := paramPattern.ReplaceAllString(route, "")

	// Ensure it ends with a slash for proper link joining
	if !strings.HasSuffix(basePath, "/") {
		basePath += "/"
	}

	return basePath
}

// ProcessMarkdownWithLayout processes markdown content and applies a layout template.
func (e *Engine) ProcessMarkdownWithLayout(layoutFile, markdownContent, title string) (string, error) {
	// Convert markdown to HTML
	htmlContent := convertMarkdownToHTML(markdownContent)

	// Adjust relative links if base path is provided
	if e.basePath != "" {
		htmlContent = e.adjustMarkdownLinks(htmlContent)
	}

	// Apply layout
	return e.ProcessLayout(layoutFile, htmlContent, title)
}

// ProcessMarkdownToHTML converts markdown to HTML with optional link adjustment.
func (e *Engine) ProcessMarkdownToHTML(markdownContent string) string {
	// Convert markdown to HTML
	htmlContent := convertMarkdownToHTML(markdownContent)

	// Adjust relative links if base path is provided
	if e.basePath != "" {
		htmlContent = e.adjustMarkdownLinks(htmlContent)
	}

	return htmlContent
}

// ConvertMarkdownToHTML exposes the markdown conversion function for external use.
func ConvertMarkdownToHTML(markdown string) string {
	return convertMarkdownToHTML(markdown)
}

// ProcessHTMLContent processes HTML content with templating directives like @turboHtml() and @turboMarkdownHtml().
func (e *Engine) ProcessHTMLContent(htmlContent string) string {
	return e.processTemplate(htmlContent, "", "")
}
