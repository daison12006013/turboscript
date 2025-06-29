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
	"sync"
	"testing"
	"time"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/valyala/fasthttp"
)

func TestNewWebSocketManager(t *testing.T) {
	// Create a minimal server for testing
	server := &Server{}

	channels := []config.WebSocketChannelConfig{
		{
			Room:           "public-.*",
			Type:           "public",
			Handle:         "app/websockets/public.ts",
			MaxConnections: 100,
		},
	}

	wsConfig := &config.WebSocketConfig{
		PingInterval:      30,
		PongTimeout:       60,
		MaxMessageSize:    1024 * 1024,
		ReadBufferSize:    2048,
		WriteBufferSize:   2048,
		EnableCompression: true,
		CompressionLevel:  6,
		EnableKafka:       true,
		EnableRedis:       true,
		KafkaTopic:        "websocket-events",
		RedisChannel:      "ws-channel",
	}

	manager := NewWebSocketManager(server, channels, wsConfig)

	// Test manager initialization
	if manager == nil {
		t.Fatal("Expected manager to be created, got nil")
	}
	if manager.server != server {
		t.Errorf("Expected server to be %v, got %v", server, manager.server)
	}
	if len(manager.channels) != len(channels) {
		t.Errorf("Expected %d channels, got %d", len(channels), len(manager.channels))
	}
	if manager.pingInterval != 30*time.Second {
		t.Errorf("Expected ping interval 30s, got %v", manager.pingInterval)
	}
	if manager.pongTimeout != 60*time.Second {
		t.Errorf("Expected pong timeout 60s, got %v", manager.pongTimeout)
	}
	if manager.maxMsgSize != int64(1024*1024) {
		t.Errorf("Expected max message size 1MB, got %d", manager.maxMsgSize)
	}
	if !manager.compression {
		t.Error("Expected compression to be enabled")
	}
	if !manager.enableKafka {
		t.Error("Expected Kafka to be enabled")
	}
	if !manager.enableRedis {
		t.Error("Expected Redis to be enabled")
	}

	// Test maps initialization
	if manager.rooms == nil {
		t.Error("Expected rooms map to be initialized")
	}
	if manager.connections == nil {
		t.Error("Expected connections map to be initialized")
	}
}

func TestNewWebSocketManagerWithDefaults(t *testing.T) {
	server := &Server{}

	manager := NewWebSocketManager(server, nil, nil)

	// Test default values
	if manager == nil {
		t.Fatal("Expected manager to be created, got nil")
	}
	if manager.pingInterval != 30*time.Second {
		t.Errorf("Expected default ping interval 30s, got %v", manager.pingInterval)
	}
	if manager.pongTimeout != 60*time.Second {
		t.Errorf("Expected default pong timeout 60s, got %v", manager.pongTimeout)
	}
	if manager.maxMsgSize != int64(512*1024) {
		t.Errorf("Expected default max message size 512KB, got %d", manager.maxMsgSize)
	}
	if manager.compression {
		t.Error("Expected compression to be disabled by default")
	}
	if manager.enableKafka {
		t.Error("Expected Kafka to be disabled by default")
	}
	if manager.enableRedis {
		t.Error("Expected Redis to be disabled by default")
	}
}

func TestWebSocketManager_GenerateConnectionID(t *testing.T) {
	server := &Server{}
	manager := NewWebSocketManager(server, nil, nil)

	id1 := manager.generateConnectionID()
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	id2 := manager.generateConnectionID()

	if id1 == "" {
		t.Error("Expected non-empty connection ID")
	}
	if id2 == "" {
		t.Error("Expected non-empty connection ID")
	}
	if id1 == id2 {
		t.Error("Expected unique connection IDs")
	}
	if id1[:3] != "ws_" {
		t.Errorf("Expected connection ID to start with 'ws_', got %s", id1)
	}
}

