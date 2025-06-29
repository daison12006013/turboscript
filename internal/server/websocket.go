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
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

// generateInstanceID creates a unique instance ID for this server instance.
func generateInstanceID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		logger.Error("Failed to generate random bytes for instance ID: %v", err)
		// Fallback to timestamp-based ID
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(bytes)[:22] // Remove padding
}

// WebSocketConnection represents a WebSocket connection with metadata.
type WebSocketConnection struct {
	ID          string                 `json:"id"`
	Conn        *websocket.Conn        `json:"-"`
	Room        string                 `json:"room"`
	UserID      string                 `json:"user_id"`
	UserData    map[string]interface{} `json:"user_data"`
	ConnectedAt time.Time              `json:"connected_at"`
	LastPing    time.Time              `json:"last_ping"`
	IsAlive     bool                   `json:"is_alive"`
	RemoteAddr  string                 `json:"remote_addr"`
	UserAgent   string                 `json:"user_agent"`
	writeMutex  sync.Mutex             `json:"-"` // Protects writes to WebSocket connection
}

// WebSocketMessage represents a WebSocket message structure.
type WebSocketMessage struct {
	Type      string                 `json:"type"`       // Message type (join, leave, message, broadcast, etc.)
	Room      string                 `json:"room"`       // Target room
	Data      interface{}            `json:"data"`       // Message payload
	UserID    string                 `json:"user_id"`    // Sender user ID
	MessageID string                 `json:"message_id"` // Unique message ID
	Timestamp time.Time              `json:"timestamp"`  // Message timestamp
	Metadata  map[string]interface{} `json:"metadata"`   // Additional metadata
}

// WebSocketRoom manages connections for a specific room.
type WebSocketRoom struct {
	Name        string                          `json:"name"`
	Type        string                          `json:"type"`        // public, private, presence
	Connections map[string]*WebSocketConnection `json:"connections"` // Connection ID -> Connection
	MaxConns    int                             `json:"max_conns"`   // Maximum connections (0 = unlimited)
	Handler     string                          `json:"handler"`     // TypeScript handler path
	Pattern     *regexp.Regexp                  `json:"-"`           // Compiled regex pattern
	CreatedAt   time.Time                       `json:"created_at"`
	mutex       sync.RWMutex                    `json:"-"`
}

// WebSocketManager manages all WebSocket connections and rooms.
type WebSocketManager struct {
	rooms          map[string]*WebSocketRoom       // Room name -> Room
	connections    map[string]*WebSocketConnection // Connection ID -> Connection
	channels       []config.WebSocketChannelConfig // Channel configurations
	mutex          sync.RWMutex                    // Protects maps
	server         *Server                         // Reference to main server
	pingInterval   time.Duration                   // Ping interval
	pongTimeout    time.Duration                   // Pong timeout
	maxMsgSize     int64                           // Maximum message size
	readBuffer     int                             // Read buffer size
	writeBuffer    int                             // Write buffer size
	enableKafka    bool                            // Kafka integration enabled
	enableRedis    bool                            // Redis integration enabled
	kafkaTopic     string                          // Kafka topic for scaling
	redisChannel   string                          // Redis channel for pub/sub
	compression    bool                            // Enable compression
	compressionLvl int                             // Compression level
	allowedOrigins []string                        // CORS allowed origins
	kafkaManager   *KafkaManager                   // Kafka manager for scaling
	sessionManager *SessionAffinityManager         // Session affinity manager
}

