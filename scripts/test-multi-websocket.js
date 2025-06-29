import WebSocket from 'ws';

console.log('🔄 Testing multiple WebSocket connections to debug broadcast issue...\n');

let ws1Connected = false;
let ws2Connected = false;
let ws1Joined = false;
let ws2Joined = false;

// First WebSocket Connection
const ws1 = new WebSocket('ws://localhost:7890/ws');

ws1.on('open', function() {
    console.log('🔗 WS1: Connected successfully');
    ws1Connected = true;

    // Join the room
    ws1.send(JSON.stringify({
        type: 'join',
        room: 'room-test123'
    }));
    console.log('📩 WS1: Sent join request for room-test123');
});

ws1.on('message', function(data) {
    const msg = JSON.parse(data.toString());
    console.log(`📨 WS1 Received: ${msg.type} ${msg.room ? `(room: ${msg.room})` : ''}`);

    if (msg.type === 'joined') {
        ws1Joined = true;
        console.log('✅ WS1: Successfully joined room');
    }
});

ws1.on('error', function(err) {
    console.error('❌ WS1 Error:', err.message);
});

ws1.on('close', function() {
    console.log('🔌 WS1: Connection closed');
});

// Second WebSocket Connection (delayed)
setTimeout(() => {
    console.log('\n--- Starting second WebSocket connection ---');

    const ws2 = new WebSocket('ws://localhost:7890/ws');

    ws2.on('open', function() {
        console.log('🔗 WS2: Connected successfully');
        ws2Connected = true;

        // Join the same room
        ws2.send(JSON.stringify({
            type: 'join',
            room: 'room-test123'
        }));
        console.log('📩 WS2: Sent join request for room-test123');
    });

    ws2.on('message', function(data) {
        const msg = JSON.parse(data.toString());
        console.log(`📨 WS2 Received: ${msg.type} ${msg.room ? `(room: ${msg.room})` : ''}`);

        if (msg.type === 'joined') {
            ws2Joined = true;
            console.log('✅ WS2: Successfully joined room');
        }
    });

    ws2.on('error', function(err) {
        console.error('❌ WS2 Error:', err.message);
    });

    ws2.on('close', function() {
        console.log('🔌 WS2: Connection closed');
    });

}, 2000);

// Test broadcast after both connections are established
setTimeout(() => {
    console.log('\n--- Testing broadcast to both connections ---');
    console.log(`Status: WS1 Connected: ${ws1Connected}, Joined: ${ws1Joined}`);
    console.log(`Status: WS2 Connected: ${ws2Connected}, Joined: ${ws2Joined}`);

    if (ws1Joined && ws2Joined) {
        console.log('🚀 Both clients joined. Broadcasting message...\n');

        // Send broadcast via HTTP API
        fetch('http://localhost:7890/demo-broadcasting/websocket/broadcast', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                room: 'room-test123',
                type: 'test',
                message: {
                    text: 'Test broadcast to multiple clients',
                    timestamp: new Date().toISOString()
                }
            })
        })
        .then(response => response.json())
        .then(data => {
            console.log('📡 Broadcast API Response:');
            console.log(`   Status: ${data.status}`);
            console.log(`   Connections notified: ${data.data?.connections_notified || 0}`);

            if (data.data?.connections_notified === 0) {
                console.log('⚠️  WARNING: No connections were notified!');
            } else if (data.data?.connections_notified < 2) {
                console.log('⚠️  WARNING: Expected 2 connections, but only notified ' + data.data.connections_notified);
            } else {
                console.log('✅ SUCCESS: Both connections should receive the message');
            }
        })
        .catch(err => {
            console.error('❌ Broadcast API Error:', err.message);
        });
    } else {
        console.log('⚠️  Not all clients joined successfully. Cannot test broadcast.');
    }
}, 5000);

// Status check
setTimeout(() => {
    console.log('\n--- Final Status Report ---');
    console.log(`WS1: Connected: ${ws1Connected}, Joined: ${ws1Joined}`);
    console.log(`WS2: Connected: ${ws2Connected}, Joined: ${ws2Joined}`);

    if (!ws1Joined || !ws2Joined) {
        console.log('\n🔍 DEBUGGING TIPS:');
        console.log('1. Check if the server is running on port 7890');
        console.log('2. Verify WebSocket endpoint /ws is accessible');
        console.log('3. Check server logs for any room join errors');
        console.log('4. Make sure room pattern "room-test123" matches server config');
    }
}, 8000);

// Clean exit
setTimeout(() => {
    console.log('\n🏁 Test completed. Exiting...');
    process.exit(0);
}, 10000);
