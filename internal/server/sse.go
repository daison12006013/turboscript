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
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/valyala/fasthttp"
)

// SSEConnection represents a Server-Sent Events connection.
type SSEConnection struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	UserData     map[string]interface{} `json:"user_data"`
	ConnectedAt  time.Time              `json:"connected_at"`
	LastActivity time.Time              `json:"last_activity"`
	RemoteAddr   string                 `json:"remote_addr"`
	UserAgent    string                 `json:"user_agent"`
	Channel      chan SSEMessage        `json:"-"`
	Done         chan bool              `json:"-"`
	IsActive     bool                   `json:"is_active"`
}

// SSEMessage represents a Server-Sent Events message.
type SSEMessage struct {
	ID      string      `json:"id"`      // Message ID
	Event   string      `json:"event"`   // Event type
	Data    interface{} `json:"data"`    // Message data
	Retry   int         `json:"retry"`   // Retry interval in milliseconds
	Comment string      `json:"comment"` // Comment (not sent to client)
	Raw     bool        `json:"raw"`     // Send as raw data without JSON encoding
}

// SSEManager manages all Server-Sent Events connections.
type SSEManager struct {
	connections       map[string]*SSEConnection   // Connection ID -> Connection
	userConnections   map[string][]*SSEConnection // User ID -> Connections
	mutex             sync.RWMutex                // Protects maps
	server            *Server                     // Reference to main server
	keepAliveInterval time.Duration               // Keep-alive interval
	retryInterval     int                         // Default retry interval for clients
	maxConnections    int                         // Maximum concurrent connections
	allowedOrigins    []string                    // CORS allowed origins
	bufferSize        int                         // Buffer size for message channels
	enableCompression bool                        // Enable gzip compression
	enableHTTP2       bool                        // Force HTTP/2
	kafkaManager      *KafkaManager               // Kafka manager for scaling
	sessionManager    *SessionAffinityManager     // Session affinity manager
}

// NewSSEManager creates a new Server-Sent Events manager.
func NewSSEManager(server *Server, sseConfig *config.SSEConfig) *SSEManager {
	// Set default values using config helper methods
	var keepAliveInterval time.Duration
	var retryInterval int
	var maxConnections int
	var bufferSize int
	var allowedOrigins []string

	if sseConfig != nil {
		keepAliveInterval = time.Duration(sseConfig.GetKeepAliveInterval()) * time.Second
		retryInterval = sseConfig.GetRetry()
		maxConnections = sseConfig.MaxConnections
		bufferSize = sseConfig.GetBufferSize()
		allowedOrigins = sseConfig.AllowedOrigins
	} else {
		// Fallback defaults if no config provided
		keepAliveInterval = 30 * time.Second
		retryInterval = 3000 // 3 seconds
		maxConnections = 0   // Unlimited
		bufferSize = 1024
	}

	return &SSEManager{
		connections:       make(map[string]*SSEConnection),
		userConnections:   make(map[string][]*SSEConnection),
		server:            server,
		keepAliveInterval: keepAliveInterval,
		retryInterval:     retryInterval,
		maxConnections:    maxConnections,
		allowedOrigins:    allowedOrigins,
		bufferSize:        bufferSize,
		enableCompression: sseConfig != nil && sseConfig.EnableCompression,
		enableHTTP2:       sseConfig != nil && sseConfig.EnableHTTP2,
		kafkaManager:      nil, // Will be set when Kafka is configured
		sessionManager:    NewSessionAffinityManager(generateInstanceID()),
	}
}