// NewWebSocketManager creates a new WebSocket manager.
func NewWebSocketManager(server *Server, channels []config.WebSocketChannelConfig, wsConfig *config.WebSocketConfig) *WebSocketManager {
	// Set default values using config helper methods
	var pingInterval time.Duration
	var pongTimeout time.Duration
	var maxMsgSize int64
	var readBuffer int
	var writeBuffer int

	if wsConfig != nil {
		pingInterval = time.Duration(wsConfig.GetPingInterval()) * time.Second
		pongTimeout = time.Duration(wsConfig.GetPongTimeout()) * time.Second
		maxMsgSize = wsConfig.GetMaxMessageSize()
		readBuffer = wsConfig.GetReadBufferSize()
		writeBuffer = wsConfig.GetWriteBufferSize()
	} else {
		// Fallback defaults if no config provided
		pingInterval = 30 * time.Second
		pongTimeout = 60 * time.Second
		maxMsgSize = 512 * 1024 // 512KB
		readBuffer = 1024
		writeBuffer = 1024
	}

	manager := &WebSocketManager{
		rooms:          make(map[string]*WebSocketRoom),
		connections:    make(map[string]*WebSocketConnection),
		channels:       channels,
		server:         server,
		pingInterval:   pingInterval,
		pongTimeout:    pongTimeout,
		maxMsgSize:     maxMsgSize,
		readBuffer:     readBuffer,
		writeBuffer:    writeBuffer,
		enableKafka:    wsConfig != nil && wsConfig.EnableKafka,
		enableRedis:    wsConfig != nil && wsConfig.EnableRedis,
		kafkaTopic:     "",
		redisChannel:   "",
		compression:    wsConfig != nil && wsConfig.EnableCompression,
		compressionLvl: 6, // Default compression level
		allowedOrigins: []string{},
		sessionManager: NewSessionAffinityManager(generateInstanceID()),
	}

	if wsConfig != nil {
		manager.kafkaTopic = wsConfig.KafkaTopic
		manager.redisChannel = wsConfig.RedisChannel
		if wsConfig.CompressionLevel > 0 {
			manager.compressionLvl = wsConfig.CompressionLevel
		}
		if len(wsConfig.AllowedOrigins) > 0 {
			manager.allowedOrigins = wsConfig.AllowedOrigins
		}
		// Initialize Kafka manager if enabled
		if wsConfig.EnableKafka && len(wsConfig.KafkaBrokers) > 0 && wsConfig.KafkaTopic != "" {
			manager.kafkaManager = NewKafkaManager(wsConfig.KafkaBrokers, wsConfig.KafkaTopic)
		}
	}

	return manager
}

// StartKafka starts the Kafka integration for cross-instance scaling.
func (wm *WebSocketManager) StartKafka(sseManager *SSEManager) error {
	if wm.kafkaManager == nil {
		return nil // Kafka not configured
	}

	// Set managers for message routing
	wm.kafkaManager.SetManagers(wm, sseManager)

	// Start consuming messages
	return wm.kafkaManager.Start()
}

// StopKafka stops the Kafka integration.
func (wm *WebSocketManager) StopKafka() error {
	if wm.kafkaManager == nil {
		return nil
	}
	return wm.kafkaManager.Stop()
}

// HandleWebSocket handles WebSocket upgrade and connection management.
func (wm *WebSocketManager) HandleWebSocket(ctx *fasthttp.RequestCtx, ep config.EndpointConfig) {
	logger.Info("WebSocket connection attempt from %s", ctx.RemoteAddr())

	// Create upgrader
	upgrader := websocket.FastHTTPUpgrader{
		ReadBufferSize:    wm.readBuffer,
		WriteBufferSize:   wm.writeBuffer,
		EnableCompression: wm.compression,
		CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
			return wm.checkCORS(ctx)
		},
	}

	// Upgrade the connection using fasthttp websocket
	err := upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		wm.handleConnection(conn, ep, ctx)
	})

	if err != nil {
		logger.Error("WebSocket upgrade failed: %v", err)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString("WebSocket upgrade failed")
		return
	}
}

// handleConnection manages an individual WebSocket connection.
func (wm *WebSocketManager) handleConnection(conn *websocket.Conn, ep config.EndpointConfig, ctx *fasthttp.RequestCtx) {
	// Set connection limits
	conn.SetReadLimit(wm.maxMsgSize)

	// Create connection metadata
	connID := wm.generateConnectionID()
	remoteAddr := conn.RemoteAddr().String()
	userAgent := string(ctx.Request.Header.Peek("User-Agent"))

	wsConn := &WebSocketConnection{
		ID:          connID,
		Conn:        conn,
		ConnectedAt: time.Now(),
		LastPing:    time.Now(),
		IsAlive:     true,
		RemoteAddr:  remoteAddr,
		UserAgent:   userAgent,
		UserData:    make(map[string]interface{}),
	}

	// Add to connections map
	wm.mutex.Lock()
	wm.connections[connID] = wsConn
	wm.mutex.Unlock()

	logger.Info("WebSocket connection established: %s from %s", connID, remoteAddr)

	// Set up ping/pong handlers
	conn.SetPongHandler(func(string) error {
		wsConn.LastPing = time.Now()
		wsConn.IsAlive = true
		return nil
	})

	// Start ping ticker
	ticker := time.NewTicker(wm.pingInterval)
	defer ticker.Stop()

	// Handle messages in a separate goroutine
	go wm.handleMessages(wsConn, ep)

	// Ping loop
	for {
		select {
		case <-ticker.C:
			if !wsConn.IsAlive {
				logger.Info("WebSocket connection %s timed out", connID)
				wm.closeConnection(wsConn)
				return
			}
			wsConn.IsAlive = false

			// Protect WebSocket write with mutex
			wsConn.writeMutex.Lock()
			err := conn.WriteMessage(websocket.PingMessage, nil)
			wsConn.writeMutex.Unlock()

			if err != nil {
				logger.Error("Failed to send ping to %s: %v", connID, err)
				wm.closeConnection(wsConn)
				return
			}
		}
	}
}