func TestWebSocketManager_GenerateMessageID(t *testing.T) {
	server := &Server{}
	manager := NewWebSocketManager(server, nil, nil)

	id1 := manager.generateMessageID()
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	id2 := manager.generateMessageID()

	if id1 == "" {
		t.Error("Expected non-empty message ID")
	}
	if id2 == "" {
		t.Error("Expected non-empty message ID")
	}
	if id1 == id2 {
		t.Error("Expected unique message IDs")
	}
	if id1[:4] != "msg_" {
		t.Errorf("Expected message ID to start with 'msg_', got %s", id1)
	}
}

func TestWebSocketManager_FindMatchingChannel(t *testing.T) {
	server := &Server{}

	channels := []config.WebSocketChannelConfig{
		{
			Room:   "public-.*",
			Type:   "public",
			Handle: "app/websockets/public.ts",
		},
		{
			Room:   "room-[0-9]+",
			Type:   "private",
			Handle: "app/websockets/private-room.ts",
		},
		{
			Room:   "presence-.*",
			Type:   "presence",
			Handle: "app/websockets/presence.ts",
		},
	}

	manager := NewWebSocketManager(server, channels, nil)

	tests := []struct {
		roomName      string
		expectedFound bool
		expectedType  string
	}{
		{"public-lobby", true, "public"},
		{"public-chat", true, "public"},
		{"room-12345", true, "private"},
		{"room-abc", false, ""},
		{"presence-dashboard", true, "presence"},
		{"invalid-room", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.roomName, func(t *testing.T) {
			channel := manager.findMatchingChannel(tt.roomName)
			if tt.expectedFound {
				if channel == nil {
					t.Errorf("Expected to find channel for %s, got nil", tt.roomName)
				} else if channel.Type != tt.expectedType {
					t.Errorf("Expected channel type %s, got %s", tt.expectedType, channel.Type)
				}
			} else {
				if channel != nil {
					t.Errorf("Expected no channel for %s, got %v", tt.roomName, channel)
				}
			}
		})
	}
}

func TestWebSocketManager_GetOrCreateRoom(t *testing.T) {
	server := &Server{}
	manager := NewWebSocketManager(server, nil, nil)

	channel := config.WebSocketChannelConfig{
		Room:           "public-.*",
		Type:           "public",
		Handle:         "app/websockets/public.ts",
		MaxConnections: 100,
	}

	// Test creating new room
	room1 := manager.getOrCreateRoom("public-lobby", channel)
	if room1 == nil {
		t.Fatal("Expected room to be created, got nil")
	}
	if room1.Name != "public-lobby" {
		t.Errorf("Expected room name 'public-lobby', got %s", room1.Name)
	}
	if room1.Type != "public" {
		t.Errorf("Expected room type 'public', got %s", room1.Type)
	}
	if room1.Handler != "app/websockets/public.ts" {
		t.Errorf("Expected handler 'app/websockets/public.ts', got %s", room1.Handler)
	}
	if room1.MaxConns != 100 {
		t.Errorf("Expected max connections 100, got %d", room1.MaxConns)
	}
	if room1.Connections == nil {
		t.Error("Expected connections map to be initialized")
	}
	if len(room1.Connections) != 0 {
		t.Errorf("Expected empty connections map, got %d connections", len(room1.Connections))
	}

	// Test getting existing room
	room2 := manager.getOrCreateRoom("public-lobby", channel)
	if room1 != room2 {
		t.Error("Expected to get the same room instance")
	}
}

