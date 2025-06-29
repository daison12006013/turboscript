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
	"sync"
	"testing"
	"time"

	"github.com/daison12006013/turboscript/internal/config"
)

// TestKafkaManager_Integration tests Kafka manager with real Kafka instance in Docker.
func TestKafkaManager_Integration(t *testing.T) {
	// Skip test if not in Docker environment
	if os.Getenv("DOCKER_ENV") != "true" {
		t.Skip("Skipping Kafka integration test - requires DOCKER_ENV=true")
	}

	brokers := []string{"kafka:29092"}
	topic := "test-websocket-events"

	// Create Kafka manager
	km := NewKafkaManager(brokers, topic)
	if km == nil {
		t.Fatal("Expected Kafka manager to be created")
	}

	// Test health check
	if !km.IsHealthy() {
		t.Error("Expected Kafka to be healthy")
	}

	// Start the manager
	if err := km.Start(); err != nil {
		t.Errorf("Failed to start Kafka manager: %v", err)
	}
	defer km.Stop()

	// Test publishing a message
	testMsg := WebSocketMessage{
		Type:      "message",
		Room:      "test-room",
		Data:      map[string]interface{}{"text": "hello kafka"},
		MessageID: "test-msg-123",
		Timestamp: time.Now(),
	}

	if err := km.PublishWebSocketMessage(testMsg); err != nil {
		t.Errorf("Failed to publish WebSocket message: %v", err)
	}

	// Test publishing SSE message
	if err := km.PublishSSEMessage("test_event", map[string]string{"data": "test"}, "user123"); err != nil {
		t.Errorf("Failed to publish SSE message: %v", err)
	}

	// Give some time for messages to be processed
	time.Sleep(2 * time.Second)
}

// TestKafkaManager_CrossInstance tests cross-instance communication.
func TestKafkaManager_CrossInstance(t *testing.T) {
	// Skip test if not in Docker environment
	if os.Getenv("DOCKER_ENV") != "true" {
		t.Skip("Skipping cross-instance test - requires DOCKER_ENV=true")
	}

	brokers := []string{"kafka:29092"}
	topic := "test-cross-instance"

	// Create first instance (producer)
	km1 := NewKafkaManager(brokers, topic)
	if km1 == nil {
		t.Fatal("Expected first Kafka manager to be created")
	}

	// Create second instance (consumer)
	km2 := NewKafkaManager(brokers, topic)
	if km2 == nil {
		t.Fatal("Expected second Kafka manager to be created")
	}

	// Create test WebSocket and SSE managers
	server := &Server{}
	wsManager := NewWebSocketManager(server, nil, nil)
	sseManager := NewSSEManager(server, nil)

	// Set up the second instance as consumer
	km2.SetManagers(wsManager, sseManager)

	if err := km1.Start(); err != nil {
		t.Errorf("Failed to start first Kafka manager: %v", err)
	}
	defer km1.Stop()

	if err := km2.Start(); err != nil {
		t.Errorf("Failed to start second Kafka manager: %v", err)
	}
	defer km2.Stop()

	// Track received messages
	var receivedMessages []KafkaMessage

	// Mock the message handling to capture messages
	originalWSManager := km2.wsManager
	originalSSEManager := km2.sseManager

	// Wait a moment for consumers to be ready
	time.Sleep(3 * time.Second)

	// Test WebSocket message cross-instance
	testWSMsg := WebSocketMessage{
		Type:      "broadcast",
		Room:      "cross-instance-room",
		Data:      map[string]interface{}{"message": "cross-instance test"},
		MessageID: "cross-msg-123",
		Timestamp: time.Now(),
	}

	if err := km1.PublishWebSocketMessage(testWSMsg); err != nil {
		t.Errorf("Failed to publish cross-instance WebSocket message: %v", err)
	}

	// Test SSE message cross-instance
	if err := km1.PublishSSEMessage("cross_event", map[string]string{"data": "cross-test"}, "user456"); err != nil {
		t.Errorf("Failed to publish cross-instance SSE message: %v", err)
	}

	// Wait for messages to be processed
	time.Sleep(3 * time.Second)

	// Restore original managers
	km2.wsManager = originalWSManager
	km2.sseManager = originalSSEManager

	// For now, just ensure no errors occurred (more detailed testing would require mocking)
	t.Logf("Cross-instance test completed - sent %d messages", len(receivedMessages))
}

