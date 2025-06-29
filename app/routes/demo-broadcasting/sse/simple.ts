export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const userInfo = event.queryParameters;
        const userId = userInfo.user_id || 'anonymous';
        const channel = userInfo.channel || 'general';

        console.log(`SSE connection from user: ${userId}, channel: ${channel}`);

        // Create the initial SSE response with proper formatting
        const sseEvents = [
            // Connection event
            `event: connected\ndata: ${JSON.stringify({
                message: 'Successfully connected to SSE',
                user_id: userId,
                channel: channel,
                server_time: new Date().toISOString()
            })}\n\n`,

            // Welcome message
            `event: welcome\ndata: ${JSON.stringify({
                message: `Welcome ${userId}! You are now receiving real-time updates.`,
                tips: [
                    'Open multiple browser tabs to test broadcasting',
                    'Use the broadcast button to send messages to all connected clients',
                    'Check the network tab to see SSE connection details'
                ]
            })}\n\n`,

            // Initial time update
            `event: time_update\ndata: ${JSON.stringify({
                current_time: new Date().toISOString(),
                message: 'This timestamp shows when you connected'
            })}\n\n`
        ].join('');

        return {
            code: 200,
            response: sseEvents,
            sse: {
                event: 'connected',
                data: {
                    message: 'Successfully connected to SSE',
                    user_id: userId,
                    channel: channel,
                    server_time: new Date().toISOString()
                }
            }
        };

    } catch (error) {
        console.error('SSE connection error:', error);
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "SSE connection failed"
            }
        };
    }
};
