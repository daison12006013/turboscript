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
	"testing"

	"github.com/daison12006013/turboscript/internal/config"
)

// TestKafkaConditional tests that Kafka integration is properly conditional.
func TestKafkaConditional_WebSocketWithoutKafka(t *testing.T) {
	server := &Server{}
	channels := []config.WebSocketChannelConfig{
		{
			Room:   "test-room",
			Type:   "public",
			Handle: "test.ts",
		},
	}

	// Configuration with Kafka disabled
	wsConfig := &config.WebSocketConfig{
		EnableKafka:  false,      // Explicitly disabled
		KafkaBrokers: []string{}, // No brokers
		KafkaTopic:   "",         // No topic
	}

	// Create WebSocket manager without Kafka
	wsManager := NewWebSocketManager(server, channels, wsConfig)
	if wsManager == nil {
		t.Fatal("Expected WebSocket manager to be created")
	}

	// Kafka manager should be nil when disabled
	if wsManager.kafkaManager != nil {
		t.Error("Expected Kafka manager to be nil when Kafka is disabled")
	}

	// WebSocket should still work for local broadcasting
	err := wsManager.BroadcastToRoom("test-room", map[string]string{"message": "local test"})
	if err != nil {
		t.Errorf("Expected local broadcast to work without Kafka: %v", err)
	}

	// Starting/stopping Kafka should be safe when disabled
	err = wsManager.StartKafka(nil)
	if err != nil {
		t.Errorf("Expected StartKafka to be safe when Kafka is disabled: %v", err)
	}

	err = wsManager.StopKafka()
	if err != nil {
		t.Errorf("Expected StopKafka to be safe when Kafka is disabled: %v", err)
	}
}

// TestKafkaConditional_SSEWithoutKafka tests SSE without Kafka.
func TestKafkaConditional_SSEWithoutKafka(t *testing.T) {
	server := &Server{}

	// Create SSE manager without Kafka configuration
	sseManager := NewSSEManager(server, nil)
	if sseManager == nil {
		t.Fatal("Expected SSE manager to be created")
	}

	// Kafka manager should be nil by default
	if sseManager.kafkaManager != nil {
		t.Error("Expected Kafka manager to be nil when not configured")
	}

	// SSE should still work for local broadcasting
	err := sseManager.Broadcast("test_event", map[string]string{"message": "local test"})
	if err != nil {
		t.Errorf("Expected local broadcast to work without Kafka: %v", err)
	}
}

// TestKafkaConditional_WebSocketWithKafka tests that Kafka is properly initialized when enabled.
func TestKafkaConditional_WebSocketWithKafka(t *testing.T) {
	server := &Server{}
	channels := []config.WebSocketChannelConfig{
		{
			Room:   "test-room",
			Type:   "public",
			Handle: "test.ts",
		},
	}

	// Configuration with Kafka enabled
	wsConfig := &config.WebSocketConfig{
		EnableKafka:  true,
		KafkaBrokers: []string{"localhost:9092"},
		KafkaTopic:   "test-topic",
	}

	// Create WebSocket manager with Kafka
	wsManager := NewWebSocketManager(server, channels, wsConfig)
	if wsManager == nil {
		t.Fatal("Expected WebSocket manager to be created")
	}

	// Kafka manager should be initialized when enabled
	if wsManager.kafkaManager == nil {
		t.Error("Expected Kafka manager to be initialized when Kafka is enabled")
	}

	// Verify Kafka configuration
	if !wsManager.enableKafka {
		t.Error("Expected enableKafka to be true")
	}

	if wsManager.kafkaTopic != "test-topic" {
		t.Errorf("Expected Kafka topic to be 'test-topic', got %s", wsManager.kafkaTopic)
	}
}

// TestKafkaConditional_PartialConfig tests behavior with partial Kafka configuration.
func TestKafkaConditional_PartialConfig(t *testing.T) {
	server := &Server{}
	channels := []config.WebSocketChannelConfig{
		{
			Room:   "test-room",
			Type:   "public",
			Handle: "test.ts",
		},
	}

	testCases := []struct {
		name        string
		config      *config.WebSocketConfig
		expectKafka bool
		description string
	}{
		{
			name: "Kafka enabled but no brokers",
			config: &config.WebSocketConfig{
				EnableKafka:  true,
				KafkaBrokers: []string{},
				KafkaTopic:   "test-topic",
			},
			expectKafka: false,
			description: "Should not initialize Kafka without brokers",
		},
		{
			name: "Kafka enabled but no topic",
			config: &config.WebSocketConfig{
				EnableKafka:  true,
				KafkaBrokers: []string{"localhost:9092"},
				KafkaTopic:   "",
			},
			expectKafka: false,
			description: "Should not initialize Kafka without topic",
		},
		{
			name: "Kafka disabled with brokers and topic",
			config: &config.WebSocketConfig{
				EnableKafka:  false,
				KafkaBrokers: []string{"localhost:9092"},
				KafkaTopic:   "test-topic",
			},
			expectKafka: false,
			description: "Should not initialize Kafka when disabled",
		},
		{
			name: "Complete Kafka configuration",
			config: &config.WebSocketConfig{
				EnableKafka:  true,
				KafkaBrokers: []string{"localhost:9092"},
				KafkaTopic:   "test-topic",
			},
			expectKafka: true,
			description: "Should initialize Kafka with complete config",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wsManager := NewWebSocketManager(server, channels, tc.config)
			if wsManager == nil {
				t.Fatal("Expected WebSocket manager to be created")
			}

			hasKafka := wsManager.kafkaManager != nil
			if hasKafka != tc.expectKafka {
				t.Errorf("%s: expected Kafka manager to be %v, got %v",
					tc.description, tc.expectKafka, hasKafka)
			}

			// Broadcasting should always work regardless of Kafka configuration
			err := wsManager.BroadcastToRoom("test-room", map[string]string{"test": "message"})
			if err != nil {
				t.Errorf("Broadcasting should work regardless of Kafka config: %v", err)
			}
		})
	}
}
