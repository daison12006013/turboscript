# WebSocket and Server-Sent Events (SSE) Support

TurboScript provides comprehensive support for real-time communication through WebSocket and Server-Sent Events (SSE). This enables you to build interactive applications with live updates, chat systems, notifications, and more.

## WebSocket Configuration

WebSocket endpoints support channel-based routing with flexible room patterns and connection management.

### Basic WebSocket Endpoint

```yaml
endpoints:
  - route: /ws
    method: WebSocket
    type: websocket
    websocket:
      enable_presence: true
      enable_redis: true
      redis_channel: "turboscript_ws"
      ping_interval: 30
      pong_timeout: 60
      max_message_size: 524288  # 512KB
      enable_compression: true
      compression_level: 6
      channels:
        - room: "public-(.*)"
          type: "public"
          handle: ./app/websockets/public
          max_connections: 0       # Unlimited

        - room: "room-(.*)"
          type: "private"
          handle: ./app/websockets/private-room
          max_connections: 10

        - room: "presence-(.*)"
          type: "presence"
          handle: ./app/websockets/presence
          max_connections: 500
```

### WebSocket Configuration Options

#### Global WebSocket Settings

- **`enable_presence`**: Enable presence tracking for connections
- **`enable_redis`**: Enable Redis for sticky sessions and scaling
- **`enable_kafka`**: Enable Kafka integration for cross-instance message distribution (optional)
- **`kafka_brokers`**: List of Kafka broker addresses (required when `enable_kafka: true`)
- **`kafka_topic`**: Kafka topic for WebSocket messages (required when `enable_kafka: true`)
- **`redis_channel`**: Redis channel name for pub/sub
- **`ping_interval`**: Ping interval in seconds (default: 30)
- **`pong_timeout`**: Pong timeout in seconds (default: 60)
- **`max_message_size`**: Maximum message size in bytes (default: 512KB)
- **`enable_compression`**: Enable per-message compression
- **`compression_level`**: Compression level 1-9 (default: 6)

#### Channel Configuration

- **`room`**: Regex pattern for room matching (e.g., "public-(.*)", "room-([0-9]+)")
- **`type`**: Channel type (`public`, `private`, `presence`)
- **`handle`**: Path to TypeScript handler file
- **`max_connections`**: Maximum connections per room (0 = unlimited)

### Channel Types

#### Public Channels

- Open to all users
- No authentication required to join
- Example: `public-general`, `public-announcements`

#### Private Channels

- Require authentication and permission checking
- Limited access based on user permissions
- Example: `room-meeting-123`, `room-project-456`

#### Presence Channels

- Track who is online in the channel
- Broadcast join/leave events to other members
- Example: `presence-lobby`, `presence-dashboard`

## WebSocket Message Types

TurboScript supports several built-in WebSocket message types that are handled automatically by the server. All messages must include a `type` field to specify the message type.

### Core Message Types

#### `join` - Join a Room

Connects the WebSocket client to a specific room.

```json
{
  "type": "join",
  "room": "public-general"
}
```

**Response:**

```json
{
  "type": "joined",
  "room": "public-general",
  "data": {"status": "success"},
  "timestamp": "2025-07-15T14:00:00.000Z"
}
```

#### `leave` - Leave a Room

Disconnects the WebSocket client from their current room.

```json
{
  "type": "leave",
  "room": "public-general"
}
```

#### `message` - Send Room Message

Sends a message to all members of the current room.

```json
{
  "type": "message",
  "room": "public-general",
  "data": {
    "text": "Hello everyone!",
    "type": "text"
  }
}
```

**Broadcast to Room:**

```json
{
  "type": "new_message",
  "room": "public-general",
  "data": {
    "id": "msg_1234567890",
    "user": {
      "id": "user123",
      "name": "John Doe",
      "avatar": "https://example.com/avatar.jpg"
    },
    "text": "Hello everyone!",
    "timestamp": "2025-07-15T14:00:00.000Z",
    "type": "text"
  }
}
```

### Extended Message Types

#### `typing` - Typing Indicator

Shows/hides typing indicator for the user in the room.

```json
{
  "type": "typing",
  "room": "public-general",
  "data": {
    "typing": true
  }
}
```

#### `get_history` - Retrieve Chat History

Requests the chat history for the current room.

```json
{
  "type": "get_history",
  "room": "public-general"
}
```

#### Custom Message Types

Any message type not listed above will be passed to the `handleCustomMessage` function in your WebSocket handler, allowing you to implement custom functionality:

```json
{
  "type": "custom_action",
  "room": "public-general",
  "data": {
    "action": "vote",
    "poll_id": "poll123",
    "choice": "option_a"
  }
}
```

