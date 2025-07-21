// Global type definitions for TurboScript endpoints

/// <reference path="../turbo_modules/argon2/index.d.ts" />

declare global {
    // Context types for different endpoint types
    interface HttpContext {
        type: 'http';
        method: string;
        path: string;
        userAgent: string;
        remoteAddr: string;
    }

    interface WebSocketContext {
        type: 'websocket';
        eventType: 'connect' | 'disconnect' | 'join' | 'leave' | 'message' | string;
        connection: WebSocketConnection;
        message?: WebSocketMessage;
        room?: string;
    }

    interface SSEContext {
        type: 'sse';
        eventType: 'connect' | 'disconnect' | 'message' | string;
        connection: SSEConnection;
        data?: unknown;
    }

    // Union type for all possible contexts
    type Context = HttpContext | WebSocketContext | SSEContext;

    // Base Event interface for all request types
    interface Event {
        headers: Record<string, string>;
        queryParameters: Record<string, string>;
        pathParameters: Record<string, string>;
        body: Record<string, unknown>;
        env: Record<string, string>; // Environment variables
        context: Context; // Context based on endpoint type
    }

    interface TurboScriptResponse {
        code: number;
        response: Record<string, unknown> | string;
        cookies?: string[];
        type?: 'json' | 'html' | 'markdown' | 'text' | 'markdown-html';

        // Real-time response options
        websocket?: WebSocketResponse; // only applicable for WebSocket handlers
        sse?: SSEResponse; // only applicable for SSE handlers
    }

    // WebSocket Types
    interface WebSocketConnection {
        id: string;
        room: string;
        user_id?: string;
        user_data?: Record<string, unknown>;
        connected_at: string;
        last_ping: string;
        is_alive: boolean;
        remote_addr: string;
        user_agent: string;
    }

    interface WebSocketMessage {
        type: string;
        room: string;
        data?: unknown;
        user_id?: string;
        message_id: string;
        timestamp: string;
        metadata: Record<string, unknown>;
    }

    interface WebSocketResponse {
        type: string;
        room?: string;
        data?: unknown;
        broadcast?: boolean;
        target?: string; // Connection ID to send to specific connection
        error?: string;
    }

    // SSE Types
    interface SSEConnection {
        id: string;
        user_id: string;
        user_data: Record<string, unknown>;
        connected_at: string;
        last_activity: string;
        remote_addr: string;
        user_agent: string;
        is_active: boolean;
    }

    interface SSEResponse {
        event: string;
        data: unknown;
        id?: string;
        retry?: number;
        target?: string; // Connection ID to send to specific connection
        broadcast?: boolean;
        user_id?: string; // Send to all connections for this user
    }

    // turboQuery function for executing database queries.
    // Always returns a Promise that resolves to an array of results.
    // When no rows are found, returns an empty array (never null).
    // Timeout behavior is controlled by the global 'timeout' setting in turboscript.yml
    // or per-endpoint 'timeout' setting in the endpoint configuration.
    // Database queries that exceed the timeout will throw an error.

    // Standard signature (uses default connection)
    function turboQuery<T = any>(query: string, params?: Record<string, any>): Promise<T[]>;

    // Object signature with connection support
    function turboQuery<T = any>(options: {
        query: string;
        bindings?: unknown[];
        connection?: string; // Connection name from turboscript.yml database.connections
    }): Promise<T[]>;

    // Email configuration interface for different drivers
    interface EmailConfig {
        to: string | string[];
        from?: string;
        subject: string;
        content: string;
        html?: string;
        driver: 'mailgun' | 'smtp' | 'ses' | 'sendgrid' | 'postmark';
        cc?: string | string[];
        bcc?: string | string[];
        attachments?: EmailAttachment[];
    }

    interface EmailAttachment {
        filename: string;
        content: string; // base64 encoded content
        contentType?: string;
    }

    // turboEmail function for sending emails
    // Uses the configured email driver from turboscript.yml
    // Returns a Promise that resolves when the email is sent
    function turboEmail(config: EmailConfig): Promise<void>;

    // Job status types
    type JobStatus = 'pending' | 'processing' | 'completed' | 'failed' | 'cancelled';

    // Job history status types
    type JobHistoryStatus = 'started' | 'completed' | 'failed' | 'timeout' | 'retry_requested' | 'cancelled';

    // Job information interface
    interface JobInfo {
        id: number;
        uid: string;
        type: string;
        path: string;
        status: JobStatus;
        priority: number;
        retry_count: number;
        max_retries: number;
        next_retry_at: string | null;
        scheduled_at: string;
        started_at: string | null;
        completed_at: string | null;
        error_message: string | null;
        created_at: string;
        updated_at: string;
    }

    // Job history entry interface
    interface JobHistoryEntry {
        id: number;
        job_id: number;
        job_uid: string;
        attempt_number: number;
        status: JobHistoryStatus;
        worker_id: string | null;
        started_at: string;
        completed_at: string | null;
        duration_ms: number | null;
        error_message: string | null;
        created_at: string;
    }

    // turboJob function for dispatching background jobs
    // Jobs are processed asynchronously in background goroutines
    // Returns a Promise that resolves with the job UID when the job is queued (not when it's processed)
    function turboJob(jobPath: string, payload: Record<string, unknown>): Promise<string>;

    // turboMarkdownHtml function for processing markdown content with layout
    // Converts markdown files to HTML using the templating engine
    // Useful for creating dynamic content with markdown processing
    function turboMarkdownHtml(filePath: string, data?: Record<string, unknown>): Promise<string>;

    // turboHtml function for processing HTML content with template substitution
    // Loads HTML files and performs template variable substitution
    // Useful for serving static HTML content with dynamic data
    function turboHtml(filePath: string, data?: Record<string, unknown>): Promise<string>;

    // File Upload Plugin Types
    interface FileInfo {
        originalName: string;   // Original filename
        filename: string;       // Generated unique filename
        size: number;          // File size in bytes
        mimeType: string;      // MIME type (e.g., "image/jpeg")
        extension: string;     // File extension (e.g., ".jpg")
        path: string;          // Full filesystem path
        url: string;           // URL to access the file
        md5Hash?: string;      // MD5 hash (if generateHash: true)
        sha256Hash?: string;   // SHA256 hash (if generateHash: true)
        uploadedAt: string;    // Upload timestamp (ISO 8601)
    }

    interface UploadOptions {
        directory?: string;        // Subdirectory within upload_dir
        filename?: string;         // Custom filename (for saveFile)
        allowedTypes?: string[];   // Allowed MIME types
        maxSize?: number;          // Maximum file size in bytes
        generateHash?: boolean;    // Generate MD5/SHA256 hashes
    }

    interface FileUploadPlugin {
        saveBase64(base64Data: string, filename: string, options?: UploadOptions): Promise<FileInfo>;
        saveFile(binaryData: Uint8Array | Buffer, options: UploadOptions & { filename: string }): Promise<FileInfo>;
        getFileInfo(filePath: string): Promise<FileInfo>;
        deleteFile(filePath: string): Promise<void>;
        validateFile(data: string | Uint8Array, contentType: string, options?: UploadOptions): Promise<void>;
        generateFilename(originalName: string): string;
        hashFile(filePath: string): Promise<{ md5: string; sha256: string }>;
    }

    // turboPlugin function for accessing TurboScript plugins
    // This provides type-safe access to plugins without using require()
    function turboPlugin<T = string>(pluginName: string): T;
    function turboPlugin(pluginName: string): unknown;

    // turboBroadcast function for broadcasting WebSocket and SSE messages
    // This allows HTTP endpoints to send real-time messages to connected clients
    interface BroadcastResult {
        success: boolean;
        connections_notified: number;
        message_type?: string;
        room?: string;
        target?: string;
        event?: string;
        message_id?: string;
        user_id?: string;
    }

    interface WebSocketBroadcastMessage {
        type: string;
        room?: string;
        data: Record<string, unknown>;
        target?: string; // Specific connection ID
        broadcast?: boolean; // Broadcast to all in room (default: true)
    }

    interface SSEBroadcastMessage {
        event: string;
        data: Record<string, unknown>;
        id?: string;
        retry?: number;
        target?: string; // Specific connection ID
        broadcast?: boolean; // Broadcast to all connections (default: true)
        user_id?: string; // Send to all connections for this user
    }

    // Broadcast WebSocket message to connected clients
    function turboBroadcastWebSocket(message: WebSocketBroadcastMessage): Promise<BroadcastResult>;

    // Broadcast SSE message to connected clients
    function turboBroadcastSSE(message: SSEBroadcastMessage): Promise<BroadcastResult>;

    // Get connection statistics
    function turboGetConnections(filter?: string): Promise<{
        websocket: {
            total_connections: number;
            rooms: Record<string, number>;
        };
        sse: {
            total_connections: number;
            users: Record<string, number>;
        };
    }>;
}

// This export is required to make this file a module
// Without it, the global declarations won't work properly
export { };
