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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/daison12006013/turboscript/internal/performance"
	"github.com/daison12006013/turboscript/internal/templating"
	"github.com/valyala/fasthttp"
)

// Constants for response types.
const (
	responseTypeMarkdownHTML  = "markdown-html"
	responseTypeTemplatedHTML = "templated-html"
)

// handleFolderEndpoint handles folder-based routing, scanning directories for files.
func (s *Server) handleFolderEndpoint(ctx *fasthttp.RequestCtx, ep config.EndpointConfig, event map[string]any, perfCtx *performance.RequestContext) {
	if perfCtx != nil {
		perfCtx.StartResponseProcessing()
		defer perfCtx.EndResponseProcessing()
	}

	// Extract file path from URL path parameters if available
	var targetFile string

	// Check if there's a file parameter in the path
	if pathParams, ok := event["pathParameters"].(map[string]string); ok {
		if file, exists := pathParams["file"]; exists {
			targetFile = file
			// Remove leading slash if present (from regex capture group)
			targetFile = strings.TrimPrefix(targetFile, "/")
		}
	}

	// If no file specified and index is configured, use index file
	if targetFile == "" && ep.GetIndexFile() != "" {
		targetFile = ep.GetIndexFile()
	}

	// Determine base path for link adjustment from the route pattern
	basePath := templating.ExtractBasePath(ep.Route)

	var responseResult json.RawMessage
	var err error

	if targetFile != "" {
		// Serve specific file
		responseResult, err = s.serveFolderFile(ep.Path, targetFile, ep.IsMarkdownEnabled(), ep.GetType(), ep.GetIndexFile(), ep.GetLayoutFile(), basePath)
	} else {
		// List folder contents
		responseResult, err = s.listFolderContents(ep.Path, ep.IsMarkdownEnabled(), ep.GetType())
	}

	if err != nil {
		logger.Error("Folder endpoint failed: %v", err)
		s.sendError(ctx, "Folder Endpoint Error", err.Error())
		return
	}

	s.handleFinalResponse(ctx, responseResult, perfCtx)
}

// serveFolderFile serves a specific file from a folder.
func (s *Server) serveFolderFile(folderPath, fileName string, isMarkdown bool, responseType, indexFile, layoutFile, basePath string) (json.RawMessage, error) {
	filePath := filepath.Join(folderPath, fileName)

	// Security check - ensure file is within the allowed folder
	absFolder, err := filepath.Abs(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve folder path: %w", err)
	}

	absFile, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve file path: %w", err)
	}

	if !strings.HasPrefix(absFile, absFolder) {
		return nil, fmt.Errorf("access denied: file outside allowed folder")
	}

	// Handle file/directory resolution and security checks
	finalPath, err := s.resolveFolderPath(filePath, indexFile, absFolder, fileName)
	if err != nil {
		return s.createErrorResponse(err)
	}

	// Read and process the file content
	return s.processFileContent(finalPath, fileName, indexFile, isMarkdown, responseType, layoutFile, folderPath, basePath)
}

// resolveFolderPath resolves the final file path, handling directories and security checks.
func (s *Server) resolveFolderPath(filePath, indexFile, absFolder, fileName string) (string, error) {
	// Check if path exists and get file info
	fileInfo, err := os.Lstat(filePath) // Use Lstat to detect symlinks
	if os.IsNotExist(err) {
		return "", fmt.Errorf("file not found")
	}

	// Security check: Block symlinks that could lead outside the allowed folder
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		if err := s.validateSymlink(filePath, absFolder); err != nil {
			return "", err
		}
		// Re-check the target file info
		fileInfo, err = os.Stat(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to stat symlink target: %w", err)
		}
	}

	// Handle directory case
	if fileInfo.IsDir() {
		return s.resolveDirectoryIndex(filePath, indexFile, absFolder, fileName)
	}

	return filePath, nil
}

// validateSymlink validates that a symlink target is within the allowed folder.
func (s *Server) validateSymlink(symlinkPath, absFolder string) error {
	targetPath, err := filepath.EvalSymlinks(symlinkPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlink: %w", err)
	}

	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlink target path: %w", err)
	}

	if !strings.HasPrefix(absTarget, absFolder) {
		return fmt.Errorf("access denied: symlink target outside allowed folder")
	}

	return nil
}

// resolveDirectoryIndex resolves the index file path for a directory.
func (s *Server) resolveDirectoryIndex(dirPath, indexFile, absFolder, fileName string) (string, error) {
	if indexFile == "" {
		return "", fmt.Errorf("failed to read file: read %s: is a directory", fileName)
	}

	// Construct path to index file within the directory
	indexPath := filepath.Join(dirPath, indexFile)

	// Security check for the index file path
	absIndexFile, err := filepath.Abs(indexPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve index file path: %w", err)
	}

	if !strings.HasPrefix(absIndexFile, absFolder) {
		return "", fmt.Errorf("access denied: index file outside allowed folder")
	}

	// Check if index file exists and verify it's not a malicious symlink
	indexInfo, err := os.Lstat(indexPath) // Use Lstat to detect symlinks
	if os.IsNotExist(err) {
		return "", fmt.Errorf("index file '%s' not found in directory '%s'", indexFile, fileName)
	}

	// Security check: Block symlinks in index files
	if indexInfo.Mode()&os.ModeSymlink != 0 {
		if err := s.validateSymlink(indexPath, absFolder); err != nil {
			return "", fmt.Errorf("index file symlink security violation: %w", err)
		}
	}

	return indexPath, nil
}

// createErrorResponse creates a JSON error response for the given error.
func (s *Server) createErrorResponse(err error) (json.RawMessage, error) {
	if strings.Contains(err.Error(), "not found") {
		response := map[string]any{
			"code": 404,
			"response": map[string]any{
				"status":  "error",
				"message": err.Error(),
			},
		}
		return json.Marshal(response)
	}
	return nil, err
}