// handleMessages processes incoming WebSocket messages.
func (wm *WebSocketManager) handleMessages(wsConn *WebSocketConnection, ep config.EndpointConfig) {
	defer wm.closeConnection(wsConn)

	for {
		messageType, message, err := wsConn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket unexpected close for %s: %v", wsConn.ID, err)
			} else {
				logger.Debug("WebSocket connection %s closed: %v", wsConn.ID, err)
			}
			break
		}

		if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
			continue
		}

		logger.Debug("Received WebSocket message from %s: %s", wsConn.ID, string(message))

		// Parse the message
		var wsMsg WebSocketMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			logger.Error("Failed to parse WebSocket message from %s: %v", wsConn.ID, err)
			wm.sendErrorMessage(wsConn, "Invalid message format")
			continue
		}

		// Set message metadata
		wsMsg.UserID = wsConn.UserID
		wsMsg.Timestamp = time.Now()
		if wsMsg.MessageID == "" {
			wsMsg.MessageID = wm.generateMessageID()
		}

		// Handle different message types
		switch wsMsg.Type {
		case "join":
			wm.handleJoinRoom(wsConn, wsMsg, ep)
		case "leave":
			wm.handleLeaveRoom(wsConn, wsMsg)
		case "message":
			wm.handleRoomMessage(wsConn, wsMsg, ep)
		default:
			wm.handleCustomMessage(wsConn, wsMsg, ep)
		}
	}
}

// handleJoinRoom handles room join requests.
func (wm *WebSocketManager) handleJoinRoom(wsConn *WebSocketConnection, msg WebSocketMessage, ep config.EndpointConfig) {
	roomName := msg.Room
	if roomName == "" {
		wm.sendErrorMessage(wsConn, "Room name is required")
		return
	}

	// Find matching channel configuration
	channel := wm.findMatchingChannel(roomName)
	if channel == nil {
		wm.sendErrorMessage(wsConn, "Room not allowed")
		return
	}

	// Get or create room
	room := wm.getOrCreateRoom(roomName, *channel)

	// Check connection limits
	if room.MaxConns > 0 && len(room.Connections) >= room.MaxConns {
		wm.sendErrorMessage(wsConn, "Room is full")
		return
	}

	// Add connection to room
	room.mutex.Lock()
	room.Connections[wsConn.ID] = wsConn
	room.mutex.Unlock()

	wsConn.Room = roomName

	logger.Info("WebSocket %s joined room %s", wsConn.ID, roomName)

	// Send join confirmation
	if err := wm.sendMessage(wsConn, WebSocketMessage{
		Type:      "joined",
		Room:      roomName,
		Data:      map[string]interface{}{"status": "success"},
		Timestamp: time.Now(),
	}); err != nil {
		logger.Error("Failed to send join confirmation to WebSocket %s: %v", wsConn.ID, err)
	}

	// Notify other room members (for presence channels)
	if room.Type == "presence" {
		wm.broadcastToRoom(roomName, WebSocketMessage{
			Type:      "user_joined",
			Room:      roomName,
			UserID:    wsConn.UserID,
			Data:      map[string]interface{}{"user": wsConn.UserData},
			Timestamp: time.Now(),
		}, wsConn.ID)
	}

	// Execute TypeScript handler for join event
	wm.executeHandler(channel.Handle, "join", wsConn, msg)
}

// handleLeaveRoom handles room leave requests.
func (wm *WebSocketManager) handleLeaveRoom(wsConn *WebSocketConnection, msg WebSocketMessage) {
	if wsConn.Room == "" {
		return
	}

	room := wm.getRoom(wsConn.Room)
	if room != nil {
		room.mutex.Lock()
		delete(room.Connections, wsConn.ID)
		room.mutex.Unlock()

		// Notify other room members (for presence channels)
		if room.Type == "presence" {
			wm.broadcastToRoom(wsConn.Room, WebSocketMessage{
				Type:      "user_left",
				Room:      wsConn.Room,
				UserID:    wsConn.UserID,
				Data:      map[string]interface{}{"user": wsConn.UserData},
				Timestamp: time.Now(),
			}, wsConn.ID)
		}
	}

	wsConn.Room = ""
	logger.Info("WebSocket %s left room", wsConn.ID)
}