// HandleSSE handles Server-Sent Events endpoint.
func (sm *SSEManager) HandleSSE(ctx *fasthttp.RequestCtx, ep config.EndpointConfig) {
	logger.Info("SSE connection attempt from %s", ctx.RemoteAddr())

	// Check connection limits
	if sm.maxConnections > 0 && len(sm.connections) >= sm.maxConnections {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		ctx.SetBodyString("Maximum connections reached")
		return
	}

	// Check CORS if configured
	if !sm.checkCORS(ctx) {
		ctx.SetStatusCode(fasthttp.StatusForbidden)
		ctx.SetBodyString("Origin not allowed")
		return
	}

	// Set SSE headers
	sm.setSSEHeaders(ctx)

	// Enable streaming mode - this is crucial for SSE in fasthttp
	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		// Create connection inside the stream writer
		connID := sm.generateConnectionID()
		remoteAddr := ctx.RemoteAddr().String()
		userAgent := string(ctx.Request.Header.Peek("User-Agent"))

		sseConn := &SSEConnection{
			ID:           connID,
			ConnectedAt:  time.Now(),
			LastActivity: time.Now(),
			RemoteAddr:   remoteAddr,
			UserAgent:    userAgent,
			Channel:      make(chan SSEMessage, sm.bufferSize),
			Done:         make(chan bool, 1),
			IsActive:     true,
			UserData:     make(map[string]interface{}),
		}

		// Add to connections map
		sm.mutex.Lock()
		sm.connections[connID] = sseConn
		sm.mutex.Unlock()

		logger.Info("SSE connection established: %s from %s", connID, remoteAddr)

		// Execute TypeScript handler for connection establishment and get response
		handlerResponse := sm.executeHandlerWithResponse(ep.Path, "connect", sseConn, ctx)

		// Process handler response and send initial events
		if handlerResponse != nil {
			logger.Debug("Handler returned response for connection %s", sseConn.ID)
			sm.processHandlerResponseStream(sseConn, handlerResponse, w)
		} else {
			logger.Debug("Handler returned nil response for connection %s, using default", sseConn.ID)
			// Send default connection message if handler doesn't return SSE events
			sm.sendMessageStream(sseConn, SSEMessage{
				ID:    sm.generateMessageID(),
				Event: "connected",
				Data:  map[string]interface{}{"connection_id": connID, "retry": sm.retryInterval},
				Retry: sm.retryInterval,
			}, w)
		}

		// Start keep-alive ticker
		ticker := time.NewTicker(sm.keepAliveInterval)
		defer ticker.Stop()

		logger.Debug("Starting SSE connection loop for %s", connID)

		// Handle the connection
		for {
			select {
			case msg := <-sseConn.Channel:
				logger.Debug("Received message from channel for connection %s", connID)
				if err := sm.writeSSEMessageStream(w, msg); err != nil {
					logger.Error("Failed to write SSE message to %s: %v", connID, err)
					sm.closeConnection(sseConn)
					return
				}
				logger.Debug("Successfully wrote SSE message to connection %s", connID)
				sseConn.LastActivity = time.Now()

			case <-ticker.C:
				logger.Debug("Sending keep-alive ping to connection %s", connID)
				// Send keep-alive ping
				pingMsg := SSEMessage{
					Event: "ping",
					Data:  map[string]interface{}{"timestamp": time.Now().Unix()},
				}
				if err := sm.writeSSEMessageStream(w, pingMsg); err != nil {
					logger.Error("Failed to send SSE keep-alive to %s: %v", connID, err)
					sm.closeConnection(sseConn)
					return
				}

			case <-sseConn.Done:
				logger.Info("SSE connection %s finished", connID)
				sm.closeConnection(sseConn)
				return

			case <-ctx.Done():
				logger.Info("SSE connection %s context cancelled", connID)
				sm.closeConnection(sseConn)
				return
			}
		}
	})
}

// setSSEHeaders sets the appropriate headers for Server-Sent Events.
func (sm *SSEManager) setSSEHeaders(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Content-Type", "text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "Cache-Control")

	// Enable compression if configured
	if sm.enableCompression {
		ctx.Response.Header.Set("Content-Encoding", "gzip")
	}

	// Set HTTP/2 headers if enabled
	if sm.enableHTTP2 {
		ctx.Response.Header.Set("HTTP2-Settings", "")
	}
}

// checkCORS checks if the request origin is allowed.
func (sm *SSEManager) checkCORS(ctx *fasthttp.RequestCtx) bool {
	if len(sm.allowedOrigins) == 0 {
		return true // Allow all origins if none specified
	}

	origin := string(ctx.Request.Header.Peek("Origin"))
	if origin == "" {
		return true // Allow requests without origin header
	}

	for _, allowedOrigin := range sm.allowedOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
	}

	return false
}

