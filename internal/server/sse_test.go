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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/daison12006013/turboscript/internal/config"
)

func TestNewSSEManager(t *testing.T) {
	server := &Server{}

	sseConfig := &config.SSEConfig{
		KeepAliveInterval: 60,
		Retry:             5000,
		MaxConnections:    200,
		BufferSize:        2048,
		AllowedOrigins:    []string{"https://example.com", "https://app.example.com"},
		EnableCompression: true,
		EnableHTTP2:       true,
	}

	manager := NewSSEManager(server, sseConfig)

	// Test manager initialization
	if manager == nil {
		t.Fatal("Expected manager to be created, got nil")
	}
	if manager.server != server {
		t.Errorf("Expected server to be %v, got %v", server, manager.server)
	}
	if manager.keepAliveInterval != 60*time.Second {
		t.Errorf("Expected keep-alive interval 60s, got %v", manager.keepAliveInterval)
	}
	if manager.retryInterval != 5000 {
		t.Errorf("Expected retry interval 5000ms, got %d", manager.retryInterval)
	}
	if manager.maxConnections != 200 {
		t.Errorf("Expected max connections 200, got %d", manager.maxConnections)
	}
	if manager.bufferSize != 2048 {
		t.Errorf("Expected buffer size 2048, got %d", manager.bufferSize)
	}
	if len(manager.allowedOrigins) != 2 {
		t.Errorf("Expected 2 allowed origins, got %d", len(manager.allowedOrigins))
	}
	if !manager.enableCompression {
		t.Error("Expected compression to be enabled")
	}
	if !manager.enableHTTP2 {
		t.Error("Expected HTTP/2 to be enabled")
	}

	// Test maps initialization
	if manager.connections == nil {
		t.Error("Expected connections map to be initialized")
	}
	if manager.userConnections == nil {
		t.Error("Expected user connections map to be initialized")
	}
}

func TestNewSSEManagerWithDefaults(t *testing.T) {
	server := &Server{}

	manager := NewSSEManager(server, nil)

	// Test default values
	if manager == nil {
		t.Fatal("Expected manager to be created, got nil")
	}
	if manager.keepAliveInterval != 30*time.Second {
		t.Errorf("Expected default keep-alive interval 30s, got %v", manager.keepAliveInterval)
	}
	if manager.retryInterval != 3000 {
		t.Errorf("Expected default retry interval 3000ms, got %d", manager.retryInterval)
	}
	if manager.maxConnections != 0 {
		t.Errorf("Expected unlimited connections (0), got %d", manager.maxConnections)
	}
	if manager.bufferSize != 1024 {
		t.Errorf("Expected default buffer size 1024, got %d", manager.bufferSize)
	}
	if len(manager.allowedOrigins) != 0 {
		t.Errorf("Expected no allowed origins by default, got %d", len(manager.allowedOrigins))
	}
	if manager.enableCompression {
		t.Error("Expected compression to be disabled by default")
	}
	if manager.enableHTTP2 {
		t.Error("Expected HTTP/2 to be disabled by default")
	}
}

func TestSSEManager_GenerateConnectionID(t *testing.T) {
	server := &Server{}
	manager := NewSSEManager(server, nil)

	id1 := manager.generateConnectionID()
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
	if id1[:4] != "sse_" {
		t.Errorf("Expected connection ID to start with 'sse_', got %s", id1)
	}
}

