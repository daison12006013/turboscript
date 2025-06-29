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
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/segmentio/kafka-go"
)

// KafkaMessage represents a message for cross-instance communication.
type KafkaMessage struct {
	Type       string      `json:"type"`        // "websocket" or "sse"
	Action     string      `json:"action"`      // "broadcast", "user_message", "room_message"
	InstanceID string      `json:"instance_id"` // Unique instance identifier
	Timestamp  time.Time   `json:"timestamp"`
	Data       interface{} `json:"data"`
}

// KafkaManager provides comprehensive Kafka integration for scaling WebSocket and SSE.
type KafkaManager struct {
	brokers    []string
	writer     *kafka.Writer
	reader     *kafka.Reader
	topic      string
	instanceID string
	isRunning  bool
	stopChan   chan struct{}
	mutex      sync.RWMutex

	// Callbacks for message handling
	wsManager  *WebSocketManager
	sseManager *SSEManager
}

// NewKafkaManager creates a new Kafka manager for real-time message distribution.
func NewKafkaManager(brokers []string, topic string) *KafkaManager {
	if len(brokers) == 0 {
		logger.Warn("Kafka brokers not configured, scaling will be limited to single instance")
		return nil
	}

	instanceID := fmt.Sprintf("turboscript-%d", time.Now().UnixNano())

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      brokers,
		Topic:        topic,
		BatchTimeout: 10 * time.Millisecond,
		Async:        true, // Non-blocking writes for performance
	})

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     "turboscript-cluster",
		MinBytes:    10e1, // 100B
		MaxBytes:    10e6, // 10MB
		MaxWait:     1 * time.Second,
		StartOffset: kafka.LastOffset,
	})

	logger.Info("Kafka manager initialized for topic: %s, instance: %s", topic, instanceID)

	km := &KafkaManager{
		brokers:    brokers,
		writer:     writer,
		reader:     reader,
		topic:      topic,
		instanceID: instanceID,
		stopChan:   make(chan struct{}),
	}

	return km
}

// SetManagers sets the WebSocket and SSE managers for message routing.
func (km *KafkaManager) SetManagers(wsManager *WebSocketManager, sseManager *SSEManager) {
	if km == nil {
		return
	}
	km.mutex.Lock()
	km.wsManager = wsManager
	km.sseManager = sseManager
	km.mutex.Unlock()
}

// createTopic creates the Kafka topic if it doesn't exist.
func (km *KafkaManager) createTopic() error {
	if len(km.brokers) == 0 {
		return fmt.Errorf("no brokers configured")
	}

	conn, err := kafka.Dial("tcp", km.brokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to Kafka: %v", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %v", err)
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("failed to connect to controller: %v", err)
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             km.topic,
			NumPartitions:     3,
			ReplicationFactor: 1,
		},
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		// Topic might already exist, which is fine
		return fmt.Errorf("failed to create topic: %v", err)
	}

	return nil
}

// Start begins consuming Kafka messages for cross-instance communication.
func (km *KafkaManager) Start() error {
	if km == nil {
		return nil
	}

	// Create topic if it doesn't exist
	if err := km.createTopic(); err != nil {
		logger.Warn("Failed to create topic %s: %v", km.topic, err)
		// Continue anyway, topic might already exist
	}

	km.mutex.Lock()
	if km.isRunning {
		km.mutex.Unlock()
		return fmt.Errorf("kafka manager already running")
	}
	km.isRunning = true
	km.mutex.Unlock()

	logger.Info("Starting Kafka consumer for instance %s", km.instanceID)

	go km.consumeMessages()
	return nil
}

// Stop stops the Kafka manager and closes connections.
func (km *KafkaManager) Stop() error {
	if km == nil {
		return nil
	}

	km.mutex.Lock()
	if !km.isRunning {
		km.mutex.Unlock()
		return nil
	}
	km.isRunning = false
	km.mutex.Unlock()

	close(km.stopChan)

	// Close Kafka connections
	if err := km.writer.Close(); err != nil {
		logger.Error("Failed to close Kafka writer: %v", err)
	}
	if err := km.reader.Close(); err != nil {
		logger.Error("Failed to close Kafka reader: %v", err)
	}

	logger.Info("Kafka manager stopped for instance %s", km.instanceID)
	return nil
}

// consumeMessages continuously consumes messages from Kafka.
func (km *KafkaManager) consumeMessages() {
	ctx := context.Background()

	for {
		select {
		case <-km.stopChan:
			return
		default:
			// Read message with timeout
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			m, err := km.reader.ReadMessage(ctx)
			cancel()

			if err != nil {
				// Handle timeout gracefully
				continue
			}

			km.handleKafkaMessage(m)
		}
	}
}