func TestWebSocketManager_CheckCORS(t *testing.T) {
	server := &Server{}

	// Test with no allowed origins (allow all)
	manager := NewWebSocketManager(server, nil, nil)

	// Mock fasthttp request context
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set("Origin", "https://example.com")

	if !manager.checkCORS(ctx) {
		t.Error("Expected CORS to allow all origins when none specified")
	}

	// Test with specific allowed origins
	wsConfig := &config.WebSocketConfig{
		AllowedOrigins: []string{"https://allowed.com", "https://another.com"},
	}
	manager = NewWebSocketManager(server, nil, wsConfig)

	// Test allowed origin
	ctx.Request.Header.Set("Origin", "https://allowed.com")
	if !manager.checkCORS(ctx) {
		t.Error("Expected CORS to allow specified origin")
	}

	// Test disallowed origin
	ctx.Request.Header.Set("Origin", "https://evil.com")
	if manager.checkCORS(ctx) {
		t.Error("Expected CORS to reject non-allowed origin")
	}

	// Test wildcard origin
	wsConfig.AllowedOrigins = []string{"*"}
	manager = NewWebSocketManager(server, nil, wsConfig)
	if !manager.checkCORS(ctx) {
		t.Error("Expected CORS to allow wildcard origin")
	}

	// Test no origin header
	ctx.Request.Header.Del("Origin")
	if !manager.checkCORS(ctx) {
		t.Error("Expected CORS to allow requests without origin header")
	}
}

func TestWebSocketManager_BroadcastToRoom(t *testing.T) {
	server := &Server{}
	manager := NewWebSocketManager(server, nil, nil)

	// Test broadcasting to non-existent room
	err := manager.BroadcastToRoom("non-existent", map[string]string{"message": "hello"})
	if err != nil {
		t.Errorf("Expected no error for non-existent room, got %v", err)
	}

	// Create a room
	channel := config.WebSocketChannelConfig{
		Room:   "test-room",
		Type:   "public",
		Handle: "test.ts",
	}
	room := manager.getOrCreateRoom("test-room", channel)

	// Test broadcasting to empty room
	err = manager.BroadcastToRoom("test-room", map[string]string{"message": "hello"})
	if err != nil {
		t.Errorf("Expected no error for empty room, got %v", err)
	}

	// Add mock connections to the room (without actual WebSocket connections)
	conn1 := &WebSocketConnection{
		ID:   "conn1",
		Room: "test-room",
	}
	conn2 := &WebSocketConnection{
		ID:   "conn2",
		Room: "test-room",
	}

	room.Connections["conn1"] = conn1
	room.Connections["conn2"] = conn2

	// Test broadcasting (will fail to send but shouldn't error)
	err = manager.BroadcastToRoom("test-room", map[string]string{"message": "hello"})
	if err != nil {
		t.Errorf("Expected no error for broadcast, got %v", err)
	}
}

func TestWebSocketManager_GetRoomConnections(t *testing.T) {
	server := &Server{}
	manager := NewWebSocketManager(server, nil, nil)

	// Test getting connections from non-existent room
	connections := manager.GetRoomConnections("non-existent")
	if connections != nil {
		t.Errorf("Expected nil for non-existent room, got %v", connections)
	}

	// Create a room with connections
	channel := config.WebSocketChannelConfig{
		Room:   "test-room",
		Type:   "public",
		Handle: "test.ts",
	}
	room := manager.getOrCreateRoom("test-room", channel)

	conn1 := &WebSocketConnection{ID: "conn1", Room: "test-room"}
	conn2 := &WebSocketConnection{ID: "conn2", Room: "test-room"}

	room.Connections["conn1"] = conn1
	room.Connections["conn2"] = conn2

	connections = manager.GetRoomConnections("test-room")
	if connections == nil {
		t.Error("Expected connections list, got nil")
	}
	if len(connections) != 2 {
		t.Errorf("Expected 2 connections, got %d", len(connections))
	}
}

func TestWebSocketManager_GetRoomCount(t *testing.T) {
	server := &Server{}
	manager := NewWebSocketManager(server, nil, nil)

	// Test getting count from non-existent room
	count := manager.GetRoomCount("non-existent")
	if count != 0 {
		t.Errorf("Expected 0 for non-existent room, got %d", count)
	}

	// Create a room with connections
	channel := config.WebSocketChannelConfig{
		Room:   "test-room",
		Type:   "public",
		Handle: "test.ts",
	}
	room := manager.getOrCreateRoom("test-room", channel)

	conn1 := &WebSocketConnection{ID: "conn1", Room: "test-room"}
	conn2 := &WebSocketConnection{ID: "conn2", Room: "test-room"}

	room.Connections["conn1"] = conn1
	room.Connections["conn2"] = conn2

	count = manager.GetRoomCount("test-room")
	if count != 2 {
		t.Errorf("Expected 2 connections, got %d", count)
	}
}

