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

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/daison12006013/turboscript/internal/performance"
	"github.com/valyala/fasthttp"
)

// routeHandler is the main handler that processes all incoming HTTP requests.
func (s *Server) routeHandler(ctx *fasthttp.RequestCtx) {
	// Panic recovery for individual requests
	defer func() {
		if r := recover(); r != nil {
			logger.Error("🚨 REQUEST PANIC RECOVERED: %v", r)
			logger.Error("Request: %s %s", ctx.Method(), ctx.Path())
			s.sendPanicErrorResponse(ctx)
		}
	}()

	// Start performance monitoring for this request if enabled
	var perfCtx *performance.RequestContext
	if s.cfg.Monitoring {
		perfCtx = performance.NewRequestContext(string(ctx.Method()), string(ctx.Path()))
		defer func() {
			perfCtx.Finish(ctx.Response.StatusCode())
		}()
	}

	// Find the matching endpoint
	ep := s.findMatchingEndpointWithUpgrade(string(ctx.Path()), string(ctx.Method()), ctx)
	if ep == nil {
		s.handleNotFound(ctx, perfCtx)
		return
	}

	s.handleEndpoint(ctx, *ep, perfCtx)
}

// compilePathPattern converts a path like "/users/{uid}" or "/demo/*" to a regex and extracts parameter names.
func (s *Server) compilePathPattern(path string) (*regexp.Regexp, []string) {
	pattern := path

	// Find all path parameters in curly braces
	paramRegex := regexp.MustCompile(`\{([^}]+)\}`)
	matches := paramRegex.FindAllStringSubmatch(path, -1)

	// Pre-allocate params slice with known capacity
	params := make([]string, 0, len(matches))

	for _, match := range matches {
		fullParam := match[1]
		paramName := fullParam
		replacementPattern := "([^/]+)" // Default: match non-slash characters

		// Check if parameter has a custom pattern (e.g., {file:.*})
		if strings.Contains(fullParam, ":") {
			parts := strings.SplitN(fullParam, ":", 2)
			paramName = parts[0]
			customPattern := parts[1]

			// Handle common patterns
			switch customPattern {
			case ".*":
				replacementPattern = "(.*)" // Match everything including slashes
			default:
				replacementPattern = "(" + customPattern + ")"
			}
		}

		params = append(params, paramName)
		// Replace {param} or {param:pattern} with the appropriate regex group
		pattern = strings.Replace(pattern, match[0], replacementPattern, 1)
	}

	// Handle wildcard routing (/*) - this should come after parameter processing
	if strings.HasSuffix(path, "/*") {
		// Add a wildcard parameter to capture the remaining path
		params = append(params, "wildcard")
		// Replace /* with a regex that captures everything after the base path
		// Make the trailing slash optional to handle both /demo and /demo/
		basePath := strings.TrimSuffix(pattern, "/*")
		pattern = basePath + "(?:/(.*))?"
	}

	// Anchor the pattern
	pattern = "^" + pattern + "$"

	compiled, _ := regexp.Compile(pattern)
	return compiled, params
}

// extractPathParams extracts path parameter values from a URL using the compiled regex.
func (s *Server) extractPathParams(path string, url string) map[string]string {
	params := make(map[string]string)

	pattern := s.pathPatterns[path]
	paramNames := s.pathParams[path]

	if pattern == nil {
		return params
	}

	matches := pattern.FindStringSubmatch(url)
	if len(matches) > 1 {
		for i, paramName := range paramNames {
			if i+1 < len(matches) {
				params[paramName] = matches[i+1]
			}
		}
	}

	return params
}

// FindMatchingEndpoint finds the endpoint that matches the request URL.
func (s *Server) FindMatchingEndpoint(requestURL string, method string) *config.EndpointConfig {
	for _, ep := range s.cfg.Endpoints {
		if ep.Method == method {
			if matchedEp := s.matchEndpoint(ep, requestURL); matchedEp != nil {
				return matchedEp
			}
		}
	}
	return nil
}

