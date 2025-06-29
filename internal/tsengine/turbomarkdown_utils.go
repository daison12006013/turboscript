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

// Package tsengine provides turboMarkdownHtml and turboHtml utilities for content processing.
package tsengine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/daison12006013/turboscript/internal/templating"
	"github.com/dop251/goja"
)

// CachedContent holds processed content with metadata for caching.
type CachedContent struct {
	Content   string
	Timestamp time.Time
	FileSize  int64
	ModTime   time.Time
}

// TurboMarkdownUtils handles markdown and HTML processing for TypeScript functions.
type TurboMarkdownUtils struct {
	basePath string       // Base path for resolving relative file paths
	cache    sync.Map     // Cache for processed content
	mutex    sync.RWMutex // Mutex for thread-safe operations
}

// NewTurboMarkdownUtils creates a new turbo markdown utilities instance.
func NewTurboMarkdownUtils(basePath string) *TurboMarkdownUtils {
	return &TurboMarkdownUtils{
		basePath: basePath,
		cache:    sync.Map{},
	}
}

// SetBasePath sets the base path for resolving relative file paths.
func (tmu *TurboMarkdownUtils) SetBasePath(basePath string) {
	tmu.mutex.Lock()
	defer tmu.mutex.Unlock()
	tmu.basePath = basePath
	// Clear cache when base path changes
	tmu.cache = sync.Map{}
}

// parseFileProcessingArgs extracts file path and data from function call arguments.
func (tmu *TurboMarkdownUtils) parseFileProcessingArgs(call goja.FunctionCall, rt *goja.Runtime, functionName string) (string, map[string]any) {
	logger.Debug("%s called with %d arguments", functionName, len(call.Arguments))

	// Validate arguments
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError(functionName + " requires at least 1 argument (filePath)"))
	}

	// Get file path argument
	filePath := call.Arguments[0].String()
	if filePath == "" {
		panic(rt.NewTypeError(functionName + " filePath cannot be empty"))
	}

	// Get optional data argument
	var data map[string]any
	if len(call.Arguments) > 1 && !goja.IsNull(call.Arguments[1]) && !goja.IsUndefined(call.Arguments[1]) {
		exported := call.Arguments[1].Export()
		if exportedMap, ok := exported.(map[string]any); ok {
			data = exportedMap
		}
	}

	return filePath, data
}

// ExecuteMarkdownHTML processes markdown content and returns HTML.
func (tmu *TurboMarkdownUtils) ExecuteMarkdownHTML(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	filePath, data := tmu.parseFileProcessingArgs(call, rt, "turboMarkdownHtml")

	// Process the markdown file
	htmlContent, err := tmu.processMarkdownFile(filePath, data)
	if err != nil {
		logger.Error("turboMarkdownHtml error: %v", err)
		panic(rt.NewTypeError(err.Error()))
	}

	logger.Debug("turboMarkdownHtml processed file: %s", filePath)
	return rt.ToValue(htmlContent)
}

// ExecuteHTML processes HTML content with template substitution and returns HTML.
func (tmu *TurboMarkdownUtils) ExecuteHTML(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	filePath, data := tmu.parseFileProcessingArgs(call, rt, "turboHtml")

	// Process the HTML file
	htmlContent, err := tmu.processHTMLFile(filePath, data)
	if err != nil {
		logger.Error("turboHtml error: %v", err)
		panic(rt.NewTypeError(err.Error()))
	}

	logger.Debug("turboHtml processed file: %s", filePath)
	return rt.ToValue(htmlContent)
}

// resolveFilePath resolves a file path relative to the base path.
func (tmu *TurboMarkdownUtils) resolveFilePath(filePath string) string {
	// If path is absolute, use it directly
	if filepath.IsAbs(filePath) {
		return filePath
	}

	// If no base path set, resolve relative to current working directory
	if tmu.basePath == "" {
		if cwd, err := os.Getwd(); err == nil {
			return filepath.Join(cwd, filePath)
		}
		return filePath
	}

	// Resolve relative to base path
	resolvedPath := filepath.Join(tmu.basePath, filePath)

	// Security check - ensure resolved path is within base path
	absBasePath, err := filepath.Abs(tmu.basePath)
	if err != nil {
		logger.Warn("Failed to get absolute base path: %v", err)
		return resolvedPath
	}

	absResolvedPath, err := filepath.Abs(resolvedPath)
	if err != nil {
		logger.Warn("Failed to get absolute resolved path: %v", err)
		return resolvedPath
	}

	// Ensure the resolved path is within the base path (security check)
	if !strings.HasPrefix(absResolvedPath, absBasePath+string(filepath.Separator)) && absResolvedPath != absBasePath {
		logger.Warn("Security violation: path %s is outside base path %s", absResolvedPath, absBasePath)
		// Return a safe path within the base directory
		return filepath.Join(absBasePath, filepath.Base(filePath))
	}

	return absResolvedPath
}