func TestWebSocketMessage_JSON(t *testing.T) {
	originalMsg := WebSocketMessage{
		Type:      "message",
		Room:      "test-room",
		Data:      map[string]interface{}{"text": "hello world"},
		UserID:    "user123",
		MessageID: "msg_123",
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{"source": "test"},
	}

	// Test marshaling
	data, err := json.Marshal(originalMsg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty JSON data")
	}

	// Test unmarshaling
	var parsedMsg WebSocketMessage
	err = json.Unmarshal(data, &parsedMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}
	if parsedMsg.Type != originalMsg.Type {
		t.Errorf("Expected type %s, got %s", originalMsg.Type, parsedMsg.Type)
	}
	if parsedMsg.Room != originalMsg.Room {
		t.Errorf("Expected room %s, got %s", originalMsg.Room, parsedMsg.Room)
	}
	if parsedMsg.UserID != originalMsg.UserID {
		t.Errorf("Expected user ID %s, got %s", originalMsg.UserID, parsedMsg.UserID)
	}
	if parsedMsg.MessageID != originalMsg.MessageID {
		t.Errorf("Expected message ID %s, got %s", originalMsg.MessageID, parsedMsg.MessageID)
	}
}

func TestWebSocketRoom_ConcurrentAccess(t *testing.T) {
	server := &Server{}
	manager := NewWebSocketManager(server, nil, nil)

	channel := config.WebSocketChannelConfig{
		Room:   "concurrent-test",
		Type:   "public",
		Handle: "test.ts",
	}
	room := manager.getOrCreateRoom("concurrent-test", channel)

	// Test concurrent access to room connections
	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn := &WebSocketConnection{
				ID:   fmt.Sprintf("conn_%d", id),
				Room: "concurrent-test",
			}

			// Add connection
			room.mutex.Lock()
			room.Connections[conn.ID] = conn
			room.mutex.Unlock()

			// Read connection count
			room.mutex.RLock()
			_ = len(room.Connections)
			room.mutex.RUnlock()

			// Remove connection
			room.mutex.Lock()
			delete(room.Connections, conn.ID)
			room.mutex.Unlock()
		}(i)
	}

	wg.Wait()

	// Room should be empty after all goroutines complete
	room.mutex.RLock()
	connectionCount := len(room.Connections)
	room.mutex.RUnlock()

	if connectionCount != 0 {
		t.Errorf("Expected 0 connections after concurrent test, got %d", connectionCount)
	}
}

// Benchmark tests
func BenchmarkWebSocketManager_GenerateConnectionID(b *testing.B) {
	server := &Server{}
	manager := NewWebSocketManager(server, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.generateConnectionID()
	}
}

func BenchmarkWebSocketManager_FindMatchingChannel(b *testing.B) {
	server := &Server{}

	channels := []config.WebSocketChannelConfig{
		{Room: "public-.*", Type: "public"},
		{Room: "room-[0-9]+", Type: "private"},
		{Room: "presence-.*", Type: "presence"},
	}

	manager := NewWebSocketManager(server, channels, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.findMatchingChannel("public-lobby")
	}
}

func BenchmarkWebSocketManager_GetOrCreateRoom(b *testing.B) {
	server := &Server{}
	manager := NewWebSocketManager(server, nil, nil)

	channel := config.WebSocketChannelConfig{
		Room:   "bench-room",
		Type:   "public",
		Handle: "test.ts",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.getOrCreateRoom("bench-room", channel)
	}
}