// handleKafkaMessage processes incoming Kafka messages.
func (km *KafkaManager) handleKafkaMessage(m kafka.Message) {
	var kafkaMsg KafkaMessage
	if err := json.Unmarshal(m.Value, &kafkaMsg); err != nil {
		logger.Error("Failed to unmarshal Kafka message: %v", err)
		return
	}

	// Skip messages from the same instance to avoid loops
	if kafkaMsg.InstanceID == km.instanceID {
		return
	}

	logger.Debug("Received Kafka message: type=%s, action=%s, from=%s",
		kafkaMsg.Type, kafkaMsg.Action, kafkaMsg.InstanceID)

	km.mutex.RLock()
	wsManager := km.wsManager
	sseManager := km.sseManager
	km.mutex.RUnlock()

	// Route message to appropriate manager
	switch kafkaMsg.Type {
	case "websocket":
		if wsManager != nil {
			km.handleWebSocketKafkaMessage(wsManager, kafkaMsg)
		}
	case "sse":
		if sseManager != nil {
			km.handleSSEKafkaMessage(sseManager, kafkaMsg)
		}
	default:
		logger.Warn("Unknown Kafka message type: %s", kafkaMsg.Type)
	}
}

// handleWebSocketKafkaMessage handles WebSocket-specific Kafka messages.
func (km *KafkaManager) handleWebSocketKafkaMessage(wsManager *WebSocketManager, kafkaMsg KafkaMessage) {
	switch kafkaMsg.Action {
	case "room_broadcast":
		// Extract WebSocket message from Kafka data
		dataBytes, _ := json.Marshal(kafkaMsg.Data)
		var wsMsg WebSocketMessage
		if err := json.Unmarshal(dataBytes, &wsMsg); err != nil {
			logger.Error("Failed to unmarshal WebSocket message from Kafka: %v", err)
			return
		}

		// Broadcast to local connections only (no Kafka re-publish)
		wsManager.broadcastToRoomLocal(wsMsg.Room, wsMsg, "")

	case "user_message":
		// Handle user-specific messages if needed
		logger.Debug("Received user message from Kafka: %v", kafkaMsg.Data)

	default:
		logger.Warn("Unknown WebSocket Kafka action: %s", kafkaMsg.Action)
	}
}

// handleSSEKafkaMessage handles SSE-specific Kafka messages.
func (km *KafkaManager) handleSSEKafkaMessage(sseManager *SSEManager, kafkaMsg KafkaMessage) {
	switch kafkaMsg.Action {
	case "broadcast":
		// Extract SSE message from Kafka data
		dataBytes, _ := json.Marshal(kafkaMsg.Data)
		var sseMsg SSEMessage
		if err := json.Unmarshal(dataBytes, &sseMsg); err != nil {
			logger.Error("Failed to unmarshal SSE message from Kafka: %v", err)
			return
		}

		// Broadcast to local connections only (no Kafka re-publish)
		if err := sseManager.broadcastLocal(sseMsg.Event, sseMsg.Data); err != nil {
			logger.Error("Failed to broadcast local SSE message: %v", err)
		}

	case "user_message":
		// Handle user-specific SSE messages
		logger.Debug("Received user SSE message from Kafka: %v", kafkaMsg.Data)

	default:
		logger.Warn("Unknown SSE Kafka action: %s", kafkaMsg.Action)
	}
}

// PublishWebSocketMessage publishes a WebSocket message to Kafka for multi-instance scaling.
func (km *KafkaManager) PublishWebSocketMessage(msg WebSocketMessage) error {
	if km == nil {
		return nil // Kafka not configured, skip
	}

	kafkaMsg := KafkaMessage{
		Type:       "websocket",
		Action:     "room_broadcast",
		InstanceID: km.instanceID,
		Timestamp:  time.Now(),
		Data:       msg,
	}

	return km.publishMessage(kafkaMsg, msg.Room)
}

// PublishSSEMessage publishes an SSE message to Kafka for multi-instance scaling.
func (km *KafkaManager) PublishSSEMessage(event string, data interface{}, userID string) error {
	if km == nil {
		return nil // Kafka not configured, skip
	}

	sseMsg := SSEMessage{
		Event: event,
		Data:  data,
	}

	kafkaMsg := KafkaMessage{
		Type:       "sse",
		Action:     "broadcast",
		InstanceID: km.instanceID,
		Timestamp:  time.Now(),
		Data:       sseMsg,
	}

	return km.publishMessage(kafkaMsg, userID)
}

// publishMessage publishes a Kafka message with the given partition key.
func (km *KafkaManager) publishMessage(kafkaMsg KafkaMessage, partitionKey string) error {
	data, err := json.Marshal(kafkaMsg)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(partitionKey), // Use room/user as partition key for ordering
		Value: data,
		Headers: []kafka.Header{
			{Key: "type", Value: []byte(kafkaMsg.Type)},
			{Key: "action", Value: []byte(kafkaMsg.Action)},
			{Key: "instance_id", Value: []byte(kafkaMsg.InstanceID)},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return km.writer.WriteMessages(ctx, msg)
}

// IsHealthy checks if Kafka is reachable and functional.
func (km *KafkaManager) IsHealthy() bool {
	if km == nil || len(km.brokers) == 0 {
		return false
	}

	// Try to connect to Kafka broker
	conn, err := kafka.Dial("tcp", km.brokers[0])
	if err != nil {
		return false
	}
	defer conn.Close()

	// Try to get broker list which indicates Kafka is responsive
	_, err = conn.Brokers()
	return err == nil
}