// processMarkdownFile processes a markdown file with caching.
func (tmu *TurboMarkdownUtils) processMarkdownFile(filePath string, data map[string]any) (string, error) {
	// Resolve file path
	resolvedPath := tmu.resolveFilePath(filePath)

	// Check cache first
	if cachedContent := tmu.getCachedContent(resolvedPath); cachedContent != "" {
		// Apply template substitution if data provided
		if data != nil {
			return tmu.applyTemplateSubstitution(cachedContent, data), nil
		}
		return cachedContent, nil
	}

	// Check if file exists
	if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		return "", fmt.Errorf("markdown file not found: %s (resolved to: %s)", filePath, resolvedPath)
	}

	// #nosec G304: resolvedPath is validated by resolveFilePath to be within basePath
	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read markdown file: %w", err)
	}

	// Process markdown content - convert to HTML using the templating engine
	htmlContent := templating.ConvertMarkdownToHTML(string(content))

	// Cache the processed content
	tmu.setCachedContent(resolvedPath, htmlContent)

	// Apply template substitution if data provided
	if data != nil {
		return tmu.applyTemplateSubstitution(htmlContent, data), nil
	}

	return htmlContent, nil
}

// processHTMLFile processes an HTML file with caching.
func (tmu *TurboMarkdownUtils) processHTMLFile(filePath string, data map[string]any) (string, error) {
	// Resolve file path
	resolvedPath := tmu.resolveFilePath(filePath)

	// Check cache first
	if cachedContent := tmu.getCachedContent(resolvedPath); cachedContent != "" {
		// Apply template substitution if data provided
		if data != nil {
			return tmu.applyTemplateSubstitution(cachedContent, data), nil
		}
		return cachedContent, nil
	}

	// Check if file exists
	if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		return "", fmt.Errorf("HTML file not found: %s (resolved to: %s)", filePath, resolvedPath)
	}

	// #nosec G304: resolvedPath is validated by resolveFilePath to be within basePath
	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read HTML file: %w", err)
	}

	htmlContent := string(content)

	// Apply template substitution if data provided
	if data != nil {
		htmlContent = tmu.applyTemplateSubstitution(htmlContent, data)
	}

	// Process templating directives like @turboHtml() and @turboMarkdownHtml()
	if tmu.basePath != "" {
		// Create templating engine with the base path as the folder path
		engine := templating.NewEngine(tmu.basePath)
		processedContent := engine.ProcessHTMLContent(htmlContent)
		htmlContent = processedContent
	}

	// Cache the processed content
	tmu.setCachedContent(resolvedPath, htmlContent)

	return htmlContent, nil
}

// getCachedContent retrieves content from cache if valid.
func (tmu *TurboMarkdownUtils) getCachedContent(filePath string) string {
	// Get file info for cache validation
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return ""
	}

	// Check cache
	if cached, ok := tmu.cache.Load(filePath); ok {
		if cachedContent, ok := cached.(*CachedContent); ok {
			// Validate cache - check if file has been modified
			if cachedContent.ModTime.Equal(fileInfo.ModTime()) && cachedContent.FileSize == fileInfo.Size() {
				logger.Debug("Using cached content for %s", filePath)
				return cachedContent.Content
			}
			// Cache is stale, remove it
			tmu.cache.Delete(filePath)
		}
	}

	return ""
}

// setCachedContent stores content in cache.
func (tmu *TurboMarkdownUtils) setCachedContent(filePath, content string) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		logger.Warn("Failed to get file info for caching: %v", err)
		return
	}

	cachedContent := &CachedContent{
		Content:   content,
		Timestamp: time.Now(),
		FileSize:  fileInfo.Size(),
		ModTime:   fileInfo.ModTime(),
	}

	tmu.cache.Store(filePath, cachedContent)
	logger.Debug("Cached content for %s", filePath)
}

// applyTemplateSubstitution applies simple template variable substitution.
func (tmu *TurboMarkdownUtils) applyTemplateSubstitution(content string, data map[string]any) string {
	result := content

	// Simple template substitution using {{key}} format
	for key, value := range data {
		placeholder := fmt.Sprintf("{{%s}}", key)
		valueStr := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, valueStr)
	}

	return result
}