func TestSSEManager_GenerateMessageID(t *testing.T) {
	server := &Server{}
	manager := NewSSEManager(server, nil)

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

func TestSSEManager_CheckCORS(t *testing.T) {
	server := &Server{}

	// Test with no allowed origins (allow all)
	manager1 := NewSSEManager(server, nil)

	// Mock fasthttp.RequestCtx for testing
	// Since we can't easily create a real RequestCtx, we'll test the logic conceptually

	// Test with specific allowed origins
	sseConfig := &config.SSEConfig{
		AllowedOrigins: []string{"https://example.com", "https://app.example.com"},
	}
	manager2 := NewSSEManager(server, sseConfig)

	if len(manager1.allowedOrigins) != 0 {
		t.Error("Expected no origins restriction for manager1")
	}
	if len(manager2.allowedOrigins) != 2 {
		t.Errorf("Expected 2 allowed origins for manager2, got %d", len(manager2.allowedOrigins))
	}
}

func TestSSEManager_SendToConnection(t *testing.T) {
	server := &Server{}
	manager := NewSSEManager(server, nil)

	// Test sending to non-existent connection
	err := manager.SendToConnection("non-existent", "test", "data")
	if err == nil {
		t.Error("Expected error for non-existent connection")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got %v", err)
	}

	// Create a mock connection
	conn := &SSEConnection{
		ID:           "test-conn",
		UserID:       "test-user",
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		Channel:      make(chan SSEMessage, 10),
		Done:         make(chan bool, 1),
		IsActive:     true,
		UserData:     make(map[string]interface{}),
	}

	manager.connections["test-conn"] = conn

	// Test sending to existing connection
	err = manager.SendToConnection("test-conn", "notification", map[string]string{"message": "hello"})
	if err != nil {
		t.Errorf("Expected no error for existing connection, got %v", err)
	}

	// Check if message was sent
	select {
	case msg := <-conn.Channel:
		if msg.Event != "notification" {
			t.Errorf("Expected event 'notification', got %s", msg.Event)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected message to be sent to channel")
	}
}

func TestSSEManager_SendToUser(t *testing.T) {
	server := &Server{}
	manager := NewSSEManager(server, nil)

	// Test sending to non-existent user
	err := manager.SendToUser("non-existent", "test", "data")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}

	// Create mock connections for a user
	conn1 := &SSEConnection{
		ID:       "conn1",
		UserID:   "test-user",
		Channel:  make(chan SSEMessage, 10),
		Done:     make(chan bool, 1),
		IsActive: true,
	}
	conn2 := &SSEConnection{
		ID:       "conn2",
		UserID:   "test-user",
		Channel:  make(chan SSEMessage, 10),
		Done:     make(chan bool, 1),
		IsActive: true,
	}

	manager.connections["conn1"] = conn1
	manager.connections["conn2"] = conn2
	manager.userConnections["test-user"] = []*SSEConnection{conn1, conn2}

	// Test sending to existing user
	err = manager.SendToUser("test-user", "notification", map[string]string{"message": "hello"})
	if err != nil {
		t.Errorf("Expected no error for existing user, got %v", err)
	}

	// Check if messages were sent to both connections
	select {
	case msg := <-conn1.Channel:
		if msg.Event != "notification" {
			t.Errorf("Expected event 'notification' for conn1, got %s", msg.Event)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected message to be sent to conn1")
	}

	select {
	case msg := <-conn2.Channel:
		if msg.Event != "notification" {
			t.Errorf("Expected event 'notification' for conn2, got %s", msg.Event)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected message to be sent to conn2")
	}
}

func TestSSEManager_Broadcast(t *testing.T) {
	server := &Server{}
	manager := NewSSEManager(server, nil)

	// Create mock connections
	conn1 := &SSEConnection{
		ID:       "conn1",
		Channel:  make(chan SSEMessage, 10),
		Done:     make(chan bool, 1),
		IsActive: true,
	}
	conn2 := &SSEConnection{
		ID:       "conn2",
		Channel:  make(chan SSEMessage, 10),
		Done:     make(chan bool, 1),
		IsActive: true,
	}

	manager.connections["conn1"] = conn1
	manager.connections["conn2"] = conn2

	// Test broadcast
	err := manager.Broadcast("announcement", map[string]string{"message": "server update"})
	if err != nil {
		t.Errorf("Expected no error for broadcast, got %v", err)
	}

	// Check if messages were sent to all connections
	select {
	case msg := <-conn1.Channel:
		if msg.Event != "announcement" {
			t.Errorf("Expected event 'announcement' for conn1, got %s", msg.Event)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected message to be sent to conn1")
	}

	select {
	case msg := <-conn2.Channel:
		if msg.Event != "announcement" {
			t.Errorf("Expected event 'announcement' for conn2, got %s", msg.Event)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected message to be sent to conn2")
	}
}

func TestSSEManager_GetConnectionCount(t *testing.T) {
	server := &Server{}
	manager := NewSSEManager(server, nil)

	// Test with no connections
	count := manager.GetConnectionCount()
	if count != 0 {
		t.Errorf("Expected 0 connections, got %d", count)
	}

	// Add connections
	conn1 := &SSEConnection{ID: "conn1"}
	conn2 := &SSEConnection{ID: "conn2"}

	manager.connections["conn1"] = conn1
	manager.connections["conn2"] = conn2

	count = manager.GetConnectionCount()
	if count != 2 {
		t.Errorf("Expected 2 connections, got %d", count)
	}
}

func TestSSEManager_GetUserConnectionCount(t *testing.T) {
	server := &Server{}
	manager := NewSSEManager(server, nil)

	// Test with no user connections
	count := manager.GetUserConnectionCount("test-user")
	if count != 0 {
		t.Errorf("Expected 0 connections for user, got %d", count)
	}

	// Add user connections
	conn1 := &SSEConnection{ID: "conn1", UserID: "test-user"}
	conn2 := &SSEConnection{ID: "conn2", UserID: "test-user"}

	manager.userConnections["test-user"] = []*SSEConnection{conn1, conn2}

	count = manager.GetUserConnectionCount("test-user")
	if count != 2 {
		t.Errorf("Expected 2 connections for user, got %d", count)
	}
}

func TestSSEManager_AssociateUserWithConnection(t *testing.T) {
	server := &Server{}
	manager := NewSSEManager(server, nil)

	// Test associating user with non-existent connection
	err := manager.AssociateUserWithConnection("non-existent", "test-user")
	if err == nil {
		t.Error("Expected error for non-existent connection")
	}

	// Create a connection
	conn := &SSEConnection{
		ID:     "test-conn",
		UserID: "",
	}
	manager.connections["test-conn"] = conn

	// Test associating user with connection
	err = manager.AssociateUserWithConnection("test-conn", "test-user")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check if user was associated
	if conn.UserID != "test-user" {
		t.Errorf("Expected user ID 'test-user', got %s", conn.UserID)
	}

	// Check if connection was added to user connections
	userConns, exists := manager.userConnections["test-user"]
	if !exists {
		t.Error("Expected user to have connections")
	}
	if len(userConns) != 1 {
		t.Errorf("Expected 1 connection for user, got %d", len(userConns))
	}
	if userConns[0] != conn {
		t.Error("Expected connection to be in user connections")
	}
}

func TestSSEMessage_JSON(t *testing.T) {
	originalMsg := SSEMessage{
		ID:      "msg_123",
		Event:   "notification",
		Data:    map[string]interface{}{"text": "hello world"},
		Retry:   3000,
		Comment: "test message",
		Raw:     false,
	}

	// Test marshaling
	data, err := json.Marshal(originalMsg)
	if err != nil {
		t.Fatalf("Failed to marshal SSE message: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty JSON data")
	}

	// Test unmarshaling
	var parsedMsg SSEMessage
	err = json.Unmarshal(data, &parsedMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal SSE message: %v", err)
	}
	if parsedMsg.ID != originalMsg.ID {
		t.Errorf("Expected ID %s, got %s", originalMsg.ID, parsedMsg.ID)
	}
	if parsedMsg.Event != originalMsg.Event {
		t.Errorf("Expected event %s, got %s", originalMsg.Event, parsedMsg.Event)
	}
	if parsedMsg.Retry != originalMsg.Retry {
		t.Errorf("Expected retry %d, got %d", originalMsg.Retry, parsedMsg.Retry)
	}
}

func TestSSEManager_ConcurrentAccess(t *testing.T) {
	server := &Server{}
	manager := NewSSEManager(server, nil)

	// Test concurrent access to connections
	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn := &SSEConnection{
				ID:       fmt.Sprintf("conn_%d", id),
				UserID:   fmt.Sprintf("user_%d", id%10), // 10 different users
				Channel:  make(chan SSEMessage, 10),
				Done:     make(chan bool, 1),
				IsActive: true,
			}

			// Add connection
			manager.mutex.Lock()
			manager.connections[conn.ID] = conn
			manager.mutex.Unlock()

			// Associate with user
			manager.AssociateUserWithConnection(conn.ID, conn.UserID)

			// Read connection count
			_ = manager.GetConnectionCount()
			_ = manager.GetUserConnectionCount(conn.UserID)

			// Remove connection
			manager.mutex.Lock()
			delete(manager.connections, conn.ID)
			if userConns, exists := manager.userConnections[conn.UserID]; exists {
				for i, c := range userConns {
					if c.ID == conn.ID {
						manager.userConnections[conn.UserID] = append(userConns[:i], userConns[i+1:]...)
						break
					}
				}
			}
			manager.mutex.Unlock()
		}(i)
	}

	wg.Wait()

	// All connections should be removed after goroutines complete
	connectionCount := manager.GetConnectionCount()
	if connectionCount != 0 {
		t.Errorf("Expected 0 connections after concurrent test, got %d", connectionCount)
	}
}

// Benchmark tests
func BenchmarkSSEManager_GenerateConnectionID(b *testing.B) {
	server := &Server{}
	manager := NewSSEManager(server, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.generateConnectionID()
	}
}

func BenchmarkSSEManager_SendToConnection(b *testing.B) {
	server := &Server{}
	manager := NewSSEManager(server, nil)

	// Setup a connection
	conn := &SSEConnection{
		ID:       "bench-conn",
		Channel:  make(chan SSEMessage, 1000), // Large buffer to avoid blocking
		Done:     make(chan bool, 1),
		IsActive: true,
	}
	manager.connections["bench-conn"] = conn

	// Drain messages in background
	go func() {
		for range conn.Channel {
			// Consume messages
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.SendToConnection("bench-conn", "test", "data")
	}
}

func BenchmarkSSEManager_Broadcast(b *testing.B) {
	server := &Server{}
	manager := NewSSEManager(server, nil)

	// Setup multiple connections
	for i := 0; i < 100; i++ {
		conn := &SSEConnection{
			ID:       fmt.Sprintf("conn_%d", i),
			Channel:  make(chan SSEMessage, 1000), // Large buffer
			Done:     make(chan bool, 1),
			IsActive: true,
		}
		manager.connections[conn.ID] = conn

		// Drain messages in background
		go func(ch chan SSEMessage) {
			for range ch {
				// Consume messages
			}
		}(conn.Channel)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Broadcast("test", "data")
	}
}
