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

// Package server provides the HTTP server implementation for TurboScript.
//
// This package implements a high-performance HTTP server using FastHTTP that handles
// routing, request processing, and integration with the TypeScript execution engine.
// It provides the core runtime environment for TurboScript applications.
//
// Key Features:
//   - Dynamic routing based on configuration
//   - TypeScript code execution via JavaScript VM
//   - Database query execution with security restrictions
//   - Request/response transformation
//   - Error handling and logging
//   - Performance monitoring integration
//
// The server acts as a bridge between HTTP requests and TypeScript business logic,
// executing TypeScript functions in a secure sandboxed environment and managing
// database operations through a controlled interface.
package server

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/email"
	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/daison12006013/turboscript/internal/performance"
	"github.com/daison12006013/turboscript/internal/tsengine"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

// Constants for content types and file extensions.
const (
	contentTypeHTML     = "html"
	contentTypeMarkdown = "markdown"
	contentTypeText     = "text"
	fileExtMD           = ".md"
	fileExtMarkdown     = ".markdown"
	fileExtHTML         = ".html"
	fileExtHTM          = ".htm"
)

// autoDetectResponseType attempts to detect the response type from the content.
func autoDetectResponseType(responseRaw json.RawMessage) string {
	// Try to parse as string first
	var stringContent string
	if err := json.Unmarshal(responseRaw, &stringContent); err == nil {
		// Check if it looks like HTML
		if strings.HasPrefix(strings.TrimSpace(strings.ToLower(stringContent)), "<html") ||
			strings.Contains(strings.ToLower(stringContent), "<html>") ||
			strings.Contains(strings.ToLower(stringContent), "<!doctype html") {
			return contentTypeHTML
		}

		// Check if it looks like markdown (simple heuristics)
		if strings.Contains(stringContent, "# ") ||
			strings.Contains(stringContent, "## ") ||
			strings.Contains(stringContent, "```") ||
			strings.Contains(stringContent, "* ") {
			return contentTypeMarkdown
		}

		// If it's a string but doesn't match HTML or markdown patterns, consider it text
		return contentTypeText
	}

	// If it's not a string, it's likely JSON
	return tsengine.JSONFormat
}

// unwrapResponseWithType extracts the "response" field from a JSON wrapper
// while preserving the original key ordering, handling cookies, and detecting response type.
func unwrapResponseWithType(wrappedJSON []byte) ([]byte, int, []string, string, error) {
	// Parse as raw JSON to preserve key ordering
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(wrappedJSON, &wrapper); err != nil {
		return nil, 0, nil, "", err
	}

	// Extract the code
	code := 200 // default
	if codeRaw, exists := wrapper["code"]; exists {
		var codeValue any
		if err := json.Unmarshal(codeRaw, &codeValue); err == nil {
			switch v := codeValue.(type) {
			case int:
				code = v
			case float64:
				code = int(v)
			case int64:
				code = int(v)
			}
		}
	}

	// Extract cookies
	var cookies []string
	if cookiesRaw, exists := wrapper["cookies"]; exists {
		var cookieArray []string
		if err := json.Unmarshal(cookiesRaw, &cookieArray); err == nil {
			cookies = cookieArray
		}
	}

	// Extract response type
	responseType := tsengine.JSONFormat // default
	if typeRaw, exists := wrapper["type"]; exists {
		var typeValue string
		if err := json.Unmarshal(typeRaw, &typeValue); err == nil {
			responseType = typeValue
		}
	}

	// Auto-detect response type if not explicitly set
	if responseType == tsengine.JSONFormat {
		if responseRaw, exists := wrapper["response"]; exists {
			responseType = autoDetectResponseType(responseRaw)
		}
	}

	// Extract the response while preserving ordering
	if responseRaw, exists := wrapper["response"]; exists {
		return responseRaw, code, cookies, responseType, nil
	}

	return nil, code, nil, responseType, fmt.Errorf("no response field found in wrapper")
}

// Server represents the main HTTP server instance for TurboScript.
//
// It handles incoming HTTP requests, routes them to appropriate TypeScript handlers,
// manages database connections, and provides performance monitoring capabilities.
type Server struct {
	cfg          *config.Config
	pathPatterns map[string]*regexp.Regexp
	pathParams   map[string][]string
	executorPool chan *tsengine.TSExecutor // Pool of isolated executors for request handling
	dbManager    *config.DatabaseManager   // Database manager for multiple connections
	jobManager   any                       // Job manager for background processing
	emailService *email.Service            // Email service for sending emails
	wsManager    *WebSocketManager         // WebSocket connection manager
	sseManager   *SSEManager               // Server-Sent Events manager
}

