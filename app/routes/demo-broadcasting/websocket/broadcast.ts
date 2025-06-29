import { meta } from "../../../utils/meta";

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const input = event.body as {
            room?: string;
            message?: Record<string, unknown>;
            type?: string;
            from?: string;
        };

        // Validation
        if (!input.room) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Validation failed",
                    errors: ["Room is required"],
                    ...meta(event),
                }
            };
        }

        if (!input.message) {
            return {
                code: 422,
                response: {
                    status: "error",
                    message: "Validation failed",
                    errors: ["Message is required"],
                    ...meta(event),
                }
            };
        }

        // Prepare broadcast data
        const broadcastData = {
            from: input.from ?? "server",
            text: input.message.text ?? "Test message from server",
            timestamp: new Date().toISOString(),
            server_sent: true,
            ...input.message
        };

        // Broadcast via turboBroadcastWebSocket function
        const result = await turboBroadcastWebSocket({
            type: input.type ?? "server_message",
            room: input.room,
            data: broadcastData,
            broadcast: true
        });

        return {
            code: 200,
            response: {
                status: "success",
                message: "Message broadcasted successfully",
                data: {
                    room: input.room,
                    type: input.type ?? "server_message",
                    broadcast_data: broadcastData,
                    connections_notified: result.connections_notified,
                },
                ...meta(event),
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "An unexpected error occurred",
                ...meta(event),
            }
        };
    }
};
