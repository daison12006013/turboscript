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
	"crypto/md5" // #nosec G501 -- MD5 is used for asset versioning, not security
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/daison12006013/turboscript/internal/performance"
	"github.com/valyala/fasthttp"
)

// HybridData represents the data structure passed to hybrid frontend app template.
type HybridData struct {
	Route  string
	Data   string            // JSON-encoded data
	Assets map[string]string // Asset path to versioned URL mapping
}

// SanitizeJSONForHTML sanitizes JSON data to prevent XSS attacks when embedding in HTML.
// It escapes HTML characters that could be used for script injection.
func SanitizeJSONForHTML(jsonData string) string {
	// HTML escape the JSON string to prevent XSS attacks
	// This ensures that any HTML/JS in the JSON won't be executed when embedded in HTML
	return template.JSEscapeString(jsonData)
}

// handleHybridEndpoint handles hybrid frontend endpoints and static assets.
func (s *Server) handleHybridEndpoint(ctx *fasthttp.RequestCtx, ep config.EndpointConfig, event map[string]any, perfCtx *performance.RequestContext) {
	if perfCtx != nil {
		perfCtx.StartResponseProcessing()
		defer perfCtx.EndResponseProcessing()
	}

	requestPath := string(ctx.Path())
	logger.Info("Handling hybrid frontend endpoint: %s", requestPath)

	// Check if this is a request for static assets
	assetsPath := ep.GetOptionString("assets", "assets")
	if strings.Contains(requestPath, "/"+assetsPath+"/") {
		s.handleStaticAsset(ctx, ep, requestPath, assetsPath)
		return
	}

	// Check if this is an API request for JSON data
	if s.isAPIRequest(ctx, ep) {
		s.handleHybridAPIRequest(ctx, ep, event, requestPath)
		return
	}

	// Handle hybrid frontend app (HTML pages) for browser navigation
	s.handleHybridApp(ctx, ep, event, requestPath)
}

