import { meta } from "../../../utils/meta";

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Get room from query parameters or default to 'general'
        const room = event.queryParameters.room || 'general';
        const clientId = event.queryParameters.client_id || `client_${Date.now()}`;

        console.log(`SSE client connecting to room: ${room}, client_id: ${clientId}`);

        // Create initial SSE response with proper formatting
        const sseEvents = [
            // Connection event
            `event: connected\ndata: ${JSON.stringify({
                type: 'connected',
                room: room,
                client_id: clientId,
                message: `Connected to SSE stream for room: ${room}`,
                timestamp: new Date().toISOString()
            })}\n\n`,

            // Welcome message
            `event: welcome\ndata: ${JSON.stringify({
                type: 'welcome',
                room: room,
                client_id: clientId,
                message: 'Welcome to TurboScript SSE stream!',
                timestamp: new Date().toISOString()
            })}\n\n`
        ].join('');

        return {
            code: 200,
            response: sseEvents,
            sse: {
                event: 'connected',
                data: {
                    type: 'connected',
                    room: room,
                    client_id: clientId,
                    message: `Connected to SSE stream for room: ${room}`,
                    timestamp: new Date().toISOString()
                }
            }
        };

    } catch (error) {
        console.error('SSE Error:', error);
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "SSE connection failed",
                ...meta(event),
            }
        };
    }
};