// writeSSEMessage writes a Server-Sent Events message to the response.
func (sm *SSEManager) writeSSEMessage(ctx *fasthttp.RequestCtx, msg SSEMessage) error {
	// If this is a raw message, write it directly without formatting
	if msg.Raw && msg.Data != nil {
		dataStr := fmt.Sprintf("%v", msg.Data)
		data := []byte(dataStr)
		_, err := ctx.Write(data)
		if err == nil {
			// Force immediate flush to client
			if flusher, ok := ctx.Response.BodyWriter().(interface{ Flush() error }); ok {
				if flushErr := flusher.Flush(); flushErr != nil {
					logger.Error("Failed to flush SSE heartbeat: %v", flushErr)
				}
			}
		}
		return err
	}

	var output strings.Builder

	// Write message ID
	if msg.ID != "" {
		output.WriteString(fmt.Sprintf("id: %s\n", msg.ID))
	}

	// Write event type
	if msg.Event != "" {
		output.WriteString(fmt.Sprintf("event: %s\n", msg.Event))
	}

	// Write retry interval
	if msg.Retry > 0 {
		output.WriteString(fmt.Sprintf("retry: %d\n", msg.Retry))
	}

	// Write data
	if msg.Data != nil {
		var dataStr string
		dataBytes, err := json.Marshal(msg.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal SSE data: %w", err)
		}
		dataStr = string(dataBytes)

		// Handle multi-line data
		lines := strings.Split(dataStr, "\n")
		for _, line := range lines {
			output.WriteString(fmt.Sprintf("data: %s\n", line))
		}
	}

	// End message with double newline
	output.WriteString("\n")

	// Write to response
	data := []byte(output.String())

	_, err := ctx.Write(data)
	if err == nil {
		// Force immediate flush to client
		if flusher, ok := ctx.Response.BodyWriter().(interface{ Flush() error }); ok {
			if flushErr := flusher.Flush(); flushErr != nil {
				logger.Error("Failed to flush SSE response: %v", flushErr)
			}
		}
	}
	return err
}

// writeSSEMessageStream writes an SSE message using a bufio.Writer for proper streaming
func (sm *SSEManager) writeSSEMessageStream(w *bufio.Writer, msg SSEMessage) error {
	// If this is a raw message, write it directly without formatting
	if msg.Raw && msg.Data != nil {
		dataStr := fmt.Sprintf("%v", msg.Data)
		_, err := w.WriteString(dataStr)
		if err == nil {
			if flushErr := w.Flush(); flushErr != nil {
				logger.Error("Failed to flush SSE keep-alive: %v", flushErr)
			}
		}
		return err
	}

	var output strings.Builder

	// Write message ID
	if msg.ID != "" {
		output.WriteString(fmt.Sprintf("id: %s\n", msg.ID))
	}

	// Write event type
	if msg.Event != "" {
		output.WriteString(fmt.Sprintf("event: %s\n", msg.Event))
	}

	// Write retry interval
	if msg.Retry > 0 {
		output.WriteString(fmt.Sprintf("retry: %d\n", msg.Retry))
	}

	// Write data
	if msg.Data != nil {
		var dataStr string
		dataBytes, err := json.Marshal(msg.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal SSE data: %w", err)
		}
		dataStr = string(dataBytes)

		// Handle multi-line data
		lines := strings.Split(dataStr, "\n")
		for _, line := range lines {
			output.WriteString(fmt.Sprintf("data: %s\n", line))
		}
	}

	// End message with double newline
	output.WriteString("\n")

	// Write to stream and flush
	_, err := w.WriteString(output.String())
	if err == nil {
		if flushErr := w.Flush(); flushErr != nil {
			logger.Error("Failed to flush SSE data: %v", flushErr)
		}
	}
	return err
}