// NewServerWithServices creates a new HTTP server instance with job manager and email service.
//
// This function initializes the server with full isolation support for concurrent request processing.
func NewServerWithServices(cfg *config.Config, dbManager *config.DatabaseManager, jobManager any, emailService any) *Server {
	// Create a pool of isolated executors for handling HTTP requests
	// This prevents interference between concurrent request processing
	poolSize := cfg.Server.PoolSize
	executorPool := make(chan *tsengine.TSExecutor, poolSize)

	// Create file resolver based on configuration
	fileResolver := tsengine.GetResolverFromConfig(cfg.PreferTS, cfg.PreferJS)

	// Pre-populate the pool with isolated executors
	for i := 0; i < poolSize; i++ {
		isolatedExecutor := tsengine.NewIsolatedTSExecutorWithResolverAndConfig(cfg.PreserveResponse, fileResolver, &cfg.TypeScript)
		isolatedExecutor.SetDatabaseManager(dbManager)

		// Also set the default database for backward compatibility
		if defaultDB, err := dbManager.GetDefaultConnection(); err == nil {
			isolatedExecutor.SetDatabase(defaultDB)
		}

		// Set cache configuration for turboCache operations
		isolatedExecutor.SetCacheConfig(&cfg.Cache)

		// Set markdown base path to app/routes for static file access
		isolatedExecutor.SetMarkdownBasePath("app/routes")

		// Set up job manager if provided
		if jobManager != nil {
			if jm, ok := jobManager.(interface {
				DispatchJob(string, map[string]any) error
			}); ok {
				isolatedExecutor.SetJobManager(jm)
			}
		}

		// Set up email service if provided
		if emailService != nil {
			isolatedExecutor.SetEmailService(emailService.(*email.Service))
		}

		executorPool <- isolatedExecutor
	}

	var emailSvc *email.Service
	if emailService != nil {
		emailSvc = emailService.(*email.Service)
	}

	server := &Server{
		cfg:          cfg,
		executorPool: executorPool, // Pool of isolated executors
		dbManager:    dbManager,    // Store database manager reference
		jobManager:   jobManager,   // Store job manager reference
		emailService: emailSvc,     // Store email service reference
		pathPatterns: make(map[string]*regexp.Regexp),
		pathParams:   make(map[string][]string),
	}

	// Initialize WebSocket manager
	server.wsManager = server.initializeWebSocketManager()

	// Initialize SSE manager
	server.sseManager = server.initializeSSEManager()

	// Set server reference in all executors for broadcasting capabilities
	server.updateExecutorsWithServerReference()

	// Pre-compile path patterns for each endpoint
	for _, ep := range cfg.Endpoints {
		pattern, params := server.compilePathPattern(ep.Route)
		server.pathPatterns[ep.Route] = pattern
		server.pathParams[ep.Route] = params
	}

	return server
}

// Start initializes and starts the HTTP server.
//
// This method sets up performance monitoring if enabled, compiles route patterns,
// and starts the FastHTTP server listening on the configured port.
func (s *Server) Start() {
	// Start performance monitoring if enabled
	if s.cfg.Monitoring {
		logger.Info("[MONITORING] Performance monitoring enabled - starting profiling and metrics")
		performance.StartProfileServer(s.cfg.Server.ProfilerPort)
		performance.LogProfilingInstructionsWithPort(s.cfg.Server.ProfilerPort)

		// Start periodic metrics logging
		performance.StartPeriodicMetrics(time.Duration(s.cfg.Server.PerformanceInterval) * time.Second)
	} else {
		logger.Info("Performance monitoring disabled")
	}

	// Create fasthttp router
	r := router.New()

	// Add a catch-all route that handles all requests
	r.NotFound = s.routeHandler

	logger.Info("Server starting on port %d", s.cfg.Port)
	logger.Error("Server failed to start: %v", fasthttp.ListenAndServe(fmt.Sprintf(":%d", s.cfg.Port), r.Handler))
}

// cleanHTMLResponse removes unwanted whitespace, newlines, and tabs from HTML content
// for optimized response size and performance.
func cleanHTMLResponse(html string) string {
	// Remove carriage returns
	html = strings.ReplaceAll(html, "\r", "")
	// Remove tabs
	html = strings.ReplaceAll(html, "\t", "")
	// Remove newlines
	html = strings.ReplaceAll(html, "\n", "")
	// Replace multiple consecutive spaces with single space
	for strings.Contains(html, "  ") {
		html = strings.ReplaceAll(html, "  ", " ")
	}
	// Trim leading and trailing whitespace
	return strings.TrimSpace(html)
}

