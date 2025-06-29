package broadcast

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/daison12006013/turboscript/internal/server"
	"github.com/dop251/goja"
)

// BroadcastPlugin provides WebSocket and SSE broadcasting functionality.
type BroadcastPlugin struct {
	server *server.Server
}

// NewBroadcastPlugin creates a new broadcast plugin instance.
func NewBroadcastPlugin(s *server.Server) *BroadcastPlugin {
	return &BroadcastPlugin{
		server: s,
	}
}

// WebSocketMessage represents a WebSocket message to broadcast.
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	Room      string                 `json:"room"`
	Data      map[string]interface{} `json:"data"`
	Target    string                 `json:"target,omitempty"`    // Specific connection ID
	Broadcast bool                   `json:"broadcast,omitempty"` // Broadcast to all in room
}

// SSEMessage represents an SSE message to broadcast.
type SSEMessage struct {
	Event     string                 `json:"event"`
	Data      map[string]interface{} `json:"data"`
	ID        string                 `json:"id,omitempty"`
	Retry     int                    `json:"retry,omitempty"`
	Target    string                 `json:"target,omitempty"`    // Specific connection ID
	Broadcast bool                   `json:"broadcast,omitempty"` // Broadcast to all connections
	UserID    string                 `json:"user_id,omitempty"`   // Send to all connections for this user
}

// BroadcastWebSocket broadcasts a message to WebSocket connections.
func (p *BroadcastPlugin) BroadcastWebSocket(call goja.FunctionCall, runtime *goja.Runtime) goja.Value {
	if len(call.Arguments) < 1 {
		panic(runtime.NewTypeError("broadcastWebSocket requires at least 1 argument"))
	}

	// Parse the message object
	msgObj := call.Arguments[0].ToObject(runtime)
	msgJSON := msgObj.String()

	var msg WebSocketMessage
	if err := json.Unmarshal([]byte(msgJSON), &msg); err != nil {
		panic(runtime.NewTypeError("Invalid WebSocket message format: " + err.Error()))
	}

	// Set default values
	if msg.Type == "" {
		msg.Type = "broadcast"
	}
	if msg.Data == nil {
		msg.Data = make(map[string]interface{})
	}
	if _, exists := msg.Data["timestamp"]; !exists {
		msg.Data["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	}
	if _, exists := msg.Data["server_sent"]; !exists {
		msg.Data["server_sent"] = true
	}

	// Broadcast the message
	var connections int
	if msg.Target != "" {
		// Send to specific connection
		connections = p.server.BroadcastToConnection(msg.Target, msg.Type, msg.Data)
	} else if msg.Room != "" {
		// Send to all connections in room
		connections = p.server.BroadcastToRoom(msg.Room, msg.Type, msg.Data)
	} else {
		// Broadcast to all connections
		connections = p.server.BroadcastToAll(msg.Type, msg.Data)
	}

	// Return result
	result := make(map[string]interface{})
	result["success"] = true
	result["connections_notified"] = connections
	result["message_type"] = msg.Type
	result["room"] = msg.Room
	result["target"] = msg.Target

	return runtime.ToValue(result)
}

// BroadcastSSE broadcasts a message to SSE connections.
func (p *BroadcastPlugin) BroadcastSSE(call goja.FunctionCall, runtime *goja.Runtime) goja.Value {
	if len(call.Arguments) < 1 {
		panic(runtime.NewTypeError("broadcastSSE requires at least 1 argument"))
	}

	// Parse the message object
	msgObj := call.Arguments[0].ToObject(runtime)
	msgJSON := msgObj.String()

	var msg SSEMessage
	if err := json.Unmarshal([]byte(msgJSON), &msg); err != nil {
		panic(runtime.NewTypeError("Invalid SSE message format: " + err.Error()))
	}

	// Set default values
	if msg.Event == "" {
		msg.Event = "message"
	}
	if msg.Data == nil {
		msg.Data = make(map[string]interface{})
	}
	if msg.ID == "" {
		msg.ID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
	}
	if _, exists := msg.Data["timestamp"]; !exists {
		msg.Data["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	}
	if _, exists := msg.Data["server_sent"]; !exists {
		msg.Data["server_sent"] = true
	}

	// Broadcast the message
	var connections int
	if msg.Target != "" {
		// Send to specific connection
		connections = p.server.BroadcastSSEToConnection(msg.Target, msg.Event, msg.Data, msg.ID)
	} else if msg.UserID != "" {
		// Send to all connections for a specific user
		connections = p.server.BroadcastSSEToUser(msg.UserID, msg.Event, msg.Data, msg.ID)
	} else {
		// Broadcast to all SSE connections
		connections = p.server.BroadcastSSEToAll(msg.Event, msg.Data, msg.ID)
	}

	// Return result
	result := make(map[string]interface{})
	result["success"] = true
	result["connections_notified"] = connections
	result["event"] = msg.Event
	result["message_id"] = msg.ID
	result["user_id"] = msg.UserID
	result["target"] = msg.Target

	return runtime.ToValue(result)
}

// GetConnections returns information about active connections.
func (p *BroadcastPlugin) GetConnections(call goja.FunctionCall, runtime *goja.Runtime) goja.Value {
	var filter string
	if len(call.Arguments) > 0 {
		filter = call.Arguments[0].String()
	}

	// Get connection statistics
	stats := p.server.GetConnectionStats(filter)

	return runtime.ToValue(stats)
}

// RegisterPlugin registers the broadcast plugin with the plugin manager.
func RegisterPlugin() map[string]interface{} {
	return map[string]interface{}{
		"name":        "broadcast",
		"version":     "1.0.0",
		"description": "WebSocket and SSE broadcasting plugin",
		"methods": map[string]interface{}{
			"websocket":   "BroadcastWebSocket",
			"sse":         "BroadcastSSE",
			"connections": "GetConnections",
		},
	}
}