// sendMessageStream sends a message through SSE connection using streaming writer
func (sm *SSEManager) sendMessageStream(sseConn *SSEConnection, msg SSEMessage, w *bufio.Writer) {
	logger.Debug("Attempting to send message to connection %s via stream", sseConn.ID)

	if err := sm.writeSSEMessageStream(w, msg); err != nil {
		logger.Error("Failed to write SSE message to %s: %v", sseConn.ID, err)
	} else {
		logger.Debug("Message successfully written to stream for connection %s", sseConn.ID)
	}
}

// processHandlerResponseStream processes handler response for streaming connections
func (sm *SSEManager) processHandlerResponseStream(sseConn *SSEConnection, response map[string]interface{}, w *bufio.Writer) {
	logger.Debug("Processing handler response: %+v", response)

	// Check if there's a direct 'response' field (raw SSE data)
	if rawResponse, exists := response["response"]; exists {
		logger.Debug("Found 'response' field: %v", rawResponse)
		rawMsg := SSEMessage{
			Raw:  true,
			Data: rawResponse,
		}
		sm.sendMessageStream(sseConn, rawMsg, w)
		return
	}

	// Check if there's an 'sse' field with structured SSE event
	if sseField, exists := response["sse"]; exists {
		if sseData, ok := sseField.(map[string]interface{}); ok {
			msg := SSEMessage{}

			if id, exists := sseData["id"]; exists {
				msg.ID = fmt.Sprintf("%v", id)
			} else {
				msg.ID = sm.generateMessageID()
			}

			if event, exists := sseData["event"]; exists {
				msg.Event = fmt.Sprintf("%v", event)
			}

			if data, exists := sseData["data"]; exists {
				msg.Data = data
			}

			if retry, exists := sseData["retry"]; exists {
				if retryInt, ok := retry.(int); ok {
					msg.Retry = retryInt
				}
			}

			sm.sendMessageStream(sseConn, msg, w)
		}
	}
}

// executeHandlerWithResponse executes a TypeScript SSE handler and returns the response.
func (sm *SSEManager) executeHandlerWithResponse(handlerPath, eventType string, sseConn *SSEConnection, ctx *fasthttp.RequestCtx) map[string]interface{} {
	// Create proper event context for TypeScript handler
	sseContext := map[string]interface{}{
		"type":      "sse",
		"eventType": eventType,
		"connection": map[string]interface{}{
			"id":           sseConn.ID,
			"user_id":      sseConn.UserID,
			"connected_at": sseConn.ConnectedAt,
			"remote_addr":  sseConn.RemoteAddr,
		},
		"data": nil,
	}

	// Extract query parameters from request
	queryParams := make(map[string]string)
	ctx.QueryArgs().VisitAll(func(key, value []byte) {
		queryParams[string(key)] = string(value)
	})

	// Extract headers
	headers := make(map[string]string)
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	event := map[string]interface{}{
		"headers":         headers,
		"queryParameters": queryParams,
		"pathParameters":  make(map[string]string),
		"body":            make(map[string]interface{}),
		"env":             make(map[string]string),
		"context":         sseContext,
	}

	// Get an executor from the pool
	executor := sm.server.getExecutor()
	defer sm.server.returnExecutor(executor)

	// Execute the handler and capture response
	result, err := executor.ExecuteHandleAutoWithTimeout(handlerPath, event, 30)
	if err != nil {
		logger.Error("SSE handler execution failed for %s: %v", handlerPath, err)
		return nil
	}

	// Parse JSON response
	var resultMap map[string]interface{}
	if err := json.Unmarshal(result, &resultMap); err != nil {
		logger.Error("Failed to parse SSE handler response: %v", err)
		return nil
	}

	return resultMap
}