// findMatchingEndpointWithUpgrade finds an endpoint that matches the URL and handles WebSocket upgrades.
func (s *Server) findMatchingEndpointWithUpgrade(requestURL string, method string, ctx *fasthttp.RequestCtx) *config.EndpointConfig {
	// First, try normal method matching
	for _, ep := range s.cfg.Endpoints {
		if ep.Method == method {
			if matchedEp := s.matchEndpoint(ep, requestURL); matchedEp != nil {
				return matchedEp
			}
		}
	}

	// If it's a GET request with WebSocket upgrade headers, check for WebSocket endpoints
	if method == "GET" && s.isWebSocketUpgrade(ctx) {
		for _, ep := range s.cfg.Endpoints {
			if ep.Method == "WebSocket" {
				if matchedEp := s.matchEndpoint(ep, requestURL); matchedEp != nil {
					return matchedEp
				}
			}
		}
	}

	// If it's a GET request with SSE headers, check for SSE endpoints
	if method == "GET" && s.isSSERequest(ctx) {
		for _, ep := range s.cfg.Endpoints {
			if ep.Method == "SSE" {
				if matchedEp := s.matchEndpoint(ep, requestURL); matchedEp != nil {
					return matchedEp
				}
			}
		}
	}

	return nil
}

// isWebSocketUpgrade checks if the request is a WebSocket upgrade request.
func (s *Server) isWebSocketUpgrade(ctx *fasthttp.RequestCtx) bool {
	connection := string(ctx.Request.Header.Peek("Connection"))
	upgrade := string(ctx.Request.Header.Peek("Upgrade"))

	return strings.ToLower(connection) == "upgrade" && strings.ToLower(upgrade) == "websocket"
}

// isSSERequest checks if the request is an SSE request.
func (s *Server) isSSERequest(ctx *fasthttp.RequestCtx) bool {
	accept := string(ctx.Request.Header.Peek("Accept"))
	return strings.Contains(accept, "text/event-stream")
}

// matchEndpoint checks if an endpoint matches the request URL and returns the resolved endpoint.
func (s *Server) matchEndpoint(ep config.EndpointConfig, requestURL string) *config.EndpointConfig {
	pattern := s.pathPatterns[ep.Route]
	if pattern == nil || !pattern.MatchString(requestURL) {
		return nil
	}

	// Check if this is a special endpoint type that doesn't need file resolution
	endpointType := ep.GetType()
	if endpointType == "hybrid" || endpointType == "markdown-html" {
		// For special endpoint types, return the endpoint as-is and let type-based routing handle it
		return &ep
	}

	// For wildcard routes, resolve the actual file path
	if strings.HasSuffix(ep.Route, "/*") {
		return s.resolveWildcardEndpoint(ep, requestURL)
	}

	// For non-wildcard routes with parameters, resolve dynamic parameters
	if strings.Contains(ep.Route, "{") && strings.Contains(ep.Path, "{") {
		return s.resolveParameterizedEndpoint(ep, requestURL)
	}

	return &ep
}

