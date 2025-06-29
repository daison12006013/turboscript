package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Event string      `json:"event,omitempty"`
	Data  interface{} `json:"data,omitempty"`
	ID    string      `json:"id,omitempty"`
	Retry int         `json:"retry,omitempty"`
}

// SSEBroadcastRequest represents the HTTP API SSE broadcast request
type SSEBroadcastRequest struct {
	Event     string      `json:"event"`
	Data      interface{} `json:"data"`
	UserID    string      `json:"user_id,omitempty"`
	Target    string      `json:"target,omitempty"`
	Broadcast bool        `json:"broadcast,omitempty"`
	ID        string      `json:"id,omitempty"`
	Retry     int         `json:"retry,omitempty"`
}

// SSEBroadcastResponse represents the HTTP API SSE broadcast response
type SSEBroadcastResponse struct {
	Status string `json:"status"`
	Data   struct {
		ConnectionsNotified int    `json:"connections_notified"`
		Event               string `json:"event"`
		MessageID           string `json:"message_id,omitempty"`
		Target              string `json:"target,omitempty"`
		UserID              string `json:"user_id,omitempty"`
	} `json:"data"`
}

// SSEConnection represents an SSE connection for testing
type SSEConnection struct {
	Response *http.Response
	Reader   *bufio.Scanner
	Events   []SSEEvent
	mu       sync.Mutex
}

// NewSSEConnection creates a new SSE connection
func NewSSEConnection(room, clientID string) (*SSEConnection, error) {
	// Use the correct parameters for /demo-broadcasting/sse/simple endpoint
	url := fmt.Sprintf("http://localhost:7890/demo-broadcasting/sse/simple?user_id=%s&channel=%s", clientID, room)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{
		Timeout: 10 * time.Second, // Add timeout to prevent hanging
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("SSE connection failed with status: %d", resp.StatusCode)
	}

	return &SSEConnection{
		Response: resp,
		Reader:   bufio.NewScanner(resp.Body),
		Events:   make([]SSEEvent, 0),
	}, nil
}

// ReadEvents reads SSE events from the connection
func (c *SSEConnection) ReadEvents(ctx context.Context) {
	go func() {
		defer c.Response.Body.Close()

		var currentEvent SSEEvent

		for c.Reader.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := c.Reader.Text()

			if line == "" {
				// Empty line indicates end of event
				if currentEvent.Event != "" || currentEvent.Data != nil {
					c.mu.Lock()
					c.Events = append(c.Events, currentEvent)
					c.mu.Unlock()
					currentEvent = SSEEvent{}
				}
				continue
			}

			if strings.HasPrefix(line, "event:") {
				currentEvent.Event = strings.TrimSpace(line[6:])
			} else if strings.HasPrefix(line, "data:") {
				dataStr := strings.TrimSpace(line[5:])
				var data interface{}
				if err := json.Unmarshal([]byte(dataStr), &data); err == nil {
					currentEvent.Data = data
				} else {
					currentEvent.Data = dataStr
				}
			} else if strings.HasPrefix(line, "id:") {
				currentEvent.ID = strings.TrimSpace(line[3:])
			} else if strings.HasPrefix(line, "retry:") {
				// Handle retry field if needed
			}
		}
	}()
}

// GetEvents returns a copy of received events
func (c *SSEConnection) GetEvents() []SSEEvent {
	c.mu.Lock()
	defer c.mu.Unlock()

	events := make([]SSEEvent, len(c.Events))
	copy(events, c.Events)
	return events
}

// Close closes the SSE connection
func (c *SSEConnection) Close() {
	if c.Response != nil {
		c.Response.Body.Close()
	}
}

// TestMultipleSSEConnections tests multiple SSE connections and broadcasting
func TestMultipleSSEConnections(t *testing.T) {
	if os.Getenv("E2E_TEST") == "" {
		t.Skip("Skipping E2E SSE test (set E2E_TEST=true to run)")
	}

	httpBaseURL := "http://localhost:7890"
	roomName := "sse-test-multi-" + fmt.Sprintf("%d", time.Now().Unix())

	// Test multiple connections
	connectionCounts := []int{2, 3, 5}

	for _, connCount := range connectionCounts {
		t.Run(fmt.Sprintf("Test_%d_SSE_connections", connCount), func(t *testing.T) {
			testMultipleSSEConnections(t, httpBaseURL, roomName, connCount)
		})
	}
}

