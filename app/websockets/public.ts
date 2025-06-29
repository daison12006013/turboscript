// Public WebSocket channel handler for rooms like "public-general", "public-announcements"
// This handles public chat rooms where anyone can join and participate

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

        console.log(`WebSocket ${eventType} event for connection ${connection.id}`);
        console.log('Connection user_data:', connection.user_data);

        switch (eventType) {
            case 'connect':
                return {
                    code: 200,
                    response: {
                        status: "connected",
                        message: "WebSocket connected successfully",
                        connection_id: connection.id,
                    }
                };

            case 'join': {
                if (!room?.startsWith('public-')) {
                    return {
                        code: 400,
                        response: {
                            status: "error",
                            message: "Invalid room name. Public rooms must start with 'public-'",
                        }
                    };
                }

                // Extract user data from message (for authenticated users) or use connection defaults
                let userId = connection.user_id;
                let userName = (connection.user_data?.name as string) || 'Anonymous';
                let userEmail = connection.user_data?.email as string;
                let userAvatar = (connection.user_data?.avatar as string) || '';

                // Check if authenticated user data is provided in the message
                console.log('Message data:', message?.data);
                if (message?.data && typeof message.data === 'object') {
                    const messageData = message.data as Record<string, unknown>;
                    console.log('Message data type check passed, data:', messageData);
                    if (messageData.authenticated_user && typeof messageData.authenticated_user === 'object') {
                        console.log('Authenticated user found in message:', messageData.authenticated_user);
                        const authUser = messageData.authenticated_user as Record<string, unknown>;
                        userId = (authUser.user_id as string) || userId;
                        userName = (authUser.name as string) || userName;
                        userEmail = (authUser.email as string) || userEmail;
                        userAvatar = (authUser.avatar as string) || userAvatar;
                        console.log(`Authenticated user data received: ${userName} (${userId})`);
                    } else {
                        console.log('No authenticated_user in message data');
                    }
                } else {
                    console.log('No message data or invalid type');
                }

                console.log(`User ${userId ?? 'anonymous'} joined public room: ${room}`);

                return {
                    code: 200,
                    response: {
                        status: "joined",
                        room: room,
                        message: `Welcome to ${room}!`,
                    },
                    websocket: {
                        type: "room_joined",
                        room: room,
                        data: {
                            user: {
                                id: userId ?? 'anonymous',
                                name: userName,
                                email: userEmail,
                                avatar: userAvatar
                            },
                            message: `${userName} joined the room`
                        },
                        broadcast: true,
                    }
                };
            }

            case 'leave': {
                console.log(`WebSocket leave event for connection ${connection.id}`);
                console.log(`User ${connection.user_id ?? 'anonymous'} left public room: ${room}`);

                // Get user information safely - for anonymous connections, user_data is empty {}
                const leaveUserName = (connection.user_data?.name as string) || 'Anonymous';

                return {
                    code: 200,
                    response: {
                        status: "left",
                        room: room,
                        type: "left",  // Add type field to make it consistent with other responses
                    },
                    websocket: {
                        type: "left",   // Send "left" message back to the sender
                        room: room,
                        data: {
                            user: {
                                id: connection.user_id,
                                name: leaveUserName
                            },
                            message: `You left the room ${room}`
                        },
                        broadcast: true,  // Broadcast this confirmation so it reaches the connection
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
                            message: "Message text is required",
                        }
                    };
                }

                // Store message in database (optional)
                await turboQuery(
                    'INSERT INTO public_chat_messages (room, user_id, message, created_at) VALUES ($1, $2, $3, NOW())',
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
                            id: message.message_id,
                            user: {
                                id: connection.user_id,
                                name: (connection.user_data?.name as string) || 'Anonymous',
                                avatar: (connection.user_data?.avatar as string) || '',
                            },
                            text: messageData.text,
                            timestamp: new Date().toISOString(),
                            type: messageData.type ?? 'text',
                        },
                        broadcast: true,
                    }
                };
            }

            case 'typing': {
                const typingData = message?.data as { typing?: boolean } | undefined;
                return {
                    code: 200,
                    response: { status: "typing_broadcast" },
                    websocket: {
                        type: "user_typing",
                        room: room,
                        data: {
                            user: {
                                id: connection.user_id,
                                name: (connection.user_data?.name as string) || 'Anonymous',
                            },
                            typing: Boolean(typingData?.typing),
                        },
                        broadcast: true,
                    }
                };
            }

            case 'get_history': {
                const history = await turboQuery(
                    `SELECT pcm.*, u.name, u.avatar
                     FROM public_chat_messages pcm
                     LEFT JOIN users u ON pcm.user_id = u.uid
                     WHERE pcm.room = $1
                     ORDER BY pcm.created_at DESC
                     LIMIT 50`,
                    [room]
                );

                return {
                    code: 200,
                    response: {
                        status: "history",
                        room: room,
                        messages: history.reverse(),
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
        console.error('Public WebSocket handler error:', error);
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Internal server error",
            }
        };
    }
};
