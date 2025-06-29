import { meta } from "../../../utils/meta";

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const input = event.body as {
            event?: string;
            data?: Record<string, unknown>;
            user_id?: string;
            target?: string;
            broadcast?: boolean;
            id?: string;
            retry?: number;
        };

        // Validation
        if (!input.event) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Validation failed",
                    errors: ["Event type is required"],
                    ...meta(event),
                }
            };
        }

        if (!input.data) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Validation failed",
                    errors: ["Data is required"],
                    ...meta(event),
                }
            };
        }

        // Prepare SSE broadcast message
        const sseMessage = {
            event: input.event,
            data: input.data,
            id: input.id,
            retry: input.retry,
            target: input.target,
            broadcast: input.broadcast !== false, // Default to true
            user_id: input.user_id
        };

        // Log the broadcast attempt
        console.log(`Broadcasting SSE message: ${JSON.stringify({
            event: sseMessage.event,
            target: sseMessage.target ?? 'all',
            user_id: sseMessage.user_id ?? 'none',
            broadcast: sseMessage.broadcast
        })}`);

        // Broadcast the SSE message using global function
        const result = await turboBroadcastSSE(sseMessage);

        return {
            code: 200,
            response: {
                status: "success",
                message: `SSE broadcast sent successfully`,
                data: {
                    event: sseMessage.event,
                    connections_notified: result.connections_notified,
                    success: result.success,
                    message_id: result.message_id,
                    target: sseMessage.target,
                    user_id: sseMessage.user_id,
                    broadcast: sseMessage.broadcast
                },
                ...meta(event),
            }
        };

    } catch (error) {
        console.error('SSE Broadcast Error:', error);
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "SSE broadcast failed",
                ...meta(event),
            }
        };
    }
};