func testMultipleSSEConnections(t *testing.T, httpBaseURL, roomName string, connectionCount int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connections := make([]*SSEConnection, connectionCount)

	// Create SSE connections
	for i := 0; i < connectionCount; i++ {
		clientID := fmt.Sprintf("sse_client_%d", i)

		conn, err := NewSSEConnection(roomName, clientID)
		require.NoError(t, err, "Failed to create SSE connection %d", i)

		connections[i] = conn

		// Start reading events
		conn.ReadEvents(ctx)
	}

	defer func() {
		// Clean up connections
		for i, conn := range connections {
			if conn != nil {
				conn.Close()
				t.Logf("🔌 SSE Connection %d closed", i)
			}
		}
	}()

	t.Logf("✅ Created %d SSE connections for room: %s", connectionCount, roomName)

	// Wait for connections to establish
	time.Sleep(2 * time.Second)

	// Test broadcasting via HTTP API
	broadcastMsg := SSEBroadcastRequest{
		Event: "test_message",
		Data: map[string]interface{}{
			"text":      fmt.Sprintf("SSE broadcast test for %d connections", connectionCount),
			"timestamp": time.Now().Unix(),
			"test_id":   fmt.Sprintf("sse-multi-test-%d", connectionCount),
		},
		Broadcast: true,
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
	}

	// Send broadcast request
	jsonData, err := json.Marshal(broadcastMsg)
	require.NoError(t, err, "Failed to marshal SSE broadcast request")

	resp, err := http.Post(
		httpBaseURL+"/demo-broadcasting/sse/broadcast",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(t, err, "Failed to send SSE broadcast request")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "SSE broadcast request failed")

	var broadcastResp SSEBroadcastResponse
	err = json.NewDecoder(resp.Body).Decode(&broadcastResp)
	require.NoError(t, err, "Failed to decode SSE broadcast response")

	assert.Equal(t, "success", broadcastResp.Status, "SSE broadcast status should be success")

	t.Logf("📡 SSE Broadcast sent to %d connections (response: %d)",
		broadcastResp.Data.ConnectionsNotified, broadcastResp.Data.ConnectionsNotified)

	// Wait for broadcast messages to be received
	time.Sleep(3 * time.Second)

	// Verify all connections received the message
	receivedCount := 0
	for i, conn := range connections {
		events := conn.GetEvents()

		hasTestMessage := false
		for _, event := range events {
			if event.Event == "test_message" {
				hasTestMessage = true

				// Verify message content
				if dataMap, ok := event.Data.(map[string]interface{}); ok {
					assert.Contains(t, dataMap, "text", "Event should contain text field")
					assert.Contains(t, dataMap, "timestamp", "Event should contain timestamp field")
					assert.Contains(t, dataMap, "test_id", "Event should contain test_id field")
				}
				break
			}
		}

		if hasTestMessage {
			receivedCount++
			t.Logf("✅ Connection %d received test message", i)
		} else {
			t.Logf("❌ Connection %d did not receive test message (events: %d)", i, len(events))
		}
	}

	assert.GreaterOrEqual(t, receivedCount, connectionCount*80/100,
		"At least 80%% of SSE connections should receive the broadcast message")
}