// handleRoomMessage handles messages sent to a room.
func (wm *WebSocketManager) handleRoomMessage(wsConn *WebSocketConnection, msg WebSocketMessage, ep config.EndpointConfig) {
	if wsConn.Room == "" {
		wm.sendErrorMessage(wsConn, "Not in a room")
		return
	}

	// Find channel configuration
	channel := wm.findMatchingChannel(wsConn.Room)
	if channel == nil {
		wm.sendErrorMessage(wsConn, "Invalid room")
		return
	}

	// Execute TypeScript handler
	wm.executeHandler(channel.Handle, "message", wsConn, msg)

	// Broadcast to room members
	wm.broadcastToRoom(wsConn.Room, msg, "")
}

// handleCustomMessage handles custom message types via TypeScript handlers.
func (wm *WebSocketManager) handleCustomMessage(wsConn *WebSocketConnection, msg WebSocketMessage, ep config.EndpointConfig) {
	// Find channel configuration
	var channel *config.WebSocketChannelConfig
	if wsConn.Room != "" {
		channel = wm.findMatchingChannel(wsConn.Room)
	}

	if channel != nil {
		wm.executeHandler(channel.Handle, msg.Type, wsConn, msg)
	}
}

// executeHandler executes a TypeScript WebSocket handler.
func (wm *WebSocketManager) executeHandler(handlerPath, eventType string, wsConn *WebSocketConnection, msg WebSocketMessage) {
	// Create a serializable connection object for TypeScript (exclude websocket.Conn which can't be serialized)
	serializableConn := map[string]interface{}{
		"id":           wsConn.ID,
		"room":         wsConn.Room,
		"user_id":      wsConn.UserID,
		"user_data":    wsConn.UserData,
		"connected_at": wsConn.ConnectedAt.Format(time.RFC3339),
		"last_ping":    wsConn.LastPing.Format(time.RFC3339),
		"is_alive":     wsConn.IsAlive,
		"remote_addr":  wsConn.RemoteAddr,
		"user_agent":   wsConn.UserAgent,
	}

	// Create proper event context for TypeScript handler
	wsContext := map[string]interface{}{
		"type":       "websocket",
		"eventType":  eventType,
		"connection": serializableConn,
		"message":    msg,
		"room":       wsConn.Room,
	}

	event := map[string]interface{}{
		"headers":         make(map[string]string),
		"queryParameters": make(map[string]string),
		"pathParameters":  make(map[string]string),
		"body":            make(map[string]interface{}),
		"env":             make(map[string]string),
		"context":         wsContext,
	}

	// Get an executor from the pool
	executor := wm.server.getExecutor()
	defer wm.server.returnExecutor(executor)

	// Execute the handler
	_, err := executor.ExecuteHandleAutoWithTimeout(handlerPath, event, 30)
	if err != nil {
		logger.Error("WebSocket handler execution failed for %s: %v", handlerPath, err)
	}
}

// Helper methods

func (wm *WebSocketManager) generateConnectionID() string {
	return fmt.Sprintf("ws_%d_%d", time.Now().UnixNano(), len(wm.connections))
}

func (wm *WebSocketManager) generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// checkCORS checks if the request origin is allowed for WebSocket connections.
func (wm *WebSocketManager) checkCORS(ctx *fasthttp.RequestCtx) bool {
	if len(wm.allowedOrigins) == 0 {
		return true // Allow all origins if none specified
	}

	origin := string(ctx.Request.Header.Peek("Origin"))
	if origin == "" {
		return true // Allow requests without origin header
	}

	for _, allowedOrigin := range wm.allowedOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
	}

	return false
}

func (wm *WebSocketManager) findMatchingChannel(roomName string) *config.WebSocketChannelConfig {
	for _, channel := range wm.channels {
		matched, _ := regexp.MatchString(channel.Room, roomName)
		if matched {
			return &channel
		}
	}
	return nil
}