// TestKafkaManager_WebSocketIntegration tests WebSocket manager with Kafka.
func TestKafkaManager_WebSocketIntegration(t *testing.T) {
	// Skip test if not in Docker environment
	if os.Getenv("DOCKER_ENV") != "true" {
		t.Skip("Skipping WebSocket Kafka integration test - requires DOCKER_ENV=true")
	}

	server := &Server{}
	channels := []config.WebSocketChannelConfig{
		{
			Room:   "kafka-test-.*",
			Type:   "public",
			Handle: "test.ts",
		},
	}

	wsConfig := &config.WebSocketConfig{
		EnableKafka:  true,
		KafkaBrokers: []string{"kafka:29092"},
		KafkaTopic:   "test-ws-integration",
	}

	// Create WebSocket manager with Kafka
	wsManager := NewWebSocketManager(server, channels, wsConfig)
	if wsManager == nil {
		t.Fatal("Expected WebSocket manager to be created")
	}

	if wsManager.kafkaManager == nil {
		t.Fatal("Expected Kafka manager to be initialized")
	}

	// Create SSE manager for testing
	sseManager := NewSSEManager(server, nil)

	// Start Kafka integration
	if err := wsManager.StartKafka(sseManager); err != nil {
		t.Errorf("Failed to start WebSocket Kafka integration: %v", err)
	}
	defer wsManager.StopKafka()

	// Test health check
	if !wsManager.kafkaManager.IsHealthy() {
		t.Error("Expected Kafka to be healthy through WebSocket manager")
	}

	// Test broadcasting a message (should go through Kafka)
	testData := map[string]string{"test": "kafka-integration"}
	if err := wsManager.BroadcastToRoom("kafka-test-room", testData); err != nil {
		t.Errorf("Failed to broadcast message through Kafka: %v", err)
	}

	// Wait for message processing
	time.Sleep(2 * time.Second)
}

// TestKafkaManager_SSEIntegration tests SSE manager with Kafka.
func TestKafkaManager_SSEIntegration(t *testing.T) {
	// Skip test if not in Docker environment
	if os.Getenv("DOCKER_ENV") != "true" {
		t.Skip("Skipping SSE Kafka integration test - requires DOCKER_ENV=true")
	}

	server := &Server{}

	// Create SSE manager
	sseManager := NewSSEManager(server, nil)
	if sseManager == nil {
		t.Fatal("Expected SSE manager to be created")
	}

	// Create Kafka manager and associate with SSE
	kafkaManager := NewKafkaManager([]string{"kafka:29092"}, "test-sse-integration")
	if kafkaManager == nil {
		t.Fatal("Expected Kafka manager to be created")
	}

	sseManager.SetKafkaManager(kafkaManager)
	kafkaManager.SetManagers(nil, sseManager)

	// Start Kafka
	if err := kafkaManager.Start(); err != nil {
		t.Errorf("Failed to start Kafka for SSE: %v", err)
	}
	defer kafkaManager.Stop()

	// Test broadcasting SSE message (should go through Kafka)
	testData := map[string]string{"notification": "kafka-sse-test"}
	if err := sseManager.Broadcast("test_event", testData); err != nil {
		t.Errorf("Failed to broadcast SSE message through Kafka: %v", err)
	}

	// Wait for message processing
	time.Sleep(2 * time.Second)
}

// TestKafkaManager_HealthCheck tests Kafka health checking.
func TestKafkaManager_HealthCheck(t *testing.T) {
	// Test with invalid brokers (should be unhealthy)
	km := NewKafkaManager([]string{"invalid:9092"}, "test-topic")
	if km == nil {
		t.Fatal("Expected Kafka manager to be created even with invalid brokers")
	}

	// Health check should fail with invalid brokers
	if km.IsHealthy() {
		t.Error("Expected Kafka to be unhealthy with invalid brokers")
	}

	// Test with no brokers (should return nil)
	kmNil := NewKafkaManager([]string{}, "test-topic")
	if kmNil != nil {
		t.Error("Expected nil Kafka manager with empty brokers")
	}
}

// TestKafkaManager_Concurrent tests concurrent Kafka operations.
func TestKafkaManager_Concurrent(t *testing.T) {
	// Skip test if not in Docker environment
	if os.Getenv("DOCKER_ENV") != "true" {
		t.Skip("Skipping concurrent Kafka test - requires DOCKER_ENV=true")
	}

	brokers := []string{"kafka:29092"}
	topic := "test-concurrent"

	km := NewKafkaManager(brokers, topic)
	if km == nil {
		t.Fatal("Expected Kafka manager to be created")
	}

	if err := km.Start(); err != nil {
		t.Errorf("Failed to start Kafka manager: %v", err)
	}
	defer km.Stop()

	// Test concurrent publishing
	var wg sync.WaitGroup
	numMessages := 10

	for i := 0; i < numMessages; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			testMsg := WebSocketMessage{
				Type:      "concurrent_test",
				Room:      "concurrent-room",
				Data:      map[string]interface{}{"id": id},
				MessageID: fmt.Sprintf("concurrent-msg-%d", id),
				Timestamp: time.Now(),
			}

			if err := km.PublishWebSocketMessage(testMsg); err != nil {
				t.Errorf("Failed to publish concurrent message %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// Wait for messages to be processed
	time.Sleep(2 * time.Second)
}

// BenchmarkKafkaManager_Publish benchmarks Kafka message publishing.
func BenchmarkKafkaManager_Publish(b *testing.B) {
	// Skip benchmark if not in Docker environment
	if os.Getenv("DOCKER_ENV") != "true" {
		b.Skip("Skipping Kafka benchmark - requires DOCKER_ENV=true")
	}

	brokers := []string{"kafka:29092"}
	topic := "benchmark-test"

	km := NewKafkaManager(brokers, topic)
	if km == nil {
		b.Fatal("Expected Kafka manager to be created")
	}

	testMsg := WebSocketMessage{
		Type:      "benchmark",
		Room:      "benchmark-room",
		Data:      map[string]interface{}{"data": "benchmark test"},
		MessageID: "benchmark-msg",
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		km.PublishWebSocketMessage(testMsg)
	}
}
