# How to Use Server-Sent Events (SSE) in TurboScript

Server-Sent Events (SSE) provide a simple way to receive real-time updates from your server. Unlike WebSockets, SSE is a one-way communication channel where the server can push data to the client.

## What is SSE?

SSE allows your server to automatically send data to a web page. This is perfect for:

- Real-time notifications
- Live dashboards
- Chat applications
- Stock price updates
- Social media feeds
- Live sports scores

## How SSE Works

1. **Client connects** to an SSE endpoint
2. **Server keeps the connection open** and can send events at any time
3. **Client receives events** in real-time through JavaScript

## Demo

Visit the interactive demo at: **<http://localhost:7890/demo/sse>**

## Basic Usage

### 1. Connect to SSE Stream (Client-side JavaScript)

```javascript
// Connect to SSE endpoint
const eventSource = new EventSource('/demo/sse/simple?user_id=my_user&channel=general');

// Listen for connection events
eventSource.addEventListener('connected', function(event) {
    const data = JSON.parse(event.data);
    console.log('Connected:', data.message);
});

// Listen for welcome messages
eventSource.addEventListener('welcome', function(event) {
    const data = JSON.parse(event.data);
    console.log('Welcome:', data.message);
});

// Listen for custom events
eventSource.addEventListener('test_message', function(event) {
    const data = JSON.parse(event.data);
    console.log('Received message:', data.message);
});

// Handle connection errors
eventSource.onerror = function() {
    console.log('SSE connection error');
};

// Close connection when done
// eventSource.close();
```

### 2. Server-side SSE Endpoint (TypeScript)

```typescript
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const userId = event.queryParameters.user_id || 'anonymous';
    const channel = event.queryParameters.channel || 'general';

    // Create SSE response with multiple events
    const sseEvents = [
        // Connection confirmation
        \`event: connected\\ndata: \${JSON.stringify({
            message: 'Successfully connected',
            user_id: userId,
            channel: channel,
            timestamp: new Date().toISOString()
        })}\\n\\n\`,

        // Welcome message
        \`event: welcome\\ndata: \${JSON.stringify({
            message: \`Welcome \${userId}!\`,
            tips: ['Open multiple tabs to test', 'Use broadcast API for real-time messaging']
        })}\\n\\n\`
    ].join('');

    return {
        code: 200,
        type: 'text',
        response: sseEvents
    };
};
```

### 3. Broadcasting Messages to All Connected Clients

Send a POST request to broadcast messages to all SSE connections:

```bash
curl -X POST http://localhost:7890/api/sse/broadcast \\
  -H "Content-Type: application/json" \\
  -d '{
    "event": "notification",
    "data": {
      "message": "Hello everyone!",
      "type": "announcement"
    },
    "broadcast": true
  }'
```

Or from TypeScript code:

```typescript
// Broadcast to all SSE connections
const result = await turboBroadcastSSE({
    event: 'notification',
    data: {
        message: 'New update available!',
        priority: 'high'
    },
    broadcast: true
});

console.log(\`Notified \${result.connections_notified} connections\`);
```

## Available SSE Events

The demo includes these event types:

- **`connected`** - Sent when client first connects
- **`welcome`** - Welcome message with tips
- **`time_update`** - Periodic timestamp updates
- **`test_message`** - Custom messages from broadcast API

## Key Benefits of SSE

✅ **Simple to implement** - No complex protocols like WebSockets
✅ **Automatic reconnection** - Browser handles reconnects automatically
✅ **Built-in error handling** - Standard HTTP error codes
✅ **Firewall friendly** - Uses standard HTTP
✅ **Lightweight** - Lower overhead than WebSockets for one-way communication

## When to Use SSE vs WebSockets

**Use SSE when:**

- You only need server-to-client communication
- Real-time notifications
- Live data feeds
- Simple chat applications (receive-only)

**Use WebSockets when:**

- You need bi-directional communication
- Real-time gaming
- Collaborative editing
- Complex interactive applications

## Testing Your SSE Implementation

1. **Open the demo**: <http://localhost:7890/demo/sse>
2. **Open multiple browser tabs** to see multi-client support
3. **Use the broadcast button** to send messages to all connected clients
4. **Check browser Network tab** to see the SSE connection details

## Troubleshooting

- **Connection not working?** Check that your endpoint returns proper SSE format
- **Events not received?** Ensure event names match between server and client
- **Reconnection issues?** Browser will auto-reconnect unless you call `eventSource.close()`

---

**Try the interactive demo now**: <http://localhost:7890/demo/sse>

The demo shows real-time SSE in action with multiple clients, broadcasting, and practical examples!