// getFileType returns a descriptive file type based on extension.
func getFileType(ext string) string {
	switch ext {
	case fileExtMD, fileExtMarkdown:
		return contentTypeMarkdown
	case fileExtHTML, fileExtHTM:
		return contentTypeHTML
	case ".txt":
		return contentTypeText
	case ".json":
		return tsengine.JSONFormat
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".css":
		return "css"
	case ".scss", ".sass":
		return "sass"
	case ".yml", ".yaml":
		return "yaml"
	default:
		return "file"
	}
}

// initializeWebSocketManager creates and configures the WebSocket manager.
func (s *Server) initializeWebSocketManager() *WebSocketManager {
	var channels []config.WebSocketChannelConfig
	var wsConfig *config.WebSocketConfig

	// Collect WebSocket channels from endpoints
	for _, ep := range s.cfg.Endpoints {
		logger.Debug("Checking endpoint %s: method=%s type=%s isWebSocket=%v hasWebSocketConfig=%v",
			ep.Route, ep.Method, ep.GetType(), ep.IsWebSocketEndpoint(), ep.WebSocket != nil)

		if ep.IsWebSocketEndpoint() && ep.WebSocket != nil {
			logger.Debug("Found WebSocket endpoint %s with %d channels", ep.Route, len(ep.WebSocket.Channels))
			for i, ch := range ep.WebSocket.Channels {
				logger.Debug("Channel %d: room=%s type=%s handle=%s maxConnections=%d",
					i, ch.Room, ch.Type, ch.Handle, ch.MaxConnections)
			}
			channels = append(channels, ep.WebSocket.Channels...)
			if wsConfig == nil {
				wsConfig = ep.WebSocket // Use first WebSocket config as global config
			}
		}
	}

	// If no WebSocket configuration found, log warning
	if len(channels) == 0 || wsConfig == nil {
		logger.Warn("No WebSocket configuration found in endpoints. WebSocket functionality will be disabled.")
		return NewWebSocketManager(s, []config.WebSocketChannelConfig{}, nil)
	}

	logger.Debug("Initialized WebSocket manager with %d total channels", len(channels))

	return NewWebSocketManager(s, channels, wsConfig)
}

// initializeSSEManager creates and configures the SSE manager.
func (s *Server) initializeSSEManager() *SSEManager {
	var sseConfig *config.SSEConfig

	// Find SSE configuration from endpoints
	for _, ep := range s.cfg.Endpoints {
		if ep.IsSSEEndpoint() && ep.SSE != nil {
			sseConfig = ep.SSE
			break // Use first SSE config as global config
		}
	}

	return NewSSEManager(s, sseConfig)
}

// GetWebSocketManager returns the WebSocket manager instance.
func (s *Server) GetWebSocketManager() *WebSocketManager {
	return s.wsManager
}

// GetSSEManager returns the SSE manager instance.
func (s *Server) GetSSEManager() *SSEManager {
	return s.sseManager
}

// BroadcastToRoom broadcasts a message to all WebSocket connections in a room.
func (s *Server) BroadcastToRoom(roomName string, msgType string, data map[string]interface{}) int {
	if s.wsManager == nil {
		return 0
	}

	// Get room connections count before broadcasting
	connections := s.wsManager.GetRoomConnections(roomName)
	if len(connections) == 0 {
		return 0
	}

	// Create WebSocket message
	wsMsg := WebSocketMessage{
		Type:      msgType,
		Room:      roomName,
		Data:      data,
		Timestamp: time.Now(),
		MessageID: s.wsManager.generateMessageID(),
	}

	// Broadcast to room
	s.wsManager.broadcastToRoom(roomName, wsMsg, "")
	return len(connections)
}

// BroadcastToConnection broadcasts a message to a specific WebSocket connection.
func (s *Server) BroadcastToConnection(connectionID string, msgType string, data map[string]interface{}) int {
	if s.wsManager == nil {
		return 0
	}

	// Find the connection
	var targetConn *WebSocketConnection
	s.wsManager.mutex.RLock()
	if conn, exists := s.wsManager.connections[connectionID]; exists {
		targetConn = conn
	}
	s.wsManager.mutex.RUnlock()

	if targetConn == nil {
		return 0
	}

	// Create WebSocket message
	wsMsg := WebSocketMessage{
		Type:      msgType,
		Room:      targetConn.Room,
		Data:      data,
		Timestamp: time.Now(),
		MessageID: s.wsManager.generateMessageID(),
	}

	// Send via WebSocket manager
	if err := targetConn.Conn.WriteJSON(wsMsg); err != nil {
		return 0
	}
	return 1
}

