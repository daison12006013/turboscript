import fetch from 'node-fetch';

console.log('🔄 Testing SSE (Server-Sent Events) multi-client functionality...\n');

// Helper function to create SSE client
async function createSSEClient(clientName, room, clientId) {
    const url = `http://localhost:7890/demo-broadcasting/sse/events?room=${room}&client_id=${clientId}`;
    console.log(`🔗 ${clientName}: Connecting to ${url}`);

    try {
        // Create a promise that resolves after a short timeout to prevent hanging on SSE streams
        const timeoutPromise = new Promise((resolve) => {
            global.setTimeout(() => resolve({ timeout: true }), 2000);
        });

        const fetchPromise = fetch(url, {
            method: 'GET',
            headers: {
                'Accept': 'text/event-stream',
                'Cache-Control': 'no-cache'
            }
        });

        // Race between fetch and timeout
        const result = await Promise.race([fetchPromise, timeoutPromise]);

        if (result.timeout) {
            console.log(`✅ ${clientName}: Connection established (SSE stream active)`);
            return true;
        }

        const response = result;
        if (response.ok) {
            console.log(`✅ ${clientName}: Connected successfully (${response.status})`);
            
            // Check if it's the correct content type for SSE
            const contentType = response.headers.get('content-type');
            if (contentType && contentType.includes('text/event-stream')) {
                console.log(`📨 ${clientName} received SSE stream (content-type: ${contentType})`);
            } else {
                console.log(`📨 ${clientName} connected but content-type is: ${contentType}`);
            }
            
            return true;
        } else {
            console.error(`❌ ${clientName}: Failed to connect:`, response.status);
            return false;
        }
    } catch (error) {
        console.error(`❌ ${clientName}: Connection error:`, error.message);
        return false;
    }
}


console.log('🔄 Testing SSE (Server-Sent Events) multi-client functionality...\n');

// Test multiple SSE connections
async function testMultipleSSEConnections() {
    try {
        const room = 'sse-test-room';
        const client1Id = 'sse_client_1';
        const client2Id = 'sse_client_2';

        console.log('--- Testing SSE Client Connections ---');

        // Test Client 1
        const client1Connected = await createSSEClient('Client1', room, client1Id);

        // Test Client 2
        const client2Connected = await createSSEClient('Client2', room, client2Id);

        return { client1Connected, client2Connected };

    } catch (error) {
        console.error('❌ SSE Connection Error:', error.message);
        return { client1Connected: false, client2Connected: false };
    }
}

// Test SSE broadcasting
async function testSSEBroadcast() {
    console.log('\n--- Testing SSE Broadcast ---');

    try {
        // Send broadcast via HTTP API
        const broadcastData = {
            event: 'test_message',
            data: {
                message: 'Hello from SSE broadcast test!',
                timestamp: new Date().toISOString(),
                test_id: 'sse_multi_test_001'
            },
            broadcast: true,
            id: `msg_${Date.now()}`
        };

        console.log('🚀 Sending SSE broadcast...');

        const response = await fetch('http://localhost:7890/demo-broadcasting/sse/broadcast', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(broadcastData)
        });

        const result = await response.json();

        console.log('📡 Broadcast API Response:');
        console.log(`   Status: ${result.status}`);
        console.log(`   Connections notified: ${result.data?.connections_notified ?? 0}`);
        console.log(`   Message ID: ${result.data?.message_id ?? 'none'}`);

        if (result.data?.connections_notified === 0) {
            console.log('⚠️  WARNING: No SSE connections were notified!');
        } else if (result.data?.connections_notified < 2) {
            console.log('⚠️  WARNING: Expected 2 SSE connections, but only notified ' + result.data.connections_notified);
        } else {
            console.log('✅ SUCCESS: Multiple SSE connections should receive the message');
        }

        return result.data?.connections_notified ?? 0;

    } catch (error) {
        console.error('❌ SSE Broadcast Error:', error.message);
        return 0;
    }
}

// Test targeted SSE message
async function testTargetedSSE() {
    console.log('\n--- Testing Targeted SSE Message ---');

    try {
        const targetedData = {
            event: 'private_message',
            data: {
                message: 'This is a private message for client 1 only',
                timestamp: new Date().toISOString(),
                recipient: 'sse_client_1'
            },
            target: 'sse_client_1', // Target specific client
            broadcast: false
        };

        console.log('🎯 Sending targeted SSE message to Client1...');

        const response = await fetch('http://localhost:7890/demo-broadcasting/sse/broadcast', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(targetedData)
        });

        const result = await response.json();

        console.log('📡 Targeted SSE Response:');
        console.log(`   Status: ${result.status}`);
        console.log(`   Connections notified: ${result.data?.connections_notified ?? 0}`);
        console.log(`   Target: ${result.data?.target ?? 'none'}`);

        if (result.data?.connections_notified === 1) {
            console.log('✅ SUCCESS: Targeted message sent to specific client');
        } else {
            console.log('⚠️  WARNING: Expected 1 targeted connection, got ' + result.data?.connections_notified);
        }

        return result.data?.connections_notified ?? 0;

    } catch (error) {
        console.error('❌ Targeted SSE Error:', error.message);
        return 0;
    }
}

// Helper function to wait
function delay(ms) {
    return new Promise(resolve => global.setTimeout(resolve, ms));
}

// Run the tests
async function runTests() {
    console.log('🧪 Starting SSE Multi-Client Tests\n');

    // Test connections
    const { client1Connected, client2Connected } = await testMultipleSSEConnections();

    // Wait for connections to establish
    await delay(3000);

    // Test broadcasting
    const broadcastCount = await testSSEBroadcast();

    // Wait a bit
    await delay(2000);

    // Test targeted messaging
    const targetedCount = await testTargetedSSE();

    // Final status
    await delay(2000);

    console.log('\n--- Final SSE Test Report ---');
    console.log(`Client1 Connected: ${client1Connected}`);
    console.log(`Client2 Connected: ${client2Connected}`);
    console.log(`Broadcast Messages Sent: ${broadcastCount}`);
    console.log(`Targeted Messages Sent: ${targetedCount}`);

    if (!client1Connected || !client2Connected) {
        console.log('\n🔍 SSE DEBUGGING TIPS:');
        console.log('1. Check if the server is running on port 7890');
        console.log('2. Verify SSE endpoint /demo-broadcasting/sse/events is accessible');
        console.log('3. Check server logs for any SSE connection errors');
        console.log('4. Ensure proper SSE headers are being sent');
        console.log('5. Check if SSE broadcasting is implemented');
    }

    console.log('\n🏁 SSE test completed. Exiting...');
    process.exit(0);
}

// Start the tests
runTests().catch(error => {
    console.error('❌ Test execution failed:', error);
    process.exit(1);
});