// resolveWildcardEndpoint resolves a wildcard endpoint to a specific file path.
func (s *Server) resolveWildcardEndpoint(ep config.EndpointConfig, requestURL string) *config.EndpointConfig {
	// Extract all path parameters (including wildcard and dynamic parameters)
	params := s.extractPathParams(ep.Route, requestURL)
	wildcardPath, exists := params["wildcard"]
	if !exists {
		logger.Error("Failed to extract wildcard path from URL: %s", requestURL)
		return nil
	}

	// Remove query parameters and fragments from the wildcard path
	if queryIndex := strings.Index(wildcardPath, "?"); queryIndex != -1 {
		wildcardPath = wildcardPath[:queryIndex]
	}
	if fragmentIndex := strings.Index(wildcardPath, "#"); fragmentIndex != -1 {
		wildcardPath = wildcardPath[:fragmentIndex]
	}

	// Start with the endpoint path and replace dynamic parameters
	resolvedPath := ep.Path

	// Replace dynamic path parameters in the file path
	for paramName, paramValue := range params {
		if paramName != "wildcard" {
			placeholder := "{" + paramName + "}"
			resolvedPath = strings.ReplaceAll(resolvedPath, placeholder, paramValue)
		}
	}

	// Construct the file path by replacing the wildcard in the resolved path
	basePath := strings.TrimSuffix(resolvedPath, "/*")

	// Determine the file extension to look for
	fileExtensions := []string{".ts", ".js"}

	// Try each extension to find the file
	for _, ext := range fileExtensions {
		var filePath string
		if wildcardPath == "" {
			// Root wildcard request - look for index file
			filePath = filepath.Join(basePath, "index"+ext)
		} else {
			// Specific file request
			filePath = filepath.Join(basePath, wildcardPath+ext)
		}

		// Clean the path to prevent directory traversal
		filePath = filepath.Clean(filePath)

		// Ensure the resolved path is still within the resolved base path
		// For dynamic paths, we need to check against the resolved base path
		safeBasePath := filepath.Clean(basePath)
		if !strings.HasPrefix(filePath, safeBasePath) {
			logger.Error("Security: Wildcard path traversal attempt blocked: %s -> %s", requestURL, filePath)
			return nil
		}

		// Check if the file exists
		if s.fileExists(filePath) {
			// Create a new endpoint config with the resolved path
			resolvedEp := ep
			resolvedEp.Path = filePath

			logger.Debug("Wildcard route resolved: %s -> %s (dynamic path: %s)", requestURL, filePath, basePath)
			return &resolvedEp
		}
	}

	logger.Debug("No file found for wildcard route: %s (looked for: %s in %s)", requestURL, wildcardPath, basePath)
	return nil
}

// resolveParameterizedEndpoint resolves a parameterized endpoint to a specific file path.
func (s *Server) resolveParameterizedEndpoint(ep config.EndpointConfig, requestURL string) *config.EndpointConfig {
	// Extract all path parameters
	params := s.extractPathParams(ep.Route, requestURL)
	if len(params) == 0 {
		logger.Error("Failed to extract parameters from URL: %s", requestURL)
		return nil
	}

	// Start with the endpoint path and replace dynamic parameters
	resolvedPath := ep.Path

	// Replace dynamic path parameters in the file path
	for paramName, paramValue := range params {
		// Security check: ensure parameter values don't contain dangerous patterns
		if strings.Contains(paramValue, "..") || strings.Contains(paramValue, "/") {
			logger.Error("Security: Parameter contains unsafe characters: %s -> %s", requestURL, paramValue)
			return nil
		}

		placeholder := "{" + paramName + "}"
		resolvedPath = strings.ReplaceAll(resolvedPath, placeholder, paramValue)
	}

	// Determine the file extension to look for
	fileExtensions := []string{".ts", ".js"}

	// Try each extension to find the file
	for _, ext := range fileExtensions {
		filePath := resolvedPath + ext

		// Clean the path to prevent directory traversal
		filePath = filepath.Clean(filePath)

		// Check if the file exists
		if s.fileExists(filePath) {
			// Create a new endpoint config with the resolved path
			resolvedEp := ep
			resolvedEp.Path = filePath

			logger.Debug("Parameterized route resolved: %s -> %s", requestURL, filePath)
			return &resolvedEp
		}
	}

	logger.Debug("No file found for parameterized route: %s (looked for: %s)", requestURL, resolvedPath)
	return nil
}