// processHandlerResponse processes the TypeScript handler response and sends SSE events.
func (sm *SSEManager) processHandlerResponse(sseConn *SSEConnection, response map[string]interface{}, ctx *fasthttp.RequestCtx) {
	logger.Debug("Processing handler response: %v", response)

	// Check if response contains SSE events
	if responseData, ok := response["response"]; ok {
		logger.Debug("Found 'response' field: %v", responseData)
		if responseStr, isString := responseData.(string); isString {
			// If response is a string, it might be raw SSE data
			if strings.Contains(responseStr, "event:") || strings.Contains(responseStr, "data:") {
				logger.Debug("Sending raw SSE data via channel to connection %s", sseConn.ID)

				// Send raw SSE data through the channel system as a raw message
				sm.sendMessage(sseConn, SSEMessage{
					ID:   sm.generateMessageID(),
					Data: responseStr,
					Raw:  true, // This tells writeSSEMessage to send it as-is
				})
				return
			}
		}
	}

	// Check if response contains structured SSE data
	if sseData, ok := response["sse"]; ok {
		logger.Debug("Found 'sse' field: %v", sseData)
		if sseMap, isMap := sseData.(map[string]interface{}); isMap {
			event := ""
			data := make(map[string]interface{})

			if eventVal, exists := sseMap["event"]; exists {
				if eventStr, isStr := eventVal.(string); isStr {
					event = eventStr
				}
			}

			if dataVal, exists := sseMap["data"]; exists {
				if dataMap, isMap := dataVal.(map[string]interface{}); isMap {
					data = dataMap
				}
			}

			logger.Debug("Sending structured SSE event: %s with data: %v", event, data)

			// Send the SSE message
			sm.sendMessage(sseConn, SSEMessage{
				ID:    sm.generateMessageID(),
				Event: event,
				Data:  data,
				Retry: sm.retryInterval,
			})
		}
	}

	// If no SSE data found, log it
	logger.Debug("No SSE data found in response, connection %s will use default behavior", sseConn.ID)
}

// executeHandler executes a TypeScript SSE handler.
func (sm *SSEManager) executeHandler(handlerPath, eventType string, sseConn *SSEConnection, data interface{}) {
	// Create proper event context for TypeScript handler
	sseContext := map[string]interface{}{
		"type":       "sse",
		"eventType":  eventType,
		"connection": sseConn,
		"data":       data,
	}

	event := map[string]interface{}{
		"headers":         make(map[string]string),
		"queryParameters": make(map[string]string),
		"pathParameters":  make(map[string]string),
		"body":            make(map[string]interface{}),
		"env":             make(map[string]string),
		"context":         sseContext,
	}

	// Get an executor from the pool
	executor := sm.server.getExecutor()
	defer sm.server.returnExecutor(executor)

	// Execute the handler
	_, err := executor.ExecuteHandleAutoWithTimeout(handlerPath, event, 30)
	if err != nil {
		logger.Error("SSE handler execution failed for %s: %v", handlerPath, err)
	}
}

// Helper methods

func (sm *SSEManager) generateConnectionID() string {
	return fmt.Sprintf("sse_%d_%d", time.Now().UnixNano(), len(sm.connections))
}

func (sm *SSEManager) generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

func (sm *SSEManager) sendMessage(sseConn *SSEConnection, msg SSEMessage) {
	if !sseConn.IsActive {
		logger.Debug("Cannot send message to inactive connection %s", sseConn.ID)
		return
	}

	logger.Debug("Attempting to send message to connection %s, channel buffer: %d/%d", sseConn.ID, len(sseConn.Channel), cap(sseConn.Channel))

	select {
	case sseConn.Channel <- msg:
		logger.Debug("Message successfully sent to channel for connection %s", sseConn.ID)
		// Message sent successfully
	default:
		// Channel is full, close connection
		logger.Info("SSE connection %s channel full, closing", sseConn.ID)
		sm.closeConnection(sseConn)
	}
}

func (sm *SSEManager) closeConnection(sseConn *SSEConnection) {
	if !sseConn.IsActive {
		return
	}

	sseConn.IsActive = false
	close(sseConn.Channel)
	sseConn.Done <- true

	// Remove from connections
	sm.mutex.Lock()
	delete(sm.connections, sseConn.ID)

	// Remove from user connections
	if sseConn.UserID != "" {
		userConns := sm.userConnections[sseConn.UserID]
		for i, conn := range userConns {
			if conn.ID == sseConn.ID {
				sm.userConnections[sseConn.UserID] = append(userConns[:i], userConns[i+1:]...)
				break
			}
		}
		if len(sm.userConnections[sseConn.UserID]) == 0 {
			delete(sm.userConnections, sseConn.UserID)
		}
	}
	sm.mutex.Unlock()

	logger.Info("SSE connection %s closed", sseConn.ID)
}