// BroadcastToAll broadcasts a message to all WebSocket connections.
func (s *Server) BroadcastToAll(msgType string, data map[string]interface{}) int {
	if s.wsManager == nil {
		return 0
	}

	totalConnections := 0
	s.wsManager.mutex.RLock()
	for roomName := range s.wsManager.rooms {
		connections := s.wsManager.GetRoomConnections(roomName)
		totalConnections += len(connections)

		if len(connections) > 0 {
			// Create WebSocket message
			wsMsg := WebSocketMessage{
				Type:      msgType,
				Room:      roomName,
				Data:      data,
				Timestamp: time.Now(),
				MessageID: s.wsManager.generateMessageID(),
			}

			// Broadcast to room
			s.wsManager.broadcastToRoom(roomName, wsMsg, "")
		}
	}
	s.wsManager.mutex.RUnlock()

	return totalConnections
}

// BroadcastSSEToConnection broadcasts an SSE message to a specific connection.
func (s *Server) BroadcastSSEToConnection(connectionID string, event string, data map[string]interface{}, messageID string) int {
	if s.sseManager == nil {
		return 0
	}

	// Use the existing SendToConnection method
	err := s.sseManager.SendToConnection(connectionID, event, data)
	if err != nil {
		return 0
	}
	return 1
}

// BroadcastSSEToUser broadcasts an SSE message to all connections for a specific user.
func (s *Server) BroadcastSSEToUser(userID string, event string, data map[string]interface{}, messageID string) int {
	if s.sseManager == nil {
		return 0
	}

	connectionCount := 0
	s.sseManager.mutex.RLock()
	if userConns, exists := s.sseManager.userConnections[userID]; exists {
		for _, conn := range userConns {
			err := s.sseManager.SendToConnection(conn.ID, event, data)
			if err == nil {
				connectionCount++
			}
		}
	}
	s.sseManager.mutex.RUnlock()

	return connectionCount
}

// BroadcastSSEToAll broadcasts an SSE message to all connections.
func (s *Server) BroadcastSSEToAll(event string, data map[string]interface{}, messageID string) int {
	if s.sseManager == nil {
		return 0
	}

	connectionCount := 0
	s.sseManager.mutex.RLock()
	for connID := range s.sseManager.connections {
		err := s.sseManager.SendToConnection(connID, event, data)
		if err == nil {
			connectionCount++
		}
	}
	s.sseManager.mutex.RUnlock()

	return connectionCount
}

// GetConnectionStats returns statistics about active connections.
func (s *Server) GetConnectionStats(filter string) map[string]interface{} {
	stats := make(map[string]interface{})

	// WebSocket stats
	if s.wsManager != nil {
		wsStats := make(map[string]interface{})
		totalWS := 0
		roomStats := make(map[string]int)

		s.wsManager.mutex.RLock()
		for roomName := range s.wsManager.rooms {
			if filter == "" || roomName == filter {
				connections := s.wsManager.GetRoomConnections(roomName)
				roomStats[roomName] = len(connections)
				totalWS += len(connections)
			}
		}
		s.wsManager.mutex.RUnlock()

		wsStats["total_connections"] = totalWS
		wsStats["rooms"] = roomStats
		stats["websocket"] = wsStats
	}

	// SSE stats
	if s.sseManager != nil {
		sseStats := make(map[string]interface{})
		totalSSE := 0
		userStats := make(map[string]int)

		s.sseManager.mutex.RLock()
		for _, conn := range s.sseManager.connections {
			if filter == "" || conn.UserID == filter {
				userStats[conn.UserID]++
				totalSSE++
			}
		}
		s.sseManager.mutex.RUnlock()

		sseStats["total_connections"] = totalSSE
		sseStats["users"] = userStats
		stats["sse"] = sseStats
	}

	return stats
}

// updateExecutorsWithServerReference updates all executors in the pool with server reference.
func (s *Server) updateExecutorsWithServerReference() {
	// Create a temporary slice to hold all executors
	var executors []*tsengine.TSExecutor

	// Drain the pool
	for {
		select {
		case executor := <-s.executorPool:
			executors = append(executors, executor)
		default:
			// Pool is empty
			goto updateExecutors
		}
	}

updateExecutors:
	// Update each executor with server reference and put back in pool
	for _, executor := range executors {
		executor.SetServer(s)
		s.executorPool <- executor
	}
}