// fileExists checks if a file exists at the given path.
func (s *Server) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// handleEndpoint processes a matched endpoint request.
func (s *Server) handleEndpoint(ctx *fasthttp.RequestCtx, ep config.EndpointConfig, perfCtx *performance.RequestContext) {
	// Panic recovery for endpoint handling
	defer func() {
		if r := recover(); r != nil {
			logger.Error("🚨 ENDPOINT PANIC RECOVERED: %v", r)
			logger.Error("Endpoint: %s %s -> %s", string(ctx.Method()), string(ctx.Path()), ep.Path)

			s.sendError(ctx, "Endpoint Execution Error", fmt.Sprintf("Internal error in endpoint processing: %v", r))
		}
	}()

	logger.Info("Handling %s %s -> %s", string(ctx.Method()), string(ctx.Path()), ep.Path)

	// Parse request and create event
	event, err := s.parseRequestToEvent(ctx, ep, perfCtx)
	if err != nil {
		s.sendJSONParseError(ctx, err)
		return
	}

	// Check endpoint type and route accordingly
	endpointType := ep.GetType()
	logger.Debug("Endpoint type for %s: %s", string(ctx.Path()), endpointType)
	switch endpointType {
	case "websocket":
		// Handle WebSocket endpoints
		logger.Debug("Handling WebSocket endpoint for %s", string(ctx.Path()))
		s.wsManager.HandleWebSocket(ctx, ep)
		return
	case "sse":
		// Handle Server-Sent Events endpoints
		logger.Debug("Handling SSE endpoint for %s", string(ctx.Path()))
		s.sseManager.HandleSSE(ctx, ep)
		return
	case "markdown-html":
		// Handle folder-based endpoints (markdown-html, etc.)
		s.handleFolderEndpoint(ctx, ep, event, perfCtx)
		return
	case "hybrid":
		// Handle Frontend HYBRID endpoints
		logger.Debug("Handling Frontend HYBRID endpoint for %s", string(ctx.Path()))
		s.handleHybridEndpoint(ctx, ep, event, perfCtx)
		return
	case "api":
		// Handle standard API endpoints
		break
	default:
		// Handle unknown types as folder endpoints for now (future extensibility)
		logger.Debug("Unknown endpoint type '%s', checking if it's a folder endpoint", endpointType)
		if ep.IsFolderEndpoint() {
			s.handleFolderEndpoint(ctx, ep, event, perfCtx)
			return
		}
		// Default to API handling for unknown types
	}

	// Execute the handle function directly - use async if available
	if perfCtx != nil {
		perfCtx.StartResponseProcessing()
	}

	// Get an isolated executor from the pool for this request
	executor := s.getExecutor()
	defer s.returnExecutor(executor)

	rawResponseResult, err := executor.ExecuteHandleAutoWithTimeout(ep.Path, event, s.cfg.GetEndpointTimeout(&ep))
	if err != nil {
		logger.Error("1. TypeScript handle function failed for %s: %v", ep.Path, err)
		if perfCtx != nil {
			perfCtx.EndResponseProcessing()
		}
		s.sendDetailedError(ctx, "TypeScript Handle Function Error", err)
		return
	}

	s.handleFinalResponse(ctx, rawResponseResult, perfCtx)
}

// handleNotFound handles 404 responses, using custom handler if configured.
func (s *Server) handleNotFound(ctx *fasthttp.RequestCtx, perfCtx *performance.RequestContext) {
	logger.Debug("NotFoundPage configuration: '%s'", s.cfg.NotFoundPage)

	// Check if a custom 404 page is configured
	if s.cfg.NotFoundPage != "" {
		s.handleCustomNotFound(ctx, perfCtx)
		return
	}

	// Use default 404 response
	s.sendDefaultNotFound(ctx)
}

// handleCustomNotFound processes custom 404 handler.
func (s *Server) handleCustomNotFound(ctx *fasthttp.RequestCtx, perfCtx *performance.RequestContext) {
	logger.Info("Using custom 404 handler: %s", s.cfg.NotFoundPage)

	// Create a fake endpoint config for the 404 handler
	notFoundEndpoint := config.EndpointConfig{
		Route:  string(ctx.Path()),
		Method: string(ctx.Method()),
		Path:   s.cfg.NotFoundPage,
	}

	// Execute the 404 handler
	s.handleEndpoint(ctx, notFoundEndpoint, perfCtx)
}

// sendDefaultNotFound sends the default JSON 404 response.
func (s *Server) sendDefaultNotFound(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusNotFound)
	ctx.SetContentType("application/json")
	if writeErr := s.writeCompressedResponse(ctx, []byte(`{"status":"not_found","message":"Route not found"}`)); writeErr != nil {
		logger.Error("Failed to write not found response: %v", writeErr)
	}
}