func (wm *WebSocketManager) getOrCreateRoom(name string, channel config.WebSocketChannelConfig) *WebSocketRoom {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	if room, exists := wm.rooms[name]; exists {
		return room
	}

	pattern, _ := regexp.Compile(channel.Room)
	room := &WebSocketRoom{
		Name:        name,
		Type:        channel.Type,
		Connections: make(map[string]*WebSocketConnection),
		MaxConns:    channel.MaxConnections,
		Handler:     channel.Handle,
		Pattern:     pattern,
		CreatedAt:   time.Now(),
	}

	wm.rooms[name] = room
	return room
}

func (wm *WebSocketManager) getRoom(name string) *WebSocketRoom {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()
	return wm.rooms[name]
}

func (wm *WebSocketManager) sendMessage(wsConn *WebSocketConnection, msg WebSocketMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Protect WebSocket write with mutex
	wsConn.writeMutex.Lock()
	defer wsConn.writeMutex.Unlock()

	return wsConn.Conn.WriteMessage(websocket.TextMessage, data)
}

func (wm *WebSocketManager) sendErrorMessage(wsConn *WebSocketConnection, errorMsg string) {
	msg := WebSocketMessage{
		Type:      "error",
		Data:      map[string]interface{}{"message": errorMsg},
		Timestamp: time.Now(),
	}
	if err := wm.sendMessage(wsConn, msg); err != nil {
		logger.Error("Failed to send leave message to WebSocket %s: %v", wsConn.ID, err)
	}
}

func (wm *WebSocketManager) broadcastToRoom(roomName string, msg WebSocketMessage, excludeConnID string) {
	room := wm.getRoom(roomName)
	if room == nil {
		return
	}

	// Publish to Kafka for multi-instance scaling if enabled
	if wm.kafkaManager != nil {
		if err := wm.kafkaManager.PublishWebSocketMessage(msg); err != nil {
			logger.Error("Failed to publish WebSocket message to Kafka: %v", err)
		}
	}

	// Broadcast to local connections
	wm.broadcastToRoomLocal(roomName, msg, excludeConnID)
}

// broadcastToRoomLocal broadcasts to local connections only (no Kafka).
func (wm *WebSocketManager) broadcastToRoomLocal(roomName string, msg WebSocketMessage, excludeConnID string) {
	room := wm.getRoom(roomName)
	if room == nil {
		return
	}

	room.mutex.RLock()
	defer room.mutex.RUnlock()

	for connID, conn := range room.Connections {
		if connID != excludeConnID {
			if err := wm.sendMessage(conn, msg); err != nil {
				logger.Error("Failed to send message to %s: %v", connID, err)
			}
		}
	}
}

func (wm *WebSocketManager) closeConnection(wsConn *WebSocketConnection) {
	// Leave room if in one
	if wsConn.Room != "" {
		wm.handleLeaveRoom(wsConn, WebSocketMessage{Type: "leave", Room: wsConn.Room})
	}

	// Remove from connections
	wm.mutex.Lock()
	delete(wm.connections, wsConn.ID)
	wm.mutex.Unlock()

	// Close the connection
	if err := wsConn.Conn.Close(); err != nil {
		logger.Error("Failed to close WebSocket connection %s: %v", wsConn.ID, err)
	}
	logger.Info("WebSocket connection %s closed", wsConn.ID)
}

// BroadcastToRoom broadcasts a message to all connections in a room (public API).
func (wm *WebSocketManager) BroadcastToRoom(roomName string, msg interface{}) error {
	wsMsg := WebSocketMessage{
		Type:      "broadcast",
		Room:      roomName,
		Data:      msg,
		Timestamp: time.Now(),
		MessageID: wm.generateMessageID(),
	}

	wm.broadcastToRoom(roomName, wsMsg, "")
	return nil
}

// GetRoomConnections returns the list of connections in a room.
func (wm *WebSocketManager) GetRoomConnections(roomName string) []*WebSocketConnection {
	room := wm.getRoom(roomName)
	if room == nil {
		return nil
	}

	room.mutex.RLock()
	defer room.mutex.RUnlock()

	connections := make([]*WebSocketConnection, 0, len(room.Connections))
	for _, conn := range room.Connections {
		connections = append(connections, conn)
	}
	return connections
}

// GetRoomCount returns the number of connections in a room.
func (wm *WebSocketManager) GetRoomCount(roomName string) int {
	room := wm.getRoom(roomName)
	if room == nil {
		return 0
	}

	room.mutex.RLock()
	defer room.mutex.RUnlock()
	return len(room.Connections)
}