### Message Structure

All WebSocket messages follow this structure:

```typescript
interface WebSocketMessage {
  type: string;                    // Message type (required)
  room?: string;                   // Target room
  data?: any;                      // Message payload
  user_id?: string;                // Sender user ID (set by server)
  message_id?: string;             // Unique message ID (set by server)
  timestamp?: string;              // Message timestamp (set by server)
  metadata?: Record<string, any>;  // Additional metadata
}
```

### Error Handling

If a message is invalid or cannot be processed, the server will respond with an error message:

```json
{
  "type": "error",
  "data": {
    "message": "Room name is required"
  },
  "timestamp": "2025-07-15T14:00:00.000Z"
}
```

## WebSocket Handler Pattern

WebSocket handlers receive an Event object with WebSocket-specific properties:

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const eventType = event.type;        // 'connect', 'join', 'leave', 'message', etc.
    const connection = event.connection; // WebSocket connection details
    const message = event.message;       // Incoming message data

    switch (eventType) {
        case 'connect':
            // Handle new WebSocket connection
            return {
                code: 200,
                response: {
                    status: "connected",
                    connection_id: connection?.id,
                }
            };

        case 'join':
            // Handle room join request
            return {
                code: 200,
                response: { status: "joined" },
                websocket: {
                    type: "room_joined",
                    room: message?.room,
                    data: { user: connection?.user_data },
                    broadcast: true,
                }
            };

        case 'message':
            // Handle incoming chat message
            return {
                code: 200,
                response: { status: "message_sent" },
                websocket: {
                    type: "new_message",
                    room: message?.room,
                    data: {
                        text: message?.data?.text,
                        user: connection?.user_data,
                        timestamp: new Date().toISOString(),
                    },
                    broadcast: true,
                }
            };
    }
};
```

### WebSocket Response Options

- **`type`**: Message type for the client
- **`room`**: Target room for the message
- **`data`**: Message payload
- **`broadcast`**: Send to all room members (default: false)
- **`target`**: Send to specific connection ID
- **`error`**: Error message for the client

## Server-Sent Events (SSE) Configuration

SSE provides unidirectional real-time communication from server to client, perfect for notifications and live updates.

### Basic SSE Endpoint

```yaml
endpoints:
  - route: /sse/notifications
    method: SSE
    type: sse
    path: ./app/sse/notifications
    sse:
      enable_http2: true
      keepalive_interval: 30
      retry: 3000
      max_connections: 1000
      buffer_size: 2048
      enable_compression: true
      allowed_origins:
        - "http://localhost:7890"
        - "https://yourdomain.com"
```

### SSE Configuration Options

- **`enable_http2`**: Force HTTP/2 for better performance
- **`keepalive_interval`**: Keep-alive ping interval in seconds
- **`retry`**: Client retry interval in milliseconds
- **`max_connections`**: Maximum concurrent SSE connections
- **`buffer_size`**: Message buffer size
- **`enable_compression`**: Enable gzip compression
- **`allowed_origins`**: CORS allowed origins

## SSE Handler Pattern

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const eventType = event.type;
    const connection = event.connection;

    switch (eventType) {
        case 'connect':
            // Client connected to SSE endpoint
            return {
                code: 200,
                response: { status: "connected" },
                sse: {
                    event: "connected",
                    data: {
                        connection_id: connection?.id,
                        server_time: new Date().toISOString(),
                    }
                }
            };

        case 'subscribe':
            // Client wants to subscribe to channels
            const channels = event.data?.channels;
            return {
                code: 200,
                response: { status: "subscribed" },
                sse: {
                    event: "subscription_confirmed",
                    data: { subscribed_channels: channels }
                }
            };
    }
};
```

### SSE Response Options

- **`event`**: Event type for the client
- **`data`**: Event payload (JSON serialized)
- **`id`**: Message ID for client tracking
- **`retry`**: Override retry interval
- **`target`**: Send to specific connection ID
- **`broadcast`**: Send to all connected clients
- **`user_id`**: Send to all connections for a user

## Client-Side Integration

### WebSocket Client Example

```javascript
const ws = new WebSocket('ws://localhost:7890/ws');

ws.onopen = function() {
    // Join a public room
    ws.send(JSON.stringify({
        type: 'join',
        room: 'public-general',
        data: {}
    }));
};

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('Received:', message);

    if (message.type === 'new_message') {
        displayMessage(message.data);
    }
};

// Send a chat message
function sendMessage(text) {
    ws.send(JSON.stringify({
        type: 'message',
        room: 'public-general',
        data: { text: text }
    }));
}
```

### SSE Client Example