// handleStaticAsset serves static assets for hybrid frontend apps.
func (s *Server) handleStaticAsset(ctx *fasthttp.RequestCtx, ep config.EndpointConfig, requestPath, assetsPath string) {
	// Extract the asset file path
	parts := strings.Split(requestPath, "/"+assetsPath+"/")
	if len(parts) < 2 {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	assetFile := parts[1]

	// Check if assetFile is empty (e.g., requesting "/assets/" directly)
	if assetFile == "" {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	assetPath := filepath.Join(ep.Path, assetsPath, assetFile)

	// Security check - ensure file is within the assets folder
	absAssetsFolder, err := filepath.Abs(filepath.Join(ep.Path, assetsPath))
	if err != nil {
		logger.Error("Failed to resolve assets folder path: %v", err)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	absAssetFile, err := filepath.Abs(assetPath)
	if err != nil {
		logger.Error("Failed to resolve asset file path: %v", err)
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	if !strings.HasPrefix(absAssetFile, absAssetsFolder) {
		logger.Error("Security: Asset file outside allowed folder: %s", requestPath)
		ctx.SetStatusCode(fasthttp.StatusForbidden)
		return
	}

	// Check if file exists
	if !s.fileExists(assetPath) {
		logger.Debug("Asset file not found: %s", assetPath)
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	// Read and serve the file
	content, err := os.ReadFile(assetPath) // #nosec G304 -- assetPath is validated by security checks above
	if err != nil {
		logger.Error("Failed to read asset file %s: %v", assetPath, err)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	// Set appropriate content type
	ext := strings.ToLower(filepath.Ext(assetFile))
	switch ext {
	case ".css":
		ctx.SetContentType("text/css")
	case ".js":
		ctx.SetContentType("application/javascript")
	case ".html":
		ctx.SetContentType("text/html")
	case ".json":
		ctx.SetContentType("application/json")
	case ".png":
		ctx.SetContentType("image/png")
	case ".jpg", ".jpeg":
		ctx.SetContentType("image/jpeg")
	case ".gif":
		ctx.SetContentType("image/gif")
	case ".svg":
		ctx.SetContentType("image/svg+xml")
	case ".ico":
		ctx.SetContentType("image/x-icon")
	case ".woff":
		ctx.SetContentType("font/woff")
	case ".woff2":
		ctx.SetContentType("font/woff2")
	case ".ttf":
		ctx.SetContentType("font/ttf")
	case ".eot":
		ctx.SetContentType("application/vnd.ms-fontobject")
	default:
		ctx.SetContentType("application/octet-stream")
	}

	// Set cache headers for static assets
	cacheControl := s.getCacheControlForAssets(ep, "static_assets")
	if cacheControl != "" {
		ctx.Response.Header.Set("Cache-Control", cacheControl)
		if s.cfg.Debug {
			// Add additional headers for development
			ctx.Response.Header.Set("Pragma", "no-cache")
			ctx.Response.Header.Set("Expires", "0")
		}
	} else {
		// Fallback to default behavior if not configured
		if s.cfg.Debug {
			ctx.Response.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
			ctx.Response.Header.Set("Pragma", "no-cache")
			ctx.Response.Header.Set("Expires", "0")
		} else {
			ctx.Response.Header.Set("Cache-Control", "public, max-age=31536000") // 1 year
		}
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	if _, err := ctx.Write(content); err != nil {
		logger.Error("Failed to write static asset response: %v", err)
		return
	}

	logger.Debug("Successfully served static asset: %s", assetPath)
}

// handleHybridApp serves the hybrid frontend application with initial data.
func (s *Server) handleHybridApp(ctx *fasthttp.RequestCtx, ep config.EndpointConfig, event map[string]any, requestPath string) {
	logger.Info("Handling hybrid frontend app for path: %s", requestPath)

	// Determine the data endpoint path for initial data loading
	dataPath := s.determineDataPath(requestPath, ep)

	// Load initial data from the corresponding data endpoint
	initialData, err := s.loadInitialDataForHybrid(dataPath, event)
	if err != nil {
		logger.Error("Failed to load initial data for hybrid frontend: %v", err)
		// Continue with empty data instead of failing
		initialData = map[string]any{
			"error": "Failed to load initial data",
		}
	}

	// Get the app template file path
	appFile := ep.GetOptionString("app", "App.html")
	appPath := filepath.Join(ep.Path, appFile)

	// Check if asset versioning is enabled
	assetVersioning := s.getAssetVersioningSetting(ep)

	// Serve the hybrid frontend app with initial data
	err = s.serveHybridApp(ctx, ep, appPath, requestPath, initialData, assetVersioning)
	if err != nil {
		logger.Error("Failed to serve hybrid frontend app: %v", err)
		s.sendError(ctx, "Hybrid Frontend App Error", err.Error())
		return
	}

	logger.Debug("Successfully served hybrid frontend app for %s", requestPath)
}

// isAPIRequest determines if the request is asking for JSON data rather than HTML.
// This helps distinguish between browser navigation (which needs HTML) and AJAX requests (which need JSON).
func (s *Server) isAPIRequest(ctx *fasthttp.RequestCtx, ep config.EndpointConfig) bool {
	requestPath := string(ctx.Path())

	// Check if this is a direct request to the data path (e.g., /frontend/data/settings)
	// These should always be treated as API requests
	dataFolder := ep.GetOptionString("data", "api")
	baseRoute := strings.TrimSuffix(ep.Route, "/*")
	dataPrefix := fmt.Sprintf("%s/%s/", baseRoute, dataFolder)
	if strings.HasPrefix(requestPath, dataPrefix) {
		return true
	}

	// Check Accept header for JSON preference
	acceptHeader := string(ctx.Request.Header.Peek("Accept"))
	if strings.Contains(acceptHeader, "application/json") {
		// If both JSON and HTML are accepted, check which comes first or has higher priority
		jsonIndex := strings.Index(acceptHeader, "application/json")
		htmlIndex := strings.Index(acceptHeader, "text/html")

		if htmlIndex == -1 || jsonIndex < htmlIndex {
			return true
		}

		// Check for quality values (q=) - JSON with higher quality wins
		// For simplicity, if JSON is explicitly mentioned, treat as API request
		return true
	}

	// Check for common AJAX headers
	if string(ctx.Request.Header.Peek("X-Requested-With")) == "XMLHttpRequest" {
		return true
	}

	// Check for API path prefix (configurable)
	apiPrefix := ep.GetOptionString("api_prefix", "/api")
	if apiPrefix != "" && strings.HasPrefix(requestPath, apiPrefix) {
		return true
	}

	// Check for content-type indicating a non-browser request
	contentType := string(ctx.Request.Header.Peek("Content-Type"))
	return strings.Contains(contentType, "application/json")
}

// handleHybridAPIRequest handles API requests that should return JSON data.
func (s *Server) handleHybridAPIRequest(ctx *fasthttp.RequestCtx, ep config.EndpointConfig, event map[string]any, requestPath string) {
	logger.Info("Handling hybrid frontend API request for path: %s", requestPath)

	// Update the event object with the correct HTTP method
	event["httpMethod"] = string(ctx.Method())

	// For non-GET requests, ensure we have the request body
	if string(ctx.Method()) != "GET" && string(ctx.Method()) != "HEAD" {
		var bodyData any
		if len(ctx.PostBody()) > 0 {
			// Try to parse JSON body
			if err := json.Unmarshal(ctx.PostBody(), &bodyData); err != nil {
				logger.Error("Failed to parse JSON body for Hybrid API: %v", err)
				s.sendError(ctx, "Invalid JSON", "Request body must be valid JSON")
				return
			}
			event["body"] = bodyData
		}
	}

	// Determine the data endpoint path for this API request
	dataPath := s.determineDataPath(requestPath, ep)

	// Check if the data file exists
	if !s.fileExists(dataPath) {
		logger.Error("Data endpoint not found: %s", dataPath)
		s.sendError(ctx, "Not Found", "Data endpoint not found")
		return
	}

	// Get an isolated executor from the pool for this request
	executor := s.getExecutor()
	defer s.returnExecutor(executor)

	// Execute the data endpoint with proper timeout and get the full response
	rawResponseResult, err := executor.ExecuteHandleAutoWithTimeout(dataPath, event, s.cfg.GetEndpointTimeout(&ep))
	if err != nil {
		logger.Error("Failed to execute data endpoint for Hybrid API %s: %v", dataPath, err)
		s.sendError(ctx, "API Error", err.Error())
		return
	}

	// Use the existing unwrapResponseWithType utility to extract response content
	responseContent, statusCode, _, _, err := unwrapResponseWithType(rawResponseResult)
	if err != nil {
		logger.Error("Failed to unwrap API response: %v", err)
		s.sendError(ctx, "Response Error", "Failed to process response")
		return
	}

	// Set response headers
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(statusCode)

	// Return only the response content
	if _, err := ctx.Write(responseContent); err != nil {
		logger.Error("Failed to write API response: %v", err)
		return
	}

	logger.Debug("Successfully served hybrid frontend API response for %s", requestPath)
}

// determineDataPath determines the data endpoint path based on the frontend route.
func (s *Server) determineDataPath(frontendPath string, ep config.EndpointConfig) string {
	// Get the data folder from configuration (defaults to "api")
	dataFolder := ep.GetOptionString("data", "api")

	// Get the base route pattern (e.g., "/frontend/*")
	baseRoute := strings.TrimSuffix(ep.Route, "/*")

	// Check if this is a direct data path request (e.g., /frontend/data/settings)
	dataPrefix := fmt.Sprintf("%s/%s/", baseRoute, dataFolder)
	if strings.HasPrefix(frontendPath, dataPrefix) {
		// This is a direct data request - extract the data path
		// /frontend/data/settings -> settings
		subPath := strings.TrimPrefix(frontendPath, dataPrefix)
		if subPath == "" {
			// Direct request to data folder - use index
			return filepath.Join(ep.Path, dataFolder, "index.ts")
		}

		// Clean up the sub-path and create data endpoint path
		subPath = strings.TrimSuffix(subPath, "/")
		return filepath.Join(ep.Path, dataFolder, subPath+".ts")
	}

	// This is a regular frontend request (e.g., /frontend/settings)
	// Extract the sub-path for data lookup
	subPath := strings.TrimPrefix(frontendPath, baseRoute)
	if subPath == "" || subPath == "/" {
		// Root path - use index endpoint
		return filepath.Join(ep.Path, dataFolder, "index.ts")
	}

	// Clean up the sub-path and create data endpoint path
	subPath = strings.TrimPrefix(subPath, "/")

	// Route frontend paths to their corresponding data endpoints
	// /frontend/settings -> data/settings.ts (based on config)
	// /frontend/about -> data/about.ts (based on config)
	dataPath := filepath.Join(ep.Path, dataFolder, subPath+".ts")

	// Check if the specific data file exists, otherwise use index.ts for HTML requests
	if !s.fileExists(dataPath) {
		// For HTML requests, fall back to index.ts
		// For API requests, we'll let the caller handle the not found case
		return filepath.Join(ep.Path, dataFolder, "index.ts")
	}

	return dataPath
}

// loadInitialDataForHybrid loads initial data from the data endpoint.
func (s *Server) loadInitialDataForHybrid(dataPath string, event map[string]any) (map[string]any, error) {
	// Check if the data file exists
	if !s.fileExists(dataPath) {
		return map[string]any{
			"message": "Data endpoint not found",
			"path":    dataPath,
		}, nil
	}

	// Get an isolated executor from the pool for this request
	executor := s.getExecutor()
	defer s.returnExecutor(executor)

	// Execute the data endpoint to get initial data
	rawResponseResult, err := executor.ExecuteHandleAutoWithTimeout(dataPath, event, s.cfg.GetEndpointTimeout(nil))
	if err != nil {
		return nil, fmt.Errorf("failed to execute data endpoint %s: %w", dataPath, err)
	}

	// Parse the response to extract the data
	var response map[string]any
	if err := json.Unmarshal(rawResponseResult, &response); err != nil {
		return nil, fmt.Errorf("failed to parse data response: %w", err)
	}

	// Extract the actual response data
	if responseData, ok := response["response"]; ok {
		if dataMap, ok := responseData.(map[string]any); ok {
			return dataMap, nil
		}
	}

	// If response structure is different, return the whole response
	return response, nil
}

// serveHybridApp serves the Hybrid application with initial data.
func (s *Server) serveHybridApp(ctx *fasthttp.RequestCtx, ep config.EndpointConfig, appPath, route string, data map[string]any, assetVersioning bool) error {
	// Read the app template file
	templateContent, err := os.ReadFile(appPath) // #nosec G304 -- appPath is derived from endpoint configuration
	if err != nil {
		return fmt.Errorf("failed to read app template: %w", err)
	}

	// Convert data to JSON string for embedding
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	// Sanitize the JSON data to prevent XSS attacks when embedding in HTML
	sanitizedData := SanitizeJSONForHTML(string(dataJSON))

	// Generate asset map for versioning if enabled
	var assetMap map[string]string
	if assetVersioning {
		assetMap = s.generateAssetMap(ep)
		logger.Debug("Generated asset map: %+v", assetMap)
	} else {
		assetMap = make(map[string]string)
		logger.Debug("Asset versioning disabled, using empty asset map")
	}

	// Create template data
	templateData := HybridData{
		Route:  route,
		Data:   sanitizedData,
		Assets: assetMap,
	}
	logger.Debug("Template data: %+v", templateData)

	// Parse and execute the template
	tmpl, err := template.New("hybrid-app").Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse app template: %w", err)
	}

	// Create a buffer to capture the rendered template
	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, templateData); err != nil {
		return fmt.Errorf("failed to execute app template: %w", err)
	}

	// Set response headers
	ctx.SetContentType("text/html; charset=utf-8")

	// Set cache headers for HTML pages
	cacheControl := s.getCacheControlForAssets(ep, "html_pages")
	if cacheControl != "" {
		ctx.Response.Header.Set("Cache-Control", cacheControl)
		if s.cfg.Debug {
			// Add additional headers for development
			ctx.Response.Header.Set("Pragma", "no-cache")
			ctx.Response.Header.Set("Expires", "0")
		}
	} else {
		// Fallback to default behavior if not configured
		if s.cfg.Debug {
			ctx.Response.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
			ctx.Response.Header.Set("Pragma", "no-cache")
			ctx.Response.Header.Set("Expires", "0")
		} else {
			ctx.Response.Header.Set("Cache-Control", "public, max-age=300") // 5 minutes
		}
	}

	ctx.SetStatusCode(fasthttp.StatusOK)

	// Write the rendered HTML
	if _, err := ctx.WriteString(rendered.String()); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}

// getCacheControlForAssets retrieves cache control settings from endpoint configuration.
func (s *Server) getCacheControlForAssets(ep config.EndpointConfig, assetType string) string {
	// Get cache_control configuration from options
	cacheControlConfig, exists := ep.GetOption("cache_control")
	if !exists || cacheControlConfig == nil {
		return ""
	}

	cacheControlMap, ok := cacheControlConfig.(map[string]any)
	if !ok {
		return ""
	}

	// Get the specific asset type configuration
	assetConfig, exists := cacheControlMap[assetType]
	if !exists {
		return ""
	}

	assetConfigMap, ok := assetConfig.(map[string]any)
	if !ok {
		return ""
	}

	// Choose the appropriate setting based on debug mode
	var setting string
	if s.cfg.Debug {
		if dev, exists := assetConfigMap["development"]; exists {
			if devStr, ok := dev.(string); ok {
				setting = devStr
			}
		}
	} else {
		if prod, exists := assetConfigMap["production"]; exists {
			if prodStr, ok := prod.(string); ok {
				setting = prodStr
			}
		}
	}

	return setting
}

// getAssetVersion generates a hash-based version for an asset file.
func (s *Server) getAssetVersion(assetPath string) string {
	// Read file content for hashing
	content, err := os.ReadFile(assetPath) // #nosec G304 -- assetPath is validated by caller
	if err != nil {
		// If we can't read the file, use a timestamp-based fallback
		if stat, statErr := os.Stat(assetPath); statErr == nil {
			return fmt.Sprintf("%d", stat.ModTime().Unix())
		}
		return "1" // fallback version
	}

	// Generate MD5 hash of file content
	hash := md5.Sum(content)            // #nosec G401 -- MD5 is sufficient for asset versioning
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes for shorter URLs
}

// generateAssetMap creates a map of asset paths to their versioned URLs.
func (s *Server) generateAssetMap(ep config.EndpointConfig) map[string]string {
	assetMap := make(map[string]string)

	// Check if asset versioning is enabled
	cacheControlConfig, exists := ep.GetOption("cache_control")
	if !exists {
		logger.Debug("No cache_control configuration found")
		return assetMap
	}

	cacheConfig, ok := cacheControlConfig.(map[string]any)
	if !ok {
		logger.Debug("cache_control is not a map")
		return assetMap
	}

	enableVersioning, exists := cacheConfig["asset_versioning"]
	if !exists {
		logger.Debug("asset_versioning not found in cache_control")
		return assetMap
	}

	enabled, ok := enableVersioning.(bool)
	if !ok || !enabled {
		logger.Debug("asset_versioning is not enabled or not a boolean: %v", enableVersioning)
		return assetMap
	}

	logger.Debug("Asset versioning is enabled")

	// Get assets directory
	assetsPath := ep.GetOptionString("assets", "assets")
	assetDir := filepath.Join(ep.Path, assetsPath)
	logger.Debug("Asset directory: %s", assetDir)

	// Get configurable assets to version from cache_control.versioned_assets
	var assetsToVersion []string
	if versionedAssetsConfig, exists := cacheConfig["versioned_assets"]; exists {
		if versionedAssetsList, ok := versionedAssetsConfig.([]any); ok {
			for _, asset := range versionedAssetsList {
				if assetStr, ok := asset.(string); ok {
					assetsToVersion = append(assetsToVersion, assetStr)
				}
			}
		}
	}

	// Fallback to common assets if no configuration is provided
	if len(assetsToVersion) == 0 {
		assetsToVersion = []string{"styles.css", "app.js"}
		logger.Debug("No versioned_assets configured, using fallback: %v", assetsToVersion)
	} else {
		logger.Debug("Using configured versioned_assets: %v", assetsToVersion)
	}

	for _, asset := range assetsToVersion {
		assetPath := filepath.Join(assetDir, asset)
		logger.Debug("Checking asset: %s", assetPath)
		if s.fileExists(assetPath) {
			version := s.getAssetVersion(assetPath)
			// Create versioned URL: /frontend/assets/styles.css?v=abc123def
			baseRoute := strings.TrimSuffix(ep.Route, "/*")
			versionedURL := fmt.Sprintf("%s/%s/%s?v=%s", baseRoute, assetsPath, asset, version)
			assetMap[asset] = versionedURL
			logger.Debug("Added versioned asset: %s -> %s", asset, versionedURL)
		} else {
			logger.Debug("Asset file not found: %s", assetPath)
		}
	}

	return assetMap
}

// getAssetVersioningSetting retrieves the asset versioning setting from endpoint configuration.
func (s *Server) getAssetVersioningSetting(ep config.EndpointConfig) bool {
	// Get cache_control configuration from options
	cacheControlConfig, exists := ep.GetOption("cache_control")
	if !exists || cacheControlConfig == nil {
		return false
	}

	cacheControlMap, ok := cacheControlConfig.(map[string]any)
	if !ok {
		return false
	}

	// Get the asset_versioning setting
	assetVersioning, exists := cacheControlMap["asset_versioning"]
	if !exists {
		return false
	}

	if versioningBool, ok := assetVersioning.(bool); ok {
		return versioningBool
	}

	return false
}
