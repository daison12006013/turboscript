// Private WebSocket channel handler for rooms like "room-12345", "room-thread-67890"
// This handles private chat rooms with limited access and authentication

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Type guard to ensure we have a WebSocket context
        if (event.context.type !== 'websocket') {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Invalid context type for WebSocket handler"
                }
            };
        }

        const wsContext = event.context;
        const { eventType, connection, message, room } = wsContext;

        // Private rooms require authentication
        if (!connection.user_id) {
            return {
                code: 401,
                response: {
                    status: "error",
                    message: "Authentication required for private rooms"
                }
            };
        }

        switch (eventType) {
            case 'connect':
                return {
                    code: 200,
                    response: {
                        status: "connected",
                        message: "Connected to private room handler",
                        connection_id: connection.id,
                    }
                };

            case 'join': {
                if (!room?.startsWith('room-')) {
                    return {
                        code: 400,
                        response: {
                            status: "error",
                            message: "Invalid private room name. Must start with 'room-'",
                        }
                    };
                }

                // Check if user has permission to join this room
                const hasAccess = await checkRoomAccess(connection.user_id, room);
                if (!hasAccess) {
                    return {
                        code: 403,
                        response: {
                            status: "error",
                            message: "Access denied to this private room",
                        }
                    };
                }

                console.log(`User ${connection.user_id} joined private room: ${room}`);

                return {
                    code: 200,
                    response: {
                        status: "joined",
                        room: room,
                        message: `Joined private room ${room}`,
                    },
                    websocket: {
                        type: "user_joined",
                        room: room,
                        data: {
                            user: {
                                id: connection.user_id,
                                name: connection.user_data?.name as string,
                            },
                            message: `${connection.user_data?.name as string} joined the room`,
                        },
                        broadcast: true,
                    }
                };
            }

            case 'message': {
                if (!message?.data || typeof message.data !== 'object') {
                    return {
                        code: 400,
                        response: {
                            status: "error",
                            message: "Invalid message format",
                        }
                    };
                }

                const messageData = message.data as { text?: string; type?: string };
                if (!messageData.text) {
                    return {
                        code: 400,
                        response: {
                            status: "error",
                            message: "Message text required",
                        }
                    };
                }

                // Check room access
                const hasAccess = await checkRoomAccess(connection.user_id, message.room);
                if (!hasAccess) {
                    return {
                        code: 403,
                        response: {
                            status: "error",
                            message: "Access denied",
                        }
                    };
                }

                // Store message in database
                await turboQuery(
                    'INSERT INTO private_messages (room, user_id, message, created_at) VALUES ($1, $2, $3, NOW())',
                    [message.room, connection.user_id, messageData.text]
                );

                return {
                    code: 200,
                    response: {
                        status: "message_sent",
                        room: message.room,
                    },
                    websocket: {
                        type: "new_message",
                        room: message.room,
                        data: {
                            user: {
                                id: connection.user_id,
                                name: connection.user_data?.name as string,
                            },
                            text: messageData.text,
                            timestamp: new Date().toISOString(),
                        },
                        broadcast: true,
                    }
                };
            }

            case 'leave': {
                console.log(`User ${connection.user_id} left private room: ${room}`);

                return {
                    code: 200,
                    response: {
                        status: "left",
                        room: room,
                        message: `Left private room ${room}`,
                    },
                    websocket: {
                        type: "user_left",
                        room: room,
                        data: {
                            user: {
                                id: connection.user_id,
                                name: connection.user_data?.name as string,
                            },
                            message: `${connection.user_data?.name as string} left the room`,
                        },
                        broadcast: true,
                    }
                };
            }

            default:
                return {
                    code: 400,
                    response: {
                        status: "error",
                        message: `Unknown event type: ${eventType}`,
                    }
                };
        }
    } catch (error) {
        console.error('Private WebSocket handler error:', error);
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Internal server error",
            }
        };
    }
};

// Helper function to check if user has access to a room
async function checkRoomAccess(userId: string, room: string): Promise<boolean> {
    try {
        const access = await turboQuery(
            'SELECT 1 FROM room_members WHERE room = $1 AND user_id = $2 AND active = true LIMIT 1',
            [room, userId]
        );

        return access.length > 0;
    } catch (error) {
        console.error('Error checking room access:', error);
        return false;
    }
}