// Public API methods

// SendToConnection sends a message to a specific SSE connection.
func (sm *SSEManager) SendToConnection(connectionID string, event string, data interface{}) error {
	sm.mutex.RLock()
	conn, exists := sm.connections[connectionID]
	sm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("connection %s not found", connectionID)
	}

	msg := SSEMessage{
		ID:    sm.generateMessageID(),
		Event: event,
		Data:  data,
		Retry: sm.retryInterval,
	}

	sm.sendMessage(conn, msg)
	return nil
}

// SendToUser sends a message to all SSE connections for a specific user.
func (sm *SSEManager) SendToUser(userID string, event string, data interface{}) error {
	sm.mutex.RLock()
	userConns, exists := sm.userConnections[userID]
	sm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("no connections found for user %s", userID)
	}

	msg := SSEMessage{
		ID:    sm.generateMessageID(),
		Event: event,
		Data:  data,
		Retry: sm.retryInterval,
	}

	for _, conn := range userConns {
		sm.sendMessage(conn, msg)
	}

	return nil
}

// Broadcast sends a message to all active SSE connections.
func (sm *SSEManager) Broadcast(event string, data interface{}) error {
	// Publish to Kafka for multi-instance scaling if enabled
	if sm.kafkaManager != nil {
		if err := sm.kafkaManager.PublishSSEMessage(event, data, ""); err != nil {
			logger.Error("Failed to publish SSE message to Kafka: %v", err)
		}
	}

	// Broadcast to local connections
	return sm.broadcastLocal(event, data)
}

// broadcastLocal sends a message to all local SSE connections only (no Kafka).
func (sm *SSEManager) broadcastLocal(event string, data interface{}) error {
	sm.mutex.RLock()
	connections := make([]*SSEConnection, 0, len(sm.connections))
	for _, conn := range sm.connections {
		connections = append(connections, conn)
	}
	sm.mutex.RUnlock()

	msg := SSEMessage{
		ID:    sm.generateMessageID(),
		Event: event,
		Data:  data,
		Retry: sm.retryInterval,
	}

	for _, conn := range connections {
		sm.sendMessage(conn, msg)
	}

	return nil
}

// GetConnectionCount returns the total number of active SSE connections.
func (sm *SSEManager) GetConnectionCount() int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return len(sm.connections)
}

// GetUserConnectionCount returns the number of connections for a specific user.
func (sm *SSEManager) GetUserConnectionCount(userID string) int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return len(sm.userConnections[userID])
}

// AssociateUserWithConnection associates a user ID with an SSE connection.
func (sm *SSEManager) AssociateUserWithConnection(connectionID, userID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	conn, exists := sm.connections[connectionID]
	if !exists {
		return fmt.Errorf("connection %s not found", connectionID)
	}

	// Remove from old user if any
	if conn.UserID != "" {
		oldUserConns := sm.userConnections[conn.UserID]
		for i, c := range oldUserConns {
			if c.ID == connectionID {
				sm.userConnections[conn.UserID] = append(oldUserConns[:i], oldUserConns[i+1:]...)
				break
			}
		}
		if len(sm.userConnections[conn.UserID]) == 0 {
			delete(sm.userConnections, conn.UserID)
		}
	}

	// Associate with new user
	conn.UserID = userID
	if sm.userConnections[userID] == nil {
		sm.userConnections[userID] = make([]*SSEConnection, 0)
	}
	sm.userConnections[userID] = append(sm.userConnections[userID], conn)

	return nil
}

// SetKafkaManager sets the Kafka manager for SSE scaling.
func (sm *SSEManager) SetKafkaManager(kafkaManager *KafkaManager) {
	if sm == nil {
		return
	}
	sm.kafkaManager = kafkaManager
}