```javascript
const eventSource = new EventSource('/sse/notifications');

eventSource.addEventListener('connected', function(event) {
    const data = JSON.parse(event.data);
    console.log('Connected:', data.connection_id);
});

eventSource.addEventListener('notification', function(event) {
    const notification = JSON.parse(event.data);
    showNotification(notification);
});

eventSource.addEventListener('unread_count', function(event) {
    const data = JSON.parse(event.data);
    updateUnreadBadge(data.count);
});

eventSource.onerror = function(event) {
    console.error('SSE connection error:', event);
};
```

## Database Integration

Both WebSocket and SSE handlers have full access to the database through `turboQuery`:

```typescript
// Store chat message
await turboQuery(
    'INSERT INTO chat_messages (room, user_id, message, created_at) VALUES ($1, $2, $3, NOW())',
    [room, userId, messageText]
);

// Get unread notifications
const unread = await turboQuery(
    'SELECT COUNT(*) as count FROM notifications WHERE user_id = $1 AND read = false',
    [userId]
);
```

## Scaling Considerations

### Local Operation (Default)

By default, WebSocket and SSE operate locally on a single instance without any external dependencies:

```yaml
websocket:
  enable_redis: false  # Default: false
  enable_kafka: false  # Default: false
```

This is perfect for development and single-instance deployments.

### Redis Integration (Optional)

Enable Redis for horizontal scaling across multiple server instances:

```yaml
websocket:
  enable_redis: true
  redis_channel: "turboscript_ws"
```

**When Redis is enabled:**

- Messages are published to Redis pub/sub for cross-instance communication
- Sticky sessions can be managed across instances
- Suitable for moderate scaling scenarios

### Kafka Integration (Optional)

For high-throughput scenarios and maximum horizontal scaling, enable Kafka:

```yaml
websocket:
  enable_kafka: true
  kafka_brokers:
    - "localhost:9092"
    - "kafka-node-2:9092"
  kafka_topic: "turboscript_messages"
```

**When Kafka is enabled:**

- Messages are published to Kafka topics for cross-instance distribution
- Supports unlimited horizontal scaling
- Provides message durability and replay capabilities
- Recommended for production environments with multiple instances

**When Kafka is disabled (`enable_kafka: false`):**

- All messages are handled locally within the single instance
- No external Kafka dependency required
- Perfect for development, testing, and single-server deployments
- WebSocket rooms and SSE broadcasts work normally within the instance

### Scaling Decision Matrix

| Scenario                      | Redis | Kafka | Use Case                      |
|-------------------------------|-------|-------|-------------------------------|
| Single Instance               | ❌    | ❌    | Development, small apps       |
| Multi-Instance (Low Traffic)  | ✅    | ❌    | Small production deployments  |
| Multi-Instance (High Traffic) | ❌    | ✅    | Large production deployments  |
| Enterprise Scale              | ✅    | ✅    | Maximum reliability and scale |

## Security Best Practices

1. **Authentication**: Verify user tokens in handlers
2. **Room Permissions**: Check user access to private rooms
3. **Rate Limiting**: Implement message rate limiting
4. **Input Validation**: Validate all incoming data
5. **CORS**: Configure allowed origins for SSE

## Error Handling

Both WebSocket and SSE handlers should include comprehensive error handling:

```typescript
try {
    // Handler logic
} catch (error) {
    console.error('Handler error:', error);
    return {
        code: 500,
        response: {
            status: "error",
            message: "Internal server error",
        },
        websocket: {
            type: "error",
            data: { message: "An error occurred" }
        }
    };
}
```

## Performance Tips

1. **Connection Limits**: Set appropriate `max_connections` per room
2. **Message Size**: Limit `max_message_size` to prevent abuse
3. **Compression**: Enable compression for large messages
4. **HTTP/2**: Use HTTP/2 for SSE for better multiplexing
5. **Buffer Size**: Tune buffer sizes based on message volume

## Use Cases

### WebSocket Use Cases

- Real-time chat applications
- Live collaborative editing
- Gaming applications
- Live dashboards
- Interactive notifications

### SSE Use Cases

- Live notifications
- Real-time analytics dashboards
- Live news feeds
- Status updates
- Progress indicators

## Migration from HTTP Polling

Replace inefficient HTTP polling with efficient real-time connections:

**Before (HTTP Polling):**

```javascript
setInterval(() => {
    fetch('/api/notifications')
        .then(r => r.json())
        .then(data => updateUI(data));
}, 5000);
```

**After (SSE):**

```javascript
const eventSource = new EventSource('/sse/notifications');
eventSource.addEventListener('notification', (event) => {
    const data = JSON.parse(event.data);
    updateUI(data);
});
```

This reduces server load and provides instant updates to users.