// TestSSETargetedMessaging tests targeted SSE messaging
func TestSSETargetedMessaging(t *testing.T) {
	if os.Getenv("E2E_TEST") == "" {
		t.Skip("Skipping E2E SSE test (set E2E_TEST=true to run)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	httpBaseURL := "http://localhost:7890"
	roomName := "sse-test-targeted-" + fmt.Sprintf("%d", time.Now().Unix())

	// Create two SSE connections
	conn1, err := NewSSEConnection(roomName, "target_client_1")
	require.NoError(t, err)
	defer conn1.Close()

	conn2, err := NewSSEConnection(roomName, "target_client_2")
	require.NoError(t, err)
	defer conn2.Close()

	// Start reading events
	conn1.ReadEvents(ctx)
	conn2.ReadEvents(ctx)

	// Wait for connections to establish
	time.Sleep(2 * time.Second)

	t.Log("✅ Created 2 SSE connections for targeted messaging test")

	// Send targeted message to client 1 only
	targetedMsg := SSEBroadcastRequest{
		Event: "private_message",
		Data: map[string]interface{}{
			"text":      "This message is only for client 1",
			"recipient": "target_client_1",
		},
		Target:    "target_client_1",
		Broadcast: false,
	}

	jsonData, err := json.Marshal(targetedMsg)
	require.NoError(t, err)

	resp, err := http.Post(
		httpBaseURL+"/demo-broadcasting/sse/broadcast",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var broadcastResp SSEBroadcastResponse
	err = json.NewDecoder(resp.Body).Decode(&broadcastResp)
	require.NoError(t, err)

	assert.Equal(t, "success", broadcastResp.Status)

	// Wait for message to be processed
	time.Sleep(2 * time.Second)

	// Verify only client 1 received the message
	events1 := conn1.GetEvents()
	events2 := conn2.GetEvents()

	hasPrivateMessage1 := false
	hasPrivateMessage2 := false

	for _, event := range events1 {
		if event.Event == "private_message" {
			hasPrivateMessage1 = true
			break
		}
	}

	for _, event := range events2 {
		if event.Event == "private_message" {
			hasPrivateMessage2 = true
			break
		}
	}

	assert.True(t, hasPrivateMessage1, "Client 1 should receive the targeted message")
	assert.False(t, hasPrivateMessage2, "Client 2 should NOT receive the targeted message")

	t.Log("✅ Targeted SSE messaging works correctly")
}

// TestSSEConnectionFlow tests the full SSE connection flow
func TestSSEConnectionFlow(t *testing.T) {
	if os.Getenv("E2E_TEST") == "" {
		t.Skip("Skipping E2E SSE test (set E2E_TEST=true to run)")
	}

	// First, test basic server connectivity
	resp, err := http.Get("http://localhost:7890/")
	if err != nil {
		t.Fatalf("❌ Server is not reachable: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("❌ Server returned status %d, expected 200", resp.StatusCode)
	}
	t.Log("✅ Basic server connectivity confirmed")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	roomName := "sse-test-flow-" + fmt.Sprintf("%d", time.Now().Unix())
	clientID := "flow_test_client"

	t.Logf("🔧 Testing SSE connection to room: %s, client: %s", roomName, clientID)

	// Create SSE connection
	conn, err := NewSSEConnection(roomName, clientID)
	require.NoError(t, err)
	defer conn.Close()

	// Start reading events
	conn.ReadEvents(ctx)

	// Wait for initial connection events
	time.Sleep(2 * time.Second)

	events := conn.GetEvents()

	// Debug: Log all received events
	t.Logf("📊 Total events received: %d", len(events))
	for i, event := range events {
		t.Logf("Event %d: Type='%s', ID='%s', Data=%+v", i, event.Event, event.ID, event.Data)
	}

	assert.NotEmpty(t, events, "Should receive initial connection events")

	// Check for connected event
	hasConnectedEvent := false
	hasWelcomeEvent := false

	for _, event := range events {
		if event.Event == "connected" {
			hasConnectedEvent = true

			if dataMap, ok := event.Data.(map[string]interface{}); ok {
				assert.Equal(t, "Successfully connected to SSE", dataMap["message"])
				assert.Equal(t, clientID, dataMap["user_id"])
				assert.Equal(t, roomName, dataMap["channel"])
			} else {
				t.Logf("⚠️ Connected event data is not a map: %T %+v", event.Data, event.Data)
			}
		} else if event.Event == "welcome" {
			hasWelcomeEvent = true
		}
	}

	if !hasConnectedEvent {
		t.Log("❌ No connected event found in received events")
	}
	if !hasWelcomeEvent {
		t.Log("❌ No welcome event found in received events")
	}

	assert.True(t, hasConnectedEvent, "Should receive connected event")
	assert.True(t, hasWelcomeEvent, "Should receive welcome event")

	t.Log("✅ SSE connection flow completed successfully")
}

// BenchmarkSSEConnections benchmarks SSE connection performance
func BenchmarkSSEConnections(b *testing.B) {
	if os.Getenv("E2E_TEST") == "" {
		b.Skip("Skipping E2E SSE benchmark (set E2E_TEST=true to run)")
	}

	// Test SSE endpoint availability before running benchmark
	testURL := "http://localhost:7890/demo-broadcasting/sse/simple?room=test&client_id=benchmark_test"
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		b.Skip("Failed to create SSE test request")
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := client.Do(req)
	if err != nil {
		b.Skipf("SSE server not available for benchmarking: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b.Skipf("SSE endpoint returned status %d, skipping benchmark", resp.StatusCode)
	}

	b.Run("SSE_Connection_establishment", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			roomName := fmt.Sprintf("bench-room-%d", i)
			clientID := fmt.Sprintf("bench-client-%d", i)

			conn, err := NewSSEConnection(roomName, clientID)
			if err != nil {
				b.Fatalf("Failed to connect: %v", err)
			}
			conn.Close()
		}
	})

	b.Run("SSE_Broadcast_performance", func(b *testing.B) {
		httpBaseURL := "http://localhost:7890"

		broadcastMsg := SSEBroadcastRequest{
			Event: "benchmark",
			Data: map[string]interface{}{
				"text": "benchmark message",
			},
			Broadcast: true,
		}

		jsonData, _ := json.Marshal(broadcastMsg)

		// Test broadcast endpoint availability
		testResp, err := http.Post(
			httpBaseURL+"/demo-broadcasting/sse/broadcast",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			b.Skipf("SSE broadcast endpoint not available: %v", err)
		}
		io.Copy(io.Discard, testResp.Body)
		testResp.Body.Close()

		if testResp.StatusCode != http.StatusOK {
			b.Skipf("SSE broadcast endpoint returned status %d, skipping benchmark", testResp.StatusCode)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := http.Post(
				httpBaseURL+"/demo-broadcasting/sse/broadcast",
				"application/json",
				bytes.NewBuffer(jsonData),
			)
			if err != nil {
				b.Fatalf("Failed to send broadcast: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// TestSSEConnectionLimits tests SSE connection limits and cleanup
func TestSSEConnectionLimits(t *testing.T) {
	if os.Getenv("E2E_TEST") == "" {
		t.Skip("Skipping E2E SSE test (set E2E_TEST=true to run)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test with a reasonable number of connections
	maxConnections := 25 // Lower than WebSocket since SSE uses more resources
	connections := make([]*SSEConnection, maxConnections)
	roomName := "sse-stress-test"

	t.Logf("Testing %d simultaneous SSE connections", maxConnections)

	// Create connections
	successfulConnections := 0
	for i := 0; i < maxConnections; i++ {
		clientID := fmt.Sprintf("stress_client_%d", i)

		conn, err := NewSSEConnection(roomName, clientID)
		if err != nil {
			t.Logf("Connection %d failed: %v", i, err)
			continue
		}

		connections[i] = conn
		conn.ReadEvents(ctx)
		successfulConnections++
	}

	t.Logf("✅ Successfully created %d/%d SSE connections", successfulConnections, maxConnections)

	// Clean up all connections
	for i, conn := range connections {
		if conn != nil {
			conn.Close()
			if i%5 == 0 {
				t.Logf("Closed %d SSE connections", i+1)
			}
		}
	}

	assert.GreaterOrEqual(t, successfulConnections, maxConnections*70/100,
		"At least 70%% of SSE connections should succeed")
}

// testSSEConnectivity tests if SSE server is responding properly
func testSSEConnectivity(b *testing.B) bool {
	testURL := "http://localhost:7890/demo-broadcasting/sse/simple?room=test&client_id=connectivity_test"

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		b.Logf("Failed to create SSE request: %v", err)
		return false
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		b.Logf("SSE connectivity test failed: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b.Logf("SSE endpoint returned status %d", resp.StatusCode)
		return false
	}

	// Check if it's actually an SSE stream
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		b.Logf("SSE endpoint doesn't return proper content type: %s", contentType)
		return false
	}

	return true
}
