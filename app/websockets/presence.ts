// Presence WebSocket channel handler for rooms like "presence-lobby", "presence-dashboard"
// This handles presence channels where users can see who else is online

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

        switch (eventType) {
            case 'connect':
                return {
                    code: 200,
                    response: {
                        status: "connected",
                        connection_id: connection.id,
                    }
                };

            case 'join': {
                if (!room?.startsWith('presence-')) {
                    return {
                        code: 400,
                        response: {
                            status: "error",
                            message: "Invalid room name. Presence rooms must start with 'presence-'",
                        }
                    };
                }

                console.log(`User ${connection.user_id ?? 'anonymous'} joined presence room: ${room}`);

                // Get current room members
                const members = await turboQuery(
                    'SELECT DISTINCT user_id, user_data FROM websocket_presence WHERE room = $1',
                    [room]
                );

                // Store this user's presence
                await turboQuery(
                    'INSERT INTO websocket_presence (room, user_id, user_data, connected_at) VALUES ($1, $2, $3, NOW()) ON CONFLICT (room, user_id) DO UPDATE SET connected_at = NOW(), user_data = $3',
                    [room, connection.user_id, JSON.stringify(connection.user_data)]
                );

                return {
                    code: 200,
                    response: {
                        status: "joined",
                        room: room,
                        members: members,
                    },
                    websocket: {
                        type: "user_joined",
                        room: room,
                        data: {
                            user: {
                                id: connection.user_id,
                                name: (connection.user_data?.name as string) || 'Anonymous',
                                avatar: connection.user_data?.avatar as string,
                                status: (connection.user_data?.status as string) || 'online',
                            },
                            timestamp: new Date().toISOString(),
                        },
                        broadcast: true,
                    }
                };
            }

            case 'leave': {
                console.log(`User ${connection.user_id ?? 'anonymous'} left presence room: ${room}`);

                // Remove user's presence
                await turboQuery(
                    'DELETE FROM websocket_presence WHERE room = $1 AND user_id = $2',
                    [room, connection.user_id]
                );

                return {
                    code: 200,
                    response: {
                        status: "left",
                        room: room,
                    },
                    websocket: {
                        type: "user_left",
                        room: room,
                        data: {
                            user: {
                                id: connection.user_id,
                                name: (connection.user_data?.name as string) || 'Anonymous',
                            },
                            timestamp: new Date().toISOString(),
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

                const messageData = message.data as { type?: string; status?: string; typing?: boolean };

                if (messageData.type === 'status_update' && messageData.status) {
                    // Update user status in presence table
                    await turboQuery(
                        'UPDATE websocket_presence SET user_data = jsonb_set(user_data, \'{status}\', $3) WHERE room = $1 AND user_id = $2',
                        [message.room, connection.user_id, JSON.stringify(messageData.status)]
                    );

                    return {
                        code: 200,
                        response: { status: "status_updated" },
                        websocket: {
                            type: "status_changed",
                            room: message.room,
                            data: {
                                user: {
                                    id: connection.user_id,
                                    name: connection.user_data?.name as string,
                                    status: messageData.status,
                                },
                                timestamp: new Date().toISOString(),
                            },
                            broadcast: true,
                        }
                    };
                }

                if (messageData.type === 'typing') {
                    return {
                        code: 200,
                        response: { status: "typing_broadcast" },
                        websocket: {
                            type: "user_typing",
                            room: message.room,
                            data: {
                                user: {
                                    id: connection.user_id,
                                    name: connection.user_data?.name as string,
                                    avatar: connection.user_data?.avatar as string,
                                },
                                typing: Boolean(messageData.typing),
                                timestamp: new Date().toISOString(),
                            },
                            broadcast: true,
                        }
                    };
                }

                return {
                    code: 400,
                    response: {
                        status: "error",
                        message: "Unknown message type",
                    }
                };
            }

            case 'get_members': {
                // Get current room members
                const members = await turboQuery(
                    'SELECT user_id, user_data, connected_at FROM websocket_presence WHERE room = $1 ORDER BY connected_at',
                    [room]
                );

                return {
                    code: 200,
                    response: {
                        status: "members",
                        room: room,
                        members: members,
                        count: members.length,
                    }
                };
            }

            case 'disconnect': {
                // Clean up presence when user disconnects
                if (connection.room) {
                    await turboQuery(
                        'DELETE FROM websocket_presence WHERE room = $1 AND user_id = $2',
                        [connection.room, connection.user_id]
                    );

                    // Broadcast that user left
                    return {
                        code: 200,
                        response: { status: "disconnected" },
                        websocket: {
                            type: "user_disconnected",
                            room: connection.room,
                            data: {
                                user: {
                                    id: connection.user_id,
                                    name: (connection.user_data?.name as string) || 'Anonymous',
                                },
                                timestamp: new Date().toISOString(),
                            },
                            broadcast: true,
                        }
                    };
                }

                return {
                    code: 200,
                    response: { status: "disconnected" }
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
        console.error('Presence WebSocket handler error:', error);
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Internal server error",
            }
        };
    }
};