// processFileContent reads and processes file content based on the response type.
func (s *Server) processFileContent(filePath, originalFileName, indexFile string, isMarkdown bool, responseType, layoutFile, folderPath, basePath string) (json.RawMessage, error) {
	// #nosec G304: filePath is validated by serveFolderFile to be within allowed folder
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Determine the filename to use for extension detection
	fileName := originalFileName
	if filepath.Base(filePath) == indexFile {
		fileName = indexFile
	}

	// Process content and create response
	processedContent, finalResponseType, _ := s.processContentByType(content, fileName, isMarkdown, responseType, layoutFile, folderPath, basePath)

	response := map[string]any{
		"code":     200,
		"type":     finalResponseType,
		"response": processedContent,
	}

	return json.Marshal(response)
}

// processContentByType processes content based on its type and configuration.
func (s *Server) processContentByType(content []byte, fileName string, isMarkdown bool, responseType, layoutFile, folderPath, basePath string) (string, string, string) {
	fileExt := strings.ToLower(filepath.Ext(fileName))
	var finalResponseType string
	title := "Document" // Default title

	// Determine response type
	if responseType != "" {
		finalResponseType = responseType
	} else {
		finalResponseType = s.detectResponseType(fileExt, isMarkdown)
	}

	// Process content based on type
	if finalResponseType == responseTypeMarkdownHTML && (fileExt == fileExtMD || fileExt == fileExtMarkdown) {
		return s.processMarkdownContent(content, layoutFile, folderPath, basePath)
	} else if finalResponseType == contentTypeHTML && (fileExt == fileExtHTML || fileExt == fileExtHTM) {
		return s.processHTMLContent(content, layoutFile, folderPath, basePath)
	}

	return string(content), finalResponseType, title
}

// detectResponseType determines the response type based on file extension and settings.
func (s *Server) detectResponseType(fileExt string, isMarkdown bool) string {
	switch fileExt {
	case fileExtMD, fileExtMarkdown:
		if isMarkdown {
			return "markdown"
		}
		return contentTypeText
	case ".html", ".htm":
		return contentTypeHTML
	case ".txt":
		return contentTypeText
	default:
		// For other files, try to detect type from content
		return autoDetectResponseType(json.RawMessage(fmt.Sprintf(`"%s"`, fileExt)))
	}
}

// processMarkdownContent processes markdown content with optional layout.
func (s *Server) processMarkdownContent(content []byte, layoutFile, folderPath, basePath string) (string, string, string) {
	title := templating.ExtractTitleFromMarkdown(string(content))

	if layoutFile != "" {
		engine := templating.NewEngineWithBasePath(folderPath, basePath)
		processedContent, err := engine.ProcessMarkdownWithLayout(layoutFile, string(content), title)
		if err != nil {
			logger.Warn("Failed to apply layout %s: %v, serving raw HTML", layoutFile, err)
			processedContent = templating.ConvertMarkdownToHTML(string(content))
		}
		return processedContent, responseTypeTemplatedHTML, title
	}

	engine := templating.NewEngineWithBasePath(folderPath, basePath)
	processedContent := engine.ProcessMarkdownToHTML(string(content))
	return processedContent, responseTypeMarkdownHTML, title
}

// processHTMLContent processes HTML content with optional layout.
func (s *Server) processHTMLContent(content []byte, layoutFile, folderPath, basePath string) (string, string, string) {
	htmlContent := string(content)
	title := templating.ExtractTitleFromHTML(htmlContent)

	if layoutFile != "" {
		engine := templating.NewEngineWithBasePath(folderPath, basePath)
		processedContent, err := engine.ProcessLayout(layoutFile, htmlContent, title)
		if err != nil {
			logger.Warn("Failed to apply layout %s: %v, serving raw HTML", layoutFile, err)
			return htmlContent, contentTypeHTML, title
		}
		return processedContent, responseTypeTemplatedHTML, title
	}

	return htmlContent, contentTypeHTML, title
}

// listFolderContents lists the contents of a folder.
func (s *Server) listFolderContents(folderPath string, markdownOnly bool, responseType string) (json.RawMessage, error) {
	// Check if folder exists
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		response := map[string]any{
			"code": 404,
			"response": map[string]any{
				"status":  "error",
				"message": "Folder not found",
			},
		}
		return json.Marshal(response)
	}

	// Read folder contents
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read folder: %w", err)
	}

	// Pre-allocate slice with approximate size
	fileList := make([]map[string]any, 0, len(files))
	for _, file := range files {
		// Skip directories for now
		if file.IsDir() {
			continue
		}

		fileName := file.Name()
		fileExt := strings.ToLower(filepath.Ext(fileName))

		// If markdown only mode, filter for markdown files
		if markdownOnly && fileExt != ".md" && fileExt != ".markdown" {
			continue
		}

		// If response type is markdown-html, prioritize markdown files
		if responseType == responseTypeMarkdownHTML && fileExt != ".md" && fileExt != ".markdown" {
			continue
		}

		// Get file info for size and modification time
		fileInfo, err := file.Info()
		if err != nil {
			// Skip files we can't get info for
			continue
		}

		fileItem := map[string]any{
			"name":     fileName,
			"size":     fileInfo.Size(),
			"modified": fileInfo.ModTime().Format(time.RFC3339),
			"type":     getFileType(fileExt),
		}

		fileList = append(fileList, fileItem)
	}

	response := map[string]any{
		"code": 200,
		"response": map[string]any{
			"status": "success",
			"data": map[string]any{
				"folder": folderPath,
				"files":  fileList,
				"count":  len(fileList),
			},
		},
	}

	return json.Marshal(response)
}
